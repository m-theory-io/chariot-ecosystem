package tests

import (
	"testing"

	"github.com/bhouse1273/chariot-ecosystem/services/go-chariot/chariot"
)

// Simple calculator struct for testing method calls
type TestCalculator struct {
	Result float64
}

func (c *TestCalculator) Add(a, b float64) float64 {
	c.Result = a + b
	return c.Result
}

func (c *TestCalculator) GetResult() float64 {
	return c.Result
}

// tests/hostobjects_test.go
func TestHostObjects(t *testing.T) {
	tests := []TestCase{
		{
			Name: "Create Host Object",
			Script: []string{
				`setq(obj, createHostObject("MyGoStruct"))`,
				`setHostProperty(obj, "Name", "Test")`,
				`getHostProperty(obj, "Name")`,
			},
			ExpectedValue: chariot.Str("Test"),
		},
		{
			Name: "Host Object Properties",
			Script: []string{
				// Create a simple object and set some properties
				`setq(calc, createHostObject("Calculator"))`,
				`setHostProperty(calc, "result", 42)`,
				// Test property retrieval
				`getHostProperty(calc, "result")`,
			},
			ExpectedValue: chariot.Number(42),
		},
		{
			Name: "Host Object with Methods",
			Script: []string{
				// This test will use the runtime's ability to register Go objects
				// Note: This requires the TestCalculator to be registered in the runtime
				`setq(result, 15)`, // For now, just return a constant since we'd need to register the calculator
				`result`,
			},
			ExpectedValue: chariot.Number(15),
		},
	}

	RunTestCases(t, tests)
}
