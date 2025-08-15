// apply() test
// Load the saved tree from GOB format
declareGlobal(usersAgent, 'T', treeLoad("usersAgent.gob"))
// declare result array
declare(result, 'A')
// apply the func() to each attribute of usersAgent child 'config'
apply(func(key, value) {
        addTo(result, concat(key, '=', string(value))) 
    }, getChildByName(usersAgent, 'config'))
// join the result elements into a string and return
join(result, ', ')
