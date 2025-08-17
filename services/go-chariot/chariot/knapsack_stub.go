//go:build !knapsack
// +build !knapsack

package chariot

import "fmt"

// Stub implementations when knapsack CGO support not built.
func KnapsackSolve(csvPath string, teamSize int) ([]int, error) {
	return nil, fmt.Errorf("knapsack support not built (missing 'knapsack' build tag)")
}

func KnapsackSolveAlt(csvPath string, teamSize int) error {
	return fmt.Errorf("knapsack support not built (missing 'knapsack' build tag)")
}
