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

func TestRangeFunction(t *testing.T) {
	tests := []TestCase{
		{
			Name: "Simple range 0 to 5",
			Script: []string{
				`setq(arr, range(0, 5))`,
				`length(arr)`,
			},
			ExpectedValue: chariot.Number(5),
		},
		{
			Name: "Range first element",
			Script: []string{
				`setq(arr, range(0, 5))`,
				`getAt(arr, 0)`,
			},
			ExpectedValue: chariot.Number(0),
		},
		{
			Name: "Range last element",
			Script: []string{
				`setq(arr, range(0, 5))`,
				`getAt(arr, 4)`,
			},
			ExpectedValue: chariot.Number(4),
		},
		{
			Name: "Range with start and end",
			Script: []string{
				`setq(arr, range(1, 6))`,
				`length(arr)`,
			},
			ExpectedValue: chariot.Number(5),
		},
		{
			Name: "Range start-end first element",
			Script: []string{
				`setq(arr, range(10, 15))`,
				`getAt(arr, 0)`,
			},
			ExpectedValue: chariot.Number(10),
		},
		{
			Name: "Range start-end last element",
			Script: []string{
				`setq(arr, range(10, 15))`,
				`getAt(arr, 4)`,
			},
			ExpectedValue: chariot.Number(14),
		},
		{
			Name: "Range 0 to 10",
			Script: []string{
				`setq(arr, range(0, 10))`,
				`length(arr)`,
			},
			ExpectedValue: chariot.Number(10),
		},
		{
			Name: "Range check multiple values",
			Script: []string{
				`setq(arr, range(5, 10))`,
				`setq(v0, getAt(arr, 0))`,
				`setq(v1, getAt(arr, 1))`,
				`setq(v2, getAt(arr, 2))`,
				`setq(v3, getAt(arr, 3))`,
				`setq(v4, getAt(arr, 4))`,
				`and(equal(v0, 5), and(equal(v1, 6), and(equal(v2, 7), and(equal(v3, 8), equal(v4, 9)))))`,
			},
			ExpectedValue: chariot.Bool(true),
		},
		{
			Name: "Empty range when start equals end",
			Script: []string{
				`setq(arr, range(5, 5))`,
				`length(arr)`,
			},
			ExpectedValue: chariot.Number(0),
		},
		{
			Name: "Empty range when start > end",
			Script: []string{
				`setq(arr, range(10, 5))`,
				`length(arr)`,
			},
			ExpectedValue: chariot.Number(0),
		},
		{
			Name: "Large range",
			Script: []string{
				`setq(arr, range(1, 101))`,
				`length(arr)`,
			},
			ExpectedValue: chariot.Number(100),
		},
		{
			Name: "Large range first element",
			Script: []string{
				`setq(arr, range(1, 101))`,
				`getAt(arr, 0)`,
			},
			ExpectedValue: chariot.Number(1),
		},
		{
			Name: "Large range last element",
			Script: []string{
				`setq(arr, range(1, 101))`,
				`getAt(arr, 99)`,
			},
			ExpectedValue: chariot.Number(100),
		},
		{
			Name: "Negative range",
			Script: []string{
				`setq(arr, range(-5, 0))`,
				`length(arr)`,
			},
			ExpectedValue: chariot.Number(5),
		},
		{
			Name: "Negative range first element",
			Script: []string{
				`setq(arr, range(-5, 0))`,
				`getAt(arr, 0)`,
			},
			ExpectedValue: chariot.Number(-5),
		},
		{
			Name: "Negative to positive range",
			Script: []string{
				`setq(arr, range(-3, 3))`,
				`length(arr)`,
			},
			ExpectedValue: chariot.Number(6),
		},
		{
			Name: "Use range in array composition",
			Script: []string{
				`setq(r1, range(0, 3))`,
				`setq(r2, range(5, 8))`,
				`setq(combined, array(r1, 99, r2))`,
				`length(combined)`,
			},
			ExpectedValue: chariot.Number(3),
		},
		{
			Name: "Range error: wrong number of arguments (1)",
			Script: []string{
				`range(5)`,
			},
			ExpectedError:  true,
			ErrorSubstring: "range requires 2 arguments",
		},
		{
			Name: "Range error: wrong number of arguments (0)",
			Script: []string{
				`range()`,
			},
			ExpectedError:  true,
			ErrorSubstring: "range requires 2 arguments",
		},
		{
			Name: "Range error: wrong number of arguments (3)",
			Script: []string{
				`range(0, 10, 2)`,
			},
			ExpectedError:  true,
			ErrorSubstring: "range requires 2 arguments",
		},
		{
			Name: "Range error: non-numeric start",
			Script: []string{
				`range('start', 10)`,
			},
			ExpectedError:  true,
			ErrorSubstring: "start must be a number",
		},
		{
			Name: "Range error: non-numeric end",
			Script: []string{
				`range(0, 'end')`,
			},
			ExpectedError:  true,
			ErrorSubstring: "end must be a number",
		},
	}

	RunTestCases(t, tests)
}
