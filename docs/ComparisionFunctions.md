# Chariot Language Reference

## Comparison Functions

Chariot provides a set of closure functions for comparing values, supporting numbers, strings, booleans, and nulls. These functions are essential for decision-making and control flow.

### Available Comparison Functions

| Function         | Description                                               |
|------------------|----------------------------------------------------------|
| `equal(a, b)`    | Returns `true` if `a` and `b` are equal                  |
| `equals(a, b)`   | Alias for `equal`                                        |
| `unequal(a, b)`  | Returns `true` if `a` and `b` are not equal              |
| `bigger(a, b)`   | Returns `true` if `a` is greater than `b` (number/string)|
| `smaller(a, b)`  | Returns `true` if `a` is less than `b` (number/string)   |
| `biggerEq(a, b)` | Returns `true` if `a` is greater than or equal to `b`    |
| `smallerEq(a, b)`| Returns `true` if `a` is less than or equal to `b`       |
| `and(a, b, ...)` | Logical AND of all arguments                             |
| `or(a, b, ...)`  | Logical OR of all arguments                              |
| `not(a)`         | Logical NOT                                              |
| `iif(cond, x, y)`| Returns `x` if `cond` is true, else returns `y`          |

---

### Function Details

#### `equal(a, b)` / `equals(a, b)`

Returns `true` if `a` and `b` are equal. Supports numbers, strings, booleans, and nulls.

```chariot
equal(5, 5)         // true
equal('a', 'b')     // false
equal(true, false)  // false
equal(DBNull, DBNull) // true
```

#### `unequal(a, b)`

Returns `true` if `a` and `b` are not equal.

```chariot
unequal(5, 3)       // true
unequal('x', 'x')   // false
```

#### `bigger(a, b)`

Returns `true` if `a` is greater than `b`. Works for numbers and strings.

```chariot
bigger(10, 5)       // true
bigger('b', 'a')    // true
```

#### `smaller(a, b)`

Returns `true` if `a` is less than `b`. Works for numbers and strings.

```chariot
smaller(2, 3)       // true
smaller('a', 'b')   // true
```

#### `biggerEq(a, b)`

Returns `true` if `a` is greater than or equal to `b`.

```chariot
biggerEq(5, 5)      // true
biggerEq(6, 5)      // true
biggerEq(4, 5)      // false
```

#### `smallerEq(a, b)`

Returns `true` if `a` is less than or equal to `b`.

```chariot
smallerEq(5, 5)     // true
smallerEq(4, 5)     // true
smallerEq(6, 5)     // false
```

#### `and(a, b, ...)`

Logical AND of all arguments. Returns `true` if all are true, otherwise `false`. If any argument is `DBNull`, returns `false`. Short-circuits on first `false` value.

```chariot
and(true, true, true)   // true
and(true, false)        // false
and(true, DBNull)       // false (null in AND returns false)
```

#### `or(a, b, ...)`

Logical OR of all arguments. Returns `true` if any argument is true. If any argument is `DBNull`, returns `true`. Short-circuits on first `true` value.

```chariot
or(false, false, true)  // true
or(false, false)        // false
or(false, DBNull)       // true (null in OR returns true)
```

#### `not(a)`

Logical NOT. Returns `true` if `a` is false, and vice versa. `not(DBNull)` returns `true`.

```chariot
not(true)    // false
not(false)   // true
not(DBNull)  // true
```

#### `iif(cond, x, y)`

Immediate if: returns `x` if `cond` is true, otherwise returns `y`.

```chariot
iif(true, 'yes', 'no')   // 'yes'
iif(false, 1, 2)         // 2
```

---

### Notes

- All comparison functions handle `DBNull` (null) values safely.
- `bigger`, `smaller`, `biggerEq`, and `smallerEq` work for both numbers and strings.
- Logical functions (`and`, `or`, `not`) require boolean arguments.
- `iif` is a functional alternative to `if` for simple conditional expressions.
- All functions are closures and must be called as such.

---