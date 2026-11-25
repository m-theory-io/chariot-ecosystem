// Ordinary function-valued map fields never receive an injected self.
setq(service, mapValue("client", map("send", func(message) { message })))
setq(send, service.client.send)
// Final result: [42, 42, 42, true].
array(service.client.send(42), call(send, 42), (send)(42),
    isNull(mapValue().missing))
