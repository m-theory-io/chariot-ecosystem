package chariot

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"

	cfg "github.com/bhouse1273/chariot-ecosystem/services/go-chariot/configs"
)

// RegisterValues registers all value-related functions
func RegisterValues(rt *Runtime) {
	// Variable declaration and manipulation
	rt.Register("declare", func(args ...Value) (Value, error) {
		if len(args) < 2 {
			return nil, errors.New("declare requires at least 2 arguments: name and type")
		}

		// Get variable name
		if tvar, ok := args[0].(ScopeEntry); ok {
			// If first argument is a ScopeEntry, use its value
			// This allows for variable references to be used as declare variable names
			args[0] = tvar.Value
		}

		name, ok := args[0].(Str)
		if !ok {
			return nil, errors.New("variable name must be a string")
		}

		// Get type
		if tvar, ok := args[1].(ScopeEntry); ok {
			// If second argument is a ScopeEntry, use its value
			args[1] = tvar.Value
		}
		typeStr, ok := args[1].(Str)
		if !ok {
			return nil, errors.New("type must be a string")
		}
		validType := isValidTypeCode(string(typeStr))
		if !validType {
			return nil, fmt.Errorf("invalid type specifier '%s'", typeStr)
		}

		// Get initial value if provided
		var initialValue Value
		if len(args) > 2 {
			if tvar, ok := args[2].(ScopeEntry); ok {
				// If third argument is a ScopeEntry, use its value
				initialValue = tvar.Value
			} else {
				// Otherwise, use the provided value directly
				initialValue = args[2]
			}
		} else {
			// Default values based on type
			var err error
			initialValue, err = defaultValue(string(typeStr))
			if err != nil {
				return nil, err
			}
		}

		// Validate type
		if err := validateTypeCompatibility(string(typeStr), initialValue); err != nil {
			return nil, err
		}

		// Set in CURRENT scope
		rt.CurrentScope().SetWithType(string(name), initialValue, string(typeStr))

		return initialValue, nil
	})

	rt.Register("declareGlobal", func(args ...Value) (Value, error) {
		if len(args) != 3 {
			return nil, fmt.Errorf("declareGlobal requires 3 arguments: variable name, type, initial value")
		}

		if tvar, ok := args[0].(ScopeEntry); ok {
			// If first argument is a ScopeEntry, use its value
			args[0] = tvar.Value
		}
		if tvar, ok := args[1].(ScopeEntry); ok {
			// If second argument is a ScopeEntry, use its value
			args[1] = tvar.Value
		}
		if tvar, ok := args[2].(ScopeEntry); ok {
			// If third argument is a ScopeEntry, use its value
			args[2] = tvar.Value
		}

		varName, ok := args[0].(Str)
		if !ok {
			return nil, fmt.Errorf("variable name must be a string")
		}

		varType, ok := args[1].(Str)
		if !ok {
			return nil, fmt.Errorf("variable type must be a string")
		}

		// Get initial value if provided
		var initialValue Value
		if len(args) > 2 {
			initialValue = args[2]
		} else {
			// Default values based on type
			var err error
			initialValue, err = defaultValue(string(varType))
			if err != nil {
				return nil, err
			}
		}

		// Same type checking logic as declare, but store in GLOBAL scope
		varNameStr := string(varName)
		typeStr := string(varType)

		// Type validation (same as declare)
		if err := validateTypeCompatibility(typeStr, initialValue); err != nil {
			return nil, err
		}
		// Store in GLOBAL scope (this is the key difference from declare)
		rt.GlobalScope().SetWithType(varNameStr, initialValue, typeStr)

		return initialValue, nil
	})

	// deleteFunction - delete a function from the runtime functions map
	rt.Register("deleteFunction", func(args ...Value) (Value, error) {
		if len(args) != 1 {
			return nil, errors.New("deleteFunction requires exactly 1 argument: function name")
		}
		funcName, ok := args[0].(Str)
		if !ok {
			return nil, errors.New("function name must be a string")
		}
		rt.DeleteFunction(string(funcName))
		return nil, nil
	})

	// inspectRuntime - inspect the current runtime state
	rt.Register("inspectRuntime", func(args ...Value) (Value, error) {
		// Construct a SimpleJSON of runtime state
		if len(args) != 0 {
			return nil, errors.New("inspectRuntime does not take any arguments")
		}
		// Create a map to hold the runtime state
		rtState := map[string]interface{}{
			"globals":          ConvertToNativeJSON(rt.ListGlobalVariables()),
			"variables":        ConvertToNativeJSON(rt.ListLocalVariables()),
			"objects":          ConvertToNativeJSON(rt.ListObjects()),
			"lists":            ConvertToNativeJSON(rt.ListLists()),
			"namespaces":       ConvertToNativeJSON(rt.ListNamespaces()),
			"nodes":            ConvertToNativeJSON(rt.ListNodes()),
			"tables":           ConvertToNativeJSON(rt.ListTables()),
			"keycolumns":       ConvertToNativeJSON(rt.ListKeyColumns()),
			"default_template": rt.DefaultTemplateID,
			"timeoffset":       rt.timeOffset,
		}

		return rtState, nil
	})

	// getVariable - get a variable value from the current scope
	rt.Register("getVariable", func(args ...Value) (Value, error) {
		if len(args) != 1 {
			return nil, errors.New("getVariable requires exactly 1 argument: variable name")
		}
		varName, ok := args[0].(Str)
		if !ok {
			return nil, errors.New("variable name must be a string")
		}

		val, ok := rt.GetVariable(string(varName))
		if !ok {
			return nil, fmt.Errorf("variable not found: %s", varName)
		}

		return val, nil
	})

	// OfferVariable construction closure
	rt.Register("offerVariable", func(args ...Value) (Value, error) {
		if len(args) < 2 {
			return nil, errors.New("offerVariable requires at least 2 arguments: value and format tag")
		}

		value := args[0]
		formatTag, ok := args[1].(Str)
		if !ok {
			return nil, errors.New("format tag must be a string")
		}

		return NewOfferVariable(value, string(formatTag)), nil
	})

	// Alias for offerVariable - offerVar
	rt.Register("offerVar", func(args ...Value) (Value, error) {
		return rt.funcs["offerVariable"](args...)
	})

	// In value_funcs.go
	rt.Register("function", func(args ...Value) (Value, error) {
		if len(args) < 1 {
			return nil, errors.New("function requires at least a body argument")
		}

		// Parse parameters (if provided)
		var params []string
		if len(args) > 1 {
			paramList, ok := args[0].(*ArrayValue)
			if !ok {
				return nil, errors.New("parameters must be an array")
			}

			for i := 0; i < paramList.Length(); i++ {
				param, ok := paramList.Get(i).(Str)
				if !ok {
					return nil, fmt.Errorf("parameter names must be strings")
				}
				params = append(params, string(param))
			}

			// Body is the second argument in this case
			bodyCode := args[1].(Str)
			// Parse body from string to AST
			body, err := rt.Parser.ParseCode(string(bodyCode))
			if err != nil {
				return nil, err
			}

			return &FunctionValue{
				Body:       body,
				Parameters: params,
				SourceCode: string(bodyCode),
				Scope:      rt.currentScope, // Capture current scope
			}, nil
		}

		// Simple case: just a body with no parameters
		bodyCode := args[0].(Str)
		body, err := rt.Parser.ParseCode(string(bodyCode))
		if err != nil {
			return nil, err
		}

		return &FunctionValue{
			Body:       body,
			Parameters: nil,
			SourceCode: string(bodyCode),
			Scope:      rt.currentScope,
		}, nil
	})

	// In value_funcs.go
	rt.Register("func", func(args ...Value) (Value, error) {
		return rt.funcs["function"](args...)
	})

	rt.Register("call", func(args ...Value) (Value, error) {
		if len(args) < 1 {
			return nil, errors.New("call requires at least a function argument")
		}

		// Get function value
		fn, ok := args[0].(*FunctionValue)
		if !ok {
			return nil, fmt.Errorf("expected function value, got %T", args[0])
		}

		// Pass remaining args to function
		return executeFunctionValue(rt, fn, args[1:])
	})

	rt.Register("listFunctions", func(args ...Value) (Value, error) {
		return rt.ListFunctions(), nil
	})

	rt.Register("getFunction", func(args ...Value) (Value, error) {
		if len(args) != 1 {
			return nil, errors.New("getFunction requires exactly 1 argument: function name")
		}
		funcName, ok := args[0].(Str)
		if !ok {
			return nil, errors.New("function name must be a string")
		}
		fn, exists := rt.GetFunction(string(funcName))
		if !exists {
			return nil, fmt.Errorf("function not found: %s", funcName)
		}
		return Str(PrettyPrintFunction(fn, string(funcName))), nil
	})

	// exists() test for the existence of a variable or object
	rt.Register("exists", func(args ...Value) (Value, error) {
		if len(args) != 1 {
			return nil, errors.New("exists requires exactly 1 argument: variable name")
		}
		varName, ok := args[0].(Str)
		if !ok {
			return nil, errors.New("variable name must be a string")
		}
		varNameStr := string(varName)
		// Check if variable exists in any scope
		_, exists := rt.GetVariable(varNameStr)
		if exists {
			return Bool(exists), nil
		}
		// check in objects
		if _, exists = rt.objects[varNameStr]; ok {
			return Bool(exists), nil
		}
		return Bool(exists), nil
	})

	rt.Register("mapValue", func(args ...Value) (Value, error) {
		// Create empty MapValue
		mapVal := NewMap()

		// If arguments provided, they should be key-value pairs
		if len(args)%2 != 0 {
			return nil, fmt.Errorf("mapValue requires even number of arguments (key-value pairs)")
		}

		// Add key-value pairs
		for i := 0; i < len(args); i += 2 {
			key, ok := args[i].(Str)
			if !ok {
				return nil, fmt.Errorf("map keys must be strings, got %T", args[i])
			}
			value := args[i+1]
			mapVal.Set(string(key), value)
		}

		return mapVal, nil
	})

	// merge a text string containing {} tokens with map values
	rt.Register("merge", func(args ...Value) (Value, error) {
		if len(args) < 2 {
			return nil, fmt.Errorf("merge requires 2-3 arguments: template string, map node, [profile, offer]")
		}

		template, ok := args[0].(Str)
		if !ok {
			return nil, fmt.Errorf("template must be a string")
		}

		node := args[1]
		var mnode AttributedType

		params := func() Value {
			if len(args) < 3 {
				return nil
			}
			return args[2]
		}()

		profile := func(params Value) Value {
			if tvar, ok := params.(*ArrayValue).Elements[0].(TreeNode); ok {
				return tvar
			}
			return nil
		}(params)
		if profile == nil {
			return nil, fmt.Errorf("profile must be provided")
		}

		offer := func(params Value) Value {
			if tvar, ok := params.(*ArrayValue).Elements[1].(TreeNode); ok {
				return tvar
			}
			return nil
		}(params)
		if offer == nil {
			if tvar, ok := params.(*ArrayValue).Elements[1].(*MapValue); ok {
				offer = tvar
			}
		}
		if offer == nil {
			return nil, fmt.Errorf("offer must be provided")
		}

		switch node.(type) {
		case *MapNode, *JSONNode, *MapValue, *TreeNodeImpl:
			mnode = node.(AttributedType) // Ensure node is an AttributedType
		default:
			return nil, fmt.Errorf("unsupported node type: %T", node)
		}

		// Perform the merge
		var err error
		var tvar Value
		merged := string(template)
		for key, value := range mnode.GetAttributes() {
			if tv, ok := value.(*FunctionValue); ok {
				// If value is a function, call it to set tvar
				tvar, err = executeFunctionValue(rt, tv, []Value{profile, offer})
				if err != nil {
					return nil, err
				}
			} else {
				// Otherwise, use the value directly
				tvar = value
			}
			// We assume that tvar is not a *FunctionValue
			switch tvar := tvar.(type) {
			case *OfferVariable:
				// If the value is an OfferVariable, we need to format it
				// according to its format tag
				switch tvar.FormatTag {
				case "uint":
					// Replace the {key} with the unsigned integer value
					merged = strings.ReplaceAll(merged, "{"+key+"}", fmt.Sprintf("%d", convertValueToUInt64(tvar.Value)))
				case "int", "integer":
					// Replace the {key} with the integer value
					merged = strings.ReplaceAll(merged, "{"+key+"}", fmt.Sprintf("%d", convertValueToInt64(tvar.Value)))
				case "float", "double":
					// Replace the {key} with the float value
					merged = strings.ReplaceAll(merged, "{"+key+"}", fmt.Sprintf("%.2f", convertValueToFloat64(tvar.Value)))
				case "currency":
					// Replace the {key} with the formatted value
					merged = strings.ReplaceAll(merged, "{"+key+"}", fmt.Sprintf("$%.2f", convertValueToFloat64(tvar.Value)))
				case "percent", "percentage":
					// Replace the {key} with the formatted value as percentage
					merged = strings.ReplaceAll(merged, "{"+key+"}", fmt.Sprintf("%.2f%%", convertValueToFloat64(tvar.Value)*100))
				default:
					// For any other format tag, use the string representation
					merged = strings.ReplaceAll(merged, "{"+key+"}", fmt.Sprintf("%v", tvar.Value))
				}
			case Str:
				merged = strings.ReplaceAll(merged, "{"+key+"}", string(tvar))
			case Number:
				merged = strings.ReplaceAll(merged, "{"+key+"}", fmt.Sprintf("%v", tvar))
			case Bool:
				// If the function returns a boolean, format it as "true" or "false"
				merged = strings.ReplaceAll(merged, "{"+key+"}", strconv.FormatBool(bool(tvar)))
			case *ArrayValue:
				// If the function returns an array, we can join its elements
				var elements []string
				for _, elem := range tvar.Elements {
					elements = append(elements, fmt.Sprintf("%v", elem))
				}
				merged = strings.ReplaceAll(merged, "{"+key+"}", strings.Join(elements, ", "))
			case *MapValue:
				// If the function returns a map, we can format it as a string
				var mapElements []string
				for mapKey, mapValue := range tvar.Values {
					mapElements = append(mapElements, fmt.Sprintf("%s: %v", mapKey, mapValue))
				}
				merged = strings.ReplaceAll(merged, "{"+key+"}", strings.Join(mapElements, ", "))
			case TreeNode:
				// If the function returns a TreeNode, we can use its string representation
				merged = strings.ReplaceAll(merged, "{"+key+"}", tvar.String())
			default:
				merged = strings.ReplaceAll(merged, "{"+key+"}", fmt.Sprintf("%v", tvar))
			}
		}
		return Str(merged), nil
	})

	rt.Register("toMapValue", func(args ...Value) (Value, error) {
		if len(args) != 1 {
			return nil, fmt.Errorf("toMapValue requires 1 argument")
		}

		switch v := args[0].(type) {
		case *MapNode:
			// Convert MapNode to MapValue
			mapVal := NewMap()
			for key, value := range v.Attributes {
				mapVal.Set(key, value)
			}
			return mapVal, nil

		case *JSONNode:
			// Convert JSONNode to MapValue if it represents an object
			jsonData := v.GetJSONValue()
			if objData, ok := jsonData.(map[string]interface{}); ok {
				mapVal := NewMap()
				for key, value := range objData {
					mapVal.Set(key, convertToChariotValue(value))
				}
				return mapVal, nil
			}
			return nil, fmt.Errorf("JSONNode does not represent an object")

		case *MapValue:
			// Already a MapValue, return as-is
			return v, nil

		default:
			return nil, fmt.Errorf("cannot convert %T to MapValue", v)
		}
	})

	rt.Register("setq", func(args ...Value) (Value, error) {
		// Check argument count
		if len(args) < 2 || len(args) > 4 {
			return nil, errors.New("setq requires 2-4 arguments: variable, value, [namespace, key]")
		}

		// Get variable name
		varName, ok := args[0].(Str)
		if !ok {
			return nil, errors.New("variable name must be a string")
		}

		// Get the value to set - ensure we unwrap any ScopeEntry if present
		valueToSet := args[1]
		if entry, ok := valueToSet.(ScopeEntry); ok {
			valueToSet = entry.Value
		}

		// Case 1: Simple variable assignment (2 args)
		if len(args) == 2 {
			varNameStr := string(varName)

			// First check if variable exists in any scope
			entry, found := rt.FindVariable(varNameStr)

			if found {
				// Variable exists - update it with type checking
				if entry.IsTyped && entry.TypeCode != TypeVariableExpr {
					// Check type compatibility
					if err := validateTypeCompatibility(entry.TypeCode, valueToSet); err != nil {
						return nil, fmt.Errorf("cannot assign to %s: %v", varNameStr, err)
					}
				}

				// Apply the change to the variable in its original scope
				rt.SetVariableInScope(varNameStr, valueToSet)
			} else {
				// Create new variable in current scope
				rt.CurrentScope().Set(varNameStr, valueToSet)
			}

			return valueToSet, nil
		}

		// Case 2: Namespace assignment (3-4 args)
		namespace, ok := args[2].(Str)
		if !ok {
			return nil, errors.New("namespace must be a string")
		}

		// Optional key for certain namespace types
		var key string
		if len(args) == 4 {
			if keyStr, ok := args[3].(Str); ok {
				key = string(keyStr)
			} else {
				return nil, errors.New("namespace key must be a string")
			}
		}

		// Handle different namespaces
		switch string(namespace) {
		case "xml":
			// Handle XML namespace
			xmlNode, err := rt.findXMLNode(string(varName))
			if err != nil {
				return nil, err
			}

			// Set XML node value
			switch valueNode := valueToSet.(type) {
			case Str:
				xmlNode.SetContent(string(valueNode))
			case TreeNode:
				// Handle node replacement
				if xmlParent := xmlNode.Parent(); xmlParent != nil {
					xmlParent.RemoveChild(xmlNode)
					xmlParent.AddChild(valueNode)
				}
			default:
				xmlNode.SetContent(fmt.Sprintf("%v", valueToSet))
			}

			return valueToSet, nil

		case "array":
			// Handle array namespace
			arr, err := rt.findArray(string(varName))
			if err != nil {
				return nil, err
			}

			if key == "" {
				return nil, errors.New("array operations require an index key")
			}

			// Convert key to index
			index, err := strconv.Atoi(key)
			if err != nil {
				return nil, fmt.Errorf("invalid array index: %s", key)
			}

			// Set array element
			if index < 0 || index >= arr.Length() {
				return nil, fmt.Errorf("array index out of bounds: %d", index)
			}

			_ = arr.Set(index, valueToSet)
			return valueToSet, nil

		case "table":
			// Handle table namespace
			tableRows, exists := rt.tables[string(varName)]
			if !exists {
				return nil, fmt.Errorf("unknown table: %s", varName)
			}

			if key == "" {
				return nil, errors.New("table operations require a row key")
			}

			// key is expected to be "rowIndex:columnName"
			parts := strings.SplitN(key, ":", 2)
			if len(parts) != 2 {
				return nil, errors.New("table key must be in the format 'rowIndex:columnName'")
			}
			rowIndex, err := strconv.Atoi(parts[0])
			if err != nil || rowIndex < 0 || rowIndex >= len(tableRows) {
				return nil, fmt.Errorf("invalid table row: %s", parts[0])
			}
			columnName := parts[1]

			// Set the value in the table cell
			tableRows[rowIndex][columnName] = valueToSet
			return valueToSet, nil
		case "object":
			// Handle host object namespace
			obj, exists := rt.objects[string(varName)]
			if !exists {
				return nil, fmt.Errorf("unknown object: %s", varName)
			}
			if key == "" {
				return nil, errors.New("object operations require a property key")
			}

			// If obj is a map[string]Value
			if m, ok := obj.(map[string]Value); ok {
				m[key] = valueToSet
				return valueToSet, nil
			}

			// If obj is a struct, use reflection
			rv := reflect.ValueOf(obj)
			if rv.Kind() == reflect.Ptr {
				rv = rv.Elem()
			}
			if rv.Kind() == reflect.Struct {
				field := rv.FieldByName(key)
				if field.IsValid() && field.CanSet() {
					val := reflect.ValueOf(valueToSet)
					if val.Type().AssignableTo(field.Type()) {
						field.Set(val)
						return valueToSet, nil
					}
					return nil, fmt.Errorf("type mismatch for field %s", key)
				}
				return nil, fmt.Errorf("field %s not found in object", key)
			}

			return nil, fmt.Errorf("unsupported object type for setq")
		default:
			return nil, fmt.Errorf("unknown namespace: %s", namespace)
		}
	})

	// symbol accepts a string variable name and returns the variable or an error if not found
	//    equivalent to a Chariot line with just a variable name, used for visual-dsl
	rt.Register("symbol", func(args ...Value) (Value, error) {
		if len(args) != 1 {
			return nil, errors.New("symbol requires 1 argument: string of variable name")
		}

		// Get variable name
		name, ok := args[0].(Str)
		if !ok {
			return nil, errors.New("variable name must be a string")
		}

		// Search for name in runtime current scope
		if tvar, exists := rt.CurrentScope().Get(string(name)); exists {
			return tvar, nil
		}

		// Search for name in runtime global scope
		if tvar, exists := rt.GlobalScope().Get(string(name)); exists {
			return tvar, nil
		}

		return nil, errors.New("variable name not found")

	})

	rt.Register("destroy", func(args ...Value) (Value, error) {
		if len(args) != 1 {
			return nil, errors.New("destroy requires 1 argument: variable name")
		}

		// Get variable name
		name, ok := args[0].(Str)
		if !ok {
			return nil, errors.New("variable name must be a string")
		}

		// Remove from current scope if present
		if _, exists := rt.CurrentScope().Get(string(name)); exists {
			rt.CurrentScope().Delete(string(name))
			return DBNull, nil
		}

		// Remove from global scope if present
		if _, exists := rt.GlobalScope().Get(string(name)); exists {
			rt.GlobalScope().Delete(string(name))
			return DBNull, nil
		}

		return nil, fmt.Errorf("variable '%s' not found", name)
	})

	// Type functions
	rt.Register("typeOf", func(args ...Value) (Value, error) {
		if len(args) != 1 {
			return nil, errors.New("typeOf requires 1 argument")
		}

		// Unwrap argument
		arg := args[0]
		if tvar, ok := arg.(ScopeEntry); ok {
			arg = tvar.Value
		}

		return Str(GetValueTypeSpec(arg)), nil
	})

	rt.Register("valueType", func(args ...Value) (Value, error) {
		if len(args) != 1 {
			return nil, errors.New("valueType requires 1 argument")
		}

		// Unwrap argument
		arg := args[0]
		if tvar, ok := arg.(ScopeEntry); ok {
			arg = tvar.Value
		}

		// Get the actual underlying type by examining the Go type directly
		switch arg.(type) {
		case Number:
			return Str("N"), nil
		case Str:
			return Str("S"), nil
		case Bool:
			return Str("L"), nil
		case *ArrayValue:
			return Str("A"), nil
		case *MapValue:
			return Str("M"), nil
		case *TableValue:
			return Str("R"), nil
		case *HostObjectValue:
			return Str("H"), nil
		case *FunctionValue:
			return Str("F"), nil
		case *JSONNode:
			return Str("J"), nil
		case *XMLNode:
			return Str("X"), nil
		case TreeNode, *TreeNodeImpl, *Transform:
			return Str("T"), nil
		case nil:
			return Str("V"), nil
		default:
			// For unknown types, return the Go type name as a fallback
			return Str(fmt.Sprintf("%T", arg)), nil
		}
	})

	rt.Register("valueOf", func(args ...Value) (Value, error) {
		if len(args) < 1 {
			return nil, errors.New("valueOf requires at least 1 argument")
		}

		// If no type specified, just return the value
		if len(args) == 1 {
			return args[0], nil
		}

		// Get target type
		typeStr, ok := args[1].(Str)
		if !ok {
			return nil, errors.New("type must be a string")
		}

		// Convert based on target type
		switch typeStr {
		case "N", "number":
			// Convert to number
			switch v := args[0].(type) {
			case Number:
				return v, nil
			case Str:
				num, err := strconv.ParseFloat(string(v), 64)
				if err != nil {
					return Number(0), nil
				}
				return Number(num), nil
			case Bool:
				if bool(v) {
					return Number(1), nil
				}
				return Number(0), nil
			default:
				return Number(0), nil
			}
		case "S", "string":
			// Convert to string
			return Str(fmt.Sprintf("%v", args[0])), nil
		case "B", "boolean":
			// Convert to boolean
			switch v := args[0].(type) {
			case Bool:
				return v, nil
			case Number:
				return Bool(v != 0), nil
			case Str:
				return Bool(v != ""), nil
			default:
				return Bool(false), nil
			}
		default:
			return args[0], nil
		}
	})

	rt.Register("boolean", func(args ...Value) (Value, error) {
		if len(args) != 1 {
			return nil, errors.New("boolean requires 1 argument")
		}

		switch v := args[0].(type) {
		case Bool:
			return v, nil
		case Number:
			return Bool(v != 0), nil
		case Str:
			s := strings.ToLower(string(v))
			return Bool(s == "true" || s == "yes" || s == "1" || s == "y" || s == "on"), nil
		default:
			return Bool(false), nil
		}
	})

	// Null testing
	rt.Register("isNull", func(args ...Value) (Value, error) {
		if len(args) != 1 {
			return nil, errors.New("isNull requires 1 argument")
		}

		return Bool(args[0] == DBNull), nil
	})

	rt.Register("isNumeric", func(args ...Value) (Value, error) {
		if len(args) != 1 {
			return nil, fmt.Errorf("isNumeric requires 1 argument")
		}

		str := string(args[0].(Str))
		_, err := strconv.ParseUint(str, 10, 64)
		return Bool(err == nil), nil
	})

	rt.Register("empty", func(args ...Value) (Value, error) {
		if len(args) != 1 {
			return nil, errors.New("empty requires 1 argument")
		}

		switch v := args[0].(type) {
		case Str:
			return Bool(v == ""), nil
		case Number:
			return Bool(v == 0), nil
		default:
			return Bool(args[0] == DBNull), nil
		}
	})

	rt.Register("saveFunctions", func(args ...Value) (Value, error) {
		if len(args) != 1 {
			return nil, errors.New("saveFunctions requires a filename")
		}
		filename, ok := args[0].(Str)
		if !ok {
			return nil, errors.New("filename must be a string")
		}
		err := SaveFunctionsToFile(rt.functions, string(filename))
		if err != nil {
			return nil, err
		}
		return Bool(true), nil
	})

	rt.Register("loadFunctions", func(args ...Value) (Value, error) {
		if len(args) != 1 {
			return nil, errors.New("loadFunctions requires a filename")
		}
		filename, ok := args[0].(Str)
		if !ok {
			return nil, errors.New("filename must be a string")
		}
		funcs, err := LoadFunctionsFromFile(string(filename))
		if err != nil {
			return nil, err
		}
		for name, fn := range funcs {
			rt.RegisterFunction(name, fn)
		}
		return Bool(true), nil
	})

	rt.Register("registerFunction", func(args ...Value) (Value, error) {
		if len(args) < 2 || len(args) > 3 {
			return nil, errors.New("registerFunction requires 2-3 arguments: name, function, [formatted_source]")
		}
		name, ok := args[0].(Str)
		if !ok {
			return nil, errors.New("name must be a string")
		}
		fn, ok := args[1].(*FunctionValue)
		if !ok {
			return nil, errors.New("function must be a FunctionValue")
		}

		// Optional third argument: formatted source from editor
		if len(args) == 3 {
			if formattedSrc, ok := args[2].(Str); ok {
				fn.FormattedSource = string(formattedSrc)
			}
		}

		rt.RegisterFunction(string(name), fn)
		return Bool(true), nil
	})

	rt.Register("setValue", func(args ...Value) (Value, error) {
		if len(args) != 3 {
			return nil, errors.New("setValue requires 3 arguments: array, index, value")
		}
		arr, ok := args[0].(*ArrayValue)
		if !ok {
			return nil, errors.New("first argument must be an array")
		}
		index, ok := args[1].(Number)
		if !ok {
			return nil, errors.New("second argument must be a number (index)")
		}
		intIndex := int(index)
		if intIndex < 0 {
			return nil, fmt.Errorf("index cannot be negative")
		}
		value := args[2]

		// Expand array if needed
		for arr.Length() <= intIndex {
			arr.Append(DBNull)
		}
		_ = arr.Set(intIndex, value)
		return value, nil
	})

	rt.Register("hasMeta", func(args ...Value) (Value, error) {
		if len(args) != 2 {
			return nil, errors.New("hasMeta requires 2 arguments: doc, metaKey")
		}

		// Check if first arg is MetadataHolder or TreeNode
		var metaExists bool
		if holder, ok := args[0].(interface{ HasMeta(string) bool }); ok {
			metaKey := string(args[1].(Str))
			metaExists = holder.HasMeta(metaKey)
		} else {
			metaExists = false
		}

		return Bool(metaExists), nil
	})

	// toString function
	rt.Register("toString", func(args ...Value) (Value, error) {
		if len(args) != 1 {
			return nil, errors.New("toString requires 1 argument")
		}

		// Convert value to string representation
		val := args[0]
		switch v := val.(type) {
		case Str:
			return v, nil
		case Number:
			return Str(fmt.Sprintf("%v", v)), nil
		case Bool:
			return Str(fmt.Sprintf("%v", v)), nil
		case *ArrayValue:
			elements := make([]string, v.Length())
			for i := 0; i < v.Length(); i++ {
				elements[i] = fmt.Sprintf("%v", v.Get(i))
			}
			return Str(strings.Join(elements, ", ")), nil
		case *MapValue:
			pairs := make([]string, 0, len(v.Values))
			for k, val := range v.Values {
				pairs = append(pairs, fmt.Sprintf("%s: %v", k, val))
			}
			return Str("{" + strings.Join(pairs, ", ") + "}"), nil
		default:
			return Str(fmt.Sprintf("%v", val)), nil
		}
	})

	// toNumber function
	rt.Register("toNumber", func(args ...Value) (Value, error) {
		if len(args) != 1 {
			return nil, errors.New("toNumber requires 1 argument")
		}

		val := args[0]
		switch v := val.(type) {
		case Number:
			return v, nil
		case Str:
			num, err := strconv.ParseFloat(string(v), 64)
			if err != nil {
				return Number(0), nil // or handle error as needed
			}
			return Number(num), nil
		case Bool:
			if v {
				return Number(1), nil
			}
			return Number(0), nil
		default:
			return Number(0), nil // or handle other types as needed
		}
	})

	// toBool function
	rt.Register("toBool", func(args ...Value) (Value, error) {
		if len(args) != 1 {
			return nil, errors.New("toBool requires 1 argument")
		}

		val := args[0]
		switch v := val.(type) {
		case Bool:
			return v, nil
		case Number:
			return Bool(v != 0), nil
		case Str:
			s := strings.ToLower(string(v))
			return Bool(s == "true" || s == "yes" || s == "1"), nil
		default:
			return Bool(false), nil // or handle other types as needed
		}
	})
}

