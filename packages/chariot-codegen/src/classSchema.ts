import type { VisualDSLDiagram } from './chariotCodeGenerator';

/** Stable palette labels; category key is `classes`, display name is CLASSES. */
export const CLASS_NODE_LABELS = ['Class', 'Field', 'Method', 'New', 'Interface', 'Requires',
  'Signature', 'WithSignature', 'Satisfies', 'AssertSatisfies'] as const;
export type ClassNodeLabel = typeof CLASS_NODE_LABELS[number];
export const CLASS_CONTRACT_CODES = ['V', 'N', 'S', 'L', 'D', 'A', 'M', 'F', 'O', 'J', 'X', 'T', 'P', 'E'] as const;
export type ClassContractCode = typeof CLASS_CONTRACT_CODES[number];

/** Expressions are verbatim Chariot, never quoted or parsed as identifiers. */
export type ClassExpression = string;
/** Same parameter/default-value rows used by the existing Function editor. */
export interface ClassFunctionParameter { name: string; value: string }
export interface ClassProperties { identifier: string }
export interface InterfaceProperties { identifier: string }
export interface FieldProperties {
  memberName: string;
  contract: ClassContractCode;
  hasDefault: boolean;
  defaultExpression: ClassExpression;
}
export interface MethodProperties {
  classExpression: ClassExpression;
  memberName: string;
  functionMode: 'editor' | 'expression';
  parameters: ClassFunctionParameter[];
  body: string;
  functionExpression: ClassExpression;
  hasSignature: boolean;
  /** Includes the receiver contract (normally leading "V"). */
  signatureExpression: ClassExpression;
}
export interface NewProperties {
  classExpression: ClassExpression;
  hasAttributes: boolean;
  attributesExpression: ClassExpression;
}
export interface RequiresProperties {
  memberName: string;
  /** Excludes the receiver contract. */
  signatureExpression: ClassExpression;
}
export interface SignatureProperties { parameterContracts: ClassContractCode[]; returnContract: ClassContractCode }
export interface WithSignatureProperties { functionExpression: ClassExpression; signatureExpression: ClassExpression }
export interface SatisfiesProperties { valueExpression: ClassExpression; interfaceExpression: ClassExpression }
export interface AssertSatisfiesProperties extends SatisfiesProperties {}
export interface ClassNodePropertiesMap {
  Class: ClassProperties;
  Field: FieldProperties;
  Method: MethodProperties;
  New: NewProperties;
  Interface: InterfaceProperties;
  Requires: RequiresProperties;
  Signature: SignatureProperties;
  WithSignature: WithSignatureProperties;
  Satisfies: SatisfiesProperties;
  AssertSatisfies: AssertSatisfiesProperties;
}

export type ClassPropertyKind = 'identifier' | 'memberName' | 'expression' | 'contract' |
  'toggle' | 'functionMode' | 'parameters' | 'body' | 'contracts';
export interface ClassPropertySchema {
  kind: ClassPropertyKind;
  /** Inactive fields remain persisted but are not emitted or semantically validated. */
  activeWhen?: { property: string; equals: boolean | string };
}
export interface ClassNodeSchema {
  category: 'classes';
  categoryLabel: 'CLASSES';
  functionName: string;
  bodyChildLabel?: 'Field' | 'Requires';
  properties: Record<string, ClassPropertySchema>;
  help: string;
}
const expression = { kind: 'expression' } as const;
const memberName = { kind: 'memberName' } as const;
const signatureExpression = expression;
const schema = (functionName: string, properties: ClassNodeSchema['properties'], help: string,
  bodyChildLabel?: ClassNodeSchema['bodyChildLabel']): ClassNodeSchema =>
  ({ category: 'classes', categoryLabel: 'CLASSES', functionName, properties, help, bodyChildLabel });

