package tests

import (
	"fmt"
	"os"
	"sync"
	"testing"

	"github.com/bhouse1273/chariot-ecosystem/services/go-chariot/chariot"
	cfg "github.com/bhouse1273/chariot-ecosystem/services/go-chariot/configs"
	"github.com/bhouse1273/kissflag"
)

var (
	couchbaseConfigOnce sync.Once
	cbURL               string
	cbUser              string
	cbPassword          string
	cbBucket            string
	cbScope             string
	dataPath            string
	treePath            string
)

// initCouchbaseConfig ensures config is loaded once
func initCouchbaseConfig() {
	couchbaseConfigOnce.Do(func() {
		// Set the env vars in the os
		os.Setenv("CHARIOT_COUCHBASE_URL", "localhost")
		os.Setenv("CHARIOT_COUCHBASE_USER", "mtheory")
		os.Setenv("CHARIOT_COUCHBASE_PASSWORD", "Borg12731273")
		os.Setenv("CHARIOT_COUCHBASE_BUCKET", "chariot")
		os.Setenv("CHARIOT_COUCHBASE_SCOPE", "_default")
		os.Setenv("CHARIOT_CBDL", "true")
		os.Setenv("CHARIOT_DATA_PATH", "/Users/williamhouse/go/src/github.com/bhouse1273/chariot-ecosystem/services/go-chariot/tests/data")
		os.Setenv("CHARIOT_TREE_PATH", "/Users/williamhouse/go/src/github.com/bhouse1273/chariot-ecosystem/services/go-chariot/tests/data/tree")
		os.Setenv("CHARIOT_DIAGRAM_PATH", "/Users/williamhouse/go/src/github.com/bhouse1273/chariot-ecosystem/services/go-chariot/tests/data/diagrams")

		kissflag.SetPrefix("CHARIOT_")
		kissflag.BindAllEVars(cfg.ChariotConfig)

		// Capture values after binding
		cbURL = fmt.Sprintf("couchbase://%s", cfg.ChariotConfig.CBUrl)
		cbUser = cfg.ChariotConfig.CBUser
		cbPassword = cfg.ChariotConfig.CBPassword
		cbBucket = cfg.ChariotConfig.CBBucket
		cbScope = cfg.ChariotConfig.CBScope
		// Persistence
		dataPath = cfg.ChariotConfig.DataPath
		treePath = cfg.ChariotConfig.TreePath

		// Debug logging
		fmt.Printf("🔧 Couchbase Config Initialized:\n")
		fmt.Printf("  URL: %s\n", cbURL)
		fmt.Printf("  User: %s\n", cbUser)
		fmt.Printf("  Bucket: %s\n", cbBucket)
		fmt.Printf("  Scope: %s\n", cbScope)
	})
}

