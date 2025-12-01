package tests

import (
	"testing"

	"github.com/bhouse1273/chariot-ecosystem/services/go-chariot/chariot"
)

// TestPolymorphicSlice tests slice() function on strings and arrays
func TestPolymorphicSlice(t *testing.T) {
	tests := []TestCase{
		// String slice tests
		{
			Name:          "String Slice - Basic",
			Script:        []string{`slice("hello world", 0, 5)`},
			ExpectedValue: chariot.Str("hello"),
		},
		{
			Name:          "String Slice - From Middle",
			Script:        []string{`slice("hello world", 6, 11)`},
			ExpectedValue: chariot.Str("world"),
		},
		{
			Name:          "String Slice - No End Index",
			Script:        []string{`slice("hello", 2)`},
			ExpectedValue: chariot.Str("llo"),
		},
		{
			Name:          "String Slice - Empty Result",
			Script:        []string{`slice("hello", 10, 15)`},
			ExpectedValue: chariot.Str(""),
		},
		{
			Name:          "String Slice - Negative Start",
			Script:        []string{`slice("hello", -1, 3)`},
			ExpectedValue: chariot.Str("hel"),
		},
		{
			Name:          "String Slice - End Beyond Length",
			Script:        []string{`slice("hello", 2, 100)`},
			ExpectedValue: chariot.Str("llo"),
		},
		{
			Name:          "String Slice - Unicode",
			Script:        []string{`slice("héllo 世界", 0, 3)`},
			ExpectedValue: chariot.Str("hél"),
		},

		// Array slice tests
		{
			Name: "Array Slice - Basic",
			Script: []string{
				`setq(arr, array(1, 2, 3, 4, 5))`,
				`slice(arr, 1, 4)`,
			},
			ExpectedValue: func() chariot.Value {
				arr := chariot.NewArray()
				arr.Append(chariot.Number(2))
				arr.Append(chariot.Number(3))
				arr.Append(chariot.Number(4))
				return arr
			}(),
		},
		{
			Name: "Array Slice - No End Index",
			Script: []string{
				`setq(arr, array("a", "b", "c", "d"))`,
				`slice(arr, 2)`,
			},
			ExpectedValue: func() chariot.Value {
				arr := chariot.NewArray()
				arr.Append(chariot.Str("c"))
				arr.Append(chariot.Str("d"))
				return arr
			}(),
		},
		{
			Name: "Array Slice - Empty Result",
			Script: []string{
				`setq(arr, array(1, 2, 3))`,
				`slice(arr, 5, 10)`,
			},
			ExpectedValue: chariot.NewArray(),
		},
		{
			Name: "Array Slice - Mixed Types",
			Script: []string{
				`setq(arr, array("hello", 42, true, "world"))`,
				`slice(arr, 1, 3)`,
			},
			ExpectedValue: func() chariot.Value {
				arr := chariot.NewArray()
				arr.Append(chariot.Number(42))
				arr.Append(chariot.Bool(true))
				return arr
			}(),
		},
	}

	RunTestCases(t, tests)
}

