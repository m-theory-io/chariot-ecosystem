import { deserializeChariotAST } from 'chariot-codegen';
import type { VisualDSLDiagram } from 'chariot-codegen';

// AST imports use the same lossless Source model as codegen; never project
// unfamiliar nodes into existing property dialogs or discard them as empty flows.
export function deserializeDiagramInput(value: unknown): VisualDSLDiagram {
  if (!value || typeof value !== 'object' || Array.isArray(value)) {
    throw new Error('Expected a Visual DSL diagram or serialized Chariot AST');
  }
  if ('_node_type' in value) return deserializeChariotAST(value);
  if (!('nodes' in value) || !Array.isArray(value.nodes) ||
      !('edges' in value) || !Array.isArray(value.edges)) {
    throw new Error('Expected diagram nodes/edges or an AST _node_type');
  }
  // Return existing diagrams unchanged, including optional persistence metadata.
  return value as VisualDSLDiagram;
}