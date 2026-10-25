package chariot

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"strconv"
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
	defaultListenerMaxBodyBytes = int64(1 << 20)
)

type listenerOptions struct {
	Port           int
	BindAddress    string
	HandlerName    string
	HandlerFn      *FunctionValue
	OnStartProgram string
	OnExitProgram  string
	AllowedMethods map[string]struct{}
	BasePath       string
	ReadTimeout    time.Duration
	WriteTimeout   time.Duration
	IdleTimeout    time.Duration
	MaxBodyBytes   int64
	StrictJSON     bool
}

type runtimeListener struct {
	runtime      *Runtime
	opts         listenerOptions
	server       *http.Server
	netListener  net.Listener
	metrics      listenerMetrics
	done         chan struct{}
	shutdownOnce sync.Once
	forceOnce    sync.Once
	handlersMu   sync.Mutex
	draining     bool
	handlers     sync.WaitGroup
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
	if opts.MaxBodyBytes == 0 {
		opts.MaxBodyBytes = defaultListenerMaxBodyBytes
	}
	opts.BasePath = normalizeBasePath(opts.BasePath)

	listener := &runtimeListener{
		runtime: rt,
		opts:    opts,
		metrics: newListenerMetrics(),
		done:    make(chan struct{}),
	}
	server := &http.Server{
		Addr:         net.JoinHostPort(strings.TrimSpace(opts.BindAddress), strconv.Itoa(opts.Port)),
		Handler:      http.HandlerFunc(listener.handleRequest),
		ReadTimeout:  opts.ReadTimeout,
		WriteTimeout: opts.WriteTimeout,
		IdleTimeout:  opts.IdleTimeout,
	}
	listener.server = server

	netListener, err := net.Listen("tcp", server.Addr)
	if err != nil {
		return nil, fmt.Errorf("failed to bind listener on port %d: %w", opts.Port, err)
	}
	listener.netListener = netListener

	if err := registerRuntimeListener(rt, listener); err != nil {
		_ = netListener.Close()
		return nil, err
	}

	if opts.OnStartProgram != "" {
		if err := rt.RunProgram(opts.OnStartProgram, opts.Port); err != nil {
			unregisterRuntimeListener(rt, opts.Port)
			_ = netListener.Close()
			listener.metrics.recordError(err)
			listener.metrics.setState("failed")
			return nil, fmt.Errorf("listen onStart script failed: %w", err)
		}
	}

	listener.metrics.setState("running")
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
	defer close(l.done)
	defer unregisterRuntimeListener(l.runtime, l.opts.Port)

	cfg.ChariotLogger.Info("listen started",
		zapPort(l.opts.Port),
		zapHandler(l.opts.HandlerName))

	err := l.server.Serve(l.netListener)
	// Serve returns as soon as the accept loop stops, not when handlers exit.
	// Seal admission before Wait so a late accepted connection cannot add work.
	l.handlersMu.Lock()
	l.draining = true
	l.handlersMu.Unlock()
	if err != nil && !errors.Is(err, http.ErrServerClosed) {
		l.metrics.recordTerminalError(err)
		l.metrics.setState("failed")
		_ = l.server.Close()
	}
	l.handlers.Wait()

	if l.opts.OnExitProgram != "" {
		if exitErr := l.runtime.WithExecution(func() error {
			return l.runtime.RunProgram(l.opts.OnExitProgram, l.opts.Port)
		}); exitErr != nil {
			l.metrics.recordTerminalError(fmt.Errorf("listen onExit script failed: %w", exitErr))
			cfg.ChariotLogger.Warn("listen onExit script failed",
				zapError(exitErr),
				zapPort(l.opts.Port))
		}
	}

	if err != nil && !errors.Is(err, http.ErrServerClosed) {
		l.metrics.setState("failed")
		cfg.ChariotLogger.Error("listen server stopped with error",
			zapError(err),
			zapPort(l.opts.Port))
	} else {
		if l.metrics.terminalErr() != nil {
			l.metrics.setState("failed")
		} else {
			l.metrics.setState("stopped")
		}
		cfg.ChariotLogger.Info("listen server stopped",
			zapPort(l.opts.Port))
	}
}

// requestStop seals admission and publishes stopping before returning, but does
// not wait for handlers or hooks. Shutdown and language requests share this once.
func (l *runtimeListener) requestStop(ctx context.Context) (started bool) {
	l.shutdownOnce.Do(func() {
		select {
		case <-l.done:
			return
		default:
		}
		started = true
		l.handlersMu.Lock()
		l.draining = true
		l.handlersMu.Unlock()
		l.metrics.setState("stopping")
		go func() {
			if err := l.server.Shutdown(ctx); err != nil {
				l.forceClose(err)
			}
		}()
	})
	return started
}

func (l *runtimeListener) Shutdown(ctx context.Context) error {
	if l.server == nil {
		return nil
	}
	select {
	case <-l.done:
		return l.metrics.terminalErr()
	default:
	}

	l.requestStop(ctx)
	select {
	case <-l.done:
		return l.metrics.terminalErr()
	case <-ctx.Done():
		l.forceClose(ctx.Err())
		return ctx.Err()
	}
}

func (l *runtimeListener) forceClose(err error) {
	// A stop-request waiter cancels its context after done. Server.Shutdown
	// may still be polling for idle connections; that cancellation is not a
	// failure of an already completed listener.
	select {
	case <-l.done:
		return
	default:
	}
	l.forceOnce.Do(func() {
		l.metrics.recordTerminalError(err)
		l.metrics.setState("failed")
		_ = l.server.Close()
	})
}

