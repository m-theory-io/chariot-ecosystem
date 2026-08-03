// Lease Decision BDI Agent
declare(minAge, 'N', 18)
declare(maxDebt, 'N', 10000)
declare(monthlyRent, 'N', 2500.00)
declare(leaseTerm, 'N', 12)
declare(employedDeposit, 'N', 1000.00)
declare(unemployedDeposit, 'N', 2500.00)

declare(applicantAge, 'N', 0)
declare(applicantDebt, 'N', 0)
declare(applicantEmployed, 'L', false)

declare(planName, 'S', 'LeaseDecision')
declare(planParams, 'A', array(applicantAge, applicantDebt, applicantEmployed))

declare(planTrigger, 'F', func() {
  true
})

declare(planGuard, 'F', func() {
  true
})

declare(evaluateLeaseDecision, 'F', func() {
  setq(ageOk, biggerEq(belief('leasedecision', 'applicantAge'), belief('leasedecision', 'minAge')))
  setq(debtOk, smallerEq(belief('leasedecision', 'applicantDebt'), belief('leasedecision', 'maxDebt')))
  setq(employmentOk, equal(belief('leasedecision', 'applicantEmployed'), true))
  setq(approved, and(ageOk, debtOk, employmentOk))

  if(approved) {
    setq(deposit, belief('leasedecision', 'employedDeposit'))
    setq(rent, belief('leasedecision', 'monthlyRent'))
    setq(term, belief('leasedecision', 'leaseTerm'))
    setq(startDate, formatDate(dateAdd(now(), 'day', 15), 'YYYY-MM-DD'))
    setq(offerText, concat('Pay ', deposit, ' deposit and ', rent, ' 1st-month rent today, to start your lease on ', startDate, '. Remaining rent of ', rent, ' for ', term, ' months, to be paid on the 1st of every month.'))

    setStepResult(map(
      'decision', 'approved',
      'agent', 'LeaseDecision',
      'offerName', 'Green Lease Plan',
      'offer', offerText,
      'monthly', rent,
      'term', term,
      'deposit', deposit,
      'startDate', startDate,
      'reasons', array('age_ok', 'debt_ok', 'employment_ok')
    ))
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

    setStepResult(map(
      'decision', 'denied',
      'agent', 'LeaseDecision',
      'reason', 'Profile does not meet requirements.',
      'reasons', reasons
    ))
  }
})

declare(planSteps, 'A', array(evaluateLeaseDecision))
declare(planDrop, 'F', func() { false })

declareGlobal(pLeaseDecision, 'P', plan(planName, planParams, planTrigger, planGuard, planSteps, planDrop))
setAttribute(BDI, 'leasedecision', pLeaseDecision)

treeSave(BDI, 'bdi_tree.json')
pLeaseDecision