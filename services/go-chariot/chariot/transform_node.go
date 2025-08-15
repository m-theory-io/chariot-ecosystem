package chariot

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

type Transform struct {
	TreeNodeImpl
	// Transform-specific data in attributes
}

type FieldMapping struct {
	SourceField  Str  // CSV column name
	TargetColumn Str  // SQL column name
	Program      Str  // Chariot transformation program (supports multi-line with backticks)
	DataType     Str  // Target SQL data type
	Required     Bool // Whether field is required
	DefaultValue Str  // Default if missing/null
	Transform    Str  // Optional: predefined transform name
}

// Support types for enhanced error tracking
type TransformResult struct {
	RowIndex      int
	SuccessFields []string
	ErrorFields   []FieldError
	TransformTime time.Time
	Duration      time.Duration
	Success       bool
}

type FieldError struct {
	FieldName     string
	Error         string
	SourceValue   string
	TransformTime time.Duration
}

func NewTransform(name string) *Transform {
	node := &Transform{}
	node.TreeNodeImpl = *NewTreeNode(name)

	// Initialize with empty mappings
	node.SetAttribute("mappings", convertFromNativeValue([]FieldMapping{}))

	// Transform metadata
	node.SetMeta("transformType", "CSV_TO_SQL")
	node.SetMeta("version", "1.0")
	node.SetMeta("created", time.Now().Unix())

	return node
}

func (t *Transform) AddMapping(mapping FieldMapping) {
	mappings := t.GetMappings()
	mappings = append(mappings, mapping)
	t.SetAttribute("mappings", convertFromNativeValue(mappings))
}

func (t *Transform) GetMappings() []FieldMapping {
	if attr, exists := t.GetAttribute("mappings"); exists {
		switch v := attr.(type) {
		case *ArrayValue:
			// If it's an ArrayValue, extract the elements
			mappings := make([]FieldMapping, v.Length())
			for i := 0; i < v.Length(); i++ {
				elem := v.Get(i)
				// Try direct conversion first
				if mapValue, ok := elem.(*MapValue); ok {
					mappings[i] = mapValueToFieldMapping(mapValue)
				} else if chariotMap, ok := elem.(map[string]Value); ok {
					mappings[i] = mapChariotValueToFieldMapping(chariotMap)
				} else if mapData, ok := convertValueToNative(elem).(map[string]interface{}); ok {
					mappings[i] = mapDataToFieldMapping(mapData)
				}
			}
			return mappings
		case []FieldMapping:
			// Direct slice of FieldMapping (ideal case)
			return v
		case []interface{}:
			// Slice of interfaces, try to convert
			mappings := make([]FieldMapping, len(v))
			for i, m := range v {
				if mapData, ok := m.(map[string]interface{}); ok {
					mappings[i] = mapDataToFieldMapping(mapData)
				}
			}
			return mappings
		default:
			// Try the original approach with convertValueToNative
			if mappingsData, ok := convertValueToNative(attr).([]interface{}); ok {
				mappings := make([]FieldMapping, len(mappingsData))
				for i, m := range mappingsData {
					if mapData, ok := m.(map[string]interface{}); ok {
						mappings[i] = mapDataToFieldMapping(mapData)
					}
				}
				return mappings
			}
		}
	}
	return []FieldMapping{}
}

// Helper function to convert MapValue to FieldMapping
func mapValueToFieldMapping(mapValue *MapValue) FieldMapping {
	mapping := FieldMapping{}

	// Extract values from MapValue using GetAttribute
	if sf, ok := mapValue.Get("SourceField"); ok {
		if str, ok := sf.(Str); ok {
			mapping.SourceField = str
		}
	}
	if tc, ok := mapValue.Get("TargetColumn"); ok {
		if str, ok := tc.(Str); ok {
			mapping.TargetColumn = str
		}
	}
	if dt, ok := mapValue.Get("DataType"); ok {
		if str, ok := dt.(Str); ok {
			mapping.DataType = str
		}
	}
	if dv, ok := mapValue.Get("DefaultValue"); ok {
		if str, ok := dv.(Str); ok {
			mapping.DefaultValue = str
		}
	}
	if tf, ok := mapValue.Get("Transform"); ok {
		if str, ok := tf.(Str); ok {
			mapping.Transform = str
		}
	}
	if req, ok := mapValue.Get("Required"); ok {
		if b, ok := req.(Bool); ok {
			mapping.Required = b
		}
	}

	// Handle Program field
	if prog, ok := mapValue.Get("Program"); ok {
		if progStr, ok := prog.(Str); ok {
			mapping.Program = progStr
		}
	}

	return mapping
}

