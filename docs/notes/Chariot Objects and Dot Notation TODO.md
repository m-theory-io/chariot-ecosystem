# Chariot Objects, Interfaces, and Dot Notation — Implementation TODO

Status: Core implementation and source/AST tooling validated; deferred tooling and broad release reviews remain open.
Date: 2026-10-27

Checked boxes record implementation and the completed validation below, not
blanket release acceptance. Fresh parent and integration-agent results from
2026-10-25 are recorded here as historical evidence. The 2026-10-27 follow-up
below validates newline-style TS codegen/examples after the parser fix; it does
not rerun the full Go/race suites. Unsupported tooling and broader reviews remain unchecked.

Canonical references: [classes and interfaces](../ClassesAndInterfaces.md),
[member notation](../MemberNotation.md), and [object examples](../../examples/objects/README.md).

## Direction

Implement three independently reviewable features in dependency order:

1. Member access and callable expressions for existing types.
2. Classes with explicit receiver methods and composition.
3. Structural interfaces backed by a consistent signature model.

Use Chariot's function-oriented syntax. Do not introduce inheritance, implicit
`this`, member assignment operators, or multiple competing method syntaxes in
the initial release. The following Phase 0 record describes the approved syntax
implemented in this work; it is not blanket approval of all tooling/release gates.

### Phase 0 implementation decision record

- `class(Name) { field(name, "S", "") }`, `method(Name, hello, func(self) { self.name })`,
  and `new(Name, map("name", "Ada"))` are canonical. `mapValue` works too; both
  constructors return `MapValue` (registration differs in restricted embeddings).
- Class/interface declaration names are identifiers referring to lexical values;
  field/method/requirement names accept symbols or strings. Class field schemas
  freeze after declaration, but method tables remain additive even after `new`.
  Replacement, deletion, collisions, inheritance, and initialization hooks are out.
- `interface(G) { requires(hello, signature(array(), "S")) }` describes public args.
  An annotated stored method uses `withSignature(func(self) { self.name },
  signature(array("V"), "S"))`: explicit `self`, including its `V`, belongs only to
  the stored function signature. Metadata is exact; there is no variance.
- Field/signature contracts: `V N S L D A M F O J X T P E`. `F` includes runtime
  `Callable` values/bound methods; `O` includes script instances/host wrappers,
  including in `declare`/`declareGlobal`. No class/interface-name contracts yet.
- Postfix reads/calls chain left to right with single receiver/argument evaluation.
  Named-call precedence remains builtin → registered function → scope callable;
  `(name)()` selects the scope value. Writes remain `setProp`, not assignment.
- A newline before `(` ends a completed expression, including a bare name;
  calls require same-line parentheses. Explicit leading/trailing dots continue
  chains across newlines; multiline arguments, arrays, and headers are preserved.
  Semicolons remain optional, not mandatory defensive statement terminators.
  Go renderers and shared TS/Visual DSL codegen emit newline-separated statements.
- Declared/reflected methods bind receivers; ordinary function fields do not.
  Missing `MapValue` keys yield null; unknown class members error. Required fields
  cannot be omitted even with contract `V`; explicit null is permitted by `V`.
  Dereferencing null errors. Host fields retain legacy case-insensitive lookup,
  while method names remain exact.
- Runtime/default graph copies preserve graph structure with their documented
  scope/ownership rules; checked native persistence rejects aliases/cycles,
  captured scopes, and bound calls. Wire JSON is a field projection, not that
  persistence format. Re-executed source/AST reconstructs declarations, not a live
  snapshot. Source-node visual round trips do not imply dedicated visual editors.

## Phase 0 — Lock down language contracts

- [x] Review current lexer/parser, AST evaluation, function dispatch, property
  dispatch, HostObject reflection, and closure behavior; record canonical APIs.
- [x] Add characterization tests for existing `getProp`, `setProp`, `call`, named
  calls, function values, maps, JSON/tree values, and HostObject methods.
