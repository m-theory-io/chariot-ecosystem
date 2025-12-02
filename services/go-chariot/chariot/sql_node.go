package chariot

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	_ "github.com/go-sql-driver/mysql"

	cfg "github.com/bhouse1273/chariot-ecosystem/services/go-chariot/configs"
	"github.com/bhouse1273/chariot-ecosystem/services/go-chariot/vault"
	"go.uber.org/zap"
	// Register database drivers as needed
	// _ "github.com/lib/pq"
	// _ "github.com/mattn/go-sqlite3"
)

// SQLNode implements TreeNode for SQL database operations
type SQLNode struct {
	TreeNodeImpl
	// SQL-specific fields
	DriverName       string            // Database driver name (mysql, postgres, sqlite3, couchbase, MSSQL)
	ConnectionString string            // Database connection string
	DB               *sql.DB           // Database connection
	LastQuery        string            // Most recently executed query
	QueryParams      []interface{}     // Parameters for the last query
	tx               *sql.Tx           // Current transaction if one is active
	mu               sync.Mutex        // Mutex for thread safety
	cachedResult     [][]interface{}   // Optionally cached results
	columnNames      []string          // Column names from last query
	columnTypes      []*sql.ColumnType // Column types from last query
	rowCount         int               // Number of rows in last result
	autoConnect      bool              // Whether to auto-reconnect on query
	connected        bool              // Whether the database is currently connected
	lastError        error             // Last error encountered
}

// NewSQLNode creates a new SQLNode with the given name
func NewSQLNode(name string) *SQLNode {
	node := &SQLNode{
		autoConnect: true,
	}
	node.TreeNodeImpl = *NewTreeNode(name)
	// Initialize with empty connection details
	node.SetMeta("user", "")
	node.SetMeta("password", "")
	node.SetMeta("database", "")
	return node
}

func (n *SQLNode) GetTypeLabel() string {
	return "SQLNode" // Return a string label for the type
}

// Connect establishes a database connection
func (n *SQLNode) Connect(driverName, sqlURL string) error {
	n.mu.Lock()
	defer n.mu.Unlock()

	// Close existing connection if any
	if n.DB != nil {
		n.DB.Close()
		n.connected = false
	}

	connStr, err := n.BuildDSN(driverName, sqlURL)
	if err != nil {
		cfg.ChariotLogger.Error("unable to make DSN", zap.String("error", err.Error()))
	}

	n.DriverName = driverName
	n.ConnectionString = connStr

	// Open connection
	db, err := sql.Open(driverName, connStr)
	if err != nil {
		n.lastError = err
		return err
	}

	// Test connection
	if err := db.Ping(); err != nil {
		db.Close()
		n.lastError = err
		return err
	}

	// Select database if specified

	n.DB = db
	n.connected = true
	return nil
}

// Close closes the database connection
func (n *SQLNode) Close() error {
	n.mu.Lock()
	defer n.mu.Unlock()

	if n.DB != nil {
		err := n.DB.Close()
		n.DB = nil
		n.connected = false
		if err != nil {
			n.lastError = err
			return err
		}
	}
	return nil
}

// ensureConnected makes sure we have a database connection
func (n *SQLNode) ensureConnected() error {
	if n.DB != nil && n.connected {
		return nil
	}

	if !n.autoConnect || n.DriverName == "" || n.ConnectionString == "" {
		return errors.New("not connected to database")
	}

	// Try to reconnect
	return n.Connect(n.DriverName, n.ConnectionString)
}

// GetRows returns the cached query results as an ArrayValue of JSONNodes
func (n *SQLNode) GetRows() (*ArrayValue, error) {
	n.mu.Lock()
	defer n.mu.Unlock()

	if n.cachedResult == nil {
		return nil, errors.New("no query results available")
	}

	// Create ArrayValue to hold the rows
	arrayResult := NewArray()

	// Convert each row to a JSONNode
	for i, row := range n.cachedResult {
		// Create a JSONNode for this row
		rowNode := NewJSONNode(fmt.Sprintf("row_%d", i))

		// Build row data as map
		rowData := make(map[string]interface{})
		for j, colName := range n.columnNames {
			if j < len(row) {
				rowData[colName] = row[j]
			}
		}

		// Set the JSON data
		rowNode.SetJSONValue(rowData)

		// Add to array
		arrayResult.Append(rowNode)
	}

	return arrayResult, nil
}

