package chariot

import (
	"encoding/json"
	"fmt"
	"io"
	"strconv"
	"strings"
)

// JSONNode extends MapNode with JSON-specific functionality
type JSONNode struct {
	MapNode
	// JSON-specific fields
	decoder *json.Decoder
}

// NewJSONNode creates a new JSONNode
func NewJSONNode(name string) *JSONNode {
	node := &JSONNode{}
	node.MapNode = *NewMapNode(name)
	// Initialize _meta
	// node.SetMeta("created_at", time.Now().Format(time.RFC3339))
	return node
}

// NewJSONNodeFromArray creates a JSONNode from a []interface{} array
func NewJSONNodeFromArray(arr []interface{}, name string) *JSONNode {
	node := NewJSONNode(name)
	node.SetJSONValue(arr)
	return node
}

// NewJSONNodeFromInterface creates a JSONNode from any interface{} value
func NewJSONNodeFromInterface(value interface{}, name string) *JSONNode {
	node := NewJSONNode(name)
	node.SetJSONValue(value)
	return node
}

func (n *JSONNode) GetTypeLabel() string {
	return "JSONNode" // Return a string label for the type
}

// JSONNode implementation
func (n *JSONNode) Clone() TreeNode {
	clone := &JSONNode{
		MapNode: *n.MapNode.Clone().(*MapNode), // Clone the MapNode part
		decoder: n.decoder,                     // Keep the decoder reference (not cloned)
	}
	return clone
}

// Helper to clone JSON values
func cloneJSONValue(v interface{}) interface{} {
	switch val := v.(type) {
	case map[string]interface{}:
		newMap := make(map[string]interface{})
		for k, v := range val {
			newMap[k] = cloneJSONValue(v)
		}
		return newMap
	case []interface{}:
		newArr := make([]interface{}, len(val))
		for i, v := range val {
			newArr[i] = cloneJSONValue(v)
		}
		return newArr
	default:
		// Primitive values (strings, numbers, booleans, null) can be copied directly
		return val
	}
}

// DecodeStream processes a JSON stream incrementally
func (n *JSONNode) DecodeStream(r io.Reader, callback func(*JSONNode, string) bool) error {
	n.decoder = json.NewDecoder(r)

	// Start by verifying we have an object
	t, err := n.decoder.Token()
	if err != nil {
		return err
	}

	// Expected opening delimiter
	if delim, ok := t.(json.Delim); !ok || delim != '{' {
		return fmt.Errorf("expected object, got %v", t)
	}

	return n.decodeObject(n, callback)
}

// decodeObject processes a JSON object incrementally
func (n *JSONNode) decodeObject(parent *JSONNode, callback func(*JSONNode, string) bool) error {
	for n.decoder.More() {
		// Read the key
		key, err := n.decoder.Token()
		if err != nil {
			return err
		}

		keyStr, ok := key.(string)
		if !ok {
			return fmt.Errorf("expected string key, got %v", key)
		}

		// Build current path from TreeNode hierarchy (computed on-demand)
		path := n.computeJSONPath(keyStr) // ← Compute from tree structure

		// Read the value token
		t, err := n.decoder.Token()
		if err != nil {
			return err
		}

		// Process based on token type
		switch v := t.(type) {
		case json.Delim:
			switch v {
			case '{':
				// New object - create child node
				child := NewJSONNode(keyStr)
				parent.AddChild(child)

				// Process the child object
				if err := n.decodeObject(child, callback); err != nil {
					return err
				}

				// Call callback with the complete object
				if !callback(child, child.GetJSONPath()) {
					return nil // Stop processing if callback returns false
				}

			case '[':
				// New array
				arrayNodes, err := n.decodeArray(keyStr, callback)
				if err != nil {
					return err
				}

				// Add array items as children
				for _, item := range arrayNodes {
					parent.AddChild(item)
				}

			default:
				return fmt.Errorf("unexpected delimiter: %v", v)
			}

		case string:
			parent.Set(keyStr, Str(v))
			if !callback(nil, path) {
				return nil
			}

		case float64:
			parent.Set(keyStr, Number(v))
			if !callback(nil, path) {
				return nil
			}

		case bool:
			parent.Set(keyStr, Bool(v))
			if !callback(nil, path) {
				return nil
			}

		case nil:
			parent.Set(keyStr, nil)
			if !callback(nil, path) {
				return nil
			}

		default:
			return fmt.Errorf("unexpected token: %v", t)
		}
	}

	// Read the closing delimiter
	_, err := n.decoder.Token()
	return err
}

