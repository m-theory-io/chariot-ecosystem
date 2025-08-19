package chariot

import (
	"encoding/csv"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
)

// RegisterNode registers all node-related functions
func RegisterNode(rt *Runtime) {
	// Node creation
	rt.Register("create", func(args ...Value) (Value, error) {
		// Check if we have the correct number of arguments
		if len(args) > 1 {
			return nil, errors.New("create takes at most 1 argument")
		}

		// Unwrap the first argument if it exists
		for i, arg := range args {
			if tvar, ok := arg.(ScopeEntry); ok {
				args[i] = tvar.Value
			}
		}

		// Create a new empty node
		nodeName := "node"
		if len(args) > 0 {
			if name, ok := args[0].(Str); ok {
				nodeName = string(name)
			} else {
				return nil, fmt.Errorf("node name must be a string, got %T", args[0])
			}
		}

		// Create and return a new TreeNode with the given name
		node := NewTreeNode(nodeName)

		return node, nil
	})

	rt.Register("jsonNode", func(args ...Value) (Value, error) {
		// Create an empty JSON object if no arguments
		if len(args) == 0 {
			return NewJSONNode("json"), nil
		}

		// Unwrap the first argument if it exists
		if tvar, ok := args[0].(ScopeEntry); ok {
			args[0] = tvar.Value
		}

		// Parse JSON from string
		jsonStr, ok := args[0].(Str)
		if !ok {
			return nil, fmt.Errorf("expected JSON string, got %T", args[0])
		}

		var data interface{}
		name := "json"
		if strings.HasPrefix(string(jsonStr), "{") || strings.HasPrefix(string(jsonStr), "[") {
			// Parse the JSON data
			if err := json.Unmarshal([]byte(jsonStr), &data); err != nil {
				return nil, fmt.Errorf("invalid JSON: %v", err)
			}
		} else {
			name = string(jsonStr)
		}

		// Create a TreeNode with the parsed data
		node := NewJSONNode(name)
		if data != nil {
			node.SetJSONValue(data)
		}
		return node, nil
	})
	rt.Register("mapNode", func(args ...Value) (Value, error) {
		// Create an empty map node if no arguments
		if len(args) == 0 {
			return NewMapNode("map"), nil
		}

		// Unwrap the first argument if it exists
		if tvar, ok := args[0].(ScopeEntry); ok {
			args[0] = tvar.Value
		}

		// Parse map from string
		mapStr, ok := args[0].(Str)
		if !ok {
			return nil, fmt.Errorf("expected map string, got %T", args[0])
		}
		// Create a new map node
		node := NewMapNode("map")
		// Use a string reader to load the map data
		reader := strings.NewReader(string(mapStr))
		err := node.LoadFromReader(reader)
		if err != nil {
			return nil, fmt.Errorf("failed to parse map data: %v", err)
		}
		return node, nil
	})

	rt.Register("xmlNode", func(args ...Value) (Value, error) {
		// Create an empty XML document if no arguments
		if len(args) == 0 {
			return NewXMLNode("xml"), nil
		}

		// Unwrap the first argument if it exists
		if tvar, ok := args[0].(ScopeEntry); ok {
			args[0] = tvar.Value
		}

		// Parse XML from string
		xmlStr, ok := args[0].(Str)
		if !ok {
			return nil, fmt.Errorf("expected XML string, got %T", args[0])
		}

		// Create a new XMLNode
		rootNode := NewXMLNode("root") // Temporary root node
		_ = rootNode

		// Use Go's xml decoder to parse the XML string
		decoder := xml.NewDecoder(strings.NewReader(string(xmlStr)))
		token, err := decoder.Token()
		if err != nil {
			return nil, fmt.Errorf("invalid XML: %v", err)
		}

		// Find the first start element
		for {
			if start, ok := token.(xml.StartElement); ok {
				// Create an XMLNode with the correct name
				node := NewXMLNode(start.Name.Local)

				// Unmarshal the XML into our node
				if err := node.UnmarshalXML(decoder, start); err != nil {
					return nil, fmt.Errorf("failed to parse XML: %v", err)
				}

				return node, nil
			}

			// Get next token
			token, err = decoder.Token()
			if err != nil {
				if err == io.EOF {
					break
				}
				return nil, fmt.Errorf("invalid XML: %v", err)
			}
		}

		return nil, fmt.Errorf("no valid XML elements found")
	})

	rt.Register("csvNode", func(args ...Value) (Value, error) {
		// csvNode requires at least 1 argument: filename
		if len(args) == 0 {
			return nil, errors.New("csvNode requires at least 1 argument: filename")
		}

		// Unwrap arguments
		for i, arg := range args {
			if tvar, ok := arg.(ScopeEntry); ok {
				args[i] = tvar.Value
			}
		}

		// Get filename (first argument)
		filename, ok := args[0].(Str)
		if !ok {
			return nil, fmt.Errorf("filename must be a string, got %T", args[0])
		}

		// Optional delimiter (second argument, default: comma)
		delimiter := ","
		if len(args) > 1 {
			if delStr, ok := args[1].(Str); ok {
				delimiter = string(delStr)
			} else {
				return nil, fmt.Errorf("delimiter must be a string, got %T", args[1])
			}
		}

		// Optional hasHeaders (third argument, default: true)
		hasHeaders := true
		if len(args) > 2 {
			if headers, ok := args[2].(Bool); ok {
				hasHeaders = bool(headers)
			} else {
				return nil, fmt.Errorf("hasHeaders must be a boolean, got %T", args[2])
			}
		}

		// Create a new CSVNode with the filename
		node := NewCSVNode(string(filename))

		// Store metadata including the filename
		node.SetMeta("filename", string(filename))
		node.SetMeta("delimiter", delimiter)
		node.SetMeta("hasHeaders", hasHeaders)
		node.SetMeta("encoding", "UTF-8") // Default encoding

		// Get secure file path for the CSV file
		csvPath, err := GetSecureFilePath(string(filename), "data")
		if err != nil {
			return nil, fmt.Errorf("failed to get secure file path for %s: %v", filename, err)
		}

		// Store the resolved file path
		node.SetMeta("resolvedPath", csvPath)

		// Parse headers only (lazy loading - don't load full data yet)
		headers, err := parseCSVHeaders(csvPath, delimiter, hasHeaders)
		if err != nil {
			return nil, fmt.Errorf("failed to parse CSV headers from %s: %v", filename, err)
		}

		// Store headers as ArrayValue in metadata
		headersArray := NewArray()
		for _, header := range headers {
			headersArray.Append(Str(header))
		}
		node.SetMeta("headers", headersArray)

		// Set data loading status
		node.SetMeta("dataLoaded", false)

		return node, nil
	})

	rt.Register("yamlNode", func(args ...Value) (Value, error) {
		// Create an empty YAML document if no arguments
		if len(args) == 0 {
			return NewYAMLNode("yaml"), nil
		}

		// Unwrap the first argument if it exists
		if tvar, ok := args[0].(ScopeEntry); ok {
			args[0] = tvar.Value
		}

		// Parse YAML from string
		yamlStr, ok := args[0].(Str)
		if !ok {
			return nil, fmt.Errorf("expected YAML string, got %T", args[0])
		}

		// Create a new YAMLNode
		node := NewYAMLNode("yaml")

		// Use a string reader to load the YAML data
		reader := strings.NewReader(string(yamlStr))
		err := node.LoadFromReader(reader)
		if err != nil {
			return nil, fmt.Errorf("failed to parse YAML data: %v", err)
		}

		return node, nil
	})

	// Node structure
	rt.Register("addChild", func(args ...Value) (Value, error) {
		if len(args) != 2 {
			return nil, errors.New("addChild requires 2 arguments: parent node and child node")
		}

		// Unwrap the first two arguments if they are ScopeEntries
		for i, arg := range args {
			if tvar, ok := arg.(ScopeEntry); ok {
				args[i] = tvar.Value
			}
		}

		// Get the parent node
		tparent, ok := args[0].(TreeNode)
		if !ok {
			return nil, fmt.Errorf("expected parent node, got %T", args[0])
		}
		parent := tparent

		// Get the child node
		child, ok := args[1].(TreeNode)
		if !ok {
			return nil, fmt.Errorf("expected child node, got %T", args[1])
		}

		// Add the child
		if parent.GetChildren() == nil {
			parent.SetAttribute("Children", make([]TreeNode, 0))
		}
		parent.AddChild(child)

		return parent, nil
	})

	rt.Register("firstChild", func(args ...Value) (Value, error) {
		if len(args) != 1 {
			return nil, errors.New("firstChild requires 1 argument: node")
		}

		// Unwrap the first argument if it is a ScopeEntry
		if tvar, ok := args[0].(ScopeEntry); ok {
			args[0] = tvar.Value
		}

		// Get the node
		node, ok := args[0].(TreeNode)
		if !ok {
			return nil, fmt.Errorf("expected node, got %T", args[0])
		}

		// Check if it has children
		if node.GetChildren() == nil || len(node.GetChildren()) == 0 {
			return DBNull, nil
		}

		return node.GetChildren()[0], nil
	})

	// Add removeChild function
	rt.Register("removeChild", func(args ...Value) (Value, error) {
		if len(args) != 2 {
			return nil, errors.New("removeChild requires 2 arguments: parent node and child node")
		}

		// Unwrap arguments if they're scope entries
		for i, arg := range args {
			if tvar, ok := arg.(ScopeEntry); ok {
				args[i] = tvar.Value
			}
		}

		// Get the parent node
		parent, ok := args[0].(TreeNode)
		if !ok {
			return nil, fmt.Errorf("expected parent node, got %T", args[0])
		}

		// Get the child node
		child, ok := args[1].(TreeNode)
		if !ok {
			return nil, fmt.Errorf("expected child node, got %T", args[1])
		}

		// Remove the child
		parent.RemoveChild(child)
		return parent, nil
	})

	// Add setName function
	rt.Register("setName", func(args ...Value) (Value, error) {
		if len(args) != 2 {
			return nil, errors.New("setName requires 2 arguments: node and new name")
		}

		// Unwrap arguments if they're scope entries
		for i, arg := range args {
			if tvar, ok := arg.(ScopeEntry); ok {
				args[i] = tvar.Value
			}
		}

		// Get the node
		node, ok := args[0].(TreeNode)
		if !ok {
			return nil, fmt.Errorf("expected node, got %T", args[0])
		}

		// Get the new name
		newName, ok := args[1].(Str)
		if !ok {
			return nil, fmt.Errorf("name must be a string, got %T", args[1])
		}

		// Set the name and return the new name
		actualName := node.SetName(string(newName))
		return Str(actualName), nil
	})

	// Add removeAttribute function
	rt.Register("removeAttribute", func(args ...Value) (Value, error) {
		if len(args) != 2 {
			return nil, errors.New("removeAttribute requires 2 arguments: node and attribute name")
		}

		// Unwrap arguments if they're scope entries
		for i, arg := range args {
			if tvar, ok := arg.(ScopeEntry); ok {
				args[i] = tvar.Value
			}
		}

		// Get the node
		node, ok := args[0].(TreeNode)
		if !ok {
			return nil, fmt.Errorf("expected node, got %T", args[0])
		}

		// Get the attribute name
		attrName, ok := args[1].(Str)
		if !ok {
			return nil, fmt.Errorf("attribute name must be a string, got %T", args[1])
		}

		// Remove the attribute
		node.RemoveAttribute(string(attrName))
		return node, nil
	})

	// Add isLeaf function
	rt.Register("isLeaf", func(args ...Value) (Value, error) {
		if len(args) != 1 {
			return nil, errors.New("isLeaf requires 1 argument: node")
		}

		// Unwrap argument if it's a scope entry
		if tvar, ok := args[0].(ScopeEntry); ok {
			args[0] = tvar.Value
		}

		// Get the node
		node, ok := args[0].(TreeNode)
		if !ok {
			return nil, fmt.Errorf("expected node, got %T", args[0])
		}

		return Bool(node.IsLeaf()), nil
	})

	// Add isRoot function
	rt.Register("isRoot", func(args ...Value) (Value, error) {
		if len(args) != 1 {
			return nil, errors.New("isRoot requires 1 argument: node")
		}

		// Unwrap argument if it's a scope entry
		if tvar, ok := args[0].(ScopeEntry); ok {
			args[0] = tvar.Value
		}

		// Get the node
		node, ok := args[0].(TreeNode)
		if !ok {
			return nil, fmt.Errorf("expected node, got %T", args[0])
		}

		return Bool(node.IsRoot()), nil
	})

	// Add getDepth function
	rt.Register("getDepth", func(args ...Value) (Value, error) {
		if len(args) != 1 {
			return nil, errors.New("getDepth requires 1 argument: node")
		}

		// Unwrap argument if it's a scope entry
		if tvar, ok := args[0].(ScopeEntry); ok {
			args[0] = tvar.Value
		}

		// Get the node
		node, ok := args[0].(TreeNode)
		if !ok {
			return nil, fmt.Errorf("expected node, got %T", args[0])
		}

		return Number(node.GetDepth()), nil
	})

	// Add getLevel function
	rt.Register("getLevel", func(args ...Value) (Value, error) {
		if len(args) != 1 {
			return nil, errors.New("getLevel requires 1 argument: node")
		}

		// Unwrap argument if it's a scope entry
		if tvar, ok := args[0].(ScopeEntry); ok {
			args[0] = tvar.Value
		}

		// Get the node
		node, ok := args[0].(TreeNode)
		if !ok {
			return nil, fmt.Errorf("expected node, got %T", args[0])
		}

		return Number(node.GetLevel()), nil
	})

	// Add getPath function
	rt.Register("getPath", func(args ...Value) (Value, error) {
		if len(args) != 1 {
			return nil, errors.New("getPath requires 1 argument: node")
		}

		// Unwrap argument if it's a scope entry
		if tvar, ok := args[0].(ScopeEntry); ok {
			args[0] = tvar.Value
		}

		// Get the node
		node, ok := args[0].(TreeNode)
		if !ok {
			return nil, fmt.Errorf("expected node, got %T", args[0])
		}

		// Get path and convert to array
		path := node.GetPath()
		result := NewArray()
		for _, segment := range path {
			result.Append(Str(segment))
		}

		return result, nil
	})

	// Add findByName function
	rt.Register("findByName", func(args ...Value) (Value, error) {
		if len(args) != 2 {
			return nil, errors.New("findByName requires 2 arguments: node and name")
		}

		// Unwrap arguments if they're scope entries
		for i, arg := range args {
			if tvar, ok := arg.(ScopeEntry); ok {
				args[i] = tvar.Value
			}
		}

		// Get the node
		node, ok := args[0].(TreeNode)
		if !ok {
			return nil, fmt.Errorf("expected node, got %T", args[0])
		}

		// Get the name to search for
		name, ok := args[1].(Str)
		if !ok {
			return nil, fmt.Errorf("name must be a string, got %T", args[1])
		}

		// Find the node
		found, exists := node.FindByName(string(name))
		if !exists {
			return DBNull, nil
		}

		return found, nil
	})

	// Add getRoot function
	rt.Register("getRoot", func(args ...Value) (Value, error) {
		if len(args) != 1 {
			return nil, errors.New("getRoot requires 1 argument: node")
		}

		// Unwrap argument if it's a scope entry
		if tvar, ok := args[0].(ScopeEntry); ok {
			args[0] = tvar.Value
		}

		// Get the node
		node, ok := args[0].(TreeNode)
		if !ok {
			return nil, fmt.Errorf("expected node, got %T", args[0])
		}

		return node.GetRoot(), nil
	})

	// Add getSiblings function
	rt.Register("getSiblings", func(args ...Value) (Value, error) {
		if len(args) != 1 {
			return nil, errors.New("getSiblings requires 1 argument: node")
		}

		// Unwrap argument if it's a scope entry
		if tvar, ok := args[0].(ScopeEntry); ok {
			args[0] = tvar.Value
		}

		// Get the node
		node, ok := args[0].(TreeNode)
		if !ok {
			return nil, fmt.Errorf("expected node, got %T", args[0])
		}

		// Get siblings and convert to array
		siblings := node.GetSiblings()
		result := NewArray()
		for _, sibling := range siblings {
			result.Append(sibling)
		}

		return result, nil
	})

	// Add getParent function
	rt.Register("getParent", func(args ...Value) (Value, error) {
		if len(args) != 1 {
			return nil, errors.New("getParent requires 1 argument: node")
		}

		// Unwrap argument if it's a scope entry
		if tvar, ok := args[0].(ScopeEntry); ok {
			args[0] = tvar.Value
		}

		// Get the node
		node, ok := args[0].(TreeNode)
		if !ok {
			return nil, fmt.Errorf("expected node, got %T", args[0])
		}

		parent := node.Parent()
		if parent == nil {
			return DBNull, nil
		}

		return parent, nil
	})

	// Add traverseNode function
	rt.Register("traverseNode", func(args ...Value) (Value, error) {
		if len(args) != 2 {
			return nil, errors.New("traverseNode requires 2 arguments: node and function")
		}

		// Unwrap arguments if they're scope entries
		for i, arg := range args {
			if tvar, ok := arg.(ScopeEntry); ok {
				args[i] = tvar.Value
			}
		}

		// Get the node
		node, ok := args[0].(TreeNode)
		if !ok {
			return nil, fmt.Errorf("expected node, got %T", args[0])
		}

		// Get the function
		fn, ok := args[1].(*FunctionValue)
		if !ok {
			return nil, fmt.Errorf("second argument must be a function, got %T", args[1])
		}

		// Get the call function
		callFn, exists := rt.funcs["call"]
		if !exists {
			return nil, errors.New("call function not found")
		}

		// Traverse the tree
		err := node.Traverse(func(n TreeNode) error {
			_, err := callFn(fn, n)
			return err
		})

		if err != nil {
			return nil, err
		}

		return node, nil
	})

	// Add queryNode function
	rt.Register("queryNode", func(args ...Value) (Value, error) {
		if len(args) != 2 {
			return nil, errors.New("queryNode requires 2 arguments: node and predicate function")
		}

		// Unwrap arguments if they're scope entries
		for i, arg := range args {
			if tvar, ok := arg.(ScopeEntry); ok {
				args[i] = tvar.Value
			}
		}

		// Get the node
		node, ok := args[0].(TreeNode)
		if !ok {
			return nil, fmt.Errorf("expected node, got %T", args[0])
		}

		// Get the predicate function
		fn, ok := args[1].(*FunctionValue)
		if !ok {
			return nil, fmt.Errorf("second argument must be a function, got %T", args[1])
		}

		// Get the call function
		callFn, exists := rt.funcs["call"]
		if !exists {
			return nil, errors.New("call function not found")
		}

		// Query the tree
		matches := node.QueryTree(func(n TreeNode) bool {
			result, err := callFn(fn, n)
			if err != nil {
				return false
			}
			if boolResult, ok := result.(Bool); ok {
				return bool(boolResult)
			}
			return false
		})

		// Convert matches to array
		result := NewArray()
		for _, match := range matches {
			result.Append(match)
		}

		return result, nil
	})

	rt.Register("lastChild", func(args ...Value) (Value, error) {
		if len(args) != 1 {
			return nil, errors.New("lastChild requires 1 argument: node")
		}

		// Unwrap the first argument if it is a ScopeEntry
		if tvar, ok := args[0].(ScopeEntry); ok {
			args[0] = tvar.Value
		}

		// Get the node
		node, ok := args[0].(TreeNode)
		if !ok {
			return nil, fmt.Errorf("expected node, got %T", args[0])
		}

		// Check if it has children
		children := node.GetChildren()
		if len(children) == 0 {
			return DBNull, nil
		}

		return children[len(children)-1], nil
	})

	rt.Register("getName", func(args ...Value) (Value, error) {
		if len(args) != 1 {
			return nil, errors.New("getName requires 1 argument: node")
		}
		// Unwrap the first argument if it is a ScopeEntry
		if tvar, ok := args[0].(ScopeEntry); ok {
			args[0] = tvar.Value
		}
		// Get the node
		node, ok := args[0].(TreeNode)
		if !ok {
			return nil, fmt.Errorf("expected node, got %T", args[0])
		}
		// Return the name of the node
		return Str(node.Name()), nil
	})

	rt.Register("hasAttribute", func(args ...Value) (Value, error) {
		if len(args) != 2 {
			return nil, fmt.Errorf("hasAttribute requires 2 arguments: node, key")
		}

		// Unwrap the first two arguments if they are ScopeEntries
		for i, arg := range args {
			if tvar, ok := arg.(ScopeEntry); ok {
				args[i] = tvar.Value
			}
		}

		key := string(args[1].(Str))

		// Type switch to handle different node types and their specific attribute storage

		switch node := args[0].(type) {
		case *JSONNode:
			// Check JSONNode attributes first
			if _, exists := node.GetAttribute(key); exists {
				return Bool(true), nil
			}
			// Check JSONNode metadata
			if _, exists := node.GetMeta(key); exists {
				return Bool(true), nil
			}
			return Bool(false), nil

		case *CouchbaseNode:
			// Check CouchbaseNode attributes
			if _, exists := node.GetAttribute(key); exists {
				return Bool(true), nil
			}
			// Check CouchbaseNode metadata
			if _, exists := node.GetMeta(key); exists {
				return Bool(true), nil
			}
			return Bool(false), nil

		case *SQLNode:
			// Check SQLNode attributes
			if _, exists := node.GetAttribute(key); exists {
				return Bool(true), nil
			}
			// Check SQLNode metadata
			if _, exists := node.GetMeta(key); exists {
				return Bool(true), nil
			}
			return Bool(false), nil

		case *CSVNode:
			// Check CSVNode attributes
			if _, exists := node.GetAttribute(key); exists {
				return Bool(true), nil
			}
			// Check CSVNode metadata
			if _, exists := node.GetMeta(key); exists {
				return Bool(true), nil
			}
			return Bool(false), nil

		case *XMLNode:
			// Check XMLNode attributes
			if _, exists := node.GetAttribute(key); exists {
				return Bool(true), nil
			}
			// Check XMLNode metadata (if it has metadata support)
			if _, exists := node.GetMeta(key); exists {
				return Bool(true), nil
			}
			return Bool(false), nil

		case *MapNode:
			// Check MapNode attributes
			if _, exists := node.GetAttribute(key); exists {
				return Bool(true), nil
			}
			// Check if key exists in the map data
			if _, exists := node.Get(key); exists {
				return Bool(true), nil
			}
			// Check MapNode metadata (if it has metadata support)
			if _, exists := node.GetMeta(key); exists {
				return Bool(true), nil
			}
			return Bool(false), nil

		case TreeNode:
			// Fallback for any TreeNode implementation
			if _, exists := node.GetAttribute(key); exists {
				return Bool(true), nil
			}
			// Use helper function
			if hasMetadata(node, key) {
				return Bool(true), nil
			}
			return Bool(false), nil

		default:
			fmt.Println("DEBUG: hasAttribute called with unsupported node type:", fmt.Sprintf("%T", node))
			return Bool(false), nil
		}
	})

	rt.Register("setAttribute", func(args ...Value) (Value, error) {
		if len(args) != 3 {
			return nil, errors.New("setAttribute requires 3 arguments: node, key, value")
		}

		// Unwrap arguments if they're scope entries
		for i, arg := range args {
			if tvar, ok := arg.(ScopeEntry); ok {
				args[i] = tvar.Value
			}
		}

		// Get the node (first argument)
		node := args[0]

		// Get the key (second argument)
		key, ok := args[1].(Str)
		if !ok {
			return nil, fmt.Errorf("attribute key must be a string, got %T", args[1])
		}

		// Get the value (third argument) - keep as original Value type
		value := args[2]

		// Handle different node types
		switch n := node.(type) {
		case *JSONNode:
			if n.Attributes == nil {
				n.Attributes = make(map[string]Value)
			}
			// Store the raw Value without conversion
			n.Attributes[string(key)] = value
			return n, nil

		case *TreeNodeImpl:
			if n.Attributes == nil {
				n.Attributes = make(map[string]Value)
			}
			// Store the raw Value without conversion
			n.Attributes[string(key)] = value
			return n, nil

		case TreeNode:
			// Handle generic TreeNode interface
			// This requires TreeNode interface to have SetAttribute method
			// or we need to type assert to concrete types
			if tn, ok := n.(*TreeNodeImpl); ok {
				if tn.Attributes == nil {
					tn.Attributes = make(map[string]Value)
				}
				tn.Attributes[string(key)] = value
				return tn, nil
			}
			if jn, ok := n.(*JSONNode); ok {
				if jn.Attributes == nil {
					jn.Attributes = make(map[string]Value)
				}
				jn.Attributes[string(key)] = value
				return jn, nil
			}
			if mn, ok := n.(*MapNode); ok {
				if mn.Attributes == nil {
					mn.Attributes = make(map[string]Value)
				}
				mn.Attributes[string(key)] = value
				return mn, nil
			}
			return nil, fmt.Errorf("unsupported TreeNode type: %T", n)

		default:
			return nil, fmt.Errorf("setAttribute requires a node with attributes (JSONNode or TreeNodeImpl), got %T", node)
		}
	})

	rt.Register("setAttributes", func(args ...Value) (Value, error) {
		if len(args) != 2 {
			return nil, errors.New("setAttributes requires 2 arguments: node, value")
		}

		// Unwrap arguments if they're scope entries
		for i, arg := range args {
			if tvar, ok := arg.(ScopeEntry); ok {
				args[i] = tvar.Value
			}
		}

		// Get the node (first argument)
		node := args[0]
		value := args[1]

		// Get the value (second argument) - keep as original Value type
		if tvar, ok := value.(*MapValue); !ok {
			return nil, fmt.Errorf("expected map[string]Value for attributes, got %T", args[1])
		} else {
			value = tvar.Values
		}

		switch n := node.(type) {
		case *JSONNode:
			n.Attributes = value.(map[string]Value)
			return n, nil
		case *TreeNodeImpl:
			n.Attributes = value.(map[string]Value)
			return n, nil
		case *MapNode:
			n.Attributes = value.(map[string]Value)
			return n, nil
		default:
			return nil, fmt.Errorf("setAttributes requires a node with attributes (JSONNode, TreeNodeImpl, or MapNode), got %T", node)
		}
	})

	rt.Register("setText", func(args ...Value) (Value, error) {
		if len(args) != 2 {
			return nil, errors.New("setText requires 2 arguments: node and text")
		}

		// Unwrap the first two arguments if they are ScopeEntries
		for i, arg := range args {
			if tvar, ok := arg.(ScopeEntry); ok {
				args[i] = tvar.Value
			}
		}

		// Get the node
		node, ok := args[0].(TreeNode)
		if !ok {
			return nil, fmt.Errorf("expected node, got %T", args[0])
		}

		// Validate the node type - only XML nodes can have text content
		xmlNode, ok := node.(*XMLNode)
		if !ok {
			return nil, fmt.Errorf("setText is only valid for XML nodes, got %T", node)
		}

		// Get the text from the second argument
		var text string
		switch v := args[1].(type) {
		case Str:
			text = string(v)
		default:
			text = fmt.Sprintf("%v", v)
		}

		// Set the text content using the XMLNode-specific method
		xmlNode.SetContent(text)

		return node, nil
	})

	rt.Register("getText", func(args ...Value) (Value, error) {
		if len(args) != 1 {
			return nil, errors.New("getText requires 1 argument: node")
		}

		// Unwrap the first argument if it is a ScopeEntry
		for i, arg := range args {
			if tvar, ok := arg.(ScopeEntry); ok {
				args[i] = tvar.Value
			}
		}

		// Get the node
		node, ok := args[0].(TreeNode)
		if !ok {
			return nil, fmt.Errorf("expected node, got %T", args[0])
		}

		// Validate the node type - only XML nodes can have text content
		xmlNode, ok := node.(*XMLNode)
		if !ok {
			return nil, fmt.Errorf("getText is only valid for XML nodes, got %T", node)
		}

		// Get the text content using the XMLNode-specific method
		return Str(xmlNode.GetContent()), nil
	})

	rt.Register("getChildAt", func(args ...Value) (Value, error) {
		if len(args) != 2 {
			return nil, errors.New("getChildAt requires 2 arguments: node and index")
		}

		// Unwrap the first two arguments if they are ScopeEntries
		for i, arg := range args {
			if tvar, ok := arg.(ScopeEntry); ok {
				args[i] = tvar.Value
			}
		}

		// Get the node
		node, ok := args[0].(TreeNode)
		if !ok {
			return nil, fmt.Errorf("expected node, got %T", args[0])
		}

		// Get the index
		idx, ok := args[1].(Number)
		if !ok {
			return nil, fmt.Errorf("index must be a number, got %T", args[1])
		}

		// Check bounds
		if node.GetChildren() == nil || int(idx) < 0 || int(idx) >= len(node.GetChildren()) {
			return DBNull, nil
		}

		return node.GetChildren()[int(idx)], nil
	})

	rt.Register("getChildByName", func(args ...Value) (Value, error) {
		if len(args) != 2 {
			return nil, errors.New("getChildByName requires 2 arguments: node and name")
		}

		// Unwrap the first two arguments if they are ScopeEntries
		for i, arg := range args {
			if tvar, ok := arg.(ScopeEntry); ok {
				args[i] = tvar.Value
			}
		}

		// Get the node
		node, ok := args[0].(TreeNode)
		if !ok {
			return nil, fmt.Errorf("expected node, got %T", args[0])
		}

		// Get the child name
		if len(args) < 2 {
			return nil, errors.New("getChildByName requires a child name")
		}
		if _, ok := args[1].(Str); !ok {
			return nil, fmt.Errorf("child name must be a string, got %T", args[1])
		}

		// Get the child name, converting to string if necessary
		var name string
		switch v := args[1].(type) {
		case string:
			name = v
		case Str:
			name = string(v)
		default:
			return nil, fmt.Errorf("second argument must be a string or Str, got %T", args[1])
		}

		child, found := node.GetChildByName(name)
		if !found {
			return nil, fmt.Errorf("child not found: %s", name)
		}

		return child, nil
	})

	rt.Register("setChildByName", func(args ...Value) (Value, error) {
		if len(args) != 3 {
			return nil, errors.New("setChildByName requires 3 arguments: node, name, and child")
		}

		// Unwrap the first two arguments if they are ScopeEntries
		for i, arg := range args {
			if tvar, ok := arg.(ScopeEntry); ok {
				args[i] = tvar.Value
			}
		}

		// Get the node
		node, ok := args[0].(TreeNode)
		if !ok {
			return nil, fmt.Errorf("expected node, got %T", args[0])
		}

		// Get the child name
		if len(args) < 2 {
			return nil, errors.New("setChildByName requires a child name")
		}
		if _, ok := args[1].(Str); !ok {
			return nil, fmt.Errorf("child name must be a string, got %T", args[1])
		}

		// Get the child name, converting to string if necessary
		var name string
		switch v := args[1].(type) {
		case string:
			name = v
		case Str:
			name = string(v)
		default:
			return nil, fmt.Errorf("second argument must be a string or Str, got %T", args[1])
		}

		// Set the new child
		childNew, ok := args[2].(TreeNode)
		if !ok {
			return nil, fmt.Errorf("third argument must be a node, got %T", args[2])
		}

		node.SetChildByName(name, childNew)

		return childNew, nil
	})

	rt.Register("childCount", func(args ...Value) (Value, error) {
		if len(args) != 1 {
			return nil, errors.New("childCount requires 1 argument: node")
		}

		// Unwrap the first argument if it is a ScopeEntry
		for i, arg := range args {
			if tvar, ok := arg.(ScopeEntry); ok {
				args[i] = tvar.Value
			}
		}

		// Get the node
		node, ok := args[0].(TreeNode)
		if !ok {
			return nil, fmt.Errorf("expected node, got %T", args[0])
		}

		if node.GetChildren() == nil {
			return Number(0), nil
		}

		return Number(len(node.GetChildren())), nil
	})

	// Node utilities
	rt.Register("clear", func(args ...Value) (Value, error) {
		if len(args) != 1 {
			return nil, errors.New("clear requires 1 argument: node")
		}

		// Unwrap the first argument if it is a ScopeEntry
		for i, arg := range args {
			if tvar, ok := arg.(ScopeEntry); ok {
				args[i] = tvar.Value
			}
		}

		// Get the node
		node, ok := args[0].(TreeNode)
		if !ok {
			return nil, fmt.Errorf("expected node, got %T", args[0])
		}

		// Remove all children using the TreeNode interface method
		node.RemoveChildren()

		return node, nil
	})

	rt.Register("list", func(args ...Value) (Value, error) {
		if len(args) != 1 {
			return nil, errors.New("list requires 1 argument: node")
		}

		// Unwrap the first argument if it is a ScopeEntry
		for i, arg := range args {
			if tvar, ok := arg.(ScopeEntry); ok {
				args[i] = tvar.Value
			}
		}

		// Get the node
		node, ok := args[0].(TreeNode)
		if !ok {
			return nil, fmt.Errorf("expected node, got %T", args[0])
		}

		// For JSON nodes, list properties
		if jsonNode, ok := node.(*JSONNode); ok {
			var result strings.Builder
			for key := range jsonNode.Attributes {
				if result.Len() > 0 {
					result.WriteString(", ")
				}
				result.WriteString(key)
			}
			return Str(result.String()), nil
		}

		// For nodes with children, list child names
		children := node.GetChildren()
		if len(children) > 0 {
			var result strings.Builder
			for _, child := range children {
				if result.Len() > 0 {
					result.WriteString(", ")
				}
				result.WriteString(child.Name())
			}
			return Str(result.String()), nil
		}

		return Str(""), nil
	})

	// Add nodeToString function as well
	rt.Register("nodeToString", func(args ...Value) (Value, error) {
		if len(args) != 1 {
			return nil, errors.New("nodeToString requires 1 argument: node")
		}

		// Unwrap the first argument if it is a ScopeEntry
		for i, arg := range args {
			if tvar, ok := arg.(ScopeEntry); ok {
				args[i] = tvar.Value
			}
		}

		node, ok := args[0].(TreeNode)
		if !ok {
			return nil, fmt.Errorf("argument must be a node, got %T", args[0])
		}

		return Str(node.String()), nil
	})

}

