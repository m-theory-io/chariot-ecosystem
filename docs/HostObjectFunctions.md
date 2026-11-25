# Chariot Language Reference

## Host Object Functions

Chariot supports integration with host (Go) objects, allowing scripts to create, store, retrieve, and invoke methods on objects managed by the runtime. This enables advanced interoperability with Go code and external systems.

## Dot access and bound methods

See [Member notation](MemberNotation.md) for canonical access, binding, and null
rules. On an exposed host object, `host.Add(2)`, `setq(bound, host.Add)` followed
by `bound(2)`, and `call(host.Add, 2)` use the actual receiver, including pointer
receivers. Exported method names are exact/case-sensitive; legacy reflected field
lookup remains case-insensitive. Ordinary script function-valued host fields do
not receive an injected receiver. Existing named host/property APIs remain valid.

Bound host methods satisfy the `F` callable type; host wrappers and script
instances satisfy `O`. This does **not** make hosts conform to script structural
interfaces: host/map conformance remains deferred. Host resources remain borrowed
in runtime clones and are not serializable class defaults or portable persistence
values. See [Classes and interfaces](ClassesAndInterfaces.md).

---

### Available Host Object Functions

| Function                | Description                                                      |
|-------------------------|------------------------------------------------------------------|
| `hostObject(name [, obj])` | Register a new host object with the given name, optionally wrapping an existing object |
| `getHostObject(name)`   | Retrieve a registered host object by name                        |
| `callMethod(objOrName, methodName, [args...])` | Call a method on a host object by name or reference, passing arguments |

---

### Function Details

#### `hostObject(name [, obj])`

Registers a new host object in the runtime.  
- `name`: The name to register the object under (string).
- `obj`: (Optional) The object to wrap. If omitted, creates an empty map object.

Returns a host object reference.

```chariot
// Register an empty host object
setq(myObj, hostObject('myObj'))

// Register an existing object (e.g., a map or struct)
setq(myObj, hostObject('myObj', someGoStructOrMap))
```

#### `getHostObject(name)`

Retrieves a registered host object by name and returns a reference.

```chariot
setq(objRef, getHostObject('myObj'))
```

#### `callMethod(objOrName, methodName, [args...])`

Calls a method on a host object.  
- `objOrName`: Either a host object reference or the name of a registered host object.
- `methodName`: The method to call (string).
- `args...`: Arguments to pass to the method.

Returns the result of the method call.

```chariot
// Call a method by object name
callMethod('myObj', 'DoSomething', 42, 'hello')

// Call a method by object reference
callMethod(objRef, 'SetValue', 'key', 'value')
```

---

### Notes

- Host objects can be Go maps, structs, or other values.
- Property access on host objects is supported via reflection (see also `getProp`/`setProp` in Polymorphic Functions).
- If a method or property does not exist, an error is returned.
- Host object integration is intended for advanced scenarios and Go/Chariot interoperability.

---
