// tests/tree_operations_test.go
package tests

import (
	"testing"

	"github.com/bhouse1273/chariot-ecosystem/services/go-chariot/chariot"
)

func TestTreeOperations(t *testing.T) {
	initCouchbaseConfig()

	tests := []TestCase{
		{
			Name: "Tree Save and Load - Simple",
			Script: []string{
				`setq(data, parseJSON('{"name": "John", "age": 30}', 'root'))`,
				`treeSave(data, 'simple_tree.json')`,
				`setq(loaded, treeLoad('simple_tree.json'))`,
				`getAttribute(loaded, 'name')`,
			},
			ExpectedValue: chariot.Str("John"),
		},
		{
			Name: "Tree Save and Load - Complex Structure",
			Script: []string{
				`setq(data, parseJSON('{"name": "TechCorp", "employees": [{"name": "Alice", "role": "dev"}, {"name": "Bob", "role": "manager"}]}', 'company'))`,
				`treeSave(data, 'complex_tree.json')`,
				`setq(loaded, treeLoad('complex_tree.json'))`,
				`setq(employees, getAttribute(loaded, 'employees'))`,
				`setq(firstEmp, getAt(employees, 0))`,
				`getAttribute(firstEmp, 'name')`,
			},
			ExpectedValue: chariot.Str("Alice"),
		},
		{
			Name: "Tree Save and Load - Array Root",
			Script: []string{
				`setq(data, parseJSON('[1, 2, 3, 4, 5]', 'items'))`,
				`treeSave(data, 'array_tree.json')`,
				`setq(loaded, treeLoad('array_tree.json'))`,
				`getAt(loaded, 2)`,
			},
			ExpectedValue: chariot.Number(3),
		},
		{
			Name: "Tree Save and Load - Nested Objects",
			Script: []string{
				`setq(data, parseJSON('{"database": {"host": "localhost", "port": 5432}, "cache": {"ttl": 300}}', 'config'))`,
				`treeSave(data, 'nested_tree.json')`,
				`setq(loaded, treeLoad('nested_tree.json'))`,
				`setq(db, getAttribute(loaded, 'database'))`,
				`getAttribute(db, 'host')`,
			},
			ExpectedValue: chariot.Str("localhost"),
		},
		{
			Name: "Tree Save - Overwrite Existing",
			Script: []string{
				`setq(data1, parseJSON('{"version": 1}', 'test'))`,
				`treeSave(data1, 'overwrite_test.json')`,
				`setq(data2, parseJSON('{"version": 2}', 'test'))`,
				`treeSave(data2, 'overwrite_test.json')`,
				`setq(loaded, treeLoad('overwrite_test.json'))`,
				`getAttribute(loaded, 'version')`,
			},
			ExpectedValue: chariot.Number(2),
		},
		{
			Name: "Tree Save - Subdirectory",
			Script: []string{
				`setq(data, parseJSON('{"location": "subdir"}', 'test'))`,
				`treeSave(data, 'trees/subdir_tree.json')`,
				`setq(loaded, treeLoad('trees/subdir_tree.json'))`,
				`getAttribute(loaded, 'location')`,
			},
			ExpectedValue: chariot.Str("subdir"),
		},
		// Update the TestTreeOperations test cases
		{
			Name: "Tree Find - Simple Search",
			Script: []string{
				`setq(data, parseJSON('[{"id": 1, "name": "Alice"}, {"id": 2, "name": "Bob"}]', 'users'))`,
				`setq(found, treeFind(data, 'name', 'Bob'))`,
				`setq(firstResult, getAt(found, 0))`,
				`getAttribute(firstResult, 'id')`,
			},
			ExpectedValue: chariot.Number(2),
		},
		{
			Name: "Tree Find - Not Found",
			Script: []string{
				`setq(data, parseJSON('[{"id": 1, "name": "Alice"}, {"id": 2, "name": "Bob"}]', 'users'))`,
				`setq(results, treeFind(data, 'name', 'Charlie'))`,
				`length(results)`,
			},
			ExpectedValue: chariot.Number(0),
		},
		{
			Name: "Tree Find - Nested Search",
			Script: []string{
				`setq(data, jsonNode('{"departments": [{"name": "IT", "head": "Alice"}, {"name": "HR", "head": "Bob"}]}', 'company'))`,
				`setq(found, treeFind(data, 'head', 'Alice'))`,
				`setq(firstResult, getAt(found, 0))`,
				`getAttribute(firstResult, 'name')`,
			},
			ExpectedValue: chariot.Str("IT"),
		},
		{
			Name: "Tree Search - Multiple Results",
			Script: []string{
				`setq(data, parseJSON('[{"id": 1, "name": "Alice", "role": "dev"}, {"id": 2, "name": "Bob", "role": "manager"}, {"id": 3, "name": "Charlie", "role": "dev"}]', 'users'))`,
				`setq(devUsers, treeFind(data, 'role', 'dev'))`,
				`length(devUsers)`,
			},
			ExpectedValue: chariot.Number(2),
		},
		{
			Name: "Tree Search - Get Names of Dev Users",
			Script: []string{
				`setq(data, parseJSON('[{"id": 1, "name": "Alice", "role": "dev"}, {"id": 2, "name": "Bob", "role": "manager"}, {"id": 3, "name": "Charlie", "role": "dev"}]', 'users'))`,
				`setq(devUsers, treeFind(data, 'role', 'dev'))`,
				`setq(firstDev, getAt(devUsers, 0))`,
				`getAttribute(firstDev, 'name')`,
			},
			ExpectedValue: chariot.Str("Alice"),
		},
		{
			Name: "Tree Walk - Count Nodes",
			Script: []string{
				`setq(data, parseJSON('{"a": {"b": {"c": 1}}, "d": [1, 2, 3]}', 'root'))`,
				`setq(count, 0)`,
				`treeWalk(data, func(node) { setq(count, add(count, 1)) })`,
				`count`,
			},
			ExpectedValue: chariot.Number(8), // Depends on tree structure
		},
		{
			Name: "Tree Walk - Collect Values",
			Script: []string{
				`setq(data, parseJSON('[1, 2, 3, 4, 5]', 'numbers'))`,
				`setq(sum, 0)`,
				`treeWalk(data, func(node) { if(equal(typeOf(node), 'N')) { setq(sum, add(sum, node)) } })`,
				`sum`,
			},
			ExpectedValue: chariot.Number(15),
		},
	}

	RunTestCases(t, tests)

	// Cleanup
	// os.RemoveAll("trees/")
}

