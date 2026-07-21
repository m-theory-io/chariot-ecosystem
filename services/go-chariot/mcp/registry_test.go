package mcp

import (
	"context"
	"path/filepath"
	"runtime"
	"testing"

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

func TestRegistryAgentMissingReturnsError(t *testing.T) {
	service := newRegistryTestService()
	_, err := service.Call(context.Background(), RegistryCallInput{ID: "agent:missing", Action: "info"})
	if err == nil {
		t.Fatal("expected missing agent error")
	}
}
