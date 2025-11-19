# Chariot Language Reference

## Knapsack Functions

Chariot provides high-performance knapsack optimization functions powered by a native C++ solver. These functions enable solving 0/1 knapsack problems and item selection optimization with capacity constraints.

### Available Knapsack Functions

| Function                  | Description                                      |
|---------------------------|--------------------------------------------------|
| `knapsack(config, [options])` | Solve knapsack problem with given configuration |
| `knapsackConfig(items, capacity, weights, values)` | Generate V2 JSON configuration for knapsack solver |

---

### Function Details

#### `knapsack(config, [options])`

Solve a knapsack optimization problem using the native C++ solver.

**Parameters:**
- `config` (String): JSON configuration string in V2 format (use `knapsackConfig()` to generate)
- `options` (String, optional): JSON options for solver behavior (default: `{}`)

**Returns:** MapValue with solution containing:
- `numItems` (Number): Total number of items in the problem
- `select` (Array): Binary selection array (0 = not selected, 1 = selected)
- `objective` (Number): Total objective value achieved
- `penalty` (Number): Penalty value (if constraints violated)
- `total` (Number): Total score (objective - penalty)

**Example:**
```chariot
# Simple knapsack problem
setq(cfg, knapsackConfig([1, 2, 3], 10.0, [2.0, 3.0, 4.0], [5.0, 6.0, 7.0]))
setq(solution, knapsack(cfg))

# Access solution properties
setq(selected, getProp(solution, "select"))      # [0, 1, 1] or similar
setq(totalValue, getProp(solution, "objective")) # Maximum value achieved
setq(score, getProp(solution, "total"))          # Final score
```

**With Options:**
```chariot
setq(opts, '{"maxIterations": 10000, "seed": 42}')
setq(solution, knapsack(cfg, opts))
```

---

#### `knapsackConfig(items, capacity, weights, values)`

Generate a properly formatted V2 JSON configuration string for the knapsack solver.

**Parameters:**
- `items` (Array): Array of item identifiers (typically sequential numbers)
- `capacity` (Number): Maximum weight capacity constraint
- `weights` (Array): Weight of each item (must match length of `items`)
- `values` (Array): Value of each item (must match length of `items`)

**Returns:** String containing V2 JSON configuration

**Example:**
```chariot
# Create configuration for 3 items
setq(items, [1, 2, 3])
setq(capacity, 10.0)
setq(weights, [2.0, 3.0, 4.0])
setq(values, [5.0, 6.0, 7.0])

setq(config, knapsackConfig(items, capacity, weights, values))
# Returns JSON string with V2 schema
```

**Generated JSON Structure:**
```json
{
  "version": 2,
  "mode": "select",
  "items": {
    "count": 3,
    "attributes": {
      "value": [5.0, 6.0, 7.0],
      "weight": [2.0, 3.0, 4.0]
    }
  },
  "blocks": [
    {"name": "all", "start": 0, "count": 3}
  ],
  "objective": [
    {"attr": "value", "weight": 1.0}
  ],
  "constraints": [
    {"kind": "capacity", "attr": "weight", "limit": 10.0}
  ]
}
```

---

### Complete Workflow Example

```chariot
# Define knapsack problem
setq(itemIds, range(1, 11))  # 10 items
setq(capacity, 50.0)
setq(weights, [5.0, 10.0, 15.0, 20.0, 25.0, 8.0, 12.0, 18.0, 22.0, 7.0])
setq(values, [10.0, 20.0, 30.0, 40.0, 50.0, 15.0, 25.0, 35.0, 45.0, 12.0])

# Generate configuration
setq(cfg, knapsackConfig(itemIds, capacity, weights, values))

# Solve
setq(solution, knapsack(cfg))

# Extract results
setq(selection, getProp(solution, "select"))
setq(totalValue, getProp(solution, "objective"))

# Find selected items
setq(selectedItems, array())
setq(idxCounter, 0)
while(smaller(idxCounter, length(itemIds)), {
  if(equal(getAt(selection, idxCounter), 1), {
    addTo(selectedItems, getAt(itemIds, idxCounter))
  })
  setq(idxCounter, add(idxCounter, 1))
})

print("Selected items:", selectedItems)
print("Total value:", totalValue)
```

