# Go-Chariot

A lightweight, embeddable interpreter for the **Chariot** data‑scripting language in Go.

Chariot is a functional, data‑centric scripting language where everything is a function call. This project provides:

- **`chariot/`**: core Go library to parse, interpret, and execute Chariot scripts
- **`cmd/chariotctl/`**: CLI tool to run `.ch` scripts from the command line
- **`handlers/`**: Echo HTTP handler exposing an API endpoint to execute scripts over REST

## Features

- Recursive‑descent parser and lexer for Chariot syntax (identifiers, literals, calls, blocks)
- AST (`VarRef`, `Literal`, `FuncCall`, `Block`) with execution via `Node.Exec`
- Built‑in functions for variables (`declare`, `setq`, `valueOf`), arithmetic (`add`, `smallerEq`), string ops (`append`, `format`), and control flow (`while`)
- Host‑binding: expose Go objects/methods to Chariot via `Runtime.BindObject`
- Modular design: clean separation of **ast.go**, **parser.go**, **runtime.go**, **builtins.go**

## Installation

```bash
go get github.com/bhouse1273/go-chariot/chariot
```

Or include in your `go.mod`:

```go
require github.com/bhouse1273/go-chariot v0.0.0
```

## Usage

### As a library

```go
import (
    "github.com/bhouse1273/go-chariot/chariot"
)

func main() {
    rt := chariot.NewRuntime()
    chariot.RegisterBuiltins(rt)

    // Optionally bind host objects
    // rt.BindObject("db", dbClient)

    script := `
        declare(n, 'N', 0)
        while(smallerEq(n, 5)) {
            setq(n, add(n, 1))
        }
        append('Result: n=', n)
    `

    result, err := rt.ExecProgram(script)
    if err != nil {
        log.Fatal(err)
    }
    fmt.Println(result)
}
```

### Command‑Line Tool

Build the `chariotctl` CLI:

```bash
cd github.com/bhouse1273/go-chariot
go build -o chariotctl ./cmd/chariotctl
```

Run your script:

```bash
./chariotctl -f path/to/script.ch
```

Or install globally:

```bash
go install github.com/bhouse1273/go-chariot/cmd/chariotctl@latest
```

### HTTP Handler

Use the Echo handler in your web service:

```go
import (
    "github.com/bhouse1273/go-chariot/handlers"
)

e := echo.New()
e.POST("/execute", handlers.Execute)
e.Start(":8080")
```

Request JSON:

```json
{ "program": "declare(x,'N',10); append('x=', x)" }
```

Response JSON:

```json
{ "result": "x=10" }
```

## Project Structure

```
go-chariot/
├── chariot/            # Core interpreter library
│   ├── ast.go
│   ├── parser.go
│   ├── runtime.go
│   ├── builtins.go
├── cmd/
│   └── chariotctl/     # CLI entrypoint
├── handlers/           # HTTP handler for Echo
├── go.mod
└── README.md           # This file
```

## Listener Registry

Go‑Chariot includes a lightweight Listener Registry for managing long‑running scripts with lifecycle hooks. This is useful for background services (for example, decision agents, ETL listeners, webhook consumers) that need a Start/Exit lifecycle and persistence across restarts.

### Configuration

- CHARIOT_DEV_REST_ENABLED (bool, default true): Enables the Dev REST API server. Can run with or without headless mode.
- CHARIOT_LISTENERS_FILE (string, default "listeners.json"): Filename (under CHARIOT_DATA_PATH) where the registry is persisted.
- CHARIOT_DATA_PATH (string, default "./data"): Base path for persisted data.

The full persistence path is: `${CHARIOT_DATA_PATH}/${CHARIOT_LISTENERS_FILE}`.

### JSON format: listeners.json

The file stores a Snapshot object containing a version and a map of listeners keyed by name:

```json
{
    "version": 1,
    "listeners": {
        "orders-listener": {
            "name": "orders-listener",
            "script": "processOrders.ch",
            "on_start": "startOrdersService()",
            "on_exit": "stopOrdersService()",
            "status": "stopped",
            "start_time": "2025-09-29T15:04:05.000Z",
            "last_active": "2025-09-29T15:04:05.000Z",
            "is_healthy": false
        }
    }
}
```

Field meanings:

- name: Unique listener name (key and field value should match).
- script: Primary script or program identifier (optional; for display/traceability).
- on_start: Chariot program text to run when starting the listener.
- on_exit: Chariot program text to run when exiting the listener.
- status: One of "running" | "stopped" | "error". Maintained by the manager.
- start_time: RFC3339 timestamp when last started.
- last_active: RFC3339 timestamp of last heartbeat/activity (manager sets initially; your scripts may update it through future APIs).
- is_healthy: Boolean health indicator set by the manager or your scripts.

### Managing listeners via API

All endpoints are under `/api/listeners` (protected by session auth):

- GET `/api/listeners` → list all listeners
- POST `/api/listeners` with body:
    `{ "name": "orders-listener", "script": "processOrders.ch", "on_start": "startOrdersService()", "on_exit": "stopOrdersService()" }`
- DELETE `/api/listeners/:name` → delete (must be stopped)
- POST `/api/listeners/:name/start` → run the on_start program and mark running
- POST `/api/listeners/:name/stop` → run the on_exit program and mark stopped

When headless mode is enabled, the Dev REST server can still be enabled or disabled independently using `CHARIOT_DEV_REST_ENABLED`.

## Contributing

1. Fork the repo
2. Run `go test ./...`
3. Submit a pull request with tests and documentation

## License

MIT © 2025 William J House

