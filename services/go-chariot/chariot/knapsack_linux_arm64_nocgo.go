//go:build linux && arm64 && !cgo

package chariot

import "errors"

// SolveKnapsack is unavailable when CGO is disabled on linux/arm64 builds.
func SolveKnapsack(configJSON string, optionsJSON string) (*V2Solution, error) {
	return nil, errors.New("SolveKnapsack: linux/arm64 build requires cgo-enabled knapsack library")
}
