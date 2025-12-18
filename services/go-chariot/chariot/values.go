package chariot

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"
)

const (
	ValueNumber       ValueType = iota // "N"
	ValueString                        // "S"
	ValueBoolean                       // "L" (logical, from dBASE)
	ValueNil                           // Internal bookkeeping only - not for declaration
	ValueArray                         // "A"
	ValueMap                           // "M"
	ValueTable                         // "R" (Relation - SQL result set)
	ValueObject                        // "O"
	ValueHostObject                    // "H"
	ValueNode                          // "T" (TreeNode - aligns with GetValueType)
	ValueFunction                      // "F"
	ValueJSON                          // "J"
	ValueXML                           // "X"
	ValueVariableExpr                  // "V" (untyped variables from setq)
	ValuePlan                          // "P" (Plan)
	ValueETLTransform                  // "E" (ETL Transform)
)

// Basic value types
type Number float64
type Str string
type Bool bool

type OfferVariable struct {
	Value     Value
	FormatTag string // e.g. "currency", "int", "ssn", "phone", etc.
}

func NewOfferVariable(value Value, formatTag string) *OfferVariable {
	return &OfferVariable{
		Value:     value,
		FormatTag: formatTag,
	}
}

// ETLTransformValue wraps an ETLTransform so that it can participate in declare/type-check flows.
type ETLTransformValue struct {
	Transform *ETLTransform
}

// NewETLTransformValue creates a Value wrapper for the transform pointer.
func NewETLTransformValue(transform *ETLTransform) *ETLTransformValue {
	return &ETLTransformValue{Transform: transform}
}

func (v *ETLTransformValue) String() string {
	if v == nil || v.Transform == nil {
		return "ETLTransform(nil)"
	}
	return fmt.Sprintf("ETLTransform(%s)", v.Transform.Name)
}

// GetAttribute returns one of the ETL transform fields as a Value.
func (v *ETLTransformValue) GetAttribute(name string) (Value, bool) {
	if v == nil || v.Transform == nil {
		return nil, false
	}
	attrs := v.GetAttributes()
	key := strings.ToLower(name)
	if val, ok := attrs[key]; ok {
		return val, true
	}
	val, ok := attrs[name]
	return val, ok
}

// SetAttribute allows limited mutation of transform metadata from scripts.
func (v *ETLTransformValue) SetAttribute(name string, value Value) {
	if v == nil || v.Transform == nil {
		return
	}
	switch strings.ToLower(name) {
	case "name":
		if str, ok := value.(Str); ok {
			v.Transform.Name = string(str)
		}
	case "description":
		if str, ok := value.(Str); ok {
			v.Transform.Description = string(str)
		}
	case "datatype":
		if str, ok := value.(Str); ok {
			v.Transform.DataType = string(str)
		}
	case "category":
		if str, ok := value.(Str); ok {
			v.Transform.Category = string(str)
		}
	case "program":
		if arr, ok := value.(*ArrayValue); ok {
			v.Transform.Program = arrayValueToStrings(arr)
		}
	case "examples":
		if arr, ok := value.(*ArrayValue); ok {
			v.Transform.Examples = arrayValueToStrings(arr)
		}
	}
}

// GetAttributes exposes a snapshot of transform fields for attribute-based access.
func (v *ETLTransformValue) GetAttributes() map[string]Value {
	result := make(map[string]Value)
	if v == nil || v.Transform == nil {
		return result
	}
	result["name"] = Str(v.Transform.Name)
	result["description"] = Str(v.Transform.Description)
	result["datatype"] = Str(v.Transform.DataType)
	result["category"] = Str(v.Transform.Category)
	result["program"] = stringsToArrayValue(v.Transform.Program)
	result["examples"] = stringsToArrayValue(v.Transform.Examples)
	return result
}

// ToNativeMap returns a JSON-friendly map representation of the transform.
func (v *ETLTransformValue) ToNativeMap() map[string]interface{} {
	return etlTransformToNativeMap(v.Transform)
}

