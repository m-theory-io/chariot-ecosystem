// Run thermostat

// Plan agent
declareGlobal(pThermostat, 'P', getAttribute(BDI, 'thermostat'))

agentStartNamed('thermostat', pThermostat, 1, 3)
agentBelief('thermostat', 'lower', 68)
agentBelief('thermostat', 'upper', 72)
agentBelief('thermostat', 'currentTemp', 65)

// Execute
runPlanOnce(pThermostat) 
