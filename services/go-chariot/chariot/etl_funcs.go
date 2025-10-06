package chariot

import (
	"crypto/rand"
	"encoding/csv"
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"strings"
	"time"

	cfg "github.com/bhouse1273/chariot-ecosystem/services/go-chariot/configs"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

// Add to RegisterCouchbaseFunctions or create RegisterETLFunctions
func RegisterETLFunctions(rt *Runtime) {
	// Simple map() function for creating MapValue from key-value pairs
	rt.Register("map", func(args ...Value) (Value, error) {
		// Create empty MapValue
		mapVal := NewMap()

		// If arguments provided, they should be key-value pairs
		if len(args)%2 != 0 {
			return nil, fmt.Errorf("map requires even number of arguments (key-value pairs)")
		}

		// Add key-value pairs
		for i := 0; i < len(args); i += 2 {
			key, ok := args[i].(Str)
			if !ok {
				return nil, fmt.Errorf("map keys must be strings, got %T", args[i])
			}
			value := args[i+1]
			mapVal.Set(string(key), value)
		}

		return mapVal, nil
	})

	rt.Register("doETL", func(args ...Value) (Value, error) {
		if len(args) < 4 {
			return nil, fmt.Errorf("doETL requires at least 4 arguments: jobId, csvFile, transformConfig, targetConfig")
		}

		// Unwrap arguments
		for i, arg := range args {
			if tvar, ok := arg.(ScopeEntry); ok {
				args[i] = tvar.Value
			}
		}

		// Debug: Print argument types (can be removed in production)
		// fmt.Printf("DEBUG doETL args[2] (transformConfig) type: %T\n", args[2])

		jobId := string(args[0].(Str))
		csvFile := string(args[1].(Str))
		transformConfig := args[2] // Transform configuration (should be *Transform)
		targetConfig := args[3]    // Target database configuration

		// Optional parameters
		options := make(map[string]Value)
		if len(args) > 4 {
			if optionsArg, ok := args[4].(*MapValue); ok {
				for k, v := range optionsArg.Values {
					options[k] = v
				}
			}
		}

		// Execute ETL job
		result, err := executeETLJob(rt, jobId, csvFile, transformConfig, targetConfig, options)
		if err != nil {
			return nil, err
		}

		return result, nil
	})

	rt.Register("createTransform", func(args ...Value) (Value, error) {
		if len(args) < 1 {
			return nil, fmt.Errorf("createTransform requires 1 argument: transformName")
		}

		// Handle naked symbols (like setq does)
		// The first argument should be a string that represents the transform name
		var transformName string

		// Check if first argument is already unwrapped
		arg := args[0]
		if tvar, ok := arg.(ScopeEntry); ok {
			arg = tvar.Value
		}

		if nameStr, ok := arg.(Str); ok {
			transformName = string(nameStr)
		} else {
			return nil, fmt.Errorf("createTransform: first argument must be a transform name")
		}

		transform := NewTransform(transformName)
		return transform, nil
	})

	rt.Register("addMapping", func(args ...Value) (Value, error) {
		if len(args) < 6 { // ← Changed from != 6 to < 6 for optional defaultValue
			return nil, fmt.Errorf("addMapping requires at least 6 arguments: transform, sourceField, targetColumn, program, dataType, required")
		}

		// Unwrap arguments
		for i, arg := range args {
			if tvar, ok := arg.(ScopeEntry); ok {
				args[i] = tvar.Value
			}
		}

		transform, ok := args[0].(*Transform)
		if !ok {
			// Try for string
			if name, ok := args[0].(Str); ok {
				// If first argument is a string, try to look it up in global variables
				if tvar, exists := rt.GetVariable(string(name)); exists {
					if t, ok := tvar.(*Transform); ok {
						transform = t
					} else {
						return nil, fmt.Errorf("first argument must be an existing Transform")
					}
				}
			}
		}

		// Handle program as string (supports multi-line with backticks)
		var program Str
		switch p := args[3].(type) {
		case Str:
			// Use string directly
			program = p
		case *ArrayValue:
			// Convert ArrayValue to single string (join with newlines for backward compatibility)
			lines := make([]string, p.Length())
			for i := 0; i < p.Length(); i++ {
				if str, ok := p.Get(i).(Str); ok {
					lines[i] = string(str)
				}
			}
			program = Str(strings.Join(lines, "\n"))
		default:
			return nil, fmt.Errorf("program must be string or array of strings, got %T", args[3])
		}

		mapping := FieldMapping{
			SourceField:  args[1].(Str),
			TargetColumn: args[2].(Str),
			Program:      program,
			DataType:     args[4].(Str),
			Required:     args[5].(Bool),
			DefaultValue: Str(""),
		}

		if len(args) > 6 {
			mapping.DefaultValue = args[6].(Str)
		}

		// Debug: Check mapping before adding (can be removed in production)
		// fmt.Printf("DEBUG addMapping: created mapping with Program=%+v (type=%T)\n", mapping.Program, mapping.Program)
		// fmt.Printf("DEBUG addMapping: Creating mapping with Program='%s'\n", string(mapping.Program))

		transform.AddMapping(mapping)

		// Debug: Check mappings after adding (can be removed in production)
		// retrievedMappings := transform.GetMappings()
		// if len(retrievedMappings) > 0 {
		//	lastMapping := retrievedMappings[len(retrievedMappings)-1]
		//	fmt.Printf("DEBUG addMapping: retrieved mapping has Program=%+v (type=%T)\n", lastMapping.Program, lastMapping.Program)
		// }

		return transform, nil
	})

	rt.Register("etlStatus", func(args ...Value) (Value, error) {
		if len(args) != 1 {
			return nil, fmt.Errorf("etlStatus requires 1 argument: jobId")
		}

		// Unwrap arguments
		for i, arg := range args {
			if tvar, ok := arg.(ScopeEntry); ok {
				args[i] = tvar.Value
			}
		}

		jobId := string(args[0].(Str))

		// Get ETL job status from runtime or Couchbase
		status, err := getETLJobStatus(rt, jobId)
		if err != nil {
			return nil, err
		}

		return status, nil
	})

	// Create and register transform registry
	registry := NewETLTransformRegistry()
	rt.SetVariable("etlTransformRegistry", &HostObjectValue{Value: registry})

	rt.Register("listTransforms", func(args ...Value) (Value, error) {
		transforms := registry.List()
		return convertFromNativeValue(transforms), nil
	})

	rt.Register("getTransform", func(args ...Value) (Value, error) {
		if len(args) != 1 {
			return nil, fmt.Errorf("getTransform requires 1 argument: transformName")
		}

		name := string(args[0].(Str))
		if transform, exists := registry.Get(name); exists {
			return convertFromNativeValue(map[string]interface{}{
				"name":        transform.Name,
				"description": transform.Description,
				"dataType":    transform.DataType,
				"category":    transform.Category,
				"examples":    transform.Examples,
			}), nil
		}

		return nil, fmt.Errorf("transform '%s' not found", name)
	})

	rt.Register("registerTransform", func(args ...Value) (Value, error) {
		if len(args) != 2 {
			return nil, fmt.Errorf("registerTransform requires 2 arguments: name, config")
		}

		// Unwrap arguments
		for i, arg := range args {
			if tvar, ok := arg.(ScopeEntry); ok {
				args[i] = tvar.Value
			}
		}

		name := string(args[0].(Str))

		if configMap, ok := args[1].(*MapValue); ok {
			transform := &ETLTransform{
				Name:        name,
				Description: string(configMap.Values["description"].(Str)),
				DataType:    string(configMap.Values["dataType"].(Str)),
				Category:    string(configMap.Values["category"].(Str)),
			}

			// Convert program array
			if programArray, ok := configMap.Values["program"].(*ArrayValue); ok {
				transform.Program = make([]string, len(programArray.Elements))
				for i, line := range programArray.Elements {
					transform.Program[i] = string(line.(Str))
				}
			}

			registry.Register(transform)
			return Str(fmt.Sprintf("Transform '%s' registered", name)), nil
		}

		return nil, fmt.Errorf("config must be a map")
	})

	rt.Register("addMappingWithTransform", func(args ...Value) (Value, error) {
		if len(args) < 6 {
			return nil, fmt.Errorf("addMappingWithTransform requires at least 6 arguments: transform, sourceField, targetColumn, transformName, dataType, required")
		}

		// Unwrap arguments
		for i, arg := range args {
			if tvar, ok := arg.(ScopeEntry); ok {
				args[i] = tvar.Value
			}
		}

		transform, ok := args[0].(*Transform)
		if !ok {
			return nil, fmt.Errorf("first argument must be a Transform")
		}

		mapping := FieldMapping{
			SourceField:  args[1].(Str),
			TargetColumn: args[2].(Str),
			Transform:    args[3].(Str), // Predefined transform name
			DataType:     args[4].(Str),
			Required:     args[5].(Bool),
			DefaultValue: Str(""),
		}

		if len(args) > 6 {
			mapping.DefaultValue = args[6].(Str)
		}

		transform.AddMapping(mapping)

		return transform, nil
	})

	rt.Register("getMappings", func(args ...Value) (Value, error) {
		if len(args) != 1 {
			return nil, fmt.Errorf("getMappings requires 1 argument: transform")
		}

		// Unwrap arguments
		for i, arg := range args {
			if tvar, ok := arg.(ScopeEntry); ok {
				args[i] = tvar.Value
			}
		}

		transform, ok := args[0].(*Transform)
		if !ok {
			return nil, fmt.Errorf("first argument must be a Transform")
		}

		mappings := transform.GetMappings()
		return convertFromNativeValue(mappings), nil
	})

	rt.Register("generateCreateTable", func(args ...Value) (Value, error) {
		if len(args) < 2 {
			return nil, fmt.Errorf("generateCreateTable requires at least 2 arguments: csvFile, tableName [, options]")
		}

		// Unwrap arguments
		for i, arg := range args {
			if tvar, ok := arg.(ScopeEntry); ok {
				args[i] = tvar.Value
			}
		}

		csvFile := string(args[0].(Str))
		tableName := string(args[1].(Str))

		// Optional parameters
		options := make(map[string]Value)
		if len(args) > 2 {
			if optionsArg, ok := args[2].(*MapValue); ok {
				for k, v := range optionsArg.Values {
					options[k] = v
				}
			}
		}

		// Generate CREATE TABLE statement
		createSQL, err := generateCreateTableSQL(csvFile, tableName, options)
		if err != nil {
			return nil, err
		}

		return Str(createSQL), nil
	})

	rt.Register("generateHeaders", func(args ...Value) (Value, error) {
		if len(args) < 2 {
			return nil, fmt.Errorf("generateHeaders requires 1 argument: csvFile")
		}

		// Unwrap arguments
		for i, arg := range args {
			if tvar, ok := arg.(ScopeEntry); ok {
				args[i] = tvar.Value
			}
		}

		csvFile := string(args[0].(Str))

		// Optional parameters
		options := make(map[string]Value)
		if len(args) > 2 {
			if optionsArg, ok := args[2].(*MapValue); ok {
				for k, v := range optionsArg.Values {
					options[k] = v
				}
			}
		}

		// Generate headers
		headers, err := generateCSVHeaders(csvFile)
		if err != nil {
			return nil, err
		}

		return convertFromNativeValue(headers), nil
	})

	// transform(data, func) - applies a function to each row of CSV data
	rt.Register("transform", func(args ...Value) (Value, error) {
		if len(args) != 2 {
			return nil, fmt.Errorf("transform requires 2 arguments: data, transformFunction")
		}

		// Unwrap arguments
		for i, arg := range args {
			if tvar, ok := arg.(ScopeEntry); ok {
				args[i] = tvar.Value
			}
		}

		data := args[0]
		transformFunc := args[1]

		// Check if transformFunc is a function
		funcValue, ok := transformFunc.(*FunctionValue)
		if !ok {
			return nil, fmt.Errorf("second argument must be a function")
		}

		// Handle different data types
		switch d := data.(type) {
		case *ArrayValue:
			// Data is already an ArrayValue - transform each element
			result := NewArray()
			for i := 0; i < d.Length(); i++ {
				row := d.Get(i)

				// Call the transform function with the row using executeFunctionValue
				transformedRow, err := executeFunctionValue(rt, funcValue, []Value{row})
				if err != nil {
					return nil, fmt.Errorf("error transforming row %d: %v", i, err)
				}
				result.Append(transformedRow)
			}
			return result, nil

		case *JSONNode:
			// Data is a JSONNode (from loadCSV) - get the JSON array and transform each row
			jsonValue := d.GetJSONValue()

			// Check if it's an array
			if jsonArray, ok := jsonValue.([]interface{}); ok {
				result := NewArray()

				for i, rowData := range jsonArray {
					// Convert row data to MapNode for easier access
					rowNode := NewMapNode(fmt.Sprintf("row_%d", i))

					if rowMap, ok := rowData.(map[string]interface{}); ok {
						// Set all the row properties
						for key, value := range rowMap {
							rowNode.Set(key, convertFromNativeValue(value))
						}
					} else {
						return nil, fmt.Errorf("expected row to be an object, got %T", rowData)
					}

					// Call the transform function with the row using executeFunctionValue
					transformedRow, err := executeFunctionValue(rt, funcValue, []Value{rowNode})
					if err != nil {
						return nil, fmt.Errorf("error transforming row %d: %v", i, err)
					}
					result.Append(transformedRow)
				}
				return result, nil
			} else {
				return nil, fmt.Errorf("JSONNode does not contain an array")
			}

		case *CSVNode:
			// Data is a CSVNode - get rows and transform them
			rows, err := d.GetRows()
			if err != nil {
				return nil, fmt.Errorf("error reading CSV data: %v", err)
			}

			headers := d.GetHeaders()
			result := NewArray()

			for i, row := range rows {
				// Convert row to MapNode for easier access
				rowNode := NewMapNode(fmt.Sprintf("row_%d", i))

				for j, header := range headers {
					if j < len(row) {
						rowNode.Set(header, Str(row[j]))
					}
				}

				// Call the transform function with the row using executeFunctionValue
				transformedRow, err := executeFunctionValue(rt, funcValue, []Value{rowNode})
				if err != nil {
					return nil, fmt.Errorf("error transforming row %d: %v", i, err)
				}
				result.Append(transformedRow)
			}
			return result, nil

		default:
			return nil, fmt.Errorf("first argument must be an ArrayValue, JSONNode, or CSVNode, got %T", data)
		}
	})

	// extractCSV(csvFile) - extract CSV data as ArrayValue for ETL operations
	rt.Register("extractCSV", func(args ...Value) (Value, error) {
		if len(args) < 1 || len(args) > 2 {
			return nil, fmt.Errorf("extractCSV requires 1-2 arguments: csvFile [, hasHeaders]")
		}

		// Unwrap arguments
		for i, arg := range args {
			if tvar, ok := arg.(ScopeEntry); ok {
				args[i] = tvar.Value
			}
		}

		csvFile := string(args[0].(Str))
		hasHeaders := true // default

		if len(args) > 1 {
			if headerFlag, ok := args[1].(Bool); ok {
				hasHeaders = bool(headerFlag)
			}
		}

		// Get secure file path
		csvPath, err := getSecureFilePath(csvFile, "data")
		if err != nil {
			return nil, fmt.Errorf("failed to get secure file path: %v", err)
		}

		// Create CSV node for reading
		csvNode := NewCSVNode("extract_csv")
		csvNode.SetMeta("delimiter", ",")
		csvNode.SetMeta("hasHeaders", hasHeaders)
		csvNode.SetMeta("encoding", "UTF-8")

		err = csvNode.LoadFromFile(csvPath)
		if err != nil {
			return nil, fmt.Errorf("failed to load CSV file: %v", err)
		}

		// Get rows and headers
		rows, err := csvNode.GetRows()
		if err != nil {
			return nil, fmt.Errorf("failed to read CSV rows: %v", err)
		}

		// Convert to ArrayValue of MapNodes
		result := NewArray()

		if hasHeaders && len(rows) > 0 {
			headers := csvNode.GetHeaders()

			// rows already has headers removed by CSVNode.LoadFromReader
			for i, row := range rows {
				rowNode := NewMapNode(fmt.Sprintf("row_%d", i))

				for j, header := range headers {
					if j < len(row) {
						// Try to convert numbers and booleans
						value := row[j]
						if num, err := parseNumber(value); err == nil {
							// Handle both int64 and float64 from parseNumber
							switch n := num.(type) {
							case float64:
								rowNode.Set(header, Number(n))
							case int64:
								rowNode.Set(header, Number(float64(n)))
							default:
								rowNode.Set(header, Number(float64(n.(int))))
							}
						} else if value == "true" {
							rowNode.Set(header, Bool(true))
						} else if value == "false" {
							rowNode.Set(header, Bool(false))
						} else if value == "" {
							rowNode.Set(header, DBNull)
						} else {
							rowNode.Set(header, Str(value))
						}
					}
				}

				result.Append(rowNode)
			}
		} else {
			// No headers - return array of arrays
			for _, row := range rows {
				rowArray := NewArray()
				for _, value := range row {
					// Try to convert numbers and booleans
					if num, err := parseNumber(value); err == nil {
						// Handle both int64 and float64 from parseNumber
						switch n := num.(type) {
						case float64:
							rowArray.Append(Number(n))
						case int64:
							rowArray.Append(Number(float64(n)))
						default:
							rowArray.Append(Number(float64(n.(int))))
						}
					} else if value == "true" {
						rowArray.Append(Bool(true))
					} else if value == "false" {
						rowArray.Append(Bool(false))
					} else if value == "" {
						rowArray.Append(DBNull)
					} else {
						rowArray.Append(Str(value))
					}
				}
				result.Append(rowArray)
			}
		}

		return result, nil
	})

	// Register test support functions
	RegisterFieldMappingTestFunctions(rt)
}

// Add this function (replaces the current ProcessETLJob logic):
func executeETLJob(rt *Runtime, jobId, csvFile string, transformConfig, targetConfig Value, options map[string]Value) (Value, error) {
	// Create ETL job TreeNode hierarchy
	etlJob := NewTreeNode(fmt.Sprintf("etl_job_%s", jobId))
	etlJob.SetMeta("jobId", jobId)
	etlJob.SetMeta("clientId", getOptionString(options, "clientId", "unknown"))
	etlJob.SetMeta("startTime", time.Now().Format(time.RFC3339)) // ✅ RFC3339
	etlJob.SetMeta("status", "initializing")
	etlJob.SetMeta("tableName", targetConfig.(*MapValue).Values["tableName"])

	// 1. Create and configure CSV node
	csvNode := NewCSVNode("source_data")
	csvNode.SetMeta("filename", csvFile)
	csvNode.SetMeta("delimiter", getOptionString(options, "delimiter", ","))
	csvNode.SetMeta("hasHeaders", getOptionBool(options, "hasHeaders", true))
	csvNode.SetMeta("encoding", getOptionString(options, "encoding", "UTF-8"))

	// Get secure file path for the CSV file
	csvPath, err := GetSecureFilePath(csvFile, "data")
	if err != nil {
		return nil, fmt.Errorf("failed to get secure file path for %s: %v", csvFile, err)
	}
	csvNode.SetMeta("resolvedPath", csvPath)

	// Parse headers only for validation (lazy loading approach)
	delimiter := getOptionString(options, "delimiter", ",")
	hasHeaders := getOptionBool(options, "hasHeaders", true)
	headers, err := parseCSVHeaders(csvPath, delimiter, hasHeaders)
	if err != nil {
		return nil, fmt.Errorf("failed to parse CSV headers: %v", err)
	}

	// Store headers as ArrayValue
	headersArray := NewArray()
	for _, header := range headers {
		headersArray.Append(Str(header))
	}
	csvNode.SetMeta("headers", headersArray)
	csvNode.SetMeta("dataLoaded", false)

	// Initialize the CSV node with the file for streaming processing
	err = csvNode.LoadFromFile(csvPath)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize CSV node with file %s: %v", csvPath, err)
	}

	etlJob.AddChild(csvNode)

	// 2. Create transform from configuration
	var transform *Transform

	// Debug: Print detailed type information (can be removed in production)
	// fmt.Printf("DEBUG transformConfig type: %T\n", transformConfig)
	// fmt.Printf("DEBUG transformConfig value: %+v\n", transformConfig)

	switch t := transformConfig.(type) {
	case *Transform:
		// Direct transform object
		// fmt.Printf("DEBUG: Using direct *Transform\n")
		transform = t
	case *TreeNodeImpl:
		// Check if this TreeNodeImpl is actually a Transform
		if t.GetType() == ValueObject {
			// Create a new Transform based on the TreeNodeImpl
			transform = NewTransform(t.Name())
			// Copy attributes and children from the TreeNodeImpl
			for key, value := range t.GetAttributes() {
				transform.SetAttribute(key, value)
			}
			for _, child := range t.GetChildren() {
				transform.AddChild(child)
			}
			// fmt.Printf("DEBUG: Created Transform from TreeNodeImpl\n")
		} else {
			// fmt.Printf("DEBUG: TreeNodeImpl is not a transform type, got: %s\n", t.GetType().String())
			return nil, fmt.Errorf("TreeNode is not a Transform, got node type: %v", t.GetType())
		}
	case *MapValue:
		// Transform configuration as map
		// fmt.Printf("DEBUG: Creating new transform from MapValue config\n")
		transform = NewTransform("etl_transform")

		// Extract mappings from configuration
		if mappingsVal, exists := t.Values["mappings"]; exists {
			if mappingsArray, ok := mappingsVal.(*ArrayValue); ok {
				for _, mappingVal := range mappingsArray.Elements {
					if mappingMap, ok := mappingVal.(*MapValue); ok {
						mapping := FieldMapping{
							SourceField:  mappingMap.Values["sourceField"].(Str),
							TargetColumn: mappingMap.Values["targetColumn"].(Str),
							DataType:     mappingMap.Values["dataType"].(Str),
							Required:     mappingMap.Values["required"].(Bool),
						}

						// Handle program as string array or transform name
						if transformName, exists := mappingMap.Values["transform"]; exists {
							mapping.Transform = transformName.(Str)
						} else if programVal, exists := mappingMap.Values["program"]; exists {
							if programArray, ok := programVal.(*ArrayValue); ok {
								// Convert ArrayValue to string (join with newlines)
								lines := make([]string, programArray.Length())
								for i := 0; i < programArray.Length(); i++ {
									if str, ok := programArray.Get(i).(Str); ok {
										lines[i] = string(str)
									}
								}
								mapping.Program = Str(strings.Join(lines, "\n"))
							} else if programStr, ok := programVal.(Str); ok {
								// Use string directly
								mapping.Program = programStr
							}
						}

						if defaultVal, exists := mappingMap.Values["defaultValue"]; exists {
							mapping.DefaultValue = defaultVal.(Str)
						}

						transform.AddMapping(mapping)
					}
				}
			}
		}
	case TreeNode:
		// Try to cast TreeNode to Transform (handles *TreeNodeImpl case)
		// fmt.Printf("DEBUG: Got TreeNode, attempting cast to *Transform\n")
		if transformNode, ok := t.(*Transform); ok {
			// fmt.Printf("DEBUG: Successfully cast TreeNode to *Transform\n")
			transform = transformNode
		} else {
			// fmt.Printf("DEBUG: Failed to cast TreeNode to *Transform, actual type: %T\n", t)
			return nil, fmt.Errorf("TreeNode is not a Transform: %T", t)
		}
	default:
		// fmt.Printf("DEBUG: Unknown transform config type: %T\n", transformConfig)
		return nil, fmt.Errorf("invalid transform configuration type: %T", transformConfig)
	}

	etlJob.AddChild(transform)

	// 3. Create target configuration node (for ProcessETLJob compatibility)
	targetNode := NewJSONNode("target_config")
	targetNode.SetJSONValue(convertValueToNative(targetConfig))
	etlJob.AddChild(targetNode)

	// 4. Create log node for tracking (must be at index 3 for ProcessETLJob)
	logNode := NewJSONNode("processing_log")
	etlJob.AddChild(logNode)

	// 5. Initialize actual target database nodes and store in ETL job metadata
	var actualTargetNode TreeNode
	if targetMap, ok := targetConfig.(*MapValue); ok {
		// Check if type field exists and is not nil
		typeValue, exists := targetMap.Values["type"]
		if !exists || typeValue == nil {
			return nil, fmt.Errorf("target configuration missing required 'type' field")
		}

		typeStr, ok := typeValue.(Str)
		if !ok {
			return nil, fmt.Errorf("target configuration 'type' field must be a string, got %T", typeValue)
		}
		targetType := string(typeStr)

		switch targetType {
		case "sql":
			// Check if a connection name is provided (use existing connection)
			if connectionName, exists := targetMap.Values["connectionName"]; exists && connectionName != nil {
				if connectionNameStr, ok := connectionName.(Str); ok {
					connectionNameStrValue := string(connectionNameStr)

					// Debug: Show what we're looking for and what's available
					cfg.ChariotLogger.Info("Looking for SQL connection", zap.String("connection_name", connectionNameStrValue))
					cfg.ChariotLogger.Info("Available runtime objects", zap.Int("count", len(rt.objects)))
					for key, obj := range rt.objects {
						cfg.ChariotLogger.Info("Runtime object", zap.String("key", key), zap.String("type", fmt.Sprintf("%T", obj)))
					}

					// Look for existing SQL connection in runtime objects
					if existingConnection, exists := rt.objects[connectionNameStrValue]; exists {
						if sqlNode, ok := existingConnection.(*SQLNode); ok {
							// Use existing connected SQLNode
							actualTargetNode = sqlNode
							cfg.ChariotLogger.Info("Using existing SQL connection", zap.String("connection_name", connectionNameStrValue))
						} else {
							return nil, fmt.Errorf("object '%s' is not a SQLNode", connectionNameStrValue)
						}
					} else {
						return nil, fmt.Errorf("SQL connection '%s' not found in runtime", connectionNameStrValue)
					}
				} else {
					return nil, fmt.Errorf("connectionName must be a string, got %T", connectionName)
				}
			} else {
				// Fallback: Create new SQL connection (legacy behavior)
				sqlNode := NewSQLNode("target_sql")
				sqlNode.SetMeta("driver", string(targetMap.Values["driver"].(Str)))
				sqlNode.SetMeta("connectionString", string(targetMap.Values["connectionString"].(Str)))
				sqlNode.SetMeta("tableName", string(targetMap.Values["tableName"].(Str)))

				if batchSizeVal, exists := targetMap.Values["batchSize"]; exists {
					sqlNode.SetMeta("batchSize", int(batchSizeVal.(Number)))
				} else {
					sqlNode.SetMeta("batchSize", 1000)
				}

				// Initialize SQL connection
				err := sqlNode.ConnectMeta()
				if err != nil {
					return nil, fmt.Errorf("failed to connect to SQL database: %v", err)
				}

				actualTargetNode = sqlNode
			}

		case "couchbase":
			// Check if a connection name is provided (use existing connection)
			if connectionName, exists := targetMap.Values["connectionName"]; exists {
				connectionNameStr := string(connectionName.(Str))

				// Look for existing Couchbase connection in runtime
				if existingConnection, exists := rt.GetVariable(connectionNameStr); exists {
					if cbNode, ok := existingConnection.(*CouchbaseNode); ok {
						// Use existing connected CouchbaseNode
						actualTargetNode = cbNode
						fmt.Printf("✅ Using existing Couchbase connection: %s\n", connectionNameStr)
					} else {
						return nil, fmt.Errorf("variable '%s' is not a CouchbaseNode", connectionNameStr)
					}
				} else {
					return nil, fmt.Errorf("Couchbase connection '%s' not found in runtime", connectionNameStr)
				}
			} else {
				// Fallback: Create new Couchbase connection (legacy behavior)
				cbNode := NewCouchbaseNode("target_couchbase")
				cbNode.SetMeta("connectionString", string(targetMap.Values["connectionString"].(Str)))
				cbNode.SetMeta("bucket", string(targetMap.Values["bucket"].(Str)))
				cbNode.SetMeta("scope", string(targetMap.Values["scope"].(Str)))
				cbNode.SetMeta("collection", string(targetMap.Values["collection"].(Str)))

				// Initialize Couchbase connection using metadata
				err := cbNode.ConnectMeta()
				if err != nil {
					return nil, fmt.Errorf("failed to connect to Couchbase: %v", err)
				}

				actualTargetNode = cbNode
			}

		case "test":
			// For testing - no actual database
			testNode := NewJSONNode("target_test")
			testNode.SetMeta("type", "test")
			actualTargetNode = testNode

		default:
			return nil, fmt.Errorf("unsupported target type: %s", targetType)
		}

		// Store the actual target node in ETL job metadata for ProcessETLJob to access
		etlJob.SetMeta("actualTargetNode", actualTargetNode)
	} else {
		return nil, fmt.Errorf("target configuration must be a map")
	}

	// 5. Execute the ETL pipeline
	err = ProcessETLJob(rt, etlJob)
	if err != nil {
		return nil, fmt.Errorf("ETL job failed: %v", err)
	}

	// 6. Return job result
	result := NewJSONNode("etl_result")
	result.SetJSONValue(map[string]interface{}{
		"jobId":            jobId,
		"status":           GetMetaString(etlJob, "status", "unknown"),
		"startTime":        GetMetaTimeString(etlJob, "startTime", time.Now()), // ✅ RFC3339
		"endTime":          GetMetaTimeString(etlJob, "endTime", time.Time{}),  // ✅ RFC3339
		"rowsProcessed":    GetMetaInt(csvNode, "rowsProcessed", 0),            // ✅ Fixed: csvNode
		"batchesProcessed": GetMetaInt(csvNode, "batchesProcessed", 0),         // ✅ Fixed: csvNode
		"duration":         GetMetaDurationString(etlJob, "totalDuration", 0),  // ✅ Duration as string
		"archivePath":      GetMetaString(csvNode, "archivedTo", ""),
	})

	return result, nil
}