// At the top of your values.go or similar file
var DBNull = &dbNullType{}

// Define the null type
type dbNullType struct{}

// Implement Value interface methods for dbNullType
func (n *dbNullType) String() string {
	return "DBNull"
}

type AttributedType interface {
	// GetAttribute retrieves an attribute by name
	GetAttribute(name string) (Value, bool)
	// SetAttribute sets an attribute by name
	SetAttribute(name string, value Value)
	// GetAttributes returns all attributes as a map
	GetAttributes() map[string]Value
}

// MetadataHolder is an interface for objects that can hold metadata.
type MetadataHolder interface {
	GetMeta(key string) (Value, bool)
	SetMeta(key string, value Value)
	HasMeta(key string) bool
}

// Value is the minimal interface for all Chariot values.
// Using an empty interface gives maximum flexibility.
type Value interface{}

// ValueType represents the type of a Value for when type information is needed
type ValueType int

type FunctionValue struct {
	Body            Node     // AST node representing the function body
	Parameters      []string // Parameter names
	SourceCode      string   // Original source (for debugging)
	FormattedSource string   // Formatted source code for display
	IsParsed        bool     // Whether the function has been parsed
	Scope           *Scope   // Captured scope (closure)
}

// Implement Value interface methods
func (f *FunctionValue) String() string {
	return fmt.Sprintf("Function(%s)", f.SourceCode)
}

// ToString returns the function as a string representation, a complete Chariot program including
// the function definition, parameters and its body.
func (f *FunctionValue) ToString() string {
	if f == nil {
		return "nil Function"
	}
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Function(%s)", f.SourceCode))
	sb.WriteString(" {\n")
	for _, param := range f.Parameters {
		sb.WriteString(fmt.Sprintf("    %s\n", param))
	}
	sb.WriteString("    ")
	sb.WriteString(f.Body.ToString())
	sb.WriteString("\n")
	sb.WriteString("}")
	return sb.String()
}

// Utility functions for working with values

// GetValueType returns the type of a value
func GetValueType(v Value) ValueType {
	switch v.(type) {
	case Number:
		return ValueNumber
	case Str:
		return ValueString
	case Bool:
		return ValueBoolean
	case *ArrayValue, ArrayValue:
		return ValueArray
	case *MapValue, MapValue:
		return ValueMap
	case *TableValue, TableValue:
		return ValueTable
	case *HostObjectValue, HostObjectValue:
		return ValueHostObject
	case *FunctionValue, FunctionValue:
		return ValueFunction
	case *JSONNode, JSONNode:
		return ValueJSON
	case *XMLNode, XMLNode:
		return ValueXML
	case *Plan, Plan:
		return ValuePlan
	case *ETLTransformValue:
		return ValueETLTransform
	case TreeNode:
		return ValueNode
	case nil:
		return ValueNil
	default:
		return ValueVariableExpr // Unknown types
	}
}

// Returns single-character type specifiers
func GetValueTypeSpec(val Value) string {
	switch val.(type) {
	case Number:
		return "N"
	case Str:
		return "S"
	case Bool:
		return "L" // Logical
	case ArrayValue, *ArrayValue:
		return "A"
	case MapValue, *MapValue, map[string]Value:
		return "M"
	case TableValue, *TableValue:
		return "R" // Relation
	case HostObjectValue, *HostObjectValue:
		return "H"
	case FunctionValue, *FunctionValue:
		return "F"
	case JSONNode, *JSONNode:
		return "J"
	case XMLNode, *XMLNode:
		return "X"
	case TreeNode, *TreeNode, TreeNodeImpl, *TreeNodeImpl:
		return "T" // TreeNode
	case Plan, *Plan:
		return "P"
	case *ETLTransformValue:
		return "E"
	case nil:
		return "V" // Variable/untyped
	default:
		return "V" // Unknown types default to Variable
	}
}

