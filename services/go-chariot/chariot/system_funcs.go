package chariot

import (
	"errors"
	"fmt"
	"os"
	"runtime"
	"strconv"
	"strings"
	"time"

	cfg "github.com/bhouse1273/chariot-ecosystem/services/go-chariot/configs"
	"go.uber.org/zap"
)

// RegisterSystem registers all system-related functions
func RegisterSystem(rt *Runtime) {
	// Environment information
	rt.Register("getEnv", func(args ...Value) (Value, error) {
		if len(args) != 1 {
			return nil, errors.New("getEnv requires 1 argument: variable name")
		}

		// Unwrap arguments
		for i, arg := range args {
			if tvar, ok := arg.(ScopeEntry); ok {
				args[i] = tvar.Value
			}
		}

		name, ok := args[0].(Str)
		if !ok {
			return nil, fmt.Errorf("variable name must be a string, got %T", args[0])
		}

		value, exists := os.LookupEnv(string(name))
		if !exists {
			return DBNull, nil
		}

		return Str(value), nil
	})

	rt.Register("hasEnv", func(args ...Value) (Value, error) {
		if len(args) != 1 {
			return nil, errors.New("hasEnv requires 1 argument: variable name")
		}

		// Unwrap arguments
		for i, arg := range args {
			if tvar, ok := arg.(ScopeEntry); ok {
				args[i] = tvar.Value
			}
		}

		name, ok := args[0].(Str)
		if !ok {
			return nil, fmt.Errorf("variable name must be a string, got %T", args[0])
		}

		_, exists := os.LookupEnv(string(name))
		return Bool(exists), nil
	})

	rt.Register("logPrint", func(args ...Value) (Value, error) {
		if len(args) < 1 {
			return nil, errors.New("logPrint requires at least a message argument")
		}
		// Accept any Value, convert to string
		msg := fmt.Sprintf("%v", args[0])
		level := "info"
		if len(args) > 1 {
			if lvl, ok := args[1].(Str); ok {
				level = string(lvl)
			}
		}

		// Optional: add more structured fields from further args
		fields := make(map[string]Value)
		for i := 2; i < len(args); i++ {
			if kv, ok := args[i].(TreeNode); ok {
				for k, v := range kv.GetAttributes() {
					fields[k] = v
				}
			} else {
				fields["arg"+strconv.Itoa(i)] = args[i]
			}
		}

		// Use your global or context logger
		logger := cfg.ChariotLogger

		switch level {
		case "debug":
			if len(fields) > 0 {
				logger.Debug(string(msg), ChariotValueToZapFields(fields)...)
			} else {
				logger.Debug(string(msg))
			}
			rt.WriteLog("DEBUG", msg)
		case "warn":
			if len(fields) > 0 {
				logger.Warn(string(msg), ChariotValueToZapFields(fields)...)
			} else {
				logger.Warn(string(msg))
			}
			rt.WriteLog("WARN", msg)
		case "error":
			if len(fields) > 0 {
				logger.Error(string(msg), ChariotValueToZapFields(fields)...)
			} else {
				logger.Error(string(msg))
			}
			rt.WriteLog("ERROR", msg)
		default:
			if len(fields) > 0 {
				logger.Info(string(msg), ChariotValueToZapFields(fields)...)
			} else {
				logger.Info(string(msg))
			}
			rt.WriteLog("INFO", msg)
		}

		return nil, nil
	})

	// Runtime information
	rt.Register("platform", func(args ...Value) (Value, error) {
		if len(args) != 0 {
			return nil, errors.New("platform accepts no arguments")
		}

		return Str(runtime.GOOS), nil
	})

	rt.Register("timestamp", func(args ...Value) (Value, error) {
		if len(args) != 0 {
			return nil, errors.New("timestamp accepts no arguments")
		}

		return Number(time.Now().Unix()), nil
	})

	rt.Register("timeFormat", func(args ...Value) (Value, error) {
		if len(args) != 2 {
			return nil, errors.New("timeFormat requires 2 arguments: timestamp and format string")
		}

		// Unwrap arguments
		for i, arg := range args {
			if tvar, ok := arg.(ScopeEntry); ok {
				args[i] = tvar.Value
			}
		}

		// Get timestamp
		ts, ok := args[0].(Number)
		if !ok {
			return nil, fmt.Errorf("timestamp must be a number, got %T", args[0])
		}

		// Get format string
		format, ok := args[1].(Str)
		if !ok {
			return nil, fmt.Errorf("format must be a string, got %T", args[1])
		}

		// Convert timestamp to time and format
		t := time.Unix(int64(ts), 0)
		formatted := t.Format(string(format))

		return Str(formatted), nil
	})

	// Program execution
	rt.Register("exit", func(args ...Value) (Value, error) {
		if len(args) > 1 {
			return nil, errors.New("exit accepts at most 1 argument: exit code")
		}

		// Unwrap arguments
		for i, arg := range args {
			if tvar, ok := arg.(ScopeEntry); ok {
				args[i] = tvar.Value
			}
		}

		code := 0
		if len(args) == 1 {
			exitCode, ok := args[0].(Number)
			if !ok {
				return nil, fmt.Errorf("exit code must be a number, got %T", args[0])
			}
			code = int(exitCode)
		}

		// Since we can't actually exit here (as it would terminate the whole program),
		// we'll return a special value that the runtime should handle
		return &ExitRequest{Code: code}, nil
	})

	rt.Register("sleep", func(args ...Value) (Value, error) {
		if len(args) != 1 {
			return nil, errors.New("sleep requires 1 argument: milliseconds")
		}

		// Unwrap arguments
		for i, arg := range args {
			if tvar, ok := arg.(ScopeEntry); ok {
				args[i] = tvar.Value
			}
		}

		ms, ok := args[0].(Number)
		if !ok {
			return nil, fmt.Errorf("milliseconds must be a number, got %T", args[0])
		}

		if ms < 0 {
			return nil, errors.New("sleep duration cannot be negative")
		}

		time.Sleep(time.Duration(ms) * time.Millisecond)

		return DBNull, nil
	})

	rt.Register("listen", func(args ...Value) (Value, error) {
		if len(args) < 2 {
			return nil, errors.New("listen requires at least 2 arguments: port and handler")
		}

		for i, arg := range args {
			if entry, ok := arg.(ScopeEntry); ok {
				args[i] = entry.Value
			}
		}

		port, err := parseListenerPort(args[0])
		if err != nil {
			return nil, err
		}

		handlerRef, err := resolveListenerHandler(args[1])
		if err != nil {
			return nil, err
		}

		opts := listenerOptions{
			Port:        port,
			HandlerName: handlerRef.name,
			HandlerFn:   handlerRef.fn,
		}

		if len(args) >= 3 {
			if err := applyListenerOptionValue(&opts, args[2]); err != nil {
				return nil, err
			}
		}

		listener, err := startRuntimeListener(rt, opts)
		if err != nil {
			return nil, err
		}

		return buildListenerResult(listener.opts), nil
	})

	rt.Register("listenNSQ", func(args ...Value) (Value, error) {
		if len(args) < 3 {
			return nil, errors.New("listenNSQ requires 3 arguments: topic, channel, handlers map")
		}
		for i, arg := range args {
			if entry, ok := arg.(ScopeEntry); ok {
				args[i] = entry.Value
			}
		}
		var optsArg Value
		if len(args) >= 4 {
			optsArg = args[3]
		}
		opts, err := buildNSQListenerOptions(args[0], args[1], args[2], optsArg)
		if err != nil {
			return nil, err
		}
		listener, err := startRuntimeNSQListener(rt, opts)
		if err != nil {
			return nil, err
		}
		return buildNSQListenerResult(listener.opts), nil
	})

	rt.Register("registerResponseTopic", func(args ...Value) (Value, error) {
		if len(args) != 2 {
			return nil, errors.New("registerResponseTopic requires 2 arguments: key and topic")
		}
		key, ok := args[0].(Str)
		if !ok {
			return nil, fmt.Errorf("response topic key must be a string, got %T", args[0])
		}
		topic, ok := args[1].(Str)
		if !ok {
			return nil, fmt.Errorf("response topic value must be a string, got %T", args[1])
		}
		if err := RegisterResponseTopic(string(key), string(topic)); err != nil {
			return nil, err
		}
		return Bool(true), nil
	})

	rt.Register("listResponseTopics", func(args ...Value) (Value, error) {
		if len(args) != 0 {
			return nil, errors.New("listResponseTopics accepts no arguments")
		}
		registered := ListResponseTopics()
		result := NewMap()
		for k, v := range registered {
			result.Set(k, Str(v))
		}
		return result, nil
	})

}

