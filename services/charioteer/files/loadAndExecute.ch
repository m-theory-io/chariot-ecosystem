// Load and execute decisionAgent1.json
declareGlobal(agent, 'T', treeLoad("decisionAgent1.json"))
// Construct the profile
declare(request, 'J', jsonNode("{\"profile\": {\"age\": 30,\"debt\": 7000.25,\"is_employed\": true}}"))
setq(profile, getProp(request, "profile"))
// Grab the rules and offer
setq(offer, getChildByName(agent, 'offer'))
setq(rules, getChildByName(agent, "rules"))
log('rules:', 'debug', rules)
// Grab the handlers and handler
setq(handlers, getChildByName(agent, "handlers"))
setq(handler, getAttribute(handlers, 'onDecisionRequest'))
setq(result, call(handler, request))
log('result:', 'debug', result)
setq(decisionResult, jsonNode("{}"))
if(result) {
  setq(offerText,getAttribute(offer, 'text'))
  setq(mergedText, merge(offerText, offer, [profile, offer]))
  setq(jsonString, concat("{\"decision\": \"approved\", \"offer\": \"", mergedText, "\"}"))
  setq(decisionResult, jsonNode(jsonString))
} else {
  setq(jsonString, "{\"decision\": \"denied\", \"reason\": \"Profile does not meet requirements.\"}")
  setq(decisionResult, jsonNode(jsonString))
}
log("Decision result", "debug", decisionResult)
getProp(decisionResult, "decision")