// Add this function to etl_funcs.go after executeETLJob:

func ProcessETLJob(rt *Runtime, etlJob TreeNode) error {
	// Extract components from ETL job hierarchy
	children := etlJob.GetChildren()
	if len(children) < 4 {
		return fmt.Errorf("ETL job missing required components")
	}

	csvNode, ok := children[0].(*CSVNode)
	if !ok {
		return fmt.Errorf("first child must be CSVNode")
	}

	transform, ok := children[1].(*Transform)
	if !ok {
		return fmt.Errorf("second child must be Transform")
	}

	// targetNode := children[2] // JSON config for now
	logNode, ok := children[3].(*JSONNode)
	if !ok {
		return fmt.Errorf("fourth child must be JSONNode for logging")
	}

	// Initialize processing log
	processingLog := map[string]interface{}{
		"jobId":       GetMetaString(etlJob, "jobId", "unknown"), // ✅ Safe
		"startTime":   time.Now().Format(time.RFC3339),           // ✅ RFC3339
		"status":      "processing",
		"batches":     []map[string]interface{}{},
		"errors":      []string{},
		"totalRows":   0,
		"successRows": 0,
		"errorRows":   0,
	}

	batchSize := 1000 // Process 1000 rows at a time
	globalRowIndex := 0

	// Update ETL job status
	etlJob.SetMeta("status", "processing")

	err := csvNode.StreamProcess(batchSize, func(batch [][]string) error {
		batchStartTime := time.Now()
		batchNumber := GetMetaInt(csvNode, "batchesProcessed", 0)
		batchNumber++
		csvNode.SetMeta("batchesProcessed", batchNumber)
		batchLog := map[string]interface{}{
			"batchNumber": batchNumber,
			"rowCount":    len(batch),
			"startTime":   batchStartTime.Format(time.RFC3339), // ✅ RFC3339
			"successRows": 0,
			"errorRows":   0,
			"errors":      []string{},
		}

		// Convert CSV headers for row mapping
		headers := csvNode.GetHeaders()

		// Process each row in the batch
		successCount := 0
		errorCount := 0

		// Start a transaction for this batch (if SQL target)
		var sqlNodeForBatch *SQLNode
		if len(batch) > 0 {
			// Get the actual target node from ETL job metadata
			actualTargetNodeMeta, exists := etlJob.GetMeta("actualTargetNode")
			if exists {
				if actualTargetNode, ok := actualTargetNodeMeta.(TreeNode); ok {
					targetNode := children[2]
					if targetConfig, ok := targetNode.(*JSONNode); ok {
						targetData := targetConfig.GetJSONValue()
						if targetMap, ok := targetData.(map[string]interface{}); ok {
							targetType := targetMap["type"].(string)
							if targetType == "sql" {
								if sqlNode, ok := actualTargetNode.(*SQLNode); ok {
									sqlNodeForBatch = sqlNode
									// Start transaction for the batch
									err := sqlNode.Begin()
									if err != nil {
										cfg.ChariotLogger.Warn("Failed to start transaction, using autocommit", zap.Error(err))
									}
								}
							}
						}
					}
				}
			}
		}

		for i, csvRow := range batch {
			// Convert CSV row to map
			rowMap := make(map[string]string)
			for j, value := range csvRow {
				if j < len(headers) {
					rowMap[headers[j]] = value
				} else {
					// Handle rows with more columns than headers
					rowMap[fmt.Sprintf("col_%d", j)] = value
				}
			}

			// Apply transformation
			sqlRow, err := transform.ApplyToRow(rt, rowMap)
			if err != nil {
				errorMsg := fmt.Sprintf("Row %d: %v", globalRowIndex+i, err)
				batchLog["errors"] = append(batchLog["errors"].([]string), errorMsg)
				errorCount++
				continue
			}
			// Insert into target database
			if len(sqlRow) > 0 {
				// Get the actual target node from ETL job metadata
				actualTargetNodeMeta, exists := etlJob.GetMeta("actualTargetNode")
				if !exists {
					errorMsg := fmt.Sprintf("No target node found in ETL job metadata for row %d", globalRowIndex+i)
					batchLog["errors"] = append(batchLog["errors"].([]string), errorMsg)
					errorCount++
					continue
				}

				actualTargetNode, ok := actualTargetNodeMeta.(TreeNode)
				if !ok {
					errorMsg := fmt.Sprintf("Invalid target node type in metadata for row %d", globalRowIndex+i)
					batchLog["errors"] = append(batchLog["errors"].([]string), errorMsg)
					errorCount++
					continue
				}
				// Populate tableName meta from etlJob meta
				tableName, ok := etlJob.GetMeta("tableName")
				if ok {
					actualTargetNode.SetMeta("tableName", tableName)
				}

				// Get target configuration to determine type
				targetNode := children[2]
				if targetConfig, ok := targetNode.(*JSONNode); ok {
					targetData := targetConfig.GetJSONValue()
					if targetMap, ok := targetData.(map[string]interface{}); ok {
						targetType := targetMap["type"].(string)

						switch targetType {
						case "sql":
							// Cast to SQL node and insert
							if sqlNode, ok := actualTargetNode.(*SQLNode); ok {
								// fmt.Printf("sqlRow: %v", sqlRow)
								err := sqlNode.Insert(sqlRow)
								if err != nil {
									errorMsg := fmt.Sprintf("SQL insert failed for row %d: %v", globalRowIndex+i, err)
									batchLog["errors"] = append(batchLog["errors"].([]string), errorMsg)
									errorCount++
								} else {
									successCount++
								}
							} else {
								errorMsg := fmt.Sprintf("Target node is not a SQLNode for row %d", globalRowIndex+i)
								batchLog["errors"] = append(batchLog["errors"].([]string), errorMsg)
								errorCount++
							}
						case "couchbase":
							// Cast to Couchbase node and insert
							if cbNode, ok := actualTargetNode.(*CouchbaseNode); ok {
								docId := generateDocIdWithConfig(sqlRow, cbNode)
								_, err := cbNode.Insert(docId, sqlRow, 0)
								if err != nil {
									errorMsg := fmt.Sprintf("Couchbase insert failed for row %d: %v", globalRowIndex+i, err)
									batchLog["errors"] = append(batchLog["errors"].([]string), errorMsg)
									errorCount++
								} else {
									successCount++
								}
							} else {
								errorMsg := fmt.Sprintf("Target node is not a CouchbaseNode for row %d", globalRowIndex+i)
								batchLog["errors"] = append(batchLog["errors"].([]string), errorMsg)
								errorCount++
							}
						case "test":
							// For testing - just count as success
							successCount++
						default:
							// Just count as success for testing
							successCount++
						}
					} else {
						successCount++ // Fallback for testing
					}
				} else {
					successCount++ // Fallback for testing
				}
			} else {
				errorCount++
			}
		}

		// Commit the transaction if we started one for this batch
		if sqlNodeForBatch != nil {
			err := sqlNodeForBatch.Commit()
			if err != nil {
				cfg.ChariotLogger.Error("Failed to commit transaction", zap.Error(err))
				// If commit fails, the inserts will be rolled back automatically
			} else {
				cfg.ChariotLogger.Info("Successfully committed batch transaction",
					zap.Int("batch_number", batchNumber),
					zap.Int("success_rows", successCount))
			}
		}

		// Update batch statistics
		batchLog["successRows"] = successCount
		batchLog["errorRows"] = errorCount
		batchLog["endTime"] = time.Now().Format(time.RFC3339)      // ✅ RFC3339
		batchLog["duration"] = time.Since(batchStartTime).String() // ✅ Duration string

		// Update global processing statistics
		totalRowsProcessed := GetMetaInt(csvNode, "rowsProcessed", 0) + len(batch)
		csvNode.SetMeta("rowsProcessed", totalRowsProcessed) // ✅ Track total

		processingLog["totalRows"] = processingLog["totalRows"].(int) + len(batch)
		processingLog["successRows"] = processingLog["successRows"].(int) + successCount
		processingLog["errorRows"] = processingLog["errorRows"].(int) + errorCount
		globalRowIndex += len(batch)

		// Append to processing log
		batches := processingLog["batches"].([]map[string]interface{})
		processingLog["batches"] = append(batches, batchLog)

		// Update log document every batch
		logNode.SetJSONValue(processingLog)

		// Optional: Update log in Couchbase if connection available
		if couchbaseNode, exists := rt.GetVariable("couchbaseConnection"); exists {
			if cbNode, ok := couchbaseNode.(interface{ Upsert(string, TreeNode) error }); ok {
				jobId := GetMetaString(etlJob, "jobId", "unknown") // ✅ Safe
				err := cbNode.Upsert(jobId, logNode)
				if err != nil {
					cfg.ChariotLogger.Warn("Failed to update log in Couchbase", zap.Error(err))
				}
			}
		}

		return nil
	})

	if err != nil {
		// Handle processing error
		processingLog["status"] = "failed"
		processingLog["error"] = err.Error()
		processingLog["endTime"] = time.Now().Format(time.RFC3339)

		// Update ETL job status
		etlJob.SetMeta("status", "failed")
		etlJob.SetMeta("error", err.Error())
		etlJob.SetMeta("endTime", time.Now().Format(time.RFC3339))

		// Send failure message to NSQ (if implemented)
		nsqErr := sendNSQMessage("etl.failed", processingLog)
		if nsqErr != nil {
			cfg.ChariotLogger.Warn("Failed to send NSQ message", zap.Error(nsqErr))
		}

		return err
	}

	// Success path
	processingLog["endTime"] = time.Now().Format(time.RFC3339) // ✅ RFC3339
	startTimeStr := processingLog["startTime"].(string)
	if startTime, err := time.Parse(time.RFC3339, startTimeStr); err == nil {
		processingLog["totalDuration"] = time.Since(startTime).String() // ✅ Duration string
	}
	// Update ETL job metadata
	etlJob.SetMeta("status", "completed")
	etlJob.SetMeta("endTime", time.Now().Format(time.RFC3339))
	etlJob.SetMeta("totalDuration", processingLog["totalDuration"])

	// Archive file
	if sourceFile, exists := csvNode.GetMeta("sourceFile"); exists {
		jobId := GetMetaString(etlJob, "jobId", "unknown") // ✅ Safe

		// Create archive directory if it doesn't exist
		archiveDir := "/archive"
		if err := os.MkdirAll(archiveDir, 0755); err != nil {
			processingLog["archiveError"] = fmt.Sprintf("Failed to create archive directory: %v", err)
		} else {
			archivePath := fmt.Sprintf("%s/%s_%s", archiveDir, jobId, filepath.Base(sourceFile.(string)))

			err = os.Rename(sourceFile.(string), archivePath)
			if err != nil {
				processingLog["archiveError"] = err.Error()
			} else {
				processingLog["archivedTo"] = archivePath
				csvNode.SetMeta("archivedTo", archivePath)
				setMetaTime(csvNode, "archivedAt", time.Now()) // ✅ RFC3339
				etlJob.SetMeta("archivedTo", archivePath)
			}
		}
	}

	// Final log update
	logNode.SetJSONValue(processingLog)

	// Update log in Couchbase if connection available
	if couchbaseNode, exists := rt.GetVariable("couchbaseConnection"); exists {
		if cbNode, ok := couchbaseNode.(interface{ Upsert(string, TreeNode) error }); ok {
			jobId := GetMetaString(etlJob, "jobId", "unknown") // ✅ Safe
			err := cbNode.Upsert(jobId, logNode)
			if err != nil {
				cfg.ChariotLogger.Warn("Failed to update log in Couchbase", zap.Error(err))
			}
		}
	}

	// Send success message to NSQ
	nsqErr := sendNSQMessage("etl.completed", processingLog)
	if nsqErr != nil {
		cfg.ChariotLogger.Warn("Failed to send NSQ message", zap.Error(nsqErr))
	}

	return nil
}

