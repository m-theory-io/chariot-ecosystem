// tests/csv_functions_test.go
package tests

import (
	"os"
	"testing"

	"github.com/bhouse1273/chariot-ecosystem/services/go-chariot/chariot"
	cfg "github.com/bhouse1273/chariot-ecosystem/services/go-chariot/configs"
)

func TestCSVFileOperations(t *testing.T) {
	initCouchbaseConfig()

	tests := []TestCase{
		{
			Name: "Load and Save CSV Raw",
			Script: []string{
				`setq(csvStr, 'col1,col2\nval1,val2\n')`,
				`saveCSVRaw(csvStr, 'test-raw.csv')`,
				`setq(loaded, loadCSVRaw('test-raw.csv'))`,
				`loaded`,
			},
			ExpectedValue: chariot.Str("col1,col2\nval1,val2\n"),
		},
	}

	RunTestCases(t, tests)

	// Cleanup
	folder := cfg.ChariotConfig.DataPath
	if folder == "" {
		folder = "."
	}
	os.Remove(folder + "/test-raw.csv")
}

func TestCSVHeaders(t *testing.T) {
	initCouchbaseConfig()

	tests := []TestCase{
		{
			Name: "Get CSV Headers from Path",
			Script: []string{
				`writeFile('test-headers.csv', 'id,name,email\n1,Alice,alice@test.com')`,
				`setq(headers, csvHeaders('test-headers.csv'))`,
				`getAt(headers, 0)`,
			},
			ExpectedValue: chariot.Str("id"),
		},
	}

	RunTestCases(t, tests)

	// Cleanup
	folder := cfg.ChariotConfig.DataPath
	if folder == "" {
		folder = "."
	}
	os.Remove(folder + "/test-headers.csv")
}

func TestCSVRowAndColumnCount(t *testing.T) {
	initCouchbaseConfig()

	tests := []TestCase{
		{
			Name: "CSV Row Count",
			Script: []string{
				`writeFile('test-count.csv', 'a,b,c\n1,2,3\n4,5,6\n7,8,9')`,
				`csvRowCount('test-count.csv')`,
			},
			ExpectedValue: chariot.Number(3),
		},
		{
			Name: "CSV Column Count",
			Script: []string{
				`writeFile('test-cols.csv', 'col1,col2,col3,col4\nv1,v2,v3,v4')`,
				`csvColumnCount('test-cols.csv')`,
			},
			ExpectedValue: chariot.Number(4),
		},
		{
			Name: "Empty CSV",
			Script: []string{
				`writeFile('test-empty.csv', 'header1,header2')`,
				`csvRowCount('test-empty.csv')`,
			},
			ExpectedValue: chariot.Number(0),
		},
	}

	RunTestCases(t, tests)

	// Cleanup
	folder := cfg.ChariotConfig.DataPath
	if folder == "" {
		folder = "."
	}
	os.Remove(folder + "/test-count.csv")
	os.Remove(folder + "/test-cols.csv")
	os.Remove(folder + "/test-empty.csv")
}

func TestCSVGetRow(t *testing.T) {
	initCouchbaseConfig()

	tests := []TestCase{
		{
			Name: "Get CSV Row by Index",
			Script: []string{
				`writeFile('test-rows.csv', 'id,name,age\n1,Alice,30\n2,Bob,25')`,
				`csvGetCell('test-rows.csv', 0, 'name')`,
			},
			ExpectedValue: chariot.Str("Alice"),
		},
		{
			Name: "Get Second Row",
			Script: []string{
				`writeFile('test-rows2.csv', 'id,value\n1,first\n2,second\n3,third')`,
				`csvGetCell('test-rows2.csv', 1, 'value')`,
			},
			ExpectedValue: chariot.Str("second"),
		},
	}

	RunTestCases(t, tests)

	// Cleanup
	folder := cfg.ChariotConfig.DataPath
	if folder == "" {
		folder = "."
	}
	os.Remove(folder + "/test-rows.csv")
	os.Remove(folder + "/test-rows2.csv")
}

