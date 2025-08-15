# Chariot Language Reference

## Polymorphic Functions

Chariot provides a set of polymorphic (type-dispatched) functions that operate on multiple data types, such as arrays, strings, JSON, maps, and tree nodes. These functions automatically select the correct behavior based on the type of their arguments.

### Available Polymorphic Functions

| Function                | Description                                                                                   |
|-------------------------|-----------------------------------------------------------------------------------------------|
| `addTo(arr, v1, ...)`   | Add one or more values to an array or slice (in-place for ArrayValue)                        |
| `apply(fn, collection)` | Apply a function to each element of a collection (array, map, JSON, MapNode, TreeNode, etc.) |
| `clone(obj [, name])`   | Deep clone an array, map, JSON, MapNode, TreeNode, or SimpleJSON                             |
| `getAt(obj, idx)`       | Get element at index/key from array, string, JSON, or similar types                          |
| `setAt(obj, idx, value)`| Set element at index/key for array, JSON array, or TreeNode                                  |
| `getAttribute(obj, key)`| Get attribute value from node, map, or JSON object                                           |
| `getAttributes(obj)`    | Get all attributes as a map from node, map, or JSON object                                   |
| `getProp(obj, key)`     | Get property or attribute from map, JSON, SimpleJSON, MapNode, MapValue, or host object      |
| `setProp(obj, key, value)` | Set property or attribute on map, JSON, SimpleJSON, MapNode, MapValue, TreeNode, or host object |
| `getMeta(obj, key)`     | Get metadata value for a node or object                                                      |
| `setMeta(obj, key, value)` | Set metadata value for a node or object                                                   |
| `getAllMeta(obj)`       | Get all metadata as a map for a node or object                                               |
| `indexOf(obj, value [, start])` | Find index of value in string or array                                               |
| `length(obj)`           | Get length of string, array, map, JSON, or tree node                                         |
| `slice(obj, start [, end])` | Return a slice/subset of a string or array                                               |
| `reverse(obj)`          | Reverse a string or array                                                                    |
| `contains(obj, value)`  | Check if string contains substring, or array contains value                                  |
| `split(str, delimiter)` | Split a string into an array using the delimiter                                             |
| `join(arr, delimiter)`  | Join array elements into a string with the delimiter                                         |

---

### Function Details

#### `addTo(arr, v1, ...)`

Add one or more values to an array or slice. For `ArrayValue`, modifies in-place and returns the array.

```chariot
addTo(array(1,2), 3, 4)   // [1, 2, 3, 4]
```

#### `apply(fn, collection)`

Apply a function to each element of a collection (array, map, JSON, MapNode, TreeNode, etc.). The function receives (key, value) for maps, or (index, value) for arrays.

```chariot
apply(myFunc, array(1,2,3))
apply(myFunc, map("a",1,"b",2))
```

#### `clone(obj [, name])`

Deep clone an array, map, JSON, MapNode, TreeNode, or SimpleJSON. For nodes, an optional new name can be provided.

```chariot
clone(array(1,2,3))
clone(myMap)
clone(myTree, "copyName")
```

#### `getAt(obj, idx)`

Get element at index/key from array, string, JSON array, or slice. Returns `DBNull` if out of bounds.

```chariot
getAt(array(1,2,3), 1)   // 2
getAt("hello", 0)        // "h"
getAt(jsonArray, 2)
```

#### `setAt(obj, idx, value)`

Set element at index/key for array, JSON array, or TreeNode. Returns the value set.

```chariot
setAt(array(1,2,3), 1, 99)   // [1, 99, 3]
setAt(jsonArray, 0, "foo")
```

#### `getAttribute(obj, key)`

Get attribute value from node, map, or JSON object.

```chariot
getAttribute(treeNode, "name")
getAttribute(jsonObj, "field")
```

#### `getAttributes(obj)`

Get all attributes as a map from node, map, or JSON object.

```chariot
getAttributes(treeNode)
getAttributes(jsonObj)
```

#### `getProp(obj, key)`

Get property or attribute from map, JSON, SimpleJSON, MapNode, MapValue, or host object.

```chariot
getProp(map("a",1), "a")      // 1
getProp(jsonObj, "name")
```

#### `setProp(obj, key, value)`

Set property or attribute on map, JSON, SimpleJSON, MapNode, MapValue, TreeNode, or host object.

```chariot
setProp(mapNode, "score", 99)
setProp(jsonObj, "active", true)
```

#### `getMeta(obj, key)`

Get metadata value for a node or object, if supported.

```chariot
getMeta(treeNode, "createdBy")
```

#### `setMeta(obj, key, value)`

Set metadata value for a node or object, if supported.

```chariot
setMeta(treeNode, "approved", true)
```

#### `getAllMeta(obj)`

Get all metadata as a map for a node or object, if supported.

```chariot
getAllMeta(treeNode)
```

#### `indexOf(obj, value [, start])`

Find index of value in string or array. For strings, an optional start index can be provided.

```chariot
indexOf("banana", "a")        // 1
indexOf(array(1,2,3,2), 2)    // 1
```

#### `length(obj)`

Get length of string, array, map, JSON node, or tree node.

```chariot
length("hello")               // 5
length(array(1,2,3))          // 3
length(map("a",1,"b",2))      // 2
```

#### `slice(obj, start [, end])`

Return a slice/subset of a string or array from `start` (inclusive) to `end` (exclusive). If `end` is omitted, slices to the end.

```chariot
slice("abcdef", 2, 4)         // "cd"
slice(array(1,2,3,4), 1, 3)   // [2, 3]
```

#### `reverse(obj)`

Reverse a string or array.

```chariot
reverse("abc")                // "cba"
reverse(array(1,2,3))         // [3, 2, 1]
```

#### `contains(obj, value)`

Check if string contains substring, or array contains value.

```chariot
contains("hello", "ll")       // true
contains(array(1,2,3), 2)     // true
```

#### `split(str, delimiter)`

Split a string into an array using the delimiter.

```chariot
split("a,b,c", ",")           // ["a", "b", "c"]
```

#### `join(arr, delimiter)`

Join array elements into a string with the delimiter.

```chariot
join(array("a","b","c"),