// Add helper functions at the end of the file:

// Add this helper function before executeETLJob:
func GetMetaInt(node TreeNode, key string, defaultValue int) int {
	if val, exists := node.GetMeta(key); exists {
		switch v := val.(type) {
		case int:
			return v
		case int64:
			return int(v)
		case float64:
			return int(v)
		}
	}
	return defaultValue
}

func GetMetaString(node TreeNode, key string, defaultValue string) string {
	if val, exists := node.GetMeta(key); exists {
		tstr := convertFromChariotValue(val)
		if str, ok := tstr.(string); ok {
			return str
		}
	}
	return defaultValue
}

func GetMetaInt64(node TreeNode, key string, defaultValue int64) int64 {
	if val, exists := node.GetMeta(key); exists {
		switch v := val.(type) {
		case int64:
			return v
		case int:
			return int64(v)
		case float64:
			return int64(v)
		}
	}
	return defaultValue
}

// Helper functions for option extraction
func getOptionString(options map[string]Value, key, defaultValue string) string {
	if val, exists := options[key]; exists {
		if str, ok := val.(Str); ok {
			return string(str)
		}
	}
	return defaultValue
}

func getOptionBool(options map[string]Value, key string, defaultValue bool) bool {
	if val, exists := options[key]; exists {
		if b, ok := val.(Bool); ok {
			return bool(b)
		}
	}
	return defaultValue
}

