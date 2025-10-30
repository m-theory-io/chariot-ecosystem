package tests

import (
	"testing"

	ch "github.com/bhouse1273/chariot-ecosystem/services/go-chariot/chariot"
)

// Verify getProp/setProp work for Plan fields
func TestPlan_GetSetProp(t *testing.T) {
	rt := createNamedRuntime("plan_props")
	defer ch.UnregisterRuntime("plan_props")

	code := []string{
		"declare(trig,'F', func(){ True })",
		"declare(guard,'F', func(){ True })",
		"declare(step1,'F', func(){ True })",
		"declare(steps,'A', array(step1))",
		"declare(drop,'F', func(){ False })",
		"declare(p,'P', plan('Thermostat', array('min','max'), trig, guard, steps, drop))",
		"// update name and params",
		"setProp(p,'name','ThermostatV2')",
		"setProp(p,'params', array('lo','hi'))",
		"p",
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
		t.Fatalf("exec error: %v", err)
	}
	native := ch.ConvertToNativeJSON(val)
	m, ok := native.(map[string]interface{})
	if !ok {
		t.Fatalf("expected native map, got %T", native)
	}
	if typ, _ := m["_type"].(string); typ != "plan" {
		t.Fatalf("expected _type 'plan', got %v", m["_type"])
	}
	if nm, _ := m["name"].(string); nm != "ThermostatV2" {
		t.Fatalf("expected name ThermostatV2, got %v", nm)
	}
	// params may be []string or []interface{} depending on converter
	switch pr := m["params"].(type) {
	case []string:
		if len(pr) != 2 || pr[0] != "lo" || pr[1] != "hi" {
			t.Fatalf("expected params ['lo','hi'], got %v", pr)
		}
	case []interface{}:
		if len(pr) != 2 || pr[0] != "lo" || pr[1] != "hi" {
			t.Fatalf("expected params ['lo','hi'], got %v", pr)
		}
	default:
		t.Fatalf("unexpected params type %T: %v", m["params"], m["params"])
	}
}

// Verify Plan serialization via GetVariableNative structure
func TestPlan_Serialize_ToNative(t *testing.T) {
	rt := createNamedRuntime("plan_native")
	defer ch.UnregisterRuntime("plan_native")

	code := []string{
		"declare(trig,'F', func(){ True })",
		"declare(guard,'F', func(){ True })",
		"declare(step,'F', func(){ True })",
		"declare(steps,'A', array(step))",
		"declare(drop,'F', func(){ False })",
		"declare(p,'P', plan('SerializeMe', array('x','y'), trig, guard, steps, drop))",
		"p",
	}

	program := ""
	for i, ln := range code {
		if i > 0 {
			program += "\n"
		}
		program += ln
	}
	pVal, err := rt.ExecProgram(program)
	if err != nil {
		t.Fatalf("exec error: %v", err)
	}
	// Convert the returned plan value to native for inspection
	native := ch.ConvertToNativeJSON(pVal)
	m, ok := native.(map[string]interface{})
	if !ok {
		t.Fatalf("expected native map, got %T", native)
	}
	if typ, _ := m["_type"].(string); typ != "plan" {
		t.Fatalf("expected _type 'plan', got %v", m["_type"])
	}
	if nm, _ := m["name"].(string); nm != "SerializeMe" {
		t.Fatalf("expected name SerializeMe, got %v", nm)
	}
	if params, _ := m["params"].([]string); params != nil {
		// Some JSON marshallers may decode to []interface{} instead of []string
	}
}
