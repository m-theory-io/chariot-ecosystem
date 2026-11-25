# Browser working drafts

Visual DSL automatically restores its latest working copy before mounting Canvas
and its flow/nesting contexts. Committed edits are saved synchronously to
`chariot.visual-dsl.draft.v1` in localStorage, independently of named Save/Export.
This covers landing-page app switches (which navigate a single iframe), reloads,
and opening Visual DSL directly on the **same browser origin**.

The versioned draft contains nodes (including properties/Source AST, positions,
selection, parent/extent and styles), edges/handles, nesting order, subflow
metadata, group count, name, creation/modification timestamps, flow direction,
selected parent/node, nesting mode and viewport. It excludes transient dragging
state, modal state and auth credentials. Do not put secrets into diagram literals;
drafts are local browser data, not encrypted backups.

There is no stable authenticated user identity exposed to this frontend. Drafts
are therefore browser/origin-local, **not account-isolated**; shared browser
profiles share the working copy. Different schemes, hosts and ports do not share
storage. Concurrent tabs are last-writer-wins. Export important work separately.

New intentionally replaces the working copy with an empty diagram and clears
subflows/selections. Imports replace the working copy through the existing import
path. Corrupt/unknown draft versions are ignored; the reader never deletes them
or overwrites them with default state. Subsequent authoring replaces that draft.
Blocked/quota-exceeded writes show an export reminder and preserve the last
successful durable copy. Legacy named saves (`diagram_*`) remain unchanged;
they are not guessed as the current working draft.

## Validation

From this directory, run `npx --yes tsx --test src/test/*.test.ts src/test/*.test.tsx`
and `npm run build`. `npx tsc --noEmit` currently has unrelated baseline errors
in two transform dialogs (`InputProps.list`) and Canvas's branch helper
(`logicon`/`defaultProperties`).

Browser regression:

1. Open the stack's landing address with `/#visual-dsl`, author/connect nodes,
   rename, move nodes, change flow direction and optionally create a subflow.
2. Generate code, copy it, switch to Charioteer, execute, then switch back.
   The diagram should remain editable with the same positions and metadata.
3. Reload the landing page and open `/visual-dsl/` directly on that same origin.
4. Confirm New, reload: the diagram must stay empty, not resurrect old work.
5. Import another diagram, reload: the imported diagram is now the working copy.

For local frontend-only checks use the Vite address (normally
`http://localhost:5174/`), consistently. Do not alternate between `localhost`,
`127.0.0.1`, a LAN IP, HTTPS or another port and expect the same browser draft.
Use a disposable origin/profile when testing corrupt storage or clearing drafts.
Frontend-only checks do not verify backend execution or clipboard permissions.