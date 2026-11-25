# Browser editor drafts and paste

Charioteer restores its workspace when the shared landing page replaces the iframe,
and on refresh. No landing or Visual DSL implementation changes are required.

## Persisted scope

- File, Function Library, and generated diagram-code tabs: code (including empty
  unsaved deletions), saved baseline, dirty state, names, per-tab scope, tab order,
  active tab, last tab for each code workspace, Monaco selection/cursor/scroll.
- Standalone code before opening a tab, including its selection/cursor/scroll.
- Pinned reference tab and its viewport, selected toolbar and bottom tab,
  file/diagram browsing scopes, stream-logs toggle, left/bottom panel sizes.
- Agent plan-code textarea. Its draft is restored when the dynamic form appears.
- Runtime/output/problems **viewport offsets only**, not data or HTML.

The allowlist deliberately excludes auth tokens, passwords, MCP URLs/commands/
arguments/environment/headers, arbitrary MCP payloads, agent variable maps,
listener connection forms, runtime trees, log/output contents, responses and live
execution/debug sessions. These may contain credentials or stale server state.
Existing server-backed fixtures/settings are unchanged. Generated diagram **code**
and source name are restored; diagram JSON is reloaded from the server when needed.
Undo history, runtime tree expansion and transient modal forms are not restored.
Code itself can contain secrets: drafts are plain browser-local data, not an
encrypted vault. Do not embed credentials in code on shared machines.

Storage key: `charioteer.editor.draft.v1:` followed by JSON-encoded username;
anonymous drafts use `null`. The existing login username (`chariot_user`) is the
available stable frontend identity; tokens are never used as keys or draft values.
Tabs retain their individual global/sandbox/listeners/bootstrap/function scopes.
Namespaces prevent accidental cross-account restoration, not access by another
person with the same browser profile or origin access. Anonymous drafts are shared
within that browser profile/origin. Concurrent browser tabs are last-writer-wins.

Restoration completes before the writer is enabled; server profile defaults cannot
reset restored scopes. New creates a separate tab and retains other drafts. Opening
an existing file uses its open tab; explicit diagram generation/import updates its
normal target tab (there is no delayed background restore to resurrect older code).
Explicit close/discard removes the tab from the next snapshot. Logout saves before
clearing and switches namespaces; login restores that username's draft before API
list refreshes. No execution or saving to the server is replayed.

Writes are debounced 150ms; pagehide and hidden-document transitions flush
synchronously for iframe replacement. Corrupt/unsupported drafts are left
byte-for-byte intact and autosave is paused, with a persistent warning. Storage
denial/quota failures warn and retain the previous successful snapshot; later writes
retry. Export/copy current code before leaving on any warning. Recover/remove an
invalid draft through browser storage tools after copying its original bytes.
Browser eviction, crashes before the debounce flush, or disabled storage cannot
guarantee recovery. No localStorage mechanism can guarantee durability on disk.

## Paste ownership

There is no CodeMirror and no custom native paste listener in Charioteer.
Monaco 0.45 owns native keyboard paste. Its original menu command first attempts
`document.execCommand('paste')`, then calls `navigator.clipboard.readText()` when
the host reports false. If a host dispatches paste but returns false, a delayed
clipboard read inserts the text again. An actual Chrome run against the original
template, injecting that host return/event behavior and a deferred read, produced
`PASTEPASTE` from **one** paste event. Normal native keyboard paste already inserted
once; the problematic host return was simulated, not observed spontaneously.

Charioteer now registers the canonical menu command with **one** Clipboard API read
and **one** Monaco paste operation. It does not call execCommand, intercept keyboard
paste, deduplicate text, or bind Ctrl/Cmd+V. If permission is denied, a warning asks
for native keyboard paste (which does not require readText). In-flight menu paste
is cancelled if the model, version or selection changes. Read-only reference panes
cannot be pasted into. Rebuilding Monaco does not stack toolbar/keyboard handlers.

## Tests

From the Charioteer service directory:

- `go test ./...`
- `node --test editor_state.test.cjs`

Optional real-browser suite: install Playwright outside the repository with
`npm install --prefix /tmp/charioteer-browser-tests playwright`. Start Charioteer
with `go run . -port 18080`. Then run
`CHARIOTEER_PLAYWRIGHT=/tmp/charioteer-browser-tests/node_modules/playwright CHARIOTEER_CHROME='/Applications/Google Chrome.app/Contents/MacOS/Google Chrome' node --test editor_browser.test.cjs`.
On other platforms set `CHARIOTEER_CHROME` to a Chrome executable, or omit it and
install Playwright Chromium. `CHARIOTEER_TEST_URL` overrides the base URL.

The suite uses the actual rendered template and CDN-pinned Monaco, real OS clipboard
keyboard paste, deferred menu reads, denied readText fallback, a same-origin iframe
src replacement, refresh, per-user login/logout, New, dirty deletions, view state,
layout, quota and corrupt draft protection. APIs are mocked; this is not a live
backend execution test. Monaco CDN access is required.

Known unrelated UI limitation: the existing Files toolbar can overlap New with
the file dropdown. The handler-count regression uses a DOM button click rather
than claiming pointer-click layout coverage. Browser validation was Chrome-only;
Firefox/Safari and live backend execution were not exercised. Explicit diagram
import/generation semantics were inspected but not separately browser-tested.