// GetRowsAsMap returns the cached query results as a slice of maps (alternative method)
func (n *SQLNode) GetRowsAsMap() ([]map[string]interface{}, error) {
	n.mu.Lock()
	defer n.mu.Unlock()

	if n.cachedResult == nil {
		return nil, errors.New("no query results available")
	}

	// Convert each row to a map
	results := make([]map[string]interface{}, len(n.cachedResult))
	for i, row := range n.cachedResult {
		rowData := make(map[string]interface{})
		for j, colName := range n.columnNames {
			if j < len(row) {
				rowData[colName] = row[j]
			}
		}
		results[i] = rowData
	}

	return results, nil
}

// GetRowsAsNative returns the raw cached results (for internal use)
func (n *SQLNode) GetRowsAsNative() ([][]interface{}, []string, error) {
	n.mu.Lock()
	defer n.mu.Unlock()

	if n.cachedResult == nil {
		return nil, nil, errors.New("no query results available")
	}

	// Return copies to prevent external modification
	resultsCopy := make([][]interface{}, len(n.cachedResult))
	for i, row := range n.cachedResult {
		rowCopy := make([]interface{}, len(row))
		copy(rowCopy, row)
		resultsCopy[i] = rowCopy
	}

	columnsCopy := make([]string, len(n.columnNames))
	copy(columnsCopy, n.columnNames)

	return resultsCopy, columnsCopy, nil
}

// QueryMeta executes a SQL query using metadata configuration
func (n *SQLNode) QueryMeta() error {
	// Extract query from metadata
	query := GetMetaString(n, "query", "")
	if query == "" {
		return fmt.Errorf("query metadata is required")
	}

	// Extract optional parameters
	var params []interface{}
	if paramsMeta, exists := n.GetMeta("params"); exists {
		if paramArray, ok := paramsMeta.([]interface{}); ok {
			params = paramArray
		} else if paramSlice, ok := paramsMeta.([]string); ok {
			// Convert string slice to interface slice
			params = make([]interface{}, len(paramSlice))
			for i, p := range paramSlice {
				params[i] = p
			}
		}
	}

	// Execute the query
	_, err := n.QuerySQL(query, params...)
	if err != nil {
		return fmt.Errorf("failed to execute query from metadata: %v", err)
	}

	return nil
}

// ConnectMeta establishes a database connection using metadata
func (n *SQLNode) ConnectMeta() error {
	driver := GetMetaString(n, "driver", "")
	if driver == "" {
		return fmt.Errorf("driver metadata is required")
	}

	connectionString := GetMetaString(n, "connectionString", "")
	if connectionString == "" {
		return fmt.Errorf("connectionString metadata is required")
	}

	// Set optional configuration from metadata
	if timeoutMeta, exists := n.GetMeta("queryTimeout"); exists {
		if timeout, ok := timeoutMeta.(int); ok {
			defer n.SetQueryTimeout(timeout)
		}
	}

	if autoConnectMeta, exists := n.GetMeta("autoConnect"); exists {
		if autoConnect, ok := autoConnectMeta.(bool); ok {
			n.autoConnect = autoConnect
		}
	}

	// Connect using metadata values
	return n.Connect(driver, connectionString)
}

// ExecuteMeta executes a SQL statement using metadata configuration
func (n *SQLNode) ExecuteMeta() (int64, error) {
	// Extract statement from metadata
	stmt := GetMetaString(n, "statement", "")
	if stmt == "" {
		return 0, fmt.Errorf("statement metadata is required")
	}

	// Extract optional parameters
	var params []interface{}
	if paramsMeta, exists := n.GetMeta("params"); exists {
		if paramArray, ok := paramsMeta.([]interface{}); ok {
			params = paramArray
		}
	}

	// Execute the statement
	return n.Execute(stmt, params...)
}

// Implement the TreeNode interface method
func (n *SQLNode) QueryTree(filter func(TreeNode) bool) []TreeNode {
	result := make([]TreeNode, 0)

	// Apply filter to all children
	for _, child := range n.GetChildren() {
		if filter(child) {
			result = append(result, child)
		}
	}

	return result
}