func TestCouchbaseOperations(t *testing.T) {

	initCouchbaseConfig() // ← Initialize config first
	var rt *chariot.Runtime
	if tvar, exists := chariot.GetRuntimeByID("test_db"); exists {
		rt = tvar
	} else {
		rt = createNamedRuntime("test_db")
		defer chariot.UnregisterRuntime("test_db")
	}

	tests := []TestCase{

		// Connection Management
		{
			Name:          "Connect to Couchbase Cluster",
			Script:        []string{`cbConnect('testcluster', '` + cbURL + `', '` + cbUser + `', '` + cbPassword + `')`},
			ExpectedValue: chariot.Str("Connected to Couchbase cluster"),
		},
		{
			Name:          "Open Bucket",
			Script:        []string{`cbOpenBucket('testcluster', '` + cbBucket + `')`},
			ExpectedValue: chariot.Str("Opened bucket: " + cbBucket),
		},
		{
			Name:          "Set Scope and Collection",
			Script:        []string{`cbSetScope('testcluster', '_default', '_default')`},
			ExpectedValue: chariot.Str("Set scope: _default._default"),
		},

		// Document Operations
		// Preemptive clean up of test-specific bucket
		{
			Name: "Cleanup Bucket",
			Script: []string{
				`setq(result, cbQuery('testcluster', 'DELETE FROM ` + fmt.Sprint(cbBucket, ".", "_default", ".", "_default") + ` WHERE true'))`, // Delete all documents
				`iif(unequal(length(result), 0), 'All documents deleted', 'All documents deleted')`,
			},
			ExpectedValue: chariot.Str("All documents deleted"),
		},
		{
			Name: "Insert Document",
			Script: []string{
				`setq(testDoc, parseJSON('{\"type\": \"user\", \"name\": \"John Doe\", \"age\": 30}'))`,
				`declareGlobal(useKey, 'S', newID('user', 'short', DBNull))`,
				`setq(result, cbInsert('testcluster', useKey, testDoc))`,
				`hasMeta(result, 'cas')`, // Check that metadata exists
			},
			ExpectedValue: chariot.Bool(true),
		},
		{
			Name: "Get Document",
			Script: []string{
				`setq(doc, cbGet('testcluster', useKey))`,
				`getProp(doc, 'name')`,
			},
			ExpectedValue: chariot.Str("John Doe"),
		},
		{
			Name: "Upsert Document",
			Script: []string{
				`setq(updatedDoc, parseJSON('{"type": "user", "name": "John Doe", "age": 31}'))`,
				`setq(result, cbUpsert('testcluster', useKey, updatedDoc))`,
				`hasMeta(result, 'cas')`, // Check that metadata exists
			},
			ExpectedValue: chariot.Bool(true),
		},
		{
			Name: "Verify Upsert",
			Script: []string{
				`setq(doc, cbGet('testcluster', useKey))`,
				`getProp(doc, 'name')`,
			},
			ExpectedValue: chariot.Str("John Doe"),
		},
		// N1QL Query Operations
		{
			Name: "Basic N1QL Query",
			Script: []string{
				`sleep(1000)`, // Ensure previous operations are complete
				`setq(results, cbQuery('testcluster', 'SELECT _default.* FROM ` + fmt.Sprint(cbBucket, ".", "_default", ".", "_default") + ` WHERE type = "user" LIMIT 5'))`,
				`length(results)`,
			},
			ExpectedValue: chariot.Number(1), // Should find our test user
		},
		{
			Name: "Parameterized N1QL Query",
			Script: []string{
				`setq(params, parseJSON('{"userType": "user", "minAge": 25}'))`,
				`setq(results, cbQuery('testcluster', 'SELECT _default.name, _default.age FROM ` + fmt.Sprint(cbBucket, ".", "_default._default") + ` WHERE type = $userType AND age >= $minAge LIMIT 1', params))`,
				`length(results)`,
			},
			ExpectedValue: chariot.Number(1),
		},
		{
			Name: "Query Result Content",
			Script: []string{
				`setq(results, cbQuery('testcluster', 'SELECT name FROM ` + fmt.Sprint(cbBucket, ".", "_default._default") + ` WHERE type = "user"'))`,
				`setq(firstResult, getAt(results, 0))`,
				`getProp(firstResult, 'name')`,
			},
			ExpectedValue: chariot.Str("John Doe"),
		},

		// Complex Document Operations
		{
			Name: "Insert Complex Document",
			Script: []string{
				`setq(complexDoc, parseJSON('{"type": "profile", "user": {"name": "Alice", "details": {"email": "alice@example.com", "preferences": {"theme": "dark", "notifications": true}}}}'))`,
				`setq(result, cbInsert('testcluster', 'profile::alice', complexDoc))`,
				`hasMeta(result, 'cas')`, // Check that metadata exists
			},
			ExpectedValue: chariot.Bool(true),
		},
		{
			Name: "Query Nested Properties",
			Script: []string{
				`sleep(1000)`, // Ensure previous insert is complete
				`setq(results, cbQuery('testcluster', 'SELECT user.details.email AS email FROM ` + fmt.Sprint(cbBucket, ".", "_default._default") + ` WHERE type = "profile"'))`,
				`setq(firstResult, getAt(results, 0))`,
				`getProp(firstResult, 'email')`,
			},
			ExpectedValue: chariot.Str("alice@example.com"),
		},

		// Bulk Operations Test
		{
			Name: "Insert Multiple Documents",
			Script: []string{
				`setq(doc1, parseJSON('{"type": "product", "name": "Laptop", "price": 999.99}'))`,
				`setq(doc2, parseJSON('{"type": "product", "name": "Mouse", "price": 29.99}'))`,
				`setq(doc3, parseJSON('{"type": "product", "name": "Keyboard", "price": 79.99}'))`,
				`setq(res1, cbInsert('testcluster', 'product::laptop', doc1))`,
				`setq(res2, cbInsert('testcluster', 'product::mouse', doc2))`,
				`setq(res3, cbInsert('testcluster', 'product::keyboard', doc3))`,
				`iif(and(hasMeta(res1, 'cas'), hasMeta(res2, 'cas'), hasMeta(res3, 'cas')), 'Bulk insert completed', 'Bulk insert failed')`,
			},
			ExpectedValue: chariot.Str("Bulk insert completed"),
		},
		{
			Name: "Query Multiple Products",
			Script: []string{
				`sleep(1000)`, // Ensure previous inserts are complete
				`setq(results, cbQuery('testcluster', 'SELECT name, price FROM ` + fmt.Sprint(cbBucket, ".", "_default._default") + ` WHERE type = "product" ORDER BY price'))`,
				`length(results)`,
			},
			ExpectedValue: chariot.Number(3),
		},
		{
			Name: "Aggregate Query",
			Script: []string{
				`setq(results, cbQuery('testcluster', 'SELECT COUNT(*) as total_products FROM ` + fmt.Sprint(cbBucket, ".", "_default._default") + ` WHERE type = "product"'))`,
				`setq(firstResult, getAt(results, 0))`,
				`getProp(firstResult, 'total_products')`,
			},
			ExpectedValue: chariot.Number(3),
		},

		// Document Removal
		{
			Name:          "Remove Document",
			Script:        []string{`cbRemove('testcluster', useKey)`},
			ExpectedValue: chariot.Str("Document removed"),
		},
		{
			Name: "Verify Removal",
			Script: []string{
				`setq(results, cbQuery('testcluster', 'SELECT * FROM ` + fmt.Sprint(cbBucket, ".", "_default._default") + ` WHERE META().id = "${useKey}"'))`,
				`length(results)`,
			},
			ExpectedValue: chariot.Number(0),
		},

		{
			Name: "CAS String Preservation",
			Script: []string{

				// Insert document
				`setq(testDoc, parseJSON('{"type": "castest", "value": 123}'))`,
				`setq(result, cbInsert('testcluster', 'cas::test', testDoc))`,

				`if(hasMeta(result, 'cas')) {`,
				// Verify CAS is string
				`logPrint('CAS exists in metadata')`,
				`setq(cas, getMeta(result, 'cas'))`,
				`setq(casType, typeOf(cas))`,
				`equal(casType, 'S')`,
				`} else {`,
				`false`,
				`}`,
			},
			ExpectedValue: chariot.Bool(true),
		},
		{
			Name: "CAS-based Replace Operation",
			Script: []string{
				// Get document with CAS
				`setq(doc, cbGet('testcluster', 'cas::test'))`,
				`setq(cas, getMeta(doc, 'cas'))`,

				// Modify document
				`setProp(doc, 'value', 456)`,

				// Replace with CAS (need new function)
				`cbReplace('testcluster', 'cas::test', doc, cas)`,

				// Verify update
				`setq(updatedDoc, cbGet('testcluster', 'cas::test'))`,
				`getProp(updatedDoc, 'value')`,
			},
			ExpectedValue: chariot.Number(456),
		},

		{
			Name: "Couchbase Metadata Structure",
			Script: []string{
				`setq(doc, cbGet('testcluster', 'cas::test'))`,

				// Verify _meta exists
				`setq(meta, getAllMeta(doc))`,

				// Verify CAS in metadata (FIXED)
				`setq(cas, getMeta(doc, 'cas'))`,
				`setq(casExists, unequal(cas, ''))`, // ← Use 'unequal', not 'notEqual'

				// Verify expiry in metadata (FIXED)
				`setq(expiry, getMeta(doc, 'expiry'))`,
				`setq(expiryExists, unequal(expiry, null))`, // ← Use 'unequal'

				`and(casExists, expiryExists)`,
			},
			ExpectedValue: chariot.Bool(true),
		},

		{
			Name: "CAS Precision Preservation",
			Script: []string{
				`setq(doc1, cbGet('testcluster', 'cas::test'))`,
				`setq(cas1, getMeta(doc1, 'cas'))`,

				`setq(casLength, length(cas1))`,

				// Use correct Chariot function (FIXED)
				`setq(isLongCAS, bigger(casLength, 10))`, // ← Use 'bigger', not 'greaterThan'

				`setq(isNumericString, isNumeric(cas1))`,

				`and(isLongCAS, isNumericString)`,
			},
			ExpectedValue: chariot.Bool(true),
		},

		{
			Name: "Clean JSON Serialization (No Metadata Pollution)",
			Script: []string{
				// Get document
				`setq(doc, cbGet('testcluster', 'cas::test'))`,

				// Serialize to JSON string
				`setq(jsonStr, toJSON(doc))`,

				// Verify metadata NOT in JSON output (FIXED)
				`setq(hasCAS, contains(jsonStr, 'cas'))`,
				`setq(hasExpiry, contains(jsonStr, 'expiry'))`,
				`setq(hasMeta, contains(jsonStr, '_meta'))`,

				// Should be clean JSON without metadata (FIXED)
				`and(not(hasCAS), and(not(hasExpiry), not(hasMeta)))`, // ← Use 'not', correct syntax
			},
			ExpectedValue: chariot.Bool(true),
		},
	}

	RunStatefulTestCases(t, tests, rt)
}

