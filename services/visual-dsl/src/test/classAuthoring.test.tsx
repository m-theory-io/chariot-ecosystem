import assert from 'node:assert/strict';
import { test } from 'node:test';
import React from 'react';
import { renderToStaticMarkup } from 'react-dom/server';
import {
  CLASS_NODE_LABELS, CLASS_NODE_SCHEMA, CLASS_CONTRACT_CODES, getClassNodeDefaults,
  validateClassNodeProperties, validateClassDiagram, generateChariotCodeFromDiagram, extractDiagramFromCode,
  type ClassNodeLabel, type VisualDSLDiagram, type VisualDSLNode,
} from 'chariot-codegen';
import { ClassNodePropertiesDialog } from '../components/dialogs/ClassNodeProperties';
import { createLogiconData, classNestingError, declarationPlacementMessage } from '../utils/classAuthoring';
import { logiconDefinitions } from '../data/logicons';
import { deserializeDiagramInput } from '../utils/diagramInput';
import { parseDiagramDraft, serializeDiagramDraft } from '../utils/diagramDraft';

const noop = () => {};
function node(label: ClassNodeLabel, id: string = label): VisualDSLNode {
  return { id, type: 'logicon', position: { x: 0, y: 0 }, data: createLogiconData(label, '', 'classes') };
}
const diagram = (...nodes: VisualDSLNode[]): VisualDSLDiagram => ({ name: 'Classes', nodes, edges: [], nestingRelations: [] });

test('palette has exactly the ten canonical labels and category', () => {
  const classes = logiconDefinitions.filter(definition => definition.category === 'classes');
  assert.deepEqual(classes.map(definition => definition.label), CLASS_NODE_LABELS);
  classes.forEach(definition => assert.equal(definition.id, CLASS_NODE_SCHEMA[definition.label as ClassNodeLabel].functionName));
});

for (const label of CLASS_NODE_LABELS) {
  test(`${label} dialog: canonical defaults, active controls, labels, validation and fresh constructor`, () => {
    const defaults = getClassNodeDefaults(label);
    assert.deepEqual(validateClassNodeProperties(label, defaults), []);
    const html = renderToStaticMarkup(<ClassNodePropertiesDialog nodeLabel={label} initialProperties={defaults} onSave={noop} onClose={noop} onDelete={noop} />);
    assert.match(html, new RegExp(`${label} Properties`));
    assert.match(html, /role="dialog"/);
    assert.match(html, /aria-modal="true"/);
    assert.match(html, /Save Properties/);
    assert.match(html, /Cancel/);
    assert.match(html, /Delete/);
    for (const [property, field] of Object.entries(CLASS_NODE_SCHEMA[label].properties)) {
      const active = !field.activeWhen || (defaults as unknown as Record<string, unknown>)[field.activeWhen.property] === field.activeWhen.equals;
      assert.equal(html.includes(`-${property}"`), active, `${property} control visibility`);
      if (active && field.kind === 'contract') CLASS_CONTRACT_CODES.forEach(code => assert.ok(html.includes(`value="${code}"`)));
    }
    const first = createLogiconData(label, '', 'classes');
    const second = createLogiconData(label, '', 'classes');
    assert.deepEqual(first.properties, defaults);
    assert.notEqual(first.properties, second.properties);
    if (label === 'Method') {
      const props = first.properties as ReturnType<typeof getClassNodeDefaults<'Method'>>;
      props.parameters[0].name = 'changed';
      assert.equal((second.properties as typeof props).parameters[0].name, 'self');
      assert.match(html, /Parameter 1 name/);
      assert.match(html, /Parameter 1 value/);
      assert.match(html, /h-80/);
    }
  });
}

test('conditional controls retain explicit null, zero, empty-string expression and inactive edits', () => {
  for (const expression of ['null', '0', '""']) {
    const props = { ...getClassNodeDefaults('Field'), hasDefault: true, defaultExpression: expression };
    const html = renderToStaticMarkup(<ClassNodePropertiesDialog nodeLabel="Field" initialProperties={props} onSave={noop} onClose={noop} onDelete={noop} />);
    assert.match(html, /Default expression/);
    assert.deepEqual(validateClassNodeProperties('Field', props), []);
    assert.equal(JSON.parse(JSON.stringify(props)).defaultExpression, expression);
  }
  const props = { ...getClassNodeDefaults('Method'), functionMode: 'expression' as const, functionExpression: 'bound.method', hasSignature: true };
  const html = renderToStaticMarkup(<ClassNodePropertiesDialog nodeLabel="Method" initialProperties={props} onSave={noop} onClose={noop} onDelete={noop} />);
  assert.match(html, /bound.method/);
  assert.match(html, /Signature expression/);
  assert.ok(!html.includes('Parameter 1 name'));
  assert.deepEqual(validateClassNodeProperties('Method', props), []);
});

