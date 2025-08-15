package chariot

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
	"unicode"
)

// RegisterString registers all string-related functions
func RegisterString(rt *Runtime) {
	// String creation and basics
	rt.Register("concat", func(args ...Value) (Value, error) {
		var builder strings.Builder

		// Unwrap arguments
		for i, arg := range args {
			if tvar, ok := arg.(ScopeEntry); ok {
				args[i] = tvar.Value
			}
		}

		for _, arg := range args {
			// Convert any type to string
			builder.WriteString(fmt.Sprintf("%v", arg))
		}

		return Str(builder.String()), nil
	})

	rt.Register("string", func(args ...Value) (Value, error) {
		if len(args) != 1 {
			return nil, errors.New("string requires 1 argument")
		}

		// Unwrap argument
		if tvar, ok := args[0].(ScopeEntry); ok {
			args[0] = tvar.Value
		}

		return Str(fmt.Sprintf("%v", args[0])), nil
	})

	rt.Register("format", func(args ...Value) (Value, error) {
		if len(args) < 1 {
			return nil, errors.New("format requires at least 1 argument")
		}

		// Unwrap first argument as format string
		if tvar, ok := args[0].(ScopeEntry); ok {
			args[0] = tvar.Value
		}

		format, ok := args[0].(Str)
		if !ok {
			return nil, fmt.Errorf("format string must be a string, got %T", args[0])
		}

		// Convert remaining args to interface{} for fmt.Sprintf
		fmtArgs := make([]interface{}, len(args)-1)
		for i, arg := range args[1:] {
			switch v := arg.(type) {
			case Str:
				fmtArgs[i] = string(v)
			case Number:
				fmtArgs[i] = float64(v) // Convert Number to float64 for fmt.Sprintf
			case Bool:
				fmtArgs[i] = bool(v) // Convert Bool to bool for fmt.Sprintf
			default:
				fmtArgs[i] = arg
			}
		}

		result := fmt.Sprintf(string(format), fmtArgs...)
		return Str(result), nil
	})

	// Alias for format
	rt.Register("sprintf", rt.funcs["format"])

	rt.Register("char", func(args ...Value) (Value, error) {
		if len(args) != 2 {
			return nil, errors.New("char requires 2 arguments: string and position")
		}

		// Unwrap arguments
		for i, arg := range args {
			if tvar, ok := arg.(ScopeEntry); ok {
				args[i] = tvar.Value
			}
		}

		str, ok := args[0].(Str)
		if !ok {
			return nil, fmt.Errorf("expected string, got %T", args[0])
		}

		pos, ok := args[1].(Number)
		if !ok {
			return nil, fmt.Errorf("position must be a number, got %T", args[1])
		}

		if pos < 0 || int(pos) >= len(str) {
			return nil, fmt.Errorf("position %d out of bounds for string of length %d", int(pos), len(str))
		}

		return Str(string(str[int(pos)])), nil
	})

	rt.Register("ascii", func(args ...Value) (Value, error) {
		if len(args) != 1 {
			return nil, errors.New("ascii requires 1 argument")
		}

		// Unwrap argument
		for i, arg := range args {
			if tvar, ok := arg.(ScopeEntry); ok {
				args[i] = tvar.Value
			}
		}

		str, ok := args[0].(Str)
		if !ok {
			return nil, fmt.Errorf("expected string, got %T", args[0])
		}

		if len(str) == 0 {
			return nil, errors.New("cannot get ASCII code of empty string")
		}

		return Number(str[0]), nil
	})

	// In dispatchers.go or string_funcs.go
	rt.Register("charAt", func(args ...Value) (Value, error) {
		if len(args) != 2 {
			return nil, errors.New("charAt requires 2 arguments")
		}
		// Unwrap arguments
		for i, arg := range args {
			if tvar, ok := arg.(ScopeEntry); ok {
				args[i] = tvar.Value
			}
		}

		return stringGetAt(args...)
	})

	// String manipulation
	rt.Register("lower", func(args ...Value) (Value, error) {
		if len(args) != 1 {
			return nil, errors.New("lower requires 1 argument")
		}

		// Unwrap argument
		for i, arg := range args {
			if tvar, ok := arg.(ScopeEntry); ok {
				args[i] = tvar.Value
			}
		}

		str, ok := args[0].(Str)
		if !ok {
			return nil, fmt.Errorf("expected string, got %T", args[0])
		}

		return Str(strings.ToLower(string(str))), nil
	})

	rt.Register("upper", func(args ...Value) (Value, error) {
		if len(args) != 1 {
			return nil, errors.New("upper requires 1 argument")
		}

		// Unwrap argument
		for i, arg := range args {
			if tvar, ok := arg.(ScopeEntry); ok {
				args[i] = tvar.Value
			}
		}

		str, ok := args[0].(Str)
		if !ok {
			return nil, fmt.Errorf("expected string, got %T", args[0])
		}

		return Str(strings.ToUpper(string(str))), nil
	})

	rt.Register("trimLeft", func(args ...Value) (Value, error) {
		if len(args) != 1 {
			return nil, errors.New("trimRight requires 1 argument")
		}

		// Unwrap argument
		for i, arg := range args {
			if tvar, ok := arg.(ScopeEntry); ok {
				args[i] = tvar.Value
			}
		}

		str, ok := args[0].(Str)
		if !ok {
			return nil, fmt.Errorf("expected string, got %T", args[0])
		}

		return Str(strings.TrimRightFunc(string(str), unicode.IsSpace)), nil
	})

	rt.Register("trimRight", func(args ...Value) (Value, error) {
		if len(args) != 1 {
			return nil, errors.New("trimRight requires 1 argument")
		}

		// Unwrap argument
		for i, arg := range args {
			if tvar, ok := arg.(ScopeEntry); ok {
				args[i] = tvar.Value
			}
		}

		str, ok := args[0].(Str)
		if !ok {
			return nil, fmt.Errorf("expected string, got %T", args[0])
		}

		return Str(strings.TrimRightFunc(string(str), unicode.IsSpace)), nil
	})

	rt.Register("trim", func(args ...Value) (Value, error) {
		if len(args) < 1 || len(args) > 2 {
			return nil, errors.New("trim requires 1 or 2 arguments")
		}
		// Unwrap argument
		for i, arg := range args {
			if tvar, ok := arg.(ScopeEntry); ok {
				args[i] = tvar.Value
			}
		}
		str, ok := args[0].(Str)
		if !ok {
			return nil, fmt.Errorf("expected string, got %T", args[0])
		}
		if len(args) == 1 {
			// Default to trimming whitespace
			return Str(strings.TrimSpace(string(str))), nil
		}
		// Custom trim characters
		chars, ok := args[1].(Str)
		if !ok {
			return nil, fmt.Errorf("expected string, got %T", args[1])
		}
		return Str(strings.Trim(string(str), string(chars))), nil
	})

	rt.Register("replace", func(args ...Value) (Value, error) {
		if len(args) != 3 && len(args) != 4 {
			return nil, errors.New("replace requires 3 or 4 arguments: string, old, new, [count]")
		}

		// Unwrap arguments
		for i, arg := range args {
			if tvar, ok := arg.(ScopeEntry); ok {
				args[i] = tvar.Value
			}
		}

		// Get the string
		str, ok := args[0].(Str)
		if !ok {
			return nil, fmt.Errorf("expected string, got %T", args[0])
		}

		// Get old and new substrings
		oldStr, ok := args[1].(Str)
		if !ok {
			return nil, fmt.Errorf("old substring must be a string, got %T", args[1])
		}

		newStr, ok := args[2].(Str)
		if !ok {
			return nil, fmt.Errorf("new substring must be a string, got %T", args[2])
		}

		// Get optional count
		count := -1 // Default is replace all
		if len(args) == 4 {
			countVal, ok := args[3].(Number)
			if !ok {
				return nil, fmt.Errorf("count must be a number, got %T", args[3])
			}
			count = int(countVal)
		}

		// Perform replacement
		result := strings.Replace(string(str), string(oldStr), string(newStr), count)
		return Str(result), nil
	})

	rt.Register("substring", func(args ...Value) (Value, error) {
		if !(len(args) >= 2 && len(args) <= 3) {
			return nil, errors.New("substring requires 2 or 3 arguments: string, start, length")
		}

		// Unwrap arguments
		for i, arg := range args {
			if tvar, ok := arg.(ScopeEntry); ok {
				args[i] = tvar.Value
			}
		}

		str, ok := args[0].(Str)
		if !ok {
			return nil, fmt.Errorf("expected string, got %T", args[0])
		}

		start, ok := args[1].(Number)
		if !ok {
			return nil, fmt.Errorf("start must be a number, got %T", args[1])
		}

		// validate legal start index
		if start < 0 {
			return nil, fmt.Errorf("start index must be non-negative")
		}

		if len(args) == 2 {
			// If only start is provided, return substring from start to end
			return Str(str[int(start):]), nil
		}
		length, ok := args[2].(Number)
		if !ok {
			return nil, fmt.Errorf("length must be a number, got %T", args[2])
		}
		// validate legal length
		if length < 0 {
			return nil, fmt.Errorf("length must be non-negative")
		}

		// Handle bounds
		strLen := len(str)
		startIdx := int(start)
		if startIdx < 0 {
			startIdx = 0
		} else if startIdx >= strLen {
			return Str(""), nil
		}

		endIdx := startIdx + int(length)
		if endIdx > strLen {
			endIdx = strLen
		}

		return Str(str[startIdx:endIdx]), nil
	})

	rt.Register("substr", func(args ...Value) (Value, error) {
		return rt.funcs["substring"](args...)
	})

	rt.Register("right", func(args ...Value) (Value, error) {
		if len(args) != 2 {
			return nil, errors.New("right requires 2 arguments: string and count")
		}

		// Unwrap arguments
		for i, arg := range args {
			if tvar, ok := arg.(ScopeEntry); ok {
				args[i] = tvar.Value
			}
		}

		str, ok := args[0].(Str)
		if !ok {
			return nil, fmt.Errorf("expected string, got %T", args[0])
		}

		count, ok := args[1].(Number)
		if !ok {
			return nil, fmt.Errorf("count must be a number, got %T", args[1])
		}

		n := int(count)
		if n <= 0 {
			return Str(""), nil
		}

		strLen := len(str)
		if n >= strLen {
			return str, nil
		}

		return Str(str[strLen-n:]), nil
	})

	rt.Register("strlen", func(args ...Value) (Value, error) {
		return rt.funcs["length"](args...)
	})

	rt.Register("digits", func(args ...Value) (Value, error) {
		if len(args) != 1 {
			return nil, errors.New("digits requires 1 argument")
		}

		// Unwrap argument
		for i, arg := range args {
			if tvar, ok := arg.(ScopeEntry); ok {
				args[i] = tvar.Value
			}
		}

		str, ok := args[0].(Str)
		if !ok {
			return nil, fmt.Errorf("expected string, got %T", args[0])
		}

		var result strings.Builder
		for _, ch := range string(str) {
			if unicode.IsDigit(ch) {
				result.WriteRune(ch)
			}
		}

		return Str(result.String()), nil
	})

	rt.Register("hasPrefix", func(args ...Value) (Value, error) {
		if len(args) != 2 {
			return nil, errors.New("hasPrefix requires 2 arguments: string and prefix")
		}

		// Unwrap arguments
		for i, arg := range args {
			if tvar, ok := arg.(ScopeEntry); ok {
				args[i] = tvar.Value
			}
		}

		str, ok := args[0].(Str)
		if !ok {
			return nil, fmt.Errorf("expected string, got %T", args[0])
		}

		prefix, ok := args[1].(Str)
		if !ok {
			return nil, fmt.Errorf("prefix must be a string, got %T", args[1])
		}

		return Bool(strings.HasPrefix(string(str), string(prefix))), nil
	})

	rt.Register("hasSuffix", func(args ...Value) (Value, error) {
		if len(args) != 2 {
			return nil, errors.New("hasSuffix requires 2 arguments: string and suffix")
		}

		// Unwrap arguments
		for i, arg := range args {
			if tvar, ok := arg.(ScopeEntry); ok {
				args[i] = tvar.Value
			}
		}

		str, ok := args[0].(Str)
		if !ok {
			return nil, fmt.Errorf("expected string, got %T", args[0])
		}

		suffix, ok := args[1].(Str)
		if !ok {
			return nil, fmt.Errorf("suffix must be a string, got %T", args[1])
		}

		return Bool(strings.HasSuffix(string(str), string(suffix))), nil
	})

	rt.Register("lastPos", func(args ...Value) (Value, error) {
		if len(args) != 2 {
			return nil, errors.New("lastPos requires 2 arguments: string and substring")
		}

		// Unwrap arguments
		for i, arg := range args {
			if tvar, ok := arg.(ScopeEntry); ok {
				args[i] = tvar.Value
			}
		}

		str, ok := args[0].(Str)
		if !ok {
			return nil, fmt.Errorf("expected string, got %T", args[0])
		}

		substr, ok := args[1].(Str)
		if !ok {
			return nil, fmt.Errorf("substring must be a string, got %T", args[1])
		}

		position := strings.LastIndex(string(str), string(substr))
		return Number(position), nil
	})

	rt.Register("atPos", func(args ...Value) (Value, error) {
		if len(args) != 2 {
			return nil, errors.New("atPos requires 2 arguments: string and position")
		}

		// Unwrap arguments
		for i, arg := range args {
			if tvar, ok := arg.(ScopeEntry); ok {
				args[i] = tvar.Value
			}
		}

		str, ok := args[0].(Str)
		if !ok {
			return nil, fmt.Errorf("expected string, got %T", args[0])
		}

		pos, ok := args[1].(Number)
		if !ok {
			return nil, fmt.Errorf("position must be a number, got %T", args[1])
		}

		if pos < 0 || int(pos) >= len(str) {
			return nil, fmt.Errorf("position %d out of bounds for string of length %d", int(pos), len(str))
		}

		return Str(string(str[int(pos)])), nil
	})

	rt.Register("occurs", func(args ...Value) (Value, error) {
		if len(args) != 2 {
			return nil, errors.New("occurs requires 2 arguments: string and substring")
		}

		// Unwrap arguments
		for i, arg := range args {
			if tvar, ok := arg.(ScopeEntry); ok {
				args[i] = tvar.Value
			}
		}

		str, ok := args[0].(Str)
		if !ok {
			return nil, fmt.Errorf("expected string, got %T", args[0])
		}

		substr, ok := args[1].(Str)
		if !ok {
			return nil, fmt.Errorf("substring must be a string, got %T", args[1])
		}

		// Count occurrences
		if substr == "" {
			// Special case for empty string
			return Number(len(str) + 1), nil
		}

		count := 0
		offset := 0
		for {
			i := strings.Index(string(str)[offset:], string(substr))
			if i == -1 {
				break
			}
			count++
			offset += i + len(substr)
			if offset >= len(str) {
				break
			}
		}

		return Number(count), nil
	})

	// String collections
	rt.Register("split", func(args ...Value) (Value, error) {
		if len(args) != 2 {
			return nil, errors.New("split requires 2 arguments: string and delimiter")
		}

		// Unwrap arguments
		for i, arg := range args {
			if tvar, ok := arg.(ScopeEntry); ok {
				args[i] = tvar.Value
			}
		}

		str, ok := args[0].(Str)
		if !ok {
			return nil, fmt.Errorf("expected string, got %T", args[0])
		}

		delim, ok := args[1].(Str)
		if !ok {
			return nil, fmt.Errorf("delimiter must be a string, got %T", args[1])
		}

		parts := strings.Split(string(str), string(delim))

		// Convert []string to *ArrayValue
		arr := NewArray()
		for _, part := range parts {
			arr.Append(Str(part))
		}
		return arr, nil
	})

	rt.Register("join", func(args ...Value) (Value, error) {
		if len(args) != 2 {
			return nil, errors.New("join requires 2 arguments: array and delimiter")
		}

		// Unwrap arguments
		for i, arg := range args {
			if tvar, ok := arg.(ScopeEntry); ok {
				args[i] = tvar.Value
			}
		}

		arr, ok := args[0].(*ArrayValue)
		if !ok {
			return nil, fmt.Errorf("first argument must be an array, got %T", args[0])
		}

		delim, ok := args[1].(Str)
		if !ok {
			return nil, fmt.Errorf("delimiter must be a string, got %T", args[1])
		}

		parts := make([]string, arr.Length())
		for i := 0; i < arr.Length(); i++ {
			// Convert each element to string using fmt.Sprintf for flexibility
			parts[i] = fmt.Sprintf("%v", arr.Get(i))
		}

		return Str(strings.Join(parts, string(delim))), nil
	})

	rt.Register("append", func(args ...Value) (Value, error) {
		return rt.funcs["concat"](args...)
	})

	// padLeft function
	rt.Register("padLeft", func(args ...Value) (Value, error) {
		if len(args) != 3 {
			return nil, errors.New("padLeft requires 3 arguments: string, total length, and padding character")
		}

		// Unwrap arguments
		for i, arg := range args {
			if tvar, ok := arg.(ScopeEntry); ok {
				args[i] = tvar.Value
			}
		}

		str, ok := args[0].(Str)
		if !ok {
			return nil, fmt.Errorf("expected string, got %T", args[0])
		}

		totalLen, ok := args[1].(Number)
		if !ok {
			return nil, fmt.Errorf("total length must be a number, got %T", args[1])
		}

		padChar, ok := args[2].(Str)
		if !ok {
			return nil, fmt.Errorf("padding character must be a string, got %T", args[2])
		}

		if len(str) >= int(totalLen) {
			return str, nil
		}

		padding := strings.Repeat(string(padChar), int(totalLen)-len(str))
		return Str(padding + string(str)), nil
	})

	// padRight function
	rt.Register("padRight", func(args ...Value) (Value, error) {
		if len(args) != 3 {
			return nil, errors.New("padRight requires 3 arguments: string, total length, and padding character")
		}

		// Unwrap arguments
		for i, arg := range args {
			if tvar, ok := arg.(ScopeEntry); ok {
				args[i] = tvar.Value
			}
		}

		str, ok := args[0].(Str)
		if !ok {
			return nil, fmt.Errorf("expected string, got %T", args[0])
		}

		totalLen, ok := args[1].(Number)
		if !ok {
			return nil, fmt.Errorf("total length must be a number, got %T", args[1])
		}

		padChar, ok := args[2].(Str)
		if !ok {
			return nil, fmt.Errorf("padding character must be a string, got %T", args[2])
		}

		if len(str) >= int(totalLen) {
			return str, nil
		}

		padding := strings.Repeat(string(padChar), int(totalLen)-len(str))
		return Str(string(str) + padding), nil
	})

	// String interpolation function
	rt.Register("interpolate", func(args ...Value) (Value, error) {
		if len(args) != 1 {
			return nil, errors.New("interpolate requires 1 argument: template string")
		}

		// Unwrap arguments
		for i, arg := range args {
			if tvar, ok := arg.(ScopeEntry); ok {
				args[i] = tvar.Value
			}
		}

		template, ok := args[0].(Str)
		if !ok {
			return nil, fmt.Errorf("template must be a string, got %T", args[0])
		}

		result, err := interpolateString(rt, string(template))
		if err != nil {
			return nil, err
		}

		return Str(result), nil
	})
}