// Helper function to convert Go values to Chariot values
func convertToChariotValue(value interface{}) Value {
	switch v := value.(type) {
	case string:
		return Str(v)
	case float64:
		return Number(v)
	case bool:
		return Bool(v)
	case nil:
		return DBNull
	case map[string]interface{}:
		// Create a map node to hold the data
		node := NewMapNode("object")

		// Add all key/value pairs to the node
		for key, val := range v {
			// Recursively convert nested values
			chariotVal := convertToChariotValue(val)
			node.Set(key, chariotVal)
		}
		return node

	case []interface{}:
		// Create an array to hold the values
		arr := NewArray()

		// Add all elements to the array
		for _, val := range v {
			// Recursively convert nested values
			chariotVal := convertToChariotValue(val)
			arr.Append(chariotVal)
		}
		return arr

	default:
		// For any other types, convert to string
		return Str(fmt.Sprintf("%v", v))
	}
}

// Helper function to convert Chariot values to Go values
//
//lint:ignore U1000 This function is used in the code
func convertFromChariotValue(value Value) interface{} {
	switch v := value.(type) {
	case Str:
		return string(v)
	case Number:
		return float64(v)
	case Bool:
		return bool(v)
	case *ArrayValue:
		// Convert array to Go slice
		result := make([]interface{}, v.Length())
		for i := 0; i < v.Length(); i++ {
			result[i] = convertFromChariotValue(v.Get(i))
		}
		return result
	case *MapNode:
		// Convert map node to Go map
		result := make(map[string]interface{})
		for _, key := range v.Keys() {
			val, _ := v.Get(key)
			result[key] = convertFromChariotValue(val)
		}
		return result
	case *JSONNode:
		// If JSON node has direct access to its native value, use that
		if val := v.GetJSONValue(); val != nil {
			return val
		}
		// Otherwise use the string representation
		return v.String()
	case *XMLNode:
		// For XML, return its string representation
		xmlStr, _ := v.ToXML()
		return xmlStr
	case *YAMLNode:
		// For YAML, return its native value or string representation
		if val := v.Value; val != nil {
			return val
		}
		return v.String()
	case *CSVNode:
		// For CSV, return its string representation
		csvStr, _ := v.ToCSV()
		return csvStr
	default:
		if value == DBNull {
			return nil
		}
		// For any other type, return string representation
		return fmt.Sprintf("%v", value)
	}
}