// Helper method to compute JSON path from TreeNode hierarchy
func (n *JSONNode) computeJSONPath(additionalKey string) string {
	// Get path from TreeNode hierarchy
	path := n.GetPath()

	// Add the additional key if provided
	if additionalKey != "" {
		path = append(path, additionalKey)
	}

	// Join with dots for JSONPath notation
	return strings.Join(path, ".")
}

// Helper method to compute JSON path from TreeNode hierarchy
func (n *JSONNode) GetJSONPath() string {
	path := n.GetPath()
	return strings.Join(path, ".")
}

// decodeArray processes a JSON array incrementally
func (n *JSONNode) decodeArray(key string, callback func(*JSONNode, string) bool) ([]*JSONNode, error) {
	var nodes []*JSONNode
	index := 0

	for n.decoder.More() {
		// Read the token
		t, err := n.decoder.Token()
		if err != nil {
			return nil, err
		}

		// Process based on token type
		switch v := t.(type) {
		case json.Delim:
			if v == '{' {
				// Create node for this array item
				indexKey := key + "[" + strconv.Itoa(index) + "]"
				child := NewJSONNode(indexKey)

				// Process the object
				if err := n.decodeObject(child, callback); err != nil {
					return nil, err
				}

				// Add to result
				nodes = append(nodes, child)

				// Call callback with the complete object
				if !callback(child, child.GetJSONPath()) {
					return nodes, nil // Stop processing if callback returns false
				}
			} else {
				// Handle nested arrays if needed
				return nil, fmt.Errorf("nested arrays not yet supported")
			}

		case string, float64, bool, nil:
			// Simple values in array - create a node for each
			indexKey := key + "[" + strconv.Itoa(index) + "]"
			child := NewJSONNode(indexKey)

			// Set the value based on type
			switch val := v.(type) {
			case string:
				child.Set("value", Str(val))
			case float64:
				child.Set("value", Number(val))
			case bool:
				child.Set("value", Bool(val))
			case nil:
				child.Set("value", nil)
			}

			// Add to result
			nodes = append(nodes, child)

			// Call callback
			if !callback(child, child.GetJSONPath()) {
				return nodes, nil
			}

		default:
			return nil, fmt.Errorf("unexpected token in array: %v", t)
		}

		index++
	}

	// Read closing bracket
	_, err := n.decoder.Token()
	if err != nil {
		return nil, err
	}

	return nodes, nil
}

// JSONPath evaluates a JSONPath expression against the node
func (n *JSONNode) JSONPath(path string) (Value, bool) {
	// Implementation of JSONPath query language
	// This is a simplified version that handles basic dot notation

	parts := strings.Split(path, ".")
	current := interface{}(n)

	for _, part := range parts {
		// Handle array indexing
		if idx := strings.Index(part, "["); idx != -1 && strings.HasSuffix(part, "]") {
			name := part[:idx]
			idxStr := part[idx+1 : len(part)-1]
			index, err := strconv.Atoi(idxStr)
			if err != nil {
				return nil, false
			}

			// Find the named node
			if jsonNode, ok := current.(*JSONNode); ok {
				found := false
				for _, child := range jsonNode.GetChildren() {
					if child.Name() == name {
						// We found an array element, get its children
						if arrayNode, ok := child.(*JSONNode); ok {
							// Try to find the indexed child
							children := arrayNode.GetChildren()
							if index >= 0 && index < len(children) {
								current = children[index]
								found = true
								break
							}
						}
					}
				}
				if !found {
					return nil, false
				}
			} else {
				return nil, false
			}
		} else {
			// Regular property access
			if jsonNode, ok := current.(*JSONNode); ok {
				// First try as a child node
				found := false
				for _, child := range jsonNode.GetChildren() {
					if child.Name() == part {
						current = child
						found = true
						break
					}
				}

				// If not a child, try as a property
				if !found {
					if val, ok := jsonNode.Get(part); ok {
						return val, true
					}
					return nil, false
				}
			} else {
				return nil, false
			}
		}
	}

	// If we reach here and current is a JSONNode, return it
	if jsonNode, ok := current.(*JSONNode); ok {
		return jsonNode, true
	}

	return nil, false
}