func ChariotValueToZapFields(fields map[string]Value) []zap.Field {
	var zapFields []zap.Field
	for k, v := range fields {
		// Always convert to string as the safest fallback
		var fieldValue interface{}

		switch val := v.(type) {
		case Str:
			fieldValue = string(val)
		case Number:
			fieldValue = float64(val)
		case Bool:
			fieldValue = bool(val)
		case *JSONNode:
			fieldValue = fmt.Sprintf("JSONNode(%s)", val.Name())
		case *FunctionValue:
			fieldValue = "FunctionValue"
		case *ArrayValue:
			fieldValue = fmt.Sprintf("ArrayValue(len=%d)", val.Length())
		case TreeNode:
			fieldValue = fmt.Sprintf("TreeNode(%s)", val.Name())
		case nil:
			fieldValue = "null"
		default:
			// Force string conversion for safety
			fieldValue = fmt.Sprintf("%T", v)
		}

		// Use zap.Any but with safe converted values
		zapFields = append(zapFields, zap.Any(k, fieldValue))
	}
	return zapFields
}

type listenerHandlerRef struct {
	name string
	fn   *FunctionValue
}

func parseListenerPort(value Value) (int, error) {
	switch v := value.(type) {
	case Number:
		port := int(v)
		if float64(port) != float64(v) {
			return 0, errors.New("port must be a whole number")
		}
		if port <= 0 || port > 65535 {
			return 0, fmt.Errorf("port must be between 1 and 65535")
		}
		return port, nil
	case Str:
		trimmed := strings.TrimSpace(string(v))
		if trimmed == "" {
			return 0, errors.New("port cannot be empty")
		}
		port, err := strconv.Atoi(trimmed)
		if err != nil {
			return 0, fmt.Errorf("invalid port '%s'", trimmed)
		}
		if port <= 0 || port > 65535 {
			return 0, fmt.Errorf("port must be between 1 and 65535")
		}
		return port, nil
	default:
		return 0, fmt.Errorf("port must be a number or string, got %T", value)
	}
}

