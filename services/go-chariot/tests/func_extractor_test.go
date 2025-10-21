package tests

import (
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"

	"github.com/bhouse1273/chariot-ecosystem/services/go-chariot/chariot"
	cfg "github.com/bhouse1273/chariot-ecosystem/services/go-chariot/configs"
)

var (
	// These will be set by initCouchbaseConfig()
	testDataPath string
	testTreePath string
)

func initTestPaths() {
	if testDataPath == "" {
		initCouchbaseConfig() // This sets up the config
		testDataPath = cfg.ChariotConfig.DataPath
		testTreePath = cfg.ChariotConfig.TreePath
	}
}

func TestNewFunctionExtractor(t *testing.T) {
	extractor := chariot.NewFunctionExtractor()

	if extractor == nil {
		t.Fatal("NewFunctionExtractor() returned nil")
	}

	if extractor.FoundFunctions == nil {
		t.Error("FoundFunctions map not initialized")
	}

	if extractor.FunctionRegex == nil {
		t.Error("FunctionRegex not initialized")
	}
}

func TestExtractFromText(t *testing.T) {
	extractor := chariot.NewFunctionExtractor()

	testCases := []struct {
		name     string
		text     string
		expected []string
	}{
		{
			name:     "single function call",
			text:     "add(1, 2)",
			expected: []string{"add"},
		},
		{
			name:     "multiple function calls",
			text:     "add(1, 2), sub(5, 3)",
			expected: []string{"add", "sub"},
		},
		{
			name:     "nested function calls",
			text:     "add(mul(2, 3), div(10, 2))",
			expected: []string{"add", "div", "mul"},
		},
		{
			name:     "function with string arguments",
			text:     `concat("hello", " world")`,
			expected: []string{"concat"},
		},
		{
			name:     "mixed with non-functions",
			text:     "if(condition) { add(1, 2) }",
			expected: []string{"add", "if"},
		},
		{
			name:     "no functions",
			text:     "just some text without function calls",
			expected: []string{},
		},
		{
			name:     "empty text",
			text:     "",
			expected: []string{},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			extractor.Reset()
			extractor.ExtractFromText(tc.text)

			actual := extractor.GetExtractedFunctions()

			if !slicesEqual(actual, tc.expected) {
				t.Errorf("Expected %v, got %v", tc.expected, actual)
			}
		})
	}
}

func TestIsLikelyFunction(t *testing.T) {
	extractor := chariot.NewFunctionExtractor()

	testCases := []struct {
		name     string
		funcName string
		expected bool
	}{
		{"valid function name", "add", true},
		{"valid function with underscore", "get_value", true},
		{"valid function with number", "func1", true},
		{"control flow function", "if", true},        // if IS a function in Chariot
		{"control flow function", "while", true},     // while IS a function in Chariot
		{"control flow function", "switch", true},    // switch IS a function in Chariot
		{"switch branch function", "case", true},     // case IS a function in Chariot
		{"switch default function", "default", true}, // default IS a function in Chariot
		{"single letter", "a", false},                // Too short, likely a variable
		{"keyword else", "else", false},              // One of the 3 keywords
		{"keyword break", "break", false},            // One of the 3 keywords
		{"keyword continue", "continue", false},      // One of the 3 keywords
		{"boolean literals", "true", false},          // Boolean literal
		{"boolean literals", "false", false},         // Boolean literal
		{"null literal", "null", false},              // Null literal
		{"property access", "obj.method", false},     // Property access
		{"empty string", "", false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			actual := extractor.IsLikelyFunction(tc.funcName)
			if actual != tc.expected {
				t.Errorf("Expected %v, got %v for '%s'", tc.expected, actual, tc.funcName)
			}
		})
	}
}

func TestExtractFromNode(t *testing.T) {
	extractor := chariot.NewFunctionExtractor()

	// Test FuncCall node
	funcCall := &chariot.FuncCall{
		Name: "add",
		Args: []chariot.Node{
			&chariot.Literal{Val: chariot.Number(1)},
			&chariot.FuncCall{Name: "sub", Args: []chariot.Node{&chariot.Literal{Val: chariot.Number(5)}, &chariot.Literal{Val: chariot.Number(3)}}},
		},
	}

	extractor.ExtractFromNode(funcCall)

	expected := []string{"add", "sub"}
	actual := extractor.GetExtractedFunctions()
	sort.Strings(actual)
	sort.Strings(expected)

	if !reflect.DeepEqual(actual, expected) {
		t.Errorf("Expected %v, got %v", expected, actual)
	}
}

