// tests/format_conversion_test.go
package tests

import (
	"os"
	"testing"

	"github.com/bhouse1273/chariot-ecosystem/services/go-chariot/chariot"
	cfg "github.com/bhouse1273/chariot-ecosystem/services/go-chariot/configs"
)

func TestJSONToYAML(t *testing.T) {
	initCouchbaseConfig()

	tests := []TestCase{
		{
			Name: "Convert JSON String to YAML",
			Script: []string{
				`setq(jsonStr, '{"name":"test","value":42}')`,
				`setq(jsonNode, parseJSON(jsonStr))`,
				`setq(yamlStr, jsonToYAML(jsonNode))`,
				`contains(yamlStr, 'name: test')`,
			},
			ExpectedValue: chariot.Bool(true),
		},
		{
			Name: "Convert Complex JSON to YAML",
			Script: []string{
				`setq(jsonStr, '{"database":{"host":"localhost","port":5432},"cache":{"enabled":true}}')`,
				`setq(jsonNode, parseJSON(jsonStr))`,
				`setq(yamlStr, jsonToYAML(jsonNode))`,
				`contains(yamlStr, 'database:')`,
			},
			ExpectedValue: chariot.Bool(true),
		},
		{
			Name: "Convert JSON Array to YAML",
			Script: []string{
				`setq(jsonStr, '["item1","item2","item3"]')`,
				`setq(jsonNode, parseJSON(jsonStr))`,
				`setq(yamlStr, jsonToYAML(jsonNode))`,
				`contains(yamlStr, 'item1')`,
			},
			ExpectedValue: chariot.Bool(true),
		},
	}

	RunTestCases(t, tests)
}

func TestYAMLToJSON(t *testing.T) {
	initCouchbaseConfig()

	tests := []TestCase{
		{
			Name: "Convert YAML String to JSON",
			Script: []string{
				`setq(yamlStr, 'name: test\nvalue: 42\n')`,
				`setq(jsonNode, yamlToJSON(yamlStr))`,
				`setq(jsonStr, toString(jsonNode))`,
				`contains(jsonStr, '"name"')`,
			},
			ExpectedValue: chariot.Bool(true),
		},
		{
			Name: "Convert Nested YAML to JSON",
			Script: []string{
				`setq(yamlStr, 'server:\n  host: localhost\n  port: 8080\n')`,
				`setq(jsonNode, yamlToJSON(yamlStr))`,
				`setq(jsonStr, toString(jsonNode))`,
				`contains(jsonStr, '"server"')`,
			},
			ExpectedValue: chariot.Bool(true),
		},
		{
			Name: "Round-trip YAML to JSON to YAML",
			Script: []string{
				`setq(originalYAML, 'key: value\ncount: 5\n')`,
				`setq(jsonNode, yamlToJSON(originalYAML))`,
				`setq(backToYAML, jsonToYAML(jsonNode))`,
				`contains(backToYAML, 'key')`,
			},
			ExpectedValue: chariot.Bool(true),
		},
	}

	RunTestCases(t, tests)
}

