# Chariot Language Reference

## SQL Functions

Chariot provides built-in functions for connecting to SQL databases, executing queries and statements, managing transactions, and introspecting database schema. SQL nodes are registered in the runtime and referenced by name for all operations.

---

### Available SQL Functions

| Function              | Description                                                      |
|-----------------------|------------------------------------------------------------------|
| `sqlConnect(nodeName, driver, connectionString, [options...])` | Connect to a SQL database and register the node |
| `sqlQuery(nodeName, query, [params...])`           | Execute a SQL query (SELECT); returns array of maps |
| `sqlExecute(nodeName, statement, [params...])`     | Execute a SQL statement (INSERT, UPDATE, DELETE); returns affected row count |
| `sqlClose(nodeName)`                              | Close the SQL connection and remove the node        |
| `sqlBegin(nodeName)`                              | Begin a transaction on the SQL node                 |
| `sqlCommit(nodeName)`                             | Commit the current transaction                      |
| `sqlRollback(nodeName)`                           | Roll back the current transaction                   |
| `sqlListTables(nodeName)`                         | List all table names in the connected database      |

---

### Function Details

#### `sqlConnect(nodeName, driver, connectionString, [options...])`

Connects to a SQL database and registers the node for future operations.
- `nodeName`: Unique name for the SQL node (string)
- `driver`: SQL driver name (e.g., `"mysql"`, `"sqlite3"`)
- `connectionString`: Connection string or URL
- `[options...]`: Optional additional options (see your config)

Returns a confirmation string.

```chariot
sqlConnect('db1', 'sqlite3', 'file:mydb.sqlite')
```

#### `sqlQuery(nodeName, query, [params...])`

Executes a SQL SELECT query with optional parameters.
- Returns an array of maps (one per row).

```chariot
sqlQuery('db1', 'SELECT * FROM customers WHERE age > ?', 30)
```

#### `sqlExecute(nodeName, statement, [params...])`

Executes a SQL statement (INSERT, UPDATE, DELETE) with optional parameters.
- Returns the number of affected rows.

```chariot
sqlExecute('db1', 'UPDATE customers SET active = ? WHERE id = ?', true, 123)
```

#### `sqlClose(nodeName)`

Closes the SQL connection and removes the node from the runtime.

```chariot
sqlClose('db1')
```

#### `sqlBegin(nodeName)`

Begins a transaction on the SQL node.

```chariot
sqlBegin('db1')
```

#### `sqlCommit(nodeName)`

Commits the current transaction.

```chariot
sqlCommit('db1')
```

#### `sqlRollback(nodeName)`

Rolls back the current transaction.

```chariot
sqlRollback('db1')
```

#### `sqlListTables(nodeName)`

Lists all table names in the connected database.
- Returns an array of table name strings.

```chariot
sqlListTables('db1')
```

---

### Notes

- All SQL operations require a valid node connection (`sqlConnect`).
- Parameters for queries and statements are passed as additional arguments.
- Transactions (`sqlBegin`, `sqlCommit`, `sqlRollback`) are per node.
- `sqlQuery` returns an array of maps, each representing a row.
- `sqlExecute` returns the number of affected rows as a number.
- `sqlListTables` returns an array of table names as strings.
- Closing a node with `sqlClose` removes it from the runtime.

---
