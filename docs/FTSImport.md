# FTS modelseed import

The first milestone imports a local Frames Theory System (FTS) modelseed into
one executable Chariot `.ch` file. No FTS service, database, or network access is
required. This is a one-way schema import, not instance migration or runtime
graph persistence.

## Generate a model

From the monorepo root, with the shared package's development dependencies
available:

```sh
npm --prefix packages/chariot-codegen run build
node packages/chariot-codegen/scripts/import-fts.mjs \
  ../../m-theory-io/fts/models/business-core.modelseed.json \
  examples/fts/business-core.ch --force
```

The destination directory must already exist. Omit `--force` to refuse to replace
an existing file. Output is written atomically after validation; an input model,
including an imported input, cannot be used as the output path. Errors go to
stderr and return a nonzero exit code.

Alternatively, from `packages/chariot-codegen`, use:

```sh
npm run import:fts -- /absolute/path/to/model.modelseed.json /absolute/path/to/model.ch
```

Relative CLI arguments are relative to the process working directory. npm runs
package scripts in the package directory, including when invoked with `--prefix`.
Import `source` paths are relative to their containing modelseed, not the CLI's
working directory.

Imports follow FTS's composition rules: depth-first, imports before the local
model, frames merged by alias and edges/archetypes by key. Identical definitions
are deduplicated; conflicting definitions, missing files, and import cycles fail.
Local descriptor/catalog values override imported values, metadata is merged, and
source notes are appended. Import aliases do not rename definitions, matching the
current FTS composer. Files are intended to be trusted local model documents;
imports can reference files outside the root model directory.

## Generated API

Load/execute the generated source before calling its definitions. The
[business-core example](../examples/fts/business-core.ch) defines `Organization`,
`Person`, `Address`, and `Phone`, plus two functions per class:

```chariot
setq(org, createOrganization(
    mapValue("organization_id", "org-1", "legalName", "Acme"),
    mapValue()
))
org.title
org.ftsState.status

validatePerson(
    mapValue("person_id", "person-1", "dob", "2000-01-01"),
    mapValue()
)
```

Both functions require exactly two maps: attributes and state. An empty map
requests the available defaults; it does not invent missing required values.
`validate<Class>` applies defaults, checks the result, and returns `true` or
raises a runtime error. `create<Class>` validates and constructs the instance.

Each class declares its business properties and two reserved fields:

- `ftsState`: a map, validated against the separate FTS state schema.
- `ftsExtraAttributes`: extra attributes permitted by the property schema.

Additional properties are allowed unless the source schema explicitly says
`additionalProperties: false`. Extra attributes are retained in
`ftsExtraAttributes`, not discarded or added to the frozen class schema. Extra
state entries stay in `ftsState`. A source property with either reserved field
name is rejected.

Required fields without defaults have no class-field default. Optional fields
without defaults use a null storage sentinel with a `V` contract: omission is
allowed, but an explicitly supplied null still fails validation unless the
schema allows null. Defaults are evaluated afresh for each checked construction.
An optional omitted value and an explicit nullable null are not distinguishable
from the resulting field value alone.

**Use the checked functions at input boundaries.** Direct `new(Class, ...)`,
`setProp`, and later mutations of maps/arrays do not invoke the generated schema
validators. Class contracts enforce only what Chariot supports natively.
To validate edited data, pass the desired attribute/state maps to
`validate<Class>` again. This import does not add immutable objects or
always-on schema enforcement.

## Mapping and model metadata

Business classes come from `archetypes`, not every entry in `frames`.
Class names are derived from archetype keys: remove a leading `instance.`,
omit `entity`/`record` path segments, and convert the remaining segments to
PascalCase. Invalid or colliding names fail instead of receiving unstable suffixes.
`create<Class>` and `validate<Class>` names are also checked for collisions.

