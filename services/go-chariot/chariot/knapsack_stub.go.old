//go:build !cgo || (!linux && !darwin) || (linux && !amd64 && !arm64) || (linux && arm64 && !cuda)

package chariot

import "errors"

// SolveKnapsack stub implementation for unsupported platforms.
// Real signature: SolveKnapsack(configJSON string, optionsJSON string) (*V2Solution, error)
func SolveKnapsack(configJSON string, optionsJSON string) (*V2Solution, error) {
	return nil, errors.New("knapsack solver not available on this platform - requires CGO and specific platform support")
}
