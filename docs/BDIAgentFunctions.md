# BDI Agent Functions and Plan Type

This document describes Chariot's BDI (Belief–Desire–Intention) helpers and the Plan type introduced as a first-class value. It covers how to construct plans, inspect/mutate their properties, run them in different modes, and manage agents.

## Plan as a first-class value

- Type code: `P` (e.g., `typeOf(p) == 'P'`)
- Construct via: `plan(name, paramsArray, triggerFn, guardFn, stepsArray, dropFn)`
- Persistable: Plans survive `treeSave`/`treeLoad` as attributes on tree nodes
- Introspection/mutation: Use `getProp`/`setProp` for key properties

### Constructor

plan(name, paramsArray, triggerFn, guardFn, stepsArray, dropFn) -> plan

- name: string plan name
- paramsArray: array of strings, parameter names
- triggerFn: function with no args; returns truthy when plan should consider running
- guardFn: function with no args; returns truthy to allow execution (after trigger)
- stepsArray: array of functions, executed sequentially if run proceeds
- dropFn: function with no args; if truthy before/within execution, the run is dropped

Example:

```
setq(trig,  func() { true })
setq(guard, func() { true })
setq(steps, array(
  func() { /* step 1 */ 1 },
  func() { /* step 2 */ 2 },
))
setq(drop,  func() { false })

setq(p, plan('Thermostat', array('min','max'), trig, guard, steps, drop))
```

### Properties via getProp/setProp

- name: string
- params: array of strings
- trigger: function
- guard: function
- steps: array of functions
- drop: function

Examples:

```
getProp(p, 'name')           // -> 'Thermostat'
getProp(p, 'params')         // -> ['min','max']
length(getProp(p, 'steps'))  // -> 2

setProp(p, 'name', 'Heater')     // rename
setProp(p, 'params', array('t')) // replace params
```

### Serialization and persistence

- `typeOf(p) == 'P'`
- Plans can be stored as TreeNode attributes and round-trip via `treeSave`/`treeLoad`.
- Inspectors and JSON export use structured representations; functions within a plan (trigger/guard/steps/drop) are serialized using the engine's function/AST serializers.

Example roundtrip:

```
setq(root, create('agent'))
setAttribute(root, 'plan', p)
treeSave(root, 'agent.json')
setq(loaded, treeLoad('agent.json'))
getProp(getAttribute(loaded, 'plan'), 'name') // 'Thermostat'
```

## One-shot plan execution helpers

These helpers run a single plan instance with per-run isolation. They clone the current runtime, rebind the plan's functions to the cloned runtime, and optionally accept per-instance variables.

### runPlanOnce

runPlanOnce(plan) -> true

- Runs the plan once, executing steps if trigger+guard allow.
- Returns true on successful execution; errors if a step fails.

```
runPlanOnce(p)
```

### runPlanOnceBDI

runPlanOnceBDI(plan[, varsMap]) -> true|false

- BDI evaluation mode: requires Trigger and Guard, respects Drop.
- varsMap (optional): map of instance-scope variables available to the run.
- Returns true if executed, false if no-op (e.g., trigger/guard prevented run).

```
runPlanOnceBDI(p)
runPlanOnceBDI(p, map('min', 68, 'max', 72))
```

### runPlanOnceEx

runPlanOnceEx(plan[, mode][, varsMap]) -> true|false

- Extended one-shot helper.
- mode: string, defaults to "bdi". Supported values:
  - "bdi": require Trigger && Guard, respect Drop
  - "dry-run": evaluate using BDI rules but do not execute steps; returns whether it would execute
  - unknown values default to "bdi"
- varsMap: optional per-run variables map

```
runPlanOnceEx(p)                                 // bdi
runPlanOnceEx(p, 'dry-run')                      // no side effects, just eligibility
runPlanOnceEx(p, 'bdi', map('min', 60))          // with instance variables
```

## Agent APIs

Agents are lightweight schedulers that can host one or more plans with polling and simple belief storage.

### Basic agent lifecycle

- agentNew([maxConcurrent],[pollSeconds]) -> agent
  - maxConcurrent: number (default 1)
  - pollSeconds: number (default 3)

- agentRegister(agent, plan) -> true
- agentStart(agent) -> true
- agentStop(agent) -> true

Example:

```
setq(agent, agentNew(2, 5))
agentRegister(agent, p)
agentStart(agent)
// ... later
agentStop(agent)
```

### Named agent registry (global helpers)

Manage agents by name—useful for REST control surfaces and dashboards.

- agentStartNamed(name, plan[, maxConcurrent=1][, pollSeconds=3]) -> true
- agentStopNamed(name) -> true
- agentList() -> array of names
- agentPublish(name) -> true|false  // nudge scheduler loop
- agentBelief(name, key, value) -> true|false // set a belief and nudge
- belief(name, key) -> value|null

Examples:

```
agentStartNamed('heating', p, 1, 3)
agentPublish('heating')
agentBelief('heating', 'roomTemp', 72)
belief('heating', 'roomTemp') // -> 72
agentList()                   // -> ['heating']
agentStopNamed('heating')
```

### Beliefs vs variables

- `getVariable(name)` looks up the current scope first, then the global scope; it does not read agent beliefs.
- To use the belief store with plans, call `belief(agentName, key)` inside your plan's trigger/guard/steps to read values, and `agentBelief(agentName, key, value)` to write values (e.g., from within a step). These target agents started via `agentStartNamed` (default registry).
- For one-shot execution (`runPlanOnce`, `runPlanOnceBDI`, `runPlanOnceEx`), prefer passing a `varsMap` to overlay per-run variables that your plan accesses via `getVariable(...)`. The `agentBelief(...)` API is not applied to these ephemeral agents.

Example using beliefs with a named agent:

```
// Build plan p (uses belief('thermostat', 'currentTemp') etc.)
agentStartNamed('thermostat', p)
agentBelief('thermostat', 'lower', 68)
agentBelief('thermostat', 'upper', 72)
agentBelief('thermostat', 'currentTemp', 65)
agentPublish('thermostat') // optional nudge
```

## Error modes and notes

- All functions validate argument counts and types; mismatches produce errors like `"runPlanOnce(plan)"` or `"first argument must be plan"`.
- `getProp`/`setProp` on `Plan` only accept the documented property names; unknown properties return an error.
- `runPlanOnceEx` treats unknown `mode` as `"bdi"`.
- When storing Plans in trees, ensure you use `setAttribute(node, 'key', plan)`—no special conversion is needed.

## Quick reference

- Plan constructor: `plan(name, params[], trigger, guard, steps[], drop)`
- Plan props: `name`, `params`, `trigger`, `guard`, `steps`, `drop`
- One-shot helpers: `runPlanOnce`, `runPlanOnceBDI`, `runPlanOnceEx`
- Agents: `agentNew`, `agentRegister`, `agentStart`, `agentStop`
- Named agents: `agentStartNamed`, `agentStopNamed`, `agentList`, `agentPublish`, `agentBelief`, `belief`