test('declarations reject wrong children, members reject root placement, methods reject declaration bodies', () => {
  const d = diagram(...CLASS_NODE_LABELS.map(label => node(label)));
  for (const label of CLASS_NODE_LABELS) {
    const classError = classNestingError(d, 'Class', label);
    const interfaceError = classNestingError(d, 'Interface', label);
    assert.equal(classError === undefined, label === 'Field', `Class -> ${label}`);
    assert.equal(interfaceError === undefined, label === 'Requires', `Interface -> ${label}`);
  }
  for (const label of ['Field', 'Requires'] as const) {
    assert.match(declarationPlacementMessage(label)!, /enable Nesting Mode/);
    assert.ok(validateClassDiagram(diagram(node(label))).some(issue => issue.property === 'nestingRelations'));
  }
  assert.equal(declarationPlacementMessage('Method'), undefined);
  assert.deepEqual(validateClassDiagram(diagram(node('Method'))), []);
  assert.match(classNestingError(d, 'Method', 'New')!, /not supported|property inputs/);
});

test('guards detect duplicates, cycles, unsupported ancestors, and never mutate the original draft', () => {
  const d = diagram(node('Class'), node('Field', 'one'), node('Field', 'two'));
  d.nestingRelations = [{ parentId: 'Class', childId: 'one', order: 4 }];
  const before = structuredClone(d);
  assert.match(classNestingError(d, 'Class', 'two')!, /Duplicate declaration member/);
  assert.match(classNestingError(d, 'one', 'Class')!, /cycle|not supported|property inputs/i);
  assert.deepEqual(d, before);
  const legacy: VisualDSLNode = { id: 'legacy', type: 'logicon', position: { x: 0, y: 0 }, data: { label: 'Function', icon: '', category: 'control' } };
  assert.match(classNestingError(diagram(legacy, node('Class')), 'legacy', 'Class')!, /not supported|ignored/);
});

test('ordered declaration members and explicit defaults are emitted without normalizing expressions', () => {
  const d = diagram(node('Class'), node('Field', 'late'), node('Field', 'early'), node('Field', 'required'));
  d.nodes[1].data.properties = { ...getClassNodeDefaults('Field'), memberName: 'escaped"name', hasDefault: true, defaultExpression: '""' };
  d.nodes[2].data.properties = { ...getClassNodeDefaults('Field'), memberName: 'zero', contract: 'N', hasDefault: true, defaultExpression: '0' };
  d.nodes[3].data.properties = { ...getClassNodeDefaults('Field'), memberName: 'required', hasDefault: false, defaultExpression: 'null' };
  d.nestingRelations = [{ parentId: 'Class', childId: 'late', order: 9 }, { parentId: 'Class', childId: 'early', order: 2 }, { parentId: 'Class', childId: 'required', order: 20 }];
  const code = generateChariotCodeFromDiagram(JSON.stringify(d), { embedSource: false });
  assert.ok(code.indexOf('field("zero", "N", 0)') < code.indexOf('field("escaped\\"name", "V", "")'));
  assert.match(code, /field\("required", "V"\)/);
  assert.ok(!code.includes(';'));
  for (const [label, properties] of [
    ['Class', { identifier: 'quoted name' }], ['Field', { ...getClassNodeDefaults('Field'), contract: 'BAD' }],
    ['New', { ...getClassNodeDefaults('New'), classExpression: ' ' }],
    ['Requires', { ...getClassNodeDefaults('Requires'), signatureExpression: '' }],
  ] as const) assert.ok(validateClassNodeProperties(label, properties).length > 0);
});

test('all properties, ordered bodies and Source metadata survive import, generated metadata and automatic drafts', () => {
  const d = diagram(...CLASS_NODE_LABELS.map(label => node(label)));
  d.nestingRelations = [{ parentId: 'Class', childId: 'Field', order: 8 }, { parentId: 'Interface', childId: 'Requires', order: 3 }];
  d.nodes.push({ id: 'source', type: 'logicon', position: { x: 1, y: 2 }, data: { label: 'Source', category: 'control', icon: '', properties: {
    ast: { _node_type: 'VarRef', name: 'Example', pos: { line: 42 } }, source: 'Example', custom: 'preserved',
  } } });
  assert.deepEqual(validateClassDiagram(d), []);
  assert.match(classNestingError(d, 'Class', 'source')!, /read-only/);
  const loaded = deserializeDiagramInput(JSON.parse(JSON.stringify(d)));
  assert.deepEqual(loaded, d);
  const generated = generateChariotCodeFromDiagram(JSON.stringify(d));
  assert.match(generated, /class\(Example\)/);
  assert.match(generated, /method\(Example/);
  assert.match(generated, /assertSatisfies\(/);
  assert.deepEqual(JSON.parse(extractDiagramFromCode(generated)), d);
  const draft = { version: 1 as const, diagram: { ...d, groupCount: 2, created: 'then', modified: 'now' }, editor: {
    selectedNodeId: 'Method', selectedParentId: 'Class', nestingMode: true, direction: 'down' as const, viewport: { x: 0, y: 0, zoom: 1 },
  } };
  assert.deepEqual(parseDiagramDraft(serializeDiagramDraft(draft)), draft);
});