// Helper function to convert map[string]Value to FieldMapping
func mapChariotValueToFieldMapping(chariotMap map[string]Value) FieldMapping {
	mapping := FieldMapping{}

	// Extract values from map[string]Value
	if sf, ok := chariotMap["SourceField"]; ok {
		if str, ok := sf.(Str); ok {
			mapping.SourceField = str
		}
	}
	if tc, ok := chariotMap["TargetColumn"]; ok {
		if str, ok := tc.(Str); ok {
			mapping.TargetColumn = str
		}
	}
	if dt, ok := chariotMap["DataType"]; ok {
		if str, ok := dt.(Str); ok {
			mapping.DataType = str
		}
	}
	if dv, ok := chariotMap["DefaultValue"]; ok {
		if str, ok := dv.(Str); ok {
			mapping.DefaultValue = str
		}
	}
	if tf, ok := chariotMap["Transform"]; ok {
		if str, ok := tf.(Str); ok {
			mapping.Transform = str
		}
	}
	if req, ok := chariotMap["Required"]; ok {
		if b, ok := req.(Bool); ok {
			mapping.Required = b
		}
	}

	// Handle Program field (single string, supports multi-line with backticks)
	if prog, ok := chariotMap["Program"]; ok {
		if progStr, ok := prog.(Str); ok {
			mapping.Program = progStr
		}
	}

	return mapping
}

// Helper function to convert map data to FieldMapping
func mapDataToFieldMapping(mapData map[string]interface{}) FieldMapping {
	mapping := FieldMapping{}

	// Safe string extraction
	if sf, ok := mapData["SourceField"].(string); ok {
		mapping.SourceField = Str(sf)
	}
	if tc, ok := mapData["TargetColumn"].(string); ok {
		mapping.TargetColumn = Str(tc)
	}
	if dt, ok := mapData["DataType"].(string); ok {
		mapping.DataType = Str(dt)
	}
	if dv, ok := mapData["DefaultValue"].(string); ok {
		mapping.DefaultValue = Str(dv)
	}
	if tf, ok := mapData["Transform"].(string); ok {
		mapping.Transform = Str(tf)
	}
	if req, ok := mapData["Required"].(bool); ok {
		mapping.Required = Bool(req)
	}

	// Handle Program field (single string)
	if prog, ok := mapData["Program"]; ok {
		if progStr, ok := prog.(string); ok {
			mapping.Program = Str(progStr)
		}
	}

	return mapping
}

// ApplyToRow applies the transformation to a single row of data
func (t *Transform) ApplyToRow(rt *Runtime, row map[string]string) (map[string]interface{}, error) {
	result := make(map[string]interface{})

	// Get transform registry from runtime
	registry, _ := rt.GetVariable("etlTransformRegistry")
	var etlRegistry *ETLTransformRegistry
	if reg, ok := registry.(*ETLTransformRegistry); ok {
		etlRegistry = reg
	}

	for _, mapping := range t.GetMappings() {
		// Get source value
		sourceValue, exists := row[string(mapping.SourceField)]
		if !exists || sourceValue == "" {
			if bool(mapping.Required) {
				return nil, fmt.Errorf("required field '%s' missing", string(mapping.SourceField))
			}
			sourceValue = string(mapping.DefaultValue)
		}

		var program []string

		// Use predefined transform or custom program
		if string(mapping.Transform) != "" && etlRegistry != nil {
			if etlTransform, exists := etlRegistry.Get(string(mapping.Transform)); exists {
				program = etlTransform.Program
			} else {
				return nil, fmt.Errorf("unknown transform '%s' for field '%s'", string(mapping.Transform), string(mapping.SourceField))
			}
		} else {
			// Use the program string directly
			program = []string{string(mapping.Program)}
		}

		// Join multi-line program into single string
		programCode := strings.Join(program, "\n")

		// Set up transformation variables
		variables := map[string]Value{
			"sourceValue":  Str(sourceValue),
			"sourceRow":    convertFromNativeValue(row),
			"fieldName":    mapping.SourceField,
			"targetType":   mapping.DataType,
			"targetColumn": mapping.TargetColumn,
			"required":     mapping.Required,
			"defaultValue": mapping.DefaultValue,
		}

		// Execute transformation with temporary variables
		transformedValue, err := rt.ExecuteWithVariables(programCode, variables)
		if err != nil {
			return nil, fmt.Errorf("transform error for field '%s': %v", string(mapping.SourceField), err)
		}

		// Convert to target data type
		finalValue, err := t.convertToSQLType(transformedValue, string(mapping.DataType))
		if err != nil {
			return nil, fmt.Errorf("data type conversion error for field '%s': %v", string(mapping.TargetColumn), err)
		}

		result[string(mapping.TargetColumn)] = finalValue
	}

	return result, nil
}

