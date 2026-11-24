package chariot

import "fmt"

// RegisterFlow registers all flow control functions
func RegisterFlow(rt *Runtime) {
	rt.Register("error", func(args ...Value) (Value, error) {
		if len(args) != 1 {
			return nil, fmt.Errorf("error requires 1 argument: message")
		}
		value := args[0]
		if entry, ok := value.(ScopeEntry); ok {
			value = entry.Value
		}
		message, ok := value.(Str)
		if !ok {
			return nil, fmt.Errorf("error message must be a string")
		}
		return nil, fmt.Errorf("%s", message)
	})
	// Flow control functions - most are handled directly by the parser
	rt.Register("break", func(args ...Value) (Value, error) {
		return nil, &BreakError{}
	})
	rt.Register("return", func(args ...Value) (Value, error) {
		if len(args) == 0 {
			return DBNull, &ReturnError{Value: DBNull}
		}
		return args[0], &ReturnError{Value: args[0]}
	})
}
