// Decision test
setq(offer, getChildAt(agent, 0))
setq(rules, getChildAt(agent, 1))
setq(handlers, getChildAt(agent, 2))
// Simulate a request with a profile parameter (not serialized in agent)
declare(request, 'J', jsonNode("{\"profile\": {\"age\": 30,\"debt\": 7000.25,\"is_employed\": true}}"))
setq(profile, getProp(request, "profile"))
setq(handler, getAttribute(handlers, 'onDecisionRequest'))
setq(result, call(handler, request))