// Query executes a SQL query and returns results
func (n *SQLNode) QuerySQL(query string, args ...interface{}) (*ArrayValue, error) {
	n.mu.Lock()
	defer n.mu.Unlock()

	// Make sure we're connected
	if err := n.ensureConnected(); err != nil {
		n.lastError = err
		return nil, err
	}

	// Store query info
	n.LastQuery = query
	n.QueryParams = args

	// Reset previous results
	n.cachedResult = nil
	n.columnNames = nil
	n.columnTypes = nil
	n.Children = nil

	// Execute the query
	rows, err := n.DB.Query(query, args...)
	if err != nil {
		n.lastError = err
		return nil, err
	}
	defer rows.Close()

	// Get column information
	n.columnNames, err = rows.Columns()
	if err != nil {
		n.lastError = err
		return nil, err
	}

	n.columnTypes, err = rows.ColumnTypes()
	if err != nil {
		n.lastError = err
		return nil, err
	}

	// Process rows
	n.cachedResult = [][]interface{}{}
	rowIndex := 0
	results := []map[string]interface{}{}

	for rows.Next() {
		// Create slice for row values
		rowValues := make([]interface{}, len(n.columnNames))
		rowValuePtrs := make([]interface{}, len(n.columnNames))

		// Create pointers to scan into
		for i := range rowValues {
			rowValuePtrs[i] = &rowValues[i]
		}

		// Scan row data
		if err := rows.Scan(rowValuePtrs...); err != nil {
			n.lastError = err
			return nil, err
		}

		// Create a simple map for this row
		row := make(map[string]interface{})
		for i, col := range n.columnNames {
			val := rowValuePtrs[i]

			actualValue := *val.(*interface{})

			// Convert byte slices to strings (common MySQL issue)
			if b, ok := actualValue.([]byte); ok {
				row[col] = string(b)
			} else {
				row[col] = actualValue
			}
		}

		// Add to cached results and as child node
		n.cachedResult = append(n.cachedResult, rowValues)
		results = append(results, row) // Add to result collection
		rowIndex++
	}

	// Check for errors after iteration
	if err := rows.Err(); err != nil {
		n.lastError = err
		return nil, err
	}

	n.rowCount = rowIndex

	// Return as a simple Chariot array of maps
	chariotResults := make([]Value, len(results))
	for i, row := range results {
		chariotRow := make(map[string]Value)
		for key, val := range row {
			switch v := val.(type) {
			case string:
				chariotRow[key] = Str(v)
			case int64:
				chariotRow[key] = Number(v)
			case float64:
				chariotRow[key] = Number(v)
			case bool:
				chariotRow[key] = Bool(v)
			case nil:
				chariotRow[key] = DBNull
			default:
				chariotRow[key] = Str(fmt.Sprintf("%v", v))
			}
		}
		chariotResults[i] = chariotRow
	}
	return NewArrayWithValues(chariotResults), nil
}

// QueryStream executes a SQL query and processes results incrementally
func (n *SQLNode) QuerySQLStream(query string, callback func(int, map[string]interface{}) bool, args ...interface{}) error {
	n.mu.Lock()

	// Make sure we're connected
	if err := n.ensureConnected(); err != nil {
		n.mu.Unlock()
		n.lastError = err
		return err
	}

	// Store query info
	n.LastQuery = query
	n.QueryParams = args

	// Reset previous results
	n.cachedResult = nil
	n.columnNames = nil
	n.Children = nil

	// Execute the query
	rows, err := n.DB.Query(query, args...)
	if err != nil {
		n.mu.Unlock()
		n.lastError = err
		return err
	}

	// Get column information
	n.columnNames, err = rows.Columns()
	if err != nil {
		rows.Close()
		n.mu.Unlock()
		n.lastError = err
		return err
	}

	// Release mutex for processing
	n.mu.Unlock()

	defer rows.Close()
	rowIndex := 0

	for rows.Next() {
		// Create slice for row values
		rowValues := make([]interface{}, len(n.columnNames))
		rowValuePtrs := make([]interface{}, len(n.columnNames))

		// Create pointers to scan into
		for i := range rowValues {
			rowValuePtrs[i] = &rowValues[i]
		}

		// Scan row data
		if err := rows.Scan(rowValuePtrs...); err != nil {
			n.lastError = err
			return err
		}

		// Convert to map for callback
		rowMap := make(map[string]interface{})
		for i, colName := range n.columnNames {
			rowMap[colName] = rowValues[i]
		}

		// Call the callback
		if !callback(rowIndex, rowMap) {
			break
		}

		rowIndex++
	}

	// Check for errors after iteration
	if err := rows.Err(); err != nil {
		n.lastError = err
		return err
	}

	n.rowCount = rowIndex
	return nil
}

