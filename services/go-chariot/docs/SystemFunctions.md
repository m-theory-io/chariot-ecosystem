# Chariot Language Reference

## System Functions

Chariot provides system-level functions for environment access, logging, runtime/platform information, time utilities, program control, and server listening.

---

### Available System Functions

| Function           | Description                                                      |
|--------------------|------------------------------------------------------------------|
| `getEnv(name)`     | Get the value of an environment variable (returns `DBNull` if not set) |
| `hasEnv(name)`     | Returns `true` if the environment variable is set                |
| `logPrint(message [, level, ...fields])` | Log a message at the specified level (`info`, `debug`, `warn`, `error`) with optional structured fields |
| `platform()`       | Returns the current OS platform as a string (e.g., `"linux"`)    |
| `timestamp()`      | Returns the current Unix timestamp (seconds since epoch)         |
| `timeFormat(timestamp, format)` | Format a Unix timestamp using a Go-style format string |
| `exit([code])`     | Request program exit with optional exit code (default: 0)        |
| `sleep(ms)`        | Pause execution for the specified milliseconds                   |
| `listen(port [, onstart, onexit])` | Start a server listener on the given port, with optional startup/shutdown programs |

---

### Function Details

#### `getEnv(name)`

Returns the value of the environment variable `name`, or `DBNull` if not set.

```chariot
getEnv('HOME')
```

#### `hasEnv(name)`

Returns `true` if the environment variable `name` is set.

```chariot
hasEnv('PATH')
```

#### `logPrint(message [, level, ...fields])`

Logs a message.
- `message`: String to log.
- `level`: Optional log level (`"info"` (default), `"debug"`, `"warn"`, `"error"`).
- Additional arguments can be TreeNodes or values for structured logging.

```chariot
logPrint('Starting process')
logPrint('Something went wrong', 'error')
logPrint('User login', 'info', mapNode('user', 'alice', 'ip', '127.0.0.1'))
```

#### `platform()`

Returns the current OS platform as a string (e.g., `"linux"`, `"darwin"`, `"windows"`).

```chariot
platform()  // "linux"
```

#### `timestamp()`

Returns the current Unix timestamp (seconds since epoch).

```chariot
timestamp()  // 1724784000
```

#### `timeFormat(timestamp, format)`

Formats a Unix timestamp using a Go-style format string.

```chariot
timeFormat(1724784000, "2006-01-02 15:04:05")  // "2024-08-28 00:00:00"
```

#### `exit([code])`

Requests program exit with the given exit code (default: 0).

```chariot
exit()      // Exit with code 0
exit(1)     // Exit with code 1
```

#### `sleep(ms)`

Pauses execution for the specified number of milliseconds.

```chariot
sleep(500)  // Sleep for 500 milliseconds
```

#### `listen(port [, onstart, onexit])`

Starts a server listener on the given port.
- `port`: Port number (number)
- `onstart`: (Optional) Chariot program to run on startup
- `onexit`: (Optional) Chariot program to run on shutdown

```chariot
listen(8080)
listen(8081, "onStartProgram", "onExitProgram")
```

---

### Notes

- `getEnv` and `hasEnv` are useful for accessing environment configuration.
- `logPrint` integrates with the Chariot logging system and supports structured logging with additional fields.
- `platform` and `timestamp` provide runtime introspection.
- `timeFormat` uses Go's time formatting (e.g., `"2006-01-02 15:04:05"`).
- `exit` requests program termination; actual exit handling is managed by the runtime.
- `listen` is for server or event-driven applications; `onstart` and `onexit` are optional Chariot programs to run at server lifecycle events.
- All arguments are automatically unwrapped from `ScopeEntry` if needed.

---