package tests

import (
	"testing"

	"github.com/bhouse1273/go-chariot/chariot"
)

// tests/etl_test.go
func TestETLOperations(t *testing.T) {
	initCouchbaseConfig()
	tests := []TestCase{
		{
			Name: "Extract CSV Data",
			Script: []string{
				`setq(data, extractCSV("test_data.csv"))`,
				`length(data)`,
			},
			ExpectedValue: chariot.Number(54), // expected row count
		},
		{
			Name: "Transform Data Pipeline",
			Script: []string{
				`setq(data, extractCSV("sales_data.csv"))`,
				`setq(transformed, transform(data, func(row) {`,
				`  setProp(row, "total", mul(getProp(row, "quantity"), getProp(row, "price")))`,
				`  row`,
				`}))`,
				`getProp(getAt(transformed, 0), "total")`,
			},
			ExpectedValue: chariot.Number(150.00),
		},
	}

	RunTestCases(t, tests)
}