// Helper function to get nested properties with dot notation
//
//lint:ignore U1000 This function is used in the code
func getNestedProperty(node TreeNode, path string) (Value, error) {
	if path == "" {
		return nil, errors.New("property path cannot be empty")
	}

	// Split the path by dots
	parts := strings.Split(path, ".")
	current := node

	for i, part := range parts {
		// Handle JSON nodes specially
		if jsonNode, ok := current.(*JSONNode); ok {
			// Get the JSON value and navigate through it
			jsonValue := jsonNode.GetJSONValue()
			if jsonValue == nil {
				return DBNull, nil
			}

			// Navigate through remaining path in JSON data
			remainingPath := strings.Join(parts[i:], ".")
			return getJSONProperty(jsonValue, remainingPath), nil
		}

		// For other node types, check attributes first
		if attr, exists := current.GetAttribute(part); exists {
			if i == len(parts)-1 {
				// Last part - return the attribute value
				return convertToChariotValue(attr), nil
			}

			// Not last part - the attribute should be a node
			if childNode, ok := attr.(TreeNode); ok {
				current = childNode
				continue
			} else {
				return DBNull, nil
			}
		}

		// Check children by name
		children := current.GetChildren()
		var found TreeNode
		for _, child := range children {
			if child.Name() == part {
				found = child
				break
			}
		}

		if found == nil {
			return DBNull, nil
		}

		if i == len(parts)-1 {
			// Last part - return the child node
			return found, nil
		}

		current = found
	}

	return current, nil
}

