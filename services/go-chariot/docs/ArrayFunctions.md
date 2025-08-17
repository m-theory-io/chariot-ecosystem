# Chariot Language Reference

## Array Functions

Chariot arrays are mutable, zero-based collections of values. The following functions are available for array creation and manipulation. All functions are called as closures.

### Available Array Functions

| Function                  | Description                                      |
|---------------------------|--------------------------------------------------|
| `array(...)`              | Create a new array from arguments                |
| `addTo(arr, v1, ...)`     | Append one or more values to the array (in-place)|
| `removeAt(arr, i)`        | Remove the element at index `i` (in-place)       |
| `lastIndex(arr, v)`       | Return the last index of `v` in the array, or -1 if not found |
| `slice(arr, start, [end])`| Return a subarray from `start` (inclusive) to `end` (exclusive); `end` defaults to array length |
| `reverse(arr)`            | Return a new array with elements in reverse order|

---

### Function Details

#### `array(...)`

Create a new array from the provided arguments.

```chariot
setq(numbers, array(1, 2, 3, 4))
```

#### `addTo(arr, v1, v2, ...)`

Append one or more values to the array. Returns the modified array.

```chariot
addTo(numbers, 5, 6)  // numbers becomes [1, 2, 3, 4, 5, 6]
```

#### `removeAt(arr, i)`

Remove the element at index `i`. Returns the modified array.

```chariot
removeAt(numbers, 2)  // numbers becomes [1, 2, 4]
```

#### `lastIndex(arr, value)`

Return the last index of `value` in the array, or -1 if not found.

```chariot
lastIndex(array(1, 2, 3, 2), 2) // 3
lastIndex(array(1, 2, 3), 9)    // -1
```

#### `slice(arr, start, [end])`

Return a subarray from `start` (inclusive) to `end` (exclusive). If `end` is omitted, it defaults to the array length. Negative indices count from the end.

```chariot
slice(numbers, 1, 3)   // [2, 3]
slice(numbers, -2)     // Last two elements
```

#### `reverse(arr)`

Return a new array with elements in reverse order.

```chariot
reverse(numbers) // [4, 3, 2, 1]
```

---

### Notes

- Arrays are zero-indexed.
- All array functions in Chariot are closures and must be called as such.
- Most array functions mutate the array in-place and return the modified array or value, except `slice` and `reverse`, which return new arrays.
- Use `array(...)` to create new arrays.
- Polymorphic functions that accept arrays as inputs are documented in PolymorphicFunctions.md

---