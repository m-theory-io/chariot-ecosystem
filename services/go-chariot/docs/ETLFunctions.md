# Chariot Language Reference

## ETL Functions

Chariot provides a robust set of ETL (Extract, Transform, Load) functions for data pipeline automation, CSV ingestion, transformation, and loading into SQL or Couchbase targets. ETL transforms can be defined inline or reused from a registry of common data cleaning and validation routines.

---

### ETL Job Functions

| Function                  | Description                                                                 |
|---------------------------|-----------------------------------------------------------------------------|
| `doETL(jobId, csvFile, transformConfig, targetConfig [, options])` | Run an ETL job from CSV to target with transformation and options |
| `etlStatus(jobId)`        | Get the status or log of an ETL job                                         |

#### `doETL(jobId, csvFile, transformConfig, targetConfig [, options])`

Runs an ETL job:
- `jobId`: Unique job identifier (string)
- `csvFile`: Path to the CSV file
- `transformConfig`: Transform object or mapping config (see below)
- `targetConfig`: Target database config (map: type, connection info, etc.)
- `options`: (optional) Map of options (e.g., delimiter, encoding, clientId)

Returns a result node with job status, timing, and processing stats.

```chariot
doETL('job42', 'customers.csv', myTransform, map('type', 'sql', 'driver', 'sqlite3', 'connectionString', 'file.db', 'tableName', 'customers'))
```

#### `etlStatus(jobId)`

Returns the status or log node for the given ETL job.

```chariot
etlStatus('job42')
```

---

### ETL Transform Construction

| Function                  | Description                                                                 |
|---------------------------|-----------------------------------------------------------------------------|
| `createTransform(name)`   | Create a new ETL transform object                                           |
| `addMapping(transform, sourceField, targetColumn, program, dataType, required [, defaultValue])` | Add a mapping with inline program |
| `addMappingWithTransform(transform, sourceField, targetColumn, transformName, dataType, required [, defaultValue])` | Add a mapping using a named transform |

#### `createTransform(name)`

Creates a new transform object for mapping and validation.

```chariot
setq(t, createTransform('customerTransform'))
```

#### `addMapping(transform, sourceField, targetColumn, program, dataType, required [, defaultValue])`

Adds a mapping to a transform.  
- `program` can be a string or array of strings (Chariot code).
- `required` is a boolean.

```chariot
addMapping(t, 'email', 'email_address', "toLowerCase(trim(sourceValue))", 'VARCHAR', true)
```

#### `addMappingWithTransform(transform, sourceField, targetColumn, transformName, dataType, required [, defaultValue])`

Adds a mapping using a named transform from the registry.

```chariot
addMappingWithTransform(t, 'ssn', 'ssn_formatted', 'ssn', 'VARCHAR', true)
```

---

### Transform Registry Functions

| Function                  | Description                                                                 |
|---------------------------|-----------------------------------------------------------------------------|
| `listTransforms()`        | List all available named transforms                                         |
| `getTransform(name)`      | Get details about a named transform (description, dataType, category, etc.) |
| `registerTransform(name, config)` | Register a new named transform at runtime                           |

#### `listTransforms()`

Returns an array of available transform names.

```chariot
listTransforms() // ["ssn", "email", "phone_us", ...]
```

#### `getTransform(name)`

Returns a map with details about the named transform.

```chariot
getTransform('ssn')
```

#### `registerTransform(name, config)`

Registers a new named transform at runtime.  
`config` is a map with keys: `description`, `dataType`, `category`, `program` (array of strings).

```chariot
registerTransform('zip5', map(
  'description', 'Validates and formats US ZIP codes',
  'dataType', 'VARCHAR',
  'category', 'validation',
  'program', array(
    "setq(cleaned, regexReplace(sourceValue, '[^0-9]', ''))",
    "if(equal(length(cleaned), 5), cleaned, error('Invalid ZIP code'))"
  )
))
```

---

## ETL Transform Registry

Chariot includes a built-in registry of common ETL transforms for validation, formatting, and conversion. These can be referenced by name in ETL mappings.

