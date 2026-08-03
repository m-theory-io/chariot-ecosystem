package chariot

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"
)

const (
	AgentLifecyclePolling  = "polling"
	AgentLifecycleEvent    = "eventOnly"
	AgentLifecycleCallOnly = "callOnly"
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

const planStepResultVar = "__plan_step_result"

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

// PlanRunResult is the structured outcome of a single plan execution.
type PlanRunResult struct {
	Agent        string
	Plan         string
	Mode         string
	Status       string
	Executed     bool
	WouldExecute bool
	Reason       string
	Steps        []PlanStepResult
	Output       Value
	Error        string
}

// PlanStepResult records the outcome of one plan step.
type PlanStepResult struct {
	Index  int
	Status string
	Result Value
	Error  string
}

func (r *PlanRunResult) ToJSON() map[string]interface{} {
	if r == nil {
		return nil
	}
	out := map[string]interface{}{
		"agent":        r.Agent,
		"plan":         r.Plan,
		"mode":         r.Mode,
		"status":       r.Status,
		"executed":     r.Executed,
		"wouldExecute": r.WouldExecute,
	}
	if r.Reason != "" {
		out["reason"] = r.Reason
	}
	if r.Output != nil {
		out["output"] = ValueToJSON(r.Output)
	}
	if r.Error != "" {
		out["error"] = r.Error
	}
	steps := make([]map[string]interface{}, 0, len(r.Steps))
	for _, step := range r.Steps {
		stepJSON := map[string]interface{}{
			"index":  step.Index,
			"status": step.Status,
		}
		if step.Result != nil {
			stepJSON["result"] = ValueToJSON(step.Result)
		}
		if step.Error != "" {
			stepJSON["error"] = step.Error
		}
		steps = append(steps, stepJSON)
	}
	out["steps"] = steps
	return out
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
	lifecycle string

	// simple belief store for this agent (plan trigger/guard/steps can consult)
	beliefsMu sync.RWMutex
	beliefs   map[string]Value

	resultsMu sync.RWMutex
	results   []*PlanRunResult
}

func normalizeAgentLifecycle(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "event", "eventonly", "event-only", "manual", "manualevent":
		return AgentLifecycleEvent
	case "call", "callonly", "call-only", "mcp", "runonce", "run-once":
		return AgentLifecycleCallOnly
	case "", "poll", "polling", "active":
		return AgentLifecyclePolling
	default:
		return AgentLifecyclePolling
	}
}