// isValidTypeSpec checks if a type specification string is valid
func isValidTypeSpec(typeSpec string) bool {
	if len(typeSpec) != 1 {
		return false
	}

	switch typeSpec[0] {
	case 'N': // Number
		return true
	case 'S': // String
		return true
	case 'L': // Logical/Boolean
		return true
	case 'A': // Array
		return true
	case 'M': // Map
		return true
	case 'R': // Relation/Table (SQL result)
		return true
	case 'O': // Object
		return true
	case 'H': // HostObject
		return true
	case 'T': // TreeNode
		return true
	case 'F': // Function
		return true
	case 'J': // JSON
		return true
	case 'X': // XML
		return true
	case 'V': // Variable/untyped
		return true
	case 'P': // Plan
		return true
	case 'E': // ETL Transform / expression alias
		return true
	default:
		return false
	}
}

// ValueToString converts a Value to its string representation
func ValueToString(v Value) string {
	if stringer, ok := v.(interface{ String() string }); ok {
		return stringer.String()
	}
	return fmt.Sprintf("%v", v)
}

// Helper function to convert various numeric Go types to float64
func convertToNativeFloat64(v interface{}) float64 {
	switch num := v.(type) {
	case float64:
		return num
	case float32:
		return float64(num)
	case int:
		return float64(num)
	case int32:
		return float64(num)
	case int64:
		return float64(num)
	case uint:
		return float64(num)
	case uint32:
		return float64(num)
	case uint64:
		return float64(num)
	default:
		return 0
	}
}

// convertValueToFloat64 converts a Value to float64, handling various types
func convertValueToFloat64(val Value) float64 {
	switch v := val.(type) {
	case Number:
		return float64(v)
	case Str:
		// Try to parse string as float
		if f, err := strconv.ParseFloat(string(v), 64); err == nil {
			return f
		}
		return 0.0 // Default if parsing fails
	case Bool:
		if v {
			return 1.0 // true as 1.0
		}
		return 0.0 // false as 0.0
	case int, int64:
		return float64(v.(int64)) // Convert int to float64
	default:
		return 0.0 // Default for unsupported types
	}
}

// convertValueToInt64 converts a Value to int64, handling various types
func convertValueToInt64(val Value) int64 {
	switch v := val.(type) {
	case Number:
		return int64(v)
	case Str:
		// Try to parse string as int
		if i, err := strconv.ParseInt(string(v), 10, 64); err == nil {
			return i
		}
		return 0 // Default if parsing fails
	case Bool:
		if v {
			return 1 // true as 1
		}
		return 0 // false as 0
	case int, int64:
		return v.(int64) // Convert int to int64
	default:
		return 0 // Default for unsupported types
	}
}

// convertValueToUInt64 converts a Value to uint64, handling various types
func convertValueToUInt64(val Value) uint64 {
	switch v := val.(type) {
	case Number:
		return uint64(v)
	case Str:
		// Try to parse string as uint
		if i, err := strconv.ParseUint(string(v), 10, 64); err == nil {
			return i
		}
		return 0 // Default if parsing fails
	case Bool:
		if v {
			return 1 // true as 1
		}
		return 0 // false as 0
	case int, int64:
		return uint64(v.(int64)) // Convert int to uint64
	default:
		return 0 // Default for unsupported types
	}
}

