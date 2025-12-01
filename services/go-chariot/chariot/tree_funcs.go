package chariot

import (
	"errors"
	"fmt"
	"path/filepath"
	"reflect"
	"strings"
	"time"
)

// RegisterTree registers all tree-related functions
func RegisterTreeFunctions(rt *Runtime) {
	// TreeNode creation helpers (newTree is canonical, treeNode kept for backwards compatibility)
	createTreeFunc := func(args ...Value) (Value, error) {
		if len(args) != 1 {
			return nil, fmt.Errorf("tree creation requires exactly one argument")
		}

		for i, arg := range args {
			if tvar, ok := arg.(ScopeEntry); ok {
				args[i] = tvar.Value
			}
		}

		name, ok := args[0].(Str)
		if !ok {
			return nil, fmt.Errorf("tree name must be a string, got %T", args[0])
		}

		if name == "" {
			return nil, fmt.Errorf("tree name cannot be empty")
		}

		return NewTreeNode(string(name)), nil
	}

	rt.Register("newTree", createTreeFunc)
	rt.Register("treeNode", createTreeFunc)

	// treeSave(treeNode, filename) - saves tree to JSON file
	rt.Register("treeSave", func(args ...Value) (Value, error) {
		if len(args) < 2 || len(args) > 4 {
			return nil, errors.New("treeSave requires 2-4 arguments: treeNode, filename, [format], [compression]")
		}

		// Unwrap scope entries
		for i, arg := range args {
			if tvar, ok := arg.(ScopeEntry); ok {
				args[i] = tvar.Value
			}
		}

		// Get tree node
		node, ok := args[0].(TreeNode)
		if !ok {
			return nil, fmt.Errorf("first argument must be a TreeNode, got %T", args[0])
		}

		// Get filename
		filename, ok := args[1].(Str)
		if !ok {
			return nil, fmt.Errorf("second argument must be a string filename, got %T", args[1])
		}

		// Parse options
		options := &SerializationOptions{
			Format:      "json", // default
			Compression: false,  // default
			PrettyPrint: true,   // default
		}

		if len(args) > 2 {
			if format, ok := args[2].(Str); ok {
				options.Format = string(format)
			}
		}

		if len(args) > 3 {
			if compress, ok := args[3].(Bool); ok {
				options.Compression = bool(compress)
			}
		}

		// Use global service with timeout
		service := getTreeSerializerService()
		err := service.SaveAsync(node, string(filename), options, 30*time.Second)
		if err != nil {
			return nil, fmt.Errorf("failed to save tree: %v", err)
		}

		return Bool(true), nil
	})

	// treeLoad(filename)
	rt.Register("treeLoad", func(args ...Value) (Value, error) {
		if len(args) != 1 {
			return nil, errors.New("treeLoad requires 1 argument: filename")
		}

		// Unwrap scope entries
		for i, arg := range args {
			if tvar, ok := arg.(ScopeEntry); ok {
				args[i] = tvar.Value
			}
		}

		// Get filename
		filename, ok := args[0].(Str)
		if !ok {
			return nil, fmt.Errorf("argument must be a string filename, got %T", args[0])
		}

		// Detect format from extension
		ext := strings.ToLower(filepath.Ext(string(filename)))
		format := "json" // default
		switch ext {
		case ".gob":
			format = "gob"
		case ".json":
			format = "json"
		case ".xml":
			format = "xml"
		case ".yaml", ".yml":
			format = "yaml"
		}
		_ = format

		// Use global service with timeout
		service := getTreeSerializerService()
		node, err := service.LoadAsync(string(filename), rt, 30*time.Second)
		if err != nil {
			return nil, fmt.Errorf("failed to load tree: %v", err)
		}

		return node, nil
	})

	// treeToXML(treeNode, [prettyPrint]) - leverages existing XML functionality
	rt.Register("treeToXML", func(args ...Value) (Value, error) {
		if len(args) < 1 || len(args) > 2 {
			return nil, errors.New("treeToXML requires 1-2 arguments: treeNode, [prettyPrint]")
		}

		// Unwrap scope entries
		for i, arg := range args {
			if tvar, ok := arg.(ScopeEntry); ok {
				args[i] = tvar.Value
			}
		}

		// Get tree node
		node, ok := args[0].(TreeNode)
		if !ok {
			return nil, fmt.Errorf("first argument must be a TreeNode, got %T", args[0])
		}

		// Get pretty print option
		prettyPrint := true // default
		if len(args) > 1 {
			if pretty, ok := args[1].(Bool); ok {
				prettyPrint = bool(pretty)
			}
		}

		// Use existing serializer directly for immediate conversion
		serializer := NewTreeNodeSerializer()
		var xmlStr string
		var err error

		if prettyPrint {
			xmlStr, err = serializer.ToXMLPretty(node)
		} else {
			xmlStr, err = serializer.ToXML(node)
		}

		if err != nil {
			return nil, fmt.Errorf("failed to convert to XML: %v", err)
		}

		return Str(xmlStr), nil
	})

	// treeToYAML(treeNode) - leverages existing YAML functionality
	rt.Register("treeToYAML", func(args ...Value) (Value, error) {
		if len(args) != 1 {
			return nil, errors.New("treeToYAML requires 1 argument: treeNode")
		}

		// Unwrap scope entries
		for i, arg := range args {
			if tvar, ok := arg.(ScopeEntry); ok {
				args[i] = tvar.Value
			}
		}

		// Get tree node
		node, ok := args[0].(TreeNode)
		if !ok {
			return nil, fmt.Errorf("argument must be a TreeNode, got %T", args[0])
		}

		// Use existing serializer directly
		serializer := NewTreeNodeSerializer()
		yamlStr, err := serializer.ToYAML(node)
		if err != nil {
			return nil, fmt.Errorf("failed to convert to YAML: %v", err)
		}

		return Str(yamlStr), nil
	})

	// treeSaveSecure(treeNode, filename, encryptionKeyID, signingKeyID, watermark [, options])
	rt.Register("treeSaveSecure", func(args ...Value) (Value, error) {
		if len(args) < 5 || len(args) > 6 {
			return nil, errors.New("treeSaveSecure requires 5-6 arguments: treeNode, filename, encryptionKeyID, signingKeyID, watermark, [options]")
		}

		// Unwrap scope entries
		for i, arg := range args {
			if tvar, ok := arg.(ScopeEntry); ok {
				args[i] = tvar.Value
			}
		}

		// Get tree node
		node, ok := args[0].(TreeNode)
		if !ok {
			return nil, fmt.Errorf("first argument must be a TreeNode, got %T", args[0])
		}

		// Get filename
		filename, ok := args[1].(Str)
		if !ok {
			return nil, fmt.Errorf("second argument must be a string filename, got %T", args[1])
		}

		// Get encryption key ID
		encryptionKeyID, ok := args[2].(Str)
		if !ok {
			return nil, fmt.Errorf("third argument must be a string encryptionKeyID, got %T", args[2])
		}

		// Get signing key ID
		signingKeyID, ok := args[3].(Str)
		if !ok {
			return nil, fmt.Errorf("fourth argument must be a string signingKeyID, got %T", args[3])
		}

		// Get watermark
		watermark, ok := args[4].(Str)
		if !ok {
			return nil, fmt.Errorf("fifth argument must be a string watermark, got %T", args[4])
		}

		// Parse options (optional)
		options := &SecureSerializationOptions{
			EncryptionKeyID:   string(encryptionKeyID),
			SigningKeyID:      string(signingKeyID),
			VerificationKeyID: string(signingKeyID), // Default to same as signing
			Watermark:         string(watermark),
			Checksum:          true, // Default enabled
			CompressionLevel:  9,    // Default high compression
			AuditTrail:        true, // Default enabled
		}

		if len(args) > 5 {
			// Parse options map if provided
			if optMap, ok := args[5].(MapValue); ok {
				if val, exists := optMap.Values["verificationKeyID"]; exists {
					if vkid, ok := val.(Str); ok {
						options.VerificationKeyID = string(vkid)
					}
				}
				if val, exists := optMap.Values["checksum"]; exists {
					if checksum, ok := val.(Bool); ok {
						options.Checksum = bool(checksum)
					}
				}
				if val, exists := optMap.Values["auditTrail"]; exists {
					if audit, ok := val.(Bool); ok {
						options.AuditTrail = bool(audit)
					}
				}
				if val, exists := optMap.Values["compressionLevel"]; exists {
					if level, ok := val.(Number); ok {
						options.CompressionLevel = int(level)
					}
				}
			}
		}

		// Use global service
		service := getTreeSerializerService()
		err := service.SaveSecureAgent(node, string(filename), options)
		if err != nil {
			return nil, fmt.Errorf("failed to save secure tree: %v", err)
		}

		return Bool(true), nil
	})

	// treeLoadSecure(filename, decryptionKeyID, verificationKeyID)
	rt.Register("treeLoadSecure", func(args ...Value) (Value, error) {
		if len(args) != 3 {
			return nil, errors.New("treeLoadSecure requires 3 arguments: filename, decryptionKeyID, verificationKeyID")
		}

		// Unwrap scope entries
		for i, arg := range args {
			if tvar, ok := arg.(ScopeEntry); ok {
				args[i] = tvar.Value
			}
		}

		// Get filename
		filename, ok := args[0].(Str)
		if !ok {
			return nil, fmt.Errorf("first argument must be a string filename, got %T", args[0])
		}

		// Get decryption key ID
		decryptionKeyID, ok := args[1].(Str)
		if !ok {
			return nil, fmt.Errorf("second argument must be a string decryptionKeyID, got %T", args[1])
		}

		// Get verification key ID
		verificationKeyID, ok := args[2].(Str)
		if !ok {
			return nil, fmt.Errorf("third argument must be a string verificationKeyID, got %T", args[2])
		}

		// Create options struct to match service API
		options := &SecureDeserializationOptions{
			DecryptionKeyID:   string(decryptionKeyID),
			VerificationKeyID: string(verificationKeyID),
			RequireSignature:  true, // Default to requiring signature
			AuditTrail:        true, // Default enabled
		}

		// Use global service with the correct API
		service := getTreeSerializerService()
		node, err := service.LoadSecureAgent(string(filename), options)
		if err != nil {
			return nil, fmt.Errorf("failed to load secure tree: %v", err)
		}

		return node, nil
	})

	// treeValidateSecure(filename, verificationKeyID)
	rt.Register("treeValidateSecure", func(args ...Value) (Value, error) {
		if len(args) != 2 {
			return nil, errors.New("treeValidateSecure requires 2 arguments: filename, verificationKeyID")
		}

		// Unwrap scope entries
		for i, arg := range args {
			if tvar, ok := arg.(ScopeEntry); ok {
				args[i] = tvar.Value
			}
		}

		// Get filename
		filename, ok := args[0].(Str)
		if !ok {
			return nil, fmt.Errorf("first argument must be a string filename, got %T", args[0])
		}

		// Get verification key ID
		verificationKeyID, ok := args[1].(Str)
		if !ok {
			return nil, fmt.Errorf("second argument must be a string verificationKeyID, got %T", args[1])
		}

		// Use global service
		service := getTreeSerializerService()
		isValid, err := service.ValidateSecureAgent(string(filename), string(verificationKeyID))
		if err != nil {
			return nil, fmt.Errorf("failed to validate secure tree: %v", err)
		}

		return Bool(isValid), nil
	})

	// treeGetMetadata(filename) - Get metadata without loading/decrypting
	rt.Register("treeGetMetadata", func(args ...Value) (Value, error) {
		if len(args) != 1 {
			return nil, errors.New("treeGetMetadata requires 1 argument: filename")
		}

		// Unwrap scope entries
		for i, arg := range args {
			if tvar, ok := arg.(ScopeEntry); ok {
				args[i] = tvar.Value
			}
		}

		// Get filename
		filename, ok := args[0].(Str)
		if !ok {
			return nil, fmt.Errorf("argument must be a string filename, got %T", args[0])
		}

		// Use global service
		service := getTreeSerializerService()
		metadata, err := service.GetMetadata(string(filename))
		if err != nil {
			return nil, fmt.Errorf("failed to get metadata: %v", err)
		}

		// Convert metadata map to MapValue
		values := make(map[string]Value)
		for k, v := range metadata {
			values[k] = convertToValue(v) // Helper function to convert interface{} to Value
		}

		return MapValue{Values: values}, nil
	})

	// treeFind function - returns all matching records
	rt.Register("treeFind", func(args ...Value) (Value, error) {
		// New semantics:
		// treeFind(forest, attrName, value [, operator])
		// treeFind(attrName, value [, operator])  // implicit forest from runtime variables
		if len(args) < 2 || len(args) > 4 {
			return nil, errors.New("treeFind requires 2-4 arguments: [forest,] attributeName, value, [operator]")
		}

		// Unwrap scope entries
		for i, arg := range args {
			if tvar, ok := arg.(ScopeEntry); ok {
				args[i] = tvar.Value
			}
		}

		var (
			forest    Value
			attrName  Str
			searchVal Value
			operator  = "="
		)

		// Determine signature
		if s, ok := args[0].(Str); ok {
			// No forest provided; build forest from runtime variables
			attrName = s
			if len(args) < 2 {
				return nil, errors.New("missing value argument")
			}
			searchVal = args[1]
			if len(args) > 2 {
				if op, ok := args[2].(Str); ok {
					operator = string(op)
				}
			}

			// Collect all tree-like candidates from runtime variables
			vars := rt.ListVariables()
			forestArr := NewArray()
			var collectTrees func(Value)
			collectTrees = func(v Value) {
				if v == nil {
					return
				}
				switch tv := v.(type) {
				case ScopeEntry:
					collectTrees(tv.Value)
				case *ArrayValue:
					for i := 0; i < tv.Length(); i++ {
						collectTrees(tv.Get(i))
					}
				case []Value:
					for _, e := range tv {
						collectTrees(e)
					}
				case map[string]Value:
					for _, mv := range tv {
						collectTrees(mv)
					}
				case *JSONNode:
					forestArr.Append(tv)
					for _, ch := range tv.GetChildren() {
						collectTrees(ch)
					}
				case TreeNode:
					forestArr.Append(tv)
					for _, ch := range tv.GetChildren() {
						collectTrees(ch)
					}
				default:
					// Ignore other types
				}
			}
			for _, val := range vars {
				collectTrees(val)
			}
			forest = forestArr
		} else {
			// forest is provided explicitly
			forest = args[0]
			if s2, ok := args[1].(Str); ok {
				attrName = s2
			} else {
				return nil, fmt.Errorf("attribute name must be a string, got %T", args[1])
			}
			if len(args) < 3 {
				return nil, errors.New("missing value argument")
			}
			searchVal = args[2]
			if len(args) > 3 {
				if op, ok := args[3].(Str); ok {
					operator = string(op)
				}
			}
		}

		results := NewArray()

		// Deduplicate by pointer identity where possible
		seen := make(map[uintptr]struct{})
		ptrKey := func(v Value) (uintptr, bool) {
			switch tv := v.(type) {
			case *JSONNode:
				return reflect.ValueOf(tv).Pointer(), true
			case *MapValue:
				return reflect.ValueOf(tv).Pointer(), true
			case TreeNode:
				rv := reflect.ValueOf(tv)
				if rv.Kind() == reflect.Ptr {
					return rv.Pointer(), true
				}
			case map[string]Value:
				rv := reflect.ValueOf(tv)
				if rv.Kind() == reflect.Map {
					return rv.Pointer(), true
				}
			}
			return 0, false
		}

		tryAdd := func(candidate Value) {
			// Only consider top-level tree-like candidates
			switch candidate.(type) {
			case *MapNode, *JSONNode, TreeNode, TreeNodeImpl, *MapValue, map[string]Value:
				if anyMatchInValue(candidate, string(attrName), searchVal, operator) {
					if key, ok := ptrKey(candidate); ok {
						if _, exists := seen[key]; exists {
							return
						}
						seen[key] = struct{}{}
					}
					results.Append(candidate)
				}
			}
		}

		var walkForest func(Value)
		walkForest = func(f Value) {
			switch t := f.(type) {
			case *ArrayValue:
				for i := 0; i < t.Length(); i++ {
					walkForest(t.Get(i))
				}
			case *MapValue:
				tryAdd(t)
				for _, v := range t.GetAttributes() {
					walkForest(v)
				}
			case *MapNode:
				walkForest(t.Attributes)
			case []Value:
				for _, e := range t {
					walkForest(e)
				}
			case map[string]Value:
				tryAdd(t)
				for _, v := range t {
					walkForest(v)
				}
			default:
				tryAdd(t)
			}
		}

		walkForest(forest)
		return results, nil
	})

	// treeWalk function
	rt.Register("treeWalk", func(args ...Value) (Value, error) {
		if len(args) != 2 {
			return nil, errors.New("treeWalk requires 2 arguments: node and function")
		}

		// Unwrap arguments
		for i, arg := range args {
			if tvar, ok := arg.(ScopeEntry); ok {
				args[i] = tvar.Value
			}
		}

		// Get the root node
		rootData := args[0]

		// Get the function to apply
		fn, ok := args[1].(*FunctionValue)
		if !ok {
			return nil, fmt.Errorf("second argument must be a function, got %T", args[1])
		}

		// Get the call function
		callFn, exists := rt.funcs["call"]
		if !exists {
			return nil, errors.New("call function not found")
		}

		// Recursive walk function that visits ALL nodes and values
		var walkNode func(Value) error
		walkNode = func(node Value) error {
			// Call the function on this node
			_, err := callFn(fn, node)
			if err != nil {
				return err
			}

			// Recurse based on node type
			switch v := node.(type) {
			case *JSONNode:
				// Walk through all attributes (this is what was missing!)
				for _, attrValue := range v.GetAttributes() {
					if err := walkNode(attrValue); err != nil {
						return err
					}
				}

				// Walk through children (if any)
				for _, child := range v.GetChildren() {
					if err := walkNode(child); err != nil {
						return err
					}
				}

			case *ArrayValue:
				// Walk through array elements
				for i := 0; i < v.Length(); i++ {
					elem := v.Get(i)
					if err := walkNode(elem); err != nil {
						return err
					}
				}

			case *MapValue:
				for _, mapValue := range v.GetAttributes() {
					if err := walkNode(mapValue); err != nil {
						return err
					}
				}

			case TreeNode:
				// Walk through attributes
				for _, attrValue := range v.GetAttributes() {
					if err := walkNode(attrValue); err != nil {
						return err
					}
				}

				// Walk through children
				for _, child := range v.GetChildren() {
					if err := walkNode(child); err != nil {
						return err
					}
				}

			case map[string]Value:
				// Walk through map values
				for _, mapValue := range v {
					if err := walkNode(mapValue); err != nil {
						return err
					}
				}

			case []Value:
				// Walk through slice elements
				for _, elem := range v {
					if err := walkNode(elem); err != nil {
						return err
					}
				}
			}

			return nil
		}

		// Start the walk
		err := walkNode(rootData)
		if err != nil {
			return nil, err
		}

		return rootData, nil
	})

	// Update the treeSearch function in tree_funcs.go
	rt.Register("treeSearch", func(args ...Value) (Value, error) {
		if len(args) < 3 || len(args) > 5 {
			return nil, errors.New("treeSearch requires 3-5 arguments: node, attributeName, value, [operator], [existsOnly]")
		}

		// Unwrap arguments
		for i, arg := range args {
			if tvar, ok := arg.(ScopeEntry); ok {
				args[i] = tvar.Value
			}
		}

		rootData := args[0]
		attrName, ok := args[1].(Str)
		if !ok {
			return nil, fmt.Errorf("attribute name must be a string, got %T", args[1])
		}

		searchValue := args[2]

		// Default operator is "="
		operator := "="
		if len(args) > 3 {
			if op, ok := args[3].(Str); ok {
				operator = string(op)
			}
		}

		// Optional existsOnly short-circuit (5th arg)
		existsOnly := false
		if len(args) > 4 {
			if b, ok := args[4].(Bool); ok {
				existsOnly = bool(b)
			} else {
				return nil, fmt.Errorf("existsOnly argument (arg[4]) must be boolean, got %T", args[4])
			}
		}
		if existsOnly {
			if anyMatchInValue(rootData, string(attrName), searchValue, operator) {
				return Bool(true), nil
			}
			return Bool(false), nil
		}

		results := NewArray()

		var searchInValue func(Value)
		searchInValue = func(val Value) {
			switch v := val.(type) {
			case map[string]Value:
				// Check if this map has the attribute we're looking for
				if attrValue, exists := v[string(attrName)]; exists {
					if compareValuesOp(attrValue, searchValue, operator) {
						results.Append(v)
					}
				}

				// Search recursively in all map values
				for _, mapVal := range v {
					searchInValue(mapVal)
				}

			case *MapValue:
				if attrValue, exists := v.GetAttribute(string(attrName)); exists {
					if compareValuesOp(attrValue, searchValue, operator) {
						results.Append(v)
					}
				}
				for _, mapVal := range v.GetAttributes() {
					searchInValue(mapVal)
				}

			case *ArrayValue:
				// Search in array elements
				for i := 0; i < v.Length(); i++ {
					elem := v.Get(i)
					searchInValue(elem)
				}

			case *JSONNode:
				// Check JSONNode attributes first
				if attrValue, exists := v.GetAttribute(string(attrName)); exists {
					if compareValuesOp(attrValue, searchValue, operator) {
						results.Append(v)
					}
				}

				// Check JSONNode array data (e.g., "_company" -> departments array)
				aKey := "_" + v.Name()
				if arrayValue, exists := v.GetAttribute(aKey); exists {
					searchInValue(arrayValue)
				}

				// Also search through all other attributes for nested data
				for attrKey, attrVal := range v.GetAttributes() {
					if attrKey != aKey { // Don't double-search the array key
						searchInValue(attrVal)
					}
				}

				// Search in children
				for _, child := range v.GetChildren() {
					searchInValue(child)
				}

			case TreeNode:
				// Check TreeNode attributes
				if attrValue, exists := v.GetAttribute(string(attrName)); exists {
					if compareValuesOp(attrValue, searchValue, operator) {
						results.Append(v)
					}
				}

				// Search in all TreeNode attributes
				for _, attrVal := range v.GetAttributes() {
					searchInValue(attrVal)
				}

				// Search in TreeNode children
				for _, child := range v.GetChildren() {
					searchInValue(child)
				}

			case []Value:
				// Handle native Go slice
				for _, elem := range v {
					searchInValue(elem)
				}

			default:
				// For any other types, try to see if they have the attribute
				// This handles cases where nested objects might be other Value types
			}
		}

		searchInValue(rootData)
		return results, nil
	})

}

