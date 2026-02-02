package chariot

import (
	"errors"
	"fmt"
	"math"
	"strconv"

	"gonum.org/v1/gonum/lapack"
	gonumlapack "gonum.org/v1/gonum/lapack/gonum"
	"gonum.org/v1/gonum/mat"
)

const (
	leastSquaresPivotTolerance = 1e-10
	eigenSymmetryTolerance     = 1e-9
	schurZeroTolerance         = 1e-12
)

// resolveValue unwraps scope entries to their underlying values.
func resolveValue(val Value) Value {
	if entry, ok := val.(ScopeEntry); ok {
		return entry.Value
	}
	return val
}

// ensureArrayValue normalizes a Value into an *ArrayValue if possible.
func ensureArrayValue(val Value) (*ArrayValue, error) {
	val = resolveValue(val)
	switch v := val.(type) {
	case *ArrayValue:
		return v, nil
	case ArrayValue:
		copyElems := make([]Value, len(v.Elements))
		copy(copyElems, v.Elements)
		return &ArrayValue{Elements: copyElems}, nil
	default:
		return nil, fmt.Errorf("expected array value, got %T", val)
	}
}

// valueToFloat64 converts a Value into a float64 if possible.
func valueToFloat64(val Value) (float64, error) {
	val = resolveValue(val)
	switch v := val.(type) {
	case Number:
		return float64(v), nil
	case Bool:
		if bool(v) {
			return 1, nil
		}
		return 0, nil
	case Str:
		parsed, err := strconv.ParseFloat(string(v), 64)
		if err != nil {
			return 0, fmt.Errorf("expected numeric string, got %q", v)
		}
		return parsed, nil
	default:
		return 0, fmt.Errorf("expected number, got %T", val)
	}
}

// valueToVector converts an array-like Value into a float64 slice.
func valueToVector(val Value) ([]float64, error) {
	arr, err := ensureArrayValue(val)
	if err != nil {
		return nil, err
	}
	if len(arr.Elements) == 0 {
		return nil, errors.New("vector cannot be empty")
	}
	vector := make([]float64, len(arr.Elements))
	for i, elem := range arr.Elements {
		num, err := valueToFloat64(elem)
		if err != nil {
			return nil, fmt.Errorf("vector element %d: %w", i, err)
		}
		vector[i] = num
	}
	return vector, nil
}

// valueToMatrix converts a nested array Value into a [][]float64 matrix.
func valueToMatrix(val Value) ([][]float64, error) {
	outer, err := ensureArrayValue(val)
	if err != nil {
		return nil, err
	}
	if len(outer.Elements) == 0 {
		return nil, errors.New("matrix cannot be empty")
	}

	matrix := make([][]float64, len(outer.Elements))
	expectedCols := -1
	for i, rowVal := range outer.Elements {
		rowArr, err := ensureArrayValue(rowVal)
		if err != nil {
			return nil, fmt.Errorf("matrix row %d: %w", i, err)
		}
		if len(rowArr.Elements) == 0 {
			return nil, fmt.Errorf("matrix row %d cannot be empty", i)
		}
		if expectedCols == -1 {
			expectedCols = len(rowArr.Elements)
		} else if len(rowArr.Elements) != expectedCols {
			return nil, errors.New("all matrix rows must have the same length")
		}
		row := make([]float64, expectedCols)
		for j, cell := range rowArr.Elements {
			num, err := valueToFloat64(cell)
			if err != nil {
				return nil, fmt.Errorf("matrix element (%d,%d): %w", i, j, err)
			}
			row[j] = num
		}
		matrix[i] = row
	}

	return matrix, nil
}

// floatsToArrayValue converts a float slice back into an ArrayValue.
func floatsToArrayValue(data []float64) *ArrayValue {
	arr := &ArrayValue{}
	for _, v := range data {
		arr.Append(Number(v))
	}
	return arr
}

// matrixToArrayValue converts a matrix into a nested ArrayValue.
func matrixToArrayValue(matrix [][]float64) *ArrayValue {
	outer := &ArrayValue{}
	for _, row := range matrix {
		rowArr := &ArrayValue{}
		for _, v := range row {
			rowArr.Append(Number(v))
		}
		outer.Append(rowArr)
	}
	return outer
}

