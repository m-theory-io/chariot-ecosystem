package chariot

import (
	"fmt"
	"strings"
)

// Master function registry - all available built-in functions
var MasterFunctionRegistry = map[string]func(...Value) (Value, error){}

// Category identifies a group of built-in functions that can be
// conditionally excluded from registration via RegisterAll options.
type Category string

const (
	CategoryValues    Category = "values"
	CategoryFlow      Category = "flow"
	CategoryArray     Category = "array"
	CategoryCompare   Category = "compare"
	CategoryMath      Category = "math"
	CategoryDate      Category = "date"
	CategoryString    Category = "string"
	CategoryNode      Category = "node"
	CategoryFile      Category = "file"
	CategoryJSON      Category = "json"
	CategorySystem    Category = "system"
	CategoryListeners Category = "listeners"
	CategoryHost      Category = "host"
	CategorySQL       Category = "sql"
	CategoryCouchbase Category = "couchbase"
	CategoryETL       Category = "etl"
	CategoryTree      Category = "tree"
	CategoryCrypto    Category = "crypto"
	CategoryAuth      Category = "auth"
	CategoryRBAC      Category = "rbac"
	CategoryCSV       Category = "csv"
	CategoryMCP       Category = "mcp"
	CategoryKnapsack  Category = "knapsack"
	CategoryRL        Category = "rl"
	CategorySignals   Category = "signals"
	CategoryDispatch  Category = "dispatch"
	CategoryPlan      Category = "plan"
)

// registrationOrder maps each category to its registration function, in the
// order registration must run. CategoryDispatch (type-dispatched/polymorphic
// functions) must run after all of its dispatch targets are registered.
// This is a function rather than a package var to avoid an initialization
// cycle (RegisterAll -> registrationOrder -> RegisterPlanFunctions -> ... -> RegisterAll).
func registrationOrder() []struct {
	category Category
	register func(*Runtime)
} {
	return []struct {
		category Category
		register func(*Runtime)
	}{
		{CategoryValues, RegisterValues},
		{CategoryFlow, RegisterFlow},
		{CategoryArray, RegisterArray},
		{CategoryCompare, RegisterCompares},
		{CategoryMath, RegisterMath},
		{CategoryDate, RegisterDate},
		{CategoryString, RegisterString},
		{CategoryNode, RegisterNode},
		{CategoryFile, RegisterFile},
		{CategoryJSON, RegisterJSON},
		{CategorySystem, RegisterSystem},
		{CategoryListeners, RegisterListenerFunctions},
		{CategoryHost, RegisterHostFunctions},
		{CategorySQL, RegisterSQLFunctions},
		{CategoryCouchbase, RegisterCouchbaseFunctions},
		{CategoryETL, RegisterETLFunctions},
		{CategoryTree, RegisterTreeFunctions},
		{CategoryCrypto, RegisterCryptoFunctions},
		{CategoryAuth, RegisterAuthFuncs},
		{CategoryRBAC, RegisterRBACFuncs},
		{CategoryCSV, RegisterCSVFunctions},
		{CategoryMCP, RegisterMCPFunctions},
		{CategoryKnapsack, RegisterKnapsackFunctions},
		{CategoryRL, RegisterRLFunctions},
		{CategorySignals, RegisterSignalFunctions},
		{CategoryDispatch, RegisterTypeDispatchedFunctions},
		{CategoryPlan, RegisterPlanFunctions},
	}
}

type registerConfig struct {
	excluded map[Category]bool
}

// RegisterOption customizes which function categories RegisterAll loads.
type RegisterOption func(*registerConfig)

// Without excludes the given categories from registration.
func Without(categories ...Category) RegisterOption {
	return func(cfg *registerConfig) {
		for _, c := range categories {
			cfg.excluded[c] = true
		}
	}
}