// Helper functions to compare values
func valuesEqual(a, b Value) bool {
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
	return false
}

// Helper function for comparison operations
func compareValuesOp(a, b Value, operator string) bool {
	switch operator {
	case "=", "==":
		return valuesEqual(a, b)
	case "!=", "<>":
		return !valuesEqual(a, b)
	case ">":
		return compareNumeric(a, b, func(x, y float64) bool { return x > y })
	case ">=":
		return compareNumeric(a, b, func(x, y float64) bool { return x >= y })
	case "<":
		return compareNumeric(a, b, func(x, y float64) bool { return x < y })
	case "<=":
		return compareNumeric(a, b, func(x, y float64) bool { return x <= y })
	case "contains":
		return stringIncludes(a, b)
	case "startswith":
		return stringStartsWith(a, b)
	case "endswith":
		return stringEndsWith(a, b)
	default:
		return valuesEqual(a, b)
	}
}

func compareNumeric(a, b Value, compare func(float64, float64) bool) bool {
	numA, okA := a.(Number)
	numB, okB := b.(Number)
	if okA && okB {
		return compare(float64(numA), float64(numB))
	}
	return false
}

func stringIncludes(a, b Value) bool {
	strA, okA := a.(Str)
	strB, okB := b.(Str)
	if okA && okB {
		return strings.Contains(string(strA), string(strB))
	}
	return false
}