func FunctionValueToMap(fn *FunctionValue) map[string]interface{} {
	return map[string]interface{}{
		"_value_type":      "function",
		"parameters":       fn.Parameters,
		"body":             fn.Body.ToMap(),
		"source":           fn.SourceCode,      // Original formatted source
		"formatted_source": fn.FormattedSource, // Add this field for editor formatting
	}
}

// Place this in a shared utils file or in value_funcs.go if needed
func ConvertToNativeJSON(val interface{}) interface{} {
	// Unwrap ScopeEntry and Value wrappers recursively
	for {
		switch v := val.(type) {
		case ScopeEntry:
			val = v.Value
		case *ScopeEntry:
			val = v.Value
		case Value:
			val = v
		default:
			break
		}
		break
	}

	// Handle DBNull and Null
	if val == DBNull || val == nil {
		return nil
	}
	// Handle True/False as bool
	if val == true || val == Bool(true) {
		return true
	}
	if val == false || val == Bool(false) {
		return false
	}
	switch v := val.(type) {
	case Bool:
		return bool(v)
	case bool:
		return v
	case Str:
		// Optionally, convert "true"/"false" strings to bools
		s := strings.ToLower(string(v))
		if s == "true" {
			return true
		}
		if s == "false" {
			return false
		}
		return string(v)
	case Number:
		return float64(v)
	case *ArrayValue:
		arr := make([]interface{}, len(v.Elements))
		for i, elem := range v.Elements {
			arr[i] = ConvertToNativeJSON(elem)
		}
		return arr
	case *MapValue:
		m := make(map[string]interface{})
		for k, v2 := range v.Values {
			m[k] = ConvertToNativeJSON(v2)
		}
		return m
	case *OfferVariable:
		return map[string]interface{}{
			"_type":      "offer_variable",
			"value":      ConvertToNativeJSON(v.Value),
			"format_tag": v.FormatTag,
		}
	case *JSONNode:
		// Assuming JSONNode has a method to get its value as a map
		jsonData := v.GetJSONValue()
		if objData, ok := jsonData.(map[string]interface{}); ok {
			m := make(map[string]interface{})
			for key, value := range objData {
				m[key] = ConvertToNativeJSON(value)
			}
			return m
		}
		if arrData, ok := jsonData.([]interface{}); ok {
			return ConvertToNativeJSON(arrData)
		}
		return nil // or handle other JSON types as needed
	case *Transform:
		// Assuming Transform has a method to get its attributes as a map
		m := make(map[string]interface{})
		for key, value := range v.GetAttributes() {
			m[key] = ConvertToNativeJSON(value)
		}
		// Collection children
		for _, child := range v.Children {
			m[child.Name()] = ConvertToNativeJSON(child)
		}
		return m
	case *TreeNodeImpl:
		// Assuming TreeNodeImpl has a method to get its attributes as a map
		m := make(map[string]interface{})
		for key, value := range v.GetAttributes() {
			m[key] = ConvertToNativeJSON(value)
		}
		// Collection children
		for _, child := range v.Children {
			m[child.Name()] = ConvertToNativeJSON(child)
		}
		return m
	case map[string]Value:
		m := make(map[string]interface{})
		for k, v2 := range v {
			m[k] = ConvertToNativeJSON(v2)
		}
		return m
	case map[string]interface{}:
		m := make(map[string]interface{})
		for k, v2 := range v {
			m[k] = ConvertToNativeJSON(v2)
		}
		return m
	case []interface{}:
		arr := make([]interface{}, len(v))
		for i, elem := range v {
			arr[i] = ConvertToNativeJSON(elem)
		}
		return arr
	case []Value:
		arr := make([]interface{}, len(v))
		for i, elem := range v {
			arr[i] = ConvertToNativeJSON(elem)
		}
		return arr
	case map[string]map[string]Value:
		m := make(map[string]interface{})
		for k, v2 := range v {
			subMap := make(map[string]interface{})
			for subKey, subValue := range v2 {
				subMap[subKey] = ConvertToNativeJSON(subValue)
			}
			m[k] = subMap
		}
		return m
	case map[string]TreeNode:
		m := make(map[string]interface{})
		for k, v2 := range v {
			m[k] = ConvertToNativeJSON(v2)
		}
		return m
	case map[string][]map[string]Value:
		m := make(map[string]interface{})
		for k, v2 := range v {
			arr := make([]interface{}, len(v2))
			for i, elem := range v2 {
				arr[i] = ConvertToNativeJSON(elem)
			}
			m[k] = arr
		}
		return m
	case map[string]string:
		m := make(map[string]interface{})
		for k, v2 := range v {
			// Convert string values to interface{}
			m[k] = v2
		}
		return m
	default:
		return fmt.Sprintf("%v", v)
	}
}

