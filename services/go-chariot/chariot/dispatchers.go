package chariot

import (
	"errors"
	"fmt"
	"reflect"
	"strings"

	cfg "github.com/bhouse1273/chariot-ecosystem/services/go-chariot/configs"
)

// RegisterTypeDispatchedFunctions registers functions that dispatch based on argument types
func RegisterTypeDispatchedFunctions(rt *Runtime) {
	// dynamic addTo function
	rt.Register("addTo", func(args ...Value) (Value, error) {
		if cfg.ChariotConfig.Verbose {
			fmt.Printf("DEBUG: addToFunc called with '%v'", args)
		}

		if len(args) < 2 {
			return nil, errors.New("addTo requires at least 2 arguments: array and value(s)")
		}
		var result Value
		// Get the array
		switch firstArg := args[0].(type) {
		case []map[string]Value:
			// Handle slice of maps
			for _, tval := range args[1:] {
				if tvar, ok := tval.(ScopeEntry); ok {
					firstArg = append(firstArg, tvar.Value.(map[string]Value))
				} else {
					firstArg = append(firstArg, tval.(map[string]Value))
				}
			}
			result = firstArg
		case *ArrayValue:
			// Add each value to the array
			for _, value := range args[1:] {
				firstArg.Append(value)
			}
			result = firstArg
		default:
			return nil, fmt.Errorf("expected array, got %T", args[0])
		}

		return result, nil
	})

	// dynamic apply() function
	rt.Register("apply", func(args ...Value) (Value, error) {
		if len(args) != 2 {
			return nil, errors.New("apply requires 2 arguments: function and collection")
		}

		// Unwrap arguments if they're scope entries
		for i, arg := range args {
			if tvar, ok := arg.(ScopeEntry); ok {
				args[i] = tvar.Value
			}
		}

		// Get the function
		fn, ok := args[0].(*FunctionValue)
		if !ok {
			return nil, fmt.Errorf("first argument must be a function, got %T", args[0])
		}

		switch coll := args[1].(type) {
		case map[string]Value:
			// Iterate the map and execute the function for each key-value pair
			for key, value := range coll {
				if _, err := rt.funcs["call"](fn, key, value); err != nil {
					return nil, err
				}
			}
		case *ArrayValue:
			// Iterate the coll and execute the function for each element
			for key, value := range coll.Elements {
				if _, err := rt.funcs["call"](fn, key, value); err != nil {
					return nil, err
				}
			}
		case *MapValue:
			// Iterate the map and execute the function for each key-value pair
			for key, value := range coll.Values {
				if _, err := rt.funcs["call"](fn, key, value); err != nil {
					return nil, err
				}
			}
		case *JSONNode:
			// Iterate the attributes and execute the function for each key-value pair
			for key, value := range coll.Attributes {
				if _, err := rt.funcs["call"](fn, key, value); err != nil {
					return nil, err
				}
			}
		case *SimpleJSON:
			// Iterate the attributes and execute the function for each key-value pair
			for key, value := range coll.GetValue().(map[string]Value) {
				if _, err := rt.funcs["call"](fn, key, value); err != nil {
					return nil, err
				}
			}
		case *MapNode:
			// Iterate the attributes and execute the function for each key-value pair
			for key, value := range coll.Attributes {
				if _, err := rt.funcs["call"](fn, key, value); err != nil {
					return nil, err
				}
			}
		case TreeNode:
			// Iterate the children and execute the function for each child
			for _, child := range coll.GetChildren() {
				if _, err := rt.funcs["call"](fn, child); err != nil {
					return nil, err
				}
			}
		default:
			return nil, fmt.Errorf("unsupported collection type %T", coll)
		}
		return nil, nil
	})

	// dynamic clone() function
	rt.Register("clone", func(args ...Value) (Value, error) {
		if len(args) != 1 {
			return nil, errors.New("clone requires 1 argument: node")
		}

		// Unwrap argument if it's a scope entry
		if tvar, ok := args[0].(ScopeEntry); ok {
			args[0] = tvar.Value
		}

		// type dispatch for clone
		switch firstArg := args[0].(type) {
		case *ArrayValue:
			// Handle ArrayValue cloning
			clone, err := ArrayValueClone(firstArg)
			return clone, err
		case *MapValue:
			// Handle MapValue cloning
			clone, err := MapValueClone(firstArg)
			return clone, err
		case *MapNode:
			// Handle MapNode cloning
			clone, err := MapNodeClone(firstArg)
			return clone, err
		case *JSONNode:
			// Handle JSONNode cloning
			clone, err := JSONNodeClone(firstArg)
			return clone, err
		case *SimpleJSON:
			// Handle SimpleJSON cloning
			clone, err := SimpleJSONClone(firstArg)
			return clone, err
		case *TreeNode:
			// Handle TreeNode cloning
			clone, err := TreeNodeClone(firstArg)
			return clone, err
		}

		// Get the node
		node, ok := args[0].(TreeNode)
		if !ok {
			return nil, fmt.Errorf("expected node, got %T", args[0])
		}

		// Clone the node
		cloned := node.Clone()
		return cloned, nil
	})

	// getAt function
	rt.Register("getAt", func(args ...Value) (Value, error) {
		if len(args) != 2 {
			return nil, errors.New("getAt requires 2 arguments")
		}

		switch firstArg := args[0].(type) {
		case *ArrayValue:
			return arrayGetAt(args...)
		case *JSONNode:
			return jsonGetAt(args...)
		case *SimpleJSON:
			return simpleJSONGetAt(firstArg, args...)
		case Str:
			return stringGetAt(args...)
		case []Value:
			return valueSliceGetAt(args...)
		case []interface{}:
			return arrayInterfaceGetAt(args...)
		default:
			return nil, fmt.Errorf("getAt not supported for type %T", firstArg)
		}
	})

	// setAt function
	rt.Register("setAt", func(args ...Value) (Value, error) {
		if len(args) != 3 {
			return nil, errors.New("setAt requires 3 arguments: object, index, and value")
		}

		switch firstArg := args[0].(type) {
		case *ArrayValue:
			return arraySetAt(args...)
		case *JSONNode:
			return jsonSetAt(args...)
		case *TreeNodeImpl:
			return treeNodeImplSetAt(args...)
		case map[string]Value:
			return mapSetAt(args...)
		default:
			return nil, fmt.Errorf("setAt not supported for type %T", firstArg)
		}
	})

	// Dynamic getAttribute function
	rt.Register("getAttribute", func(args ...Value) (Value, error) {
		if len(args) != 2 {
			return nil, errors.New("getAttribute requires 2 arguments: object and attribute")
		}

		// Unwrap arguments if they're scope entries
		for i, arg := range args {
			if tvar, ok := arg.(ScopeEntry); ok {
				args[i] = tvar.Value
			}
		}

		// Get the node
		node := args[0]

		// Test that the key exists
		_, ok := args[1].(Str)
		if !ok {
			return nil, fmt.Errorf("attribute key must be a string, got %T", args[1])
		}
		attrName := string(args[1].(Str))

		// Add detailed debugging
		if cfg.ChariotConfig.Verbose {
			fmt.Printf("DEBUG: getAttribute called with node type %T, attribute '%s'\n", node, attrName)
		}

		// Handle different node types using helper functions
		switch n := node.(type) {
		case *Transform:
			// Try using the embedded TreeNodeImpl's GetAttribute directly
			if n.TreeNodeImpl.Attributes == nil {
				return nil, fmt.Errorf("Transform node has nil attributes map")
			}
			if val, exists := n.TreeNodeImpl.Attributes[attrName]; exists {
				return val, nil
			}
			return nil, fmt.Errorf("attribute '%s' not found in Transform attributes (has %d attributes)", attrName, len(n.TreeNodeImpl.Attributes))
		case map[string]interface{}:
			// Handle map[string]interface{} case
			if attr, exists := n[string(args[1].(Str))]; exists {
				return convertFromInterface(attr), nil
			}
			return nil, fmt.Errorf("attribute '%s' not found in map[string]interface{}", attrName)
		case *JSONNode:
			return jsonNodeGetAttribute(args...)

		case *TreeNodeImpl:
			return treeNodeImplGetAttribute(args...)

		case *MapNode:
			return mapNodeGetAttribute(args...)

		case map[string]Value:
			return mapGetAttribute(args...)

		case TreeNode:
			// Debug what type we're actually dealing with
			return nil, fmt.Errorf("TreeNode interface case: concrete type is %T, checking *Transform assertion: %v", n, func() interface{} {
				if tn, ok := n.(*Transform); ok {
					return fmt.Sprintf("SUCCESS - Transform has %d attributes", len(tn.TreeNodeImpl.Attributes))
				} else {
					return "FAILED - not *Transform"
				}
			}())

		default:
			return nil, fmt.Errorf("DEBUG: getAttribute called with unsupported type %T for attribute '%s'", node, attrName)
		}
	})

	// setAttribute - simple key-value attribute setting (no path traversal)
	rt.Register("setAttribute", func(args ...Value) (Value, error) {
		if len(args) != 3 {
			return nil, errors.New("setAttribute requires 3 arguments: node, key, value")
		}

		// Unwrap arguments if they are ScopeEntries
		for i, arg := range args {
			if tvar, ok := arg.(ScopeEntry); ok {
				args[i] = tvar.Value
			}
		}

		node := args[0]
		key, ok := args[1].(Str)
		if !ok {
			return nil, fmt.Errorf("attribute key must be a string, got %T", args[1])
		}
		value := args[2]

		// Handle different node types
		switch n := node.(type) {
		case *JSONNode:
			n.Attributes[string(key)] = value
			return value, nil
		case *XMLNode:
			// XML attributes must be strings
			if str, ok := value.(Str); ok {
				n.XMLAttributes[string(key)] = string(str)
				return value, nil
			}
			return nil, fmt.Errorf("XML attributes must be strings, got %T", value)
		case *TreeNodeImpl:
			if n.Attributes == nil {
				n.Attributes = make(map[string]Value)
			}
			n.Attributes[string(key)] = value
			return value, nil
		case TreeNode:
			if n.GetAttributes() == nil {
				n.SetAttribute("init", DBNull)
			}
			n.SetAttribute(string(key), value)
			return value, nil
		default:
			return nil, fmt.Errorf("setAttribute not supported for type %T", node)
		}
	})

	// Dynamic getAttributes
	rt.Register("getAttributes", func(args ...Value) (Value, error) {
		if len(args) != 1 {
			return nil, errors.New("getAttributes requires 1 argument: node")
		}
		// Unwrap argument if it's a scope entry
		if tvar, ok := args[0].(ScopeEntry); ok {
			args[0] = tvar.Value
		}
		// Get the node
		node := args[0]
		switch n := node.(type) {
		case *SimpleJSON:
			return n.GetValue().(map[string]Value), nil
		case *MapNode:
			return n.GetAttributes(), nil
		case *JSONNode:
			return n.GetAttributes(), nil
		case *MapValue:
			return n.GetAttributes(), nil
		case *TreeNodeImpl:
			return n.GetAttributes(), nil
		case TreeNode:
			return n.GetAttributes(), nil
		default:
			return nil, fmt.Errorf("getAttributes not supported for type %T", n)
		}
	})

	// Dynamic getProp function
	rt.Register("getProp", func(args ...Value) (Value, error) {
		if len(args) != 2 {
			return nil, errors.New("getProp requires 2 arguments: object and property name")
		}
		// Unwrap arguments if they are ScopeEntries
		for i, arg := range args {
			if tvar, ok := arg.(ScopeEntry); ok {
				args[i] = tvar.Value
			}
		}
		switch args[0].(type) {
		case *SimpleJSON:
			return getPropSimpleJSON(args...)
		case *MapNode:
			return getPropMapNode(args...)
		case *MapValue:
			return getPropMapValue(args...)
		case *JSONNode:
			return getPropJSON(args...)
		case *XMLNode:
			return getPropXML(args...)
		case *TreeNodeImpl:
			return getPropTreeNode(args...)
		case *Plan:
			return getPropPlan(args...)
		case map[string]Value:
			return getPropMap(args...)
		case map[string]interface{}:
			return getPropMapInterface(args...)
		case *HostObjectValue:
			return getPropHostObject(rt, args...)
		}
		return nil, fmt.Errorf("getProp not supported for type %T", args[0])
	})

	// Dynamic setProp function
	rt.Register("setProp", func(args ...Value) (Value, error) {
		if len(args) != 3 {
			return nil, errors.New("setProp requires 3 arguments: object, property path, and value")
		}

		// Unwrap arguments if they are ScopeEntries
		for i, arg := range args {
			if tvar, ok := arg.(ScopeEntry); ok {
				args[i] = tvar.Value
			}
		}

		switch args[0].(type) {
		case map[string]Value:
			return setPropMap(args...)
		case *SimpleJSON:
			return setPropSimpleJSON(args...)
		case *MapNode:
			return setPropMapNode(args...)
		case *MapValue:
			return setPropMapValue(args...)
		case *JSONNode:
			return setPropJSON(args...)
		case *XMLNode:
			return setPropXML(args...)
		case *TreeNodeImpl:
			return setPropTreeNodeImpl(args...)
		case *Plan:
			return setPropPlan(args...)
		case *HostObjectValue:
			return setPropHostObject(rt, args...)
		}
		return nil, fmt.Errorf("setProp not supported for type %T", args[0])
	})

	// Dynamic getMeta() function
	rt.Register("getMeta", func(args ...Value) (Value, error) {
		if len(args) != 2 {
			return nil, errors.New("getMeta requires 2 arguments: object and key")
		}

		// Unwrap arguments if they are ScopeEntries
		for i, arg := range args {
			if tvar, ok := arg.(ScopeEntry); ok {
				args[i] = tvar.Value
			}
		}

		// Get the key
		key, ok := args[1].(Str)
		if !ok {
			return nil, errors.New("key must be a string")
		}

		// Use the helper function
		if value, exists := getMetadata(args[0], string(key)); exists {
			return value, nil
		}

		return DBNull, nil
	})

	rt.Register("setMeta", func(args ...Value) (Value, error) {
		if len(args) != 3 {
			return nil, errors.New("setMeta requires 3 arguments: object, key, and value")
		}

		// Unwrap arguments if they are ScopeEntries
		for i, arg := range args {
			if tvar, ok := arg.(ScopeEntry); ok {
				args[i] = tvar.Value
			}
		}

		// Get the key
		key, ok := args[1].(Str)
		if !ok {
			return nil, errors.New("key must be a string")
		}

		// Use the helper function
		if ok := setMetadata(args[0], string(key), args[2]); ok {
			return args[0], nil
		}

		return nil, fmt.Errorf("object does not support metadata")
	})

	// Dynamic getAllMeta() function
	rt.Register("getAllMeta", func(args ...Value) (Value, error) {
		if len(args) != 1 {
			return nil, fmt.Errorf("getAllMeta requires 1 argument: node")
		}
		node := args[0]
		switch n := node.(type) {
		case *SimpleJSON:
			if n.meta == nil {
				return NewMap(), nil // No metadata available
			}
			return n.meta, nil // Return the metadata map
		case *JSONNode:
			return n.GetAllMeta(), nil // Return all metadata as a MapValue
		case *CouchbaseNode:
			return n.GetAllMeta(), nil // Return all metadata as a MapValue
		case *SQLNode:
			return n.GetAllMeta(), nil // Return all metadata as a MapValue
		case *CSVNode:
			return n.GetAllMeta(), nil // Return all metadata as a MapValue
		case *MapNode:
			return n.GetAllMeta(), nil // Return all metadata as a MapValue
		case TreeNode:
			// Generic TreeNode fallback
			if meta := n.GetAllMeta(); meta != nil {
				return meta, nil // Return all metadata as a MapValue
			}
			return NewMap(), nil // No metadata available
		default:
			return nil, fmt.Errorf("getAllMeta not supported for type %T", node)
		}
	})

	// Dynamic indexOf that dispatches based on first argument type
	rt.Register("indexOf", func(args ...Value) (Value, error) {
		if len(args) < 2 {
			return nil, errors.New("indexOf requires at least 2 arguments")
		}

		switch args[0].(type) {
		case Str:
			return stringIndexOf(args...)
		case *ArrayValue:
			return arrayIndexOf(args...)
		default:
			return nil, fmt.Errorf("indexOf not supported for type %T", args[0])
		}
	})

	// Dynamic length function
	rt.Register("length", func(args ...Value) (Value, error) {
		if len(args) != 1 {
			return nil, errors.New("length requires 1 argument")
		}

		switch val := args[0].(type) {
		case Str:
			return Number(len(val)), nil

		case *ArrayValue:
			return Number(val.Length()), nil
		case []interface{}:
			return Number(len(val)), nil
		case *SimpleJSON:
			// Handle SimpleJSON based on internal value type
			switch jsonVal := val.value.(type) {
			case []interface{}:
				// JSON array
				return Number(len(jsonVal)), nil

			case map[string]interface{}:
				// JSON object - return number of keys
				return Number(len(jsonVal)), nil

			case string:
				// JSON string
				return Number(len(jsonVal)), nil

			case nil:
				// null value
				return Number(0), nil

			default:
				// Primitive values (numbers, booleans) have length 1
				return Number(1), nil
			}

		case *JSONNode:
			// For JSON nodes representing arrays, count children
			if val.IsJSONArray() {
				return Number(len(val.GetChildren())), nil
			}
			// For JSON nodes representing objects, count properties
			if val.IsJSONObject() {
				return Number(len(val.GetChildren())), nil
			}
			// For primitive values
			return Number(1), nil

		case *MapValue:
			// Return number of keys in map
			return Number(len(val.Values)), nil

		case TreeNode:
			// Generic case for any TreeNode implementation
			return Number(len(val.GetChildren())), nil

		default:
			return Number(0), nil
			// return nil, fmt.Errorf("length not supported for type %T", val)
		}
	})

	// Generic slice function
	rt.Register("slice", func(args ...Value) (Value, error) {
		if len(args) < 2 {
			return nil, errors.New("slice requires at least 2 arguments")
		}

		switch args[0].(type) {
		case Str:
			return stringSlice(args...)
		case *ArrayValue:
			return arraySlice(args...)
		default:
			return nil, fmt.Errorf("slice not supported for type %T", args[0])
		}
	})

	// Generic reverse function
	rt.Register("reverse", func(args ...Value) (Value, error) {
		if len(args) != 1 {
			return nil, errors.New("reverse requires 1 argument")
		}

		switch args[0].(type) {
		case Str:
			return stringReverse(args...)
		case *ArrayValue:
			return arrayReverse(args...)
		default:
			return nil, fmt.Errorf("reverse not supported for type %T", args[0])
		}
	})

	// Generic contains function
	rt.Register("contains", func(args ...Value) (Value, error) {
		if len(args) != 2 {
			return nil, errors.New("contains requires 2 arguments")
		}

		switch args[0].(type) {
		case Str:
			return stringContains(args...)
		case *ArrayValue:
			return arrayContains(args...)
		default:
			return nil, fmt.Errorf("contains not supported for type %T", args[0])
		}
	})

	// Bidirectional split/join
	rt.Register("split", func(args ...Value) (Value, error) {
		if len(args) != 2 {
			return nil, errors.New("split requires 2 arguments: string and delimiter")
		}

		switch args[0].(type) {
		case Str:
			return stringSplit(args...)
		default:
			return nil, fmt.Errorf("split only supported for strings")
		}
	})

	rt.Register("join", func(args ...Value) (Value, error) {
		if len(args) != 2 {
			return nil, errors.New("join requires 2 arguments: array and delimiter")
		}

		switch args[0].(type) {
		case *ArrayValue:
			return arrayJoin(args...)
		default:
			return nil, fmt.Errorf("join only supported for arrays")
		}
	})

}