// convertToNativeValue converts Chariot Value types to native Go types
func convertValueToNative(val Value) interface{} {
	switch v := val.(type) {
	case *ETLTransformValue:
		return v.ToNativeMap()

	case *OfferVariable:
		return map[string]interface{}{
			"_value_type": "offer_variable",
			"value":       convertValueToNative(v.Value),
			"format_tag":  v.FormatTag,
		}

	case *FunctionValue: // ADD THIS CASE
		result := map[string]interface{}{
			"_value_type": "function",
			"source":      v.SourceCode, // The "func(profile)" part
			"parameters":  v.Parameters,
		}

		// Serialize the Body (the actual executable statements)
		if v.Body != nil {
			result["body"] = serializeNode(v.Body)
		}

		return result

	case *Plan:
		// Serialize Plan as a native map with nested serialized function values
		res := map[string]interface{}{
			"_value_type": "plan",
			"name":        v.Name,
			"params":      v.Params,
		}
		if v.Trigger != nil {
			res["trigger"] = convertValueToNative(v.Trigger)
		} else {
			res["trigger"] = nil
		}
		if v.Guard != nil {
			res["guard"] = convertValueToNative(v.Guard)
		} else {
			res["guard"] = nil
		}
		if len(v.Steps) > 0 {
			steps := make([]interface{}, 0, len(v.Steps))
			for _, s := range v.Steps {
				steps = append(steps, convertValueToNative(s))
			}
			res["steps"] = steps
		} else {
			res["steps"] = []interface{}{}
		}
		if v.Drop != nil {
			res["drop"] = convertValueToNative(v.Drop)
		} else {
			res["drop"] = nil
		}
		return res

	case Str:
		return string(v)
	case Number:
		return float64(v)
	case Bool:
		return bool(v)
	case *MapValue:
		result := make(map[string]interface{})
		for k, mv := range v.Values {
			result[k] = convertValueToNative(mv)
		}
		return result
	case *ArrayValue:
		result := make([]interface{}, v.Length())
		for i := 0; i < v.Length(); i++ {
			result[i] = convertValueToNative(v.Get(i))
		}
		return result
	case *JSONNode:
		// This is crucial - get the actual JSON value, not the node itself
		return v.GetJSONValue()
	case *MapNode:
		// Convert MapNode to native map
		return v.ToMap()
		// Add TreeNode handling for recursive conversion
	case TreeNode:
		return convertTreeNodeToNative(v)
	case nil:
		return nil
	default:
		// Handle Go structs using reflection
		rv := reflect.ValueOf(val)
		if rv.Kind() == reflect.Struct {
			return structToMap(val)
		}

		// Handle slices of structs
		if rv.Kind() == reflect.Slice && rv.Len() > 0 {
			elemType := rv.Type().Elem()
			if elemType.Kind() == reflect.Struct {
				result := make([]interface{}, rv.Len())
				for i := 0; i < rv.Len(); i++ {
					result[i] = structToMap(rv.Index(i).Interface())
				}
				return result
			}
		}

		// Don't call String() or fmt.Sprintf here - that creates the "map[...]" string
		// Instead, try to extract the underlying value
		if jsonNode, ok := val.(*JSONNode); ok {
			return jsonNode.GetJSONValue()
		}
		return val
	}
}

// New function to recursively convert TreeNode to native structure
func convertTreeNodeToNative(node TreeNode) interface{} {
	result := make(map[string]interface{})

	// Handle TreeNodeImpl specific fields
	if impl, ok := node.(*TreeNodeImpl); ok {
		result["NameStr"] = impl.NameStr
		result["ParentNode"] = nil // Avoid circular references

		// Convert Attributes recursively
		if impl.Attributes != nil {
			attrs := make(map[string]interface{})
			for key, val := range impl.Attributes {
				attrs[key] = convertValueToNative(val)
			}
			result["Attributes"] = attrs
		} else {
			result["Attributes"] = make(map[string]interface{})
		}

		// Convert Children recursively
		children := node.GetChildren()
		if len(children) > 0 {
			childArray := make([]interface{}, len(children))
			for i, child := range children {
				childArray[i] = convertTreeNodeToNative(child)
			}
			result["Children"] = childArray
		} else {
			result["Children"] = []interface{}{}
		}
	}

	return result
}

