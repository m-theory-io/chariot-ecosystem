//go:build linux && cuda

package chariot

/*
#cgo linux,cuda CFLAGS: -I/usr/local/include
#cgo linux,cuda LDFLAGS: -L/usr/local/lib -lknapsack -lstdc++
#cgo linux,cuda,amd64 LDFLAGS: -L/usr/local/cuda/lib64 -Wl,-rpath,/usr/local/cuda/lib64 -lcudart -lcuda
#cgo linux,cuda,arm64 LDFLAGS: -L/usr/local/cuda/lib64 -Wl,-rpath,/usr/local/cuda/lib64 -lcudart -lcuda
#include <stdlib.h>
#include "knapsack_c.h"
*/
import "C"

import (
	"errors"
	"unsafe"
)

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

	return &V2Solution{
		NumItems:  n,
		Select:    sel,
		Objective: float64(out.objective),
		Penalty:   float64(out.penalty),
		Total:     float64(out.total),
	}, nil
}
