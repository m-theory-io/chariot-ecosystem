// Load the saved tree from GOB format
declareGlobal(usersAgent, 'T', treeLoad("usersAgent.gob"))

// Display basic tree information
logPrint("=== Tree Load Verification ===")
logPrint("Tree name:", 'info', getName(usersAgent))
logPrint("Tree type:", 'info', typeOf(usersAgent))

// Check children
logPrint("Number of children:", 'info', childCount(usersAgent))

// Inspect each child
setq(ndx, 0)
while(smaller(ndx, childCount(usersAgent))) {
    setq(child, getChildAt(usersAgent, ndx))
    logPrint("Child", "info", ndx, ":")
    logPrint("  Name:", "info", getName(child))
    logPrint("  Type:", "info", typeOf(child))
    // Attributes
    apply(func(key, value) { logPrint('Element', 'info', key, value) }, getAttributes(usersAgent))
    // increment ndx
    setq(ndx add(ndx, 1))
}
// Test specific nodes
setq(config, getChildAt(usersAgent, 2)) // config should be index 2
if(equal(getName(config), "config")) {
    logPrint('Config:', 'info', config)
}

// Return the tree for verification
usersAgent
