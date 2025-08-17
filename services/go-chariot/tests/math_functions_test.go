// tests/math_functions_test.go
package tests

import (
	"testing"

	"github.com/bhouse1273/go-chariot/chariot"
)

func TestMathFunctions(t *testing.T) {
	tests := []TestCase{
		{
			Name: "Absolute Value - Positive",
			Script: []string{
				`abs(5.5)`,
			},
			ExpectedValue: chariot.Number(5.5),
		},
		{
			Name: "Absolute Value - Negative",
			Script: []string{
				`abs(-5.5)`,
			},
			ExpectedValue: chariot.Number(5.5),
		},
		{
			Name: "Absolute Value - Zero",
			Script: []string{
				`abs(0)`,
			},
			ExpectedValue: chariot.Number(0),
		},
		{
			Name: "Minimum of Two Numbers",
			Script: []string{
				`min(5, 3)`,
			},
			ExpectedValue: chariot.Number(3),
		},
		{
			Name: "Maximum of Two Numbers",
			Script: []string{
				`max(5, 3)`,
			},
			ExpectedValue: chariot.Number(5),
		},
		{
			Name: "Minimum of Multiple Numbers",
			Script: []string{
				`min(5, 3, 8, 1, 9)`,
			},
			ExpectedValue: chariot.Number(1),
		},
		{
			Name: "Maximum of Multiple Numbers",
			Script: []string{
				`max(5, 3, 8, 1, 9)`,
			},
			ExpectedValue: chariot.Number(9),
		},
		{
			Name: "Round - Positive",
			Script: []string{
				`round(3.7)`,
			},
			ExpectedValue: chariot.Number(4),
		},
		{
			Name: "Round - Negative",
			Script: []string{
				`round(-3.7)`,
			},
			ExpectedValue: chariot.Number(-4),
		},
		{
			Name: "Round - Half",
			Script: []string{
				`round(3.5)`,
			},
			ExpectedValue: chariot.Number(4),
		},
		{
			Name: "Floor Function",
			Script: []string{
				`floor(3.7)`,
			},
			ExpectedValue: chariot.Number(3),
		},
		{
			Name: "Floor - Negative",
			Script: []string{
				`floor(-3.2)`,
			},
			ExpectedValue: chariot.Number(-4),
		},
		{
			Name: "Ceiling Function",
			Script: []string{
				`ceiling(3.2)`,
			},
			ExpectedValue: chariot.Number(4),
		},
		{
			Name: "Ceiling - Negative",
			Script: []string{
				`ceiling(-3.7)`,
			},
			ExpectedValue: chariot.Number(-3),
		},
		{
			Name: "Square Root",
			Script: []string{
				`sqrt(16)`,
			},
			ExpectedValue: chariot.Number(4),
		},
		{
			Name: "Square Root - Decimal",
			Script: []string{
				`sqrt(2)`,
			},
			ExpectedValue: chariot.Number(1.4142135623730951),
		},
		{
			Name: "Power Function",
			Script: []string{
				`pow(2, 3)`,
			},
			ExpectedValue: chariot.Number(8),
		},
		{
			Name: "Power - Decimal Base",
			Script: []string{
				`pow(2.5, 2)`,
			},
			ExpectedValue: chariot.Number(6.25),
		},
		{
			Name: "Random Number Range",
			Script: []string{
				`setq(r, random(1, 10))`,
				`and(biggerEq(r, 1), smallerEq(r, 10))`,
			},
			ExpectedValue: chariot.Bool(true),
		},
		{
			Name: "Random - Different Results",
			Script: []string{
				`setq(r1, random(1, 100))`,
				`setq(r2, random(1, 100))`,
				`unequal(r1, r2)`, // Should be different (probably)
			},
			ExpectedValue: chariot.Bool(true),
		},
	}

	RunTestCases(t, tests)
}

func TestMathErrorHandling(t *testing.T) {
	tests := []TestCase{
		{
			Name: "Square Root - Negative Number",
			Script: []string{
				`sqrt(-1)`,
			},
			ExpectedError: true,
		},
		{
			Name: "Min - No Arguments",
			Script: []string{
				`min()`,
			},
			ExpectedError: true,
		},
		{
			Name: "Max - Non-numeric Argument",
			Script: []string{
				`max(5, 'hello')`,
			},
			ExpectedError: true,
		},
		{
			Name: "Power - Invalid Arguments",
			Script: []string{
				`pow('hello', 2)`,
			},
			ExpectedError: true,
		},
	}

	RunTestCases(t, tests)
}
