# Chariot Language Reference

## Tree Functions

Chariot tree functions provide creation, serialization, secure storage, metadata access, searching, and format conversion for tree nodes, which are the core structure for agent logic, hierarchical data, and configuration.

---

### Available Tree Functions
| Function                      | Description                                                      |
|-------------------------------|------------------------------------------------------------------|
| `newTree(name)`               | Create a new tree node with the given name                       |
| `treeSave(treeNode, filename [, format [, compression]])` | Save a tree node (recognized extension, otherwise JSON) |
| `treeLoad(filename)`          | Load a tree node from a file                                     |
| `treeToXML(treeNode [, prettyPrint])` | Convert a tree node to XML string (prettyPrint default: true) |
| `treeToYAML(treeNode)`        | Convert a tree node to YAML string                               |
| `treeSaveSecure(treeNode, filename, encryptionKeyID, signingKeyID, watermark [, options])` | Save tree node securely with encryption/signature    |
| `treeLoadSecure(filename, decryptionKeyID, verificationKeyID)` | Load a secure tree node with decryption and verification |
| `treeValidateSecure(filename, verificationKeyID)` | Validate the signature of a secure tree file          |
| `treeGetMetadata(filename)`   | Get metadata from a tree file without loading/decrypting         |
| `treeFind([forest,] attributeName, value [, operator])` | Find trees where any element matches the expression; supports implicit runtime search when forest omitted |
| `treeSearch(node, attributeName, value [, operator [, existsOnly]])` | Search nodes with attribute matching value and operator; optional existsOnly short-circuits to boolean |
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
- `format`: (Optional) `"json"`, `"yaml"`, `"xml"`, `"gob"`. Defaults to the recognized filename extension (`.yml` means YAML); otherwise JSON. An explicit format overrides this default.
- `compression`: (Optional) Boolean; if `true`, compresses the file (default: `false`).

```chariot
treeSave(agent, 'agent.json')
treeSave(agent, 'agent.gob') // writes stateful GOB, not JSON
treeSave(agent, 'agent.yaml', 'yaml')
treeSave(agent, 'agent.json.gz', 'json', true)
```

#### `treeLoad(filename)`

Loads a tree node from a file under `CHARIOT_TREE_PATH`. Format is selected from the filename extension; a final `.gz` is ignored for format selection and gzip compression is detected from the file bytes.

Historical `.gob` files containing JSON (created by the former JSON-only save default) are read through the normal JSON tree decoder. Stateful GOB, legacy direct TreeNode GOB, and legacy GOB byte-slice wrappers are also supported. Invalid JSON, truncated GOB, and unsupported binary representations remain errors; a GOB decoder's “non-zero delta for singleton” alone does not prove the file is corrupt.

Loading never rewrites or migrates the original file. To normalize an old artifact, load it and save to a **new filename**, specifying the intended format. Keep the original until the new file is verified. JSON and GOB persistence are not interchangeable for all runtime values: the stateful GOB value encoder does not preserve executable functions or the new class/instance values (it stringifies unsupported values). Use source declarations for classes with captured scopes; do not migrate such trees to GOB expecting executable state to survive.

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

#### `treeFind([forest,] attributeName, value [, operator])`

Finds and returns an array of trees (TreeNode/JSONNode) for which the expression holds true for at least one element somewhere inside the tree. If the first argument is omitted, the function searches across all tree-like values found in the runtime's local and global variables (implicit "forest"). Results are de-duplicated.

- `forest` (optional): A container of candidates (array, map) or a single tree; if omitted, all runtime trees are considered.
- `attributeName`: String name of the attribute to match.
- `value`: Value to compare against.
- `operator` (optional): Same operators supported as `treeSearch` (`"=", "!=", ">", ">=", "<", "<=", "contains", "startswith", "endswith"`). Defaults to `"="`.

Examples:

```chariot
// Implicit runtime search (no forest argument)
treeFind('status', 'active')
treeFind('price', 100, '>')

// Explicit forest
treeFind(agents, 'region', 'EMEA')
treeFind(fleet, 'odometer', 50000, '>=')

// Single tree argument still supported
treeFind(agent, 'name', 'Smith', 'contains')
```

#### `treeSearch(node, attributeName, value [, operator [, existsOnly]])`

Searches nodes recursively with attribute matching value using operator.
 - Supported operators: `"=", "!=", ">", ">=", "<", "<=", "contains", "startswith", "endswith"`
 - `existsOnly` (optional): When `true`, returns a boolean and short-circuits on first match; default `false` returns an array of matches.

```chariot
treeSearch(agent, 'score', 90, '>=')
treeSearch(agent, 'name', 'Smith', 'contains')
treeSearch(agent, 'status', 'active', '=', true)
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