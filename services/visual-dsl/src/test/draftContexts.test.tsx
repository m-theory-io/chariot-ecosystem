import assert from 'node:assert/strict';
import { test } from 'node:test';
import React from 'react';
import { renderToStaticMarkup } from 'react-dom/server';
import { FlowControlProvider, useFlowControl } from '../contexts/FlowControlContext';
import { NestingProvider, useNesting } from '../contexts/NestingContext';
import type { DiagramDraft } from '../utils/diagramDraft';

test('both React contexts expose restored state on the very first render, not a later effect', () => {
  const draft: DiagramDraft = {
    version: 1,
    diagram: {
      name: 'Restored', nodes: [], edges: [], groupCount: 1, created: 'then', modified: 'now',
      nestingRelations: [{ parentId: 'parent', childId: 'child', order: 2 }],
      subflows: { 'group-parent': { id: 'group-parent', name: 'Custom', collapsed: true, inputPorts: [], outputPorts: [] } },
    },
    editor: { direction: 'up', selectedNodeId: 'child', selectedParentId: 'parent', nestingMode: true, viewport: { x: 0, y: 0, zoom: 1 } },
  };
  let renders = 0;
  function Probe() {
    const flow = useFlowControl();
    const nesting = useNesting();
    assert.equal(flow.direction, 'up');
    assert.equal(flow.selectedNodeId, 'child');
    assert.equal(nesting.selectedParentId, 'parent');
    assert.equal(nesting.nestingMode, true);
    assert.deepEqual(nesting.nestingRelations, draft.diagram.nestingRelations);
    assert.deepEqual(nesting.getAllSubflows(), draft.diagram.subflows);
    renders++;
    return null;
  }
  renderToStaticMarkup(<FlowControlProvider initialDraft={draft}><NestingProvider initialDraft={draft}><Probe /></NestingProvider></FlowControlProvider>);
  assert.equal(renders, 1);
});