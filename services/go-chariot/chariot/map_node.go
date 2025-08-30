// map_node.go
package chariot

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strings"
)

// MapNode implements TreeNode for map/object data structures
type MapNode struct {
	TreeNodeImpl
}

// NewMapNode creates a new MapNode with the given name
func NewMapNode(name string) *MapNode {
	node := &MapNode{}
	node.TreeNodeImpl = *NewTreeNode(name)
	// node.Attributes = make(map[string]Value)
	return node
}

// Clone makes a deep copy of the MapNode
func (n *MapNode) Clone() TreeNode {
	clone := NewMapNode(n.Name())

	// Clone attributes
	for key, value := range n.Attributes {
		clone.SetAttribute(key, value)
	}

	// Clone children
	for _, child := range n.Children {
		clone.AddChild(child.Clone())
	}

	return clone
}

func (n *MapNode) GetTypeLabel() string {
	return "MapNode" // Return a string label for the type
}

// Get retrieves a property value by key
func (n *MapNode) Get(key string) (Value, bool) {
	val, ok := n.Attributes[key]
	return val, ok
}

// Set stores a property value by key
func (n *MapNode) Set(key string, value Value) {
	n.Attributes[key] = value
}

// Remove deletes a property
func (n *MapNode) Remove(key string) {
	delete(n.Attributes, key)
}