// TestPolymorphicReverse tests reverse() function on strings and arrays
func TestPolymorphicReverse(t *testing.T) {
	tests := []TestCase{
		// String reverse tests
		{
			Name:          "String Reverse - Basic",
			Script:        []string{`reverse("hello")`},
			ExpectedValue: chariot.Str("olleh"),
		},
		{
			Name:          "String Reverse - With Spaces",
			Script:        []string{`reverse("hello world")`},
			ExpectedValue: chariot.Str("dlrow olleh"),
		},
		{
			Name:          "String Reverse - Single Character",
			Script:        []string{`reverse("a")`},
			ExpectedValue: chariot.Str("a"),
		},
		{
			Name:          "String Reverse - Empty String",
			Script:        []string{`reverse("")`},
			ExpectedValue: chariot.Str(""),
		},
		{
			Name:          "String Reverse - Unicode",
			Script:        []string{`reverse("héllo 世界")`},
			ExpectedValue: chariot.Str("界世 olléh"),
		},
		{
			Name:          "String Reverse - Numbers",
			Script:        []string{`reverse("12345")`},
			ExpectedValue: chariot.Str("54321"),
		},

		// Array reverse tests
		{
			Name: "Array Reverse - Numbers",
			Script: []string{
				`setq(arr, array(1, 2, 3, 4, 5))`,
				`reverse(arr)`,
			},
			ExpectedValue: func() chariot.Value {
				arr := chariot.NewArray()
				arr.Append(chariot.Number(5))
				arr.Append(chariot.Number(4))
				arr.Append(chariot.Number(3))
				arr.Append(chariot.Number(2))
				arr.Append(chariot.Number(1))
				return arr
			}(),
		},
		{
			Name: "Array Reverse - Strings",
			Script: []string{
				`setq(arr, array("apple", "banana", "cherry"))`,
				`reverse(arr)`,
			},
			ExpectedValue: func() chariot.Value {
				arr := chariot.NewArray()
				arr.Append(chariot.Str("cherry"))
				arr.Append(chariot.Str("banana"))
				arr.Append(chariot.Str("apple"))
				return arr
			}(),
		},
		{
			Name: "Array Reverse - Mixed Types",
			Script: []string{
				`setq(arr, array("hello", 42, true))`,
				`reverse(arr)`,
			},
			ExpectedValue: func() chariot.Value {
				arr := chariot.NewArray()
				arr.Append(chariot.Bool(true))
				arr.Append(chariot.Number(42))
				arr.Append(chariot.Str("hello"))
				return arr
			}(),
		},
		{
			Name: "Array Reverse - Single Element",
			Script: []string{
				`setq(arr, array("only"))`,
				`reverse(arr)`,
			},
			ExpectedValue: func() chariot.Value {
				arr := chariot.NewArray()
				arr.Append(chariot.Str("only"))
				return arr
			}(),
		},
		{
			Name: "Array Reverse - Empty Array",
			Script: []string{
				`setq(arr, array())`,
				`reverse(arr)`,
			},
			ExpectedValue: chariot.NewArray(),
		},
	}

	RunTestCases(t, tests)
}

// TestPolymorphicContains tests contains() function on strings and arrays
func TestPolymorphicContains(t *testing.T) {
	tests := []TestCase{
		// String contains tests
		{
			Name:          "String Contains - Found",
			Script:        []string{`contains("hello world", "world")`},
			ExpectedValue: chariot.Bool(true),
		},
		{
			Name:          "String Contains - Not Found",
			Script:        []string{`contains("hello world", "goodbye")`},
			ExpectedValue: chariot.Bool(false),
		},
		{
			Name:          "String Contains - Partial Match",
			Script:        []string{`contains("programming", "gram")`},
			ExpectedValue: chariot.Bool(true),
		},
		{
			Name:          "String Contains - Case Sensitive",
			Script:        []string{`contains("Hello", "hello")`},
			ExpectedValue: chariot.Bool(false),
		},
		{
			Name:          "String Contains - Empty Substring",
			Script:        []string{`contains("hello", "")`},
			ExpectedValue: chariot.Bool(true),
		},
		{
			Name:          "String Contains - Unicode",
			Script:        []string{`contains("héllo 世界", "世界")`},
			ExpectedValue: chariot.Bool(true),
		},
		{
			Name:          "String Contains - Self",
			Script:        []string{`contains("test", "test")`},
			ExpectedValue: chariot.Bool(true),
		},

		// Array contains tests
		{
			Name: "Array Contains - String Found",
			Script: []string{
				`setq(arr, array("apple", "banana", "cherry"))`,
				`contains(arr, "banana")`,
			},
			ExpectedValue: chariot.Bool(true),
		},
		{
			Name: "Array Contains - String Not Found",
			Script: []string{
				`setq(arr, array("apple", "banana", "cherry"))`,
				`contains(arr, "orange")`,
			},
			ExpectedValue: chariot.Bool(false),
		},
		{
			Name: "Array Contains - Number Found",
			Script: []string{
				`setq(arr, array(1, 2, 3, 4, 5))`,
				`contains(arr, 3)`,
			},
			ExpectedValue: chariot.Bool(true),
		},
		{
			Name: "Array Contains - Number Not Found",
			Script: []string{
				`setq(arr, array(1, 2, 3, 4, 5))`,
				`contains(arr, 10)`,
			},
			ExpectedValue: chariot.Bool(false),
		},
		{
			Name: "Array Contains - Boolean Found",
			Script: []string{
				`setq(arr, array(true, false, "test"))`,
				`contains(arr, true)`,
			},
			ExpectedValue: chariot.Bool(true),
		},
		{
			Name: "Array Contains - Mixed Types",
			Script: []string{
				`setq(arr, array("hello", 42, true, "world"))`,
				`contains(arr, 42)`,
			},
			ExpectedValue: chariot.Bool(true),
		},
		{
			Name: "Array Contains - Empty Array",
			Script: []string{
				`setq(arr, array())`,
				`contains(arr, "anything")`,
			},
			ExpectedValue: chariot.Bool(false),
		},
	}

	RunTestCases(t, tests)
}

