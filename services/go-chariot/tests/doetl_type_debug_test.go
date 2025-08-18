package tests

import (
	"testing"

	"github.com/bhouse1273/chariot-ecosystem/services/go-chariot/chariot"
)

func TestDoETLTypeDebugging(t *testing.T) {
	testCases := []TestCase{
		{
			Name: "Transform Creation and Mapping Success",
			Script: []string{
				`// Test transform creation and mapping`,
				`setq(transform, createTransform(TestTransform))`,
				`// Add simple mapping with string program`,
				`addMapping(transform, "name", "full_name", "sourceValue", "VARCHAR", true, "Unknown")`,
				`// Check that we have 1 mapping`,
				`length(getMappings(transform))`,
			},
			ExpectedValue: chariot.Number(1), // Should have 1 mapping
		},
		{
			Name: "doETL Processing Success",
			Script: []string{
				`// Test complete doETL processing`,
				`setq(transform, createTransform(TestTransform))`,
				`addMapping(transform, "name", "full_name", "sourceValue", "VARCHAR", true, "Unknown")`,
				``,
				`// Execute doETL and check status`,
				`setq(result, doETL("test_job", "test_data.csv", transform, map("type", "test", "tableName", "test_table")))`,
				`getProp(result, "status")`,
			},
			ExpectedValue: chariot.Str("completed"), // Should complete successfully
		},
		{
			Name: "Transform Type Verification",
			Script: []string{
				`// Verify transform type`,
				`setq(transform, createTransform(TestTransform))`,
				`typeOf(transform)`,
			},
			ExpectedValue: chariot.Str("T"), // T = Transform type code
		},
	}

	RunTestCases(t, testCases)
}

func TestDoETLStringProgramFeatures(t *testing.T) {
	testCases := []TestCase{
		{
			Name: "Multi-line Program with Backticks",
			Script: []string{
				`// Test multi-line program support`,
				`setq(transform, createTransform(MultiLineTransform))`,
				"addMapping(transform, \"email\", \"email_clean\", `trim(sourceValue)\nlowerCase(sourceValue)`, \"VARCHAR\", true)",
				`// Check that mapping was created`,
				`length(getMappings(transform))`,
			},
			ExpectedValue: chariot.Number(1), // Should have 1 mapping
		},
		{
			Name: "Empty Program String",
			Script: []string{
				`// Test empty program (passthrough)`,
				`setq(transform, createTransform(EmptyProgram))`,
				`addMapping(transform, "age", "person_age", "", "INT", true)`,
				`// Check that mapping was created with empty program`,
				`length(getMappings(transform))`,
			},
			ExpectedValue: chariot.Number(1), // Should have 1 mapping
		},
		{
			Name: "Complex Program String",
			Script: []string{
				`// Test complex single-line program`,
				`setq(transform, createTransform(ComplexProgram))`,
				`addMapping(transform, "price", "price_cents", "toNumber(sourceValue) * 100", "INT", true)`,
				`// Verify the complex program was stored correctly`,
				`length(getMappings(transform))`,
			},
			ExpectedValue: chariot.Number(1), // Should have 1 mapping
		},
	}

	RunTestCases(t, testCases)
}
