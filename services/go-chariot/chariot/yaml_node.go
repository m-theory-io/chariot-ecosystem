package chariot

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// YAMLNode implements TreeNode for YAML data
type YAMLNode struct {
	TreeNodeImpl
	// YAML-specific fields
	Value       interface{} // The native value of this node
	Tag         string      // YAML tag (e.g., !!str, !!int)
	AnchorName  string      // Anchor name if this is an anchor
	Aliases     []*YAMLNode // Nodes that alias to this node
	IsAlias     bool        // Whether this node is an alias
	AliasTarget *YAMLNode   // The target node if this is an alias
	Comment     string      // Comment associated with this node
	LineComment string      // Line comment associated with this node
	Style       yaml.Style  // Node style (flow, block, etc.)
	Line        int         // Line number in source (for error reporting)
	Column      int         // Column number in source (for error reporting)
	yamlNode    *yaml.Node  // Original yaml.Node reference (if loaded from YAML)
	modified    bool        // Tracks if node was modified since last save
}

// NewYAMLNode creates a new YAMLNode with the given name
func NewYAMLNode(name string) *YAMLNode {
	node := &YAMLNode{}
	node.TreeNodeImpl = *NewTreeNode(name)
	return node
}

func (n *YAMLNode) GetTypeLabel() string {
	return "YAMLNode" // Return a string label for the type
}

// LoadFromFile loads YAML from a file
func (n *YAMLNode) LoadFromFile(path string) error {
	// Read file
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	// Load from bytes
	err = n.LoadFromBytes(data)
	if err != nil {
		return err
	}

	// Set name to file basename if not already set
	if n.Name() == "" {
		n.NameStr = filepath.Base(path)
	}

	return nil
}

// LoadFromReader loads YAML from an io.Reader
func (n *YAMLNode) LoadFromReader(r io.Reader) error {
	data, err := io.ReadAll(r)
	if err != nil {
		return err
	}
	return n.LoadFromBytes(data)
}

// LoadFromBytes loads YAML from a byte slice
func (n *YAMLNode) LoadFromBytes(data []byte) error {
	// Parse YAML into nodes
	var rootNode yaml.Node
	err := yaml.Unmarshal(data, &rootNode)
	if err != nil {
		return err
	}

	// Convert yaml.Node to our YAMLNode structure
	return n.buildFromYAMLNode(&rootNode)
}

// buildFromYAMLNode recursively converts yaml.Node to YAMLNode
func (n *YAMLNode) buildFromYAMLNode(node *yaml.Node) error {
	// Store reference to original node
	n.yamlNode = node

	// Set position info
	n.Line = node.Line
	n.Column = node.Column
	n.Style = node.Style

	// Set comment info
	n.Comment = node.HeadComment
	if n.Comment == "" {
		n.Comment = node.FootComment
	}
	n.LineComment = node.LineComment

	// Set anchor/alias info
	n.AnchorName = node.Anchor
	n.IsAlias = node.Kind == yaml.AliasNode
	n.Tag = node.Tag

	// Convert based on kind
	switch node.Kind {
	case yaml.DocumentNode:
		// Process the document content (typically there's just one child)
		if len(node.Content) > 0 {
			// For document node, we process its content directly
			return n.buildFromYAMLNode(node.Content[0])
		}
		return nil

	case yaml.SequenceNode:
		// Array
		n.Value = make([]interface{}, len(node.Content))
		valueArray := n.Value.([]interface{})

		// Process each item
		for i, item := range node.Content {
			childNode := NewYAMLNode(fmt.Sprintf("%d", i))
			if err := childNode.buildFromYAMLNode(item); err != nil {
				return err
			}

			valueArray[i] = childNode.Value
			n.AddChild(childNode)
		}

	case yaml.MappingNode:
		// Object/Map
		n.Value = make(map[string]interface{})
		valueMap := n.Value.(map[string]interface{})

		// Process key-value pairs
		for i := 0; i < len(node.Content); i += 2 {
			// Key should be a scalar
			if node.Content[i].Kind != yaml.ScalarNode {
				return fmt.Errorf("non-scalar key at line %d, column %d",
					node.Content[i].Line, node.Content[i].Column)
			}

			key := node.Content[i].Value
			childNode := NewYAMLNode(key)

			// Process the value
			if err := childNode.buildFromYAMLNode(node.Content[i+1]); err != nil {
				return err
			}

			valueMap[key] = childNode.Value
			n.AddChild(childNode)
		}

	case yaml.ScalarNode:
		// Convert scalar value based on tag
		n.Value = convertYAMLScalar(node)

	case yaml.AliasNode:
		// Alias nodes will be resolved in a second pass
		n.IsAlias = true
		n.Value = nil

	default:
		return fmt.Errorf("unknown YAML node kind: %d", node.Kind)
	}

	return nil
}

