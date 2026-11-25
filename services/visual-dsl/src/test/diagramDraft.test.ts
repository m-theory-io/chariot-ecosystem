import assert from 'node:assert/strict';
import { test } from 'node:test';
import { createDraftWriter, DIAGRAM_DRAFT_KEY, parseDiagramDraft, readDiagramDraft, serializeDiagramDraft } from '../utils/diagramDraft';
import type { DiagramDraft } from '../utils/diagramDraft';

function fixture(): DiagramDraft {
  return {
    version: 1,
    diagram: {
      name: 'Iterative authoring', created: '2026-01-01', modified: '2026-01-02', groupCount: 1,
      nodes: [
        { id: 'start', type: 'logicon', position: { x: 17, y: -30 }, selected: true, data: { label: 'Start', properties: { name: 'Rule' } } },
        { id: 'source', type: 'logicon', position: { x: 7, y: 12 }, parentNode: 'start', extent: 'parent', style: { width: 300 }, data: { label: 'Source', properties: { ast: { _node_type: 'FutureNode', payload: [1, 'keep'] } } } },
      ],
      edges: [{ id: 'e', source: 'start', target: 'source', sourceHandle: 'out', targetHandle: 'in', selected: true }],
      nestingRelations: [{ parentId: 'start', childId: 'source', order: 0 }],
      subflows: { 'group-start': { id: 'group-start', name: 'Saved subflow', collapsed: true, inputPorts: [{ id: 'in', label: 'In' }], outputPorts: [] } },
    },
    editor: { selectedNodeId: 'start', selectedParentId: 'start', nestingMode: true, direction: 'right', viewport: { x: 10, y: -20, zoom: 0.7 } },
  };
}

function memoryStorage(raw: string | null = null) {
  let writes = 0;
  return {
    getItem(key: string) { assert.equal(key, DIAGRAM_DRAFT_KEY); return raw; },
    setItem(key: string, value: string) { assert.equal(key, DIAGRAM_DRAFT_KEY); writes++; raw = value; },
    get writes() { return writes; },
  };
}

test('canonical draft roundtrip retains graph, AST, positions, selection, grouping, name and viewport', () => {
  const draft = fixture();
  draft.diagram.nodes[0].dragging = true;
  const restored = parseDiagramDraft(serializeDiagramDraft(draft));
  delete draft.diagram.nodes[0].dragging;
  assert.deepEqual(restored, draft);
});

test('iframe navigation/remount restores latest edit immediately; initial and StrictMode mounts do not overwrite', () => {
  const original = fixture();
  const storage = memoryStorage(serializeDiagramDraft(original));
  const restored = readDiagramDraft(() => storage)!;
  const write = createDraftWriter(restored, () => storage);
  write(restored);
  write(restored);
  assert.equal(storage.writes, 0);
  const edited = structuredClone(restored);
  edited.diagram.nodes[0].position.x = 999;
  edited.diagram.name = 'Edited';
  assert.equal(write(edited), true);
  const remounted = readDiagramDraft(() => storage)!;
  assert.equal(remounted.diagram.nodes[0].position.x, 999);
  assert.equal(remounted.diagram.name, 'Edited');
  assert.deepEqual(remounted.editor, edited.editor);
  assert.equal(remounted.diagram.created, original.diagram.created);
  assert.notEqual(remounted.diagram.modified, original.diagram.modified);
  createDraftWriter(remounted, () => storage)(remounted);
  assert.equal(storage.writes, 1);
});

test('intentional New/clear remains empty on reload; import replaces that working copy', () => {
  const initial = fixture();
  const storage = memoryStorage(serializeDiagramDraft(initial));
  const write = createDraftWriter(initial, () => storage);
  const empty: DiagramDraft = {
    ...initial,
    diagram: { ...initial.diagram, name: 'Untitled Diagram', nodes: [], edges: [], nestingRelations: [], subflows: {}, groupCount: 0 },
    editor: { ...initial.editor, selectedNodeId: null, selectedParentId: null, nestingMode: false },
  };
  write(empty);
  const reloaded = readDiagramDraft(() => storage)!;
  assert.deepEqual(reloaded.diagram.nodes, []);
  assert.deepEqual(reloaded.diagram.subflows, {});
  const imported = fixture();
  imported.diagram.name = 'Imported';
  createDraftWriter(reloaded, () => storage)(imported);
  assert.equal(readDiagramDraft(() => storage)!.diagram.name, 'Imported');
});

test('corrupt, unknown-version and malformed drafts are ignored without an initial destructive write', () => {
  const badNode = fixture();
  badNode.diagram.nodes[0].position.x = Infinity;
  const badEdge = fixture();
  badEdge.diagram.edges[0].target = 'missing';
  for (const raw of ['{', '{}', JSON.stringify({ ...fixture(), version: 0 }), JSON.stringify(badNode), JSON.stringify(badEdge)]) {
    const storage = memoryStorage(raw);
    assert.equal(readDiagramDraft(() => storage), undefined);
    const fresh = fixture();
    createDraftWriter(fresh, () => storage)(fresh);
    assert.equal(storage.writes, 0);
    assert.equal(storage.getItem(DIAGRAM_DRAFT_KEY), raw);
  }
});

test('denied storage/quota does not crash and failed writes can retry without deleting prior draft', () => {
  assert.equal(readDiagramDraft(() => { throw new Error('denied'); }), undefined);
  const storage = memoryStorage(serializeDiagramDraft(fixture()));
  let denied = true;
  const write = createDraftWriter(fixture(), () => {
    if (denied) throw new Error('quota');
    return storage;
  });
  const edit = fixture();
  edit.diagram.name = 'Retry';
  assert.equal(write(edit), false);
  assert.equal(readDiagramDraft(() => storage)!.diagram.name, 'Iterative authoring');
  denied = false;
  assert.equal(write(edit), true);
  assert.equal(readDiagramDraft(() => storage)!.diagram.name, 'Retry');
});