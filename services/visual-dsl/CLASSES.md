# CLASSES: visual classes and interfaces

The **CLASSES** palette contains ten editable constructs backed by the
[shared schema](../../packages/chariot-codegen/src/classSchema.ts) and canonical
[generator](../../packages/chariot-codegen/src/chariotCodeGenerator.ts). This is
visual authoring for the existing [Chariot object model](../../docs/ClassesAndInterfaces.md),
not a second runtime class system.

## Start with the complete diagram

1. Open Visual DSL and choose **Paste JSON**. Paste the contents of the
   [Counter example](../../examples/objects/classes-diagram.json), then click
   **Load Diagram**. Import replaces the working diagram; export important work first.
2. Use **fit view** to see the graph. Right-click a node and choose
   **Properties...**. The example uses all ten CLASSES entries, plus Declare and
   Set Q nodes. It is an exact copy of the
   [shared test fixture](../../packages/chariot-codegen/test/fixtures/classes-complete.json),
   not a serialized live instance.
3. Open **Generate .ch Code**, then click **Generate Code** in that panel.
   Inspect the generated source and use **Copy**.
4. On the same-origin landing page, switch to **Charioteer**, paste into Monaco
   and click **Run**. Use a fresh execution scope: declarations cannot be
   redeclared in an already populated scope. For the validated local harness path,
   streaming is off. A working deployed connection/authentication is a separate
   prerequisite; see the validation limits below.
5. The final elements are `[42,42,true,false,7,0]`: extracted increment result,
   checked instance's value, positive and negative interface checks, a `mapValue`
   instance's count, and a default instance's count.
6. Switch back to Visual DSL. Open the Counter Method whose **Member name** is
   `value` (node `value-method`), change **Body** from `return(self.count)` to
   `return(add(self.count, 1))`, and choose **Save Properties**. Generate, copy
   and run in a fresh scope again: `[42,43,true,false,7,0]`.

The example builds `Counter` with numeric `count` defaulting to `0`, registers
`increment` and `value`, declares `Counting` and an intentionally unsatisfied
`Missing` interface, constructs instances using both map constructors, extracts
the bound `counter.increment`, mutates by `2`, and checks conformance. The checked
instance is the original mutable instance, not a copy.

## Author declarations with the actual nesting controls

Expand **CLASSES** and click **Class**. Select its canvas node, click
**Create Nesting**, then click **Field** for each field in the intended order.
Click **Exit Nesting** before adding Method or another top-level construct.
For Interface, repeat with **Requires** children instead.

- Class accepts **only direct Field children**; Interface accepts **only direct
  Requires children**. Members are emitted by nesting relation order, not merely
  by positions or flow-edge order. Methods are registered **outside** declarations.
- Field/Requires cannot be inserted at the root through the palette. An imported
  misplaced member is retained with a diagnostic, not silently deleted. Repair
  it by dragging the matching declaration's right connector to its left connector.
  A wrong parent is rejected without changing the graph.
- Other CLASSES nodes take their inputs through properties, not nested argument
  nodes. New, Signature, WithSignature, Satisfies and AssertSatisfies can be
  expression children of supported Declare/assignment nodes. Declare requires
  exactly one expression child. Arbitrary nesting under Array, Function or Source
  is not supported; generation reports unsupported/ignored hierarchies.
- Imported Source nodes preserve AST/source metadata and remain read-only. They
  are not automatically converted to these editable class nodes.

## Dialog reference

All dialogs have **Save Properties**, **Cancel** and **Delete**. Escape cancels;
Tab/Shift+Tab stay within the dialog and focus returns to the node. Invalid Save
retains edits and shows diagnostics. Hidden inputs retain their values when
toggles/modes change, but inactive expressions are not generated.

