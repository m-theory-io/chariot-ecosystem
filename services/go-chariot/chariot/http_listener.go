package chariot

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	cfg "github.com/bhouse1273/chariot-ecosystem/services/go-chariot/configs"
	"go.uber.org/zap"
)

const (
	defaultListenerReadTimeout  = 15 * time.Second
	defaultListenerWriteTimeout = 30 * time.Second
	defaultListenerIdleTimeout  = 60 * time.Second
)

type listenerOptions struct {
	Port           int
	HandlerName    string
	HandlerFn      *FunctionValue
	OnStartProgram string
	OnExitProgram  string
	AllowedMethods map[string]struct{}
	BasePath       string
	ReadTimeout    time.Duration
	WriteTimeout   time.Duration
	IdleTimeout    time.Duration
}

type runtimeListener struct {
	runtime *Runtime
	opts    listenerOptions
	server  *http.Server
	execMu  sync.Mutex
}

var (
	runtimeListenersMu sync.Mutex
	runtimeListeners   = make(map[*Runtime]map[int]*runtimeListener)
	portsInUse         = make(map[int]*runtimeListener)
)

func startRuntimeListener(rt *Runtime, opts listenerOptions) (*runtimeListener, error) {
	if opts.Port <= 0 || opts.Port > 65535 {
		return nil, fmt.Errorf("invalid port: %d", opts.Port)
	}
	if opts.HandlerName == "" && opts.HandlerFn == nil {
		return nil, errors.New("listen requires a handler name or function")
	}
	if opts.ReadTimeout == 0 {
		opts.ReadTimeout = defaultListenerReadTimeout
	}
	if opts.WriteTimeout == 0 {
		opts.WriteTimeout = defaultListenerWriteTimeout
	}
	if opts.IdleTimeout == 0 {
		opts.IdleTimeout = defaultListenerIdleTimeout
	}
	opts.BasePath = normalizeBasePath(opts.BasePath)

	listener := &runtimeListener{runtime: rt, opts: opts}
	server := &http.Server{
		Addr:         fmt.Sprintf(":%d", opts.Port),
		Handler:      http.HandlerFunc(listener.handleRequest),
		ReadTimeout:  opts.ReadTimeout,
		WriteTimeout: opts.WriteTimeout,
		IdleTimeout:  opts.IdleTimeout,
	}
	listener.server = server

	if err := registerRuntimeListener(rt, listener); err != nil {
		return nil, err
	}

	go listener.serve()
	return listener, nil
}

func registerRuntimeListener(rt *Runtime, l *runtimeListener) error {
	runtimeListenersMu.Lock()
	defer runtimeListenersMu.Unlock()

	if existing := portsInUse[l.opts.Port]; existing != nil {
		return fmt.Errorf("port %d is already in use", l.opts.Port)
	}

	if _, ok := runtimeListeners[rt]; !ok {
		runtimeListeners[rt] = make(map[int]*runtimeListener)
	}
	runtimeListeners[rt][l.opts.Port] = l
	portsInUse[l.opts.Port] = l
	return nil
}

func unregisterRuntimeListener(rt *Runtime, port int) {
	runtimeListenersMu.Lock()
	defer runtimeListenersMu.Unlock()

	if perRuntime, ok := runtimeListeners[rt]; ok {
		delete(perRuntime, port)
		if len(perRuntime) == 0 {
			delete(runtimeListeners, rt)
		}
	}
	delete(portsInUse, port)
}

func (l *runtimeListener) serve() {
	defer unregisterRuntimeListener(l.runtime, l.opts.Port)

	if l.opts.OnStartProgram != "" {
		if err := l.runtime.RunProgram(l.opts.OnStartProgram, l.opts.Port); err != nil {
			cfg.ChariotLogger.Warn("listen onStart script failed",
				zapError(err),
				zapPort(l.opts.Port))
		}
	}

	cfg.ChariotLogger.Info("listen started",
		zapPort(l.opts.Port),
		zapHandler(l.opts.HandlerName))

	err := l.server.ListenAndServe()

	if l.opts.OnExitProgram != "" {
		if exitErr := l.runtime.RunProgram(l.opts.OnExitProgram, l.opts.Port); exitErr != nil {
			cfg.ChariotLogger.Warn("listen onExit script failed",
				zapError(exitErr),
				zapPort(l.opts.Port))
		}
	}

	if err != nil && !errors.Is(err, http.ErrServerClosed) {
		cfg.ChariotLogger.Error("listen server stopped with error",
			zapError(err),
			zapPort(l.opts.Port))
	} else {
		cfg.ChariotLogger.Info("listen server stopped",
			zapPort(l.opts.Port))
	}
}

func (l *runtimeListener) Shutdown(ctx context.Context) error {
	if l.server == nil {
		return nil
	}
	return l.server.Shutdown(ctx)
}