// Execute runs a SQL statement and returns affected rows
func (n *SQLNode) Execute(stmt string, args ...interface{}) (int64, error) {
	n.mu.Lock()
	defer n.mu.Unlock()

	// Make sure we're connected
	if err := n.ensureConnected(); err != nil {
		n.lastError = err
		return 0, err
	}

	// Store query info
	n.LastQuery = stmt
	n.QueryParams = args

	// Execute the statement
	var result sql.Result
	var err error

	if n.tx != nil {
		// In transaction
		result, err = n.tx.Exec(stmt, args...)
	} else {
		// No transaction
		result, err = n.DB.Exec(stmt, args...)
	}

	if err != nil {
		n.lastError = err
		return 0, err
	}

	// Check if this is a DDL statement (Data Definition Language)
	queryUpper := strings.ToUpper(strings.TrimSpace(stmt))
	isDDL := strings.HasPrefix(queryUpper, "CREATE") ||
		strings.HasPrefix(queryUpper, "DROP") ||
		strings.HasPrefix(queryUpper, "ALTER") ||
		strings.HasPrefix(queryUpper, "TRUNCATE")

	if isDDL {
		// For DDL statements, if no error occurred, consider it successful
		return int64(1), nil
	}

	// For DML statements (INSERT, UPDATE, DELETE), return actual rows affected
	affected, err := result.RowsAffected()
	if err != nil {
		n.lastError = err
		return 0, err
	}

	return affected, nil
}

// Begin starts a new transaction
func (n *SQLNode) Begin() error {
	n.mu.Lock()
	defer n.mu.Unlock()

	// Make sure we're connected
	if err := n.ensureConnected(); err != nil {
		n.lastError = err
		return err
	}

	// Check if transaction already exists
	if n.tx != nil {
		return errors.New("transaction already in progress")
	}

	// Start transaction
	tx, err := n.DB.Begin()
	if err != nil {
		n.lastError = err
		return err
	}

	n.tx = tx
	return nil
}

// Commit commits the current transaction
func (n *SQLNode) Commit() error {
	n.mu.Lock()
	defer n.mu.Unlock()

	if n.tx == nil {
		return errors.New("no transaction in progress")
	}

	err := n.tx.Commit()
	n.tx = nil

	if err != nil {
		n.lastError = err
		return err
	}

	return nil
}

// Rollback aborts the current transaction
func (n *SQLNode) Rollback() error {
	n.mu.Lock()
	defer n.mu.Unlock()

	if n.tx == nil {
		return errors.New("no transaction in progress")
	}

	err := n.tx.Rollback()
	n.tx = nil

	if err != nil {
		n.lastError = err
		return err
	}

	return nil
}

// GetColumnNames returns column names from the last query
func (n *SQLNode) GetColumnNames() []string {
	n.mu.Lock()
	defer n.mu.Unlock()
	return n.columnNames
}

// GetRowCount returns the number of rows in the last result
func (n *SQLNode) GetRowCount() int {
	n.mu.Lock()
	defer n.mu.Unlock()
	return n.rowCount
}

// GetCell returns a specific cell value from the cached results
func (n *SQLNode) GetCell(row int, col interface{}) (interface{}, error) {
	n.mu.Lock()
	defer n.mu.Unlock()

	if n.cachedResult == nil {
		return nil, errors.New("no query results available")
	}

	if row < 0 || row >= len(n.cachedResult) {
		return nil, fmt.Errorf("row index %d out of range", row)
	}

	var colIdx int
	switch v := col.(type) {
	case int:
		colIdx = v
		if colIdx < 0 || colIdx >= len(n.columnNames) {
			return nil, fmt.Errorf("column index %d out of range", colIdx)
		}
	case string:
		found := false
		for i, name := range n.columnNames {
			if name == v {
				colIdx = i
				found = true
				break
			}
		}
		if !found {
			return nil, fmt.Errorf("column '%s' not found", v)
		}
	default:
		return nil, fmt.Errorf("column must be string or int, got %T", col)
	}

	return n.cachedResult[row][colIdx], nil
}

