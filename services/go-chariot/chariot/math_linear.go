package chariot

import (
	"errors"
	"fmt"
	"math"
	"strconv"
)

const leastSquaresPivotTolerance = 1e-10

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