// structToMap converts a Go struct to a map[string]interface{} using reflection
func structToMap(val interface{}) map[string]interface{} {
	result := make(map[string]interface{})
	rv := reflect.ValueOf(val)
	rt := reflect.TypeOf(val)

	// Handle pointers
	if rv.Kind() == reflect.Ptr {
		rv = rv.Elem()
		rt = rt.Elem()
	}

	if rv.Kind() != reflect.Struct {
		return result
	}

	for i := 0; i < rv.NumField(); i++ {
		field := rv.Field(i)
		fieldType := rt.Field(i)

		// Skip unexported fields
		if !field.CanInterface() {
			continue
		}

		fieldName := fieldType.Name
		fieldValue := field.Interface()

		// Recursively convert nested structs
		if field.Kind() == reflect.Struct {
			result[fieldName] = structToMap(fieldValue)
		} else if field.Kind() == reflect.Slice {
			// Handle slices
			sliceVal := reflect.ValueOf(fieldValue)
			if sliceVal.Len() > 0 {
				elemType := sliceVal.Type().Elem()
				if elemType.Kind() == reflect.Struct {
					// Slice of structs
					sliceResult := make([]interface{}, sliceVal.Len())
					for j := 0; j < sliceVal.Len(); j++ {
						sliceResult[j] = structToMap(sliceVal.Index(j).Interface())
					}
					result[fieldName] = sliceResult
				} else {
					// Slice of primitives
					result[fieldName] = fieldValue
				}
			} else {
				result[fieldName] = fieldValue
			}
		} else {
			result[fieldName] = fieldValue
		}
	}

	return result
}

func convertFromNativeValue(val interface{}) Value {
	// Standalone converter in the opposite direction
	switch v := val.(type) {
	case string:
		return Str(v)
	case float64:
		return Number(v)
	case int:
		return Number(float64(v))
	case int64:
		return Number(float64(v))
	case uint:
		return Number(float64(v))
	case uint64:
		return Number(float64(v))
	case bool:
		return Bool(v)
	case nil:
		return nil
	case map[string]Value:
		mv := NewMap()
		for k, item := range v {
			mv.Set(k, item)
		}
		return mv
	case map[string]interface{}:
		// Check if this is a serialized OfferVariable
		if valueType, ok := v["_value_type"]; ok && valueType == "offer_variable" {
			value := convertFromNativeValue(v["value"])
			formatTag, _ := v["format_tag"].(string)
			return NewOfferVariable(value, formatTag)
		}
		// Check if this is a serialized FunctionValue
		if valueType, ok := v["_value_type"]; ok && valueType == "function" {
			source, _ := v["source"].(string)
			paramsInterface, _ := v["parameters"].([]interface{})

			// Convert parameters to string slice
			params := make([]string, len(paramsInterface))
			for i, p := range paramsInterface {
				params[i] = fmt.Sprintf("%v", p)
			}

			// Deserialize the body
			var body Node
			if bodyData, exists := v["body"]; exists && bodyData != nil {
				body = deserializeNode(bodyData)
			}

			// Create FunctionValue with reconstructed body
			return &FunctionValue{
				Parameters: params,
				SourceCode: source,
				Body:       body,
				IsParsed:   body != nil, // Parsed if we have a body
				Scope:      nil,         // Will be set during execution
			}
		}

		// Check if this is a serialized ETL transform
		if valueType, ok := v["_value_type"]; ok && valueType == "etl_transform" {
			transform := mapToETLTransform(v)
			return NewETLTransformValue(transform)
		}

		// Check if this is a serialized Plan
		if valueType, ok := v["_value_type"]; ok && valueType == "plan" {
			p := &Plan{}
			if name, ok := v["name"].(string); ok {
				p.Name = name
			}
			if paramsIface, ok := v["params"].([]interface{}); ok {
				params := make([]string, 0, len(paramsIface))
				for _, pi := range paramsIface {
					params = append(params, fmt.Sprintf("%v", pi))
				}
				p.Params = params
			}
			if trg, ok := v["trigger"]; ok && trg != nil {
				if fv, ok := convertFromNativeValue(trg).(*FunctionValue); ok {
					p.Trigger = fv
				}
			}
			if grd, ok := v["guard"]; ok && grd != nil {
				if fv, ok := convertFromNativeValue(grd).(*FunctionValue); ok {
					p.Guard = fv
				}
			}
			if stepsIface, ok := v["steps"].([]interface{}); ok {
				steps := make([]*FunctionValue, 0, len(stepsIface))
				for _, si := range stepsIface {
					if fv, ok := convertFromNativeValue(si).(*FunctionValue); ok {
						steps = append(steps, fv)
					}
				}
				p.Steps = steps
			}
			if drp, ok := v["drop"]; ok && drp != nil {
				if fv, ok := convertFromNativeValue(drp).(*FunctionValue); ok {
					p.Drop = fv
				}
			}
			return p
		}

		// Regular map handling
		mv := NewMap()
		for k, item := range v {
			mv.Set(k, convertFromNativeValue(item))
		}
		return mv

	case []interface{}:
		array := NewArray()
		for _, item := range v {
			array.Append(convertFromNativeValue(item))
		}
		return array
	case []string:
		// Handle string slices
		array := NewArray()
		for _, item := range v {
			array.Append(Str(item))
		}
		return array
	case [][]string:
		// Handle string slice slices (like CSV rows)
		array := NewArray()
		for _, row := range v {
			rowArray := NewArray()
			for _, cell := range row {
				rowArray.Append(Str(cell))
			}
			array.Append(rowArray)
		}
		return array
	case []FieldMapping:
		// Handle slice of FieldMapping structs specifically
		array := NewArray()
		for _, mapping := range v {
			mappingMap := structToMap(mapping)
			array.Append(convertFromNativeValue(mappingMap))
		}
		return array
	default:
		// Handle Go structs using reflection before falling back to string
		rv := reflect.ValueOf(v)
		if rv.Kind() == reflect.Struct {
			return convertFromNativeValue(structToMap(v))
		}

		// Handle slices of structs
		if rv.Kind() == reflect.Slice && rv.Len() > 0 {
			elemType := rv.Type().Elem()
			if elemType.Kind() == reflect.Struct {
				array := NewArray()
				for i := 0; i < rv.Len(); i++ {
					structMap := structToMap(rv.Index(i).Interface())
					array.Append(convertFromNativeValue(structMap))
				}
				return array
			}
		}

		return Str(fmt.Sprintf("%v", v))
	}
}

