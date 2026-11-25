import assert from 'node:assert/strict';
import { execFileSync, spawnSync } from 'node:child_process';
import { mkdtempSync, readFileSync, rmSync, writeFileSync } from 'node:fs';
import { tmpdir } from 'node:os';
import { join } from 'node:path';
import { fileURLToPath } from 'node:url';
import { test } from 'node:test';
import {
  CLASS_NODE_LABELS, CLASS_NODE_SCHEMA, CLASS_CONTRACT_CODES, getClassNodeDefaults,
  validateClassNodeProperties, validateClassDiagram, ChariotCodeGenerator,
  generateChariotCodeFromDiagram, extractDiagramFromCode, deserializeChariotAST,
} from '../dist/index.js';
import type { ClassNodeLabel, ClassNodePropertiesMap, VisualDSLNode, VisualDSLDiagram } from '../dist/index.js';

const node = (label: string, properties: Record<string, unknown>, id = label): VisualDSLNode =>
  ({ id, type: 'logicon', position: { x: 0, y: 0 }, data: { label, icon: '', category: 'classes', properties } });
const cls = <L extends ClassNodeLabel>(label: L, overrides: Partial<ClassNodePropertiesMap[L]> = {}, id: string = label) =>
  node(label, { ...getClassNodeDefaults(label), ...overrides }, id);
const diagram = (nodes: VisualDSLNode[], nestingRelations: VisualDSLDiagram['nestingRelations'] = []): VisualDSLDiagram =>
  ({ name: 'Classes', nodes, edges: [], nestingRelations });
const rel = (parentId: string, childId: string, order = 0) => ({ parentId, childId, order });
const generate = (d: VisualDSLDiagram, embedSource = false) =>
  generateChariotCodeFromDiagram(JSON.stringify(d), { embedSource });
const fixture = (): VisualDSLDiagram => JSON.parse(readFileSync(new URL('./fixtures/classes-complete.json', import.meta.url), 'utf8'));

test('all ten exported defaults, contracts and dialog metadata are usable and independent', () => {
  assert.deepEqual(CLASS_NODE_LABELS, ['Class', 'Field', 'Method', 'New', 'Interface', 'Requires',
    'Signature', 'WithSignature', 'Satisfies', 'AssertSatisfies']);
  assert.deepEqual(CLASS_CONTRACT_CODES, 'V N S L D A M F O J X T P E'.split(' '));
  for (const label of CLASS_NODE_LABELS) {
    assert.deepEqual(validateClassNodeProperties(label, getClassNodeDefaults(label)), [], label);
    assert.equal(CLASS_NODE_SCHEMA[label].category, 'classes');
    assert.equal(CLASS_NODE_SCHEMA[label].categoryLabel, 'CLASSES');
    assert.deepEqual(Object.keys(CLASS_NODE_SCHEMA[label].properties).sort(), Object.keys(getClassNodeDefaults(label)).sort());
  }
  const m = getClassNodeDefaults('Method');
  m.parameters[0].name = 'changed';
  assert.equal(getClassNodeDefaults('Method').parameters[0].name, 'self');
  const s = getClassNodeDefaults('Signature');
  s.parameterContracts.push('N');
  assert.deepEqual(getClassNodeDefaults('Signature').parameterContracts, []);
});