func TestCouchbaseErrorHandling(t *testing.T) {

	if cbURL == "" || cbUser == "" || cbPassword == "" || cbBucket == "" {
		initCouchbaseConfig()
	}

	tests := []TestCase{
		// Connection Error Tests
		{
			Name:           "Invalid Connection String",
			Script:         []string{`cbConnect('badcluster', 'invalid://connection', 'user', 'pass')`},
			ExpectedError:  false,
			ErrorSubstring: "Connected to Couchbase cluster",
		},
		{
			Name: "Invalid Credentials",
			Script: []string{
				`cbConnect('badcluster', '` + cbURL + `', 'baduser', 'badpass')`,
				`cbOpenBucket('badcluster', '` + cbBucket + `')`,
			},
			ExpectedError:  false,
			ErrorSubstring: "Connected to Couchbase cluster", // Expecting this error because bucket won't open with bad credentials
		},

		// Operation on Non-existent Node
		{
			Name:           "Query Non-existent Node",
			Script:         []string{`cbQuery('nonexistent', 'SELECT * FROM default')`},
			ExpectedError:  true,
			ErrorSubstring: "couchbase node 'nonexistent' not found",
		},
		{
			Name:           "Get from Non-existent Node",
			Script:         []string{`cbGet('nonexistent', 'doc::123')`},
			ExpectedError:  true,
			ErrorSubstring: "couchbase node 'nonexistent' not found",
		},

		// Document Not Found
		{
			Name: "Get Non-existent Document",
			Script: []string{
				`cbConnect('testcluster', '` + cbURL + `', '` + cbUser + `', '` + cbPassword + `')`,
				`cbOpenBucket('testcluster', '` + cbBucket + `')`,
				`cbGet('testcluster', 'nonexistent::document')`,
			},
			ExpectedError:  true,
			ErrorSubstring: "failed to get document",
		},

		// Invalid N1QL Query
		{
			Name: "Invalid N1QL Syntax",
			Script: []string{
				`cbConnect('testcluster', '` + cbURL + `', '` + cbUser + `', '` + cbPassword + `')`,
				`cbOpenBucket('testcluster', '` + cbBucket + `')`,
				`cbQuery('testcluster', 'INVALID SQL SYNTAX HERE')`,
			},
			ExpectedError:  true,
			ErrorSubstring: "query failed",
		},

		// Bucket Operations Errors
		{
			Name: "Open Non-existent Bucket",
			Script: []string{
				`cbConnect('testcluster', '` + cbURL + `', '` + cbUser + `', '` + cbPassword + `')`,
				`cbOpenBucket('testcluster', 'nonexistent_bucket')`,
			},
			ExpectedError:  true,
			ErrorSubstring: "failed to open bucket",
		},

		// Parameter Validation
		{
			Name:           "Connect with Wrong Argument Count",
			Script:         []string{`cbConnect('testcluster')`},
			ExpectedError:  true,
			ErrorSubstring: "requires 4 arguments",
		},
		{
			Name:           "Query with Wrong Argument Count",
			Script:         []string{`cbQuery('testcluster')`},
			ExpectedError:  true,
			ErrorSubstring: "requires at least 2 arguments",
		},
		{
			Name:           "Insert with Wrong Argument Count",
			Script:         []string{`cbInsert('testcluster', 'doc')`},
			ExpectedError:  true,
			ErrorSubstring: "requires 3-4 arguments",
		},

		{
			Name: "Invalid CAS Format",
			Script: []string{
				`cbConnect('testcluster', '` + cbURL + `', '` + cbUser + `', '` + cbPassword + `')`,
				`cbOpenBucket('testcluster', '` + cbBucket + `')`,
				`setq(doc, parseJSON('{"test": "data"}'))`,

				// Try replace with invalid CAS
				`cbReplace('testcluster', 'any::doc', doc, 'invalid_cas')`,
			},
			ExpectedError:  true,
			ErrorSubstring: "invalid CAS",
		},
		{
			Name: "CAS Mismatch",
			Script: []string{
				`setq(doc, parseJSON('{"test": "data"}'))`,
				`cbInsert('testcluster', 'cas_mismatch::test', doc)`,

				// Get real CAS
				`setq(realDoc, cbGet('testcluster', 'cas_mismatch::test'))`,

				// Try replace with wrong CAS
				`cbReplace('testcluster', 'cas_mismatch::test', doc, '999999999999999999')`,
			},
			ExpectedError:  true,
			ErrorSubstring: "CAS mismatch",
		},

		{
			Name: "Enhanced Cleanup",
			Script: []string{
				`setq(result, cbQuery('testcluster', 'DELETE FROM ` + fmt.Sprint(cbBucket, ".", "_default", ".", "_default") + ` WHERE true'))`, // Delete all documents
				`iif(unequal(length(result), 0), 'All documents deleted', 'All documents deleted')`,
			},
			ExpectedValue: chariot.Str("All documents deleted"),
		},
	}

	RunStatefulTestCases(t, tests, nil)
}