// TestSplitJoinFunctions tests split() and join() functions
func TestSplitJoinFunctions(t *testing.T) {
	tests := []TestCase{
		// String split tests
		{
			Name:   "Split - Comma Separated",
			Script: []string{`split("apple,banana,cherry", ",")`},
			ExpectedValue: func() chariot.Value {
				arr := chariot.NewArray()
				arr.Append(chariot.Str("apple"))
				arr.Append(chariot.Str("banana"))
				arr.Append(chariot.Str("cherry"))
				return arr
			}(),
		},
		{
			Name:   "Split - Space Separated",
			Script: []string{`split("hello world test", " ")`},
			ExpectedValue: func() chariot.Value {
				arr := chariot.NewArray()
				arr.Append(chariot.Str("hello"))
				arr.Append(chariot.Str("world"))
				arr.Append(chariot.Str("test"))
				return arr
			}(),
		},
		{
			Name:   "Split - No Delimiter Found",
			Script: []string{`split("hello", ",")`},
			ExpectedValue: func() chariot.Value {
				arr := chariot.NewArray()
				arr.Append(chariot.Str("hello"))
				return arr
			}(),
		},
		{
			Name:   "Split - Empty String",
			Script: []string{`split("", ",")`},
			ExpectedValue: func() chariot.Value {
				arr := chariot.NewArray()
				arr.Append(chariot.Str(""))
				return arr
			}(),
		},
		{
			Name:   "Split - Multiple Character Delimiter",
			Script: []string{`split("one::two::three", "::")`},
			ExpectedValue: func() chariot.Value {
				arr := chariot.NewArray()
				arr.Append(chariot.Str("one"))
				arr.Append(chariot.Str("two"))
				arr.Append(chariot.Str("three"))
				return arr
			}(),
		},

		// Array join tests
		{
			Name: "Join - String Array with Comma",
			Script: []string{
				`setq(arr, array("apple", "banana", "cherry"))`,
				`join(arr, ",")`,
			},
			ExpectedValue: chariot.Str("apple,banana,cherry"),
		},
		{
			Name: "Join - Mixed Array with Pipe",
			Script: []string{
				`setq(arr, array("hello", 42, true))`,
				`join(arr, " | ")`,
			},
			ExpectedValue: chariot.Str("hello | 42 | true"),
		},
		{
			Name: "Join - Single Element",
			Script: []string{
				`setq(arr, array("only"))`,
				`join(arr, ",")`,
			},
			ExpectedValue: chariot.Str("only"),
		},
		{
			Name: "Join - Empty Array",
			Script: []string{
				`setq(arr, array())`,
				`join(arr, ",")`,
			},
			ExpectedValue: chariot.Str(""),
		},
		{
			Name: "Join - Numbers with Dash",
			Script: []string{
				`setq(arr, array(1, 2, 3, 4))`,
				`join(arr, "-")`,
			},
			ExpectedValue: chariot.Str("1-2-3-4"),
		},

		// Round-trip tests (split then join)
		{
			Name: "Round Trip - CSV Data",
			Script: []string{
				`setq(original, "red,green,blue")`,
				`setq(parts, split(original, ","))`,
				`join(parts, ",")`,
			},
			ExpectedValue: chariot.Str("red,green,blue"),
		},
		{
			Name: "Round Trip - Path Data",
			Script: []string{
				`setq(original, "home/user/documents")`,
				`setq(parts, split(original, "/"))`,
				`join(parts, "/")`,
			},
			ExpectedValue: chariot.Str("home/user/documents"),
		},
	}

	RunTestCases(t, tests)
}

