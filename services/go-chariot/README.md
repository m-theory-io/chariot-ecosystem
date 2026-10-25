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

The Listener Registry is a **DevTest supervisor**, not the production startup
source. Production executes `CHARIOT_BOOTSTRAP` once in one application runtime;
HTTP/NSQ listeners belong to that runtime. Optional Dev REST and network MCP
reuse it. Session clones inherit state/functions without rerunning bootstrap or
owning application listeners.

### Configuration

- `CHARIOT_HEADLESS=true` keeps application listeners alive without Dev REST.
- `CHARIOT_DEV_REST_ENABLED` defaults to true; explicitly disable for production.
- `CHARIOT_PORT` defaults to **8087**; debug examples explicitly select 8090.
- `CHARIOT_DATA_PATH` defaults to `./data`, relative to the working directory.
- `CHARIOT_BOOTSTRAP` defaults to `bootstrap.ch`, resolved under DataPath.
- `CHARIOT_TREE_PATH` independently defaults to `./data/trees`;
    `CHARIOT_FUNCTION_LIB` defaults to `stlib.json` under TreePath. A function
    library file must exist even when bootstrap defines handlers inline.
- `CHARIOT_LISTENERS_FILE` defaults to `listeners.json` under DataPath.
- `CHARIOT_LISTENER_SCRIPTS_DIR` defaults to `listeners` under DataPath; an
    absolute setting is used directly.
- `CHARIOT_SHUTDOWN_TIMEOUT` defaults to 30 seconds.
- `CHARIOT_MCP_EXECUTE_ENABLED` defaults to false. Leave disabled outside trusted
    local development; stdio MCP is an isolated tooling mode, not application startup.

The unprefixed `DATA_PATH`, legacy `CHARIOT_BOOTSTRAP_FILE`, and unsupported
`CHARIOT_ON_START_FILE` are not substitutes for those canonical settings.

### Configuration-only v2 persistence

Only definitions are persisted, never authoritative live status:

```json
{
        "version": 2,
    "listeners": {
        "orders-listener": {
            "name": "orders-listener",
                        "script": "Order test application",
                        "script_file": "processOrders.ch",
                        "on_start": "",
                        "on_start_file": "",
                        "on_exit": "",
                        "on_exit_file": "",
                        "auto_start_devtest": false
        }
    }
}
```

Loading v1 ignores persisted live fields and legacy `auto_start`, and resets
`auto_start_devtest` to false; the next successful definition mutation saves v2.
Start/Stop do not persist transient status. **Auto-start is disabled in both
production and DevTest**, including when the inert v2 preference is true.

`script` is only a display label. `script_file` is required and must create at
least one real, running listener. New definitions and updates require it; repair
legacy definitions before Start. Script/hook files are single `.ch` filenames
confined to the configured scripts root, not arbitrary paths. Registry hook
strings are Chariot programs (or registered library function names), not implied
callbacks from bare `setq` variables; hook files take precedence over strings.
See the canonical reference for ordering and duplicate-hook suppression.

### Managing listeners via API

With Dev REST enabled and session authentication:

- GET/POST `/api/listeners`: list/create definitions.
- PUT/DELETE `/api/listeners/:name`: update/delete after stopping the instance.
- POST `/api/listeners/:name/start`: execute `script_file` in an owned clone and
    require real listener resources; startup failure rolls them back.
- POST `/api/listeners/:name/stop`: drain listeners and hooks, dispose owned
    resources and the runtime. Charioteer's Exit uses this operation.
- `/api/listener-scripts`: list/read/save/delete script files.

API state/health comes from the live test instance and its resources. Restart
loads definitions stopped. Timed-out Stop retains ownership until work drains.
Cloning provides **lifecycle isolation, not a security sandbox**: host objects
may be borrowed and global values shallow-copied. Do not execute untrusted code.

### References and runnable examples

- [Canonical listener reference](../../docs/ListenerFunctions.md): HTTP/NSQ
    options, request/response maps, program hooks, lifecycle, and v2 migration.
- [Canonical System Functions](../../docs/SystemFunctions.md); the service-local
    reference is a navigation link rather than a divergent duplicate.
- [Runnable listener workflow](../../examples/listeners/README.md): two HTTP
    listeners plus NSQ, HTTP-only alternative, debug bootstrap-once counter,
    DevTest script, and real publishing steps with known helper limitations.
- [Container guide](../../docs/Containerizing%20a%20Chariot%20Agent.md): canonical
    environment, data/library resolution, headless/debug profiles and shutdown.

SIGINT/SIGTERM initiate ordered graceful shutdown. Listener startup failures
fail bootstrap and roll back earlier resources. Deadlines cannot forcibly kill
arbitrary handler/hook code; keep it bounded and allow adequate termination grace.

## Contributing

1. Fork the repo
2. Run `go test ./...`
3. Submit a pull request with tests and documentation

## License

MIT © 2025 William J House

