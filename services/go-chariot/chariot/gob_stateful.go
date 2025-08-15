package chariot

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"reflect"
	"unsafe"
)

// Register types with GOB during package initialization
func init() {
	gob.Register(GobReference{})
	gob.Register(GobTreeNode{})
	gob.Register(map[string]interface{}{})
	gob.Register([]interface{}{})
}

type StatefulGobSerializer struct {
	visited   map[uintptr]int // object pointer -> reference ID
	objects   []interface{}   // reference ID -> object
	nextRefID int
}

type GobReference struct {
	RefID int `gob:"ref_id"`
}

type GobTreeNode struct {
	Name       string                 `gob:"name"`
	Attributes map[string]interface{} `gob:"attributes,omitempty"`
	Children   []interface{}          `gob:"children,omitempty"` // Can be GobTreeNode or GobReference
	NodeType   string                 `gob:"node_type"`
}

func NewStatefulGobSerializer() *StatefulGobSerializer {
	return &StatefulGobSerializer{
		visited:   make(map[uintptr]int),
		objects:   make([]interface{}, 0),
		nextRefID: 0,
	}
}

func (s *StatefulGobSerializer) SerializeTree(node TreeNode) ([]byte, error) {
	// Reset state
	s.visited = make(map[uintptr]int)
	s.objects = make([]interface{}, 0)
	s.nextRefID = 0

	// Convert tree to stateful representation
	gobNode := s.convertNodeStateful(node)

	// Create wrapper with reference table
	wrapper := struct {
		Root       interface{}   `gob:"root"`
		References []interface{} `gob:"references"`
	}{
		Root:       gobNode,
		References: s.objects,
	}

	var buf bytes.Buffer
	encoder := gob.NewEncoder(&buf)
	err := encoder.Encode(wrapper)
	return buf.Bytes(), err
}

func (s *StatefulGobSerializer) convertNodeStateful(node TreeNode) interface{} {
	if node == nil {
		return nil
	}

	// Get object pointer
	nodePtr := getObjectPointer(node)

	// Check if already visited
	if refID, exists := s.visited[nodePtr]; exists {
		return GobReference{RefID: refID}
	}

	// Mark as visited and assign reference ID
	refID := s.nextRefID
	s.visited[nodePtr] = refID
	s.nextRefID++

	// Convert to serializable form
	gobNode := GobTreeNode{
		Name:     node.Name(), // Fixed: use GetName() not Name()
		NodeType: fmt.Sprintf("%T", node),
	}

	// Convert attributes safely
	if attrs := node.GetAttributes(); attrs != nil {
		gobNode.Attributes = make(map[string]interface{})
		for k, v := range attrs {
			gobNode.Attributes[k] = s.convertValueStateful(v)
		}
	}

	// Store in reference table first
	s.objects = append(s.objects, gobNode)

	// Convert children (this is where cycles would normally break naive GOB)
	children := node.GetChildren()
	if len(children) > 0 {
		gobNode.Children = make([]interface{}, len(children))
		for i, child := range children {
			gobNode.Children[i] = s.convertNodeStateful(child)
		}

		// Update the stored object with children
		s.objects[refID] = gobNode
	}

	return GobReference{RefID: refID}
}

// Fix the convertValueStateful function to preserve array types
func (s *StatefulGobSerializer) convertValueStateful(v Value) interface{} {
	if v == nil {
		return nil
	}

	switch val := v.(type) {
	case string, int, int64, float64, bool:
		return val
	case Str:
		return string(val)
	case Number:
		return float64(val)
	case Bool:
		return bool(val)
	case *ArrayValue:
		// Convert ArrayValue to a special wrapper to preserve type
		result := make([]interface{}, val.Length())
		for i := 0; i < val.Length(); i++ {
			result[i] = s.convertValueStateful(val.Get(i))
		}
		return map[string]interface{}{
			"_chariot_type": "ArrayValue",
			"_elements":     result,
		}
	case []Value:
		result := make([]interface{}, len(val))
		for i, item := range val {
			result[i] = s.convertValueStateful(item)
		}
		return map[string]interface{}{
			"_chariot_type": "ValueArray",
			"_elements":     result,
		}
	case map[string]Value:
		result := make(map[string]interface{})
		for k, item := range val {
			result[k] = s.convertValueStateful(item)
		}
		return result
	case TreeNode:
		return s.convertNodeStateful(val)
	default:
		return fmt.Sprintf("%v", val)
	}
}

// Safe helper to get object pointer using reflection
func getObjectPointer(obj interface{}) uintptr {
	if obj == nil {
		return 0
	}

	v := reflect.ValueOf(obj)

	// Handle different kinds of values
	switch v.Kind() {
	case reflect.Ptr:
		if v.IsNil() {
			return 0
		}
		return v.Pointer()
	case reflect.Interface:
		if v.IsNil() {
			return 0
		}
		// Get the underlying concrete value
		elem := v.Elem()
		if elem.Kind() == reflect.Ptr {
			return elem.Pointer()
		}
		// For non-pointer interface values, use the address of the interface header
		return uintptr(unsafe.Pointer(&obj))
	default:
		// For value types, use the address of the interface header
		return uintptr(unsafe.Pointer(&obj))
	}
}

// Add the deserializer too
type StatefulGobDeserializer struct {
	references []interface{}
}

func (s *StatefulGobDeserializer) DeserializeTree(data []byte) (TreeNode, error) {
	var wrapper struct {
		Root       interface{}   `gob:"root"`
		References []interface{} `gob:"references"`
	}

	buf := bytes.NewBuffer(data)
	decoder := gob.NewDecoder(buf)
	err := decoder.Decode(&wrapper)
	if err != nil {
		return nil, fmt.Errorf("failed to decode GOB wrapper: %w", err)
	}

	s.references = wrapper.References

	// Convert root back to TreeNode
	return s.convertToTreeNode(wrapper.Root)
}

func (s *StatefulGobDeserializer) convertToTreeNode(obj interface{}) (TreeNode, error) {
	switch v := obj.(type) {
	case GobReference:
		if v.RefID < 0 || v.RefID >= len(s.references) {
			return nil, fmt.Errorf("invalid reference ID: %d", v.RefID)
		}
		return s.convertToTreeNode(s.references[v.RefID])

	case GobTreeNode:
		// Create appropriate node type based on NodeType
		var node TreeNode

		switch v.NodeType {
		case "*chariot.TreeNodeImpl":
			node = NewTreeNode(v.Name)
		case "*chariot.JSONNode":
			node = NewJSONNode(v.Name)
		case "*chariot.XMLNode":
			node = NewXMLNode(v.Name)
		default:
			node = NewTreeNode(v.Name)
		}

		// Set attributes
		for k, val := range v.Attributes {
			node.SetAttribute(k, convertToValue(val))
		}

		// Add children
		for _, childObj := range v.Children {
			child, err := s.convertToTreeNode(childObj)
			if err != nil {
				return nil, err
			}
			if child != nil {
				node.AddChild(child)
			}
		}

		return node, nil

	default:
		return nil, fmt.Errorf("unknown object type in deserialization: %T", obj)
	}
}