// GetRow returns a map with column values for a specific row
func (n *SQLNode) GetRow(row int) (map[string]interface{}, error) {
	n.mu.Lock()
	defer n.mu.Unlock()

	if n.cachedResult == nil {
		return nil, errors.New("no query results available")
	}

	if row < 0 || row >= len(n.cachedResult) {
		return nil, fmt.Errorf("row index %d out of range", row)
	}

	result := make(map[string]interface{})
	for i, name := range n.columnNames {
		result[name] = n.cachedResult[row][i]
	}

	return result, nil
}

// ListTables retrieves all table names in the database
func (n *SQLNode) ListTables() (*ArrayValue, error) {
	var query string

	// Different query based on database type
	switch n.DriverName {
	case "mysql":
		query = "SHOW TABLES"
	case "postgres":
		query = "SELECT tablename FROM pg_catalog.pg_tables WHERE schemaname != 'pg_catalog' AND schemaname != 'information_schema'"
	case "sqlite3":
		query = "SELECT name FROM sqlite_master WHERE type='table'"
	default:
		return nil, fmt.Errorf("unsupported database type: %s", n.DriverName)
	}

	return n.QuerySQL(query, nil)
}

// DescribeTable retrieves table schema
func (n *SQLNode) DescribeTable(tableName string) (*ArrayValue, error) {
	var query string

	// Different query based on database type
	switch n.DriverName {
	case "mysql":
		query = fmt.Sprintf("DESCRIBE `%s`", tableName)
	case "postgres":
		query = fmt.Sprintf("SELECT column_name, data_type, character_maximum_length, is_nullable, column_default FROM information_schema.columns WHERE table_name = '%s'", tableName)
	case "sqlite3":
		query = fmt.Sprintf("PRAGMA table_info('%s')", tableName)
	default:
		return nil, fmt.Errorf("unsupported database type: %s", n.DriverName)
	}

	return n.QuerySQL(query)
}

// SetQueryTimeout sets a timeout for queries
func (n *SQLNode) SetQueryTimeout(seconds int) error {
	n.mu.Lock()
	defer n.mu.Unlock()

	if n.DB == nil {
		return errors.New("not connected to database")
	}

	n.DB.SetConnMaxLifetime(time.Duration(seconds) * time.Second)
	return nil
}

// ToCSVNode converts SQL results to a CSVNode
func (n *SQLNode) ToCSVNode() (*CSVNode, error) {
	n.mu.Lock()
	defer n.mu.Unlock()

	if n.cachedResult == nil || len(n.columnNames) == 0 {
		return nil, errors.New("no query results available")
	}

	// Create CSV node
	csvNode := NewCSVNode(n.Name() + "_csv")

	// Set headers using the new metadata pattern
	csvNode.SetAttribute("headers", convertFromNativeValue(n.columnNames))
	csvNode.SetMeta("columnCount", len(n.columnNames))
	csvNode.SetMeta("hasHeaders", true)

	// Create column index mapping as metadata
	columnIndex := make(map[string]int)
	for i, name := range n.columnNames {
		columnIndex[name] = i
	}
	csvNode.SetMeta("columnIndex", columnIndex)

	// Convert rows to string format
	csvRows := make([][]string, len(n.cachedResult))
	for i, row := range n.cachedResult {
		csvRow := make([]string, len(row))
		for j, val := range row {
			// Convert to string based on the value type
			if val == nil {
				csvRow[j] = ""
			} else {
				switch v := val.(type) {
				case string:
					csvRow[j] = v
				case []byte:
					csvRow[j] = string(v)
				case time.Time:
					csvRow[j] = v.Format("2006-01-02 15:04:05")
				default:
					csvRow[j] = fmt.Sprintf("%v", v)
				}
			}
		}
		csvRows[i] = csvRow

		// Create child nodes for each row (following TreeNode pattern)
		rowNode := NewJSONNode(fmt.Sprintf("row_%d", i))
		rowData := make(map[string]interface{})
		for j, colName := range n.columnNames {
			if j < len(csvRow) {
				rowData[colName] = csvRow[j]
			}
		}
		rowNode.SetJSONValue(rowData)
		csvNode.AddChild(rowNode)
	}

	// Store all rows as attribute using the new pattern
	csvNode.SetAttribute("rows", convertFromNativeValue(csvRows))
	csvNode.SetMeta("rowCount", len(csvRows))

	// Set completion metadata
	csvNode.SetMeta("convertedFrom", "sql")
	csvNode.SetMeta("convertedAt", time.Now().Unix())

	return csvNode, nil
}

