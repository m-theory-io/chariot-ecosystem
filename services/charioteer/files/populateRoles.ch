// New Chariot Script
setq(roles, getChildByName(usersAgent, 'roles'))
setq(roleList, array('admin', 'contributor', 'viewer'))
setAttribute(roles, '_roles', roleList)