// Helper function for string interpolation
func interpolateString(rt *Runtime, template string) (string, error) {
	result := template

	// Find all ${variable} patterns
	re := regexp.MustCompile(`\$\{([a-zA-Z_][a-zA-Z0-9_]*)\}`)
	matches := re.FindAllStringSubmatch(template, -1)

	for _, match := range matches {
		placeholder := match[0] // Full match: ${varname}
		varName := match[1]     // Variable name: varname

		// Look up the variable
		entry, found := rt.FindVariable(varName)
		if !found {
			return "", fmt.Errorf("variable '%s' not found in interpolation", varName)
		}

		// Convert value to string
		var replacement string
		switch v := entry.Value.(type) {
		case Str:
			replacement = string(v)
		case Number:
			replacement = fmt.Sprintf("%g", float64(v))
		case Bool:
			if bool(v) {
				replacement = "true"
			} else {
				replacement = "false"
			}
		case nil:
			replacement = ""
		default:
			replacement = fmt.Sprintf("%v", v)
		}

		// Replace in result string
		result = strings.Replace(result, placeholder, replacement, -1)
	}

	return result, nil
}

// stringGetAt - for string character access
func stringGetAt(args ...Value) (Value, error) {

	if len(args) != 2 {
		return nil, errors.New("charAt requires 2 arguments: string and index")
	}
	// Unwrap arguments
	for i, arg := range args {
		if tvar, ok := arg.(ScopeEntry); ok {
			args[i] = tvar.Value
		}
	}

	str, ok := args[0].(Str)
	if !ok {
		return nil, fmt.Errorf("expected string, got %T", args[0])
	}

	num, ok := args[1].(Number)
	if !ok {
		return nil, fmt.Errorf("index must be a number, got %T", args[1])
	}

	index := int(num)
	runes := []rune(string(str))

	// Return DBNull for invalid indices
	if index < 0 || index >= len(runes) {
		return DBNull, nil
	}

	return Str(string(runes[index])), nil
}
