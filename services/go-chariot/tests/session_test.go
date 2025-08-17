package tests

import (
	"testing"

	"github.com/bhouse1273/go-chariot/chariot"
)

// session_test.go
func TestUserAuthenticationSystem(t *testing.T) {

	var rt *chariot.Runtime
	if tvar, exists := chariot.GetRuntimeByID("test_auth"); exists {
		rt = tvar
	} else {
		rt = createNamedRuntime("test_auth")
		defer chariot.UnregisterRuntime("test_auth")
	}

	tests := []TestCase{
		{
			Name: "Create Users Agent",
			Script: []string{
				`declareGlobal(usersAgent, 'T', create('usersAgent'))`,
				`addChild(usersAgent, parseJSON('[]','users'))`,
				`addChild(usersAgent, parseJSON('[]','roles'))`,
				`addChild(usersAgent, parseJSON('{}','config'))`,
				`usersAgent`,
			},
			ExpectedType: "*chariot.TreeNodeImpl",
		},
		{
			Name: "Create Test User",
			Script: []string{
				`setq(users, getChildByName(usersAgent, 'users'))`,
				`setq(userList, getAttribute(users, '_users'))`,
				`setq(userJSON, '{"userID": "testuser@example.com", "email": "testuser@example.com", "roles": ["viewer"], "password": "9f735e0df9a1ddc702bf0a1a7b83033f9f7153a00c29de82cedadc9957289b05"}')`,
				`addTo(userList, parseJSON(userJSON, 'userProfile'))`,
				`setAttribute(users, '_users', userList)`,
				`length(userList)`,
			},
			ExpectedValue: chariot.Number(1),
		},
		{
			Name: "Find User by Username",
			Script: []string{
				`setq(users, getChildByName(usersAgent, 'users'))`,
				`setq(user, findUser(users, 'testuser@example.com'))`,
				`getAttribute(user, 'userID')`,
			},
			ExpectedValue: chariot.Str("testuser@example.com"),
		},
		{
			Name: "Authenticate Valid User",
			Script: []string{
				`setq(users, getChildByName(usersAgent, 'users'))`,
				`setq(authResult, authenticateUser(users, 'testuser@example.com', 'testpassword'))`,
				`getAttribute(authResult, 'userID')`,
			},
			ExpectedValue: chariot.Str("testuser@example.com"),
		},
		{
			Name: "Reject Invalid Password",
			Script: []string{
				`setq(users, getChildByName(usersAgent, 'users'))`,
				`authenticateUser(users, 'testuser@example.com', 'wrongpassword')`,
			},
			ExpectedError: true,
		},
		{
			Name: "User Not Found",
			Script: []string{
				`setq(users, getChildByName(usersAgent, 'users'))`,
				`authenticateUser(users, 'nonexistent@example.com', 'anypassword')`,
			},
			ExpectedError: true,
		},
	}

	RunStatefulTestCases(t, tests, rt)
}
