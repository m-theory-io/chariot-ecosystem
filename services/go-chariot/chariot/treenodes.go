package chariot

import (
	"fmt"
	"io"
)

// XMLCapable defines XML-specific operations
type XMLCapable interface {
	SetContent(content string)
	GetContent() string
	SetCDATA(isCDATA bool)
	SetComment(isComment bool)
	SetXMLAttribute(key, value string)
	GetXMLAttribute(key string) (string, bool)
	ToXML() (string, error)
	SelectSingleNode(path string) *XMLNode
}

// JSONCapable defines JSON-specific operations
type JSONCapable interface {
	DecodeStream(r io.Reader, callback func(*JSONNode, string) bool) error
	GetJSONValue() interface{}
	SetJSONValue(value interface{})
	GetJSONObject() map[string]interface{}
	GetJSONArray() []interface{}
	IsJSONArray() bool
	IsJSONObject() bool
	ToJSON() (string, error)
}

// YAMLCapable defines YAML-specific operations
type YAMLCapable interface {
	GetYAMLValue() interface{}
	SetYAMLValue(value interface{})
	ToYAML() (string, error)
}

// SQLCapable defines SQL-specific operations
type SQLCapable interface {
	// Connection operations
	Connect(driverName, connStr string) error
	Close() error

	// Transaction management
	Begin() error
	Commit() error
	Rollback() error

	// Query execution
	QuerySQL(query string, args ...interface{}) ([]TreeNode, error)
	QuerySQLStream(query string, callback func(int, map[string]interface{}) bool, args ...interface{}) error
	Execute(stmt string, args ...interface{}) (int64, error)

	// Result access
	GetColumnNames() []string
	GetRowCount() int
	GetCell(row int, col interface{}) (interface{}, error)
	GetRow(row int) (map[string]interface{}, error)

	// Schema operations
	ListTables() ([]TreeNode, error)
	DescribeTable(tableName string) ([]TreeNode, error)

	// Configuration
	SetQueryTimeout(seconds int) error

	// Format conversion
	ToCSVNode() (*CSVNode, error)

	// Error handling
	GetLastError() error
}

// TreeNode represents any hierarchical node in a tree structure
type TreeNode interface {
	Value                                 // Embed Value interface so TreeNodes are Values
	Name() string                         // Get node name/identifier
	GetChildren() []TreeNode              // Get child nodes
	AddChild(child TreeNode)              // Add a child node
	AddChildAt(index int, child TreeNode) // Add a child at a specific index
	Clone() TreeNode                      // Create a deep copy of this node
	RemoveChild(child TreeNode)           // Remove a child from this node
	RemoveChildren()                      // Remove all children from this node
	Parent() TreeNode                     // Get parent node
	SetParent(parent TreeNode)            // Set parent node
	GetAttribute(name string) (Value, bool)
	SetAttribute(name string, value Value)
	GetMeta(key string) (Value, bool) // Returns Chariot Value type
	SetMeta(key string, value Value)  // Takes Chariot Value type
	HasMeta(key string) bool
	ClearMeta()
	GetAllMeta() *MapValue         // Return all metadata as MapValue
	SetAllMeta(meta *MapValue)     // Set all metadata from MapValue
	SetName(newName string) string // Set and return the name of the node
	RemoveAttribute(name string)
	GetAttributes() map[string]Value
	GetType() ValueType   // Returns the type of the node (ValueObject, ValueArray, etc.)
	GetTypeLabel() string // Returns a string label for the type (e.g., "TreeNode", "MapNode", etc.)
	// Optional: method to traverse the tree, applying the function to each node
	Traverse(fn func(TreeNode) error) error
	// Optional: method to find a node by name
	FindByName(name string) (TreeNode, bool)
	// Optional: method to get the depth of the node in the tree
	GetDepth() int
	// Optional: method to get the path from the root to this node
	GetPath() []string
	// Optional method to get the root node of the tree
	GetRoot() TreeNode
	// Optional: method to get the sibling nodes
	GetSiblings() []TreeNode
	// Optional: method to get the first child
	GetFirstChild() TreeNode
	// Optional: method to get the last child
	GetLastChild() TreeNode
	// Optional: method to get the number of children
	GetChildCount() int
	// GetChildByName(name string) (TreeNode, bool) // Optional: method to get a child by name
	GetChildByName(name string) (TreeNode, bool)
	// Optional: method to check if the node is a leaf (no children)
	IsLeaf() bool
	// Optional: method to check if the node is a root (no parent)
	IsRoot() bool
	// Optional: method to get the level of the node in the tree
	GetLevel() int
	// Optional: method to query the subtree rooted by this node
	QueryTree(fn func(TreeNode) bool) []TreeNode
	// Optional: method to return a string representation of the node
	String() string
}

// Add this to your value types
type TreeNodeValue struct {
	Node TreeNode
}

