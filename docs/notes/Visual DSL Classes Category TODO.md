# Visual DSL CLASSES Category — Implementation TODO

Status: Local implementation and harness acceptance complete; deployed-stack acceptance pending.
Date: 2026-10-27

## Objective and scope

Add a **CLASSES** palette category with logicons and property dialogs for all ten
class/interface constructs. Users should be able to author, save, reopen, generate,
and execute a complete object-oriented example without editing diagram JSON.

Use the existing runtime contracts in [Classes and Interfaces](../ClassesAndInterfaces.md)
and preserve [newline-oriented Chariot syntax](../ChariotLanguageDesign.md#newlines-and-expression-continuation).
This feature adds visual authoring, not a second class model or new runtime APIs.

General dot/member expression editors are a separate follow-up. Existing read-only
Source nodes remain supported; do not silently convert them into lossy editable
nodes. Expressions entered in dialogs may use supported dot notation.

## Phase 1 — Define visual properties and nesting contracts

- [x] Define typed property models, defaults, and validation for the ten logicons.
- [x] Distinguish declaration identifiers, member names, expression inputs, and
  quoted contract strings. Never auto-quote class/interface reference expressions.
- [x] Define ordered declaration bodies separately from function arguments and
  ordinary statement nesting.
- [x] Allow only `field` children directly inside a class body and `requires`
  children directly inside an interface body; reject misplaced declarations.
- [x] Keep `method` registration outside class bodies, matching runtime syntax.
- [x] Define one consistent method-function editor using the existing `func`
  parameter/body model. Decide how expression slots and nested nodes interact;
  do not generate duplicate arguments from both representations.
- [x] Preserve the distinction between an omitted optional input and an explicit
  expression such as `null`, `""`, `0`, or an empty map.
- [x] Reuse canonical contract codes: `V N S L D A M F O J X T P E`.

### Required logicons and dialogs

| Construct | Required property dialog |
| --- | --- |
| `class(Name) { ... }` | Definition identifier and ordered field body. |
| `field(name, type[, default])` | Member name, contract selector, default-present toggle, default expression; omitted default makes the field required. |
| `method(Class, name, function)` | Class expression, member name, function editor/expression; require explicit `self` as first parameter for a declared function. |
| `new(Class[, attributes])` | Class expression and optional attributes-map expression; accept both `map` and `mapValue`. |
| `interface(Name) { ... }` | Definition identifier and ordered method requirements. |
| `requires(name, signature)` | Member name and required signature expression/editor. |
| `signature(array(types...), returnType)` | Ordered parameter-contract rows, zero-parameter support, and return contract. |
| `withSignature(function, signature)` | Function and signature inputs; preserve the annotation's copy semantics. |
| `satisfies(value, Interface)` | Value and interface expressions; boolean result. |
| `assertSatisfies(value, Interface)` | Value and interface expressions; returns the original instance or errors. |

- [x] Explain receiver signatures in the dialogs: method function annotations
  include leading `"V"` for `self`; interface requirements exclude the receiver.
- [x] Validate what is statically knowable; leave dynamic expression/type checks
  to the runtime rather than claiming the UI can prove conformance.

Gate: **met locally**. The [shared schema](../../packages/chariot-codegen/src/classSchema.ts)
defines typed properties, fresh defaults, active controls and graph validation.
Expression fields are nonblank raw Chariot, not a parser/typechecker. Method uses
either parameter/default-expression rows plus a body, or a function expression;
inactive values persist but are not emitted. Class accepts only Field children;
Interface accepts only Requires children. Other CLASSES nodes use properties,
not nested argument nodes. Supported expression constructs can themselves be
nested under Declare/assignment nodes. See the [user guide](../../services/visual-dsl/CLASSES.md).

## Phase 2 — Palette and property dialogs

- [x] Add category key `classes` and display label **CLASSES** to the palette.
- [x] Choose consistent icons, descriptions, labels, and category placement.
- [x] Add all ten logicon definitions with meaningful initial properties.
- [x] Implement a property dialog for every entry, sharing small input controls
  where useful without hiding distinct function contracts.
- [x] Wire dialog opening, validation, Save, Cancel, and existing deletion behavior.
- [x] Initialize the same properties in every insertion path: normal flow,
  random placement, and nested insertion. Consolidate defaults where practical.
- [x] Preserve expression strings, parameter order, explicit empty values, and
  optional-input toggles when reopening dialogs.
- [x] Provide clear inline diagnostics for invalid identifiers, contracts,
  declaration placement, and missing inputs; retain unsaved form edits on error.
- [x] Provide Escape, Tab/Shift+Tab focus trapping/restoration and a large method
  body textarea; exercised in Chrome. This is not a full accessibility audit.

Primary implementation locations:

- [logicon definitions](../../services/visual-dsl/src/data/logicons.ts)
- [palette categories](../../services/visual-dsl/src/components/LogiconPalette.tsx)
- [dialog and insertion wiring](../../services/visual-dsl/src/pages/Canvas.tsx)
- [existing function dialog](../../services/visual-dsl/src/components/dialogs/FunctionNodeProperties.tsx)
- [nesting state](../../services/visual-dsl/src/contexts/NestingContext.tsx)
- [generic logicon rendering](../../services/visual-dsl/src/components/LogiconNode.tsx)

Gate: **met in the local Chrome harness** for all ten dialogs. Evidence:
[frontend tests](../../services/visual-dsl/src/test/classAuthoring.test.tsx) and
[Chrome authoring tests](../../services/visual-dsl/src/test/classes.browser.test.cjs).
Select the declaration, click **Create Nesting**, add its members, then click
**Exit Nesting** before adding Method. Root Field/Requires insertion is blocked;
imported invalid members remain visible with diagnostics and can be repaired by
dragging the matching declaration's right connector to the member's left connector.

## Phase 3 — Canonical code generation

- [x] Add explicit shared-generator handling for all ten constructs, including
  nested argument/expression paths; do not rely on lowercasing display labels.
- [x] Preserve exact names such as `withSignature` and `assertSatisfies`.
- [x] Emit class/interface declaration blocks and children by nesting relation order.
- [x] Emit method functions with the approved `func(self, ...) { ... }` syntax.
- [x] Emit signature parameter arrays and quoted contract codes correctly.
- [x] Omit optional arguments when disabled; preserve enabled explicit null/empty
  values without substituting defaults.
- [x] Quote/escape member names correctly and keep definition identifiers and
  class/interface expressions unquoted.
- [x] Validate declaration placement and invalid property structures during
  generation as well as in the UI; report the responsible node/property.
- [x] Use newline-based statements and keep call parentheses attached to callees;
  never introduce mandatory statement-end semicolons.
- [x] Preserve complete Source ASTs and existing mixed-diagram behavior. Keep
  unsupported Source nesting explicit rather than dropping or duplicating code.
- [x] Retain tested legacy generation and embedded diagram/Source metadata round trips.
- [ ] Complete broad legacy-diagram acceptance beyond the checked regression cases.

Use the [shared generator](../../packages/chariot-codegen/src/chariotCodeGenerator.ts)
and [AST/source support](../../packages/chariot-codegen/src/chariotAST.ts).
The frontend [codegen module](../../services/visual-dsl/src/utils/codegen.ts)
re-exports the shared implementation; do not create a parallel generator.

Gate: **met for tested programs**, not an exhaustive compatibility proof. The
[shared classes tests](../../packages/chariot-codegen/test/classes.test.ts) exercise
all ten constructs, ordered bodies, invalid graphs, omission/explicit defaults,
receiver annotations, assertion identity/failure and annotation copy semantics.
The opt-in CLI tests build the current Go interpreter and execute generated source.

## Phase 4 — Persistence, regression tests, and workflow acceptance

- [x] Round-trip properties/nesting through JSON serialization/import, generated
  embedded metadata and automatic drafts; browser-test reload and iframe app switching.
- [x] Exercise named **Save** followed by reload in Chrome. Reload restores the
  working draft; this does not prove loading the named saved entry.
- [ ] Exercise named **Load** selection and server-backed diagram save/load with
  all ten constructs; verify exported/downloaded JSON through the browser file path.
- [x] Retain tested unrelated legacy nodes and Source metadata without migration.
- [x] Add tests for each dialog's defaults, Save/Cancel, validation, and reopen.
- [x] Exercise normal, Shift/random and nested insertion plus ordered children;
  defaults share one constructor (not an exhaustive path-by-label browser matrix).
- [x] Test absent/default/null fields, escaped member names, zero-argument
  signatures, receiver annotations, and malformed/incomplete diagrams.
- [x] Add shared-codegen tests for all ten constructs and supported nested compositions.
- [x] Execute generated programs through the Go CLI: class construction, mutable
  methods, bound extraction, positive/negative interface checks, and assertions.
- [x] Test mixed visual/Source diagrams without losing AST metadata or raw source.
- [x] Browser-test palette → declarations/methods/interface → generate/copy →
  Charioteer Run → editable draft restore, using the local proxy/CLI harness below.
- [x] Run frontend tests, shared codegen tests/build, Visual DSL production build,
  and relevant type/static checks. Record any baseline failures separately.
- [x] Publish a complete example diagram and update user documentation; explain
  current runtime limits such as script-instance-only interface conformance.
- [ ] Validate the deployed stack's real authentication, REST backend session
  lifecycle/isolation, streaming execution and server diagram persistence.

Gate: **local iterative authoring accepted; full deployment gate still open**.
The palette-only browser case needs no JSON edits. The all-ten case imports the
shared fixture through Paste JSON, then edits Method through its real dialog.
Neither case proves named Load or server diagram storage. No deployment was
performed for this documentation/example pass.

## Validation record — 2026-10-27

Reproduced against the current local source with Node v22.17.1, Go 1.25.6
(darwin/arm64), and headless installed Google Chrome:

| Check | Observed result |
| --- | --- |
| Shared build (ESM/IIFE and TypeScript declarations) | Passed |
| Shared tests with Go CLI opt-in | **38 passed**, no skips (includes nested subtests) |
| Frontend tests | **27 passed**, no skips |
| Chrome authoring | **12 passed**, no skips |
| Cross-app Chrome harness | **2 passed**, no skips |
| Visual DSL Vite production build | Passed; stale Browserslist data and large-chunk warnings |
| Frontend full typecheck | Failed with **2 baseline TS2322 errors**, both `InputProps.list` |

The remaining diagnostics are in
[GetTransformNodeProperties](../../services/visual-dsl/src/components/dialogs/GetTransformNodeProperties.tsx#L97)
and [RegisterTransformNodeProperties](../../services/visual-dsl/src/components/dialogs/RegisterTransformNodeProperties.tsx#L191).
The production build is not a clean full typecheck. No source fix is included here.
Go CLI builds also emitted a duplicate `-lc++` linker warning without failing.

### Reproducible commands

Run the following from the repository root with installed package dependencies,
a Go toolchain and a Node version supporting `--experimental-strip-types`:

```sh
npm --prefix packages/chariot-codegen run build
CHARIOT_CODEGEN_GO_TESTS=1 node --experimental-strip-types --test packages/chariot-codegen/test/*.test.ts
cd services/visual-dsl
npx --yes tsx --test src/test/*.test.ts src/test/*.test.tsx
npm run build
npx tsc --noEmit
```

The last command is expected to report the two baseline errors above. Without
`CHARIOT_CODEGEN_GO_TESTS=1`, Go-backed tests skip and do not establish CLI acceptance.

Browser prerequisites: a source Vite server on `http://localhost:5186`, Playwright
and a Chrome executable. The verified environment already had that server running.
To start an equivalent source server in a separate terminal, from the repository
root (HTTP requires no auto-detected local certificates):

```sh
npm --prefix services/visual-dsl run dev -- --host 127.0.0.1 --port 5186 --strictPort
```

From the repository root in another terminal, reproduce the browser runs with the
actual local dependency/browser paths used on this machine:

```sh
cd services/visual-dsl
export VISUAL_DSL_TEST_URL=http://localhost:5186
export VISUAL_DSL_PLAYWRIGHT=/tmp/charioteer-browser-tests/node_modules/playwright
export VISUAL_DSL_CHROME='/Applications/Google Chrome.app/Contents/MacOS/Google Chrome'
node --test src/test/classes.browser.test.cjs
VISUAL_DSL_ROUNDTRIP=1 node --test src/test/classes.roundtrip.browser.test.cjs
```

Use paths to your installed Playwright/Chrome if different; these are environment
prerequisites, not repository dependencies installed by this pass. The authoring
suite skips without `VISUAL_DSL_TEST_URL`; the cross-app suite also requires
`VISUAL_DSL_ROUNDTRIP=1`. Default Playwright resolution/browser may be used if
the corresponding path overrides are omitted. Skips are not passing acceptance.

### What the cross-app evidence establishes

The [cross-app harness](../../services/visual-dsl/src/test/classes.roundtrip.browser.test.cjs)
builds isolated current Charioteer and Go CLI binaries, copies Charioteer templates
to a temporary directory, serves the unchanged landing page on a temporary
same-origin proxy, and uses the real local frontend, clipboard, Monaco paste
action, Charioteer HTTP proxy and Run/output UI. Its backend execution envelope
runs the exact submitted source through the real Go CLI; interpreter execution
is **not** stubbed. Temporary processes/files are cleaned up by the harness.

- Palette-authored example: `true`, then return to editable Field properties;
  Charioteer's pasted program also survives iframe switching.
- All-ten fixture: `[42,42,true,false,7,0]`.
- After editing Method `value` Body to `return(add(self.count, 1))`, refreshing,
  copying and running again: `[42,43,true,false,7,0]`.
- It compares nodes/edges/nesting/name after iframe recreation and checks the
  exact submitted program against CLI output, not unrelated background executions.

**Limits:** login identity/token and session/files/runtime metadata are synthetic;
the observed metadata requests were `/api/etl/transforms`, `/api/files`, and
`/api/session/profile`. The HTTP execution envelope is supplied by the harness,
not the deployed Go REST backend. Streaming is explicitly disabled. It does not
validate deployed authentication, real backend sessions, streaming, server saves,
container routing or a deployed stack. Frontend-only authoring tests mock API
responses as well. Those limitations keep the deployment gate open.

### Documentation and example delivery

- [CLASSES user guide](../../services/visual-dsl/CLASSES.md): actual dialogs,
  nesting controls, receiver contracts, persistence boundaries and workflow.
- [Editable Counter diagram](../../examples/objects/classes-diagram.json): exact
  copy of the [shared fixture](../../packages/chariot-codegen/test/fixtures/classes-complete.json),
  retaining its diagram name and stable node IDs; the public filename is different.
- [Object example index](../../examples/objects/README.md): import/run instructions
  and expected values. Older dated object-language records describe their own
  earlier scope; this record covers dedicated CLASSES authoring.

## Follow-up, outside this category update

- Dedicated member-read, bound-member, and expression-call visual editors.
- Class/interface debugger inspectors and step-through support.
- Runtime interface conformance for HostObjects or callable maps.
- Inheritance, generics, assignment operators, or other new language semantics.

Completion checkboxes refer only to the evidence and scope recorded above; open
items are not waived by the local harness results.