// TestPolymorphicErrorHandling tests error cases for polymorphic functions
func TestPolymorphicErrorHandling(t *testing.T) {
	tests := []TestCase{
		// Type error tests
		{
			Name:           "Slice - Unsupported Type",
			Script:         []string{`slice(42, 0, 2)`},
			ExpectedError:  true,
			ErrorSubstring: "slice not supported for type",
		},
		{
			Name:           "Reverse - Unsupported Type",
			Script:         []string{`reverse(123)`},
			ExpectedError:  true,
			ErrorSubstring: "reverse not supported for type",
		},
		{
			Name:           "Contains - Unsupported Type",
			Script:         []string{`contains(true, "test")`},
			ExpectedError:  true,
			ErrorSubstring: "contains not supported for type",
		},
		{
			Name:           "Split - Non-String",
			Script:         []string{`split(array(1,2,3), ",")`},
			ExpectedError:  true,
			ErrorSubstring: "split only supported for strings",
		},
		{
			Name:           "Join - Non-Array",
			Script:         []string{`join("hello", ",")`},
			ExpectedError:  true,
			ErrorSubstring: "join only supported for arrays",
		},

		// Argument count error tests
		{
			Name:           "Slice - Too Few Arguments",
			Script:         []string{`slice("hello")`},
			ExpectedError:  true,
			ErrorSubstring: "slice requires at least 2 arguments",
		},
		{
			Name:           "Reverse - Too Many Arguments",
			Script:         []string{`reverse("hello", "extra")`},
			ExpectedError:  true,
			ErrorSubstring: "reverse requires 1 argument",
		},
		{
			Name:           "Contains - Too Few Arguments",
			Script:         []string{`contains("hello")`},
			ExpectedError:  true,
			ErrorSubstring: "contains requires 2 arguments",
		},
		{
			Name:           "Split - Too Few Arguments",
			Script:         []string{`split("hello")`},
			ExpectedError:  true,
			ErrorSubstring: "split requires 2 arguments",
		},
		{
			Name: "Join - Too Few Arguments",
			Script: []string{
				`setq(arr, array("a", "b"))`,
				`join(arr)`,
			},
			ExpectedError:  true,
			ErrorSubstring: "join requires 2 arguments",
		},

		// Type mismatch in arguments
		{
			Name:           "Slice - Non-Number Index",
			Script:         []string{`slice("hello", "invalid", 3)`},
			ExpectedError:  true,
			ErrorSubstring: "start index must be a number",
		},
		{
			Name: "Join - Non-String Delimiter",
			Script: []string{
				`setq(arr, array("a", "b"))`,
				`join(arr, 123)`,
			},
			ExpectedError:  true,
			ErrorSubstring: "delimiter must be a string",
		},
	}

	RunTestCases(t, tests)
}

// TestPolymorphicIntegration tests polymorphic functions working together
func TestPolymorphicIntegration(t *testing.T) {
	tests := []TestCase{
		{
			Name: "Complex String Processing",
			Script: []string{
				`setq(text, "apple,banana,cherry")`,
				`setq(fruits, split(text, ","))`,
				`setq(reversed, reverse(fruits))`,
				`setq(firstReversed, slice(getAt(reversed, 0), 0, 6))`,
				`firstReversed`,
			},
			ExpectedValue: func() chariot.Str {
				return chariot.Str("cherry")
			}(),
		},
		{
			Name: "Array Manipulation Pipeline",
			Script: []string{
				`setq(numbers, array(1, 2, 3, 4, 5))`,
				`setq(subset, slice(numbers, 1, 4))`,
				`setq(reversed, reverse(subset))`,
				`setq(hasThree, contains(reversed, 3))`,
				`hasThree`,
			},
			ExpectedValue: chariot.Bool(true),
		},
		{
			Name: "String to Array to String",
			Script: []string{
				`setq(sentence, "hello beautiful world")`,
				`setq(words, split(sentence, " "))`,
				`setq(reversed, reverse(words))`,
				`setq(result, join(reversed, " "))`,
				`result`,
			},
			ExpectedValue: chariot.Str("world beautiful hello"),
		},
		{
			Name: "Mixed Type Array Processing",
			Script: []string{
				`setq(mixed, array("start", 42, true, "end"))`,
				`setq(sliced, slice(mixed, 1, 3))`,
				`setq(joined, join(sliced, " -> "))`,
				`joined`,
			},
			ExpectedValue: chariot.Str("42 -> true"),
		},
		{
			Name: "Conditional Processing with Contains",
			Script: []string{
				`declare(result,'V')`,
				`setq(data, "error,warning,info,debug")`,
				`setq(levels, split(data, ","))`,
				`if (contains(levels, "error")) {`,
				`    setq(result, "has_error")`,
				`} else {`,
				`    setq(result, "no_error")`,
				`}`,
				`result`,
			},
			ExpectedValue: chariot.Str("has_error"),
		},
	}

	RunTestCases(t, tests)
}