func etlTransformToNativeMap(transform *ETLTransform) map[string]interface{} {
	result := map[string]interface{}{
		"_value_type": "etl_transform",
	}
	if transform == nil {
		return result
	}
	result["name"] = transform.Name
	result["description"] = transform.Description
	result["datatype"] = transform.DataType
	result["category"] = transform.Category
	result["program"] = append([]string{}, transform.Program...)
	result["examples"] = append([]string{}, transform.Examples...)
	return result
}

func mapToETLTransform(data map[string]interface{}) *ETLTransform {
	if data == nil {
		return nil
	}
	transform := &ETLTransform{}
	if name, ok := data["name"].(string); ok {
		transform.Name = name
	}
	if desc, ok := data["description"].(string); ok {
		transform.Description = desc
	}
	if dtype, ok := data["datatype"].(string); ok {
		transform.DataType = dtype
	}
	if category, ok := data["category"].(string); ok {
		transform.Category = category
	}
	if program := interfaceToStringSlice(data["program"]); len(program) > 0 {
		transform.Program = program
	}
	if examples := interfaceToStringSlice(data["examples"]); len(examples) > 0 {
		transform.Examples = examples
	}
	return transform
}

func stringsToArrayValue(items []string) *ArrayValue {
	arr := NewArray()
	for _, item := range items {
		arr.Append(Str(item))
	}
	return arr
}

func arrayValueToStrings(arr *ArrayValue) []string {
	if arr == nil {
		return nil
	}
	result := make([]string, 0, arr.Length())
	for i := 0; i < arr.Length(); i++ {
		if str, ok := arr.Get(i).(Str); ok {
			result = append(result, string(str))
			continue
		}
		result = append(result, fmt.Sprintf("%v", arr.Get(i)))
	}
	return result
}