func TestTreeErrorHandling(t *testing.T) {
	tests := []TestCase{
		{
			Name: "Tree Load - Non-existent File",
			Script: []string{
				`treeLoad('nonexistent.json')`,
			},
			ExpectedError: true,
		},
		{
			Name: "Tree Save - Invalid Path",
			Script: []string{
				`setq(data, jsonNode('test', '{"key": "value"}'))`,
				`treeSave(data, '../../../invalid/path.json')`,
			},
			ExpectedError: true,
		},
		{
			Name: "Tree Save - Invalid Data",
			Script: []string{
				`treeSave('not-a-tree', 'test.json')`,
			},
			ExpectedError: true,
		},
		{
			Name: "Tree Find - Wrong Argument Count",
			Script: []string{
				`setq(data, jsonNode('test', '{}'))`,
				`treeFind(data, 'key')`,
			},
			ExpectedError: true,
		},
		{
			Name: "Tree Walk - Invalid Function",
			Script: []string{
				`setq(data, jsonNode('test', '{}'))`,
				`treeWalk(data, 'not-a-function')`,
			},
			ExpectedError: true,
		},
	}

	RunTestCases(t, tests)
}

func TestTreeSearchOperations(t *testing.T) {
	initCouchbaseConfig()

	tests := []TestCase{
		{
			Name: "Tree Search - Equality (default)",
			Script: []string{
				`setq(data, parseJSON('[{"id": 1, "name": "Alice", "role": "dev"}, {"id": 2, "name": "Bob", "role": "manager"}, {"id": 3, "name": "Charlie", "role": "dev"}]', 'users'))`,
				`setq(devUsers, treeSearch(data, 'role', 'dev'))`,
				`length(devUsers)`,
			},
			ExpectedValue: chariot.Number(2),
		},
		{
			Name: "Tree Search - Explicit Equality",
			Script: []string{
				`setq(data, parseJSON('[{"id": 1, "name": "Alice", "role": "dev"}, {"id": 2, "name": "Bob", "role": "manager"}, {"id": 3, "name": "Charlie", "role": "dev"}]', 'users'))`,
				`setq(devUsers, treeSearch(data, 'role', 'dev', '='))`,
				`length(devUsers)`,
			},
			ExpectedValue: chariot.Number(2),
		},
		{
			Name: "Tree Search - Not Equal",
			Script: []string{
				`setq(data, parseJSON('[{"id": 1, "name": "Alice", "role": "dev"}, {"id": 2, "name": "Bob", "role": "manager"}, {"id": 3, "name": "Charlie", "role": "dev"}]', 'users'))`,
				`setq(nonDevUsers, treeSearch(data, 'role', 'dev', '!='))`,
				`length(nonDevUsers)`,
			},
			ExpectedValue: chariot.Number(1),
		},
		{
			Name: "Tree Search - Greater Than",
			Script: []string{
				`setq(data, parseJSON('[{"id": 1, "name": "Laptop", "price": 999}, {"id": 2, "name": "Mouse", "price": 25}, {"id": 3, "name": "Monitor", "price": 300}]', 'products'))`,
				`setq(expensiveProducts, treeSearch(data, 'price', 100, '>'))`,
				`length(expensiveProducts)`,
			},
			ExpectedValue: chariot.Number(2),
		},
		{
			Name: "Tree Search - Greater Than or Equal",
			Script: []string{
				`setq(data, parseJSON('[{"id": 1, "name": "Laptop", "price": 999}, {"id": 2, "name": "Mouse", "price": 25}, {"id": 3, "name": "Monitor", "price": 300}]', 'products'))`,
				`setq(expensiveProducts, treeSearch(data, 'price', 300, '>='))`,
				`length(expensiveProducts)`,
			},
			ExpectedValue: chariot.Number(2),
		},
		{
			Name: "Tree Search - Less Than",
			Script: []string{
				`setq(data, parseJSON('[{"id": 1, "name": "Laptop", "price": 999}, {"id": 2, "name": "Mouse", "price": 25}, {"id": 3, "name": "Monitor", "price": 300}]', 'products'))`,
				`setq(cheapProducts, treeSearch(data, 'price', 100, '<'))`,
				`length(cheapProducts)`,
			},
			ExpectedValue: chariot.Number(1),
		},
		{
			Name: "Tree Search - Less Than or Equal",
			Script: []string{
				`setq(data, parseJSON('[{"id": 1, "name": "Laptop", "price": 999}, {"id": 2, "name": "Mouse", "price": 25}, {"id": 3, "name": "Monitor", "price": 300}]', 'products'))`,
				`setq(cheapProducts, treeSearch(data, 'price', 300, '<='))`,
				`length(cheapProducts)`,
			},
			ExpectedValue: chariot.Number(2),
		},
		{
			Name: "Tree Search - Contains",
			Script: []string{
				`setq(data, parseJSON('[{"id": 1, "name": "Alice Smith", "role": "dev"}, {"id": 2, "name": "Bob Jones", "role": "manager"}, {"id": 3, "name": "Charlie Brown", "role": "dev"}]', 'users'))`,
				`setq(smithUsers, treeSearch(data, 'name', 'Smith', 'contains'))`,
				`length(smithUsers)`,
			},
			ExpectedValue: chariot.Number(1),
		},
		{
			Name: "Tree Search - Starts With",
			Script: []string{
				`setq(data, parseJSON('[{"id": 1, "name": "Alice", "role": "dev"}, {"id": 2, "name": "Bob", "role": "manager"}, {"id": 3, "name": "Alan", "role": "dev"}]', 'users'))`,
				`setq(aUsers, treeSearch(data, 'name', 'A', 'startswith'))`,
				`length(aUsers)`,
			},
			ExpectedValue: chariot.Number(2),
		},
		{
			Name: "Tree Search - Ends With",
			Script: []string{
				`setq(data, parseJSON('[{"id": 1, "name": "user@example.com", "role": "dev"}, {"id": 2, "name": "admin@test.org", "role": "admin"}, {"id": 3, "name": "guest@example.com", "role": "guest"}]', 'users'))`,
				`setq(exampleUsers, treeSearch(data, 'name', 'example.com', 'endswith'))`,
				`length(exampleUsers)`,
			},
			ExpectedValue: chariot.Number(2),
		},
		{
			Name: "Tree Search - Get Results Data",
			Script: []string{
				`setq(data, parseJSON('[{"id": 1, "name": "Alice", "role": "dev"}, {"id": 2, "name": "Bob", "role": "manager"}, {"id": 3, "name": "Charlie", "role": "dev"}]', 'users'))`,
				`setq(devUsers, treeSearch(data, 'role', 'dev'))`,
				`setq(firstDev, getAt(devUsers, 0))`,
				`getAttribute(firstDev, 'name')`,
			},
			ExpectedValue: chariot.Str("Alice"),
		},
		{
			Name: "Tree Search - No Results",
			Script: []string{
				`setq(data, parseJSON('[{"id": 1, "name": "Alice", "role": "dev"}, {"id": 2, "name": "Bob", "role": "manager"}]', 'users'))`,
				`setq(adminUsers, treeSearch(data, 'role', 'admin'))`,
				`length(adminUsers)`,
			},
			ExpectedValue: chariot.Number(0),
		},
		{
			Name: "Tree Search - Multiple Fields Match",
			Script: []string{
				`setq(data, parseJSON('[{"id": 1, "name": "Alice", "age": 25}, {"id": 2, "name": "Bob", "age": 30}, {"id": 3, "name": "Charlie", "age": 25}]', 'users'))`,
				`setq(youngUsers, treeSearch(data, 'age', 25))`,
				`length(youngUsers)`,
			},
			ExpectedValue: chariot.Number(2),
		},
		{
			Name: "Tree Search - Nested Data",
			Script: []string{
				`setq(data, parseJSON('{"departments": [{"name": "IT", "head": "Alice", "budget": 50000}, {"name": "HR", "head": "Bob", "budget": 30000}]}', 'company'))`,
				`setq(richDepts, treeSearch(data, 'budget', 40000, '>'))`,
				`length(richDepts)`,
			},
			ExpectedValue: chariot.Number(1),
		},
		{
			Name: "Tree Search - Complex String Operations",
			Script: []string{
				`setq(data, parseJSON('[{"id": 1, "email": "alice@company.com"}, {"id": 2, "email": "bob@external.org"}, {"id": 3, "email": "charlie@company.com"}]', 'users'))`,
				`setq(companyEmails, treeSearch(data, 'email', 'company.com', 'contains'))`,
				`length(companyEmails)`,
			},
			ExpectedValue: chariot.Number(2),
		},
	}

	RunTestCases(t, tests)
}