`descriptor.rootFrameId` identifies the root, and `descriptor.anchorPrefix`
identifies canonical neighborhood anchors. Other frames and their edges are
instance fixtures and are excluded. Archetypes resolve their
`parentNeighborhoodKey` to a canonical neighborhood. Business edges resolve
frame aliases to their neighborhood's archetype; an ambiguous endpoint fails.
Association/join-table `sourceArchetypeKey` and `targetArchetypeKey` can
disambiguate those endpoints.

The `ftsModel` map contains:

| Member | Contents |
| --- | --- |
| `name`, `descriptor`, `catalog`, `metadata`, `sourceNotes` | Source model context |
| `classes` | Archetype definitions, original IDs/keys, schemas, defaults, and generated `className` |
| `frames` | Root and canonical neighborhood frames only |
| `hierarchy` | Schema edges involving the model root |
| `relationships` | Directed business edges, augmented with source/target class names and archetype keys |
| `excludedInstanceFrames`, `excludedInstanceEdges` | Excluded fixture counts |
| `x-relational` | Model-level relational projection, when present |

These are schema data, not Visual DSL execution-flow edges or live instance
connections. No cardinality is inferred. Archetype and edge `x-relational`
metadata is preserved, including association payload columns, SQL types,
nullability, and indexes. SQL projection metadata is not interpreted as
additional domain constraints or executable database operations.

For the supplied business-core model the output has four classes, six directed
business relationships, five structural frames, and four root hierarchy edges.
Seven example frames and thirteen example-related edges are excluded.

## Supported validation and limitations

Supported schema types are `string`, `boolean`, `number`, `integer`, `null`,
`array`, and `object`, including type unions. Supported constraints are:

- Object `properties`, `required`, and boolean `additionalProperties`.
- Recursive array `items`, `minItems`, and `maxItems`.
- Numeric `minimum` and `maximum`, plus integer checks.
- String `minLength` and `maxLength`, counted as Unicode code points.
- Primitive-value `enum`.

Chariot numbers use floating-point representation; this is not an exact-decimal
or arbitrary-precision integer importer. Date-looking strings remain strings.
Only explicit `defaultAttributes` and `defaultState` supply construction
defaults; a JSON Schema `default` annotation does not fill a value.

Schema annotations (`title`, `description`, `$comment`, `$schema`, `$id`,
`default`, `examples`) are retained without execution. Other schema keywords
(including `$ref`, `pattern`, and `format`), schema-valued
`additionalProperties`, nonprimitive enums, and nonempty FTS `validationRules`
are rejected, not silently weakened. Defaults are validated at import time;
required values not present in defaults may still be supplied at construction.
Malformed references, duplicate identities, and ambiguous relationships fail.

The generated source uses ordinary Chariot classes, functions, maps, and the
`error(message)` builtin from `RegisterFlow`; there is no new language syntax.
A restricted embedding must register values/classes, arrays, comparisons,
math, polymorphic dispatch (including `split`), and flow functions. Update/rebuild
the Chariot runtime before executing generated files on an older deployment.

The same conversion is available without filesystem dependencies through
`loadFTSModel(root, read, resolve)` and `importFTSModel(model)` exported by
`chariot-codegen`. Conversion returns source, AST, class names, relationship
keys, and exclusion counts. Pass composed input to `importFTSModel`; unresolved
imports are rejected.

## Validation commands

```sh
npm --prefix packages/chariot-codegen run test:fts
go test ./services/go-chariot/chariot -run '^TestFTS' -count=1
```

The Go tests execute the checked-in business-core output and a generated
validation fixture. They check construction, defaults, state separation,
relationship payload metadata, invalid inputs, and error propagation. The Node
tests cover composition, deterministic output, diagnostics, and CLI overwrite
protection.

To regenerate the validation fixture after intentional generator changes:

```sh
node packages/chariot-codegen/scripts/import-fts.mjs \
  packages/chariot-codegen/test/fixtures/fts-validation.modelseed.json \
  packages/chariot-codegen/test/fixtures/fts-validation.ch --force
```
