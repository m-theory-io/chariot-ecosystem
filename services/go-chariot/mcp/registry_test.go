package mcp

import (
	"context"
	"errors"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/bhouse1273/chariot-ecosystem/services/go-chariot/chariot"
	cfg "github.com/bhouse1273/chariot-ecosystem/services/go-chariot/configs"
)

func repoRootForRegistryTest(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(file), "../../.."))
}

func configureRegistryTestPaths(t *testing.T) {
	t.Helper()
	root := repoRootForRegistryTest(t)
	cfg.ChariotConfig.DataPath = filepath.Join(root, "services/go-chariot/data")
	cfg.ChariotConfig.TreePath = filepath.Join(root, "services/go-chariot/data/trees")
}

func configureLocalDataRegistryTestPaths(t *testing.T) {
	t.Helper()
	root := repoRootForRegistryTest(t)
	cfg.ChariotConfig.DataPath = filepath.Join(root, "data/go-chariot")
	cfg.ChariotConfig.TreePath = filepath.Join(root, "data/go-chariot/trees")
}

func newRegistryTestService() *RegistryService {
	return NewRegistryService(func() *chariot.Runtime {
		rt := chariot.NewRuntime()
		chariot.RegisterAll(rt)
		return rt
	})
}

func TestRegistryListIncludesDecisionAgentTree(t *testing.T) {
	configureRegistryTestPaths(t)
	service := newRegistryTestService()

	items, err := service.List(context.Background())
	if err != nil {
		t.Fatalf("List: %v", err)
	}

	for _, item := range items {
		if item.ID == "tree:decisionAgent1" && item.Kind == "programTree" && item.Path == "decisionAgent1.json" {
			return
		}
	}
	t.Fatalf("expected tree:decisionAgent1 in registry list, got %#v", items)
}

func TestRegistryDescribeDecisionAgentTreeCallables(t *testing.T) {
	configureRegistryTestPaths(t)
	service := newRegistryTestService()

	description, err := service.Describe(context.Background(), "tree:decisionAgent1")
	if err != nil {
		t.Fatalf("Describe: %v", err)
	}
	callables, ok := description["callables"].([]RegistryItem)
	if !ok {
		t.Fatalf("expected []RegistryItem callables, got %#v", description["callables"])
	}

	for _, item := range callables {
		if item.ID == "tree:decisionAgent1.rules.ageFilter" && item.Callable && len(item.Parameters) == 1 && item.Parameters[0] == "profile" {
			return
		}
	}
	t.Fatalf("expected rules.ageFilter callable, got %#v", callables)
}

func TestRegistryCallDecisionAgentTreeFunction(t *testing.T) {
	configureRegistryTestPaths(t)
	service := newRegistryTestService()

	result, err := service.Call(context.Background(), RegistryCallInput{
		ID: "tree:decisionAgent1.rules.ageFilter",
		Args: map[string]any{
			"profile": map[string]any{"age": float64(42)},
		},
	})
	if err != nil {
		t.Fatalf("Call: %v", err)
	}
	if result.Kind != "treeFunction" || result.Result != true {
		t.Fatalf("expected true treeFunction result, got %#v", result)
	}
}

