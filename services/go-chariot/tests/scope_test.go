package tests

import (
	"strings"
	"testing"

	"github.com/bhouse1273/chariot-ecosystem/services/go-chariot/chariot"
)

func TestGlobalVariablePersistence(t *testing.T) {
	rt := chariot.NewRuntime()
	chariot.RegisterAll(rt)

	// Test 1: Declare a global variable
	t.Run("Declare global variable", func(t *testing.T) {
		code := `declareGlobal(testVar, "S", "hello world")`
		result, err := rt.ExecProgram(code)
		if err != nil {
			t.Fatalf("Failed to declare global variable: %v", err)
		}

		expected := "hello world"
		if result.(chariot.Str) != chariot.Str(expected) {
			t.Errorf("Expected %v, got %v", expected, result)
		}
	})

	// Test 2: Verify the variable persists and can be referenced
	t.Run("Reference global variable", func(t *testing.T) {
		code := `testVar`
		result, err := rt.ExecProgram(code)
		if err != nil {
			t.Fatalf("Failed to reference global variable: %v", err)
		}

		expected := "hello world"
		if result.(chariot.Str) != chariot.Str(expected) {
			t.Errorf("Expected %v, got %v", expected, result)
		}
	})

	// Test 3: Verify it shows up in ListGlobalVariables
	t.Run("List global variables", func(t *testing.T) {
		globals := rt.ListGlobalVariables()

		if val, exists := globals["testVar"]; !exists {
			t.Error("testVar not found in global variables")
		} else if val.(chariot.Str) != "hello world" {
			t.Errorf("Expected 'hello world', got %v", val)
		}
	})

	// Test 4: Modify the global variable
	t.Run("Modify global variable", func(t *testing.T) {
		code := `setq(testVar, "modified value")`
		result, err := rt.ExecProgram(code)
		if err != nil {
			t.Fatalf("Failed to modify global variable: %v", err)
		}

		expected := "modified value"
		if result.(chariot.Str) != chariot.Str(expected) {
			t.Errorf("Expected %v, got %v", expected, result)
		}
	})

	// Test 5: Verify modification persisted
	t.Run("Verify modification persisted", func(t *testing.T) {
		code := `testVar`
		result, err := rt.ExecProgram(code)
		if err != nil {
			t.Fatalf("Failed to reference modified global variable: %v", err)
		}

		expected := "modified value"
		if result.(chariot.Str) != chariot.Str(expected) {
			t.Errorf("Expected %v, got %v", expected, result)
		}
	})
}

func TestLocalVariablesDoNotPersist(t *testing.T) {
	rt := chariot.NewRuntime()
	chariot.RegisterAll(rt)

	// Test 1: Local variable is accessible within the same lexical scope
	t.Run("Local variable accessible within program", func(t *testing.T) {
		code := `
declare(localVar, "S", "local value")
localVar
`
		result, err := rt.ExecProgram(code)
		if err != nil {
			t.Fatalf("Failed to declare/reference local variable: %v", err)
		}

		expected := "local value"
		if result.(chariot.Str) != chariot.Str(expected) {
			t.Errorf("Expected %v, got %v", expected, result)
		}

		if _, exists := rt.ListGlobalVariables()["localVar"]; exists {
			t.Fatal("localVar should not be promoted to global scope")
		}
	})

	// Test 2: Local variable is not available to subsequent ExecProgram calls
	t.Run("Local variable unavailable after program", func(t *testing.T) {
		_, err := rt.ExecProgram(`localVar`)
		if err == nil {
			t.Fatalf("Expected error when referencing out-of-scope local variable")
		}
		if !strings.Contains(err.Error(), "variable 'localVar' not defined") {
			t.Fatalf("Expected 'variable 'localVar' not defined' error, got %v", err)
		}
	})
}

func TestScopeSearchOrder(t *testing.T) {
	rt := chariot.NewRuntime()
	chariot.RegisterAll(rt)

	// Test that scope chain is searched correctly: current -> global -> objects -> lists -> nodes -> functions

	// Setup: Create a global variable
	code := `declareGlobal(sharedName, "S", "global value")`
	_, err := rt.ExecProgram(code)
	if err != nil {
		t.Fatalf("Failed to setup global variable: %v", err)
	}

	// Test 1: Reference finds global
	t.Run("Find global variable", func(t *testing.T) {
		code := `sharedName`
		result, err := rt.ExecProgram(code)
		if err != nil {
			t.Fatalf("Failed to reference variable: %v", err)
		}

		if result.(chariot.Str) != "global value" {
			t.Errorf("Expected 'global value', got %v", result)
		}
	})

	// Test 2: Local shadows global only within the same program
	t.Run("Local variable shadows global within program", func(t *testing.T) {
		code := `
declare(sharedName, "S", "local value")
sharedName
`
		result, err := rt.ExecProgram(code)
		if err != nil {
			t.Fatalf("Failed to run program with local shadow: %v", err)
		}

		if result.(chariot.Str) != "local value" {
			t.Errorf("Expected 'local value' (shadowing), got %v", result)
		}
	})

	// Test 3: After scope reset, global value is visible again
	t.Run("Global value restored after scope reset", func(t *testing.T) {
		result, err := rt.ExecProgram(`sharedName`)
		if err != nil {
			t.Fatalf("Failed to reference global variable: %v", err)
		}
		if result.(chariot.Str) != "global value" {
			t.Errorf("Expected 'global value', got %v", result)
		}
	})
}