### Built-in Transforms

| Name         | Description                              | Category      | Example Usage                |
|--------------|------------------------------------------|---------------|------------------------------|
| `ssn`        | Validates and formats Social Security Numbers | validation | addMappingWithTransform(t, 'ssn', 'ssn_formatted', 'ssn', 'VARCHAR', true) |
| `email`      | Validates and normalizes email addresses | validation    | addMappingWithTransform(t, 'email', 'email', 'email', 'VARCHAR', true)      |
| `phone_us`   | Validates and formats US phone numbers   | formatting    | addMappingWithTransform(t, 'phone', 'phone', 'phone_us', 'VARCHAR', false)  |
| `currency_usd` | Converts currency strings to decimal cents | conversion  | addMappingWithTransform(t, 'amount', 'amount_cents', 'currency_usd', 'INT', true) |
| `date_mdy`   | Parses MM/DD/YYYY dates to ISO format    | conversion    | addMappingWithTransform(t, 'dob', 'birth_date', 'date_mdy', 'DATE', true)   |
| `boolean`    | Converts various boolean representations | conversion    | addMappingWithTransform(t, 'active', 'is_active', 'boolean', 'BOOLEAN', false) |

#### Example: SSN Transform

```chariot
addMappingWithTransform(t, 'ssn', 'ssn_formatted', 'ssn', 'VARCHAR', true)
```

- Cleans input, validates length, formats as `XXX-XX-XXXX`.

#### Example: Email Transform

```chariot
addMappingWithTransform(t, 'email', 'email', 'email', 'VARCHAR', true)
```

- Trims, lowercases, validates format.

#### Example: Phone Number Transform

```chariot
addMappingWithTransform(t, 'phone', 'phone', 'phone_us', 'VARCHAR', false)
```

- Cleans, validates, formats as `(XXX) XXX-XXXX`.

#### Example: Currency Transform

```chariot
addMappingWithTransform(t, 'amount', 'amount_cents', 'currency_usd', 'INT', true)
```

- Removes symbols, parses as float, multiplies by 100 for cents.

#### Example: Date Transform

```chariot
addMappingWithTransform(t, 'dob', 'birth_date', 'date_mdy', 'DATE', true)
```

- Parses MM/DD/YYYY or ISO dates, returns ISO format.

#### Example: Boolean Transform

```chariot
addMappingWithTransform(t, 'active', 'is_active', 'boolean', 'BOOLEAN', false)
```

- Converts "yes", "no", "1", "0", "active", etc. to boolean.

---

## Example: Complete ETL Pipeline

```chariot
// 1. Create a transform and add mappings
setq(t, createTransform('customerTransform'))
addMappingWithTransform(t, 'ssn', 'ssn_formatted', 'ssn', 'VARCHAR', true)
addMappingWithTransform(t, 'email', 'email', 'email', 'VARCHAR', true)
addMappingWithTransform(t, 'phone', 'phone', 'phone_us', 'VARCHAR', false)
addMappingWithTransform(t, 'amount', 'amount_cents', 'currency_usd', 'INT', true)
addMappingWithTransform(t, 'dob', 'birth_date', 'date_mdy', 'DATE', true)
addMappingWithTransform(t, 'active', 'is_active', 'boolean', 'BOOLEAN', false)

// 2. Define target config
setq(target, map(
  'type', 'sql',
  'driver', 'sqlite3',
  'connectionString', 'file:customers.db',
  'tableName', 'customers'
))

// 3. Run ETL job
doETL('job42', 'customers.csv', t, target)
```

---

### Notes

- **Transforms** can be defined inline (with `addMapping`) or by reference (with `addMappingWithTransform`).
- **Transform programs** are written in Chariot and can use any built-in function.
- **Target config** supports SQL and Couchbase; see your system for supported fields.
- **Options** for `doETL` include `delimiter`, `encoding`, `hasHeaders`, and `clientId`.
- **Job status** and logs can be retrieved with `etlStatus(jobId)`.
- The transform registry is extensible at runtime with `registerTransform`.

---