func newAgent(rt *Runtime, maxConcurrent int, pollEvery time.Duration, lifecycleOpt ...string) *Agent {
	if maxConcurrent <= 0 {
		maxConcurrent = 1
	}
	lifecycle := AgentLifecyclePolling
	if len(lifecycleOpt) > 0 {
		lifecycle = normalizeAgentLifecycle(lifecycleOpt[0])
	}
	if pollEvery <= 0 && lifecycle == AgentLifecyclePolling {
		pollEvery = 3 * time.Second
	}
	if lifecycle != AgentLifecyclePolling {
		pollEvery = 0
	}
	return &Agent{
		name:      "",
		rt:        rt,
		sem:       make(chan struct{}, maxConcurrent),
		events:    make(chan struct{}, 64),
		pollEvery: pollEvery,
		lifecycle: lifecycle,
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

	lastResults := a.RecentResultsJSON()
	var lastResult map[string]interface{}
	if len(lastResults) > 0 {
		lastResult = lastResults[len(lastResults)-1]
	}

	beliefs := a.GetBeliefs()
	beliefCount := len(beliefs)

	return map[string]interface{}{
		"name":          a.name,
		"plans":         planNames,
		"running":       running,
		"lifecycle":     a.lifecycle,
		"pollSeconds":   pollSeconds,
		"beliefCount":   beliefCount,
		"lastResult":    lastResult,
		"recentResults": lastResults,
	}
}

func (a *Agent) start(ctx context.Context) {
	if a.lifecycle == AgentLifecycleCallOnly {
		return
	}
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
	var ticker *time.Ticker
	var tickerC <-chan time.Time
	if a.lifecycle == AgentLifecyclePolling && a.pollEvery > 0 {
		ticker = time.NewTicker(a.pollEvery)
		tickerC = ticker.C
		defer ticker.Stop()
	}
	for {
		select {
		case <-a.ctx.Done():
			return
		case <-tickerC:
			a.tryScheduleAsync()
		case <-a.events:
			a.tryScheduleAsync()
		}
	}
}

func (a *Agent) tryScheduleAsync() {
	a.mu.RLock()
	plans := append([]*Plan(nil), a.plans...)
	a.mu.RUnlock()

	for _, p := range plans {
		select {
		case a.sem <- struct{}{}:
			go func(pl *Plan) {
				defer func() { <-a.sem }()
				result, _ := a.runPlanOnceWithResult(pl, nil, "bdi")
				a.recordResult(result)
			}(p)
		default:
			return
		}
	}
}

func (a *Agent) tryScheduleSync(mode string) []*PlanRunResult {
	a.mu.RLock()
	plans := append([]*Plan(nil), a.plans...)
	a.mu.RUnlock()

	results := []*PlanRunResult{}
	for _, p := range plans {
		select {
		case a.sem <- struct{}{}:
			result, _ := a.runPlanOnceWithResult(p, nil, mode)
			<-a.sem
			a.recordResult(result)
			if result != nil {
				results = append(results, result)
			}
		default:
			return results
		}
	}
	return results
}

func (a *Agent) recordResult(result *PlanRunResult) {
	if result == nil || !result.Executed {
		return
	}
	a.resultsMu.Lock()
	defer a.resultsMu.Unlock()
	a.results = append(a.results, result)
	if len(a.results) > 20 {
		a.results = append([]*PlanRunResult(nil), a.results[len(a.results)-20:]...)
	}
}

func (a *Agent) RecentResultsJSON() []map[string]interface{} {
	a.resultsMu.RLock()
	defer a.resultsMu.RUnlock()
	out := make([]map[string]interface{}, 0, len(a.results))
	for _, result := range a.results {
		if result != nil {
			out = append(out, result.ToJSON())
		}
	}
	return out
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
	_, err := a.runPlanOnceWithResult(p, nil, "plain")
	return err
}

// runPlanOnceWithOptions runs a single plan instance with optional instance-scope variables and mode.
// Modes:
//   - "bdi" (default): require Trigger && Guard, respect Drop
//   - "guard-only": bypass Trigger, require Guard, respect Drop
//   - "force": bypass Trigger and Guard, respect Drop
//   - "force-all": bypass Trigger and Guard, bypass Drop
//   - "dry-run": evaluate according to BDI (or other provided mode) but do not execute steps; returns whether it WOULD run
func (a *Agent) runPlanOnceWithOptions(p *Plan, instanceVars map[string]Value, mode string) (bool, error) {
	result, err := a.runPlanOnceWithResult(p, instanceVars, mode)
	if result == nil {
		return false, err
	}
	return result.Executed || result.WouldExecute, err
}

func (a *Agent) runPlanOnceWithResult(p *Plan, instanceVars map[string]Value, mode string) (*PlanRunResult, error) {
	if p == nil {
		return nil, errors.New("nil plan")
	}

	m := strings.ToLower(strings.TrimSpace(mode))
	if m == "" {
		m = "bdi"
	}
	dryRun := false
	checkTrig, checkGuard, respectDrop := true, true, true
	switch m {
	case "plain":
		checkTrig = false
		checkGuard = false
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
	result := &PlanRunResult{
		Agent:  a.name,
		Plan:   p.Name,
		Mode:   m,
		Status: "skipped",
		Steps:  []PlanStepResult{},
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
		ok, err := a.evalBool(p.Trigger)
		if err != nil {
			result.Reason = "trigger_error"
			result.Error = err.Error()
			return result, nil
		}
		if !ok {
			result.Reason = "trigger_false"
			return result, nil
		}
	}
	if checkGuard {
		ok, err := a.evalBool(p.Guard)
		if err != nil {
			result.Reason = "guard_error"
			result.Error = err.Error()
			return result, nil
		}
		if !ok {
			result.Reason = "guard_false"
			return result, nil
		}
	}

	if dryRun {
		result.Status = "would_run"
		result.WouldExecute = true
		return result, nil
	}

	// Execute steps
	ctx := a.ctx
	if ctx == nil {
		ctx = context.Background()
	}
	broadcastAgentEvent(AgentEvent{Type: "plan", Agent: a.name, Plan: p.Name, Status: "start", Time: time.Now()})
	for i, step := range p.Steps {
		if respectDrop {
			drop, err := a.evalBool(p.Drop)
			if err != nil {
				result.Status = "failed"
				result.Reason = "drop_error"
				result.Error = err.Error()
				broadcastAgentEvent(AgentEvent{Type: "plan", Agent: a.name, Plan: p.Name, Status: "error", Step: i, Error: err.Error(), Time: time.Now()})
				return result, err
			}
			if drop {
				result.Status = "dropped"
				result.Reason = "drop_true"
				broadcastAgentEvent(AgentEvent{Type: "plan", Agent: a.name, Plan: p.Name, Status: "drop", Step: i, Time: time.Now()})
				return result, nil
			}
		}
		broadcastAgentEvent(AgentEvent{Type: "step", Agent: a.name, Plan: p.Name, Step: i, Status: "start", Time: time.Now()})
		instanceScope.Set(planStepResultVar, DBNull)
		a.rtMu.Lock()
		stepValue, err := a.execFnInScope(step, instanceScope)
		a.rtMu.Unlock()
		if err != nil {
			result.Status = "failed"
			result.Reason = "step_error"
			result.Error = err.Error()
			result.Steps = append(result.Steps, PlanStepResult{Index: i, Status: "failed", Error: err.Error()})
			broadcastAgentEvent(AgentEvent{Type: "step", Agent: a.name, Plan: p.Name, Step: i, Status: "error", Error: err.Error(), Time: time.Now()})
			return result, fmt.Errorf("step %d failed: %w", i, err)
		}
		if explicitResult, ok := instanceScope.Get(planStepResultVar); ok && explicitResult != DBNull {
			stepValue = explicitResult
		}
		result.Executed = true
		result.Output = stepValue
		result.Steps = append(result.Steps, PlanStepResult{Index: i, Status: "completed", Result: stepValue})
		broadcastAgentEvent(AgentEvent{Type: "step", Agent: a.name, Plan: p.Name, Step: i, Status: "finish", Time: time.Now()})
		select {
		case <-ctx.Done():
			result.Status = "canceled"
			result.Reason = "canceled"
			result.Error = ctx.Err().Error()
			broadcastAgentEvent(AgentEvent{Type: "plan", Agent: a.name, Plan: p.Name, Status: "cancel", Time: time.Now()})
			return result, ctx.Err()
		default:
		}
	}
	result.Status = "completed"
	broadcastAgentEvent(AgentEvent{Type: "plan", Agent: a.name, Plan: p.Name, Status: "finish", Time: time.Now()})
	return result, nil
}

// RunPlanOnceResult executes a plan once and returns a structured execution result.
func RunPlanOnceResult(rt *Runtime, agentName string, p *Plan, instanceVars map[string]Value, mode string) (*PlanRunResult, error) {
	if rt == nil {
		return nil, errors.New("nil runtime")
	}
	m := strings.ToLower(strings.TrimSpace(mode))
	if m == "" {
		m = "bdi"
	}
	if m == "plain" {
		ag := newAgent(rt, 1, 0)
		ag.name = agentName
		return ag.runPlanOnceWithResult(p, instanceVars, m)
	}
	agentRT := rt.CloneRuntime()
	rp := rebindPlanToRuntime(p, agentRT)
	ag := newAgent(agentRT, 1, 0)
	ag.name = agentName
	return ag.runPlanOnceWithResult(rp, instanceVars, m)
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
	setPlanResult := func(args ...Value) (Value, error) {
		if len(args) != 1 {
			return nil, errors.New("setStepResult(value)")
		}
		scope := rt.currentScope
		if scope == nil {
			scope = rt.globalScope
		}
		scope.Set(planStepResultVar, args[0])
		return args[0], nil
	}
	rt.Register("setStepResult", setPlanResult)
	rt.Register("setPlanResult", setPlanResult)

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

	// agentStartNamed(name, plan[, maxConcurrent=1][, pollSeconds=3][, lifecycle='polling']) -> true
	rt.Register("agentStartNamed", func(args ...Value) (Value, error) {
		if len(args) < 2 {
			return nil, errors.New("agentStartNamed(name, plan[, maxConcurrent][, pollSeconds][, lifecycle])")
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
		lifecycle := AgentLifecyclePolling
		numberArg := 0
		for _, arg := range args[2:] {
			switch v := arg.(type) {
			case Number:
				if v <= 0 {
					continue
				}
				if numberArg == 0 {
					maxC = int(v)
				} else if numberArg == 1 {
					pollSec = int(v)
				}
				numberArg++
			case Str:
				lifecycle = normalizeAgentLifecycle(string(v))
			}
		}
		if err := defaultAgents.StartWithLifecycle(string(name), rt, p, maxC, time.Duration(pollSec)*time.Second, lifecycle); err != nil {
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
	return r.StartWithLifecycle(name, rt, pl, maxC, pollEvery, AgentLifecyclePolling)
}

func (r *agentRegistry) StartWithLifecycle(name string, rt *Runtime, pl *Plan, maxC int, pollEvery time.Duration, lifecycle string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	lifecycle = normalizeAgentLifecycle(lifecycle)
	if pollEvery <= 0 && lifecycle == AgentLifecyclePolling {
		pollEvery = 3 * time.Second
	}
	if lifecycle != AgentLifecyclePolling {
		pollEvery = 0
	}
	if ag, ok := r.agents[name]; ok {
		// re-use existing agent: register plan and ensure running
		// rebind plan to the existing agent's runtime before registering
		ag.register(rebindPlanToRuntime(pl, ag.rt))
		ag.lifecycle = lifecycle
		ag.pollEvery = pollEvery
		if lifecycle == AgentLifecycleCallOnly {
			ag.stop()
		} else {
			ag.start(context.Background())
		}
		return nil
	}
	// Create an isolated per-agent runtime cloned from the provided bootstrap runtime
	agentRT := rt.CloneRuntime()
	ag := newAgent(agentRT, maxC, pollEvery, lifecycle)
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

func DefaultAgentStartWithLifecycle(name string, rt *Runtime, pl *Plan, maxC int, pollEvery time.Duration, lifecycle string) error {
	return defaultAgents.StartWithLifecycle(name, rt, pl, maxC, pollEvery, lifecycle)
}

func DefaultAgentStop(name string) { defaultAgents.Stop(name) }

func DefaultAgentPublish(name string) bool {
	if ag := defaultAgents.Get(name); ag != nil {
		ag.publish()
		return true
	}
	return false
}

func DefaultAgentActivate(name string) ([]map[string]interface{}, bool) {
	if ag := defaultAgents.Get(name); ag != nil {
		if ag.lifecycle == AgentLifecycleCallOnly {
			return []map[string]interface{}{}, true
		}
		results := ag.tryScheduleSync("bdi")
		out := make([]map[string]interface{}, 0, len(results))
		for _, result := range results {
			if result != nil {
				out = append(out, result.ToJSON())
			}
		}
		return out, true
	}
	return nil, false
}

func DefaultAgentBelief(name, key string, v Value) bool {
	if ag := defaultAgents.Get(name); ag != nil {
		ag.SetBelief(key, v)
		return true
	}
	return false
}

func DefaultAgentSetBeliefQuiet(name, key string, v Value) bool {
	if ag := defaultAgents.Get(name); ag != nil {
		ag.beliefsMu.Lock()
		ag.beliefs[key] = v
		ag.beliefsMu.Unlock()
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
