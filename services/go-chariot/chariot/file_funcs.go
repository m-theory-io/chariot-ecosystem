package chariot

import (
	"encoding/csv"
	"encoding/xml"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	cfg "github.com/bhouse1273/chariot-ecosystem/services/go-chariot/configs"
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
	case "diagram":
		basePath = cfg.ChariotConfig.DiagramPath
		if basePath == "" {
			return "", fmt.Errorf("DiagramPath is not configured")
		}
	default:
		return "", fmt.Errorf("invalid path type: %s", pathType)
	}

	// Ensure base path exists and is a directory
	if info, err := os.Stat(basePath); err != nil {
		if os.IsNotExist(err) {
			if mkErr := os.MkdirAll(basePath, 0755); mkErr != nil {
				return "", fmt.Errorf("failed to create %s path '%s': %w", pathType, basePath, mkErr)
			}
		} else {
			return "", fmt.Errorf("failed to access %s path '%s': %w", pathType, basePath, err)
		}
	} else if !info.IsDir() {
		return "", fmt.Errorf("%s path '%s' is not a directory", pathType, basePath)
	}

	// Clean the filename to prevent directory traversal attacks
	cleanFilename := filepath.Clean(filename)

	// Prevent directory traversal
	if strings.Contains(cleanFilename, "..") {
		return "", fmt.Errorf("invalid filename: directory traversal not allowed")
	}

	// If caller gave an absolute path, honor it directly after validation.
	// This lets configs point to explicit files under DataPath without being duplicated.
	var fullPath string
	if filepath.IsAbs(cleanFilename) {
		fullPath = cleanFilename
	} else {
		fullPath = filepath.Join(basePath, cleanFilename)
	}

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
	RegisterFormatConversions(rt)
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
	// Return the content of the root element, not wrapped in another map
	return xmlElementToMap(element), nil
}

func xmlElementToMap(element XMLElement) interface{} {
	// For simple text elements (no children, no attributes), return just the text
	if len(element.Nodes) == 0 && len(element.Attrs) == 0 && strings.TrimSpace(element.Content) != "" {
		return strings.TrimSpace(element.Content)
	}

	result := make(map[string]interface{})

	// Add attributes directly to result
	if len(element.Attrs) > 0 {
		for _, attr := range element.Attrs {
			result[attr.Name.Local] = attr.Value
		}
	}

	// Add child elements
	if len(element.Nodes) > 0 {
		childArrays := make(map[string][]interface{})

		for _, child := range element.Nodes {
			childName := child.XMLName.Local
			if childName == "" {
				continue
			}

			childData := xmlElementToMap(child)

			// Check if this child name already exists
			if existing, exists := result[childName]; exists {
				// Convert to array if not already
				if arr, isArray := childArrays[childName]; isArray {
					childArrays[childName] = append(arr, childData)
				} else {
					childArrays[childName] = []interface{}{existing, childData}
					delete(result, childName)
				}
			} else {
				result[childName] = childData
			}
		}

		// Merge arrays back into result
		for name, arr := range childArrays {
			result[name] = arr
		}
	}

	// Add text content if present with attributes or children
	if strings.TrimSpace(element.Content) != "" && (len(element.Attrs) > 0 || len(element.Nodes) > 0) {
		result["#text"] = strings.TrimSpace(element.Content)
	}

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
