package chariot

import (
	"errors"
	"fmt"
	"os"
	"runtime"
	"strconv"
	"time"

	cfg "github.com/bhouse1273/go-chariot/configs"
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
		case "warn":
			if len(fields) > 0 {
				logger.Warn(string(msg), ChariotValueToZapFields(fields)...)
			} else {
				logger.Warn(string(msg))
			}
		case "error":
			if len(fields) > 0 {
				logger.Error(string(msg), ChariotValueToZapFields(fields)...)
			} else {
				logger.Error(string(msg))
			}
		default:
			if len(fields) > 0 {
				logger.Info(string(msg), ChariotValueToZapFields(fields)...)
			} else {
				logger.Info(string(msg))
			}
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
		if len(args) < 1 {
			return nil, errors.New("listen requires at least a port argument")
		}

		// Unwrap arguments
		for i, arg := range args {
			if tvar, ok := arg.(ScopeEntry); ok {
				args[i] = tvar.Value
			}
		}

		port, ok := args[0].(Number)
		if !ok {
			return nil, errors.New("port must be a number")
		}
		onstart := ""
		onexit := ""
		if len(args) > 1 {
			if s, ok := args[1].(Str); ok {
				onstart = string(s)
			}
		}
		if len(args) > 2 {
			if s, ok := args[2].(Str); ok {
				onexit = string(s)
			}
		}
		go func() {
			// Run onstart program
			if onstart != "" {
				rt.RunProgram(onstart, int(port))
			}
			// Listen on port and handle requests...
			// On shutdown, run onexit program
			if onexit != "" {
				rt.RunProgram(onexit, int(port))
			}
		}()
		return nil, nil
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
