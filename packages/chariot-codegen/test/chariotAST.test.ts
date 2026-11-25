import assert from 'node:assert/strict';
import { test } from 'node:test';
import {
  deserializeChariotAST, generateChariotSourceFromAST, generateChariotCodeFromDiagram,
  extractDiagramFromCode, getSourceNodeDiagnostic,
} from '../dist/index.js';

const ref = (name: string) => ({ _node_type: 'VarRef', name });
const lit = (val: unknown) => ({ _node_type: 'Literal', val });
const call = (name: string, ...args: unknown[]) => ({ _node_type: 'FuncCall', name, args });
const block = (...stmts: unknown[]) => ({ _node_type: 'Block', stmts });
const member = (receiver: unknown, name: string) => ({ _node_type: 'MemberAccessNode', receiver, name, pos: { file: 'fixture.ch', line: 4, column: 2 } });
const invoke = (funcExpr: unknown, ...args: unknown[]) => ({ _node_type: 'FunctionCallNode', funcExpr, args });
const fn = (parameters: string[], ...stmts: unknown[]) => ({ _node_type: 'FunctionDefNode', parameters, body: block(...stmts), source: `func(${parameters.join(',')})` });
const generate = (diagram: unknown, embedSource = false) => generateChariotCodeFromDiagram(JSON.stringify(diagram), { embedSource });

test('member reads, chained receivers, bound calls, arbitrary callees, and nested arguments', () => {
  assert.equal(generateChariotSourceFromAST(member(member(ref('person'), 'address'), 'city')), '((person).address).city');
  assert.equal(generateChariotSourceFromAST(invoke(member(ref('person'), 'greet'), lit('Hi'), member(ref('other'), 'name'))), "((person).greet)('Hi', (other).name)");
  assert.equal(generateChariotSourceFromAST(invoke(invoke(ref('factory')), lit(4))), '((factory)())(4)');
  assert.equal(generateChariotSourceFromAST(invoke(fn([], lit(1)))), '(func() {\n1\n})()');
});

