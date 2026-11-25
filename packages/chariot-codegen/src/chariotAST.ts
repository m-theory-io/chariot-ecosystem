import type { VisualDSLDiagram, VisualDSLNode } from './chariotCodeGenerator';

// Wire names and fields match chariot.Node.ToMap/serializeNode. Class and
// interface declarations are FuncCall nodes, not separate AST discriminants.
type WireFields = { [key: string]: unknown };
type NodeList = ChariotASTNode[] | null;
export type ChariotASTNode = WireFields & (
  | { _node_type: 'MemberAccessNode'; receiver: ChariotASTNode; name: string }
  | { _node_type: 'FunctionCallNode'; funcExpr: ChariotASTNode; args: NodeList }
  | { _node_type: 'FuncCall'; name: string; args: NodeList }
  | { _node_type: 'VarRef'; name: string }
  | { _node_type: 'Literal'; val: unknown }
  | { _node_type: 'ArrayLiteralNode'; elements: NodeList }
  | { _node_type: 'Block'; stmts: NodeList }
  | { _node_type: 'FunctionDefNode'; parameters: string[] | null; body: ChariotASTNode; source?: string }
  | { _node_type: 'IfNode'; condition: ChariotASTNode; trueBranch: NodeList; falseBranch: NodeList }
  | { _node_type: 'WhileNode'; condition: ChariotASTNode; body: NodeList }
);

export interface SourceNodeProperties {
  // Keep the entire wire object, including positions and unknown future fields.
  ast?: ChariotASTNode | (WireFields & { _node_type: string });
  code?: string;
}

function object(value: unknown, path: string): WireFields {
  if (!value || typeof value !== 'object' || Array.isArray(value)) {
    throw new Error(`${path}: expected an AST object`);
  }
  return value as WireFields;
}

function identifier(value: unknown, path: string, dotted = false): string {
  const pattern = dotted ? /^[A-Za-z_][A-Za-z_0-9]*(\.[A-Za-z_][A-Za-z_0-9]*)*$/ : /^[A-Za-z_][A-Za-z_0-9]*$/;
  if (typeof value !== 'string' || !pattern.test(value)) {
    throw new Error(`${path}: invalid identifier`);
  }
  return value;
}

function list(value: unknown, path: string): unknown[] {
  // Go nil slices serialize as null; an absent required field is not a nil slice.
  if (value === null) return [];
  if (!Array.isArray(value)) throw new Error(`${path}: expected an array or null`);
  return value;
}