- [x] Decide which existing types support members in the first release and
  document each type's read, write, and missing-member behavior.
- [x] Specify precedence and evaluation order for member access and calls;
  evaluate receivers and arguments once, left to right.
- [x] Specify resolution precedence for built-ins, registered functions, and
  callable variables without accidentally changing existing named-call behavior.
- [x] Specify bound-method behavior and distinguish methods from function-valued
  fields: only declared/reflected methods bind a receiver automatically.
- [x] Decide whether null receivers are errors; defer optional chaining.
- [x] Specify class instance identity/reference semantics, field visibility,
  exact case-sensitive names, and field/method collision errors.
- [x] Record approved implementation `class`, `field`, `method`, and `new` forms, including the syntax
  for referencing a class as a value and the type codes accepted by fields.
- [x] Define signatures once for ordinary functions, methods, and interfaces.
  Resolve optional annotations, argument-count validation, and compatibility
  with existing permissive functions before finalizing interface syntax.
- [x] Inventory AST/value persistence, source rendering, debugging, cloning,
  and Visual DSL consumers; define format/version compatibility requirements.

Gate: syntax/semantics recorded above; parent characterization validation passed
on 2026-10-25 (see the validation record below).

## Phase 1 — Dot notation and callable expressions

Suggested branch: `feature/member-access`.

### Parser and AST

- [x] Add a real dot token without breaking decimal numbers or quoted strings.
- [x] Reject unsupported punctuation with a useful location-aware diagnostic
  rather than silently discarding it.
- [x] Add member-access and expression-call AST nodes while retaining canonical
  named-call behavior where needed for language special forms.
- [x] Parse `object.name`, `object.method(args)`, and chained expressions such as
  `service.client.send(message)` and `factory().method()`.
- [x] Support extracting a member and invoking a callable expression, with
  unambiguous grouping and zero-argument calls.
- [x] Add test cases for malformed chains, missing names, call precedence, whitespace,
  multiline expressions, decimals, and unsupported assignment syntax.

### Runtime integration

- [x] Define a canonical member-resolution path and callable invocation path.
- [x] Repair inconsistent existing HostObject property dispatch at its root;
  do not add compatibility wrappers around mismatched signatures.
- [x] Route supported map/JSON/tree member reads through their canonical
  property semantics; explicitly list any deferred types.
- [x] Bind exposed HostObject methods to the actual receiver, including pointer
  receivers; preserve registered exposure restrictions and exact method names.
- [x] Support both `object.method(args)` and `call(object.method, args)`.
- [x] Preserve function-valued field behavior without injecting a receiver.
- [x] Preserve existing property writes through `setProp` where supported;
  leave `object.field = value` out of scope.
- [x] Diagnose missing methods, non-callable members, invalid receivers, arity
  errors, and host conversion failures without panics.
- [x] Replace or reconcile the existing partial dotted-name runtime hook with
  normal member resolution; document any serialized-AST migration.

### Cross-cutting support and proof

- [x] Update AST serialization/deserialization and source rendering.
- [x] Retain member source positions in AST maps and location-aware errors.
- [ ] Accept stack traces and debugger stepping through chains (dedicated work).
- [x] Add test cases for single receiver/argument evaluation, bound-method extraction, nesting,
  host pointer methods, and old API compatibility using real behavior.
- [x] Add parse → serialize → deserialize → execute/source round-trip tests.
- [x] Retain new AST shapes in read-only Visual DSL Source nodes and generate source;
  unknown types are retained and fail codegen explicitly.
- [x] Publish examples and serialized-AST migration notes.
- [x] Run affected tests, examples, and race tests; record results.

Gate: implementation/test cases present for existing types without requiring
classes; affected tests, examples, and race validation passed on 2026-10-25.

## Phase 2 — Classes and receiver methods

Suggested branch: `feature/classes`, based on Phase 1.

