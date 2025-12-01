package chariot

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"
)

// AgentEvent is emitted on plan/step lifecycle transitions for dashboards/clients.
type AgentEvent struct {
	Type   string    `json:"type"` // "plan" | "step"
	Agent  string    `json:"agent"`
	Plan   string    `json:"plan"`
	Step   int       `json:"step,omitempty"`
	Status string    `json:"status"` // start|finish|drop|error|cancel
	Error  string    `json:"error,omitempty"`
	Time   time.Time `json:"time"`
}

var (
	agentEventMu    sync.RWMutex
	agentEventSinks = map[chan AgentEvent]struct{}{}
)

// RegisterAgentEventSink registers a channel to receive AgentEvent notifications.
// Call the returned function to unregister.
func RegisterAgentEventSink(ch chan AgentEvent) func() {
	agentEventMu.Lock()
	agentEventSinks[ch] = struct{}{}
	agentEventMu.Unlock()
	return func() {
		agentEventMu.Lock()
		delete(agentEventSinks, ch)
		agentEventMu.Unlock()
	}
}

func broadcastAgentEvent(ev AgentEvent) {
	agentEventMu.RLock()
	for ch := range agentEventSinks {
		select {
		case ch <- ev:
		default: /* drop on slow consumer */
		}
	}
	agentEventMu.RUnlock()
}

// Plan represents a first-class BDI plan constructed via plan(...)
type Plan struct {
	Name    string
	Params  []string
	Trigger *FunctionValue
	Guard   *FunctionValue
	Steps   []*FunctionValue
	Drop    *FunctionValue
}

func (p *Plan) String() string {
	if p == nil {
		return "<nil plan>"
	}
	return fmt.Sprintf("Plan(%s)", p.Name)
}

// rebindPlanToRuntime returns a copy of p whose closures point at rt’s global scope.
// This preserves lexical scoping when moving a plan from a bootstrap runtime to a per-agent runtime.
func rebindPlanToRuntime(p *Plan, rt *Runtime) *Plan {
	if p == nil || rt == nil {
		return nil
	}
	g := rt.GlobalScope()
	cp := &Plan{
		Name:    p.Name,
		Params:  append([]string(nil), p.Params...),
		Trigger: cloneFunctionValueWithScope(p.Trigger, g),
		Guard:   cloneFunctionValueWithScope(p.Guard, g),
		Drop:    cloneFunctionValueWithScope(p.Drop, g),
	}
	if len(p.Steps) > 0 {
		cp.Steps = make([]*FunctionValue, len(p.Steps))
		for i, s := range p.Steps {
			cp.Steps[i] = cloneFunctionValueWithScope(s, g)
		}
	}
	return cp
}

// Agent coordinates plan execution with bounded concurrency
type Agent struct {
	name      string
	rt        *Runtime
	mu        sync.RWMutex
	plans     []*Plan
	sem       chan struct{}
	events    chan struct{}
	running   bool
	ctx       context.Context
	cancel    context.CancelFunc
	rtMu      sync.Mutex // serialize runtime usage across goroutines
	pollEvery time.Duration

	// simple belief store for this agent (plan trigger/guard/steps can consult)
	beliefsMu sync.RWMutex
	beliefs   map[string]Value
}

func newAgent(rt *Runtime, maxConcurrent int, pollEvery time.Duration) *Agent {
	if maxConcurrent <= 0 {
		maxConcurrent = 1
	}
	if pollEvery <= 0 {
		pollEvery = 3 * time.Second
	}
	return &Agent{
		name:      "",
		rt:        rt,
		sem:       make(chan struct{}, maxConcurrent),
		events:    make(chan struct{}, 64),
		pollEvery: pollEvery,
		beliefs:   make(map[string]Value),
	}
}

func (a *Agent) register(p *Plan) {
	a.mu.Lock()
	defer a.mu.Unlock()

	// Check if plan already registered by name (idempotent)
	for _, existing := range a.plans {
		if existing.Name == p.Name {
			return // Already registered, skip duplicate
		}
	}

	a.plans = append(a.plans, p)
}

func (a *Agent) publish() {
	select {
	case a.events <- struct{}{}:
	default:
	}
}

// SetBelief sets a key/value on this agent and nudges the scheduler
func (a *Agent) SetBelief(key string, v Value) {
	a.beliefsMu.Lock()
	a.beliefs[key] = v
	a.beliefsMu.Unlock()
	a.publish()
}

