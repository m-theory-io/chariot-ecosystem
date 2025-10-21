package tests

import (
	"testing"

	"github.com/bhouse1273/chariot-ecosystem/services/go-chariot/chariot"
)

// Test basic expressions
func TestBasicExpressions(t *testing.T) {
	tests := []TestCase{
		{
			Name: "Number Literal",
			Script: []string{
				"42",
			},
			ExpectedValue: chariot.Number(42),
		},
		{
			Name: "String Literal",
			Script: []string{
				"'Hello, world!'",
			},
			ExpectedValue: chariot.Str("Hello, world!"),
		},
		{
			Name: "Addition",
			Script: []string{
				"add(2, 3)",
			},
			ExpectedValue: chariot.Number(5),
		},
		{
			Name: "Subtraction",
			Script: []string{
				"sub(10, 4)",
			},
			ExpectedValue: chariot.Number(6),
		},
		{
			Name: "Variable Declaration and Access",
			Script: []string{
				"declare(x, 'N', 25)",
				"x",
			},
			ExpectedValue: chariot.Number(25),
		},
		{
			Name: "Global Variable Declaration and Access",
			Script: []string{
				"declareGlobal(xg, 'N', 25)",
				"xg",
			},
			ExpectedValue: chariot.Number(25),
		},
		{
			Name: "Global Variable Access",
			Script: []string{
				"xg",
			},
			ExpectedValue: chariot.Number(25),
		},
		{
			Name: "Function Variable Access",
			Script: []string{
				"declare(funcVar, 'F', func() { mul(2, 15) })",
				"call(funcVar)",
			},
			ExpectedValue: chariot.Number(30),
		},
		{
			Name: "Function Variable Access with Args",
			Script: []string{
				"declare(funcVar, 'F', func(a, b) { mul(a, b) })",
				"call(funcVar, 2, 60)",
			},
			ExpectedValue: chariot.Number(120),
		},
		{
			Name: "Symbol Variable Access",
			Script: []string{
				"declare(myVar, 'N', 42)",
				"symbol('myVar')",
			},
			ExpectedValue: chariot.Number(42),
		},
	}

	RunTestCases(t, tests)
}

// Test control flow
func TestControlFlow(t *testing.T) {
	tests := []TestCase{
		{
			Name: "If True Branch",
			Script: []string{
				`declare(n, 'N', 10)`,
				`declare(result, 'L', false)`,
				`if(bigger(n, 5)) {`,
				`    setq(result, True)`,
				`} else {`,
				`    setq(result, False)`,
				`}`,
				`result`,
			},
			ExpectedValue: chariot.Bool(true),
		},
		{
			Name: "If False Branch",
			Script: []string{
				`declare(n, 'N', 3)`,
				`declare(result, 'L', false)`,
				`if(bigger(n, 5)) {`,
				`    setq(result, True)`,
				`} else {`,
				`    setq(result, False)`,
				`}`,
				`result`,
			},
			ExpectedValue: chariot.Bool(false),
		},
		{
			Name: "Else If",
			Script: []string{
				`declare(n, 'N', 5)`,
				`declare(result, 'V')`,
				`if(bigger(n, 5)) {`,
				`    setq(result, 'greater')`,
				`} else if(equal(n, 5)) {`,
				`    setq(result, 'equal')`,
				`} else {`,
				`    setq(result, 'less')`,
				`}`,
				`result`,
			},
			ExpectedValue: chariot.Str("equal"),
		},
		{
			Name: "While Loop",
			Script: []string{
				`declare(counter, 'N', 0)`,
				`declare(sum, 'N', 0)`,
				`while(smaller(counter, 5)) {`,
				`    setq(sum, add(sum, counter))`,
				`    setq(counter, add(counter, 1))`,
				`}`,
				`sum`,
			},
			ExpectedValue: chariot.Number(10), // 0+1+2+3+4
		},
		{
			Name: "Loop With Break",
			Script: []string{
				`declare(counter, 'N', 0)`,
				`declare(sum, 'N', 0)`,
				`while(smaller(counter, 10)) {`,
				`    setq(counter, add(counter, 1))`,
				`    if(equal(counter, 5)) {`,
				`        break()`,
				`    }`,
				`    setq(sum, add(sum, counter))`,
				`}`,
				`sum`,
			},
			ExpectedValue: chariot.Number(10), // 1+2+3+4 (break at 5)
		},
	}

	RunTestCases(t, tests)
}