Implemented declaration shapes: `class(Person) { field(name, "S", "") }`,
`method(Person, hello, func(self) { ... })`, and
`new(Person, map("name", "Ada"))`. See the Phase 0 decision record.

### Types and construction

- [x] Add class-definition and instance value types with runtime-scoped lookup.
- [x] Parse/evaluate class and field declarations with duplicate-name errors.
- [x] Validate field type codes and defaults at the appropriate boundary.
- [x] Implement `new` with named attributes, unknown-field rejection, required
  field handling, default application, and clear construction errors.
- [x] Give each instance independent mutable defaults; define copy/clone rules
  for host resources and other values that cannot safely be deep-copied.
- [x] Preserve reference semantics for aliases and method-driven mutation.
- [x] Reject undeclared field writes and invalid field types; define deletion
  policy rather than inheriting map deletion accidentally.

### Methods

- [x] Register shared per-type methods with an explicit `self` parameter.
- [x] Supply/bind the receiver for dot invocation and extracted methods.
- [x] Validate receiver and method declarations and reject field/method collisions.
- [x] Define method registration/redefinition policy before and after instances
  exist; prevent accidental cross-runtime registration leakage.
- [x] Specify constructor behavior: named-field initialization only,
  with no initialization hook.
- [x] Add test cases for mutation, aliases, multiple instances, recursion, method-to-method
  calls, extracted methods, errors, and function-valued fields.

### Runtime lifecycle and tools

- [x] Define persistence for class definitions and instances, including identity,
  cycles, and non-serializable members; do not serialize arbitrary host resources.
- [x] Define how runtime/bootstrap-session cloning copies definitions and instances
  and rebinds closures/receivers without leaking mutable runtime ownership.
- [x] Keep method invocation inside existing runtime execution/lifecycle rules;
  do not introduce nested execution-guard deadlocks.
- [x] Add non-recursive class/instance text displays and `F`/`O` classification.
- [ ] Add dedicated class/instance introspection and debugger inspectors.
- [x] Update source rendering, read-only Source-node codegen, and language references.
- [x] Add serialization, runtime-clone/bootstrap-session isolation, contract,
  default-scope, instance JSON projection, and HTTP regression test cases.

Gate: core construction/mutation/method paths and executable examples validated
on 2026-10-25; dedicated inspectors remain deferred.

## Phase 3 — Structural interfaces

Suggested branch: `feature/interfaces`, based on Phase 2 and approved signatures.

- [x] Finalize `interface`, `requires(..., signature(...))`, and signature syntax
  as recorded in Phase 0.
- [x] Define satisfaction rules for method names, caller-visible arity, and
  optional parameter/return contracts. Exclude the receiver from public arity.
- [x] Define exact matching versus assignability for annotated types; avoid
  introducing variance, generics, or overload resolution implicitly.
- [x] Implement structural satisfaction without mandatory `implements` clauses.
- [x] Implement `satisfies(value, Interface)` and `assertSatisfies` as a checked boundary
  with useful conformance diagnostics (no interface-named parameter contracts).
- [x] Define compatibility for unannotated functions: what can be checked from
  metadata and what must be checked at invocation time.
- [x] Initially require methods only; defer mutable-field requirements.
- [x] Decide initial conformance scope: script instances only.
- [ ] Implement HostObject/callable-map conformance (deferred; no host adapters claimed).
- [x] Use uncached checks and additive-only method registration; no replacement.
- [x] Add test cases for conforming/missing/unannotated/mismatched methods,
  bound receivers, receiver metadata validation, and diagnostics.
- [x] Extend signature/interface persistence, cloning, docs, and Source-node tooling.
- [ ] Add dedicated interface introspection/visual editing.

Gate: conformance shares receiver validation/signature contracts with invocation
and decoding; parent validation passed on 2026-10-25 for script instances only.

## Phase 4 — Acceptance and release