func (l *runtimeListener) handleRequest(w http.ResponseWriter, r *http.Request) {
	l.handlersMu.Lock()
	if l.draining {
		l.handlersMu.Unlock()
		http.Error(w, "listener stopping", http.StatusServiceUnavailable)
		return
	}
	l.handlers.Add(1)
	l.handlersMu.Unlock()
	defer l.handlers.Done()
	l.metrics.recordStart()
	completed := false
	var requestErr error
	defer func() {
		if completed {
			l.metrics.recordSuccess()
		} else {
			l.metrics.recordFailure(requestErr)
		}
	}()

	if l.opts.BasePath != "" && l.opts.BasePath != "/" {
		if r.URL.Path != l.opts.BasePath && !strings.HasPrefix(r.URL.Path, l.opts.BasePath+"/") {
			http.NotFound(w, r)
			completed = true
			return
		}
	}

	if len(l.opts.AllowedMethods) > 0 && !l.methodAllowed(r.Method) {
		writeJSONError(w, http.StatusMethodNotAllowed, fmt.Sprintf("method %s not allowed", r.Method))
		completed = true
		return
	}

	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		completed = true
		return
	}

	defer r.Body.Close()
	body, err := io.ReadAll(io.LimitReader(r.Body, l.opts.MaxBodyBytes+1))
	if err != nil {
		requestErr = err
		writeJSONError(w, http.StatusBadRequest, "failed to read request body")
		return
	}
	if int64(len(body)) > l.opts.MaxBodyBytes {
		requestErr = fmt.Errorf("request body exceeds %d bytes", l.opts.MaxBodyBytes)
		writeJSONError(w, http.StatusRequestEntityTooLarge, requestErr.Error())
		return
	}

	payload, err := buildListenerRequestValue(r, body, l.opts.StrictJSON)
	if err != nil {
		requestErr = err
		writeJSONError(w, http.StatusBadRequest, err.Error())
		return
	}

	result, execErr := l.invokeHandler(payload)
	if execErr != nil {
		requestErr = execErr
		writeJSONError(w, http.StatusInternalServerError, execErr.Error())
		return
	}

	if result == nil || result == DBNull {
		w.WriteHeader(http.StatusNoContent)
		completed = true
		return
	}

	if err := writeValueResponse(w, result); err != nil {
		requestErr = err
		writeJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}
	completed = true
}

func (l *runtimeListener) id() string {
	return fmt.Sprintf("http:%s", l.netListener.Addr().String())
}

func (l *runtimeListener) status() ListenerStatus {
	return l.metrics.snapshot(ListenerStatus{
		ID:      l.id(),
		Kind:    ListenerHTTP,
		Address: l.netListener.Addr().String(),
		Port:    l.opts.Port,
	})
}

func runtimeHTTPListeners(rt *Runtime) []*runtimeListener {
	runtimeListenersMu.Lock()
	defer runtimeListenersMu.Unlock()
	perRuntime := runtimeListeners[rt]
	listeners := make([]*runtimeListener, 0, len(perRuntime))
	for _, listener := range perRuntime {
		listeners = append(listeners, listener)
	}
	return listeners
}

func listRuntimeHTTPListenerStatuses(rt *Runtime) []ListenerStatus {
	listeners := runtimeHTTPListeners(rt)
	statuses := make([]ListenerStatus, 0, len(listeners))
	for _, listener := range listeners {
		statuses = append(statuses, listener.status())
	}
	return statuses
}

func stopRuntimeHTTPListener(ctx context.Context, rt *Runtime, id string) (bool, error) {
	for _, listener := range runtimeHTTPListeners(rt) {
		if listener.id() == id {
			return true, listener.Shutdown(ctx)
		}
	}
	return false, nil
}

func (l *runtimeListener) invokeHandler(payload Value) (result Value, err error) {
	err = l.runtime.WithExecution(func() error {
		fn := l.opts.HandlerFn
		if fn == nil {
			var ok bool
			fn, ok = l.runtime.GetFunction(l.opts.HandlerName)
			if !ok {
				return fmt.Errorf("handler '%s' not found", l.opts.HandlerName)
			}
		}
		var execErr error
		result, execErr = executeFunctionValue(l.runtime, fn, []Value{payload})
		return execErr
	})
	return result, err
}

func (l *runtimeListener) methodAllowed(method string) bool {
	_, ok := l.opts.AllowedMethods[strings.ToUpper(method)]
	return ok
}

func buildListenerRequestValue(r *http.Request, body []byte, strictJSON ...bool) (Value, error) {
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
				if len(strictJSON) > 0 && strictJSON[0] {
					return nil, fmt.Errorf("invalid JSON request body: %w", err)
				}
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
	if response, ok := result.(*httpResponseValue); ok {
		for name, values := range response.Headers {
			for _, value := range values {
				w.Header().Add(name, value)
			}
		}
		if response.Body != nil && response.Body != DBNull && w.Header().Get("Content-Type") == "" {
			w.Header().Set("Content-Type", "application/json")
		}
		w.WriteHeader(response.StatusCode)
		if response.Body == nil || response.Body == DBNull || response.StatusCode == http.StatusNoContent || response.StatusCode == http.StatusNotModified {
			return nil
		}
		return writeValueBody(w, response.Body)
	}
	w.Header().Set("Content-Type", "application/json")
	return writeValueBody(w, result)
}

func writeValueBody(w http.ResponseWriter, result Value) error {
	switch val := result.(type) {
	case *JSONNode:
		jsonStr, err := val.ToJSON()
		if err != nil {
			return fmt.Errorf("failed to serialize JSONNode: %w", err)
		}
		_, err = w.Write([]byte(jsonStr))
		return err
	default:
		native := ValueToJSON(result)
		data, err := json.Marshal(native)
		if err != nil {
			return fmt.Errorf("failed to serialize response: %w", err)
		}
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
