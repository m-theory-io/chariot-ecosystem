## Example: Agent Handlers Node

**Chariot code to define agent handlers:**
```chariot
// Create handlers node
setq(handlers, create('handlers'))

// Decision request handler
setAttribute(handlers, 'onDecisionRequest', func(req) {
    let input = getProp(req, "input")
    let rules = getChildAt(agent, 1)
    let ageFilter = getAttribute(rules, 'ageFilter')
    let debtFilter = getAttribute(rules, 'debtFilter')
    let employmentFilter = getAttribute(rules, 'employmentFilter')
    let result = and(
        call(ageFilter, input),
        call(debtFilter, input),
        call(employmentFilter, input)
    )
    return map("approved", result)
})

// Health check handler
setAttribute(handlers, 'onHealthCheck', func(req) {
    return map("status", "ok")
})

// Attach handlers to agent
addChild(agent, handlers)
```

---

## Example: Full Dockerfile

```dockerfile
FROM ubuntu:22.04

# Install dependencies if needed (e.g., ca-certificates)
RUN apt-get update && apt-get install -y ca-certificates && rm -rf /var/lib/apt/lists/*

# Copy Chariot server binary
COPY chariot-server /usr/local/bin/chariot-server

# Copy agent and scripts
COPY agent.json /app/agent.json
COPY onstart.chariot /app/onstart.chariot

# Set environment variables
ENV CHARIOT_HEADLESS=true
ENV CHARIOT_AGENT_FILE=/app/agent.json
ENV CHARIOT_ON_START_FILE=/app/onstart.chariot

WORKDIR /app

# Entrypoint
ENTRYPOINT ["/usr/local/bin/chariot-server"]
```

---

## Example: onStart Script with Routing

```chariot
// Load the agent
setq(agentNode, treeLoad(getEnv("CHARIOT_AGENT_FILE")))
declareGlobal(agent, 'T', agentNode)

// Start HTTP listener with multiple handlers
listen(8080, "routeRequest")

// Route requests by path
function routeRequest(req) {
    let path = getProp(req, "path")
    let handlers = getChildByName(agent, 'handlers')
    if(equal(path, "/decision")) {
        let handler = getAttribute(handlers, 'onDecisionRequest')
        return call(handler, req)
    } else if(equal(path, "/health")) {
        let handler = getAttribute(handlers, 'onHealthCheck')
        return call(handler, req)
    } else {
        return map("error", "Unknown endpoint")
    }
}
```

---

## Usage Summary

- **Handlers** are functions stored in a `handlers` node, attached to the agent.
- **onStart** script loads the agent and sets up a listener.
- **Requests** are routed by the main handler (e.g., `routeRequest`) to the appropriate function.
- **Dockerfile** ensures all components are present and environment is set for headless execution.

---