// Add arraySetAt function
func arraySetAt(args ...Value) (Value, error) {
	arr, ok := args[0].(*ArrayValue)
	if !ok {
		return nil, fmt.Errorf("expected ArrayValue, got %T", args[0])
	}

	num, ok := args[1].(Number)
	if !ok {
		return nil, fmt.Errorf("index must be a number, got %T", args[1])
	}

	index := int(num)
	value := args[2]

	// Check bounds
	if index < 0 || index >= arr.Length() {
		return nil, fmt.Errorf("index %d out of bounds for array of length %d", index, arr.Length())
	}

	// Set the element
	_ = arr.Set(index, value)

	return value, nil
}

// jsonSetAt function
func jsonSetAt(args ...Value) (Value, error) {
	jsonNode, ok := args[0].(*JSONNode)
	if !ok {
		return nil, fmt.Errorf("expected JSONNode, got %T", args[0])
	}

	num, ok := args[1].(Number)
	if !ok {
		return nil, fmt.Errorf("index must be a number, got %T", args[1])
	}

	index := int(num)
	value := args[2]

	// Check if this JSONNode represents an array
	if !jsonNode.IsJSONArray() {
		return nil, errors.New("JSONNode is not an array")
	}

	// Get the array data
	arrayValue := jsonNode.GetArrayValue()
	if arrayValue == nil {
		return nil, errors.New("JSONNode has no array data")
	}

	// Check bounds
	if index < 0 || index >= arrayValue.Length() {
		return nil, fmt.Errorf("index %d out of bounds for array of length %d", index, arrayValue.Length())
	}

	// Set the element
	_ = arrayValue.Set(index, value)

	return value, nil
}