// Helper function to check type compatibility
func ensureTypeCompatibility(existing, newValue Value) error {
	switch existing.(type) {
	case Number:
		if _, ok := newValue.(Number); !ok {
			return fmt.Errorf("type mismatch: expected number, got %T", newValue)
		}
	case Str:
		if _, ok := newValue.(Str); !ok {
			return fmt.Errorf("type mismatch: expected string, got %T", newValue)
		}
	case Bool:
		if _, ok := newValue.(Bool); !ok {
			return fmt.Errorf("type mismatch: expected boolean, got %T", newValue)
		}
	case *ArrayValue:
		if _, ok := newValue.(*ArrayValue); !ok {
			return fmt.Errorf("type mismatch: expected array, got %T", newValue)
		}
	}
	return nil
}

// Update your declare/declareGlobal functions
func validateTypeCompatibility(typeStr string, value Value) error {
	switch typeStr {
	case TypeNumber:
		if _, ok := value.(Number); !ok {
			return fmt.Errorf("type mismatch: expected number, got %T", value)
		}
	case TypeString:
		if _, ok := value.(Str); !ok {
			return fmt.Errorf("type mismatch: expected string, got %T", value)
		}
	case TypeBoolean:
		if _, ok := value.(Bool); !ok {
			return fmt.Errorf("type mismatch: expected boolean, got %T", value)
		}
	case TypeArray:
		if _, ok := value.(*ArrayValue); !ok {
			return fmt.Errorf("type mismatch: expected array, got %T", value)
		}
	case TypeMap:
		if _, ok := value.(*MapValue); !ok {
			return fmt.Errorf("type mismatch: expected map, got %T", value)
		}
	case TypeJSON:
		if _, ok := value.(*JSONNode); !ok {
			return fmt.Errorf("type mismatch: expected JSON, got %T", value)
		}
	case TypeObject:
		// Host Object -accept only references to registered HostObjects
		if _, ok := value.(*HostObjectValue); !ok {
			return fmt.Errorf("type mismatch: expected host object, got %T", value)
		}
	case TypeXML:
		if _, ok := value.(*XMLNode); !ok {
			return fmt.Errorf("type mismatch: expected XML, got %T", value)
		}
	case TypeTree:
		if _, ok := value.(*TreeNode); !ok {
			if _, ok := value.(*TreeNodeImpl); !ok {
				if _, ok := value.(*Transform); !ok {
					return fmt.Errorf("type mismatch: expected tree, got %T", value)
				}
			}
		}
	case TypeDate:
		if str, ok := value.(Str); ok {
			ok = IsDateString(string(str))
			if !ok {
				return fmt.Errorf("type mismatch: expected date, got %T", value)
			}
		}
	case TypeFunction:
		if _, ok := value.(*FunctionValue); !ok {
			return fmt.Errorf("type mismatch: expected function, got %T", value)
		}
	case TypePlan:
		if _, ok := value.(*Plan); !ok {
			return fmt.Errorf("type mismatch: expected plan, got %T", value)
		}
	case "V", "any":
		// Accept any type
		return nil
	default:
		return fmt.Errorf("unknown type specifier '%s'", typeStr)
	}
	return nil
}

