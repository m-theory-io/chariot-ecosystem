package chariot

import (
	"errors"
	"fmt"
)

// Move these functions:
// - RegisterTable
// - SetTableKeyColumn
// - SetTableValue
// - Table cursor functions

func (rt *Runtime) RegisterTable(name string, rows []map[string]Value) {
	if rt.tables == nil {
		rt.tables = make(map[string][]map[string]Value)
		rt.cursors = make(map[string]int)
	}
	rt.tables[name] = rows
	rt.cursors[name] = 0
	if rt.currentTable == "" {
		rt.currentTable = name
	}
}

// SetTableKeyColumn sets the primary key column for a table
func (rt *Runtime) SetTableKeyColumn(tableName, columnName string) {
	rt.keyColumns[tableName] = columnName
}

func (rt *Runtime) SetTableValue(columnName string, value Value, rowContext Value) (Value, error) {
	// Implementation for Table value setting
	if rt.tables == nil {
		rt.tables = make(map[string][]map[string]Value)
	}
	if rt.currentTable == "" {
		return nil, errors.New("no current table set")
	}
	if _, exists := rt.tables[rt.currentTable]; !exists {
		return nil, fmt.Errorf("table not found: %s", rt.currentTable)
	}
	rows := rt.tables[rt.currentTable]
	if len(rows) == 0 {
		return nil, errors.New("no rows in current table")
	}
	row := rows[0] // Assuming we are working with the first row
	row[columnName] = value
	return value, nil
}