func matrixToColMajor(matrix [][]float64) []float64 {
	size := len(matrix)
	if size == 0 {
		return nil
	}
	data := make([]float64, size*size)
	for r := 0; r < size; r++ {
		for c := 0; c < size; c++ {
			data[c*size+r] = matrix[r][c]
		}
	}
	return data
}

func colMajorToMatrix(data []float64, size int) [][]float64 {
	matrix := make([][]float64, size)
	for r := 0; r < size; r++ {
		matrix[r] = make([]float64, size)
		for c := 0; c < size; c++ {
			matrix[r][c] = data[c*size+r]
		}
	}
	return matrix
}

func intsToArrayValue(data []int) *ArrayValue {
	arr := &ArrayValue{}
	for _, v := range data {
		arr.Append(Number(v))
	}
	return arr
}

// leastSquaresProjection solves the normal equations and returns coefficients, projection, residual, and its norm.
func leastSquaresProjection(matrix [][]float64, vector []float64) ([]float64, []float64, []float64, float64, error) {
	rows := len(matrix)
	if rows == 0 {
		return nil, nil, nil, 0, errors.New("matrix cannot be empty")
	}
	cols := len(matrix[0])
	if cols == 0 {
		return nil, nil, nil, 0, errors.New("matrix must have at least one column")
	}
	for i, row := range matrix {
		if len(row) != cols {
			return nil, nil, nil, 0, fmt.Errorf("matrix row %d expected %d columns, got %d", i, cols, len(row))
		}
	}
	if len(vector) != rows {
		return nil, nil, nil, 0, fmt.Errorf("vector length %d does not match matrix rows %d", len(vector), rows)
	}

	ata := buildNormalMatrix(matrix)
	atb := buildNormalVector(matrix, vector)

	coeffs, err := gaussianSolve(ata, atb)
	if err != nil {
		return nil, nil, nil, 0, err
	}

	projection := make([]float64, rows)
	for i := 0; i < rows; i++ {
		sum := 0.0
		for j := 0; j < cols; j++ {
			sum += matrix[i][j] * coeffs[j]
		}
		projection[i] = sum
	}

	residual := make([]float64, rows)
	var residualNorm float64
	for i := range vector {
		residual[i] = vector[i] - projection[i]
		residualNorm += residual[i] * residual[i]
	}

	return coeffs, projection, residual, math.Sqrt(residualNorm), nil
}

func buildNormalMatrix(matrix [][]float64) [][]float64 {
	rows := len(matrix)
	cols := len(matrix[0])
	ata := make([][]float64, cols)
	for i := 0; i < cols; i++ {
		ata[i] = make([]float64, cols)
	}
	for i := 0; i < cols; i++ {
		for j := i; j < cols; j++ {
			sum := 0.0
			for k := 0; k < rows; k++ {
				sum += matrix[k][i] * matrix[k][j]
			}
			ata[i][j] = sum
			ata[j][i] = sum
		}
	}
	return ata
}

func buildNormalVector(matrix [][]float64, vector []float64) []float64 {
	cols := len(matrix[0])
	atb := make([]float64, cols)
	for i := 0; i < cols; i++ {
		sum := 0.0
		for k := 0; k < len(matrix); k++ {
			sum += matrix[k][i] * vector[k]
		}
		atb[i] = sum
	}
	return atb
}

// gaussianSolve solves a linear system using Gauss-Jordan elimination with partial pivoting.
func gaussianSolve(matrix [][]float64, vector []float64) ([]float64, error) {
	n := len(matrix)
	if n == 0 {
		return nil, errors.New("matrix cannot be empty")
	}
	if len(vector) != n {
		return nil, errors.New("vector length must match matrix dimensions")
	}

	work := make([][]float64, n)
	for i := 0; i < n; i++ {
		if len(matrix[i]) != n {
			return nil, errors.New("matrix must be square")
		}
		rowCopy := make([]float64, n)
		copy(rowCopy, matrix[i])
		work[i] = rowCopy
	}
	rhs := make([]float64, n)
	copy(rhs, vector)

	for i := 0; i < n; i++ {
		pivot := i
		maxVal := math.Abs(work[i][i])
		for r := i + 1; r < n; r++ {
			if val := math.Abs(work[r][i]); val > maxVal {
				pivot = r
				maxVal = val
			}
		}
		if maxVal < leastSquaresPivotTolerance {
			return nil, errors.New("matrix is singular or rank-deficient")
		}
		if pivot != i {
			work[i], work[pivot] = work[pivot], work[i]
			rhs[i], rhs[pivot] = rhs[pivot], rhs[i]
		}

		pivotVal := work[i][i]
		for c := i; c < n; c++ {
			work[i][c] /= pivotVal
		}
		rhs[i] /= pivotVal

		for r := 0; r < n; r++ {
			if r == i {
				continue
			}
			factor := work[r][i]
			if factor == 0 {
				continue
			}
			for c := i; c < n; c++ {
				work[r][c] -= factor * work[i][c]
			}
			rhs[r] -= factor * rhs[i]
		}
	}

	return rhs, nil
}

