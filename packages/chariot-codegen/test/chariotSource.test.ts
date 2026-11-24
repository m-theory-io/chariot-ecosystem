import assert from 'node:assert/strict';
import { execFileSync } from 'node:child_process';
import { mkdtempSync, rmSync, writeFileSync } from 'node:fs';
import { tmpdir } from 'node:os';
import { join } from 'node:path';
import { fileURLToPath } from 'node:url';
import { test } from 'node:test';
import { deserializeChariotAST, generateChariotCodeFromDiagram, generateChariotSourceFromAST } from '../dist/index.js';

const ref = (name: string) => ({ _node_type: 'VarRef', name });
const lit = (val: unknown) => ({ _node_type: 'Literal', val });
const call = (name: string, ...args: unknown[]) => ({ _node_type: 'FuncCall', name, args });
const block = (...stmts: unknown[]) => ({ _node_type: 'Block', stmts });
const member = (receiver: unknown, name: string) => ({ _node_type: 'MemberAccessNode', receiver, name });
const invoke = (funcExpr: unknown, ...args: unknown[]) => ({ _node_type: 'FunctionCallNode', funcExpr, args });
const fn = (...stmts: unknown[]) => ({ _node_type: 'FunctionDefNode', parameters: [], body: block(...stmts) });

// Opt in to the local Go toolchain; ordinary TS tests remain self-contained.
test('generated newline-style Chariot and object examples execute through the Go CLI', {
  skip: process.env.CHARIOT_CODEGEN_GO_TESTS !== '1',
}, async t => {
  const root = fileURLToPath(new URL('../../../', import.meta.url));
  const temp = mkdtempSync(join(tmpdir(), 'chariot-codegen-'));
  t.after(() => rmSync(temp, { recursive: true, force: true }));
  const cli = join(temp, 'chariotctl');
  execFileSync('go', ['build', '-o', cli, './services/go-chariot/cmd/chariotctl'], { cwd: root, encoding: 'utf8' });
  const execute = (source: string) => {
    const path = join(temp, 'generated.ch');
    writeFileSync(path, source);
    return execFileSync(cli, ['-f', path], { cwd: root, encoding: 'utf8' }).trim();
  };
  const statements = block(
    call('setq', ref('answer'), lit(40)),
    call('setq', ref('next'), fn(call('setq', ref('answer'), call('add', ref('answer'), lit(1))))),
    call('length', { _node_type: 'ArrayLiteralNode', elements: [lit(1), lit(2)] }),
    invoke(ref('next')),
    ref('answer'),
    invoke(ref('next')),
    ref('answer'),
  );
  await t.test('named call and bare name before grouped next statements', () => {
    const source = generateChariotSourceFromAST(statements);
    assert.ok(!source.includes(';'));
    assert.match(source, /answer\n\(next\)\(\)\nanswer$/);
    assert.equal(execute(source), '42');
  });
  await t.test('mixed visual/Source nodes and separate Source node boundaries', () => {
    const diagram = deserializeChariotAST(statements);
    diagram.nodes.unshift({ id: 'sleep', type: 'logicon', position: { x: 0, y: 0 },
      data: { label: 'Sleep', icon: '', category: 'system', properties: { milliseconds: '0' } } });
    diagram.nodes.push({ ...deserializeChariotAST(invoke(fn(lit(42)))).nodes[0], id: 'last' });
    const source = generateChariotCodeFromDiagram(JSON.stringify(diagram), { embedSource: false });
    assert.ok(!source.includes(';'));
    assert.equal(execute(source), '42');
  });
  await t.test('declarations, nested blocks, member calls and chained arbitrary callees', () => {
    const ast = block(
      call('class', ref('Answer'), block(call('field', ref('value'), lit('N'), lit(42)))),
      call('interface', ref('Answerer'), block(call('requires', ref('answer'), call('signature', call('array'), lit('N'))))),
      call('method', ref('Answer'), ref('answer'), call('withSignature',
        { _node_type: 'FunctionDefNode', parameters: ['self'], body: block(member(ref('self'), 'value')) },
        call('signature', call('array', lit('V')), lit('N')))),
      call('setq', ref('instance'), call('new', ref('Answer'))),
      { _node_type: 'WhileNode', condition: lit(false), body: [invoke(ref('never'))] },
      { _node_type: 'IfNode', condition: call('satisfies', ref('instance'), ref('Answerer')),
        trueBranch: [lit(0), invoke(member(ref('instance'), 'answer'))], falseBranch: [lit(-1)] },
      invoke(invoke(fn(fn(invoke(member(ref('instance'), 'answer')))))),
    );
    const source = generateChariotSourceFromAST(ast);
    assert.ok(!source.includes(';'));
    assert.equal(execute(source), '42');
  });
  const examples = {
    'member-calls': '[42, 42, 42, true]',
    classes: '[Grace, Grace, Ada, 1, 0, 42, Grace]',
    interfaces: '[true, 42, 42, F, O]',
    serialization: '[Ada, Ada, Grace, O]',
  };
  for (const [name, expected] of Object.entries(examples)) {
    await t.test(`object example: ${name}`, () => {
      assert.equal(execFileSync(cli, ['-f', join(root, 'examples/objects', `${name}.ch`)], { cwd: root, encoding: 'utf8' }).trim(), expected);
    });
  }
});
