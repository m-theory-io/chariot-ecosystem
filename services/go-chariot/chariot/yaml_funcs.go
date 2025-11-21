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

		// Create JSONNode with array of documents
		node := NewJSONNode("yaml_multidoc")
		node.SetJSONValue(documents)

		return node, nil
	})

	rt.Register("saveYAMLMultiDoc", func(args ...Value) (Value, error) {
		if len(args) != 2 {
			return nil, errors.New("saveYAMLMultiDoc requires 2 arguments: array_node and filepath")
		}

		// Unwrap arguments
		for i, arg := range args {
			if tvar, ok := arg.(ScopeEntry); ok {
				args[i] = tvar.Value
			}
		}

		jsonNode, ok := args[0].(*JSONNode)
		if !ok {
			return nil, fmt.Errorf("first argument must be a JSONNode containing an array, got %T", args[0])
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

		// Get array data
		data := jsonNode.GetJSONValue()
		documents, ok := data.([]interface{})
		if !ok {
			return nil, fmt.Errorf("JSONNode must contain an array for multi-document YAML, got %T", data)
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
