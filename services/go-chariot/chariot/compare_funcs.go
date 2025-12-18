package chariot

import (
	"errors"
	"fmt"
)

// RegisterCompares registers all comparison functions
func RegisterCompares(rt *Runtime) {
	rt.Register("equal", func(args ...Value) (Value, error) {
		if len(args) < 2 {
			return nil, errors.New("equal requires at least 2 arguments")
		}

		// Unwrap ScopeEntry args
		processed := make([]Value, len(args))
		for i, arg := range args {
			if targ, ok := arg.(ScopeEntry); ok {
				arg = targ.Value
			}
			processed[i] = arg
		}

		compareValues := func(left, right Value) (bool, error) {
			if left == DBNull && right == DBNull {
				return true, nil
			}
			if left == DBNull || right == DBNull {
				return false, nil
			}

			switch v1 := left.(type) {
			case Number:
				if v2, ok := right.(Number); ok {
					return v1 == v2, nil
				}
			case string:
				if v2, ok := right.(Str); ok {
					return v1 == string(v2), nil
				}
				return false, nil
			case Str:
				if v2, ok := right.(string); ok {
					return v1 == Str(v2), nil
				}
				if v2, ok := right.(Str); ok {
					return v1 == v2, nil
				}
			case Bool:
				if v2, ok := right.(Bool); ok {
					return v1 == v2, nil
				}
			}

			return false, nil
		}

		baseline := processed[0]
		for i := 1; i < len(processed); i++ {
			equal, err := compareValues(baseline, processed[i])
			if err != nil {
				return nil, err
			}
			if !equal {
				return Bool(false), nil
			}
		}

		return Bool(true), nil
	})
	rt.Register("equals", rt.funcs["equal"]) // Alias for equal

	rt.Register("unequal", func(args ...Value) (Value, error) {
		if len(args) < 2 {
			return nil, errors.New("unequal requires at least 2 arguments")
		}

		// Unwrap ScopeEntry args
		processed := make([]Value, len(args))
		for i, arg := range args {
			if targ, ok := arg.(ScopeEntry); ok {
				arg = targ.Value
			}
			processed[i] = arg
		}

		equalClosure := rt.funcs["equal"]
		for i := 0; i < len(processed); i++ {
			for j := i + 1; j < len(processed); j++ {
				result, err := equalClosure(processed[i], processed[j])
				if err != nil {
					return nil, err
				}
				if boolResult, ok := result.(Bool); ok {
					if bool(boolResult) {
						return Bool(false), nil
					}
				} else {
					return nil, fmt.Errorf("equal returned non-bool %T", result)
				}
			}
		}

		return Bool(true), nil
	})

	rt.Register("bigger", func(args ...Value) (Value, error) {
		if len(args) != 2 {
			return nil, errors.New("bigger requires 2 arguments")
		}

		// Unwrap ScopeEntry args
		if targ, ok := args[0].(ScopeEntry); ok {
			args[0] = targ.Value
		}
		if targ, ok := args[1].(ScopeEntry); ok {
			args[1] = targ.Value
		}

		// Handle numeric comparison
		num1, ok1 := args[0].(Number)
		num2, ok2 := args[1].(Number)

		if ok1 && ok2 {
			return Bool(num1 > num2), nil
		}

		// Handle string comparison
		str1, ok1 := args[0].(Str)
		str2, ok2 := args[1].(Str)

		if ok1 && ok2 {
			return Bool(str1 > str2), nil
		}

		return nil, fmt.Errorf("bigger requires comparable types (numbers or strings)")
	})

	rt.Register("smaller", func(args ...Value) (Value, error) {
		if len(args) != 2 {
			return nil, errors.New("smaller requires 2 arguments")
		}

		// Unwrap ScopeEntry args
		if targ, ok := args[0].(ScopeEntry); ok {
			args[0] = targ.Value
		}
		if targ, ok := args[1].(ScopeEntry); ok {
			args[1] = targ.Value
		}

		// Handle numeric comparison
		num1, ok1 := args[0].(Number)
		num2, ok2 := args[1].(Number)

		if ok1 && ok2 {
			return Bool(num1 < num2), nil
		}

		// Handle string comparison
		str1, ok1 := args[0].(Str)
		str2, ok2 := args[1].(Str)

		if ok1 && ok2 {
			return Bool(str1 < str2), nil
		}

		return nil, fmt.Errorf("smaller requires comparable types (numbers or strings)")
	})

	rt.Register("biggerEq", func(args ...Value) (Value, error) {
		if len(args) != 2 {
			return nil, errors.New("biggerEq requires 2 arguments")
		}

		// Try smaller first
		result, err := rt.funcs["smaller"](args...)
		if err != nil {
			return nil, err
		}
		if result == Bool(true) {
			// If smaller is true, then biggerEq is false
			return Bool(false), nil
		}

		// If smaller is false, then we can test for equality
		equalResult, err := rt.funcs["equal"](args...)
		if err != nil {
			return nil, err
		}
		if equalResult == Bool(true) {
			// If equal, then biggerEq is true
			return Bool(true), nil
		}

		// if smaller was false and equal was false, then bigger is true
		return Bool(true), nil
	})

	rt.Register("smallerEq", func(args ...Value) (Value, error) {
		if len(args) != 2 {
			return nil, errors.New("smallerEq requires 2 arguments")
		}

		// Try bigger first
		result, err := rt.funcs["bigger"](args...)
		if err != nil {
			return nil, err
		}
		if result == Bool(true) {
			// If bigger is true, then smallerEq is false
			return Bool(false), nil
		}
		// If bigger is false, then we can test for equality
		equalResult, err := rt.funcs["equal"](args...)
		if err != nil {
			return nil, err
		}
		if equalResult == Bool(true) {
			// If equal, then smallerEq is true
			return Bool(true), nil
		}
		// if bigger was false and equal was false, then smaller is true
		return Bool(true), nil
	})

	rt.Register("and", func(args ...Value) (Value, error) {
		if len(args) == 0 {
			return nil, errors.New("missing argument(s) -- expected 1 or more booleans")
		}

		// Unwrap ScopeEntry args
		for i, arg := range args {
			if targ, ok := arg.(ScopeEntry); ok {
				args[i] = targ.Value
			}
		}
		// Handle null values
		for _, arg := range args {
			if arg == DBNull {
				return Bool(false), nil // AND with null is false
			}
			if _, ok := arg.(Bool); !ok {
				return nil, fmt.Errorf("type mismatch: expected boolean, got %T", arg)
			}
		}
		// If no arguments are provided, return true (AND of no values is true)
		if len(args) == 0 {
			return Bool(true), nil
		}
		// If any argument is false, return false immediately
		// If all arguments are true, return true

		// Evaluate each argument and perform AND with short-circuit
		result := true
		for _, arg := range args {
			b, ok := arg.(Bool)
			if !ok {
				return nil, fmt.Errorf("type mismatch: expected boolean, got %T", arg)
			}

			result = result && bool(b)
			if !result {
				// Short-circuit on first false value
				break
			}
		}

		return Bool(result), nil
	})

	rt.Register("or", func(args ...Value) (Value, error) {
		if len(args) == 0 {
			return nil, errors.New("missing argument(s) -- expected 1 or more booleans")
		}

		// Unwrap ScopeEntry args
		for i, arg := range args {
			if targ, ok := arg.(ScopeEntry); ok {
				args[i] = targ.Value
			}
		}
		// Handle null values
		for _, arg := range args {
			if arg == DBNull {
				return Bool(true), nil // OR with null is true
			}
			if _, ok := arg.(Bool); !ok {
				return nil, fmt.Errorf("type mismatch: expected boolean, got %T", arg)
			}
		}
		// If no arguments are provided, return false (OR of no values is false)
		if len(args) == 0 {
			return Bool(false), nil
		}

		// Evaluate each argument and perform OR with short-circuit
		result := false
		for _, arg := range args {
			b, ok := arg.(Bool)
			if !ok {
				return nil, fmt.Errorf("type mismatch: expected boolean, got %T", arg)
			}

			result = result || bool(b)
			if result {
				// Short-circuit on first true value
				break
			}
		}

		return Bool(result), nil
	})

	rt.Register("not", func(args ...Value) (Value, error) {
		if len(args) == 0 {
			return nil, errors.New("not requires at least 1 argument")
		}

		// Unwrap ScopeEntry args
		for i, arg := range args {
			if targ, ok := arg.(ScopeEntry); ok {
				args[i] = targ.Value
			}
		}

		for _, arg := range args {
			if arg == DBNull {
				continue // DBNull counts as false, so it satisfies the NOT condition
			}

			b, ok := arg.(Bool)
			if !ok {
				return nil, fmt.Errorf("type mismatch: expected boolean, got %T", arg)
			}

			if bool(b) {
				return Bool(false), nil // any true operand makes the NOT false
			}
		}

		return Bool(true), nil
	})

	rt.Register("iif", func(args ...Value) (Value, error) {
		if len(args) != 3 {
			return nil, errors.New("iif requires 3 arguments")
		}

		// Unwrap ScopeEntry args
		if targ, ok := args[0].(ScopeEntry); ok {
			args[0] = targ.Value
		}
		if targ, ok := args[1].(ScopeEntry); ok {
			args[1] = targ.Value
		}
		if targ, ok := args[2].(ScopeEntry); ok {
			args[2] = targ.Value
		}

		// First argument should be a boolean condition
		cond, ok := args[0].(Bool)
		if !ok {
			return nil, fmt.Errorf("type mismatch: expected boolean condition, got %T", args[0])
		}

		if bool(cond) {
			return args[1], nil // Return second argument if condition is true
		}
		return args[2], nil // Return third argument if condition is false
	})
}