// Helper to convert YAML scalar values to appropriate Go types
func convertYAMLScalar(node *yaml.Node) interface{} {
	// Handle explicit tags
	switch node.Tag {
	case "!!null":
		return nil
	case "!!bool":
		return node.Value == "true"
	case "!!int":
		// Parse as Number for our type system
		return Number(parseFloat(node.Value))
	case "!!float":
		return Number(parseFloat(node.Value))
	case "!!binary":
		// We could decode base64 here
		return node.Value
	case "!!timestamp":
		// Could parse time.Time but we'll keep as string
		return Str(node.Value)
	case "!!str", "":
		return Str(node.Value)
	default:
		// Unknown tag, treat as string
		return Str(node.Value)
	}
}

// Helper to parse a number safely
func parseFloat(s string) float64 {
	var f float64
	_, _ = fmt.Sscanf(s, "%f", &f)
	return f
}

// SaveToFile writes YAML to a file
func (n *YAMLNode) SaveToFile(path string) error {
	// Convert to YAML bytes
	data, err := n.ToYAML()
	if err != nil {
		return err
	}

	// Write to file
	return os.WriteFile(path, data, 0644)
}

// ToYAML returns the YAML representation as bytes
func (n *YAMLNode) ToYAML() ([]byte, error) {
	// Convert our tree to yaml.Node structure
	yamlNode := n.toYAMLNode()

	// Marshal to YAML bytes
	return yaml.Marshal(yamlNode)
}

// toYAMLNode converts YAMLNode to yaml.Node for serialization
func (n *YAMLNode) toYAMLNode() *yaml.Node {
	var node yaml.Node

	// Set common properties
	node.Line = n.Line
	node.Column = n.Column
	node.Style = n.Style
	node.Anchor = n.AnchorName
	node.Tag = n.Tag

	// Set comments
	if n.Comment != "" {
		node.HeadComment = n.Comment
	}
	if n.LineComment != "" {
		node.LineComment = n.LineComment
	}

	// Convert based on value type
	switch v := n.Value.(type) {
	case nil:
		node.Kind = yaml.ScalarNode
		node.Tag = "!!null"
		node.Value = ""

	case bool:
		node.Kind = yaml.ScalarNode
		node.Tag = "!!bool"
		if v {
			node.Value = "true"
		} else {
			node.Value = "false"
		}

	case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64:
		node.Kind = yaml.ScalarNode
		node.Tag = "!!int"
		node.Value = fmt.Sprintf("%d", v)

	case float32, float64, Number:
		node.Kind = yaml.ScalarNode
		node.Tag = "!!float"
		node.Value = fmt.Sprintf("%g", v)

	case string, Str:
		node.Kind = yaml.ScalarNode
		node.Tag = "!!str"
		node.Value = fmt.Sprintf("%s", v)

	case []interface{}:
		node.Kind = yaml.SequenceNode
		node.Content = make([]*yaml.Node, len(v))

		// If we have children, use them for content
		if len(n.Children) > 0 && len(n.Children) == len(v) {
			for i, child := range n.Children {
				if yamlChild, ok := child.(*YAMLNode); ok {
					node.Content[i] = yamlChild.toYAMLNode()
				} else {
					// Create a simple node for non-YAML children
					childNode := &yaml.Node{
						Kind:  yaml.ScalarNode,
						Value: fmt.Sprintf("%v", v[i]),
					}
					node.Content[i] = childNode
				}
			}
		} else {
			// No children or length mismatch, create from values
			for i, item := range v {
				childNode := &yaml.Node{
					Kind:  yaml.ScalarNode,
					Value: fmt.Sprintf("%v", item),
				}
				node.Content[i] = childNode
			}
		}

	case map[string]interface{}:
		node.Kind = yaml.MappingNode
		node.Content = make([]*yaml.Node, 0, len(v)*2)

		// Build a map of children by name
		childMap := make(map[string]*YAMLNode)
		for _, child := range n.Children {
			if yamlChild, ok := child.(*YAMLNode); ok {
				childMap[yamlChild.Name()] = yamlChild
			}
		}

		// Process map entries
		for key, value := range v {
			// Key node
			keyNode := &yaml.Node{
				Kind:  yaml.ScalarNode,
				Value: key,
			}
			node.Content = append(node.Content, keyNode)

			// Value node
			if child, exists := childMap[key]; exists {
				// Use existing child node
				node.Content = append(node.Content, child.toYAMLNode())
			} else {
				// Create a simple value node
				valueNode := &yaml.Node{
					Kind:  yaml.ScalarNode,
					Value: fmt.Sprintf("%v", value),
				}
				node.Content = append(node.Content, valueNode)
			}
		}

	default:
		// Unknown type, convert to string
		node.Kind = yaml.ScalarNode
		node.Value = fmt.Sprintf("%v", v)
	}

	return &node
}

