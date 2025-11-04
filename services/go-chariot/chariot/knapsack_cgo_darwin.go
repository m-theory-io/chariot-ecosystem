//go:build darwin && arm64

package chariot

/*
#cgo CFLAGS: -I/usr/local/include
#cgo LDFLAGS: -L/usr/local/lib -lknapsack -framework Metal -framework Foundation -lc++
#include <stdlib.h>
#include "knapsack_c.h"
*/
import "C"

import (
	"errors"
	"unsafe"
)

// SolveKnapsack is the V2 entry point for macOS Apple Silicon.
//
// Provide a V2 config JSON string (see knapsack/docs/v2/) and an optional options JSON.
// Example options:
//
//	{"beam_width":32,"iters":5,"seed":42,"debug":true,
//	 "dom_enable":true,"dom_eps":1e-9,"dom_surrogate":true}
//
// GPU note: The library attempts to compile a Metal shader at runtime. If unavailable,
// it falls back to a CPU evaluator automatically.
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
		// Header prototype: int* ks_v2_select_ptr(KnapsackSolutionV2* sol)
		ptr := C.ks_v2_select_ptr(out)
		if ptr != nil {
			slice := (*[1 << 30]C.int)(unsafe.Pointer(ptr))[:n:n]
			for i := 0; i < n; i++ {
				sel[i] = int(slice[i])
			}
		}
	}

	return &V2Solution{
		NumItems:  n,
		Select:    sel,
		Objective: float64(out.objective),
		Penalty:   float64(out.penalty),
		Total:     float64(out.total),
	}, nil
}
