// filepath: /Users/williamhouse/go/src/github.com/bhouse1273/knapsack/tests/CHARIOT_CGO_TESTS.md
# Chariot CGO Integration Test Suite

This directory contains comprehensive tests for the knapsack C API to help debug CGO integration issues in go-chariot.

## Files

### Python Tests
- **[`python/test_knapsack_c_api.py`](python/test_knapsack_c_api.py )** - Complete validation suite in Python
  - 7 test cases covering edge cases and typical usage
  - Generates Go/CGO code examples
  - Validates memory alignment and pointer handling

### Go Tests  
- **[`go/knapsack_cgo_test.go`](go/knapsack_cgo_test.go )** - Reference CGO implementation
  - Shows correct pointer conversion
  - Validates results and memory integrity
  - Includes benchmarks and examples

## Running Python Tests

### Prerequisites
```bash
pip install numpy
```

### Build the library
```bash
cd knapsack-library
mkdir -p build && cd build
cmake .. -DCMAKE_BUILD_TYPE=Release -DBUILD_CPU_ONLY=ON
cmake --build . --target knapsack -j8
cd ../..
```

### Run tests
```bash
python tests/python/test_knapsack_c_api.py
```

### Expected output
```
============================================================
KNAPSACK C API VALIDATION TESTS
============================================================
Library: knapsack-library/build/libknapsack.dylib

=== Test 1: Basic Knapsack ===
Items: 5
Weights: [2, 3, 4, 5, 9]
Values: [3, 4, 5, 8, 10]
Capacity: 10
Result: Total value = 13
Selection: [1, 1, 0, 1, 0]
Selected items: [0, 1, 3]
Total weight used: 10/10
Total value: 13
✅ PASSED

[... more tests ...]

SUMMARY: 7 passed, 0 failed out of 7 tests
```

## Running Go Tests

### Prerequisites
```bash
# Ensure knapsack library is built (see above)
```

### Run tests
```bash
cd tests/go
go test -v
```

### Expected output
```
=== RUN   TestBasicKnapsack
    knapsack_cgo_test.go:23: Test 1: Basic Knapsack Problem
    knapsack_cgo_test.go:27: Items: 5
    knapsack_cgo_test.go:28: Weights: [2 3 4 5 9]
    knapsack_cgo_test.go:29: Values: [3 4 5 8 10]
    knapsack_cgo_test.go:30: Capacity: 10
    knapsack_cgo_test.go:41: Total value: 13
    knapsack_cgo_test.go:42: Selection: [1 1 0 1 0]
    [... validation logs ...]
    knapsack_cgo_test.go:62: ✅ Total weight used: 10/10
    knapsack_cgo_test.go:63: ✅ Total value: 13
--- PASS: TestBasicKnapsack (0.00s)

[... more tests ...]

PASS
ok      knapsack_test   0.234s
```

## Common CGO Issues & Solutions

### Issue 1: Wrong pointer conversion
❌ **Wrong:**
```go
C.knapsack(n, weights, values, capacity, selection)  // Passing Go slices directly
```

✅ **Correct:**
```go
C.knapsack(
    n,
    (*C.int)(unsafe.Pointer(&weights[0])),
    (*C.int)(unsafe.Pointer(&values[0])),
    capacity,
    (*C.int)(unsafe.Pointer(&selection[0])),
)
```

### Issue 2: Type mismatch
❌ **Wrong:**
```go
n := 5                  // Go int (might be int64)
weights := []int{...}   // Go int slice
```

✅ **Correct:**
```go
n := C.int(5)                    // C int32
weights := []C.int{C.int(2), ...} // C int32 slice
```

### Issue 3: Array size mismatch
❌ **Wrong:**
```go
weights := []C.int{1, 2, 3}
values := []C.int{10, 20}      // Different size!
n := C.int(3)
```

✅ **Correct:**
```go
n := C.int(3)
weights := []C.int{1, 2, 3}    // Size matches n
values := []C.int{10, 20, 30}  // Size matches n
selection := make([]C.int, 3)  // Size matches n
```

### Issue 4: Not allocating selection array
❌ **Wrong:**
```go
var selection []C.int  // nil slice
C.knapsack(n, weights, values, capacity, (*C.int)(unsafe.Pointer(&selection[0])))
// Panic: index out of range
```

✅ **Correct:**
```go
selection := make([]C.int, n)  // Pre-allocated
C.knapsack(n, (*C.int)(unsafe.Pointer(&weights[0])), ...)
```

### Issue 5: Memory corruption check
Always verify input arrays aren't corrupted after the call:

```go
// Before call
origWeights := make([]C.int, len(weights))
copy(origWeights, weights)

// Call function
totalValue := C.knapsack(...)

// After call - verify memory
for i := range weights {
    if weights[i] != origWeights[i] {
        panic("Input array was corrupted!")
    }
}
```

## Debugging Checklist

When debugging CGO issues in go-chariot:

- [ ] **Verify library path**: Check LDFLAGS points to correct `.a` or `.dylib`
- [ ] **Check array types**: All arrays should be `[]C.int`, not `[]int`
- [ ] **Validate array sizes**: `len(weights) == len(values) == n`
- [ ] **Allocate output**: `selection := make([]C.int, n)` before call
- [ ] **Use unsafe.Pointer**: `(*C.int)(unsafe.Pointer(&array[0]))`
- [ ] **Check return value**: Should match sum of selected item values
- [ ] **Validate constraints**: Sum of selected weights <= capacity
- [ ] **Print pointers**: Log memory addresses to verify no corruption
- [ ] **Test small cases**: Start with n=1, n=3 before larger problems
- [ ] **Compare with Python**: Run equivalent test in Python first

## Sharing with Chariot Team

To help chariot-ecosystem debug their CGO integration:

1. **Share Python test results:**
   ```bash
   python tests/python/test_knapsack_c_api.py > python_results.txt
   ```

2. **Share this test file** so they can replicate exact scenarios

3. **Compare outputs**: Their CGO code should produce identical results to Python

4. **Check generated Go examples**: Python script outputs correct CGO code snippets

## Test Cases Covered

| Test | Purpose | Key Validation |
|------|---------|----------------|
| Basic Knapsack | Typical usage | Multiple items, optimal selection |
| Single Item | Edge case | Single item that fits |
| Single Item Too Heavy | Rejection | Item exceeds capacity |
| Zero Capacity | Edge case | No items can fit |
| Large Problem | Scalability | 20 items, various weights/values |
| Memory Alignment | Safety | Odd array sizes, pointer validity |
| Greedy Comparison | Correctness | Optimal >= greedy solution |

## Expected Behavior

All tests should:
- ✅ Return total value >= 0
- ✅ Selected weight <= capacity
- ✅ Sum of selected values == return value
- ✅ No memory corruption in input arrays
- ✅ Selection array contains only 0 or 1
- ✅ Selection array size matches n

## Contact

If chariot-ecosystem team finds discrepancies between these tests and their CGO implementation, please share:
1. The exact Go code being used
2. Input values (n, weights, values, capacity)
3. Expected vs actual output
4. Any error messages or panics

This will help isolate whether the issue is in:
- CGO bindings
- Pointer conversion
- Type marshaling
- Library linkage
- Algorithm implementation