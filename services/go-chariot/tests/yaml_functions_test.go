// tests/yaml_functions_test.go
package tests

import (
	"os"
	"testing"

	"github.com/bhouse1273/chariot-ecosystem/services/go-chariot/chariot"
	cfg "github.com/bhouse1273/chariot-ecosystem/services/go-chariot/configs"
)

func TestYAMLFileOperations(t *testing.T) {
	initCouchbaseConfig()

	tests := []TestCase{
		{
			Name: "Load YAML File",
			Script: []string{
				`writeFile('test-config.yaml', 'database:\n  host: localhost\n  port: 5432\n')`,
				`setq(data, loadYAML('test-config.yaml'))`,
				`setName(data, 'config')`,
				`getProp(data, 'database.host')`,
			},
			ExpectedValue: chariot.Str("localhost"),
		},
		{
			Name: "Save YAML File",
			Script: []string{
				`setq(config, jsonNode('settings'))`,
				`setProp(config, 'app.name', 'MyApp')`,
				`setProp(config, 'app.version', '1.0.0')`,
				`saveYAML(config, 'test-output.yaml')`,
				`fileExists('test-output.yaml')`,
			},
			ExpectedValue: chariot.Bool(true),
		},
		{
			Name: "Load and Save YAML Raw",
			Script: []string{
				`setq(yamlStr, 'name: test\nvalue: 42\n')`,
				`saveYAMLRaw(yamlStr, 'test-raw.yaml')`,
				`setq(loaded, loadYAMLRaw('test-raw.yaml'))`,
				`contains(loaded, 'name: test')`,
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
	os.Remove(folder + "/test-config.yaml")
	os.Remove(folder + "/test-output.yaml")
	os.Remove(folder + "/test-raw.yaml")
}

func TestYAMLLoadAndAccess(t *testing.T) {
	initCouchbaseConfig()

	tests := []TestCase{
		{
			Name: "Load YAML and Access Nested Properties",
			Script: []string{
				`writeFile('test-nested.yaml', 'server:\n  host: example.com\n  port: 8080\n  ssl: true\n')`,
				`setq(config, loadYAML('test-nested.yaml'))`,
				`setName(config, 'config')`,
				`getProp(config, 'server.port')`,
			},
			ExpectedValue: chariot.Number(8080),
		},
		{
			Name: "Load YAML with Arrays",
			Script: []string{
				`writeFile('test-array.yaml', 'users:\n  - Alice\n  - Bob\n  - Charlie\n')`,
				`setq(data, loadYAML('test-array.yaml'))`,
				`setName(data, 'data')`,
				`getProp(data, 'users.0')`,
			},
			ExpectedValue: chariot.Str("Alice"),
		},
		{
			Name: "Load YAML with Mixed Types",
			Script: []string{
				`writeFile('test-mixed.yaml', 'name: TestApp\nversion: 1.5\nenabled: true\ncount: 42\n')`,
				`setq(config, loadYAML('test-mixed.yaml'))`,
				`setName(config, 'config')`,
				`getProp(config, 'enabled')`,
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
	os.Remove(folder + "/test-nested.yaml")
	os.Remove(folder + "/test-array.yaml")
	os.Remove(folder + "/test-mixed.yaml")
}

func TestYAMLSaveOperations(t *testing.T) {
	initCouchbaseConfig()

	tests := []TestCase{
		{
			Name: "Save TreeNode as YAML",
			Script: []string{
				`setq(data, jsonNode('app'))`,
				`setProp(data, 'name', 'MyService')`,
				`setProp(data, 'port', 3000)`,
				`saveYAML(data, 'test-save.yaml')`,
				`setq(loaded, loadYAML('test-save.yaml'))`,
				`setName(loaded, 'app')`,
				`getProp(loaded, 'name')`,
			},
			ExpectedValue: chariot.Str("MyService"),
		},
		{
			Name: "Save Complex Tree Structure",
			Script: []string{
				`setq(root, jsonNode('config'))`,
				`setProp(root, 'database.host', 'localhost')`,
				`setProp(root, 'database.port', 5432)`,
				`setProp(root, 'cache.enabled', true)`,
				`saveYAML(root, 'test-complex.yaml')`,
				`setq(loaded, loadYAML('test-complex.yaml'))`,
				`setName(loaded, 'config')`,
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
	os.Remove(folder + "/test-save.yaml")
	os.Remove(folder + "/test-complex.yaml")
}

func TestYAMLRawOperations(t *testing.T) {
	initCouchbaseConfig()

	tests := []TestCase{
		{
			Name: "Load YAML Raw String",
			Script: []string{
				`writeFile('test-raw-load.yaml', '# Config file\nname: test\nvalue: 123\n')`,
				`setq(raw, loadYAMLRaw('test-raw-load.yaml'))`,
				`contains(raw, '# Config file')`,
			},
			ExpectedValue: chariot.Bool(true),
		},
		{
			Name: "Save YAML Raw String",
			Script: []string{
				`setq(yamlContent, 'key1: value1\nkey2: value2\n')`,
				`saveYAMLRaw(yamlContent, 'test-raw-save.yaml')`,
				`setq(loaded, loadYAMLRaw('test-raw-save.yaml'))`,
				`contains(loaded, 'key1: value1')`,
			},
			ExpectedValue: chariot.Bool(true),
		},
		{
			Name: "Modify Raw YAML String",
			Script: []string{
				`setq(original, 'setting: old_value\n')`,
				`saveYAMLRaw(original, 'test-modify.yaml')`,
				`setq(loaded, loadYAMLRaw('test-modify.yaml'))`,
				`setq(modified, replace(loaded, 'old_value', 'new_value'))`,
				`saveYAMLRaw(modified, 'test-modify.yaml')`,
				`setq(final, loadYAMLRaw('test-modify.yaml'))`,
				`contains(final, 'new_value')`,
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
	os.Remove(folder + "/test-raw-load.yaml")
	os.Remove(folder + "/test-raw-save.yaml")
	os.Remove(folder + "/test-modify.yaml")
}

func TestYAMLMultiDocument(t *testing.T) {
	initCouchbaseConfig()

	tests := []TestCase{
		{
			Name: "Load Multi-Document YAML",
			Script: []string{
				`writeFile('test-multidoc.yaml', 'name: doc1\n---\nname: doc2\n---\nname: doc3\n')`,
				`setq(docs, loadYAMLMultiDoc('test-multidoc.yaml'))`,
				`length(docs)`,
			},
			ExpectedValue: chariot.Number(3),
		},
		{
			Name: "Access Individual Documents",
			Script: []string{
				`writeFile('test-multidoc2.yaml', 'id: 1\nvalue: first\n---\nid: 2\nvalue: second\n')`,
				`setq(docs, loadYAMLMultiDoc('test-multidoc2.yaml'))`,
				`setq(doc2, getAt(docs, 1))`,
				`setName(doc2, 'doc')`,
				`getProp(doc2, 'value')`,
			},
			ExpectedValue: chariot.Str("second"),
		},
		{
			Name: "Save Multi-Document YAML",
			Script: []string{
				`setq(docs, array())`,
				`setq(doc1, jsonNode('doc1'))`,
				`setProp(doc1, 'name', 'First')`,
				`addTo(docs, doc1)`,
				`setq(doc2, jsonNode('doc2'))`,
				`setProp(doc2, 'name', 'Second')`,
				`addTo(docs, doc2)`,
				`saveYAMLMultiDoc(docs, 'test-multidoc-save.yaml')`,
				`setq(loaded, loadYAMLMultiDoc('test-multidoc-save.yaml'))`,
				`length(loaded)`,
			},
			ExpectedValue: chariot.Number(2),
		},
	}

	RunTestCases(t, tests)

	// Cleanup
	folder := cfg.ChariotConfig.DataPath
	if folder == "" {
		folder = "."
	}
	os.Remove(folder + "/test-multidoc.yaml")
	os.Remove(folder + "/test-multidoc2.yaml")
	os.Remove(folder + "/test-multidoc-save.yaml")
}

func TestYAMLIntegration(t *testing.T) {
	initCouchbaseConfig()

	tests := []TestCase{
		{
			Name: "YAML Configuration Processing",
			Script: []string{
				`writeFile('test-app-config.yaml', 'app:\n  name: MyApp\n  version: 2.0\n  features:\n    - auth\n    - api\n    - cache\n')`,
				`setq(config, loadYAML('test-app-config.yaml'))`,
				`setName(config, 'config')`,
				`setq(appName, getProp(config, 'app.name'))`,
				`setq(version, getProp(config, 'app.version'))`,
				`concat(appName, ' v', toString(version))`,
			},
			ExpectedValue: chariot.Str("MyApp v2"),
		},
		{
			Name: "Build YAML from Variables",
			Script: []string{
				`setq(config, jsonNode('settings'))`,
				`setq(dbHost, 'db.example.com')`,
				`setq(dbPort, 5432)`,
				`setProp(config, 'database.host', dbHost)`,
				`setProp(config, 'database.port', dbPort)`,
				`setProp(config, 'database.ssl', true)`,
				`saveYAML(config, 'test-build.yaml')`,
				`setq(loaded, loadYAML('test-build.yaml'))`,
				`setName(loaded, 'settings')`,
				`getProp(loaded, 'database.ssl')`,
			},
			ExpectedValue: chariot.Bool(true),
		},
		{
			Name: "YAML with String Manipulation",
			Script: []string{
				`writeFile('test-env.yaml', 'environment: production\nregion: us-west-2\n')`,
				`setq(config, loadYAML('test-env.yaml'))`,
				`setName(config, 'config')`,
				`setq(env, getProp(config, 'environment'))`,
				`upper(env)`,
			},
			ExpectedValue: chariot.Str("PRODUCTION"),
		},
	}

	RunTestCases(t, tests)

	// Cleanup
	folder := cfg.ChariotConfig.DataPath
	if folder == "" {
		folder = "."
	}
	os.Remove(folder + "/test-app-config.yaml")
	os.Remove(folder + "/test-build.yaml")
	os.Remove(folder + "/test-env.yaml")
}