// EmbeddedProfile excludes application-server categories that are
// inappropriate when Chariot is embedded as a scripting language in a host
// application: SQL and Couchbase (the host owns persistence), listeners
// (inbound HTTP/NSQ traffic), auth and RBAC (the host owns identity), and
// signals (server-side belief-feed pollers with process-global state).
func EmbeddedProfile() RegisterOption {
	return Without(
		CategorySQL,
		CategoryCouchbase,
		CategoryListeners,
		CategoryAuth,
		CategoryRBAC,
		CategorySignals,
	)
}

// RegisterAll registers built-in functions with the runtime. With no options
// every category is registered; options such as Without or EmbeddedProfile
// exclude categories. The options are remembered on the runtime so internal
// re-registrations (CloneRuntime, session creation) preserve the exclusions.
func RegisterAll(rt *Runtime, opts ...RegisterOption) {
	// Initialize master registry if not already done
	if MasterFunctionRegistry == nil {
		MasterFunctionRegistry = make(map[string]func(...Value) (Value, error))
	}

	cfg := &registerConfig{excluded: make(map[Category]bool)}
	for _, opt := range opts {
		opt(cfg)
	}
	rt.registerOptions = opts

	for _, entry := range registrationOrder() {
		if cfg.excluded[entry.category] {
			continue
		}
		entry.register(rt)
	}

	// Populate master registry from the runtime
	PopulateMasterRegistryFromRuntime(rt)
}

// RegisterToMaster - add a function to the master registry
func RegisterToMaster(name string, fn func(...Value) (Value, error)) {
	if MasterFunctionRegistry == nil {
		MasterFunctionRegistry = make(map[string]func(...Value) (Value, error))
	}
	MasterFunctionRegistry[name] = fn
}

// PopulateMasterRegistryFromRuntime - extract all functions from a runtime to
// master registry. This is merge-only: a runtime registered with exclusions
// (e.g. EmbeddedProfile) never shrinks the master registry, so role-based
// runtimes and the MCP registry are unaffected by restricted registrations.
func PopulateMasterRegistryFromRuntime(rt *Runtime) {
	registeredFuncs := rt.GetRegisteredFunctions()
	for name, fn := range registeredFuncs {
		MasterFunctionRegistry[name] = fn
	}
}

// GetMasterRegistry - get the master function registry
func GetMasterRegistry() map[string]func(...Value) (Value, error) {
	return MasterFunctionRegistry
}

// CreateRoleBasedRuntime - create a runtime with only specified built-in functions
func CreateRoleBasedRuntime(allowedFunctions []string) *Runtime {
	rt := NewRuntime()

	// Register only allowed functions from master registry
	for _, funcName := range allowedFunctions {
		if funcImpl, exists := MasterFunctionRegistry[funcName]; exists {
			rt.Register(funcName, funcImpl)
		}
		// Handle wildcards like "sql*"
		if strings.HasSuffix(funcName, "*") {
			prefix := strings.TrimSuffix(funcName, "*")
			for name, impl := range MasterFunctionRegistry {
				if strings.HasPrefix(name, prefix) {
					rt.Register(name, impl)
				}
			}
		}
	}

	// Always register core RBAC and auth functions for role management
	// These are needed for the role-based runtime to function
	RegisterRBACFuncs(rt)
	RegisterAuthFuncs(rt)

	return rt
}

// GetAvailableFunctions - get list of all available function names
func GetAvailableFunctions() []string {
	var functions []string
	for name := range MasterFunctionRegistry {
		functions = append(functions, name)
	}
	return functions
}

