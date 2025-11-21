// tests/xml_functions_test.go
package tests

import (
	"os"
	"testing"

	"github.com/bhouse1273/chariot-ecosystem/services/go-chariot/chariot"
	cfg "github.com/bhouse1273/chariot-ecosystem/services/go-chariot/configs"
)

func TestXMLFileOperations(t *testing.T) {
	initCouchbaseConfig()

	tests := []TestCase{
		{
			Name: "Load XML File",
			Script: []string{
				`writeFile('test-config.xml', '<config><database><host>localhost</host><port>5432</port></database></config>')`,
				`setq(data, loadXML('test-config.xml'))`,
				`setName(data, 'config')`,
				`getName(data)`,
			},
			ExpectedValue: chariot.Str("config"),
		},
		{
			Name: "Save XML File",
			Script: []string{
				`setq(root, jsonNode('settings'))`,
				`setq(db, jsonNode('database'))`,
				`addChild(root, db)`,
				`setProp(db, 'host', 'localhost')`,
				`saveXML(root, 'test-output.xml')`,
				`fileExists('test-output.xml')`,
			},
			ExpectedValue: chariot.Bool(true),
		},
		{
			Name: "Load and Save XML Raw",
			Script: []string{
				`setq(xmlStr, '<root><item>test</item></root>')`,
				`saveXMLRaw(xmlStr, 'test-raw.xml')`,
				`setq(loaded, loadXMLRaw('test-raw.xml'))`,
				`contains(loaded, '<root>')`,
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
	os.Remove(folder + "/test-config.xml")
	os.Remove(folder + "/test-output.xml")
	os.Remove(folder + "/test-raw.xml")
}

func TestParseXMLString(t *testing.T) {
	initCouchbaseConfig()

	tests := []TestCase{
		{
			Name: "Parse Simple XML String",
			Script: []string{
				`setq(xmlStr, '<person><name>Alice</name><age>30</age></person>')`,
				`setq(node, parseXMLString(xmlStr))`,
				`setName(node, 'person')`,
				`getName(node)`,
			},
			ExpectedValue: chariot.Str("person"),
		},
		{
			Name: "Parse XML and Access Properties",
			Script: []string{
				`setq(xmlStr, '<user><id>123</id><username>alice</username></user>')`,
				`setq(node, parseXMLString(xmlStr))`,
				`setName(node, 'user')`,
				`getProp(node, 'username')`,
			},
			ExpectedValue: chariot.Str("alice"),
		},
		{
			Name: "Parse Nested XML",
			Script: []string{
				`setq(xmlStr, '<config><server><host>localhost</host></server></config>')`,
				`setq(node, parseXMLString(xmlStr))`,
				`setName(node, 'config')`,
				`getProp(node, 'server.host')`,
			},
			ExpectedValue: chariot.Str("localhost"),
		},
	}

	RunTestCases(t, tests)
}

func TestXMLTreeOperations(t *testing.T) {
	initCouchbaseConfig()

	tests := []TestCase{
		{
			Name: "Load XML and Navigate Tree",
			Script: []string{
				`writeFile('test-tree.xml', '<root><item>value1</item><item>value2</item></root>')`,
				`setq(root, loadXML('test-tree.xml'))`,
				`setName(root, 'root')`,
				`getProp(root, 'item')`,
			},
			ExpectedValue: chariot.Str("value1"),
		},
		{
			Name: "Modify XML Tree",
			Script: []string{
				`setq(root, jsonNode('config'))`,
				`setProp(root, 'version', '1.0')`,
				`setq(section, jsonNode('section'))`,
				`setProp(section, 'name', 'database')`,
				`addChild(root, section)`,
				`getProp(root, 'section.name')`,
			},
			ExpectedValue: chariot.Str("database"),
		},
		{
			Name: "Build and Save XML",
			Script: []string{
				`setq(doc, jsonNode('document'))`,
				`setProp(doc, 'title', 'Test Document')`,
				`setq(body, jsonNode('body'))`,
				`setProp(body, 'content', 'Hello World')`,
				`addChild(doc, body)`,
				`saveXML(doc, 'test-build.xml')`,
				`setq(loaded, loadXML('test-build.xml'))`,
				`setName(loaded, 'document')`,
				`getProp(loaded, 'body.content')`,
			},
			ExpectedValue: chariot.Str("Hello World"),
		},
	}

	RunTestCases(t, tests)

	// Cleanup
	folder := cfg.ChariotConfig.DataPath
	if folder == "" {
		folder = "."
	}
	os.Remove(folder + "/test-tree.xml")
	os.Remove(folder + "/test-build.xml")
}

func TestXMLWithAttributes(t *testing.T) {
	initCouchbaseConfig()

	tests := []TestCase{
		{
			Name: "Parse XML with Attributes",
			Script: []string{
				`setq(xmlStr, '<user id="123" role="admin"><name>Alice</name></user>')`,
				`setq(node, parseXMLString(xmlStr))`,
				`setName(node, 'user')`,
				`getProp(node, 'id')`,
			},
			ExpectedValue: chariot.Str("123"),
		},
		{
			Name: "XML Attributes and Elements",
			Script: []string{
				`setq(xmlStr, '<product sku="ABC123"><name>Widget</name><price>19.99</price></product>')`,
				`setq(node, parseXMLString(xmlStr))`,
				`setName(node, 'product')`,
				`setq(sku, getProp(node, 'sku'))`,
				`setq(name, getProp(node, 'name'))`,
				`concat(sku, '-', name)`,
			},
			ExpectedValue: chariot.Str("ABC123-Widget"),
		},
	}

	RunTestCases(t, tests)
}

func TestXMLRawOperations(t *testing.T) {
	initCouchbaseConfig()

	tests := []TestCase{
		{
			Name: "Load XML Raw String",
			Script: []string{
				`writeFile('test-raw-load.xml', '<?xml version="1.0"?><root><test>value</test></root>')`,
				`setq(raw, loadXMLRaw('test-raw-load.xml'))`,
				`contains(raw, '<?xml version')`,
			},
			ExpectedValue: chariot.Bool(true),
		},
		{
			Name: "Save XML Raw String",
			Script: []string{
				`setq(xmlContent, '<data><field>test</field></data>')`,
				`saveXMLRaw(xmlContent, 'test-raw-save.xml')`,
				`setq(loaded, loadXMLRaw('test-raw-save.xml'))`,
				`contains(loaded, '<field>test</field>')`,
			},
			ExpectedValue: chariot.Bool(true),
		},
		{
			Name: "Modify Raw XML String",
			Script: []string{
				`setq(original, '<config><setting>old</setting></config>')`,
				`saveXMLRaw(original, 'test-modify.xml')`,
				`setq(loaded, loadXMLRaw('test-modify.xml'))`,
				`setq(modified, replace(loaded, 'old', 'new'))`,
				`saveXMLRaw(modified, 'test-modify.xml')`,
				`setq(final, loadXMLRaw('test-modify.xml'))`,
				`contains(final, 'new')`,
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
	os.Remove(folder + "/test-raw-load.xml")
	os.Remove(folder + "/test-raw-save.xml")
	os.Remove(folder + "/test-modify.xml")
}

func TestXMLIntegration(t *testing.T) {
	initCouchbaseConfig()

	tests := []TestCase{
		{
			Name: "XML Configuration Processing",
			Script: []string{
				`writeFile('test-config-proc.xml', '<config><database><host>db.example.com</host><port>3306</port></database></config>')`,
				`setq(config, loadXML('test-config-proc.xml'))`,
				`setName(config, 'config')`,
				`setq(host, getProp(config, 'database.host'))`,
				`setq(port, getProp(config, 'database.port'))`,
				`concat(host, ':', port)`,
			},
			ExpectedValue: chariot.Str("db.example.com:3306"),
		},
		{
			Name: "Build XML from Data",
			Script: []string{
				`setq(users, jsonNode('users'))`,
				`setq(user1, jsonNode('user'))`,
				`setProp(user1, 'name', 'Alice')`,
				`setProp(user1, 'email', 'alice@test.com')`,
				`addChild(users, user1)`,
				`setq(user2, jsonNode('user'))`,
				`setProp(user2, 'name', 'Bob')`,
				`setProp(user2, 'email', 'bob@test.com')`,
				`addChild(users, user2)`,
				`saveXML(users, 'test-users.xml')`,
				`setq(loaded, loadXML('test-users.xml'))`,
				`setName(loaded, 'users')`,
				`setq(children, getChildren(loaded))`,
				`length(children)`,
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
	os.Remove(folder + "/test-config-proc.xml")
	os.Remove(folder + "/test-users.xml")
}
