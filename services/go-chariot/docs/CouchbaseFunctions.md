# Chariot Language Reference

## Couchbase Functions

Chariot provides built-in support for Couchbase database operations, including connection management, bucket and collection access, document CRUD, and N1QL queries.

### Available Couchbase Functions

| Function            | Description                                                      |
|---------------------|------------------------------------------------------------------|
| `cbConnect(nodeName, connectionString, username, password)` | Connect to a Couchbase cluster and register the node in the runtime |
| `cbOpenBucket(nodeName, bucketName)`                | Open a bucket on a connected Couchbase node                         |
| `cbSetScope(nodeName, scopeName, collectionName)`   | Set the active scope and collection for the node                    |
| `cbQuery(nodeName, query, [params])`                | Execute a N1QL query with optional parameters                       |
| `cbInsert(nodeName, documentId, document, [expiryDuration])` | Insert a document with optional expiry (seconds or duration string) |
| `cbUpsert(nodeName, documentId, document, [expiryDuration])` | Insert or update a document with optional expiry                    |
| `cbGet(nodeName, documentId)`                       | Retrieve a document by ID                                           |
| `cbRemove(nodeName, documentId)`                    | Remove a document by ID                                             |
| `cbReplace(nodeName, documentId, document, [cas], [expiryDuration])` | Replace a document by ID, with optional CAS and expiry              |
| `cbClose(nodeName)`                                 | Close the Couchbase connection and remove the node from runtime     |
| `newID([prefix], [format])`                         | Generate a new document ID with optional prefix and format          |
---

### Function Details

#### `cbConnect(nodeName, connectionString, username, password)`

Connect to a Couchbase cluster and register the node for future operations.

```chariot
cbConnect('cb1', 'couchbase://localhost', 'admin', 'password')
```

#### `cbOpenBucket(nodeName, bucketName)`

Open a bucket on a connected Couchbase node.

```chariot
cbOpenBucket('cb1', 'myBucket')
```

#### `cbSetScope(nodeName, scopeName, collectionName)`

Set the active scope and collection for the node.

```chariot
cbSetScope('cb1', 'myScope', 'myCollection')
```

#### `cbQuery(nodeName, query, [params])`

Execute a N1QL query. `params` can be a JSON object, map, or array.

```chariot
cbQuery('cb1', 'SELECT * FROM myBucket WHERE type = $type', map('type', 'customer'))
```

#### `cbInsert(nodeName, documentId, document, [expiryDuration])`

Insert a document. `expiryDuration` is optional (seconds as number or duration string like `"24h"`).

```chariot
cbInsert('cb1', 'doc123', map('name', 'Alice', 'age', 30))
cbInsert('cb1', 'doc124', map('name', 'Bob'), '48h')
```

#### `cbUpsert(nodeName, documentId, document, [expiryDuration])`

Insert or update a document. Same expiry options as `cbInsert`.

```chariot
cbUpsert('cb1', 'doc123', map('name', 'Alice', 'age', 31))
```

#### `cbGet(nodeName, documentId)`

Retrieve a document by ID.

```chariot
cbGet('cb1', 'doc123')
```

#### `cbRemove(nodeName, documentId)`

Remove a document by ID.

```chariot
cbRemove('cb1', 'doc123')
```

#### `cbReplace(nodeName, documentId, document, [cas], [expiryDuration])`

Replace a document by ID, with optional CAS (for concurrency control) and expiry.

```chariot
cbReplace('cb1', 'doc123', map('name', 'Alice', 'age', 32))
cbReplace('cb1', 'doc123', map('name', 'Alice'), '123456789', '24h')
```

#### `cbClose(nodeName)`

Close the Couchbase connection and remove the node from the runtime.

```chariot
cbClose('cb1')
```

#### `newID([prefix], [format])`

Generate a new document ID. `prefix` is optional (default: `"doc"`), `format` is optional (default: `"short"`).

```chariot
newID()              // e.g., "doc_20240627_001"
newID('customer')    // e.g., "customer_20240627_002"
newID('order', 'long')
```

---

### Notes

- All Couchbase operations require a valid node connection (`cbConnect`) and an open bucket (`cbOpenBucket`).
- Document arguments can be Chariot maps, JSON nodes, or SimpleJSON objects.
- Expiry durations can be specified as seconds (number) or as a duration string (e.g., `"24h"`, `"7d"`).
- Query parameters can be provided as maps or arrays.
- Returned documents include metadata such as CAS and expiry when available.

---
