//go:build linux && cuda && cgo

package chariot

/*
#cgo CFLAGS: -I${SRCDIR}/../knapsack-library/lib/linux-cuda
#cgo LDFLAGS: -Wl,--start-group ${SRCDIR}/../knapsack-library/lib/linux-cuda/librl_support.a -lstdc++ -Wl,--end-group -lm -lonnxruntime

#include <stdlib.h>
#include "rl_api.h"
*/
import "C"
import (
	"encoding/json"
	"errors"
	"fmt"
	"unsafe"
)

// rlInit_impl is the Linux CUDA implementation of rlInit
// It wraps rl_init_from_json from librl_support.a (CUDA variant)
func rlInit_impl(configJSON string) (interface{}, error) {
	if configJSON == "" {
		return nil, errors.New("rlInit: empty config JSON")
	}

	// Validate JSON structure
	var cfg RLConfig
	if err := json.Unmarshal([]byte(configJSON), &cfg); err != nil {
		return nil, fmt.Errorf("rlInit: invalid JSON: %v", err)
	}

	// Validate required fields
	if cfg.FeatDim <= 0 {
		return nil, errors.New("rlInit: feat_dim must be > 0")
	}
	if cfg.Alpha <= 0 {
		return nil, errors.New("rlInit: alpha must be > 0")
	}

	cCfg := C.CString(configJSON)
	defer C.free(unsafe.Pointer(cCfg))

	// Error buffer for C API
	const errLen = 512
	errBuf := make([]byte, errLen)
	cErr := (*C.char)(unsafe.Pointer(&errBuf[0]))

	handle := C.rl_init_from_json(cCfg, cErr, C.int(errLen))
	if handle == nil {
		errMsg := C.GoString(cErr)
		if errMsg == "" {
			errMsg = "rl_init_from_json failed (unknown error)"
		}
		return nil, fmt.Errorf("rlInit: %s", errMsg)
	}

	return handle, nil
}

// rlScore_impl is the Linux CUDA implementation of rlScore
// It wraps rl_score_batch_with_features from librl_support.a
func rlScore_impl(handle interface{}, features []float64, featDim int) ([]float64, error) {
	if handle == nil {
		return nil, errors.New("rlScore: nil handle")
	}

	h, ok := handle.(C.rl_handle_t)
	if !ok {
		return nil, fmt.Errorf("rlScore: invalid handle type %T", handle)
	}

	if len(features) == 0 {
		return nil, errors.New("rlScore: empty features array")
	}

	if featDim <= 0 {
		return nil, errors.New("rlScore: featDim must be > 0")
	}

	numCandidates := len(features) / featDim
	if len(features)%featDim != 0 {
		return nil, fmt.Errorf("rlScore: features length %d not divisible by featDim %d", len(features), featDim)
	}

	// Convert features to C float array
	cFeatures := make([]C.float, len(features))
	for i, f := range features {
		cFeatures[i] = C.float(f)
	}

	// Allocate output scores array
	scores := make([]C.double, numCandidates)

	// Error buffer
	const errLen = 512
	errBuf := make([]byte, errLen)
	cErr := (*C.char)(unsafe.Pointer(&errBuf[0]))

	rc := C.rl_score_batch_with_features(
		h,
		(*C.float)(unsafe.Pointer(&cFeatures[0])),
		C.int(featDim),
		C.int(numCandidates),
		(*C.double)(unsafe.Pointer(&scores[0])),
		cErr,
		C.int(errLen),
	)

	if rc != 0 {
		errMsg := C.GoString(cErr)
		if errMsg == "" {
			errMsg = "rl_score_batch_with_features failed (unknown error)"
		}
		return nil, fmt.Errorf("rlScore: %s", errMsg)
	}

	// Convert C scores to Go slice
	result := make([]float64, numCandidates)
	for i := 0; i < numCandidates; i++ {
		result[i] = float64(scores[i])
	}

	return result, nil
}

// rlLearn_impl is the Linux CUDA implementation of rlLearn
// It wraps rl_learn_batch from librl_support.a
func rlLearn_impl(handle interface{}, feedbackJSON string) error {
	if handle == nil {
		return errors.New("rlLearn: nil handle")
	}

	h, ok := handle.(C.rl_handle_t)
	if !ok {
		return fmt.Errorf("rlLearn: invalid handle type %T", handle)
	}

	if feedbackJSON == "" {
		return errors.New("rlLearn: empty feedback JSON")
	}

	// Validate JSON structure
	var feedback RLFeedback
	if err := json.Unmarshal([]byte(feedbackJSON), &feedback); err != nil {
		return fmt.Errorf("rlLearn: invalid JSON: %v", err)
	}

	cFeedback := C.CString(feedbackJSON)
	defer C.free(unsafe.Pointer(cFeedback))

	// Error buffer
	const errLen = 512
	errBuf := make([]byte, errLen)
	cErr := (*C.char)(unsafe.Pointer(&errBuf[0]))

	rc := C.rl_learn_batch(h, cFeedback, cErr, C.int(errLen))
	if rc != 0 {
		errMsg := C.GoString(cErr)
		if errMsg == "" {
			errMsg = "rl_learn_batch failed (unknown error)"
		}
		return fmt.Errorf("rlLearn: %s", errMsg)
	}

	return nil
}

// rlClose_impl is the Linux CUDA implementation of rlClose
// It wraps rl_close from librl_support.a
func rlClose_impl(handle interface{}) {
	if handle == nil {
		return
	}

	h, ok := handle.(C.rl_handle_t)
	if !ok {
		return
	}

	C.rl_close(h)
}
