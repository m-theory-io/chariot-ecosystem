package chariot

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// === FORMAT CONVERSIONS ===
func RegisterFormatConversions(rt *Runtime) {
	// JSON to YAML conversion
	rt.Register("jsonToYAML", func(args ...Value) (Value, error) {
		if len(args) != 1 {
			return nil, errors.New("jsonToYAML requires 1 argument: JSONNode")
		}

		// Unwrap arguments
		for i, arg := range args {
			if tvar, ok := arg.(ScopeEntry); ok {
				args[i] = tvar.Value
			}
		}

		jsonNode, ok := args[0].(*JSONNode)
		if !ok {
			return nil, fmt.Errorf("argument must be a JSONNode, got %T", args[0])
		}

		// Get the underlying JSON data
		jsonData := jsonNode.GetJSONValue()

		// Convert to YAML using the yaml library
		yamlBytes, err := yaml.Marshal(jsonData)
		if err != nil {
			return nil, fmt.Errorf("failed to convert JSON to YAML: %v", err)
		}

		return Str(string(yamlBytes)), nil
	})

	// YAML to JSON conversion
	rt.Register("yamlToJSON", func(args ...Value) (Value, error) {
		if len(args) != 1 {
			return nil, errors.New("yamlToJSON requires 1 argument: YAML string")
		}

		// Unwrap arguments
		for i, arg := range args {
			if tvar, ok := arg.(ScopeEntry); ok {
				args[i] = tvar.Value
			}
		}

		yamlStr, ok := args[0].(Str)
		if !ok {
			return nil, fmt.Errorf("argument must be a string containing YAML, got %T", args[0])
		}

		// Parse YAML into Go data structure
		var yamlData interface{}
		if err := yaml.Unmarshal([]byte(string(yamlStr)), &yamlData); err != nil {
			return nil, fmt.Errorf("failed to parse YAML: %v", err)
		}

		// Create JSONNode and populate it
		jsonNode := NewJSONNode("converted")
		jsonNode.SetJSONValue(yamlData)

		return jsonNode, nil
	})

	// JSON to YAML Node conversion (creates a YAMLNode if you implement one)
	rt.Register("jsonToYAMLNode", func(args ...Value) (Value, error) {
		if len(args) != 1 {
			return nil, errors.New("jsonToYAMLNode requires 1 argument: JSONNode")
		}

		// Unwrap arguments
		for i, arg := range args {
			if tvar, ok := arg.(ScopeEntry); ok {
				args[i] = tvar.Value
			}
		}

		jsonNode, ok := args[0].(*JSONNode)
		if !ok {
			return nil, fmt.Errorf("argument must be a JSONNode, got %T", args[0])
		}

		// Get the underlying JSON data
		jsonData := jsonNode.GetJSONValue()

		// Create YAMLNode (you'll need to implement this similar to JSONNode)
		// For now, return a JSONNode with YAML-compatible data
		yamlNode := NewJSONNode("yaml_converted")
		yamlNode.SetJSONValue(jsonData)

		return yamlNode, nil
	})

	// YAML Node to JSON conversion
	rt.Register("yamlToJSONNode", func(args ...Value) (Value, error) {
		if len(args) != 1 {
			return nil, errors.New("yamlToJSONNode requires 1 argument: YAMLNode or JSONNode")
		}

		// Unwrap arguments
		for i, arg := range args {
			if tvar, ok := arg.(ScopeEntry); ok {
				args[i] = tvar.Value
			}
		}

		// Handle both YAMLNode and JSONNode (since they're structurally similar)
		var sourceData interface{}

		switch node := args[0].(type) {
		case *JSONNode:
			sourceData = node.GetJSONValue()
		// case *YAMLNode:  // When you implement YAMLNode
		//     sourceData = node.GetYAMLValue()
		default:
			return nil, fmt.Errorf("argument must be a JSONNode or YAMLNode, got %T", args[0])
		}

		// Create new JSONNode
		jsonNode := NewJSONNode("yaml_to_json")
		jsonNode.SetJSONValue(sourceData)

		return jsonNode, nil
	})

	// Convert JSON file to YAML file directly
	rt.Register("convertJSONFileToYAML", func(args ...Value) (Value, error) {
		if len(args) != 2 {
			return nil, errors.New("convertJSONFileToYAML requires 2 arguments: input_json_path, output_yaml_path")
		}

		// Unwrap arguments
		for i, arg := range args {
			if tvar, ok := arg.(ScopeEntry); ok {
				args[i] = tvar.Value
			}
		}

		inputPath, ok := args[0].(Str)
		if !ok {
			return nil, fmt.Errorf("input path must be a string, got %T", args[0])
		}
		inputPathStr := string(inputPath)
		// Validate JSON file extension
		if !isValidJSONFile(inputPathStr) {
			return nil, fmt.Errorf("file must have .json extension, got '%s'", inputPathStr)
		}
		// Get secure path
		fullInputPath, err := getSecureFilePath(inputPathStr, "data")
		if err != nil {
			return nil, err
		}

		outputPath, ok := args[1].(Str)
		if !ok {
			return nil, fmt.Errorf("output path must be a string, got %T", args[1])
		}
		outputPathStr := string(outputPath)
		// Validate YAML file extension
		if !isValidYAMLFile(outputPathStr) {
			return nil, fmt.Errorf("file must have .yaml or .yml extension, got '%s'", outputPathStr)
		}
		// Get secure path
		fullOutputPath, err := getSecureFilePath(outputPathStr, "data")
		if err != nil {
			return nil, err
		}

		// Read JSON file
		jsonData, err := os.ReadFile(fullInputPath)
		if err != nil {
			return nil, fmt.Errorf("failed to read JSON file '%s': %v", inputPathStr, err)
		}

		// Parse JSON
		var data interface{}
		if err := json.Unmarshal(jsonData, &data); err != nil {
			return nil, fmt.Errorf("failed to parse JSON from '%s': %v", inputPathStr, err)
		}

		// Convert to YAML
		yamlData, err := yaml.Marshal(data)
		if err != nil {
			return nil, fmt.Errorf("failed to convert to YAML: %v", err)
		}

		// Write YAML file
		if err := os.WriteFile(fullOutputPath, yamlData, 0644); err != nil {
			return nil, fmt.Errorf("failed to write YAML file '%s': %v", outputPathStr, err)
		}

		return Bool(true), nil
	})

	// Convert YAML file to JSON file directly
	rt.Register("convertYAMLFileToJSON", func(args ...Value) (Value, error) {
		if len(args) != 2 {
			return nil, errors.New("convertYAMLFileToJSON requires 2 arguments: input_yaml_path, output_json_path")
		}

		// Unwrap arguments
		for i, arg := range args {
			if tvar, ok := arg.(ScopeEntry); ok {
				args[i] = tvar.Value
			}
		}

		inputPath, ok := args[0].(Str)
		if !ok {
			return nil, fmt.Errorf("input path must be a string, got %T", args[0])
		}
		inputPathStr := string(inputPath)
		// Validate YAML file extension
		if !isValidYAMLFile(inputPathStr) {
			return nil, fmt.Errorf("file must have .yaml or .yml extension, got '%s'", inputPathStr)
		}
		// Get secure path
		fullInputPath, err := getSecureFilePath(inputPathStr, "data")
		if err != nil {
			return nil, err
		}

		outputPath, ok := args[1].(Str)
		if !ok {
			return nil, fmt.Errorf("output path must be a string, got %T", args[1])
		}
		outputPathStr := string(outputPath)
		// Validate JSON file extension
		if !isValidJSONFile(outputPathStr) {
			return nil, fmt.Errorf("file must have .json extension, got '%s'", outputPathStr)
		}
		// Get secure path
		fullOutputPath, err := getSecureFilePath(outputPathStr, "data")
		if err != nil {
			return nil, err
		}

		// Read YAML file
		yamlData, err := os.ReadFile(fullInputPath)
		if err != nil {
			return nil, fmt.Errorf("failed to read YAML file '%s': %v", inputPathStr, err)
		}

		// Parse YAML
		var data interface{}
		if err := yaml.Unmarshal(yamlData, &data); err != nil {
			return nil, fmt.Errorf("failed to parse YAML from '%s': %v", inputPathStr, err)
		}

		// Convert to JSON
		jsonData, err := json.MarshalIndent(data, "", "  ")
		if err != nil {
			return nil, fmt.Errorf("failed to convert to JSON: %v", err)
		}

		// Write JSON file
		if err := os.WriteFile(fullOutputPath, jsonData, 0644); err != nil {
			return nil, fmt.Errorf("failed to write JSON file '%s': %v", outputPathStr, err)
		}

		return Bool(true), nil
	})
}