func interfaceToStringSlice(value interface{}) []string {
	switch items := value.(type) {
	case []string:
		return append([]string{}, items...)
	case []interface{}:
		result := make([]string, 0, len(items))
		for _, item := range items {
			result = append(result, fmt.Sprintf("%v", item))
		}
		return result
	default:
		return nil
	}
}

// Helper function to serialize Node (not ASTNode)
func serializeNode(node Node) interface{} {
	if node == nil {
		return nil
	}

	switch n := node.(type) {
	case *Block:
		return map[string]interface{}{
			"_node_type": "Block",
			"stmts":      serializeNodes(n.Stmts), // Use correct field name
		}
	case *FuncCall:
		return map[string]interface{}{
			"_node_type": "FuncCall",
			"name":       n.Name,
			"args":       serializeNodes(n.Args), // Use correct field name
		}
	case *VarRef:
		return map[string]interface{}{
			"_node_type": "VarRef",
			"name":       n.Name,
		}
	case *Literal:
		return map[string]interface{}{
			"_node_type": "Literal",
			"val":        convertValueToNative(n.Val),
		}
	case *FunctionDefNode:
		return map[string]interface{}{
			"_node_type": "FunctionDefNode",
			"parameters": n.Parameters,
			"body":       serializeNode(n.Body),
			"source":     n.Source,
			"position":   n.Position,
		}
	case *IfNode:
		return map[string]interface{}{
			"_node_type":  "IfNode",
			"condition":   serializeNode(n.Condition),
			"trueBranch":  serializeNodes(n.TrueBranch),
			"falseBranch": serializeNodes(n.FalseBranch),
			"position":    n.Position,
		}
	case *WhileNode:
		return map[string]interface{}{
			"_node_type": "WhileNode",
			"condition":  serializeNode(n.Condition),
			"body":       serializeNodes(n.Body),
			"position":   n.Position,
		}
	case *ArrayLiteralNode:
		return map[string]interface{}{
			"_node_type": "ArrayLiteralNode",
			"elements":   serializeNodes(n.Elements),
		}
	// Add other node types as needed
	default:
		// Fallback
		return map[string]interface{}{
			"_node_type": fmt.Sprintf("%T", n),
			"_fallback":  fmt.Sprintf("%v", n),
		}
	}
}

func serializeNodes(nodes []Node) []interface{} {
	result := make([]interface{}, len(nodes))
	for i, node := range nodes {
		result[i] = serializeNode(node)
	}
	return result
}