// Get retrieves a value by path using dot notation
func (n *YAMLNode) Get(path string) (interface{}, bool) {
	if path == "" {
		return n.Value, true
	}

	parts := strings.Split(path, ".")
	current := n

	for _, part := range parts {
		// Check if this is an array index
		if strings.HasSuffix(part, "]") && strings.Contains(part, "[") {
			// Extract array name and index
			idx := strings.Index(part, "[")
			name := part[:idx]
			idxStr := part[idx+1 : len(part)-1]
			index := parseFloat(idxStr)

			// Find child with this name
			found := false
			for _, child := range current.Children {
				if child.Name() == name {
					if yamlChild, ok := child.(*YAMLNode); ok {
						current = yamlChild
						found = true
						break
					}
				}
			}
			if !found {
				return nil, false
			}

			// Now handle the array index
			if arrayValue, ok := current.Value.([]interface{}); ok {
				if int(index) >= 0 && int(index) < len(arrayValue) {
					// Find the child node at this index
					if int(index) < len(current.Children) {
						if yamlChild, ok := current.Children[int(index)].(*YAMLNode); ok {
							current = yamlChild
							continue
						}
					}
					// Fallback to just returning the value
					return arrayValue[int(index)], true
				}
			}
			return nil, false
		}

		// Regular property access
		found := false
		for _, child := range current.Children {
			if child.Name() == part {
				if yamlChild, ok := child.(*YAMLNode); ok {
					current = yamlChild
					found = true
					break
				}
			}
		}
		if !found {
			return nil, false
		}
	}

	return current.Value, true
}

