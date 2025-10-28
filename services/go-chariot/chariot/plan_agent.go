package chariot

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"
)

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

// Agent coordinates plan execution with bounded concurrency
type Agent struct {
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
}

func newAgent(rt *Runtime, maxConcurrent int, pollEvery time.Duration) *Agent {
	if maxConcurrent <= 0 {
		maxConcurrent = 1
	}
	if pollEvery <= 0 {
		pollEvery = 3 * time.Second
	}
	return &Agent{
		rt:        rt,
		sem:       make(chan struct{}, maxConcurrent),
		events:    make(chan struct{}, 64),
		pollEvery: pollEvery,
	}
}

func (a *Agent) register(p *Plan) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.plans = append(a.plans, p)
}

func (a *Agent) publish() {
	select {
	case a.events <- struct{}{}:
	default:
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
	// Shared plan instance scope so variables persist across steps.
	// Use the runtime's global scope so results are observable after execution.
	instanceScope := a.rt.globalScope
	for i, step := range p.Steps {
		// Drop before step
		drop, _ := a.evalBool(p.Drop)
		if drop {
			return nil
		}
		// Execute step
		a.rtMu.Lock()
		_, err := a.execFnInScope(step, instanceScope)
		a.rtMu.Unlock()
		if err != nil {
			return fmt.Errorf("step %d failed: %w", i, err)
		}
		// Cooperative cancellation point
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
	}
	return nil
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
}

func asAgent(v Value) (*Agent, bool) {
	if ho, ok := v.(*HostObjectValue); ok {
		if ag, ok := ho.Value.(*Agent); ok {
			return ag, true
		}
	}
	return nil, false
}
