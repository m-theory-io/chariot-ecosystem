# Chariot Language Reference

## MCP Client Functions

Chariot exposes a small set of closures to act as an MCP (Model Context Protocol) client from within a flow. The helpers let you start an MCP transport, inspect the remote tool registry, call tools, and cleanly close the client.

> **Transport reminder:** stdio is useful when a client launches an MCP server process directly. HTTP/SSE is the usual transport for the local `go-chariot` service and Charioteer. WebSocket support is available, but the server endpoint must already be running and clients have to send `Sec-WebSocket-Protocol: modelcontextprotocol.mcp.v1` during the handshake.

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

---

## Chariot MCP Server Tools

The `go-chariot` service can also run as an MCP server. It exposes general Chariot execution tools, a registry of running agents and program trees, and convenience tools for common registry operations.

Available transports:

| Transport | Entry point | Notes |
|-----------|-------------|-------|
| stdio | `go-chariot mcp` / configured command | Good for local MCP clients that launch the server process. |
| HTTP/SSE | `/mcp` | Used by VS Code and Charioteer HTTP MCP clients. The handler supports MCP SSE fallback. |
| WebSocket | `/mcp/ws` | Requires `Sec-WebSocket-Protocol: modelcontextprotocol.mcp.v1`. |

When the MCP server is mounted by the `go-chariot` HTTP app, registry calls use the live bootstrap runtime. That means tools such as `agentCall runPlanOnce` can resolve plan variables created by `bootstrap.ch`.

### Core Tools

| Tool | Input | Output | Notes |
|------|-------|--------|-------|
| `ping` | `{ "message": "hello" }` | `{ "reply": "pong: hello" }` | Connectivity check. |
| `execute` | `{ "code": "setq(x, add(5,5))" }` | `{ "result": "10" }` | Executes a standalone Chariot program in a fresh runtime. |
| `codeToDiagram` | `{ "code": "..." }` | _not implemented_ | Reserved for future Visual DSL conversion. |

### Registry Tools

The registry gives MCP callers a discoverable surface for running agents and saved program trees.

| Tool | Purpose |
|------|---------|
| `registryList` | List all discoverable running agents and program tree files. |
| `registryDescribe` | Describe one registry item by id, such as `agent:thermostat` or `tree:decisionAgent1`. |
| `registryCall` | Call any registry item, including agent actions and tree functions. |
| `agentList` | Convenience wrapper that lists only running agents. |
| `agentCall` | Convenience wrapper for agent actions by agent name. |
| `treeList` | Convenience wrapper that lists only program tree files. |
| `treeDescribe` | Convenience wrapper for describing a program tree by name or id. |
| `treeCall` | Convenience wrapper for calling a function-valued tree attribute. |

Registry item ids use these forms:

| Id form | Meaning |
|---------|---------|
| `agent:<name>` | A running Chariot agent. |
| `tree:<name>` | A saved program tree file from the configured tree path. |
| `tree:<tree>.<node>.<attribute>` | A callable function-valued attribute in a program tree. |

---

## Agent MCP Calls

Use `agentList` to discover running agents:

```json
{}
```

Example result:

```json
{
  "agents": [
    {
      "id": "agent:thermostat",
      "kind": "agent",
      "name": "thermostat",
      "callable": true,
      "metadata": {
        "name": "thermostat",
        "plans": ["Thermostat"],
        "running": true,
        "pollSeconds": 3,
        "beliefCount": 0
      }
    }
  ],
  "count": 1
}
```

### `agentCall`

Input shape:

```json
{
  "name": "thermostat",
  "action": "runPlanOnce",
  "input": {}
}
```

Supported actions:

| Action | Input | Result |
|--------|-------|--------|
| `info` / `describe` | none | Agent metadata. |
| `getBeliefs` / `beliefs` | none | Current belief values converted to JSON. |
| `publish` / `nudge` | none | `{ "published": true }` when the scheduler accepts the nudge. This is a wake-up signal, not a plan result. |
| `setBelief` / `setBeliefs` | map of belief keys to JSON values | Records the beliefs and nudges the agent. |
| `runPlanOnce` / `runOnce` / `execute` | plan execution input | Runs a plan once and returns a structured plan result. |

### Setting Beliefs

```json
{
  "name": "thermostat",
  "action": "setBeliefs",
  "input": {
    "currentTemp": 85,
    "lower": 70,
    "upper": 74
  }
}
```

Example result:

```json
{
  "call": {
    "id": "agent:thermostat",
    "kind": "agent",
    "action": "setbeliefs",
    "result": {
      "set": {
        "currentTemp": 85,
        "lower": 70,
        "upper": 74
      }
    }
  }
}
```

### Running a Plan Once

Use `runPlanOnce` when an MCP caller needs an actual result object. `publish` only wakes the scheduler and returns whether that nudge was accepted.

```json
{
  "name": "thermostat",
  "action": "runPlanOnce",
  "input": {
    "plan": "pThermostat",
    "beliefs": {
      "currentTemp": 85,
      "lower": 70,
      "upper": 74
    }
  }
}
```

The `plan` field is the plan variable name in the bootstrap runtime, for example `pThermostat`. The result object's `plan` field is the plan's human-readable name, for example `Thermostat`.

Optional fields:

