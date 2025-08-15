// New Chariot Script
setq(users, getChildByName(usersAgent, 'users'))
setq(rules, getChildByName(usersAgent, 'rules'))
setq(userCount, getAttribute(rules, 'userCount'))
call(userCount, users)
