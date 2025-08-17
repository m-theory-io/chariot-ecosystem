// Chariot Agent
declareGlobal(agentDecision, 'T', create('agentDecision'))
addChild(agentDecision, jsonNode("profile"))
addChild(agentDecision, jsonNode("rules"))
toJSON(agentDecision)

