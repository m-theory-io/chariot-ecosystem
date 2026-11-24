import type { Node, Edge, Viewport } from 'reactflow';
import type { Subflow, NestingRelation } from '../contexts/NestingContext';

// Browser/origin-local working copy, separate from named saves and server files.
// No stable authenticated principal is exposed to this frontend. Never key by or
// copy chariot_auth_token (a credential) into the draft.
export const DIAGRAM_DRAFT_KEY = 'chariot.visual-dsl.draft.v1';

export interface DiagramData {
  name: string;
  nodes: Node[];
  edges: Edge[];
  nestingRelations: NestingRelation[];
  subflows?: Record<string, Subflow>;
  groupCount: number;
  created: string;
  modified: string;
}

export interface DraftEditorState {
  selectedNodeId: string | null;
  selectedParentId: string | null;
  nestingMode: boolean;
  direction: 'down' | 'right' | 'up';
  viewport: Viewport;
}

export interface DiagramDraft {
  version: 1;
  diagram: DiagramData;
  editor: DraftEditorState;
}

type StorageAccess = () => Pick<Storage, 'getItem' | 'setItem'>;
const browserStorage: StorageAccess = () => window.localStorage;
const object = (value: unknown): value is Record<string, any> =>
  !!value && typeof value === 'object' && !Array.isArray(value);
const point = (value: unknown): boolean =>
  object(value) && Number.isFinite(value.x) && Number.isFinite(value.y);

// Reject malformed/unknown versions as a whole, rather than partially restoring
// a graph that could silently lose executable Source AST or nesting metadata.
export function parseDiagramDraft(raw: string): DiagramDraft {
  const draft = JSON.parse(raw);
  if (!object(draft) || draft.version !== 1 || !object(draft.diagram) || !object(draft.editor)) {
    throw new Error('Unsupported draft version or shape');
  }
  const d = draft.diagram;
  if (typeof d.name !== 'string' || !Array.isArray(d.nodes) || !Array.isArray(d.edges) ||
      !Array.isArray(d.nestingRelations) || !Number.isInteger(d.groupCount) || d.groupCount < 0 ||
      typeof d.created !== 'string' || typeof d.modified !== 'string') throw new Error('Invalid diagram');
  const ids = new Set<string>();
  for (const node of d.nodes) {
    if (!object(node) || typeof node.id !== 'string' || !node.id || ids.has(node.id) ||
        !point(node.position) || !object(node.data)) throw new Error('Invalid draft node');
    ids.add(node.id);
  }
  const edgeIds = new Set<string>();
  for (const edge of d.edges) {
    if (!object(edge) || typeof edge.id !== 'string' || edgeIds.has(edge.id) ||
        !ids.has(edge.source) || !ids.has(edge.target)) throw new Error('Invalid draft edge');
    edgeIds.add(edge.id);
  }
  for (const relation of d.nestingRelations) {
    if (!object(relation) || !ids.has(relation.parentId) || !ids.has(relation.childId) ||
        !Number.isFinite(relation.order)) throw new Error('Invalid draft nesting');
  }
  if (d.subflows !== undefined) {
    if (!object(d.subflows)) throw new Error('Invalid subflows');
    for (const sf of Object.values(d.subflows)) {
      if (!object(sf) || typeof sf.id !== 'string' || typeof sf.name !== 'string' ||
          typeof sf.collapsed !== 'boolean' || !Array.isArray(sf.inputPorts) || !Array.isArray(sf.outputPorts) ||
          ![...sf.inputPorts, ...sf.outputPorts].every(p => object(p) && typeof p.id === 'string' && typeof p.label === 'string')) {
        throw new Error('Invalid subflow');
      }
    }
  }
  const e = draft.editor;
  if (![null, ...ids].includes(e.selectedNodeId) || ![null, ...ids].includes(e.selectedParentId) ||
      typeof e.nestingMode !== 'boolean' || !['down', 'right', 'up'].includes(e.direction) ||
      !point(e.viewport) || !Number.isFinite(e.viewport.zoom) || e.viewport.zoom <= 0) throw new Error('Invalid draft editor state');
  return draft as DiagramDraft;
}

export function readDiagramDraft(storage: StorageAccess = browserStorage): DiagramDraft | undefined {
  try {
    const raw = storage().getItem(DIAGRAM_DRAFT_KEY);
    return raw === null ? undefined : parseDiagramDraft(raw);
  } catch {
    // Corrupt/old drafts and denied storage must never prevent authoring. Do not
    // delete the original: an untouched initial mount must not overwrite it.
    return undefined;
  }
}

export function serializeDiagramDraft(draft: DiagramDraft): string {
  return JSON.stringify({
    ...draft,
    diagram: {
      ...draft.diagram,
      // React Flow runtime state is not authoring data. Keep positions, sizes,
      // selection, parent/extent, style, handles, properties and full Source AST.
      nodes: draft.diagram.nodes.map(({ dragging: _dragging, ...node }) => node),
    },
  });
}

// One writer per mounted editor. The baseline is already restored synchronously
// by App, including both contexts. Repeated mount effects (StrictMode) are no-ops.
// No timer/unload handler: every committed edit is durable before navigation.
export function createDraftWriter(initial: DiagramDraft, storage: StorageAccess = browserStorage) {
  let previous = serializeDiagramDraft(initial);
  return (draft: DiagramDraft): boolean => {
    try {
      const serialized = serializeDiagramDraft(draft);
      if (serialized === previous) return true;
      storage().setItem(DIAGRAM_DRAFT_KEY, serializeDiagramDraft({
        ...draft,
        diagram: { ...draft.diagram, modified: new Date().toISOString() },
      }));
      previous = serialized;
      return true;
    } catch {
      // Quota/security failures leave the previous durable draft intact and can
      // be retried on the next edit. The caller displays an export reminder.
      return false;
    }
  };
}