// usersAgent
declareGlobal(usersAgent, 'T', create('usersAgent'))
// Users collection
declare(users, 'J', parseJSON('[]', 'users'))
addChild(usersAgent, users)
// Add users to users collection
setq(userList, getAttribute(users, '_users'))
setq(userJSON, `{ 
    "userID": "bhouse",
    "email": "bhouse@m-theory,io",
    "roles": ["admin","contributor"],
    "password": "9d61f70341d8db28d31419bf1f203ac8e71746b6ef774815f260427b89476eb2"
    }`)
addTo(userList, parseJSON(userJSON, "userProfile"))
setAttribute(users, '_users', userList)
setAt(usersAgent, 0, users)
// Roles collection
declare(roles, 'J', parseJSON('[]', 'roles'))
addChild(usersAgent, roles)
setq(roleList, getAttribute(roles, '_roles'))
addTo(roleList, 'admin', 'contributor', 'viewer')
// Configuration
declare(config, 'J', parseJSON('{}', 'config'))
setAttribute(config, 'encryptionEnabled', true)
setAttribute(config, 'sessionTimeout', 3600)
setAttribute(config, 'maxLoginAttempts', 5)
addChild(usersAgent, config)
// Rules
setAttribute(declare(rules, 'J', parseJSON('{}', 'rules')), 'userCount', func(users) {
    length(getAttribute(users, '_users'))
})
addChild(usersAgent, rules)
treeSave(usersAgent, 'usersAgent.gob', 'gob', false)
usersAgent