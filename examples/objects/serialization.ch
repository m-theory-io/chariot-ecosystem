class(Person) {
    field(name, "S", "Ada")
}
method(Person, hello,
    withSignature(func(self) { self.name }, signature(array("V"), "S")))
setq(original, new(Person))
// Exchange data explicitly, not a captured method/class persistence envelope.
setq(text, toJSON(mapValue("name", original.name)))
setq(attributes, parseJSONValue(text))
// JSON parsing does not restore a class. Reconstruct with the live definition.
setq(restored, new(Person, attributes))
setProp(original, "name", "Grace")
// Final result: ["Ada", "Ada", "Grace", "O"].
array(attributes.name, restored.hello(), original.hello(), valueType(restored))
