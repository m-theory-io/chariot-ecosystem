// Thermostat Agent
declareGlobal(BDI, 'T')

// Beliefs
declare(lower,'N', 68)
declare(upper,'N', 72)
declare(currentTemp,'N', 65)

// Plan definition
declare(name, 'S', 'Thermostat')
declare(params, 'A', array(upper, lower))

// Trigger fires when currentTemp outside range
declare(trig,'F', func(){ or(smaller(belief('thermostat','currentTemp'), belief('thermostat','lower')), bigger(belief('thermostat','currentTemp'), belief('thermostat','upper'))) })

// Default guard to true
declare(guard,'F', func(){ true })

// step1 computes reflex booleans from beliefs
declare(step1,'F', func(){
        logPrint('Performing step 1')
        setq(needHeat, smaller(belief('thermostat','currentTemp'), belief('thermostat','lower')))
        setq(needCool, bigger(belief('thermostat','currentTemp'), belief('thermostat','upper')))
        if(needHeat) {
            logPrint('turn A/C off')
            setStepResult(map('action', 'heating_on', 'message', 'turn A/C off'))
        } else {
            if (needCool) {
                logPrint('turn A/C on')
                setStepResult(map('action', 'cooling_on', 'message', 'turn A/C on'))
            } else {
                logPrint('temp OK')
                setStepResult(map('action', 'none', 'message', 'temp OK'))
            }
        }
    })
declare(steps, 'A', array(step1))
declare(drop, 'F', func() { false })

// Declare plan agent
declare(tstatAgent, 'P', plan(name, params, trig, guard, steps, drop))

// Add to BDI tree
setAttribute(BDI, 'thermostat', tstatAgent)

// Save the BDI tree
treeSave(BDI, 'bdi_tree.json')