func treeNodeImplSetAt(args ...Value) (Value, error) {
	node, ok := args[0].(*TreeNodeImpl)
	if !ok {
		return nil, fmt.Errorf("expected TreeNodeImpl, got %T", args[0])
	}

	num, ok := args[1].(Number)
	if !ok {
		return nil, fmt.Errorf("index must be a number, got %T", args[1])
	}

	index := int(num)
	value := args[2]

	// Check bounds
	if index < 0 || index >= len(node.Children) {
		return nil, fmt.Errorf("index %d out of bounds for node with %d children", index, len(node.Children))
	}

	// Set the child node
	switch v := value.(type) {
	case *TreeNodeImpl:
		// If the value is a TreeNodeImpl, we can set it directly
		node.Children[index] = v
	case *JSONNode:
		// If the value is a JSONNode, we can set it directly
		node.Children[index] = v
	default:
		return nil, fmt.Errorf("unsupported child node type: %T", value)
	}

	return value, nil
}

func mapSetAt(args ...Value) (Value, error) {
	if len(args) != 3 {
		return nil, errors.New("setAt requires 3 arguments: object, index, and value")
	}

	obj, ok := args[0].(map[string]Value)
	if !ok {
		return nil, fmt.Errorf("expected map[string]Value, got %T", args[0])
	}

	indexValue := args[1]
	value := args[2]

	// Convert index to string for map key
	var key string
	switch idx := indexValue.(type) {
	case Str:
		key = string(idx)
	case Number:
		key = fmt.Sprintf("%.0f", float64(idx))
	default:
		return nil, fmt.Errorf("index must be a string or number, got %T", indexValue)
	}

	// Set the value in the map
	obj[key] = value

	return value, nil
}