test('class/interface declarations use FuncCall plus Block and canonical function names', () => {
  const ast = block(
    call('class', ref('Person'), block(call('field', lit('age'), lit('N'), lit(0)))),
    call('interface', ref('Greeter'), block(call('requires', lit('greet'), call('signature', call('array', lit('S')), lit('S'))))),
    call('method', ref('Person'), lit('greet'), call('withSignature', fn(['self', 'name'], member(ref('self'), 'age')), call('signature', call('array', lit('V'), lit('S')), lit('S')))),
    call('setq', ref('p'), call('new', ref('Person'), call('mapValue', lit('age'), lit(20)))),
    call('setq', member(ref('p'), 'age'), lit(21)),
    call('satisfies', ref('p'), ref('Greeter')),
    call('assertSatisfies', ref('p'), ref('Greeter')),
    invoke(member(ref('p'), 'greet'), lit('Ada')),
  );
  const source = generateChariotSourceFromAST(ast);
  assert.match(source, /class\(Person\) \{\nfield\('age', 'N', 0\)\n\}\n/);
  assert.match(source, /interface\(Greeter\) \{\nrequires\('greet', signature\(array\('S'\), 'S'\)\)\n\}\n/);
  assert.match(source, /method\(Person, 'greet', withSignature\(func\(self, name\) \{\n\(self\).age\n/);
  assert.ok(source.includes("new(Person, mapValue('age', 20))"));
  assert.ok(source.includes('setq((p).age, 21)'));
  assert.ok(source.includes('satisfies(p, Greeter)\nassertSatisfies(p, Greeter)'));
  assert.ok(source.endsWith("((p).greet)('Ada')"));
  assert.ok(!source.includes(';'));
  const diagram = deserializeChariotAST(ast);
  assert.deepEqual(JSON.parse(extractDiagramFromCode(generate(diagram, true))), diagram);
  assert.deepEqual(diagram.nodes[0].data.properties?.ast, ast);
});

test('structured function body wins over incomplete or stale Source header', () => {
  const ast = fn(['self'], invoke(member(ref('self'), 'run')));
  ast.source = 'func(stale)';
  assert.equal(generateChariotSourceFromAST(ast), 'func(self) {\n((self).run)()\n}');
});

test('newlines separate grouped next statements, including after bare names', () => {
  const ast = block(call('logPrint', lit('first')), ref('bare'), invoke(member(ref('p'), 'run')), invoke(ref('next')));
  assert.equal(generateChariotSourceFromAST(ast), "logPrint('first')\nbare\n((p).run)()\n(next)()");
  const diagram = deserializeChariotAST(ast);
  diagram.nodes.unshift({ id: 'legacy', type: 'logicon', position: { x: 0, y: 0 }, data: { label: 'Sleep', icon: '', category: 'system', properties: { milliseconds: '1' } } });
  assert.match(generate(diagram), /sleep\([^)]*\)\nlogPrint/);
  diagram.nodes.push({ ...deserializeChariotAST(invoke(ref('last'))).nodes[0], id: 'last' });
  const source = generate(diagram);
  assert.ok(source.endsWith('(next)()\n(last)()'));
  assert.ok(!source.includes(';'));
});

test('arrays, null slices, control flow and escaped strings survive source generation', () => {
  const ast = fn([], { _node_type: 'IfNode', condition: ref('ok'), trueBranch: [invoke(ref('yes'))], falseBranch: [invoke(ref('no'))] },
    { _node_type: 'WhileNode', condition: ref('ok'), body: [call('setq', ref('ok'), ref('false'))] });
  assert.match(generateChariotSourceFromAST(ast), /if \(ok\) \{\n\(yes\)\(\)\n\} else \{\n\(no\)\(\)\n\}\nwhile/);
  assert.equal(generateChariotSourceFromAST({ _node_type: 'FunctionCallNode', funcExpr: ref('run'), args: null }), '(run)()');
  assert.equal(generateChariotSourceFromAST({ _node_type: 'ArrayLiteralNode', elements: [lit(1), lit(true), lit(null), lit("a'\\\n\t雪")] }), "[1, true, null, 'a\\'\\\\\\n\\t雪']");
  assert.equal(generateChariotSourceFromAST({ _node_type: 'Block', stmts: null }), '');
});

test('unknown nested nodes are retained exactly and block codegen, not silently dropped', () => {
  const ast = block(fn([], { _node_type: 'FutureNode', payload: { important: [1, 2] } }));
  const diagram = deserializeChariotAST(ast);
  assert.deepEqual(diagram.nodes[0].data.properties?.ast, ast);
  assert.match(getSourceNodeDiagnostic(diagram.nodes[0])!, /code generation unsupported.*FutureNode/);
  assert.throws(() => generate(diagram), /stmts\[0\].body.stmts\[0\].*FutureNode/);
  diagram.nodes[0].data.properties!.code = "logPrint('stale fallback')";
  assert.throws(() => generate(diagram), /FutureNode/);
});

test('deserialization snapshots metadata, including unknown root nodes and position fields', () => {
  const ast = member(ref('p'), 'age');
  const diagram = deserializeChariotAST(ast);
  ast.pos.line = 99;
  assert.equal((diagram.nodes[0].data.properties?.ast as typeof ast).pos.line, 4);
  const future = deserializeChariotAST({ _node_type: 'FutureDeclaration', source: 'header', metadata: [1] });
  assert.deepEqual(JSON.parse(JSON.stringify(future)).nodes[0].data.properties.ast, { _node_type: 'FutureDeclaration', source: 'header', metadata: [1] });
  assert.throws(() => generate(future), /FutureDeclaration/);
});

test('malformed known nodes fail with paths, not misleading partial source', () => {
  for (const ast of [
    { _node_type: 'MemberAccessNode', name: 'x' },
    { _node_type: 'MemberAccessNode', receiver: ref('p'), name: 'bad;name' },
    { _node_type: 'FunctionCallNode', funcExpr: ref('p') },
    call('class', lit('WrongName'), block()),
    call('interface', ref('Bad'), lit('not a block')),
    { _node_type: 'Literal', val: { unsupported: true } },
    { _node_type: 'Literal', val: Infinity },
    { _node_type: 'FunctionDefNode', parameters: ['x;evil'], body: block() },
    { _node_type: 'FunctionDefNode', parameters: [], body: ref('notBlock') },
  ]) assert.throws(() => generateChariotSourceFromAST(ast), /\$/);
});

test('legacy named/dotted calls and diagram metadata round trips stay compatible', () => {
  assert.equal(generateChariotSourceFromAST(call('legacy.Run', lit(2))), 'legacy.Run(2)');
  const diagram = { name: 'legacy', nodes: [{ id: 's', type: 'logicon', position: { x: 0, y: 0 }, data: { label: 'Start', icon: '', category: 'control' } }], edges: [], nestingRelations: [] };
  assert.equal(generate(diagram), '// legacy\n\n// Starting legacy');
  assert.deepEqual(JSON.parse(extractDiagramFromCode(generate(diagram, true))), diagram);
});

test('text Source nodes work but missing source, unknown visual nodes and nesting fail explicitly', () => {
  const diagram = deserializeChariotAST(ref('x'));
  diagram.nodes[0].data.properties = { code: "logPrint('raw source')" };
  assert.ok(generate(diagram).includes("logPrint('raw source')"));
  assert.match(getSourceNodeDiagnostic(diagram.nodes[0])!, /Visual editing unsupported.*code generation is supported/);
  diagram.nodes[0].data.properties = {};
  assert.throws(() => generate(diagram), /requires structured ast/);
  diagram.nestingRelations.push({ parentId: 'source-1', childId: 'x', order: 0 });
  assert.throws(() => generate(diagram), /cannot participate in visual nesting/);
  diagram.nestingRelations = [];
  diagram.nodes[0].data.label = 'Future UI node';
  assert.throws(() => generate(diagram), /Unsupported Visual DSL node source-1/);
});

test('function defaults use newline style without rewriting user source or string semicolons', () => {
  const diagram = deserializeChariotAST(lit('keep;literal'));
  diagram.nodes.unshift({ id: 'function', type: 'logicon', position: { x: 0, y: 0 },
    data: { label: 'func', icon: '', category: 'function', properties: {
      parameters: [{ name: 'value', value: '42' }], body: 'value',
    } } });
  assert.ok(generate(diagram).includes('func(value) {\n  if(equal(value, DBNull)) {\n    setq(value, 42)\n  }\nvalue\n}\n\'keep;literal\''));
  diagram.nodes[1].data.properties = { code: '42;\n(next)()' };
  assert.ok(generate(diagram).endsWith('42;\n(next)()'));
});
