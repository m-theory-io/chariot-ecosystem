// Add users to users collection
setq(users, getChildByName(usersAgent, 'users'))
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
