package chariot

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	cfg "github.com/bhouse1273/go-chariot/configs"
	"github.com/bhouse1273/go-chariot/vault"
	"go.uber.org/zap"
)

// RegisterSQLFunctions registers all SQL-related functions
func RegisterSQLFunctions(rt *Runtime) {
	// SQL connection management
	rt.Register("sqlConnect", func(args ...Value) (Value, error) {
		if len(args) < 1 || len(args) > 5 {
			return nil, fmt.Errorf("sqlConnect requires 1-5 arguments: nodeName, driver, connectionString, [options...]")
		}

		// Unwrap arguments
		for i, arg := range args {
			if tvar, ok := arg.(ScopeEntry); ok {
				args[i] = tvar.Value
			}
		}

		nodeName, ok := args[0].(Str)
		if !ok {
			return nil, fmt.Errorf("node name must be a string")
		}
		// Create SQL node
		sqlNode := NewSQLNode(string(nodeName))

		driver := "mysql"
		sqlUser := cfg.ChariotConfig.SQLUser
		sqlPassword := cfg.ChariotConfig.SQLPassword
		_ = sqlUser
		_ = sqlPassword

		sqlURL, ok := args[1].(Str)
		if !ok {
			return nil, fmt.Errorf("connection URL string must be a string")
		}

		// Check how complete arguments are
		if sqlURL == "" {
			ctx := context.Background()
			// get connection parameters from vault
			orgSecret, err := vault.GetOrgSecret(ctx, cfg.ChariotKey)
			if err != nil {
				return nil, fmt.Errorf("failed to get org secret: %v", err)
			}
			sqlURL = Str(orgSecret.SQLHost)
			if orgSecret.SQLPort > 0 {
				sqlURL = Str(fmt.Sprintf("%s:%d", sqlURL, orgSecret.SQLPort))
			}
			driver = orgSecret.SQLDriver
			if orgSecret.SQLUser != "" {
				sqlUser = orgSecret.SQLUser
			}
			if orgSecret.SQLPassword != "" {
				sqlPassword = orgSecret.SQLPassword
			}
			sqlNode.SetMeta("database", orgSecret.SQLDatabase)
			sqlNode.SetMeta("user", orgSecret.SQLUser)
			sqlNode.SetMeta("password", orgSecret.SQLPassword)
		}

		// support for user putting a port in the connection URL
		port := int64(cfg.ChariotConfig.SQLPort) // Default to configured port
		parts := strings.Split(string(sqlURL), ":")

		if len(parts) == 2 {
			var err error
			port, err = strconv.ParseInt(parts[1], 10, 64)
			if err != nil {
				return nil, fmt.Errorf("invalid port in connection URL: %v", err)
			}
			if port <= 0 {
				return nil, fmt.Errorf("port must be a positive integer")
			}
			// Compare with configured port
			if cfg.ChariotConfig.SQLPort > 0 {
				if int64(cfg.ChariotConfig.SQLPort) != port {
					// warning: port in connection URL does not match configured port
					cfg.ChariotLogger.Warn("port mismatch in connection URL", zap.Int("configured_port", cfg.ChariotConfig.SQLPort), zap.Int64("url_port", port))
				}
				port = int64(cfg.ChariotConfig.SQLPort)
				sqlURL = Str(fmt.Sprintf("%s:%d", parts[0], port))
			}

		} else if cfg.ChariotConfig.SQLPort == 0 {
			return nil, fmt.Errorf("SQLPort must be configured")
		} else {
			// Inject configured port
			sqlURL = Str(fmt.Sprintf("%s:%d", string(sqlURL), port))
		}

		// Connect to database
		if err := sqlNode.Connect(string(driver), string(sqlURL)); err != nil {
			return nil, fmt.Errorf("failed to connect: %v", err)
		}

		// Store in runtime objects
		rt.objects[string(nodeName)] = sqlNode

		return Str(fmt.Sprintf("Connected to %s database", driver)), nil
	})

	// SQL query execution
	rt.Register("sqlQuery", func(args ...Value) (Value, error) {
		if len(args) < 2 {
			return nil, fmt.Errorf("sqlQuery requires at least 2 arguments: nodeName, query, [params...]")
		}

		// Unwrap arguments
		for i, arg := range args {
			if tvar, ok := arg.(ScopeEntry); ok {
				args[i] = tvar.Value
			}
		}

		nodeName, ok := args[0].(Str)
		if !ok {
			return nil, fmt.Errorf("node name must be a string")
		}

		query, ok := args[1].(Str)
		if !ok {
			return nil, fmt.Errorf("query must be a string")
		}

		// Get SQL node from runtime
		obj, exists := rt.objects[string(nodeName)]
		if !exists {
			return nil, fmt.Errorf("SQL node '%s' not found", nodeName)
		}

		sqlNode, ok := obj.(*SQLNode)
		if !ok {
			return nil, fmt.Errorf("object '%s' is not a SQL node", nodeName)
		}

		// Prepare parameters
		var params []interface{}
		for i := 2; i < len(args); i++ {
			params = append(params, convertToInterface(args[i]))
		}

		// Substitute parameter macros
		queryClean, err := interpolateString(rt, string(query))
		if err != nil {
			return nil, fmt.Errorf("failed to interpolate query: %v", err)
		}

		// Execute query
		results, err := sqlNode.QuerySQL(queryClean, params...)
		if err != nil {
			return nil, fmt.Errorf("query failed: %v", err)
		}

		tmaps := convertArrayToInterface(results)

		return tmaps, nil
	})

	// SQL close
	rt.Register("sqlClose", func(args ...Value) (Value, error) {
		if len(args) != 1 {
			return nil, fmt.Errorf("sqlClose requires 1 argument: nodeName")
		}

		// Unwrap arguments
		for i, arg := range args {
			if tvar, ok := arg.(ScopeEntry); ok {
				args[i] = tvar.Value
			}
		}

		nodeName, ok := args[0].(Str)
		if !ok {
			return nil, fmt.Errorf("node name must be a string")
		}

		// Get SQL node from runtime
		obj, exists := rt.objects[string(nodeName)]
		if !exists {
			return nil, fmt.Errorf("SQL node '%s' not found", nodeName)
		}

		sqlNode, ok := obj.(*SQLNode)
		if !ok {
			return nil, fmt.Errorf("object '%s' is not a SQL node", nodeName)
		}

		// Close the SQL connection
		if err := sqlNode.Close(); err != nil {
			return nil, fmt.Errorf("failed to close SQL connection: %v", err)
		}

		// Remove from runtime objects
		delete(rt.objects, string(nodeName))

		return Str("SQL connection closed"), nil
	})

	// SQL execution (INSERT, UPDATE, DELETE)
	rt.Register("sqlExecute", func(args ...Value) (Value, error) {
		if len(args) < 2 {
			return nil, fmt.Errorf("sqlExecute requires at least 2 arguments: nodeName, statement, [params...]")
		}

		// Unwrap arguments
		for i, arg := range args {
			if tvar, ok := arg.(ScopeEntry); ok {
				args[i] = tvar.Value
			}
		}

		nodeName, ok := args[0].(Str)
		if !ok {
			return nil, fmt.Errorf("node name must be a string")
		}

		stmt, ok := args[1].(Str)
		if !ok {
			return nil, fmt.Errorf("statement must be a string")
		}

		// Get SQL node
		obj, exists := rt.objects[string(nodeName)]
		if !exists {
			return nil, fmt.Errorf("SQL node '%s' not found", nodeName)
		}

		sqlNode, ok := obj.(*SQLNode)
		if !ok {
			return nil, fmt.Errorf("object '%s' is not a SQL node", nodeName)
		}

		// Prepare parameters
		var params []interface{}
		for i := 2; i < len(args); i++ {
			params = append(params, convertToInterface(args[i]))
		}

		// Execute statement
		affected, err := sqlNode.Execute(string(stmt), params...)
		if err != nil {
			return nil, fmt.Errorf("execution failed: %v", err)
		}

		return Number(affected), nil
	})

	// Transaction management
	rt.Register("sqlBegin", func(args ...Value) (Value, error) {
		if len(args) != 1 {
			return nil, fmt.Errorf("sqlBegin requires 1 argument: nodeName")
		}

		// Unwrap arguments
		for i, arg := range args {
			if tvar, ok := arg.(ScopeEntry); ok {
				args[i] = tvar.Value
			}
		}

		nodeName, ok := args[0].(Str)
		if !ok {
			return nil, fmt.Errorf("node name must be a string")
		}

		obj, exists := rt.objects[string(nodeName)]
		if !exists {
			return nil, fmt.Errorf("SQL node '%s' not found", nodeName)
		}

		sqlNode, ok := obj.(*SQLNode)
		if !ok {
			return nil, fmt.Errorf("object '%s' is not a SQL node", nodeName)
		}

		if err := sqlNode.Begin(); err != nil {
			return nil, fmt.Errorf("failed to begin transaction: %v", err)
		}

		return Str("Transaction started"), nil
	})

	rt.Register("sqlCommit", func(args ...Value) (Value, error) {
		if len(args) != 1 {
			return nil, fmt.Errorf("sqlCommit requires 1 argument: nodeName")
		}

		// Unwrap arguments
		for i, arg := range args {
			if tvar, ok := arg.(ScopeEntry); ok {
				args[i] = tvar.Value
			}
		}

		nodeName, ok := args[0].(Str)
		if !ok {
			return nil, fmt.Errorf("node name must be a string")
		}

		obj, exists := rt.objects[string(nodeName)]
		if !exists {
			return nil, fmt.Errorf("SQL node '%s' not found", nodeName)
		}

		sqlNode, ok := obj.(*SQLNode)
		if !ok {
			return nil, fmt.Errorf("object '%s' is not a SQL node", nodeName)
		}

		if err := sqlNode.Commit(); err != nil {
			return nil, fmt.Errorf("failed to commit transaction: %v", err)
		}

		return Str("Transaction committed"), nil
	})

	rt.Register("sqlRollback", func(args ...Value) (Value, error) {
		if len(args) != 1 {
			return nil, fmt.Errorf("sqlRollback requires 1 argument: nodeName")
		}

		// Unwrap arguments
		for i, arg := range args {
			if tvar, ok := arg.(ScopeEntry); ok {
				args[i] = tvar.Value
			}
		}

		nodeName, ok := args[0].(Str)
		if !ok {
			return nil, fmt.Errorf("node name must be a string")
		}

		obj, exists := rt.objects[string(nodeName)]
		if !exists {
			return nil, fmt.Errorf("SQL node '%s' not found", nodeName)
		}

		sqlNode, ok := obj.(*SQLNode)
		if !ok {
			return nil, fmt.Errorf("object '%s' is not a SQL node", nodeName)
		}

		if err := sqlNode.Rollback(); err != nil {
			return nil, fmt.Errorf("failed to rollback transaction: %v", err)
		}

		return Str("Transaction rolled back"), nil
	})

	// Database introspection
	rt.Register("sqlListTables", func(args ...Value) (Value, error) {
		if len(args) != 1 {
			return nil, fmt.Errorf("sqlListTables requires 1 argument: nodeName")
		}

		// Unwrap arguments
		for i, arg := range args {
			if tvar, ok := arg.(ScopeEntry); ok {
				args[i] = tvar.Value
			}
		}

		nodeName, ok := args[0].(Str)
		if !ok {
			return nil, fmt.Errorf("node name must be a string")
		}

		obj, exists := rt.objects[string(nodeName)]
		if !exists {
			return nil, fmt.Errorf("SQL node '%s' not found", nodeName)
		}

		sqlNode, ok := obj.(*SQLNode)
		if !ok {
			return nil, fmt.Errorf("object '%s' is not a SQL node", nodeName)
		}

		tables, err := sqlNode.ListTables()
		if err != nil {
			return nil, fmt.Errorf("failed to list tables: %v", err)
		}

		// Convert to array of table names
		arr := NewArray()
		for _, tableNode := range tables.Elements {
			if sqlRow, ok := tableNode.(*MapValue); ok {
				// Get first column value (table name)
				for _, value := range sqlRow.Values {
					arr.Append(Str(fmt.Sprintf("%v", value)))
					break
				}
			}
		}

		return arr, nil
	})
}

