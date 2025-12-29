# Containerizing a Chariot Agent

Chariot agents can be published in two container flavors:

- **Production** – minimal surface area; only the headless runtime and any listeners the agent configures are exposed.
- **Debug** – the same headless runtime plus the Dev REST + debugger surface that Charioteer/Visual-DSL rely on.

### Profiles at a Glance

| Profile   | Primary Use Case                         | Ports to Expose                 | Key Env Overrides                                      | Notes |
|-----------|------------------------------------------|---------------------------------|--------------------------------------------------------|-------|
| Production| Deploy the agent as an API/microservice  | Listener port(s) declared via `listen()` (e.g., 8080) | `CHARIOT_HEADLESS=true`, `CHARIOT_DEV_REST_ENABLED=false` | Keeps attack surface small; no IDE endpoints. |
| Debug     | Live-inspect plans via Charioteer/Visual-DSL | Listener port(s) **and** Dev REST port (default 8090) | `CHARIOT_HEADLESS=true`, `CHARIOT_DEV_REST_ENABLED=true`, optional `CHARIOT_PORT=8090` | Ideal for QA or building “publish container” previews. |

Both images share the same agent artifacts; the entrypoint toggles the appropriate environment before launching `chariot-server`.

## 1. Container Structure

**Both profiles ship the same artifacts:**
- The Chariot Server binary (built for Linux, e.g., `/usr/local/bin/chariot-server`)
- The agent file (e.g., `agent.json` or `agent.secure`)
- An `onStart` Chariot script (either as a file or via env var)
- A default entrypoint script (e.g., `/entrypoint.sh`)

**Common environment variables:**
- `CHARIOT_AGENT_FILE=/app/agent.json` (or `agent.secure`)
- `CHARIOT_ON_START_FILE=/app/onstart.chariot` (or inline via `CHARIOT_ON_START`)
- `CHARIOT_AGENT_NAME=MyAgent`
- Any secrets, listener keys, or data roots the agent expects

**Profile-specific overrides:**

| Profile | Required Flags | Typical Port Exposure |
|---------|----------------|-----------------------|
| Production | `CHARIOT_HEADLESS=true`, `CHARIOT_DEV_REST_ENABLED=false` | Only the listener(s) started via your `listen()` calls (e.g., 8080) |
| Debug | `CHARIOT_HEADLESS=true`, `CHARIOT_DEV_REST_ENABLED=true`, optionally `CHARIOT_PORT=8090` for Dev REST/IDE traffic | Listener ports **plus** the Dev REST/debugger port (default 8090) |

In debug images you usually keep the same listener port so QA can hit the API while simultaneously attaching Charioteer/Visual-DSL through the Dev REST surface.

**Example Dockerfiles:** use the same build context but bake in the profile-specific defaults.

`Dockerfile.prod`
```dockerfile
FROM ubuntu:22.04

COPY chariot-server /usr/local/bin/chariot-server
COPY agent.json /app/agent.json
COPY onstart.chariot /app/onstart.chariot
COPY entrypoint.sh /entrypoint.sh

ENV CHARIOT_HEADLESS=true \
    CHARIOT_DEV_REST_ENABLED=false \
    CHARIOT_AGENT_FILE=/app/agent.json \
    CHARIOT_ON_START_FILE=/app/onstart.chariot

EXPOSE 8080
ENTRYPOINT ["/entrypoint.sh"]
```

`Dockerfile.debug`
```dockerfile
FROM ubuntu:22.04

COPY chariot-server /usr/local/bin/chariot-server
COPY agent.json /app/agent.json
COPY onstart.chariot /app/onstart.chariot
COPY entrypoint.sh /entrypoint.sh

ENV CHARIOT_HEADLESS=true \
    CHARIOT_DEV_REST_ENABLED=true \
    CHARIOT_AGENT_FILE=/app/agent.json \
    CHARIOT_ON_START_FILE=/app/onstart.chariot \
    CHARIOT_PORT=8090

EXPOSE 8080 8090
ENTRYPOINT ["/entrypoint.sh"]
```

Both Dockerfiles call the same entrypoint so you can also collapse to a single file and switch via build args if preferred.