func TestCSVGetCell(t *testing.T) {
	initCouchbaseConfig()

	tests := []TestCase{
		{
			Name: "Get Cell by Column Name",
			Script: []string{
				`writeFile('test-cell.csv', 'id,name,email\n1,Alice,alice@test.com\n2,Bob,bob@test.com')`,
				`csvGetCell('test-cell.csv', 0, 'email')`,
			},
			ExpectedValue: chariot.Str("alice@test.com"),
		},
		{
			Name: "Get Cell by Column Index",
			Script: []string{
				`writeFile('test-cell2.csv', 'a,b,c\n1,2,3\n4,5,6')`,
				`csvGetCell('test-cell2.csv', 1, 2)`,
			},
			ExpectedValue: chariot.Str("6"),
		},
		{
			Name: "Get Cell from First Row First Column",
			Script: []string{
				`writeFile('test-cell3.csv', 'col1,col2\nvalue1,value2')`,
				`csvGetCell('test-cell3.csv', 0, 0)`,
			},
			ExpectedValue: chariot.Str("value1"),
		},
	}

	RunTestCases(t, tests)

	// Cleanup
	folder := cfg.ChariotConfig.DataPath
	if folder == "" {
		folder = "."
	}
	os.Remove(folder + "/test-cell.csv")
	os.Remove(folder + "/test-cell2.csv")
	os.Remove(folder + "/test-cell3.csv")
}

func TestCSVGetRows(t *testing.T) {
	initCouchbaseConfig()

	tests := []TestCase{
		{
			Name: "Get All CSV Rows",
			Script: []string{
				`writeFile('test-allrows.csv', 'a,b\n1,2\n3,4\n5,6')`,
				`setq(rows, csvGetRows('test-allrows.csv'))`,
				`length(rows)`,
			},
			ExpectedValue: chariot.Number(3),
		},
		{
			Name: "Get Specific Cell from All Rows",
			Script: []string{
				`writeFile('test-allrows2.csv', 'x,y\n10,20\n30,40')`,
				`setq(rows, csvGetRows('test-allrows2.csv'))`,
				`setq(firstRow, getAt(rows, 0))`,
				`getAt(firstRow, 1)`,
			},
			ExpectedValue: chariot.Str("20"),
		},
	}

	RunTestCases(t, tests)

	// Cleanup
	folder := cfg.ChariotConfig.DataPath
	if folder == "" {
		folder = "."
	}
	os.Remove(folder + "/test-allrows.csv")
	os.Remove(folder + "/test-allrows2.csv")
}

func TestCSVToCSV(t *testing.T) {
	initCouchbaseConfig()

	tests := []TestCase{
		{
			Name: "Convert CSV Path to String",
			Script: []string{
				`writeFile('test-tocsv.csv', 'a,b,c\n1,2,3')`,
				`setq(csvStr, csvToCSV('test-tocsv.csv'))`,
				`contains(csvStr, 'a,b,c')`,
			},
			ExpectedValue: chariot.Bool(true),
		},
	}

	RunTestCases(t, tests)

	// Cleanup
	folder := cfg.ChariotConfig.DataPath
	if folder == "" {
		folder = "."
	}
	os.Remove(folder + "/test-tocsv.csv")
}

func TestCSVIntegration(t *testing.T) {
	initCouchbaseConfig()

	tests := []TestCase{
		{
			Name: "CSV Processing Pipeline with Path",
			Script: []string{
				`writeFile('test-pipeline.csv', 'name,score\nAlice,85\nBob,92\nCharlie,78')`,
				`setq(count, csvRowCount('test-pipeline.csv'))`,
				`setq(headers, csvHeaders('test-pipeline.csv'))`,
				`csvGetCell('test-pipeline.csv', 0, 'score')`,
			},
			ExpectedValue: chariot.Str("85"),
		},
		{
			Name: "Iterate Through CSV Rows",
			Script: []string{
				`writeFile('test-iterate.csv', 'num\n5\n10\n15')`,
				`setq(total, 0)`,
				`setq(i, 0)`,
				`setq(count, csvRowCount('test-iterate.csv'))`,
				`while(smaller(i, count)) { setq(numStr, csvGetCell('test-iterate.csv', i, 'num')); setq(num, toNumber(numStr)); setq(total, add(total, num)); setq(i, add(i, 1)) }`,
				`total`,
			},
			ExpectedValue: chariot.Number(30),
		},
	}

	RunTestCases(t, tests)

	// Cleanup
	folder := cfg.ChariotConfig.DataPath
	if folder == "" {
		folder = "."
	}
	os.Remove(folder + "/test-pipeline.csv")
	os.Remove(folder + "/test-iterate.csv")
}
