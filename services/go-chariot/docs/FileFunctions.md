# Chariot Language Reference

## File Functions

Chariot provides generic file I/O functions for working with files in a format-agnostic way. These functions handle basic file operations like reading, writing, checking existence, and listing directory contents.

For format-specific operations (JSON, CSV, YAML, XML), see the respective documentation linked below.

---

### Available File Functions

| Function                  | Description                                         |
|---------------------------|-----------------------------------------------------|
| `readFile(path)`          | Read the contents of a file as a string             |
| `writeFile(path, content)`| Write a string to a file                            |
| `fileExists(path)`        | Returns `true` if the file exists                   |
| `getFileSize(path)`       | Returns the file size in bytes                      |
| `deleteFile(path)`        | Delete a file                                       |
| `listFiles(dir)`          | List file names in a directory (returns array)      |

---

### Function Details

#### `readFile(path)`

Reads the entire contents of a file as a string.

**Parameters:**
- `path`: String path to the file to read

**Returns:** String containing the file contents

**Example:**
```chariot
setq(content, readFile('data/document.txt'))
setq(lines, split(content, '\n'))
```

---

#### `writeFile(path, content)`

Writes a string to a file, creating the file if it doesn't exist or overwriting it if it does.

**Parameters:**
- `path`: String path to the file to write
- `content`: String content to write to the file

**Returns:** `true` on success

**Example:**
```chariot
setq(report, 'Report Date: 2024-01-15\nStatus: Complete\n')
writeFile('reports/daily.txt', report)
```

---

#### `fileExists(path)`

Checks whether a file exists at the specified path.

**Parameters:**
- `path`: String path to check

**Returns:** `true` if the file exists, `false` otherwise

**Example:**
```chariot
if(fileExists('config/app.conf'),
  setq(config, readFile('config/app.conf')),
  setq(config, 'default configuration')
)
```

---

#### `getFileSize(path)`

Returns the size of a file in bytes.

**Parameters:**
- `path`: String path to the file

**Returns:** Number representing the file size in bytes

**Example:**
```chariot
setq(size, getFileSize('data/large-file.bin'))
if(greaterThan(size, 1048576),
  print('File is larger than 1MB')
)
```

---

#### `deleteFile(path)`

Deletes a file from the filesystem.

**Parameters:**
- `path`: String path to the file to delete

**Returns:** `true` on success

**Example:**
```chariot
if(fileExists('temp/cache.tmp'),
  deleteFile('temp/cache.tmp')
)
```

---

#### `listFiles(dir)`

Lists all files in a directory.

**Parameters:**
- `dir`: String path to the directory

**Returns:** Array of filenames (strings) in the directory

**Example:**
```chariot
setq(files, listFiles('data'))
setq(i, 0)
while(lessThan(i, length(files)),
  setq(file, at(files, i)),
  print(file),
  setq(i, add(i, 1))
)
```

---

### Usage Patterns

#### Reading and Processing Text Files

```chariot
// Read file
setq(content, readFile('data/log.txt'))

// Process line by line
setq(lines, split(content, '\n'))
setq(errorCount, 0)

setq(i, 0)
while(lessThan(i, length(lines)),
  setq(line, at(lines, i)),
  if(contains(line, 'ERROR'),
    setq(errorCount, add(errorCount, 1))
  ),
  setq(i, add(i, 1))
)

print(concat('Found ', toString(errorCount), ' errors'))
```

#### Batch File Processing

```chariot
// List all files in directory
setq(files, listFiles('input'))

// Process each file
setq(i, 0)
while(lessThan(i, length(files)),
  setq(filename, at(files, i)),
  setq(inputPath, concat('input/', filename)),
  setq(outputPath, concat('output/', filename)),
  
  // Read, process, and write
  setq(content, readFile(inputPath)),
  setq(processed, toUpperCase(content)),
  writeFile(outputPath, processed),
  
  setq(i, add(i, 1))
)
```

#### Conditional File Operations

```chariot
// Check if file exists before reading
if(fileExists('config/custom.conf'),
  setq(config, readFile('config/custom.conf')),
  setq(config, readFile('config/default.conf'))
)

// Check file size before processing
setq(path, 'data/large-dataset.txt')
if(fileExists(path),
  setq(size, getFileSize(path)),
  if(lessThan(size, 10485760),  // 10MB
    setq(data, readFile(path)),
    print('File too large to process')
  ),
  print('File not found')
)
```

#### Cleanup Operations

```chariot
// List and delete old temporary files
setq(tempFiles, listFiles('temp'))

setq(i, 0)
while(lessThan(i, length(tempFiles)),
  setq(file, at(tempFiles, i)),
  if(endsWith(file, '.tmp'),
    deleteFile(concat('temp/', file))
  ),
  setq(i, add(i, 1))
)
```

---

### Notes

- All file paths are resolved relative to the Chariot runtime's data directory
- `readFile()` and `writeFile()` work with text files; use format-specific functions for structured data
- `writeFile()` creates parent directories automatically if they don't exist
- `listFiles()` returns only filenames, not full paths - prepend the directory path when accessing files
- File operations use secure path validation to prevent directory traversal attacks
- Large files should be processed in chunks or using streaming approaches when possible

---

### Format-Specific File Operations

For working with structured data formats, see these specialized function sets:

- **[JSONFunctions.md](JSONFunctions.md)** - JSON parsing, manipulation, and file I/O (`parseJSON`, `toJSON`, `loadJSON`, `saveJSON`)
- **[CSVFunctions.md](CSVFunctions.md)** - CSV file operations and data access (`csvHeaders`, `csvRowCount`, `csvGetRow`, `loadCSV`, `saveCSV`)
- **[YAMLFunctions.md](YAMLFunctions.md)** - YAML file operations (`loadYAML`, `saveYAML`, `loadYAMLMultiDoc`)
- **[XMLFunctions.md](XMLFunctions.md)** - XML file operations and parsing (`loadXML`, `saveXML`, `parseXMLString`)
- **[FormatConversionFunctions.md](FormatConversionFunctions.md)** - Converting between formats (`jsonToYAML`, `yamlToJSON`, `convertJSONFileToYAML`)

---

### See Also

- [StringFunctions.md](StringFunctions.md) - String manipulation for processing file contents
- [ArrayFunctions.md](ArrayFunctions.md) - Array operations for working with file lists
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