// Insert inserts a row into the specified table
func (n *SQLNode) Insert(data map[string]interface{}) error {
	n.mu.Lock()
	defer n.mu.Unlock()

	// Make sure we're connected
	if err := n.ensureConnected(); err != nil {
		n.lastError = err
		return err
	}

	// Get table name from metadata
	tableName := GetMetaString(n, "tableName", "")
	if tableName == "" {
		return fmt.Errorf("tableName metadata is required for insert")
	}

	// Build column list and placeholders
	columns := make([]string, 0, len(data))
	placeholders := make([]string, 0, len(data))
	values := make([]interface{}, 0, len(data))

	for col, val := range data {
		columns = append(columns, col)
		placeholders = append(placeholders, "?")
		values = append(values, val)
	}

	// Build INSERT statement
	query := fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s)",
		tableName,
		strings.Join(columns, ", "),
		strings.Join(placeholders, ", "))

	// Execute the insert
	var err error
	if n.tx != nil {
		// In transaction
		_, err = n.tx.Exec(query, values...)
	} else {
		// No transaction
		_, err = n.DB.Exec(query, values...)
	}

	if err != nil {
		n.lastError = err
		return fmt.Errorf("insert failed: %v", err)
	}

	// Store query info for debugging
	n.LastQuery = query
	n.QueryParams = values

	return nil
}

// InsertBatch inserts multiple rows in a single transaction for better performance
func (n *SQLNode) InsertBatch(data []map[string]interface{}) error {
	if len(data) == 0 {
		return nil
	}

	n.mu.Lock()
	defer n.mu.Unlock()

	// Make sure we're connected
	if err := n.ensureConnected(); err != nil {
		n.lastError = err
		return err
	}

	// Get table name from metadata
	tableName := GetMetaString(n, "tableName", "")
	if tableName == "" {
		return fmt.Errorf("tableName metadata is required for insert")
	}

	// Use first row to determine column structure
	firstRow := data[0]
	columns := make([]string, 0, len(firstRow))
	for col := range firstRow {
		columns = append(columns, col)
	}

	// Build INSERT statement template
	placeholders := make([]string, len(columns))
	for i := range placeholders {
		placeholders[i] = "?"
	}

	query := fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s)",
		tableName,
		strings.Join(columns, ", "),
		strings.Join(placeholders, ", "))

	// Prepare statement
	var stmt *sql.Stmt
	var err error

	if n.tx != nil {
		stmt, err = n.tx.Prepare(query)
	} else {
		stmt, err = n.DB.Prepare(query)
	}

	if err != nil {
		n.lastError = err
		return fmt.Errorf("failed to prepare insert statement: %v", err)
	}
	defer stmt.Close()

	// Execute for each row
	for i, row := range data {
		values := make([]interface{}, len(columns))
		for j, col := range columns {
			values[j] = row[col]
		}

		_, err = stmt.Exec(values...)
		if err != nil {
			n.lastError = err
			return fmt.Errorf("insert failed for row %d: %v", i, err)
		}
	}

	// Store query info for debugging
	n.LastQuery = query
	n.QueryParams = nil // Too many to store effectively

	return nil
}

// Upsert performs an INSERT or UPDATE depending on whether the record exists
func (n *SQLNode) Upsert(data map[string]interface{}, keyColumns []string) error {
	n.mu.Lock()
	defer n.mu.Unlock()

	// Make sure we're connected
	if err := n.ensureConnected(); err != nil {
		n.lastError = err
		return err
	}

	// Get table name from metadata
	tableName := GetMetaString(n, "tableName", "")
	if tableName == "" {
		return fmt.Errorf("tableName metadata is required for upsert")
	}

	// Build the upsert query based on database type
	var query string
	var values []interface{}

	switch n.DriverName {
	case "mysql":
		query, values = n.buildMySQLUpsert(tableName, data)
	case "postgres":
		query, values = n.buildPostgreSQLUpsert(tableName, data, keyColumns)
	case "sqlite3":
		query, values = n.buildSQLiteUpsert(tableName, data, keyColumns)
	default:
		// Fallback to separate SELECT then INSERT/UPDATE
		return n.upsertFallback(tableName, data, keyColumns)
	}

	// Execute the upsert
	var err error
	if n.tx != nil {
		_, err = n.tx.Exec(query, values...)
	} else {
		_, err = n.DB.Exec(query, values...)
	}

	if err != nil {
		n.lastError = err
		return fmt.Errorf("upsert failed: %v", err)
	}

	// Store query info for debugging
	n.LastQuery = query
	n.QueryParams = values

	return nil
}

