# Chariot Language Reference

## File Functions

Chariot provides a comprehensive set of file I/O functions for working with plain text, JSON, CSV, YAML, and XML files, as well as format conversion utilities. These functions support reading, writing, deleting, listing, and converting files in various formats.

---

### Generic File Operations

| Function           | Description                                         |
|--------------------|-----------------------------------------------------|
| `readFile(path)`   | Read the contents of a file as a string             |
| `writeFile(path, content)` | Write a string to a file                    |
| `fileExists(path)` | Returns `true` if the file exists                   |
| `getFileSize(path)`| Returns the file size in bytes                      |
| `deleteFile(path)` | Delete a file                                       |
| `listFiles(dir)`   | List file names in a directory (returns array)      |

---

### JSON File Operations

| Function           | Description                                         |
|--------------------|-----------------------------------------------------|
| `loadJSON(path)`   | Load a JSON file and return as a JSONNode           |
| `saveJSON(obj, path [, indent])` | Save an object as pretty JSON to a file; `indent` is optional (default 2 spaces) |
| `loadJSONRaw(path)`| Load a JSON file as a raw JSON string               |
| `saveJSONRaw(jsonStr, path)` | Save a raw JSON string to a file           |

---

### CSV File Operations

| Function           | Description                                         |
|--------------------|-----------------------------------------------------|
| `loadCSV(path [, hasHeaders])` | Load a CSV file as a JSONNode; `hasHeaders` is optional (default `true`) |
| `saveCSV(jsonNode, path [, includeHeaders])` | Save a JSONNode as a CSV file; `includeHeaders` is optional (default `true`) |
| `loadCSVRaw(path)` | Load a CSV file as a raw string                     |
| `saveCSVRaw(csvStr, path)` | Save a raw CSV string to a file              |

---

### YAML File Operations

| Function           | Description                                         |
|--------------------|-----------------------------------------------------|
| `loadYAML(path)`   | Load a YAML file and return as a JSONNode           |
| `saveYAML(jsonNode, path)` | Save a JSONNode as a YAML file              |
| `loadYAMLRaw(path)`| Load a YAML file as a raw string                    |
| `saveYAMLRaw(yamlStr, path)` | Save a raw YAML string to a file           |
| `loadYAMLMultiDoc(path)` | Load a YAML file with multiple documents as a JSONNode (array of docs) |
| `saveYAMLMultiDoc(jsonNode, path)` | Save an array of docs as a multi-document YAML file |

---

### XML File Operations

| Function           | Description                                         |
|--------------------|-----------------------------------------------------|
| `loadXML(path)`    | Load an XML file and return as a JSONNode           |
| `saveXML(jsonNode, path [, rootElementName])` | Save a JSONNode as an XML file; `rootElementName` is optional (default `"root"`) |
| `loadXMLRaw(path)` | Load an XML file as a raw string                    |
| `saveXMLRaw(xmlStr, path)` | Save a raw XML string to a file              |
| `parseXMLString(xmlStr)` | Parse an XML string and return as a JSONNode   |

---

### Format Conversion Functions

| Function                   | Description                                  |
|----------------------------|----------------------------------------------|
| `jsonToYAML(jsonNode)`     | Convert a JSONNode to a YAML string          |
| `yamlToJSON(yamlStr)`      | Convert a YAML string to a JSONNode          |
| `jsonToYAMLNode(jsonNode)` | Convert a JSONNode to a YAML-compatible node |
| `yamlToJSONNode(yamlNode)` | Convert a YAML node to a JSONNode            |
| `convertJSONFileToYAML(jsonPath, yamlPath)` | Convert a JSON file to a YAML file |
| `convertYAMLFileToJSON(yamlPath, jsonPath)` | Convert a YAML file to a JSON file |

---

### Function Details

#### Generic File Operations

```chariot
readFile('notes.txt')                  // Returns file contents as string
writeFile('output.txt', 'Hello!')      // Writes string to file
fileExists('data.csv')                 // true if file exists
getFileSize('data.csv')                // Returns file size in bytes
deleteFile('old.txt')                  // Deletes the file
listFiles('/tmp')                      // Returns array of file names in directory
```

#### JSON File Operations

```chariot
loadJSON('data.json')                  // Loads JSON as JSONNode
saveJSON(obj, 'out.json', 4)           // Saves object as pretty JSON (4 spaces indent)
loadJSONRaw('data.json')               // Loads JSON as raw string
saveJSONRaw(jsonStr, 'out.json')       // Saves raw JSON string to file
```

#### CSV File Operations

```chariot
loadCSV('data.csv', true)              // Loads CSV as JSONNode (with headers)
saveCSV(jsonNode, 'out.csv', true)     // Saves JSONNode as CSV (with headers)
loadCSVRaw('data.csv')                 // Loads CSV as raw string
saveCSVRaw(csvStr, 'out.csv')          // Saves raw CSV string to file
```

#### YAML File Operations

```chariot
loadYAML('data.yaml')                  // Loads YAML as JSONNode
saveYAML(jsonNode, 'out.yaml')         // Saves JSONNode as YAML
loadYAMLRaw('data.yaml')               // Loads YAML as raw string
saveYAMLRaw(yamlStr, 'out.yaml')       // Saves raw YAML string to file
loadYAMLMultiDoc('multi.yaml')         // Loads multi-document YAML as array node
saveYAMLMultiDoc(jsonNode, 'multi.yaml') // Saves array node as multi-document YAML
```

#### XML File Operations

```chariot
loadXML('data.xml')                    // Loads XML as JSONNode
saveXML(jsonNode, 'out.xml', 'root')   // Saves JSONNode as XML with root element
loadXMLRaw('data.xml')                 // Loads XML as raw string
saveXMLRaw(xmlStr, 'out.xml')          // Saves raw XML string to file
parseXMLString(xmlStr)                 // Parses XML string as JSONNode
```

#### Format Conversion Functions

```chariot
jsonToYAML(jsonNode)                   // Converts JSONNode to YAML string
yamlToJSON(yamlStr)                    // Converts YAML string to JSONNode
jsonToYAMLNode(jsonNode)               // Converts JSONNode to YAML-compatible node
yamlToJSONNode(yamlNode)               // Converts YAML node to JSONNode
convertJSONFileToYAML('in.json', 'out.yaml') // Converts JSON file to YAML file
convertYAMLFileToJSON('in.yaml', 'out.json') // Converts YAML file to JSON file
```

---

### Notes

- JSON, YAML, and XML file operations use Chariot's `JSONNode` for structured data.
- CSV loading returns a `JSONNode` with an array of objects (if headers) or arrays (if no headers).
- All file paths must be strings.
- Raw load/save functions operate on unparsed strings.
- Format conversion functions allow seamless transformation between JSON, YAML, and XML.

---