/** Dialog field semantics and declaration nesting contract, shared with the frontend. */
export const CLASS_NODE_SCHEMA: Readonly<Record<ClassNodeLabel, ClassNodeSchema>> = {
  Class: schema('class', { identifier: { kind: 'identifier' } }, 'Ordered Field children only.', 'Field'),
  Field: schema('field', { memberName, contract: { kind: 'contract' }, hasDefault: { kind: 'toggle' },
    defaultExpression: { kind: 'expression', activeWhen: { property: 'hasDefault', equals: true } } },
  'Only inside Class. No default means required; enabled null, "", and 0 are explicit defaults.'),
  Method: schema('method', { classExpression: expression, memberName, functionMode: { kind: 'functionMode' },
    parameters: { kind: 'parameters', activeWhen: { property: 'functionMode', equals: 'editor' } },
    body: { kind: 'body', activeWhen: { property: 'functionMode', equals: 'editor' } },
    functionExpression: { kind: 'expression', activeWhen: { property: 'functionMode', equals: 'expression' } },
    hasSignature: { kind: 'toggle' },
    signatureExpression: { kind: 'expression', activeWhen: { property: 'hasSignature', equals: true } } },
  'Register outside Class. Editor parameters start with self. Annotations include leading V for self. No nested inputs.'),
  New: schema('new', { classExpression: expression, hasAttributes: { kind: 'toggle' },
    attributesExpression: { kind: 'expression', activeWhen: { property: 'hasAttributes', equals: true } } },
  'Class is a raw expression. Attributes may use map(...) or mapValue(...).'),
  Interface: schema('interface', { identifier: { kind: 'identifier' } }, 'Ordered Requires children only.', 'Requires'),
  Requires: schema('requires', { memberName, signatureExpression },
    'Only inside Interface. Requirement signatures exclude the receiver.'),
  Signature: schema('signature', { parameterContracts: { kind: 'contracts' }, returnContract: { kind: 'contract' } },
    'Ordered parameter contracts; an empty list means zero parameters.'),
  WithSignature: schema('withSignature', { functionExpression: expression, signatureExpression },
    'Returns an annotated copy of the function; it does not mutate the original.'),
  Satisfies: schema('satisfies', { valueExpression: expression, interfaceExpression: expression },
    'Returns boolean. Runtime conformance currently supports script instances only.'),
  AssertSatisfies: schema('assertSatisfies', { valueExpression: expression, interfaceExpression: expression },
    'Returns the original instance or errors. Dynamic conformance is checked at runtime.'),
};

export function isClassNodeLabel(label: string): label is ClassNodeLabel {
  return (CLASS_NODE_LABELS as readonly string[]).includes(label);
}
export function isClassExpressionLabel(label: string): boolean {
  return ['New', 'Signature', 'WithSignature', 'Satisfies', 'AssertSatisfies'].includes(label);
}

/** Always returns fresh arrays/objects so insertion paths cannot share mutable defaults. */
export function getClassNodeDefaults<L extends ClassNodeLabel>(label: L): ClassNodePropertiesMap[L] {
  const defaults: ClassNodePropertiesMap = {
    Class: { identifier: 'Example' },
    Field: { memberName: 'value', contract: 'V', hasDefault: false, defaultExpression: 'null' },
    Method: { classExpression: 'Example', memberName: 'getValue', functionMode: 'editor',
      parameters: [{ name: 'self', value: '' }], body: 'return(self.value)', functionExpression: '',
      hasSignature: false, signatureExpression: 'signature(array("V"), "V")' },
    New: { classExpression: 'Example', hasAttributes: false, attributesExpression: 'map()' },
    Interface: { identifier: 'ExampleInterface' },
    Requires: { memberName: 'getValue', signatureExpression: 'signature(array(), "V")' },
    Signature: { parameterContracts: [], returnContract: 'V' },
    WithSignature: { functionExpression: 'func() { return(null) }', signatureExpression: 'signature(array(), "V")' },
    Satisfies: { valueExpression: 'instance', interfaceExpression: 'ExampleInterface' },
    AssertSatisfies: { valueExpression: 'instance', interfaceExpression: 'ExampleInterface' },
  };
  return defaults[label];
}

export interface ClassValidationIssue { nodeId?: string; property: string; message: string }
const identifierPattern = /^[A-Za-z_][A-Za-z0-9_]*$/;
const isRecord = (value: unknown): value is Record<string, unknown> =>
  value !== null && typeof value === 'object' && !Array.isArray(value);
const isContract = (value: unknown): boolean => (CLASS_CONTRACT_CODES as readonly unknown[]).includes(value);

