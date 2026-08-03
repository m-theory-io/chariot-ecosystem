// bootstrap.ch - loads startup runtime environment in dev server mode

// 1. connect to Couchbase
cbConnect('testCluster', '', '', '')
// 2. connect to RDBMS
sqlConnect('mysql1', '', '', '', 'justpaid-unity')
// 3. load usersAgent
declareGlobal(usersAgent, 'T', treeLoad('usersAgent.json'))
// 4. load ETL
declareGlobal(Fan47KtoSQL, 'T', treeLoad('Fan47KtoSQL.json'))
// 4.1 load lease decision
declareGlobal(decisionAgent2, 'T', treeLoad('decisionAgent2.json'))
// 5. load BDI
declareGlobal(BDI, 'T', treeLoad('bdi_tree.json'))
// 6. instantiate plan vars
declareGlobal(pThermostat, 'P', getAttribute(BDI, 'thermostat'))
declareGlobal(pLeaseDecision, 'P', getAttribute(BDI, 'leasedecision'))
// 7. agent start
agentStartNamed('thermostat', pThermostat, 1, 3, 'polling')
agentBelief('thermostat', 'lower', 68)
agentBelief('thermostat', 'upper', 72)
agentBelief('thermostat', 'currentTemp', 65)
// 7.1 agent start
agentStartNamed('leasedecision', pLeaseDecision, 1, 0, 'callOnly')
agentBelief('leasedecision', 'minAge', 18)
agentBelief('leasedecision', 'maxDebt', 10000)
agentBelief('leasedecision', 'monthlyRent', 2500)
agentBelief('leasedecision', 'leaseTerm', 12)
agentBelief('leasedecision', 'employedDeposit', 1000)
agentBelief('leasedecision', 'unemployedDeposit', 2500)
agentBelief('leasedecision', 'applicantAge', 25)
agentBelief('leasedecision', 'applicantDebt', 10000)
agentBelief('leasedecision', 'applicantEmployed', true)

