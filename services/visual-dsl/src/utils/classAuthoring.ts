import {
  getClassNodeDefaults, isClassNodeLabel, validateClassDiagram,
  type ClassNodeLabel, type ClassNodePropertiesMap, type VisualDSLDiagram,
} from 'chariot-codegen';

export type ClassProperties = ClassNodePropertiesMap[ClassNodeLabel];

/** One constructor for flow, random, and nested palette insertion. Never repair imports. */
export function createLogiconData(label: string, icon: string, category: string, legacyProperties: object = {}) {
  return { label, icon, category, properties: isClassNodeLabel(label) ? getClassNodeDefaults(label) : legacyProperties };
}

export function declarationPlacementMessage(label: string): string | undefined {
  const parent = label === 'Field' ? 'Class' : label === 'Requires' ? 'Interface' : undefined;
  return parent && `Select a ${parent}, enable Nesting Mode, then add ${label}; or drag a connector from ${parent} to an imported ${label}.`;
}

/** Reject new structural errors, without blocking repair of unrelated imported drafts. */
export function classNestingError(diagram: VisualDSLDiagram, parentId: string, childId: string): string | undefined {
  const parent = diagram.nodes.find(node => node.id === parentId);
  const child = diagram.nodes.find(node => node.id === childId);
  if (!parent || !child) return 'Missing nesting endpoint';
  if (parent.data.label === 'Source' || child.data.label === 'Source') return 'Source nodes are read-only and cannot be nested.';
  const baseline = new Set(validateClassDiagram(diagram)
    .filter(issue => issue.property.startsWith('nestingRelations') || issue.message === 'Duplicate declaration member')
    .map(issue => JSON.stringify(issue)));
  const remaining = diagram.nestingRelations.filter(rel => rel.childId !== childId);
  const candidate = { ...diagram, nestingRelations: [...remaining, {
    parentId, childId, order: Math.max(-1, ...remaining.filter(rel => rel.parentId === parentId).map(rel => rel.order)) + 1,
  }] };
  return validateClassDiagram(candidate).find(issue =>
    (issue.property.startsWith('nestingRelations') || issue.message === 'Duplicate declaration member') &&
    (issue.nodeId === childId || !baseline.has(JSON.stringify(issue))))?.message;
}