func TestExtractFromValue(t *testing.T) {
	extractor := chariot.NewFunctionExtractor()

	// Test string value
	extractor.ExtractFromValue(chariot.Str("call add(1, 2)"))

	// Test array value
	array := chariot.NewArray()
	array.Append(chariot.Str("mul(3, 4)"))
	array.Append(chariot.Str("div(10, 2)"))
	extractor.ExtractFromValue(array)

	expected := []string{"add", "div", "mul"}
	actual := extractor.GetExtractedFunctions()
	sort.Strings(actual)
	sort.Strings(expected)

	if !reflect.DeepEqual(actual, expected) {
		t.Errorf("Expected %v, got %v", expected, actual)
	}
}

func TestExtractFromTreeNode(t *testing.T) {
	extractor := chariot.NewFunctionExtractor()

	// Create a test tree node
	root := chariot.NewTreeNode("root")
	root.SetAttribute("script", chariot.Str("add(1, 2)"))
	root.SetAttribute("condition", chariot.Str("bigger(x, 5)"))

	child := chariot.NewTreeNode("child")
	child.SetAttribute("action", chariot.Str("sub(10, 3)"))
	root.AddChild(child)

	err := extractor.ExtractFromTreeNode(root)
	if err != nil {
		t.Fatalf("ExtractFromTreeNode failed: %v", err)
	}

	expected := []string{"add", "bigger", "sub"}
	actual := extractor.GetExtractedFunctions()
	sort.Strings(actual)
	sort.Strings(expected)

	if !reflect.DeepEqual(actual, expected) {
		t.Errorf("Expected %v, got %v", expected, actual)
	}
}

func TestExtractFromSerializedNode(t *testing.T) {
	extractor := chariot.NewFunctionExtractor()

	// Test serialized FuncCall node
	funcCallData := map[string]interface{}{
		"_node_type": "FuncCall",
		"name":       "multiply",
		"args": []interface{}{
			map[string]interface{}{
				"_node_type": "Literal",
				"val":        "2",
			},
			map[string]interface{}{
				"_node_type": "FuncCall",
				"name":       "add",
				"args": []interface{}{
					map[string]interface{}{
						"_node_type": "Literal",
						"val":        "3",
					},
					map[string]interface{}{
						"_node_type": "Literal",
						"val":        "4",
					},
				},
			},
		},
	}

	extractor.ExtractFromSerializedNode("FuncCall", funcCallData)

	expected := []string{"add", "multiply"}
	actual := extractor.GetExtractedFunctions()
	sort.Strings(actual)
	sort.Strings(expected)

	if !reflect.DeepEqual(actual, expected) {
		t.Errorf("Expected %v, got %v", expected, actual)
	}
}

func TestExtractFromInterface(t *testing.T) {
	extractor := chariot.NewFunctionExtractor()

	// Test with nested map structure
	data := map[string]interface{}{
		"script": "add(1, 2)",
		"nested": map[string]interface{}{
			"action":     "sub(5, 3)",
			"_node_type": "FuncCall",
			"name":       "multiply",
		},
		"array": []interface{}{
			"div(10, 2)",
			map[string]interface{}{
				"_node_type": "FuncCall",
				"name":       "mod",
			},
		},
	}

	extractor.ExtractFromInterface(data)

	expected := []string{"add", "div", "mod", "multiply", "sub"}
	actual := extractor.GetExtractedFunctions()
	sort.Strings(actual)
	sort.Strings(expected)

	if !reflect.DeepEqual(actual, expected) {
		t.Errorf("Expected %v, got %v", expected, actual)
	}
}

