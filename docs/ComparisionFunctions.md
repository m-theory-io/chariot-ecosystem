# Chariot Language Reference

## Comparison Functions

Chariot provides a set of closure functions for comparing values, supporting numbers, strings, booleans, and nulls. These functions are essential for decision-making and control flow.

### Available Comparison Functions

| Function         | Description                                               |
|------------------|----------------------------------------------------------|
| `equal(a, b, ...)` | Returns `true` when every argument matches the first value |
| `unequal(a, b, ...)` | Returns `true` only when every argument differs from the rest |
| `bigger(a, b)`   | Returns `true` if `a` is greater than `b` (number/string)|
| `smaller(a, b)`  | Returns `true` if `a` is less than `b` (number/string)   |
| `biggerEq(a, b)` | Returns `true` if `a` is greater than or equal to `b`    |
| `smallerEq(a, b)`| Returns `true` if `a` is less than or equal to `b`       |
| `and(a, b, ...)` | Logical AND of all arguments                             |
| `or(a, b, ...)`  | Logical OR of all arguments                              |
| `not(a, ...)`    | Logical NOT (true only when all args are false/DBNull)   |
| `iif(cond, x, y)`| Returns `x` if `cond` is true, else returns `y`          |

---

### Function Details

#### `equal(a, b, ...)`

Returns `true` only when every provided argument evaluates to the same value. Supports numbers, strings, booleans, and nulls. The `equals()` helper remains available as an alias but simply forwards to `equal()`.

```chariot
equal(5, 5, 5)          // true
equal('a', 'b')         // false
equal(true, true, true) // true
equal(DBNull, DBNull)   // true
equal(1, 1, 2)          // false
```

#### `unequal(a, b, ...)`

Returns `true` only when no two arguments are equal to one another. Supply at least two operands.

```chariot
unequal(5, 3)            // true
unequal('x', 'x')        // false
unequal(1, 2, 3)         // true
unequal('a', 'b', 'a')   // false (duplicate)
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

#### `not(a, ...)`

Logical NOT. Accepts one or more arguments and returns `true` only when every argument evaluates to `false` or `DBNull`. If any argument is `true`, the closure returns `false` immediately.

```chariot
not(false)                 // true
not(false, DBNull)         // true (both operands are falsey)
not(false, true)           // false (one operand is true)
not(DBNull, false, false)  // true
not(DBNull, true)          // false
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