// Helper function to set nested properties with dot notation
//
//lint:ignore U1000 This function is used in the code
func setNestedProperty(node TreeNode, path string, value Value) error {
	if path == "" {
		return errors.New("property path cannot be empty")
	}

	parts := strings.Split(path, ".")
	current := node

	// Navigate to the parent of the property to set
	for i, part := range parts[:len(parts)-1] {
		// Handle JSON nodes specially
		if jsonNode, ok := current.(*JSONNode); ok {
			// For JSON nodes, we need to ensure the path exists
			remainingPath := strings.Join(parts[i:], ".")
			return setJSONProperty(jsonNode, remainingPath, value)
		}

		// Check if attribute exists
		if attr, exists := current.GetAttribute(part); exists {
			if childNode, ok := attr.(TreeNode); ok {
				current = childNode
				continue
			}
		}

		// Check children by name
		children := current.GetChildren()
		var found TreeNode
		for _, child := range children {
			if child.Name() == part {
				found = child
				break
			}
		}

		if found == nil {
			// Create a new node for missing path segments
			newNode := NewTreeNode(part)
			current.AddChild(newNode)
			current = newNode
		} else {
			current = found
		}
	}

	// Set the final property
	finalPart := parts[len(parts)-1]

	// Handle JSON nodes specially for the final assignment
	if jsonNode, ok := current.(*JSONNode); ok {
		return setJSONProperty(jsonNode, finalPart, value)
	}

	// For other nodes, set as attribute
	current.SetAttribute(finalPart, convertFromChariotValue(value))
	return nil
}