// Set updates or creates a value at the specified path
func (n *YAMLNode) Set(path string, value interface{}) error {
	if path == "" {
		n.Value = value
		n.modified = true
		return nil
	}

	parts := strings.Split(path, ".")
	current := n

	// Navigate to the parent of the path we want to set
	for i := 0; i < len(parts)-1; i++ {
		part := parts[i]

		// Check for array index notation
		if strings.HasSuffix(part, "]") && strings.Contains(part, "[") {
			// Extract array name and index
			idx := strings.Index(part, "[")
			name := part[:idx]
			idxStr := part[idx+1 : len(part)-1]
			index := int(parseFloat(idxStr))

			// Find or create the array
			arrayChild := n.getOrCreateChild(current, name)

			// Ensure it's an array
			if arrayChild.Value == nil {
				arrayChild.Value = make([]interface{}, 0)
			}

			// Convert to array if not already
			arrayValue, ok := arrayChild.Value.([]interface{})
			if !ok {
				arrayValue = make([]interface{}, 0)
				arrayChild.Value = arrayValue
			}

			// Ensure array is large enough
			if index >= len(arrayValue) {
				// Extend array
				newArray := make([]interface{}, index+1)
				copy(newArray, arrayValue)
				arrayValue = newArray
				arrayChild.Value = arrayValue

				// Add empty nodes for new elements
				for j := len(arrayChild.Children); j <= index; j++ {
					indexNode := NewYAMLNode(fmt.Sprintf("%d", j))
					arrayChild.AddChild(indexNode)
				}
			}

			// Get the array element
			current = arrayChild.Children[index].(*YAMLNode)
		} else {
			// Regular property access
			current = n.getOrCreateChild(current, part)
		}
	}

	// Last part of the path
	lastPart := parts[len(parts)-1]

	// Handle array index in the last part
	if strings.HasSuffix(lastPart, "]") && strings.Contains(lastPart, "[") {
		// Extract array name and index
		idx := strings.Index(lastPart, "[")
		name := lastPart[:idx]
		idxStr := lastPart[idx+1 : len(lastPart)-1]
		index := int(parseFloat(idxStr))

		// Find or create the array
		arrayChild := n.getOrCreateChild(current, name)

		// Ensure it's an array
		if arrayChild.Value == nil {
			arrayChild.Value = make([]interface{}, 0)
		}

		// Convert to array if not already
		arrayValue, ok := arrayChild.Value.([]interface{})
		if !ok {
			arrayValue = make([]interface{}, 0)
			arrayChild.Value = arrayValue
		}

		// Ensure array is large enough
		if index >= len(arrayValue) {
			// Extend array
			newArray := make([]interface{}, index+1)
			copy(newArray, arrayValue)
			arrayValue = newArray
			arrayChild.Value = arrayValue

			// Add empty nodes for new elements
			for j := len(arrayChild.Children); j <= index; j++ {
				indexNode := NewYAMLNode(fmt.Sprintf("%d", j))
				arrayChild.AddChild(indexNode)
				if j < index {
					arrayValue[j] = nil
				}
			}
		}

		// Set the value
		arrayValue[index] = value
		if index < len(arrayChild.Children) {
			indexNode := arrayChild.Children[index].(*YAMLNode)
			indexNode.Value = value
		} else {
			indexNode := NewYAMLNode(fmt.Sprintf("%d", index))
			indexNode.Value = value
			arrayChild.AddChild(indexNode)
		}
	} else {
		// Regular property update
		child := n.getOrCreateChild(current, lastPart)
		child.Value = value
	}

	n.modified = true
	return nil
}

// getOrCreateChild finds a child by name or creates it if not found
func (n *YAMLNode) getOrCreateChild(parent *YAMLNode, name string) *YAMLNode {
	// Try to find existing child
	for _, child := range parent.Children {
		if child.Name() == name {
			if yamlChild, ok := child.(*YAMLNode); ok {
				return yamlChild
			}
		}
	}

	// Create new child
	child := NewYAMLNode(name)

	// Update parent's map value if needed
	if parent.Value == nil {
		parent.Value = make(map[string]interface{})
	}

	if mapValue, ok := parent.Value.(map[string]interface{}); ok {
		mapValue[name] = nil
	}

	parent.AddChild(child)
	return child
}

// Delete removes a value at the specified path
func (n *YAMLNode) Delete(path string) bool {
	if path == "" {
		return false
	}

	parts := strings.Split(path, ".")
	current := n

	// Navigate to the parent of what we want to delete
	for i := 0; i < len(parts)-1; i++ {
		found := false
		for _, child := range current.Children {
			if child.Name() == parts[i] {
				if yamlChild, ok := child.(*YAMLNode); ok {
					current = yamlChild
					found = true
					break
				}
			}
		}
		if !found {
			return false
		}
	}

	// Find and remove the target child
	lastPart := parts[len(parts)-1]
	for i, child := range current.Children {
		if child.Name() == lastPart {
			// Remove from children slice
			current.Children = append(current.Children[:i], current.Children[i+1:]...)

			// Remove from value map if applicable
			if mapValue, ok := current.Value.(map[string]interface{}); ok {
				delete(mapValue, lastPart)
			}

			n.modified = true
			return true
		}
	}

	return false
}

// MergeFrom merges another YAML node into this one
func (n *YAMLNode) MergeFrom(other *YAMLNode) error {
	return n.mergeValues(n, other)
}

