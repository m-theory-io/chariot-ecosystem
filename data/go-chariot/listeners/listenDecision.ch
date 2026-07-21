// Load the agent
setq(agentNode, treeLoad("debtorPlans.json"))
declareGlobal(agent, 'T', agentNode)

// Start HTTP listener with multiple handlers
listen(8050, "routeRequest")