func simpleJSONGetAt(firstArg *SimpleJSON, args ...Value) (Value, error) {
	// Handle SimpleJSON based on internal value type
	switch jsonValue := firstArg.value.(type) {
	case []interface{}:
		// JSON array
		if num, ok := args[1].(Number); ok {
			index := int(num)
			if index < 0 || index >= len(jsonValue) {
				return DBNull, nil // Return DBNull for out of bounds
			}
			// Extract the value at the specified index and wrap it as a SimpleJSON
			return NewSimpleJSON(jsonValue[index]), nil
		}
		return nil, fmt.Errorf("getAt index must be a number")
	case map[string]interface{}:
		// JSON object - return number of keys
		return Number(len(jsonValue)), nil
	case string:
		// JSON string - treat as single character access
		if num, ok := args[1].(Number); ok {
			index := int(num)
			strValue := firstArg.value.(string)
			if index < 0 || index >= len(strValue) {
				return DBNull, nil // Return DBNull for out of bounds
			}
			return Str(string(strValue[index])), nil
		}
		return nil, fmt.Errorf("getAt not supported for string access in SimpleJSON")
	case nil:
		// null value - return DBNull
		return DBNull, nil
	default:
		// Primitive values (numbers, booleans) have length 1
		if num, ok := args[1].(Number); ok && int(num) == 0 {
			return Str(fmt.Sprintf("%v", firstArg.value)), nil
		}
		return nil, fmt.Errorf("getAt not supported for type %T in SimpleJSON", firstArg.value)
	}
}

// stringIndexOf - extracted from string_funcs.go
func stringIndexOf(args ...Value) (Value, error) {
	if len(args) != 2 && len(args) != 3 {
		return nil, errors.New("indexOf requires 2 or 3 arguments: string, substring, [start]")
	}

	str, ok := args[0].(Str)
	if !ok {
		return nil, fmt.Errorf("expected string, got %T", args[0])
	}

	substr, ok := args[1].(Str)
	if !ok {
		return nil, fmt.Errorf("substring must be a string, got %T", args[1])
	}

	start := 0
	if len(args) == 3 {
		startPos, ok := args[2].(Number)
		if !ok {
			return nil, fmt.Errorf("start position must be a number, got %T", args[2])
		}
		start = int(startPos)
		if start < 0 {
			start = 0
		} else if start >= len(str) {
			return Number(-1), nil
		}
	}

	position := strings.Index(string(str)[start:], string(substr))
	if position == -1 {
		return Number(-1), nil
	}
	return Number(position + start), nil
}

// arrayIndexOf - for arrays
func arrayIndexOf(args ...Value) (Value, error) {
	if len(args) != 2 {
		return nil, errors.New("indexOf requires 2 arguments: array and value")
	}

	arr, ok := args[0].(*ArrayValue)
	if !ok {
		return nil, fmt.Errorf("expected array, got %T", args[0])
	}

	searchValue := args[1]

	for i := 0; i < arr.Length(); i++ {
		if compareValues(arr.Get(i), searchValue) {
			return Number(i), nil
		}
	}

	return Number(-1), nil
}

// Helper function to compare values for array searching
func compareValues(a, b Value) bool {
	// Handle exact type matches first
	switch va := a.(type) {
	case Str:
		if vb, ok := b.(Str); ok {
			return string(va) == string(vb)
		}
	case Number:
		if vb, ok := b.(Number); ok {
			return float64(va) == float64(vb)
		}
	case Bool:
		if vb, ok := b.(Bool); ok {
			return bool(va) == bool(vb)
		}
	}

	// Handle DBNull
	if a == DBNull && b == DBNull {
		return true
	}

	return false
}

// getPropHostObject - for HostObjectValue
func getPropHostObject(rt *Runtime, args ...Value) (Value, error) {
	if len(args) != 2 {
		return nil, errors.New("getProp requires 2 arguments: object and propertyName")
	}

	var objName string
	switch obj := args[0].(type) {
	case Str:
		objName = string(obj)
	default:
		return nil, errors.New("first argument must be object name")
	}

	// Get the property name
	propName, ok := args[1].(Str)
	if !ok {
		return nil, errors.New("second argument must be property name string")
	}

	// Get the object
	hostObj, exists := rt.objects[objName]
	if !exists {
		return nil, fmt.Errorf("host object '%s' not found", objName)
	}

	// Use reflection to get the property
	val, err := getObjectProperty(hostObj, string(propName))
	if err != nil {
		return nil, err
	}

	// Convert to Chariot value
	return convertToChariotValue(reflect.ValueOf(val)), nil
}

// getPropMapNode
func getPropMapNode(args ...Value) (Value, error) {
	if len(args) != 2 {
		return nil, errors.New("getProp requires 2 arguments: object and property name")
	}

	obj, ok := args[0].(*MapNode)
	if !ok {
		return nil, fmt.Errorf("expected MapNode, got %T", args[0])
	}

	propName, ok := args[1].(Str)
	if !ok {
		return nil, fmt.Errorf("property name must be a string, got %T", args[1])
	}

	value, ok := obj.Get(string(propName))
	if !ok {
		return nil, fmt.Errorf("property '%s' not found in MapNode", propName)
	}

	return value, nil
}

// getPropSimpleJSON
func getPropSimpleJSON(args ...Value) (Value, error) {
	if len(args) != 2 {
		return nil, errors.New("getProp requires 2 arguments: object and property name")
	}

	obj, ok := args[0].(*SimpleJSON)
	if !ok {
		return nil, fmt.Errorf("expected SimpleJSON, got %T", args[0])
	}

	propName, ok := args[1].(Str)
	if !ok {
		return nil, fmt.Errorf("property name must be a string, got %T", args[1])
	}

	value, ok := obj.Get(string(propName))
	if !ok {
		return nil, fmt.Errorf("property '%s' not found in SimpleJSON", propName)
	}

	return value, nil
}

// getPropMapValue
func getPropMapValue(args ...Value) (Value, error) {
	if len(args) != 2 {
		return nil, errors.New("getProp requires 2 arguments: object and property name")
	}

	obj, ok := args[0].(*MapValue)
	if !ok {
		return nil, fmt.Errorf("expected MapValue, got %T", args[0])
	}

	propName, ok := args[1].(Str)
	if !ok {
		return nil, fmt.Errorf("property name must be a string, got %T", args[1])
	}

	value, ok := obj.Get(string(propName))
	if !ok {
		return nil, fmt.Errorf("property '%s' not found in MapValue", propName)
	}

	return value, nil
}

// getPropMap
func getPropMap(args ...Value) (Value, error) {
	if len(args) != 2 {
		return nil, errors.New("getProp requires 2 arguments: object and property name")
	}

	obj, ok := args[0].(map[string]Value)
	if !ok {
		return nil, fmt.Errorf("expected MapValue, got %T", args[0])
	}

	propName, ok := args[1].(Str)
	if !ok {
		return nil, fmt.Errorf("property name must be a string, got %T", args[1])
	}

	value, ok := obj[string(propName)]
	if !ok {
		return nil, fmt.Errorf("property '%s' not found in map[string]Value", propName)
	}

	return value, nil
}

