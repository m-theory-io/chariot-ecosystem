# Chariot Language Reference

## Value Functions

Chariot value functions provide variable declaration, assignment, type conversion, existence checks, and utility operations for working with values of any type. These functions are foundational for scripting, type safety, and dynamic programming.

For current receiver methods and expression calls, see
[Member notation](MemberNotation.md) and [Classes and interfaces](ClassesAndInterfaces.md).
The latter is canonical for field/signature contracts. `F` accepts ordinary functions and
runtime `Callable` values (including extracted bound methods); `O` accepts script
instances and host wrappers. This also applies to `declare`/`declareGlobal`.
Use identifier declaration targets, e.g. `declare(f, "F", person.hello)` and
`declare(o, "O", person)`, not quoted names.

---

### Supported Value Types

Chariot uses single-character type codes to specify the type for `declare()` and `declareGlobal()`:

| Type Code | Type Name     | Description                                      |
|-----------|---------------|--------------------------------------------------|
| `"N"`     | Number        | Floating-point numeric values                    |
| `"S"`     | String        | Text strings                                     |
| `"L"`     | Boolean       | Logical values (`true`/`false`)                  |
| `"D"`     | Date          | Date-string contract                             |
| `"A"`     | Array         | Ordered collections of values                    |
| `"M"`     | Map           | Key-value pairs                                  |
| `"O"`     | Object        | Script instances or host object wrappers         |
| `"T"`     | TreeNode      | Hierarchical tree structures                     |
| `"F"`     | Function      | Function values and runtime-aware callables      |
| `"J"`     | JSON          | JSON document nodes                              |
| `"X"`     | XML           | XML document nodes                               |
| `"P"`     | Plan          | Agent plan values                                |
| `"E"`     | ETL transform | ETL transform values                             |
| `"V"`     | Variable      | Untyped variables (default from `setq`)          |

Runtime classification can also report `R` (table) and `H` (host object), but
these are not accepted `declare` contracts. Supply initial callable/object values
for `F`/`O`; do not assume every type has a usable default initializer.

---

### Variable Declaration and Assignment

| Function                | Description                                                      |
|-------------------------|------------------------------------------------------------------|
| `declare(name, type [, initialValue])` | Declare a variable in the current scope with type and optional initial value |
| `declareGlobal(name, type, initialValue)` | Declare a global variable with type and initial value      |
| `setq(name, value [, namespace, key])` | Assign a value to a variable, or set a value in a namespace (array, table, object, xml) |
| `destroy(name)`         | Remove a variable from the current or global scope               |

---

### Function Definition and Invocation

| Function                | Description                                                      |
|-------------------------|------------------------------------------------------------------|
| `function([params], body)` | Define a function with parameters and body (as code string)   |
| `func(param, ...) { body }` | Function literal with identifier parameters and a block body |
| `call(fn, [args...])`   | Call a function value with arguments                             |

