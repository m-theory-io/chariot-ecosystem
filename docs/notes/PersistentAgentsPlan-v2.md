GitHub Copilot

Rationale
To deliver long‑lived, named Agents that run Plans, accept beliefs via REST/NSQ, auto-start on boot, and surface work progress in Charioteer, we need a small, canonical Agent manager in go-chariot, REST WS endpoints, optional NSQ ingestion, and Dashboard wiring. Below is a concrete, minimal checklist by area.

Remaining TODOs

Go: Agent core and functions
- plan_agent.go
  - [ ] Finalize name-based registry (defaultAgents) with:
    - Start(name, rt, plan, maxConcurrent, pollEvery)
    - Stop(name)
    - Get(name)
    - List()
  - [ ] Agent state
    - events chan (scheduler nudge), sem (bounded concurrency)
    - beliefs map + SetBelief/GetBelief
    - context cancel on stop
  - [ ] Scheduler
    - Periodic tick + event-driven (belief updates)
    - Evaluate trigger/guard outside locks
    - Run steps with ctx and pre-step drop checks
  - [ ] Chariot functions (registered once in init/RegisterPlanFunctions)
    - agentStartNamed(name, plan[, maxConcurrent][, pollSec])
    - agentStopNamed(name)
    - agentList()
    - agentPublish(name)
    - agentBelief(name, key, value)
    - belief(name, key)
  - [ ] Keep legacy agentNew/agentRegister/agentStart/agentStop working; mark deprecated in comments.

Go: REST WS surface
- services/go-chariot/handlers
  - [ ] Routes
    - GET /api/agents → list
    - POST /api/agents/start {name, planVar, maxConcurrent?, pollSec?}
    - POST /api/agents/:name/stop
    - POST /api/agents/:name/publish
    - PUT /api/agents/:name/beliefs {key, value}
  - [ ] WS stream
    - GET /ws/agents → upgrades to WS; broadcasts progress events (agent started/stopped, plan started/finished, step started/finished)
  - [ ] Auth: match existing /api protection (oauth2-proxy/headers).
- main.go
  - [ ] Wire new routes
  - [ ] StopAll agents on server shutdown (graceful)
  - [ ] Load agents on boot (see Config below)

Config and bootstrap
- main.go (config binding)
  - [ ] Add envs (with sane defaults):
    - CHARIOT_AGENT_MAX_CONCURRENCY (int, default 4)
    - CHARIOT_AGENT_POLL_MS (int, default 750)
    - CHARIOT_NSQ_ENABLED (bool)
    - CHARIOT_NSQ_ADDR, CHARIOT_NSQ_TOPIC, CHARIOT_NSQ_CHANNEL
  - [ ] Load startup agents
    - Option A: data/agents.yaml with entries: {name, planVar, maxConcurrent?, pollSec?}
    - Option B: bootstrap.ch calls agentStartNamed(...)
  - [ ] Call loader after bootstrap so plan variables exist.
- nginx.conf
  - [ ] Proxy WS for /ws/agents with Upgrade/Connection headers.

NSQ ingestion (optional)
- services/go-chariot/internal/nsq/beliefs.go
  - [ ] Consumer: parse {"agent","key","value"} and call defaultAgents.PublishBelief or agentBelief via runtime
  - [ ] Start consumer when CHARIOT_NSQ_ENABLED=true
- docs/README.md
  - [ ] Document envs and message format

Charioteer (Dashboard + chat progress)
- main.go
  - [ ] Dashboard panel: Agents
    - List via GET /api/agents
    - Start/Stop buttons (optional for MVP)
    - Belief update form (agent, key, value → PUT /api/agents/:name/beliefs)
  - [ ] Progress stream
    - Open WS wss://.../ws/agents
    - Append progress lines to the chat Output and restore “work in progress” indicators
    - Basic structure: {type:"agent|plan|step", agent, index?, label?, status:"start|finish|drop", error?}
  - [ ] Dev proxy (vite) add ws: true for /ws/agents (Visual DSL dev server too if shared)
- UX polish
  - [ ] Dirty indicator when edited code diverges from last saved diagram code (optional)
  - [ ] Agent status badges (running/stopped)

Tests
- services/go-chariot/tests
  - [ ] plan_agent_test.go:
    - Start agent with simple plan; publish belief; assert step executed
    - Stop agent; ensure no goroutine leaks (use goleak or timeout-based assertions)
  - [ ] REST roundtrips:
    - Start via POST, list, publish belief, stop
  - [ ] WS:
    - Subscribe to /ws/agents; run a short plan; assert receipt of plan/step events
  - [ ] NSQ (if enabled):
    - Spin up nsqd in test or fake producer; send belief; assert agent receives

Observability and safety
- [ ] Log plan/step transitions at debug; errors at warn
- [ ] No blocking I/O under locks; step execution never holds Agent mutex
- [ ] All waits/selects honor ctx.Done()
- [ ] Backpressure: bounded channels; drop or coalesce non-critical events
- [ ] Metrics (optional): counters for plans started/finished, steps run, drops

Docs
- ChariotPlanExtensions.md
  - [ ] Update with name-based API, REST WS, NSQ options, startup agents, examples
- README
  - [ ] Quickstart: define plan var in bootstrap, agentStartNamed via REST, watch progress in Dashboard

Migration notes
- Existing code using agentNew/agentStart still works; prefer agentStartNamed going forward.
- For auto-start, migrate prior ad hoc loops to agents.yaml or bootstrap.ch with agentStartNamed.

If you want, I can start by:
- Patching plan_agent.go to finalize registry + belief functions.
- Adding REST endpoints and a simple WS broadcaster.
- Wiring Dashboard to display progress again.