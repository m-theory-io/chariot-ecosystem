package chariot

import (
	"encoding/xml"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
)

// Move these functions:
// - SetXMLValue
// - findNodeByPath
// - evaluateXMLPredicate
// - findXMLNode

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

// SetXMLValue sets a value in an XML document using an XPath-like expression
func (rt *Runtime) SetXMLValue(path string, value Value, context ...Value) (Value, error) {
	// Determine which XML document to use
	var xmlDoc TreeNode

	if len(context) > 0 {
		// If context is provided, it should be a reference to an XML document
		switch ctx := context[0].(type) {
		case Str:
			// Context is a document ID/name
			docName := string(ctx)
			if doc, exists := rt.vars[docName]; exists {
				if node, ok := doc.(TreeNode); ok {
					xmlDoc = node
				} else {
					return nil, fmt.Errorf("variable '%s' is not an XML document", docName)
				}
			} else {
				return nil, fmt.Errorf("unknown document: %s", docName)
			}

		case TreeNode:
			// Context is directly a TreeNode
			xmlDoc = ctx

		default:
			return nil, fmt.Errorf("invalid XML document context: %v", context[0])
		}
	} else {
		// Use default document if no context provided
		xmlDoc = rt.document
	}

	// Check if we have an XMLCapable node for optimized handling
	if xmlCapable, ok := xmlDoc.(XMLCapable); ok {
		// Use XML-specific capabilities
		targetNode := xmlCapable.SelectSingleNode(path)
		if targetNode == nil {
			return nil, fmt.Errorf("path not found: %s", path)
		}

		// Set value based on type
		switch v := value.(type) {
		case Str:
			targetNode.SetContent(string(v))
			return value, nil

		case Number:
			targetNode.SetContent(fmt.Sprintf("%g", v))
			return value, nil

		case Bool:
			if bool(v) {
				targetNode.SetContent("true")
			} else {
				targetNode.SetContent("false")
			}
			return value, nil

		case TreeNode:
			// Handle node replacement for XMLCapable
			parent := targetNode.Parent()
			if parent == nil {
				return nil, errors.New("cannot replace root node")
			}

			// Fallback to generic operations
			parent.RemoveChild(targetNode)
			parent.AddChild(v)

			return v, nil

		case nil:
			targetNode.SetContent("")
			return nil, nil

		default:
			targetNode.SetContent(fmt.Sprintf("%v", v))
			return value, nil
		}
	}

	// Fall back to generic implementation for non-XMLCapable nodes
	node, err := findNodeByPath(xmlDoc, path)
	if err != nil {
		return nil, err
	}

	if node == nil {
		return nil, fmt.Errorf("path not found: %s", path)
	}

	// Set the value based on its type
	switch v := value.(type) {
	case Str:
		// For strings, set the content attribute
		node.SetAttribute("content", v)

		// If this is a text node or element with text content, update text children too
		for _, child := range node.GetChildren() {
			if nodeType, hasType := child.GetAttribute("type"); hasType && nodeType == Str("text") {
				child.SetAttribute("value", v)
			}
		}

	case Number:
		// For numbers, convert to string and set content
		strVal := Str(fmt.Sprintf("%g", v))
		node.SetAttribute("content", strVal)

		// Update text node if present
		for _, child := range node.GetChildren() {
			if nodeType, hasType := child.GetAttribute("type"); hasType && nodeType == Str("text") {
				child.SetAttribute("value", strVal)
			}
		}

	case Bool:
		// For booleans, convert to string and set content
		var strVal string
		if bool(v) {
			strVal = "true"
		} else {
			strVal = "false"
		}
		node.SetAttribute("content", Str(strVal))

		// Update text node if present
		for _, child := range node.GetChildren() {
			if nodeType, hasType := child.GetAttribute("type"); hasType && nodeType == Str("text") {
				child.SetAttribute("value", Str(strVal))
			}
		}

	case TreeNode:
		// Replace the entire node with the new one
		parent := node.Parent()
		if parent == nil {
			return nil, errors.New("cannot replace root node")
		}

		// Remove the old node
		parent.RemoveChild(node)

		// Add the new node
		parent.AddChild(v)

		return v, nil

	case nil:
		// For nil, clear content
		node.SetAttribute("content", Str(""))

		// Remove text children
		var nonTextChildren []TreeNode
		for _, child := range node.GetChildren() {
			if nodeType, hasType := child.GetAttribute("type"); !hasType || nodeType != Str("text") {
				nonTextChildren = append(nonTextChildren, child)
			}
		}

		// Clear children and re-add only non-text children
		for _, child := range node.GetChildren() {
			node.RemoveChild(child)
		}

		for _, child := range nonTextChildren {
			node.AddChild(child)
		}

	default:
		// For other types, convert to string
		strVal := fmt.Sprintf("%v", v)
		node.SetAttribute("content", Str(strVal))

		// Update text node if present
		for _, child := range node.GetChildren() {
			if nodeType, hasType := child.GetAttribute("type"); hasType && nodeType == Str("text") {
				child.SetAttribute("value", Str(strVal))
			}
		}
	}

	return value, nil
}