func TestCouchbaseIntegration(t *testing.T) {

	if cbURL == "" || cbUser == "" || cbPassword == "" || cbBucket == "" {
		// Initialize config if not already done
		initCouchbaseConfig()
	}

	tests := []TestCase{
		// End-to-end workflow test
		{
			Name: "Complete Workflow",
			Script: []string{
				`cbConnect('testcluster', '` + cbURL + `', '` + cbUser + `', '` + cbPassword + `')`,
				`cbOpenBucket('testcluster', '` + cbBucket + `')`,
				`cbQuery('testcluster', 'DELETE FROM ` + fmt.Sprint(cbBucket, ".", "_default", ".", "_default") + ` WHERE true')`, // Delete all documents
				`sleep(1000)`, // Ensure previous operations are complete
				// Create user profile
				`setq(userProfile, parseJSONValue('{"type": "user", "username": "testuser", "email": "test@example.com", "created": "2024-01-01"}'))`,
				`cbInsert('testcluster', 'user::testuser', userProfile)`,

				// Create user preferences
				`setq(preferences, parseJSONValue('{"type": "preferences", "userId": "testuser", "theme": "dark", "language": "en"}'))`,
				`cbInsert('testcluster', 'prefs::testuser', preferences)`,
				`sleep(1000)`, // Ensure previous operations are complete

				// Query user with preferences
				`setq(results, cbQuery('testcluster', 'SELECT u.username, u.email, p.theme FROM ` + fmt.Sprint(cbBucket, ".", "_default", ".", "_default") + ` u JOIN ` + fmt.Sprint(cbBucket, ".", "_default", ".", "_default") + ` p ON p.userId = u.username WHERE u.type = "user" AND p.type = "preferences"'))`,

				// Verify results
				`setq(result, getAt(results, 0))`,
				`setq(username, getProp(result, 'username'))`,
				`setq(theme, getProp(result, 'theme'))`,

				// Cleanup
				`cbRemove('testcluster', 'user::testuser')`,
				`cbRemove('testcluster', 'prefs::testuser')`,
				`cbClose('testcluster')`,

				// Return combined result
				`concat(username, ':', theme)`,
			},
			ExpectedValue: chariot.Str("testuser:dark"),
		},
		// JSON Integration Test
		{
			Name: "JSON File to Couchbase",
			Script: []string{
				`// Create test JSON file`,
				`setq(testData, parseJSONValue('{"users": [{"id": 1, "name": "Alice"}, {"id": 2, "name": "Bob"}]}'))`,
				`saveJSON(testData, 'test_users.json')`,

				// Connect to Couchbase
				`cbConnect('testcluster', '` + cbURL + `', '` + cbUser + `', '` + cbPassword + `')`,
				`cbOpenBucket('testcluster', 'chariot')`,
				`cbQuery('testcluster', 'DELETE FROM ` + fmt.Sprint(cbBucket, ".", "_default", ".", "_default") + ` WHERE true')`, // Delete all documents
				`sleep(1000)`, // Ensure previous operations are complete
				// Load and insert data
				`setq(loadedData, loadJSONRaw('test_users.json'))`,
				`setq(jsonData, parseJSONSimple(loadedData))`,
				`setq(users, getProp(jsonData, 'users'))`,

				// Insert each user
				`setq(user1, getAt(users, 0))`,
				`setq(user2, getAt(users, 1))`,

				`cbInsert('testcluster', 'user::1', toSimpleJSON(user1))`,
				`cbInsert('testcluster', 'user::2', toSimpleJSON(user2))`,
				`sleep(1000)`, // Ensure inserts are complete
				// Query back
				`setq(results, cbQuery('testcluster', 'SELECT name FROM ` + fmt.Sprint(cbBucket, ".", "_default._default") + ` WHERE META().id LIKE "user::%"'))`,
				`setq(count, length(results))`,

				// Cleanup
				`cbRemove('testcluster', 'user::1')`,
				`cbRemove('testcluster', 'user::2')`,
				`cbClose('testcluster')`,
				`deleteFile('test_users.json')`,
				`count`,
			},
			ExpectedValue: chariot.Number(2),
		},

		// Array Integration Test
		{
			Name: "Array Processing with Couchbase",
			Script: []string{
				`// Connect`,
				`cbConnect('testcluster', '` + cbURL + `', '` + cbUser + `', '` + cbPassword + `')`,
				`cbOpenBucket('testcluster', 'chariot')`,

				// Create array of product data
				`setq(products, array('Laptop:999.99', 'Mouse:29.99', 'Keyboard:79.99'))`,
				`setq(productDocs, array())`,

				// Process each product
				`setq(i, 0)`,
				`while(smaller(i, length(products))) {`,
				`setq(productStr, getAt(products, i))`,
				`setq(parts, split(productStr, ':'))`,
				`setq(name, getAt(parts, 0))`,
				`setq(price, getAt(parts, 1))`,

				`setq(doc, parseJSONValue(concat('{"type": "product", "name": "', name, '", "price": ', price '}')))`,
				`setq(docId, concat('product::', lower(name)))`,

				`cbInsert('testcluster', docId, doc)`,
				`append(productDocs, docId)`,

				`setq(i, add(i, 1))`,
				`}`,
				`sleep(1000)`, // Ensure inserts are complete
				// Query all products
				`setq(results, cbQuery('testcluster', 'SELECT COUNT(*) as total FROM ` + fmt.Sprint(cbBucket, "._default._default") + ` WHERE type = "product"'))`,
				`setq(totalCount, getProp(getAt(results, 0), 'total'))`,

				// Cleanup using array
				`setq(j, 0)`,
				`while(smaller(j, length(productDocs))) {`,
				`cbRemove('testcluster', getAt(productDocs, j))`,
				`setq(j, add(j, 1))`,
				`}`,

				`cbClose('testcluster')`,
				`totalCount`,
			},
			ExpectedValue: chariot.Number(3),
		},
	}

	RunStatefulTestCases(t, tests, nil)
}