- [x] Add end-to-end programs combining maps, HostObjects, classes, and interfaces.
- [x] Exercise new objects through listener handlers, sessions, DevTest clones,
  and MCP execution where those surfaces support the relevant values.
- [x] Run affected unit/integration packages and race tests; record commands,
  dates, results, skipped external dependencies, and known limitations.
- [x] Run static analysis and the Visual DSL production build.
- [ ] Validate saved functions/program trees and source round trips against
  legacy fixtures; provide migration instructions for intentional changes.
- [x] Document executable examples alongside the canonical language references;
  all four inputs executed through the CLI with exact expected outputs.
- [ ] Review compatibility, exposure boundaries, and resource ownership before
  accepting each feature independently.

## Explicitly deferred

- Dedicated class/interface/member visual editors and inspectors; read-only
  Source AST preservation/code generation is implemented, not visual editing.
- Debugger chain stepping/stack-trace acceptance and dedicated object inspection.
- HostObject/callable-map structural interface conformance and mutable-field requirements.
- Portable identity-preserving persistence, closure/receiver snapshots, arbitrary
  host resource serialization, and automatic class restoration from wire JSON.
- Inheritance, mixins, `super`, implicit `this`, and Java-style mandatory
  `implements` declarations.
- Member assignment operators, optional chaining, overloaded methods/operators,
  generics, and a broad new statement grammar.
- Treating runtime cloning or script classes as a security sandbox.

## Implementation and test evidence

- Runtime: [members](../../services/go-chariot/chariot/member_access.go),
  [classes](../../services/go-chariot/chariot/classes.go),
  [interfaces](../../services/go-chariot/chariot/interfaces.go),
  [graph copying](../../services/go-chariot/chariot/class_clone.go),
  [checked persistence](../../services/go-chariot/chariot/class_persistence.go),
  [wire JSON](../../services/go-chariot/chariot/value_json.go).
- Test cases: [member tests](../../services/go-chariot/chariot/member_access_test.go),
  [class tests](../../services/go-chariot/chariot/classes_test.go),
  [audit regressions](../../services/go-chariot/chariot/audit_regression_test.go).
- Tooling test cases: [codegen](../../packages/chariot-codegen/test/chariotAST.test.ts)
  and [frontend Source nodes](../../services/visual-dsl/src/test/sourceNodes.test.ts).

## Final validation record — 2026-10-25

Results below were supplied by the parent and integration agent. Commands are
listed with their working directory; this documentation-only update did not
execute them again.

### Go runtime, integration, and static analysis

Working directory: [services/go-chariot](../../services/go-chariot).

- `go test ./chariot ./internal/... ./mcp ./tests ./cmd -count=1` — fresh parent run passed.
- `go test -race ./chariot ./internal/application ./internal/listeners ./internal/handlers ./mcp` — affected-package parent race run passed.
- `go vet ./chariot ./internal/... ./mcp ./tests ./cmd` — parent static analysis passed.
- `go test -race ./internal/listeners ./internal/application ./mcp -run 'TestDevTestDotClassInterfaceAndHostEndToEnd|TestHostRESTDotClassSessionsIsolateScriptStateAndBorrowHost|TestMCPExecuteDotClassGlobalsInterfacesAndHost' -count=10` — new combined scenarios passed all ten repetitions under the race detector.

The integration scenarios combine HostObjects, maps, classes, and interfaces via
[DevTest listeners](../../services/go-chariot/internal/listeners/dot_class_integration_test.go),
[REST sessions](../../services/go-chariot/internal/application/dot_class_integration_test.go),
and [MCP execution](../../services/go-chariot/mcp/dot_class_integration_test.go).
They exercise listener handler responses, clone/session isolation, bound receivers,
borrowed host effects, and signature/conformance errors. These are runtime/API
checks, not broad listener/MCP UI or host-exposure acceptance.

### Executable examples

Working directory: repository root. Each command passed with exactly the expected
final elements in the [example result table](../../examples/objects/README.md).