**Entrypoint toggle example (`entrypoint.sh`):**
```sh
#!/usr/bin/env sh
set -euo pipefail

PROFILE=${CHARIOT_PROFILE:-production}

case "${PROFILE}" in
  debug)
    export CHARIOT_DEV_REST_ENABLED=true
    : "${CHARIOT_PORT:=8090}"
    ;;
  *)
    export CHARIOT_DEV_REST_ENABLED=false
    ;;
esac

exec /usr/local/bin/chariot-server "$@"
```
Set `CHARIOT_PROFILE=debug` at runtime (or bake it into the debug image) to enable the IDE/debugger surface while leaving production containers untouched.

---

## 2. onStart Script Example

**onstart.chariot:**
```chariot
// Load the agent tree from file
setq(agentNode, treeLoad(getEnv("CHARIOT_AGENT_FILE")))

// Optionally register agentNode globally
declareGlobal(agent, 'T', agentNode)

// Start HTTP listener for decision requests
listen(8080, "onDecisionRequest")
```

---

## 3. Agent Handler Design

To make the agent extensible and able to handle incoming requests, add a `handlers` JSONNode (or child node) to the agent tree. Each handler is a named function.

**Agent Structure Example:**
```chariot
// Agent tree structure
agent
 ├── profile
 ├── rules
 └── handlers
      ├── onDecisionRequest: func(req) { ... }
      └── onHealthCheck: func(req) { ... }
```

**How to define handlers:**
```chariot
setq(handlers, create('handlers'))
setAttribute(handlers, 'onDecisionRequest', func(req) {
    let input = getProp(req, "input")
    let rules = getChildAt(agent, 1)
    // ...run rules, return result...
    return map("result", true)
})
addChild(agent, handlers)
```

---

## 4. Handler Dispatch Logic

**In your onStart or listener handler:**
```chariot
function onDecisionRequest(req) {
    // Load agent and handlers
    let agentNode = getGlobal('agent')
    let handlers = getChildByName(agentNode, 'handlers')
    let handler = getAttribute(handlers, 'onDecisionRequest')
    // Call the handler with the request
    return call(handler, req)
}
```
- The `listen` function binds the port and associates `"onDecisionRequest"` with incoming HTTP POSTs (or similar).
- You can extend this to route by path or action.

---

## 5. Example: Minimal Agent Container

**Directory structure:**
```
/app/
  chariot-server
  agent.json
  onstart.chariot
  entrypoint.sh
  Dockerfile.prod
  Dockerfile.debug
```

**Build and push:**
```sh
docker build -f Dockerfile.prod -t myorg/chariot-agent:prod .
docker build -f Dockerfile.debug -t myorg/chariot-agent:debug .
docker push myorg/chariot-agent:prod
docker push myorg/chariot-agent:debug
```

**Run (locally or in cloud):**

Production profile:
```sh
docker run -p 8080:8080 \
  -e CHARIOT_PROFILE=production \
  myorg/chariot-agent:prod
```

Debug profile (listener + Dev REST/IDE):
```sh
docker run -p 8080:8080 -p 8090:8090 \
  -e CHARIOT_PROFILE=debug \
  myorg/chariot-agent:debug
```
The `CHARIOT_PROFILE` toggle feeds the entrypoint script shown earlier; if you baked the values directly into separate images, you can omit it.

---

## 6. Next Steps & Extensions

- **Handler Routing:** Extend `listen` to support multiple endpoints and HTTP verbs.
- **Secure Agent Loading:** Use `treeLoadSecure` if agent is encrypted.
- **Health Checks:** Add a default `onHealthCheck` handler.
- **Registry Integration:** Automate image build and push to Azure Container Registry or DockerHub.
- **Publish CLI:** Wire a `chariot publish-container --profile prod|debug` helper (or CI workflow) so both images stay in sync with the same agent artifact bundle.
- **API Gateway:** Optionally front with API gateway for auth/rate limiting.

---

## Summary

- **Base image** bundles the Chariot server, agent tree, and startup script exactly once.
- **Production profile** keeps only headless/listener surfaces alive; **debug profile** flips on the Dev REST + debugger stack so IDE tooling can attach.
- **Handlers** still live under a `handlers` node and are dispatched from listener callbacks or Dev REST invocations.
- **onStart** seeds listeners (and optional belief/bootstrap data) so both profiles behave identically once traffic arrives.
- **Requests** are routed to handler functions via Chariot’s `call`, whether they originate from your published API, MCP tooling, or the debugger.

---
