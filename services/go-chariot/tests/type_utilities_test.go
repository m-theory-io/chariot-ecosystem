// tests/type_utilities_test.go
package tests

import (
	"testing"

	"github.com/bhouse1273/chariot-ecosystem/services/go-chariot/chariot"
)

func TestTypeUtilities(t *testing.T) {
	tests := []TestCase{
		{
			Name: "Type Of - String",
			Script: []string{
				`typeOf('hello')`,
			},
			ExpectedValue: chariot.Str("S"),
		},
		{
			Name: "Type Of - Number",
			Script: []string{
				`typeOf(42)`,
			},
			ExpectedValue: chariot.Str("N"),
		},
		{
			Name: "Type Of - Boolean",
			Script: []string{
				`typeOf(true)`,
			},
			ExpectedValue: chariot.Str("L"),
		},
		{
			Name: "Type Of - Array",
			Script: []string{
				`typeOf(array(1, 2, 3))`,
			},
			ExpectedValue: chariot.Str("A"),
		},
		{
			Name: "Type Of - JSONNode",
			Script: []string{
				`typeOf(jsonNode('{"key": "value"}'))`,
			},
			ExpectedValue: chariot.Str("J"),
		},
		{
			Name: "Is Null - True",
			Script: []string{
				`isNull(null)`,
			},
			ExpectedValue: chariot.Bool(true),
		},
		{
			Name: "Is Null - False",
			Script: []string{
				`isNull('hello')`,
			},
			ExpectedValue: chariot.Bool(false),
		},
		{
			Name: "Is Null - Number Zero",
			Script: []string{
				`isNull(0)`,
			},
			ExpectedValue: chariot.Bool(false),
		},
		{
			Name: "To String - Number",
			Script: []string{
				`toString(42)`,
			},
			ExpectedValue: chariot.Str("42"),
		},
		{
			Name: "To String - Boolean",
			Script: []string{
				`toString(true)`,
			},
			ExpectedValue: chariot.Str("true"),
		},
		{
			Name: "To String - Already String",
			Script: []string{
				`toString('hello')`,
			},
			ExpectedValue: chariot.Str("hello"),
		},
		{
			Name: "To Number - String",
			Script: []string{
				`toNumber('42')`,
			},
			ExpectedValue: chariot.Number(42),
		},
		{
			Name: "To Number - String Float",
			Script: []string{
				`toNumber('3.14')`,
			},
			ExpectedValue: chariot.Number(3.14),
		},
		{
			Name: "To Number - Already Number",
			Script: []string{
				`toNumber(42)`,
			},
			ExpectedValue: chariot.Number(42),
		},
		{
			Name: "To Bool - String True",
			Script: []string{
				`toBool('true')`,
			},
			ExpectedValue: chariot.Bool(true),
		},
		{
			Name: "To Bool - String False",
			Script: []string{
				`toBool('false')`,
			},
			ExpectedValue: chariot.Bool(false),
		},
		{
			Name: "To Bool - Number Non-zero",
			Script: []string{
				`toBool(42)`,
			},
			ExpectedValue: chariot.Bool(true),
		},
		{
			Name: "To Bool - Number Zero",
			Script: []string{
				`toBool(0)`,
			},
			ExpectedValue: chariot.Bool(false),
		},
		{
			Name: "To Bool - Already Boolean",
			Script: []string{
				`toBool(true)`,
			},
			ExpectedValue: chariot.Bool(true),
		},
	}

	RunTestCases(t, tests)
}

func TestTypeUtilityErrorHandling(t *testing.T) {
	tests := []TestCase{
		{
			Name: "To Number - Invalid String",
			Script: []string{
				`toNumber('hello')`,
			},
			ExpectedValue: chariot.Number(0),
		},
		{
			Name: "To Bool - Invalid String",
			Script: []string{
				`toBool('maybe')`,
			},
			ExpectedValue: chariot.Bool(false),
		},
		{
			Name: "Type Of - No Arguments",
			Script: []string{
				`typeOf()`,
			},
			ExpectedError: true,
		},
	}

	RunTestCases(t, tests)
}