// Helper function to get property from JSON data structure
func getJSONProperty(data interface{}, path string) Value {
	fmt.Printf("DEBUG: getJSONProperty called with path='%s', data=%v\n", path, data)

	parts := strings.Split(path, ".")
	current := data

	for i, part := range parts {
		fmt.Printf("DEBUG: Processing part[%d]='%s', current=%v (type %T)\n", i, part, current, current)

		switch v := current.(type) {
		case map[string]interface{}:
			if val, exists := v[part]; exists {
				current = val
			} else {
				return DBNull
			}
		case []interface{}:
			if idx := parseIndex(part); idx >= 0 && idx < len(v) {
				fmt.Printf("DEBUG: Array access: v[%d] = %v\n", idx, v[idx])
				current = v[idx]
			} else {
				return DBNull
			}
		default:
			return DBNull
		}
	}

	result := convertToChariotValue(current)
	fmt.Printf("DEBUG: getJSONProperty returning: %v\n", result)
	return result
}

// Helper function to set property in JSON data structure
//
//lint:ignore U1000 This function is used in the code
func setJSONProperty(jsonNode *JSONNode, path string, value Value) error {
	fmt.Printf("DEBUG: setJSONProperty called with path='%s', value=%v\n", path, value)

	parts := strings.Split(path, ".")
	data := jsonNode.GetJSONValue()

	fmt.Printf("DEBUG: parts=%v, data=%v (type %T)\n", parts, data, data)

	if data == nil {
		data = make(map[string]interface{})
		jsonNode.SetJSONValue(data)
	}

	// Special case: if there's only one part and data is an array,
	// we're setting an array element directly
	if len(parts) == 1 {
		finalPart := parts[0]
		fmt.Printf("DEBUG: Single part case, finalPart='%s'\n", finalPart)

		switch v := data.(type) {
		case map[string]interface{}:
			fmt.Printf("DEBUG: Setting property on object\n")
			v[finalPart] = convertFromChariotValue(value)
			jsonNode.SetJSONValue(data)
			return nil
		case []interface{}:
			fmt.Printf("DEBUG: Setting array element, array=%v\n", v)
			idx := parseIndex(finalPart)
			fmt.Printf("DEBUG: parsed index=%d\n", idx)

			// Handle direct array element assignment
			if idx := parseIndex(finalPart); idx >= 0 && idx < len(v) {
				// Don't modify the slice and call SetJSONValue
				// Instead, directly update the child node
				return updateArrayElement(jsonNode, idx, value)
			} else {
				return fmt.Errorf("array index out of bounds: %s", finalPart)
			}
		default:
			return fmt.Errorf("cannot set property on: %T", data)
		}
	}

	// Handle multi-part paths (navigate through nested structure)
	current := data
	for _, part := range parts[:len(parts)-1] {
		switch v := current.(type) {
		case map[string]interface{}:
			if _, exists := v[part]; !exists {
				v[part] = make(map[string]interface{})
			}
			current = v[part]
		case []interface{}:
			// Handle array navigation
			if idx := parseIndex(part); idx >= 0 && idx < len(v) {
				current = v[idx]
			} else {
				return fmt.Errorf("array index out of bounds: %s", part)
			}
		default:
			return fmt.Errorf("cannot navigate through: %T", current)
		}
	}

	finalPart := parts[len(parts)-1]

	// Handle final assignment to array or object
	switch v := current.(type) {
	case map[string]interface{}:
		v[finalPart] = convertFromChariotValue(value)
	case []interface{}:
		// Handle array element assignment
		if idx := parseIndex(finalPart); idx >= 0 && idx < len(v) {
			v[idx] = convertFromChariotValue(value)
		} else {
			return fmt.Errorf("array index out of bounds: %s", finalPart)
		}
	default:
		return fmt.Errorf("cannot set property on: %T", current)
	}

	// Update the JSONNode with modified data
	jsonNode.SetJSONValue(data)
	return nil
}

