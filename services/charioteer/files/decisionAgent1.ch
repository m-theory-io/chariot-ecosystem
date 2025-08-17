// Create agent, offer and rules as before
declareGlobal(agent, 'T', create('agent'))
addChild(agent, jsonNode("offer"))
logPrint("DEBUG: agent created...", "info", agent)

// Create offer node and set attributes
setq(offer, getChildAt(agent, 0))
setAttribute(offer, 'name', 'Green Lease Plan')
setAttribute(offer, 'description', 'A lease plan for well-qualified applicants')
setAttribute(offer, 'monthly', offerVar(2500.00, 'currency'))
setAttribute(offer, 'term', offerVar(12, 'int'))
setAttribute(offer, 'deposit', func(profile, offer) {
   if(getAttribute(profile, 'is_employed')) {
     offerVar(1000.00, 'currency')
   } else {
     offerVar(2500.0, 'currency')
   }
})
setAttribute(offer, 'start_date', func(profile, offer) {
  formatDate(dateAdd(now(), 'day', 15), 'YYYY-MM-DD')
})
setAttribute(offer, 'text', 'Pay {deposit} deposit and {monthly} 1st-month rent today, to start your lease on {start_date}. Remaining rent of {monthly} for {term} months, to be paid on the 1st of every month.')
logPrint("DEBUG", "Offer created: ", offer)
// Create rules node and add filters
addChild(agent, jsonNode("rules"))
setq(rules, getChildAt(agent, 1))
setAttribute(rules, 'ageFilter', func(profile) { biggerEq(getAttribute(profile, 'age'), 18) })
setAttribute(rules, 'debtFilter', func(profile) { smallerEq(getAttribute(profile, 'debt'), 10000) })
setAttribute(rules, 'employmentFilter', func(profile) { equal(getAttribute(profile, 'is_employed'), true) })
logPrint("DEBUG", "Rules created: ", rules)
// Create handlers node and add onDecisionRequest handler
addChild(agent, jsonNode("handlers"))
setq(handlers, getChildAt(agent, 2))
setAttribute(handlers, 'onDecisionRequest', func(req) {
setq(profile, getProp(req, "profile"))
setq(ageFilter, getAttribute(rules, 'ageFilter'))
setq(debtFilter, getAttribute(rules, 'debtFilter'))
setq(employmentFilter, getAttribute(rules, 'employmentFilter'))
setq(approved, and(call(ageFilter, profile), call(debtFilter, profile), call(employmentFilter, profile)))
approved
})
logPrint("DEBUG", "Handlers created: ", handlers)
// Simulate a request with a profile parameter (not serialized in agent)
declare(request, 'J', jsonNode("{\"profile\": {\"age\": 30,\"debt\": 7000.25,\"is_employed\": true}}"))
setq(profile, getProp(request, "profile"))
setq(handler, getAttribute(handlers, 'onDecisionRequest'))
setq(result, call(handler, request))
// Process decision result
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
treeSave(agent, "decisionAgent1.json")
decisionResult