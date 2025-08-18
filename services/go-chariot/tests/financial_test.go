package tests

import (
	"testing"

	"github.com/bhouse1273/chariot-ecosystem/services/go-chariot/chariot"
)

// tests/financial_test.go
func TestFinancialOperations(t *testing.T) {
	tests := []TestCase{
		{
			Name: "Present Value Calculation",
			Script: []string{
				`pv(1000, 0.05, 5)`, // FV, rate, periods
			},
			ExpectedValue: chariot.Number(783.53), // rounded to 2 decimal places
		},
		{
			Name: "Loan Payment Calculation",
			Script: []string{
				`pmt(250000, 0.045, 30)`, // principal, rate, years
			},
			ExpectedValue: chariot.Number(1266.71), // rounded monthly payment
		},
	}

	RunTestCases(t, tests)
}
