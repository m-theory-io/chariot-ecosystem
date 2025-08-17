# Chariot Language Reference

## Tree Functions

Chariot tree functions provide creation, serialization, secure storage, metadata access, searching, and format conversion for tree nodes, which are the core structure for agent logic, hierarchical data, and configuration.

---

### Available Tree Functions

| Function                      | Description                                                      |
|-------------------------------|------------------------------------------------------------------|
| `newTree(name)`               | Create a new tree node with the given name                       |
| `treeSave(treeNode, filename [, format [, compression]])` | Save a tree node to a file (default: JSON)           |
| `treeLoad(filename)`          | Load a tree node from a file                                     |
| `treeToXML(treeNode [, prettyPrint])` | Convert a tree node to XML string (prettyPrint default: true) |
| `treeToYAML(treeNode)`        | Convert a tree node to YAML string                               |
| `treeSaveSecure(treeNode, filename, encryptionKeyID, signingKeyID, watermark [, options])` | Save tree node securely with encryption/signature    |
| `treeLoadSecure(filename, decryptionKeyID, verificationKeyID)` | Load a secure tree node with decryption and verification |
| `treeValidateSecure(filename, verificationKeyID)` | Validate the signature of a secure tree file          |
| `treeGetMetadata(filename)`   | Get metadata from a tree file without loading/decrypting         |
| `treeFind(node, attributeName, value)` | Find all nodes with attribute equal to value           |
| `treeSearch(node, attributeName, value [, operator])` | Search nodes with attribute matching value and operator |
| `treeWalk(node, fn)`          | Recursively apply a function to all nodes and values             |

---

### Function Details

#### `newTree(name)`

Creates a new tree node with the specified name.

```chariot
setq(agent, newTree('MyAgent'))
```

#### `treeSave(treeNode, filename [, format [, compression]])`

Saves a tree node to a file.
- `treeNode`: The tree node to save.
- `filename`: The file path.
- `format`: (Optional) `"json"`, `"yaml"`, `"xml"`, `"gob"` (default: `"json"`).
- `compression`: (Optional) Boolean; if `true`, compresses the file (default: `false`).

```chariot
treeSave(agent, 'agent.json')
treeSave(agent, 'agent.yaml', 'yaml')
treeSave(agent, 'agent.json.gz', 'json', true)
```

#### `treeLoad(filename)`

Loads a tree node from a file. Format is auto-detected from file extension.

```chariot
setq(agent, treeLoad('agent.json'))
```

#### `treeToXML(treeNode [, prettyPrint])`

Converts a tree node to an XML string.
- `prettyPrint`: (Optional) Boolean; if `true`, output is pretty-printed (default: `true`).

```chariot
treeToXML(agent)                // Pretty-printed XML
treeToXML(agent, false)         // Compact XML
```

#### `treeToYAML(treeNode)`

Converts a tree node to a YAML string.

```chariot
treeToYAML(agent)
```

#### `treeSaveSecure(treeNode, filename, encryptionKeyID, signingKeyID, watermark [, options])`

Saves a tree node securely with encryption, signing, watermark, and optional options.
- `options` map keys: `verificationKeyID`, `checksum`, `auditTrail`, `compressionLevel`

```chariot
treeSaveSecure(agent, 'secure.json', 'encKey', 'signKey', 'watermark', map('checksum', true, 'compressionLevel', 9))
```

#### `treeLoadSecure(filename, decryptionKeyID, verificationKeyID)`

Loads a secure tree node, decrypting and verifying signature.

```chariot
setq(agent, treeLoadSecure('secure.json', 'decKey', 'verifyKey'))
```

#### `treeValidateSecure(filename, verificationKeyID)`

Validates the signature of a secure tree file.

```chariot
treeValidateSecure('secure.json', 'verifyKey') // true or false
```

#### `treeGetMetadata(filename)`

Gets metadata from a tree file without loading or decrypting the full tree.

```chariot
treeGetMetadata('agent.json') // returns map of metadata
```

#### `treeFind(node, attributeName, value)`

Finds all nodes (recursively) with the given attribute equal to the value.

```chariot
treeFind(agent, 'status', 'active')
```

#### `treeSearch(node, attributeName, value [, operator])`

Searches nodes recursively with attribute matching value using operator.
- Supported operators: `"="`, `"!="`, `">"`, `">="`, `"<"`, `"<="`, `"contains"`, `"startswith"`, `"endswith"`

```chariot
treeSearch(agent, 'score', 90, '>=')
treeSearch(agent, 'name', 'Smith', 'contains')
```

#### `treeWalk(node, fn)`

Recursively applies a function to all nodes and values in the tree.

```chariot
treeWalk(agent, myFunc)
```

---

### Notes

- Tree nodes are hierarchical structures used for agent logic, configuration, and data.
- `treeSave` and `treeLoad` support JSON, YAML, XML, and GOB formats (auto-detected by extension).
- Secure tree functions use encryption, signing, and watermarking for sensitive data.
- Metadata can be accessed without loading the full tree.
- Search and find functions traverse all nested nodes, arrays, and maps.
- All file paths and key IDs must be strings.
- All arguments are automatically unwrapped from `ScopeEntry` if needed.

---