func parseIndex(s string) int {
	if idx := 0; len(s) > 0 {
		for _, r := range s {
			if r < '0' || r > '9' {
				return -1
			}
			idx = idx*10 + int(r-'0')
		}
		return idx
	}
	return -1
}

// Add this helper function to node_funcs.go
func updateArrayElement(jsonNode *JSONNode, index int, value Value) error {
	// For arrays, we can directly update the child node instead of rebuilding everything
	children := jsonNode.GetChildren()

	if index < 0 || index >= len(children) {
		return fmt.Errorf("array index out of bounds: %d", index)
	}

	// Find the child with the matching index name
	indexName := fmt.Sprintf("[%d]", index)
	for _, child := range children {
		if child.Name() == indexName {
			if jsonChild, ok := child.(*JSONNode); ok {
				// Just update the child's value, don't rebuild everything
				jsonChild.Set("value", value)
				return nil
			}
		}
	}

	return fmt.Errorf("array element not found at index %d", index)
}

// Helper function to get metadata from any object implementing MetadataHolder
func getMetadata(obj interface{}, key string) (Value, bool) {
	if holder, ok := obj.(MetadataHolder); ok {
		return holder.GetMeta(key)
	}
	return nil, false
}

// Helper function to set metadata on any object implementing MetadataHolder
func setMetadata(obj interface{}, key string, value Value) bool {
	if holder, ok := obj.(MetadataHolder); ok {
		holder.SetMeta(key, value)
		return true
	}
	return false
}