// Enhanced version with better error handling and metadata
func (t *Transform) ApplyToRowWithMetadata(rt *Runtime, row map[string]string, rowIndex int) (map[string]interface{}, *TransformResult, error) {
	result := make(map[string]interface{})
	transformResult := &TransformResult{
		RowIndex:      rowIndex,
		SuccessFields: []string{},
		ErrorFields:   []FieldError{},
		TransformTime: time.Now(),
	}

	for _, mapping := range t.GetMappings() {
		fieldStartTime := time.Now()

		// Get source value
		sourceValue, exists := row[string(mapping.SourceField)]
		if !exists || sourceValue == "" {
			if bool(mapping.Required) {
				fieldError := FieldError{
					FieldName:   string(mapping.SourceField),
					Error:       fmt.Sprintf("required field '%s' missing", string(mapping.SourceField)),
					SourceValue: sourceValue,
				}
				transformResult.ErrorFields = append(transformResult.ErrorFields, fieldError)
				continue
			}
			sourceValue = string(mapping.DefaultValue)
		}

		// Set up transformation variables with enhanced context
		variables := map[string]Value{
			"sourceValue":  Str(sourceValue),
			"sourceRow":    convertFromNativeValue(row),
			"fieldName":    mapping.SourceField,
			"targetType":   mapping.DataType,
			"targetColumn": mapping.TargetColumn,
			"required":     mapping.Required,
			"defaultValue": mapping.DefaultValue,
			"rowIndex":     Number(rowIndex),
			"isEmpty":      Bool(sourceValue == ""),
			"isDefault":    Bool(!exists && sourceValue == string(mapping.DefaultValue)),
		}

		// Execute transformation with temporary variables
		// Use the program string directly
		tprogram := string(mapping.Program)
		transformedValue, err := rt.ExecuteWithVariables(tprogram, variables)
		if err != nil {
			fieldError := FieldError{
				FieldName:     string(mapping.SourceField),
				Error:         fmt.Sprintf("transform error: %v", err),
				SourceValue:   sourceValue,
				TransformTime: time.Since(fieldStartTime),
			}
			transformResult.ErrorFields = append(transformResult.ErrorFields, fieldError)
			continue
		}

		// Convert to target data type
		finalValue, err := t.convertToSQLType(transformedValue, string(mapping.DataType))
		if err != nil {
			fieldError := FieldError{
				FieldName:     string(mapping.SourceField),
				Error:         fmt.Sprintf("data type conversion error: %v", err),
				SourceValue:   sourceValue,
				TransformTime: time.Since(fieldStartTime),
			}
			transformResult.ErrorFields = append(transformResult.ErrorFields, fieldError)
			continue
		}

		// Success!
		result[string(mapping.TargetColumn)] = finalValue
		transformResult.SuccessFields = append(transformResult.SuccessFields, string(mapping.SourceField))
	}

	transformResult.Duration = time.Since(transformResult.TransformTime)
	transformResult.Success = len(transformResult.ErrorFields) == 0

	return result, transformResult, nil
}

func (t *Transform) convertToSQLType(value Value, dataType string) (interface{}, error) {
	switch strings.ToUpper(dataType) {
	case "INT", "INTEGER":
		if num, ok := value.(Number); ok {
			return int(num), nil
		}
		if str, ok := value.(Str); ok {
			return strconv.Atoi(string(str))
		}

	case "DECIMAL", "FLOAT":
		if num, ok := value.(Number); ok {
			return float64(num), nil
		}
		if str, ok := value.(Str); ok {
			return strconv.ParseFloat(string(str), 64)
		}

	case "VARCHAR", "TEXT", "STRING":
		return string(value.(Str)), nil

	case "DATETIME", "TIMESTAMP":
		if str, ok := value.(Str); ok {
			return time.Parse("2006-01-02 15:04:05", string(str))
		}

	case "BOOL", "BOOLEAN":
		if b, ok := value.(Bool); ok {
			return bool(b), nil
		}
		if str, ok := value.(Str); ok {
			return strconv.ParseBool(string(str))
		}
	}

	return nil, fmt.Errorf("unsupported data type: %s", dataType)
}
