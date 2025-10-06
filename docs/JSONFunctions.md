# Chariot Language Reference

## JSON Functions

Chariot provides robust support for parsing, generating, and manipulating JSON data. These functions allow you to convert between Chariot values and JSON, work with JSON nodes, and perform lightweight or strict parsing as needed.

---

### Available JSON Functions

| Function                | Description                                                      |
|-------------------------|------------------------------------------------------------------|
| `parseJSON(str)`        | Parse a JSON string into a JSONNode                              |
| `parseJSONValue(str)`   | Parse a JSON string into a Chariot value (MapValue, ArrayValue, etc.) |
| `parseJSONSimple(str)`  | Parse a JSON string into a SimpleJSON node                       |
| `toJSON(value)`         | Convert a Chariot value or JSONNode to a JSON string             |
| `toSimpleJSON(value)`   | Convert a Chariot value to a SimpleJSON node                     |

---

### Function Details

#### `parseJSON(str)`

Parses a JSON string and returns a `JSONNode` object.  
- The node's properties are populated from the JSON object.
- Returns an error if the string is not valid JSON.

```chariot
setq(j, parseJSON('{"a": 1, "b": 2}'))
```

#### `parseJSONValue(str)`

Parses a JSON string and returns a Chariot value:
- Returns a `MapValue` for JSON objects,
- Returns an `ArrayValue` for JSON arrays,
- Returns a primitive value for numbers, strings, booleans, or null.

```chariot
setq(val, parseJSONValue('{"x": 10, "y": 20}')) // MapValue
setq(arr, parseJSONValue('[1,2,3]'))            // ArrayValue
setq(num, parseJSONValue('42'))                 // Number
```

#### `parseJSONSimple(str)`

Parses a JSON string and returns a `SimpleJSON` node.  
- Useful for lightweight or high-performance scenarios.
- Returns an error if the string is empty or invalid.

```chariot
setq(sj, parseJSONSimple('{"foo": "bar"}'))
```

#### `toJSON(value)`

Converts a Chariot value or `JSONNode` to a JSON string.  
- If the value is a `JSONNode`, uses its serialization.
- For other types, converts to native Go types and serializes.

```chariot
setq(jsonStr, toJSON(j))
setq(jsonStr2, toJSON(map("a", 1, "b", 2)))
```

#### `toSimpleJSON(value)`

Converts a Chariot value to a `SimpleJSON` node.  
- Converts to native Go types, then wraps as a `SimpleJSON`.

```chariot
setq(sj, toSimpleJSON(map("x", 1, "y", 2)))
```

---

### Notes

- All JSON parsing functions return an error if the input is not valid JSON.
- `parseJSON` and `parseJSONSimple` return node objects for structured manipulation.
- `parseJSONValue` is best for extracting values directly as Chariot types.
- `toJSON` always returns a string in valid JSON format.
- Use `toSimpleJSON` for lightweight JSON node creation.

---
