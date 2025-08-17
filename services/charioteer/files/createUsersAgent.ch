// createUsersAgent
declareGlobal(usersAgent, 'T', create('usersAgent'))
// declare child nodes
declare(users, 'J', parseJSON('[{"userID": "bhouse", "email": "bhouse@m-theory,io", "roles": ["admin","contributor"], "password": "9d61f70341d8db28d31419bf1f203ac8e71746b6ef774815f260427b89476eb2"}]', 'users'))
declare(roles,'J', parseJSON('["admin", "contributor", "viewer"]', 'roles'))
declare(rules, 'J', parseJSON('{}', 'rules'))
declare(config,'J', parseJSON('{}', 'config'))
// populate default config node
setAttribute(config, 'encryptionEnabled', true)
setAttribute(config, 'sessionTimeout', 3600)
setAttribute(config, 'maxLoginAttempts', 5)
// populate rules
setAttribute(rules, 'userCount', func(users) {
    if(hasAttribute(users, '_users')) {
        length(getAttribute(users, '_users'))
    } else {
        0
    }
})
// Add children
addChild(usersAgent, users)
addChild(usersAgent, roles)
addChild(usersAgent, rules)
addChild(usersAgent, config)
// Save usersAgent
treeSave(usersAgent, 'usersAgent.json')