func (tnv TreeNodeValue) String() string {
	return fmt.Sprintf("TreeNode<%s>", tnv.Node.GetTypeLabel())
}

// TreeNodeImpl is a basic implementation of the TreeNode interface
type TreeNodeImpl struct {
	NameStr    string
	Children   []TreeNode
	ParentNode TreeNode
	Attributes map[string]Value
	meta       *MapValue // dedicated metadata field
}

// NewTreeNode creates a new TreeNode with the given name
func NewTreeNode(name string) *TreeNodeImpl {
	tnode := &TreeNodeImpl{
		NameStr:    name,
		Children:   []TreeNode{},
		ParentNode: nil,
		Attributes: make(map[string]Value),
		meta:       NewMap(), // Initialize metadata map
	}
	return tnode
}

func (n *TreeNodeImpl) GetType() ValueType {
	return ValueObject // All TreeNodes are considered objects
}

func (n *TreeNodeImpl) GetTypeLabel() string {
	return "TreeNode" // Return a string label for the type
}

// Name returns the name of the node
func (n *TreeNodeImpl) Name() string {
	return n.NameStr
}

// Name sets and returns the name of the node
func (n *TreeNodeImpl) SetName(newName string) string {
	n.NameStr = newName
	return n.NameStr
}

// Clone creates a deep copy of the TreeNode
func (n *TreeNodeImpl) Clone() TreeNode {
	// Create a new node with the same name
	clone := NewTreeNode(n.NameStr)

	// Copy attributes
	for key, value := range n.Attributes {
		clone.SetAttribute(key, value)
	}

	// Copy metadata
	if n.meta != nil {
		for _, key := range n.meta.Keys() {
			if val, ok := n.meta.Get(key); ok {
				clone.SetMeta(key, val)
			}
		}
	}

	// Recursively clone children
	for _, child := range n.Children {
		cloneChild := child.Clone()
		clone.AddChild(cloneChild)
	}

	// Set parent to nil for cloned node
	clone.SetParent(nil)

	return clone
}

// GetAllMeta returns all metadata as a MapValue
func (n *TreeNodeImpl) GetAllMeta() *MapValue {
	if n.meta == nil {
		n.meta = NewMap() // Ensure metadata map is initialized
	}
	return n.meta
}

// GetChildren returns the child nodes
func (n *TreeNodeImpl) GetChildren() []TreeNode {
	return n.Children
}

// GetChildByName returns a child node by name
func (n *TreeNodeImpl) GetChildByName(name string) (TreeNode, bool) {
	for _, child := range n.Children {
		if child.Name() == name {
			return child, true
		}
	}
	return nil, false
}

// GetRoot returns the root node of the tree
func (n *TreeNodeImpl) GetRoot() TreeNode {
	current := TreeNode(n)
	for current.Parent() != nil { // Use interface method
		current = current.Parent() // Use interface method
	}
	return current
}

// AddChild adds a child node
func (n *TreeNodeImpl) AddChild(child TreeNode) {
	n.Children = append(n.Children, child)
	child.SetParent(n)
}

// AddChildAt adds a child node at the specified index position
func (n *TreeNodeImpl) AddChildAt(index int, child TreeNode) {
	// First check if child already has a parent, and detach if needed
	if child.Parent() != nil {
		child.Parent().RemoveChild(child)
	}

	// Set this node as the child's parent
	if setParenter, ok := child.(interface{ SetParent(TreeNode) }); ok {
		setParenter.SetParent(n)
	}

	// Handle index out of bounds
	if index < 0 {
		index = 0
	}

	// If index is beyond the current children count, just append
	if index >= len(n.Children) {
		n.Children = append(n.Children, child)
		return
	}

	// Insert at specific position by making room and shifting elements
	n.Children = append(n.Children, nil) // Extend slice by one

	// Move all elements after index position
	copy(n.Children[index+1:], n.Children[index:])

	// Place new child at the index position
	n.Children[index] = child
}

// RemoveChild removes a child node
func (n *TreeNodeImpl) RemoveChild(child TreeNode) {
	for i, c := range n.Children {
		if c == child {
			n.Children = append(n.Children[:i], n.Children[i+1:]...)
			child.SetParent(nil)
			break
		}
	}
}

// RemoveChildren removes all child nodes
func (n *TreeNodeImpl) RemoveChildren() {
	for _, child := range n.Children {
		child.SetParent(nil)
	}
	n.Children = []TreeNode{}
}

// Parent returns the parent node
func (n *TreeNodeImpl) Parent() TreeNode {
	return n.ParentNode
}

// SetParent sets the parent node
func (n *TreeNodeImpl) SetParent(parent TreeNode) {
	n.ParentNode = parent
}

// Standard metadata methods for ALL TreeNode instances
func (n *TreeNodeImpl) GetMeta(key string) (Value, bool) {
	if n.meta == nil {
		n.meta = NewMap() // Ensure metadata map is initialized
	}
	return n.meta.Get(key)
}

