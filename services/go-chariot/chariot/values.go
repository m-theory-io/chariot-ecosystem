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
	case ArrayValue:
		return ValueArray
	case MapValue:
		return ValueMap
	case TableValue:
		return ValueTable
	//case ObjectValue:
	//    return ValueObject
	case HostObjectValue:
		return ValueHostObject
	case TreeNode:
		return ValueNode
	case FunctionValue:
		return ValueFunction
	case JSONNode:
		return ValueJSON
	case XMLNode:
		return ValueXML
	case nil:
		return ValueNil
	default:
		return ValueVariableExpr // Unknown types
	}
}

// New function that returns single-character type specifiers
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
	case MapValue, *MapValue:
		return "M"
	case TableValue, *TableValue:
		return "R" // Relation
	//case Object:
	//    return "O"
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

		// Regular map handling
		result := make(map[string]Value)
		for k, item := range v {
			result[k] = convertFromNativeValue(item)
		}
		return result

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
