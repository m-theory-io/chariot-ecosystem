# Chariot Language Reference

## JSON Functions

Chariot provides robust support for parsing, generating, manipulating, and persisting JSON data. These functions allow you to convert between Chariot values and JSON, work with JSON nodes, perform lightweight or strict parsing, and handle JSON file I/O operations.

---

### Available JSON Functions

| Function                       | Description                                                      |
|--------------------------------|------------------------------------------------------------------|
| `parseJSON(str, [nodeName])`   | Parse a JSON string into a JSONNode with optional name          |
| `parseJSONSimple(str)`         | Parse a JSON string into a SimpleJSON node                       |
| `toJSON(value)`                | Convert a Chariot value or JSONNode to a JSON string             |
| `toSimpleJSON(value)`          | Convert a Chariot value to a SimpleJSON node                     |
| `loadJSON(path)`               | Load a JSON file and return as a JSONNode                        |
| `saveJSON(obj, path, [indent])`| Save an object as pretty JSON to a file                          |
| `loadJSONRaw(path)`            | Load a JSON file as a raw JSON string                            |
| `saveJSONRaw(jsonStr, path)`   | Save a raw JSON string to a file                                 |

---

### Function Details

#### `parseJSON(str, [nodeName])`

Parses a JSON string and returns a `JSONNode` object.  
- The node's properties are populated from the JSON object.
- Optional `nodeName` parameter sets the node's name (default: "root")
- Returns an error if the string is not valid JSON.

**Parameters:**
- `str`: JSON string to parse
- `nodeName` (optional): Name for the created JSONNode

**Returns:** JSONNode

**Example:**
```chariot
setq(config, parseJSON('{"host": "localhost", "port": 8080}', 'appConfig'))
setq(host, valueOf(config, 'host'))
```

---

#### `parseJSONSimple(str)`

Parses a JSON string and returns a `SimpleJSON` node.  
- Useful for lightweight or high-performance scenarios where full JSONNode features aren't needed
- Returns an error if the string is empty or invalid

**Parameters:**
- `str`: JSON string to parse

**Returns:** SimpleJSON node

**Example:**
```chariot
setq(data, parseJSONSimple('{"foo": "bar", "count": 42}'))
```

---

#### `toJSON(value)`

Converts a Chariot value or `JSONNode` to a JSON string.  
- If the value is a `JSONNode`, uses its serialization
- For other types, converts to native Go types and serializes
- Output is compact (no indentation)

**Parameters:**
- `value`: Chariot value, JSONNode, or SimpleJSON to convert

**Returns:** JSON string

**Example:**
```chariot
setq(config, create('config', 'J'))
setValue(config, 'name', 'myapp')
setValue(config, 'version', '1.0')
setq(jsonStr, toJSON(config))
// Result: '{"name":"myapp","version":"1.0"}'
```

---

#### `toSimpleJSON(value)`

Converts a Chariot value to a `SimpleJSON` node.  
- Converts to native Go types, then wraps as a `SimpleJSON`
- Useful for creating lightweight JSON structures

**Parameters:**
- `value`: Chariot value to convert

**Returns:** SimpleJSON node

**Example:**
```chariot
setq(data, create('data', 'M'))
setValue(data, 'x', 10)
setValue(data, 'y', 20)
setq(jsonNode, toSimpleJSON(data))
```

---

#### `loadJSON(path)`

Loads a JSON file from disk and parses it into a JSONNode.

**Parameters:**
- `path`: String path to the JSON file

**Returns:** JSONNode representing the file contents

**Example:**
```chariot
setq(config, loadJSON('config/app.json'))
setq(dbHost, valueOf(config, 'database.host'))
setq(dbPort, valueOf(config, 'database.port'))
```

---

#### `saveJSON(obj, path, [indent])`

Saves a Chariot value or JSONNode as a formatted JSON file.

**Parameters:**
- `obj`: Chariot value or JSONNode to save
- `path`: String path where the JSON file should be saved
- `indent` (optional): Number of spaces for indentation (default: 2)