func (n *TreeNodeImpl) SetMeta(key string, value Value) {
	// Get or create _meta map
	if n.meta == nil {
		n.meta = NewMap()
	}
	n.meta.Set(key, value)
}

// SetAllMeta sets all metadata from a MapValue
func (n *TreeNodeImpl) SetAllMeta(meta *MapValue) {
	n.meta = meta
}

// ClearMeta resets the meta map to an empty state
func (n *TreeNodeImpl) ClearMeta() {
	n.meta = NewMap()
}

// HasMeta checks if the metadata contains the specified key
func (n *TreeNodeImpl) HasMeta(key string) bool {
	if n.meta == nil {
		n.meta = NewMap()
	}
	_, exists := n.meta.Get(key)
	return exists
}

// GetAttribute returns the value of the specified attribute
func (n *TreeNodeImpl) GetAttribute(name string) (Value, bool) {
	if val, ok := n.Attributes[name]; ok {
		return val, true
	}
	return nil, false
}

// SetAttribute sets the value of the specified attribute
func (n *TreeNodeImpl) SetAttribute(name string, value Value) {
	n.Attributes[name] = value
}

// RemoveAttribute removes the specified attribute
func (n *TreeNodeImpl) RemoveAttribute(name string) {
	delete(n.Attributes, name)
}

// GetAttributes returns all attributes
func (n *TreeNodeImpl) GetAttributes() map[string]Value {
	return n.Attributes
}

// Traverse applies the function to each node in the tree
func (n *TreeNodeImpl) Traverse(fn func(TreeNode) error) error {
	if err := fn(n); err != nil {
		return err
	}
	for _, child := range n.Children {
		if err := child.Traverse(fn); err != nil {
			return err
		}
	}
	return nil
}

// FindByName finds a node by name
func (n *TreeNodeImpl) FindByName(name string) (TreeNode, bool) {
	if n.NameStr == name {
		return n, true
	}
	for _, child := range n.Children {
		if found, ok := child.FindByName(name); ok {
			return found, true
		}
	}
	return nil, false
}

// GetDepth returns the depth of the node in the tree
func (n *TreeNodeImpl) GetDepth() int {
	depth := 0
	current := n.Parent() // Use interface method, not field
	for current != nil {
		depth++
		current = current.Parent() // ← Use interface method
	}
	return depth
}

// GetPath returns the path from the root to this node (computed on-demand)
func (n *TreeNodeImpl) GetPath() []string {
	// Count depth first to pre-allocate slice
	depth := n.GetDepth() + 1
	path := make([]string, depth)

	// Fill from end to beginning using interface methods
	current := TreeNode(n) // Start with this node as interface
	for i := depth - 1; i >= 0; i-- {
		path[i] = current.Name()   // Use interface method
		current = current.Parent() // Use interface method
	}

	return path
}

// GetSiblings returns the sibling nodes
func (n *TreeNodeImpl) GetSiblings() []TreeNode {
	if n.Parent() == nil {
		return nil
	}
	siblings := []TreeNode{}
	for _, sibling := range n.Parent().GetChildren() {
		if sibling != n {
			siblings = append(siblings, sibling)
		}
	}
	return siblings
}

// GetFirstChild returns the first child node
func (n *TreeNodeImpl) GetFirstChild() TreeNode {
	if len(n.Children) > 0 {
		return n.Children[0]
	}
	return nil
}

// GetLastChild returns the last child node
func (n *TreeNodeImpl) GetLastChild() TreeNode {
	if len(n.Children) > 0 {
		return n.Children[len(n.Children)-1]
	}
	return nil
}

// GetChildCount returns the number of child nodes
func (n *TreeNodeImpl) GetChildCount() int {
	return len(n.Children)
}

// IsLeaf checks if the node is a leaf (no children)
func (n *TreeNodeImpl) IsLeaf() bool {
	return len(n.Children) == 0
}

// IsRoot checks if the node is a root (no parent)
func (n *TreeNodeImpl) IsRoot() bool {
	return n.ParentNode == nil
}

// GetLevel returns the level of the node in the tree
func (n *TreeNodeImpl) GetLevel() int {
	return n.GetDepth() + 1 // Level is depth + 1 (root is level 1)
}

// Query returns the nodes that match the query function
func (n *TreeNodeImpl) QueryTree(fn func(TreeNode) bool) []TreeNode {
	matches := []TreeNode{}
	if fn(n) {
		matches = append(matches, n)
	}
	for _, child := range n.Children {
		matches = append(matches, child.QueryTree(fn)...)
	}
	return matches
}

func (n *TreeNodeImpl) String() string {
	return fmt.Sprintf("Node(%s)", n.NameStr)
}

func (n *TreeNodeImpl) Type() ValueType {
	return ValueObject
}
