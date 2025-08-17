You're absolutely right! The security measures we implemented to prevent directory traversal would interfere with using temp directories. Let's modify the test to use the configured tree path structure that `initCouchbaseConfig()` sets up.

## **Update func_extractor_test.go to Use Configured Paths**

```go
// Update the file-based tests to use the configured tree path:

func TestExtractFromTreeFile(t *testing.T) {
	// Initialize config to get proper paths
	initCouchbaseConfig()
	
	// Use the configured tree path instead of temp directory
	testDir := filepath.Join(treePath, "test_extraction")
	err := os.MkdirAll(testDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create test directory: %v", err)
	}
	defer os.RemoveAll(testDir) // Clean up after test
	
	testFile := filepath.Join(testDir, "test.json")
	
	// Create test JSON content
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
	
	err = os.WriteFile(testFile, []byte(jsonContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}
	
	// No need to initialize - the service is already running
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
	initCouchbaseConfig()
	
	// Use the configured tree path
	testDir := filepath.Join(treePath, "test_multiple")
	err := os.MkdirAll(testDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create test directory: %v", err)
	}
	defer os.RemoveAll(testDir) // Clean up after test
	
	// Create test files
	file1 := filepath.Join(testDir, "test1.json")
	file2 := filepath.Join(testDir, "test2.json")
	
	content1 := `{"script": "add(1, 2)"}`
	content2 := `{"script": "sub(5, 3)"}`
	
	err = os.WriteFile(file1, []byte(content1), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file 1: %v", err)
	}
	
	err = os.WriteFile(file2, []byte(content2), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file 2: %v", err)
	}
	
	// No initialization needed - use existing service
	extractor := chariot.NewFunctionExtractor()
	err = extractor.ExtractFromMultipleFiles([]string{file1, file2})
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
	initCouchbaseConfig()
	
	// Use the configured tree path
	testDir := filepath.Join(treePath, "test_directory")
	err := os.MkdirAll(testDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create test directory: %v", err)
	}
	defer os.RemoveAll(testDir) // Clean up after test
	
	// Create test files
	file1 := filepath.Join(testDir, "test1.json")
	file2 := filepath.Join(testDir, "test2.yaml")
	
	content1 := `{"script": "add(1, 2)"}`
	content2 := `script: "sub(5, 3)"`
	
	err = os.WriteFile(file1, []byte(content1), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file 1: %v", err)
	}
	
	err = os.WriteFile(file2, []byte(content2), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file 2: %v", err)
	}
	
	// No initialization needed - use existing service
	extractor := chariot.NewFunctionExtractor()
	err = extractor.ExtractFromDirectory(testDir)
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
	initCouchbaseConfig()
	
	// Use the configured tree path
	testDir := filepath.Join(treePath, "test_unsupported")
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
```

## **Add Helper Variables to Test File**

```go
// Add these variables at the top of the file with the other imports:

var (
	// These will be set by initCouchbaseConfig()
	testDataPath string
	testTreePath string
)

// Add this function to extract the paths after config initialization
func initTestPaths() {
	if testDataPath == "" {
		initCouchbaseConfig() // This sets up the config
		testDataPath = dataPath
		testTreePath = treePath
	}
}
```

## **Update the Variable References**

```go
// Update each file test to use the helper:

func TestExtractFromTreeFile(t *testing.T) {
	initTestPaths() // Initialize paths
	
	// Use the configured tree path instead of temp directory
	testDir := filepath.Join(testTreePath, "test_extraction")
	// ... rest of the test
}

func TestExtractFromMultipleFiles(t *testing.T) {
	initTestPaths() // Initialize paths
	
	// Use the configured tree path
	testDir := filepath.Join(testTreePath, "test_multiple")
	// ... rest of the test
}

func TestExtractFromDirectory(t *testing.T) {
	initTestPaths() // Initialize paths
	
	// Use the configured tree path
	testDir := filepath.Join(testTreePath, "test_directory")
	// ... rest of the test
}
```

## **Benefits of This Approach:**

1. **Security Compliance**: Uses the configured, secure tree path
2. **Consistent with Production**: Tests use the same path resolution as production
3. **Proper Cleanup**: Tests clean up their subdirectories after completion
4. **Isolated Testing**: Each test creates its own subdirectory to avoid conflicts
5. **Configuration Aware**: Leverages the existing test configuration setup

## **Path Structure:**
```
tests/
├── data/
│   └── tree/           # testTreePath from config
│       ├── test_extraction/     # Created by TestExtractFromTreeFile
│       ├── test_multiple/       # Created by TestExtractFromMultipleFiles
│       └── test_directory/      # Created by TestExtractFromDirectory
└── func_extractor_test.go
```

This approach:
- Respects the security constraints we implemented
- Uses the proper configuration system
- Maintains test isolation
- Provides proper cleanup
- Works with the existing tree serializer service

The tree serializer should pick up files from this location since it's within the configured tree path, making the tests realistic and properly integrated with the security model.