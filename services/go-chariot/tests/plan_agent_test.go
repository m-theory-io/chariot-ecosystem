package tests

import (
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
		"declare(guard,'F', func(){ True })",
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
	// Ensure step2 did not run
	v, _ := rt.GetVariable("flag")
	if num, ok := v.(ch.Number); !ok || int(num) != 1 {
		t.Fatalf("expected flag=1, got %v", v)
	}
}
