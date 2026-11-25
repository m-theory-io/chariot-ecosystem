import { useLayoutEffect, useState } from 'react';
import { createDraftWriter } from '../utils/diagramDraft';
import type { DiagramDraft } from '../utils/diagramDraft';

export function useDiagramDraft(draft: DiagramDraft): boolean {
  const [write] = useState(() => createDraftWriter(draft));
  const [saved, setSaved] = useState(true);
  useLayoutEffect(() => {
    setSaved(write(draft));
  }, [draft, write]);
  return saved;
}