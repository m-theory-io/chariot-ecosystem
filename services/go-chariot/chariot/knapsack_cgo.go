//go:build knapsack && linux && arm64
// +build knapsack,linux,arm64

package chariot

/*
#cgo CXXFLAGS: -std=c++11 -I/home/nvidia/go/src/github.com/bhouse1273/knapsack
#cgo LDFLAGS: -L/usr/local/lib -L/usr/local/cuda-12.6/targets/aarch64-linux/lib -lknapsack -lstdc++ -lm -lcudart -lcuda -lcurand
#include <stdlib.h>
#include "/usr/local/include/knapsack_c.h"
*/
import "C"
import (
	"errors"
	"unsafe"
)

// KnapsackSolve wraps the C++ knapsack solver that takes a CSV file and team size
func KnapsackSolve(csvPath string, teamSize int) ([]int, error) {
	// Convert Go string to C string
	cCsvPath := C.CString(csvPath)
	defer C.free(unsafe.Pointer(cCsvPath))

	// Call the C function - assuming it returns the number of villages selected
	// and fills an output array with the selected village indices
	var resultSize C.int
	cResult := C.solve_knapsack(cCsvPath, C.int(teamSize))

	if cResult == nil {
		return nil, errors.New("knapsack solver failed")
	}
	defer C.free(unsafe.Pointer(cResult))

	// Copy result back to Go
	result := make([]int, int(resultSize))
	for i := 0; i < int(resultSize); i++ {
		result[i] = int(*(*C.int)(unsafe.Pointer(uintptr(unsafe.Pointer(cResult)) + uintptr(i)*unsafe.Sizeof(C.int(0)))))
	}

	return result, nil
}

// Alternative version if your C function has a different signature
func KnapsackSolveAlt(csvPath string, teamSize int) error {
	cCsvPath := C.CString(csvPath)
	defer C.free(unsafe.Pointer(cCsvPath))

	ret := C.solve_knapsack(cCsvPath, C.int(teamSize))
	if ret == nil {
		return errors.New("knapsack solver failed")
	}

	return nil
}