// GetBelief reads a belief by key; returns nil if unset
func (a *Agent) GetBelief(key string) Value {
	a.beliefsMu.RLock()
	defer a.beliefsMu.RUnlock()
	return a.beliefs[key]
}

// GetBeliefs returns a copy of all beliefs
func (a *Agent) GetBeliefs() map[string]Value {
	a.beliefsMu.RLock()
	defer a.beliefsMu.RUnlock()
	copy := make(map[string]Value, len(a.beliefs))
	for k, v := range a.beliefs {
		copy[k] = v
	}
	return copy
}

// GetInfo returns agent metadata including name, plans, and status
func (a *Agent) GetInfo() map[string]interface{} {
	a.mu.RLock()
	planNames := make([]string, len(a.plans))
	for i, p := range a.plans {
		planNames[i] = p.Name
	}
	running := a.running
	pollSeconds := a.pollEvery.Seconds()
	a.mu.RUnlock()

	beliefs := a.GetBeliefs()
	beliefCount := len(beliefs)

	return map[string]interface{}{
		"name":        a.name,
		"plans":       planNames,
		"running":     running,
		"pollSeconds": pollSeconds,
		"beliefCount": beliefCount,
	}
}

func (a *Agent) start(ctx context.Context) {
	if a.running {
		return
	}
	a.running = true
	a.ctx, a.cancel = context.WithCancel(ctx)
	go a.loop()
}

func (a *Agent) stop() {
	if !a.running {
		return
	}
	a.cancel()
	a.running = false
}

func (a *Agent) loop() {
	ticker := time.NewTicker(a.pollEvery)
	defer ticker.Stop()
	for {
		select {
		case <-a.ctx.Done():
			return
		case <-ticker.C:
			a.trySchedule()
		case <-a.events:
			a.trySchedule()
		}
	}
}

func (a *Agent) trySchedule() {
	a.mu.RLock()
	plans := append([]*Plan(nil), a.plans...)
	a.mu.RUnlock()

	for _, p := range plans {
		// Evaluate trigger and guard quickly; ignore errors as false
		if ok, _ := a.evalBool(p.Trigger); !ok {
			continue
		}
		if ok, _ := a.evalBool(p.Guard); !ok {
			continue
		}
		select {
		case a.sem <- struct{}{}:
			go func(pl *Plan) {
				defer func() { <-a.sem }()
				_ = a.runPlanOnce(pl)
			}(p)
		default:
			return
		}
	}
}

func (a *Agent) evalBool(fn *FunctionValue) (bool, error) {
	if fn == nil {
		return false, nil
	}
	a.rtMu.Lock()
	defer a.rtMu.Unlock()
	v, err := executeFunctionValue(a.rt, fn, nil)
	if err != nil {
		return false, err
	}
	switch b := v.(type) {
	case Bool:
		return bool(b), nil
	case bool:
		return b, nil
	case Number:
		return b != 0, nil
	default:
		return false, nil
	}
}

// runPlanOnce executes steps sequentially with drop checks; returns error if a step fails
func (a *Agent) runPlanOnce(p *Plan) error {
	ctx := a.ctx
	if ctx == nil {
		ctx = context.Background()
	}
	// Broadcast plan start
	broadcastAgentEvent(AgentEvent{Type: "plan", Agent: a.name, Plan: p.Name, Status: "start", Time: time.Now()})
	// Plan instance scope so variables persist across steps for this run only.
	// Use a child scope of the agent runtime's global scope to avoid polluting globals.
	instanceScope := NewScope(a.rt.globalScope)
	for i, step := range p.Steps {
		// Drop before step
		drop, _ := a.evalBool(p.Drop)
		if drop {
			broadcastAgentEvent(AgentEvent{Type: "plan", Agent: a.name, Plan: p.Name, Status: "drop", Step: i, Time: time.Now()})
			return nil
		}
		// Execute step
		broadcastAgentEvent(AgentEvent{Type: "step", Agent: a.name, Plan: p.Name, Step: i, Status: "start", Time: time.Now()})
		a.rtMu.Lock()
		_, err := a.execFnInScope(step, instanceScope)
		a.rtMu.Unlock()
		if err != nil {
			broadcastAgentEvent(AgentEvent{Type: "step", Agent: a.name, Plan: p.Name, Step: i, Status: "error", Error: err.Error(), Time: time.Now()})
			return fmt.Errorf("step %d failed: %w", i, err)
		}
		broadcastAgentEvent(AgentEvent{Type: "step", Agent: a.name, Plan: p.Name, Step: i, Status: "finish", Time: time.Now()})
		// Cooperative cancellation point
		select {
		case <-ctx.Done():
			broadcastAgentEvent(AgentEvent{Type: "plan", Agent: a.name, Plan: p.Name, Status: "cancel", Time: time.Now()})
			return ctx.Err()
		default:
		}
	}
	broadcastAgentEvent(AgentEvent{Type: "plan", Agent: a.name, Plan: p.Name, Status: "finish", Time: time.Now()})
	return nil
}

