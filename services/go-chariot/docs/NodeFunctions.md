# Chariot Language Reference

## Node Functions

Chariot nodes are the core data structure for representing trees, JSON, XML, CSV, YAML, and map-like objects. Node functions allow you to create, inspect, and manipulate these structures in a unified way.

---

### Node Creation Functions

| Function           | Description                                         |
|--------------------|-----------------------------------------------------|
| `create([name])`   | Create a new empty tree node (default name: `"node"`) |
| `jsonNode([jsonStr])` | Create a JSON node from a string or as empty     |
| `mapNode([mapStr])`   | Create a map node from a string or as empty      |
| `xmlNode([xmlStr])`   | Create an XML node from a string or as empty     |
| `csvNode([csvStr [, delimiter [, hasHeaders]]])` | Create a CSV node from a string or as empty |
| `yamlNode([yamlStr])` | Create a YAML node from a string or as empty     |

---

### Node Structure and Navigation Functions

| Function           | Description                                         |
|--------------------|-----------------------------------------------------|
| `addChild(parent, child)` | Add a child node to a parent node            |
| `removeChild(parent, child)` | Remove a child node from a parent         |
| `firstChild(node)` | Get the first child node (or `DBNull` if none)      |
| `lastChild(node)`  | Get the last child node (or `DBNull` if none)       |
| `getChildAt(node, idx)` | Get the child at index `idx` (or `DBNull` if out of bounds) |
| `getChildByName(node, name)` | Get a child node by name (or error if not found) |
| `childCount(node)` | Get the number of children                          |
| `clear(node)`      | Remove all children from a node                     |

---

### Attribute and Property Functions

| Function           | Description                                         |
|--------------------|-----------------------------------------------------|
| `getAttribute(node, key)` | Get the value of an attribute by key         |
| `setAttribute(node, key, value)` | Set the value of an attribute         |
| `setAttributes(node, mapValue)` | Set multiple attributes from a MapValue |
| `removeAttribute(node, key)` | Remove an attribute by key                |
| `hasAttribute(node, key)` | Returns `true` if the node has the attribute |
| `getName(node)`    | Get the name of the node                            |
| `setName(node, newName)` | Set the name of the node and return the new name |

---

### XML-Specific Content Functions

| Function           | Description                                         |
|--------------------|-----------------------------------------------------|
| `setText(node, text)` | Set the text content of an XML node              |
| `getText(node)`    | Get the text content of an XML node                 |

---

### Node Utility Functions

| Function           | Description                                         |
|--------------------|-----------------------------------------------------|
| `list(node)`       | List attribute keys (for JSON/map nodes) or child names (for tree nodes) as a comma-separated string |
| `nodeToString(node)` | Get a string representation of the node           |

---

### Node Hierarchy and Navigation

| Function           | Description                                         |
|--------------------|-----------------------------------------------------|
| `getParent(node)`  | Get the parent node (or `DBNull` if none)           |
| `getRoot(node)`    | Get the root node of the hierarchy                  |
| `getSiblings(node)`| Get all sibling nodes as an array                   |
| `getDepth(node)`   | Get the depth of the node in the tree               |
| `getLevel(node)`   | Get the level of the node in the tree               |
| `getPath(node)`    | Get the path from root to node as an array of names |
| `findByName(node, name)` | Find a descendant node by name                |
| `isLeaf(node)`     | Returns `true` if the node is a leaf (no children)  |
| `isRoot(node)`     | Returns `true` if the node is the root              |

---

### Node Traversal and Query

| Function           | Description                                         |
|--------------------|-----------------------------------------------------|
| `traverseNode(node, fn)` | Traverse the node tree, applying `fn` to each node |
| `queryNode(node, predicateFn)` | Return array of nodes matching predicate function |

---

### Example Usage

#### Tree Node Operations

