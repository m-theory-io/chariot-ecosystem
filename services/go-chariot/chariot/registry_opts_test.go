package chariot

import "testing"

// Representative builtin per optional category.
var categoryProbes = map[Category]string{
	CategorySQL:       "sqlQuery",
	CategoryCouchbase: "cbQuery",
	CategoryListeners: "listen",
	CategoryAuth:      "authenticateUser",
	CategoryRBAC:      "findRole",
	CategorySignals:   "signalRegister",
}

// Core builtins that must survive every exclusion profile.
var coreProbes = []string{"add", "getEnv", "logPrint", "sleep", "createHostObject"}

func runtimeHas(rt *Runtime, name string) bool {
	_, ok := rt.GetRegisteredFunctions()[name]
	return ok
}

func TestRegisterAllDefaultIncludesAllCategories(t *testing.T) {
	rt := NewRuntime()
	RegisterAll(rt)

	for category, name := range categoryProbes {
		if !runtimeHas(rt, name) {
			t.Errorf("default RegisterAll missing %q (category %s)", name, category)
		}
	}
	for _, name := range coreProbes {
		if !runtimeHas(rt, name) {
			t.Errorf("default RegisterAll missing core builtin %q", name)
		}
	}
}

func TestRegisterAllWithoutSQL(t *testing.T) {
	rt := NewRuntime()
	RegisterAll(rt, Without(CategorySQL))

	if runtimeHas(rt, "sqlQuery") {
		t.Error("Without(CategorySQL) should omit sqlQuery")
	}
	if !runtimeHas(rt, "cbQuery") {
		t.Error("Without(CategorySQL) should keep Couchbase functions")
	}
	for _, name := range coreProbes {
		if !runtimeHas(rt, name) {
			t.Errorf("Without(CategorySQL) should keep core builtin %q", name)
		}
	}
}

func TestEmbeddedProfileExclusions(t *testing.T) {
	rt := NewRuntime()
	RegisterAll(rt, EmbeddedProfile())

	for category, name := range categoryProbes {
		if runtimeHas(rt, name) {
			t.Errorf("EmbeddedProfile should omit %q (category %s)", name, category)
		}
	}
	// Listener builtins moved out of RegisterSystem stay excluded together.
	for _, name := range []string{"listenNSQ", "listenerList", "listenerStatus", "listenerStop", "httpResponse", "registerResponseTopic", "listResponseTopics"} {
		if runtimeHas(rt, name) {
			t.Errorf("EmbeddedProfile should omit listener builtin %q", name)
		}
	}
	for _, name := range coreProbes {
		if !runtimeHas(rt, name) {
			t.Errorf("EmbeddedProfile should keep core builtin %q", name)
		}
	}
	// Plan/agent functions remain available in embedded mode.
	for _, name := range []string{"plan", "agentNew"} {
		if !runtimeHas(rt, name) {
			t.Errorf("EmbeddedProfile should keep plan/agent builtin %q", name)
		}
	}
}

func TestCloneRuntimePreservesExclusions(t *testing.T) {
	rt := NewRuntime()
	RegisterAll(rt, EmbeddedProfile())

	var clone *Runtime
	err := rt.WithExecution(func() error {
		clone = rt.CloneRuntime()
		return nil
	})
	if err != nil {
		t.Fatalf("CloneRuntime failed: %v", err)
	}

	for category, name := range categoryProbes {
		if runtimeHas(clone, name) {
			t.Errorf("clone re-introduced %q (category %s)", name, category)
		}
	}
	for _, name := range coreProbes {
		if !runtimeHas(clone, name) {
			t.Errorf("clone missing core builtin %q", name)
		}
	}
}

func TestMasterRegistryMergeOnly(t *testing.T) {
	full := NewRuntime()
	RegisterAll(full)

	restricted := NewRuntime()
	RegisterAll(restricted, EmbeddedProfile())

	// A restricted registration must never shrink the master registry.
	for _, name := range []string{"sqlQuery", "cbQuery", "listen"} {
		if !HasFunction(name) {
			t.Errorf("master registry lost %q after restricted RegisterAll", name)
		}
	}
}
