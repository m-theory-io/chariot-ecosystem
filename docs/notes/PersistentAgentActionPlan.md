Short rationale
Your Plan tests run a one-shot helper. For production you want long-lived Agents that run Plans continuously, accept belief updates via REST/NSQ, auto-start at server boot, and surface status in Charioteer. To keep interfaces canonical and avoid shims, introduce a small AgentManager in go-chariot, expose agentStart/agentStop/beliefUpdate as Chariot functions and REST, wire optional NSQ ingestion, auto-load agents from a data/agents.yaml at startup, and stream agent progress to the Dashboard over WS.

Proposed changes (step-by-step)

1) Add an AgentManager (lifecycle, concurrency, progress)
- Single scheduler goroutine; one goroutine per active intention; bounded via a semaphore.
- Context-cancel on stop/drop; no blocking I/O under locks.
- Progress events emitted to subscribers (WS) and log.

````go
package agent

import (
	"context"
	"sync"
	"time"

	ch "github.com/bhouse1273/chariot-ecosystem/services/go-chariot/chariot"
)

type ProgressSink interface {
	OnAgentStarted(name string)
	OnAgentStopped(name string, reason string)
	OnPlanStarted(agent string)
	OnStepStarted(agent string, idx int, label string)
	OnStepFinished(agent string, idx int, err error)
	OnPlanFinished(agent string)
}

type Options struct {
	MaxConcurrent int
	PollEvery     time.Duration
	Jitter        time.Duration
}

type Manager struct {
	mu      sync.RWMutex
	agents  map[string]*runner
	sinks   []ProgressSink
	opts    Options
}

func NewManager(opts Options) *Manager {
	return &Manager{
		agents: make(map[string]*runner),
		opts:   opts,
	}
}

func (m *Manager) AddSink(s ProgressSink) { m.mu.Lock(); m.sinks = append(m.sinks, s); m.mu.Unlock() }

func (m *Manager) Start(ctx context.Context, name string, rt *ch.Runtime, pl *ch.Plan) error {
	m.mu.Lock()
	if _, exists := m.agents[name]; exists {
		m.mu.Unlock()
		return nil
	}
	r := newRunner(name, rt, pl, m.opts, m.broadcast)
	m.agents[name] = r
	m.mu.Unlock()
	m.broadcast(func(s ProgressSink) { s.OnAgentStarted(name) })
	go r.run(ctx)
	return nil
}

func (m *Manager) Stop(name string, reason string) {
	m.mu.Lock()
	r := m.agents[name]
	delete(m.agents, name)
	m.mu.Unlock()
	if r != nil {
		r.stop(reason)
		m.broadcast(func(s ProgressSink) { s.OnAgentStopped(name, reason) })
	}
}

func (m *Manager) List() []string {
	m.mu.RLock(); defer m.mu.RUnlock()
	out := make([]string, 0, len(m.agents))
	for k := range m.agents { out = append(out, k) }
	return out
}

func (m *Manager) PublishBelief(agent string, key string, val ch.Value) {
	m.mu.RLock(); r := m.agents[agent]; m.mu.RUnlock()
	if r != nil {
		r.publishBelief(key, val)
	}
}

func (m *Manager) broadcast(fn func(ProgressSink)) {
	m.mu.RLock(); sinks := append([]ProgressSink(nil), m.sinks...); m.mu.RUnlock()
	for _, s := range sinks { fn(s) }
}
````

````go
package agent

import (
	"context"
	"time"

	ch "github.com/bhouse1273/chariot-ecosystem/services/go-chariot/chariot"
)

type runner struct {
	name   string
	rt     *ch.Runtime
	pl     *ch.Plan
	opts   Options
	events chan beliefEvent
	cancel context.CancelFunc
	signal func(func(ProgressSink))
}

type beliefEvent struct{ key string; val ch.Value }

func newRunner(name string, rt *ch.Runtime, pl *ch.Plan, opts Options, signal func(func(ProgressSink))) *runner {
	return &runner{
		name:   name,
		rt:     rt,
		pl:     pl,
		opts:   opts,
		events: make(chan beliefEvent, 128),
		signal: signal,
	}
}

