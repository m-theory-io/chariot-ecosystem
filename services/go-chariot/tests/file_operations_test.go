// tests/file_operations_test.go
package tests

import (
	"os"
	"testing"

	"github.com/bhouse1273/chariot-ecosystem/services/go-chariot/chariot"
)

func TestFileOperations(t *testing.T) {
	initCouchbaseConfig()

	tests := []TestCase{
		{
			Name: "Write and Read Text File",
			Script: []string{
				`writeFile('test.txt', 'Hello, World!')`,
				`readFile('test.txt')`,
			},
			ExpectedValue: chariot.Str("Hello, World!"),
		},
		{
			Name: "File Exists Check - Existing File",
			Script: []string{
				`writeFile('exists.txt', 'content')`,
				`fileExists('exists.txt')`,
			},
			ExpectedValue: chariot.Bool(true),
		},
		{
			Name: "File Exists Check - Non-existent File",
			Script: []string{
				`fileExists('nonexistent.txt')`,
			},
			ExpectedValue: chariot.Bool(false),
		},
		{
			Name: "Get File Size",
			Script: []string{
				`writeFile('size_test.txt', 'Hello')`,
				`getFileSize('size_test.txt')`,
			},
			ExpectedValue: chariot.Number(5), // "Hello" is 5 bytes
		},
		{
			Name: "Delete File",
			Script: []string{
				`writeFile('delete_me.txt', 'content')`,
				`deleteFile('delete_me.txt')`,
				`fileExists('delete_me.txt')`,
			},
			ExpectedValue: chariot.Bool(false),
		},
		{
			Name: "List Files in Directory",
			Script: []string{
				`writeFile('file1.txt', 'content1')`,
				`writeFile('file2.txt', 'content2')`,
				`setq(files, listFiles())`,
				`contains(files, 'file1.txt')`,
			},
			ExpectedValue: chariot.Bool(true),
		},
		{
			Name: "Read Non-existent File - Error",
			Script: []string{
				`readFile('does_not_exist.txt')`,
			},
			ExpectedError: true,
		},
		{
			Name: "Write to Subdirectory",
			Script: []string{
				`writeFile('subdir/nested.txt', 'nested content')`,
				`readFile('subdir/nested.txt')`,
			},
			ExpectedValue: chariot.Str("nested content"),
		},
	}

	RunTestCases(t, tests)

	// Cleanup
	os.RemoveAll("data/subdir")
}