// matrixTranspose transposes the provided matrix.
func matrixTranspose(matrix [][]float64) [][]float64 {
	if len(matrix) == 0 {
		return [][]float64{}
	}
	rows := len(matrix)
	cols := len(matrix[0])
	result := make([][]float64, cols)
	for i := 0; i < cols; i++ {
		result[i] = make([]float64, rows)
		for j := 0; j < rows; j++ {
			result[i][j] = matrix[j][i]
		}
	}
	return result
}

// matrixMultiply multiplies two matrices when their dimensions align.
func matrixMultiply(a, b [][]float64) ([][]float64, error) {
	if len(a) == 0 || len(b) == 0 {
		return nil, errors.New("matrices cannot be empty")
	}
	inner := len(a[0])
	if inner == 0 {
		return nil, errors.New("matrix must have at least one column")
	}
	for i, row := range a {
		if len(row) != inner {
			return nil, fmt.Errorf("left matrix row %d expected %d columns, got %d", i, inner, len(row))
		}
	}
	rowsB := len(b)
	colsB := len(b[0])
	if rowsB != inner {
		return nil, fmt.Errorf("dimension mismatch: left matrix columns %d != right matrix rows %d", inner, rowsB)
	}
	for i, row := range b {
		if len(row) != colsB {
			return nil, fmt.Errorf("right matrix row %d expected %d columns, got %d", i, colsB, len(row))
		}
	}
	result := make([][]float64, len(a))
	for i := range result {
		result[i] = make([]float64, colsB)
		for j := 0; j < colsB; j++ {
			sum := 0.0
			for k := 0; k < inner; k++ {
				sum += a[i][k] * b[k][j]
			}
			result[i][j] = sum
		}
	}
	return result, nil
}

// solveLinearSystem solves a square system A*x = b.
func solveLinearSystem(matrix [][]float64, vector []float64) ([]float64, error) {
	size := len(matrix)
	if size == 0 {
		return nil, errors.New("matrix cannot be empty")
	}
	for i, row := range matrix {
		if len(row) != size {
			return nil, fmt.Errorf("matrix row %d expected %d columns, got %d", i, size, len(row))
		}
	}
	if len(vector) != size {
		return nil, fmt.Errorf("vector length %d does not match matrix size %d", len(vector), size)
	}
	return gaussianSolve(matrix, vector)
}

// vectorScale scales a vector by a scalar value.
func vectorScale(vector *[]float64, scalar float64) []float64 {
	result := make([]float64, len(*vector))
	for i, v := range *vector {
		result[i] = v * scalar
	}
	return result
}

// eigenSymmetricDecomposition returns eigenvalues and eigenvectors for symmetric matrices.
func eigenSymmetricDecomposition(matrix [][]float64) ([]float64, [][]float64, error) {
	size := len(matrix)
	if size == 0 {
		return nil, nil, errors.New("matrix cannot be empty")
	}
	for i := range matrix {
		if len(matrix[i]) != size {
			return nil, nil, errors.New("matrix must be square for eigen decomposition")
		}
	}
	for i := 0; i < size; i++ {
		for j := i + 1; j < size; j++ {
			if math.Abs(matrix[i][j]-matrix[j][i]) > eigenSymmetryTolerance {
				return nil, nil, fmt.Errorf("matrix must be symmetric within tolerance %.1e", eigenSymmetryTolerance)
			}
		}
	}

	sym := mat.NewSymDense(size, nil)
	for i := 0; i < size; i++ {
		for j := 0; j <= i; j++ {
			sym.SetSym(i, j, matrix[i][j])
		}
	}

	var eig mat.EigenSym
	if ok := eig.Factorize(sym, true); !ok {
		return nil, nil, errors.New("failed to factorize symmetric matrix")
	}

	values := eig.Values(nil)
	var vec mat.Dense
	eig.VectorsTo(&vec)

	rows, cols := vec.Dims()
	vectors := make([][]float64, rows)
	for i := 0; i < rows; i++ {
		vectors[i] = make([]float64, cols)
		for j := 0; j < cols; j++ {
			vectors[i][j] = vec.At(i, j)
		}
	}

	return values, vectors, nil
}

