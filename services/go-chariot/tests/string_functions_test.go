// tests/string_functions_test.go
package tests

import (
	"testing"

	"github.com/bhouse1273/go-chariot/chariot"
)

func TestStringFunctions(t *testing.T) {

	tests := []TestCase{
		{
			Name: "Substring - Basic",
			Script: []string{
				`substr('Hello World', 0, 5)`,
			},
			ExpectedValue: chariot.Str("Hello"),
		},
		{
			Name: "Substring - From Middle",
			Script: []string{
				`substr('Hello World', 6, 5)`,
			},
			ExpectedValue: chariot.Str("World"),
		},
		{
			Name: "Substring - No Length (to end)",
			Script: []string{
				`substr('Hello World', 6)`,
			},
			ExpectedValue: chariot.Str("World"),
		},
		{
			Name: "Index Of - Found",
			Script: []string{
				`indexOf('Hello World', 'World')`,
			},
			ExpectedValue: chariot.Number(6),
		},
		{
			Name: "Index Of - Not Found",
			Script: []string{
				`indexOf('Hello World', 'xyz')`,
			},
			ExpectedValue: chariot.Number(-1),
		},
		{
			Name: "Index Of - Case Sensitive",
			Script: []string{
				`indexOf('Hello World', 'world')`,
			},
			ExpectedValue: chariot.Number(-1),
		},
		{
			Name: "String Replace - Single",
			Script: []string{
				`replace('Hello World', 'World', 'Universe')`,
			},
			ExpectedValue: chariot.Str("Hello Universe"),
		},
		{
			Name: "String Replace - Multiple",
			Script: []string{
				`replace('foo bar foo', 'foo', 'baz')`,
			},
			ExpectedValue: chariot.Str("baz bar baz"),
		},
		{
			Name: "String Trim - Whitespace",
			Script: []string{
				`trim('  Hello World  ')`,
			},
			ExpectedValue: chariot.Str("Hello World"),
		},
		{
			Name: "String Trim - Custom Characters",
			Script: []string{
				`trim('***Hello***', '*')`,
			},
			ExpectedValue: chariot.Str("Hello"),
		},
		{
			Name: "Upper Case",
			Script: []string{
				`upper('hello world')`,
			},
			ExpectedValue: chariot.Str("HELLO WORLD"),
		},
		{
			Name: "Lower Case",
			Script: []string{
				`lower('HELLO WORLD')`,
			},
			ExpectedValue: chariot.Str("hello world"),
		},
		{
			Name: "Has Prefix - True",
			Script: []string{
				`hasPrefix('Hello World', 'Hello')`,
			},
			ExpectedValue: chariot.Bool(true),
		},
		{
			Name: "Has Prefix - False",
			Script: []string{
				`hasPrefix('Hello World', 'World')`,
			},
			ExpectedValue: chariot.Bool(false),
		},
		{
			Name: "Has Suffix - True",
			Script: []string{
				`hasSuffix('Hello World', 'World')`,
			},
			ExpectedValue: chariot.Bool(true),
		},
		{
			Name: "Has Suffix - False",
			Script: []string{
				`hasSuffix('Hello World', 'Hello')`,
			},
			ExpectedValue: chariot.Bool(false),
		},
		{
			Name: "Pad Left",
			Script: []string{
				`padLeft('123', 5, '0')`,
			},
			ExpectedValue: chariot.Str("00123"),
		},
		{
			Name: "Pad Right",
			Script: []string{
				`padRight('123', 5, '0')`,
			},
			ExpectedValue: chariot.Str("12300"),
		},
	}

	RunTestCases(t, tests)
}

func TestStringErrorHandling(t *testing.T) {
	tests := []TestCase{
		{
			Name: "Substring - Invalid Start Index",
			Script: []string{
				`substr('Hello', -1, 3)`,
			},
			ExpectedError: true,
		},
		{
			Name: "Substring - Start Beyond Length",
			Script: []string{
				`substr('Hello', 10, 3)`,
			},
			ExpectedValue: chariot.Str(""), // Should return empty string
		},
		{
			Name: "Index Of - Wrong Argument Count",
			Script: []string{
				`indexOf('Hello')`,
			},
			ExpectedError: true,
		},
		{
			Name: "Replace - Non-string Input",
			Script: []string{
				`replace(123, 'old', 'new')`,
			},
			ExpectedError: true,
		},
	}

	RunTestCases(t, tests)
}