func (n *JSONNode) GetJSONValue() interface{} {
	attrs := n.GetAttributes()

	// Check if this is an array (has "_" + nodeName key with ArrayValue)
	arrayKey := "_" + n.Name()
	if arrayValue, exists := attrs[arrayKey]; exists {
		if arr, ok := arrayValue.(*ArrayValue); ok {
			// Convert ArrayValue back to []interface{}
			result := make([]interface{}, arr.Length())
			for i := 0; i < arr.Length(); i++ {
				elem := arr.Get(i)
				result[i] = ConvertToNativeJSON(elem)
			}
			return result
		}
	}

	// Check if this is a single primitive value stored as "value" attribute
	if len(attrs) == 1 {
		if valueAttr, exists := attrs["value"]; exists {
			return ConvertToNativeJSON(valueAttr)
		}
	}

	// This is an object - convert all non-array attributes to a map
	result := make(map[string]interface{})
	for key, value := range attrs {
		// Skip array keys (they start with "_")
		if !strings.HasPrefix(key, "_") {
			result[key] = ConvertToNativeJSON(value)
		}
	}

	return result
}

// Add helper method to check if node contains an array
func (n *JSONNode) IsJSONArray() bool {
	arrayKey := "_" + n.Name()
	if arrayValue, exists := n.GetAttributes()[arrayKey]; exists {
		_, ok := arrayValue.(*ArrayValue)
		return ok
	}
	return false
}

// Add helper method to get the array data
func (n *JSONNode) GetArrayValue() *ArrayValue {
	arrayKey := "_" + n.Name()
	if arrayValue, exists := n.GetAttributes()[arrayKey]; exists {
		if arr, ok := arrayValue.(*ArrayValue); ok {
			return arr
		}
	}
	return nil
}

// SetJSONValue sets this node's value from a native Go value
func (n *JSONNode) SetJSONValue(value interface{}) {
	// Clear any existing attributes since we're setting new data
	n.Attributes = make(map[string]Value)

	switch v := value.(type) {
	case map[string]interface{}:
		// For objects, store each key-value pair as an attribute
		for key, val := range v {
			n.SetAttribute(key, convertFromNativeValue(val))
		}

	case []interface{}:
		// For arrays, store as ArrayValue under "_" + nodeName
		array := NewArray()
		for _, val := range v {
			array.Append(convertFromNativeValue(val))
		}
		arrayKey := "_" + n.Name()
		n.SetAttribute(arrayKey, array)

	default:
		// For primitive values, store as a single "value" attribute
		n.SetAttribute("value", convertFromNativeValue(value))
	}
}

func (n *JSONNode) MarshalJSON() ([]byte, error) {
	// Use GetJSONValue() which builds clean structure from tree
	nativeValue := n.GetJSONValue()
	return json.Marshal(nativeValue)
}

// GetJSONArray returns this node's value as a []interface{}
func (n *JSONNode) GetJSONArray() []interface{} {
	// If this isn't an array, return empty slice
	if !n.IsJSONArray() {
		return []interface{}{}
	}

	// Build array from children
	result := make([]interface{}, 0, len(n.GetChildren()))
	for _, child := range n.GetChildren() {
		if jsonChild, ok := child.(*JSONNode); ok {
			result = append(result, jsonChild.GetJSONValue())
		}
	}

	return result
}

// GetJSONObject returns this node's value as a map[string]interface{}
func (n *JSONNode) GetJSONObject() map[string]interface{} {
	// If this is an array, return empty map
	if n.IsJSONArray() {
		return map[string]interface{}{}
	}

	// If this is a primitive value, return empty map
	if _, ok := n.Get("value"); ok {
		return map[string]interface{}{}
	}

	// Create result map
	result := make(map[string]interface{})

	// Process all children as object properties
	for _, child := range n.GetChildren() {
		if jsonChild, ok := child.(*JSONNode); ok {
			result[child.Name()] = jsonChild.GetJSONValue()
		}
	}

	return result
}

// IsJSONObject returns true if this node represents a JSON object (not an array or primitive)
func (n *JSONNode) IsJSONObject() bool {
	// If it has a "value", it's a primitive
	if _, hasValue := n.Get("value"); hasValue {
		return false
	}

	// If it's an array, it's not an object
	if n.IsJSONArray() {
		return false
	}

	// If it has children, it's an object
	return len(n.GetChildren()) > 0
}

// IsJSONPrimitive - compute from tree structure
func (n *JSONNode) IsJSONPrimitive() bool {
	_, hasValue := n.Get("value")
	return hasValue
}

// Helper function to check if a string is numeric
func isNumeric(s string) bool {
	_, err := strconv.Atoi(s)
	return err == nil
}