test('all ten constructs generate explicitly, without automatic semicolons or lossy roundtrips', () => {
  const d = fixture();
  assert.deepEqual(validateClassDiagram(d), []);
  const source = generate(d, true);
  for (const label of CLASS_NODE_LABELS) {
    assert.ok(d.nodes.some(n => n.data.label === label), label);
    assert.ok(source.includes(`${CLASS_NODE_SCHEMA[label].functionName}(`), label);
  }
  assert.ok(!source.includes(';'));
  assert.equal((source.match(/field\(/g) || []).length, 1);
  assert.equal((source.match(/new\(/g) || []).length, 3);
  assert.match(source, /declare\(incrementSignature, 'V', signature\(array\("V", "N"\), "N"\)\)/);
  assert.match(source, /method\(Counter, "value", withSignature\(func\(self\)/);
  assert.deepEqual(JSON.parse(extractDiagramFromCode(source)!), d);
  assert.equal(generate(JSON.parse(JSON.stringify(d))), generate(d));
});

test('ordered declaration children use relation order, not node order, flow edges or relation insertion order', () => {
  const d = diagram([cls('Class'), cls('Field', { memberName: 'second' }, 'b'), cls('Field', { memberName: 'first' }, 'a'),
    cls('Interface'), cls('Requires', { memberName: 'second' }, 'rb'), cls('Requires', { memberName: 'first' }, 'ra'), cls('Method')],
  [rel('Class', 'b', 10), rel('Interface', 'rb', 10), rel('Class', 'a', 0), rel('Interface', 'ra', 0)]);
  d.edges.push({ id: 'flow', source: 'a', target: 'Method', sourceHandle: 'right' });
  const source = generate(d);
  assert.match(source, /class\(Example\) \{\n  field\("first", "V"\)\n  field\("second", "V"\)\n\}/);
  assert.match(source, /interface\(ExampleInterface\) \{\n  requires\("first"[^\n]+\n  requires\("second"/);
  assert.equal((source.match(/method\(/g) || []).length, 1);
});

test('optional expressions preserve omission versus explicit null, empty string, zero and empty maps', () => {
  for (const expr of ['null', '""', '0', 'map()', 'mapValue()']) {
    const d = diagram([cls('Class'), cls('Field', { hasDefault: true, defaultExpression: expr })], [rel('Class', 'Field')]);
    assert.ok(generate(d).includes(`field("value", "V", ${expr})`));
    assert.ok(generate(diagram([cls('New', { hasAttributes: true, attributesExpression: expr })])).includes(`new(Example, ${expr})`));
  }
  assert.ok(generate(diagram([cls('New', { hasAttributes: false, attributesExpression: '' })])).endsWith('new(Example)'));
  assert.ok(generate(diagram([cls('Class'), cls('Field', { defaultExpression: '' })], [rel('Class', 'Field')])).includes('field("value", "V")'));
  assert.ok(validateClassNodeProperties('New', { ...getClassNodeDefaults('New'), hasAttributes: true, attributesExpression: '' }).length);
  assert.ok(validateClassNodeProperties('Field', { ...getClassNodeDefaults('Field'), hasDefault: true, defaultExpression: '' }).length);
});

test('member strings escape; identifiers and expression references never auto-quote', () => {
  const name = 'say "hello"\\line\nnext';
  const source = generate(diagram([cls('Class', { identifier: 'Person' }), cls('Field', { memberName: name }),
    cls('Method', { memberName: name, classExpression: 'registry.current' }),
    cls('New', { classExpression: 'registry.current' }), cls('Satisfies', { valueExpression: 'items.at(0)', interfaceExpression: 'registry.iface' })],
  [rel('Class', 'Field')]));
  assert.ok(source.includes(`field(${JSON.stringify(name)}, "V")`));
  assert.ok(source.includes(`method(registry.current, ${JSON.stringify(name)}, func(self)`));
  assert.ok(source.includes('new(registry.current)'));
  assert.ok(source.includes('satisfies(items.at(0), registry.iface)'));
});

test('method editor reuses canonical parameters/defaults/body; modes and annotations do not compete', () => {
  const source = generate(diagram([cls('Method', { parameters: [{ name: 'self', value: '' }, { name: 'amount', value: '1' }],
    body: 'return(add(self.value, amount))\r\n', functionExpression: 'ignored()',
    hasSignature: true, signatureExpression: 'methodSignature' })]));
  assert.match(source, /func\(self, amount\) \{\n  if\(equal\(amount, DBNull\)\) \{\n    setq\(amount, 1\)\n  \}\nreturn/);
  assert.ok(source.includes('}, methodSignature)'));
  assert.ok(!source.includes('ignored'));
  assert.ok(!source.includes('\r'));
  const expr = cls('Method', { functionMode: 'expression', functionExpression: 'callbacks.increment', body: 'ignored', parameters: [] });
  assert.ok(generate(diagram([expr])).endsWith('method(Example, "getValue", callbacks.increment)'));
  for (const parameters of [[], [{ name: 'other', value: '' }], [{ name: 'self', value: 'null' }],
    [{ name: 'self', value: '' }, { name: 'self', value: '' }], [{ name: 'self', value: '' }, { name: 'bad-name', value: '' }]]) {
    assert.ok(validateClassNodeProperties('Method', { ...getClassNodeDefaults('Method'), parameters }).length);
  }
});

test('signature lists preserve parameter order and support every code and zero parameters', () => {
  assert.ok(generate(diagram([cls('Signature')])).endsWith('signature(array(), "V")'));
  assert.ok(generate(diagram([cls('Signature', { parameterContracts: [...CLASS_CONTRACT_CODES], returnContract: 'N' })]))
    .endsWith(`signature(array(${CLASS_CONTRACT_CODES.map(c => JSON.stringify(c)).join(', ')}), "N")`));
});

test('all expression constructs use explicit codegen in nested Declare and assignment paths exactly once', () => {
  for (const label of ['New', 'Signature', 'WithSignature', 'Satisfies', 'AssertSatisfies'] as const) {
    const expected = generate(diagram([cls(label)])).split('\n').slice(2).join('\n');
    for (const parent of [node('Declare', { variableName: 'x', typeSpecifier: 'V' }), node('Set Q', { variableName: 'x' })]) {
      const source = generate(diagram([parent, cls(label)], [rel(parent.id, label)]));
      assert.equal(source.split(expected).length - 1, 1, `${parent.data.label} / ${label}`);
      assert.ok(source.endsWith(`${expected})`));
    }
  }
});

test('malformed properties give node/property diagnostics, not coerced or substituted values', () => {
  const invalid: [ClassNodeLabel, Record<string, unknown>, string][] = [
    ['Class', { identifier: '"Quoted"' }, 'identifier'], ['Interface', { identifier: 'bad-name' }, 'identifier'],
    ['Field', { contract: 'Q' }, 'contract'], ['Field', { memberName: 3 }, 'memberName'],
    ['Field', { hasDefault: 'false' }, 'hasDefault'], ['New', { classExpression: {} }, 'classExpression'],
    ['Method', { functionMode: 'nested' }, 'functionMode'], ['Method', { body: 0 }, 'body'],
    ['Method', { parameters: ['self'] }, 'parameters.0'], ['Method', { hasSignature: true, signatureExpression: '' }, 'signatureExpression'],
    ['Requires', { signatureExpression: ' ' }, 'signatureExpression'], ['Signature', { parameterContracts: 'N' }, 'parameterContracts'],
    ['Signature', { parameterContracts: ['Q'] }, 'parameterContracts.0'], ['Signature', { returnContract: 'n' }, 'returnContract'],
    ['WithSignature', { functionExpression: '' }, 'functionExpression'], ['Satisfies', { valueExpression: null }, 'valueExpression'],
    ['AssertSatisfies', { interfaceExpression: '' }, 'interfaceExpression'],
  ];
  for (const [label, overrides, property] of invalid) {
    assert.ok(validateClassNodeProperties(label, { ...getClassNodeDefaults(label), ...overrides }).some(i => i.property === property), `${label}.${property}`);
  }
  for (const properties of [null, [], '']) assert.ok(validateClassNodeProperties('Class', properties).length);
  assert.throws(() => generate(diagram([cls('Class', { identifier: '' }, 'broken')])), /Node broken: identifier/);
  const d = diagram([cls('New')]);
  const generator = new ChariotCodeGenerator(d);
  d.nodes[0].data.properties!.classExpression = '';
  assert.throws(() => generator.generateChariotCode(), /Node New: classExpression/);
});

test('malformed and unsupported hierarchies reject rather than silently dropping declaration nodes', () => {
  const malformed = [
    diagram([cls('Field')]), diagram([cls('Requires')]),
    diagram([cls('Class'), cls('Method')], [rel('Class', 'Method')]),
    diagram([cls('Interface'), cls('Field')], [rel('Interface', 'Field')]),
    diagram([cls('Class'), cls('Requires')], [rel('Class', 'Requires')]),
    diagram([cls('New'), cls('Signature')], [rel('New', 'Signature')]),
    diagram([cls('Class'), cls('Field'), cls('Signature')], [rel('Class', 'Field'), rel('Field', 'Signature')]),
    diagram([cls('Class'), cls('Field')], [rel('Class', 'Field'), rel('Class', 'Field')]),
    diagram([cls('Class')], [rel('Class', 'missing')]),
    diagram([cls('Class')], [rel('missing', 'Class')]),
    diagram([cls('Class')], [rel('Class', 'Class')]),
    diagram([cls('Class'), cls('Field')], [rel('Class', 'Field'), rel('Field', 'Class')]),
    diagram([cls('Class'), cls('Class')]),
    diagram([cls('Class'), cls('Field')], [rel('Class', 'Field', Number.NaN)]),
    diagram([node('Declare', {}), cls('New'), cls('Signature')], [rel('Declare', 'New'), rel('Declare', 'Signature')]),
    diagram([node('Array', {}), cls('New')], [rel('Array', 'New')]),
    diagram([node('Function', {}), node('If', {}), cls('Class')], [rel('Function', 'If'), rel('If', 'Class')]),
    diagram([node('Then', {}), cls('Class')], [rel('Then', 'Class')]),
    diagram([node('If', {}), node('Then', {}), cls('Class')], [rel('If', 'Then'), rel('If', 'Class')]),
    diagram([cls('Class'), cls('Field'), cls('Field', {}, 'duplicate')], [rel('Class', 'Field'), rel('Class', 'duplicate')]),
  ];
  for (const d of malformed) {
    assert.ok(validateClassDiagram(d).length, JSON.stringify(d));
    assert.throws(() => new ChariotCodeGenerator(d), /Node .+: /);
  }
});

test('declarations work in consumed statement bodies; branches do not duplicate fields', () => {
  const d = diagram([node('If', { condition: 'true' }), node('Then', {}), cls('Class'), cls('Field')],
    [rel('If', 'Then'), rel('Then', 'Class'), rel('Class', 'Field')]);
  assert.deepEqual(validateClassDiagram(d), []);
  assert.equal((generate(d).match(/field\(/g) || []).length, 1);
  assert.match(generate(d), /if\(true\) \{\n  class\(Example\) \{\n    field/);
});

test('mixed legacy and Source AST nodes remain lossless; Source nesting remains explicit', () => {
  const ast = { _node_type: 'FuncCall', name: 'logPrint', args: [{ _node_type: 'Literal', val: 'source' }], pos: { line: 42 } };
  const sourceNode = { ...deserializeChariotAST(ast).nodes[0], id: 'source' };
  const d = diagram([node('Sleep', { milliseconds: '0' }), sourceNode, cls('Class'), cls('Field')], [rel('Class', 'Field')]);
  assert.match(generate(d), /sleep\(0\)\nlogPrint\('source'\)\nclass/);
  assert.deepEqual(JSON.parse(extractDiagramFromCode(generate(d, true))!).nodes[1].data.properties.ast, ast);
  d.nestingRelations.push(rel('Class', 'source'));
  assert.throws(() => generate(d), /Source|Class accepts only Field/);
  const legacy = diagram([node('Sleep', { milliseconds: '0' }), node('Set Q', { variableName: 'x', value: '42', valueType: 'expression' })]);
  assert.equal(generate(legacy), '// Classes\n\nsleep(0)\nsetq(x, 42)');
});

test('visual Classes fixture and edge cases execute with the current Go CLI', {
  skip: process.env.CHARIOT_CODEGEN_GO_TESTS !== '1',
}, async t => {
  const root = fileURLToPath(new URL('../../../', import.meta.url));
  const temp = mkdtempSync(join(tmpdir(), 'chariot-classes-'));
  t.after(() => rmSync(temp, { recursive: true, force: true }));
  const cli = join(temp, 'chariotctl');
  execFileSync('go', ['build', '-o', cli, './services/go-chariot/cmd/chariotctl'], { cwd: root, encoding: 'utf8' });
  const execute = (d: VisualDSLDiagram) => {
    const path = join(temp, 'generated.ch');
    writeFileSync(path, generate(d));
    return spawnSync(cli, ['-f', path], { cwd: root, encoding: 'utf8' });
  };
  const success = (d: VisualDSLDiagram, expected: string) => {
    const result = execute(d);
    assert.equal(result.status, 0, result.stderr + result.stdout);
    assert.equal(result.stdout.trim(), expected);
  };
  await t.test('all ten constructs: mutable method, bound extraction, both map forms, zero-arg requirements, positive/negative checks', () => {
    success(fixture(), '[42, 42, true, false, 7, 0]');
  });
  await t.test('assertSatisfies errors on negative conformance', () => {
    const d = fixture();
    d.nodes.find(n => n.id === 'checked')!.data.properties!.interfaceExpression = 'Missing';
    const result = execute(d);
    assert.ok(result.status !== 0 || /error/i.test(result.stdout + result.stderr));
    assert.match(result.stdout + result.stderr, /missing|conform|satisf/i);
  });
  await t.test('assertSatisfies returns the original mutable instance, not a copy', () => {
    const d = fixture();
    const resultIndex = d.nodes.findIndex(n => n.id === 'result');
    d.nodes.splice(resultIndex, 0, node('Set Q', {
      variableName: 'changedThroughAssertion', valueType: 'expression', value: 'checked.increment(1)',
    }, 'assertion-alias'));
    d.nodes.find(n => n.id === 'result')!.data.properties!.value = 'array(counter.count, checked.count, answer)';
    success(d, '[43, 43, 42]');
  });
  await t.test('required fields are not silently defaulted', () => {
    const d = fixture();
    d.nodes.find(n => n.id === 'field')!.data.properties!.hasDefault = false;
    const result = execute(d);
    assert.ok(result.status !== 0 || /error/i.test(result.stdout + result.stderr));
    assert.match(result.stdout + result.stderr, /count|required|missing/i);
  });
  await t.test('explicit null/empty/zero defaults and escaped member names survive the parser', () => {
    const fields = [cls('Field', { memberName: 'nil', contract: 'V', hasDefault: true, defaultExpression: 'null' }, 'nil'),
      cls('Field', { memberName: 'empty', contract: 'S', hasDefault: true, defaultExpression: '""' }, 'empty'),
      cls('Field', { memberName: 'say "hi"\\next', contract: 'N', hasDefault: true, defaultExpression: '0' }, 'escaped')];
    success(diagram([cls('Class'), ...fields,
      node('Set Q', { variableName: 'instance' }), cls('New'),
      node('Set Q', { variableName: 'result', valueType: 'expression', value: `array(isNull(instance.nil), strlen(instance.empty), getProp(instance, ${JSON.stringify('say "hi"\\next')}))` }, 'result')],
    [rel('Class', 'nil'), rel('Class', 'empty', 1), rel('Class', 'escaped', 2), rel('Set Q', 'New')]), '[true, 0, 0]');
  });
  await t.test('method annotation copies rather than mutating the original function', () => {
    const d = fixture();
    d.nodes.unshift(node('Set Q', { variableName: 'plain', valueType: 'expression', value: 'func(self, amount) { return(amount) }' }, 'plain'));
    d.nodes.find(n => n.id === 'annotate')!.data.properties!.functionExpression = 'plain';
    // Annotation is checked by the existing Counting requirement; a separate raw method lacks metadata.
    d.nodes.find(n => n.id === 'result')!.data.properties!.value = 'array(satisfies(counter, Counting), satisfies(raw, Counting))';
    const resultIndex = d.nodes.findIndex(n => n.id === 'result');
    d.nodes.splice(resultIndex, 0, cls('Class', { identifier: 'Raw' }, 'raw-class'),
      cls('Method', { classExpression: 'Raw', memberName: 'increment', functionMode: 'expression', functionExpression: 'plain' }, 'raw-method'),
      cls('Method', { classExpression: 'Raw', memberName: 'value', body: 'return(0)', hasSignature: true, signatureExpression: 'signature(array("V"), "N")' }, 'raw-value'),
      node('Set Q', { variableName: 'raw', valueType: 'expression', value: 'new(Raw)' }, 'raw-instance'));
    success(d, '[true, false]');
  });
});