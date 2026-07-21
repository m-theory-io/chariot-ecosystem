# Chariot MCP Tools Registry
Yes. The next iteration should be a **Chariot MCP registry layer**, not just more one-off MCP tools.

Right now `go-chariot` MCP is intentionally thin: server.go exposes only `ping`, `execute`, and WIP `codeToDiagram`. Separately, Chariot already has two richer registries:

- **Runtime agent registry** in `chariot/plan_agent.go`: `DefaultAgentNames`, `DefaultAgentGetInfo`, `DefaultAgentBelief`, `DefaultAgentPublish`, etc.
- **Persisted tree/program assets** under `data/trees`, loadable through `treeLoad` / serializer APIs. decisionAgent1.json contains callable function-valued attributes under nodes like `rules`, `offer`, and `handlers`.

The main design question is whether MCP should expose each Chariot thing as its own MCP tool, or expose a stable generic registry API.

**Recommended: Generic Registry First**

Add a small fixed MCP API:

```text
registry.list
registry.describe
registry.call
agent.list
agent.call
tree.list
tree.describe
tree.call
```

The important one is `registry.list`, returning structured entries like:

```json
{
  "items": [
    {
      "id": "agent:thermostat",
      "kind": "agent",
      "name": "thermostat",
      "callable": true,
      "description": "Running Chariot BDI agent"
    },
    {
      "id": "tree:decisionAgent1",
      "kind": "programTree",
      "name": "decisionAgent1",
      "path": "decisionAgent1.json",
      "callable": true
    },
    {
      "id": "tree:decisionAgent1.handlers.approve",
      "kind": "treeFunction",
      "tree": "decisionAgent1",
      "node": "handlers",
      "attribute": "approve",
      "callable": true
    }
  ]
}
```

Then `registry.call` accepts:

```json
{
  "id": "tree:decisionAgent1.handlers.approve",
  "input": {
    "profile": { "age": 42, "debt": 3000, "is_employed": true },
    "req": {}
  },
  "mode": "bdi"
}
```

This avoids dynamically adding/removing MCP tools every time a tree file or agent changes. VS Code and other MCP clients cache tool lists; a stable generic MCP surface is easier to reason about.

**Option A: Runtime Registry Only**

Expose running agents and currently loaded plans/functions.

Pros:
- Fastest.
- Uses existing `DefaultAgentNames`, `DefaultAgentGetInfo`, `DefaultAgentBeliefs`.
- Good for “thermostat” if it is already running in the agent registry.

Cons:
- Does not discover persisted trees unless they are loaded into runtime.
- MCP server currently creates isolated runtimes in server.go, so it needs access to the real bootstrap/runtime state.

Implementation shape:
- Refactor MCP server construction to accept a runtime/registry provider:
  ```go
  type RegistryProvider interface {
      ListAgents() []RegistryItem
      GetAgentInfo(name string) map[string]any
      ListPlans() []RegistryItem
      CallRegistryItem(id string, args map[string]any) (any, error)
  }
  ```
- In REST/http mode, construct MCP using the same bootstrap runtime that handlers use.
- In stdio mode, build a bootstrap runtime and run bootstrap script, same as handlers.

**Option B: Persisted Tree Registry**

Scan `cfg.ChariotConfig.TreePath` and expose tree files as registry entries.

Pros:
- Makes things like `go-chariot/data/trees/decisionAgent1.json` discoverable even before runtime load.
- Natural fit for “program tree as callable artifact.”

Cons:
- Need rules for what “call tree” means.
- Need tree introspection to find function-valued attributes safely.

Implementation shape:
- `tree.list`: list `.json`, `.gob`, `.yaml`, `.xml` under tree storage.
- `tree.describe`: load tree and return:
  - root name
  - children names
  - function-valued attributes
  - offer variables
  - metadata
- `tree.call`: load tree, find a callable function attribute, execute it in a fresh runtime with provided arguments.

For decisionAgent1.json, the registry could expose function-valued attributes found under nodes such as `rules` and `offer`. Function attributes already serialize with `_value_type: "function"` and parameter metadata, so the registry can infer a useful input schema.

**Option C: Dynamic MCP Tools Per Entry**

Materialize MCP tools like:

```text
agent_thermostat_call
tree_decisionAgent1_call
tree_decisionAgent1_handlers_approve
```

Pros:
- Very nice in AI agent UIs: tools appear directly.
- Clear call names.

Cons:
- MCP tool list churn.
- Names need sanitization and collision handling.
- Clients may need “Reset Cached Tools” after every tree/agent change.
- Harder to scale.

I would not start here. It is a good later feature once the generic registry is stable.

**Recommended Architecture**

Use a two-layer model:

1. **Internal Chariot Registry Service**
   - Knows about agents, plans, function library, tree files, tree callables.
   - Pure Go service, testable without MCP.
   - Returns normalized `RegistryItem` records.

2. **MCP Adapter**
   - Static MCP tools:
     - `registry.list`
     - `registry.describe`
     - `registry.call`
     - `agent.list`
     - `agent.publish`
     - `agent.setBelief`
     - `tree.list`
     - `tree.describe`
   - Delegates all business logic to the registry service.

This keeps MCP as a protocol adapter, not the owner of Chariot semantics.

**Callable Semantics**

For agents:

```json
{
  "id": "agent:thermostat",
  "action": "setBelief",
  "input": {
    "currentTemp": 71,
    "targetTemp": 68
  }
}
```

or:

```json
{
  "id": "agent:thermostat",
  "action": "publish"
}
```

For plans:

```json
{
  "id": "plan:thermostatControl",
  "mode": "bdi",
  "vars": {
    "currentTemp": 71,
    "targetTemp": 68
  }
}
```

For program trees:

```json
{
  "id": "tree:decisionAgent1",
  "action": "evaluate",
  "input": {
    "profile": {
      "age": 44,
      "debt": 2000,
      "is_employed": true
    },
    "req": {}
  }
}
```

For specific tree functions:

```json
{
  "id": "tree:decisionAgent1.rules.ageFilter",
  "args": {
    "profile": { "age": 44 }
  }
}
```

**Key Implementation Choice**

The biggest root-cause change is this: MCP needs access to the same meaningful runtime state as the REST/Charioteer side.

Currently `execute` in server.go creates a new runtime per call:

```go
rt := chariot.NewRuntime()
chariot.RegisterAll(rt)
```

That is fine for `add(1,2)`, but not enough for registered agents, bootstrap plans, or loaded trees. For registry work, MCP should be constructed with:

- bootstrap runtime
- tree storage paths
- agent registry access
- optional session/user context later

**Suggested Milestones**

1. **MVP Registry**
   - Add `registry.list`.
   - Return running agents from `DefaultAgentNames`.
   - Return tree files from `cfg.ChariotConfig.TreePath`.
   - No dynamic tools yet.

2. **Tree Describe**
   - Load tree files.
   - Return root/children/function attributes.
   - Include inferred callable IDs.

3. **Registry Call**
   - Support:
     - `agent:<name>` actions: `publish`, `setBelief`, `getBeliefs`, `info`
     - `tree:<name>.<node>.<attribute>` function call
   - Return structured output.

4. **Runtime-Aware MCP**
   - Refactor MCP server construction to receive bootstrap runtime in REST/http mode.
   - Preserve stdio by bootstrapping its own runtime.

5. **Optional Dynamic Tools**
   - Generate MCP tools from registry entries.
   - Add a cache/version mechanism and docs telling clients when to reset cached tools.

My recommendation: implement milestones 1-3 with generic registry tools first. That gives Chariot agents and program trees real discoverability/callability over MCP without fighting MCP client tool caching.