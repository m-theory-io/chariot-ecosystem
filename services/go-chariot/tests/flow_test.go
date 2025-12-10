package tests

import (
	"testing"

	"github.com/bhouse1273/chariot-ecosystem/services/go-chariot/chariot"
)

// TestFlowControlFunctions tests return(), exit(), and break() control flow functions
func TestFlowControlFunctions(t *testing.T) {
	tests := []TestCase{
		{
			Name: "Return with value from function",
			Script: []string{
				`setq(testFunc, func() { return(42) })`,
				`call(testFunc)`,
			},
			ExpectedType:  "chariot.Number",
			ExpectedValue: chariot.Number(42),
		},
		{
			Name: "Return with no value (DBNull)",
			Script: []string{
				`setq(noReturnFunc, func() { return() })`,
				`call(noReturnFunc)`,
			},
			ExpectedValue: chariot.DBNull,
		},
		{
			Name: "Return from nested function calls",
			Script: []string{
				`setq(inner, func() { return(100) })`,
				`setq(outer, func() { setq(result, call(inner)) return(add(result, 50)) })`,
				`call(outer)`,
			},
			ExpectedType:  "chariot.Number",
			ExpectedValue: chariot.Number(150),
		},
		{
			Name: "Exit with code 0 (success)",
			Script: []string{
				`exit(0)`,
			},
			ExpectedValue: &chariot.ExitRequest{Code: 0},
		},
		{
			Name: "Exit with non-zero code (error)",
			Script: []string{
				`exit(1)`,
			},
			ExpectedValue: &chariot.ExitRequest{Code: 1},
		},
		{
			Name: "Exit with no argument (default 0)",
			Script: []string{
				`exit()`,
			},
			ExpectedValue: &chariot.ExitRequest{Code: 0},
		},
		{
			Name: "Break in loop test",
			Script: []string{
				`declare(sum, 'N', 0)`,
				`declare(i, 'N', 0)`,
				`while(smaller(i, 10)) {`,
				`    if(equal(i, 5)) {`,
				`        break()`,
				`    }`,
				`    setq(sum, add(sum, i))`,
				`    setq(i, add(i, 1))`,
				`}`,
				`sum`,
			},
			ExpectedType:  "chariot.Number",
			ExpectedValue: chariot.Number(10), // 0+1+2+3+4
		},
		{
			Name: "Return with complex value",
			Script: []string{
				`setq(makeArray, func() { return(array(1, 2, 3)) })`,
				`setq(arr, call(makeArray))`,
				`length(arr)`,
			},
			ExpectedType:  "chariot.Number",
			ExpectedValue: chariot.Number(3),
		},
	}

	RunTestCases(t, tests)
}
