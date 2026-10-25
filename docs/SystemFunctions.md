# Chariot Language Reference

> Canonical System Functions reference. The service-local document links here;
> update this file rather than maintaining a second copy.

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
| `listen(port, handler [, options])` | Start a validated HTTP application listener; see [Listener Functions](ListenerFunctions.md) |
| `listenNSQ(topic, channel, handlers [, options])` | Start an NSQ application listener; see [Listener Functions](ListenerFunctions.md) |
| `httpResponse(status [, headers], body)` | Create explicit HTTP response metadata |
| `listenerList()`, `listenerStatus(id)`, `listenerStop(id)` | Inspect and control listeners owned by the current runtime |

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

#### Listener functions

The canonical `listen()`, `listenNSQ()`, `httpResponse()`, and runtime lifecycle contracts are documented in [Listener Functions](ListenerFunctions.md). Unknown option names and legacy aliases are rejected.

`listenerStop(id)` requests graceful stop asynchronously: `true` means accepted,
not completed. Unknown or already-removed IDs return an error. The listener
remains visible as `stopping` through active handlers and exit hooks; after
completion, `listenerStatus(id)` returns `DBNull` with no tombstone. Go's
`StopListener(ctx, id)` waits for completion and must be called outside
`WithExecution`; use `RequestStopListener(id)` for request-only semantics.

---

### Notes

- `getEnv` and `hasEnv` are useful for accessing environment configuration.
- `logPrint` integrates with the Chariot logging system and supports structured logging with additional fields.
- `platform` and `timestamp` provide runtime introspection.
- `timeFormat` uses Go's time formatting (e.g., `"2006-01-02 15:04:05"`).
- `exit` requests program termination; actual exit handling is managed by the runtime.
- Application listeners belong to the runtime that creates them and are shut down by the application host.
- All arguments are automatically unwrapped from `ScopeEntry` if needed.

---