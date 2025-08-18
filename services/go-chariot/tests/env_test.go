// Create tests/env_test.go
package tests

import (
	"os"
	"testing"

	cfg "github.com/bhouse1273/chariot-ecosystem/services/go-chariot/configs"
	"github.com/bhouse1273/kissflag"
)

func TestEnvironmentVariables(t *testing.T) {
	t.Logf("🔍 Environment Variable Debug Test")

	// Test 1: Check if ANY environment variables work
	t.Logf("USER: %s", os.Getenv("USER"))
	t.Logf("HOME: %s", os.Getenv("HOME"))
	t.Logf("PATH: %s", os.Getenv("PATH"))

	// Test 2: Set and read immediately
	os.Setenv("TEST_VAR", "test_value")
	t.Logf("TEST_VAR (just set): %s", os.Getenv("TEST_VAR"))

	// Test 3: Check CHARIOT_ variables specifically
	t.Logf("CHARIOT_COUCHBASE_USER: '%s'", os.Getenv("CHARIOT_COUCHBASE_USER"))
	t.Logf("CHARIOT_COUCHBASE_PASSWORD: '%s'", os.Getenv("CHARIOT_COUCHBASE_PASSWORD"))

	// Try to bind to ChariotConfig using kissflag
	kissflag.SetPrefix("CHARIOT_")
	kissflag.BindAllEVars(cfg.ChariotConfig)
	t.Logf("CHARIOT_CONFIG: %+v\n", cfg.ChariotConfig)

	// Test 4: List ALL environment variables
	count := 0
	for _, env := range os.Environ() {
		if count < 10 { // Show first 10
			t.Logf("ENV[%d]: %s", count, env)
		}
		count++
	}
	t.Logf("Total environment variables: %d", count)
}
