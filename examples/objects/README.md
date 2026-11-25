# Object examples

The four Chariot inputs below are complete interpreter inputs, not fragments. Each returns a final
observable array; no listeners, host registrations, databases, external services,
file writes, or bootstrap configuration are needed. Statements use newlines,
not mandatory semicolons. A newline before `(` starts a separate grouped
statement, even after a bare name; calls keep `(` on the callee's line. Explicit
dots continue member chains across lines, and multiline arguments, arrays, and
headers remain supported. See the [newline/call rules](../../docs/MemberNotation.md#parsing-and-evaluation).

Use the repository's [interpreter CLI](../../services/go-chariot/cmd/chariotctl/main.go)
with its `-f` option and one of these files. From the repository root, invoke
`go run ./services/go-chariot/cmd/chariotctl -f examples/objects/classes.ch`,
substituting the desired example path. The CLI registers the full builtin set,
including both map constructors, and prints the final value. Array/string display
formatting may differ from the JSON-like notation used below.

| Input | Expected final elements |
| --- | --- |
| [member-calls.ch](member-calls.ch) | `[42, 42, 42, true]` — chaining, ordinary function fields, extracted calls, missing map key |
| [classes.ch](classes.ch) | `["Grace", "Grace", "Ada", 1, 0, 42, "Grace"]` — receiver binding, aliases, independent defaults, both attribute constructors, late additive method |
| [interfaces.ch](interfaces.ch) | `[true, 42, 42, "F", "O"]` — exact signatures, checked boundary, callable/instance type compatibility |
| [serialization.ch](serialization.ch) | `["Ada", "Ada", "Grace", "O"]` — explicit JSON data transfer and reconstruction with a live class |

The JSON example deliberately serializes a data map. Script `toJSON(instance)`
uses native persistence conversion, not the HTTP field projection, and a normal
captured method can yield a tagged `script_error` record. `parseJSONValue` does not
invoke the native class decoder. This example makes no closure or identity snapshot
claim, and reconstructs methods from the already executed class declaration.

In an embedding that registers only core value functions, prefer `mapValue` or
register ETL's `map` explicitly. The CLI registers both. Run each example as a
separate program in a fresh scope; definitions cannot be redeclared in place.

References: [members](../../docs/MemberNotation.md),
[classes and interfaces](../../docs/ClassesAndInterfaces.md),
[JSON boundaries](../../docs/JSONFunctions.md#objects-persistence-and-wire-json),
[implementation decisions/TODO](../../docs/notes/Chariot%20Objects%20and%20Dot%20Notation%20TODO.md).

**Final validation — 2026-10-25:** all four examples executed through the CLI in
fresh scopes and produced exactly the expected final elements above. The fresh
parent Go suite, affected-package race tests, and `go vet` passed; combined
HostObject/map/class/interface DevTest, REST-session, and MCP scenarios also passed
ten race-enabled repetitions. Shared codegen build/tests (10 tests), frontend
Source-node tests (3 tests), and the Visual DSL production build passed, with
bundle-size/Browserslist warnings. This is not a clean full frontend typecheck:
7 known pre-existing TypeScript errors remain.

Exact example commands, working directories, and the validation scope are in the
[dated validation record](../../docs/notes/Chariot%20Objects%20and%20Dot%20Notation%20TODO.md#final-validation-record--2026-10-25).
Dedicated editors, debugger stepping, host/map structural conformance, and broad
legacy-fixture and exposure/ownership review remain open.

**Newline/codegen follow-up — 2026-10-27:** all four semicolon-free inputs were
executed again through a freshly built Go CLI with exact expected outputs. An
opt-in [shared codegen test](../../packages/chariot-codegen/test/chariotSource.test.ts)
checks these examples and generated programs, including grouped next statements
after calls/bare names and nested declaration bodies. From the repository root,
build with `npm --prefix packages/chariot-codegen run build`, then run
`CHARIOT_CODEGEN_GO_TESTS=1 node --experimental-strip-types --test packages/chariot-codegen/test/*.test.ts`
(Node with type-stripping support and a local Go toolchain required). Without
the environment flag, only the Go-backed test is skipped. Temporary generated
source and CLI binaries are cleaned up automatically. See the
[follow-up validation record](../../docs/notes/Chariot%20Objects%20and%20Dot%20Notation%20TODO.md#newlinecodegen-follow-up--2026-10-27).

## Editable Visual DSL example — 2026-10-27

[classes-diagram.json](classes-diagram.json) is a complete editable diagram, not
an interpreter input or live-instance snapshot. It is an exact copy of the
[shared all-ten CLASSES fixture](../../packages/chariot-codegen/test/fixtures/classes-complete.json),
with the same diagram name and stable node IDs under a user-facing filename.

In Visual DSL choose **Paste JSON**, paste the file contents, and click
**Load Diagram** (this replaces the working diagram). Open **Generate .ch Code**,
then **Generate Code**, and **Copy** the output into Charioteer to Run in a fresh
scope. The final elements are `[42,42,true,false,7,0]`. All ten CLASSES constructs
participate: class/field/method/new, interface/requires, signature/withSignature,
and positive/negative satisfies plus assertSatisfies. The example also exercises
bound extraction, mutation, both attribute-map constructors and default fields.

To iterate without JSON edits, right-click the Counter Method named `value`
(node `value-method`), choose **Properties...**, change **Body** to
`return(add(self.count, 1))`, and **Save Properties**. Regenerate and execute in a
fresh scope: `[42,43,true,false,7,0]`.

See the [CLASSES user guide](../../services/visual-dsl/CLASSES.md) for all ten
dialogs, required fields/default toggles, receiver signatures and **Create Nesting** /
**Exit Nesting** controls. Only Field children belong inside Class, and only
Requires children inside Interface; Method is registered outside declarations.

The [dated acceptance record](../../docs/notes/Visual%20DSL%20Classes%20Category%20TODO.md#validation-record--2026-10-27)
contains exact commands and current results: shared build and 38 tests including
CLI execution, 27 frontend tests, 12 Chrome authoring cases, 2 cross-app harness
cases and the production build passed. Two baseline transform-dialog
`InputProps.list` type errors remain. Cross-app checks use the real local
frontend/Charioteer proxy/clipboard/Monaco/Go CLI, but synthetic backend execution
envelopes and login/session metadata, with streaming disabled. Named Load,
server-backed diagram persistence and deployed-stack authentication/streaming/session
validation remain pending; no deployment was performed. These results do not
retroactively expand the scope of the older dated records above.
