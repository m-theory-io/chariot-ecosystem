package chariot

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

// === YAML SPECIFIC ===
func registerYAMLFileOps(rt *Runtime) {
	rt.Register("loadYAML", func(args ...Value) (Value, error) {
		if len(args) != 1 {
			return nil, errors.New("loadYAML requires 1 argument: filepath")
		}

		// Unwrap arguments
		for i, arg := range args {
			if tvar, ok := arg.(ScopeEntry); ok {
				args[i] = tvar.Value
			}
		}

		filepath, ok := args[0].(Str)
		if !ok {
			return nil, fmt.Errorf("filepath must be a string, got %T", args[0])
		}

		fileNameStr := string(filepath)
		// Get secure path
		fullPath, err := getSecureFilePath(fileNameStr, "data")
		if err != nil {
			return nil, err
		}

		// Read file from disk
		data, err := os.ReadFile(fullPath)
		if err != nil {
			return nil, fmt.Errorf("failed to read file '%s': %v", fileNameStr, err)
		}

		// Parse YAML and create JSONNode (YAML has same structure as JSON)
		var yamlData interface{}
		if err := yaml.Unmarshal(data, &yamlData); err != nil {
			return nil, fmt.Errorf("failed to parse YAML from '%s': %v", fileNameStr, err)
		}

		// Create JSONNode and populate it (YAML data is compatible with JSON structure)
		node := NewJSONNode("yaml_loaded")
		node.SetJSONValue(yamlData)

		return node, nil
	})

	rt.Register("saveYAML", func(args ...Value) (Value, error) {
		if len(args) != 2 {
			return nil, errors.New("saveYAML requires 2 arguments: node and filepath")
		}

		// Unwrap arguments
		for i, arg := range args {
			if tvar, ok := arg.(ScopeEntry); ok {
				args[i] = tvar.Value
			}
		}

		jsonNode, ok := args[0].(*JSONNode)
		if !ok {
			return nil, fmt.Errorf("first argument must be a JSONNode, got %T", args[0])
		}

		filepath, ok := args[1].(Str)
		if !ok {
			return nil, fmt.Errorf("filepath must be a string, got %T", args[1])
		}

		fileNameStr := string(filepath)
		// Get secure path
		fullPath, err := getSecureFilePath(fileNameStr, "data")
		if err != nil {
			return nil, err
		}

		// Get data from JSONNode
		data := jsonNode.GetJSONValue()

		// Convert to YAML
		yamlBytes, err := yaml.Marshal(data)
		if err != nil {
			return nil, fmt.Errorf("failed to convert data to YAML: %v", err)
		}

		// Write to file
		if err := os.WriteFile(fullPath, yamlBytes, 0644); err != nil {
			return nil, fmt.Errorf("failed to write YAML file '%s': %v", fileNameStr, err)
		}

		return Bool(true), nil
	})

	rt.Register("loadYAMLRaw", func(args ...Value) (Value, error) {
		if len(args) != 1 {
			return nil, errors.New("loadYAMLRaw requires 1 argument: filepath")
		}

		// Unwrap arguments
		for i, arg := range args {
			if tvar, ok := arg.(ScopeEntry); ok {
				args[i] = tvar.Value
			}
		}

		filepath, ok := args[0].(Str)
		if !ok {
			return nil, fmt.Errorf("filepath must be a string, got %T", args[0])
		}

		fileNameStr := string(filepath)
		// Get secure path
		fullPath, err := getSecureFilePath(fileNameStr, "data")
		if err != nil {
			return nil, err
		}

		// Read file and return as YAML string (no parsing)
		data, err := os.ReadFile(fullPath)
		if err != nil {
			return nil, fmt.Errorf("failed to read YAML file '%s': %v", fileNameStr, err)
		}

		// Validate it's valid YAML
		var temp interface{}
		if err := yaml.Unmarshal(data, &temp); err != nil {
			return nil, fmt.Errorf("file '%s' contains invalid YAML: %v", filepath, err)
		}

		return Str(string(data)), nil
	})

	rt.Register("saveYAMLRaw", func(args ...Value) (Value, error) {
		if len(args) != 2 {
			return nil, errors.New("saveYAMLRaw requires 2 arguments: yaml_string and filepath")
		}

		// Unwrap arguments
		for i, arg := range args {
			if tvar, ok := arg.(ScopeEntry); ok {
				args[i] = tvar.Value
			}
		}

		yamlStr, ok := args[0].(Str)
		if !ok {
			return nil, fmt.Errorf("first argument must be a YAML string, got %T", args[0])
		}

		filepath, ok := args[1].(Str)
		if !ok {
			return nil, fmt.Errorf("filepath must be a string, got %T", args[1])
		}

		fileNameStr := string(filepath)
		// Get secure path
		fullPath, err := getSecureFilePath(fileNameStr, "data")
		if err != nil {
			return nil, err
		}

		// Validate YAML string
		var temp interface{}
		if err := yaml.Unmarshal([]byte(string(yamlStr)), &temp); err != nil {
			return nil, fmt.Errorf("invalid YAML string: %v", err)
		}

		// Write raw YAML to file
		if err := os.WriteFile(fullPath, []byte(string(yamlStr)), 0644); err != nil {
			return nil, fmt.Errorf("failed to write YAML file '%s': %v", fileNameStr, err)
		}

		return Bool(true), nil
	})

	rt.Register("loadYAMLMultiDoc", func(args ...Value) (Value, error) {
		if len(args) != 1 {
			return nil, errors.New("loadYAMLMultiDoc requires 1 argument: filepath")
		}

		// Unwrap arguments
		for i, arg := range args {
			if tvar, ok := arg.(ScopeEntry); ok {
				args[i] = tvar.Value
			}
		}

		filepath, ok := args[0].(Str)
		if !ok {
			return nil, fmt.Errorf("filepath must be a string, got %T", args[0])
		}

		fileNameStr := string(filepath)

		// Validate YAML file extension
		if !isValidYAMLFile(fileNameStr) {
			return nil, fmt.Errorf("file must have .yaml or .yml extension, got '%s'", fileNameStr)
		}

		// Get secure path
		fullPath, err := getSecureFilePath(fileNameStr, "data")
		if err != nil {
			return nil, err
		}

		// Read file
		data, err := os.ReadFile(fullPath)
		if err != nil {
			return nil, fmt.Errorf("failed to read YAML file '%s': %v", fileNameStr, err)
		}

		// Parse multiple YAML documents
		decoder := yaml.NewDecoder(strings.NewReader(string(data)))
		var documents []interface{}

		for {
			var doc interface{}
			if err := decoder.Decode(&doc); err != nil {
				if err.Error() == "EOF" {
					break
				}
				return nil, fmt.Errorf("failed to parse YAML document: %v", err)
			}
			documents = append(documents, doc)
		}

		// Create ArrayValue with JSONNodes for each document
		arr := NewArray()
		for i, doc := range documents {
			node := NewJSONNode(fmt.Sprintf("doc%d", i))
			node.SetJSONValue(doc)
			arr.Append(node)
		}

		return arr, nil
	})

	rt.Register("saveYAMLMultiDoc", func(args ...Value) (Value, error) {
		if len(args) != 2 {
			return nil, errors.New("saveYAMLMultiDoc requires 2 arguments: array and filepath")
		}

		// Unwrap arguments
		for i, arg := range args {
			if tvar, ok := arg.(ScopeEntry); ok {
				args[i] = tvar.Value
			}
		}

		// Accept either ArrayValue or JSONNode containing array
		var documents []interface{}

		switch arr := args[0].(type) {
		case *ArrayValue:
			// Convert ArrayValue elements to native values
			for i := 0; i < arr.Length(); i++ {
				elem := arr.Get(i)
				if jsonNode, ok := elem.(*JSONNode); ok {
					documents = append(documents, jsonNode.GetJSONValue())
				} else {
					documents = append(documents, ConvertToNativeJSON(elem))
				}
			}
		case *JSONNode:
			// Try to get array from JSONNode
			data := arr.GetJSONValue()
			if arrData, ok := data.([]interface{}); ok {
				documents = arrData
			} else {
				return nil, fmt.Errorf("JSONNode must contain an array for multi-document YAML, got %T", data)
			}
		default:
			return nil, fmt.Errorf("first argument must be an ArrayValue or JSONNode containing array, got %T", args[0])
		}

		filepath, ok := args[1].(Str)
		if !ok {
			return nil, fmt.Errorf("filepath must be a string, got %T", args[1])
		}

		fileNameStr := string(filepath)
		// Validate YAML file extension
		if !isValidYAMLFile(fileNameStr) {
			return nil, fmt.Errorf("file must have .yaml or .yml extension, got '%s'", fileNameStr)
		}
		// Get secure path
		fullPath, err := getSecureFilePath(fileNameStr, "data")
		if err != nil {
			return nil, err
		}

		// Create file
		file, err := os.Create(string(fullPath))
		if err != nil {
			return nil, fmt.Errorf("failed to create YAML file '%s': %v", fileNameStr, err)
		}
		defer file.Close()

		// Write each document separated by ---
		encoder := yaml.NewEncoder(file)
		defer encoder.Close()

		for _, doc := range documents {
			if err := encoder.Encode(doc); err != nil {
				return nil, fmt.Errorf("failed to encode YAML document: %v", err)
			}
		}

		return Bool(true), nil
	})
}