// eigenGeneralDecomposition returns complex eigenvalues and vectors for general matrices.
func eigenGeneralDecomposition(matrix [][]float64) ([]float64, []float64, [][]float64, [][]float64, error) {
	size := len(matrix)
	if size == 0 {
		return nil, nil, nil, nil, errors.New("matrix cannot be empty")
	}
	for i := range matrix {
		if len(matrix[i]) != size {
			return nil, nil, nil, nil, errors.New("matrix must be square for eigen decomposition")
		}
	}

	data := make([]float64, size*size)
	for i := 0; i < size; i++ {
		copy(data[i*size:(i+1)*size], matrix[i])
	}
	dense := mat.NewDense(size, size, data)

	var eig mat.Eigen
	if ok := eig.Factorize(dense, mat.EigenRight); !ok {
		return nil, nil, nil, nil, errors.New("failed to factorize matrix")
	}

	complexVals := eig.Values(nil)
	realVals := make([]float64, len(complexVals))
	imagVals := make([]float64, len(complexVals))
	for i, val := range complexVals {
		realVals[i] = real(val)
		imagVals[i] = imag(val)
	}
	var vectors mat.CDense
	eig.VectorsTo(&vectors)

	rows, cols := vectors.Dims()
	realVectors := make([][]float64, rows)
	imagVectors := make([][]float64, rows)
	for i := 0; i < rows; i++ {
		realVectors[i] = make([]float64, cols)
		imagVectors[i] = make([]float64, cols)
		for j := 0; j < cols; j++ {
			val := vectors.At(i, j)
			realVectors[i][j] = real(val)
			imagVectors[i][j] = imag(val)
		}
	}

	return realVals, imagVals, realVectors, imagVectors, nil
}

// dominantEigenPair computes the dominant eigenvalue/vector via the power iteration method.
func dominantEigenPair(matrix [][]float64, tolerance float64, maxIterations int) (float64, []float64, error) {
	size := len(matrix)
	if size == 0 {
		return 0, nil, errors.New("matrix cannot be empty")
	}
	for i := range matrix {
		if len(matrix[i]) != size {
			return 0, nil, errors.New("matrix must be square for power iteration")
		}
	}
	if tolerance <= 0 {
		return 0, nil, errors.New("tolerance must be positive")
	}
	if maxIterations <= 0 {
		return 0, nil, errors.New("maxIterations must be positive")
	}

	vector := make([]float64, size)
	for i := range vector {
		vector[i] = 1 / math.Sqrt(float64(size))
	}

	prevEigen := 0.0
	for iter := 0; iter < maxIterations; iter++ {
		next := make([]float64, size)
		for i := 0; i < size; i++ {
			sum := 0.0
			for j := 0; j < size; j++ {
				sum += matrix[i][j] * vector[j]
			}
			next[i] = sum
		}

		norm := math.Sqrt(dotProduct(next, next))
		if norm == 0 {
			return 0, nil, errors.New("encountered zero vector during iteration")
		}
		for i := range next {
			next[i] /= norm
		}

		eigen := rayleighQuotient(matrix, next)
		if math.Abs(eigen-prevEigen) < tolerance {
			return eigen, next, nil
		}
		prevEigen = eigen
		vector = next
	}

	return prevEigen, vector, fmt.Errorf("power iteration did not converge within %d iterations", maxIterations)
}

func dotProduct(a, b []float64) float64 {
	sum := 0.0
	for i := range a {
		sum += a[i] * b[i]
	}
	return sum
}

