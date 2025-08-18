package tests

import (
	"testing"

	"github.com/bhouse1273/chariot-ecosystem/services/go-chariot/chariot"
)

func TestArrayMinimal(t *testing.T) {
	t.Log("🎯 ARRAY TEST FILE IS EXECUTING!")

	// Test just one function that we KNOW works
	rt := chariot.NewRuntime()
	chariot.RegisterAll(rt)

	result, err := rt.ExecProgram(`setq(arr, array(10, 'test', True))
                indexOf(arr, 10)`)
	if err != nil {
		t.Fatalf("Array failed: %v", err)
	}

	t.Logf("✅ Array getAt(-1) result: %v", result)
}

func TestArrayOperations(t *testing.T) {
	tests := []TestCase{
		{
			Name: "Create Array and Get Element",
			Script: []string{
				`setq(arr, array(10, 'test', True))`,
				`getAt(arr, 0)`,
			},
			ExpectedValue: chariot.Number(10),
		},
		{
			Name: "Get String Element",
			Script: []string{
				`setq(arr, array(10, 'test', True, 20, 'test'))`,
				`getAt(arr, 1)`,
			},
			ExpectedValue: chariot.Str("test"),
		},
		{
			Name: "Get Boolean Element",
			Script: []string{
				`setq(arr, array(10, 'test', True, 20, 'test'))`,
				`getAt(arr, 2)`,
			},
			ExpectedValue: chariot.Bool(true),
		},
		{
			Name: "Out of Bounds Index Returns DBNull",
			Script: []string{
				`setq(arr, array(10, 'test', True))`,
				`getAt(arr, 10)`,
			},
			ExpectedValue: chariot.DBNull,
		},
		{
			Name: "Negative Index Returns DBNull",
			Script: []string{
				`setq(arr, array(10, 'test', True))`,
				`getAt(arr, -1)`,
			},
			ExpectedValue: chariot.DBNull,
		},
		{
			Name: "Set Array Element",
			Script: []string{
				`setq(arr, array(10, 'test', True))`,
				`setAt(arr, 0, 99)`,
				`getAt(arr, 0)`,
			},
			ExpectedValue: chariot.Number(99),
		},
		{
			Name: "Get Array Length",
			Script: []string{
				`setq(arr, array(10, 'test', True, 20, 'test'))`,
				`length(arr)`,
			},
			ExpectedValue: chariot.Number(5),
		},
		{
			Name: "Empty Array Length",
			Script: []string{
				`setq(arr, array())`,
				`length(arr)`,
			},
			ExpectedValue: chariot.Number(0),
		},
		{
			Name: "Add Single Element",
			Script: []string{
				`setq(arr, array(10, 'test'))`,
				`addTo(arr, 99)`,
				`length(arr)`,
			},
			ExpectedValue: chariot.Number(3),
		},
		{
			Name: "Add Multiple Elements",
			Script: []string{
				`setq(arr, array(10, 'test'))`,
				`addTo(arr, 99, 'new', false)`,
				`length(arr)`,
			},
			ExpectedValue: chariot.Number(5),
		},
		{
			Name: "Verify Added Element",
			Script: []string{
				`setq(arr, array(10, 'test'))`,
				`addTo(arr, 99)`,
				`getAt(arr, 2)`,
			},
			ExpectedValue: chariot.Number(99),
		},
		{
			Name: "Remove Element",
			Script: []string{
				`setq(arr, array(10, 'test', true, 20))`,
				`removeAt(arr, 0)`,
				`getAt(arr, 0)`,
			},
			ExpectedValue: chariot.Str("test"),
		},
		{
			Name: "Array Length After Removal",
			Script: []string{
				`setq(arr, array(10, 'test', True, 20, 'test'))`,
				`removeAt(arr, 0)`,
				`length(arr)`,
			},
			ExpectedValue: chariot.Number(4),
		},
		{
			Name: "Find Index of Element",
			Script: []string{
				`setq(arr, array(10, 'test', True, 20, 'test'))`,
				`indexOf(arr, 10)`,
			},
			ExpectedValue: chariot.Number(0),
		},
		{
			Name: "Find Index of String",
			Script: []string{
				`setq(arr, array(10, 'test', True, 20, 'test'))`,
				`indexOf(arr, 'test')`,
			},
			ExpectedValue: chariot.Number(1),
		},
		{
			Name: "Find Non-existing Element",
			Script: []string{
				`setq(arr, array(10, 'test', True, 20, 'test'))`,
				`indexOf(arr, 999)`,
			},
			ExpectedValue: chariot.Number(-1),
		},
		{
			Name: "Find Last Index of Element",
			Script: []string{
				`setq(arr, array(10, 'test', True, 20, 'test'))`,
				`lastIndex(arr, 'test')`,
			},
			ExpectedValue: chariot.Number(4),
		},
		{
			Name: "Find Last Index of Unique Element",
			Script: []string{
				`setq(arr, array(10, 'test', True, 20, 'test'))`,
				`lastIndex(arr, 10)`,
			},
			ExpectedValue: chariot.Number(0),
		},
		{
			Name: "Slice Array with Start and End",
			Script: []string{
				`setq(arr, array(10, 'test', True, 20, 'test'))`,
				`setq(sliced, slice(arr, 1, 3))`,
				`length(sliced)`,
			},
			ExpectedValue: chariot.Number(2),
		},
		{
			Name: "Verify Sliced Content",
			Script: []string{
				`setq(arr, array(10, 'test', True, 20, 'test'))`,
				`setq(sliced, slice(arr, 1, 3))`,
				`getAt(sliced, 0)`,
			},
			ExpectedValue: chariot.Str("test"),
		},
		{
			Name: "Slice with Just Start Index",
			Script: []string{
				`setq(arr, array(10, 'test', True, 20, 'test'))`,
				`setq(sliced, slice(arr, 3))`,
				`length(sliced)`,
			},
			ExpectedValue: chariot.Number(2),
		},
		{
			Name: "Reverse Array",
			Script: []string{
				`setq(arr, array(10, 'test', True))`,
				`setq(reversed, reverse(arr))`,
				`getAt(reversed, 0)`,
			},
			ExpectedValue: chariot.Bool(true),
		},
		{
			Name: "Verify Reverse Order",
			Script: []string{
				`setq(arr, array(10, 'test', True))`,
				`setq(reversed, reverse(arr))`,
				`getAt(reversed, 2)`,
			},
			ExpectedValue: chariot.Number(10),
		},
		{
			Name: "Reverse Empty Array",
			Script: []string{
				`setq(arr, array())`,
				`setq(reversed, reverse(arr))`,
				`length(reversed)`,
			},
			ExpectedValue: chariot.Number(0),
		},
	}

	RunTestCases(t, tests)
}

