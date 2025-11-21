package tests

import (
	"os"
	"strings"
	"testing"

	"github.com/bhouse1273/chariot-ecosystem/services/go-chariot/chariot"
	cfg "github.com/bhouse1273/chariot-ecosystem/services/go-chariot/configs"
)

// TestJSONNodes tests JSON node creation and manipulation
func TestJSONNodes(t *testing.T) {
	tests := []TestCase{
		{
			Name: "Create Empty JSON Node",
			Script: []string{
				`nodeToString(jsonNode())`,
			},
			ExpectedValue: chariot.Str("{}"),
		},
		{
			Name: "Create Populated JSON Node",
			Script: []string{
				`nodeToString(jsonNode('{"name":"test","value":42}'))`,
			},
			ExpectedValue: chariot.Str(`{"name":"test","value":42}`),
		},
		{
			Name: "Access JSON Property",
			Script: []string{
				`setq(node, jsonNode('{"name":"test","value":42}'))`,
				`getProp(node, 'name')`,
			},
			ExpectedValue: chariot.Str("test"),
		},
		{
			Name: "Modify JSON Property",
			Script: []string{
				`setq(node, jsonNode('{"name":"test"}'))`,
				`setProp(node, 'name', 'modified')`,
				`getProp(node, 'name')`,
			},
			ExpectedValue: chariot.Str("modified"),
		},
		{
			Name: "JSON Nested Property Access",
			Script: []string{
				`setq(node, jsonNode('{"user":{"name":"test","id":123}}'))`,
				`getProp(node, 'user.name')`,
			},
			ExpectedValue: chariot.Str("test"),
		},
		{
			Name: "Convert Node to String",
			Script: []string{
				`setq(node, jsonNode('{"name":"test"}'))`,
				`nodeToString(node)`,
			},
			ExpectedValue: chariot.Str(`{"name":"test"}`),
		},
		{
			Name: "Create New Property",
			Script: []string{
				`setq(node, jsonNode('{}'))`,
				`setProp(node, 'newProp', 'value')`,
				`getProp(node, 'newProp')`,
			},
			ExpectedValue: chariot.Str("value"),
		},
		{
			Name: "Create Nested Property",
			Script: []string{
				`setq(node, jsonNode('{}'))`,
				`setProp(node, 'user.name', 'test')`,
				`getProp(node, 'user.name')`,
			},
			ExpectedValue: chariot.Str("test"),
		},
	}

	RunTestCases(t, tests)
}

// TestJSONArrayFeatures tests JSON array manipulation
func TestJSONArrayFeatures(t *testing.T) {
	tests := []TestCase{
		{
			Name: "Create JSON Array",
			Script: []string{
				`nodeToString(jsonNode('[1,2,3]'))`,
			},
			ExpectedValue: chariot.Str(`[1,2,3]`),
		},
		{
			Name: "Access Array Element by Index",
			Script: []string{
				`setq(node, jsonNode('[10,20,30]'))`,
				`getProp(node, '1')`,
			},
			ExpectedValue: chariot.Number(20),
		},
		{
			Name: "Modify Array Element",
			Script: []string{
				`setq(node, jsonNode('[1,2,3]'))`,
				`setProp(node, '0', 99)`,
				`getProp(node, '0')`,
			},
			ExpectedValue: chariot.Number(99),
		},
		{
			Name: "Mixed Array with Objects",
			Script: []string{
				`setq(node, jsonNode('[{"name":"John"},{"name":"Jane"}]'))`,
				`getProp(node, '0.name')`,
			},
			ExpectedValue: chariot.Str("John"),
		},
		{
			Name: "Complex Array Property Access",
			Script: []string{
				`setq(node, jsonNode('{"users":[{"id":1,"name":"Alice"},{"id":2,"name":"Bob"}]}'))`,
				`getProp(node, 'users.1.name')`,
			},
			ExpectedValue: chariot.Str("Bob"),
		},
	}

	RunTestCases(t, tests)
}

