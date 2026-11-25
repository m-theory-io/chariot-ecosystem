class(Person) {
    field(name, "S", "Ada")
    field(items, "A", array())
    field(callback, "F", func(x) { x })
}
method(Person, hello,
    withSignature(func(self) { self.name }, signature(array("V"), "S")))
method(Person, rename, func(self, name) {
    setProp(self, "name", name)
    self.hello()
})
// Both named-attribute constructors are supported.
setq(a, new(Person, map("name", "First")))
setq(b, new(Person, mapValue("name", "Ada")))
setq(alias, a)
setq(rename, a.rename)
rename("Grace")
addTo(a.items, 42)
// Adding a fresh method after construction is allowed; replacement is not.
method(Person, label, func(self) { self.hello() })
// Final result: ["Grace", "Grace", "Ada", 1, 0, 42, "Grace"].
array(a.hello(), alias.name, b.name, length(a.items), length(b.items),
    a.callback(42), a.label())
