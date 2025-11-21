# Chariot Language Reference

## XML Functions

Chariot provides support for working with XML files and data structures. These functions allow you to load, save, parse, and manipulate XML data in your Chariot programs.

---

### Available XML Functions

| Function                     | Description                                                      |
|------------------------------|------------------------------------------------------------------|
| `loadXML(path)`              | Load an XML file and return as a TreeNode                        |
| `saveXML(treeNode, path)`    | Save a TreeNode as an XML file                                   |
| `loadXMLRaw(path)`           | Load an XML file as a raw string                                 |
| `saveXMLRaw(xmlStr, path)`   | Save a raw XML string to a file                                  |
| `parseXMLString(xmlStr)`     | Parse an XML string into a TreeNode                              |

---

### Function Details

#### `loadXML(path)`

Loads an XML file and parses it into a TreeNode structure.

**Parameters:**
- `path`: String path to the XML file

**Returns:** TreeNode representing the XML structure

**Example:**
```chariot
setq(doc, loadXML('data/config.xml'))
setq(root, getRoot(doc))
```

---

#### `saveXML(treeNode, path)`

Saves a TreeNode as an XML file with proper formatting.

**Parameters:**
- `treeNode`: A TreeNode to save
- `path`: String path where the XML file should be saved

**Returns:** `true` on success

**Example:**
```chariot
setq(doc, create('document', 'T'))
setq(root, create('config', 'T'))
addChild(doc, root)
saveXML(doc, 'data/config.xml')
```

---

#### `loadXMLRaw(path)`

Loads an XML file as a raw string without parsing.

**Parameters:**
- `path`: String path to the XML file

**Returns:** String containing the raw XML content

**Example:**
```chariot
setq(xmlContent, loadXMLRaw('templates/template.xml'))
setq(modified, replace(xmlContent, '{{version}}', '1.0.0'))
saveXMLRaw(modified, 'output/config.xml')
```

---

#### `saveXMLRaw(xmlStr, path)`

Saves a raw XML string to a file without parsing or validation.

**Parameters:**
- `xmlStr`: String containing XML content
- `path`: String path where the XML file should be saved

**Returns:** `true` on success

**Example:**
```chariot
setq(xml, '<?xml version="1.0"?>\n<root><item>value</item></root>')
saveXMLRaw(xml, 'data/simple.xml')
```

---

#### `parseXMLString(xmlStr)`

Parses an XML string into a TreeNode structure.

**Parameters:**
- `xmlStr`: String containing XML content

**Returns:** TreeNode representing the XML structure

**Example:**
```chariot
setq(xmlString, '<?xml version="1.0"?><root><item>value</item></root>')
setq(doc, parseXMLString(xmlString))
setq(root, getRoot(doc))
```

---

### Usage Patterns

#### Reading XML Files

```chariot
// Load XML document
setq(doc, loadXML('data/books.xml'))
setq(root, getRoot(doc))

// Navigate structure
setq(firstBook, firstChild(root))
setq(title, valueOf(firstBook, 'title'))
```

#### Creating XML Documents

```chariot
// Create document structure
setq(doc, create('document', 'T'))
setq(root, create('catalog', 'T'))
addChild(doc, root)

// Add book element
setq(book, create('book', 'T'))
setValue(book, 'id', '1')
setValue(book, 'title', 'Chariot Guide')
setValue(book, 'author', 'System')
addChild(root, book)

// Save as XML
saveXML(doc, 'data/catalog.xml')
```

#### Template Processing

```chariot
// Load XML template
setq(template, loadXMLRaw('templates/deployment.xml'))

// Replace placeholders
setq(xml, replace(template, '{{APP_NAME}}', 'myapp'))
setq(xml, replace(xml, '{{VERSION}}', '1.0.0'))

// Save processed XML
saveXMLRaw(xml, 'deployments/myapp.xml')
```

#### Parsing XML Strings

```chariot
// Receive XML from API or other source
setq(response, '<response><status>success</status><data>value</data></response>')

// Parse into TreeNode
setq(doc, parseXMLString(response))
setq(root, getRoot(doc))

// Extract values
setq(status, valueOf(root, 'status'))
setq(data, valueOf(root, 'data'))
```

---

### Notes

- XML files are parsed into TreeNode structures for manipulation
- XML attributes are stored as node values with special naming conventions
- Use `loadXMLRaw()` / `saveXMLRaw()` for template processing when you need to preserve exact formatting
- XML namespaces are supported in parsing but require special handling in navigation
- File paths are resolved relative to the Chariot runtime's data directory
- The XML parser preserves element hierarchy but may not preserve all formatting details

---

### See Also

- [TreeFunctions.md](TreeFunctions.md) - Functions for manipulating TreeNode structures
- [FileFunctions.md](FileFunctions.md) - Generic file I/O operations
- [JSONFunctions.md](JSONFunctions.md) - Similar operations for JSON files
