//go:build !cgo
// +build !cgo

package chariot

import "errors"

// Stub implementation when knapsack CGO support is not built.
var ErrKnapsackUnavailable = errors.New("knapsack library not available (build with CGO_ENABLED=1 and libknapsack installed)")

// SolveKnapsack stub for non-cgo builds.
// Real signature: SolveKnapsack(configJSON string, optionsJSON string) (*V2Solution, error)
func SolveKnapsack(configJSON string, optionsJSON string) (*V2Solution, error) {
	return nil, ErrKnapsackUnavailable
}