func resolveListenerHandler(value Value) (listenerHandlerRef, error) {
	switch v := value.(type) {
	case Str:
		name := strings.TrimSpace(string(v))
		if name == "" {
			return listenerHandlerRef{}, errors.New("handler name cannot be empty")
		}
		return listenerHandlerRef{name: name}, nil
	case *FunctionValue:
		return listenerHandlerRef{fn: v}, nil
	default:
		return listenerHandlerRef{}, fmt.Errorf("handler must be a function or name, got %T", value)
	}
}

func applyListenerOptionValue(opts *listenerOptions, raw Value) error {
	switch v := raw.(type) {
	case Str:
		opts.OnExitProgram = strings.TrimSpace(string(v))
		return nil
	case *MapValue, *JSONNode:
		native := convertValueToNative(v)
		data, ok := native.(map[string]interface{})
		if !ok {
			return fmt.Errorf("listen options must be an object, got %T", native)
		}
		return applyListenerOptionsMap(opts, data)
	default:
		return fmt.Errorf("unsupported listen options type %T", raw)
	}
}

func applyListenerOptionsMap(opts *listenerOptions, data map[string]interface{}) error {
	for key, value := range data {
		switch strings.ToLower(key) {
		case "handler":
			if name, ok := value.(string); ok && strings.TrimSpace(name) != "" {
				opts.HandlerName = strings.TrimSpace(name)
				opts.HandlerFn = nil
			}
		case "onstart", "on_start":
			if name, ok := value.(string); ok {
				opts.OnStartProgram = strings.TrimSpace(name)
			}
		case "onexit", "on_exit":
			if name, ok := value.(string); ok {
				opts.OnExitProgram = strings.TrimSpace(name)
			}
		case "methods":
			methods, err := parseStringSlice(value)
			if err != nil {
				return fmt.Errorf("invalid methods option: %w", err)
			}
			opts.AllowedMethods = normalizeMethodSet(methods)
		case "basepath", "base_path":
			if base, ok := value.(string); ok {
				opts.BasePath = base
			}
		case "readtimeoutms", "read_timeout_ms":
			if dur, ok := parseDurationMillis(value); ok {
				opts.ReadTimeout = dur
			}
		case "writetimeoutms", "write_timeout_ms":
			if dur, ok := parseDurationMillis(value); ok {
				opts.WriteTimeout = dur
			}
		case "idletimeoutms", "idle_timeout_ms":
			if dur, ok := parseDurationMillis(value); ok {
				opts.IdleTimeout = dur
			}
		}
	}
	return nil
}

