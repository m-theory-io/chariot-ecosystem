There is a more general architecture hiding underneath the thermostat example.

A temperature sensor is just one kind of **signal source**. The better abstraction is probably:

```text
signal source -> normalize reading/event -> update belief(s) -> activate BDI agent -> agent performs action
```

So hardware sensors, Fed interest rates, file changes, HTTP APIs, market prices, queue messages, webhooks, database rows, and cron ticks can all fit the same pattern.

I’d generalize the API from “sensor” to “signal” or “source”:

```chariot
signalRegister(name, kind, config)
signalRead(name)
signalStartBeliefFeed(sourceName, agentName, beliefName, intervalSeconds)
signalStopFeed(sourceName)
signalList()
```

Then `sensorRegister` could either be a convenience alias or a specialized provider family under the signal system.

For the Fed rate example:

```chariot
signalRegister('fedFundsRate', 'httpJson', map(
  'url', 'https://...',
  'path', 'data.0.rate',
  'scale', 1
))

signalStartBeliefFeed('fedFundsRate', 'macroPolicyAgent', 'fedFundsRate', 3600)
agentStartNamed('macroPolicyAgent', pMacroPolicy, 1, 0, 'eventOnly')
```

Then the BDI plan trigger might be:

```chariot
declare(rateChangedTrigger, 'F', func() {
  not(equal(
    belief('macroPolicyAgent', 'fedFundsRate'),
    belief('macroPolicyAgent', 'previousFedFundsRate')
  ))
})
```

And the step can decide what to do:

```chariot
declare(evaluateRateMove, 'F', func() {
  setq(rate, belief('macroPolicyAgent', 'fedFundsRate'))
  setq(previous, belief('macroPolicyAgent', 'previousFedFundsRate'))

  if(bigger(rate, previous)) {
    setStepResult(map(
      'event', 'fed_rate_increase',
      'rate', rate,
      'previous', previous,
      'action', 'tighten_credit_policy'
    ))
  }

  agentBelief('macroPolicyAgent', 'previousFedFundsRate', rate)
})
```

The useful provider families become:

```text
hardware
  sysfs temperature
  GPIO
  I2C/SPI later

time
  interval
  cron

network
  httpJson
  webhook
  websocket

data
  sqlQuery
  couchbaseQuery
  fileWatch

messaging
  nsq
  mqtt
  kafka later

manual/dev
  static
  memory
  random
  command
```

For BDI, the stable idea is not “polling sensors.” It is **belief feeds**.

A feed can be:

```text
polling feed
  periodically reads a source

event feed
  receives pushes/webhooks/messages

manual feed
  MCP or user sets a belief

derived feed
  transforms one or more beliefs into another
```

And agents stay simple:

```chariot
agentStartNamed('thermostat', pThermostat, 1, 0, 'eventOnly')
agentStartNamed('macroPolicyAgent', pMacroPolicy, 1, 0, 'eventOnly')
```

They react when their beliefs change.

So yes: I’d broaden the design from `sensor*` to `signal*`, with `sensor` and `pin` as provider-specific subdomains:

```chariot
signalRegister(...)
signalStartBeliefFeed(...)

pinRegister(...)
pinWrite(...)
```

That gives us a unified Chariot model:

```text
observe world -> update beliefs -> deliberate with BDI -> act on world
```

Thermostat and Fed-rate monitoring are the same architecture; only the source and action providers differ.