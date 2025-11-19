package tests

import (
	"testing"

	"github.com/bhouse1273/chariot-ecosystem/services/go-chariot/chariot"
)

// TestKnapsackConfig tests the knapsackConfig() helper function
func TestKnapsackConfig(t *testing.T) {
	tests := []TestCase{
		{
			Name: "Basic config generation returns string",
			Script: []string{
				`setq(cfg, knapsackConfig([1, 2, 3], 10.0, [2.0, 3.0, 4.0], [5.0, 6.0, 7.0]))`,
				`typeOf(cfg)`,
			},
			ExpectedValue: chariot.Str("S"),
		},
		{
			Name:           "Config with mismatched arrays",
			Script:         []string{`knapsackConfig([1, 2, 3], 10.0, [2.0, 3.0], [5.0, 6.0, 7.0])`},
			ExpectedError:  true,
			ErrorSubstring: "weights and values arrays must match items count",
		},
		{
			Name:           "Config with missing arguments",
			Script:         []string{`knapsackConfig([1, 2, 3], 10.0)`},
			ExpectedError:  true,
			ErrorSubstring: "knapsackConfig requires at least 4 args",
		},
		{
			Name:           "Config with wrong type for items",
			Script:         []string{`knapsackConfig(5, 10.0, [2.0, 3.0], [5.0, 6.0])`},
			ExpectedError:  true,
			ErrorSubstring: "knapsackConfig arg 0 (items) must be array",
		},
		{
			Name:           "Config with wrong type for capacity",
			Script:         []string{`knapsackConfig([1, 2], "10", [2.0, 3.0], [5.0, 6.0])`},
			ExpectedError:  true,
			ErrorSubstring: "knapsackConfig arg 1 (capacity) must be number",
		},
		{
			Name:           "Config with non-numeric weights",
			Script:         []string{`knapsackConfig([1, 2], 10.0, [2.0, "bad"], [5.0, 6.0])`},
			ExpectedError:  true,
			ErrorSubstring: "knapsackConfig weights[1] must be number",
		},
		{
			Name:           "Config with non-numeric values",
			Script:         []string{`knapsackConfig([1, 2], 10.0, [2.0, 3.0], [5.0, "bad"])`},
			ExpectedError:  true,
			ErrorSubstring: "knapsackConfig values[1] must be number",
		},
	}

	RunTestCases(t, tests)
}

// TestKnapsackSolver tests the main knapsack() solver function
// NOTE: These tests require a properly configured knapsack library.
// If tests fail with "solve_knapsack_v2_from_json failed", the library
// may not be properly linked or configured for your platform.
func TestKnapsackSolver(t *testing.T) {
	tests := []TestCase{
		{
			Name: "Simple knapsack with config helper",
			Script: []string{
				`setq(cfg, knapsackConfig([1, 2, 3], 10.0, [2.0, 3.0, 4.0], [5.0, 6.0, 7.0]))`,
				`setq(result, knapsack(cfg))`,
				`getProp(result, "numItems")`,
			},
			ExpectedValue: chariot.Number(3),
		},
		{
			Name: "Check solution has select array",
			Script: []string{
				`setq(cfg, knapsackConfig([1, 2], 5.0, [2.0, 3.0], [10.0, 15.0]))`,
				`setq(result, knapsack(cfg))`,
				`setq(sel, getProp(result, "select"))`,
				`length(sel)`,
			},
			ExpectedValue: chariot.Number(2),
		},
		{
			Name: "Check solution has objective value",
			Script: []string{
				`setq(cfg, knapsackConfig([1, 2, 3], 10.0, [2.0, 3.0, 4.0], [5.0, 6.0, 7.0]))`,
				`setq(result, knapsack(cfg))`,
				`equal(typeOf(getProp(result, "objective")), "N")`,
			},
			ExpectedValue: chariot.Bool(true),
		},
		{
			Name: "Check solution has penalty value",
			Script: []string{
				`setq(cfg, knapsackConfig([1, 2], 5.0, [2.0, 3.0], [10.0, 15.0]))`,
				`setq(result, knapsack(cfg))`,
				`equal(typeOf(getProp(result, "penalty")), "N")`,
			},
			ExpectedValue: chariot.Bool(true),
		},
		{
			Name: "Check solution has total value",
			Script: []string{
				`setq(cfg, knapsackConfig([1], 5.0, [2.0], [10.0]))`,
				`setq(result, knapsack(cfg))`,
				`equal(typeOf(getProp(result, "total")), "N")`,
			},
			ExpectedValue: chariot.Bool(true),
		},
		{
			Name: "Knapsack with options JSON",
			Script: []string{
				`setq(cfg, knapsackConfig([1, 2, 3], 10.0, [2.0, 3.0, 4.0], [5.0, 6.0, 7.0]))`,
				`setq(opts, parseJSON('{"beam_width": 100, "iters": 1000}'))`,
				`setq(result, knapsack(cfg, toJSON(opts)))`,
				`getProp(result, "numItems")`,
			},
			ExpectedValue: chariot.Number(3),
		},
		{
			Name:           "Knapsack with missing config argument",
			Script:         []string{`knapsack()`},
			ExpectedError:  true,
			ErrorSubstring: "knapsack requires at least 1 argument",
		},
		{
			Name:           "Knapsack with wrong config type",
			Script:         []string{`knapsack(123)`},
			ExpectedError:  true,
			ErrorSubstring: "knapsack arg 0 (configJSON) must be string",
		},
		{
			Name:           "Knapsack with wrong options type",
			Script:         []string{`knapsack("{}", 456)`},
			ExpectedError:  true,
			ErrorSubstring: "knapsack arg 1 (optionsJSON) must be string",
		},
	}

	RunTestCases(t, tests)
}

