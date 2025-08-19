package chariot

import (
	"encoding/csv"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	cfg "github.com/bhouse1273/chariot-ecosystem/services/go-chariot/configs"
	"gopkg.in/yaml.v3"
)

// Add this helper function at the top of the file after imports
// GetSecureFilePath returns a secure file path for the given filename and path type
func GetSecureFilePath(filename string, pathType string) (string, error) {
	return getSecureFilePath(filename, pathType)
}

// getSecureFilePath is the internal implementation
func getSecureFilePath(filename string, pathType string) (string, error) {
	var basePath string

	switch pathType {
	case "data":
		basePath = cfg.ChariotConfig.DataPath
		if basePath == "" {
			return "", fmt.Errorf("DataPath is not configured")
		}
	case "tree":
		basePath = cfg.ChariotConfig.TreePath
		if basePath == "" {
			return "", fmt.Errorf("TreePath is not configured")
		}
	default:
		return "", fmt.Errorf("invalid path type: %s", pathType)
	}

	// Validate base path exists
	if _, err := os.Stat(basePath); os.IsNotExist(err) {
		return "", fmt.Errorf("%s '%s' does not exist", pathType, basePath)
	}

	// Clean the filename to prevent directory traversal attacks
	cleanFilename := filepath.Clean(filename)

	// Prevent directory traversal
	if strings.Contains(cleanFilename, "..") {
		return "", fmt.Errorf("invalid filename: directory traversal not allowed")
	}

	// Build full path
	fullPath := filepath.Join(basePath, cleanFilename)

	// Ensure the resolved path is still within the base path
	absBasePath, err := filepath.Abs(basePath)
	if err != nil {
		return "", fmt.Errorf("failed to resolve base path: %v", err)
	}

	absFullPath, err := filepath.Abs(fullPath)
	if err != nil {
		return "", fmt.Errorf("failed to resolve full path: %v", err)
	}

	if !strings.HasPrefix(absFullPath, absBasePath) {
		return "", fmt.Errorf("path traversal detected: file must be within configured directory")
	}

	return fullPath, nil
}

// RegisterFile registers all file I/O functions
func RegisterFile(rt *Runtime) {
	// === GENERIC FILE OPERATIONS ===
	registerGenericFileOps(rt)

	// === JSON OPERATIONS ===
	registerJSONFileOps(rt)

	// === CSV OPERATIONS ===
	registerCSVFileOps(rt)

	// === YAML OPERATIONS ===
	registerYAMLFileOps(rt)

	// === XML OPERATIONS ===
	registerXMLFileOps(rt)

	// === FORMAT CONVERSIONS ===
	registerFormatConversions(rt)
}

