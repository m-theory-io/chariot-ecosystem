package tests

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/bhouse1273/chariot-ecosystem/services/go-chariot/chariot"
	cfg "github.com/bhouse1273/chariot-ecosystem/services/go-chariot/configs"
	"github.com/bhouse1273/chariot-ecosystem/services/go-chariot/logs"
	"github.com/bhouse1273/chariot-ecosystem/services/go-chariot/vault"
	"github.com/bhouse1273/kissflag"
	"go.uber.org/zap"
)

// TestCase represents a Chariot script test with expected outcome
type TestCase struct {
	Name           string
	Script         []string
	ExpectedValue  chariot.Value
	ExpectedType   string // Optional type to check against
	ExpectedError  bool
	ErrorSubstring string // Optional substring to check in error message
}

func init() {
	// Initialize logger for all tests
	if cfg.ChariotLogger == nil {
		slogger := logs.NewZapLogger()
		cfg.ChariotLogger = slogger
	}

	// Resolve test dirs relative to this file (no ~, no user-specific absolute paths)
	_, thisFile, _, _ := runtime.Caller(0)
	testsDir := filepath.Dir(thisFile)
	dataDir := filepath.Join(testsDir, "data")
	treesDir := filepath.Join(dataDir, "trees")
	diagramsDir := filepath.Join(dataDir, "diagrams")
	secretsDir := filepath.Join(dataDir, "secrets")
	secretFile := filepath.Join(secretsDir, "local.json")

	// Ensure directories exist
	_ = os.MkdirAll(treesDir, 0o755)
	_ = os.MkdirAll(diagramsDir, 0o755)
	_ = os.MkdirAll(secretsDir, 0o755)

	if _, err := os.Stat(secretFile); err != nil {
		defaultSecret := map[string]interface{}{
			fmt.Sprintf("local-%s", cfg.ChariotKey): map[string]interface{}{
				"org_key":      cfg.ChariotKey,
				"cb_scope":     "_default",
				"cb_user":      "mtheory",
				"cb_password":  "Borg12731273",
				"cb_url":       "couchbase://localhost",
				"cb_bucket":    "chariot",
				"sql_host":     "127.0.0.1",
				"sql_database": "chariot",
				"sql_user":     "root",
				"sql_password": "rootpass",
				"sql_driver":   "mysql",
				"sql_port":     3306,
			},
		}
		if data, marshalErr := json.MarshalIndent(defaultSecret, "", "  "); marshalErr == nil {
			_ = os.WriteFile(secretFile, data, 0o600)
		}
	}

	// Bind via canonical env vars (one place for all tests)
	os.Setenv("CHARIOT_DATA_PATH", dataDir)
	os.Setenv("CHARIOT_TREE_PATH", treesDir)
	os.Setenv("CHARIOT_DIAGRAM_PATH", diagramsDir)
	os.Setenv("CHARIOT_VAULT_KEY_PREFIX", "local")
	os.Setenv("CHARIOT_SECRET_PROVIDER", "file")
	os.Setenv("CHARIOT_SECRET_FILE_PATH", secretFile)
	os.Setenv("CHARIOT_MCP_ENABLED", "true")
	os.Setenv("CHARIOT_MCP_TRANSPORT", "ws")
	os.Setenv("CHARIOT_MCP_WS_PATH", "/mcp")

	kissflag.SetPrefix("CHARIOT_")
	_ = kissflag.BindAllEVars(cfg.ChariotConfig)
	// Normalize any configured paths (expand ~, make absolute, clean)
	cfg.ExpandAndNormalizePaths()

	cfg.ChariotConfig.VaultName = "chariot-vault"
	// Initialize Vault client for all tests
	if err := vault.InitVaultClient(); err != nil {
		cfg.ChariotLogger.Error("vault client init failed", zap.String("error", err.Error()))
	}
}