// Helper function to deserialize Node objects
func deserializeNode(data interface{}) Node {
	if data == nil {
		return nil
	}

	nodeMap, ok := data.(map[string]interface{})
	if !ok {
		return nil
	}

	nodeType, ok := nodeMap["_node_type"].(string)
	if !ok {
		return nil
	}

	switch nodeType {
	case "Block":
		if stmtsData, ok := nodeMap["stmts"].([]interface{}); ok {
			stmts := make([]Node, 0, len(stmtsData))
			for _, stmtData := range stmtsData {
				if stmt := deserializeNode(stmtData); stmt != nil {
					stmts = append(stmts, stmt)
				}
			}
			return &Block{Stmts: stmts}
		}

	case "FuncCall":
		name, _ := nodeMap["name"].(string)
		var args []Node
		if argsData, ok := nodeMap["args"].([]interface{}); ok {
			args = make([]Node, 0, len(argsData))
			for _, argData := range argsData {
				if arg := deserializeNode(argData); arg != nil {
					args = append(args, arg)
				}
			}
		}
		return &FuncCall{Name: name, Args: args}

	case "VarRef":
		if name, ok := nodeMap["name"].(string); ok {
			return &VarRef{Name: name}
		}

	case "Literal":
		if valData, exists := nodeMap["val"]; exists {
			val := convertFromNativeValue(valData)
			return &Literal{Val: val}
		}

	case "FunctionDefNode":
		params, _ := nodeMap["parameters"].([]interface{})
		source, _ := nodeMap["source"].(string)
		position, _ := nodeMap["position"].(float64)

		// Convert params to string slice
		paramStrs := make([]string, len(params))
		for i, p := range params {
			paramStrs[i] = fmt.Sprintf("%v", p)
		}

		var body Node
		if bodyData, exists := nodeMap["body"]; exists {
			body = deserializeNode(bodyData)
		}

		return &FunctionDefNode{
			Parameters: paramStrs,
			Body:       body,
			Source:     source,
			Position:   int(position),
		}

	case "IfNode":
		position, _ := nodeMap["position"].(float64)
		condition := deserializeNode(nodeMap["condition"])

		var trueBranch, falseBranch []Node
		if tbData, ok := nodeMap["trueBranch"].([]interface{}); ok {
			trueBranch = deserializeNodes(tbData)
		}
		if fbData, ok := nodeMap["falseBranch"].([]interface{}); ok {
			falseBranch = deserializeNodes(fbData)
		}

		return &IfNode{
			Condition:   condition,
			TrueBranch:  trueBranch,
			FalseBranch: falseBranch,
			Position:    int(position),
		}

	case "WhileNode":
		position, _ := nodeMap["position"].(float64)
		condition := deserializeNode(nodeMap["condition"])

		var body []Node
		if bodyData, ok := nodeMap["body"].([]interface{}); ok {
			body = deserializeNodes(bodyData)
		}

		return &WhileNode{
			Condition: condition,
			Body:      body,
			Position:  int(position),
		}

	case "ArrayLiteralNode":
		var elements []Node
		if elemData, ok := nodeMap["elements"].([]interface{}); ok {
			elements = deserializeNodes(elemData)
		}

		return &ArrayLiteralNode{Elements: elements}

		// Add other node types as needed
	}

	return nil
}

func deserializeNodes(data []interface{}) []Node {
	nodes := make([]Node, 0, len(data))
	for _, nodeData := range data {
		if node := deserializeNode(nodeData); node != nil {
			nodes = append(nodes, node)
		}
	}
	return nodes
}

// ValueToJSON converts a Chariot Value to a JSON-serializable Go value
func ValueToJSON(v Value) interface{} {
	if v == nil {
		return nil
	}

	switch val := v.(type) {
	case Number:
		return float64(val)
	case Str:
		return string(val)
	case Bool:
		return bool(val)
	case *ArrayValue:
		arr := make([]interface{}, val.Length())
		for i := 0; i < val.Length(); i++ {
			arr[i] = ValueToJSON(val.Get(i))
		}
		return arr
	case *MapValue:
		m := make(map[string]interface{})
		for k, v := range val.Values {
			m[k] = ValueToJSON(v)
		}
		return m
	case *Plan:
		return map[string]interface{}{
			"type": "plan",
			"name": val.Name,
		}
	case *FunctionValue:
		funcName := "<anonymous>"
		if val.SourceCode != "" {
			funcName = val.SourceCode
		}
		return map[string]interface{}{
			"type": "function",
			"name": funcName,
		}
	case TreeNode:
		return map[string]interface{}{
			"type": "treenode",
			"name": val.Name(),
		}
	default:
		return fmt.Sprintf("%v", v)
	}
}

// JSONToValue converts a JSON-serializable Go value to a Chariot Value
func JSONToValue(data interface{}) (Value, error) {
	if data == nil {
		return nil, nil
	}

	switch val := data.(type) {
	case float64:
		return Number(val), nil
	case int:
		return Number(float64(val)), nil
	case string:
		return Str(val), nil
	case bool:
		return Bool(val), nil
	case []interface{}:
		arr := NewArray()
		for _, item := range val {
			v, err := JSONToValue(item)
			if err != nil {
				return nil, err
			}
			arr.Append(v)
		}
		return arr, nil
	case map[string]interface{}:
		m := NewMap()
		for k, item := range val {
			v, err := JSONToValue(item)
			if err != nil {
				return nil, err
			}
			m.Set(k, v)
		}
		return m, nil
	default:
		return nil, fmt.Errorf("unsupported JSON type: %T", data)
	}
}