func LoadFunctionsFromFile(filename string) (map[string]*FunctionValue, error) {
	// Use TreePath from config
	basePath := cfg.ChariotConfig.TreePath
	fullPath := filepath.Join(basePath, filename)

	data, err := os.ReadFile(fullPath)
	if err != nil {
		return nil, err
	}
	var funcsMap map[string]interface{}
	if err := json.Unmarshal(data, &funcsMap); err != nil {
		return nil, err
	}
	functions := make(map[string]*FunctionValue)
	for key, fnRaw := range funcsMap {
		fnMap, ok := fnRaw.(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf("function '%s' is not a valid object", key)
		}
		fn, err := MapToFunctionValue(fnMap)
		if err != nil {
			return nil, err
		}
		functions[key] = fn
	}
	return functions, nil
}

func SaveFunctionsToFile(functions map[string]*FunctionValue, filename string) error {
	// Use TreePath from config
	basePath := cfg.ChariotConfig.TreePath
	fullPath := filepath.Join(basePath, filename)

	// Ensure directory exists
	dir := filepath.Dir(fullPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	// Serialize functions (using your FunctionValueToMap helper)
	funcsList := make(map[string]interface{})
	for name, fn := range functions {
		fnMap := FunctionValueToMap(fn)
		funcsList[name] = fnMap
	}
	data, err := json.MarshalIndent(funcsList, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(fullPath, data, 0644)
}

// MapToFunctionValue reconstructs a FunctionValue from a map[string]interface{} as loaded from JSON/YAML.
func MapToFunctionValue(fnMap map[string]interface{}) (*FunctionValue, error) {
	fn := &FunctionValue{}

	// Parameters: expect []interface{} of strings
	if params, ok := fnMap["parameters"].([]interface{}); ok {
		for _, p := range params {
			if ps, ok := p.(string); ok {
				fn.Parameters = append(fn.Parameters, ps)
			}
		}
	}

	// Source code (optional)
	if src, ok := fnMap["source"].(string); ok {
		fn.SourceCode = src
	}

	// Formatted source (preserve editor formatting)
	if formattedSrc, ok := fnMap["formatted_source"].(string); ok {
		fn.FormattedSource = formattedSrc
	}

	// Body (AST)
	if body, ok := fnMap["body"]; ok {
		bodyMap, ok := body.(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf("function body is not a map; got %T", body)
		}
		node, err := NodeFromMap(bodyMap)
		if err != nil {
			return nil, fmt.Errorf("failed to reconstruct function body: %w", err)
		}
		fn.Body = node
	} else {
		return nil, fmt.Errorf("function is missing 'body' field")
	}

	// Optionally: set Scope to nil (cannot reconstruct closures from JSON)
	fn.Scope = nil

	return fn, nil
}

// Add this helper function to value_funcs.go
func defaultValue(typeStr string) (Value, error) {
	switch typeStr {
	case TypeNumber:
		return Number(0), nil
	case TypeString:
		return Str(""), nil
	case TypeBoolean:
		return Bool(false), nil
	case TypeArray:
		return NewArray(), nil
	case TypeMap:
		return NewMap(), nil
	case TypeJSON:
		// Create empty JSON object
		return NewJSONNode("{}"), nil
	case TypeXML:
		// Create empty XML node (if you have this constructor)
		return NewXMLNode("root"), nil
	case TypeTree:
		// Create empty tree node
		return NewTreeNode("root"), nil
	case TypeObject:
		// HostObject reference - return empty HostObjectValue
		return DBNull, nil
	case TypeVariableExpr:
		// Any type - return null as default
		return DBNull, nil
	case TypePlan:
		// Plans should be constructed via plan(...); default to null
		return DBNull, nil
	default:
		return nil, fmt.Errorf("unknown type specifier '%s'", typeStr)
	}
}

// Optional: function to convert ValueType to type specifier string
func ValueTypeToSpec(vt ValueType) string {
	switch vt {
	case ValueNumber:
		return "N"
	case ValueString:
		return "S"
	case ValueBoolean:
		return "L"
	case ValueArray:
		return "A"
	case ValueMap:
		return "M"
	case ValueTable:
		return "R"
	case ValueObject:
		return "O"
	case ValueHostObject:
		return "H"
	case ValueNode:
		return "T"
	case ValueFunction:
		return "F"
	case ValueJSON:
		return "J"
	case ValueXML:
		return "X"
	case ValueNil:
		return "V"
	case ValueVariableExpr:
		return "V"
	default:
		return "V"
	}
}

func PrettyPrintFunction(fn *FunctionValue, name string) string {
	// If we have preserved formatted source, use it
	if fn.FormattedSource != "" {
		return fn.FormattedSource
	}

	// Otherwise fall back to AST reconstruction
	params := strings.Join(fn.Parameters, ", ")
	body := PrettyPrintNode(fn.Body, "    ")
	return fmt.Sprintf("function %s(%s) {\n%s}", name, params, body)
}

func PrettyPrintNode(node Node, indent string) string {
	switch n := node.(type) {
	case *Block:
		var sb strings.Builder
		for _, stmt := range n.Stmts {
			sb.WriteString(indent)
			sb.WriteString(PrettyPrintNode(stmt, indent+"    "))
			sb.WriteString("\n")
		}
		return sb.String()
	case *FuncCall:
		args := make([]string, len(n.Args))
		for i, arg := range n.Args {
			args[i] = PrettyPrintNode(arg, "")
		}
		return fmt.Sprintf("%s(%s)", n.Name, strings.Join(args, ", "))
	case *Literal:
		switch v := n.Val.(type) {
		case Str:
			return fmt.Sprintf("'%s'", strings.ReplaceAll(string(v), "'", "\\'"))
		case string:
			return fmt.Sprintf("'%s'", strings.ReplaceAll(v, "'", "\\'"))
		default:
			return fmt.Sprintf("%v", v)
		}
	case *VarRef:
		return n.Name
	case *FunctionDefNode:
		params := strings.Join(n.Parameters, ", ")
		body := PrettyPrintNode(n.Body, indent+"    ")
		return fmt.Sprintf("func(%s) {\n%s%s}", params, indent+"    ", body)
	case *IfNode:
		cond := PrettyPrintNode(n.Condition, "")
		var thenBlockSb, elseBlockSb strings.Builder
		for _, stmt := range n.TrueBranch {
			thenBlockSb.WriteString(indent + "    ")
			thenBlockSb.WriteString(PrettyPrintNode(stmt, indent+"    "))
			thenBlockSb.WriteString("\n")
		}
		var elseBlock string
		if len(n.FalseBranch) > 0 {
			for _, stmt := range n.FalseBranch {
				elseBlockSb.WriteString(indent + "    ")
				elseBlockSb.WriteString(PrettyPrintNode(stmt, indent+"    "))
				elseBlockSb.WriteString("\n")
			}
			elseBlock = fmt.Sprintf("\n%selse {\n%s}", indent, elseBlockSb.String())
		}
		return fmt.Sprintf("if(%s) {\n%s}%s", cond, thenBlockSb.String(), elseBlock)
		// Add more node types as needed, e.g. *WhileNode, *SwitchNode, etc.
	case *WhileNode:
		cond := PrettyPrintNode(n.Condition, "")
		var bodySb strings.Builder
		for _, stmt := range n.Body {
			bodySb.WriteString(indent + "    ")
			bodySb.WriteString(PrettyPrintNode(stmt, indent+"    "))
			bodySb.WriteString("\n")
		}
		return fmt.Sprintf("while(%s) {\n%s}", cond, bodySb.String())
	case *SwitchNode:
		return n.ToString()
	default:
		return "<unknown node>"
	}
}