// === SHARED INFRASTRUCTURE ===
func registerGenericFileOps(rt *Runtime) {
	rt.Register("readFile", func(args ...Value) (Value, error) {
		if len(args) != 1 {
			return nil, errors.New("readFile requires 1 argument: filepath")
		}

		// Unwrap arguments
		for i, arg := range args {
			if tvar, ok := arg.(ScopeEntry); ok {
				args[i] = tvar.Value
			}
		}

		filename, ok := args[0].(Str)
		if !ok {
			return nil, fmt.Errorf("filepath must be a string, got %T", args[0])
		}

		// Get secure path
		fullPath, err := getSecureFilePath(string(filename), "data")
		if err != nil {
			return nil, err
		}

		data, err := os.ReadFile(fullPath)
		if err != nil {
			return nil, fmt.Errorf("failed to read file '%s': %v", filename, err)
		}

		return Str(string(data)), nil
	})

	rt.Register("writeFile", func(args ...Value) (Value, error) {
		if len(args) != 2 {
			return nil, errors.New("writeFile requires 2 arguments: filepath and content")
		}

		// Unwrap arguments
		for i, arg := range args {
			if tvar, ok := arg.(ScopeEntry); ok {
				args[i] = tvar.Value
			}
		}

		filename, ok := args[0].(Str)
		if !ok {
			return nil, fmt.Errorf("filepath must be a string, got %T", args[0])
		}

		content, ok := args[1].(Str)
		if !ok {
			return nil, fmt.Errorf("content must be a string, got %T", args[1])
		}

		// Get secure path
		fullPath, err := getSecureFilePath(string(filename), "data")
		if err != nil {
			return nil, err
		}

		// Ensure parent directory exists
		if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
			return nil, fmt.Errorf("failed to create directory: %v", err)
		}

		err = os.WriteFile(fullPath, []byte(string(content)), 0644)
		if err != nil {
			return nil, fmt.Errorf("failed to write file '%s': %v", filename, err)
		}

		return Bool(true), nil
	})

	rt.Register("fileExists", func(args ...Value) (Value, error) {
		if len(args) != 1 {
			return nil, errors.New("fileExists requires 1 argument: filepath")
		}

		// Unwrap arguments
		for i, arg := range args {
			if tvar, ok := arg.(ScopeEntry); ok {
				args[i] = tvar.Value
			}
		}

		filename, ok := args[0].(Str)
		if !ok {
			return nil, fmt.Errorf("filepath must be a string, got %T", args[0])
		}

		// Get secure path
		fullPath, err := getSecureFilePath(string(filename), "data")
		if err != nil {
			return Bool(false), nil // File doesn't exist if path is invalid
		}

		_, err = os.Stat(fullPath)
		return Bool(err == nil), nil
	})

	rt.Register("getFileSize", func(args ...Value) (Value, error) {
		if len(args) != 1 {
			return nil, errors.New("getFileSize requires 1 argument: filepath")
		}

		// Unwrap arguments
		for i, arg := range args {
			if tvar, ok := arg.(ScopeEntry); ok {
				args[i] = tvar.Value
			}
		}

		filename, ok := args[0].(Str)
		if !ok {
			return nil, fmt.Errorf("filepath must be a string, got %T", args[0])
		}

		// Get secure path
		fullPath, err := getSecureFilePath(string(filename), "data")
		if err != nil {
			return nil, err
		}

		fileInfo, err := os.Stat(fullPath)
		if err != nil {
			return nil, fmt.Errorf("failed to get file info for '%s': %v", filename, err)
		}

		return Number(fileInfo.Size()), nil
	})

	rt.Register("deleteFile", func(args ...Value) (Value, error) {
		if len(args) != 1 {
			return nil, errors.New("deleteFile requires 1 argument: filepath")
		}

		// Unwrap arguments
		for i, arg := range args {
			if tvar, ok := arg.(ScopeEntry); ok {
				args[i] = tvar.Value
			}
		}

		filename, ok := args[0].(Str)
		if !ok {
			return nil, fmt.Errorf("fileName must be a string, got %T", args[0])
		}

		// Get secure path
		fullPath, err := getSecureFilePath(string(filename), "data")
		if err != nil {
			return nil, err
		}

		err = os.Remove(fullPath)
		if err != nil {
			return nil, fmt.Errorf("failed to delete file '%s': %v", filename, err)
		}

		return Bool(true), nil
	})

	rt.Register("listFiles", func(args ...Value) (Value, error) {
		if len(args) > 1 {
			return nil, errors.New("listFiles requires 0-1 arguments: optional subdirectory")
		}

		// Unwrap arguments
		for i, arg := range args {
			if tvar, ok := arg.(ScopeEntry); ok {
				args[i] = tvar.Value
			}
		}

		var subdirectory string
		if len(args) == 1 {
			if subdir, ok := args[0].(Str); ok {
				subdirectory = string(subdir)
			} else {
				return nil, fmt.Errorf("subdirectory must be a string, got %T", args[0])
			}
		}

		// Get secure path
		fullPath, err := getSecureFilePath(subdirectory, "data")
		if err != nil {
			return nil, err
		}

		files, err := os.ReadDir(fullPath)
		if err != nil {
			return nil, fmt.Errorf("failed to list files in '%s': %v", subdirectory, err)
		}

		array := NewArray()
		for _, file := range files {
			array.Append(Str(file.Name()))
		}

		return array, nil
	})
}