// TestGetPropPolymorphic tests getProp() function on all supported object types
func TestGetPropPolymorphic(t *testing.T) {
	tests := []TestCase{
		// map[string]Value
		{
			Name: "getProp - map[string]Value with existing property",
			Script: []string{
				`declare(myMap, 'M')`,
				`setq(myMap, map())`,
				`setProp(myMap, "name", "test")`,
				`setProp(myMap, "count", 42)`,
				`getProp(myMap, "name")`,
			},
			ExpectedValue: chariot.Str("test"),
		},
		{
			Name: "getProp - map[string]Value with missing property returns DBNull",
			Script: []string{
				`declare(myMap, 'M')`,
				`setq(myMap, map())`,
				`setProp(myMap, "name", "test")`,
				`setq(result, getProp(myMap, "missing"))`,
				`equal(result, DBNull)`,
			},
			ExpectedValue: chariot.Bool(true),
		},

		// map[string]interface{} (from inspectRuntime)
		{
			Name: "getProp - map[string]interface{} with existing property",
			Script: []string{
				`setq(runtime, inspectRuntime())`,
				`setq(globals, getProp(runtime, "globals"))`,
				`typeOf(globals)`,
			},
			ExpectedValue: chariot.Str("M"),
		},
		{
			Name: "getProp - map[string]interface{} with missing property returns DBNull",
			Script: []string{
				`setq(testMap, map())`,
				`setProp(testMap, "exists", "value")`,
				`setq(result, getProp(testMap, "missing"))`,
				`equal(result, DBNull)`,
			},
			ExpectedValue: chariot.Bool(true),
		},

		// SimpleJSON
		{
			Name: "getProp - SimpleJSON with existing property",
			Script: []string{
				`setq(json, parseJSON('{"name":"test","value":123}'))`,
				`getProp(json, "name")`,
			},
			ExpectedValue: chariot.Str("test"),
		},
		{
			Name: "getProp - SimpleJSON with missing property returns DBNull",
			Script: []string{
				`setq(json, parseJSON('{"name":"test"}'))`,
				`setq(result, getProp(json, "missing"))`,
				`equal(result, DBNull)`,
			},
			ExpectedValue: chariot.Bool(true),
		},

		// MapNode
		{
			Name: "getProp - MapNode with existing property",
			Script: []string{
				`setq(mapNode, mapNode("test"))`,
				`setProp(mapNode, "key1", "value1")`,
				`getProp(mapNode, "key1")`,
			},
			ExpectedValue: chariot.Str("value1"),
		},
		{
			Name: "getProp - MapNode with missing property",
			Script: []string{
				`setq(mapNode, mapNode("test"))`,
				`setProp(mapNode, "key1", "value1")`,
				`getProp(mapNode, "missing")`,
			},
			ExpectedError:  true,
			ErrorSubstring: "property 'missing' not found",
		},

		// MapValue
		{
			Name: "getProp - MapValue with existing property",
			Script: []string{
				`setq(mapVal, map())`,
				`setProp(mapVal, "prop1", "val1")`,
				`getProp(mapVal, "prop1")`,
			},
			ExpectedValue: chariot.Str("val1"),
		},
		{
			Name: "getProp - MapValue with missing property returns DBNull",
			Script: []string{
				`setq(mapVal, map())`,
				`setProp(mapVal, "prop1", "val1")`,
				`setq(result, getProp(mapVal, "missing"))`,
				`equal(result, DBNull)`,
			},
			ExpectedValue: chariot.Bool(true),
		},

		// JSONNode
		{
			Name: "getProp - JSONNode with existing property",
			Script: []string{
				`setq(node, jsonNode("test"))`,
				`setAttribute(node, "attr1", "value1")`,
				`getProp(node, "attr1")`,
			},
			ExpectedValue: chariot.Str("value1"),
		},

		// XMLNode
		{
			Name: "getProp - XMLNode with existing attribute",
			Script: []string{
				`setq(xml, xmlNode("root"))`,
				`setAttribute(xml, "id", "123")`,
				`getProp(xml, "id")`,
			},
			ExpectedValue: chariot.Str("123"),
		},

		// TreeNodeImpl
		{
			Name: "getProp - TreeNodeImpl with existing attribute",
			Script: []string{
				`setq(tree, treeNode("root"))`,
				`setAttribute(tree, "label", "myLabel")`,
				`getProp(tree, "label")`,
			},
			ExpectedValue: chariot.Str("myLabel"),
		},

		// Plan
		{
			Name: "getProp - Plan with name property",
			Script: []string{
				`declare(triggerFn,'F', func(){ True })`,
				`declare(guardFn,'F', func(){ True })`,
				`declare(stepFn,'F', func(){ setq(x, 1); True })`,
				`declare(steps,'A', array(stepFn))`,
				`declare(dropFn,'F', func(){ False })`,
				`declare(MyTestPlan,'P', plan("MyTestPlan", array(), triggerFn, guardFn, steps, dropFn))`,
				`getProp(MyTestPlan, "name")`,
			},
			ExpectedValue: chariot.Str("MyTestPlan"),
		},
		{
			Name: "getProp - Plan with _type property",
			Script: []string{
				`declare(triggerFn,'F', func(){ True })`,
				`declare(guardFn,'F', func(){ True })`,
				`declare(stepFn,'F', func(){ setq(x, 1); True })`,
				`declare(steps,'A', array(stepFn))`,
				`declare(dropFn,'F', func(){ False })`,
				`declare(TypeTestPlan,'P', plan("TypeTestPlan", array(), triggerFn, guardFn, steps, dropFn))`,
				`getProp(TypeTestPlan, "_type")`,
			},
			ExpectedValue: chariot.Str("plan"),
		},

		// Nested map[string]interface{} from inspectRuntime
		{
			Name: "getProp - Nested map[string]interface{} from globals",
			Script: []string{
				`declare(triggerFn,'F', func(){ True })`,
				`declare(guardFn,'F', func(){ True })`,
				`declare(stepFn,'F', func(){ setq(x, 1); True })`,
				`declare(steps,'A', array(stepFn))`,
				`declare(dropFn,'F', func(){ False })`,
				`declareGlobal(TestPlanForInspect,'P', plan("TestPlanForInspect", array(), triggerFn, guardFn, steps, dropFn))`,
				`setq(runtime, inspectRuntime())`,
				`setq(globals, getProp(runtime, "globals"))`,
				`setq(planRef, getProp(globals, "TestPlanForInspect"))`,
				`setq(planType, getProp(planRef, "_type"))`,
				`planType`,
			},
			ExpectedValue: chariot.Str("plan"),
		},
		{
			Name: "getProp - map[string]interface{} missing property should return DBNull not error",
			Script: []string{
				`declare(triggerFn,'F', func(){ True })`,
				`declare(guardFn,'F', func(){ True })`,
				`declare(stepFn,'F', func(){ setq(x, 1); True })`,
				`declare(steps,'A', array(stepFn))`,
				`declare(dropFn,'F', func(){ False })`,
				`declareGlobal(PlanForMissingProp,'P', plan("PlanForMissingProp", array(), triggerFn, guardFn, steps, dropFn))`,
				`setq(runtime, inspectRuntime())`,
				`setq(globals, getProp(runtime, "globals"))`,
				`setq(planRef, getProp(globals, "PlanForMissingProp"))`,
				`setq(result, getProp(planRef, "nonexistent"))`,
				`equal(result, DBNull)`,
			},
			ExpectedValue: chariot.Bool(true),
		},
	}

	RunTestCases(t, tests)
}