| Field | Type | Notes |
|-------|------|-------|
| `plan` / `planVar` / `planName` | string | Required. Name of the global plan variable. |
| `beliefs` | object | Beliefs to set on the named agent before the run. Also used as instance variables when `vars` is omitted. |
| `vars` | object | Per-run instance variables used by BDI-style modes. |
| `mode` | string | Defaults to `plain`. Use `bdi`, `guard-only`, `force`, `force-all`, or `dry-run` for gated execution. |

Modes:

| Mode | Trigger | Guard | Drop | Steps |
|------|---------|-------|------|-------|
| `plain` | bypassed | bypassed | respected | executed |
| `bdi` | required | required | respected | executed only when trigger and guard pass |
| `guard-only` | bypassed | required | respected | executed when guard passes |
| `force` | bypassed | bypassed | respected | executed unless dropped |
| `force-all` | bypassed | bypassed | bypassed | executed |
| `dry-run` | required | required | respected | not executed; reports whether the plan would run |

Example successful result:

```json
{
  "call": {
    "id": "agent:thermostat",
    "kind": "agent",
    "action": "runplanonce",
    "executed": true,
    "result": {
      "agent": "thermostat",
      "plan": "Thermostat",
      "planVariable": "pThermostat",
      "mode": "plain",
      "status": "completed",
      "executed": true,
      "wouldExecute": false,
      "output": true,
      "steps": [
        {
          "index": 0,
          "status": "completed",
          "result": true
        }
      ],
      "beliefsSet": {
        "currentTemp": 85,
        "lower": 70,
        "upper": 74
      },
      "diagnostics": {
        "logs": [],
        "logMessages": []
      }
    }
  }
}
```

The primary result is the structured plan run result. `diagnostics` contains runtime log entries and log messages for developer inspection. MCP callers should prefer `status`, `executed`, `reason`, `steps`, and `output` over parsing log text.

### Setting Structured Step Results

Inside a plan step, use `setStepResult(value)` to set the caller-facing result for that step. The executor records that value in `steps[n].result`; the last completed step result is also exposed as the top-level `output`.

`setPlanResult(value)` is available as an alias for the same behavior.

```chariot
declare(step1,'F', func(){
  setq(needCool, bigger(belief('thermostat','currentTemp'), belief('thermostat','upper')))
  if (needCool) {
    logPrint('turn A/C on')
    setStepResult(map('action', 'cooling_on', 'message', 'turn A/C on'))
  } else {
    logPrint('temp OK')
    setStepResult(map('action', 'none', 'message', 'temp OK'))
  }
})
```

If a step does not call `setStepResult`, the executor falls back to the step function's last expression value. That fallback is useful for simple plans, but MCP-facing agents should prefer explicit structured results.

Example skipped BDI result:

```json
{
  "call": {
    "id": "agent:thermostat",
    "kind": "agent",
    "action": "runplanonce",
    "executed": false,
    "result": {
      "agent": "thermostat",
      "plan": "Thermostat",
      "planVariable": "pThermostat",
      "mode": "bdi",
      "status": "skipped",
      "executed": false,
      "wouldExecute": false,
      "reason": "trigger_false",
      "steps": [],
      "beliefsSet": {
        "currentTemp": 70,
        "upper": 74
      },
      "diagnostics": {
        "logs": [],
        "logMessages": []
      }
    }
  }
}
```

Plan run statuses:

| Status | Meaning |
|--------|---------|
| `completed` | One or more steps executed and the plan finished. |
| `skipped` | Trigger or guard did not pass. Check `reason`. |
| `would_run` | `dry-run` mode passed trigger and guard but did not execute steps. |
| `dropped` | Drop condition became true before a step. |
| `failed` | Trigger, guard, drop, or a step returned an error. |
| `canceled` | Runtime context canceled during execution. |

Common reasons include `trigger_false`, `guard_false`, `drop_true`, `trigger_error`, `guard_error`, `drop_error`, `step_error`, and `canceled`.

---

## Program Tree MCP Calls

Use `treeList` to list saved program trees:

```json
{}
```

Example result:

```json
{
  "items": [
    {
      "id": "tree:decisionAgent1",
      "kind": "programTree",
      "name": "decisionAgent1",
      "path": "decisionAgent1.json",
      "callable": true
    }
  ],
  "count": 1
}
```

Describe a tree to discover callable function attributes:

```json
{
  "name": "decisionAgent1"
}
```

Callable ids have the form `tree:<tree>.<node>.<attribute>`.

Call a tree function:

```json
{
  "id": "tree:decisionAgent1.rules.ageFilter",
  "args": {
    "profile": {
      "age": 42
    }
  }
}
```

Example result:

```json
{
  "call": {
    "id": "tree:decisionAgent1.rules.ageFilter",
    "kind": "treeFunction",
    "action": "call",
    "result": true
  }
}
```

---

## Result Guidance

MCP tools return structured content when possible. For agent execution, treat the `call.result` object as the contract:

- Use `status` and `reason` to understand whether the plan ran.
- Use `steps` to inspect which steps completed or failed.
- Use `output` as the caller-facing value from the last completed step.
- Use `diagnostics.logs` and `diagnostics.logMessages` for developer troubleshooting only.

This distinction matters because logs are optimized for humans debugging Chariot plans, while MCP callers need stable fields that can be consumed programmatically.