// === JSON SPECIFIC ===
func registerJSONFileOps(rt *Runtime) {
	rt.Register("loadJSON", func(args ...Value) (Value, error) {
		if len(args) != 1 {
			return nil, errors.New("loadJSON requires 1 argument: filepath")
		}

		// Unwrap arguments
		for i, arg := range args {
			if tvar, ok := arg.(ScopeEntry); ok {
				args[i] = tvar.Value
			}
		}

		filename, ok := args[0].(Str)
		if !ok {
			return nil, fmt.Errorf("filepath must be a string, got %T", args[0])
		}

		// Get secure path
		fullPath, err := getSecureFilePath(string(filename), "data")
		if err != nil {
			return nil, err
		}

		// Read file from disk
		data, err := os.ReadFile(fullPath)
		if err != nil {
			return nil, fmt.Errorf("failed to read file '%s': %v", filename, err)
		}

		// Parse JSON and create JSONNode
		var jsonData interface{}
		if err := json.Unmarshal(data, &jsonData); err != nil {
			return nil, fmt.Errorf("failed to parse JSON from '%s': %v", filename, err)
		}

		// Create JSONNode and populate it
		node := NewJSONNode("loaded")
		node.SetJSONValue(jsonData)

		return node, nil
	})

	rt.Register("saveJSON", func(args ...Value) (Value, error) {
		if len(args) < 2 {
			return nil, fmt.Errorf("saveJSON requires at least 2 arguments: object and filePath")
		}

		// Unwrap arguments
		for i, arg := range args {
			if tvar, ok := arg.(ScopeEntry); ok {
				args[i] = tvar.Value
			}
		}

		object := args[0]
		filename := args[1]

		// Optional indent parameter (default to 2 spaces)
		indent := "  "
		if len(args) > 2 {
			if indentArg, ok := args[2].(Number); ok {
				indent = strings.Repeat(" ", int(indentArg))
			} else if indentArg, ok := args[2].(Str); ok {
				indent = string(indentArg)
			}
		}

		// Convert filename to string
		var filenameStr string
		if fp, ok := filename.(Str); ok {
			filenameStr = string(fp)
		} else {
			return Bool(false), fmt.Errorf("filePath must be a string")
		}

		// Get secure path
		fullPath, err := getSecureFilePath(filenameStr, "data")
		if err != nil {
			return nil, err
		}

		// Convert object to JSON based on its type
		var jsonData interface{}

		switch obj := object.(type) {
		case *JSONNode:
			jsonData = obj.GetJSONValue()
		case *MapValue:
			jsonData = convertToInterface(obj.Values)
		case SimpleJSON:
			// Parse the JSON string to get proper structure
			var parsed interface{}
			if err := json.Unmarshal([]byte(obj.String()), &parsed); err != nil {
				return Bool(false), fmt.Errorf("failed to parse SimpleJSON: %v", err)
			}
			jsonData = parsed
		default:
			// Use the existing convertToInterface function
			jsonData = convertToInterface(object)
		}

		// Marshal to JSON with indentation
		var jsonBytes []byte
		if indent != "" {
			jsonBytes, err = json.MarshalIndent(jsonData, "", indent)
		} else {
			jsonBytes, err = json.Marshal(jsonData)
		}

		if err != nil {
			return Bool(false), fmt.Errorf("failed to marshal JSON: %v", err)
		}

		// Ensure parent directory exists
		if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
			return Bool(false), fmt.Errorf("failed to create directory: %v", err)
		}

		// Write to file
		err = os.WriteFile(fullPath, jsonBytes, 0644)
		if err != nil {
			return Bool(false), fmt.Errorf("failed to write file: %v", err)
		}

		return Bool(true), nil
	})

	rt.Register("loadJSONRaw", func(args ...Value) (Value, error) {
		if len(args) != 1 {
			return nil, errors.New("loadJSONRaw requires 1 argument: filepath")
		}

		// Unwrap arguments
		for i, arg := range args {
			if tvar, ok := arg.(ScopeEntry); ok {
				args[i] = tvar.Value
			}
		}

		fileName, ok := args[0].(Str)
		if !ok {
			return nil, fmt.Errorf("filepath must be a string, got %T", args[0])
		}

		// Convert filePath to string
		fileNameStr := string(fileName)

		// Get secure path
		fullPath, err := getSecureFilePath(fileNameStr, "data")
		if err != nil {
			return nil, err
		}

		// Read file and return as JSON string (no parsing)
		data, err := os.ReadFile(fullPath)
		if err != nil {
			return nil, fmt.Errorf("failed to read file '%s': %v", fullPath, err)
		}

		// Validate it's valid JSON
		var temp interface{}
		if err := json.Unmarshal(data, &temp); err != nil {
			return nil, fmt.Errorf("file '%s' contains invalid JSON: %v", fileName, err)
		}

		return Str(string(data)), nil
	})

	rt.Register("saveJSONRaw", func(args ...Value) (Value, error) {
		if len(args) != 2 {
			return nil, errors.New("saveJSONRaw requires 2 arguments: json_string and filepath")
		}

		// Unwrap arguments
		for i, arg := range args {
			if tvar, ok := arg.(ScopeEntry); ok {
				args[i] = tvar.Value
			}
		}

		jsonStr, ok := args[0].(Str)
		if !ok {
			return nil, fmt.Errorf("first argument must be a JSON string, got %T", args[0])
		}

		fileName, ok := args[1].(Str)
		if !ok {
			return nil, fmt.Errorf("fileName must be a string, got %T", args[1])
		}

		// Convert filePath to string
		fileNameStr := string(fileName)

		// Get secure path
		fullPath, err := getSecureFilePath(fileNameStr, "data")
		if err != nil {
			return nil, err
		}
		// Validate JSON string
		var temp interface{}
		if err := json.Unmarshal([]byte(string(jsonStr)), &temp); err != nil {
			return nil, fmt.Errorf("invalid JSON string: %v", err)
		}

		// Write raw JSON to file
		if err := os.WriteFile(string(fullPath), []byte(string(jsonStr)), 0644); err != nil {
			return nil, fmt.Errorf("failed to write file '%s': %v", fileName, err)
		}

		return Bool(true), nil
	})
}