// RunTestCases executes a batch of test cases
func RunTestCases(t *testing.T, tests []TestCase) {
	rt := createNamedRuntime("test_db")
	defer chariot.UnregisterRuntime("test_db")

	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			// Create new runtime for each test to avoid state contamination
			// Execute the script
			tscript := strings.Join(test.Script, "\n")
			result, err := rt.ExecProgram(tscript)

			// Check for expected errors
			if test.ExpectedError {
				if err == nil {
					t.Fatalf("Expected error but got none")
				}
				if test.ErrorSubstring != "" && (err.Error() == "" || !strings.Contains(err.Error(), test.ErrorSubstring)) {
					t.Fatalf("Expected error containing '%s', got: %v", test.ErrorSubstring, err)
				}
				return
			}

			// Verify no unexpected errors
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}
			// Unwrap scope entry if needed
			if tvar, ok := result.(chariot.ScopeEntry); ok {
				result = tvar.Value // Unwrap the value if it's a variable
			}
			// Verify expected type
			if test.ExpectedType != "" && fmt.Sprintf("%T", result) != test.ExpectedType {
				t.Errorf("Expected type %s, got %T", test.ExpectedType, result)
			}
			// Check value matches expectation
			if !valueEquals(result, test.ExpectedValue) {
				t.Fatalf("Expected value %v, got %v", test.ExpectedValue, result)
			}
		})
	}
}

// Helper function to compare values (complete implementation)
//
//lint:ignore U1000 This function is used in the test framework
func compareValues(expected, actual chariot.Value) bool {
	// Add debug logging
	fmt.Printf("DEBUG: compareValues called with expected=%v (%T), actual=%v (%T)\n", expected, expected, actual, actual)

	if tvar, ok := actual.(chariot.ScopeEntry); ok {
		// If actual is a ScopeEntry, we need to compare its value
		actual = tvar.Value
	}

	// Handle nil cases first
	if expected == nil && actual == nil {
		fmt.Printf("DEBUG: Both nil, returning true\n")
		return true
	}
	if expected == nil || actual == nil {
		fmt.Printf("DEBUG: One nil, returning false\n")
		return false
	}

	// Handle DBNull by checking type strings instead of direct comparison
	expectedType := fmt.Sprintf("%T", expected)
	actualType := fmt.Sprintf("%T", actual)

	// Check if both are DBNull types
	if (expectedType == fmt.Sprintf("%T", chariot.DBNull)) &&
		(actualType == fmt.Sprintf("%T", chariot.DBNull)) {
		fmt.Printf("DEBUG: Both DBNull types, returning true\n")
		return true
	}

	// If one is DBNull and the other isn't, they're not equal
	if (expectedType == fmt.Sprintf("%T", chariot.DBNull)) ||
		(actualType == fmt.Sprintf("%T", chariot.DBNull)) {
		fmt.Printf("DEBUG: One DBNull type, one not, returning false\n")
		return false
	}

	// Handle same type comparisons
	switch exp := expected.(type) {
	case chariot.Str:
		if act, ok := actual.(chariot.Str); ok {
			return string(exp) == string(act)
		}

	case chariot.Number:
		if act, ok := actual.(chariot.Number); ok {
			// Handle floating point comparison with small tolerance
			diff := float64(exp) - float64(act)
			if diff < 0 {
				diff = -diff
			}
			return diff < 1e-9 // Very small tolerance for floating point errors
		}

	case chariot.Bool:
		if act, ok := actual.(chariot.Bool); ok {
			return bool(exp) == bool(act)
		}

	case *chariot.JSONNode:
		if act, ok := actual.(*chariot.JSONNode); ok {
			return compareJSONNodes(exp, act)
		}

	case *chariot.TreeNode:
		if act, ok := actual.(*chariot.TreeNode); ok {
			return compareTreeNodes(exp, act)
		}

	case *chariot.MapNode:
		if act, ok := actual.(*chariot.MapNode); ok {
			return compareMapNodes(exp, act)
		}

	case *chariot.ArrayValue:
		if act, ok := actual.(chariot.ArrayValue); ok {
			return compareArrays((*exp), act)
		}
	}

	// Handle cross-type numeric comparisons (int vs float)
	if isNumericType(expected) && isNumericType(actual) {
		return compareNumericValues(expected, actual)
	}

	// Handle string-like comparisons
	if isStringLike(expected) && isStringLike(actual) {
		return getStringValue(expected) == getStringValue(actual)
	}

	// If types don't match and no special handling, they're not equal
	return false
}

