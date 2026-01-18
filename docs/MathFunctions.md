# Chariot Language Reference

## Math Functions

Chariot provides a comprehensive set of mathematical and financial functions, including arithmetic, statistics, random number/string generation, trigonometry, logarithms, and financial calculations. All math functions are closures.

---

### Arithmetic Functions

| Function           | Description                                 |
|--------------------|---------------------------------------------|
| `add(a, b)`        | Addition                                    |
| `sub(a, b)`        | Subtraction                                 |
| `mul(a, b)`        | Multiplication                              |
| `div(a, b)`        | Division (error if `b` is zero)             |
| `mod(a, b)`        | Modulo (error if `b` is zero)               |

---

### Advanced Math Functions

| Function           | Description                                 |
|--------------------|---------------------------------------------|
| `abs(x)`           | Absolute value                              |
| `sqrt(x)`          | Square root (error if `x` is negative)      |
| `pow(base, exp)`   | Power (base raised to exp)                  |
| `exp(x)`           | Exponential (eˣ)                            |
| `floor(x)`         | Floor (largest integer ≤ x)                 |
| `ceiling(x)` / `ceil(x)` | Ceiling (smallest integer ≥ x)        |
| `round(x [, places])` | Round to nearest integer or decimal places|
| `int(x)`           | Truncate to integer                         |
| `transpose(matrix)` | Matrix transpose                            |
| `matmul(left, right)` | Matrix multiplication (dimension-aware)   |
| `solveLinear(matrix, vector)` | Solve square system A·x = b        |
| `lsp(matrix, vector)` | Least-squares projection summary          |
| `vectorScale(vector, scalar)` | Scale a vector by a scalar factor  |
| `eigenSymmetric(matrix)` | Eigenvalues/vectors for real symmetric matrices |
| `eigen(matrix)` | Full eigen decomposition (real + imaginary parts) |
| `dominantEigen(matrix [, tolerance [, maxIterations]])` | Power-iteration dominant eigenpair |
| `schur(matrix)` | Real Schur form (orthogonal Q, quasi-triangular T) |

---

### Logarithmic Functions

| Function           | Description                                 |
|--------------------|---------------------------------------------|
| `log(x)`           | Natural logarithm (ln)                      |
| `log10(x)`         | Base-10 logarithm                           |
| `log2(x)`          | Base-2 logarithm                            |
| `ln(x)`            | Alias for natural logarithm                 |

---

### Trigonometric Functions

| Function           | Description                                 |
|--------------------|---------------------------------------------|
| `sin(x)`           | Sine                                        |
| `cos(x)`           | Cosine                                      |
| `tan(x)`           | Tangent                                     |

---

### Mathematical Constants

| Function           | Description                                 |
|--------------------|---------------------------------------------|
| `pi()`             | Returns π                                   |
| `e()`              | Returns Euler's number                      |

---

### Statistics Functions

| Function           | Description                                 |
|--------------------|---------------------------------------------|
| `max(a, b, ...)`   | Maximum of arguments                        |
| `min(a, b, ...)`   | Minimum of arguments                        |
| `sum(a, b, ...)`   | Sum of arguments                            |
| `avg(a, b, ...)`   | Average of arguments                        |

---

### Random Functions

| Function           | Description                                 |
|--------------------|---------------------------------------------|
| `random()`         | Random float in [0, 1)                      |
| `random(max)`      | Random float in [0, max)                    |
| `random(min, max)` | Random float in [min, max)                  |
| `randomString(length)` | Random alphanumeric string of given length |
| `randomSeed(seed)` | Set the random seed (for reproducibility)   |

---

### Financial Functions

| Function           | Description                                 |
|--------------------|---------------------------------------------|
| `pct(value, percent)` | Calculate percent of value               |
| `pmt(rate, nper, pv)` | Payment for loan (rate, periods, present value) |
| `nper(rate, pmt, pv)` | Number of periods for loan               |
| `rate(nper, pmt, pv)` | Interest rate for loan                   |
| `irr(cf1, cf2, ...)`  | Internal rate of return for cash flows   |
| `npv(rate, cf1, cf2, ...)` | Net present value for cash flows    |
| `fv(rate, nper, pmt)` | Future value of series                   |
| `pv(rate, nper, pmt)` | Present value of series                  |
| `amortize(rate, nper, pv)` | Amortization schedule (array of maps)|
| `balloon(rate, nper, pv, paid)` | Remaining balance after payments|
| `interestOnly(rate, pv)` | Interest-only payment                 |
| `interestOnlySchedule(rate, nper, pv)` | Interest-only schedule (array of maps) |
| `depreciation(cost, salvage, life, method)` | Depreciation (straight-line, double-declining-balance) |
| `apr(rate, nper [, fees])` | Annual percentage rate, with optional fees |
| `loanBalance(rate, nper, pv, paid)` | Remaining loan balance      |

---

### Function Details

#### Arithmetic

```chariot
add(2, 3)         // 5
sub(10, 4)        // 6
mul(6, 7)         // 42
div(8, 2)         // 4
mod(10, 3)        // 1
```

#### Advanced Math

