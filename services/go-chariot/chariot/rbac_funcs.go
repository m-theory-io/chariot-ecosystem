package chariot

import (
	"errors"
	"fmt"
	"strings"
	"time"
)

func RegisterRBACFuncs(rt *Runtime) {
	// Core RBAC functions for role management

	// findRole - find a role by name in a roles collection
	rt.Register("findRole", func(args ...Value) (Value, error) {
		if len(args) != 2 {
			return nil, errors.New("findRole requires 2 arguments: rolesCollection and roleName")
		}

		// Unwrap arguments
		for i, arg := range args {
			if tvar, ok := arg.(ScopeEntry); ok {
				args[i] = tvar.Value
			}
		}

		// Get roles collection
		rolesCollection, ok := args[0].(TreeNode)
		if !ok {
			return nil, fmt.Errorf("first argument must be a TreeNode roles collection, got %T", args[0])
		}

		// Get role name
		roleName, ok := args[1].(Str)
		if !ok {
			return nil, fmt.Errorf("second argument must be a string roleName, got %T", args[1])
		}

		// Search through role children
		for _, child := range rolesCollection.GetChildren() {
			roleNode := child
			if nameAttr, exists := roleNode.GetAttribute("name"); exists {
				if nameStr, ok := nameAttr.(Str); ok && nameStr == roleName {
					return roleNode, nil
				}
			}
		}

		return nil, fmt.Errorf("role not found: %s", roleName)
	})

	// createRole - create a new role in the roles collection
	rt.Register("createRole", func(args ...Value) (Value, error) {
		if len(args) != 3 {
			return nil, errors.New("createRole requires 3 arguments: rolesCollection, roleName, functions")
		}

		// Unwrap arguments
		for i, arg := range args {
			if tvar, ok := arg.(ScopeEntry); ok {
				args[i] = tvar.Value
			}
		}

		// Get roles collection
		rolesCollection, ok := args[0].(TreeNode)
		if !ok {
			return nil, fmt.Errorf("first argument must be a TreeNode roles collection, got %T", args[0])
		}

		// Get role name
		roleName, ok := args[1].(Str)
		if !ok {
			return nil, fmt.Errorf("second argument must be a string roleName, got %T", args[1])
		}

		// Get functions (should be an array)
		functions, ok := args[2].(*ArrayValue)
		if !ok {
			return nil, fmt.Errorf("third argument must be an array of functions, got %T", args[2])
		}

		// Check if role already exists
		for _, child := range rolesCollection.GetChildren() {
			roleNode := child
			if nameAttr, exists := roleNode.GetAttribute("name"); exists {
				if nameStr, ok := nameAttr.(Str); ok && nameStr == roleName {
					return nil, fmt.Errorf("role already exists: %s", roleName)
				}
			}
		}

		// Create new role node
		newRole := NewJSONNode("{}")
		newRole.SetAttribute("name", roleName)
		newRole.SetAttribute("functions", functions)
		newRole.SetAttribute("createdAt", Str(time.Now().Format(time.RFC3339)))
		newRole.SetAttribute("updatedAt", Str(time.Now().Format(time.RFC3339)))

		// Add to roles collection
		rolesCollection.AddChild(newRole)

		return newRole, nil
	})

	// getUserRoles - get all roles for a user
	rt.Register("getUserRoles", func(args ...Value) (Value, error) {
		if len(args) != 1 {
			return nil, errors.New("getUserRoles requires 1 argument: user")
		}

		// Unwrap arguments
		for i, arg := range args {
			if tvar, ok := arg.(ScopeEntry); ok {
				args[i] = tvar.Value
			}
		}

		// Get user
		user, ok := args[0].(TreeNode)
		if !ok {
			return nil, fmt.Errorf("argument must be a TreeNode user, got %T", args[0])
		}

		// Get roles from user
		rolesAttr, exists := user.GetAttribute("roles")
		if !exists {
			return NewArray(), nil // Return empty array
		}

		rolesArray, ok := rolesAttr.(*ArrayValue)
		if !ok {
			return NewArray(), nil
		}

		return rolesArray, nil
	})

	// CORE FUNCTION: getFunctionWhitelist - get all allowed functions for a user based on roles
	rt.Register("getFunctionWhitelist", func(args ...Value) (Value, error) {
		if len(args) != 2 {
			return nil, errors.New("getFunctionWhitelist requires 2 arguments: user and rolesCollection")
		}

		// Unwrap arguments
		for i, arg := range args {
			if tvar, ok := arg.(ScopeEntry); ok {
				args[i] = tvar.Value
			}
		}

		// Get user
		user, ok := args[0].(TreeNode)
		if !ok {
			return nil, fmt.Errorf("first argument must be a TreeNode user, got %T", args[0])
		}

		// Get roles collection
		rolesCollection, ok := args[1].(TreeNode)
		if !ok {
			return nil, fmt.Errorf("second argument must be a TreeNode roles collection, got %T", args[1])
		}

		// Get user's roles
		userRolesAttr, exists := user.GetAttribute("roles")
		if !exists {
			return NewArray(), nil
		}

		userRolesArray, ok := userRolesAttr.(*ArrayValue)
		if !ok {
			return NewArray(), nil
		}

		// Collect all allowed functions
		allowedFunctions := make(map[string]bool)
		hasWildcard := false

		// Check each role
		for i := 0; i < userRolesArray.Length(); i++ {
			roleNameVal := userRolesArray.Get(i)
			if roleName, ok := roleNameVal.(Str); ok {
				// Find the role
				for _, roleNode := range rolesCollection.GetChildren() {
					if nameAttr, exists := roleNode.GetAttribute("name"); exists {
						if nameStr, ok := nameAttr.(Str); ok && nameStr == roleName {
							// Get functions from this role
							if functionsAttr, exists := roleNode.GetAttribute("functions"); exists {
								if functionsArray, ok := functionsAttr.(*ArrayValue); ok {
									for j := 0; j < functionsArray.Length(); j++ {
										funcVal := functionsArray.Get(j)
										if funcStr, ok := funcVal.(Str); ok {
											if funcStr == "*" {
												hasWildcard = true
											} else {
												allowedFunctions[string(funcStr)] = true
											}
										}
									}
								}
							}
						}
					}
				}
			}
		}

		// Create result array
		result := NewArray()

		if hasWildcard {
			// If user has wildcard access, return all available functions
			for funcName := range MasterFunctionRegistry {
				result.Append(Str(funcName))
			}
		} else {
			// Add all allowed functions
			for funcName := range allowedFunctions {
				result.Append(Str(funcName))
			}
		}

		return result, nil
	})

	// CORE FUNCTION: expandFunctionWildcards - expand wildcard patterns into specific functions
	rt.Register("expandFunctionWildcards", func(args ...Value) (Value, error) {
		if len(args) != 1 {
			return nil, errors.New("expandFunctionWildcards requires 1 argument: functionPatterns")
		}

		// Unwrap arguments
		for i, arg := range args {
			if tvar, ok := arg.(ScopeEntry); ok {
				args[i] = tvar.Value
			}
		}

		// Get function patterns
		patterns, ok := args[0].(*ArrayValue)
		if !ok {
			return nil, fmt.Errorf("argument must be an array of function patterns, got %T", args[0])
		}

		// Expand patterns
		expandedFunctions := make(map[string]bool)

		for i := 0; i < patterns.Length(); i++ {
			patternVal := patterns.Get(i)
			if pattern, ok := patternVal.(Str); ok {
				if pattern == "*" {
					// Wildcard - add all functions
					for funcName := range MasterFunctionRegistry {
						expandedFunctions[funcName] = true
					}
				} else if strings.HasSuffix(string(pattern), "*") {
					// Prefix wildcard - add all functions with prefix
					prefix := strings.TrimSuffix(string(pattern), "*")
					for funcName := range MasterFunctionRegistry {
						if strings.HasPrefix(funcName, prefix) {
							expandedFunctions[funcName] = true
						}
					}
				} else {
					// Exact function name
					if _, exists := MasterFunctionRegistry[string(pattern)]; exists {
						expandedFunctions[string(pattern)] = true
					}
				}
			}
		}

		// Create result array
		result := NewArray()
		for funcName := range expandedFunctions {
			result.Append(Str(funcName))
		}

		return result, nil
	})

	rt.Register("createRoleBasedRuntime", func(args ...Value) (Value, error) {
		if len(args) != 2 {
			return nil, errors.New("createRoleBasedRuntime requires 2 arguments: user and rolesCollection")
		}

		// Unwrap arguments
		for i, arg := range args {
			if tvar, ok := arg.(ScopeEntry); ok {
				args[i] = tvar.Value
			}
		}

		// Get user
		user, ok := args[0].(TreeNode)
		if !ok {
			return nil, fmt.Errorf("first argument must be a TreeNode user, got %T", args[0])
		}

		// Get roles collection
		rolesCollection, ok := args[1].(TreeNode)
		if !ok {
			return nil, fmt.Errorf("second argument must be a TreeNode roles collection, got %T", args[1])
		}

		// Get user's roles directly
		userRolesAttr, exists := user.GetAttribute("roles")
		if !exists {
			// User has no roles, create empty runtime
			return CreateRoleBasedRuntime([]string{}), nil
		}

		userRolesArray, ok := userRolesAttr.(*ArrayValue)
		if !ok {
			return CreateRoleBasedRuntime([]string{}), nil
		}

		// Collect all allowed functions
		allowedFunctions := make(map[string]bool)
		hasWildcard := false

		// Check each role
		for i := 0; i < userRolesArray.Length(); i++ {
			roleNameVal := userRolesArray.Get(i)
			if roleName, ok := roleNameVal.(Str); ok {
				// Find the role
				for _, roleNode := range rolesCollection.GetChildren() {
					if nameAttr, exists := roleNode.GetAttribute("name"); exists {
						if nameStr, ok := nameAttr.(Str); ok && nameStr == roleName {
							// Get functions from this role
							if functionsAttr, exists := roleNode.GetAttribute("functions"); exists {
								if functionsArray, ok := functionsAttr.(*ArrayValue); ok {
									for j := 0; j < functionsArray.Length(); j++ {
										funcVal := functionsArray.Get(j)
										if funcStr, ok := funcVal.(Str); ok {
											if funcStr == "*" {
												hasWildcard = true
											} else {
												allowedFunctions[string(funcStr)] = true
											}
										}
									}
								}
							}
						}
					}
				}
			}
		}

		// Convert to string slice
		var allowedFunctionsList []string
		if hasWildcard {
			// If user has wildcard access, get all available functions
			for funcName := range MasterFunctionRegistry {
				allowedFunctionsList = append(allowedFunctionsList, funcName)
			}
		} else {
			// Add all specific allowed functions
			for funcName := range allowedFunctions {
				allowedFunctionsList = append(allowedFunctionsList, funcName)
			}
		}

		// Use the registry function to create runtime
		newRuntime := CreateRoleBasedRuntime(allowedFunctionsList)

		return newRuntime, nil
	})

	// Role management functions (for admin operations)

	// updateRole - update an existing role
	rt.Register("updateRole", func(args ...Value) (Value, error) {
		if len(args) != 3 {
			return nil, errors.New("updateRole requires 3 arguments: rolesCollection, roleName, updates")
		}

		// Unwrap arguments
		for i, arg := range args {
			if tvar, ok := arg.(ScopeEntry); ok {
				args[i] = tvar.Value
			}
		}

		// Get roles collection
		rolesCollection, ok := args[0].(TreeNode)
		if !ok {
			return nil, fmt.Errorf("first argument must be a TreeNode roles collection, got %T", args[0])
		}

		// Get role name
		roleName, ok := args[1].(Str)
		if !ok {
			return nil, fmt.Errorf("second argument must be a string roleName, got %T", args[1])
		}

		// Get updates
		var updates map[string]Value
		switch updatesArg := args[2].(type) {
		case TreeNode:
			updates = updatesArg.GetAttributes()
		case *MapValue:
			updates = updatesArg.GetAttributes()
		default:
			return nil, fmt.Errorf("third argument must be a TreeNode or MapValue with updates, got %T", args[2])
		}

		// Find the role
		for _, roleNode := range rolesCollection.GetChildren() {
			if nameAttr, exists := roleNode.GetAttribute("name"); exists {
				if nameStr, ok := nameAttr.(Str); ok && nameStr == roleName {
					// Update attributes
					for key, value := range updates {
						if key != "name" { // Don't allow name changes
							roleNode.SetAttribute(key, value)
						}
					}
					roleNode.SetAttribute("updatedAt", Str(time.Now().Format(time.RFC3339)))
					return roleNode, nil
				}
			}
		}

		return nil, fmt.Errorf("role not found: %s", roleName)
	})

	// deleteRole - remove a role from the roles collection
	rt.Register("deleteRole", func(args ...Value) (Value, error) {
		if len(args) != 2 {
			return nil, errors.New("deleteRole requires 2 arguments: rolesCollection and roleName")
		}

		// Unwrap arguments
		for i, arg := range args {
			if tvar, ok := arg.(ScopeEntry); ok {
				args[i] = tvar.Value
			}
		}

		// Get roles collection
		rolesCollection, ok := args[0].(TreeNode)
		if !ok {
			return nil, fmt.Errorf("first argument must be a TreeNode roles collection, got %T", args[0])
		}

		// Get role name
		roleName, ok := args[1].(Str)
		if !ok {
			return nil, fmt.Errorf("second argument must be a string roleName, got %T", args[1])
		}

		// Find and remove the role
		for _, roleNode := range rolesCollection.GetChildren() {
			if nameAttr, exists := roleNode.GetAttribute("name"); exists {
				if nameStr, ok := nameAttr.(Str); ok && nameStr == roleName {
					rolesCollection.RemoveChild(roleNode)
					return Bool(true), nil
				}
			}
		}

		return nil, fmt.Errorf("role not found: %s", roleName)
	})

	// getAvailableFunctions - get all functions in the master registry
	rt.Register("getAvailableFunctions", func(args ...Value) (Value, error) {
		result := NewArray()
		for funcName := range MasterFunctionRegistry {
			result.Append(Str(funcName))
		}
		return result, nil
	})

	// Helper to register all functions to the master registry
	rt.Register("registerFunctionToMaster", func(args ...Value) (Value, error) {
		if len(args) != 2 {
			return nil, errors.New("registerFunctionToMaster requires 2 arguments: functionName and implementation")
		}

		// This would be used internally by the system
		// Not typically called from Chariot scripts
		return Bool(true), nil
	})
}

// Helper function to populate the master registry
func PopulateMasterRegistry() {
	// This should be called after all function modules are loaded
	// Each module should call RegisterToMaster for its functions
}