**Returns:** `true` on success

**Example:**
```chariot
setq(config, create('config', 'J'))
setValue(config, 'server.host', 'localhost')
setValue(config, 'server.port', 8080)
saveJSON(config, 'config/server.json', 4)
```

---

#### `loadJSONRaw(path)`

Loads a JSON file as a raw string without parsing.

**Parameters:**
- `path`: String path to the JSON file

**Returns:** String containing the raw JSON content

**Example:**
```chariot
setq(jsonStr, loadJSONRaw('data/template.json'))
setq(modified, replace(jsonStr, '${VERSION}', '1.0.0'))
saveJSONRaw(modified, 'data/processed.json')
```

---

#### `saveJSONRaw(jsonStr, path)`

Saves a raw JSON string to a file without parsing or formatting.

**Parameters:**
- `jsonStr`: String containing JSON content
- `path`: String path where the JSON file should be saved

**Returns:** `true` on success

**Example:**
```chariot
setq(json, '{"status":"ok","timestamp":1234567890}')
saveJSONRaw(json, 'logs/status.json')
```

---

### Usage Patterns

#### Loading and Saving Configuration

```chariot
// Load existing configuration
setq(config, loadJSON('config/app.json'))

// Modify values
setValue(config, 'logging.level', 'debug')
setValue(config, 'server.maxConnections', 100)

// Save back to file
saveJSON(config, 'config/app.json')
```

#### Building JSON from Scratch

```chariot
// Create new JSONNode
setq(user, create('user', 'J'))
setValue(user, 'id', 12345)
setValue(user, 'username', 'johndoe')
setValue(user, 'email', 'john@example.com')
setValue(user, 'active', true)

// Save as JSON file
saveJSON(user, 'users/12345.json')
```

#### Template Processing

```chariot
// Load JSON template
setq(template, loadJSONRaw('templates/deployment.json'))

// Replace placeholders
setq(json, replace(template, '{{APP_NAME}}', 'myapp'))
setq(json, replace(json, '{{VERSION}}', '2.0.0'))
setq(json, replace(json, '{{REPLICAS}}', '3'))

// Save processed JSON
saveJSONRaw(json, 'deployments/myapp.json')
```

#### API Response Processing

```chariot
// Parse API response
setq(response, parseJSON(apiResponse))

// Extract data
setq(items, valueOf(response, 'data.items'))
setq(total, valueOf(response, 'data.total'))

// Save for caching
saveJSON(response, 'cache/api-response.json')
```

#### Batch JSON Processing

```chariot
// List all JSON files
setq(files, listFiles('data'))

// Process each file
setq(i, 0)
while(lessThan(i, length(files)),
  setq(file, at(files, i)),
  if(endsWith(file, '.json'),
    setq(path, concat('data/', file)),
    setq(data, loadJSON(path)),
    // Process data...
    setValue(data, 'processed', true),
    saveJSON(data, path)
  ),
  setq(i, add(i, 1))
)
```

---

### Notes

- All JSON parsing functions return an error if the input is not valid JSON
- `parseJSON` returns a full-featured JSONNode for complex manipulation
- `parseJSONSimple` is optimized for performance with simpler use cases
- `toJSON` always returns a compact string; use `saveJSON` with `indent` for readable output
- File I/O functions (`loadJSON`, `saveJSON`, etc.) handle file operations automatically
- Use `loadJSONRaw` / `saveJSONRaw` for template processing when you need exact control over formatting
- All file paths are resolved relative to the Chariot runtime's data directory
- JSONNodes support nested access via dot notation in `valueOf()` and `setValue()`

---

### See Also

- [FileFunctions.md](FileFunctions.md) - Generic file I/O operations
- [YAMLFunctions.md](YAMLFunctions.md) - Similar operations for YAML files
- [FormatConversionFunctions.md](FormatConversionFunctions.md) - Converting between JSON and YAML formats
- [NodeFunctions.md](NodeFunctions.md) - Functions for manipulating node structures