func parseStringSlice(source interface{}) ([]string, error) {
	switch v := source.(type) {
	case []interface{}:
		out := make([]string, 0, len(v))
		for _, item := range v {
			str := fmt.Sprintf("%v", item)
			if strings.TrimSpace(str) == "" {
				continue
			}
			out = append(out, str)
		}
		return out, nil
	case []string:
		return v, nil
	case string:
		if strings.TrimSpace(v) == "" {
			return nil, nil
		}
		return []string{v}, nil
	case nil:
		return nil, nil
	default:
		return nil, fmt.Errorf("expected array of strings, got %T", source)
	}
}

func parseDurationMillis(value interface{}) (time.Duration, bool) {
	switch v := value.(type) {
	case float64:
		return time.Duration(int64(v)) * time.Millisecond, true
	case int64:
		return time.Duration(v) * time.Millisecond, true
	case int:
		return time.Duration(v) * time.Millisecond, true
	case string:
		trimmed := strings.TrimSpace(v)
		if trimmed == "" {
			return 0, false
		}
		ms, err := strconv.Atoi(trimmed)
		if err != nil {
			return 0, false
		}
		return time.Duration(ms) * time.Millisecond, true
	default:
		return 0, false
	}
}

func normalizeMethodSet(methods []string) map[string]struct{} {
	if len(methods) == 0 {
		return nil
	}
	set := make(map[string]struct{}, len(methods))
	for _, method := range methods {
		trimmed := strings.ToUpper(strings.TrimSpace(method))
		if trimmed == "" {
			continue
		}
		set[trimmed] = struct{}{}
	}
	return set
}

func buildListenerResult(opts listenerOptions) *MapValue {
	values := map[string]Value{
		"status": Str("listening"),
		"port":   Number(float64(opts.Port)),
	}
	handler := opts.HandlerName
	if handler == "" && opts.HandlerFn != nil {
		handler = "<inline>"
	}
	if handler != "" {
		values["handler"] = Str(handler)
	}
	if normalized := normalizeBasePath(opts.BasePath); normalized != "" {
		values["basePath"] = Str(normalized)
	}
	if len(opts.AllowedMethods) > 0 {
		arr := NewArray()
		for method := range opts.AllowedMethods {
			arr.Append(Str(strings.ToUpper(method)))
		}
		values["methods"] = arr
	}
	return NewMapWithValues(values)
}
