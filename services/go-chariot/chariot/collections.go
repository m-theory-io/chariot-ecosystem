package chariot

// Move these functions:
// - findArray
// - findMap
// - SetListValue
// - SetArrayValue

import (
	"fmt"
	"sort"
	"strings"
)

//
// Array Types
//

// ArrayValue implements Value for arrays
type ArrayValue struct {
	Elements []Value
}

// Length returns the number of elements in the array
func (a *ArrayValue) Length() int {
	return len(a.Elements)
}

// Get retrieves the element at the specified index
func (a *ArrayValue) Get(index int) Value {
	if index < 0 || index >= len(a.Elements) {
		return nil
	}
	return a.Elements[index]
}

// Set updates the element at the specified index
func (a *ArrayValue) Set(index int, value Value) error {
	if index < 0 || index >= len(a.Elements) {
		return fmt.Errorf("index out of bounds")
	}
	a.Elements[index] = value
	return nil
}

// Append adds an element to the end of the array
func (a *ArrayValue) Append(value Value) {
	a.Elements = append(a.Elements, value)
}

// RemoveAt removes the element at the specified index
func (a *ArrayValue) RemoveAt(index int) {
	if index < 0 || index >= len(a.Elements) {
		return
	}
	a.Elements = append(a.Elements[:index], a.Elements[index+1:]...)
}

// NewArray creates a new empty array value
func NewArray() *ArrayValue {
	return &ArrayValue{
		Elements: make([]Value, 0),
	}
}

// NewArrayWithValues creates a new array with the specified initial values
func NewArrayWithValues(values []Value) *ArrayValue {
	return &ArrayValue{
		Elements: values,
	}
}

// Type returns the value type for ArrayValue
func (a *ArrayValue) Type() ValueType {
	return ValueArray
}

// String returns a string representation of the array
func (a *ArrayValue) String() string {
	if len(a.Elements) == 0 {
		return "[]"
	}

	var builder strings.Builder
	builder.WriteString("[")

	for i, v := range a.Elements {
		if i > 0 {
			builder.WriteString(", ")
		}
		builder.WriteString(fmt.Sprintf("%v", v))
	}

	builder.WriteString("]")
	return builder.String()
}

//
// Map Types
//

// MapValue represents a map/dictionary of values
type MapValue struct {
	Values map[string]Value
	meta   *MapValue // Optional metadata
}

// Type returns the value type for MapValue
func (m *MapValue) Type() ValueType {
	return ValueMap
}

// String returns a string representation of the map
func (m *MapValue) String() string {
	if len(m.Values) == 0 {
		return "{}"
	}

	items := make([]string, 0, len(m.Values))
	for k, v := range m.Values {
		items = append(items, fmt.Sprintf("%s: %v", k, v))
	}

	// Sort for consistent output
	sort.Strings(items)

	return fmt.Sprintf("{%s}", strings.Join(items, ", "))
}

// Get retrieves a value by key
func (m *MapValue) Get(key string) (Value, bool) {
	val, exists := m.Values[key]
	return val, exists
}

// Set stores a value with the given key
func (m *MapValue) Set(key string, value Value) {
	m.Values[key] = value
}

// GetAttribute retrieves a value by key
func (m *MapValue) GetAttribute(key string) (Value, bool) {
	val, exists := m.Values[key]
	return val, exists
}

// SetAttribute stores a value with the given key
func (m *MapValue) SetAttribute(key string, value Value) {
	m.Values[key] = value
}

// Keys returns a slice of all keys in the map
func (m *MapValue) Keys() []string {
	keys := make([]string, 0, len(m.Values))
	for k := range m.Values {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// Implement AttributedType interface
func (m *MapValue) GetAttributes() map[string]Value {
	return m.Values
}

// SetAttributes sets the attributes for the map
func (m *MapValue) SetAttributes(attrs map[string]Value) {
	m.meta = NewMapWithValues(attrs)
}

// NewMapWithValues creates a new map with the specified initial values
func NewMapWithValues(values map[string]Value) *MapValue {
	return &MapValue{
		Values: values,
	}
}

// NewMap creates a new empty map value
func NewMap() *MapValue {
	return &MapValue{
		Values: make(map[string]Value),
	}
}

//
// Table Types
//

// TableValue represents a collection of rows (like a database result set)
type TableValue struct {
	Rows []map[string]Value
}

// Type returns the value type for TableValue
func (t *TableValue) Type() ValueType {
	return ValueTable
}

// String returns a string representation of the table
func (t *TableValue) String() string {
	if len(t.Rows) == 0 {
		return "[]"
	}

	var builder strings.Builder
	builder.WriteString("[\n")

	// Get all column names
	colSet := make(map[string]struct{})
	for _, row := range t.Rows {
		for col := range row {
			colSet[col] = struct{}{}
		}
	}

	// Convert to ordered slice
	cols := make([]string, 0, len(colSet))
	for col := range colSet {
		cols = append(cols, col)
	}
	sort.Strings(cols)

	// Write header
	builder.WriteString("  ")
	for i, col := range cols {
		if i > 0 {
			builder.WriteString(" | ")
		}
		builder.WriteString(col)
	}
	builder.WriteString("\n")

	// Write each row
	for _, row := range t.Rows {
		builder.WriteString("  ")
		for i, col := range cols {
			if i > 0 {
				builder.WriteString(" | ")
			}
			val, exists := row[col]
			if !exists {
				builder.WriteString("null")
			} else {
				builder.WriteString(fmt.Sprintf("%v", val))
			}
		}
		builder.WriteString("\n")
	}

	builder.WriteString("]")
	return builder.String()
}

// Columns returns a slice of all column names in the table
func (t *TableValue) Columns() []string {
	colSet := make(map[string]struct{})
	for _, row := range t.Rows {
		for col := range row {
			colSet[col] = struct{}{}
		}
	}

	cols := make([]string, 0, len(colSet))
	for col := range colSet {
		cols = append(cols, col)
	}
	sort.Strings(cols)
	return cols
}

// Count returns the number of rows in the table
func (t *TableValue) Count() int {
	return len(t.Rows)
}

// NewTable creates a new empty table value
func NewTable() *TableValue {
	return &TableValue{
		Rows: make([]map[string]Value, 0),
	}
}

// AddRow adds a new row to the table
func (t *TableValue) AddRow(row map[string]Value) {
	t.Rows = append(t.Rows, row)
}

// findArray function
func (rt *Runtime) findArray(arrayName string) (*ArrayValue, error) {
	// First check if we have a document with this name
	if val, exists := rt.vars[arrayName]; exists {
		if arrayVal, ok := val.(*ArrayValue); ok {
			return arrayVal, nil
		}
	}

	// Also check global vars
	if val, exists := rt.globalVars[arrayName]; exists {
		if arrayVal, ok := val.(*ArrayValue); ok {
			return arrayVal, nil
		}
	}

	// Finally check the current scope
	if val, exists := rt.currentScope.Get(arrayName); exists {
		if arrayVal, ok := val.(*ArrayValue); ok {
			return arrayVal, nil
		}
	}

	return nil, fmt.Errorf("array '%s' not found", arrayName)
}