// ExpandFunctionWildcards - expand wildcard patterns into specific functions
func ExpandFunctionWildcards(patterns []string) []string {
	expandedFunctions := make(map[string]bool)

	for _, pattern := range patterns {
		if pattern == "*" {
			// Wildcard - add all functions
			for funcName := range MasterFunctionRegistry {
				expandedFunctions[funcName] = true
			}
		} else if strings.HasSuffix(pattern, "*") {
			// Prefix wildcard - add all functions with prefix
			prefix := strings.TrimSuffix(pattern, "*")
			for funcName := range MasterFunctionRegistry {
				if strings.HasPrefix(funcName, prefix) {
					expandedFunctions[funcName] = true
				}
			}
		} else {
			// Exact function name
			if _, exists := MasterFunctionRegistry[pattern]; exists {
				expandedFunctions[pattern] = true
			}
		}
	}

	// Convert to slice
	var result []string
	for funcName := range expandedFunctions {
		result = append(result, funcName)
	}

	return result
}

// GetFunctionsByCategory - get functions by category (prefix-based)
func GetFunctionsByCategory(category string) []string {
	var functions []string
	for name := range MasterFunctionRegistry {
		if strings.HasPrefix(name, category) {
			functions = append(functions, name)
		}
	}
	return functions
}

// HasFunction - check if a function exists in the master registry
func HasFunction(name string) bool {
	_, exists := MasterFunctionRegistry[name]
	return exists
}

// GetFunctionCount - get total number of available functions
func GetFunctionCount() int {
	return len(MasterFunctionRegistry)
}

// RegisterRuntimeFunctions - register runtime-specific functions for role-based runtime management
func RegisterRuntimeFunctions(rt *Runtime) {
	// These functions are available in runtimes that need to manage role-based runtimes

	rt.Register("getAvailableFunctions", func(args ...Value) (Value, error) {
		functions := GetAvailableFunctions()
		result := NewArray()
		for _, funcName := range functions {
			result.Append(Str(funcName))
		}
		return result, nil
	})

	rt.Register("expandFunctionWildcards", func(args ...Value) (Value, error) {
		if len(args) != 1 {
			return nil, fmt.Errorf("expandFunctionWildcards requires 1 argument: patterns array")
		}

		// Unwrap argument
		arg := args[0]
		if tvar, ok := arg.(ScopeEntry); ok {
			arg = tvar.Value
		}

		patternsArray, ok := arg.(*ArrayValue)
		if !ok {
			return nil, fmt.Errorf("argument must be an array of patterns, got %T", arg)
		}

		// Convert to string slice
		patterns := make([]string, 0, patternsArray.Length())
		for i := 0; i < patternsArray.Length(); i++ {
			elem := patternsArray.Get(i)
			if str, ok := elem.(Str); ok {
				patterns = append(patterns, string(str))
			}
		}

		// Expand wildcards
		expanded := ExpandFunctionWildcards(patterns)

		// Return as array
		result := NewArray()
		for _, funcName := range expanded {
			result.Append(Str(funcName))
		}
		return result, nil
	})

	rt.Register("getFunctionsByCategory", func(args ...Value) (Value, error) {
		if len(args) != 1 {
			return nil, fmt.Errorf("getFunctionsByCategory requires 1 argument: category")
		}

		// Unwrap argument
		arg := args[0]
		if tvar, ok := arg.(ScopeEntry); ok {
			arg = tvar.Value
		}

		category, ok := arg.(Str)
		if !ok {
			return nil, fmt.Errorf("argument must be a string category, got %T", arg)
		}

		functions := GetFunctionsByCategory(string(category))
		result := NewArray()
		for _, funcName := range functions {
			result.Append(Str(funcName))
		}
		return result, nil
	})

	rt.Register("hasFunction", func(args ...Value) (Value, error) {
		if len(args) != 1 {
			return nil, fmt.Errorf("hasFunction requires 1 argument: functionName")
		}

		// Unwrap argument
		arg := args[0]
		if tvar, ok := arg.(ScopeEntry); ok {
			arg = tvar.Value
		}

		funcName, ok := arg.(Str)
		if !ok {
			return nil, fmt.Errorf("argument must be a string function name, got %T", arg)
		}

		return Bool(HasFunction(string(funcName))), nil
	})

	rt.Register("getFunctionCount", func(args ...Value) (Value, error) {
		return Number(GetFunctionCount()), nil
	})
}