func dotProductStrict(a, b []float64) (float64, error) {
	if len(a) == 0 || len(b) == 0 {
		return 0, errors.New("vectors cannot be empty")
	}
	if len(a) != len(b) {
		return 0, fmt.Errorf("vector length mismatch: %d vs %d", len(a), len(b))
	}
	return dotProduct(a, b), nil
}

func rayleighQuotient(matrix [][]float64, vector []float64) float64 {
	numerator := 0.0
	denom := dotProduct(vector, vector)
	for i := 0; i < len(vector); i++ {
		rowSum := 0.0
		for j := 0; j < len(vector); j++ {
			rowSum += matrix[i][j] * vector[j]
		}
		numerator += vector[i] * rowSum
	}
	if denom == 0 {
		return 0
	}
	return numerator / denom
}

/* denseToMatrix converts a mat.Matrix to a [][]float64 matrix. Currently unused
func denseToMatrix(m mat.Matrix) [][]float64 {
	rows, cols := m.Dims()
	result := make([][]float64, rows)
	for i := 0; i < rows; i++ {
		result[i] = make([]float64, cols)
		for j := 0; j < cols; j++ {
			result[i][j] = m.At(i, j)
		}
	}
	return result
}
*/

func detectSchurBlocks(tMatrix [][]float64) []int {
	blocks := []int{}
	size := len(tMatrix)
	i := 0
	for i < size {
		if i < size-1 && math.Abs(tMatrix[i+1][i]) > schurZeroTolerance {
			blocks = append(blocks, 2)
			i += 2
			continue
		}
		blocks = append(blocks, 1)
		i++
	}
	return blocks
}

// realSchurDecomposition computes the real Schur form using LAPACK routines.
func realSchurDecomposition(matrix [][]float64) ([][]float64, [][]float64, []float64, []float64, []int, error) {
	size := len(matrix)
	if size == 0 {
		return nil, nil, nil, nil, nil, errors.New("matrix cannot be empty")
	}
	for i := range matrix {
		if len(matrix[i]) != size {
			return nil, nil, nil, nil, nil, errors.New("matrix must be square for Schur decomposition")
		}
	}
	if size == 1 {
		return [][]float64{{matrix[0][0]}}, [][]float64{{1}}, []float64{matrix[0][0]}, []float64{0}, []int{1}, nil
	}
	impl := gonumlapack.Implementation{}
	colMajor := matrixToColMajor(matrix)
	ilo, ihi := 0, size-1
	tau := make([]float64, size-1)
	work := make([]float64, 1)
	impl.Dgehrd(size, ilo, ihi, colMajor, size, tau, work, -1)
	lwork := int(math.Max(1, work[0]))
	work = make([]float64, lwork)
	impl.Dgehrd(size, ilo, ihi, colMajor, size, tau, work, lwork)
	hessenberg := make([]float64, len(colMajor))
	copy(hessenberg, colMajor)
	schurVecs := make([]float64, len(colMajor))
	copy(schurVecs, colMajor)
	work = make([]float64, 1)
	impl.Dorghr(size, ilo, ihi, schurVecs, size, tau, work, -1)
	lwork = int(math.Max(1, work[0]))
	work = make([]float64, lwork)
	impl.Dorghr(size, ilo, ihi, schurVecs, size, tau, work, lwork)
	wr := make([]float64, size)
	wi := make([]float64, size)
	work = make([]float64, 1)
	impl.Dhseqr(lapack.EigenvaluesAndSchur, lapack.SchurOrig, size, ilo, ihi, hessenberg, size, wr, wi, schurVecs, size, work, -1)
	lwork = int(math.Max(float64(size), work[0]))
	work = make([]float64, lwork)
	if unconverged := impl.Dhseqr(lapack.EigenvaluesAndSchur, lapack.SchurOrig, size, ilo, ihi, hessenberg, size, wr, wi, schurVecs, size, work, lwork); unconverged != 0 {
		return nil, nil, nil, nil, nil, fmt.Errorf("schur decomposition failed after %d unconverged eigenvalues", unconverged)
	}
	tMatrix := colMajorToMatrix(hessenberg, size)
	qMatrix := colMajorToMatrix(schurVecs, size)
	realVals := make([]float64, size)
	imagVals := make([]float64, size)
	copy(realVals, wr)
	copy(imagVals, wi)
	blocks := detectSchurBlocks(tMatrix)
	return tMatrix, qMatrix, realVals, imagVals, blocks, nil
}