// getPropMapInterface
func getPropMapInterface(args ...Value) (Value, error) {
	if len(args) != 2 {
		return nil, errors.New("getProp requires 2 arguments: object and property name")
	}

	obj, ok := args[0].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("expected map[string]interface{}, got %T", args[0])
	}

	propName, ok := args[1].(Str)
	if !ok {
		return nil, fmt.Errorf("property name must be a string, got %T", args[1])
	}

	value, ok := obj[string(propName)]
	if !ok {
		return nil, fmt.Errorf("property '%s' not found in map[string]interface{}", propName)
	}

	return convertFromNativeValue(value), nil
}

// getPropJSON
func getPropJSON(args ...Value) (Value, error) {
	if len(args) != 2 {
		return nil, errors.New("getProp requires 2 arguments: node and property path")
	}

	jsonNode, ok := args[0].(*JSONNode)
	if !ok {
		return nil, fmt.Errorf("getProp requires a JSONNode, got %T", args[0])
	}

	propPath, ok := args[1].(Str)
	if !ok {
		return nil, fmt.Errorf("property path must be a string, got %T", args[1])
	}

	// Empty path validation
	if string(propPath) == "" {
		return nil, errors.New("property path cannot be empty")
	}

	fmt.Printf("DEBUG: getProp called with path='%s'\n", string(propPath))
	data := jsonNode.GetJSONValue()
	fmt.Printf("DEBUG: getProp sees data=%v (type %T)\n", data, data)

	result := getJSONProperty(data, string(propPath))
	fmt.Printf("DEBUG: getProp returning: %v\n", result)
	return result, nil
}

// String implementations
func stringSlice(args ...Value) (Value, error) {

	var start int

	str := string(args[0].(Str))
	runes := []rune(str)

	if tvar, ok := args[1].(Number); ok {
		start = int(tvar)
	} else {
		return nil, fmt.Errorf("start index must be a number, got %T", args[1])
	}
	if start < 0 {
		start = 0
	}

	var end int
	if len(args) > 2 {
		if tvar, ok := args[2].(Number); ok {
			end = int(tvar)
		} else {
			return nil, fmt.Errorf("end index must be a number, got %T", args[2])
		}
		// Make sure end doesn't exceed length
		if end > len(runes) {
			end = len(runes)
		}
	} else {
		end = len(runes)
	}

	// Ensure valid range
	if start >= end || start >= len(runes) {
		return Str(""), nil
	}

	return Str(string(runes[start:end])), nil
}

func stringReverse(args ...Value) (Value, error) {
	str, ok := args[0].(Str)
	if !ok {
		return nil, fmt.Errorf("expected string, got %T", args[0])
	}

	// Convert to runes for proper Unicode handling
	runes := []rune(string(str))

	// Reverse the runes slice
	for i, j := 0, len(runes)-1; i < j; i, j = i+1, j-1 {
		runes[i], runes[j] = runes[j], runes[i]
	}

	return Str(string(runes)), nil
}

func stringContains(args ...Value) (Value, error) {
	str, ok := args[0].(Str)
	if !ok {
		return nil, fmt.Errorf("expected string, got %T", args[0])
	}

	substr, ok := args[1].(Str)
	if !ok {
		return nil, fmt.Errorf("search value must be a string, got %T", args[1])
	}

	result := strings.Contains(string(str), string(substr))
	return Bool(result), nil
}

func stringSplit(args ...Value) (Value, error) {
	str, ok := args[0].(Str)
	if !ok {
		return nil, fmt.Errorf("expected string, got %T", args[0])
	}

	delimiter, ok := args[1].(Str)
	if !ok {
		return nil, fmt.Errorf("delimiter must be a string, got %T", args[1])
	}

	parts := strings.Split(string(str), string(delimiter))

	// Create array result
	arr := NewArray()
	for _, part := range parts {
		arr.Append(Str(part))
	}

	return arr, nil
}

// Array implementations
func arraySlice(args ...Value) (Value, error) {
	arr, ok := args[0].(*ArrayValue)
	if !ok {
		return nil, fmt.Errorf("expected array, got %T", args[0])
	}

	start, ok := args[1].(Number)
	if !ok {
		return nil, fmt.Errorf("start index must be a number, got %T", args[1])
	}

	startIdx := int(start)
	length := arr.Length()

	// Handle negative indices
	if startIdx < 0 {
		startIdx = 0
	}
	if startIdx >= length {
		return NewArray(), nil // Return empty array
	}

	// Default end to array length
	endIdx := length
	if len(args) >= 3 {
		end, ok := args[2].(Number)
		if !ok {
			return nil, fmt.Errorf("end index must be a number, got %T", args[2])
		}
		endIdx = int(end)
		if endIdx > length {
			endIdx = length
		}
		if endIdx < startIdx {
			endIdx = startIdx
		}
	}

	// Create new array with sliced elements
	result := NewArray()
	for i := startIdx; i < endIdx; i++ {
		result.Append(arr.Get(i))
	}

	return result, nil
}

func arrayReverse(args ...Value) (Value, error) {
	arr, ok := args[0].(*ArrayValue)
	if !ok {
		return nil, fmt.Errorf("expected array, got %T", args[0])
	}

	length := arr.Length()
	result := NewArray()

	// Add elements in reverse order
	for i := length - 1; i >= 0; i-- {
		result.Append(arr.Get(i))
	}

	return result, nil
}

func arrayContains(args ...Value) (Value, error) {
	arr, ok := args[0].(*ArrayValue)
	if !ok {
		return nil, fmt.Errorf("expected array, got %T", args[0])
	}

	searchValue := args[1]

	// Search through array elements
	for i := 0; i < arr.Length(); i++ {
		if compareValues(arr.Get(i), searchValue) {
			return Bool(true), nil
		}
	}

	return Bool(false), nil
}

func arrayJoin(args ...Value) (Value, error) {
	arr, ok := args[0].(*ArrayValue)
	if !ok {
		return nil, fmt.Errorf("expected array, got %T", args[0])
	}

	delimiter, ok := args[1].(Str)
	if !ok {
		return nil, fmt.Errorf("delimiter must be a string, got %T", args[1])
	}

	if arr.Length() == 0 {
		return Str(""), nil
	}

	var parts []string
	for i := 0; i < arr.Length(); i++ {
		element := arr.Get(i)
		// Convert element to string
		if strVal, ok := element.(Str); ok {
			parts = append(parts, string(strVal))
		} else {
			parts = append(parts, fmt.Sprintf("%v", element))
		}
	}

	result := strings.Join(parts, string(delimiter))
	return Str(result), nil
}

// arrayGetAt - for ArrayValue
func arrayGetAt(args ...Value) (Value, error) {
	arr, ok := args[0].(*ArrayValue)
	if !ok {
		return nil, fmt.Errorf("expected array, got %T", args[0])
	}

	num, ok := args[1].(Number)
	if !ok {
		return nil, fmt.Errorf("index must be a number, got %T", args[1])
	}

	index := int(num)

	// Return DBNull for invalid indices instead of error (fixes test failures)
	if index < 0 || index >= arr.Length() {
		return DBNull, nil
	}

	return arr.Get(index), nil
}

// arrayInterfaceGetAt - for ArrayValue
func arrayInterfaceGetAt(args ...Value) (Value, error) {
	arr, ok := args[0].([]interface{})
	if !ok {
		return nil, fmt.Errorf("expected array, got %T", args[0])
	}

	num, ok := args[1].(Number)
	if !ok {
		return nil, fmt.Errorf("index must be a number, got %T", args[1])
	}

	index := int(num)

	// Return DBNull for invalid indices instead of error (fixes test failures)
	if index < 0 || index >= len(arr) {
		return DBNull, nil
	}

	return arr[index], nil
}

