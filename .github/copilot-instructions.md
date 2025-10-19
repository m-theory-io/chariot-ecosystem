# Team Guidelines (Tech Lead)

- When the project/monorepo already contains the related code, **do not** generate adapter shims to “paper over” wrong function signatures or env var names.
- **Fix root causes instead.** If a name/signature is wrong:
  - Identify the canonical API/env var.
  - Propose a minimal refactor plan and patch for the offending code.
  - Update tests and docs accordingly.
- Prefer edits that keep interfaces consistent across packages; avoid one-off wrappers.
- If a breaking change is necessary, include a migration note and search/replace hints.

# Response style
- Provide a short rationale first, then the exact diffs/commands.
- If unsure, ask for the relevant file path(s) before guessing.