// TestJSONComplexStructures tests deeply nested and complex JSON structures
func TestJSONComplexStructures(t *testing.T) {
	tests := []TestCase{
		{
			Name: "Deep Nested Object Access",
			Script: []string{
				`setq(node, jsonNode('{"level1":{"level2":{"level3":{"value":"deep"}}}}'))`,
				`getProp(node, 'level1.level2.level3.value')`,
			},
			ExpectedValue: chariot.Str("deep"),
		},
		{
			Name: "Create Deep Nested Structure",
			Script: []string{
				`setq(node, jsonNode('{}'))`,
				`setProp(node, 'app.config.database.host', 'localhost')`,
				`getProp(node, 'app.config.database.host')`,
			},
			ExpectedValue: chariot.Str("localhost"),
		},
		{
			Name: "Mixed Data Types",
			Script: []string{
				`setq(node, jsonNode('{"string":"text","number":42,"boolean":true,"null":null}'))`,
				`getProp(node, 'number')`,
			},
			ExpectedValue: chariot.Number(42),
		},
		{
			Name: "Object with Array of Objects",
			Script: []string{
				`setq(node, jsonNode('{"team":{"members":[{"name":"Alice","role":"dev"},{"name":"Bob","role":"qa"}]}}'))`,
				`getProp(node, 'team.members.0.role')`,
			},
			ExpectedValue: chariot.Str("dev"),
		},
		{
			Name: "Modify Complex Structure",
			Script: []string{
				`setq(node, jsonNode('{"settings":{"ui":{"theme":"dark"}}}'))`,
				`setProp(node, 'settings.ui.theme', 'light')`,
				`setProp(node, 'settings.ui.language', 'en')`,
				`getProp(node, 'settings.ui.language')`,
			},
			ExpectedValue: chariot.Str("en"),
		},
	}

	RunTestCases(t, tests)
}

// TestJSONDataTypes tests various JSON data type handling
func TestJSONDataTypes(t *testing.T) {
	tests := []TestCase{
		{
			Name: "Boolean Values",
			Script: []string{
				`setq(node, jsonNode('{"enabled":true,"disabled":false}'))`,
				`getProp(node, 'enabled')`,
			},
			ExpectedValue: chariot.Bool(true),
		},
		{
			Name: "Null Values",
			Script: []string{
				`setq(node, jsonNode('{"value":null}'))`,
				`getProp(node, 'value')`,
			},
			ExpectedValue: chariot.DBNull,
		},
		{
			Name: "Number Types",
			Script: []string{
				`setq(node, jsonNode('{"integer":42,"float":3.14,"negative":-5}'))`,
				`getProp(node, 'float')`,
			},
			ExpectedValue: chariot.Number(3.14),
		},
		{
			Name: "Empty String vs Null",
			Script: []string{
				`setq(node, jsonNode('{"empty":"","null":null}'))`,
				`getProp(node, 'empty')`,
			},
			ExpectedValue: chariot.Str(""),
		},
		{
			Name: "Unicode and Special Characters",
			Script: []string{
				`setq(node, jsonNode('{"unicode":"Héllo 世界","special":"@#$%^&*()"}'))`,
				`getProp(node, 'unicode')`,
			},
			ExpectedValue: chariot.Str("Héllo 世界"),
		},
	}

	RunTestCases(t, tests)
}

// TestJSONErrorHandling tests error cases and edge conditions
func TestJSONErrorHandling(t *testing.T) {
	tests := []TestCase{
		{
			Name: "Invalid JSON Syntax",
			Script: []string{
				`jsonNode('{"invalid": json}')`,
			},
			ExpectedError: true,
		},
		{
			Name: "Access Non-existent Property",
			Script: []string{
				`setq(node, jsonNode('{"existing":"value"}'))`,
				`getProp(node, 'nonexistent')`,
			},
			ExpectedValue: chariot.DBNull,
		},
		{
			Name: "Access Property on Null",
			Script: []string{
				`setq(node, jsonNode('{"value":null}'))`,
				`getProp(node, 'value.subprop')`,
			},
			ExpectedValue: chariot.DBNull,
		},
		{
			Name: "Deep Path on Primitive",
			Script: []string{
				`setq(node, jsonNode('{"number":42}'))`,
				`getProp(node, 'number.invalid')`,
			},
			ExpectedValue: chariot.DBNull,
		},
		{
			Name: "Empty Path",
			Script: []string{
				`setq(node, jsonNode('{"test":"value"}'))`,
				`getProp(node, '')`,
			},
			ExpectedError: true,
		},
	}

	// ✅ USE THE STANDARD TEST RUNNER!
	RunTestCases(t, tests)
}