func (r *runner) run(parent context.Context) {
	ctx, cancel := context.WithCancel(parent)
	r.cancel = cancel
	t := time.NewTicker(r.opts.PollEvery); defer t.Stop()
	sem := make(chan struct{}, max(1, r.opts.MaxConcurrent))
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			r.trySchedule(ctx, sem)
		case <-r.events:
			r.trySchedule(ctx, sem)
		}
	}
}

func (r *runner) trySchedule(ctx context.Context, sem chan struct{}) {
	// Evaluate trigger/guard quickly using plan’s functions within runtime
	ok := ch.PlanTriggerOK(r.rt, r.pl, ctx)
	guard := ch.PlanGuardOK(r.rt, r.pl, ctx)
	if !ok || !guard { return }
	select {
	case sem <- struct{}{}:
		go func() {
			defer func(){ <-sem }()
			r.signal(func(s ProgressSink){ s.OnPlanStarted(r.name) })
			ch.RunPlanSteps(r.rt, r.pl, ctx, func(i int, label string){ r.signal(func(s ProgressSink){ s.OnStepStarted(r.name,i,label) }) },
				func(i int, err error){ r.signal(func(s ProgressSink){ s.OnStepFinished(r.name,i,err) }) },
			)
			r.signal(func(s ProgressSink){ s.OnPlanFinished(r.name) })
		}()
	default:
		// saturated; skip
	}
}

func (r *runner) publishBelief(key string, val ch.Value) {
	select { case r.events <- beliefEvent{key,val}: default: /* drop or coalesce */ }
}

func (r *runner) stop(reason string) { if r.cancel != nil { r.cancel() } }

func max(a,b int) int { if a>b { return a }; return b }
````

Notes
- The runner delegates Plan-specific evaluation to ch.PlanTriggerOK/PlanGuardOK/RunPlanSteps you already own or we add next to your existing plan(...) support, keeping one canonical API.

2) Expose Chariot functions for runtime code
- Canonical names: agentStart, agentStop, agentBelief, agentList.

````go
package chariot

import (
	"context"
	"time"

	"github.com/bhouse1273/chariot-ecosystem/services/go-chariot/chariot/agent"
)

var DefaultAgentMgr = agent.NewManager(agent.Options{
	MaxConcurrent: 4,
	PollEvery:     750 * time.Millisecond,
	Jitter:        0,
})

func init() {
	RegisterFunction("agentStart", func(rt *Runtime, args ...Value) (Value, error) {
		// args: name 'S', plan 'P'
		name := stringFrom(args, 0)
		pl := planFrom(args, 1)
		ctx := context.Background()
		return True, DefaultAgentMgr.Start(ctx, name, rt, pl)
	})
	RegisterFunction("agentStop", func(rt *Runtime, args ...Value) (Value, error) {
		DefaultAgentMgr.Stop(stringFrom(args,0), "user-request")
		return True, nil
	})
	RegisterFunction("agentBelief", func(rt *Runtime, args ...Value) (Value, error) {
		DefaultAgentMgr.PublishBelief(stringFrom(args,0), stringFrom(args,1), args[2])
		return True, nil
	})
	RegisterFunction("agentList", func(rt *Runtime, args ...Value) (Value, error) {
		return ArrayFromStrings(DefaultAgentMgr.List()), nil
	})
}
````

3) REST + WS for Agents
- Add routes under /api/agents and a WS topic /ws/agents.

````go
package handlers

import (
	"net/http"

	"github.com/labstack/echo/v4"
	ch "github.com/bhouse1273/chariot-ecosystem/services/go-chariot/chariot"
)

type AgentStartReq struct {
	Name string `json:"name"`
	Plan string `json:"plan"` // variable name in server runtime
}

func (h *Handlers) StartAgent(c echo.Context) error {
	var req AgentStartReq
	if err := c.Bind(&req); err != nil { return c.JSON(http.StatusBadRequest, Err(err)) }
	rt := h.ServerRuntime // the long-lived server runtime
	plV, ok := rt.MustGetVariable(req.Plan).(*ch.Plan)
	if !ok { return c.JSON(http.StatusBadRequest, Errf("plan '%s' not found", req.Plan)) }
	if err := ch.DefaultAgentMgr.Start(c.Request().Context(), req.Name, rt, plV); err != nil {
		return c.JSON(http.StatusBadRequest, Err(err))
	}
	return c.JSON(http.StatusOK, OK(map[string]any{"name": req.Name}))
}

