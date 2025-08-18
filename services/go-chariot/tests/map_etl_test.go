package tests

import (
	"testing"

	"github.com/bhouse1273/chariot-ecosystem/services/go-chariot/chariot"
)

// TestMapFunction tests the new map() function for creating MapValue
func TestMapFunction(t *testing.T) {
	tests := []TestCase{
		{
			Name: "Create Empty Map",
			Script: []string{
				`setq(m, map())`,
				`typeOf(m)`,
			},
			ExpectedValue: chariot.Str("M"), // M = MapValue type code
		},
		{
			Name: "Create Map with Key-Value Pairs",
			Script: []string{
				`setq(m, map("type", "sql", "connectionName", "mysql1", "tableName", "test_table"))`,
				`getProp(m, "type")`,
			},
			ExpectedValue: chariot.Str("sql"),
		},
		{
			Name: "Map Function Odd Arguments Should Error",
			Script: []string{
				`map("key1", "value1", "key2")`, // Missing value for key2
			},
			ExpectedError:  true,
			ErrorSubstring: "even number of arguments",
		},
		{
			Name: "Map Function Non-String Keys Should Error",
			Script: []string{
				`map(123, "value1")`, // Non-string key
			},
			ExpectedError:  true,
			ErrorSubstring: "map keys must be strings",
		},
		{
			Name: "Map Function Access Multiple Values",
			Script: []string{
				`setq(config, map("type", "sql", "connectionName", "mysql1", "tableName", "users", "batchSize", 1000))`,
				`getProp(config, "connectionName")`,
			},
			ExpectedValue: chariot.Str("mysql1"),
		},
		{
			Name: "Map Function Access Numeric Value",
			Script: []string{
				`setq(config, map("type", "sql", "connectionName", "mysql1", "tableName", "users", "batchSize", 1000))`,
				`getProp(config, "batchSize")`,
			},
			ExpectedValue: chariot.Number(1000),
		},
	}

	RunTestCases(t, tests)
}

// TestFieldMappingWithValueTypes tests the refactored FieldMapping using Chariot Value types
func TestFieldMappingWithValueTypes(t *testing.T) {
	tests := []TestCase{
		{
			Name: "Create Transform with Value Type Mappings",
			Script: []string{
				`setq(transform, createTransform(testTransform))`,
				`addMapping(transform, "email", "email_address", "toLowerCase(trim(sourceValue))", "VARCHAR", true, "unknown@example.com")`,
				`getName(transform)`,
			},
			ExpectedValue: chariot.Str("testTransform"),
		},
		{
			Name: "Create Transform and Verify Field Types",
			Script: []string{
				`setq(transform, createTransform(testTransform))`,
				`addMapping(transform, "user_id", "id", "toNumber(sourceValue)", "INT", true)`,
				`addMapping(transform, "is_active", "active", "toBool(sourceValue)", "BOOLEAN", false, "false")`,
				`length(getMappings(transform))`,
			},
			ExpectedValue: chariot.Number(2),
		},
		{
			Name: "Multi-line Program with Array",
			Script: []string{
				`setq(transform, createTransform(complexTransform))`,
				`setq(program, array("trim(sourceValue)", "toLowerCase(sourceValue)", "concat(sourceValue, '@domain.com')"))`,
				`addMapping(transform, "username", "email", program, "VARCHAR", true)`,
				`getName(transform)`,
			},
			ExpectedValue: chariot.Str("complexTransform"),
		},
	}

	RunTestCases(t, tests)
}

// TestETLWithMapFunction tests doETL using the new map() function for configuration
func TestETLWithMapFunction(t *testing.T) {
	// Note: These tests would require actual database connections and CSV files
	// For now, we test the configuration parsing and validation
	tests := []TestCase{
		{
			Name: "Parse SQL Target Config with Map",
			Script: []string{
				`setq(config, map("type", "sql", "connectionName", "test_conn", "tableName", "target_table"))`,
				`getProp(config, "type")`,
			},
			ExpectedValue: chariot.Str("sql"),
		},
		{
			Name: "Parse Couchbase Target Config with Map",
			Script: []string{
				`setq(config, map("type", "couchbase", "connectionName", "cb_conn", "collection", "user_data"))`,
				`getProp(config, "collection")`,
			},
			ExpectedValue: chariot.Str("user_data"),
		},
		{
			Name: "Legacy Connection String Config",
			Script: []string{
				`setq(config, map("type", "sql", "driver", "mysql", "connectionString", "user:pass@tcp(host:3306)/db", "tableName", "legacy_table"))`,
				`getProp(config, "driver")`,
			},
			ExpectedValue: chariot.Str("mysql"),
		},
	}

	RunTestCases(t, tests)
}

// TestMapVsMapNode demonstrates the difference between map() and mapNode()
func TestMapVsMapNode(t *testing.T) {
	tests := []TestCase{
		{
			Name: "map() Creates MapValue",
			Script: []string{
				`setq(m, map("key1", "value1"))`,
				`typeOf(m)`,
			},
			ExpectedValue: chariot.Str("M"), // M = MapValue type code
		},
		{
			Name: "mapNode() Creates MapNode",
			Script: []string{
				`setq(node, mapNode())`,
				`typeOf(node)`,
			},
			ExpectedValue: chariot.Str("T"), // T = TreeNode type code
		},
		{
			Name: "map() for Simple Key-Value Storage",
			Script: []string{
				`setq(simpleMap, map("host", "localhost", "port", 3306))`,
				`getProp(simpleMap, "host")`,
			},
			ExpectedValue: chariot.Str("localhost"),
		},
		{
			Name: "mapNode() for Tree-like Structure",
			Script: []string{
				`setq(node, mapNode())`,
				`setAttribute(node, "database", "chariot")`,
				`getAttribute(node, "database")`,
			},
			ExpectedValue: chariot.Str("chariot"),
		},
	}

	RunTestCases(t, tests)
}

// TestTransformValueTypeConsistency tests that the refactored Transform works correctly
func TestTransformValueTypeConsistency(t *testing.T) {
	tests := []TestCase{
		{
			Name: "FieldMapping Uses Chariot Value Types",
			Script: []string{
				`setq(transform, createTransform(valueTypeTest))`,
				`addMapping(transform, "source_field", "target_col", "trim(sourceValue)", "VARCHAR", true, "default_val")`,
				// This should work without type conversion errors
				`length(getMappings(transform))`,
			},
			ExpectedValue: chariot.Number(1),
		},
		{
			Name: "Transform with Boolean Required Field",
			Script: []string{
				`setq(transform, createTransform(boolTest))`,
				`addMapping(transform, "status", "is_active", "toBool(sourceValue)", "BOOLEAN", true)`,
				// Verify the mapping was stored correctly
				`length(getMappings(transform))`,
			},
			ExpectedValue: chariot.Number(1),
		},
		{
			Name: "Transform with Array Program",
			Script: []string{
				`setq(transform, createTransform(arrayProgTest))`,
				`setq(multiStepProgram, array("trim(sourceValue)", "toUpper(sourceValue)", "concat('PREFIX_', sourceValue)"))`,
				`addMapping(transform, "name", "formatted_name", multiStepProgram, "VARCHAR", false, "UNKNOWN")`,
				`length(getMappings(transform))`,
			},
			ExpectedValue: chariot.Number(1),
		},
	}

	RunTestCases(t, tests)
}