func TestArrayErrorHandling(t *testing.T) {
	tests := []TestCase{
		{
			Name: "SetAt Out of Bounds",
			Script: []string{
				`setq(arr, array(10, 'test'))
                setAt(arr, 10, 99)`,
			},
			ExpectedError:  true,
			ErrorSubstring: "out of bounds",
		},
		{
			Name: "SetAt Negative Index",
			Script: []string{
				`setq(arr, array(10, 'test'))
                setAt(arr, -1, 99)`,
			},
			ExpectedError:  true,
			ErrorSubstring: "out of bounds",
		},
		{
			Name: "RemoveAt Out of Bounds",
			Script: []string{
				`setq(arr, array(10, 'test'))
                removeAt(arr, 10)`,
			},
			ExpectedError:  true,
			ErrorSubstring: "out of bounds",
		},
		{
			Name: "Invalid Array Type for GetAt",
			Script: []string{
				`getAt(123, 0)`, // Use a type that's NOT supported (Number)
			},
			ExpectedError:  true,
			ErrorSubstring: "getAt not supported for type",
		},
		{
			Name: "Invalid Index Type",
			Script: []string{
				`setq(arr, array(10, 'test'))`,
				`getAt(arr, 'not a number')`,
			},
			ExpectedError:  true,
			ErrorSubstring: "index must be a number",
		},
		{
			Name: "Too Few Arguments for GetAt",
			Script: []string{
				`setq(arr, array(10, 'test'))`,
				`getAt(arr)`,
			},
			ExpectedError:  true,
			ErrorSubstring: "getAt requires 2 arguments",
		},
		{
			Name: "Too Few Arguments for SetAt",
			Script: []string{
				`setq(arr, array(10, 'test'))`,
				`setAt(arr, 0)`,
			},
			ExpectedError:  true,
			ErrorSubstring: "setAt requires 3 arguments",
		},
		{
			Name: "Too Few Arguments for AddTo",
			Script: []string{
				`setq(arr, array(10, 'test'))`,
				`addTo(arr)`,
			},
			ExpectedError:  true,
			ErrorSubstring: "addTo requires at least 2 arguments",
		},
	}

	RunTestCases(t, tests)
}
