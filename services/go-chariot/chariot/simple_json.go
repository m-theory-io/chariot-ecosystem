package chariot

import (
	"encoding/json"
	"fmt"
	"strings"
)

// SimpleJSON wraps a native Go value from JSON without creating a tree structure
type SimpleJSON struct {
	value interface{} // Native Go value (map, slice, string, etc.)
	meta  *MapValue   // Optional metadata (for DB operations, etc.)
}

// Type returns the value type
func (j *SimpleJSON) Type() ValueType {
	return ValueJSON
}

func NewSimpleJSON(value interface{}) *SimpleJSON {
	if value == nil {
		value = make(map[string]interface{}) // Default to empty object
	}
	return &SimpleJSON{
		value: value,
		meta:  nil, // No metadata by default
	}
}

// String returns a JSON string representation
func (j *SimpleJSON) String() string {
	bytes, err := json.Marshal(j.value)
	if err != nil {
		return fmt.Sprintf("error: %v", err)
	}
	return string(bytes)
}

// Get retrieves a value by path using dot notation
func (j *SimpleJSON) Get(path string) (Value, bool) {
	if path == "" {
		return convertToChariotValue(j.value), true
	}

	parts := strings.Split(path, ".")
	current := j.value

	for _, part := range parts {
		// Handle array indexing with [n] syntax
		if len(part) > 2 && part[0] == '[' && part[len(part)-1] == ']' {
			indexStr := part[1 : len(part)-1]
			var index int
			if _, err := fmt.Sscanf(indexStr, "%d", &index); err != nil {
				return nil, false
			}

			// Try to access array
			if arr, ok := current.([]interface{}); ok {
				if index < 0 || index >= len(arr) {
					return nil, false
				}
				current = arr[index]
			} else {
				return nil, false
			}
		} else {
			// Try to access object property
			if obj, ok := current.(map[string]interface{}); ok {
				val, exists := obj[part]
				if !exists {
					return nil, false
				}
				current = val
			} else {
				return nil, false
			}
		}
	}

	return convertToChariotValue(current), true
}

// Set stores a value at the specified path
func (j *SimpleJSON) Set(path string, val Value) bool {
	goValue := convertValueToNative(val)

	// Handle empty path - replace entire value
	if path == "" {
		j.value = goValue
		return true
	}

	parts := strings.Split(path, ".")
	return j.setPath(j.value, parts, goValue)
}

// Helper for Set to recursively traverse paths
func (j *SimpleJSON) setPath(current interface{}, parts []string, value interface{}) bool {
	if len(parts) == 0 {
		return false
	}

	part := parts[0]

	// Handle array indexing with [n] syntax
	if len(part) > 2 && part[0] == '[' && part[len(part)-1] == ']' {
		// Array index handling
		// ...
	} else if obj, ok := current.(map[string]interface{}); ok {
		if len(parts) == 1 {
			obj[part] = value
			return true
		} else {
			// Need to traverse deeper
			next, exists := obj[part]
			if !exists {
				// Create path node if it doesn't exist
				obj[part] = make(map[string]interface{})
				next = obj[part]
			}
			return j.setPath(next, parts[1:], value)
		}
	}

	return false
}

// GetMeta retrieves a metadata item
func (j *SimpleJSON) GetMeta(key string) (Value, bool) {
	if j.meta == nil {
		return nil, false
	}
	return j.meta.Get(key)
}

// SetMeta stores metadata associated with the JSON
func (j *SimpleJSON) SetMeta(key string, val Value) {
	if j.meta == nil {
		j.meta = NewMap()
	}
	j.meta.Set(key, val)
}

func (j *SimpleJSON) HasMeta(key string) bool {
	if j.meta == nil {
		return false
	}
	_, exists := j.meta.Get(key)
	return exists
}

func (j *SimpleJSON) ClearMeta() {
	j.meta = nil
}

func (j *SimpleJSON) GetAllMeta() *MapValue {
	if j.meta == nil {
		j.meta = NewMap()
	}
	return j.meta
}

// ParseJSON creates a SimpleJSON from a JSON string
func ParseJSON(jsonStr string) (*SimpleJSON, error) {
	var value interface{}
	if err := json.Unmarshal([]byte(jsonStr), &value); err != nil {
		return nil, err
	}

	return &SimpleJSON{value: value, meta: nil}, nil
}

// GetValue returns the raw Go value
func (j *SimpleJSON) GetValue() interface{} {
	return j.value
}

// ToTreeNode converts to a JSONNode for tree operations when needed
func (j *SimpleJSON) ToTreeNode() *JSONNode {
	node := NewJSONNode("root")
	node.SetJSONValue(j.value)

	// Transfer metadata if present
	if j.meta != nil {
		for _, key := range j.meta.Keys() {
			if val, ok := j.meta.Get(key); ok {
				node.SetMeta(key, val)
			}
		}
	}

	return node
}
