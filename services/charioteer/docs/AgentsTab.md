# **Agents Tab**
The Agents tab is where Charioteer provides a specialized UI for creating and editing agents. An agent is defined as:

- declared globally as a TreeNode ('T')
- name ends with the suffix `Agent`
- includes child JSONNodes to hold functional and data elements
- must have at least a `rules` child node that contains a map of function definitions

*Base Agent Structure*

All agents are TreeNode objects. So, the TreeNode can be considered the base type.

| Attribute | Type | Description |
| --------------- | -------- | ----------------------------------------------------------- |
| Name | string | name of the agent |
| Children | []TreeNode | child nodes of the agent |
| Attributes | map[string]Value | attribute map of the agent

*Users Agent*

The `usersAgent` builds on the base type TreeNode by adding a specific set of children to the agent.

| Child | Type | Description |
| -------------------------- | ------------------ | ------------------------------------------ |
| config | *JSONNode | used as a map, each key contains a different configuration item.
| roles | *JSONNode | used as an array, each element is the string of a role name.
| rules | *JSONNode | used as a map, each key contains a callable function definition.
| users | *JSONNode | uses as an array, each element contains a userProfile JSONNode.

```
function createUsersAgent() {
    // Create usersAgent tree
    declareGlobal(usersAgent, 'T', create('usersAgent'))
    // Create child JSONNodes
    declare(users, 'J', parseJSON('[]')) // used as an array
    declare(roles, 'J', parseJSON('[]')) // used as an array
    declare(rules, 'J', parseJSON('{}')) // used as a map
    declare(config, 'J', parseJSON('{}')) // used as a map
    // Initialize roles
    
    // Initialize config node to default values
    setAttribute(config, 'encryptionEnabled', true)
    setAttribute(config, 'sessionTimeout', 3600)
    setAttribute(config, 'maxLoginAttempts', 5)
    // Initialize rules
    setAttribute(rules, 'userCount', func(users) {
        if(hasAttribute(users, '_users')) {
            length(getAttribute(users, '_users'))
        } else {
            0
        }
    })
    // Add children to usersAgent tree
    addChild(usersAgent, users)
    addChild(usersAgent, rules)
    addChild(usersAgent, config)
    // Save tree in GOB format
    treeSave(usersAgent, 'usersAgent.gob', 'gob')
}
```