// Keys returns all property keys in alphabetical order
func (n *MapNode) Keys() []string {
	keys := make([]string, 0, len(n.Attributes))
	for k := range n.Attributes {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// HasKey checks if a key exists
func (n *MapNode) HasKey(key string) bool {
	_, exists := n.Attributes[key]
	return exists
}

// AddChildMap creates and adds a new MapNode as child
func (n *MapNode) AddChildMap(name string) *MapNode {
	child := NewMapNode(name)
	n.AddChild(child)
	return child
}

// FindByKey finds a node that has a property with the given key and value
func (n *MapNode) FindByKey(key string, value Value) *MapNode {
	var result *MapNode

	err := n.Traverse(func(node TreeNode) error {
		if mapNode, ok := node.(*MapNode); ok {
			if val, exists := mapNode.Get(key); exists && val == value {
				result = mapNode
				// Return an error to stop traversal once found
				return fmt.Errorf("found")
			}
		}
		return nil
	})
	if err != nil && err.Error() != "found" {
		return nil
	}

	return result
}

// MarshalJSON converts the map node to JSON (without _children)
func (n *MapNode) MarshalJSON() ([]byte, error) {
	// Create a map to represent this node
	jsonMap := make(map[string]interface{})

	// Add all attributes directly as JSON properties
	for key, val := range n.Attributes {
		jsonMap[key] = ToNative(val)
	}

	// Add children directly as nested objects (no _children wrapper)
	if len(n.Children) > 0 {
		// Group children by name
		childGroups := make(map[string][]*MapNode)
		for _, child := range n.Children {
			if mapChild, ok := child.(*MapNode); ok {
				name := mapChild.Name()

				// Handle array notation (e.g., "items[0]", "items[1]")
				if idx := strings.Index(name, "["); idx != -1 {
					baseName := name[:idx]
					childGroups[baseName] = append(childGroups[baseName], mapChild)
				} else {
					childGroups[name] = append(childGroups[name], mapChild)
				}
			}
		}

		// Process each group and add directly to jsonMap
		for name, group := range childGroups {
			if len(group) == 1 {
				// Single child - add as nested object
				childJSON, err := group[0].MarshalJSON()
				if err != nil {
					return nil, err
				}

				var childObj interface{}
				if err := json.Unmarshal(childJSON, &childObj); err != nil {
					return nil, err
				}
				jsonMap[name] = childObj // ← Direct assignment, no _children

			} else {
				// Multiple children - create array
				childArray := make([]interface{}, len(group))
				for i, child := range group {
					childJSON, err := child.MarshalJSON()
					if err != nil {
						return nil, err
					}

					var childObj interface{}
					if err := json.Unmarshal(childJSON, &childObj); err != nil {
						return nil, err
					}
					childArray[i] = childObj
				}
				jsonMap[name] = childArray // ← Direct assignment, no _children
			}
		}
	}

	return json.Marshal(jsonMap)
}

// Remove the _children special case from UnmarshalJSON
func (n *MapNode) UnmarshalJSON(data []byte) error {
	var jsonMap map[string]interface{}
	if err := json.Unmarshal(data, &jsonMap); err != nil {
		return err
	}

	// Clear existing state
	n.Attributes = make(map[string]Value)
	n.RemoveChildren()

	// Process each key naturally
	for key, val := range jsonMap {
		switch v := val.(type) {
		case map[string]interface{}:
			// Nested objects become child nodes
			childNode := NewMapNode(key)
			childJSON, _ := json.Marshal(v)
			_ = childNode.UnmarshalJSON(childJSON)
			n.AddChild(childNode)

		case []interface{}:
			// Arrays of objects become multiple children with same name
			for i, item := range v {
				if itemMap, ok := item.(map[string]interface{}); ok {
					childName := fmt.Sprintf("%s[%d]", key, i)
					childNode := NewMapNode(childName)
					itemJSON, _ := json.Marshal(itemMap)
					_ = childNode.UnmarshalJSON(itemJSON)
					n.AddChild(childNode)
				} else {
					// Array of primitives - store as attribute
					n.Attributes[key] = FromNative(v)
					break
				}
			}

		default:
			// Primitive values become attributes
			n.Attributes[key] = FromNative(v)
		}
	}

	return nil
}

// ToMap converts the MapNode to a nested Go map
func (n *MapNode) ToMap() map[string]interface{} {
	result := make(map[string]interface{})

	// Add attributes
	for key, value := range n.Attributes {
		result[key] = ToNative(value)
	}

	// Process children with array notation handling
	childGroups := make(map[string][]*MapNode)
	for _, child := range n.Children {
		if mapChild, ok := child.(*MapNode); ok {
			name := mapChild.Name()

			// Handle array notation (e.g., "items[0]", "items[1]" → "items")
			if idx := strings.Index(name, "["); idx != -1 {
				baseName := name[:idx]
				childGroups[baseName] = append(childGroups[baseName], mapChild)
			} else {
				childGroups[name] = append(childGroups[name], mapChild)
			}
		}
	}

	// Convert children to maps
	for name, group := range childGroups {
		if len(group) == 1 {
			// Single child
			result[name] = group[0].ToMap()
		} else {
			// Multiple children with same name (array)
			childArray := make([]map[string]interface{}, len(group))
			for i, child := range group {
				childArray[i] = child.ToMap()
			}
			result[name] = childArray
		}
	}

	return result
}

// FromMap initializes this node from a Go map
func (n *MapNode) FromMap(data map[string]interface{}) {
	// Reset state
	n.Attributes = make(map[string]Value)
	n.RemoveChildren()

	// Process the map
	for key, val := range data {
		switch v := val.(type) {
		case map[string]interface{}:
			// Nested map becomes a child node
			childNode := NewMapNode(key)
			childNode.FromMap(v)
			n.AddChild(childNode)
		case []interface{}:
			// Array handling depends on content
			if len(v) > 0 {
				if _, ok := v[0].(map[string]interface{}); ok {
					// Array of maps becomes multiple children
					for i, item := range v {
						if itemMap, ok := item.(map[string]interface{}); ok {
							childName := fmt.Sprintf("%s[%d]", key, i) // ← Add index
							childNode := NewMapNode(childName)
							childNode.FromMap(itemMap)
							n.AddChild(childNode)
						}
					}
				} else {
					// Regular array becomes a property
					n.Attributes[key] = FromNative(v)
				}
			}
		default:
			// Regular value becomes a property
			n.Attributes[key] = FromNative(v)
		}
	}
}

// String returns a string representation of this node
func (n *MapNode) String() string {
	json, err := n.MarshalJSON()
	if err != nil {
		return fmt.Sprintf("<MapNode %s: error serializing>", n.Name())
	}
	return string(json)
}

// Flatten converts the hierarchical structure to a flat map with dot notation for paths
func (n *MapNode) Flatten() map[string]Value {
	result := make(map[string]Value)
	n.flattenRecursive(result, "")
	return result
}

// Helper function for Flatten
func (n *MapNode) flattenRecursive(result map[string]Value, prefix string) {
	// Add this node's attributes with prefix
	for key, val := range n.Attributes {
		path := key
		if prefix != "" {
			path = prefix + "." + key
		}
		result[path] = val
	}

	// Process children
	for _, child := range n.Children {
		if mapChild, ok := child.(*MapNode); ok {
			childPrefix := mapChild.Name()
			if prefix != "" {
				childPrefix = prefix + "." + childPrefix
			}
			mapChild.flattenRecursive(result, childPrefix)
		}
	}
}

// LoadFromReader parses key=value pairs from a reader into the MapNode's Properties.
func (m *MapNode) LoadFromReader(reader *strings.Reader) error {
	scanner := bufio.NewScanner(reader)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue // skip empty lines and comments
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue // skip malformed lines
		}
		key := strings.TrimSpace(parts[0])
		val := strings.TrimSpace(parts[1])
		m.Set(key, Str(val))
	}
	if err := scanner.Err(); err != nil && err != io.EOF {
		return err
	}
	return nil
}

// Helper function to convert JSON values to Chariot Values
func FromNative(val interface{}) Value {
	switch v := val.(type) {
	case string:
		return Str(v)
	case float64:
		return Number(v)
	case bool:
		return Bool(v)
	case nil:
		return nil
	default:
		// For complex types, convert to string representation
		return Str(fmt.Sprintf("%v", v))
	}
}

// Helper to convert Chariot values to native Go values
func ToNative(val Value) interface{} {
	switch v := val.(type) {
	case Str:
		return string(v)
	case Number:
		return float64(v)
	case Bool:
		return bool(v)
	case nil:
		return nil
	default:
		return fmt.Sprintf("%v", v)
	}
}