```chariot
// Create a tree node
setq(customer, create("Customer"))

// Add attributes
setAttribute(customer, "name", "Alice")
setAttribute(customer, "age", 30)
setAttribute(customer, "active", true)

// Add children
setq(address, create("Address"))
setAttribute(address, "city", "New York")
setAttribute(address, "zip", "10001")
addChild(customer, address)

// Navigate structure
getAttribute(customer, "name")        // "Alice"
getName(customer)                     // "Customer"
childCount(customer)                  // 1
firstChild(customer)                  // Address node
getName(firstChild(customer))         // "Address"

// Check attributes
hasAttribute(customer, "name")        // true
hasAttribute(customer, "email")       // false
```

#### JSON Node Operations

```chariot
// Create from JSON string
setq(data, jsonNode('{"users": [{"id": 1, "name": "John"}, {"id": 2, "name": "Jane"}]}'))

// Navigate JSON structure
getAttribute(data, "users")           // Array of user objects
getChildByName(data, "users")         // Access users array as child

// Create empty JSON node and build structure
setq(config, jsonNode())
setAttribute(config, "version", "1.0")
setAttribute(config, "debug", true)
```

#### XML Node Operations

```chariot
// Create from XML string
setq(doc, xmlNode('<root><item id="1">Hello</item><item id="2">World</item></root>'))

// Navigate XML structure
firstChild(doc)                       // First item element
getText(firstChild(doc))              // "Hello"
setAttribute(firstChild(doc), "id", "100")

// Set text content
setText(firstChild(doc), "Modified text")
getText(firstChild(doc))              // "Modified text"
```

#### CSV Node Operations

```chariot
// Create from CSV string with custom delimiter
setq(csv, csvNode("name;age;city\nJohn;30;NYC\nJane;25;LA", ";", true))

// Access CSV data through node structure
childCount(csv)                       // Number of rows
firstChild(csv)                       // First data row
```

#### Map Node Operations

```chariot
// Create from map string
setq(map, mapNode('{"key1": "value1", "key2": "value2"}'))

// Access map data
getAttribute(map, "key1")             // "value1"
hasAttribute(map, "key2")             // true
```

#### YAML Node Operations

```chariot
// Create from YAML string
setq(yaml, yamlNode('name: John\nage: 30\nhobbies:\n  - reading\n  - coding'))

// Navigate YAML structure
getAttribute(yaml, "name")            // "John"
getAttribute(yaml, "age")             // 30
```

#### Node Utilities

```chariot
// List attributes or children
list(customer)                        // "name, age, active, Address"
list(data)                            // JSON attribute keys
list(doc)                             // XML child element names

// Get string representation
nodeToString(customer)                // String representation of tree
nodeToString(data)                    // JSON string
nodeToString(doc)                     // XML string

// Clear all children
clear(customer)                       // Remove all child nodes
childCount(customer)                  // 0
```

#### Batch Attribute Setting

```chariot
// Set multiple attributes at once
setq(attrs, mapValue("name", "Bob", "age", 35, "email", "bob@example.com"))
setAttributes(customer, attrs)

// Verify attributes were set
getAttribute(customer, "name")        // "Bob"
getAttribute(customer, "age")         // 35
getAttribute(customer, "email")       // "bob@example.com"
```

---

### Notes

- All node types support the TreeNode interface and can be used interchangeably for most operations.
- `setAttribute` and `getAttribute` work with all node types, storing attributes in the node's attribute map.
- `hasAttribute` checks both direct attributes and metadata for comprehensive attribute detection.
- `setText` and `getText` are XML-specific functions for manipulating text content within XML elements.
- `list` returns attribute keys for JSON/map nodes, or child names for tree nodes as a comma-separated string.
- `nodeToString` returns the appropriate string representation for each node type (JSON, XML, CSV, YAML, etc.).
- CSV nodes support custom delimiters and header row configuration.
- JSON nodes can be created from JSON strings or as empty objects for programmatic construction.
- XML nodes preserve element structure and support both attributes and text content.
- `getChildByName` provides direct access to named children without iteration.
- `setAttributes` allows batch setting of multiple attributes from a MapValue.
- All functions handle `ScopeEntry` unwrapping automatically for seamless variable access.

---