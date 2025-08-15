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

## Contributing

1. Fork the repo
2. Run `go test ./...`
3. Submit a pull request with tests and documentation

## License

MIT © 2025 William J House

