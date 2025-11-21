# Chariot Language Reference

## YAML Functions

Chariot provides comprehensive support for working with YAML files and data structures. These functions allow you to load, save, and manipulate YAML data in your Chariot programs.

---

### Available YAML Functions

| Function                              | Description                                                      |
|---------------------------------------|------------------------------------------------------------------|
| `loadYAML(path)`                      | Load a YAML file and return as a JSONNode                        |
| `saveYAML(jsonNode, path)`            | Save a JSONNode as a YAML file                                   |
| `loadYAMLRaw(path)`                   | Load a YAML file as a raw string                                 |
| `saveYAMLRaw(yamlStr, path)`          | Save a raw YAML string to a file                                 |
| `loadYAMLMultiDoc(path)`              | Load a multi-document YAML file as an array of JSONNodes         |
| `saveYAMLMultiDoc(jsonNodeArray, path)` | Save an array of JSONNodes as a multi-document YAML file       |

---

### Function Details

#### `loadYAML(path)`

Loads a YAML file and parses it into a JSONNode structure.

**Parameters:**
- `path`: String path to the YAML file

**Returns:** JSONNode representing the YAML structure

**Example:**
```chariot
setq(config, loadYAML('config/app.yaml'))
setq(port, valueOf(config, 'server.port'))
```

---

#### `saveYAML(jsonNode, path)`

Saves a JSONNode as a YAML file with proper formatting.

**Parameters:**
- `jsonNode`: A JSONNode to save
- `path`: String path where the YAML file should be saved

**Returns:** `true` on success

**Example:**
```chariot
setq(config, create('appConfig', 'J'))
setValue(config, 'server.port', 8080)
setValue(config, 'server.host', 'localhost')
saveYAML(config, 'config/app.yaml')
```

---

#### `loadYAMLRaw(path)`

Loads a YAML file as a raw string without parsing.

**Parameters:**
- `path`: String path to the YAML file

**Returns:** String containing the raw YAML content

**Example:**
```chariot
setq(yamlContent, loadYAMLRaw('config/template.yaml'))
setq(modifiedYAML, replace(yamlContent, '${PORT}', '8080'))
saveYAMLRaw(modifiedYAML, 'config/app.yaml')
```

---

#### `saveYAMLRaw(yamlStr, path)`

Saves a raw YAML string to a file without parsing or validation.

**Parameters:**
- `yamlStr`: String containing YAML content
- `path`: String path where the YAML file should be saved

**Returns:** `true` on success

**Example:**
```chariot
setq(yamlStr, 'server:\n  port: 8080\n  host: localhost\n')
saveYAMLRaw(yamlStr, 'config/app.yaml')
```

---

#### `loadYAMLMultiDoc(path)`

Loads a multi-document YAML file (documents separated by `---`) as an array of JSONNodes.

**Parameters:**
- `path`: String path to the multi-document YAML file

**Returns:** Array of JSONNodes, one per YAML document

**Example:**
```chariot
setq(docs, loadYAMLMultiDoc('config/multi.yaml'))
setq(firstDoc, at(docs, 0))
setq(secondDoc, at(docs, 1))
```

---

#### `saveYAMLMultiDoc(jsonNodeArray, path)`

Saves an array of JSONNodes as a multi-document YAML file.

**Parameters:**
- `jsonNodeArray`: Array of JSONNodes to save
- `path`: String path where the multi-document YAML file should be saved

**Returns:** `true` on success

**Example:**
```chariot
setq(doc1, create('config1', 'J'))
setValue(doc1, 'name', 'service1')

setq(doc2, create('config2', 'J'))
setValue(doc2, 'name', 'service2')

setq(docs, array(doc1, doc2))
saveYAMLMultiDoc(docs, 'config/services.yaml')
```

---

### Usage Patterns

#### Reading Configuration Files

```chariot
// Load YAML configuration
setq(config, loadYAML('config/database.yaml'))

// Access nested values
setq(host, valueOf(config, 'database.host'))
setq(port, valueOf(config, 'database.port'))
setq(username, valueOf(config, 'database.credentials.username'))
```

#### Writing Configuration Files

```chariot
// Create configuration
setq(config, create('dbConfig', 'J'))
setValue(config, 'database.host', 'localhost')
setValue(config, 'database.port', 5432)
setValue(config, 'database.credentials.username', 'admin')

// Save as YAML
saveYAML(config, 'config/database.yaml')
```

#### Template Processing

```chariot
// Load YAML template
setq(template, loadYAMLRaw('templates/deployment.yaml'))

// Replace placeholders
setq(yaml, replace(template, '${APP_NAME}', 'myapp'))
setq(yaml, replace(yaml, '${VERSION}', '1.0.0'))
setq(yaml, replace(yaml, '${REPLICAS}', '3'))

// Save processed YAML
saveYAMLRaw(yaml, 'deployments/myapp.yaml')
```

#### Multi-Document Processing

```chariot
// Load multiple Kubernetes manifests
setq(manifests, loadYAMLMultiDoc('k8s/all-resources.yaml'))

// Process each manifest
setq(i, 0)
while(lessThan(i, length(manifests)),
  setq(manifest, at(manifests, i)),
  setq(kind, valueOf(manifest, 'kind')),
  // Process based on kind...
  setq(i, add(i, 1))
)
```

---

### Notes

- YAML files are automatically parsed into JSONNode structures for easy manipulation
- Nested YAML structures are accessible using dot notation in `valueOf()` and `setValue()`
- Multi-document YAML files are commonly used in Kubernetes and other configuration systems
- Use `loadYAMLRaw()` / `saveYAMLRaw()` for template processing or when you need to preserve exact formatting
- YAML supports comments, which are preserved when using raw operations but lost when parsing to JSONNode
- File paths are resolved relative to the Chariot runtime's data directory

---

### See Also

- [JSONFunctions.md](JSONFunctions.md) - Similar operations for JSON files
- [FileFunctions.md](FileFunctions.md) - Generic file I/O operations
- [FormatConversionFunctions.md](FormatConversionFunctions.md) - Converting between YAML and JSON formats
