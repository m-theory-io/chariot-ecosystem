// Add this to chariot/json_funcs.go
package chariot

import (
	"encoding/json"
	"errors"
	"fmt"
)

// RegisterJSON registers all JSON-related functions
func RegisterJSON(rt *Runtime) {
	rt.Register("parseJSON", func(args ...Value) (Value, error) {
		if len(args) < 1 || len(args) > 2 {
			return nil, fmt.Errorf("parseJSON requires 1 or 2 arguments")
		}

		str, ok := args[0].(Str)
		if !ok {
			return nil, fmt.Errorf("parseJSON expects a string, got %T", args[0])
		}

		jsonName := "root"
		if len(args) == 2 {
			if name, ok := args[1].(Str); ok {
				jsonName = string(name)
			} else {
				return nil, fmt.Errorf("parseJSON expects a string for the second argument, got %T", args[1])
			}
		}

		// Parse JSON into native Go value
		var data interface{}
		if err := json.Unmarshal([]byte(str), &data); err != nil {
			return nil, fmt.Errorf("invalid JSON: %v", err)
		}

		// Create JSONNode and populate Properties (not Children)
		jsonNode := NewJSONNode(jsonName)
		jsonNode.SetJSONValue(data) // ← This should populate Properties

		return jsonNode, nil
	})

	// Update parseJSON to have a lightweight option
	rt.Register("parseJSONValue", func(args ...Value) (Value, error) {
		if len(args) != 1 {
			return nil, fmt.Errorf("parseJSONValue requires 1 argument")
		}

		jsonStr := string(args[0].(Str))
		var data interface{}
		if err := json.Unmarshal([]byte(jsonStr), &data); err != nil {
			return nil, fmt.Errorf("invalid JSON: %v", err)
		}

		// Convert to MapValue for objects, ArrayValue for arrays
		switch d := data.(type) {
		case map[string]interface{}:
			mapVal := NewMap()
			for key, value := range d {
				mapVal.Set(key, convertToChariotValue(value))
			}
			return mapVal, nil

		case []interface{}:
			arrayVal := NewArray()
			for _, value := range d {
				arrayVal.Append(convertToChariotValue(value))
			}
			return arrayVal, nil

		default:
			return convertToChariotValue(data), nil
		}
	})

	// Update parseJSONSimple to have a lightweight option
	rt.Register("parseJSONSimple", func(args ...Value) (Value, error) {
		if len(args) != 1 {
			return nil, fmt.Errorf("parseJSONValue requires 1 argument")
		}

		jsonStr := string(args[0].(Str))
		// Validate that the JSON string is not empty
		if jsonStr == "" {
			return nil, fmt.Errorf("parseJSONValue requires a non-empty JSON string")
		}
		// call ParseJSON to get a SimpleJSON
		simpleJSON, err := ParseJSON(jsonStr)
		if err != nil {
			return nil, fmt.Errorf("failed to parse JSON: %v", err)
		}
		// Return a SimpleJSON value
		return simpleJSON, nil
	})

	rt.Register("toJSON", func(args ...Value) (Value, error) {
		if len(args) != 1 {
			return nil, fmt.Errorf("toJSON requires 1 argument")
		}

		// Check if it's already a JSONNode
		if jsonNode, ok := args[0].(*JSONNode); ok {
			jsonStr, err := jsonNode.ToJSON()
			if err != nil {
				return nil, fmt.Errorf("failed to serialize JSONNode: %v", err)
			}
			return Str(jsonStr), nil
		}

		// For other types, convert to JSON string
		data := convertValueToNative(args[0])
		jsonBytes, err := json.Marshal(data)
		if err != nil {
			return nil, fmt.Errorf("failed to serialize to JSON: %v", err)
		}

		return Str(string(jsonBytes)), nil
	})

	// Add to RegisterJSONFuncs function
	rt.Register("toSimpleJSON", func(args ...Value) (Value, error) {
		if len(args) != 1 {
			return nil, errors.New("toSimpleJSON requires exactly 1 argument")
		}

		// Unwrap ScopeEntry if present
		if targ, ok := args[0].(ScopeEntry); ok {
			args[0] = targ.Value
		}

		// First, convert to native Go types using the existing function
		nativeValue := convertValueToNative(args[0])

		// Then, create a fresh JSONNode from the native value
		return createCleanJSONNode("json", nativeValue)
	})

}

// findJSONNode function
func (rt *Runtime) findJSONNode(nodeName string) (*JSONNode, error) {
	// First check if we have a document with this name
	if val, exists := rt.vars[nodeName]; exists {
		if jsonNode, ok := val.(*JSONNode); ok {
			return jsonNode, nil
		}
	}

	// Also check global vars
	if val, exists := rt.globalVars[nodeName]; exists {
		if jsonNode, ok := val.(*JSONNode); ok {
			return jsonNode, nil
		}
	}

	// Finally check the current scope
	if val, exists := rt.currentScope.Get(nodeName); exists {
		if jsonNode, ok := val.(*JSONNode); ok {
			return jsonNode, nil
		}
	}

	return nil, fmt.Errorf("JSON node '%s' not found", nodeName)
}

// Create a clean JSONNode from native Go values
func createCleanJSONNode(name string, v interface{}) (Value, error) {
	_ = name
	switch val := v.(type) {
	case map[string]interface{}:
		return NewSimpleJSON(v), nil

	case []interface{}:
		// Array case - create an ArrayValue
		array := NewArray()
		for _, elem := range val {
			elemValue := convertToChariotValue(elem)
			array.Append(elemValue)
		}

		return NewSimpleJSON(array), nil

	default:
		// Simple value case - convert directly
		return convertToChariotValue(val), nil
	}
}
