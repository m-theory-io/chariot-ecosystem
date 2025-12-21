package tests

import (
	"strings"
	"testing"

	ch "github.com/bhouse1273/chariot-ecosystem/services/go-chariot/chariot"
)

// Minimal happy-path: trigger/guard true, three steps, no drop
func TestPlan_RunOnce_Happy(t *testing.T) {
	rt := createNamedRuntime("plan_test")
	defer ch.UnregisterRuntime("plan_test")

	// Build trigger/guard/steps/drop as functions in Chariot code
	code := []string{
		"declare(name,'S','PreventAuthDenials')",
		"declare(params,'A', array('serviceLine','payer'))",
		"declare(trig,'F', func(){ True })",
		"declare(guard,'F', func(){ equal(1,1) })",
		"declare(step1,'F', func(){ setq(x,1); True })",
		"declare(step2,'F', func(){ setq(x, add(x,1)); True })",
		"declare(step3,'F', func(){ setq(x, add(x,1)); True })",
		"declare(steps,'A', array(step1, step2, step3))",
		"declare(drop,'F', func(){ False })",
		"declare(p,'P', plan(name, params, trig, guard, steps, drop))",
		"runPlanOnce(p)",
	}
	program := ""
	for i, ln := range code {
		if i > 0 {
			program += "\n"
		}
		program += ln
	}
	val, err := rt.ExecProgram(program)
	if err != nil {
		t.Fatalf("exec: %v", err)
	}
	if b, ok := val.(ch.Bool); !ok || !bool(b) {
		t.Fatalf("expected true, got %v (%T)", val, val)
	}
}

// Drop before second step
func TestPlan_DropCondition(t *testing.T) {
	rt := createNamedRuntime("plan_drop")
	defer ch.UnregisterRuntime("plan_drop")

	code := []string{
		"declareGlobal(flag,'N', 0)",
		"declare(name,'S','Dropper')",
		"declare(params,'A', array())",
		"declare(trig,'F', func(){ True })",
		"declare(guard,'F', func(){ True })",
		"declare(step1,'F', func(){ setq(flag,1); True })",
		"declare(step2,'F', func(){ setq(flag, add(flag,1)); True })",
		"declare(steps,'A', array(step1, step2))",
		"declare(drop,'F', func(){ bigger(getVariable('flag'), 0) })",
		"declare(p,'P', plan(name, params, trig, guard, steps, drop))",
		"runPlanOnce(p)",
	}
	program := ""
	for i, ln := range code {
		if i > 0 {
			program += "\n"
		}
		program += ln
	}
	val, err := rt.ExecProgram(program)
	if err != nil {
		t.Fatalf("exec: %v", err)
	}
	if b, ok := val.(ch.Bool); !ok || !bool(b) {
		t.Fatalf("expected true, got %v (%T)", val, val)
	}
	// Ensure step2 did not run by inspecting global flag set by step1
	if v, _ := rt.GetVariable("flag"); v == nil || v == ch.DBNull || int(v.(ch.Number)) != 1 {
		t.Fatalf("expected global flag=1, got %v", v)
	}
}

// Mirrors the Agents tab "Run Once" invocation where plan "p" is executed with
// varsMap payload {currentTemp:80, upper:74, lower:70} and agentName "thermostat".
func TestPlan_RunOnce_WithAgentBeliefs(t *testing.T) {
	rt := createNamedRuntime("plan_runonce_agent")
	defer ch.UnregisterRuntime("plan_runonce_agent")

	setup := strings.Join([]string{
		"declare(name,'S','Thermostat')",
		"declare(params,'A', array())",
		"declare(trig,'F', func(){ or(smaller(belief('thermostat','currentTemp'), belief('thermostat','lower')), bigger(belief('thermostat','currentTemp'), belief('thermostat','upper'))) })",
		"declare(guard,'F', func(){ equal(1,1) })",
		"declare(step,'F', func(){ logPrint('thermostat running'); True })",
		"declare(steps,'A', array(step))",
		"declare(drop,'F', func(){ False })",
		"declareGlobal(p,'P', plan(name, params, trig, guard, steps, drop))",
		"agentStartNamed('thermostat', p)",
	}, "\n")
	if _, err := rt.ExecProgram(setup); err != nil {
		t.Fatalf("setup exec: %v", err)
	}
	if val, ok := rt.GetVariable("p"); ok {
		if plan, ok := val.(*ch.Plan); ok {
			if len(plan.Steps) == 0 {
				t.Fatalf("plan p has no steps")
			}
		} else {
			t.Fatalf("variable p not plan, got %T", val)
		}
	} else {
		t.Fatalf("plan p not found in runtime")
	}

	// Without beliefs, trigger should fail and runPlanOnceEx returns false
	noBeliefs := strings.Join([]string{
		"setq(__planToRun, getVariable(\"p\"))",
		"runPlanOnceEx(__planToRun, 'bdi')",
	}, "\n")
	val, err := rt.ExecProgram(noBeliefs)
	if err != nil {
		t.Fatalf("no-beliefs exec: %v", err)
	}
	if b, ok := val.(ch.Bool); !ok || bool(b) {
		t.Fatalf("expected false before beliefs, got %v (%T)", val, val)
	}

	if !ch.DefaultAgentBelief("thermostat", "currentTemp", ch.Number(80)) {
		t.Fatalf("failed to set currentTemp belief")
	}
	if !ch.DefaultAgentBelief("thermostat", "upper", ch.Number(74)) {
		t.Fatalf("failed to set upper belief")
	}
	if !ch.DefaultAgentBelief("thermostat", "lower", ch.Number(70)) {
		t.Fatalf("failed to set lower belief")
	}

	triggerEval := "or(smaller(belief('thermostat','currentTemp'), belief('thermostat','lower')), bigger(belief('thermostat','currentTemp'), belief('thermostat','upper')))"
	triggerVal, err := rt.ExecProgram(triggerEval)
	if err != nil {
		t.Fatalf("trigger eval error: %v", err)
	}
	if b, ok := triggerVal.(ch.Bool); !ok || !bool(b) {
		t.Fatalf("expected trigger expression true, got %v (%T)", triggerVal, triggerVal)
	}
	// Simulate Agents tab request payload
	runOnce := strings.Join([]string{
		"setq(__planToRun, getVariable(\"p\"))",
		"runPlanOnceEx(__planToRun, 'bdi', map('currentTemp',80,'upper',74,'lower',70))",
	}, "\n")
	val, err = rt.ExecProgram(runOnce)
	if err != nil {
		t.Fatalf("run-once exec: %v", err)
	}
	if b, ok := val.(ch.Bool); !ok || !bool(b) {
		t.Fatalf("expected true after beliefs, got %v (%T)", val, val)
	}
	bd := strings.Join([]string{
		"setq(__planToRun2, getVariable(\"p\"))",
		"runPlanOnceBDI(__planToRun2, map('currentTemp',80,'upper',74,'lower',70))",
	}, "\n")
	val, err = rt.ExecProgram(bd)
	if err != nil {
		t.Fatalf("runPlanOnceBDI exec: %v", err)
	}
	if b, ok := val.(ch.Bool); !ok || !bool(b) {
		t.Fatalf("expected true from runPlanOnceBDI, got %v (%T)", val, val)
	}
}