---

### Extracting Selected Items

```chariot
# Get solution
setq(cfg, knapsackConfig([101, 102, 103], 15.0, [5.0, 8.0, 10.0], [10.0, 15.0, 20.0]))
setq(sol, knapsack(cfg))

# Extract selected items
setq(items, [101, 102, 103])
setq(selection, getProp(sol, "select"))
setq(picked, array())

# Iterate and collect selected
setq(idx, 0)
while(smaller(idx, length(items)), {
  if(equal(getAt(selection, idx), 1), {
    addTo(picked, getAt(items, idx))
  })
  setq(idx, add(idx, 1))
})

# picked now contains: [101, 103] (or whatever was selected)
```

---

### Multiple Knapsack Calls

```chariot
# Solve different scenarios
setq(items, [1, 2, 3, 4, 5])
setq(weights, [2.0, 3.0, 4.0, 5.0, 6.0])
setq(values, [3.0, 4.0, 5.0, 6.0, 7.0])

# Scenario 1: Small capacity
setq(cfg1, knapsackConfig(items, 10.0, weights, values))
setq(sol1, knapsack(cfg1))

# Scenario 2: Large capacity
setq(cfg2, knapsackConfig(items, 20.0, weights, values))
setq(sol2, knapsack(cfg2))

# Compare objectives
setq(obj1, getProp(sol1, "objective"))
setq(obj2, getProp(sol2, "objective"))
```

---

### Error Handling

```chariot
# Check for valid configuration
setq(result, knapsack(cfg))

# Solution contains penalty field for constraint violations
setq(penalty, getProp(result, "penalty"))
if(bigger(penalty, 0), {
  print("Warning: Solution violates constraints")
}, {
  print("Valid solution found")
})
```

---

### Performance Considerations

- **Native C++ Solver**: Uses optimized compiled library (libknapsack)
- **Platform Support**: macOS (ARM64, x86_64), Linux (x86_64, ARM64)
- **Problem Size**: Efficiently handles 100s of items
- **Large Problems**: For 1000+ items, consider breaking into sub-problems
- **Caching**: Reuse configuration strings when solving similar problems

---

### V2 JSON Schema Notes

The `knapsackConfig()` helper generates the correct V2 schema automatically. If manually constructing JSON:

**Required Fields:**
- `version`: Must be `2`
- `mode`: Must be `"select"`
- `items.count`: Number of items
- `items.attributes`: Object with `value` and `weight` arrays
- `blocks`: Array with at least one block definition
- `objective`: Array of objective definitions
- `constraints`: Array of constraint definitions

**Constraint Types:**
- `capacity`: Weight limit constraint (`{"kind": "capacity", "attr": "weight", "limit": 50.0}`)

---

### Common Patterns

**Generate Test Data:**
```chariot
setq(n, 20)
setq(items, range(1, add(n, 1)))
setq(weights, array())
setq(values, array())

# Random-like weights and values
setq(i, 0)
while(smaller(i, n), {
  addTo(weights, mul(i, 2.5))
  addTo(values, mul(i, 3.0))
  setq(i, add(i, 1))
})

setq(cfg, knapsackConfig(items, 100.0, weights, values))
setq(sol, knapsack(cfg))
```

**Verify Solution Feasibility:**
```chariot
setq(solution, knapsack(cfg))
setq(selection, getProp(solution, "select"))
setq(totalWeight, 0)

setq(idx, 0)
while(smaller(idx, length(selection)), {
  if(equal(getAt(selection, idx), 1), {
    setq(totalWeight, add(totalWeight, getAt(weights, idx)))
  })
  setq(idx, add(idx, 1))
})

if(smallerEq(totalWeight, capacity), {
  print("Solution is feasible")
}, {
  print("Solution exceeds capacity!")
})
```

---

### Notes

- All knapsack functions are closures and must be called as such
- The C++ solver uses branch-and-bound with greedy heuristics
- Empty item arrays return an error (solver requires at least one item)
- Arrays must have matching lengths (items, weights, values)
- Capacity must be a positive number
- Solution is optimal for small-medium problems, near-optimal for large problems
- Integration with Chariot's polymorphic functions enables powerful data processing workflows

---