// mergeValues recursively merges values
func (n *YAMLNode) mergeValues(target, source *YAMLNode) error {
	// For maps, merge keys
	if targetMap, ok := target.Value.(map[string]interface{}); ok {
		if sourceMap, ok := source.Value.(map[string]interface{}); ok {
			// Process each key in source
			for key, sourceVal := range sourceMap {
				// Find if target has this child
				var targetChild *YAMLNode
				for _, child := range target.Children {
					if child.Name() == key {
						if yamlChild, ok := child.(*YAMLNode); ok {
							targetChild = yamlChild
							break
						}
					}
				}

				// If target doesn't have this key, add it
				if targetChild == nil {
					targetChild = NewYAMLNode(key)
					target.AddChild(targetChild)
					targetMap[key] = sourceVal
					targetChild.Value = sourceVal
				} else {
					// Find corresponding source child
					var sourceChild *YAMLNode
					for _, child := range source.Children {
						if child.Name() == key {
							if yamlChild, ok := child.(*YAMLNode); ok {
								sourceChild = yamlChild
								break
							}
						}
					}

					// Recursively merge if source child exists
					if sourceChild != nil {
						_ = n.mergeValues(targetChild, sourceChild)
					} else {
						// Just set the value
						targetChild.Value = sourceVal
					}
				}
			}
			return nil
		}
	}

	// For arrays, append elements
	if targetArray, ok := target.Value.([]interface{}); ok {
		if sourceArray, ok := source.Value.([]interface{}); ok {
			newLen := len(targetArray) + len(sourceArray)
			newArray := make([]interface{}, newLen)

			// Copy existing elements
			copy(newArray, targetArray)

			// Append new elements
			for i, val := range sourceArray {
				newArray[len(targetArray)+i] = val
			}

			// Update target
			target.Value = newArray

			// Add children from source
			for _, child := range source.Children {
				if yamlChild, ok := child.(*YAMLNode); ok {
					// Create a copy of the child
					newChild := NewYAMLNode(yamlChild.Name())
					newChild.Value = yamlChild.Value
					newChild.Tag = yamlChild.Tag
					newChild.Style = yamlChild.Style
					newChild.Comment = yamlChild.Comment
					newChild.LineComment = yamlChild.LineComment

					target.AddChild(newChild)
				}
			}

			return nil
		}
	}

	// For scalar values, replace
	target.Value = source.Value
	return nil
}

// FindByValue searches for nodes with a specific value
func (n *YAMLNode) FindByValue(value interface{}) []*YAMLNode {
	var results []*YAMLNode

	// Check this node
	if n.Value == value {
		results = append(results, n)
	}

	// Check children
	for _, child := range n.Children {
		if yamlChild, ok := child.(*YAMLNode); ok {
			childResults := yamlChild.FindByValue(value)
			results = append(results, childResults...)
		}
	}

	return results
}

// ToMapNode converts a YAMLNode to a MapNode
func (n *YAMLNode) ToMapNode() *MapNode {
	mapNode := NewMapNode(n.Name())

	// Convert value
	switch v := n.Value.(type) {
	case map[string]interface{}:
		// For maps, also convert children
		for _, child := range n.Children {
			if yamlChild, ok := child.(*YAMLNode); ok {
				childMap := yamlChild.ToMapNode()
				mapNode.AddChild(childMap)
			}
		}

	case []interface{}:
		// Convert array items
		for i, item := range v {
			// Create child nodes for array items
			childName := fmt.Sprintf("%d", i)
			childNode := NewMapNode(childName)
			childNode.Set("value", item)
			mapNode.AddChild(childNode)
		}

	default:
		// For scalar values, store as property
		mapNode.Set("value", n.Value)
	}

	// Preserve comments if present
	if n.Comment != "" {
		mapNode.SetAttribute("comment", Str(n.Comment))
	}
	if n.LineComment != "" {
		mapNode.SetAttribute("lineComment", Str(n.LineComment))
	}

	return mapNode
}

// FromMapNode initializes this YAMLNode from a MapNode
func (n *YAMLNode) FromMapNode(mapNode *MapNode) {
	// Convert property values
	if mapVal, ok := mapNode.ToMap()["value"]; ok {
		// Single scalar value
		n.Value = mapVal
	} else {
		// Map structure
		yamlMap := make(map[string]interface{})
		n.Value = yamlMap

		// Process children
		for _, child := range mapNode.GetChildren() {
			if mapChild, ok := child.(*MapNode); ok {
				yamlChild := NewYAMLNode(mapChild.Name())
				yamlChild.FromMapNode(mapChild)
				n.AddChild(yamlChild)
				yamlMap[mapChild.Name()] = yamlChild.Value
			}
		}
	}

	// Transfer comments if present
	if comment, ok := mapNode.GetAttribute("comment"); ok {
		if strComment, ok := comment.(string); ok {
			n.Comment = strComment
		} else if strComment, ok := comment.(Str); ok {
			n.Comment = string(strComment)
		}
	}

	if lineComment, ok := mapNode.GetAttribute("lineComment"); ok {
		if strComment, ok := lineComment.(string); ok {
			n.LineComment = strComment
		} else if strComment, ok := lineComment.(Str); ok {
			n.LineComment = string(strComment)
		}
	}
}

