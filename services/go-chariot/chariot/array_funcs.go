package chariot

import (
	"errors"
	"fmt"

	cfg "github.com/bhouse1273/chariot-ecosystem/services/go-chariot/configs"
)

// RegisterArray registers all array-related functions
func RegisterArray(rt *Runtime) {
	// Array creation function
	rt.Register("array", func(args ...Value) (Value, error) {
		if cfg.ChariotConfig.Verbose {
			fmt.Printf("DEBUG: array() called with %d args: %v\n", len(args), args)
		}

		// Create a new array using your NewArray() function
		arr := NewArray()

		// Add all arguments to the array
		for _, value := range args {
			arr.Append(value)
		}

		return arr, nil
	})

	// Array modification
	rt.Register("addTo", func(args ...Value) (Value, error) {
		if cfg.ChariotConfig.Verbose {
			fmt.Printf("DEBUG: addToFunc called with '%v'", args)
		}

		if len(args) < 2 {
			return nil, errors.New("addTo requires at least 2 arguments: array and value(s)")
		}

		// Get the array
		arr, ok := args[0].(*ArrayValue)
		if !ok {
			return nil, fmt.Errorf("expected array, got %T", args[0])
		}

		// Add each value to the array
		for _, value := range args[1:] {
			arr.Append(value)
		}

		return arr, nil
	})

	rt.Register("removeAt", func(args ...Value) (Value, error) {
		if cfg.ChariotConfig.Verbose {
			fmt.Printf("DEBUG: removeAtFunc called with '%v'", args)
		}

		if len(args) != 2 {
			return nil, errors.New("removeAt requires 2 arguments: array and index")
		}

		// Get the array
		arr, ok := args[0].(*ArrayValue)
		if !ok {
			return nil, fmt.Errorf("expected array, got %T", args[0])
		}

		// Get the index
		idx, ok := args[1].(Number)
		if !ok {
			return nil, fmt.Errorf("index must be a number, got %T", args[1])
		}

		// Check bounds
		if int(idx) < 0 || int(idx) >= arr.Length() {
			return nil, fmt.Errorf("index %d out of bounds for array of length %d", int(idx), arr.Length())
		}

		// Remove the element at the specified index
		arr.RemoveAt(int(idx))

		return arr, nil
	})

	rt.Register("lastIndex", func(args ...Value) (Value, error) {
		if len(args) != 2 {
			return nil, errors.New("lastIndex requires 2 arguments: array and value")
		}

		// Get the array
		arr, ok := args[0].(*ArrayValue)
		if !ok {
			return nil, fmt.Errorf("expected array, got %T", args[0])
		}

		// Get the value to find
		value := args[1]

		// Find the last index of the value
		lastIdx := -1
		for i := 0; i < arr.Length(); i++ {
			if equals(arr.Get(i), value) {
				lastIdx = i
			}
		}

		return Number(lastIdx), nil
	})

	// Array utilities
	rt.Register("slice", func(args ...Value) (Value, error) {
		if cfg.ChariotConfig.Verbose {
			fmt.Printf("DEBUG: sliceFunc called with '%v'", args)
		}

		if len(args) < 2 || len(args) > 3 {
			return nil, errors.New("slice requires 2 or 3 arguments: array, start, [end]")
		}

		// Get the array
		arr, ok := args[0].(*ArrayValue)
		if !ok {
			return nil, fmt.Errorf("expected array, got %T", args[0])
		}

		// Get the start index
		start, ok := args[1].(Number)
		if !ok {
			return nil, fmt.Errorf("start must be a number, got %T", args[1])
		}

		// Get the end index (default to array length if not provided)
		end := Number(arr.Length())
		if len(args) == 3 {
			endArg, ok := args[2].(Number)
			if !ok {
				return nil, fmt.Errorf("end must be a number, got %T", args[2])
			}
			end = endArg
		}

		// Handle negative indices
		startIdx := int(start)
		if startIdx < 0 {
			startIdx = arr.Length() + startIdx
			if startIdx < 0 {
				startIdx = 0
			}
		}

		endIdx := int(end)
		if endIdx < 0 {
			endIdx = arr.Length() + endIdx
			if endIdx < 0 {
				endIdx = 0
			}
		}

		// Ensure indices are within bounds
		if startIdx >= arr.Length() {
			startIdx = arr.Length()
		}

		if endIdx > arr.Length() {
			endIdx = arr.Length()
		}

		// Create a new array with the sliced elements
		result := NewArray()
		for i := startIdx; i < endIdx; i++ {
			result.Append(arr.Get(i))
		}

		return result, nil
	})

	rt.Register("reverse", func(args ...Value) (Value, error) {
		if cfg.ChariotConfig.Verbose {
			fmt.Printf("DEBUG: reverseFunc called with '%v'", args)
		}

		if len(args) != 1 {
			return nil, errors.New("reverse requires 1 argument: array")
		}

		// Get the array
		arr, ok := args[0].(*ArrayValue)
		if !ok {
			return nil, fmt.Errorf("expected array, got %T", args[0])
		}

		// Create a new array with the reversed elements
		result := NewArray()
		for i := arr.Length() - 1; i >= 0; i-- {
			result.Append(arr.Get(i))
		}

		return result, nil
	})

}

// Helper function to check if two values are equal
func equals(a, b Value) bool {
	// Check if both are null
	if a == DBNull && b == DBNull {
		return true
	}

	// Check by type
	switch aVal := a.(type) {
	case Number:
		if bVal, ok := b.(Number); ok {
			return aVal == bVal
		}
	case Str:
		if bVal, ok := b.(Str); ok {
			return aVal == bVal
		}
	case Bool:
		if bVal, ok := b.(Bool); ok {
			return aVal == bVal
		}
	case *TreeNode:
		if bVal, ok := b.(*TreeNode); ok {
			return aVal == bVal // Reference equality
		}
	case *ArrayValue:
		if bVal, ok := b.(*ArrayValue); ok {
			return aVal == bVal // Reference equality
		}
	}

	return false
}
