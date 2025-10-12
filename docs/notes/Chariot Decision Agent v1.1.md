# Chariot Decision Agent v1.1

## Overview

Chariot Decision Agent v1.1 extends the original agent design by introducing an `offer` node and dynamic offer text templating. This enables the agent to generate personalized offer messages by merging offer variables into a text template, based on the evaluation of decision rules.

---

## Agent Tree Structure

```
Agent
├── offer
│   ├── name: "Green Lease Plan"
│   ├── description: "A lease plan for well-qualified applicants"
│   ├── monthly: 2500.00
│   ├── term: 12
│   ├── deposit: func(profile, offer) { ... }
│   ├── start_date: func(profile, offer) { ... }
│   └── text: "Pay {deposit} deposit and {monthly} 1st-month rent today, to start your lease on {start_date}. Remaining rent of {monthly} for {term} months, to be paid on the 1st of every month."
├── rules
│   ├── ageFilter: func(profile) { bigger(getAttribute(profile, 'age'), 18) }
│   ├── debtFilter: func(profile) { smaller(getAttribute(profile, 'debt'), 10000) }
│   └── employmentFilter: func(profile) { equal(getAttribute(profile, 'is_employed'), true) }
└── handlers
    └── onDecisionRequest: func(req) { ... }
```

---

## Offer Node

The `offer` node contains all the attributes needed to describe a lease offer, including computed fields:

- **deposit**: A function that computes the deposit based on the applicant's employment status.
- **start_date**: A function that computes the lease start date.
- **text**: A template string with placeholders for dynamic values.

**Example:**
```chariot
setAttribute(offer, 'deposit', func(profile, offer) {
  if (getAttribute(profile, 'is_employed')) {
    1000.00
  }
  2500.00
})

setAttribute(offer, 'start_date', func(profile, offer) {
  formatTime(dateAdd(now(), 'day', 15), 'YYYY-MM-DD');
})

setAttribute(offer, 'text', 'Pay {deposit} deposit and {monthly} 1st-month rent today, to start your lease on {start_date}. Remaining rent of {monthly} for {term} months, to be paid on the 1st of every month.')
```

---

## Merging Offer Variables with the Text Template

When a decision is approved, the agent merges the offer variables into the `text` template to generate a personalized offer message.

**Example Chariot code:**
```chariot
if(result) {
  setq(offerText, getAttribute(offer, 'text'))
  setq(mergedText, merge(offerText, offer, [profile, offer]))
  setq(jsonString, concat("{\"decision\": \"approved\", \"offer\": \"", mergedText, "\"}"))
  setq(decisionResult, jsonNode(jsonString))
} else {
  setq(jsonString, "{\"decision\": \"denied\", \"reason\": \"Profile does not meet requirements.\"}")
  setq(decisionResult, jsonNode(jsonString))
}
```
- The `merge` function replaces placeholders like `{deposit}`, `{monthly}`, `{start_date}`, and `{term}` in the template with their computed values from the `offer` and `profile` nodes.

---

## Handler Example

The `onDecisionRequest` handler evaluates the rules and, if approved, generates the personalized offer:

```chariot
setAttribute(handlers, 'onDecisionRequest', func(req) {
  setq(profile, getProp(req, "profile"))
  setq(ageFilter, getAttribute(rules, 'ageFilter'))
  setq(debtFilter, getAttribute(rules, 'debtFilter'))
  setq(employmentFilter, getAttribute(rules, 'employmentFilter'))
  setq(approved, and(call(ageFilter, profile), call(debtFilter, profile), call(employmentFilter, profile)))
  approved
})
```

---

## Example Decision Request Flow

1. **Caller** sends a request with a `profile` parameter.
2. **Agent** evaluates rules using the profile.
3. If approved:
    - Computes offer variables (e.g., deposit, start_date).
    - Merges variables into the offer text template.
    - Returns a JSON response with `"decision": "approved"` and the personalized `"offer"` string.
4. If denied:
    - Returns a JSON response with `"decision": "denied"` and a reason.

---

## Example Output

**Approved:**
```json
{
  "decision": "approved",
  "offer": "Pay 1000 deposit and 2500 1st-month rent today, to start your lease on 2025-07-15. Remaining rent of 2500 for 12 months, to be paid on the 1st of every month."
}
```

**Denied:**
```json
{
  "decision": "denied",
  "reason": "Profile does not meet requirements."
}
```

---

## Summary

- The v1.1 agent introduces an `offer` node and dynamic text templating.
- The agent can now generate personalized offer messages by merging computed variables into a template.
- This pattern supports scalable, maintainable decision logic for real-world applications.