// ApplyKubernetesPatch applies a strategic merge patch
// This is a simplified implementation of Kubernetes strategic merge patching
func (n *YAMLNode) ApplyKubernetesPatch(patch *YAMLNode) error {
	// Special handling for Kubernetes delete directives
	if directives, ok := patch.Get("$patch"); ok {
		if directive, ok := directives.(string); ok && directive == "delete" {
			return errors.New("object marked for deletion")
		}
	}

	// Apply the patch recursively
	return n.mergeKubernetesValues(n, patch)
}

// mergeKubernetesValues applies Kubernetes-style merging
func (n *YAMLNode) mergeKubernetesValues(target, patch *YAMLNode) error {
	// For maps, merge keys
	if targetMap, ok := target.Value.(map[string]interface{}); ok {
		if patchMap, ok := patch.Value.(map[string]interface{}); ok {
			// Check for special directives
			if directive, ok := patchMap["$patch"]; ok {
				if directive == "replace" {
					// Replace entire object
					target.Value = patch.Value
					target.Children = patch.Children
					return nil
				}
			}

			// Process each key in patch
			for key, patchVal := range patchMap {
				// Skip directives
				if strings.HasPrefix(key, "$") {
					continue
				}

				// Find if target has this child
				var targetChild *YAMLNode
				for _, child := range target.Children {
					if child.Name() == key {
						if yamlChild, ok := child.(*YAMLNode); ok {
							targetChild = yamlChild
							break
						}
					}
				}

				// Find corresponding patch child
				var patchChild *YAMLNode
				for _, child := range patch.Children {
					if child.Name() == key {
						if yamlChild, ok := child.(*YAMLNode); ok {
							patchChild = yamlChild
							break
						}
					}
				}

				// If target doesn't have this key, add it
				if targetChild == nil {
					targetChild = NewYAMLNode(key)
					target.AddChild(targetChild)
					targetMap[key] = patchVal
					targetChild.Value = patchVal
				} else if patchChild != nil {
					// Recursively merge
					_ = n.mergeKubernetesValues(targetChild, patchChild)
				}
			}
			return nil
		}
	}

	// For arrays, handle specially for Kubernetes
	if targetArray, ok := target.Value.([]interface{}); ok {
		if patchArray, ok := patch.Value.([]interface{}); ok {
			// Check for merge key directive
			var mergeKey string
			if patchChild, ok := patch.GetAttribute("$mergeKey"); ok {
				if strKey, ok := patchChild.(string); ok {
					mergeKey = strKey
				} else if strKey, ok := patchChild.(Str); ok {
					mergeKey = string(strKey)
				}
			} else {
				// Default merge keys for common resources
				if target.Name() == "containers" || target.Name() == "initContainers" {
					mergeKey = "name"
				} else if target.Name() == "env" {
					mergeKey = "name"
				} else if target.Name() == "ports" {
					mergeKey = "containerPort"
				} else if target.Name() == "volumes" {
					mergeKey = "name"
				} else {
					// No merge key, just append
					newArray := make([]interface{}, len(targetArray)+len(patchArray))
					copy(newArray, targetArray)
					for i, val := range patchArray {
						newArray[len(targetArray)+i] = val
					}
					target.Value = newArray

					// Add children from patch
					for _, child := range patch.Children {
						if yamlChild, ok := child.(*YAMLNode); ok {
							target.AddChild(yamlChild)
						}
					}
					return nil
				}
			}

			// Merge by key
			for _, patchItem := range patchArray {
				if patchMap, ok := patchItem.(map[string]interface{}); ok {
					if keyVal, ok := patchMap[mergeKey]; ok {
						// Find matching item in target
						matched := false
						for i, targetItem := range targetArray {
							if targetMap, ok := targetItem.(map[string]interface{}); ok {
								if targetKeyVal, ok := targetMap[mergeKey]; ok && targetKeyVal == keyVal {
									// Found matching item, merge it
									if i < len(target.Children) {
										targetChild := target.Children[i].(*YAMLNode)
										for _, patchChild := range patch.Children {
											if yamlPatchChild, ok := patchChild.(*YAMLNode); ok {
												// Find matching patch child
												childKeyVal, _ := yamlPatchChild.Get(mergeKey)
												if childKeyVal == keyVal {
													_ = n.mergeKubernetesValues(targetChild, yamlPatchChild)
													matched = true
													break
												}
											}
										}
									}
								}
							}
						}

						// If no match found, append
						if !matched {
							targetArray = append(targetArray, patchItem)
							target.Value = targetArray

							// Find and add the corresponding patch child
							for _, patchChild := range patch.Children {
								if yamlPatchChild, ok := patchChild.(*YAMLNode); ok {
									childKeyVal, _ := yamlPatchChild.Get(mergeKey)
									if childKeyVal == keyVal {
										target.AddChild(yamlPatchChild)
										break
									}
								}
							}
						}
					}
				}
			}
			return nil
		}
	}

	// For scalar values, replace
	target.Value = patch.Value
	return nil
}