// TestKnapsackIntegration tests complete knapsack workflows
func TestKnapsackIntegration(t *testing.T) {
	tests := []TestCase{
		{
			Name: "Full workflow: build config, solve, extract selection",
			Script: []string{
				`setq(items, ["item1", "item2", "item3", "item4"])`,
				`setq(capacity, 15.0)`,
				`setq(weights, [5.0, 7.0, 4.0, 3.0])`,
				`setq(values, [10.0, 15.0, 12.0, 8.0])`,
				`setq(cfg, knapsackConfig(items, capacity, weights, values))`,
				`setq(solution, knapsack(cfg))`,
				`setq(selection, getProp(solution, "select"))`,
				`length(selection)`,
			},
			ExpectedValue: chariot.Number(4),
		},
		{
			Name: "Extract selected items from solution",
			Script: []string{
				`setq(items, ["A", "B", "C"])`,
				`setq(cfg, knapsackConfig(items, 10.0, [4.0, 5.0, 6.0], [8.0, 10.0, 12.0]))`,
				`setq(solution, knapsack(cfg))`,
				`setq(selection, getProp(solution, "select"))`,
				`biggerEq(length(selection), 0)`,
			},
			ExpectedValue: chariot.Bool(true),
		},
		{
			Name: "Verify objective calculation",
			Script: []string{
				`setq(cfg, knapsackConfig([1, 2], 10.0, [3.0, 4.0], [5.0, 6.0]))`,
				`setq(solution, knapsack(cfg))`,
				`setq(obj, getProp(solution, "objective"))`,
				`setq(pen, getProp(solution, "penalty"))`,
				`setq(tot, getProp(solution, "total"))`,
				`equal(tot, sub(obj, pen))`,
			},
			ExpectedValue: chariot.Bool(true),
		},
		{
			Name: "Multiple knapsack calls in sequence",
			Script: []string{
				`setq(cfg1, knapsackConfig([1, 2], 5.0, [2.0, 3.0], [10.0, 15.0]))`,
				`setq(sol1, knapsack(cfg1))`,
				`setq(cfg2, knapsackConfig([1, 2, 3], 10.0, [2.0, 3.0, 4.0], [5.0, 6.0, 7.0]))`,
				`setq(sol2, knapsack(cfg2))`,
				`add(getProp(sol1, "numItems"), getProp(sol2, "numItems"))`,
			},
			ExpectedValue: chariot.Number(5), // 2 + 3
		},
		{
			Name: "Empty knapsack (no items)",
			Script: []string{
				`setq(cfg, knapsackConfig([], 10.0, [], []))`,
				`knapsack(cfg)`,
			},
			ExpectedError:  true,
			ErrorSubstring: "knapsack solve failed",
		},
		{
			Name: "Single item knapsack",
			Script: []string{
				`setq(cfg, knapsackConfig([1], 10.0, [5.0], [20.0]))`,
				`setq(solution, knapsack(cfg))`,
				`setq(sel, getProp(solution, "select"))`,
				`getAt(sel, 0)`,
			},
			ExpectedValue: chariot.Number(1), // Single item should be selected
		},
	}

	RunTestCases(t, tests)
}

// TestKnapsackPerformance tests solver with larger problems
func TestKnapsackPerformance(t *testing.T) {
	tests := []TestCase{
		{
			Name: "Medium-size knapsack (10 items)",
			Script: []string{
				`setq(items, [1, 2, 3, 4, 5, 6, 7, 8, 9, 10])`,
				`setq(weights, [2.0, 4.0, 6.0, 8.0, 10.0, 12.0, 14.0, 16.0, 18.0, 20.0])`,
				`setq(values, [3.0, 6.0, 9.0, 12.0, 15.0, 18.0, 21.0, 24.0, 27.0, 30.0])`,
				`setq(cfg, knapsackConfig(items, 50.0, weights, values))`,
				`setq(solution, knapsack(cfg))`,
				`getProp(solution, "numItems")`,
			},
			ExpectedValue: chariot.Number(10),
		},
		{
			Name: "Verify solution structure for larger problem",
			Script: []string{
				`setq(items, [1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20])`,
				`setq(weights, [2.0, 3.0, 4.0, 5.0, 6.0, 7.0, 8.0, 9.0, 10.0, 11.0, 12.0, 13.0, 14.0, 15.0, 16.0, 17.0, 18.0, 19.0, 20.0, 21.0])`,
				`setq(values, [2.0, 4.0, 6.0, 8.0, 10.0, 12.0, 14.0, 16.0, 18.0, 20.0, 22.0, 24.0, 26.0, 28.0, 30.0, 32.0, 34.0, 36.0, 38.0, 40.0])`,
				`setq(cfg, knapsackConfig(items, 100.0, weights, values))`,
				`setq(solution, knapsack(cfg))`,
				`setq(hasNumItems, equal(typeOf(getProp(solution, "numItems")), "N"))`,
				`setq(hasSelect, equal(typeOf(getProp(solution, "select")), "A"))`,
				`setq(hasObjective, equal(typeOf(getProp(solution, "objective")), "N"))`,
				`setq(hasPenalty, equal(typeOf(getProp(solution, "penalty")), "N"))`,
				`setq(hasTotal, equal(typeOf(getProp(solution, "total")), "N"))`,
				`and(hasNumItems, and(hasSelect, and(hasObjective, and(hasPenalty, hasTotal))))`,
			},
			ExpectedValue: chariot.Bool(true),
		},
	}

	RunTestCases(t, tests)
}