// Helper functions for building database-specific upsert queries
func (n *SQLNode) buildMySQLUpsert(tableName string, data map[string]interface{}) (string, []interface{}) {
	columns := make([]string, 0, len(data))
	placeholders := make([]string, 0, len(data))
	updates := make([]string, 0, len(data))
	values := make([]interface{}, 0, len(data)*2)

	for col, val := range data {
		columns = append(columns, col)
		placeholders = append(placeholders, "?")
		updates = append(updates, fmt.Sprintf("%s = VALUES(%s)", col, col))
		values = append(values, val)
	}

	query := fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s) ON DUPLICATE KEY UPDATE %s",
		tableName,
		strings.Join(columns, ", "),
		strings.Join(placeholders, ", "),
		strings.Join(updates, ", "))

	return query, values
}

func (n *SQLNode) buildPostgreSQLUpsert(tableName string, data map[string]interface{}, keyColumns []string) (string, []interface{}) {
	columns := make([]string, 0, len(data))
	placeholders := make([]string, 0, len(data))
	updates := make([]string, 0, len(data))
	values := make([]interface{}, 0, len(data))

	for col, val := range data {
		columns = append(columns, col)
		placeholders = append(placeholders, "?")
		if !contains(keyColumns, col) {
			updates = append(updates, fmt.Sprintf("%s = EXCLUDED.%s", col, col))
		}
		values = append(values, val)
	}

	query := fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s) ON CONFLICT (%s) DO UPDATE SET %s",
		tableName,
		strings.Join(columns, ", "),
		strings.Join(placeholders, ", "),
		strings.Join(keyColumns, ", "),
		strings.Join(updates, ", "))

	return query, values
}

func (n *SQLNode) buildSQLiteUpsert(tableName string, data map[string]interface{}, keyColumns []string) (string, []interface{}) {
	columns := make([]string, 0, len(data))
	placeholders := make([]string, 0, len(data))
	updates := make([]string, 0, len(data))
	values := make([]interface{}, 0, len(data))

	for col, val := range data {
		columns = append(columns, col)
		placeholders = append(placeholders, "?")
		if !contains(keyColumns, col) {
			updates = append(updates, fmt.Sprintf("%s = EXCLUDED.%s", col, col))
		}
		values = append(values, val)
	}

	query := fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s) ON CONFLICT (%s) DO UPDATE SET %s",
		tableName,
		strings.Join(columns, ", "),
		strings.Join(placeholders, ", "),
		strings.Join(keyColumns, ", "),
		strings.Join(updates, ", "))

	return query, values
}

// Fallback upsert using separate SELECT then INSERT/UPDATE
func (n *SQLNode) upsertFallback(tableName string, data map[string]interface{}, keyColumns []string) error {
	// Build WHERE clause for key columns
	whereConditions := make([]string, len(keyColumns))
	whereValues := make([]interface{}, len(keyColumns))

	for i, col := range keyColumns {
		whereConditions[i] = fmt.Sprintf("%s = ?", col)
		whereValues[i] = data[col]
	}

	// Check if record exists
	selectQuery := fmt.Sprintf("SELECT COUNT(*) FROM %s WHERE %s",
		tableName, strings.Join(whereConditions, " AND "))

	var count int
	var err error

	if n.tx != nil {
		err = n.tx.QueryRow(selectQuery, whereValues...).Scan(&count)
	} else {
		err = n.DB.QueryRow(selectQuery, whereValues...).Scan(&count)
	}

	if err != nil {
		return fmt.Errorf("failed to check record existence: %v", err)
	}

	if count > 0 {
		// Record exists, do UPDATE
		return n.updateRecord(tableName, data, keyColumns)
	} else {
		// Record doesn't exist, do INSERT
		return n.Insert(data)
	}
}