// === CSV SPECIFIC ===
func registerCSVFileOps(rt *Runtime) {
	rt.Register("loadCSV", func(args ...Value) (Value, error) {
		if len(args) < 1 || len(args) > 2 {
			return nil, errors.New("loadCSV requires 1-2 arguments: filepath and optional hasHeaders (boolean)")
		}

		// Unwrap arguments
		for i, arg := range args {
			if tvar, ok := arg.(ScopeEntry); ok {
				args[i] = tvar.Value
			}
		}

		fileName, ok := args[0].(Str)
		if !ok {
			return nil, fmt.Errorf("fileName must be a string, got %T", args[0])
		}

		hasHeaders := true // default
		if len(args) == 2 {
			if headerFlag, ok := args[1].(Bool); ok {
				hasHeaders = bool(headerFlag)
			}
		}

		fileNameStr := string(fileName)
		// Validate CSV file extension
		if filepath.Ext(fileNameStr) != ".csv" {
			return nil, fmt.Errorf("file must have .csv extension, got '%s'", fileNameStr)
		}
		var reader *csv.Reader
		// Support HTTP(S) sources (e.g., Azure Blob SAS URLs) for large ETL inputs
		if strings.HasPrefix(strings.ToLower(fileNameStr), "http://") || strings.HasPrefix(strings.ToLower(fileNameStr), "https://") {
			client := &http.Client{Timeout: 2 * time.Minute}
			resp, err := client.Get(fileNameStr)
			if err != nil {
				return nil, fmt.Errorf("failed to fetch CSV from URL '%s': %v", fileNameStr, err)
			}
			if resp.StatusCode < 200 || resp.StatusCode >= 300 {
				defer resp.Body.Close()
				return nil, fmt.Errorf("failed to fetch CSV from URL '%s': HTTP %d", fileNameStr, resp.StatusCode)
			}
			defer resp.Body.Close()
			reader = csv.NewReader(resp.Body)
		} else {
			// Local file path under configured DataPath
			fullPath, err := getSecureFilePath(fileNameStr, "data")
			if err != nil {
				return nil, err
			}
			file, err := os.Open(fullPath)
			if err != nil {
				return nil, fmt.Errorf("failed to open CSV file '%s': %v", fileNameStr, err)
			}
			defer file.Close()
			reader = csv.NewReader(file)
		}

		// Parse CSV
		records, err := reader.ReadAll()
		if err != nil {
			return nil, fmt.Errorf("failed to parse CSV from '%s': %v", fileNameStr, err)
		}

		if len(records) == 0 {
			// Return empty JSONNode with array
			node := NewJSONNode("csv_data")
			node.SetJSONValue([]interface{}{})
			return node, nil
		}

		var result []interface{}

		if hasHeaders && len(records) > 0 {
			// Use first row as headers
			headers := records[0]
			for i := 1; i < len(records); i++ {
				row := records[i]
				rowObj := make(map[string]interface{})

				for j, value := range row {
					if j < len(headers) {
						// Try to convert numbers
						if num, err := parseNumber(value); err == nil {
							rowObj[headers[j]] = num
						} else if value == "true" {
							rowObj[headers[j]] = true
						} else if value == "false" {
							rowObj[headers[j]] = false
						} else if value == "" {
							rowObj[headers[j]] = nil
						} else {
							rowObj[headers[j]] = value
						}
					}
				}
				result = append(result, rowObj)
			}
		} else {
			// No headers - return array of arrays
			for _, row := range records {
				var rowArray []interface{}
				for _, value := range row {
					// Try to convert numbers
					if num, err := parseNumber(value); err == nil {
						rowArray = append(rowArray, num)
					} else if value == "true" {
						rowArray = append(rowArray, true)
					} else if value == "false" {
						rowArray = append(rowArray, false)
					} else if value == "" {
						rowArray = append(rowArray, nil)
					} else {
						rowArray = append(rowArray, value)
					}
				}
				result = append(result, rowArray)
			}
		}

		// Create JSONNode with the CSV data
		node := NewJSONNode("csv_data")
		node.SetJSONValue(result)

		return node, nil
	})

	rt.Register("saveCSV", func(args ...Value) (Value, error) {
		if len(args) < 2 || len(args) > 3 {
			return nil, errors.New("saveCSV requires 2-3 arguments: data (JSONNode), filepath, and optional includeHeaders (boolean)")
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

		fileName, ok := args[1].(Str)
		if !ok {
			return nil, fmt.Errorf("fileName must be a string, got %T", args[1])
		}

		fileNameStr := string(fileName)
		// Validate CSV file extension
		if filepath.Ext(fileNameStr) != ".csv" {
			return nil, fmt.Errorf("file must have .csv extension, got '%s'", fileNameStr)
		}

		// Get secure path
		fullPath, err := getSecureFilePath(fileNameStr, "data")
		if err != nil {
			return nil, err
		}

		includeHeaders := true // default
		if len(args) == 3 {
			if headerFlag, ok := args[2].(Bool); ok {
				includeHeaders = bool(headerFlag)
			}
		}

		// Get data from JSONNode
		data := jsonNode.GetJSONValue()

		// Convert to slice
		dataSlice, ok := data.([]interface{})
		if !ok {
			return nil, fmt.Errorf("JSONNode must contain an array, got %T", data)
		}

		if len(dataSlice) == 0 {
			// Create empty CSV file
			return writeCSVFile(fullPath, [][]string{})
		}

		var csvRows [][]string

		// Check if first element is an object (has headers) or array (no headers)
		if firstRow, ok := dataSlice[0].(map[string]interface{}); ok {
			// Data has objects - extract headers and convert to CSV
			var headers []string
			for key := range firstRow {
				headers = append(headers, key)
			}

			if includeHeaders {
				csvRows = append(csvRows, headers)
			}

			// Convert each object to CSV row
			for _, item := range dataSlice {
				if rowObj, ok := item.(map[string]interface{}); ok {
					var row []string
					for _, header := range headers {
						value := rowObj[header]
						row = append(row, interfaceToString(value))
					}
					csvRows = append(csvRows, row)
				}
			}
		} else {
			// Data is array of arrays
			for _, item := range dataSlice {
				if rowArray, ok := item.([]interface{}); ok {
					var row []string
					for _, value := range rowArray {
						row = append(row, interfaceToString(value))
					}
					csvRows = append(csvRows, row)
				}
			}
		}

		return writeCSVFile(fullPath, csvRows)
	})

	rt.Register("loadCSVRaw", func(args ...Value) (Value, error) {
		if len(args) != 1 {
			return nil, errors.New("loadCSVRaw requires 1 argument: filepath")
		}

		// Unwrap arguments
		for i, arg := range args {
			if tvar, ok := arg.(ScopeEntry); ok {
				args[i] = tvar.Value
			}
		}

		fileName, ok := args[0].(Str)
		if !ok {
			return nil, fmt.Errorf("fileName must be a string, got %T", args[0])
		}

		fileNameStr := string(fileName)
		// Validate CSV file extension
		if filepath.Ext(fileNameStr) != ".csv" {
			return nil, fmt.Errorf("file must have .csv extension, got '%s'", fileNameStr)
		}
		// Get secure path
		fullPath, err := getSecureFilePath(fileNameStr, "data")
		if err != nil {
			return nil, err
		}

		// Read file and return as string
		data, err := os.ReadFile(fullPath)
		if err != nil {
			return nil, fmt.Errorf("failed to read CSV file '%s': %v", fileNameStr, err)
		}

		return Str(string(data)), nil
	})

	rt.Register("saveCSVRaw", func(args ...Value) (Value, error) {
		if len(args) != 2 {
			return nil, errors.New("saveCSVRaw requires 2 arguments: csv_string and filepath")
		}

		// Unwrap arguments
		for i, arg := range args {
			if tvar, ok := arg.(ScopeEntry); ok {
				args[i] = tvar.Value
			}
		}

		csvStr, ok := args[0].(Str)
		if !ok {
			return nil, fmt.Errorf("first argument must be a CSV string, got %T", args[0])
		}

		fileName, ok := args[1].(Str)
		if !ok {
			return nil, fmt.Errorf("fileName must be a string, got %T", args[1])
		}

		fileNameStr := string(fileName)
		// Validate CSV file extension
		if filepath.Ext(fileNameStr) != ".csv" {
			return nil, fmt.Errorf("file must have .csv extension, got '%s'", fileNameStr)
		}

		// Get secure path
		fullPath, err := getSecureFilePath(fileNameStr, "data")
		if err != nil {
			return nil, err
		}

		// Write raw CSV to file
		if err := os.WriteFile(fullPath, []byte(string(csvStr)), 0644); err != nil {
			return nil, fmt.Errorf("failed to write CSV file '%s': %v", fileNameStr, err)
		}

		return Bool(true), nil
	})
}

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

// === XML SPECIFIC ===
func registerXMLFileOps(rt *Runtime) {
	rt.Register("loadXML", func(args ...Value) (Value, error) {
		if len(args) != 1 {
			return nil, errors.New("loadXML requires 1 argument: filepath")
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
		// Validate XML file extension
		if !isValidXMLFile(fileNameStr) {
			return nil, fmt.Errorf("file must have .xml extension, got '%s'", fileNameStr)
		}
		// Get secure path
		fullPath, err := getSecureFilePath(fileNameStr, "data")
		if err != nil {
			return nil, err
		}

		// Read file from disk
		data, err := os.ReadFile(string(fullPath))
		if err != nil {
			return nil, fmt.Errorf("failed to read XML file '%s': %v", fileNameStr, err)
		}

		// Parse XML into a generic structure
		xmlData, err := parseXMLToMap(data)
		if err != nil {
			return nil, fmt.Errorf("failed to parse XML from '%s': %v", filepath, err)
		}

		// Create JSONNode and populate it (XML data converted to JSON-compatible structure)
		node := NewJSONNode("xml_loaded")
		node.SetJSONValue(xmlData)

		return node, nil
	})

	rt.Register("saveXML", func(args ...Value) (Value, error) {
		if len(args) < 2 || len(args) > 3 {
			return nil, errors.New("saveXML requires 2-3 arguments: node, filepath, and optional rootElementName")
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
		// Validate XML file extension
		if !isValidXMLFile(fileNameStr) {
			return nil, fmt.Errorf("file must have .xml extension, got '%s'", fileNameStr)
		}
		// Get secure path
		fullPath, err := getSecureFilePath(fileNameStr, "data")
		if err != nil {
			return nil, err
		}

		rootElementName := "root" // default
		if len(args) == 3 {
			if rootName, ok := args[2].(Str); ok {
				rootElementName = string(rootName)
			}
		}

		// Get data from JSONNode
		data := jsonNode.GetJSONValue()

		// Convert to XML
		xmlData, err := convertMapToXML(data, rootElementName)
		if err != nil {
			return nil, fmt.Errorf("failed to convert data to XML: %v", err)
		}

		// Write to file with XML header
		xmlContent := fmt.Sprintf("<?xml version=\"1.0\" encoding=\"UTF-8\"?>\n%s", xmlData)
		if err := os.WriteFile(fullPath, []byte(xmlContent), 0644); err != nil {
			return nil, fmt.Errorf("failed to write XML file '%s': %v", fileNameStr, err)
		}

		return Bool(true), nil
	})

	rt.Register("loadXMLRaw", func(args ...Value) (Value, error) {
		if len(args) != 1 {
			return nil, errors.New("loadXMLRaw requires 1 argument: filepath")
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
		// Validate XML file extension
		if !isValidXMLFile(fileNameStr) {
			return nil, fmt.Errorf("file must have .xml extension, got '%s'", fileNameStr)
		}
		// Get secure path
		fullPath, err := getSecureFilePath(fileNameStr, "data")
		if err != nil {
			return nil, err
		}

		// Read file and return as XML string (no parsing)
		data, err := os.ReadFile(string(fullPath))
		if err != nil {
			return nil, fmt.Errorf("failed to read XML file '%s': %v", fileNameStr, err)
		}

		// Basic XML validation
		if err := xml.Unmarshal(data, &struct{}{}); err != nil {
			return nil, fmt.Errorf("file '%s' contains invalid XML: %v", fileNameStr, err)
		}

		return Str(string(data)), nil
	})

	rt.Register("saveXMLRaw", func(args ...Value) (Value, error) {
		if len(args) != 2 {
			return nil, errors.New("saveXMLRaw requires 2 arguments: xml_string and filepath")
		}

		// Unwrap arguments
		for i, arg := range args {
			if tvar, ok := arg.(ScopeEntry); ok {
				args[i] = tvar.Value
			}
		}

		xmlStr, ok := args[0].(Str)
		if !ok {
			return nil, fmt.Errorf("first argument must be an XML string, got %T", args[0])
		}

		filepath, ok := args[1].(Str)
		if !ok {
			return nil, fmt.Errorf("filepath must be a string, got %T", args[1])
		}

		fileNameStr := string(filepath)
		// Validate XML file extension
		if !isValidXMLFile(fileNameStr) {
			return nil, fmt.Errorf("file must have .xml extension, got '%s'", fileNameStr)
		}
		// Get secure path
		fullPath, err := getSecureFilePath(fileNameStr, "data")
		if err != nil {
			return nil, err
		}

		// Validate XML string
		if err := xml.Unmarshal([]byte(string(xmlStr)), &struct{}{}); err != nil {
			return nil, fmt.Errorf("invalid XML string: %v", err)
		}

		// Write raw XML to file
		if err := os.WriteFile(fullPath, []byte(string(xmlStr)), 0644); err != nil {
			return nil, fmt.Errorf("failed to write XML file '%s': %v", fileNameStr, err)
		}

		return Bool(true), nil
	})

	rt.Register("parseXMLString", func(args ...Value) (Value, error) {
		if len(args) != 1 {
			return nil, errors.New("parseXMLString requires 1 argument: xml_string")
		}

		// Unwrap arguments
		for i, arg := range args {
			if tvar, ok := arg.(ScopeEntry); ok {
				args[i] = tvar.Value
			}
		}

		xmlStr, ok := args[0].(Str)
		if !ok {
			return nil, fmt.Errorf("argument must be an XML string, got %T", args[0])
		}

		// Parse XML string into a map structure
		xmlData, err := parseXMLToMap([]byte(string(xmlStr)))
		if err != nil {
			return nil, fmt.Errorf("failed to parse XML string: %v", err)
		}

		// Create JSONNode with the parsed data
		node := NewJSONNode("xml_parsed")
		node.SetJSONValue(xmlData)

		return node, nil
	})
}

// === FORMAT CONVERSIONS ===
func registerFormatConversions(rt *Runtime) {
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

		outputPath, ok := args[1].(Str)
		if !ok {
			return nil, fmt.Errorf("output path must be a string, got %T", args[1])
		}

		// Read JSON file
		jsonData, err := os.ReadFile(string(inputPath))
		if err != nil {
			return nil, fmt.Errorf("failed to read JSON file '%s': %v", inputPath, err)
		}

		// Parse JSON
		var data interface{}
		if err := json.Unmarshal(jsonData, &data); err != nil {
			return nil, fmt.Errorf("failed to parse JSON from '%s': %v", inputPath, err)
		}

		// Convert to YAML
		yamlData, err := yaml.Marshal(data)
		if err != nil {
			return nil, fmt.Errorf("failed to convert to YAML: %v", err)
		}

		// Write YAML file
		if err := os.WriteFile(string(outputPath), yamlData, 0644); err != nil {
			return nil, fmt.Errorf("failed to write YAML file '%s': %v", outputPath, err)
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

// Helper functions for CSV operations
func parseNumber(s string) (interface{}, error) {
	if strings.Contains(s, ".") {
		if f, err := strconv.ParseFloat(s, 64); err == nil {
			return f, nil
		}
	} else {
		if i, err := strconv.ParseInt(s, 10, 64); err == nil {
			return i, nil
		}
	}
	return nil, fmt.Errorf("not a number")
}

func interfaceToString(value interface{}) string {
	if value == nil {
		return ""
	}
	switch v := value.(type) {
	case string:
		return v
	case int, int64:
		return fmt.Sprintf("%d", v)
	case float64:
		return fmt.Sprintf("%g", v)
	case bool:
		return fmt.Sprintf("%t", v)
	default:
		return fmt.Sprintf("%v", v)
	}
}

func writeCSVFile(filepath string, rows [][]string) (Value, error) {
	file, err := os.Create(filepath)
	if err != nil {
		return nil, fmt.Errorf("failed to create CSV file '%s': %v", filepath, err)
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	for _, row := range rows {
		if err := writer.Write(row); err != nil {
			return nil, fmt.Errorf("failed to write CSV row: %v", err)
		}
	}

	return Bool(true), nil
}

// Helper functions for XML operations
type XMLElement struct {
	XMLName xml.Name
	Attrs   []xml.Attr   `xml:",any,attr"`
	Content string       `xml:",chardata"`
	Nodes   []XMLElement `xml:",any"`
}

func parseXMLToMap(data []byte) (interface{}, error) {
	var element XMLElement
	if err := xml.Unmarshal(data, &element); err != nil {
		return nil, err
	}
	return xmlElementToMap(element), nil
}

func xmlElementToMap(element XMLElement) interface{} {
	result := make(map[string]interface{})

	// Add element name as root
	elementName := element.XMLName.Local
	if elementName == "" {
		elementName = "element"
	}

	elementData := make(map[string]interface{})

	// Add attributes
	if len(element.Attrs) > 0 {
		attrs := make(map[string]interface{})
		for _, attr := range element.Attrs {
			attrs[attr.Name.Local] = attr.Value
		}
		elementData["@attributes"] = attrs
	}

	// Add child elements
	if len(element.Nodes) > 0 {
		children := make(map[string]interface{})
		childArrays := make(map[string][]interface{})

		for _, child := range element.Nodes {
			childName := child.XMLName.Local
			if childName == "" {
				continue
			}

			childData := xmlElementToMap(child)

			// Check if this child name already exists
			if existing, exists := children[childName]; exists {
				// Convert to array if not already
				if arr, isArray := childArrays[childName]; isArray {
					childArrays[childName] = append(arr, childData)
				} else {
					childArrays[childName] = []interface{}{existing, childData}
					delete(children, childName)
				}
			} else {
				children[childName] = childData
			}
		}

		// Merge children and arrays
		for name, child := range children {
			elementData[name] = child
		}
		for name, arr := range childArrays {
			elementData[name] = arr
		}
	}

	// Add text content if present and no children
	if len(element.Nodes) == 0 && strings.TrimSpace(element.Content) != "" {
		if len(element.Attrs) == 0 {
			// Simple text content
			result[elementName] = strings.TrimSpace(element.Content)
			return result
		} else {
			// Text content with attributes
			elementData["#text"] = strings.TrimSpace(element.Content)
		}
	}

	result[elementName] = elementData
	return result
}

func convertMapToXML(data interface{}, rootName string) (string, error) {
	switch v := data.(type) {
	case map[string]interface{}:
		return mapToXMLString(v, rootName, 0), nil
	case []interface{}:
		// Convert array to XML with repeated elements
		var elements []string
		for _, item := range v {
			if itemXML, err := convertMapToXML(item, "item"); err == nil {
				elements = append(elements, itemXML)
			}
		}
		return fmt.Sprintf("<%s>\n%s\n</%s>", rootName, strings.Join(elements, "\n"), rootName), nil
	default:
		return fmt.Sprintf("<%s>%v</%s>", rootName, v, rootName), nil
	}
}

func mapToXMLString(data map[string]interface{}, elementName string, indent int) string {
	indentStr := strings.Repeat("  ", indent)
	var parts []string
	var attributes []string
	var textContent string

	// Process attributes and content
	for key, value := range data {
		if key == "@attributes" {
			if attrs, ok := value.(map[string]interface{}); ok {
				for attrName, attrValue := range attrs {
					attributes = append(attributes, fmt.Sprintf(`%s="%v"`, attrName, attrValue))
				}
			}
		} else if key == "#text" {
			textContent = fmt.Sprintf("%v", value)
		} else {
			// Regular child elements
			switch v := value.(type) {
			case map[string]interface{}:
				childXML := mapToXMLString(v, key, indent+1)
				parts = append(parts, childXML)
			case []interface{}:
				for _, item := range v {
					if itemMap, ok := item.(map[string]interface{}); ok {
						childXML := mapToXMLString(itemMap, key, indent+1)
						parts = append(parts, childXML)
					} else {
						parts = append(parts, fmt.Sprintf("%s  <%s>%v</%s>", indentStr, key, item, key))
					}
				}
			default:
				parts = append(parts, fmt.Sprintf("%s  <%s>%v</%s>", indentStr, key, v, key))
			}
		}
	}

	// Build the element
	var attrString string
	if len(attributes) > 0 {
		attrString = " " + strings.Join(attributes, " ")
	}

	if len(parts) == 0 && textContent == "" {
		return fmt.Sprintf("%s<%s%s/>", indentStr, elementName, attrString)
	} else if len(parts) == 0 {
		return fmt.Sprintf("%s<%s%s>%s</%s>", indentStr, elementName, attrString, textContent, elementName)
	} else {
		content := strings.Join(parts, "\n")
		if textContent != "" {
			content = fmt.Sprintf("%s  %s\n%s", indentStr, textContent, content)
		}
		return fmt.Sprintf("%s<%s%s>\n%s\n%s</%s>", indentStr, elementName, attrString, content, indentStr, elementName)
	}
}

// Helper function to validate file extensions
func isValidYAMLFile(filename string) bool {
	ext := strings.ToLower(filepath.Ext(filename))
	return ext == ".yaml" || ext == ".yml"
}

func isValidJSONFile(filename string) bool {
	ext := strings.ToLower(filepath.Ext(filename))
	return ext == ".json"
}

func isValidCSVFile(filename string) bool {
	ext := strings.ToLower(filepath.Ext(filename))
	return ext == ".csv"
}

func isValidXMLFile(filename string) bool {
	ext := strings.ToLower(filepath.Ext(filename))
	return ext == ".xml"
}
