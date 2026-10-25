# Containerizing a Chariot Agent

Production is a **bootstrap-driven application**, not a running Listener Registry
entry. The application host creates one runtime, loads its function library, and
executes `CHARIOT_BOOTSTRAP` once. Bootstrap creates all HTTP/NSQ listeners.
Dev REST and network MCP, when enabled, reuse that initialized runtime; sessions
inherit its state without executing listener-producing bootstrap again.

Start with the complete [listener examples](../examples/listeners/README.md).
They include two HTTP services, an NSQ consumer, an HTTP-only alternative, and a
counter visible in HTTP responses to detect duplicate bootstrap execution.

## Configuration and actual path resolution

| Setting | Default / behavior |
|---|---|
| `CHARIOT_HEADLESS` | `false`; set `true` for an application without Dev REST |
| `CHARIOT_DEV_REST_ENABLED` | `true`; explicitly set `false` in production |
| `CHARIOT_PORT` | **8087** in the server; examples explicitly use 8090 for Dev REST |
| `CHARIOT_SSL` | `false`; configure certs if using server TLS, or terminate at a gateway |
| `CHARIOT_DATA_PATH` | `./data`, normalized relative to the process working directory |
| `CHARIOT_BOOTSTRAP` | `bootstrap.ch`, resolved under DataPath |
| `CHARIOT_TREE_PATH` | `./data/trees`, independently normalized relative to working directory |
| `CHARIOT_FUNCTION_LIB` | `stlib.json`, joined to TreePath (not DataPath) |
| `CHARIOT_LISTENERS_FILE` | `listeners.json` under DataPath; DevTest configuration only |
| `CHARIOT_LISTENER_SCRIPTS_DIR` | `listeners`, joined to DataPath unless absolute |
| `CHARIOT_SHUTDOWN_TIMEOUT` | 30 seconds; graceful host shutdown budget |
| `CHARIOT_MCP_ENABLED` | `false`; network transport requires Dev REST enabled |
| `CHARIOT_MCP_EXECUTE_ENABLED` | `false`; leave disabled except trusted local development |
| `CHARIOT_NSQ_ADDR` | `localhost:4150`; reachable NSQD TCP address, not its HTTP port |

The unprefixed `DATA_PATH` variable is **not** the server setting. Setting
DataPath does not relocate TreePath, DiagramPath, CertPath, or the secret file;
set their canonical variables explicitly as needed. Paths beginning with `~`
are expanded for normalized data/tree/diagram/cert roots. Bootstrap is resolved
by the data-file resolver: use a relative filename under DataPath, or an absolute
path validated against that root; do not prepend `data/` to a filename when
DataPath already ends there. Do not rely on the resolver as a security boundary.

The function library must exist: ship the real exported library, or the example's
empty JSON object when every handler is defined inline. Secret-provider setup
also runs before bootstrap. For secret-free local examples, use the file provider
with the supplied empty JSON object; do not embed real secrets in an image.

`CHARIOT_ON_START_FILE` and `CHARIOT_AGENT_FILE` do not load application code or
agent trees in the server. `CHARIOT_BOOTSTRAP_FILE` is a legacy ignored variable.
Use `CHARIOT_BOOTSTRAP`. If an application needs an agent tree, its bootstrap must
explicitly load it using the appropriate tree function/configured TreePath; an
environment variable named for an agent file does not do that automatically.

## Production and debug profiles

| Profile | Required choices | Publish to host |
|---|---|---|
| Production | `CHARIOT_HEADLESS=true`, `CHARIOT_DEV_REST_ENABLED=false` | Only application HTTP ports (18080/18081 in examples) |
| Debug | `CHARIOT_HEADLESS=true`, `CHARIOT_DEV_REST_ENABLED=true`, `CHARIOT_PORT=8090` | Application ports plus Dev REST, bound to loopback where possible |

Headless does not mean Dev REST is disabled. With neither headless nor Dev REST,
the normal service path exits rather than starting an application. MCP `stdio`
is a separate tooling mode, not a bootstrap application launch; for debug use
`CHARIOT_MCP_TRANSPORT=http` or `ws` with Dev REST. Do not expose arbitrary
execution/debug endpoints publicly. Runtime cloning is lifecycle isolation,
**not a security sandbox**; protect credentials, network, files and host APIs
using deployment boundaries.

