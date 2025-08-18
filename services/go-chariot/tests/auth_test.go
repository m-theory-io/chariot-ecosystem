package tests

import (
	"testing"

	"github.com/bhouse1273/chariot-ecosystem/services/go-chariot/chariot"
)

// auth_test.go
func TestUserAccountManagement(t *testing.T) {
	var rt *chariot.Runtime
	if tvar, exists := chariot.GetRuntimeByID("test_auth"); exists {
		rt = tvar
	} else {
		rt = createNamedRuntime("test_auth")
		defer chariot.UnregisterRuntime("test_auth")
	}
	tests := []TestCase{
		{
			Name: "Create User Agent",
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
			Name: "Add User Account",
			Script: []string{
				`setq(users, getChildByName(usersAgent, 'users'))`,
				`setq(userList, getAttribute(users, '_users'))`,
				`setq(newUser, parseJSON('` + `{
					"userID": "bhouse",
					"displayName": "Bill House",
					"email": "bhouse@m-theory.io",
					"roles": ["admin", "contributor"],
					"status": "active"
				}'` + `, 'userProfile'))`,
				`setAttribute(newUser, 'password', hash256('Borg12731273'))`,
				`addTo(userList, newUser)`,
				`setAttribute(users, '_users', userList)`,
				`getAttribute(newUser, 'userID')`,
			},
			ExpectedValue: chariot.Str("bhouse"),
		},
		{
			Name: "Find User by ID",
			Script: []string{
				`setq(users, getChildByName(usersAgent, 'users'))`,
				`setq(foundUser, findUser(users, 'bhouse'))`,
				`getAttribute(foundUser, 'displayName')`,
			},
			ExpectedValue: chariot.Str("Bill House"),
		},
		{
			Name: "Update User Roles",
			Script: []string{
				`setq(users, getChildByName(usersAgent, 'users'))`,
				`setq(user, findUser(users, 'bhouse'))`,
				`setAttribute(user, 'roles', ['admin', 'contributor', 'analyst'])`,
				`length(getAttribute(user, 'roles'))`,
			},
			ExpectedValue: chariot.Number(3),
		},
	}

	RunStatefulTestCases(t, tests, rt)
}
