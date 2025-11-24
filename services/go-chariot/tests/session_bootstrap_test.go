package tests

import (
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/bhouse1273/chariot-ecosystem/services/go-chariot/chariot"
	cfg "github.com/bhouse1273/chariot-ecosystem/services/go-chariot/configs"
	"github.com/bhouse1273/chariot-ecosystem/services/go-chariot/logs"
)

// TestSessionInheritsBootstrapGlobals verifies that new sessions inherit
// global variables from the bootstrap runtime
func TestSessionInheritsBootstrapGlobals(t *testing.T) {
	// Save original config
	origBootstrap := cfg.ChariotConfig.Bootstrap
	defer func() { cfg.ChariotConfig.Bootstrap = origBootstrap }()

	// Disable bootstrap loading in NewSession to test only SetBootstrapRuntime
	cfg.ChariotConfig.Bootstrap = ""

	// Create a bootstrap runtime and set some globals
	bootstrapRuntime := chariot.NewRuntime()
	chariot.RegisterAll(bootstrapRuntime)

	// Execute code that sets global variables and objects
	_, err := bootstrapRuntime.ExecProgram(`
		declareGlobal(testVar1, 'S', 'bootstrap value 1')
		declareGlobal(testVar2, 'N', 42)
		declareGlobal(usersAgent, 'S', 'mock users agent')
		createHostObject('testObject')
		hostObject('testObject', 'test value')
	`)
	if err != nil {
		t.Fatalf("Failed to set bootstrap globals: %v", err)
	}

	// Verify bootstrap runtime has the globals and objects
	globals := bootstrapRuntime.ListGlobalVariables()
	if len(globals) != 3 {
		t.Fatalf("Expected 3 bootstrap globals, got %d: %+v", len(globals), globals)
	}

	objects := bootstrapRuntime.ListObjects()
	if len(objects) == 0 {
		t.Fatal("Expected at least 1 bootstrap object, got 0")
	}

	// Create session manager and set bootstrap runtime
	sm := chariot.NewSessionManager(30*time.Minute, 5*time.Minute)
	sm.SetBootstrapRuntime(bootstrapRuntime)

	// Create a new session
	logger := logs.NewZapLogger()
	session := sm.NewSession("testuser", logger, "test-token-123")

	// Verify the session runtime has inherited the bootstrap globals
	sessionGlobals := session.Runtime.ListGlobalVariables()
	t.Logf("Session globals: %+v", sessionGlobals)

	if len(sessionGlobals) < 3 {
		t.Fatalf("Expected at least 3 session globals (inherited from bootstrap), got %d: %+v", len(sessionGlobals), sessionGlobals)
	}

	// Verify the session runtime has inherited the bootstrap objects
	sessionObjects := session.Runtime.ListObjects()
	t.Logf("Session objects: %+v", sessionObjects)

	if len(sessionObjects) == 0 {
		t.Fatal("Expected at least 1 session object (inherited from bootstrap), got 0")
	}

	// Test accessing each variable via naked symbol
	tests := []struct {
		name     string
		symbol   string
		expected string
	}{
		{"testVar1", "testVar1", "bootstrap value 1"},
		{"testVar2", "testVar2", "42"},
		{"usersAgent", "usersAgent", "mock users agent"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := session.Runtime.ExecProgram(tt.symbol)
			if err != nil {
				t.Fatalf("Failed to access %s: %v", tt.symbol, err)
			}

			var resultStr string
			switch v := result.(type) {
			case chariot.Str:
				resultStr = string(v)
			case chariot.Number:
				resultStr = fmt.Sprintf("%.0f", v)
			default:
				t.Fatalf("Unexpected type: %T", v)
			}

			if resultStr != tt.expected {
				t.Errorf("Expected %s = %q, got %q", tt.symbol, tt.expected, resultStr)
			}
		})
	}

	// Test accessing the bootstrap object
	t.Run("testObject", func(t *testing.T) {
		result, err := session.Runtime.ExecProgram(`getHostObject('testObject')`)
		if err != nil {
			t.Fatalf("Failed to access testObject: %v", err)
		}

		// getHostObject returns the raw object value
		if result == nil {
			t.Fatal("getHostObject returned nil")
		}

		// The object was set with hostObject('testObject', 'test value')
		// which stores it as a HostObjectValue
		t.Logf("testObject type: %T, value: %+v", result, result)
	})

	// Test that session can add its own globals without affecting bootstrap
	_, err = session.Runtime.ExecProgram(`declareGlobal(sessionOnlyVar, 'S', 'session value')`)
	if err != nil {
		t.Fatalf("Failed to set session-only global: %v", err)
	}

	// Verify bootstrap doesn't have sessionOnlyVar
	bootstrapGlobals := bootstrapRuntime.ListGlobalVariables()
	if _, exists := bootstrapGlobals["sessionOnlyVar"]; exists {
		t.Error("Session variable leaked into bootstrap runtime")
	}

	// Verify session has sessionOnlyVar
	sessionGlobals = session.Runtime.ListGlobalVariables()
	if _, exists := sessionGlobals["sessionOnlyVar"]; !exists {
		t.Error("Session-only variable not found in session globals")
	}
}

// TestSessionBootstrapFile verifies that sessions load bootstrap.ch if configured
func TestSessionBootstrapFile(t *testing.T) {
	// Create a temporary bootstrap file
	tmpDir := t.TempDir()
	bootstrapFile := tmpDir + "/test_bootstrap.ch"

	bootstrapContent := `
declareGlobal(fileTestVar, 'S', 'from file')
`
	if err := os.WriteFile(bootstrapFile, []byte(bootstrapContent), 0644); err != nil {
		t.Fatalf("Failed to create temp bootstrap file: %v", err)
	}

	// Save and restore original config
	origBootstrap := cfg.ChariotConfig.Bootstrap
	origDataPath := cfg.ChariotConfig.DataPath
	defer func() {
		cfg.ChariotConfig.Bootstrap = origBootstrap
		cfg.ChariotConfig.DataPath = origDataPath
	}()

	cfg.ChariotConfig.Bootstrap = bootstrapFile
	cfg.ChariotConfig.DataPath = tmpDir

	// Create session manager
	sm := chariot.NewSessionManager(30*time.Minute, 5*time.Minute)

	// Create session (should load bootstrap file)
	logger := logs.NewZapLogger()
	session := sm.NewSession("testuser", logger, "test-token-456")

	// Verify session has variable from file
	result, err := session.Runtime.ExecProgram(`fileTestVar`)
	if err != nil {
		t.Fatalf("Failed to access bootstrap file variable: %v", err)
	}

	resultStr, ok := result.(chariot.Str)
	if !ok {
		t.Fatalf("Expected Str type, got %T", result)
	}

	if string(resultStr) != "from file" {
		t.Errorf("Expected 'from file', got %q", string(resultStr))
	}
}