// Add new helper functions for RFC3339 time handling:
func GetMetaTimeString(node TreeNode, key string, defaultValue time.Time) string {
	if val, exists := node.GetMeta(key); exists {
		switch v := val.(type) {
		case time.Time:
			return v.Format(time.RFC3339)
		case int64:
			// Convert Unix timestamp to RFC3339
			return time.Unix(v, 0).Format(time.RFC3339)
		case string:
			// Already a string, validate and return
			if _, err := time.Parse(time.RFC3339, v); err == nil {
				return v
			}
		}
	}
	if defaultValue.IsZero() {
		return ""
	}
	return defaultValue.Format(time.RFC3339)
}

func GetMetaDurationString(node TreeNode, key string, defaultSeconds float64) string {
	if val, exists := node.GetMeta(key); exists {
		switch v := val.(type) {
		case float64:
			return fmt.Sprintf("%.2fs", v)
		case time.Duration:
			return v.String()
		case string:
			return v
		}
	}
	return fmt.Sprintf("%.2fs", defaultSeconds)
}

func setMetaTime(node TreeNode, key string, t time.Time) {
	node.SetMeta(key, t.Format(time.RFC3339))
}

//lint:ignore U1000 This function is used in the code
func setMetaDuration(node TreeNode, key string, d time.Duration) {
	node.SetMeta(key, d.String())
}