| Entry | Actual controls and generated meaning |
| --- | --- |
| Class | **Declaration identifier** (initially `Example`); emits `class(Name) { ... }` with ordered Field children. |
| Field | **Member name**, **Contract**, **Default present**; enabling the toggle reveals **Default expression**. Toggle off emits no default and makes the field required, even for `V`. |
| Method | **Class expression**, **Member name**, **Function mode**, optional **Include method signature**. Editor mode provides parameter/default-expression rows and **Body**; expression mode provides **Function expression** instead. The signature toggle reveals **Signature expression** and generates a `withSignature` wrapper. |
| New | **Class expression**, **Attributes present**; enabling reveals **Attributes expression**. Off emits `new(Class)`; both `map(...)` and `mapValue(...)` are accepted attribute expressions by the CLI runtime. |
| Interface | **Declaration identifier** (initially `ExampleInterface`); emits an ordered Requires body. |
| Requires | **Member name** and **Signature expression**; signature excludes the receiver. |
| Signature | Ordered **Parameter contracts** rows with **Add Contract**/**Remove** and **Return contract**. No rows generates `signature(array(), returnCode)`. |
| WithSignature | **Function expression**, **Signature expression**; returns an annotated copy without mutating the original function. |
| Satisfies | **Value expression**, **Interface expression**; returns boolean conformance. |
| AssertSatisfies | Same expression inputs; returns the original instance on success or errors on failure. |

### Names, expressions and optional values

Declaration identifiers are unquoted identifiers matching
`[A-Za-z_][A-Za-z0-9_]*`. Member names are text that the generator quotes/escapes;
do not add surrounding quotes yourself. **Class expression**, **Interface
expression**, function/signature expressions, defaults and attributes are raw
Chariot. Enter `Counter` or `registry.current`, not a quoted class-name string.

**Default present** off is different from on with `null`, `""`, `0` or `map()`.
Those are explicit expressions, not missing values. An enabled blank expression
is invalid. The same omission rule applies to **Attributes present** and
**Include method signature**. A field defaults to required (`hasDefault: false`),
even though its inactive default-expression control retains `null`.

Contracts are selected from `V N S L D A M F O J X T P E` (case-sensitive).
Signature row order is argument order. See the
[runtime contract rules](../../docs/ClassesAndInterfaces.md#function-contracts)
for exact meanings and restrictions.

### Method receiver and signature example

In **Parameter / body editor** mode, the first row must be `self` with no default
value. Use **Add Parameter** for further names and optional default expressions;
the editor uses the same parameter/body generation as Function. Put Chariot
statements in **Body**, for example `return(self.count)`. For mutation use
`setProp(self, "count", add(self.count, amount))`, not member assignment syntax.

In **Function expression** mode, supply a function-valued expression such as
`incrementFunction` or `func(self) { return(self.count) }`. The inactive parameter
rows/body are retained but not emitted as another argument. The runtime validates
the supplied function; the UI cannot infer a dynamic function's signature.

For a numeric `increment(self, amount)` method, its annotation includes the
receiver: `signature(array("V", "N"), "N")`. The matching Requires signature
omits it: `signature(array("N"), "N")`. A no-public-argument `value(self)` method
uses `signature(array("V"), "N")`, whereas its requirement uses
`signature(array(), "N")`. An unannotated method does not satisfy a typed
requirement merely because its body appears to return the right type.

## Save, restore and validation boundaries

**Save** stores a named browser-local diagram; **Load** can select a saved entry.
**Export** is separate and follows the configured local-file/server storage mode.
Automatic [browser drafts](DRAFTS.md) restore committed edits, including nesting,
on reload and landing-page iframe switches without requiring named Save. Drafts
are browser/origin-local, not account-isolated, encrypted backups or cross-origin
sync. New/import replace the working copy. Keep exported backups; do not put
secrets in literals.

UI and generator validation catch identifier/contract/property errors, blank
active expressions, missing/wrong nesting, cycles and duplicate declaration
members. They do **not** parse or typecheck arbitrary expression text or prove
dynamic conformance. Runtime checks remain authoritative. Conformance currently
supports script instances/methods only, not HostObjects or callable maps; there
is no inheritance, `implements` clause or general member-expression visual editor.
Generated statements use newlines; call parentheses stay attached to their callee.

**Verified 2026-10-27:** shared TypeScript/declaration build, 38 shared tests
including real Go CLI execution, 27 frontend tests, 12 Chrome authoring tests,
2 cross-app Chrome harness cases and the production build passed. Full frontend
typechecking still has two baseline transform-dialog `InputProps.list` errors.

The cross-app harness uses a real local frontend, unchanged landing iframe,
current Charioteer proxy, clipboard, Monaco and Go CLI. Its HTTP execution
envelope and login/session/files/runtime metadata are synthetic; streaming is
disabled. Named Save/reload is exercised, but reload uses the working draft and
does not prove named Load. Server diagram persistence and deployed backend
authentication, streaming and real sessions remain **pending**. No deployment
was performed. Exact commands, test links and gate status are in the
[dated validation record](../../docs/notes/Visual%20DSL%20Classes%20Category%20TODO.md#validation-record--2026-10-27).