func (l *runtimeListener) handleRequest(w http.ResponseWriter, r *http.Request) {
	if l.opts.BasePath != "" && l.opts.BasePath != "/" {
		if !strings.HasPrefix(r.URL.Path, l.opts.BasePath) {
			http.NotFound(w, r)
			return
		}
	}

	if len(l.opts.AllowedMethods) > 0 && !l.methodAllowed(r.Method) {
		writeJSONError(w, http.StatusMethodNotAllowed, fmt.Sprintf("method %s not allowed", r.Method))
		return
	}

	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "failed to read request body")
		return
	}
	defer r.Body.Close()

	payload, err := buildListenerRequestValue(r, body)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, err.Error())
		return
	}

	result, execErr := l.invokeHandler(payload)
	if execErr != nil {
		writeJSONError(w, http.StatusInternalServerError, execErr.Error())
		return
	}

	if result == nil || result == DBNull {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	if err := writeValueResponse(w, result); err != nil {
		writeJSONError(w, http.StatusInternalServerError, err.Error())
	}
}

func (l *runtimeListener) invokeHandler(payload Value) (Value, error) {
	fn := l.opts.HandlerFn
	if fn == nil {
		handlerName := l.opts.HandlerName
		var ok bool
		fn, ok = l.runtime.GetFunction(handlerName)
		if !ok {
			return nil, fmt.Errorf("handler '%s' not found", handlerName)
		}
	}

	l.execMu.Lock()
	defer l.execMu.Unlock()

	return executeFunctionValue(l.runtime, fn, []Value{payload})
}

func (l *runtimeListener) methodAllowed(method string) bool {
	_, ok := l.opts.AllowedMethods[strings.ToUpper(method)]
	return ok
}

func buildListenerRequestValue(r *http.Request, body []byte) (Value, error) {
	reqMap := map[string]interface{}{
		"method":       r.Method,
		"path":         r.URL.Path,
		"query_string": r.URL.RawQuery,
		"remote_addr":  r.RemoteAddr,
		"host":         r.Host,
		"timestamp":    time.Now().UTC().Format(time.RFC3339Nano),
	}

	if r.TLS != nil {
		reqMap["scheme"] = "https"
	} else {
		reqMap["scheme"] = "http"
	}

	if len(body) > 0 {
		reqMap["raw_body"] = string(body)
	}

	if query := flattenStringValues(r.URL.Query()); len(query) > 0 {
		reqMap["query"] = query
	}

	if headers := flattenHeaders(r.Header); len(headers) > 0 {
		reqMap["headers"] = headers
	}

	contentType := r.Header.Get("Content-Type")
	if len(body) > 0 {
		switch {
		case strings.Contains(contentType, "application/json"):
			var payload interface{}
			if err := json.Unmarshal(body, &payload); err == nil {
				reqMap["body"] = payload
			} else {
				reqMap["body_error"] = err.Error()
			}
		case strings.Contains(contentType, "application/x-www-form-urlencoded"),
			strings.Contains(contentType, "multipart/form-data"):
			if err := r.ParseForm(); err == nil {
				reqMap["form"] = flattenStringValues(r.PostForm)
			} else {
				reqMap["form_error"] = err.Error()
			}
		}
	}

	return JSONToValue(reqMap)
}

func flattenStringValues(input map[string][]string) map[string]interface{} {
	result := make(map[string]interface{}, len(input))
	for key, values := range input {
		switch len(values) {
		case 0:
			result[key] = ""
		case 1:
			result[key] = values[0]
		default:
			arr := make([]interface{}, len(values))
			for i, val := range values {
				arr[i] = val
			}
			result[key] = arr
		}
	}
	return result
}

func flattenHeaders(input http.Header) map[string]interface{} {
	if len(input) == 0 {
		return nil
	}
	result := make(map[string]interface{}, len(input))
	for key, values := range input {
		arr := make([]interface{}, len(values))
		for i, val := range values {
			arr[i] = val
		}
		result[strings.ToLower(key)] = arr
	}
	return result
}

func writeValueResponse(w http.ResponseWriter, result Value) error {
	switch val := result.(type) {
	case *JSONNode:
		jsonStr, err := val.ToJSON()
		if err != nil {
			return fmt.Errorf("failed to serialize JSONNode: %w", err)
		}
		w.Header().Set("Content-Type", "application/json")
		_, err = w.Write([]byte(jsonStr))
		return err
	default:
		native := ValueToJSON(result)
		data, err := json.Marshal(native)
		if err != nil {
			return fmt.Errorf("failed to serialize response: %w", err)
		}
		w.Header().Set("Content-Type", "application/json")
		_, err = w.Write(data)
		return err
	}
}

func writeJSONError(w http.ResponseWriter, status int, message string) {
	payload := map[string]interface{}{
		"error":   message,
		"status":  status,
		"success": false,
	}
	data, _ := json.Marshal(payload)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_, _ = w.Write(data)
}

func normalizeBasePath(path string) string {
	trimmed := strings.TrimSpace(path)
	if trimmed == "" || trimmed == "/" {
		return ""
	}
	if !strings.HasPrefix(trimmed, "/") {
		trimmed = "/" + trimmed
	}
	trimmed = strings.TrimSuffix(trimmed, "/")
	if trimmed == "" {
		return ""
	}
	return trimmed
}

func zapPort(port int) zap.Field {
	return zap.Int("port", port)
}

func zapHandler(handler string) zap.Field {
	if handler == "" {
		handler = "<inline>"
	}
	return zap.String("handler", handler)
}

func zapError(err error) zap.Field {
	return zap.Error(err)
}