// runPlanOnceWithOptions runs a single plan instance with optional instance-scope variables and mode.
// Modes:
//   - "bdi" (default): require Trigger && Guard, respect Drop
//   - "guard-only": bypass Trigger, require Guard, respect Drop
//   - "force": bypass Trigger and Guard, respect Drop
//   - "force-all": bypass Trigger and Guard, bypass Drop
//   - "dry-run": evaluate according to BDI (or other provided mode) but do not execute steps; returns whether it WOULD run
func (a *Agent) runPlanOnceWithOptions(p *Plan, instanceVars map[string]Value, mode string) (bool, error) {
	if p == nil {
		return false, errors.New("nil plan")
	}

	m := strings.ToLower(strings.TrimSpace(mode))
	if m == "" {
		m = "bdi"
	}
	dryRun := false
	checkTrig, checkGuard, respectDrop := true, true, true
	switch m {
	case "bdi":
		// default
	case "guard-only":
		checkTrig = false
	case "force":
		checkTrig = false
		checkGuard = false
	case "force-all":
		checkTrig = false
		checkGuard = false
		respectDrop = false
	case "dry-run":
		dryRun = true
		// keep default BDI checks
	default:
		// unknown mode → treat as BDI
		m = "bdi"
	}

	// Instance scope per run, overlay any provided variables
	instanceScope := NewScope(a.rt.globalScope)
	if len(instanceVars) > 0 {
		for k, v := range instanceVars {
			instanceScope.Set(k, v)
		}
	}

	// Evaluate trigger/guard depending on mode
	if checkTrig {
		ok, _ := a.evalBool(p.Trigger)
		if !ok {
			return false, nil // not executed
		}
	}
	if checkGuard {
		ok, _ := a.evalBool(p.Guard)
		if !ok {
			return false, nil // not executed
		}
	}

	if dryRun {
		return true, nil // would run, but skip executing steps
	}

	// Execute steps
	executed := false
	ctx := a.ctx
	if ctx == nil {
		ctx = context.Background()
	}
	broadcastAgentEvent(AgentEvent{Type: "plan", Agent: a.name, Plan: p.Name, Status: "start", Time: time.Now()})
	for i, step := range p.Steps {
		if respectDrop {
			drop, _ := a.evalBool(p.Drop)
			if drop {
				broadcastAgentEvent(AgentEvent{Type: "plan", Agent: a.name, Plan: p.Name, Status: "drop", Step: i, Time: time.Now()})
				if executed {
					return true, nil
				}
				return false, nil
			}
		}
		broadcastAgentEvent(AgentEvent{Type: "step", Agent: a.name, Plan: p.Name, Step: i, Status: "start", Time: time.Now()})
		a.rtMu.Lock()
		_, err := a.execFnInScope(step, instanceScope)
		a.rtMu.Unlock()
		if err != nil {
			broadcastAgentEvent(AgentEvent{Type: "step", Agent: a.name, Plan: p.Name, Step: i, Status: "error", Error: err.Error(), Time: time.Now()})
			return false, fmt.Errorf("step %d failed: %w", i, err)
		}
		executed = true
		broadcastAgentEvent(AgentEvent{Type: "step", Agent: a.name, Plan: p.Name, Step: i, Status: "finish", Time: time.Now()})
		select {
		case <-ctx.Done():
			broadcastAgentEvent(AgentEvent{Type: "plan", Agent: a.name, Plan: p.Name, Status: "cancel", Time: time.Now()})
			return executed, ctx.Err()
		default:
		}
	}
	broadcastAgentEvent(AgentEvent{Type: "plan", Agent: a.name, Plan: p.Name, Status: "finish", Time: time.Now()})
	return executed, nil
}

