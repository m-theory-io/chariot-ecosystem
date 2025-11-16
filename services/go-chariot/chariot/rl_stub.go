//go:build !cgo

package chariot

import "fmt"

// rlInit_impl is the stub implementation when CGO is disabled
func rlInit_impl(configJSON string) (interface{}, error) {
	return nil, fmt.Errorf("rlInit: RL support not available (CGO disabled)")
}

// rlScore_impl is the stub implementation when CGO is disabled
func rlScore_impl(handle interface{}, features []float64, featDim int) ([]float64, error) {
	return nil, fmt.Errorf("rlScore: RL support not available (CGO disabled)")
}

// rlLearn_impl is the stub implementation when CGO is disabled
func rlLearn_impl(handle interface{}, feedbackJSON string) error {
	return fmt.Errorf("rlLearn: RL support not available (CGO disabled)")
}

// rlClose_impl is the stub implementation when CGO is disabled
func rlClose_impl(handle interface{}) {
	// No-op for stub
}