// findNodeByPath locates a node in an XML document using a simplified XPath-like expression
func findNodeByPath(root TreeNode, path string) (TreeNode, error) {
	// Handle absolute paths
	// Start from document root
	path = strings.TrimPrefix(path, "/")

	// Split path into segments
	segments := strings.Split(path, "/")

	current := root
	for _, segment := range segments {
		if segment == "" {
			continue
		}

		// Handle special path segments
		switch segment {
		case ".":
			// Current node, do nothing
			continue

		case "..":
			// Parent node
			if current.Parent() != nil {
				current = current.Parent()
			}
			continue
		}

		// Check for predicates
		var predicate string
		predicateStart := strings.Index(segment, "[")
		if predicateStart > 0 && strings.HasSuffix(segment, "]") {
			predicate = segment[predicateStart+1 : len(segment)-1]
			segment = segment[:predicateStart]
		}

		// Find child by name
		found := false
		for _, child := range current.GetChildren() {
			if childName, _ := child.GetAttribute("name"); childName == Str(segment) {
				// If we have a predicate, evaluate it
				if predicate != "" {
					if evaluateXMLPredicate(child, predicate) {
						current = child
						found = true
						break
					}
				} else {
					current = child
					found = true
					break
				}
			}
		}

		if !found {
			return nil, fmt.Errorf("path segment not found: %s", segment)
		}
	}

	return current, nil
}

// evaluateXMLPredicate evaluates a simple XML predicate
func evaluateXMLPredicate(node TreeNode, predicate string) bool {
	// Handle index predicates
	if i, err := strconv.Atoi(predicate); err == nil {
		// Get the i-th sibling with the same name
		if node.Parent() == nil {
			return false
		}

		name, _ := node.GetAttribute("name")
		count := 0

		for _, sibling := range node.Parent().GetChildren() {
			if siblingName, _ := sibling.GetAttribute("name"); siblingName == name {
				if count == i {
					return node == sibling
				}
				count++
			}
		}

		return false
	}

	// Handle attribute predicates (@attr='value')
	if strings.HasPrefix(predicate, "@") {
		parts := strings.Split(predicate[1:], "=")
		if len(parts) != 2 {
			return false
		}

		attrName := parts[0]
		expectedValue := strings.Trim(parts[1], "'\"")

		if attr, exists := node.GetAttribute(attrName); exists {
			return fmt.Sprintf("%v", attr) == expectedValue
		}

		return false
	}

	// For more complex predicates, you'd need a more sophisticated evaluator
	return false
}

// findXMLNode function
func (rt *Runtime) findXMLNode(nodeName string) (*XMLNode, error) {
	// First check if we have a document with this name
	if val, exists := rt.vars[nodeName]; exists {
		if xmlNode, ok := val.(*XMLNode); ok {
			return xmlNode, nil
		}
	}

	// Also check global vars
	if val, exists := rt.globalVars[nodeName]; exists {
		if xmlNode, ok := val.(*XMLNode); ok {
			return xmlNode, nil
		}
	}

	// Finally check the current scope
	if val, exists := rt.currentScope.Get(nodeName); exists {
		if xmlNode, ok := val.(*XMLNode); ok {
			return xmlNode, nil
		}
	}

	return nil, fmt.Errorf("XML node '%s' not found", nodeName)
}