// execFnInScope executes a function value with rt.currentScope set to the provided scope
// allowing step-local setq() to persist across subsequent steps.
func (a *Agent) execFnInScope(fn *FunctionValue, scope *Scope) (Value, error) {
	if fn == nil {
		return nil, errors.New("nil function")
	}
	// Bind params into scope (no args supported yet)
	prev := a.rt.currentScope
	a.rt.currentScope = scope
	defer func() { a.rt.currentScope = prev }()

	// Execute body similar to executeFunctionValue but without creating a child scope
	if block, ok := fn.Body.(*Block); ok {
		var last Value
		for _, stmt := range block.Stmts {
			v, err := stmt.Exec(a.rt)
			if err != nil {
				return nil, err
			}
			last = v
		}
		return last, nil
	}
	return fn.Body.Exec(a.rt)
}

// RegisterPlanFunctions wires plan/agent functions into the runtime
func RegisterPlanFunctions(rt *Runtime) {
	// plan(name, paramsArray, triggerFn, guardFn, stepsArray, dropFn)
	rt.Register("plan", func(args ...Value) (Value, error) {
		if len(args) != 6 {
			return nil, errors.New("plan requires 6 arguments: name, params[], triggerFn, guardFn, steps[], dropFn")
		}
		// Unwrap ScopeEntries
		for i, a := range args {
			if se, ok := a.(ScopeEntry); ok {
				args[i] = se.Value
			}
		}
		name, ok := args[0].(Str)
		if !ok {
			return nil, fmt.Errorf("name must be string, got %T", args[0])
		}
		// Params
		var params []string
		if arr, ok := args[1].(*ArrayValue); ok {
			for i := 0; i < arr.Length(); i++ {
				if s, ok := arr.Get(i).(Str); ok {
					params = append(params, string(s))
				}
			}
		}
		trg, ok := args[2].(*FunctionValue)
		if !ok {
			return nil, fmt.Errorf("trigger must be function")
		}
		grd, ok := args[3].(*FunctionValue)
		if !ok {
			return nil, fmt.Errorf("guard must be function")
		}
		var steps []*FunctionValue
		if arr, ok := args[4].(*ArrayValue); ok {
			for i := 0; i < arr.Length(); i++ {
				if fn, ok := arr.Get(i).(*FunctionValue); ok {
					steps = append(steps, fn)
				}
			}
		} else {
			return nil, fmt.Errorf("steps must be array of functions")
		}
		drp, ok := args[5].(*FunctionValue)
		if !ok {
			return nil, fmt.Errorf("dropCond must be function")
		}
		p := &Plan{Name: string(name), Params: params, Trigger: trg, Guard: grd, Steps: steps, Drop: drp}
		return p, nil
	})

	// agentNew([maxConcurrent],[pollSeconds]) -> agent
	rt.Register("agentNew", func(args ...Value) (Value, error) {
		maxC := 1
		pollSec := 3
		if len(args) > 0 {
			if n, ok := args[0].(Number); ok {
				maxC = int(n)
			}
		}
		if len(args) > 1 {
			if n, ok := args[1].(Number); ok {
				pollSec = int(n)
			}
		}
		ag := newAgent(rt, maxC, time.Duration(pollSec)*time.Second)
		return &HostObjectValue{Value: ag, Name: "agent"}, nil
	})

	rt.Register("agentRegister", func(args ...Value) (Value, error) {
		if len(args) != 2 {
			return nil, errors.New("agentRegister(agent, plan)")
		}
		ag, ok := asAgent(args[0])
		if !ok {
			return nil, errors.New("first arg not agent")
		}
		p, ok := args[1].(*Plan)
		if !ok {
			return nil, errors.New("second arg not plan")
		}
		ag.register(p)
		return Bool(true), nil
	})

	rt.Register("agentStart", func(args ...Value) (Value, error) {
		if len(args) < 1 {
			return nil, errors.New("agentStart(agent)")
		}
		ag, ok := asAgent(args[0])
		if !ok {
			return nil, errors.New("not an agent")
		}
		ag.start(context.Background())
		return Bool(true), nil
	})

	rt.Register("agentStop", func(args ...Value) (Value, error) {
		if len(args) < 1 {
			return nil, errors.New("agentStop(agent)")
		}
		ag, ok := asAgent(args[0])
		if !ok {
			return nil, errors.New("not an agent")
		}
		ag.stop()
		return Bool(true), nil
	})

	// Convenience: runPlanOnce(plan) -> true/false
	rt.Register("runPlanOnce", func(args ...Value) (Value, error) {
		if len(args) != 1 {
			return nil, errors.New("runPlanOnce(plan)")
		}
		p, ok := args[0].(*Plan)
		if !ok {
			return nil, errors.New("argument must be plan")
		}
		ag := newAgent(rt, 1, 0)
		if err := ag.runPlanOnce(p); err != nil {
			return nil, fmt.Errorf("plan failed: %w", err)
		}
		return Bool(true), nil
	})

	// runPlanOnceBDI(plan[, varsMap]) -> true if executed, false if no-op
	rt.Register("runPlanOnceBDI", func(args ...Value) (Value, error) {
		if len(args) < 1 || len(args) > 2 {
			return nil, errors.New("runPlanOnceBDI(plan[, varsMap])")
		}
		p, ok := args[0].(*Plan)
		if !ok {
			return nil, errors.New("first argument must be plan")
		}
		// Per-run cloned runtime and rebound plan
		agentRT := rt.CloneRuntime()
		rp := rebindPlanToRuntime(p, agentRT)
		ag := newAgent(agentRT, 1, 0)
		// Optional instance vars
		vars := map[string]Value{}
		if len(args) == 2 {
			if mv, ok := args[1].(*MapValue); ok && mv != nil {
				for k, v := range mv.Values {
					vars[k] = v
				}
			}
		}
		executed, err := ag.runPlanOnceWithOptions(rp, vars, "bdi")
		if err != nil {
			return nil, err
		}
		return Bool(executed), nil
	})

	// runPlanOnceEx(plan[, mode][, varsMap]) -> true if executed (or would execute for dry-run), false if no-op
	rt.Register("runPlanOnceEx", func(args ...Value) (Value, error) {
		if len(args) < 1 || len(args) > 3 {
			return nil, errors.New("runPlanOnceEx(plan[, mode][, varsMap])")
		}
		p, ok := args[0].(*Plan)
		if !ok {
			return nil, errors.New("first argument must be plan")
		}
		mode := "bdi"
		varIdx := 1
		if len(args) >= 2 {
			if s, ok := args[1].(Str); ok {
				mode = string(s)
				varIdx = 2
			}
		}
		vars := map[string]Value{}
		if len(args) > varIdx {
			if mv, ok := args[varIdx].(*MapValue); ok && mv != nil {
				for k, v := range mv.Values {
					vars[k] = v
				}
			}
		}
		agentRT := rt.CloneRuntime()
		rp := rebindPlanToRuntime(p, agentRT)
		ag := newAgent(agentRT, 1, 0)
		executed, err := ag.runPlanOnceWithOptions(rp, vars, mode)
		if err != nil {
			return nil, err
		}
		return Bool(executed), nil
	})

	// ---- Name-based Agent registry functions (for REST/NSQ control and dashboard) ----

	// agentStartNamed(name, plan[, maxConcurrent=1][, pollSeconds=3]) -> true
	rt.Register("agentStartNamed", func(args ...Value) (Value, error) {
		if len(args) < 2 {
			return nil, errors.New("agentStartNamed(name, plan[, maxConcurrent][, pollSeconds])")
		}
		name, ok := args[0].(Str)
		if !ok || name == "" {
			return nil, errors.New("first arg must be non-empty string name")
		}
		p, ok := args[1].(*Plan)
		if !ok {
			return nil, errors.New("second arg must be plan")
		}
		maxC := 1
		pollSec := 3
		if len(args) > 2 {
			if n, ok := args[2].(Number); ok && n > 0 {
				maxC = int(n)
			}
		}
		if len(args) > 3 {
			if n, ok := args[3].(Number); ok && n > 0 {
				pollSec = int(n)
			}
		}
		if err := defaultAgents.Start(string(name), rt, p, maxC, time.Duration(pollSec)*time.Second); err != nil {
			return nil, err
		}
		return Bool(true), nil
	})

	// agentStopNamed(name) -> true
	rt.Register("agentStopNamed", func(args ...Value) (Value, error) {
		if len(args) < 1 {
			return nil, errors.New("agentStopNamed(name)")
		}
		name, ok := args[0].(Str)
		if !ok || name == "" {
			return nil, errors.New("first arg must be non-empty string name")
		}
		defaultAgents.Stop(string(name))
		return Bool(true), nil
	})

	// agentList() -> array of names
	rt.Register("agentList", func(args ...Value) (Value, error) {
		names := defaultAgents.List()
		arr := NewArray()
		for _, n := range names {
			arr.Append(Str(n))
		}
		return arr, nil
	})

	// agentPublish(name) -> true  (nudge scheduler)
	rt.Register("agentPublish", func(args ...Value) (Value, error) {
		if len(args) < 1 {
			return nil, errors.New("agentPublish(name)")
		}
		name, ok := args[0].(Str)
		if !ok || name == "" {
			return nil, errors.New("first arg must be non-empty string name")
		}
		if ag := defaultAgents.Get(string(name)); ag != nil {
			ag.publish()
			return Bool(true), nil
		}
		return Bool(false), nil
	})

	// agentBelief(name, key, value) -> true (store belief and nudge)
	rt.Register("agentBelief", func(args ...Value) (Value, error) {
		if len(args) < 3 {
			return nil, errors.New("agentBelief(name, key, value)")
		}
		name, ok := args[0].(Str)
		if !ok || name == "" {
			return nil, errors.New("first arg must be non-empty string name")
		}
		key, ok := args[1].(Str)
		if !ok || key == "" {
			return nil, errors.New("second arg must be non-empty string key")
		}
		if ag := defaultAgents.Get(string(name)); ag != nil {
			ag.SetBelief(string(key), args[2])
			return Bool(true), nil
		}
		return Bool(false), nil
	})

	// belief(name, key) -> value|nil
	rt.Register("belief", func(args ...Value) (Value, error) {
		if len(args) < 2 {
			return nil, errors.New("belief(name, key)")
		}
		name, ok := args[0].(Str)
		if !ok || name == "" {
			return nil, errors.New("first arg must be non-empty string name")
		}
		key, ok := args[1].(Str)
		if !ok || key == "" {
			return nil, errors.New("second arg must be non-empty string key")
		}
		if ag := defaultAgents.Get(string(name)); ag != nil {
			return ag.GetBelief(string(key)), nil
		}
		return nil, nil
	})
}

