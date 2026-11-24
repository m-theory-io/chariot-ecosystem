# Script classes and structural interfaces

Classes compose values; there is no inheritance, implicit receiver, constructor
hook, member assignment, deletion, overload resolution, or `implements` clause.
Names are case-sensitive. Definitions live in lexical runtime scopes, not a
process-global registry. Use `declareGlobal(PublicPerson, "V", Person)` to export
a definition to bootstrap sessions, just as for other bootstrap values.

## Exact API

```chariot
class(Person) {
    field(name, "S", "")
    field(items, "A", array())
    field(requiredValue, "V")
}
method(Person, hello,
    withSignature(func(self) { self.name }, signature(array("V"), "S")))
method(Person, rename, func(self, name) {
    setProp(self, "name", name)
    self.hello()
})
setq(p, new(Person, mapValue("name", "Ada", "requiredValue", 42)))
setq(greeting, p.hello)
greeting()

interface(Greeter) {
    requires(hello, signature(array(), "S"))
}
satisfies(p, Greeter)
assertSatisfies(p, Greeter).hello()
```

Both `map(...)` and `mapValue(...)` currently construct a `MapValue` from named
key/value pairs and work as attributes to `new`. `map` is registered with ETL
functions; `mapValue` is registered with core value functions. A restricted
embedding must register the constructor it uses. `new(Class)` accepts an omitted empty
attribute map if no required fields exist. A field without a default is required,
including a `V` field. Explicit `V` values may be null. Unknown attributes and
writes, duplicate definitions/fields/requirements/methods, field/method
collisions, and mismatched types are errors. Declaration bodies accept only
`field` or `requires` respectively; defaults are evaluated once at declaration.
Member names may also be quoted strings.

The field schema is frozen when the declaration completes: fields cannot be
added, removed, or redeclared. Fields are publicly readable, and writes go
through `setProp`; there is no private-field syntax. This is schema freezing,
not immutable instances or a fully frozen class. Unknown member reads are errors,
unlike missing `MapValue` keys, which return null. A declared `V` field explicitly
set to null remains a valid field; omitting a required `V` field is an error.
Reading a member on null is always an error. See [Member notation](MemberNotation.md).

Instances and assigned attributes have reference semantics. Only default values
are copied for each construction; arrays/maps and nested script graphs are
independent. Default graph copying preserves aliases/cycles within the copied
graph and the live class identity/declaration scope, while isolating private
captured frames. Later lexical bindings remain visible; defaults are not a frozen
snapshot of the declaration environment. Host resources and unsupported mutable defaults are rejected;
provide them through named attributes instead. Script instances can contain
aliases/cycles. Displays print class names without recursively traversing fields.
Function-valued fields do not bind receivers. Methods are shared per class and
bind the original instance even when extracted. New method names may be added
after construction; existing methods cannot be replaced. Interface checks are
not cached, so additive registration does not invalidate an earlier check.

## Function contracts

`signature(array("N", "S"), "L")` describes exact argument count, parameter types,
and return type. `withSignature(function, signature)` returns an annotated copy,
without changing the original function. Stored methods require the first
parameter to be literally `self`; an annotated method includes `"V"` for that
receiver. Interface requirements omit the receiver. Ordinary functions remain
permissive unless explicitly annotated (missing arguments are still null and
extra arguments ignored). Annotations validate both normal and explicit-return
results through ordinary calls, expression calls, and `call`.

Contracts use `V` (any), `N`, `S`, `L`, `D`, `A`, `M`, `F`, `O`, `J`, `X`, `T`,
`P`, `E`; they use existing runtime type checks, with strict date validation.
`F` includes `FunctionValue` and runtime-aware `Callable` values (including bound
script and host methods); `O` includes script instances and host wrappers.
These checks also apply to `declare`/`declareGlobal`: `declare(f, "F", p.hello)`
and `declare(o, "O", p)` are supported. `valueType(p)` returns `"O"`, and
`valueType(p.hello)` returns `"F"`. A class definition is not an instance.
There are no class-name or interface-name parameter contracts yet. Interface
satisfaction requires script **methods**, not function-valued fields, exact
public parameter/return metadata, and no variance. Unannotated methods do not
satisfy typed requirements: runtime behavior alone cannot prove return types.
`satisfies` returns a boolean; `assertSatisfies` returns the original instance or
an explanatory error. It checks a boundary, not future arbitrary Go-side edits.
Host objects and callable maps do not participate in structural conformance.

## Runtime copying and persistence

Runtime clones and bootstrap sessions copy script definitions, instances,
arrays/maps, functions, extracted receivers, and their complete lexical scope
graphs using one identity memo. Aliases/cycles are preserved in the copy, not
shared with the source. ASTs are treated as immutable. Builtins are registered
against the destination runtime; method invocation does not reacquire
`WithExecution`. Registered host resources remain borrowed, as before.
Arbitrary external/legacy containers (for example Plan/JSON/tree payloads or Go
objects holding script values) are not graph-aware script containers and retain
their previous copy behavior. This is not a sandbox or a general host deep clone.

