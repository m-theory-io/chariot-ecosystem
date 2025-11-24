package tests

import (
	"testing"

	"github.com/bhouse1273/chariot-ecosystem/services/go-chariot/chariot"
)

func TestBootstrapVariableAccess(t *testing.T) {
	rt := chariot.NewRuntime()
	chariot.RegisterAll(rt)

	// Simulate bootstrap - declare a global variable
	t.Log("=== Executing declareGlobal ===")
	_, err := rt.ExecProgram(`declareGlobal(usersAgent, 'S', 'test value')`)
	if err != nil {
		t.Fatalf("Error executing declareGlobal: %v", err)
	}

	// Check if it appears in ListGlobalVariables
	t.Log("=== Checking ListGlobalVariables ===")
	globals := rt.ListGlobalVariables()
	t.Logf("Found %d global variables", len(globals))
	for name, val := range globals {
		t.Logf("  %s = %v (type: %T)", name, val, val)
	}

	if _, exists := globals["usersAgent"]; !exists {
		t.Error("usersAgent NOT found in ListGlobalVariables()")
	} else {
		t.Log("✓ usersAgent found in ListGlobalVariables()")
	}

	// Check scopes directly
	t.Log("=== Checking scopes directly ===")
	globalVars := rt.ListGlobalVariables()
	localVars := rt.ListLocalVariables()

	t.Logf("ListGlobalVariables count: %d", len(globalVars))
	for name, val := range globalVars {
		t.Logf("  global['%s'] = %v (type: %T)", name, val, val)
	}

	t.Logf("ListLocalVariables count: %d", len(localVars))
	for name, val := range localVars {
		t.Logf("  local['%s'] = %v (type: %T)", name, val, val)
	} // Try currentScope.Get()
	t.Log("=== Testing currentScope.Get() ===")
	if val, ok := rt.CurrentScope().Get("usersAgent"); ok {
		t.Logf("✓ currentScope.Get('usersAgent') = %v", val)
	} else {
		t.Error("✗ currentScope.Get('usersAgent') returned false")
	}

	// Try to access via naked symbol
	t.Log("=== Accessing via naked symbol ===")
	result, err := rt.ExecProgram(`usersAgent`)
	if err != nil {
		t.Errorf("Error accessing usersAgent: %v", err)

		// Debug: manually test VarRef
		t.Log("=== Manual VarRef test ===")
		varRef := &chariot.VarRef{Name: "usersAgent"}
		manualResult, manualErr := varRef.Exec(rt)
		if manualErr != nil {
			t.Logf("Manual VarRef.Exec error: %v", manualErr)
		} else {
			t.Logf("Manual VarRef.Exec success: %v", manualResult)
		}
	} else {
		t.Logf("✓ Success: %v", result)
	}
}

func TestInspectRuntime(t *testing.T) {
	rt := chariot.NewRuntime()
	chariot.RegisterAll(rt)

	// Declare a global
	_, err := rt.ExecProgram(`declareGlobal(testGlobal, 'S', 'hello')`)
	if err != nil {
		t.Fatalf("Error: %v", err)
	}

	// Call inspectRuntime
	result, err := rt.ExecProgram(`inspectRuntime()`)
	if err != nil {
		t.Fatalf("inspectRuntime error: %v", err)
	}

	// Convert to map
	rtState, ok := result.(map[string]interface{})
	if !ok {
		t.Fatalf("inspectRuntime didn't return a map, got %T", result)
	}

	// Check globals
	globals, ok := rtState["globals"].(map[string]interface{})
	if !ok {
		t.Fatalf("globals is not a map, got %T", rtState["globals"])
	}

	t.Logf("Globals from inspectRuntime: %d items", len(globals))
	for name, val := range globals {
		t.Logf("  %s = %v", name, val)
	}

	if _, exists := globals["testGlobal"]; !exists {
		t.Error("testGlobal NOT found in inspectRuntime() globals")
	} else {
		t.Log("✓ testGlobal found in inspectRuntime()")
	}
}