func valueSliceGetAt(args ...Value) (Value, error) {
	if len(args) != 2 {
		return nil, errors.New("getAt requires 2 arguments: array and index")
	}

	arr, ok := args[0].([]Value)
	if !ok {
		return nil, fmt.Errorf("expected []Value, got %T", args[0])
	}

	num, ok := args[1].(Number)
	if !ok {
		return nil, fmt.Errorf("index must be a number, got %T", args[1])
	}

	index := int(num)

	// Return DBNull for invalid indices instead of error (fixes test failures)
	if index < 0 || index >= len(arr) {
		return DBNull, nil
	}

	return arr[index], nil
}

// jsonGetAt - for JSONNode (array access)
func jsonGetAt(args ...Value) (Value, error) {
	jsonNode, ok := args[0].(*JSONNode)
	if !ok {
		return nil, fmt.Errorf("expected JSONNode, got %T", args[0])
	}

	num, ok := args[1].(Number)
	if !ok {
		return nil, fmt.Errorf("index must be a number, got %T", args[1])
	}

	index := int(num)

	// Check if this JSONNode represents an array using the new method
	if !jsonNode.IsJSONArray() {
		return nil, errors.New("JSONNode is not an array")
	}

	// Get the array data using the new method
	arrayValue := jsonNode.GetArrayValue()
	if arrayValue == nil {
		return DBNull, nil
	}

	// Return DBNull for invalid indices
	if index < 0 || index >= arrayValue.Length() {
		return DBNull, nil
	}

	// Get element directly from ArrayValue
	element := arrayValue.Get(index)
	if element == nil {
		return DBNull, nil
	}

	return element, nil
}

func setPropSimpleJSON(args ...Value) (Value, error) {
	if len(args) != 3 {
		return nil, errors.New("setProp requires 3 arguments: object, property name, and value")
	}

	obj, ok := args[0].(*SimpleJSON)
	if !ok {
		return nil, fmt.Errorf("expected SimpleJSON, got %T", args[0])
	}

	propName, ok := args[1].(Str)
	if !ok {
		return nil, fmt.Errorf("property name must be a string, got %T", args[1])
	}

	value := args[2]

	// Set the property in the SimpleJSON
	obj.Set(string(propName), value)

	return value, nil
}

func setPropTreeNodeImpl(args ...Value) (Value, error) {
	if len(args) != 3 {
		return nil, errors.New("setProp requires 3 arguments: object, property name, and value")
	}

	node, ok := args[0].(*TreeNodeImpl)
	if !ok {
		return nil, fmt.Errorf("expected TreeNodeImpl, got %T", args[0])
	}

	propName, ok := args[1].(Str)
	if !ok {
		return nil, fmt.Errorf("property name must be a string, got %T", args[1])
	}

	value := args[2]

	// For simple keys (no dots), set in attributes
	if !strings.Contains(string(propName), ".") {
		if node.Attributes == nil {
			node.Attributes = make(map[string]Value)
		}
		node.Attributes[string(propName)] = value
		return value, nil
	}

	switch strings.ToLower(string(propName)) {
	case "name":
		// Special case for setting the name of a TreeNode
		if tvar, ok := value.(Str); ok {
			node.SetName(string(tvar))
		}
	case "attributes":
		// Special case for setting attributes
		if attrs, ok := value.(map[string]Value); ok {
			node.Attributes = attrs
		} else {
			return nil, fmt.Errorf("attributes must be a map[string]Value, got %T", value)
		}
	}
	return value, nil
}

func setPropMap(args ...Value) (Value, error) {
	if len(args) != 3 {
		return nil, errors.New("setProp requires 3 arguments: object, property name, and value")
	}

	obj, ok := args[0].(map[string]Value)
	if !ok {
		return nil, fmt.Errorf("expected map[string]Value, got %T", args[0])
	}

	propName, ok := args[1].(Str)
	if !ok {
		return nil, fmt.Errorf("property name must be a string, got %T", args[1])
	}

	value := args[2]

	// Set the property in the map
	obj[string(propName)] = value

	return value, nil
}

func setPropMapNode(args ...Value) (Value, error) {
	if len(args) != 3 {
		return nil, errors.New("setProp requires 3 arguments: object, property name, and value")
	}

	obj, ok := args[0].(*MapNode)
	if !ok {
		return nil, fmt.Errorf("expected MapNode, got %T", args[0])
	}

	propName, ok := args[1].(Str)
	if !ok {
		return nil, fmt.Errorf("property name must be a string, got %T", args[1])
	}

	value := args[2]

	// Set the property in the MapNode
	obj.Set(string(propName), value)

	return value, nil
}

func setPropMapValue(args ...Value) (Value, error) {
	if len(args) != 3 {
		return nil, errors.New("setProp requires 3 arguments: object, property name, and value")
	}

	obj, ok := args[0].(*MapValue)
	if !ok {
		return nil, fmt.Errorf("expected MapValue, got %T", args[0])
	}

	propName, ok := args[1].(Str)
	if !ok {
		return nil, fmt.Errorf("property name must be a string, got %T", args[1])
	}

	value := args[2]

	// Set the property in the MapValue
	obj.Set(string(propName), value)

	return value, nil
}
func setPropJSON(args ...Value) (Value, error) {
	if len(args) != 3 {
		return nil, errors.New("setProp requires 3 arguments: node, property path, and value")
	}

	jsonNode, ok := args[0].(*JSONNode)
	if !ok {
		return nil, fmt.Errorf("expected JSONNode, got %T", args[0])
	}

	propPath, ok := args[1].(Str)
	if !ok {
		return nil, fmt.Errorf("property path must be a string, got %T", args[1])
	}

	value := args[2]

	// Set the property by path
	if err := jsonNode.SetJSONPath(string(propPath), value); err != nil {
		return nil, err
	}

	return value, nil
}
func setPropHostObject(rt *Runtime, args ...Value) (Value, error) {
	if len(args) != 3 {
		return nil, errors.New("setProp requires 3 arguments: object, propertyName, and value")
	}

	var objName string
	switch obj := args[0].(type) {
	case Str:
		objName = string(obj)
	default:
		return nil, errors.New("first argument must be object name")
	}

	// Get the property name
	propName, ok := args[1].(Str)
	if !ok {
		return nil, errors.New("second argument must be property name string")
	}

	// Get the value to set
	value := args[2]

	// Set the property
	return rt.SetObjectProperty(objName, string(propName), value)
}

// getPropPlan - dynamic property access for Plan
func getPropPlan(args ...Value) (Value, error) {
	if len(args) != 2 {
		return nil, errors.New("getProp requires 2 arguments: object and property name")
	}
	p, ok := args[0].(*Plan)
	if !ok {
		return nil, fmt.Errorf("expected Plan, got %T", args[0])
	}
	propName, ok := args[1].(Str)
	if !ok {
		return nil, fmt.Errorf("property name must be a string, got %T", args[1])
	}
	switch strings.ToLower(string(propName)) {
	case "name":
		return Str(p.Name), nil
	case "params":
		arr := NewArray()
		for _, s := range p.Params {
			arr.Append(Str(s))
		}
		return arr, nil
	case "trigger":
		if p.Trigger == nil {
			return DBNull, nil
		}
		return p.Trigger, nil
	case "guard":
		if p.Guard == nil {
			return DBNull, nil
		}
		return p.Guard, nil
	case "drop":
		if p.Drop == nil {
			return DBNull, nil
		}
		return p.Drop, nil
	case "steps":
		arr := NewArray()
		for _, s := range p.Steps {
			arr.Append(s)
		}
		return arr, nil
	default:
		return nil, fmt.Errorf("property '%s' not found in Plan", propName)
	}
}

