package tests

import (
	"testing"

	"github.com/bhouse1273/go-chariot/chariot"
)

func TestAgentMinimal(t *testing.T) {

	initCouchbaseConfig() // ← Initialize config first
	var rt *chariot.Runtime
	if tvar, exists := chariot.GetRuntimeByID("test_db"); exists {
		rt = tvar
	} else {
		rt = createNamedRuntime("test_db")
		defer chariot.UnregisterRuntime("test_db")
	}

	tests := []TestCase{
		{
			Name: "Agent with Handler Node - Decision Request",
			Script: []string{
				// Create agent, offer and rules as before
				`declareGlobal(agent, 'T', create('agent'))`,
				`addChild(agent, jsonNode("offer"))`,
				// Create offer node and set attributes
				`setq(offer, getChildAt(agent, 0))`,
				`setAttribute(offer, 'name', 'Green Lease Plan')`,
				`setAttribute(offer, 'description', 'A lease plan for well-qualified applicants')`,
				`setAttribute(offer, 'monthly', format('$%2.F', 2500.00))`,
				`setAttribute(offer, 'term', 12)`,
				`setAttribute(offer, 'deposit', func(profile, offer) {`,
				`  if (getAttribute(profile, 'is_employed')) {`,
				`    format('$%2.F', 1000.00)`,
				`  }`,
				`  format('$%2.F', 2500.00)`,
				`})`,
				`setAttribute(offer, 'start_date', func(profile, offer) {`,
				`  formatDate(dateAdd(now(), 'day', 15), 'YYYY-MM-DD');`,
				`})`,
				`setAttribute(offer, 'text', 'Pay {deposit} deposit and {monthly} 1st-month rent today, to start your lease on {start_date}. Remaining rent of {monthly} for {term} months, to be paid on the 1st of every month.')`,
				`logPrint("Offer created:", "debug", offer)`,
				// Create rules node and add filters
				`addChild(agent, jsonNode("rules"))`,
				`setq(rules, getChildAt(agent, 1))`,
				`setAttribute(rules, 'ageFilter', func(profile) { bigger(getAttribute(profile, 'age'), 18) })`,
				`setAttribute(rules, 'debtFilter', func(profile) { smaller(getAttribute(profile, 'debt'), 10000) })`,
				`setAttribute(rules, 'employmentFilter', func(profile) { equal(getAttribute(profile, 'is_employed'), true) })`,
				`logPrint("Rules created:", "debug", rules)`,
				// Create handlers node and add onDecisionRequest handler
				`addChild(agent, jsonNode("handlers"))`,
				`setq(handlers, getChildAt(agent, 2))`,
				`setAttribute(handlers, 'onDecisionRequest', func(req) {`,
				`setq(profile, getProp(req, "profile"))`,
				`setq(ageFilter, getAttribute(rules, 'ageFilter'))`,
				`setq(debtFilter, getAttribute(rules, 'debtFilter'))`,
				`setq(employmentFilter, getAttribute(rules, 'employmentFilter'))`,
				`setq(approved, and(call(ageFilter, profile), call(debtFilter, profile), call(employmentFilter, profile)))`,
				`approved`,
				`})`,
				`logPrint("Handlers created:", "debug", handlers)`,
				// Simulate a request with a profile parameter (not serialized in agent)
				`declare(request, 'J', jsonNode("{\"profile\": {\"age\": 30,\"debt\": 7000.25,\"is_employed\": true}}"))`,
				`setq(profile, getProp(request, "profile"))`,
				`setq(handler, getAttribute(handlers, 'onDecisionRequest'))`,
				`setq(result, call(handler, request))`,
				`setq(decisionResult, jsonNode("{}"))`,
				`if(result) {`,
				`  setq(offerText,getAttribute(offer, 'text'))`,
				`  setq(mergedText, merge(offerText, offer, [profile, offer]))`,
				`  setq(jsonString, concat("{\"decision\": \"approved\", \"offer\": \"", mergedText, "\"}"))`,
				`  setq(decisionResult, jsonNode(jsonString))`,
				`} else {`,
				`  setq(jsonString, "{\"decision\": \"denied\", \"reason\": \"Profile does not meet requirements.\"}")`,
				`  setq(decisionResult, jsonNode(jsonString))`,
				`}`,
				`logPrint("Decision result", "debug", decisionResult)`,
				`getProp(decisionResult, "decision")`,
			},
			ExpectedValue: chariot.Str("approved"),
		},
	}
	RunStatefulTestCases(t, tests, rt)
}