```chariot
abs(-5)           // 5
sqrt(9)           // 3
pow(2, 8)         // 256
exp(1)            // 2.718...
floor(3.7)        // 3
ceiling(3.1)      // 4
ceil(3.1)         // 4
round(3.14159, 2) // 3.14
int(3.99)         // 3
transpose([[1, 2, 3], [4, 5, 6]])
// [[1, 4], [2, 5], [3, 6]]
matmul([[1, 2], [3, 4]], [[2, 0], [1, 2]])
// [[4, 4], [10, 8]]
solveLinear([[2, 1], [1, 1]], [5, 3])
// [2, 1]
lsp([[1, 0], [0, 1], [1, 1]], [1, 2, 2])
// {coefficients: [0.667, 1.667], projection: [0.667, 1.667, 2.333], residual: [0.333, 0.333, -0.333], residualNorm: 0.577}
vectorScale([1, 2, 3], 0.5)
// [0.5, 1, 1.5]
eigenSymmetric([[2, 1], [1, 2]])
// {values: [1, 3], vectors: [[-0.707, 0.707], [0.707, 0.707]]}
eigen([[0, -1], [1, 0]])
// {valuesReal: [0, 0], valuesImag: [1, -1], vectorsReal: [[0.707, 0.707], [0.707, -0.707]], vectorsImag: [[0.707, -0.707], [-0.707, -0.707]]}
dominantEigen([[4, 1], [0, 2]], 1e-6, 200)
// {value: 4, vector: [0.999, 0.04]}
schur([[0, -1], [1, 0]])
// {T: [[0, 1], [-1, 0]], Q: [[0.707, -0.707], [0.707, 0.707]], valuesReal: [0, 0], valuesImag: [1, -1], blockSizes: [2]}

`eigenSymmetric` returns orthonormal eigenvectors as columns in the `vectors` matrix. The general `eigen` closure exposes separate real/imaginary matrices so you can rebuild complex eigenvectors or keep just the real portions when the spectrum is real. `dominantEigen` uses power iteration; the optional tolerance/maxIterations allow you to trade accuracy for speed.
`schur` returns the real Schur form where `Q` is orthogonal and `T` is quasi-triangular with 1×1 or 2×2 diagonal blocks—`blockSizes` tells you which is which while `valuesReal/valuesImag` expose the corresponding eigenvalues.

> **LAPACK dependency:** The `schur` closure delegates to Gonum's LAPACK implementation (`Dgehrd → Dorghr → Dhseqr`). That means:
> - `gonum.org/v1/gonum/lapack/gonum` is now a required module—`go mod tidy` must run before building or testing go-chariot.
> - Each decomposition allocates ~3·n² float64 slots for intermediate matrices and workspace; large inputs can consume several megabytes and take O(n³) time.
> - The default backend is pure Go, so `CGO_ENABLED=0` builds continue to work, but the routine is slower than hardware-accelerated BLAS/LAPACK. If you configure Gonum to use a cgo-backed BLAS, ensure the host toolchain (Accelerate/OpenBLAS/MKL) is present.
```

#### Logarithmic

```chariot
log(10)           // 2.302...
log10(100)        // 2
log2(8)           // 3
ln(10)            // 2.302...
```

#### Trigonometric

```chariot
sin(pi())         // 0
cos(0)            // 1
tan(pi())         // 0
```

#### Constants

```chariot
pi()              // 3.141592653589793
e()               // 2.718281828459045
```

#### Statistics

```chariot
max(1, 5, 3)      // 5
min(1, 5, 3)      // 1
sum(1, 2, 3)      // 6
avg(2, 4, 6)      // 4
```

#### Random

```chariot
random()          // e.g., 0.12345
random(10)        // e.g., 7.654
random(5, 10)     // e.g., 8.321
randomString(8)   // e.g., "aZ3kLmP2"
randomSeed(42)    // Sets the random seed
```

#### Financial

```chariot
pct(200, 15)                  // 30
pmt(0.05, 12, 1000)           // Payment for loan
nper(0.05, 100, 1000)         // Number of periods
rate(12, 100, 1000)           // Interest rate
irr(-1000, 300, 420, 680)     // IRR for cash flows
npv(0.1, -1000, 300, 420, 680)// NPV for cash flows
fv(0.05, 12, 100)             // Future value
pv(0.05, 12, 100)             // Present value
amortize(0.05, 12, 1000)      // Amortization schedule (array of maps)
balloon(0.05, 12, 1000, 6)    // Remaining balance after 6 payments
interestOnly(0.05, 1000)      // Interest-only payment
interestOnlySchedule(0.05, 12, 1000) // Interest-only schedule
depreciation(10000, 1000, 5, "straight-line") // Depreciation
apr(0.05, 12, 0.01)           // APR with fees
loanBalance(0.05, 12, 1000, 6)// Remaining loan balance
```

---

### Notes

- All math functions require numeric arguments (`Number`).
- Division and modulo by zero will return an error.
- Financial functions use standard formulas (see code for details).
- `amortize` and `interestOnlySchedule` return arrays of maps with period-by-period breakdowns.
- `depreciation` supports `"straight-line"` and `"double-declining-balance"` methods.
- `randomSeed` is provided for reproducibility in random number generation.

---