// Test variable scoping
func TestVariableScoping(t *testing.T) {
	tests := []TestCase{
		{
			Name: "Local Variable Shadow with Proper Scoping",
			Script: []string{
				`declare(x, 'N', 5)`,        // Outer x
				`declare(inner, 'V', null)`, // Declare before block
				`if(True) {`,
				`    declare(x, 'N', 10)`, // Inner x shadows outer x
				`    setq(inner, x)`,      // Sets outer 'inner' to inner x's value (10)
				`}`,
				`setq(result, x)`, // Outer x is still 5
				`[inner, result]`, // Should be [10, 5]
			},
			ExpectedValue: &chariot.ArrayValue{
				Elements: []chariot.Value{
					chariot.Number(10),
					chariot.Number(5),
				},
			},
		},
		{
			Name: "Accessing Outer Variable",
			Script: []string{
				`declare(x, 'N', 5)`,
				`if(True) {`,
				`    setq(x, 10) // Update the outer x`,
				`}`,
				`x // Should be 10`,
			},
			ExpectedValue: chariot.Number(10),
		},
		{
			Name: "Variable Type Enforcement",
			Script: []string{
				`declare(counter, 'N', 0)`,
				`setq(counter, 'string') // Should fail`,
			},
			ExpectedError:  true,
			ErrorSubstring: "type mismatch",
		},
	}

	RunTestCases(t, tests)
}

// Test error scenarios
func TestErrorHandling(t *testing.T) {
	tests := []TestCase{
		{
			Name: "Undefined Variable",
			Script: []string{
				`notDefined`,
			},
			ExpectedError:  true,
			ErrorSubstring: "variable 'notDefined' not defined",
		},
		{
			Name: "Undefined Function",
			Script: []string{
				`nonExistentFunction()`,
			},
			ExpectedError:  true,
			ErrorSubstring: "undefined function",
		},
		{
			Name: "Type Error",
			Script: []string{
				`add('string', 5)`,
			},
			ExpectedError:  true,
			ErrorSubstring: "add requires two numbers",
		},
	}

	RunTestCases(t, tests)
}

// Extended test for the if/else control structure
func TestIfElseLogic(t *testing.T) {
	tests := []TestCase{
		{
			Name: "Complex If-Else Chain",
			Script: []string{
				`declare(n1, 'N', 2)`,
				`declare(n2, 'N', 2)`,
				`declare(result, 'V')`,
				`if(bigger(n1, n2)) {`,
				`    setq(result, 'n1 is bigger')`,
				`} else if(bigger(n2, n1)) {`,
				`    setq(result, 'n2 is bigger')`,
				`} else {`,
				`    setq(result, 'they are equal')`,
				`}`,
				`result`,
			},
			ExpectedValue: chariot.Str("they are equal"),
		},
		{
			Name: "Nested If-Else",
			Script: []string{
				`declare(x, 'N', 10)`,
				`declare(y, 'N', 5)`,
				`declare(z, 'N', 15)`,
				`declare(result, 'V')`,
				`if(bigger(x, y)) {`,
				`    if(bigger(x, z)) {`,
				`        setq(result, 'x is biggest')`,
				`    } else {`,
				`        setq(result, 'z is biggest')`,
				`    }`,
				`} else {`,
				`    if(bigger(y, z)) {`,
				`        setq(result, 'y is biggest')`,
				`    } else {`,
				`        setq(result, 'z is biggest')`,
				`    }`,
				`}`,
				`result`,
			},
			ExpectedValue: chariot.Str("z is biggest"),
		},
	}

	RunTestCases(t, tests)
}