// ToJSON returns the JSON representation of this node as a string
func (n *JSONNode) ToJSON() (string, error) {
	// Get the native Go value
	nativeValue := n.GetJSONValue()

	// Marshal to JSON with indentation for readability
	bytes, err := json.Marshal(nativeValue)
	if err != nil {
		return "", fmt.Errorf("JSON marshaling error: %w", err)
	}

	// Return as string
	return string(bytes), nil
}

func (n *JSONNode) String() string {
	jsonStr, err := n.ToJSON()
	if err != nil {
		return "{}"
	}
	return jsonStr
}

// SetJSONPath sets a value at the specified path, creating the path if needed
func (n *JSONNode) SetJSONPath(path string, value Value) error {
	parts := strings.Split(path, ".")
	n.setJSONPathParts(parts, value)
	return nil
}

// setJSONPathParts is a recursive helper for setting values by path
// setJSONPathParts is a recursive helper for setting values by path
func (n *JSONNode) setJSONPathParts(parts []string, value Value) {
	if len(parts) == 0 {
		return
	}

	if n.Attributes == nil {
		n.Attributes = make(map[string]Value)
	}

	if len(parts) == 1 {
		// Base case: set the actual value at this key
		key := parts[0]

		// Check if this is a numeric index (array element access)
		if isNumeric(key) {
			// This is an array index - look for the array under "_" + nodeName
			arrayKey := "_" + n.Name()
			if arrayValue, exists := n.Attributes[arrayKey]; exists {
				if arr, ok := arrayValue.(*ArrayValue); ok {
					index, err := strconv.Atoi(key)
					if err != nil || index < 0 || index >= arr.Length() {
						return // Invalid index
					}

					// Set the element at this index
					err = arr.Set(index, value)
					if err != nil {
						return
					}
					return
				}
			}
			// If we reach here, the array doesn't exist - could create it, but for now return
			return
		}

		// Not a numeric index, set as regular attribute
		n.Attributes[key] = value
		return
	}

	// Recursive case: need intermediate structure
	key := parts[0]
	remaining := parts[1:]

	// Check if we're trying to access an array by numeric index
	if isNumeric(key) {
		// This is an array index - look for the array under "_" + nodeName
		arrayKey := "_" + n.Name()
		if arrayValue, exists := n.Attributes[arrayKey]; exists {
			if arr, ok := arrayValue.(*ArrayValue); ok {
				index, err := strconv.Atoi(key)
				if err != nil || index < 0 || index >= arr.Length() {
					return // Invalid index
				}

				// Get the element at this index
				element := arr.Get(index)

				// If the element is a JSONNode, recurse into it
				if jsonNode, ok := element.(*JSONNode); ok {
					jsonNode.setJSONPathParts(remaining, value)
				} else {
					// For primitive values, we can't set nested paths
					return
				}
				return
			}
		}
		// If we reach here, the array doesn't exist - could create it, but for now return
		return
	}

	// Get or create intermediate map/node
	var childMap map[string]Value
	if existing, exists := n.Attributes[key]; exists {
		if existingMap, ok := existing.(map[string]Value); ok {
			childMap = existingMap
		} else if existingNode, ok := existing.(*JSONNode); ok {
			// If it's already a JSONNode, recurse into it
			existingNode.setJSONPathParts(remaining, value)
			return
		} else {
			// Existing value is not a map or node, replace it
			childMap = make(map[string]Value)
		}
	} else {
		// Create new intermediate map
		childMap = make(map[string]Value)
	}

	n.Attributes[key] = childMap

	// Now recurse into the map for the remaining path
	setMapPathParts(childMap, remaining, value)
}

// Helper function to set nested paths in a plain map
func setMapPathParts(m map[string]Value, parts []string, value Value) {
	if len(parts) == 0 {
		return
	}

	if len(parts) == 1 {
		// Base case: set the value
		m[parts[0]] = value
		return
	}

	// Recursive case: need another intermediate map
	key := parts[0]
	remaining := parts[1:]

	var childMap map[string]Value
	if existing, exists := m[key]; exists {
		if existingMap, ok := existing.(map[string]Value); ok {
			childMap = existingMap
		} else {
			// Replace non-map value
			childMap = make(map[string]Value)
		}
	} else {
		// Create new map
		childMap = make(map[string]Value)
	}

	m[key] = childMap
	setMapPathParts(childMap, remaining, value)
}
