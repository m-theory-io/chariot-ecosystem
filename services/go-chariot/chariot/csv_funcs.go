package chariot

import (
	"encoding/csv"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"time"
)

// unwrap ScopeEntry -> Value
func unwrapCSVArg(v Value) Value {
	if se, ok := v.(ScopeEntry); ok {
		return se.Value
	}
	return v
}

// try to get *CSVNode from the first argument, or, if it's a string path,
// create a temporary CSVNode, LoadFromFile(path), and return it.
func asCSVNodeFromArg(arg Value) (*CSVNode, bool, error) {
	a := unwrapCSVArg(arg)

	// Direct CSVNode
	if n, ok := a.(*CSVNode); ok {
		return n, false, nil
	}

	// Some runtimes may wrap nodes in a generic node value; attempt to unwrap common cases
	switch t := a.(type) {
	case TreeNode: // if your codebase defines a TreeNode interface
		// Best-effort reflect unwrapping for embedded CSVNode
		if cn, ok := any(t).(*CSVNode); ok {
			return cn, false, nil
		}
	}

	// Reflection fallback (robust to different wrappers)
	rv := reflect.ValueOf(a)
	if rv.Kind() == reflect.Ptr && rv.Elem().IsValid() && rv.Elem().Type().Name() == "CSVNode" {
		if cn, ok := a.(*CSVNode); ok {
			return cn, false, nil
		}
	}

	// Convenience path: allow a string path as the "node" argument
	if s, ok := a.(Str); ok {
		cn := NewCSVNode("csv")
		// Resolve against secure data path
		fullPath, err := getSecureFilePath(string(s), "data")
		if err != nil {
			return nil, false, err
		}
		if err := cn.LoadFromFile(fullPath); err != nil {
			return nil, false, err
		}
		return cn, true, nil
	}

	return nil, false, fmt.Errorf("expected CSVNode or path string, got %T", a)
}