// TestJSONLargeStructures tests performance with larger JSON documents
func TestJSONLargeStructures(t *testing.T) {
	tests := []TestCase{
		{
			Name: "Large Array Access",
			Script: []string{
				`setq(node, jsonNode('[0,1,2,3,4,5,6,7,8,9,10,11,12,13,14,15,16,17,18,19]'))`,
				`getProp(node, '15')`,
			},
			ExpectedValue: chariot.Number(15),
		},
		{
			Name: "Object with Many Properties",
			Script: []string{
				`setq(node, jsonNode('{"prop1":"val1","prop2":"val2","prop3":"val3","prop4":"val4","prop5":"val5"}'))`,
				`setProp(node, 'newProp', 'newValue')`,
				`getProp(node, 'newProp')`,
			},
			ExpectedValue: chariot.Str("newValue"),
		},
		{
			Name: "Multi-level Object Creation",
			Script: []string{
				`setq(node, jsonNode('{}'))`,
				`setProp(node, 'a.b.c.d.e.f', 'deep_value')`,
				`getProp(node, 'a.b.c.d.e.f')`,
			},
			ExpectedValue: chariot.Str("deep_value"),
		},
	}

	RunTestCases(t, tests)
}

// TestJSONIntegration tests JSON with other Chariot features
func TestJSONIntegration(t *testing.T) {
	tests := []TestCase{
		{
			Name: "JSON with String Functions",
			Script: []string{
				`setq(node, jsonNode('{"name":"john doe"}'))`,
				`setq(name, getProp(node, 'name'))`,
				`upper(name)`,
			},
			ExpectedValue: chariot.Str("JOHN DOE"),
		},
		{
			Name: "JSON with Math Functions",
			Script: []string{
				`setq(node, jsonNode('{"radius":5}'))`,
				`setq(r, getProp(node, 'radius'))`,
				`setq(area, mul(r, r))`,
				`mul(area, 3.14159)`,
			},
			ExpectedValue: chariot.Number(78.53975),
		},
		{
			Name: "Build JSON from Variables",
			Script: []string{
				`setq(node, jsonNode('{}'))`,
				`setq(username, 'alice')`,
				`setq(age, 30)`,
				`setProp(node, 'user.name', username)`,
				`setProp(node, 'user.age', age)`,
				`getProp(node, 'user.name')`,
			},
			ExpectedValue: chariot.Str("alice"),
		},
		{
			Name: "JSON Property as Condition",
			Script: []string{
				`declare(result, 'V')`,
				`setq(node, jsonNode('{"enabled":true,"count":5}'))`,
				`setq(enabled, getProp(node, 'enabled'))`,
				`setq(count, getProp(node, 'count'))`,
				`if (enabled) {`,
				`	if (bigger(count, 3)) {`,
				`		setq(result, 'active')`,
				`	} else {`,
				`		setq(result, 'waiting')`,
				`	}`,
				`} else {`,
				`	setq(result, 'disabled')`,
				`}`,
				`result`,
			},
			ExpectedValue: chariot.Str("active"),
		},
	}

	RunTestCases(t, tests)
}

// TestJSONNodesAdvanced - Direct object validation
func TestJSONNodesAdvanced(t *testing.T) {
	tests := []TestCase{
		{
			Name:          "Create Empty JSON Node - Direct Validation",
			Script:        []string{`jsonNode()`},
			ExpectedValue: nil, // Will be validated in custom logic below
		},
	}

	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			rt := chariot.NewRuntime()
			chariot.RegisterAll(rt)
			tscript := strings.Join(test.Script, "\n")
			result, err := rt.ExecProgram(tscript)
			if err != nil {
				t.Fatalf("Script evaluation failed: %v", err)
			}

			// Custom validation for JSONNode
			if test.Name == "Create Empty JSON Node - Direct Validation" {
				jsonNode, ok := result.(*chariot.JSONNode)
				if !ok {
					t.Fatalf("Expected JSONNode, got %T", result)
				}

				jsonStr, err := jsonNode.ToJSON()
				if err != nil {
					t.Fatalf("Failed to convert to JSON: %v", err)
				}

				if jsonStr != "{}" {
					t.Errorf("Expected '{}', got '%s'", jsonStr)
				}
			}
		})
	}
}