// ValidateKubernetes performs basic Kubernetes manifest validation
func (n *YAMLNode) ValidateKubernetes() []string {
	var errors []string

	// Get apiVersion and kind
	apiVersion, hasAPI := n.Get("apiVersion")
	_ = apiVersion
	kind, hasKind := n.Get("kind")

	if !hasAPI {
		errors = append(errors, "missing required field 'apiVersion'")
	}
	if !hasKind {
		errors = append(errors, "missing required field 'kind'")
	}

	// Check metadata
	metadata, hasMetadata := n.Get("metadata")
	if !hasMetadata {
		errors = append(errors, "missing required field 'metadata'")
	} else if metadataMap, ok := metadata.(map[string]interface{}); ok {
		if _, hasName := metadataMap["name"]; !hasName {
			errors = append(errors, "metadata missing required field 'name'")
		}
	}

	// Resource-specific validation
	if hasKind && hasAPI {
		kindStr := fmt.Sprintf("%v", kind)
		switch kindStr {
		case "Deployment", "StatefulSet", "DaemonSet":
			// Check for spec.template.spec.containers
			if spec, ok := n.Get("spec"); ok {
				if specMap, ok := spec.(map[string]interface{}); ok {
					if template, ok := specMap["template"]; ok {
						if templateMap, ok := template.(map[string]interface{}); ok {
							if podSpec, ok := templateMap["spec"]; ok {
								if podSpecMap, ok := podSpec.(map[string]interface{}); ok {
									if _, ok := podSpecMap["containers"]; !ok {
										errors = append(errors, fmt.Sprintf("%s requires spec.template.spec.containers", kindStr))
									}
								}
							} else {
								errors = append(errors, fmt.Sprintf("%s requires spec.template.spec", kindStr))
							}
						}
					} else {
						errors = append(errors, fmt.Sprintf("%s requires spec.template", kindStr))
					}
				}
			} else {
				errors = append(errors, fmt.Sprintf("%s requires spec", kindStr))
			}

		case "Service":
			// Check for spec.ports
			if spec, ok := n.Get("spec"); ok {
				if specMap, ok := spec.(map[string]interface{}); ok {
					if _, ok := specMap["ports"]; !ok {
						errors = append(errors, "Service requires spec.ports")
					}
				}
			} else {
				errors = append(errors, "Service requires spec")
			}
		}
	}

	return errors
}

func (n *YAMLNode) String() string {
	return fmt.Sprintf("YAMLNode(Name: %s, Value: %v, Tag: %s, Anchor: %s, IsAlias: %t)",
		n.Name(), n.Value, n.Tag, n.AnchorName, n.IsAlias)
}

func (n *YAMLNode) Type() ValueType {
	return ValueObject
}

// Implement YAMLCapable interface
func (n *YAMLNode) GetYAMLValue() interface{} {
	return n.Value
}

func (n *YAMLNode) SetYAMLValue(value interface{}) {
	n.Value = value
	n.modified = true
}