// setPropPlan - dynamic property set for Plan
func setPropPlan(args ...Value) (Value, error) {
	if len(args) != 3 {
		return nil, errors.New("setProp requires 3 arguments: object, property name, and value")
	}
	p, ok := args[0].(*Plan)
	if !ok {
		return nil, fmt.Errorf("expected Plan, got %T", args[0])
	}
	propName, ok := args[1].(Str)
	if !ok {
		return nil, fmt.Errorf("property name must be a string, got %T", args[1])
	}
	value := args[2]
	switch strings.ToLower(string(propName)) {
	case "name":
		if s, ok := value.(Str); ok {
			p.Name = string(s)
			return value, nil
		}
		return nil, fmt.Errorf("name must be string, got %T", value)
	case "params":
		av, ok := value.(*ArrayValue)
		if !ok {
			return nil, fmt.Errorf("params must be array of strings, got %T", value)
		}
		params := make([]string, 0, av.Length())
		for i := 0; i < av.Length(); i++ {
			if s, ok := av.Get(i).(Str); ok {
				params = append(params, string(s))
			} else {
				return nil, fmt.Errorf("params[%d] must be string, got %T", i, av.Get(i))
			}
		}
		p.Params = params
		return value, nil
	case "trigger":
		if fv, ok := value.(*FunctionValue); ok {
			p.Trigger = fv
			return value, nil
		}
		return nil, fmt.Errorf("trigger must be function, got %T", value)
	case "guard":
		if fv, ok := value.(*FunctionValue); ok {
			p.Guard = fv
			return value, nil
		}
		return nil, fmt.Errorf("guard must be function, got %T", value)
	case "drop":
		if fv, ok := value.(*FunctionValue); ok {
			p.Drop = fv
			return value, nil
		}
		return nil, fmt.Errorf("drop must be function, got %T", value)
	case "steps":
		av, ok := value.(*ArrayValue)
		if !ok {
			return nil, fmt.Errorf("steps must be array of functions, got %T", value)
		}
		steps := make([]*FunctionValue, 0, av.Length())
		for i := 0; i < av.Length(); i++ {
			if fv, ok := av.Get(i).(*FunctionValue); ok {
				steps = append(steps, fv)
			} else {
				return nil, fmt.Errorf("steps[%d] must be function, got %T", i, av.Get(i))
			}
		}
		p.Steps = steps
		return value, nil
	default:
		return nil, fmt.Errorf("property '%s' not found in Plan", propName)
	}
}

// Helper for TreeNodeImpl
func treeNodeImplGetAttribute(args ...Value) (Value, error) {
	treeNode, ok := args[0].(*TreeNodeImpl)
	if !ok {
		return nil, fmt.Errorf("expected TreeNodeImpl, got %T", args[0])
	}

	attrName, ok := args[1].(Str)
	if !ok {
		return nil, fmt.Errorf("attribute name must be a string, got %T", args[1])
	}

	if treeNode.Attributes == nil {
		return nil, fmt.Errorf("node has no attributes")
	}
	if val, exists := treeNode.Attributes[string(attrName)]; exists {
		return val, nil
	}
	return nil, fmt.Errorf("attribute '%s' not found", attrName)
}

// Helper for Transform
func transformGetAttribute(args ...Value) (Value, error) {
	transformNode, ok := args[0].(*Transform)
	if !ok {
		return nil, fmt.Errorf("expected Transform, got %T", args[0])
	}

	attrName, ok := args[1].(Str)
	if !ok {
		return nil, fmt.Errorf("attribute name must be a string, got %T", args[1])
	}

	if transformNode.Attributes == nil {
		return nil, fmt.Errorf("node has no attributes")
	}
	if val, exists := transformNode.Attributes[string(attrName)]; exists {
		return val, nil
	}
	return nil, fmt.Errorf("attribute '%s' not found", attrName)
}

// Helper for MapNode
func mapNodeGetAttribute(args ...Value) (Value, error) {
	mapNode, ok := args[0].(*MapNode)
	if !ok {
		return nil, fmt.Errorf("expected MapNode, got %T", args[0])
	}

	attrName, ok := args[1].(Str)
	if !ok {
		return nil, fmt.Errorf("attribute name must be a string, got %T", args[1])
	}

	if mapNode.Attributes == nil {
		return nil, fmt.Errorf("node has no attributes")
	}
	if val, exists := mapNode.Attributes[string(attrName)]; exists {
		return val, nil
	}
	return nil, fmt.Errorf("attribute '%s' not found", attrName)
}

// Helper for JSONNode
func jsonNodeGetAttribute(args ...Value) (Value, error) {
	jsonNode, ok := args[0].(*JSONNode)
	if !ok {
		return nil, fmt.Errorf("expected JSONNode, got %T", args[0])
	}

	attrName, ok := args[1].(Str)
	if !ok {
		return nil, fmt.Errorf("attribute name must be a string, got %T", args[1])
	}

	if jsonNode.Attributes == nil {
		return nil, fmt.Errorf("node has no attributes")
	}

	if val, exists := jsonNode.Attributes[string(attrName)]; exists {
		return val, nil
	} else {
		// Check for special JSONNode case
		testAttr := "_" + jsonNode.NameStr
		if val2, exists := jsonNode.Attributes[testAttr]; exists {
			trec := val2.(*ArrayValue).Get(0).(map[string]Value)
			if val3, exists := trec[string(attrName)]; exists {
				return val3, nil
			}
		} else {
			// recursively walk jsonNode.Attributes
			for _, v := range jsonNode.Attributes {
				switch v := v.(type) {
				case *ArrayValue:
					for i := 0; i < v.Length(); i++ {
						trec := v.Elements[i]
						switch trec := trec.(type) {
						case map[string]Value:
							if tval, exists := trec[string(attrName)]; exists {
								return tval, nil
							}
						}
					}
				case *MapNode, *JSONNode, TreeNode:
					trec, err := jsonNodeGetAttribute(v, attrName)
					return trec, err
				}
			}
		}
	}
	return nil, fmt.Errorf("attribute '%s' not found", attrName)
}

// Helper for Go native map
func mapGetAttribute(args ...Value) (Value, error) {
	mapValue, ok := args[0].(map[string]Value)
	if !ok {
		return nil, fmt.Errorf("expected map[string]Value, got %T", args[0])
	}

	attrName, ok := args[1].(Str)
	if !ok {
		return nil, fmt.Errorf("attribute name must be a string, got %T", args[1])
	}

	if val, exists := mapValue[string(attrName)]; exists {
		return val, nil
	}
	return nil, fmt.Errorf("attribute '%s' not found", attrName)
}

// ArrayValueClone -- be sure not to modify the original array
func ArrayValueClone(args ...Value) (Value, error) {
	if len(args) != 1 {
		return nil, errors.New("clone requires 1 argument: array")
	}

	arr, ok := args[0].(*ArrayValue)
	if !ok {
		return nil, fmt.Errorf("expected ArrayValue, got %T", args[0])
	}

	// Create a new ArrayValue and copy elements
	newArr := NewArray()
	for i := 0; i < arr.Length(); i++ {
		newArr.Append(arr.Get(i))
	}

	return newArr, nil
}

func MapValueClone(args ...Value) (Value, error) {
	if len(args) != 1 {
		return nil, errors.New("clone requires 1 argument: map")
	}

	mapValue, ok := args[0].(*MapValue)
	if !ok {
		return nil, fmt.Errorf("expected MapValue, got %T", args[0])
	}

	// Create a new MapValue and copy elements
	newMap := NewMap()
	for key, val := range mapValue.GetAttributes() {
		newMap.Set(key, val)
	}

	return newMap, nil
}