- `go run ./services/go-chariot/cmd/chariotctl -f examples/objects/member-calls.ch`
- `go run ./services/go-chariot/cmd/chariotctl -f examples/objects/classes.ch`
- `go run ./services/go-chariot/cmd/chariotctl -f examples/objects/interfaces.ch`
- `go run ./services/go-chariot/cmd/chariotctl -f examples/objects/serialization.ch`

### Shared codegen and Visual DSL

- Working directory: [packages/chariot-codegen](../../packages/chariot-codegen).
  `npm run build` passed; `npx tsx --test test/*.test.ts` passed **10 tests**.
- Working directory: [services/visual-dsl](../../services/visual-dsl).
  [Source-node tests](../../services/visual-dsl/src/test/sourceNodes.test.ts)
  passed **3 tests**. Reproducible direct invocation:
  `npx tsx --test src/test/sourceNodes.test.ts`.
  `npm run build` passed the Vite production build, with bundle-size and
  Browserslist warnings.
- A passing Vite build is **not** a passing full frontend TypeScript check.
  The known full `tsc` baseline has **7 pre-existing errors**, involving
  `InputProps.list`, `logicon`, and `defaultProperties`; no clean full frontend
  typecheck is claimed.

No external-service acceptance or skipped-dependency inventory was supplied with
these results; do not infer either from the passing package runs. Broad legacy
saved-fixture/program-tree compatibility and exposure/resource-ownership review
remain open. Dedicated editors/inspectors, debugger stepping, and HostObject/map
structural conformance remain explicitly deferred.

## Newline/codegen follow-up — 2026-10-27

- [x] Remove automatic statement-end semicolons from shared AST blocks/programs,
  mixed visual/Source diagrams, generated parameter defaults, and all four object
  examples. Preserve JS/TS syntax semicolons, literal contents, and raw Source text.
- [x] Assert newline output and same-line call parentheses for named/member,
  grouped, zero-argument, and arbitrary/chained callees, including a grouped next
  statement after a bare name. Cover nested function/control-flow/declaration
  bodies and boundaries between separate Source nodes and ordinary visual nodes.
- [x] Add an opt-in real Go CLI check of generated source and all four examples;
  no Go source or language-design document changes belong to this follow-up.
- [x] Remove outdated defensive-semicolon claims from examples and object docs.

Fresh results (repository root unless otherwise noted):

- `npm --prefix packages/chariot-codegen run build` — ESM/IIFE/declarations passed.
- `CHARIOT_CODEGEN_GO_TESTS=1 node --experimental-strip-types --test packages/chariot-codegen/test/*.test.ts`
  — **19 passed**, including the Go-backed parent test and its seven subtests:
  three generated-program scenarios and four exact-output object examples.
  The test builds the current interpreter CLI into a temporary directory, executes
  generated Chariot with `-f`, and removes temporary files/binaries afterward.
- `node --experimental-strip-types --test packages/chariot-codegen/test/*.test.ts`
  — **11 passed, 1 skipped** without Go opt-in. The same test files therefore work
  without a local Go toolchain; build the shared package first. Node must support
  experimental type stripping (validated with Node 22.17.1).
- Frontend [Source-node tests](../../services/visual-dsl/src/test/sourceNodes.test.ts)
  bundled using local esbuild (`--bundle --platform=node --format=esm`) and run
  with `node --test` — **4 passed**. Temporary bundles were removed afterward.
- Working directory [services/visual-dsl](../../services/visual-dsl): `npm run -s build`
  — Vite production build passed. Existing Browserslist age and large-chunk
  warnings remain; this is not a full frontend TypeScript check.
- Go CLI build emitted a macOS duplicate `-lc++` linker warning; execution passed.

These are fresh TS/codegen/example checks, not a rerun of the historical full
Go, race, integration, or vet suites. Dedicated UI/debugger tooling, broad saved
fixture compatibility, and ownership/exposure review remain open as above.
