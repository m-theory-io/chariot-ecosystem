package chariot

import (
	"math"
	"math/cmplx"
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

func TestVectorScaleHelper(t *testing.T) {
	original := []float64{1, -2, 3}
	vec := make([]float64, len(original))
	copy(vec, original)
	scaled := vectorScale(&vec, 2.5)
	expected := []float64{2.5, -5, 7.5}
	if len(scaled) != len(expected) {
		t.Fatalf("expected %d entries, got %d", len(expected), len(scaled))
	}
	for i := range expected {
		if math.Abs(scaled[i]-expected[i]) > 1e-9 {
			t.Fatalf("scaled mismatch at %d: got %.9f want %.9f", i, scaled[i], expected[i])
		}
		if vec[i] != original[i] {
			t.Fatalf("vectorScale mutated input slice at %d: got %.9f", i, vec[i])
		}
	}
}

func TestVectorScaleZeroScalar(t *testing.T) {
	vec := []float64{10, -4.5}
	scaled := vectorScale(&vec, 0)
	expected := []float64{0, 0}
	for i := range expected {
		if math.Abs(scaled[i]-expected[i]) > 1e-9 {
			t.Fatalf("zero scalar mismatch at %d: got %.9f", i, scaled[i])
		}
	}
}

func TestDotProductHelper(t *testing.T) {
	a := []float64{1, 2, 3}
	b := []float64{4, -5, 6}
	value, err := dotProductStrict(a, b)
	if err != nil {
		t.Fatalf("dotProduct returned error: %v", err)
	}
	assertApprox(t, value, 12, 1e-9)
}

func TestDotProductHelperLengthMismatch(t *testing.T) {
	_, err := dotProductStrict([]float64{1, 2}, []float64{3})
	if err == nil {
		t.Fatalf("expected error for mismatched lengths")
	}
}

func TestDotProductClosure(t *testing.T) {
	rt := NewRuntime()
	RegisterMath(rt)
	vecA := arrayFromNumbers(1, 3, -2)
	vecB := arrayFromNumbers(4, 0.5, 10)
	val, err := rt.funcs["dotProduct"](vecA, vecB)
	if err != nil {
		t.Fatalf("dotProduct closure returned error: %v", err)
	}
	num, ok := val.(Number)
	if !ok {
		t.Fatalf("expected Number result, got %T", val)
	}
	assertApprox(t, float64(num), 4*1+0.5*3+10*(-2), 1e-9)
}

func TestEigenSymmetricDecomposition(t *testing.T) {
	matrix := [][]float64{{2, 1}, {1, 2}}
	values, vectors, err := eigenSymmetricDecomposition(matrix)
	if err != nil {
		t.Fatalf("eigenSymmetricDecomposition returned error: %v", err)
	}
	if len(values) != 2 || len(vectors) != 2 {
		t.Fatalf("unexpected eigen dimensions")
	}
	assertApprox(t, values[0], 1, 1e-9)
	assertApprox(t, values[1], 3, 1e-9)
	col0 := []float64{vectors[0][0], vectors[1][0]}
	col1 := []float64{vectors[0][1], vectors[1][1]}
	verifyEigenRelation(t, matrix, values[0], col0)
	verifyEigenRelation(t, matrix, values[1], col1)
}

func TestEigenGeneralDecompositionComplex(t *testing.T) {
	matrix := [][]float64{{0, -1}, {1, 0}}
	realVals, imagVals, realVecs, imagVecs, err := eigenGeneralDecomposition(matrix)
	if err != nil {
		t.Fatalf("eigenGeneralDecomposition returned error: %v", err)
	}
	if len(realVals) != 2 || len(imagVals) != 2 {
		t.Fatalf("expected two eigenvalues")
	}
	pairs := make([]complex128, len(realVals))
	for i := range realVals {
		pairs[i] = complex(realVals[i], imagVals[i])
	}
	if !containsComplexPair(pairs, complex(0, 1)) || !containsComplexPair(pairs, complex(0, -1)) {
		t.Fatalf("missing expected complex eigenvalues: %v", pairs)
	}
	for idx := range realVals {
		vector := make([]complex128, len(realVecs))
		for row := range realVecs {
			vector[row] = complex(realVecs[row][idx], imagVecs[row][idx])
		}
		if normComplex(matrixVectorDiff(matrix, pairs[idx], vector)) > 1e-6 {
			t.Fatalf("eigenvector %d does not satisfy Av=lambda v", idx)
		}
	}
}

func TestDominantEigenPair(t *testing.T) {
	matrix := [][]float64{{4, 1}, {0, 2}}
	value, vector, err := dominantEigenPair(matrix, 1e-9, 500)
	if err != nil {
		t.Fatalf("dominantEigenPair returned error: %v", err)
	}
	assertApprox(t, value, 4, 1e-6)
	if len(vector) != 2 {
		t.Fatalf("unexpected vector length: %d", len(vector))
	}
	if math.Abs(vector[1]) > 1e-3 {
		t.Fatalf("expected vector aligned with first axis, got %v", vector)
	}
}

func TestRealSchurDecomposition(t *testing.T) {
	matrix := [][]float64{{0, -1}, {1, 0}}
	T, Q, realVals, imagVals, blocks, err := realSchurDecomposition(matrix)
	if err != nil {
		t.Fatalf("realSchurDecomposition returned error: %v", err)
	}
	if len(blocks) != 1 || blocks[0] != 2 {
		t.Fatalf("expected single 2x2 block, got %v", blocks)
	}
	if !containsComplexPair(combineComplexPairs(realVals, imagVals), complex(0, 1)) {
		t.Fatalf("missing expected eigen pair")
	}
	qt, err := matrixMultiply(Q, T)
	if err != nil {
		t.Fatalf("matrixMultiply failed: %v", err)
	}
	reconstruct, err := matrixMultiply(qt, matrixTranspose(Q))
	if err != nil {
		t.Fatalf("matrixMultiply failed: %v", err)
	}
	assertMatrixApprox(t, reconstruct, matrix, 1e-9)
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

func assertApprox(t *testing.T, actual, expected, tol float64) {
	t.Helper()
	if math.Abs(actual-expected) > tol {
		t.Fatalf("value mismatch: got %.9f want %.9f", actual, expected)
	}
}

func verifyEigenRelation(t *testing.T, matrix [][]float64, eigenvalue float64, vector []float64) {
	t.Helper()
	if len(matrix) != len(vector) {
		t.Fatalf("dimension mismatch")
	}
	residual := 0.0
	for i := 0; i < len(matrix); i++ {
		sum := 0.0
		for j := 0; j < len(matrix); j++ {
			sum += matrix[i][j] * vector[j]
		}
		residual += math.Pow(sum-eigenvalue*vector[i], 2)
	}
	if math.Sqrt(residual) > 1e-6 {
		t.Fatalf("vector does not satisfy eigen relation: residual %.9f", math.Sqrt(residual))
	}
}

func containsComplexPair(values []complex128, target complex128) bool {
	for _, v := range values {
		if cmplx.Abs(v-target) < 1e-6 {
			return true
		}
	}
	return false
}

func matrixVectorDiff(matrix [][]float64, eigenvalue complex128, vector []complex128) []complex128 {
	result := make([]complex128, len(vector))
	for i := 0; i < len(matrix); i++ {
		sum := complex(0, 0)
		for j := 0; j < len(matrix); j++ {
			sum += complex(matrix[i][j], 0) * vector[j]
		}
		result[i] = sum - eigenvalue*vector[i]
	}
	return result
}

func normComplex(vec []complex128) float64 {
	sum := 0.0
	for _, v := range vec {
		sum += real(v)*real(v) + imag(v)*imag(v)
	}
	return math.Sqrt(sum)
}

func assertMatrixApprox(t *testing.T, actual, expected [][]float64, tol float64) {
	t.Helper()
	if len(actual) != len(expected) {
		t.Fatalf("row mismatch: got %d want %d", len(actual), len(expected))
	}
	for i := range actual {
		if len(actual[i]) != len(expected[i]) {
			t.Fatalf("column mismatch in row %d", i)
		}
		for j := range actual[i] {
			if math.Abs(actual[i][j]-expected[i][j]) > tol {
				t.Fatalf("matrix mismatch at (%d,%d): got %.9f want %.9f", i, j, actual[i][j], expected[i][j])
			}
		}
	}
}

func combineComplexPairs(realVals, imagVals []float64) []complex128 {
	if len(realVals) != len(imagVals) {
		return nil
	}
	result := make([]complex128, len(realVals))
	for i := range realVals {
		result[i] = complex(realVals[i], imagVals[i])
	}
	return result
}