// RegisterCSVFunctions exposes CSVNode public methods as closures.
// First argument is a CSVNode instance; alternatively, a path string is accepted for convenience.
func RegisterCSVFunctions(rt *Runtime) {
	// csvHeaders(nodeOrPath) -> [string]
	rt.Register("csvHeaders", func(args ...Value) (Value, error) {
		if len(args) != 1 {
			return nil, fmt.Errorf("csvHeaders requires 1 argument: nodeOrPath")
		}
		n, _, err := asCSVNodeFromArg(args[0])
		if err != nil {
			return nil, err
		}
		return convertFromNativeValue(n.GetHeaders()), nil
	})

	// csvRowCount(nodeOrPath) -> number
	rt.Register("csvRowCount", func(args ...Value) (Value, error) {
		if len(args) != 1 {
			return nil, fmt.Errorf("csvRowCount requires 1 argument: nodeOrPath")
		}
		n, _, err := asCSVNodeFromArg(args[0])
		if err != nil {
			return nil, err
		}
		return Number(float64(n.GetRowCount())), nil
	})

	// csvColumnCount(nodeOrPath) -> number
	rt.Register("csvColumnCount", func(args ...Value) (Value, error) {
		if len(args) != 1 {
			return nil, fmt.Errorf("csvColumnCount requires 1 argument: nodeOrPath")
		}
		n, _, err := asCSVNodeFromArg(args[0])
		if err != nil {
			return nil, err
		}
		return Number(float64(n.GetColumnCount())), nil
	})

	// csvGetRow(nodeOrPath, index) -> map
	rt.Register("csvGetRow", func(args ...Value) (Value, error) {
		if len(args) != 2 {
			return nil, fmt.Errorf("csvGetRow requires 2 arguments: nodeOrPath, index")
		}
		n, _, err := asCSVNodeFromArg(args[0])
		if err != nil {
			return nil, err
		}
		idxVal := unwrapCSVArg(args[1])
		var idx int
		switch v := idxVal.(type) {
		case Number:
			idx = int(v)
		default:
			return nil, fmt.Errorf("index must be number, got %T", idxVal)
		}
		row, err := n.GetRow(idx)
		if err != nil {
			return nil, err
		}
		return convertFromNativeValue(row), nil
	})

	// csvGetCell(nodeOrPath, rowIndex, colIndexOrName) -> string
	rt.Register("csvGetCell", func(args ...Value) (Value, error) {
		if len(args) != 3 {
			return nil, fmt.Errorf("csvGetCell requires 3 arguments: nodeOrPath, rowIndex, colIndexOrName")
		}
		n, _, err := asCSVNodeFromArg(args[0])
		if err != nil {
			return nil, err
		}

		rowV := unwrapCSVArg(args[1])
		colV := unwrapCSVArg(args[2])

		var row int
		switch v := rowV.(type) {
		case Number:
			row = int(v)
		default:
			return nil, fmt.Errorf("rowIndex must be number, got %T", rowV)
		}

		var col interface{}
		switch v := colV.(type) {
		case Number:
			col = int(v)
		case Str:
			col = string(v)
		default:
			return nil, fmt.Errorf("colIndexOrName must be number or string, got %T", colV)
		}

		val, err := n.GetCell(row, col)
		if err != nil {
			return nil, err
		}
		return Str(val), nil
	})

	// csvGetRows(nodeOrPath) -> [[string]] (safeguarded by GetRows limitations)
	rt.Register("csvGetRows", func(args ...Value) (Value, error) {
		if len(args) != 1 {
			return nil, fmt.Errorf("csvGetRows requires 1 argument: nodeOrPath")
		}
		n, _, err := asCSVNodeFromArg(args[0])
		if err != nil {
			return nil, err
		}
		rows, err := n.GetRows()
		if err != nil {
			return nil, err
		}
		return convertFromNativeValue(rows), nil
	})

	// csvToCSV(nodeOrPath) -> string
	rt.Register("csvToCSV", func(args ...Value) (Value, error) {
		if len(args) != 1 {
			return nil, fmt.Errorf("csvToCSV requires 1 argument: nodeOrPath")
		}
		n, _, err := asCSVNodeFromArg(args[0])
		if err != nil {
			return nil, err
		}
		out, err := n.ToCSV()
		if err != nil {
			return nil, err
		}
		return Str(out), nil
	})

	// Optional helper to load a CSV into an existing node:
	// csvLoad(node, path) -> true
	rt.Register("csvLoad", func(args ...Value) (Value, error) {
		if len(args) != 2 {
			return nil, fmt.Errorf("csvLoad requires 2 arguments: node, path")
		}
		n, _, err := asCSVNodeFromArg(args[0])
		if err != nil {
			return nil, err
		}
		p := unwrapCSVArg(args[1])
		s, ok := p.(Str)
		if !ok {
			return nil, fmt.Errorf("path must be string, got %T", p)
		}
		// Resolve against secure data path
		fullPath, err := getSecureFilePath(string(s), "data")
		if err != nil {
			return nil, err
		}
		if err := n.LoadFromFile(fullPath); err != nil {
			return nil, err
		}
		return true, nil
	})
}