/** No parser/typechecker claims: expressions are checked only for nonblank text. */
export function validateClassNodeProperties(label: ClassNodeLabel, properties: unknown): ClassValidationIssue[] {
  const issues: ClassValidationIssue[] = [];
  const fail = (property: string, message: string) => { issues.push({ property, message }); };
  if (!isRecord(properties)) return [{ property: 'properties', message: `${label} properties must be an object` }];
  for (const [key, field] of Object.entries(CLASS_NODE_SCHEMA[label].properties)) {
    const value = properties[key];
    const active = !field.activeWhen || properties[field.activeWhen.property] === field.activeWhen.equals;
    // Inactive persisted fields may be absent, but if present must retain the right shape.
    if (!active && value === undefined) continue;
    switch (field.kind) {
      case 'toggle':
        if (typeof value !== 'boolean') fail(key, 'Must be a boolean');
        break;
      case 'functionMode':
        if (value !== 'editor' && value !== 'expression') fail(key, 'Must be editor or expression');
        break;
      case 'contract':
        if (!isContract(value)) fail(key, `Expected contract ${CLASS_CONTRACT_CODES.join(' ')}`);
        break;
      case 'contracts':
        if (!Array.isArray(value)) fail(key, 'Must be an ordered contract array');
        else value.forEach((code, index) => { if (!isContract(code)) fail(`${key}.${index}`, 'Invalid contract'); });
        break;
      case 'parameters': {
        if (!Array.isArray(value)) { fail(key, 'Must be parameter rows {name, value}'); break; }
        const names = new Set<string>();
        value.forEach((row, index) => {
          if (!isRecord(row) || typeof row.name !== 'string' || typeof row.value !== 'string') {
            fail(`${key}.${index}`, 'Expected {name: string, value: string}');
          } else if (active) {
            if (!identifierPattern.test(row.name)) fail(`${key}.${index}.name`, 'Invalid parameter identifier');
            if (names.has(row.name)) fail(`${key}.${index}.name`, 'Duplicate parameter');
            names.add(row.name);
          }
        });
        if (active && (!isRecord(value[0]) || value[0].name !== 'self')) fail(key, 'First method parameter must be self');
        if (active && isRecord(value[0]) && value[0].value !== '') fail(`${key}.0.value`, 'self cannot have a default');
        break;
      }
      default:
        if (typeof value !== 'string') { fail(key, 'Must be a string'); break; }
        if (!active || field.kind === 'body') break;
        if (field.kind === 'identifier' && !identifierPattern.test(value)) fail(key, 'Expected an unquoted declaration identifier');
        if ((field.kind === 'expression' || field.kind === 'memberName') && !value.trim()) fail(key, 'Must not be blank');
    }
  }
  return issues;
}

/** Supported statement containers actually consumed by the canonical generator. */
const statementParents = new Set(['If', 'Then', 'Else', 'While', 'Loop Body', 'Case', 'Default']);
const assignmentParents = new Set(['Set Equal', 'Set Value', 'Set Q', 'setq', 'SetQ', 'set q', 'set equal', 'set value']);