func TestTreeSearchErrorHandling(t *testing.T) {
	tests := []TestCase{
		{
			Name: "Tree Search - Wrong Argument Count (too few)",
			Script: []string{
				`setq(data, parseJSON('{}', 'test'))`,
				`treeSearch(data, 'key')`,
			},
			ExpectedError: true,
		},
		{
			Name: "Tree Search - Wrong Argument Count (too many)",
			Script: []string{
				`setq(data, parseJSON('{}', 'test'))`,
				`treeSearch(data, 'key', 'value', '=', 'extra')`,
			},
			ExpectedError: true,
		},
		{
			Name: "Tree Search - Invalid Attribute Name",
			Script: []string{
				`setq(data, parseJSON('{}', 'test'))`,
				`treeSearch(data, 123, 'value')`,
			},
			ExpectedError: true,
		},
		{
			Name: "Tree Search - Numeric Comparison on Non-Numeric",
			Script: []string{
				`setq(data, parseJSON('[{"name": "Alice", "role": "dev"}]', 'users'))`,
				`treeSearch(data, 'name', 'Alice', '>')`,
			},
			ExpectedValue: chariot.NewArray(), // Should return empty array, not error
		},
		{
			Name: "Tree Search - Invalid Operator",
			Script: []string{
				`setq(data, parseJSON('[{"id": 1, "name": "Alice"}]', 'users'))`,
				`setq(results, treeSearch(data, 'name', 'Alice', 'invalid_op'))`,
				`length(results)`,
			},
			ExpectedValue: chariot.Number(1), // Should default to equality
		},
		{
			Name: "Tree Search - String Operations on Numbers",
			Script: []string{
				`setq(data, parseJSON('[{"id": 1, "age": 25}]', 'users'))`,
				`setq(results, treeSearch(data, 'age', '2', 'contains'))`,
				`length(results)`,
			},
			ExpectedValue: chariot.Number(0), // Should return empty array
		},
	}

	RunTestCases(t, tests)
}