// === CSV SPECIFIC ===
func registerCSVFileOps(rt *Runtime) {
	rt.Register("loadCSV", func(args ...Value) (Value, error) {
		if len(args) < 1 || len(args) > 2 {
			return nil, errors.New("loadCSV requires 1-2 arguments: filepath and optional hasHeaders (boolean)")
		}

		// Unwrap arguments
		for i, arg := range args {
			if tvar, ok := arg.(ScopeEntry); ok {
				args[i] = tvar.Value
			}
		}

		fileName, ok := args[0].(Str)
		if !ok {
			return nil, fmt.Errorf("fileName must be a string, got %T", args[0])
		}

		hasHeaders := true // default
		if len(args) == 2 {
			if headerFlag, ok := args[1].(Bool); ok {
				hasHeaders = bool(headerFlag)
			}
		}

		fileNameStr := string(fileName)
		// Validate CSV file extension
		if filepath.Ext(fileNameStr) != ".csv" {
			return nil, fmt.Errorf("file must have .csv extension, got '%s'", fileNameStr)
		}
		var reader *csv.Reader
		// Support HTTP(S) sources (e.g., Azure Blob SAS URLs) for large ETL inputs
		if strings.HasPrefix(strings.ToLower(fileNameStr), "http://") || strings.HasPrefix(strings.ToLower(fileNameStr), "https://") {
			client := &http.Client{Timeout: 2 * time.Minute}
			resp, err := client.Get(fileNameStr)
			if err != nil {
				return nil, fmt.Errorf("failed to fetch CSV from URL '%s': %v", fileNameStr, err)
			}
			if resp.StatusCode < 200 || resp.StatusCode >= 300 {
				defer resp.Body.Close()
				return nil, fmt.Errorf("failed to fetch CSV from URL '%s': HTTP %d", fileNameStr, resp.StatusCode)
			}
			defer resp.Body.Close()
			reader = csv.NewReader(resp.Body)
		} else {
			// Local file path under configured DataPath
			fullPath, err := getSecureFilePath(fileNameStr, "data")
			if err != nil {
				return nil, err
			}
			file, err := os.Open(fullPath)
			if err != nil {
				return nil, fmt.Errorf("failed to open CSV file '%s': %v", fileNameStr, err)
			}
			defer file.Close()
			reader = csv.NewReader(file)
		}

		// Parse CSV
		records, err := reader.ReadAll()
		if err != nil {
			return nil, fmt.Errorf("failed to parse CSV from '%s': %v", fileNameStr, err)
		}

		if len(records) == 0 {
			// Return empty JSONNode with array
			node := NewJSONNode("csv_data")
			node.SetJSONValue([]interface{}{})
			return node, nil
		}

		var result []interface{}

		if hasHeaders && len(records) > 0 {
			// Use first row as headers
			headers := records[0]
			for i := 1; i < len(records); i++ {
				row := records[i]
				rowObj := make(map[string]interface{})

				for j, value := range row {
					if j < len(headers) {
						// Try to convert numbers
						if num, err := parseNumber(value); err == nil {
							rowObj[headers[j]] = num
						} else if value == "true" {
							rowObj[headers[j]] = true
						} else if value == "false" {
							rowObj[headers[j]] = false
						} else if value == "" {
							rowObj[headers[j]] = nil
						} else {
							rowObj[headers[j]] = value
						}
					}
				}
				result = append(result, rowObj)
			}
		} else {
			// No headers - return array of arrays
			for _, row := range records {
				var rowArray []interface{}
				for _, value := range row {
					// Try to convert numbers
					if num, err := parseNumber(value); err == nil {
						rowArray = append(rowArray, num)
					} else if value == "true" {
						rowArray = append(rowArray, true)
					} else if value == "false" {
						rowArray = append(rowArray, false)
					} else if value == "" {
						rowArray = append(rowArray, nil)
					} else {
						rowArray = append(rowArray, value)
					}
				}
				result = append(result, rowArray)
			}
		}

		// Create JSONNode with the CSV data
		node := NewJSONNode("csv_data")
		node.SetJSONValue(result)

		return node, nil
	})

	rt.Register("saveCSV", func(args ...Value) (Value, error) {
		if len(args) < 2 || len(args) > 3 {
			return nil, errors.New("saveCSV requires 2-3 arguments: data (JSONNode), filepath, and optional includeHeaders (boolean)")
		}

		// Unwrap arguments
		for i, arg := range args {
			if tvar, ok := arg.(ScopeEntry); ok {
				args[i] = tvar.Value
			}
		}

		jsonNode, ok := args[0].(*JSONNode)
		if !ok {
			return nil, fmt.Errorf("first argument must be a JSONNode, got %T", args[0])
		}

		fileName, ok := args[1].(Str)
		if !ok {
			return nil, fmt.Errorf("fileName must be a string, got %T", args[1])
		}

		fileNameStr := string(fileName)
		// Validate CSV file extension
		if filepath.Ext(fileNameStr) != ".csv" {
			return nil, fmt.Errorf("file must have .csv extension, got '%s'", fileNameStr)
		}

		// Get secure path
		fullPath, err := getSecureFilePath(fileNameStr, "data")
		if err != nil {
			return nil, err
		}

		includeHeaders := true // default
		if len(args) == 3 {
			if headerFlag, ok := args[2].(Bool); ok {
				includeHeaders = bool(headerFlag)
			}
		}

		// Get data from JSONNode
		data := jsonNode.GetJSONValue()

		// Convert to slice
		dataSlice, ok := data.([]interface{})
		if !ok {
			return nil, fmt.Errorf("JSONNode must contain an array, got %T", data)
		}

		if len(dataSlice) == 0 {
			// Create empty CSV file
			return writeCSVFile(fullPath, [][]string{})
		}

		var csvRows [][]string

		// Check if first element is an object (has headers) or array (no headers)
		if firstRow, ok := dataSlice[0].(map[string]interface{}); ok {
			// Data has objects - extract headers and convert to CSV
			var headers []string
			for key := range firstRow {
				headers = append(headers, key)
			}

			if includeHeaders {
				csvRows = append(csvRows, headers)
			}

			// Convert each object to CSV row
			for _, item := range dataSlice {
				if rowObj, ok := item.(map[string]interface{}); ok {
					var row []string
					for _, header := range headers {
						value := rowObj[header]
						row = append(row, interfaceToString(value))
					}
					csvRows = append(csvRows, row)
				}
			}
		} else {
			// Data is array of arrays
			for _, item := range dataSlice {
				if rowArray, ok := item.([]interface{}); ok {
					var row []string
					for _, value := range rowArray {
						row = append(row, interfaceToString(value))
					}
					csvRows = append(csvRows, row)
				}
			}
		}

		return writeCSVFile(fullPath, csvRows)
	})

	rt.Register("loadCSVRaw", func(args ...Value) (Value, error) {
		if len(args) != 1 {
			return nil, errors.New("loadCSVRaw requires 1 argument: filepath")
		}

		// Unwrap arguments
		for i, arg := range args {
			if tvar, ok := arg.(ScopeEntry); ok {
				args[i] = tvar.Value
			}
		}

		fileName, ok := args[0].(Str)
		if !ok {
			return nil, fmt.Errorf("fileName must be a string, got %T", args[0])
		}

		fileNameStr := string(fileName)
		// Validate CSV file extension
		if filepath.Ext(fileNameStr) != ".csv" {
			return nil, fmt.Errorf("file must have .csv extension, got '%s'", fileNameStr)
		}
		// Get secure path
		fullPath, err := getSecureFilePath(fileNameStr, "data")
		if err != nil {
			return nil, err
		}

		// Read file and return as string
		data, err := os.ReadFile(fullPath)
		if err != nil {
			return nil, fmt.Errorf("failed to read CSV file '%s': %v", fileNameStr, err)
		}

		return Str(string(data)), nil
	})

	rt.Register("saveCSVRaw", func(args ...Value) (Value, error) {
		if len(args) != 2 {
			return nil, errors.New("saveCSVRaw requires 2 arguments: csv_string and filepath")
		}

		// Unwrap arguments
		for i, arg := range args {
			if tvar, ok := arg.(ScopeEntry); ok {
				args[i] = tvar.Value
			}
		}

		csvStr, ok := args[0].(Str)
		if !ok {
			return nil, fmt.Errorf("first argument must be a CSV string, got %T", args[0])
		}

		fileName, ok := args[1].(Str)
		if !ok {
			return nil, fmt.Errorf("fileName must be a string, got %T", args[1])
		}

		fileNameStr := string(fileName)
		// Validate CSV file extension
		if filepath.Ext(fileNameStr) != ".csv" {
			return nil, fmt.Errorf("file must have .csv extension, got '%s'", fileNameStr)
		}

		// Get secure path
		fullPath, err := getSecureFilePath(fileNameStr, "data")
		if err != nil {
			return nil, err
		}

		// Write raw CSV to file
		if err := os.WriteFile(fullPath, []byte(string(csvStr)), 0644); err != nil {
			return nil, fmt.Errorf("failed to write CSV file '%s': %v", fileNameStr, err)
		}

		return Bool(true), nil
	})
}