// Helper function to check if an object has metadata for a key
func hasMetadata(obj interface{}, key string) bool {
	if holder, ok := obj.(MetadataHolder); ok {
		return holder.HasMeta(key)
	}
	return false
}

// parseCSVHeaders reads only the header row from a CSV file
func parseCSVHeaders(filePath, delimiter string, hasHeaders bool) ([]string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open CSV file: %v", err)
	}
	defer file.Close()

	// Create CSV reader with the specified delimiter
	reader := csv.NewReader(file)
	if len(delimiter) > 0 {
		reader.Comma = rune(delimiter[0])
	}

	// Configure reader to handle various CSV formats
	reader.FieldsPerRecord = -1 // Allow variable number of fields
	reader.TrimLeadingSpace = true

	if !hasHeaders {
		// If no headers, generate default column names
		// Read first row to count columns
		firstRow, err := reader.Read()
		if err != nil {
			if err == io.EOF {
				return []string{}, nil // Empty file
			}
			return nil, fmt.Errorf("failed to read first row: %v", err)
		}

		// Generate default headers like "col_0", "col_1", etc.
		headers := make([]string, len(firstRow))
		for i := range firstRow {
			headers[i] = fmt.Sprintf("col_%d", i)
		}
		return headers, nil
	}

	// Read the header row
	headers, err := reader.Read()
	if err != nil {
		if err == io.EOF {
			return []string{}, nil // Empty file
		}
		return nil, fmt.Errorf("failed to read CSV headers: %v", err)
	}

	return headers, nil
}
