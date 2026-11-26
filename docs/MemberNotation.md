# Member notation and callable expressions

Dot notation is a property read followed, optionally, by an expression call:

```chariot
setq(service, mapValue("client", map("send", func(message) { message })))
setq(send, service.client.send)
array(service.client.send(42), call(send, 42), (send)(42))
```

The final value is an array containing `42`, `42`, `42`. The functions stored in
these maps are ordinary function-valued fields: no receiver is injected.
See [classes and interfaces](ClassesAndInterfaces.md) for declared receiver methods
and [executable examples](../examples/objects/README.md) for complete programs.

## Parsing and evaluation

- Member reads and calls are postfix operations chained left to right:
  `factory().client.send(message)` reads the returned object's client and calls
  its send member. `object.method` extracts the member without invoking it.
- A receiver/callee is evaluated once, then call arguments once, left to right.
  Parenthesized callables and zero-argument calls are supported, including
  `(func(x) { x })(42)` and `(func() { func(x) { x } })()(42)`.
- Named calls preserve existing precedence: builtins, registered script
  functions, then callable scope variables. `(name)()` explicitly calls a scope
  value rather than selecting a builtin/registered function of that name.
  Special forms such as `setq`, `declare`, `class`, and `interface` retain their
  own named syntax; they are not ordinary function values.
- A newline before `(` ends a completed expression, including a bare name;
  `(` on the same line binds as a call. Explicit dots continue member chains
  across lines in either position (`object` then `.method()`, or `object.` then
  `method()`). Semicolons are optional explicit statement delimiters, never
  mandatory defensive terminators. Go AST renderers and shared TypeScript/Visual
  DSL codegen separate statements with newlines and keep call parentheses on
  the callee's line. Multiline arguments, arrays, and declaration/control-flow
  headers remain supported.
  See the [complete continuation and compatibility rules](ChariotLanguageDesign.md#newlines-and-expression-continuation).
- Dot names are identifiers. For dynamic keys or keys that are not identifiers,
  use `getProp(object, key)`; writes use `setProp(object, key, value)`.
  Neither `object.field = value` nor optional chaining (`object?.field`) exists.
- Decimal/exponent number literals and quoted strings retain their meaning.
  Invalid punctuation and malformed chains are parser errors with locations.

For example, the grouped call below is a separate statement, not a call on the
preceding value. This also applies when that preceding expression is a bare name:

```chariot
setq(send, func(message) { message })
send
(send)(42)
```

To call `send`, write `send(42)` or `(send)(42)`, not `send` followed by `(42)`
on the next line. An explicit dot can continue a chain across a newline before
or after the dot; the method's call parentheses still stay on its name's line.

## Supported receivers

Dot reads use the canonical property/member path, not a separate map/object model.

| Receiver | Read and missing-member behavior | Write path |
| --- | --- | --- |
| Script instance | Declared field or bound declared method; unknown member is an error | `setProp` validates an existing field's contract; no field deletion/addition |
| `MapValue` from `map` or `mapValue` | Exact key; absent key returns `DBNull` (null) | Existing map `setProp` semantics; maps are not schema-frozen |
| `MapNode`, `SimpleJSON`, `JSONNode`, `TreeNodeImpl`, `XMLNode`, `Plan`, native Go maps | Their existing `getProp` helpers, including each helper's missing-property behavior; do not assume all return null | Existing type-dispatched `setProp` where supported; dot reads add no new write capabilities |
| Host wrapper or registered raw Go object | Exported method by exact Go name, otherwise legacy reflected field lookup (case-insensitive); absent/inaccessible members error | Existing host `setProp`/`setHostProperty` reflection rules |
| Go extension implementing `MemberReader` | Its `ResolveMember` implementation | `MemberWriter.SetMember` when implemented |

Numbers, strings, and arrays do not gain arbitrary named members. Null, `DBNull`,
and typed-nil receivers cannot be dereferenced. `mapValue().missing` returns null;
`mapValue().missing.next` errors when attempting to read on that null. A declared
class field containing null is valid only where its contract permits it (`V`).

## Binding and compatibility

Declared script methods and reflected host methods bind the original receiver.
`setq(bound, object.method)` followed by `bound(args)` or `call(bound, args)`
keeps that receiver. Host pointer receivers are retained, including after a host
registry entry is replaced. Method names are case-sensitive; legacy host field
case-insensitivity does not make `host.add` an alias for `host.Add`.

Function-valued fields on maps, instances, or host objects are returned unchanged;
they do not get an extra `self` argument. A non-callable member can be read but
calling it is an error. Host arity/conversion errors, errors returned by Go, and
reflected panics surface as execution errors. Dot notation is not a sandbox and
does not introduce a new host exposure/permission model.

`getProp`, `setProp`, `call`, `callMethod`, `callHostMethod`, `getHostProperty`, and
`setHostProperty` remain available. Legacy serialized `FuncCall` nodes with a
dotted name resolve through member access; newly parsed dotted calls use
`MemberAccessNode` plus `FunctionCallNode`. Consumers of saved ASTs must handle
those shapes rather than splitting arbitrary call names themselves. Node maps
retain source positions where supported, but debugger chain stepping/stack
acceptance is still pending.

## Evidence and validation boundary

Inspected test cases cover postfix chains, single evaluation/order, named-call
precedence, map/JSON/tree helper parity, extracted host/script receivers, pointer
methods, old APIs, error handling, AST/source round trips, and function persistence:
[member tests](../services/go-chariot/chariot/member_access_test.go),
[class tests](../services/go-chariot/chariot/classes_test.go), and
[audit regression tests](../services/go-chariot/chariot/audit_regression_test.go).
Final validation on **2026-10-25** passed the fresh parent Go package suite,
affected-package race tests, and `go vet`. New combined HostObject/map/class/interface
scenarios passed through DevTest listeners, REST sessions, and MCP execution under
the race detector for ten repetitions. All four CLI examples produced exactly
their expected outputs. Shared codegen build/tests (10 tests), frontend Source-node
tests (3 tests), and the Visual DSL production build passed; the latter reported
bundle-size/Browserslist warnings, not a clean full frontend typecheck (7 known
pre-existing TypeScript errors remain).

See the [dated validation record and commands](notes/Chariot%20Objects%20and%20Dot%20Notation%20TODO.md#final-validation-record--2026-10-25).
Debugger stepping, dedicated editors/inspectors, and broad legacy-fixture and
host-exposure review remain open.

**Newline/codegen follow-up — 2026-10-27:** shared codegen emits newline-separated
statements without automatic semicolons. Generated-source CLI checks cover grouped
next statements after calls and bare names, nested blocks/declarations, and mixed
visual/Source output; all four semicolon-free object examples are also checked.
See the [follow-up validation record](notes/Chariot%20Objects%20and%20Dot%20Notation%20TODO.md#newlinecodegen-follow-up--2026-10-27)
for the opt-in workflow and results; the earlier runtime/race results above are historical.
