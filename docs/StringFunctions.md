# Chariot Language Reference

## String Functions

Chariot provides a rich set of string functions for creation, formatting, manipulation, searching, splitting, joining, and interpolation. Strings in Chariot are immutable and support Unicode.

---

### String Creation and Formatting

| Function           | Description                                         |
|--------------------|-----------------------------------------------------|
| `concat(a, b, ...)`| Concatenate arguments as strings                    |
| `string(x)`        | Convert any value to a string                       |
| `format(fmt, a, ...)` | Format string using Go-style formatting          |
| `sprintf(fmt, a, ...)` | Alias for `format`                              |
| `append(a, b, ...)`| Alias for `concat`                                  |

---

### Character and ASCII Functions

| Function           | Description                                         |
|--------------------|-----------------------------------------------------|
| `char(str, pos)`   | Get character at position (0-based)                 |
| `charAt(str, pos)` | Alias for `char` (returns DBNull if out of bounds)  |
| `ascii(str)`       | ASCII code of first character                       |
| `atPos(str, pos)`  | Get character at position (error if out of bounds)  |

---

### Case and Trimming Functions

| Function           | Description                                         |
|--------------------|-----------------------------------------------------|
| `lower(str)`       | Convert string to lowercase                         |
| `upper(str)`       | Convert string to uppercase                         |
| `trimLeft(str)`    | Trim whitespace from the left                       |
| `trimRight(str)`   | Trim whitespace from the right                      |
| `trim(str [, chars])` | Trim whitespace or custom characters from both ends|

---

### Substring and Replace Functions

| Function           | Description                                         |
|--------------------|-----------------------------------------------------|
| `substring(str, start [, length])` | Substring from `start` for `length` chars|
| `substr(str, start, length)`    | Alias for `substring`                   |
| `right(str, count)`             | Rightmost `count` characters            |
| `replace(str, old, new [, count])` | Replace `old` with `new` (optionally limit count) |

---

### Length and Digit Functions

| Function           | Description                                         |
|--------------------|-----------------------------------------------------|
| `strlen(str)`      | Length of string (alias for `length`)               |
| `digits(str)`      | Extract only digit characters from string           |

---

### Search and Occurrence Functions

| Function           | Description                                         |
|--------------------|-----------------------------------------------------|
| `lastPos(str, substr)` | Last index of `substr` in `str`                 |
| `occurs(str, substr)`  | Count occurrences of `substr` in `str`          |
| `hasPrefix(str, prefix)` | Returns `true` if string starts with prefix   |
| `hasSuffix(str, suffix)` | Returns `true` if string ends with suffix     |

---

### Split and Join Functions

| Function           | Description                                         |
|--------------------|-----------------------------------------------------|
| `split(str, delim)`| Split string into array by delimiter                |
| `join(arr, delim)` | Join array elements into string with delimiter      |

---

### Padding Functions

| Function           | Description                                         |
|--------------------|-----------------------------------------------------|
| `padLeft(str, totalLen, padChar)` | Pad string on the left to total length|
| `padRight(str, totalLen, padChar)`| Pad string on the right to total length|

---

### String Interpolation

| Function           | Description                                         |
|--------------------|-----------------------------------------------------|
| `interpolate(template)` | Replace `${var}` in template with variable values from scope |

---

### Example Usage

```chariot
concat('Hello, ', 'world!')             // "Hello, world!"
string(123)                             // "123"
format("Value: %.2f", 3.14159)          // "Value: 3.14"
sprintf("Value: %d", 42)                // "Value: 42"
char("abc", 1)                          // "b"
charAt("abc", 2)                        // "c"
ascii("A")                              // 65
atPos("abc", 0)                         // "a"
lower("ABC")                            // "abc"
upper("abc")                            // "ABC"
trimLeft("   hello")                    // "hello"
trimRight("hello   ")                   // "hello"
trim("  hello  ")                       // "hello"
trim("xxhelloxx", "x")                  // "hello"
substring("abcdef", 2, 3)               // "cde"
substr("abcdef", 2, 3)                  // "cde"
right("abcdef", 2)                      // "ef"
replace("banana", "a", "o", 2)          // "bonona"
strlen("hello")                         // 5
digits("a1b2c3")                        // "123"
lastPos("banana", "a")                  // 5
occurs("banana", "an")                  // 1
hasPrefix("banana", "ba")               // true
hasSuffix("banana", "na")               // true
split("a,b,c", ",")                     // ["a", "b", "c"]
join(array("a","b","c"), "-")           // "a-b-c"
padLeft("42", 5, "0")                   // "00042"
padRight("42", 5, "x")                  // "42xxx"
interpolate("Hello, ${name}!")          // If name="Alice", returns "Hello, Alice!"
```

---

### Notes

- All string indices are zero-based.
- `charAt` returns `DBNull` for out-of-bounds; `atPos` returns an error.
- `replace` can limit the number of replacements with the optional `count` argument.
- `interpolate` replaces `${var}` with the value of `var` from the current scope.
- `split` returns an array of strings; `join` expects an array as the first argument.
- `strlen` is an alias for the generic `length` function.
- `padLeft` and `padRight` pad with the specified character to the desired length.

---