// RunStatefulTestCases runs tests with persistent runtime state
// This is essential for database tests that need to maintain connections
func RunStatefulTestCases(t *testing.T, tests []TestCase, rt *chariot.Runtime) {
	// Create ONE runtime for all tests - maintains state between tests
	if rt == nil {
		rt = createRuntime()
	}

	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			// Use the SAME runtime for all tests (critical for database connections)
			tscript := strings.Join(test.Script, "\n")
			result, err := rt.ExecProgram(tscript)

			// Handle expected errors
			if test.ExpectedError {
				if err == nil {
					t.Fatalf("Expected error but got none")
				}
				if test.ErrorSubstring != "" && !strings.Contains(err.Error(), test.ErrorSubstring) {
					t.Fatalf("Expected error containing '%s', got: %v", test.ErrorSubstring, err)
				}
				t.Logf("✅ Expected error caught: %v", err)
				return
			}

			// Handle unexpected errors
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			// Verify expected type
			resType := fmt.Sprintf("%T", result)
			if test.ExpectedType != "" && resType != test.ExpectedType {
				t.Errorf("Expected type %s, got %s", test.ExpectedType, resType)
			} else if test.ExpectedValue == nil {
				t.Logf("No expected value to compare for test %s", test.Name)
				t.Logf("✅ Test passed: %v", result)
				return
			}

			// Verify expected results
			if !valueEquals(test.ExpectedValue, result) {
				t.Errorf("Expected value %v, got %v", test.ExpectedValue, result)
			} else {
				t.Logf("✅ Test passed: %v", result)
			}
		})
	}
}

// Database-specific test runners for different scenarios

// RunDatabaseTestSuite runs a complete database test suite with connection management
func RunDatabaseTestSuite(t *testing.T, dbType string, connectionTests, operationTests, errorTests []TestCase) {
	t.Run(dbType+"_Connection", func(t *testing.T) {
		RunStatefulTestCases(t, connectionTests, nil)
	})

	t.Run(dbType+"_Operations", func(t *testing.T) {
		RunStatefulTestCases(t, operationTests, nil)
	})

	t.Run(dbType+"_ErrorHandling", func(t *testing.T) {
		// Error tests can be isolated since they test failure scenarios
		RunTestCases(t, errorTests)
	})
}

// RunConcurrentDatabaseTests runs database tests with concurrent connections
func RunConcurrentDatabaseTests(t *testing.T, tests []TestCase, concurrency int) {
	var wg sync.WaitGroup

	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()

			t.Run(fmt.Sprintf("Worker_%d", workerID), func(t *testing.T) {
				rt := createRuntime()

				for _, test := range tests {
					tscript := strings.Join(test.Script, "\n")
					result, err := rt.ExecProgram(tscript)

					if test.ExpectedError {
						if err == nil {
							t.Errorf("Worker %d: Expected error but got none", workerID)
						}
						continue
					}

					if err != nil {
						t.Errorf("Worker %d: Unexpected error: %v", workerID, err)
						continue
					}

					if !valueEquals(test.ExpectedValue, result) {
						t.Errorf("Worker %d: Expected %v, got %v", workerID, test.ExpectedValue, result)
					}
				}
			})
		}(i)
	}

	wg.Wait()
}

// Helper function to check if a value is numeric
func isNumericType(v chariot.Value) bool {
	switch v.(type) {
	case chariot.Number:
		return true
	case int, int8, int16, int32, int64:
		return true
	case uint, uint8, uint16, uint32, uint64:
		return true
	case float32, float64:
		return true
	}
	return false
}

