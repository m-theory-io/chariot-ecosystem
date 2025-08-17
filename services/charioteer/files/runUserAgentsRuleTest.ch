setq(users, getChildByName(usersAgent, 'users'))
setq(rules, getChildByName(usersAgent,'rules'))
call(getAttribute(rules, 'userCount'), users)