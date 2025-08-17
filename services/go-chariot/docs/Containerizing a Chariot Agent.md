# Containerizing a Chariot Agent

## 1. Container Structure

**The container image should include:**
- The Chariot Server binary (built for Linux, e.g., `/usr/local/bin/chariot-server`)
- The agent file (e.g., `agent.json` or `agent.secure`)
- An `onStart` Chariot script (either as a file or via env var)
- A default entrypoint script (e.g., `/entrypoint.sh`)

**Environment variables:**
- `CHARIOT_HEADLESS=true`
- `CHARIOT_ON_START` (or path to onStart script)
- `CHARIOT_AGENT_FILE=agent.json` (or `agent.secure`)
- `CHARIOT_AGENT_NAME=MyAgent`
- (Other config as needed, e.g., port, keys, etc.)

**Example Dockerfile:**
```dockerfile
FROM ubuntu:22.04

# Copy Chariot server binary
COPY chariot-server /usr/local/bin/chariot-server

# Copy agent and scripts
COPY agent.json /app/agent.json
COPY onstart.chariot /app/onstart.chariot

# Set environment variables
ENV CHARIOT_HEADLESS=true
ENV CHARIOT_AGENT_FILE=/app/agent.json
ENV CHARIOT_ON_START_FILE=/app/onstart.chariot

# Entrypoint
ENTRYPOINT ["/usr/local/bin/chariot-server"]
```

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
  Dockerfile
```

**Build and push:**
```sh
docker build -t myorg/chariot-agent:latest .
docker push myorg/chariot-agent:latest
```

**Run (locally or in cloud):**
```sh
docker run -p 8080:8080 \
  -e CHARIOT_HEADLESS=true \
  -e CHARIOT_AGENT_FILE=/app/agent.json \
  -e CHARIOT_ON_START_FILE=/app/onstart.chariot \
  myorg/chariot-agent:latest
```

---

## 6. Next Steps & Extensions

- **Handler Routing:** Extend `listen` to support multiple endpoints and HTTP verbs.
- **Secure Agent Loading:** Use `treeLoadSecure` if agent is encrypted.
- **Health Checks:** Add a default `onHealthCheck` handler.
- **Registry Integration:** Automate image build and push to Azure Container Registry or DockerHub.
- **API Gateway:** Optionally front with API gateway for auth/rate limiting.

---

## Summary

- **Agent is containerized** with Chariot server, agent tree, and startup script.
- **Handlers** are defined as functions in a `handlers` node, referenced by name.
- **onStart** script loads the agent and sets up listeners.
- **Requests** are routed to handler functions via Chariot’s `call`.

---
