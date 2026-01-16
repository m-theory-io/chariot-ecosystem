package chariot

import (
	"math"
	"testing"
)

func TestLeastSquaresProjection(t *testing.T) {
	rt := NewRuntime()
	RegisterMath(rt)

	matrix := NewArray()
	matrix.Append(arrayFromNumbers(1, 0))
	matrix.Append(arrayFromNumbers(0, 1))
	matrix.Append(arrayFromNumbers(1, 1))

	vector := arrayFromNumbers(1, 2, 2)

	val, err := rt.funcs["lsp"](matrix, vector)
	if err != nil {
		t.Fatalf("lsp returned error: %v", err)
	}

	result, ok := val.(*MapValue)
	if !ok {
		t.Fatalf("expected MapValue result, got %T", val)
	}

	coeffs := extractFloatSlice(t, result, "coefficients")
	projection := extractFloatSlice(t, result, "projection")
	residual := extractFloatSlice(t, result, "residual")

	residualNormVal, ok := result.Get("residualNorm")
	if !ok {
		t.Fatalf("missing residualNorm in result")
	}
	residualNorm, ok := residualNormVal.(Number)
	if !ok {
		t.Fatalf("residualNorm not a Number: %T", residualNormVal)
	}

	expectedCoeffs := []float64{2.0 / 3.0, 5.0 / 3.0}
	expectedProjection := []float64{2.0 / 3.0, 5.0 / 3.0, 7.0 / 3.0}
	expectedResidual := []float64{1.0 / 3.0, 1.0 / 3.0, -1.0 / 3.0}
	expectedResidualNorm := math.Sqrt(1.0 / 3.0)

	assertFloatSliceApprox(t, coeffs, expectedCoeffs, 1e-9)
	assertFloatSliceApprox(t, projection, expectedProjection, 1e-9)
	assertFloatSliceApprox(t, residual, expectedResidual, 1e-9)

	if math.Abs(float64(residualNorm)-expectedResidualNorm) > 1e-9 {
		t.Fatalf("unexpected residual norm: got %.9f want %.9f", residualNorm, expectedResidualNorm)
	}
}

func TestLeastSquaresDimensionMismatch(t *testing.T) {
	rt := NewRuntime()
	RegisterMath(rt)

	matrix := NewArray()
	matrix.Append(arrayFromNumbers(1, 2))

	vector := arrayFromNumbers(1, 2, 3)

	if _, err := rt.funcs["lsp"](matrix, vector); err == nil {
		t.Fatalf("expected error for mismatched dimensions")
	}
}

func TestValueToMatrixShape(t *testing.T) {
	matrix := NewArray()
	matrix.Append(arrayFromNumbers(1, 0))
	matrix.Append(arrayFromNumbers(0, 1))
	matrix.Append(arrayFromNumbers(1, 1))

	parsed, err := valueToMatrix(matrix)
	if err != nil {
		t.Fatalf("valueToMatrix returned error: %v", err)
	}
	if len(parsed) != 3 {
		t.Fatalf("expected 3 rows, got %d", len(parsed))
	}
	for i, row := range parsed {
		if len(row) != 2 {
			t.Fatalf("row %d expected 2 columns, got %d", i, len(row))
		}
	}
}

func TestMatrixTranspose(t *testing.T) {
	input := [][]float64{{1, 2, 3}, {4, 5, 6}}
	got := matrixTranspose(input)
	expected := [][]float64{{1, 4}, {2, 5}, {3, 6}}
	if len(got) != len(expected) {
		t.Fatalf("expected %d rows, got %d", len(expected), len(got))
	}
	for i := range expected {
		for j := range expected[i] {
			if got[i][j] != expected[i][j] {
				t.Fatalf("transpose mismatch at (%d,%d): got %.1f want %.1f", i, j, got[i][j], expected[i][j])
			}
		}
	}
}

func TestMatrixMultiply(t *testing.T) {
	left := [][]float64{{1, 2}, {3, 4}}
	right := [][]float64{{2, 0}, {1, 2}}
	product, err := matrixMultiply(left, right)
	if err != nil {
		t.Fatalf("matrixMultiply returned error: %v", err)
	}
	expected := [][]float64{{4, 4}, {10, 8}}
	for i := range expected {
		for j := range expected[i] {
			if product[i][j] != expected[i][j] {
				t.Fatalf("product mismatch at (%d,%d): got %.1f want %.1f", i, j, product[i][j], expected[i][j])
			}
		}
	}
}

func TestMatrixMultiplyDimensionMismatch(t *testing.T) {
	left := [][]float64{{1, 2, 3}}
	right := [][]float64{{1, 2}, {3, 4}}
	if _, err := matrixMultiply(left, right); err == nil {
		t.Fatalf("expected error for dimension mismatch")
	}
}

func TestSolveLinearSystem(t *testing.T) {
	matrix := [][]float64{{2, 1}, {1, 1}}
	vector := []float64{5, 3}
	solution, err := solveLinearSystem(matrix, vector)
	if err != nil {
		t.Fatalf("solveLinearSystem returned error: %v", err)
	}
	expected := []float64{2, 1}
	for i := range expected {
		if math.Abs(solution[i]-expected[i]) > 1e-9 {
			t.Fatalf("solution mismatch at %d: got %.9f want %.9f", i, solution[i], expected[i])
		}
	}
}

func arrayFromNumbers(vals ...float64) *ArrayValue {
	arr := &ArrayValue{}
	for _, v := range vals {
		arr.Append(Number(v))
	}
	return arr
}

func extractFloatSlice(t *testing.T, result *MapValue, key string) []float64 {
	t.Helper()
	val, ok := result.Get(key)
	if !ok {
		t.Fatalf("missing key %s", key)
	}
	arr, ok := val.(*ArrayValue)
	if !ok {
		t.Fatalf("%s not an ArrayValue: %T", key, val)
	}
	slice := make([]float64, len(arr.Elements))
	for i, elem := range arr.Elements {
		num, ok := elem.(Number)
		if !ok {
			t.Fatalf("%s[%d] not a Number: %T", key, i, elem)
		}
		slice[i] = float64(num)
	}
	return slice
}

func assertFloatSliceApprox(t *testing.T, actual, expected []float64, tol float64) {
	t.Helper()
	if len(actual) != len(expected) {
		t.Fatalf("slice length mismatch: got %d want %d", len(actual), len(expected))
	}
	for i := range actual {
		if math.Abs(actual[i]-expected[i]) > tol {
			t.Fatalf("slice mismatch at %d: got %.9f want %.9f", i, actual[i], expected[i])
		}
	}
}