//lint:ignore U1000 This function is used in the code
func getOptionInt(options map[string]Value, key string, defaultValue int) int {
	if val, exists := options[key]; exists {
		if num, ok := val.(Number); ok {
			return int(num)
		}
	}
	return defaultValue
}

func getETLJobStatus(rt *Runtime, jobId string) (Value, error) {
	// Try to get job status from Couchbase or runtime storage
	if couchbaseNode, exists := rt.GetVariable("couchbaseConnection"); exists {
		if cbNode, ok := couchbaseNode.(interface {
			Get(string) (TreeNode, error)
		}); ok {
			logDoc, err := cbNode.Get(jobId)
			if err != nil {
				return Str("not_found"), nil
			}
			return logDoc, nil
		}
	}

	return Str("unknown"), nil
}

// Placeholder for NSQ integration
func sendNSQMessage(topic string, data interface{}) error {
	// TODO: Implement NSQ message sending
	// For now, just log the message
	fmt.Printf("NSQ Message to %s: %+v\n", topic, data)
	return nil
}

// Helper function to generate document ID for Couchbase
func generateDocId(prefix string, format string, sqlRow map[string]interface{}) string {
	// Try to get type prefix from the row data or use default

	typePrefix := prefix
	if prefix == "" {
		typePrefix = "doc"
	}

	if len(sqlRow) > 0 {

		// Look for common type indicators in the row
		if tableName, exists := sqlRow["table_name"]; exists {
			typePrefix = fmt.Sprintf("%v", tableName)
		} else if entityType, exists := sqlRow["type"]; exists {
			typePrefix = fmt.Sprintf("%v", entityType)
		} else if category, exists := sqlRow["category"]; exists {
			typePrefix = fmt.Sprintf("%v", category)
		}
	}

	// Clean the prefix (remove spaces, convert to lowercase)
	typePrefix = strings.ToLower(strings.ReplaceAll(typePrefix, " ", "_"))

	// Check if there's a preference for ID format (could be set in metadata or config)
	// For now, default to the shorter format, but make it configurable
	useShortFormat := (format == "short") // This could come from configuration

	if useShortFormat {
		return fmt.Sprintf("%s:%s", typePrefix, generateShortId())
	} else {
		return fmt.Sprintf("%s:%s", typePrefix, uuid.New().String())
	}
}

