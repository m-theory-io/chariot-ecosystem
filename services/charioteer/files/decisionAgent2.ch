// decisionAgent2.ch
declareGlobal(agent, 'T', create('agent'))

addChild(agent, jsonNode("offer"))
setq(offer, getChildAt(agent, 0))
setAttribute(offer, 'name', 'Green Lease Plan')
setAttribute(offer, 'description', 'A lease plan for well-qualified applicants')
setAttribute(offer, 'monthly', 2500.00)
setAttribute(offer, 'term', 12)
setAttribute(offer, 'employed_deposit', 1000.00)
setAttribute(offer, 'unemployed_deposit', 2500.00)
setAttribute(offer, 'start_delay_days', 15)

addChild(agent, jsonNode("rules"))
setq(rules, getChildAt(agent, 1))
setAttribute(rules, 'min_age', 18)
setAttribute(rules, 'max_debt', 10000)
setAttribute(rules, 'requires_employment', true)

setAttribute(rules, 'ageFilter', func(profile) {
  biggerEq(getProp(profile, 'age'), 18)
})

setAttribute(rules, 'debtFilter', func(profile) {
  smallerEq(getProp(profile, 'debt'), 10000)
})

setAttribute(rules, 'employmentFilter', func(profile) {
  equal(getProp(profile, 'is_employed'), true)
})

addChild(agent, jsonNode("handlers"))
setq(handlers, getChildAt(agent, 2))

setAttribute(handlers, 'onDecisionRequest', func(req) {
  setq(profile, getProp(req, 'profile'))

  setq(ageOk, biggerEq(getProp(profile, 'age'), 18))
  setq(debtOk, smallerEq(getProp(profile, 'debt'), 10000))
  setq(employmentOk, equal(getProp(profile, 'is_employed'), true))
  setq(approved, and(ageOk, debtOk, employmentOk))

  if(approved) {
    setq(deposit, 1000.00)
    setq(monthly, 2500.00)
    setq(term, 12)
    setq(startDate, formatDate(dateAdd(now(), 'day', 15), 'YYYY-MM-DD'))
    setq(offerText, concat('Pay ', formatAs("currency", deposit), ' deposit and ', formatAs("currency", monthly), ' 1st-month rent today, to start your lease on ', startDate, '. Remaining rent of ', formatAs("currency", monthly), ' for ', term, ' months, to be paid on the 1st of every month.'))

    map(
      'decision', 'approved',
      'offerName', 'Green Lease Plan',
      'offer', offerText,
      'monthly', monthly,
      'term', term,
      'deposit', deposit,
      'startDate', startDate,
      'reasons', array('age_ok', 'debt_ok', 'employment_ok')
    )
  } else {
    setq(reasons, array())
    if(not(ageOk)) {
      addTo(reasons, 'age_below_minimum')
    }
    if(not(debtOk)) {
      addTo(reasons, 'debt_exceeds_limit')
    }
    if(not(employmentOk)) {
      addTo(reasons, 'employment_required')
    }

    map(
      'decision', 'denied',
      'reason', 'Profile does not meet requirements.',
      'reasons', reasons
    )
  }
})

treeSave(agent, 'decisionAgent2.json')
agent