/** Validate before generation AND before frontend Save/nesting changes. Edges are flow, not body membership. */
export function validateClassDiagram(diagram: VisualDSLDiagram): ClassValidationIssue[] {
  const issues: ClassValidationIssue[] = [];
  const nodes = new Map(diagram.nodes.map(node => [node.id, node]));
  const classNodes = diagram.nodes.filter(node => isClassNodeLabel(node.data.label));
  if (!classNodes.length) return issues; // Leave unrelated legacy hierarchy behavior unchanged.
  const fail = (nodeId: string, property: string, message: string) => issues.push({ nodeId, property, message });
  const parents = new Map<string, string>();
  const children = new Map<string, string[]>();
  for (const rel of diagram.nestingRelations) {
    if (!nodes.has(rel.parentId) || !nodes.has(rel.childId)) fail(rel.childId, 'nestingRelations', 'Missing nesting endpoint');
    if (!Number.isFinite(rel.order)) fail(rel.childId, 'nestingRelations.order', 'Order must be finite');
    if (parents.has(rel.childId)) fail(rel.childId, 'nestingRelations', 'A child must have exactly one nesting parent');
    parents.set(rel.childId, rel.parentId);
    children.set(rel.parentId, [...(children.get(rel.parentId) || []), rel.childId]);
  }
  const ids = new Set<string>();
  for (const node of diagram.nodes) {
    if (ids.has(node.id)) fail(node.id, 'id', 'Duplicate node ID');
    ids.add(node.id);
    const seen = new Set<string>();
    let current: string | undefined = node.id;
    while (current !== undefined) {
      if (seen.has(current)) { fail(node.id, 'nestingRelations', 'Nesting cycle'); break; }
      seen.add(current);
      current = parents.get(current);
    }
  }
  for (const node of classNodes) {
    const label = node.data.label as ClassNodeLabel;
    if (node.type === 'group') fail(node.id, 'type', 'Class constructs cannot be group nodes');
    issues.push(...validateClassNodeProperties(label, node.data.properties).map(issue => ({ ...issue, nodeId: node.id })));
    const parentId = parents.get(node.id);
    const parentLabel = parentId === undefined ? undefined : nodes.get(parentId)?.data.label;
    const bodyChild = CLASS_NODE_SCHEMA[label].bodyChildLabel;
    if (label === 'Field' || label === 'Requires') {
      const expected = label === 'Field' ? 'Class' : 'Interface';
      if (parentLabel !== expected) fail(node.id, 'nestingRelations', `${label} must be a direct child of ${expected}`);
    } else if (parentLabel !== undefined && !statementParents.has(parentLabel)) {
      if (!isClassExpressionLabel(label) || (!assignmentParents.has(parentLabel) && parentLabel !== 'Declare')) {
        fail(node.id, 'nestingRelations', `${label} is not supported inside ${parentLabel}`);
      } else if (parentLabel === 'Declare' && children.get(parentId!)?.length !== 1) {
        fail(node.id, 'nestingRelations', 'Declare requires exactly one expression child');
      }
    }
    const memberNames = new Set<string>();
    for (const childId of children.get(node.id) || []) {
      if (!bodyChild || nodes.get(childId)?.data.label !== bodyChild) {
        fail(childId, 'nestingRelations', bodyChild ? `${label} accepts only ${bodyChild} children` : `${label} uses property inputs, not nested nodes`);
      }
      const name = nodes.get(childId)?.data.properties?.memberName;
      if (bodyChild && typeof name === 'string') {
        if (memberNames.has(name)) fail(childId, 'memberName', 'Duplicate declaration member');
        memberNames.add(name);
      }
    }
    // An ignored legacy ancestor (e.g. Function) must not hide an otherwise valid class subtree.
    const seenAncestors = new Set<string>();
    let childId = node.id;
    let ancestor = parentId;
    while (ancestor !== undefined && !seenAncestors.has(ancestor)) {
      seenAncestors.add(ancestor);
      const outerLabel = nodes.get(ancestor)?.data.label ?? '';
      const childLabel = nodes.get(childId)?.data.label ?? '';
      const siblings = (children.get(ancestor) || []).slice().sort((a, b) => {
        const order = (id: string) => diagram.nestingRelations.find(rel => rel.parentId === ancestor && rel.childId === id)?.order ?? 0;
        return order(a) - order(b);
      });
      const firstWithLabel = (label: string) => siblings.find(id => nodes.get(id)?.data.label === label);
      let consumed = isClassNodeLabel(outerLabel) || assignmentParents.has(outerLabel);
      if (outerLabel === 'Declare') consumed = siblings.length === 1 && isClassExpressionLabel(childLabel);
      else if (outerLabel === 'If') {
        consumed = childLabel === 'Then' || childLabel === 'Else'
          ? childId === firstWithLabel(childLabel)
          : firstWithLabel('Then') === undefined;
      } else if (outerLabel === 'While') {
        consumed = childLabel === 'Loop Body' ? childId === firstWithLabel('Loop Body') : firstWithLabel('Loop Body') === undefined;
      } else if (outerLabel === 'Switch') {
        consumed = childLabel === 'Case' || (childLabel === 'Default' && childId === firstWithLabel('Default'));
      } else if (statementParents.has(outerLabel)) consumed = true;
      if (!consumed) fail(node.id, 'nestingRelations', `Unsupported or ignored nesting inside ${outerLabel}`);
      // Branch markers have no standalone generator; they need the owning structure.
      const owner: Record<string, string> = { Then: 'If', Else: 'If', 'Loop Body': 'While', Case: 'Switch', Default: 'Switch' };
      if (owner[outerLabel] && nodes.get(parents.get(ancestor) ?? '')?.data.label !== owner[outerLabel]) {
        fail(node.id, 'nestingRelations', `${outerLabel} must belong to ${owner[outerLabel]}`);
      }
      childId = ancestor;
      ancestor = parents.get(ancestor);
    }
  }
  return issues;
}