// Helper functions
func convertToInterface(val Value) interface{} {
	switch v := val.(type) {
	case Str:
		return string(v)
	case Number:
		return float64(v)
	case Bool:
		return bool(v)
	case *ArrayValue:
		arr := make([]interface{}, len(v.Elements))
		for i, elem := range v.Elements {
			arr[i] = convertToInterface(elem)
		}
		return arr
	case *MapNode:
		obj := make(map[string]interface{})
		for key, value := range v.Attributes {
			obj[key] = convertToInterface(value)
		}
		return obj
	case map[string]Value:
		obj := make(map[string]interface{})
		for key, value := range v {
			obj[key] = convertToInterface(value)
		}
		return obj
	case *MapValue:
		obj := make(map[string]interface{})
		for key, value := range v.Values {
			obj[key] = convertToInterface(value)
		}
		return obj
	case nil:
		return nil
	default:
		return fmt.Sprintf("%v", v)
	}
}

func convertFromInterface(val interface{}) Value {
	switch v := val.(type) {
	case string:
		return Str(v)
	case int:
		return Number(v)
	case int64:
		return Number(v)
	case float64:
		return Number(v)
	case bool:
		return Bool(v)
	case nil:
		return DBNull
	default:
		return Str(fmt.Sprintf("%v", v))
	}
}
