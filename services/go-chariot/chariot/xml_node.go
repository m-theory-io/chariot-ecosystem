// xml_node.go
package chariot

import (
	"encoding/xml"
	"fmt"
	"io"
	"strings"
)

// XMLNode implements TreeNode for XML data
type XMLNode struct {
	TreeNodeImpl
	// XML-specific fields
	Namespace     string
	XMLAttributes map[string]string // XML-specific attributes
	Content       string            // Text content of the node
	IsCDATA       bool              // Whether this node contains CDATA
	IsComment     bool              // Whether this node is a comment
}

// NewXMLNode creates a new XMLNode with the given name
func NewXMLNode(name string) *XMLNode {
	node := &XMLNode{
		XMLAttributes: make(map[string]string),
	}
	node.TreeNodeImpl = *NewTreeNode(name)
	return node
}

func (n *XMLNode) GetTypeLabel() string {
	return "XMLNode" // Return a string label for the type
}

// Clone makes a deep copy of the XMLNode
func (n *XMLNode) Clone() TreeNode {
	clone := &XMLNode{
		Content:       n.Content,
		XMLAttributes: make(map[string]string),
		IsCDATA:       n.IsCDATA,
		IsComment:     n.IsComment,
		Namespace:     n.Namespace,
		TreeNodeImpl:  *(n.TreeNodeImpl.Clone().(*TreeNodeImpl)),
	}

	// Clone attributes
	for k, v := range n.XMLAttributes {
		clone.XMLAttributes[k] = v
	}

	// Clone children
	for i, child := range n.Children {
		childClone := child.Clone()
		clone.Children[i] = childClone
		if childNode, ok := childClone.(*XMLNode); ok {
			childNode.ParentNode = clone // Set parent reference
			if n.IsCDATA {
				childNode.IsCDATA = n.IsCDATA
			}
		}
	}

	return clone
}

// NewXMLNodeWithNamespace creates an XMLNode with namespace
func NewXMLNodeWithNamespace(name, namespace string) *XMLNode {
	node := NewXMLNode(name)
	node.Namespace = namespace
	return node
}

// SetContent sets the text content of this XML node
func (n *XMLNode) SetContent(content string) {
	n.Content = content
}

// GetContent returns the text content of this XML node
func (n *XMLNode) GetContent() string {
	return n.Content
}

// SetXMLAttribute sets an XML-specific attribute
func (n *XMLNode) SetXMLAttribute(key, value string) {
	n.XMLAttributes[key] = value
}

// GetXMLAttribute gets an XML-specific attribute
func (n *XMLNode) GetXMLAttribute(key string) (string, bool) {
	val, ok := n.XMLAttributes[key]
	return val, ok
}

// SetCDATA marks this node as containing CDATA
func (n *XMLNode) SetCDATA(isCDATA bool) {
	n.IsCDATA = isCDATA
}

// SetComment marks this node as a comment
func (n *XMLNode) SetComment(isComment bool) {
	n.IsComment = isComment
}

// FullName returns the qualified name with namespace if present
func (n *XMLNode) FullName() string {
	if n.Namespace != "" {
		return fmt.Sprintf("%s:%s", n.Namespace, n.Name())
	}
	return n.Name()
}

// AddChildElement creates and adds a new XML element as child
func (n *XMLNode) AddChildElement(name string) *XMLNode {
	child := NewXMLNode(name)
	n.AddChild(child)
	return child
}

// FindElementsByTagName finds all descendant elements with the given tag name
func (n *XMLNode) FindElementsByTagName(tagName string) []*XMLNode {
	var result []*XMLNode

	_ = n.Traverse(func(node TreeNode) error {
		if xmlNode, ok := node.(*XMLNode); ok {
			if xmlNode.Name() == tagName {
				result = append(result, xmlNode)
			}
		}
		return nil
	})

	return result
}

// MarshalXML returns the XML representation of this node and its children
func (n *XMLNode) MarshalXML(encoder *xml.Encoder, start xml.StartElement) error {
	start.Name.Local = n.Name()

	// Add XML attributes
	for key, value := range n.XMLAttributes {
		start.Attr = append(start.Attr, xml.Attr{Name: xml.Name{Local: key}, Value: value})
	}

	// Encode the start element
	if err := encoder.EncodeToken(start); err != nil {
		return err
	}

	// Add content, properly escaped or as CDATA
	if n.Content != "" {
		if n.IsCDATA {
			if err := encoder.EncodeToken(xml.CharData([]byte(fmt.Sprintf("<![CDATA[%s]]>", n.Content)))); err != nil {
				return err
			}
		} else {
			if err := encoder.EncodeToken(xml.CharData([]byte(n.Content))); err != nil {
				return err
			}
		}
	}

	// Add children
	for _, child := range n.Children {
		if xmlChild, ok := child.(*XMLNode); ok {
			if err := encoder.EncodeElement(xmlChild, xml.StartElement{Name: xml.Name{Local: xmlChild.Name()}}); err != nil {
				return err
			}
		}
	}

	// Encode the end element
	if err := encoder.EncodeToken(start.End()); err != nil {
		return err
	}

	return encoder.Flush()
}

// UnmarshalXML parses XML data into this node and its children
func (n *XMLNode) UnmarshalXML(decoder *xml.Decoder, start xml.StartElement) error {
	// Reset node state
	n.Children = nil
	n.XMLAttributes = make(map[string]string)
	n.Content = ""

	// Process attributes
	for _, attr := range start.Attr {
		n.XMLAttributes[attr.Name.Local] = attr.Value
	}

	// Parse the content and children
	for {
		token, err := decoder.Token()
		if err != nil {
			if err == io.EOF {
				break
			}
			return err
		}

		switch t := token.(type) {
		case xml.StartElement:
			child := NewXMLNode(t.Name.Local)
			if err := child.UnmarshalXML(decoder, t); err != nil {
				return err
			}
			n.AddChild(child)
		case xml.CharData:
			n.Content = string(t)
		case xml.EndElement:
			if t.Name.Local == start.Name.Local {
				return nil
			}
		}
	}
	return nil
}

// SelectSingleNode finds a node by XPath-like expression (simplified)
func (n *XMLNode) SelectSingleNode(path string) *XMLNode {
	parts := strings.Split(path, "/")
	if len(parts) == 0 {
		return nil
	}

	// Handle absolute vs relative paths
	currentNode := n
	startIdx := 0

	if parts[0] == "" {
		// Absolute path, start from root
		for currentNode.Parent() != nil {
			if parent, ok := currentNode.Parent().(*XMLNode); ok {
				currentNode = parent
			} else {
				break
			}
		}
		startIdx = 1
	}

	// Navigate the path
	for i := startIdx; i < len(parts); i++ {
		if parts[i] == "" {
			continue
		}

		found := false
		for _, child := range currentNode.Children {
			if xmlChild, ok := child.(*XMLNode); ok && xmlChild.Name() == parts[i] {
				currentNode = xmlChild
				found = true
				break
			}
		}

		if !found {
			return nil
		}
	}

	return currentNode
}

// ToXML returns the XML string representation of this node and its children
func (n *XMLNode) ToXML() (string, error) {
	var builder strings.Builder
	encoder := xml.NewEncoder(&builder)
	start := xml.StartElement{Name: xml.Name{Local: n.Name()}}
	if err := n.MarshalXML(encoder, start); err != nil {
		return "", err
	}
	if err := encoder.Flush(); err != nil {
		return "", err
	}
	data := builder.String()
	return string(data), nil
}