// Generate a random 10-character alphanumeric string
func generateShortId() string {
	const charset = "abcdefghijklmnopqrstuvwxyz0123456789"
	const length = 10

	result := make([]byte, length)
	for i := range result {
		num, _ := rand.Int(rand.Reader, big.NewInt(int64(len(charset))))
		result[i] = charset[num.Int64()]
	}

	return string(result)
}

// Enhanced version that can be configured via metadata
func generateDocIdWithConfig(sqlRow map[string]interface{}, targetNode TreeNode) string {
	// Get configuration from target node metadata
	idFormat := GetMetaString(targetNode, "idFormat", "short") // "short" or "uuid"
	typePrefix := GetMetaString(targetNode, "typePrefix", "doc")

	// Override type prefix if specified in row data
	if tableName, exists := sqlRow["table_name"]; exists {
		typePrefix = fmt.Sprintf("%v", tableName)
	} else if entityType, exists := sqlRow["type"]; exists {
		typePrefix = fmt.Sprintf("%v", entityType)
	}

	// Clean the prefix
	typePrefix = strings.ToLower(strings.ReplaceAll(typePrefix, " ", "_"))

	// Generate ID based on format preference
	switch idFormat {
	case "uuid":
		return fmt.Sprintf("%s:%s", typePrefix, uuid.New().String())
	case "timestamp":
		return fmt.Sprintf("%s:%d", typePrefix, time.Now().UnixNano())
	default: // "short"
		return fmt.Sprintf("%s:%s", typePrefix, generateShortId())
	}
}