func TestJSONToYAMLNode(t *testing.T) {
	initCouchbaseConfig()

	tests := []TestCase{
		{
			Name: "Convert JSON String to YAML TreeNode",
			Script: []string{
				`setq(jsonStr, '{"app":"MyApp","version":"1.0"}')`,
				`setq(jsonNode, parseJSON(jsonStr))`,
				`setq(node, jsonToYAMLNode(jsonNode))`,
				`getProp(node, 'app')`,
			},
			ExpectedValue: chariot.Str("MyApp"),
		},
		{
			Name: "Convert Complex JSON to TreeNode",
			Script: []string{
				`setq(jsonStr, '{"database":{"host":"db.example.com","port":3306}}')`,
				`setq(jsonNode, parseJSON(jsonStr))`,
				`setq(node, jsonToYAMLNode(jsonNode))`,
				`getProp(node, 'database.host')`,
			},
			ExpectedValue: chariot.Str("db.example.com"),
		},
		{
			Name: "Save Converted Node as YAML",
			Script: []string{
				`setq(jsonStr, '{"setting":"value","enabled":true}')`,
				`setq(jsonNode, parseJSON(jsonStr))`,
				`setq(node, jsonToYAMLNode(jsonNode))`,
				`saveYAML(node, 'test-converted.yaml')`,
				`setq(loaded, loadYAML('test-converted.yaml'))`,
				`getProp(loaded, 'enabled')`,
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
	os.Remove(folder + "/test-converted.yaml")
}

func TestYAMLToJSONNode(t *testing.T) {
	initCouchbaseConfig()

	tests := []TestCase{
		{
			Name: "Convert YAML String to JSON TreeNode",
			Script: []string{
				`setq(yamlStr, 'name: TestApp\nversion: 2.0\n')`,
				`setq(node, yamlToJSON(yamlStr))`,
				`getProp(node, 'name')`,
			},
			ExpectedValue: chariot.Str("TestApp"),
		},
		{
			Name: "Convert Nested YAML to JSONNode",
			Script: []string{
				`setq(yamlStr, 'server:\n  host: localhost\n  port: 8080\n')`,
				`setq(node, yamlToJSON(yamlStr))`,
				`getProp(node, 'server.port')`,
			},
			ExpectedValue: chariot.Number(8080),
		},
		{
			Name: "Save Converted Node as JSON",
			Script: []string{
				`setq(yamlStr, 'id: 123\nactive: true\n')`,
				`setq(node, yamlToJSON(yamlStr))`,
				`saveJSON(node, 'test-yaml-to-json.json')`,
				`setq(loaded, loadJSON('test-yaml-to-json.json'))`,
				`getProp(loaded, 'active')`,
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
	os.Remove(folder + "/test-yaml-to-json.json")
}

func TestConvertJSONFileToYAML(t *testing.T) {
	initCouchbaseConfig()

	tests := []TestCase{
		{
			Name: "Convert JSON File to YAML File",
			Script: []string{
				`writeFile('test-source.json', '{"name":"Config","version":1,"enabled":true}')`,
				`convertJSONFileToYAML('test-source.json', 'test-target.yaml')`,
				`fileExists('test-target.yaml')`,
			},
			ExpectedValue: chariot.Bool(true),
		},
		{
			Name: "Verify JSON to YAML File Conversion Content",
			Script: []string{
				`writeFile('test-data.json', '{"database":{"host":"localhost","port":5432}}')`,
				`convertJSONFileToYAML('test-data.json', 'test-data.yaml')`,
				`setq(loaded, loadYAML('test-data.yaml'))`,
				`setName(loaded, 'data')`,
				`getProp(loaded, 'database.host')`,
			},
			ExpectedValue: chariot.Str("localhost"),
		},
		{
			Name: "Convert Complex JSON File",
			Script: []string{
				`writeFile('test-complex.json', '{"app":{"name":"MyApp","features":["auth","api","cache"]}}')`,
				`convertJSONFileToYAML('test-complex.json', 'test-complex.yaml')`,
				`setq(config, loadYAML('test-complex.yaml'))`,
				`setName(config, 'config')`,
				`getProp(config, 'app.name')`,
			},
			ExpectedValue: chariot.Str("MyApp"),
		},
	}

	RunTestCases(t, tests)

	// Cleanup
	folder := cfg.ChariotConfig.DataPath
	if folder == "" {
		folder = "."
	}
	os.Remove(folder + "/test-source.json")
	os.Remove(folder + "/test-target.yaml")
	os.Remove(folder + "/test-data.json")
	os.Remove(folder + "/test-data.yaml")
	os.Remove(folder + "/test-complex.json")
	os.Remove(folder + "/test-complex.yaml")
}

func TestConvertYAMLFileToJSON(t *testing.T) {
	initCouchbaseConfig()

	tests := []TestCase{
		{
			Name: "Convert YAML File to JSON File",
			Script: []string{
				`writeFile('test-source.yaml', 'name: Config\nversion: 1\nenabled: true\n')`,
				`convertYAMLFileToJSON('test-source.yaml', 'test-target.json')`,
				`fileExists('test-target.json')`,
			},
			ExpectedValue: chariot.Bool(true),
		},
		{
			Name: "Verify YAML to JSON File Conversion Content",
			Script: []string{
				`writeFile('test-config.yaml', 'server:\n  host: example.com\n  port: 8080\n')`,
				`convertYAMLFileToJSON('test-config.yaml', 'test-config.json')`,
				`setq(loaded, loadJSON('test-config.json'))`,
				`getProp(loaded, 'server.host')`,
			},
			ExpectedValue: chariot.Str("example.com"),
		},
		{
			Name: "Convert YAML Array to JSON",
			Script: []string{
				`writeFile('test-list.yaml', 'items:\n  - first\n  - second\n  - third\n')`,
				`convertYAMLFileToJSON('test-list.yaml', 'test-list.json')`,
				`setq(data, loadJSON('test-list.json'))`,
				`getProp(data, 'items.0')`,
			},
			ExpectedValue: chariot.Str("first"),
		},
	}

	RunTestCases(t, tests)

	// Cleanup
	folder := cfg.ChariotConfig.DataPath
	if folder == "" {
		folder = "."
	}
	os.Remove(folder + "/test-source.yaml")
	os.Remove(folder + "/test-target.json")
	os.Remove(folder + "/test-config.yaml")
	os.Remove(folder + "/test-config.json")
	os.Remove(folder + "/test-list.yaml")
	os.Remove(folder + "/test-list.json")
}

func TestFormatConversionRoundTrips(t *testing.T) {
	initCouchbaseConfig()

	tests := []TestCase{
		{
			Name: "Round-trip JSON to YAML and Back",
			Script: []string{
				`setq(originalJSON, '{"key":"value","number":42}')`,
				`setq(jsonNode1, parseJSON(originalJSON))`,
				`setq(yaml, jsonToYAML(jsonNode1))`,
				`setq(jsonNode2, yamlToJSON(yaml))`,
				`getProp(jsonNode2, 'number')`,
			},
			ExpectedValue: chariot.Number(42),
		},
		{
			Name: "Round-trip File Conversion",
			Script: []string{
				`writeFile('test-original.json', '{"app":"TestApp","version":1.5}')`,
				`convertJSONFileToYAML('test-original.json', 'test-intermediate.yaml')`,
				`convertYAMLFileToJSON('test-intermediate.yaml', 'test-final.json')`,
				`setq(final, loadJSON('test-final.json'))`,
				`getProp(final, 'app')`,
			},
			ExpectedValue: chariot.Str("TestApp"),
		},
		{
			Name: "Node Conversion Pipeline",
			Script: []string{
				`setq(jsonStr, '{"database":{"host":"localhost"}}')`,
				`setq(jsonNode, parseJSON(jsonStr))`,
				`setq(yamlNode, jsonToYAMLNode(jsonNode))`,
				`saveYAML(yamlNode, 'test-pipeline.yaml')`,
				`setq(loaded, loadYAML('test-pipeline.yaml'))`,
				`getProp(loaded, 'database.host')`,
			},
			ExpectedValue: chariot.Str("localhost"),
		},
	}

	RunTestCases(t, tests)

	// Cleanup
	folder := cfg.ChariotConfig.DataPath
	if folder == "" {
		folder = "."
	}
	os.Remove(folder + "/test-original.json")
	os.Remove(folder + "/test-intermediate.yaml")
	os.Remove(folder + "/test-final.json")
	os.Remove(folder + "/test-pipeline.yaml")
}

func TestFormatConversionIntegration(t *testing.T) {
	initCouchbaseConfig()

	tests := []TestCase{
		{
			Name: "Load JSON, Convert to YAML, Modify, Convert Back",
			Script: []string{
				`writeFile('test-workflow.json', '{"setting":"old","count":5}')`,
				`setq(jsonNode, loadJSON('test-workflow.json'))`,
				`saveYAML(jsonNode, 'test-workflow.yaml')`,
				`setq(yamlNode, loadYAML('test-workflow.yaml'))`,
				`setName(yamlNode, 'config')`,
				`setProp(yamlNode, 'setting', 'new')`,
				`saveJSON(yamlNode, 'test-workflow-modified.json')`,
				`setq(final, loadJSON('test-workflow-modified.json'))`,
				`getProp(final, 'setting')`,
			},
			ExpectedValue: chariot.Str("new"),
		},
		{
			Name: "Build Config as JSON, Export as YAML",
			Script: []string{
				`setq(config, jsonNode('config'))`,
				`setProp(config, 'app.name', 'MyService')`,
				`setProp(config, 'app.port', 3000)`,
				`setProp(config, 'app.debug', false)`,
				`saveJSON(config, 'test-export.json')`,
				`convertJSONFileToYAML('test-export.json', 'test-export.yaml')`,
				`setq(yaml, loadYAML('test-export.yaml'))`,
				`getProp(yaml, 'app.debug')`,
			},
			ExpectedValue: chariot.Bool(false),
		},
		{
			Name: "Batch Convert Multiple Files",
			Script: []string{
				`writeFile('batch1.json', '{"id":1}')`,
				`writeFile('batch2.json', '{"id":2}')`,
				`convertJSONFileToYAML('batch1.json', 'batch1.yaml')`,
				`convertJSONFileToYAML('batch2.json', 'batch2.yaml')`,
				`setq(yaml1, loadYAML('batch1.yaml'))`,
				`setName(yaml1, 'data1')`,
				`setq(yaml2, loadYAML('batch2.yaml'))`,
				`setName(yaml2, 'data2')`,
				`setq(id1, getProp(yaml1, 'id'))`,
				`setq(id2, getProp(yaml2, 'id'))`,
				`add(id1, id2)`,
			},
			ExpectedValue: chariot.Number(3),
		},
	}

	RunTestCases(t, tests)

	// Cleanup
	folder := cfg.ChariotConfig.DataPath
	if folder == "" {
		folder = "."
	}
	os.Remove(folder + "/test-workflow.json")
	os.Remove(folder + "/test-workflow.yaml")
	os.Remove(folder + "/test-workflow-modified.json")
	os.Remove(folder + "/test-export.json")
	os.Remove(folder + "/test-export.yaml")
	os.Remove(folder + "/batch1.json")
	os.Remove(folder + "/batch2.json")
	os.Remove(folder + "/batch1.yaml")
	os.Remove(folder + "/batch2.yaml")
}