Bootstrap must bind container HTTP listeners to `0.0.0.0`, not loopback, for
published ports to work. NSQ consumes outbound TCP; it needs no inbound listener
port mapping on the application container. The full example requires an external
NSQD reachable by its container-network hostname/address. Container `localhost`
does not mean the host's NSQD. Use the HTTP-only bootstrap if NSQD is absent.

## Packaging using supported repository steps

There is no `chariot publish-container` CLI. The repository helper is
[scripts/publish-agent-container.sh](../scripts/publish-agent-container.sh).
It accepts `--agent`, `--onstart`, `--profile production|debug`, `--agent-name`,
`--image`, `--tag`, `--platform`, and `--keep-context`. Both input files are
required; it copies the script to `/app/onstart.chariot`, copies the agent
artifact, builds an image, and **does not push** it. Use a real exported agent
artifact, not the example function library as an agent substitute.

**Current helper limitations (not fixed by this documentation):**

- It still sets `CHARIOT_ON_START_FILE`, which the server does not use. Override
  `CHARIOT_DATA_PATH=/app` and `CHARIOT_BOOTSTRAP=onstart.chariot` when running its
  output, so the copied script is selected correctly.
- It does not bundle the required function library, this example's counter,
  secrets configuration, or arbitrary dependent files. Supply them through a
  reviewed derived image or mounts, and explicitly set TreePath/FunctionLib and
  secret-provider configuration. Copying an agent file does not make bootstrap
  load it.
- Its generated builder uses Go 1.24, while the module requests at least 1.24.2
  (and dependencies may require newer); toolchain acquisition must be available.
  It explicitly requires the Linux CPU static RL support archive and hard-codes
  amd64 build tags. Do not claim arbitrary `--platform` values are validated.
- Its shell entrypoint uses `set -euo pipefail` with `/bin/sh`; Debian dash does
  not support `pipefail`. Before release, fix that helper/template in a separate
  change or bypass it using the image's server binary as the entrypoint.
- Its `EXPOSE` metadata names 8080 (and 8090 for debug). This does not publish
  ports or override script ports; map the actual ports your bootstrap starts.

For the repository base image workflow, review the actual service Dockerfile and
build prerequisites first. Mount a prepared example data directory at the
configured DataPath, set TreePath to that directory for the example library,
select `bootstrap-http-only.ch` or `bootstrap.ch`, and run the server binary as
PID 1. Detailed non-container startup and observable checks are in the
[example README](../examples/listeners/README.md). Image builds, daemon networking
and registry pushes are deployment validation gates, not assumed successes.

After validating a DevTest script, deliberately incorporate its handler
definitions and `listen`/`listenNSQ` calls into the production bootstrap (or a
reviewed library invoked there). Choose production ports/bind addresses and
dependencies, remove test-only hooks, and rerun bootstrap validation. A persisted
definition, its `script` label, and `auto_start_devtest` do not package or start
an application.

## Shutdown and failure semantics

Synchronous bind/connection and start-hook failures fail bootstrap. Resources
created earlier in that bootstrap are rolled back; startup must not be treated
as successful if only part of the application started.

Run the binary directly or use an entrypoint that `exec`s it so SIGINT/SIGTERM
reach the host. Shutdown stops accepting Dev REST work, drains owned application
listeners and test/session resources, and invokes exit hooks at most once.
Shutdown is idempotent; force-closing sockets at a deadline does not kill
arbitrary handler/hook code. Ownership must remain until that work drains, so a
timeout is an error, not proof of disposal. Keep handlers/hooks bounded and give
the container orchestrator a termination grace period longer than
`CHARIOT_SHUTDOWN_TIMEOUT`. A clean host shutdown exits successfully; startup,
server, or shutdown errors are reported as failures.

Connected `TestListenNSQEndToEnd` passed in parent validation; see the
[final verification record](notes/Chariot%20Listener%20Refactor%20Plan.md#final-validation-record).
Manual SIGINT/SIGTERM smoke validation and container image build/push were not
performed. Connected NSQ coverage does not establish container networking or
image readiness, and the publishing-helper limitations above remain unresolved.