// Helper function to convert any numeric type to float64
func toFloat64(v chariot.Value) (float64, bool) {
	switch val := v.(type) {
	case chariot.Number:
		return float64(val), true
	case int:
		return float64(val), true
	case int8:
		return float64(val), true
	case int16:
		return float64(val), true
	case int32:
		return float64(val), true
	case int64:
		return float64(val), true
	case uint:
		return float64(val), true
	case uint8:
		return float64(val), true
	case uint16:
		return float64(val), true
	case uint32:
		return float64(val), true
	case uint64:
		return float64(val), true
	case float32:
		return float64(val), true
	case float64:
		return val, true
	}
	return 0, false
}

// Compare numeric values of different types
func compareNumericValues(expected, actual chariot.Value) bool {
	exp, ok1 := toFloat64(expected)
	act, ok2 := toFloat64(actual)

	if !ok1 || !ok2 {
		return false
	}

	// Handle floating point comparison with tolerance
	diff := exp - act
	if diff < 0 {
		diff = -diff
	}
	return diff < 1e-9
}

// Helper function to check if a value is string-like
func isStringLike(v chariot.Value) bool {
	switch v.(type) {
	case chariot.Str:
		return true
	case string:
		return true
	}
	return false
}

// Compare JSONNode objects
func compareJSONNodes(expected, actual *chariot.JSONNode) bool {
	expJSON, err1 := expected.ToJSON()
	actJSON, err2 := actual.ToJSON()

	if err1 != nil || err2 != nil {
		return false
	}

	return compareJSONStrings(expJSON, actJSON)
}

// Compare TreeNode objects
func compareTreeNodes(expected, actual *chariot.TreeNode) bool {
	// Compare names
	if (*expected).Name() != (*actual).Name() {
		return false
	}

	// Compare children count
	expChildren := (*expected).GetChildren()
	actChildren := (*actual).GetChildren()

	if len(expChildren) != len(actChildren) {
		return false
	}

	// Compare each child
	for i := 0; i < len(expChildren); i++ {
		if !compareValues(expChildren[i], actChildren[i]) {
			return false
		}
	}

	return true
}

// Compare MapNode objects
func compareMapNodes(expected, actual *chariot.MapNode) bool {
	// Compare by converting to string representation
	return expected.String() == actual.String()
}

// Compare Array objects
func compareArrays(expected, actual chariot.ArrayValue) bool {
	if expected.Length() != actual.Length() {
		return false
	}

	for i := 0; i < expected.Length(); i++ {
		if !compareValues(expected.Get(i), actual.Get(i)) {
			return false
		}
	}

	return true
}

// Compare JSON strings by parsing and comparing structure
func compareJSONStrings(expected, actual string) bool {
	// For simple comparison, just compare strings
	// You could enhance this to parse JSON and compare structures
	return expected == actual
}

// Enhanced JSON comparison that parses and compares structure
func compareJSONStructure(expected, actual string) bool {
	var expData, actData interface{}

	err1 := json.Unmarshal([]byte(expected), &expData)
	err2 := json.Unmarshal([]byte(actual), &actData)

	if err1 != nil || err2 != nil {
		// If parsing fails, fall back to string comparison
		return expected == actual
	}

	return deepEqual(expData, actData)
}