// Generate CREATE TABLE SQL statement from CSV headers
func generateCreateTableSQL(csvFile, tableName string, options map[string]Value) (string, error) {
	// Create a temporary CSV node to read headers
	csvNode := NewCSVNode("temp_csv")
	csvNode.SetMeta("delimiter", getOptionString(options, "delimiter", ","))
	csvNode.SetMeta("hasHeaders", getOptionBool(options, "hasHeaders", true))
	csvNode.SetMeta("encoding", getOptionString(options, "encoding", "UTF-8"))

	csvPath, err := getSecureFilePath(csvFile, "data")
	if err != nil {
		return "", fmt.Errorf("failed to get secure file path: %v", err)
	}

	// Check if headers are required
	hasHeaders := getOptionBool(options, "hasHeaders", true)
	if !hasHeaders {
		return "", fmt.Errorf("generateCreateTable requires CSV file with headers")
	}

	// Read just the headers (don't load full file)
	file, err := os.Open(csvPath)
	if err != nil {
		return "", fmt.Errorf("failed to open CSV file: %v", err)
	}
	defer file.Close()

	// Get delimiter
	delimiter := getOptionString(options, "delimiter", ",")
	reader := csv.NewReader(file)
	reader.Comma = rune(delimiter[0])

	// Read headers
	headers, err := reader.Read()
	if err != nil {
		return "", fmt.Errorf("failed to read CSV headers: %v", err)
	}

	// Get database type and options
	dbType := getOptionString(options, "dbType", "mysql") // mysql, postgres, mssql, sqlite
	primaryKey := getOptionString(options, "primaryKey", "id")
	addPrimaryKey := getOptionBool(options, "addPrimaryKey", true)

	// Build CREATE TABLE statement
	var sql strings.Builder
	sql.WriteString(fmt.Sprintf("CREATE TABLE %s (\n", escapeIdentifier(tableName, dbType)))

	// Add primary key if requested and not already in headers
	if addPrimaryKey && !containsHeader(headers, primaryKey) {
		sql.WriteString(fmt.Sprintf("    %s %s NOT NULL,\n",
			escapeIdentifier(primaryKey, dbType),
			getPrimaryKeyType(dbType)))
	}

	// Add columns for each header
	for i, header := range headers {
		columnName := sanitizeColumnName(header)
		columnType := getVarcharMaxType(dbType)
		nullable := "NULL"

		// Check if this is the primary key
		if columnName == primaryKey {
			columnType = getPrimaryKeyType(dbType)
			nullable = "NOT NULL"
		}

		sql.WriteString(fmt.Sprintf("    %s %s %s",
			escapeIdentifier(columnName, dbType),
			columnType,
			nullable))

		// Add comma except for last column (but always add if we have PK constraint coming)
		if i < len(headers)-1 || addPrimaryKey {
			sql.WriteString(",")
		}
		sql.WriteString("\n")
	}

	// Add primary key constraint if we added a PK column
	if addPrimaryKey {
		pkColumn := primaryKey
		if containsHeader(headers, primaryKey) {
			pkColumn = sanitizeColumnName(primaryKey)
		}
		sql.WriteString(fmt.Sprintf("    PRIMARY KEY (%s)\n", escapeIdentifier(pkColumn, dbType)))
	}

	sql.WriteString(");")

	return sql.String(), nil
}

func generateCSVHeaders(csvFile string) ([]string, error) {
	csvPath, err := getSecureFilePath(csvFile, "data")
	if err != nil {
		return nil, fmt.Errorf("failed to get secure file path: %v", err)
	}

	// Read just the headers (don't load full file)
	file, err := os.Open(csvPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open CSV file: %v", err)
	}
	defer file.Close()

	// Get delimiter
	reader := csv.NewReader(file)
	reader.Comma = rune(","[0]) // Default to comma, can be overridden

	// Read headers
	headers, err := reader.Read()
	if err != nil {
		return nil, fmt.Errorf("failed to read CSV headers: %v", err)
	}

	return headers, nil
}