Declarations use existing FuncCall/Block AST shapes. Both AST serializers and
source/pretty rendering can round-trip executable declaration programs. Go source
renderers and shared TypeScript/Visual DSL codegen separate statements with
newlines, without mandatory semicolons. A newline before `(` ends a completed
expression, even a bare name, so a grouped next statement cannot accidentally
call the previous result. Keep call parentheses on the callee's line; explicit
dots continue chains across lines. Multiline arguments, arrays, and headers are
preserved. See [newline and call rules](MemberNotation.md#parsing-and-evaluation).

`ScriptValueToNative(value)` is the checked **Go API**, not a script builtin. It supports
acyclic, non-aliased data-only classes/instances, interfaces, signatures, script
arrays/maps, primitives, and scope-free functions. The matching existing native
value decoder reconstructs those types. Signature metadata also survives normal
function value/library save/load hooks. Old functions without metadata still load.
No arbitrary host resources are serialized.

**Explicit limitations:** the legacy format cannot preserve object graph
identity or lexical scopes. Checked persistence rejects repeated references,
cycles, captured closures (therefore normally declared class methods), extracted
bound methods, map metadata, and unsupported values. Save executable declaration
source/AST to reconstruct method-bearing classes, rather than claiming a closure
or instance-graph snapshot. Legacy conversion hooks cannot return errors; script
value failures become a `_value_type: "script_error"` record whose native decoder returns an error
value. Use the checked boundary to handle failures conventionally. Do not feed
cyclic legacy containers to the older unchecked generic container serializers.

### Wire JSON is not persistence

`ValueToJSONChecked` (Go API) recursively projects an instance's current fields
as a plain JSON object, including nested arrays/maps/instances. It excludes the
class definition, method table, and lexical scope; it does not recreate instances
on decode. Repeated references are projected independently, but active-path cycles
are errors. Unsupported field values can also fail projection; this is not a
general callable/resource serializer. `ValueToJSON` uses a JSON-safe error object
when its checked counterpart fails. HTTP listener response serialization uses the
checked projection before committing the status, including `httpResponse` bodies;
a projection failure produces an error response, not the requested success status.

Script `toJSON(instance)` is a **different path**: it uses native persistence
conversion and produces a tagged instance/class/fields envelope for supported
data-only values. For a captured method or other unsupported instance graph it can
produce a `script_error` JSON record rather than throw. `parseJSONValue` parses
JSON data; it is not the native tagged-value decoder and does not restore classes
or methods. Prefer an explicit data map plus `new` when exchanging script JSON.
See [JSON functions](JSONFunctions.md#objects-persistence-and-wire-json).

### Source, tools, and validation status

Persist executable declaration source/AST for method-bearing models and execute it
in a fresh scope. Re-execution in a scope already containing the same definition
fails with a duplicate-name error. This reconstructs definitions and initial state,
not later mutations or live closure identity. Declarations retain canonical
`FuncCall(name, [VarRef, Block])` shapes; reads use `MemberAccessNode`, and expression
calls use `FunctionCallNode`. Both runtime AST encodings and source/pretty rendering
have round-trip test cases. Old dotted-name `FuncCall` values remain supported.

Visual DSL/codegen now retains imported ASTs in read-only Source nodes and generates
source for declarations, member reads, and expression calls. Structured function
bodies, not incomplete source headers, drive generation. Unknown AST types are
preserved but code generation fails explicitly. Dedicated class/interface/member
editors, class-specific inspectors, debugger chain stepping, portable identity-graph
snapshots, host/map interface conformance, and broad listener/MCP UI acceptance
remain deferred or pending; Source-node support is not a dedicated visual editor.

Runnable inputs and expected results: [object examples](../examples/objects/README.md).
Implementation decisions and remaining acceptance work:
[object TODO](notes/Chariot%20Objects%20and%20Dot%20Notation%20TODO.md).

**Final validation — 2026-10-25:** the fresh parent Go suite, affected-package
race tests, and `go vet` passed. New combined HostObject/map/class/interface
scenarios passed ten race-enabled repetitions through DevTest listener handlers,
REST sessions, and MCP execution, including clone/session isolation and borrowed
host effects. All four examples executed through the CLI with exact expected
outputs. Shared codegen build and tests (10), frontend Source-node tests (3), and
the Visual DSL production build passed. The production build had bundle-size and
Browserslist warnings; it does not establish a clean full frontend typecheck:
7 known pre-existing TypeScript errors remain.

The [validation record](notes/Chariot%20Objects%20and%20Dot%20Notation%20TODO.md#final-validation-record--2026-10-25)
lists historical commands and working directories. Runtime/API scenario acceptance does not complete the deferred UI/editor,
debugger, host/map conformance, broad legacy-fixture, or exposure/ownership reviews.

**Newline/codegen follow-up — 2026-10-27:** automatic Chariot semicolons were
removed from AST, mixed visual/Source, and generated parameter-default output,
and from the object examples. User-provided Source text and literal semicolons
are preserved. The [follow-up record](notes/Chariot%20Objects%20and%20Dot%20Notation%20TODO.md#newlinecodegen-follow-up--2026-10-27)
documents fresh shared/frontend builds, Source tests, and generated-source/example
execution through the Go CLI; it does not claim a fresh full Go/race run.