// Helper function to update a record
func (n *SQLNode) updateRecord(tableName string, data map[string]interface{}, keyColumns []string) error {
	setClauses := make([]string, 0, len(data))
	whereConditions := make([]string, len(keyColumns))
	values := make([]interface{}, 0, len(data))
	whereValues := make([]interface{}, len(keyColumns))

	// Build SET clauses (exclude key columns)
	for col, val := range data {
		if !contains(keyColumns, col) {
			setClauses = append(setClauses, fmt.Sprintf("%s = ?", col))
			values = append(values, val)
		}
	}

	// Build WHERE clauses
	for i, col := range keyColumns {
		whereConditions[i] = fmt.Sprintf("%s = ?", col)
		whereValues[i] = data[col]
	}

	// Combine values
	allValues := append(values, whereValues...)

	query := fmt.Sprintf("UPDATE %s SET %s WHERE %s",
		tableName,
		strings.Join(setClauses, ", "),
		strings.Join(whereConditions, " AND "))

	var err error
	if n.tx != nil {
		_, err = n.tx.Exec(query, allValues...)
	} else {
		_, err = n.DB.Exec(query, allValues...)
	}

	return err
}

// GetLastError returns the last error that occurred
func (n *SQLNode) GetLastError() error {
	return n.lastError
}

// SQLRowNode represents a row in a SQL result set
type SQLRowNode struct {
	TreeNodeImpl
	rowIndex int
	values   map[string]interface{}
	//lint:ignore U1000 This field is used in the code
	columnNames []string
}

// NewSQLRowNode creates a new SQL row node
func NewSQLRowNode(name string, rowIndex int) *SQLRowNode {
	node := &SQLRowNode{
		rowIndex: rowIndex,
		values:   make(map[string]interface{}),
	}
	node.TreeNodeImpl = *NewTreeNode(name)
	return node
}

// GetValue returns a column value by name
func (n *SQLRowNode) GetValue(colName string) (interface{}, bool) {
	val, ok := n.values[colName]
	return val, ok
}

// SetValue sets a column value
func (n *SQLRowNode) SetValue(colName string, value interface{}) {
	n.values[colName] = value
}

// ToMap returns all column values as a map
func (n *SQLRowNode) ToMap() map[string]interface{} {
	// Return a copy to prevent modifications
	result := make(map[string]interface{}, len(n.values))
	for k, v := range n.values {
		result[k] = v
	}
	return result
}

// Helper function to check if a slice contains a string
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

// Helper to build a DSN (Data Source Name) for supported drivers.
// If credentials are missing, attempts to fetch from the configured secret provider.
func (n *SQLNode) BuildDSN(driverName, databaseURL string) (string, error) {
	var user, password, dbname Value

	// Validate args
	if driverName == "" {
		return "", errors.New("driver name is required")
	}
	if databaseURL == "" {
		return "", errors.New("database URL is required")
	}

	// Get credentials from metadata
	if tuser, ok := n.GetMeta("user"); ok {
		user = tuser
	}
	if tdbname, ok := n.GetMeta("database"); ok {
		dbname = tdbname
	}
	if tpassword, ok := n.GetMeta("password"); ok {
		password = tpassword
	}

	// If credentials are missing, try to fetch from the configured secret provider
	if user == "" || password == "" || dbname == "" {
		ctx := context.Background()
		secret, err := vault.GetOrgSecret(ctx, cfg.ChariotKey)
		if err != nil {
			return "", fmt.Errorf("failed to fetch secret from Azure Vault: %w", err)
		}
		user = secret.SQLUser
		password = secret.SQLPassword
		dbname = secret.SQLDatabase
		n.SetMeta("user", user)
		n.SetMeta("password", password)
		n.SetMeta("database", dbname)
	}

	switch driverName {
	case "mysql":
		// Example: user:password@tcp(host:port)/dbname
		return fmt.Sprintf("%s:%s@tcp(%s)/%s", user, password, databaseURL, dbname), nil
	case "postgres":
		// Example: postgres://user:password@host:port/dbname
		return fmt.Sprintf("postgres://%s:%s@%s/%s", user, password, databaseURL, dbname), nil
	case "mssql":
		// Example: sqlserver://user:password@host:port?database=dbname
		return fmt.Sprintf("sqlserver://%s:%s@%s?database=%s", user, password, databaseURL, dbname), nil
	default:
		return "", fmt.Errorf("unsupported driver: %s", driverName)
	}
}