func (h *Handlers) StopAgent(c echo.Context) error {
	name := c.Param("name")
	ch.DefaultAgentMgr.Stop(name, "api")
	return c.JSON(http.StatusOK, OK(nil))
}

func (h *Handlers) ListAgents(c echo.Context) error {
	return c.JSON(http.StatusOK, OK(map[string]any{"agents": ch.DefaultAgentMgr.List()}))
}
````

````go
// ...existing code...
// inside route wiring:
e.GET("/api/agents", h.ListAgents)
e.POST("/api/agents/start", h.StartAgent)
e.POST("/api/agents/:name/stop", h.StopAgent)
// WS stream can be added similarly on /ws/agents using DefaultAgentMgr.AddSink(...)
````

4) Auto-start agents on boot from data/agents.yaml
- Read a YAML in the data path to declare which plans to run on startup.

````yaml
agents:
  - name: PreventAuthDenials
    runtime: server         # server/global runtime
    planVar: PreventAuthDenials   # variable name holding a 'P'
````

````go
// ... after ServerRuntime initialized and bootstrap executed:
if err := handlers.LoadAgentsFromFile(h, filepath.Join(cfg.ChariotConfig.DataPath, "agents.yaml")); err != nil {
    logger.Warn("No agents.yaml or failed to load", "error", err)
}
````

````go
package handlers

import (
	"os"

	"gopkg.in/yaml.v3"
	ch "github.com/bhouse1273/chariot-ecosystem/services/go-chariot/chariot"
)

type agentsCfg struct {
	Agents []struct {
		Name    string `yaml:"name"`
		Runtime string `yaml:"runtime"`
		PlanVar string `yaml:"planVar"`
	} `yaml:"agents"`
}

func LoadAgentsFromFile(h *Handlers, path string) error {
	b, err := os.ReadFile(path)
	if err != nil { return err }
	var cfg agentsCfg
	if err := yaml.Unmarshal(b, &cfg); err != nil { return err }
	for _, a := range cfg.Agents {
		rt := h.ServerRuntime // extend for multiple runtimes if needed
		if v, ok := rt.MustGetVariable(a.PlanVar).(*ch.Plan); ok {
			_ = ch.DefaultAgentMgr.Start(h.Ctx, a.Name, rt, v)
		}
	}
	return nil
}
````

5) NSQ ingestion (optional, configurable)
- If enabled, consume a topic and publish belief updates.

````go
package nsq

import (
	"encoding/json"

	"github.com/nsqio/go-nsq"
	ch "github.com/bhouse1273/chariot-ecosystem/services/go-chariot/chariot"
)

type BeliefMsg struct {
	Agent string      `json:"agent"`
	Key   string      `json:"key"`
	Value interface{} `json:"value"`
}

func StartBeliefConsumer(addr, topic, channel string) (*nsq.Consumer, error) {
	cfg := nsq.NewConfig()
	c, err := nsq.NewConsumer(topic, channel, cfg)
	if err != nil { return nil, err }
	c.AddHandler(nsq.HandlerFunc(func(m *nsq.Message) error {
		var bm BeliefMsg
		if err := json.Unmarshal(m.Body, &bm); err != nil { return nil }
		ch.DefaultAgentMgr.PublishBelief(bm.Agent, bm.Key, ch.FromGoValue(bm.Value))
		return nil
	}))
	return c, c.ConnectToNSQD(addr)
}
````

6) Charioteer Dashboard
- Add Agents panel: list agents via GET /api/agents; subscribe to /ws/agents for progress; render statuses.
- If you want, I’ll patch main.go accordingly once you confirm the HTML section location.

7) Tests
- Keep your one-shot runPlanOnce tests for Plan semantics.
- Add Agent lifecycle tests:
  - Start a test agent with a simple plan; publish a belief; assert step execution; stop; assert goroutine leak-free (use goleak in tests).
  - REST round-trip test to start/stop/list.

Ask before patch
To avoid guessing file boundaries, confirm:
- Where the long-lived “server” Runtime is held in Handlers (field name/path).
- Your preferred YAML filename and location under data.
- Whether NSQ is already wired in the project (addresses/env keys) or you want the minimal consumer scaffold above.

Once confirmed, I’ll commit the exact patches in the shown file paths and add a small Dashboard section to display agents.