// Test switch statements
func TestSwitchStatements(t *testing.T) {
	tests := []TestCase{
		{
			Name: "Switch With Test Value - First Case Matches",
			Script: []string{
				`declare(x, 'N', 100)`,
				`declare(result, 'V')`,
				`switch(x) {`,
				`    case(100) {`,
				`        setq(result, 'x equals 100')`,
				`    }`,
				`    case(200) {`,
				`        setq(result, 'x equals 200')`,
				`    }`,
				`    default() {`,
				`        setq(result, 'x equals neither 100 nor 200')`,
				`    }`,
				`}`,
				`result`,
			},
			ExpectedValue: chariot.Str("x equals 100"),
		},
		{
			Name: "Switch With Test Value - Second Case Matches",
			Script: []string{
				`declare(x, 'N', 200)`,
				`declare(result, 'V')`,
				`switch(x) {`,
				`    case(100) {`,
				`        setq(result, 'x equals 100')`,
				`    }`,
				`    case(200) {`,
				`        setq(result, 'x equals 200')`,
				`    }`,
				`    default() {`,
				`        setq(result, 'x equals neither 100 nor 200')`,
				`    }`,
				`}`,
				`result`,
			},
			ExpectedValue: chariot.Str("x equals 200"),
		},
		{
			Name: "Switch With Test Value - Default Case",
			Script: []string{
				`declare(x, 'N', 300)`,
				`declare(result, 'V')`,
				`switch(x) {`,
				`    case(100) {`,
				`        setq(result, 'x equals 100')`,
				`    }`,
				`    case(200) {`,
				`        setq(result, 'x equals 200')`,
				`    }`,
				`    default() {`,
				`        setq(result, 'x equals neither 100 nor 200')`,
				`    }`,
				`}`,
				`result`,
			},
			ExpectedValue: chariot.Str("x equals neither 100 nor 200"),
		},
		{
			Name: "Switch Without Test Value - Expression Cases",
			Script: []string{
				`declare(x, 'N', 100)`,
				`declare(result, 'V')`,
				`switch() {`,
				`    case(equal(x, 100)) {`,
				`        setq(result, 'x equals 100')`,
				`    }`,
				`    case(equal(x, 200)) {`,
				`        setq(result, 'x equals 200')`,
				`    }`,
				`    default() {`,
				`        setq(result, 'x equals neither 100 nor 200')`,
				`    }`,
				`}`,
				`result`,
			},
			ExpectedValue: chariot.Str("x equals 100"),
		},
		{
			Name: "Switch Without Test Value - Complex Expressions",
			Script: []string{
				`declare(x, 'N', 15)`,
				`declare(result, 'V')`,
				`switch() {`,
				`    case(smaller(x, 10)) {`,
				`        setq(result, 'x is small')`,
				`    }`,
				`    case(and(biggerEq(x, 10), smaller(x, 20))) {`,
				`        setq(result, 'x is medium')`,
				`    }`,
				`    case(biggerEq(x, 20)) {`,
				`        setq(result, 'x is large')`,
				`    }`,
				`    default() {`,
				`        setq(result, 'unknown')`,
				`    }`,
				`}`,
				`result`,
			},
			ExpectedValue: chariot.Str("x is medium"),
		},
		{
			Name: "Switch With String Values",
			Script: []string{
				`declare(color, 'S', 'red')`,
				`declare(result, 'V')`,
				`switch(color) {`,
				`    case('red') {`,
				`        setq(result, 'stop')`,
				`    }`,
				`    case('yellow') {`,
				`        setq(result, 'caution')`,
				`    }`,
				`    case('green') {`,
				`        setq(result, 'go')`,
				`    }`,
				`    default() {`,
				`        setq(result, 'unknown color')`,
				`    }`,
				`}`,
				`result`,
			},
			ExpectedValue: chariot.Str("stop"),
		},
		{
			Name: "Switch With Boolean Values",
			Script: []string{
				`declare(isValid, 'L', true)`,
				`declare(result, 'V')`,
				`switch(isValid) {`,
				`    case(true) {`,
				`        setq(result, 'valid')`,
				`    }`,
				`    case(false) {`,
				`        setq(result, 'invalid')`,
				`    }`,
				`}`,
				`result`,
			},
			ExpectedValue: chariot.Str("valid"),
		},
		{
			Name: "Switch With Only Default Case",
			Script: []string{
				`declare(x, 'N', 42)`,
				`declare(result, 'V')`,
				`switch(x) {`,
				`    default() {`,
				`        setq(result, 'always executes')`,
				`    }`,
				`}`,
				`result`,
			},
			ExpectedValue: chariot.Str("always executes"),
		},
		{
			Name: "Switch Without Default - No Match",
			Script: []string{
				`declare(x, 'N', 999)`,
				`declare(result, 'V', 'initial')`,
				`switch(x) {`,
				`    case(100) {`,
				`        setq(result, 'matched 100')`,
				`    }`,
				`    case(200) {`,
				`        setq(result, 'matched 200')`,
				`    }`,
				`}`,
				`result`,
			},
			ExpectedValue: chariot.Str("initial"), // Should remain unchanged
		},
		{
			Name: "Switch Early Exit - First Match Wins",
			Script: []string{
				`declare(x, 'N', 100)`,
				`declare(result, 'V')`,
				`declare(counter, 'N', 0)`,
				`switch(x) {`,
				`    case(100) {`,
				`        setq(result, 'first match')`,
				`        setq(counter, add(counter, 1))`,
				`    }`,
				`    case(100) {`,
				`        setq(result, 'second match')`,
				`        setq(counter, add(counter, 1))`,
				`    }`,
				`    default() {`,
				`        setq(result, 'default')`,
				`        setq(counter, add(counter, 1))`,
				`    }`,
				`}`,
				`[result, counter]`,
			},
			ExpectedValue: &chariot.ArrayValue{
				Elements: []chariot.Value{
					chariot.Str("first match"),
					chariot.Number(1), // Only first case should execute
				},
			},
		},
		{
			Name: "Switch With Variable References in Cases",
			Script: []string{
				`declare(x, 'N', 10)`,
				`declare(target, 'N', 10)`,
				`declare(result, 'V')`,
				`switch(x) {`,
				`    case(target) {`,
				`        setq(result, 'matched variable')`,
				`    }`,
				`    case(20) {`,
				`        setq(result, 'matched literal')`,
				`    }`,
				`    default() {`,
				`        setq(result, 'no match')`,
				`    }`,
				`}`,
				`result`,
			},
			ExpectedValue: chariot.Str("matched variable"),
		},
		{
			Name: "Nested Switch Statements",
			Script: []string{
				`declare(x, 'N', 1)`,
				`declare(y, 'N', 2)`,
				`declare(result, 'V')`,
				`switch(x) {`,
				`    case(1) {`,
				`        switch(y) {`,
				`            case(1) {`,
				`                setq(result, 'x=1, y=1')`,
				`            }`,
				`            case(2) {`,
				`                setq(result, 'x=1, y=2')`,
				`            }`,
				`            default() {`,
				`                setq(result, 'x=1, y=other')`,
				`            }`,
				`        }`,
				`    }`,
				`    case(2) {`,
				`        setq(result, 'x=2')`,
				`    }`,
				`}`,
				`result`,
			},
			ExpectedValue: chariot.Str("x=1, y=2"),
		},
	}

	RunTestCases(t, tests)
}