// Deep comparison of parsed JSON data
func deepEqual(expected, actual interface{}) bool {
	// Handle nil cases
	if expected == nil && actual == nil {
		return true
	}
	if expected == nil || actual == nil {
		return false
	}

	switch exp := expected.(type) {
	case map[string]interface{}:
		act, ok := actual.(map[string]interface{})
		if !ok {
			return false
		}

		if len(exp) != len(act) {
			return false
		}

		for key, expVal := range exp {
			actVal, exists := act[key]
			if !exists || !deepEqual(expVal, actVal) {
				return false
			}
		}
		return true

	case []interface{}:
		act, ok := actual.([]interface{})
		if !ok {
			return false
		}

		if len(exp) != len(act) {
			return false
		}

		for i := 0; i < len(exp); i++ {
			if !deepEqual(exp[i], act[i]) {
				return false
			}
		}
		return true

	case string:
		act, ok := actual.(string)
		return ok && exp == act

	case float64:
		act, ok := actual.(float64)
		if !ok {
			return false
		}
		// Handle floating point comparison
		diff := exp - act
		if diff < 0 {
			diff = -diff
		}
		return diff < 1e-9

	case bool:
		act, ok := actual.(bool)
		return ok && exp == act

	default:
		// For any other types, use direct comparison
		return expected == actual
	}
}

// Helper function to get string representation of string-like values
func getStringValue(v chariot.Value) string {
	switch val := v.(type) {
	case chariot.Str:
		return string(val)
	case string:
		return val
	default:
		// Fallback: if it has a String() method, use it
		if stringer, ok := v.(interface{ String() string }); ok {
			return stringer.String()
		}
		// Last resort: use fmt.Sprintf
		return fmt.Sprintf("%v", v)
	}
}

// Helper to compare values
// In test_framework.go - fix the valueEquals function
func valueEquals(expected, actual chariot.Value) bool {
	// Handle DBNull cases first - this is critical!
	if isDBNull(expected) && isDBNull(actual) {
		return true
	}
	if isDBNull(expected) || isDBNull(actual) {
		return false // One is DBNull, the other isn't
	}

	// Handle array comparison
	if expectedArr, ok := expected.(*chariot.ArrayValue); ok {
		if actualArr, ok := actual.(*chariot.ArrayValue); ok {
			if expectedArr.Length() != actualArr.Length() {
				return false
			}
			for i := 0; i < expectedArr.Length(); i++ {
				if !valueEquals(expectedArr.Get(i), actualArr.Get(i)) {
					return false
				}
			}
			return true
		}
		return false
	}

	// Handle string comparison
	if expectedStr, ok := expected.(chariot.Str); ok {
		if actualStr, ok := actual.(chariot.Str); ok {
			return string(expectedStr) == string(actualStr)
		}
	}

	// Handle number comparison
	if expectedNum, ok := expected.(chariot.Number); ok {
		if actualNum, ok := actual.(chariot.Number); ok {
			return float64(expectedNum) == float64(actualNum)
		}
	}

	// Handle boolean comparison
	if expectedBool, ok := expected.(chariot.Bool); ok {
		if actualBool, ok := actual.(chariot.Bool); ok {
			return bool(expectedBool) == bool(actualBool)
		}
	}

	// Handle ExitRequest comparison
	if expectedExit, ok := expected.(*chariot.ExitRequest); ok {
		if actualExit, ok := actual.(*chariot.ExitRequest); ok {
			return expectedExit.Code == actualExit.Code
		}
	}

	// Fallback to direct comparison
	return expected == actual
}

// Helper function to check if a value is DBNull
func isDBNull(val chariot.Value) bool {
	// Check if it's the DBNull singleton
	if val == chariot.DBNull {
		return true
	}

	// Check by string representation as fallback
	return fmt.Sprintf("%v", val) == "DBNull"
}

// In test_framework.go
func createRuntime() *chariot.Runtime {
	rt := chariot.NewRuntime()
	chariot.RegisterAll(rt)

	// Register the runtime globally
	runtimeID := fmt.Sprintf("test_%d", time.Now().UnixNano())
	chariot.RegisterRuntime(runtimeID, rt)

	return rt
}

// Alternative: Create a named test runtime
func createNamedRuntime(name string) *chariot.Runtime {
	rt := chariot.NewRuntime()
	chariot.RegisterAll(rt)

	// Register with specific name
	chariot.RegisterRuntime(name, rt)

	return rt
}
