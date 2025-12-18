# Third-Party Attributions

This document records the third-party libraries that ship with or are required to build the Chariot ecosystem. It is intended to satisfy attribution requirements for the licenses listed below and to provide engineers with a quick reference to the external dependencies in use.

## How to keep this page current

- **Visual DSL (services/visual-dsl)** – run `cd services/visual-dsl && npm ls --depth=0` to confirm runtime packages and `npm ls --dev --depth=0` for build-time dependencies. Update the tables if versions change.
- **Shared code generator (packages/chariot-codegen)** – run `cd packages/chariot-codegen && npm ls --dev --depth=0` after dependency bumps.
- **Go services** – run `cd services/go-chariot && go list -m all` (and the same for other modules in `go.work`) to capture new direct dependencies. Reference upstream LICENSE files whenever new modules are introduced.
- When adding any dependency, ensure its license is compatible with our distribution model, add it to the appropriate table below, and keep a link to the upstream project for traceability.

---

## Visual DSL (services/visual-dsl)

### Runtime dependencies

| Library | Version | License | Source / Purpose |
| --- | --- | --- | --- |
| react | ^19.1.0 | MIT | https://github.com/facebook/react – UI component model.
| react-dom | ^19.1.0 | MIT | https://github.com/facebook/react – DOM renderer for React components.
| reactflow | ^11.11.4 | MIT | https://github.com/wbkd/react-flow – Canvas/graph editor powering the Visual DSL builder.
| chariot-codegen | workspace link | Project-internal | Local package exported from `packages/chariot-codegen`; emits `.ch` DSL.

### Build and development dependencies

| Library | Version | License | Source / Purpose |
| --- | --- | --- | --- |
| @eslint/js | ^9.30.1 | MIT | ESLint core configuration helpers.
| @types/node | ^24.2.0 | MIT | TypeScript declarations for Node.js APIs.
| @types/react | ^19.1.8 | MIT | TypeScript declarations for React.
| @types/react-dom | ^19.1.6 | MIT | TypeScript declarations for React DOM.
| @vitejs/plugin-react | ^4.6.0 | MIT | Vite plugin enabling fast React refresh/JSX transforms.
| autoprefixer | ^10.4.21 | MIT | PostCSS plugin generating vendor prefixes.
| eslint | ^9.30.1 | MIT | Linting engine for the Visual DSL UI.
| eslint-plugin-react-hooks | ^5.2.0 | MIT | Enforces the React Hooks rules.
| eslint-plugin-react-refresh | ^0.4.20 | MIT | Warns about misuses of React Fast Refresh semantics.
| globals | ^16.3.0 | MIT | Common global identifiers list for ESLint configs.
| postcss | ^8.5.6 | MIT | CSS transformer used by Tailwind.
| tailwindcss | ^3.4.3 | MIT | Utility-first CSS framework for styling the Visual DSL interface.
| typescript | ^5.8.3 | Apache-2.0 | TypeScript compiler for the UI source.
| vite | ^7.0.4 | MIT | Dev/build tooling for the Visual DSL frontend.

---

## Shared code generator (packages/chariot-codegen)

| Library | Version | License | Source / Purpose |
| --- | --- | --- | --- |
| tsup | ^8.0.1 | MIT | Bundles the reusable code generator into ESM + IIFE formats.
| typescript | ^5.4.0 | Apache-2.0 | TypeScript compiler for the shared package.
| rimraf | ^5.0.5 | ISC | Cross-platform `rm -rf` used during clean builds.

*(This package has no third-party runtime dependencies; it is bundled before publishing to the workspace.)*

---

## Go Runtime (services/go-chariot)

| Module | Version | License (per upstream) | Source / Purpose |
| --- | --- | --- | --- |
| github.com/Azure/azure-sdk-for-go/sdk/azidentity | v1.11.0 | MIT | Azure Identity client for credential discovery.
| github.com/Azure/azure-sdk-for-go/sdk/keyvault/azsecrets | v0.12.0 | MIT | Accesses Key Vault secrets for secure config.
| github.com/bhouse1273/kissflag | v0.0.0-20230516… | MIT | Lightweight CLI flag parser maintained by the team.
| github.com/couchbase/gocb/v2 | v2.10.1 | Apache-2.0 | Couchbase Go SDK for database access.
| github.com/go-sql-driver/mysql | v1.9.3 | MPL-2.0 | MySQL driver used for metadata stores.
| github.com/google/uuid | v1.6.0 | BSD-3-Clause | Generates RFC 4122 identifiers.
| github.com/gorilla/websocket | v1.5.3 | BSD-2-Clause | WebSocket protocol implementation for streaming interfaces.
| github.com/labstack/echo/v4 | v4.13.4 | MIT | HTTP routing/web framework for APIs.
| github.com/lib/pq | v1.10.9 | MIT | PostgreSQL driver for optional storage backends.
| github.com/modelcontextprotocol/go-sdk | v1.0.1-0… | Apache-2.0 | Model Context Protocol bindings.
| go.uber.org/zap | v1.27.0 | MIT | Structured logging.
| golang.org/x/crypto | v0.41.0 | BSD-3-Clause | Supplemental crypto primitives.
| gopkg.in/yaml.v3 | v3.0.1 | MIT | YAML parser for configuration files.

> **Transitive dependencies:** Additional modules pulled in via `go list -m all` (e.g., `golang.org/x/net`, `google.golang.org/grpc`, OpenTelemetry packages, etc.) inherit their upstream licenses. When distributing binaries, keep a copy of each transitive license in your release artifacts if required by that license.

---

## WebSocket Relay (services/charioteer)

| Module | Version | License | Source / Purpose |
| --- | --- | --- | --- |
| github.com/gorilla/websocket | v1.5.3 | BSD-2-Clause | Powers the WebSocket bridge between Visual DSL clients and the runtime.

---

## Additional notices

- This file summarizes direct dependencies. Always review upstream LICENSE files before releasing binaries/images to ensure compliance with notice, attribution, or redistribution clauses.
- When introducing a dependency under GPL/AGPL or other copyleft terms, consult legal/compliance before merging.
- For container images, consider exporting the dependency list alongside SBOM metadata so downstream consumers can automate license scans.
