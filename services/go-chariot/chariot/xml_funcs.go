package chariot

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
)

// Move these functions:
// - SetXMLValue
// - findNodeByPath
// - evaluateXMLPredicate
// - findXMLNode

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