// Helper functions for SQL generation
func escapeIdentifier(identifier, dbType string) string {
	switch strings.ToLower(dbType) {
	case "mysql":
		return fmt.Sprintf("`%s`", identifier)
	case "postgres", "postgresql":
		return fmt.Sprintf("\"%s\"", identifier)
	case "mssql", "sqlserver":
		return fmt.Sprintf("[%s]", identifier)
	case "sqlite":
		return fmt.Sprintf("\"%s\"", identifier)
	default:
		return identifier
	}
}

func getVarcharMaxType(dbType string) string {
	switch strings.ToLower(dbType) {
	case "mysql":
		return "TEXT"
	case "postgres", "postgresql":
		return "TEXT"
	case "mssql", "sqlserver":
		return "VARCHAR(MAX)"
	case "sqlite":
		return "TEXT"
	default:
		return "TEXT"
	}
}

func getPrimaryKeyType(dbType string) string {
	switch strings.ToLower(dbType) {
	case "mysql":
		return "INT AUTO_INCREMENT"
	case "postgres", "postgresql":
		return "SERIAL"
	case "mssql", "sqlserver":
		return "INT IDENTITY(1,1)"
	case "sqlite":
		return "INTEGER PRIMARY KEY AUTOINCREMENT"
	default:
		return "INT AUTO_INCREMENT"
	}
}

func sanitizeColumnName(name string) string {
	// Replace spaces and special characters with underscores
	result := strings.ReplaceAll(name, " ", "_")
	result = strings.ReplaceAll(result, "#", "num")
	result = strings.ReplaceAll(result, "%", "pct")
	result = strings.ReplaceAll(result, "$", "dollar")
	result = strings.ReplaceAll(result, "@", "at")
	result = strings.ReplaceAll(result, "&", "and")
	result = strings.ReplaceAll(result, "(", "_")
	result = strings.ReplaceAll(result, ")", "_")
	result = strings.ReplaceAll(result, "-", "_")
	result = strings.ReplaceAll(result, ".", "_")
	result = strings.ReplaceAll(result, "/", "_")
	result = strings.ReplaceAll(result, "\\", "_")

	// Remove multiple consecutive underscores
	for strings.Contains(result, "__") {
		result = strings.ReplaceAll(result, "__", "_")
	}

	// Remove leading/trailing underscores
	result = strings.Trim(result, "_")

	// Ensure it starts with a letter or underscore
	if len(result) > 0 && result[0] >= '0' && result[0] <= '9' {
		result = "col_" + result
	}

	// Convert to lowercase for consistency
	return strings.ToLower(result)
}

func containsHeader(headers []string, target string) bool {
	targetLower := strings.ToLower(target)
	for _, header := range headers {
		if strings.ToLower(sanitizeColumnName(header)) == targetLower {
			return true
		}
	}
	return false
}

// Add FieldMapping test support functions
func RegisterFieldMappingTestFunctions(rt *Runtime) {
	rt.Register("createFieldMapping", func(args ...Value) (Value, error) {
		// Create a new empty array to hold FieldMapping objects
		mappings := &ArrayValue{
			Elements: []Value{},
		}
		return mappings, nil
	})

	rt.Register("addMappingDirect", func(args ...Value) (Value, error) {
		if len(args) < 4 {
			return nil, fmt.Errorf("addMappingDirect requires 4 arguments: mappingArray, source, target, transform")
		}

		// Unwrap arguments
		for i, arg := range args {
			if tvar, ok := arg.(ScopeEntry); ok {
				args[i] = tvar.Value
			}
		}

		mappingArray, ok := args[0].(*ArrayValue)
		if !ok {
			return nil, fmt.Errorf("first argument must be a mapping array")
		}

		source := args[1].(Str)
		target := args[2].(Str)
		transformFlag := args[3].(Bool)

		// Create a new FieldMapping
		mapping := &FieldMapping{
			SourceField:  source,
			TargetColumn: target,
			Program:      Str(""), // Empty program string
			DataType:     Str("TEXT"),
			Required:     Bool(false),
			DefaultValue: Str(""),
			Transform:    Str(""),
		}

		if transformFlag {
			mapping.Transform = Str("enabled")
		}

		// Add to the array
		mappingArray.Elements = append(mappingArray.Elements, mapping)

		return Bool(true), nil
	})

	rt.Register("getMappingSource", func(args ...Value) (Value, error) {
		if len(args) < 2 {
			return nil, fmt.Errorf("getMappingSource requires 2 arguments: mappingArray, index")
		}

		// Unwrap arguments
		for i, arg := range args {
			if tvar, ok := arg.(ScopeEntry); ok {
				args[i] = tvar.Value
			}
		}

		mappingArray, ok := args[0].(*ArrayValue)
		if !ok {
			return nil, fmt.Errorf("first argument must be a mapping array")
		}

		index := int(args[1].(Number))
		if index < 0 || index >= len(mappingArray.Elements) {
			return nil, fmt.Errorf("index out of bounds")
		}

		mapping, ok := mappingArray.Elements[index].(*FieldMapping)
		if !ok {
			return nil, fmt.Errorf("element is not a FieldMapping")
		}

		return mapping.SourceField, nil
	})

	rt.Register("getMappingTarget", func(args ...Value) (Value, error) {
		if len(args) < 2 {
			return nil, fmt.Errorf("getMappingTarget requires 2 arguments: mappingArray, index")
		}

		// Unwrap arguments
		for i, arg := range args {
			if tvar, ok := arg.(ScopeEntry); ok {
				args[i] = tvar.Value
			}
		}

		mappingArray, ok := args[0].(*ArrayValue)
		if !ok {
			return nil, fmt.Errorf("first argument must be a mapping array")
		}

		index := int(args[1].(Number))
		if index < 0 || index >= len(mappingArray.Elements) {
			return nil, fmt.Errorf("index out of bounds")
		}

		mapping, ok := mappingArray.Elements[index].(*FieldMapping)
		if !ok {
			return nil, fmt.Errorf("element is not a FieldMapping")
		}

		return mapping.TargetColumn, nil
	})

	rt.Register("getMappingTransform", func(args ...Value) (Value, error) {
		if len(args) < 2 {
			return nil, fmt.Errorf("getMappingTransform requires 2 arguments: mappingArray, index")
		}

		// Unwrap arguments
		for i, arg := range args {
			if tvar, ok := arg.(ScopeEntry); ok {
				args[i] = tvar.Value
			}
		}

		mappingArray, ok := args[0].(*ArrayValue)
		if !ok {
			return nil, fmt.Errorf("first argument must be a mapping array")
		}

		index := int(args[1].(Number))
		if index < 0 || index >= len(mappingArray.Elements) {
			return nil, fmt.Errorf("index out of bounds")
		}

		mapping, ok := mappingArray.Elements[index].(*FieldMapping)
		if !ok {
			return nil, fmt.Errorf("element is not a FieldMapping")
		}

		return Bool(mapping.Transform != Str("")), nil
	})

	rt.Register("getMappingCount", func(args ...Value) (Value, error) {
		if len(args) < 1 {
			return nil, fmt.Errorf("getMappingCount requires 1 argument: mappingArray")
		}

		// Unwrap arguments
		for i, arg := range args {
			if tvar, ok := arg.(ScopeEntry); ok {
				args[i] = tvar.Value
			}
		}

		mappingArray, ok := args[0].(*ArrayValue)
		if !ok {
			return nil, fmt.Errorf("first argument must be a mapping array")
		}

		return Number(len(mappingArray.Elements)), nil
	})
}