func asAgent(v Value) (*Agent, bool) {
	if ho, ok := v.(*HostObjectValue); ok {
		if ag, ok := ho.Value.(*Agent); ok {
			return ag, true
		}
	}
	return nil, false
}

// ---- simple in-process name->Agent registry ----

type agentRegistry struct {
	mu     sync.RWMutex
	agents map[string]*Agent
}

var defaultAgents = &agentRegistry{agents: make(map[string]*Agent)}

func (r *agentRegistry) Get(name string) *Agent {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.agents[name]
}

func (r *agentRegistry) Start(name string, rt *Runtime, pl *Plan, maxC int, pollEvery time.Duration) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if ag, ok := r.agents[name]; ok {
		// re-use existing agent: register plan and ensure running
		// rebind plan to the existing agent's runtime before registering
		ag.register(rebindPlanToRuntime(pl, ag.rt))
		ag.start(context.Background())
		return nil
	}
	// Create an isolated per-agent runtime cloned from the provided bootstrap runtime
	agentRT := rt.CloneRuntime()
	ag := newAgent(agentRT, maxC, pollEvery)
	ag.name = name
	// Rebind plan functions to the agent runtime's scope
	ag.register(rebindPlanToRuntime(pl, agentRT))
	ag.start(context.Background())
	r.agents[name] = ag
	return nil
}