// Add to json_node_test.go
func TestJSONFileOperations(t *testing.T) {
	initCouchbaseConfig()

	tests := []TestCase{
		{
			Name: "Save and Load JSON File",
			Script: []string{
				`setq(config, jsonNode('{"database":{"host":"localhost","port":5432}}'))`,
				`saveJSON(config, 'test-output.json')`,
				`setq(loaded, loadJSON('test-output.json'))`,
				`getProp(loaded, 'database.host')`,
			},
			ExpectedValue: chariot.Str("localhost"),
		},
		{
			Name: "JSON to YAML Conversion",
			Script: []string{
				`setq(data, jsonNode('{"name":"test","value":42}'))`,
				`setq(yamlStr, jsonToYAML(data))`,
				`setq(converted, yamlToJSON(yamlStr))`,
				`getProp(converted, 'value')`,
			},
			ExpectedValue: chariot.Number(42),
		},
		{
			Name: "File Existence Check",
			Script: []string{
				`setq(config, jsonNode('{"test":true}'))`,
				`saveJSON(config, 'temp-test.json')`,
				`fileExists('temp-test.json')`,
			},
			ExpectedValue: chariot.Bool(true),
		},
	}

	RunTestCases(t, tests)

	// format fullPath to ensure it works correctly
	folder := cfg.ChariotConfig.DataPath
	if folder == "" {
		folder = "."
	}
	fullPath1 := folder + "/test-output.json"
	fullPath2 := folder + "/temp-test.json"

	// Cleanup test files
	os.Remove(fullPath1)
	os.Remove(fullPath2)
}

// TestJSONRawOperations tests raw JSON string file operations
func TestJSONRawOperations(t *testing.T) {
	initCouchbaseConfig()

	tests := []TestCase{
		{
			Name: "Load JSON Raw String",
			Script: []string{
				`writeFile('test-raw.json', '{"name":"test","value":123}')`,
				`setq(raw, loadJSONRaw('test-raw.json'))`,
				`contains(raw, '"name"')`,
			},
			ExpectedValue: chariot.Bool(true),
		},
		{
			Name: "Save JSON Raw String",
			Script: []string{
				`setq(jsonStr, '{"key":"value","number":42}')`,
				`saveJSONRaw(jsonStr, 'test-raw-save.json')`,
				`setq(loaded, loadJSONRaw('test-raw-save.json'))`,
				`contains(loaded, '"key":"value"')`,
			},
			ExpectedValue: chariot.Bool(true),
		},
		{
			Name: "Modify Raw JSON String",
			Script: []string{
				`setq(original, '{"setting":"old"}')`,
				`saveJSONRaw(original, 'test-modify.json')`,
				`setq(loaded, loadJSONRaw('test-modify.json'))`,
				`setq(modified, replace(loaded, 'old', 'new'))`,
				`saveJSONRaw(modified, 'test-modify.json')`,
				`setq(final, loadJSONRaw('test-modify.json'))`,
				`contains(final, '"setting":"new"')`,
			},
			ExpectedValue: chariot.Bool(true),
		},
		{
			Name: "Load Raw JSON and Parse",
			Script: []string{
				`setq(jsonStr, '{"app":"MyApp","version":2}')`,
				`saveJSONRaw(jsonStr, 'test-parse.json')`,
				`setq(raw, loadJSONRaw('test-parse.json'))`,
				`setq(node, parseJSON(raw, 'config'))`,
				`getProp(node, 'app')`,
			},
			ExpectedValue: chariot.Str("MyApp"),
		},
		{
			Name: "Save Pretty-Printed JSON Raw",
			Script: []string{
				`setq(prettyJSON, '{\n  "name": "test",\n  "nested": {\n    "value": 42\n  }\n}')`,
				`saveJSONRaw(prettyJSON, 'test-pretty.json')`,
				`setq(loaded, loadJSONRaw('test-pretty.json'))`,
				`contains(loaded, '  "name"')`,
			},
			ExpectedValue: chariot.Bool(true),
		},
	}

	RunTestCases(t, tests)

	// Cleanup
	folder := cfg.ChariotConfig.DataPath
	if folder == "" {
		folder = "."
	}
	os.Remove(folder + "/test-raw.json")
	os.Remove(folder + "/test-raw-save.json")
	os.Remove(folder + "/test-modify.json")
	os.Remove(folder + "/test-parse.json")
	os.Remove(folder + "/test-pretty.json")
}
