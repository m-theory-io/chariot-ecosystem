# Chariot Language Reference

## MCP Functions

Chariot exposes a small set of closures to act as an MCP (Model Context Protocol) client from within a flow. The helpers let you start an MCP transport, inspect the remote tool registry, call tools, and cleanly close the client.

> **Transport reminder:** stdio is the preferred transport today. WebSocket support is available, but the server endpoint must already be running and clients have to send `Sec-WebSocket-Protocol: modelcontextprotocol.mcp.v1` during the handshake.

---

### `mcpConnect(optionsMap)` → `client`

Starts a new MCP client session and returns an opaque host object handle that is consumed by the other MCP helpers.

| Option key  | Type            | Default   | Notes |
|-------------|-----------------|-----------|-------|
| `transport` | string          | `"stdio"`| `"stdio"` launches the provided command; `"ws"` dials the WebSocket endpoint. |
| `timeoutMs` | number          | `30000`   | Total timeout for the initial handshake. |
| `command`   | string          | _required for stdio_ | Executable to launch for stdio transport. |
| `args`      | array of string | `[]`      | Optional positional args passed to the stdio command. |
| `env`       | map             | inherits process env | Used to extend/override environment variables for the stdio command. |
| `url`       | string          | `ws://127.0.0.1:<port><path>` | Only read when `transport == "ws"`. If omitted, the helper builds a URL from the local config (`cfg.ChariotConfig`). |

When `transport` is `"stdio"`, the helper launches the command via `exec.Command` and keeps the child process alive until `mcpClose`. For `"ws"`, it dials the supplied (or default) WebSocket URL and pipes newline-delimited JSON frames over the connection.

Errors are raised when required keys are missing, arguments are the wrong type, or the handshake fails.

---

### `mcpListTools(client)` → array of string

Fetches the remote tool registry and returns an array of tool names. The call times out after 15 seconds. Use this to confirm that the target MCP server is healthy or to dynamically decide which tool to call.

---

### `mcpCallTool(client, name [, argsMap])` → string

Invokes a single MCP tool by name.

- `client`: Handle produced by `mcpConnect`.
- `name`: Tool identifier (string).
- `argsMap`: Optional map or struct-like value that is converted to native JSON before being sent across the wire.

The helper waits up to 30 seconds for a response.

Return value rules:

1. If the tool returns text content, the first text segment is returned as a string.
2. If the tool returns structured content, the helper tries to unwrap a `result` field or a single-key map; otherwise the entire structure is stringified.
3. If the MCP response marked `isError`, the helper raises an error. When possible, the text payload from the remote error is included.

---

### `mcpClose(client)` → `null`

Closes the active MCP session and, when using stdio transport, terminates the spawned child process. This is idempotent—calling it multiple times is safe.

Always call `mcpClose` when you are done with a connection to avoid leaking MCP sessions or orphan child processes.

---

## Usage Example

```chariot
// Launch a local MCP server over stdio
setq(conn, mcpConnect(map(
  'transport', 'stdio',
  'command', '/usr/local/bin/my-mcp-server',
  'args', array('--profile', 'dev')
)))

// Inspect tools, ensure "ping" exists
setq(tools, mcpListTools(conn))
if (smaller(lastIndex(tools, 'ping'), 0)) {
  error('ping tool missing')
}

// Execute a tool and capture the string response
setq(result, mcpCallTool(conn, 'ping', map('message', 'hello')))
logPrint('Ping response: ' + result)

// Tear down the session
mcpClose(conn)
```

This snippet launches an MCP server over stdio, validates that the `ping` tool is present, calls it, logs the response, and then closes the connection. Adjust the options map for your transport (e.g., supply `url` when dialing an existing WebSocket endpoint).
