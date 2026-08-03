package tests

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

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
		"declare(guard,'F', func(){ equal(1,1) })",
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

func TestAgentLifecycleModesAndActivationResults(t *testing.T) {
	ch.DefaultAgentStop("eventtester")
	ch.DefaultAgentStop("calltester")
	t.Cleanup(func() {
		ch.DefaultAgentStop("eventtester")
		ch.DefaultAgentStop("calltester")
	})

	rt := createNamedRuntime("plan_lifecycle")
	defer ch.UnregisterRuntime("plan_lifecycle")

	setup := strings.Join([]string{
		"declare(name,'S','LifecyclePlan')",
		"declare(params,'A', array())",
		"declare(trig,'F', func(){ belief('eventtester','ready') })",
		"declare(guard,'F', func(){ equal(1,1) })",
		"declare(step,'F', func(){ setStepResult(map('status','ran','source','activation')) })",
		"declare(steps,'A', array(step))",
		"declare(drop,'F', func(){ equal(1,0) })",
		"declareGlobal(pLifecycle,'P', plan(name, params, trig, guard, steps, drop))",
		"agentStartNamed('eventtester', pLifecycle, 1, 0, 'eventOnly')",
		"agentStartNamed('calltester', pLifecycle, 1, 0, 'callOnly')",
	}, "\n")
	if _, err := rt.ExecProgram(setup); err != nil {
		t.Fatalf("setup exec: %v", err)
	}

	info := ch.DefaultAgentGetInfo("eventtester")
	if info["lifecycle"] != ch.AgentLifecycleEvent || info["running"] != true || info["pollSeconds"] != float64(0) {
		t.Fatalf("unexpected event lifecycle info: %#v", info)
	}
	callInfo := ch.DefaultAgentGetInfo("calltester")
	if callInfo["lifecycle"] != ch.AgentLifecycleCallOnly || callInfo["running"] != false {
		t.Fatalf("unexpected call-only lifecycle info: %#v", callInfo)
	}

	if !ch.DefaultAgentSetBeliefQuiet("eventtester", "ready", ch.Bool(true)) {
		t.Fatal("failed to set eventtester belief")
	}
	beliefs := ch.DefaultAgentGetBeliefs("eventtester")
	if got, ok := beliefs["ready"].(ch.Bool); !ok || !bool(got) {
		t.Fatalf("expected ready belief true before activation, got %#v", beliefs["ready"])
	}
	results, ok := ch.DefaultAgentActivate("eventtester")
	if !ok || len(results) != 1 {
		t.Fatalf("expected one activation result, ok=%v results=%#v", ok, results)
	}
	if results[0]["status"] != "completed" || results[0]["executed"] != true {
		t.Fatalf("unexpected activation payload: %#v", results[0])
	}
	output, ok := results[0]["output"].(map[string]interface{})
	if !ok || output["status"] != "ran" || output["source"] != "activation" {
		t.Fatalf("unexpected activation output: %#v", results[0]["output"])
	}

	info = ch.DefaultAgentGetInfo("eventtester")
	last, ok := info["lastResult"].(map[string]interface{})
	if !ok || last["status"] != "completed" {
		t.Fatalf("expected lastResult in agent info, got %#v", info["lastResult"])
	}

	callResults, ok := ch.DefaultAgentActivate("calltester")
	if !ok || len(callResults) != 0 {
		t.Fatalf("call-only activation should not schedule, ok=%v results=%#v", ok, callResults)
	}
}

func TestSignalBeliefFeedActivatesEventOnlyAgent(t *testing.T) {
	ch.DefaultAgentStop("signaltester")
	t.Cleanup(func() { ch.DefaultAgentStop("signaltester") })

	rt := createNamedRuntime("signal_feed")
	defer ch.UnregisterRuntime("signal_feed")
	t.Cleanup(func() { _, _ = rt.ExecProgram("signalStopBeliefFeed('signalTempFeed')") })

	setup := strings.Join([]string{
		"declare(name,'S','SignalTemperaturePlan')",
		"declare(params,'A', array())",
		"declare(trig,'F', func(){ bigger(belief('signaltester','currentTemp'), belief('signaltester','upper')) })",
		"declare(guard,'F', func(){ equal(1,1) })",
		"declare(step,'F', func(){ setStepResult(map('action','cooling_on','source','signalFeed','temperature', belief('signaltester','currentTemp'))) })",
		"declare(steps,'A', array(step))",
		"declare(drop,'F', func(){ equal(1,0) })",
		"declareGlobal(pSignalTemperature,'P', plan(name, params, trig, guard, steps, drop))",
		"agentStartNamed('signaltester', pSignalTemperature, 1, 0, 'eventOnly')",
		"agentBelief('signaltester', 'upper', 72)",
		"signalRegister('signalTemp', 'static', map('value', 85))",
		"signalStartBeliefFeed('signalTempFeed', 'signalTemp', 'signaltester', 'currentTemp', 0.05)",
	}, "\n")
	if _, err := rt.ExecProgram(setup); err != nil {
		t.Fatalf("setup exec: %v", err)
	}

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		info := ch.DefaultAgentGetInfo("signaltester")
		if last, ok := info["lastResult"].(map[string]interface{}); ok && last["status"] == "completed" {
			output, ok := last["output"].(map[string]interface{})
			if !ok || output["action"] != "cooling_on" || output["source"] != "signalFeed" || output["temperature"] != float64(85) {
				t.Fatalf("unexpected signal feed output: %#v", last["output"])
			}
			beliefs := ch.DefaultAgentGetBeliefs("signaltester")
			if got, ok := beliefs["currentTemp"].(ch.Number); !ok || float64(got) != 85 {
				t.Fatalf("expected currentTemp belief 85, got %#v", beliefs["currentTemp"])
			}
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("timed out waiting for signal feed activation; info=%#v", ch.DefaultAgentGetInfo("signaltester"))
}

func TestSignalReadProviders(t *testing.T) {
	rt := createNamedRuntime("signal_providers")
	defer ch.UnregisterRuntime("signal_providers")

	tempPath := filepath.Join(t.TempDir(), "temp_input")
	if err := os.WriteFile(tempPath, []byte("21345\n"), 0o644); err != nil {
		t.Fatalf("write temp fixture: %v", err)
	}
	sysfsProgram := fmt.Sprintf("signalRegister('sysTemp', 'sysfs', map('path', '%s', 'scale', 0.001))\nsignalRead('sysTemp')", tempPath)
	value, err := rt.ExecProgram(sysfsProgram)
	if err != nil {
		t.Fatalf("sysfs signal exec: %v", err)
	}
	if got, ok := value.(ch.Number); !ok || float64(got) != 21.345 {
		t.Fatalf("expected sysfs temp 21.345, got %#v (%T)", value, value)
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":[{"rate":5.25}]}`))
	}))
	t.Cleanup(server.Close)
	httpProgram := fmt.Sprintf("signalRegister('fedRate', 'httpJson', map('url', '%s', 'path', 'data.0.rate'))\nsignalRead('fedRate')", server.URL)
	value, err = rt.ExecProgram(httpProgram)
	if err != nil {
		t.Fatalf("httpJson signal exec: %v", err)
	}
	if got, ok := value.(ch.Number); !ok || float64(got) != 5.25 {
		t.Fatalf("expected httpJson rate 5.25, got %#v (%T)", value, value)
	}
}