func TestExtractFromTreeFile(t *testing.T) {
	// Initialize config to get proper paths
	initTestPaths()

	// Use relative path within the configured tree directory
	testFile := "test_extraction/test.json"

	// Create the directory within the tree path
	fullDir := filepath.Join(testTreePath, "test_extraction")
	err := os.MkdirAll(fullDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create test directory: %v", err)
	}
	defer os.RemoveAll(fullDir) // Clean up after test

	// Create test JSON content in the actual file location
	jsonContent := `{
        "name": "test",
        "attributes": {
            "script": "add(1, 2)",
            "condition": "bigger(x, 5)"
        },
        "children": [
            {
                "name": "child1",
                "attributes": {
                    "action": "sub(10, 3)"
                }
            }
        ]
    }`

	fullPath := filepath.Join(fullDir, "test.json")
	err = os.WriteFile(fullPath, []byte(jsonContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Use the relative path for the extractor
	extractor := chariot.NewFunctionExtractor()
	err = extractor.ExtractFromTreeFile(testFile)
	if err != nil {
		t.Fatalf("ExtractFromTreeFile failed: %v", err)
	}

	// Should extract at least some functions from the text content
	functions := extractor.GetExtractedFunctions()
	if len(functions) == 0 {
		t.Error("Expected to extract some functions, got none")
	}

	// Verify specific functions were extracted
	expectedFunctions := []string{"add", "bigger", "sub"}
	for _, expected := range expectedFunctions {
		found := false
		for _, actual := range functions {
			if actual == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected to find function '%s' in extracted functions %v", expected, functions)
		}
	}
}

func TestExtractFromMultipleFiles(t *testing.T) {
	// Initialize config to get proper paths
	initTestPaths()

	// Create test directory
	fullDir := filepath.Join(testTreePath, "test_multiple")
	err := os.MkdirAll(fullDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create test directory: %v", err)
	}
	defer os.RemoveAll(fullDir) // Clean up after test

	// Create test files with full paths
	content1 := `{"script": "add(1, 2)"}`
	content2 := `{"script": "sub(5, 3)"}`

	file1Path := filepath.Join(fullDir, "test1.json")
	file2Path := filepath.Join(fullDir, "test2.json")

	err = os.WriteFile(file1Path, []byte(content1), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file 1: %v", err)
	}

	err = os.WriteFile(file2Path, []byte(content2), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file 2: %v", err)
	}

	// Use relative paths for the extractor
	extractor := chariot.NewFunctionExtractor()
	err = extractor.ExtractFromMultipleFiles([]string{
		"test_multiple/test1.json",
		"test_multiple/test2.json",
	})
	if err != nil {
		t.Fatalf("ExtractFromMultipleFiles failed: %v", err)
	}

	functions := extractor.GetExtractedFunctions()
	if len(functions) == 0 {
		t.Error("Expected to extract some functions, got none")
	}

	// Should contain both functions
	expectedFunctions := []string{"add", "sub"}
	for _, expected := range expectedFunctions {
		found := false
		for _, actual := range functions {
			if actual == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected to find function '%s' in extracted functions %v", expected, functions)
		}
	}
}

func TestExtractFromDirectory(t *testing.T) {
	// Initialize config to get proper paths
	initTestPaths()

	// Create test directory
	fullDir := filepath.Join(testTreePath, "test_directory")
	err := os.MkdirAll(fullDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create test directory: %v", err)
	}
	defer os.RemoveAll(fullDir) // Clean up after test

	// Create test files
	content1 := `{"script": "add(1, 2)"}`
	content2 := `script: "sub(5, 3)"`

	file1Path := filepath.Join(fullDir, "test1.json")
	file2Path := filepath.Join(fullDir, "test2.yaml")

	err = os.WriteFile(file1Path, []byte(content1), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file 1: %v", err)
	}

	err = os.WriteFile(file2Path, []byte(content2), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file 2: %v", err)
	}

	// Use relative path for the extractor
	extractor := chariot.NewFunctionExtractor()
	err = extractor.ExtractFromDirectory("test_directory")
	if err != nil {
		t.Fatalf("ExtractFromDirectory failed: %v", err)
	}

	functions := extractor.GetExtractedFunctions()
	if len(functions) == 0 {
		t.Error("Expected to extract some functions, got none")
	}

	// Should contain functions from both files
	expectedFunctions := []string{"add", "sub"}
	for _, expected := range expectedFunctions {
		found := false
		for _, actual := range functions {
			if actual == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected to find function '%s' in extracted functions %v", expected, functions)
		}
	}
}

func TestUnsupportedFileFormat(t *testing.T) {
	// Initialize config to get proper paths
	initTestPaths()

	// Use the configured tree path
	testDir := filepath.Join(cfg.ChariotConfig.TreePath, "test_unsupported")
	err := os.MkdirAll(testDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create test directory: %v", err)
	}
	defer os.RemoveAll(testDir) // Clean up after test

	testFile := filepath.Join(testDir, "test.txt")

	err = os.WriteFile(testFile, []byte("test content"), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	extractor := chariot.NewFunctionExtractor()
	err = extractor.ExtractFromTreeFile(testFile)
	if err == nil {
		t.Error("Expected error for unsupported file format")
	}

	if !strings.Contains(err.Error(), "unsupported file format") {
		t.Errorf("Expected unsupported file format error, got: %v", err)
	}
}

func TestGetTextContent(t *testing.T) {
	extractor := chariot.NewFunctionExtractor()

	// Test with node that has text attribute
	node := chariot.NewTreeNode("test")
	node.SetAttribute("text", chariot.Str("add(1, 2)"))

	textContent := extractor.GetTextContent(node)
	if textContent != "add(1, 2)" {
		t.Errorf("Expected 'add(1, 2)', got '%s'", textContent)
	}

	// Test with node that has no text
	emptyNode := chariot.NewTreeNode("empty")
	emptyContent := extractor.GetTextContent(emptyNode)
	if emptyContent != "" {
		t.Errorf("Expected empty string, got '%s'", emptyContent)
	}
}

func TestGetExtractedFunctionCount(t *testing.T) {
	extractor := chariot.NewFunctionExtractor()

	if extractor.GetExtractedFunctionCount() != 0 {
		t.Error("Expected 0 functions initially")
	}

	extractor.ExtractFromText("add(1, 2) and sub(5, 3)")

	if extractor.GetExtractedFunctionCount() != 2 {
		t.Errorf("Expected 2 functions, got %d", extractor.GetExtractedFunctionCount())
	}
}

func TestReset(t *testing.T) {
	extractor := chariot.NewFunctionExtractor()

	// Add some functions
	extractor.ExtractFromText("add(1, 2)")
	if extractor.GetExtractedFunctionCount() != 1 {
		t.Error("Expected 1 function before reset")
	}

	// Reset
	extractor.Reset()
	if extractor.GetExtractedFunctionCount() != 0 {
		t.Error("Expected 0 functions after reset")
	}
}

func TestValidateExtractedFunctions(t *testing.T) {
	// This test requires the function registry to be initialized
	// We'll mock it by manually adding functions to the extractor
	extractor := chariot.NewFunctionExtractor()

	// Add some functions (mix of valid and invalid)
	extractor.FoundFunctions["add"] = true     // Should be valid
	extractor.FoundFunctions["unknown"] = true // Should be invalid

	// Since we don't have a real registry in tests, we'll check the structure
	valid, invalid, err := extractor.ValidateExtractedFunctions()
	if err != nil {
		t.Fatalf("ValidateExtractedFunctions failed: %v", err)
	}

	// Check that we got two slices
	if len(valid)+len(invalid) != 2 {
		t.Errorf("Expected 2 total functions, got %d valid + %d invalid", len(valid), len(invalid))
	}
}

func TestExtractFromNilNode(t *testing.T) {
	extractor := chariot.NewFunctionExtractor()

	err := extractor.ExtractFromTreeNode(nil)
	if err != nil {
		t.Errorf("ExtractFromTreeNode with nil should not error, got: %v", err)
	}

	if extractor.GetExtractedFunctionCount() != 0 {
		t.Error("Expected 0 functions when extracting from nil node")
	}
}

// Benchmark tests
func BenchmarkExtractFromText(b *testing.B) {
	extractor := chariot.NewFunctionExtractor()
	text := "add(1, 2) and sub(mul(3, 4), div(10, 2)) and bigger(x, smaller(y, z))"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		extractor.Reset()
		extractor.ExtractFromText(text)
	}
}

func BenchmarkExtractFromTreeNode(b *testing.B) {
	extractor := chariot.NewFunctionExtractor()

	// Create a complex tree node
	root := chariot.NewTreeNode("root")
	for i := 0; i < 10; i++ {
		child := chariot.NewTreeNode("child")
		child.SetAttribute("script", chariot.Str("add(1, 2)"))
		child.SetAttribute("condition", chariot.Str("bigger(x, 5)"))
		root.AddChild(child)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		extractor.Reset()
		extractor.ExtractFromTreeNode(root)
	}
}

func slicesEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	if len(a) == 0 {
		return true // Both are empty
	}

	// Sort both slices
	aCopy := make([]string, len(a))
	bCopy := make([]string, len(b))
	copy(aCopy, a)
	copy(bCopy, b)

	sort.Strings(aCopy)
	sort.Strings(bCopy)

	return reflect.DeepEqual(aCopy, bCopy)
}
