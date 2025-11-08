//go:build linux && amd64 && cgo

package chariot

/*
#cgo CFLAGS: -I${SRCDIR}/../knapsack-library/lib/linux-cpu
#cgo CXXFLAGS: -I${SRCDIR}/../knapsack-library/lib/linux-cpu -std=c++14
#cgo LDFLAGS: -Wl,--start-group ${SRCDIR}/../knapsack-library/lib/linux-cpu/libknapsack_cpu.a -lstdc++ -Wl,--end-group -lm

#include <stdlib.h>
#include "knapsack_cpu.h"
*/
import "C"
import (
	"errors"
	"unsafe"
)

// SolveKnapsack is the V2 entry point for Linux AMD64 (CPU-only).
// It wraps the C API from libknapsack_cpu.a built without GPU support.
//
// configJSON: JSON string containing the V2 knapsack problem configuration
// optionsJSON: JSON string containing solver options (beam width, timeout, etc.)
//
// Returns: A pointer to V2Solution with chosen items and total value/weight, or an error.
func SolveKnapsack(configJSON string, optionsJSON string) (*V2Solution, error) {
	if configJSON == "" {
		return nil, errors.New("SolveKnapsack: empty V2 config JSON")
	}

	cCfg := C.CString(configJSON)
	defer C.free(unsafe.Pointer(cCfg))

	var cOpts *C.char
	if optionsJSON != "" {
		cOpts = C.CString(optionsJSON)
		defer C.free(unsafe.Pointer(cOpts))
	}

	var out *C.KnapsackSolutionV2
	rc := C.solve_knapsack_v2_from_json(cCfg, cOpts, &out)
	if rc != 0 || out == nil {
		return nil, errors.New("SolveKnapsack: solve_knapsack_v2_from_json failed")
	}
	defer C.free_knapsack_solution_v2(out)

	n := int(out.num_items)
	sel := make([]int, n)
	if n > 0 {
		ptr := C.ks_v2_select_ptr(out) // int* ks_v2_select_ptr(KnapsackSolutionV2* sol)
		if ptr != nil {
			slice := (*[1 << 30]C.int)(unsafe.Pointer(ptr))[:n:n]
			for i := 0; i < n; i++ {
				sel[i] = int(slice[i])
			}
		}
	}

	// Convert C result to Go V2Solution
	return &V2Solution{
		NumItems:  n,
		Select:    sel,
		Objective: float64(out.objective),
		Penalty:   float64(out.penalty),
		Total:     float64(out.total),
	}, nil
}