func MapNodeClone(args ...Value) (Value, error) {
	if len(args) != 2 {
		return nil, errors.New("clone requires 2 arguments: map node and clone name")
	}

	mapNode, ok := args[0].(*MapNode)
	if !ok {
		return nil, fmt.Errorf("expected MapNode, got %T", args[0])
	}

	cloneName, ok := args[1].(Str)
	if !ok {
		return nil, fmt.Errorf("clone name must be a string, got %T", args[1])
	}

	// Create a new MapNode and copy elements
	newMapNode := NewMapNode(string(cloneName))
	for key, val := range mapNode.GetAttributes() {
		newMapNode.Set(key, val)
	}

	// Recursively clone children
	for _, child := range mapNode.GetChildren() {
		clonedChild := child.Clone()
		newMapNode.AddChild(clonedChild)
	}

	return newMapNode, nil
}

func JSONNodeClone(args ...Value) (Value, error) {
	if len(args) != 2 {
		return nil, errors.New("clone requires 2 arguments: JSON node and clone name")
	}

	jsonNode, ok := args[0].(*JSONNode)
	if !ok {
		return nil, fmt.Errorf("expected JSONNode, got %T", args[0])
	}

	cloneName, ok := args[1].(Str)
	if !ok {
		return nil, fmt.Errorf("clone name must be a string, got %T", args[1])
	}

	// Create a new JSONNode and copy elements
	newJSONNode := NewJSONNode(string(cloneName))
	newJSONNode.SetJSONValue(jsonNode.GetJSONValue())

	// Recursively clone children
	for _, child := range jsonNode.GetChildren() {
		clonedChild := child.Clone()
		newJSONNode.AddChild(clonedChild)
	}

	return newJSONNode, nil
}

func SimpleJSONClone(args ...Value) (Value, error) {
	if len(args) != 1 {
		return nil, errors.New("clone requires 1 argument: SimpleJSON object")
	}

	simpleJSON, ok := args[0].(*SimpleJSON)
	if !ok {
		return nil, fmt.Errorf("expected SimpleJSON, got %T", args[0])
	}

	// Create a new SimpleJSON and copy the value
	newSimpleJSON := NewSimpleJSON(simpleJSON.value)

	return newSimpleJSON, nil
}

func TreeNodeClone(args ...Value) (Value, error) {
	if len(args) != 2 {
		return nil, errors.New("clone requires 2 arguments: tree node and clone name")
	}

	treeNode, ok := args[0].(*TreeNodeImpl)
	if !ok {
		return nil, fmt.Errorf("expected TreeNodeImpl, got %T", args[0])
	}

	cloneName, ok := args[1].(Str)
	if !ok {
		return nil, fmt.Errorf("clone name must be a string, got %T", args[1])
	}

	// Create a new TreeNodeImpl and copy elements
	newTreeNode := NewTreeNode(string(cloneName))
	newTreeNode.Attributes = treeNode.Attributes

	// Recursively clone children
	for _, child := range treeNode.Children {
		clonedChild := child.Clone()
		newTreeNode.AddChild(clonedChild)
	}

	return newTreeNode, nil
}

// getPropXML - dynamic property access for XMLNode
// Checks XMLAttributes first, then content, then immediate children
func getPropXML(args ...Value) (Value, error) {
	if len(args) != 2 {
		return nil, errors.New("getProp requires 2 arguments: node and property path")
	}

	xmlNode, ok := args[0].(*XMLNode)
	if !ok {
		return nil, fmt.Errorf("expected XMLNode, got %T", args[0])
	}

	propPath, ok := args[1].(Str)
	if !ok {
		return nil, fmt.Errorf("property path must be a string, got %T", args[1])
	}

	path := string(propPath)
	parts := strings.Split(path, ".")

	// For simple path (no dots), check attributes then children
	if len(parts) == 1 {
		// Check XMLAttributes first
		if val, ok := xmlNode.XMLAttributes[path]; ok {
			return Str(val), nil
		}

		// Check for special content property
		if path == "content" || path == "_text" {
			return Str(xmlNode.Content), nil
		}

		// Check immediate children by name
		for _, child := range xmlNode.Children {
			if child.Name() == path {
				return child.(Value), nil
			}
		}

		return DBNull, nil
	}

	// For path with dots, navigate through children
	currentNode := xmlNode
	for i, part := range parts {
		// Check attributes at current level
		if i == len(parts)-1 {
			if val, ok := currentNode.XMLAttributes[part]; ok {
				return Str(val), nil
			}
			if part == "content" || part == "_text" {
				return Str(currentNode.Content), nil
			}
		}

		// Look for child with this name
		found := false
		for _, child := range currentNode.Children {
			if child.Name() == part {
				if i == len(parts)-1 {
					// Last part - return the child
					return child.(Value), nil
				}
				// Navigate deeper
				if xmlChild, ok := child.(*XMLNode); ok {
					currentNode = xmlChild
					found = true
					break
				}
				// Can't navigate deeper through non-XMLNode
				return DBNull, nil
			}
		}

		if !found {
			return DBNull, nil
		}
	}

	return DBNull, nil
}

// setPropXML - dynamic property setter for XMLNode
// Sets XMLAttributes (must be strings)
func setPropXML(args ...Value) (Value, error) {
	if len(args) != 3 {
		return nil, errors.New("setProp requires 3 arguments: node, property path, and value")
	}

	xmlNode, ok := args[0].(*XMLNode)
	if !ok {
		return nil, fmt.Errorf("expected XMLNode, got %T", args[0])
	}

	propPath, ok := args[1].(Str)
	if !ok {
		return nil, fmt.Errorf("property path must be a string, got %T", args[1])
	}

	value := args[2]

	path := string(propPath)

	// Handle special content property
	if path == "content" || path == "_text" {
		if str, ok := value.(Str); ok {
			xmlNode.Content = string(str)
			return value, nil
		}
		return nil, fmt.Errorf("XML content must be a string, got %T", value)
	}

	// For simple keys (no dots), set as XML attribute
	if !strings.Contains(path, ".") {
		if str, ok := value.(Str); ok {
			xmlNode.XMLAttributes[path] = string(str)
			return value, nil
		}
		return nil, fmt.Errorf("XML attributes must be strings, got %T", value)
	}

	// For paths with dots, we don't support nested setting in XMLNode
	return nil, fmt.Errorf("nested property setting not supported for XMLNode, use setProp on child nodes directly")
}

// getPropTreeNode - dynamic property access for TreeNodeImpl
// Checks attributes first, then immediate children by name
func getPropTreeNode(args ...Value) (Value, error) {
	if len(args) != 2 {
		return nil, errors.New("getProp requires 2 arguments: node and property path")
	}

	treeNode, ok := args[0].(*TreeNodeImpl)
	if !ok {
		return nil, fmt.Errorf("expected TreeNodeImpl, got %T", args[0])
	}

	propPath, ok := args[1].(Str)
	if !ok {
		return nil, fmt.Errorf("property path must be a string, got %T", args[1])
	}

	path := string(propPath)
	parts := strings.Split(path, ".")

	// For simple path, check attributes then children
	if len(parts) == 1 {
		// Check attributes first
		if treeNode.Attributes != nil {
			if val, ok := treeNode.Attributes[path]; ok {
				return val, nil
			}
		}

		// Check immediate children by name
		for _, child := range treeNode.Children {
			if child.Name() == path {
				return child.(Value), nil
			}
		}

		return DBNull, nil
	}

	// For paths with dots, only check first part in children
	firstPart := parts[0]
	for _, child := range treeNode.Children {
		if child.Name() == firstPart {
			// Can't navigate deeper without type information
			// User should use treeFind for complex navigation
			return DBNull, nil
		}
	}

	return DBNull, nil
}
