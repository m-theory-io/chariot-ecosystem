// Run thermostat

// Plan agent
declareGlobal(p, 'P', getAttribute(BDI, 'thermostat'))

agentStartNamed('thermostat', p, 1, 3)
agentBelief('thermostat', 'lower', 68)
agentBelief('thermostat', 'upper', 72)
agentBelief('thermostat', 'currentTemp', 65)

// Execute
runPlanOnce(p) 