func TestRegistryCallDecisionAgent2DecisionRequest(t *testing.T) {
	configureLocalDataRegistryTestPaths(t)
	service := newRegistryTestService()

	result, err := service.Call(context.Background(), RegistryCallInput{
		ID: "tree:decisionAgent2.handlers.onDecisionRequest",
		Args: map[string]any{
			"req": map[string]any{
				"profile": map[string]any{
					"age":         float64(30),
					"debt":        float64(7000.25),
					"is_employed": true,
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("Call: %v", err)
	}
	if result.Kind != "treeFunction" {
		t.Fatalf("expected treeFunction result, got %#v", result)
	}
	payload, ok := result.Result.(map[string]any)
	if !ok {
		t.Fatalf("expected map result, got %#v", result.Result)
	}
	if payload["decision"] != "approved" {
		t.Fatalf("expected approved decision, got %#v", payload)
	}
	offer, ok := payload["offer"].(string)
	if !ok || !strings.Contains(offer, "$1,000.00") || !strings.Contains(offer, "$2,500.00") {
		t.Fatalf("expected formatted currency in offer, got %#v", payload["offer"])
	}
}

func TestRegistryAgentMissingReturnsError(t *testing.T) {
	service := newRegistryTestService()
	_, err := service.Call(context.Background(), RegistryCallInput{ID: "agent:missing", Action: "info"})
	if err == nil {
		t.Fatal("expected missing agent error")
	}
}

func TestRegistryAgentRunPlanOnceSetsBeliefsAndReturnsExecution(t *testing.T) {
	chariot.DefaultAgentStop("thermostat")
	t.Cleanup(func() { chariot.DefaultAgentStop("thermostat") })

	rt := chariot.NewRuntime()
	chariot.RegisterAll(rt)
	setup := strings.Join([]string{
		"declare(name,'S','Thermostat')",
		"declare(params,'A', array())",
		"declare(trig,'F', func(){ or(smaller(belief('thermostat','currentTemp'), belief('thermostat','lower')), bigger(belief('thermostat','currentTemp'), belief('thermostat','upper'))) })",
		"declare(guard,'F', func(){ equal(1,1) })",
		"declare(step,'F', func(){ logPrint('turn A/C on'); setStepResult(map('action', 'cooling_on', 'message', 'turn A/C on')) })",
		"declare(steps,'A', array(step))",
		"declare(drop,'F', func(){ False })",
		"declareGlobal(pThermostat,'P', plan(name, params, trig, guard, steps, drop))",
		"agentStartNamed('thermostat', pThermostat)",
	}, "\n")
	if _, err := rt.ExecProgram(setup); err != nil {
		t.Fatalf("setup exec: %v", err)
	}

	service := NewRegistryService(func() *chariot.Runtime { return rt })
	result, err := service.Call(context.Background(), RegistryCallInput{
		ID:     "agent:thermostat",
		Action: "runPlanOnce",
		Input: map[string]any{
			"plan": "pThermostat",
			"beliefs": map[string]any{
				"currentTemp": float64(85),
				"upper":       float64(74),
				"lower":       float64(70),
			},
		},
	})
	if err != nil {
		t.Fatalf("Call: %v", err)
	}
	if result.Executed == nil || !*result.Executed {
		t.Fatalf("expected executed true, got %#v", result)
	}
	payload, ok := result.Result.(map[string]any)
	if !ok {
		t.Fatalf("expected result payload map, got %#v", result.Result)
	}
	if payload["plan"] != "Thermostat" || payload["planVariable"] != "pThermostat" || payload["agent"] != "thermostat" || payload["status"] != "completed" || payload["executed"] != true {
		t.Fatalf("unexpected run payload: %#v", payload)
	}
	output, ok := payload["output"].(map[string]interface{})
	if !ok || output["action"] != "cooling_on" || output["message"] != "turn A/C on" {
		t.Fatalf("expected structured output, got %#v", payload["output"])
	}
	steps, ok := payload["steps"].([]map[string]interface{})
	if !ok || len(steps) != 1 || steps[0]["status"] != "completed" {
		t.Fatalf("expected completed step result, got %#v", payload["steps"])
	}
	stepResult, ok := steps[0]["result"].(map[string]interface{})
	if !ok || stepResult["action"] != "cooling_on" || stepResult["message"] != "turn A/C on" {
		t.Fatalf("expected structured step result, got %#v", steps[0]["result"])
	}
	diagnostics, ok := payload["diagnostics"].(map[string]any)
	if !ok {
		t.Fatalf("expected diagnostics, got %#v", payload["diagnostics"])
	}
	logMessages, ok := diagnostics["logMessages"].([]string)
	if !ok {
		t.Fatalf("expected logMessages, got %#v", diagnostics["logMessages"])
	}
	foundACLog := false
	for _, message := range logMessages {
		if message == "turn A/C on" {
			foundACLog = true
			break
		}
	}
	if !foundACLog {
		t.Fatalf("expected turn A/C on log, got %#v", logMessages)
	}
	beliefs := chariot.DefaultAgentGetBeliefs("thermostat")
	if got, ok := beliefs["currentTemp"].(chariot.Number); !ok || float64(got) != 85 {
		t.Fatalf("expected currentTemp belief 85, got %#v", beliefs["currentTemp"])
	}
}

func TestRegistryAgentRunPlanOnceReturnsSkippedReason(t *testing.T) {
	chariot.DefaultAgentStop("thermostat")
	t.Cleanup(func() { chariot.DefaultAgentStop("thermostat") })

	rt := chariot.NewRuntime()
	chariot.RegisterAll(rt)
	setup := strings.Join([]string{
		"declare(name,'S','Thermostat')",
		"declare(params,'A', array())",
		"declare(trig,'F', func(){ bigger(belief('thermostat','currentTemp'), belief('thermostat','upper')) })",
		"declare(guard,'F', func(){ equal(1,1) })",
		"declare(step,'F', func(){ logPrint('turn A/C on'); True })",
		"declare(steps,'A', array(step))",
		"declare(drop,'F', func(){ equal(1,0) })",
		"declareGlobal(pThermostat,'P', plan(name, params, trig, guard, steps, drop))",
		"agentStartNamed('thermostat', pThermostat)",
	}, "\n")
	if _, err := rt.ExecProgram(setup); err != nil {
		t.Fatalf("setup exec: %v", err)
	}

	service := NewRegistryService(func() *chariot.Runtime { return rt })
	result, err := service.Call(context.Background(), RegistryCallInput{
		ID:     "agent:thermostat",
		Action: "runPlanOnce",
		Input: map[string]any{
			"plan": "pThermostat",
			"mode": "bdi",
			"beliefs": map[string]any{
				"currentTemp": float64(70),
				"upper":       float64(74),
			},
		},
	})
	if err != nil {
		t.Fatalf("Call: %v", err)
	}
	if result.Executed == nil || *result.Executed {
		t.Fatalf("expected executed false, got %#v", result)
	}
	payload, ok := result.Result.(map[string]any)
	if !ok {
		t.Fatalf("expected result payload map, got %#v", result.Result)
	}
	if payload["status"] != "skipped" || payload["reason"] != "trigger_false" || payload["executed"] != false {
		t.Fatalf("unexpected skipped payload: %#v", payload)
	}
}

func TestRegistryAgentSetBeliefReturnsScheduledExecution(t *testing.T) {
	chariot.DefaultAgentStop("mcpschedule")
	t.Cleanup(func() { chariot.DefaultAgentStop("mcpschedule") })

	rt := chariot.NewRuntime()
	chariot.RegisterAll(rt)
	setup := strings.Join([]string{
		"declare(name,'S','MCPScheduledPlan')",
		"declare(params,'A', array())",
		"declare(trig,'F', func(){ belief('mcpschedule','ready') })",
		"declare(guard,'F', func(){ equal(1,1) })",
		"declare(step,'F', func(){ setStepResult(map('status','ran','source','mcp_setBelief')) })",
		"declare(steps,'A', array(step))",
		"declare(drop,'F', func(){ equal(1,0) })",
		"declareGlobal(pMCPScheduled,'P', plan(name, params, trig, guard, steps, drop))",
		"agentStartNamed('mcpschedule', pMCPScheduled)",
	}, "\n")
	if _, err := rt.ExecProgram(setup); err != nil {
		t.Fatalf("setup exec: %v", err)
	}

	service := NewRegistryService(func() *chariot.Runtime { return rt })
	result, err := service.Call(context.Background(), RegistryCallInput{
		ID:     "agent:mcpschedule",
		Action: "setBelief",
		Input:  map[string]any{"ready": true},
	})
	if err != nil {
		t.Fatalf("Call: %v", err)
	}
	payload, ok := result.Result.(map[string]any)
	if !ok {
		t.Fatalf("expected setBelief result map, got %#v", result.Result)
	}
	executions, ok := payload["executions"].([]map[string]interface{})
	if !ok || len(executions) != 1 {
		t.Fatalf("expected one execution result, got %#v", payload["executions"])
	}
	if executions[0]["status"] != "completed" || executions[0]["executed"] != true {
		t.Fatalf("unexpected execution payload: %#v", executions[0])
	}
	output, ok := executions[0]["output"].(map[string]interface{})
	if !ok || output["status"] != "ran" || output["source"] != "mcp_setBelief" {
		t.Fatalf("unexpected execution output: %#v", executions[0]["output"])
	}

	description, err := service.Describe(context.Background(), "agent:mcpschedule")
	if err != nil {
		t.Fatalf("Describe: %v", err)
	}
	info, ok := description["info"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected agent info map, got %#v", description["info"])
	}
	last, ok := info["lastResult"].(map[string]interface{})
	if !ok || last["status"] != "completed" {
		t.Fatalf("expected lastResult in describe info, got %#v", info["lastResult"])
	}
}

func TestRegistryOperationsWaitForRuntimeGuard(t *testing.T) {
	configureRegistryTestPaths(t)
	operations := map[string]func(*RegistryService) error{
		"list": func(s *RegistryService) error { _, err := s.List(context.Background()); return err },
		"describeTreeFunction": func(s *RegistryService) error {
			_, err := s.Describe(context.Background(), "tree:decisionAgent1.rules.ageFilter")
			return err
		},
		"callTreeDescription": func(s *RegistryService) error {
			_, err := s.Call(context.Background(), RegistryCallInput{ID: "tree:decisionAgent1"})
			return err
		},
		"callTreeFunction": func(s *RegistryService) error {
			_, err := s.Call(context.Background(), RegistryCallInput{ID: "tree:decisionAgent1.rules.ageFilter", Args: map[string]any{"profile": map[string]any{"age": float64(42)}}})
			return err
		},
		"agentPlanLookup": func(s *RegistryService) error {
			_, err := s.Call(context.Background(), RegistryCallInput{ID: "agent:missing", Action: "runPlanOnce", Input: map[string]any{"plan": "missingPlan"}})
			if err == nil || !strings.Contains(err.Error(), "not found") {
				t.Errorf("expected missing plan error, got %v", err)
			}
			return nil
		},
	}
	for name, operation := range operations {
		t.Run(name, func(t *testing.T) {
			rt := chariot.NewRuntime()
			chariot.RegisterAll(rt)
			acquired, release, ownerDone := make(chan struct{}), make(chan struct{}), make(chan struct{})
			go func() {
				_ = rt.WithExecution(func() error { close(acquired); <-release; return nil })
				close(ownerDone)
			}()
			<-acquired
			requested := make(chan struct{}, 1)
			service := NewRegistryService(func() *chariot.Runtime { requested <- struct{}{}; return rt })
			done := make(chan error, 1)
			go func() { done <- operation(service) }()
			<-requested
			select {
			case err := <-done:
				t.Errorf("operation bypassed runtime guard: %v", err)
				close(release)
				<-ownerDone
				return
			case <-time.After(25 * time.Millisecond):
			}
			close(release)
			<-ownerDone
			select {
			case err := <-done:
				if err != nil {
					t.Fatal(err)
				}
			case <-time.After(3 * time.Second):
				t.Fatal("registry operation deadlocked, possibly a nested execution guard")
			}
		})
	}
}

func TestRegistryTreeCallRestoresTemporaryBindings(t *testing.T) {
	configureRegistryTestPaths(t)
	rt := chariot.NewRuntime()
	chariot.RegisterAll(rt)
	scope := rt.GetCurrentScope()
	rt.SetVariable("__mcp_fn", chariot.Str("existing-function"))
	rt.SetVariable("__mcp_arg_profile", chariot.Str("existing-argument"))
	service := NewRegistryService(func() *chariot.Runtime { return rt })
	for _, profile := range []any{map[string]any{"age": float64(42)}, make(chan int)} {
		_, err := service.Call(context.Background(), RegistryCallInput{
			ID: "tree:decisionAgent1.rules.ageFilter", Args: map[string]any{"profile": profile},
		})
		if _, valid := profile.(map[string]any); valid && err != nil {
			t.Fatal(err)
		} else if !valid && err == nil {
			t.Fatal("expected argument conversion error")
		}
		if rt.GetCurrentScope() != scope {
			t.Fatal("tree call did not restore scope")
		}
		for name, want := range map[string]chariot.Value{
			"__mcp_fn": chariot.Str("existing-function"), "__mcp_arg_profile": chariot.Str("existing-argument"),
		} {
			if got, _ := rt.GetVariable(name); got != want {
				t.Fatalf("%s=%v, want %v", name, got, want)
			}
		}
	}
}

func TestRegistryCanceledWaitDoesNotStartTransaction(t *testing.T) {
	rt := chariot.NewRuntime()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	acquired, release, ownerDone := make(chan struct{}), make(chan struct{}), make(chan struct{})
	go func() {
		_ = rt.WithExecution(func() error { close(acquired); <-release; return nil })
		close(ownerDone)
	}()
	<-acquired
	requested := make(chan struct{}, 1)
	service := NewRegistryService(func() *chariot.Runtime { requested <- struct{}{}; return rt })
	done := make(chan error, 1)
	go func() {
		done <- service.withRuntime(ctx, func(rt *chariot.Runtime) error {
			rt.SetGlobalVariable("mustNotRun", chariot.Bool(true))
			return nil
		})
	}()
	<-requested
	cancel()
	close(release)
	<-ownerDone
	select {
	case err := <-done:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("error=%v, want context.Canceled", err)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("canceled operation did not finish after guard release")
	}
	if _, exists := rt.GetVariable("mustNotRun"); exists {
		t.Fatal("canceled operation mutated shared runtime")
	}
}
