import assert from 'node:assert/strict';
import { test } from 'node:test';
import { deserializeDiagramInput } from '../utils/diagramInput';
import { getSourceNodeDiagnostic, generateChariotCodeFromDiagram, extractDiagramFromCode } from 'chariot-codegen';

test('frontend import/export preserves member/call AST and exposes read-only UI diagnostic', () => {
  const ast = { _node_type: 'FunctionCallNode', funcExpr: { _node_type: 'MemberAccessNode', receiver: { _node_type: 'VarRef', name: 'person' }, name: 'greet', pos: { line: 2 } }, args: [] };
  const diagram = deserializeDiagramInput(ast);
  assert.equal(diagram.nodes[0].data.label, 'Source');
  assert.deepEqual(diagram.nodes[0].data.properties?.ast, ast);
  assert.match(getSourceNodeDiagnostic(diagram.nodes[0])!, /Visual editing unsupported/);
  const code = generateChariotCodeFromDiagram(JSON.stringify(diagram));
  assert.ok(code.includes('((person).greet)()'));
  assert.deepEqual(JSON.parse(extractDiagramFromCode(code)), diagram);
});

test('unsupported new nodes remain exportable JSON and show codegen error, not fake function UI', () => {
  const ast = { _node_type: 'FutureNode', nested: { payload: [1, 'keep'] } };
  const diagram = deserializeDiagramInput(ast);
  const reloaded = deserializeDiagramInput(JSON.parse(JSON.stringify(diagram)));
  assert.deepEqual(reloaded.nodes[0].data.properties?.ast, ast);
  assert.match(getSourceNodeDiagnostic(reloaded.nodes[0])!, /code generation unsupported.*FutureNode/);
  assert.throws(() => generateChariotCodeFromDiagram(JSON.stringify(reloaded)), /FutureNode/);
});

test('Source export separates grouped statements with newlines and keeps call parens attached', () => {
  const ref = (name: string) => ({ _node_type: 'VarRef', name });
  const invoke = (funcExpr: unknown) => ({ _node_type: 'FunctionCallNode', funcExpr, args: [] });
  const ast = { _node_type: 'Block', stmts: [ref('bare'), invoke(ref('next')),
    invoke({ _node_type: 'MemberAccessNode', receiver: ref('person'), name: 'greet' })] };
  const diagram = deserializeDiagramInput(ast);
  diagram.nodes.unshift({ id: 'legacy', type: 'logicon', position: { x: 0, y: 0 },
    data: { label: 'Sleep', icon: '', category: 'system', properties: { milliseconds: '0' } } });
  const code = generateChariotCodeFromDiagram(JSON.stringify(diagram), { embedSource: false });
  assert.match(code, /sleep\(0\)\nbare\n\(next\)\(\)\n\(\(person\)\.greet\)\(\)$/);
  assert.ok(!code.includes(';'));
  assert.deepEqual(deserializeDiagramInput(JSON.parse(JSON.stringify(diagram))).nodes[1].data.properties?.ast, ast);
});

test('existing diagram metadata is not normalized away and malformed imports fail before replacing canvas', () => {
  const existing = { name: 'Old', nodes: [], edges: [], nestingRelations: [], subflows: { saved: true }, groupCount: 2 };
  assert.equal(deserializeDiagramInput(existing), existing);
  for (const bad of [null, [], 'text', {}, { _node_type: '' }, { nodes: [], edges: 'bad' }]) {
    assert.throws(() => deserializeDiagramInput(bad));
  }
});
