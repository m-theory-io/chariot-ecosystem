# Chariot Language Reference

## Flow Control Functions

Chariot supports a set of flow control constructs for scripting, including loops, conditionals, and control keywords. These are implemented with special support in the parser, AST, and runtime, and are not ordinary closures unless noted.

---

### Supported Flow Control Functions and Keywords

| Construct / Function      | Description                                                      |
|--------------------------|------------------------------------------------------------------|
| `while(condition) { ... }` | Loop while the condition is true                               |
| `if(condition) { ... } else { ... }` | Conditional execution                               |
| `switch(expr) { case(val): ...; default(): ...; }` | Multi-branch selection on 1 input value |
| `switch() { case(condition): ...; default(): ...; }` | Multi-branch selection on conditions  |
| `case(value): ...`        | Case branch within a switch                                      |
| `default()`               | Default branch within a switch                                   |
| `break`                   | Exit the nearest enclosing loop or switch                        |
| `continue`                | Skip to the next iteration of the nearest enclosing loop         |

---

### Function Details

#### `while(condition) { ... }`

Executes the block repeatedly as long as the condition is true.

```chariot
setq(i, 0)
while(lt(i, 10)) {
    setq(i, add(i, 1))
}
```

#### `if(condition) { ... } else { ... }`

Executes the true branch if the condition is true, otherwise executes the else branch (if present).

```chariot
if(gt(score, 90)) {
    setq(grade, "A")
} else {
    setq(grade, "B")
}
```

#### `switch(expr) { case(val): ...; default(): ...; }`

Selects a branch to execute based on the value of `expr`. Each `case` is checked for equality. If no case matches, the `default()` branch is executed (if present).

```chariot
switch(color) {
    case("red"):
        setq(code, "#FF0000")
    case("green"):
        setq(code, "#00FF00")
    default():
        setq(code, "#000000")
}
```

#### `switch() { case(condition): ...; default(): ...; }`

Selects a branch to execute based on the boolean value of `condition`, for each `case`. If no case matches, the `default()` branch is executed (if present).

```chariot
switch() {
    case(equal(globalConfig.Color1, true)):
        setq(code, "#FF0000")
    case(equal(globalConfig.Color2, true)):
        setq(code, "#00FF00")
    default():
        setq(code, "#000000")
}
```

#### `case(value): ...`

Defines a branch within a `switch(expr)` statement. The block is executed if the case matches.

#### `case(condition): ...`

Defines a branch within a `switch()` statement. The block is executed if the case `condition` evaluates `true`.

#### `default()`

Defines the default branch within a `switch` statement, executed if no case matches.

#### `break`

Exits the nearest enclosing loop or switch statement immediately.

```chariot
while(true) {
    if(shouldStop()) {
        break
    }
}
```

#### `continue`

Skips the rest of the current loop iteration and continues with the next iteration.

```chariot
setq(i, 0)
while(smaller(i, 10)) {
    setq(i, add(i, 1))
    if(equal(mod(i, 2), 0)) {
        continue
    }
    logPrint(i)
}
```

---

### Notes

- `break` is implemented as a special control flow error in the AST and runtime.
- `while`, `if`, `switch`, `case`, `default`, and `else` are handled by the parser and AST nodes, not as ordinary closures (except `default()` which is a function).
- `else` is always paired with `if` and cannot be used alone.
- `case` and `default()` are only valid inside a `switch` statement.
- All arithmetic and logical operations are performed using functions (e.g., `add()`, `mul()`, `smaller()`, `equal()`, etc.), not operators.
- For iteration over map or array elements, see the `apply()` function in PolymorphicFunctions.md
- These constructs allow for expressive and readable control flow in Chariot scripts.

---