func (r *agentRegistry) Stop(name string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if ag, ok := r.agents[name]; ok {
		ag.stop()
		delete(r.agents, name)
	}
}

func (r *agentRegistry) List() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]string, 0, len(r.agents))
	for k := range r.agents {
		out = append(out, k)
	}
	return out
}

// Exported helpers for other packages (handlers) to interact with the default registry
func DefaultAgentNames() []string { return defaultAgents.List() }

func DefaultAgentStart(name string, rt *Runtime, pl *Plan, maxC int, pollEvery time.Duration) error {
	return defaultAgents.Start(name, rt, pl, maxC, pollEvery)
}

func DefaultAgentStop(name string) { defaultAgents.Stop(name) }

func DefaultAgentPublish(name string) bool {
	if ag := defaultAgents.Get(name); ag != nil {
		ag.publish()
		return true
	}
	return false
}

func DefaultAgentBelief(name, key string, v Value) bool {
	if ag := defaultAgents.Get(name); ag != nil {
		ag.SetBelief(key, v)
		return true
	}
	return false
}

// DefaultAgentGetBeliefs returns all beliefs for a named agent
func DefaultAgentGetBeliefs(name string) map[string]Value {
	if ag := defaultAgents.Get(name); ag != nil {
		return ag.GetBeliefs()
	}
	return nil
}

// DefaultAgentGetInfo returns detailed info about an agent
func DefaultAgentGetInfo(name string) map[string]interface{} {
	if ag := defaultAgents.Get(name); ag != nil {
		return ag.GetInfo()
	}
	return nil
}
