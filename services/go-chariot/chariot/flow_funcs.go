package chariot

// RegisterFlow registers all flow control functions
func RegisterFlow(rt *Runtime) {
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