Current block syntax is `func(x) { x }`. Methods require an explicit first
parameter named `self`; ordinary function-valued fields do not bind receivers.
`signature(array("N"), "N")` plus `withSignature(fn, contract)` opts into exact
arity/parameter/return checks. Unannotated functions remain permissive.
See [Function contracts](ClassesAndInterfaces.md#function-contracts) and
[object examples](../examples/objects/README.md).

---

### Function Management

| Function                | Description                                                      |
|-------------------------|------------------------------------------------------------------|
| `registerFunction(name, fn)` | Register a function value with the runtime                   |
| `deleteFunction(name)`  | Remove a function from the runtime functions map                 |
| `listFunctions()`       | Returns a list of all registered functions                       |
| `getFunction(name)`     | Get the pretty-printed source code of a function                 |
| `saveFunctions(filename)` | Save all registered functions to a JSON file                   |
| `loadFunctions(filename)` | Load functions from a JSON file                                |

---

### Runtime Inspection

| Function                | Description                                                      |
|-------------------------|------------------------------------------------------------------|
| `inspectRuntime()`      | Returns a comprehensive JSON object of runtime state             |
| `getVariable(name)`     | Get a variable value from the current scope                      |

---

### Existence and Type Checking

| Function                | Description                                                      |
|-------------------------|------------------------------------------------------------------|
| `exists(name)`          | Returns `true` if a variable or object exists                    |
| `typeOf(value)`         | Returns the type code of a value as a string                     |
| `valueOf(value [, type])` | Converts a value to the specified type (`"N"`, `"S"`, `"L"`)   |
| `boolean(value)`        | Converts a value to boolean (`true`/`false`)                     |
| `isNull(value)`         | Returns `true` if the value is `DBNull` (null)                   |
| `isNumeric(str)`        | Returns `true` if the string is numeric                          |
| `empty(value)`          | Returns `true` if the value is empty (zero, empty string, or null)|

---

### Map and Array Utilities

| Function                | Description                                                      |
|-------------------------|------------------------------------------------------------------|
| `mapValue([key1, val1, ...])` | Create a map from key-value pairs                         |
| `map([key1, val1, ...])` | Also creates a MapValue; registered with ETL functions |
| `toMapValue(obj)`       | Convert a MapNode or JSONNode to a MapValue                     |
| `setValue(array, index, value)` | Set value at index in an array, expanding as needed      |

Both constructors currently work for named class attributes:
`new(Person, map("name", "Ada"))` and `new(Person, mapValue("name", "Ada"))`.
Neither is an ETL-only mapping operator. A restricted runtime must register the
constructor it uses. Missing `MapValue` members return null; unknown class
members error. See [Member notation](MemberNotation.md#supported-receivers).

---

### Template and Merge Functions

| Function                | Description                                                      |
|-------------------------|------------------------------------------------------------------|
| `merge(template, node, params)` | Merge template string with node values using parameters     |
| `offerVariable(value, format)` | Create an OfferVariable with formatting specification       |
| `offerVar(value, format)` | Alias for `offerVariable`                                       |

---

### Metadata Functions

| Function                | Description                                                      |
|-------------------------|------------------------------------------------------------------|
| `hasMeta(doc, metaKey)` | Returns `true` if the document or node has the given metadata key|

---

### Type Conversion Functions

| Function                | Description                                                      |
|-------------------------|------------------------------------------------------------------|
| `toString(value)`       | Convert a value to string                                        |
| `toNumber(value)`       | Convert a value to number                                        |
| `toBool(value)`         | Convert a value to boolean                                       |

---

### Function Details

#### Variable Declaration and Assignment

```chariot
declare(x, "N", 42)                  // Declare variable x as number with value 42
declare(name, "S")                   // Declare variable name as string (default "")
declare(items, "A")                  // Declare array variable items
declare(config, "M")                 // Declare map variable config
declare(isActive, "L", true)          // Declare boolean variable isActive
declare(data, "J")                   // Declare JSON variable data
declare(doc, "X")                    // Declare XML variable doc
declare(root, "T")                   // Declare TreeNode variable root
declare(callback, "F", func(x) { x }) // Declare callable with an explicit value
declareGlobal(g, "L", true)           // Declare global boolean variable g

setq(x, 100)                         // Set variable x to 100
setq(arr, 5, "array", "2")           // Set arr[2] = 5
setq(tbl, "val", "table", "0:col")   // Set table cell tbl[0]["col"] = "val"
destroy("x")                         // Remove variable x
```

#### Function Definition and Invocation

```chariot
setq(f, func(a, b) { add(a, b) })         // Define function f(a, b) = add(a, b)
call(f, 2, 3)                              // Returns 5

// Register a function for reuse
registerFunction("myFunc", f)
getFunction("myFunc")                       // Returns pretty-printed source
```

#### Function Management

```chariot
listFunctions()                    // Returns array of function names
saveFunctions("mylib.json")        // Save all functions to file
loadFunctions("mylib.json")        // Load functions from file
deleteFunction("oldFunc")          // Remove function from runtime
```

#### Runtime Inspection

```chariot
inspectRuntime()                   // Returns complete runtime state as JSON
getVariable("myVar")               // Get variable value
```

#### Existence and Type Checking

```chariot
exists("x")                // true if variable x exists
typeOf(123)                // "N" (number)
valueOf("42", "N")         // 42 (number)
valueOf("true", "L")       // true (boolean)
valueOf(42, "S")           // "42" (string)
boolean("yes")             // true
isNull(DBNull)             // true
isNumeric("12345")         // true
empty("")                  // true
empty(0)                   // true
```

#### Map and Array Utilities

```chariot
mapValue("a", 1, "b", 2)   // MapValue with keys "a":1, "b":2
toMapValue(jsonNode)       // Convert JSONNode to MapValue
setValue(arr, 3, "x")      // arr[3] = "x", expanding arr if needed
```

#### Template and Merge Functions

```chariot
// Create template with placeholders
setq(template, "Hello {name}, you have {count} items")
setq(node, mapValue("name", "John", "count", 5))
merge(template, node, params)  // Returns "Hello John, you have 5 items"

// Use formatted variables
offerVariable(1234.56, "currency")  // Formats as "$1234.56"
offerVariable(0.85, "percentage")   // Formats as "85.00%"
```

#### Metadata Functions

```chariot
hasMeta(doc, "createdBy")  // true if doc has "createdBy" metadata
```

#### Type Conversion Functions

```chariot
toString(123)              // "123"
toNumber("42")             // 42
toBool("yes")              // true
```

---

### Notes

- Declaration contracts are the single-character codes in the supported table;
	runtime classifications `R`/`H` are not declaration contracts.
- `"L"` for boolean follows dBASE convention (Logical).
- `"V"` is used for untyped variables created with `setq` without explicit type declaration.
- `declare` and `declareGlobal` validate compatibility; they do not implicitly
	convert an arbitrary value into a script instance or callable.
- `setq` can assign to variables, arrays, tables, objects, or XML nodes using the namespace and key arguments.
- `function` accepts a parameter array and a code string; the canonical literal
	form is `func(a, b) { add(a, b) }`.
- `call` executes a function or runtime-aware callable (including bound methods).
- `mapValue` requires an even number of arguments (key-value pairs).
- `empty` returns `true` for zero, empty string, or null.
- `hasMeta` checks for metadata keys on nodes or documents.
- `inspectRuntime()` returns a comprehensive JSON object containing globals, variables, objects, lists, namespaces, nodes, tables, keycolumns, default_template, and timeoffset.
- `merge()` supports formatted variables with tags like "currency", "percentage", "int", "float", etc.
- Function persistence allows saving/loading function libraries as JSON files.
- `valueOf()` function uses the single-character type codes for conversion.
- `isNumeric(str)` returns true if the string is numeric.

---
