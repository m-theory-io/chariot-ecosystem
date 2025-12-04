# chariot-ecosystem
Chariot monorepo for go-chariot, charioteer and visual-dsl

## Canonical build and push (Azure)

Use this exact flow for all builds and pushes. Tags are typically versioned (e.g., v0.026).

```
./scripts/build-azure-cross-platform.sh <tag> [all|go-chariot|charioteer|visual-dsl|nginx]
./scripts/push-images.sh <tag> [all|go-chariot|charioteer|visual-dsl|nginx]
```

Notes
- docker-compose.azure.yml defaults: nginx uses tag `amd64` as the default alias; the push script updates that alias when you push a versioned tag.
- The older `deploy-azure.sh` script has been removed to avoid confusion.

## Secret providers

Chariot can now run with either Azure Key Vault (`CHARIOT_SECRET_PROVIDER=azure`) or a local JSON-backed provider (`CHARIOT_SECRET_PROVIDER=file`). See `docs/SecretManagement.md` for configuration examples and migration notes.

## Per-user sandboxes

The REST service can isolate on-disk artifacts (files, trees, Visual DSL diagrams) per authenticated user. Enable the feature with the following environment variables (defaults shown):

| Variable | Description |
| --- | --- |
| `CHARIOT_SANDBOX_ENABLED=false` | Turn on sandbox-aware storage. When disabled, all scopes collapse to `global`. |
| `CHARIOT_SANDBOX_ROOT=$CHARIOT_DATA_PATH/sandboxes` | Root directory where per-user folders are created. Each user gets `/<sanitized-user>/<data|trees|diagrams>`. |
| `CHARIOT_SANDBOX_DEFAULT_SCOPE=sandbox` | Preferred scope (`sandbox` or `global`) used when a client does not pass an explicit scope. |

Runtime details:

- The go-chariot API now exposes `GET /api/session/profile`, which returns the authenticated username, the available scopes, and the sanitized sandbox key. Clients (charioteer and visual-dsl) use this to render scope pickers.
- Diagram and file CRUD endpoints accept `?scope=sandbox|global` and always emit `X-Chariot-Scope` so callers know which scope actually handled the request.
  - Files: `GET /api/files`, `GET /api/files/:name`, `POST /api/files`, `DELETE /api/files/:name`
  - Diagrams: `GET /api/diagrams`, `GET /api/diagrams/:name`, `POST /api/diagrams`, `DELETE /api/diagrams/:name`
- Charioteer's Files tab shows a "Scope" dropdown (when sandboxes enabled) allowing users to switch between sandbox and global file storage. The dropdown appears to the left of the file selector.
- When sandboxes are enabled, both front-ends show scope controls. Charioteer's Diagrams tab and Visual DSL include a "Share to global" checkbox for one-off global saves.
- The Functions tab always uses global/server storage and does not display scope controls (function library is shared across all users).
- On first login with sandboxes enabled, user directories are auto-created at `<CHARIOT_SANDBOX_ROOT>/<sanitized-username>/{data,trees,diagrams}`.
- When sandboxes are disabled, the UI hides the additional controls and requests continue to use the legacy global paths.

These changes do not introduce new breaking storage locations for legacy deployments—the feature is opt-in via `CHARIOT_SANDBOX_ENABLED=true`.

## Model Context Protocol (MCP) integration

go-chariot includes an optional MCP server built with the official Go SDK. It supports stdio transport today and has a placeholder route for WebSocket (WS) transport.

### Transports

- stdio: Recommended. The process runs only the MCP server and exits when the MCP client ends the session.
- ws: A route is mounted, but currently returns 501 (not implemented). Prefer stdio for now.

### Configuration

You can configure via environment variables (preferred) or via flags defined in code.

- Environment variables
	- `CHARIOT_MCP_ENABLED` (bool, default: `false`)
	- `CHARIOT_MCP_TRANSPORT` (string: `stdio` | `ws`, default: `stdio`)
	- `CHARIOT_MCP_WS_PATH` (string, default: `/mcp`)

- Flags (wired in `services/go-chariot/cmd/main.go`)
	- `mcp_enabled`
	- `mcp_transport`
	- `mcp_ws_path`

### Run (stdio)

From the `services/go-chariot` folder:

```bash
# Run directly with stdio transport
export CHARIOT_MCP_ENABLED=true
export CHARIOT_MCP_TRANSPORT=stdio
go run ./cmd

# Or build the binary, then run it
go build ./cmd
CHARIOT_MCP_ENABLED=true CHARIOT_MCP_TRANSPORT=stdio ./cmd
```

Notes
- In stdio mode, the REST API is not started; the process serves MCP over stdio and exits when the client disconnects.
- In ws mode, the HTTP server starts as usual; the WebSocket route is registered at `CHARIOT_MCP_WS_PATH`, but currently returns 501 until WS transport is implemented.

### Available tools

The MCP server currently registers:
- `ping`: health check tool returning a simple response.
- `execute`: executes a Chariot program and returns the last value as a string.
- `codeToDiagram`: placeholder returning "not implemented" (planned for future AST→diagram support).

If you need a client config example (e.g., to wire this into an MCP-capable app), point the client to launch go-chariot with `CHARIOT_MCP_ENABLED=true` and `CHARIOT_MCP_TRANSPORT=stdio`, or wrap that in a small shell script.
