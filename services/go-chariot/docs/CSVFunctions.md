# Chariot Language Reference

## CSV Functions

Chariot provides robust support for loading, reading, and manipulating CSV data. These functions allow you to work with CSV files and CSVNode objects to access tabular data in your Chariot programs.

---

### Available CSV Functions

| Function                              | Description                                                      |
|---------------------------------------|------------------------------------------------------------------|
| `csvHeaders(nodeOrPath)`              | Get the header row of a CSV file                                 |
| `csvRowCount(nodeOrPath)`             | Get the number of rows in a CSV file                             |
| `csvColumnCount(nodeOrPath)`          | Get the number of columns in a CSV file                          |
| `csvGetRow(nodeOrPath, index)`        | Get a specific row as a map (header → value)                     |
| `csvGetCell(nodeOrPath, row, col)`    | Get a specific cell value by row and column index or name        |
| `csvGetRows(nodeOrPath)`              | Get all rows as an array of arrays                               |
| `csvToCSV(nodeOrPath)`                | Convert a CSVNode to a CSV string                                |
| `csvLoad(node, path)`                 | Load a CSV file into an existing CSVNode                         |

---

### Function Details

#### `csvHeaders(nodeOrPath)`

Returns the header row of a CSV file as an array of strings.

**Parameters:**
- `nodeOrPath`: A CSVNode instance or a string path to a CSV file

**Returns:** Array of header column names

```chariot
setq(headers, csvHeaders('data/users.csv'))
// headers = ["id", "name", "email", "age"]
```

#### `csvRowCount(nodeOrPath)`

Returns the number of data rows in the CSV file (excluding the header row).

**Parameters:**
- `nodeOrPath`: A CSVNode instance or a string path to a CSV file

**Returns:** Number of rows

```chariot
setq(count, csvRowCount('data/users.csv'))
// count = 100
```

#### `csvColumnCount(nodeOrPath)`

Returns the number of columns in the CSV file.

**Parameters:**
- `nodeOrPath`: A CSVNode instance or a string path to a CSV file

**Returns:** Number of columns

```chariot
setq(cols, csvColumnCount('data/users.csv'))
// cols = 4
```

#### `csvGetRow(nodeOrPath, index)`

Returns a specific row as a map where keys are column headers and values are the cell values.

**Parameters:**
- `nodeOrPath`: A CSVNode instance or a string path to a CSV file
- `index`: Row index (0-based, excluding header)

**Returns:** Map of column name → cell value

```chariot
setq(row, csvGetRow('data/users.csv', 0))
// row = {"id": "1", "name": "Alice", "email": "alice@example.com", "age": "30"}
```

#### `csvGetCell(nodeOrPath, rowIndex, colIndexOrName)`

Returns the value of a specific cell.

**Parameters:**
- `nodeOrPath`: A CSVNode instance or a string path to a CSV file
- `rowIndex`: Row index (0-based, excluding header)
- `colIndexOrName`: Column index (number) or column name (string)

**Returns:** Cell value as a string

```chariot
setq(name, csvGetCell('data/users.csv', 0, 'name'))
// name = "Alice"

setq(email, csvGetCell('data/users.csv', 0, 2))
// email = "alice@example.com"
```

#### `csvGetRows(nodeOrPath)`

Returns all data rows as an array of arrays (each inner array represents a row).

**Parameters:**
- `nodeOrPath`: A CSVNode instance or a string path to a CSV file

**Returns:** Array of arrays containing all row data

**Note:** Use with caution on large CSV files as this loads all data into memory.

```chariot
setq(allRows, csvGetRows('data/users.csv'))
// allRows = [["1", "Alice", "alice@example.com", "30"], ["2", "Bob", "bob@example.com", "25"], ...]
```

#### `csvToCSV(nodeOrPath)`

Converts a CSVNode back to a CSV-formatted string.

**Parameters:**
- `nodeOrPath`: A CSVNode instance or a string path to a CSV file

**Returns:** CSV-formatted string

```chariot
setq(csvString, csvToCSV('data/users.csv'))
// csvString = "id,name,email,age\n1,Alice,alice@example.com,30\n..."
```

#### `csvLoad(node, path)`

Loads a CSV file into an existing CSVNode instance.

**Parameters:**
- `node`: A CSVNode instance
- `path`: String path to the CSV file to load

**Returns:** `true` on success

```chariot
setq(csvNode, create('myCSV', 'C'))
csvLoad(csvNode, 'data/users.csv')
setq(headers, csvHeaders(csvNode))
```

---

### Usage Patterns

#### Using File Paths (Convenience)

Most CSV functions accept either a CSVNode instance or a file path string. When you provide a path string, the function automatically creates a temporary CSVNode and loads the file:

```chariot
// Direct path usage
setq(count, csvRowCount('data/users.csv'))
setq(headers, csvHeaders('data/users.csv'))
```

#### Using CSVNode Instances (Recommended for Multiple Operations)

For better performance when performing multiple operations on the same CSV file, create a CSVNode once and reuse it:

```chariot
// Create a CSVNode
setq(csvNode, create('users', 'C'))
csvLoad(csvNode, 'data/users.csv')

// Reuse the node for multiple operations
setq(count, csvRowCount(csvNode))
setq(headers, csvHeaders(csvNode))
setq(firstRow, csvGetRow(csvNode, 0))
```

#### Iterating Over CSV Rows

```chariot
setq(csvNode, create('data', 'C'))
csvLoad(csvNode, 'data/users.csv')
setq(rowCount, csvRowCount(csvNode))

setq(i, 0)
while(lessThan(i, rowCount),
  setq(row, csvGetRow(csvNode, i)),
  setq(name, valueOf(row, 'name')),
  // Process row...
  setq(i, add(i, 1))
)
```

---

### Notes

- CSV files must have a header row (first row contains column names)
- All cell values are returned as strings; use type conversion functions as needed
- Row indices are 0-based and exclude the header row
- Column indices can be specified as numbers (0-based) or column names (strings)
- Use `csvGetRows()` with caution on large files as it loads all data into memory
- File paths are resolved relative to the Chariot runtime's data directory

---