function quote(value: string): string {
  return "'" + value.replace(/\\/g, '\\\\').replace(/'/g, "\\'")
    .replace(/\n/g, '\\n').replace(/\r/g, '\\r').replace(/\t/g, '\\t') + "'";
}

/** Emit source from the structured body, never FunctionDefNode.source (a header only). */
export function generateChariotSourceFromAST(value: unknown): string {
  function emit(value: unknown, path: string): string {
    const n = object(value, path);
    const child = (key: string) => emit(n[key], `${path}.${key}`);
    const children = (key: string, separator: string) => list(n[key], `${path}.${key}`)
      .map((item, i) => emit(item, `${path}.${key}[${i}]`)).join(separator);
    const block = (key: string) => `{\n${children(key, '\n')}\n}`;
    switch (n._node_type) {
      case 'VarRef': return identifier(n.name, `${path}.name`);
      case 'Literal':
        if (typeof n.val === 'string') return quote(n.val);
        if (typeof n.val === 'number' && Number.isFinite(n.val)) return Object.is(n.val, -0) ? '-0' : String(n.val);
        if (typeof n.val === 'boolean') return String(n.val);
        if (n.val === null) return 'null';
        throw new Error(`${path}.val: unsupported literal value`);
      case 'MemberAccessNode': return `(${child('receiver')}).${identifier(n.name, `${path}.name`)}`;
      case 'FunctionCallNode': return `(${child('funcExpr')})(${children('args', ', ')})`;
      case 'ArrayLiteralNode': return `[${children('elements', ', ')}]`;
      case 'Block': return block('stmts');
      case 'FunctionDefNode': {
        const params = list(n.parameters, `${path}.parameters`).map((p, i) => identifier(p, `${path}.parameters[${i}]`));
        if (object(n.body, `${path}.body`)._node_type !== 'Block') throw new Error(`${path}.body: expected Block`);
        return `func(${params.join(', ')}) ${child('body')}`;
      }
      case 'FuncCall': {
        const name = identifier(n.name, `${path}.name`, true);
        const args = list(n.args, `${path}.args`);
        if (name === 'class' || name === 'interface') {
          if (args.length !== 2 || object(args[0], `${path}.args[0]`)._node_type !== 'VarRef' ||
              object(args[1], `${path}.args[1]`)._node_type !== 'Block') {
            throw new Error(`${path}: ${name} expects a VarRef name and Block body`);
          }
          return `${name}(${emit(args[0], `${path}.args[0]`)}) ${emit(args[1], `${path}.args[1]`)}`;
        }
        // A trailing Block is parser syntax (not a generic argument), e.g. listen.
        const trailingBlock = args.length > 0 && object(args[args.length - 1], `${path}.args[${args.length - 1}]`)._node_type === 'Block';
        const rendered = args.map((arg, i) => emit(arg, `${path}.args[${i}]`));
        const body = trailingBlock ? ` ${rendered.pop()}` : '';
        return `${name}(${rendered.join(', ')})${body}`;
      }
      case 'IfNode': {
        const otherwise = list(n.falseBranch, `${path}.falseBranch`);
        return `if (${child('condition')}) ${block('trueBranch')}${otherwise.length ? ` else ${block('falseBranch')}` : ''}`;
      }
      case 'WhileNode': return `while (${child('condition')}) ${block('body')}`;
      default: throw new Error(`${path}: unsupported AST node type ${String(n._node_type)}`);
    }
  }
  const root = object(value, '$');
  // A program Block is a statement list; nested Blocks retain braces.
  if (root._node_type === 'Block') {
    return list(root.stmts, '$.stmts').map((stmt, i) => emit(stmt, `$.stmts[${i}]`)).join('\n');
  }
  return emit(root, '$');
}

/** Deserialize without projecting or discarding any structured source fields. */
export function deserializeChariotAST(value: unknown, name = 'Imported AST'): VisualDSLDiagram {
  const ast = object(value, '$');
  if (typeof ast._node_type !== 'string' || !ast._node_type) throw new Error('$: missing _node_type');
  // Snapshot the JSON wire payload; unsupported nodes remain available for export.
  const preserved = JSON.parse(JSON.stringify(ast)) as SourceNodeProperties['ast'];
  return {
    name, edges: [], nestingRelations: [],
    nodes: [{ id: 'source-1', type: 'logicon', position: { x: 80, y: 80 },
      data: { label: 'Source', icon: '📄', category: 'source', properties: { ast: preserved } } }],
  };
}

export function generateSourceNodeCode(properties: SourceNodeProperties): string {
  // Never fall back to stale code when a structured node cannot be emitted.
  if (properties.ast !== undefined) return generateChariotSourceFromAST(properties.ast);
  if (typeof properties.code === 'string' && properties.code.trim()) return properties.code;
  throw new Error('Source node requires structured ast or non-empty code');
}

/** Non-mutating UI diagnostic, also used for imported/saved Source nodes. */
export function getSourceNodeDiagnostic(node: Pick<VisualDSLNode, 'data'>): string | undefined {
  if (node.data.label !== 'Source') return undefined;
  try {
    generateSourceNodeCode(node.data.properties || {});
    return 'Visual editing unsupported. Source is preserved read-only; code generation is supported.';
  } catch (error) {
    return `Visual editing and code generation unsupported. Source is preserved: ${error instanceof Error ? error.message : String(error)}`;
  }
}