func stringStartsWith(a, b Value) bool {
	strA, okA := a.(Str)
	strB, okB := b.(Str)
	if okA && okB {
		return strings.HasPrefix(string(strA), string(strB))
	}
	return false
}

func stringEndsWith(a, b Value) bool {
	strA, okA := a.(Str)
	strB, okB := b.(Str)
	if okA && okB {
		return strings.HasSuffix(string(strA), string(strB))
	}
	return false
}

// Helper: determine if any nested value in val satisfies attrName OP searchValue, with short-circuit
func anyMatchInValue(val Value, attrName string, searchValue Value, operator string) bool {
	switch v := val.(type) {
	case map[string]Value:
		if attrValue, exists := v[attrName]; exists {
			if compareValuesOp(attrValue, searchValue, operator) {
				return true
			}
		}
		for _, mv := range v {
			if anyMatchInValue(mv, attrName, searchValue, operator) {
				return true
			}
		}
	case *MapValue:
		if attrValue, exists := v.GetAttribute(attrName); exists {
			if compareValuesOp(attrValue, searchValue, operator) {
				return true
			}
		}
		for _, mv := range v.GetAttributes() {
			if anyMatchInValue(mv, attrName, searchValue, operator) {
				return true
			}
		}
	case *ArrayValue:
		for i := 0; i < v.Length(); i++ {
			if anyMatchInValue(v.Get(i), attrName, searchValue, operator) {
				return true
			}
		}
	case *JSONNode:
		if attrValue, exists := v.GetAttribute(attrName); exists {
			if compareValuesOp(attrValue, searchValue, operator) {
				return true
			}
		}
		if arrayValue, exists := v.GetAttribute("_" + v.Name()); exists {
			if anyMatchInValue(arrayValue, attrName, searchValue, operator) {
				return true
			}
		}
		for k, av := range v.GetAttributes() {
			if k != "_"+v.Name() {
				if anyMatchInValue(av, attrName, searchValue, operator) {
					return true
				}
			}
		}
		for _, ch := range v.GetChildren() {
			if anyMatchInValue(ch, attrName, searchValue, operator) {
				return true
			}
		}
	case TreeNode:
		if attrValue, exists := v.GetAttribute(attrName); exists {
			if compareValuesOp(attrValue, searchValue, operator) {
				return true
			}
		}
		for _, av := range v.GetAttributes() {
			if anyMatchInValue(av, attrName, searchValue, operator) {
				return true
			}
		}
		for _, ch := range v.GetChildren() {
			if anyMatchInValue(ch, attrName, searchValue, operator) {
				return true
			}
		}
	case []Value:
		for _, e := range v {
			if anyMatchInValue(e, attrName, searchValue, operator) {
				return true
			}
		}
	}
	return false
}
