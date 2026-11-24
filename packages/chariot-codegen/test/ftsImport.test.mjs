import assert from 'node:assert/strict';
import { test } from 'node:test';
import { mkdtempSync, writeFileSync, readFileSync, rmSync, existsSync } from 'node:fs';
import { tmpdir } from 'node:os';
import { join, resolve, dirname } from 'node:path';
import { fileURLToPath } from 'node:url';
import { spawnSync } from 'node:child_process';
import { importFTSModel, loadFTSModel } from '../dist/index.js';

const packageRoot = fileURLToPath(new URL('../', import.meta.url));
function fixture() {
  const names = ['organization', 'person', 'address', 'phone'];
  return {
    name: 'business-core',
    descriptor: { modelName: 'business-core', rootFrameId: 'root:core', anchorPrefix: 'fn:' },
    frames: [
      { alias: 'root', key: 'frame.model', canonicalId: 'root:core' },
      ...names.map(name => ({ alias: name, key: `frame.${name}`, canonicalId: `fn:${name}` })),
      { alias: 'example', key: 'frame.person', canonicalId: 'person:example' },
    ],
    archetypes: names.map(name => ({
      id: `iar.${name}`, key: `instance.${name}.entity`, parentNeighborhoodKey: `frame.${name}`,
      propertySchema: {
        type: 'object', properties: {
          id: { type: 'string' }, title: { type: 'string' }, enabled: { type: 'boolean' },
        }, required: ['id', 'title'],
      },
      stateSchema: { type: 'object', properties: { status: { type: 'string' } }, required: null },
      defaultAttributes: { title: `New ${name}`, enabled: false }, defaultState: { status: 'active' },
    })),
    edges: [
      { key: 'edge.contains', typeName: 'contains', fromAlias: 'root', toAlias: 'organization' },
      { key: 'edge.example', typeName: 'contains', fromAlias: 'person', toAlias: 'example' },
      ...[
        ['organization', 'organization', 'parent_of'], ['organization', 'person', 'has_person'],
        ['organization', 'address', 'has_address'], ['person', 'address', 'has_address'],
        ['organization', 'phone', 'has_phone'], ['person', 'phone', 'has_phone'],
      ].map(([fromAlias, toAlias, typeName]) => ({
        key: `edge.${fromAlias}.${typeName}`, typeName, fromAlias, toAlias,
        ...(typeName === 'has_person' ? {
          'x-relational': { mysql: { mode: 'association-table', associationTable: {
            sourceArchetypeKey: 'instance.organization.entity', targetArchetypeKey: 'instance.person.entity',
            payloadColumns: { role: { sqlType: 'varchar(20)', nullable: true } },
          } } },
        } : {}),
      })),
    ],
  };
}
const clone = value => JSON.parse(JSON.stringify(value));

