// usersAgent
declareGlobal(usersAgent, 'T', create('usersAgent'))
// Users collection
declare(users, 'J', parseJSON('[]', 'users'))
addChild(usersAgent, users)
// Add users to users collection
setq(userList, getAttribute(users, '_users'))

