package chariot

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"time"
)

func RegisterAuthFuncs(rt *Runtime) {
	// User management functions

	// findUser - find a user by userID in a users collection
	rt.Register("findUser", func(args ...Value) (Value, error) {
		if len(args) != 2 {
			return nil, errors.New("findUser requires 2 arguments: usersCollection and userID")
		}

		// Unwrap arguments
		for i, arg := range args {
			if tvar, ok := arg.(ScopeEntry); ok {
				args[i] = tvar.Value
			}
		}

		userID := args[1].(Str)

		switch coll := args[0].(type) {
		case *ArrayValue:
			if coll.Length() == 0 {
				return nil, fmt.Errorf("users collection is empty")
			}
			for _, item := range coll.Elements {
				switch node := item.(type) {
				case *JSONNode:
					if userIDAttr, exists := node.GetAttribute("userID"); exists {
						if userIDVal, ok := userIDAttr.(Str); ok && userID == userIDVal {
							return node, nil
						}
					}
				case *MapValue:
					if userIDAttr, exists := node.GetAttribute("userID"); exists {
						if userIDVal, ok := userIDAttr.(Str); ok && userID == userIDVal {
							return node, nil
						}
					}
				case *TreeNodeImpl:
					if userIDAttr, exists := node.GetAttribute("userID"); exists {
						if userIDVal, ok := userIDAttr.(Str); ok && userID == userIDVal {
							return node, nil
						}
					}
				default:
					return nil, fmt.Errorf("unsupported collection item type: %T", item)
				}
			}
		case *JSONNode:
			// look for underlying _users attribute
			if usersAttr, exists := coll.GetAttribute("_users"); exists {
				if usersArray, ok := usersAttr.(*ArrayValue); ok {
					return rt.funcs["findUser"](usersArray, args[1])
				}
			}
			return nil, fmt.Errorf("users collection is not an array or does not have _users attribute")
		case *TreeNodeImpl:
			// look for underlying _users attribute
			if usersAttr, exists := coll.GetAttribute("_users"); exists {
				if usersArray, ok := usersAttr.(*ArrayValue); ok {
					return rt.funcs["findUser"](usersArray, args[1])
				}
			}
			return nil, fmt.Errorf("users collection is not an array or does not have _users attriute")
		default:
			return nil, fmt.Errorf("unsupported users collection type: %T", args[0])
		}

		return nil, fmt.Errorf("user not found: %s", userID)
	})

	// createUser - create a new user in the users collection
	rt.Register("createUser", func(args ...Value) (Value, error) {
		if len(args) != 4 {
			return nil, errors.New("createUser requires 4 arguments: usersCollection, userID, displayName, roles")
		}

		// Unwrap arguments
		for i, arg := range args {
			if tvar, ok := arg.(ScopeEntry); ok {
				args[i] = tvar.Value
			}
		}

		// Get users collection
		usersCollection, ok := args[0].(TreeNode)
		if !ok {
			return nil, fmt.Errorf("first argument must be a TreeNode users collection, got %T", args[0])
		}

		// Get userID
		userID, ok := args[1].(Str)
		if !ok {
			return nil, fmt.Errorf("second argument must be a string userID, got %T", args[1])
		}

		// Get displayName
		displayName, ok := args[2].(Str)
		if !ok {
			return nil, fmt.Errorf("third argument must be a string displayName, got %T", args[2])
		}

		// Get roles (should be an array)
		roles, ok := args[3].(*ArrayValue)
		if !ok {
			return nil, fmt.Errorf("fourth argument must be an array of roles, got %T", args[3])
		}

		// Check if user already exists
		for _, child := range usersCollection.GetChildren() {
			userNode := child
			if userIDAttr, exists := userNode.GetAttribute("userID"); exists {
				if userIDStr, ok := userIDAttr.(Str); ok && userIDStr == userID {
					return nil, fmt.Errorf("user already exists: %s", userID)
				}
			}
		}

		// Create new user node
		newUser := NewJSONNode("{}")
		newUser.SetAttribute("userID", userID)
		newUser.SetAttribute("displayName", displayName)
		newUser.SetAttribute("roles", roles)
		newUser.SetAttribute("status", Str("active"))
		newUser.SetAttribute("createdAt", Str(time.Now().Format(time.RFC3339)))
		newUser.SetAttribute("updatedAt", Str(time.Now().Format(time.RFC3339)))

		// Add to users collection
		usersCollection.AddChild(newUser)

		return newUser, nil
	})

	// updateUser - update an existing user's attributes
	rt.Register("updateUser", func(args ...Value) (Value, error) {
		if len(args) != 3 {
			return nil, errors.New("updateUser requires 3 arguments: usersCollection, userID, updates")
		}

		// Unwrap arguments
		for i, arg := range args {
			if tvar, ok := arg.(ScopeEntry); ok {
				args[i] = tvar.Value
			}
		}

		// Get users collection
		usersCollection, ok := args[0].(TreeNode)
		if !ok {
			return nil, fmt.Errorf("first argument must be a TreeNode users collection, got %T", args[0])
		}

		// Get userID
		userID, ok := args[1].(Str)
		if !ok {
			return nil, fmt.Errorf("second argument must be a string userID, got %T", args[1])
		}

		// Get updates (should be a map or TreeNode with attributes)
		var updates map[string]Value
		switch updatesArg := args[2].(type) {
		case TreeNode:
			updates = updatesArg.GetAttributes()
		case *MapValue:
			updates = updatesArg.GetAttributes()
		default:
			return nil, fmt.Errorf("third argument must be a TreeNode or MapValue with updates, got %T", args[2])
		}

		// Find the user
		for _, userNode := range usersCollection.GetChildren() {
			if userIDAttr, exists := userNode.GetAttribute("userID"); exists {
				if userIDStr, ok := userIDAttr.(Str); ok && userIDStr == userID {
					// Update attributes
					for key, value := range updates {
						if key != "userID" { // Don't allow userID changes
							userNode.SetAttribute(key, value)
						}
					}
					userNode.SetAttribute("updatedAt", Str(time.Now().Format(time.RFC3339)))
					return userNode, nil
				}
			}
		}

		return nil, fmt.Errorf("user not found: %s", userID)
	})

	// deleteUser - remove a user from the users collection
	rt.Register("deleteUser", func(args ...Value) (Value, error) {
		if len(args) != 2 {
			return nil, errors.New("deleteUser requires 2 arguments: usersCollection and userID")
		}

		// Unwrap arguments
		for i, arg := range args {
			if tvar, ok := arg.(ScopeEntry); ok {
				args[i] = tvar.Value
			}
		}

		// Get users collection
		usersCollection, ok := args[0].(TreeNode)
		if !ok {
			return nil, fmt.Errorf("first argument must be a TreeNode users collection, got %T", args[0])
		}

		// Get userID
		userID, ok := args[1].(Str)
		if !ok {
			return nil, fmt.Errorf("second argument must be a string userID, got %T", args[1])
		}

		// Find and remove the user
		children := usersCollection.GetChildren()
		for _, userNode := range children {
			if userIDAttr, exists := userNode.GetAttribute("userID"); exists {
				if userIDStr, ok := userIDAttr.(Str); ok && userIDStr == userID {
					// Remove from children
					usersCollection.RemoveChild(userNode)
					return Bool(true), nil
				}
			}
		}

		return nil, fmt.Errorf("user not found: %s", userID)
	})

	// authenticateUser - basic authentication (for testing - extend with OAuth)
	rt.Register("authenticateUser", func(args ...Value) (Value, error) {
		if len(args) != 3 {
			return nil, errors.New("authenticateUser requires 3 arguments: usersCollection, userID, password")
		}

		// Unwrap arguments
		for i, arg := range args {
			if tvar, ok := arg.(ScopeEntry); ok {
				args[i] = tvar.Value
			}
		}

		// Get users collection
		usersCollection, ok := args[0].(TreeNode)
		if !ok {
			return nil, fmt.Errorf("first argument must be a TreeNode users collection, got %T", args[0])
		}

		// Get userID
		userID, ok := args[1].(Str)
		if !ok {
			return nil, fmt.Errorf("second argument must be a string userID, got %T", args[1])
		}

		// Get password
		password, ok := args[2].(Str)
		if !ok {
			return nil, fmt.Errorf("third argument must be a string password, got %T", args[2])
		}

		// Find the user in the _users attribute (consistent with findUser)
		if usersAttr, exists := usersCollection.GetAttribute("_users"); exists {
			if usersArray, ok := usersAttr.(*ArrayValue); ok {
				for _, item := range usersArray.Elements {
					switch userNode := item.(type) {
					case map[string]Value:
						if userIDAttr, exists := userNode["userID"]; exists {
							if userIDStr, ok := userIDAttr.(Str); ok && userIDStr == userID {
								// Check status
								if statusAttr, exists := userNode["status"]; exists {
									if statusStr, ok := statusAttr.(Str); ok && statusStr != "active" {
										return nil, fmt.Errorf("user account is not active: %s", statusStr)
									}
								}
								// Check password (simple hash comparison for now)
								if passwordAttr, exists := userNode["password"]; exists {
									passwordHash := hashPassword(string(password))
									if passwordHash == string(passwordAttr.(Str)) {
										// Update last login
										userNode["lastLoginAt"] = Str(time.Now().Format(time.RFC3339))
										return userNode, nil
									}
								}
								return nil, errors.New("invalid password")
							}
						}
					case *JSONNode:
						if userIDAttr, exists := userNode.GetAttribute("userID"); exists {
							if userIDStr, ok := userIDAttr.(Str); ok && userIDStr == userID {
								// Check status
								if statusAttr, exists := userNode.GetAttribute("status"); exists {
									if statusStr, ok := statusAttr.(Str); ok && statusStr != "active" {
										return nil, fmt.Errorf("user account is not active: %s", statusStr)
									}
								}

								// Check password (simple hash comparison for now)
								if passwordAttr, exists := userNode.GetAttribute("passwordHash"); exists {
									if passwordHash, ok := passwordAttr.(Str); ok {
										if hashPassword(string(password)) == string(passwordHash) {
											// Update last login
											userNode.SetAttribute("lastLoginAt", Str(time.Now().Format(time.RFC3339)))
											return userNode, nil
										}
									}
								} else if passwordAttr, exists := userNode.GetAttribute("password"); exists {
									// Support direct password attribute (as in your bootstrap file)
									if passwordHash, ok := passwordAttr.(Str); ok {
										if hashPassword(string(password)) == string(passwordHash) {
											// Update last login
											userNode.SetAttribute("lastLoginAt", Str(time.Now().Format(time.RFC3339)))
											return userNode, nil
										}
									}
								}

								return nil, errors.New("invalid password")
							}
						}
					case *TreeNodeImpl:
						if userIDAttr, exists := userNode.GetAttribute("userID"); exists {
							if userIDStr, ok := userIDAttr.(Str); ok && userIDStr == userID {
								// Check status
								if statusAttr, exists := userNode.GetAttribute("status"); exists {
									if statusStr, ok := statusAttr.(Str); ok && statusStr != "active" {
										return nil, fmt.Errorf("user account is not active: %s", statusStr)
									}
								}

								// Check password (simple hash comparison for now)
								if passwordAttr, exists := userNode.GetAttribute("passwordHash"); exists {
									if passwordHash, ok := passwordAttr.(Str); ok {
										if hashPassword(string(password)) == string(passwordHash) {
											// Update last login
											userNode.SetAttribute("lastLoginAt", Str(time.Now().Format(time.RFC3339)))
											return userNode, nil
										}
									}
								} else if passwordAttr, exists := userNode.GetAttribute("password"); exists {
									// Support direct password attribute (as in your bootstrap file)
									if passwordHash, ok := passwordAttr.(Str); ok {
										if hashPassword(string(password)) == string(passwordHash) {
											// Update last login
											userNode.SetAttribute("lastLoginAt", Str(time.Now().Format(time.RFC3339)))
											return userNode, nil
										}
									}
								}

								return nil, errors.New("invalid password")
							}
						}
					case *MapValue:
						if userIDAttr, exists := userNode.GetAttribute("userID"); exists {
							if userIDStr, ok := userIDAttr.(Str); ok && userIDStr == userID {
								// Check status
								if statusAttr, exists := userNode.GetAttribute("status"); exists {
									if statusStr, ok := statusAttr.(Str); ok && statusStr != "active" {
										return nil, fmt.Errorf("user account is not active: %s", statusStr)
									}
								}

								// Check password (simple hash comparison for now)
								if passwordAttr, exists := userNode.GetAttribute("passwordHash"); exists {
									if passwordHash, ok := passwordAttr.(Str); ok {
										if hashPassword(string(password)) == string(passwordHash) {
											// Update last login
											userNode.SetAttribute("lastLoginAt", Str(time.Now().Format(time.RFC3339)))
											return userNode, nil
										}
									}
								} else if passwordAttr, exists := userNode.GetAttribute("password"); exists {
									// Support direct password attribute (as in your bootstrap file)
									if passwordHash, ok := passwordAttr.(Str); ok {
										if hashPassword(string(password)) == string(passwordHash) {
											// Update last login
											userNode.SetAttribute("lastLoginAt", Str(time.Now().Format(time.RFC3339)))
											return userNode, nil
										}
									}
								}

								return nil, errors.New("invalid password")
							}
						}
					default:
						return nil, fmt.Errorf("unsupported user node type: %T", item)
					}
				}
			}
		}

		return nil, fmt.Errorf("user not found: %s", userID)
	})

	// setUserPassword - set a user's password (hashed)
	rt.Register("setUserPassword", func(args ...Value) (Value, error) {
		if len(args) != 3 {
			return nil, errors.New("setUserPassword requires 3 arguments: usersCollection, userID, password")
		}

		// Unwrap arguments
		for i, arg := range args {
			if tvar, ok := arg.(ScopeEntry); ok {
				args[i] = tvar.Value
			}
		}

		// Get users collection
		usersCollection, ok := args[0].(TreeNode)
		if !ok {
			return nil, fmt.Errorf("first argument must be a TreeNode users collection, got %T", args[0])
		}

		// Get userID
		userID, ok := args[1].(Str)
		if !ok {
			return nil, fmt.Errorf("second argument must be a string userID, got %T", args[1])
		}

		// Get password
		password, ok := args[2].(Str)
		if !ok {
			return nil, fmt.Errorf("third argument must be a string password, got %T", args[2])
		}

		// Find the user
		for _, userNode := range usersCollection.GetChildren() {
			if userIDAttr, exists := userNode.GetAttribute("userID"); exists {
				if userIDStr, ok := userIDAttr.(Str); ok && userIDStr == userID {
					// Set password hash
					passwordHash := hashPassword(string(password))
					userNode.SetAttribute("passwordHash", Str(passwordHash))
					userNode.SetAttribute("updatedAt", Str(time.Now().Format(time.RFC3339)))
					return Bool(true), nil
				}
			}
		}

		return nil, fmt.Errorf("user not found: %s", userID)
	})

	// generateToken - generate a secure token for sessions
	rt.Register("generateToken", func(args ...Value) (Value, error) {
		// Generate 32 random bytes
		bytes := make([]byte, 32)
		if _, err := rand.Read(bytes); err != nil {
			return nil, fmt.Errorf("failed to generate token: %v", err)
		}

		// Convert to hex string
		token := hex.EncodeToString(bytes)
		return Str(token), nil
	})

	// validateDisplayName - check if display name is unique
	rt.Register("validateDisplayName", func(args ...Value) (Value, error) {
		if len(args) != 2 {
			return nil, errors.New("validateDisplayName requires 2 arguments: usersCollection and displayName")
		}

		// Unwrap arguments
		for i, arg := range args {
			if tvar, ok := arg.(ScopeEntry); ok {
				args[i] = tvar.Value
			}
		}

		// Get users collection
		usersCollection, ok := args[0].(TreeNode)
		if !ok {
			return nil, fmt.Errorf("first argument must be a TreeNode users collection, got %T", args[0])
		}

		// Get displayName
		displayName, ok := args[1].(Str)
		if !ok {
			return nil, fmt.Errorf("second argument must be a string displayName, got %T", args[1])
		}

		// Check if display name is already taken
		for _, child := range usersCollection.GetChildren() {
			userNode := child
			if displayNameAttr, exists := userNode.GetAttribute("displayName"); exists {
				if displayNameStr, ok := displayNameAttr.(Str); ok && displayNameStr == displayName {
					return Bool(false), nil // Display name is taken
				}
			}
		}

		return Bool(true), nil // Display name is available
	})
}

// Helper function to hash passwords
func hashPassword(password string) string {
	hash := sha256.Sum256([]byte(password))
	return hex.EncodeToString(hash[:])
}
