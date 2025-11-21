# Chariot Language Reference

## Format Conversion Functions

Chariot provides utilities for converting data between different formats (JSON, YAML, etc.). These functions enable seamless transformation of configuration files and data structures between formats.

---

### Available Format Conversion Functions

| Function                              | Description                                                      |
|---------------------------------------|------------------------------------------------------------------|
| `jsonToYAML(jsonStr)`                 | Convert a JSON string to a YAML string                           |
| `yamlToJSON(yamlStr)`                 | Convert a YAML string to a JSON string                           |
| `jsonToYAMLNode(jsonNode)`            | Convert a JSONNode to a YAML string                              |
| `yamlToJSONNode(yamlStr)`             | Convert a YAML string to a JSONNode                              |
| `convertJSONFileToYAML(jsonPath, yamlPath)` | Convert a JSON file to YAML format                         |
| `convertYAMLFileToJSON(yamlPath, jsonPath)` | Convert a YAML file to JSON format                         |

---

### Function Details

#### `jsonToYAML(jsonStr)`

Converts a JSON string to a YAML string.

**Parameters:**
- `jsonStr`: String containing JSON content

**Returns:** String containing equivalent YAML content

**Example:**
```chariot
setq(json, '{"name":"app","version":"1.0","port":8080}')
setq(yaml, jsonToYAML(json))
// Result: "name: app\nversion: '1.0'\nport: 8080\n"
```

---

#### `yamlToJSON(yamlStr)`

Converts a YAML string to a JSON string.

**Parameters:**
- `yamlStr`: String containing YAML content

**Returns:** String containing equivalent JSON content

**Example:**
```chariot
setq(yaml, 'name: app\nversion: 1.0\nport: 8080\n')
setq(json, yamlToJSON(yaml))
// Result: '{"name":"app","version":1.0,"port":8080}'
```

---

#### `jsonToYAMLNode(jsonNode)`

Converts a JSONNode to a YAML string.

**Parameters:**
- `jsonNode`: A JSONNode to convert

**Returns:** String containing YAML representation

**Example:**
```chariot
setq(config, create('config', 'J'))
setValue(config, 'name', 'myapp')
setValue(config, 'port', 8080)
setq(yaml, jsonToYAMLNode(config))
saveYAMLRaw(yaml, 'config/app.yaml')
```

---

#### `yamlToJSONNode(yamlStr)`

Converts a YAML string to a JSONNode.

**Parameters:**
- `yamlStr`: String containing YAML content

**Returns:** JSONNode representing the YAML structure

**Example:**
```chariot
setq(yaml, 'name: myapp\nport: 8080\n')
setq(config, yamlToJSONNode(yaml))
setq(port, valueOf(config, 'port'))
```

---

#### `convertJSONFileToYAML(jsonPath, yamlPath)`

Converts a JSON file to YAML format.

**Parameters:**
- `jsonPath`: String path to the source JSON file
- `yamlPath`: String path where the YAML file should be saved

**Returns:** `true` on success

**Example:**
```chariot
convertJSONFileToYAML('config/app.json', 'config/app.yaml')
```

---

#### `convertYAMLFileToJSON(yamlPath, jsonPath)`

Converts a YAML file to JSON format.

**Parameters:**
- `yamlPath`: String path to the source YAML file
- `jsonPath`: String path where the JSON file should be saved

**Returns:** `true` on success

**Example:**
```chariot
convertYAMLFileToJSON('config/app.yaml', 'config/app.json')
```

---

### Usage Patterns

#### Converting Configuration Formats

```chariot
// Convert existing JSON config to YAML
convertJSONFileToYAML('config/settings.json', 'config/settings.yaml')

// Or vice versa
convertYAMLFileToJSON('config/deployment.yaml', 'config/deployment.json')
```

#### Working with API Responses

```chariot
// Receive JSON from API
setq(jsonResponse, '{"status":"ok","data":{"count":42}}')

// Convert to YAML for readability
setq(yamlOutput, jsonToYAML(jsonResponse))
writeFile('logs/response.yaml', yamlOutput)
```

#### Building Configuration Files

```chariot
// Build config as JSONNode
setq(config, create('appConfig', 'J'))
setValue(config, 'server.host', 'localhost')
setValue(config, 'server.port', 8080)
setValue(config, 'database.host', 'db.example.com')

// Export as YAML for deployment
setq(yaml, jsonToYAMLNode(config))
saveYAMLRaw(yaml, 'deploy/config.yaml')

// Also export as JSON for backup
saveJSON(config, 'backup/config.json')
```

#### Batch Conversion

```chariot
// Convert all JSON configs to YAML
setq(files, listFiles('config'))
setq(i, 0)
while(lessThan(i, length(files)),
  setq(file, at(files, i)),
  if(endsWith(file, '.json'),
    setq(base, replace(file, '.json', '')),
    setq(srcPath, concat('config/', file)),
    setq(dstPath, concat('config/', base, '.yaml')),
    convertJSONFileToYAML(srcPath, dstPath)
  ),
  setq(i, add(i, 1))
)
```

#### Processing External Data

```chariot
// Load YAML from external source
setq(yamlData, readFile('external/data.yaml'))

// Convert to JSONNode for processing
setq(data, yamlToJSONNode(yamlData))

// Process data...
setq(processed, valueOf(data, 'results'))

// Export as JSON for downstream systems
setq(json, toJSON(processed))
writeFile('output/processed.json', json)
```

---

### Notes

- Conversions preserve data structure but may not preserve formatting or comments
- YAML supports comments; JSON does not - comments will be lost in YAML → JSON conversion
- Both formats support nested structures, arrays, and basic data types
- YAML can be more human-readable for configuration files
- JSON is more widely supported by APIs and programming languages
- File conversion functions handle file I/O automatically
- String conversion functions operate on in-memory data
- All file paths are resolved relative to the Chariot runtime's data directory

---

### See Also

- [JSONFunctions.md](JSONFunctions.md) - JSON parsing and manipulation
- [YAMLFunctions.md](YAMLFunctions.md) - YAML file operations
- [FileFunctions.md](FileFunctions.md) - Generic file I/O operations
