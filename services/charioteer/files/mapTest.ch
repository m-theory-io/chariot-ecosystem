// Demonstrates creating a map of callable functions and saving it as a function library
declare(stlib, 'M', mapValue(
    'tenantModel', func() { 
        setq(t, 'tenantModel called')
        log(t)
        t 
    },
    'patientModel' func() {
        setq(t, 'patientModel called')
        log(t)
        t
    }
))
array(call(getProp(stlib, 'tenantModel')), call(getProp(stlib, 'patientModel'))
registerFunction('tenantModel', getProp(stlib, 'tenantModel'))
registerFunction('patientModel', getProp(stlib, 'patientModel'))
saveFunctions('stlib.json')