test('four classes, six directed schema relationships, defaults/state/metadata, no examples', () => {
  const result = importFTSModel(fixture());
  assert.deepEqual(result.classNames, ['Address', 'Organization', 'Person', 'Phone']);
  assert.equal(result.relationshipKeys.length, 6);
  assert.equal(result.excludedInstanceFrames, 1);
  assert.equal(result.excludedInstanceEdges, 1);
  assert.equal((result.source.match(/^class\(/gm) ?? []).length, 4);
  assert.match(result.source, /field\('id', 'S'\)/);
  assert.match(result.source, /field\('title', 'S', 'New person'\)/);
  assert.match(result.source, /field\('enabled', 'L', false\)/);
  assert.match(result.source, /field\('ftsState', 'M', mapValue\('status', 'active'\)\)/);
  assert.match(result.source, /'payloadColumns', mapValue\('role'/);
  assert.match(result.source, /'sourceClass', 'Organization'/);
  assert.match(result.source, /'targetClass', 'Person'/);
  assert.ok(!result.source.includes('person:example'));
  assert.ok(!result.source.includes('edge.example'));
  assert.ok(result.source.includes('edge.contains'));
});

test('output is deterministic, including shuffled object keys and definition arrays', () => {
  const input = fixture();
  const first = importFTSModel(input).source;
  assert.equal(importFTSModel(input).source, first);
  const reordered = clone(input);
  reordered.archetypes.reverse();
  reordered.edges.reverse();
  // Frame order is preserved as model hierarchy metadata.
  reordered.archetypes[0].defaultAttributes = { enabled: false, title: 'New phone' };
  assert.equal(importFTSModel(reordered).source, first);
});

test('composition resolves nested relative imports and deduplicates diamonds', () => {
  const base = fixture();
  const files = {
    '/base.json': base,
    '/sub/left.json': { imports: [{ source: '../base.json' }] },
    '/right.json': { imports: [{ source: 'base.json' }] },
    '/root.json': { name: 'composed', imports: [{ source: 'sub/left.json' }, { source: 'right.json' }] },
  };
  const model = loadFTSModel('/root.json', id => {
    if (!files[id]) throw new Error(`Missing ${id}`);
    return files[id];
  }, (parent, source) => resolve(dirname(parent), source));
  assert.equal(model.name, 'composed');
  assert.equal(model.archetypes.length, 4);
  assert.equal(importFTSModel(model).relationshipKeys.length, 6);
  files['/base.json'] = { imports: [{ source: 'root.json' }] };
  assert.throws(() => loadFTSModel('/root.json', id => files[id], (p, s) => resolve(dirname(p), s)), /import cycle/);
});

test('composition fails on conflicts, missing source and missing files', () => {
  const base = fixture();
  const changed = clone(base);
  changed.archetypes[0].name = 'Conflicting';
  const files = { root: { imports: [{ source: 'base' }, { source: 'changed' }] }, base, changed };
  assert.throws(() => loadFTSModel('root', id => files[id], (_, s) => s), /conflicts/);
  assert.throws(() => loadFTSModel('root', () => ({ imports: [{ model: 'missing' }] }), (_, s) => s), /imports.source/);
  assert.throws(() => loadFTSModel('root', () => { throw new Error('not found'); }, (_, s) => s), /not found/);
});

test('rejects malformed schemas/defaults, unsupported constraints and invalid references', () => {
  for (const [mutate, pattern] of [
    [m => m.edges[2].toAlias = 'missing', /unresolved edge endpoint/],
    [m => m.frames.push(clone(m.frames[0])), /duplicate alias/],
    [m => m.frames[1].canonicalId = 'fn:person', /duplicate canonicalId/],
    [m => m.archetypes.push(clone(m.archetypes[0])), /duplicate key/],
    [m => m.archetypes[0].key = 'instance.person.record', /collides/],
    [m => m.archetypes[0].parentNeighborhoodKey = 'missing', /unknown parent/],
    [m => m.archetypes[0].propertySchema.properties.id.pattern = '[A-Z]+', /unsupported schema keyword/],
    [m => m.archetypes[0].defaultAttributes.enabled = 'false', /expected boolean/],
    [m => m.archetypes[0].propertySchema.required.push('missing'), /declared properties/],
    [m => m.archetypes[0].validationRules = { custom: true }, /validationRules/],
    [m => m.archetypes[0].propertySchema.properties.ftsState = { type: 'string' }, /reserved/],
    [m => m.imports = [{ source: 'base.json' }], /compose imports/],
    [m => m.archetypes[0].propertySchema.properties.id.type = 'date', /unsupported/],
    [m => m.archetypes[0].propertySchema.properties.id.type = [], /unsupported/],
    [m => m.archetypes[0].propertySchema.properties.id.type = 'constructor', /unsupported/],
    [m => m.archetypes[0].propertySchema.properties.id.minLength = -1, /invalid numeric/],
    [m => m.archetypes[0].propertySchema.properties.id.enum = [], /nonempty primitive/],
  ]) {
    const model = fixture();
    mutate(model);
    assert.throws(() => importFTSModel(model), pattern);
  }
});

test('checked-in validation source is reproducible from its modelseed', () => {
  const model = JSON.parse(readFileSync(new URL('./fixtures/fts-validation.modelseed.json', import.meta.url), 'utf8'));
  const expected = readFileSync(new URL('./fixtures/fts-validation.ch', import.meta.url), 'utf8');
  assert.equal(importFTSModel(model).source, expected);
});

test('ambiguous neighborhood mappings require explicit relational endpoints', () => {
  const m = fixture();
  const second = clone(m.archetypes[0]);
  second.key = 'instance.organization.other';
  m.archetypes.push(second);
  assert.throws(() => importFTSModel(m), /ambiguous or missing archetype/);
  for (const e of m.edges) {
    if (e.fromAlias !== 'root' && e.toAlias !== 'example') {
      e['x-relational'] = { mysql: { joinTable: {
        sourceArchetypeKey: `instance.${e.fromAlias}.entity`,
        targetArchetypeKey: `instance.${e.toAlias}.entity`,
      } } };
    }
  }
  assert.equal(importFTSModel(m).classNames.length, 5);
});

test('nullable/nested schemas, numeric bounds, enums and escaped data generate safe AST', () => {
  const model = fixture();
  Object.assign(model.archetypes[0].propertySchema.properties, {
    score: { type: ['integer', 'null'], minimum: 0, maximum: 10 },
    tags: { type: 'array', minItems: 1, maxItems: 2, items: { type: 'string', minLength: 1, maxLength: 4 } },
    options: { type: 'object', additionalProperties: false, properties: { mode: { type: 'string', enum: ['a', 'b'] } } },
    note: { type: 'string' },
  });
  model.archetypes[0].defaultAttributes.note = "'\\\n雪";
  const result = importFTSModel(model);
  assert.ok(result.source.includes('floor('));
  assert.ok(result.source.includes("split(item"));
  assert.ok(result.source.includes("\\'\\\\\\n雪"));
  assert.ok(result.source.includes('smallerEq('));
  model.archetypes[0].defaultAttributes.score = 1.5;
  assert.throws(() => importFTSModel(model), /expected integer/);
});

test('CLI composes files, persists deterministic output and fails without overwriting', () => {
  const dir = mkdtempSync(join(tmpdir(), 'chariot-fts-'));
  try {
    const base = join(dir, 'base.json'), root = join(dir, 'root.json'), output = join(dir, 'model.ch');
    writeFileSync(base, JSON.stringify(fixture()));
    writeFileSync(root, JSON.stringify({ imports: [{ source: 'base.json' }] }));
    const cli = (...args) => spawnSync(process.execPath, ['scripts/import-fts.mjs', ...args], { cwd: packageRoot, encoding: 'utf8' });
    const first = cli(root, output);
    assert.equal(first.status, 0, first.stderr);
    assert.match(first.stdout, /4 classes, 6 relationships/);
    const contents = readFileSync(output, 'utf8');
    assert.equal(cli(root, output).status, 1);
    assert.equal(readFileSync(output, 'utf8'), contents);
    assert.equal(cli(root, output, '--force').status, 0);
    assert.equal(readFileSync(output, 'utf8'), contents);
    assert.equal(cli(root, base, '--force').status, 1);
    assert.equal(JSON.parse(readFileSync(base)).name, 'business-core');
    writeFileSync(root, '{invalid');
    assert.equal(cli(root, join(dir, 'bad.ch')).status, 1);
    assert.ok(!existsSync(join(dir, 'bad.ch')));
  } finally {
    rmSync(dir, { recursive: true, force: true });
  }
});
