# Runnable listener applications

Read the canonical [listener contract](../../docs/ListenerFunctions.md) and
[container guide](../../docs/Containerizing%20a%20Chariot%20Agent.md).
Run the following from the **repository root** with Go/module dependencies and
the platform's native build prerequisites installed. Examples bind fixed ports;
stop other processes using 18080, 18081, 18082, 8050 or 8090 as appropriate.
These are trusted local examples, not authenticated public services.

## Files

- [bootstrap.ch](bootstrap.ch): two HTTP listeners and one NSQ consumer.
- [bootstrap-http-only.ch](bootstrap-http-only.ch): identical HTTP portion,
  no external NSQD dependency.
- [devtest.ch](devtest.ch): independent loopback listener on 18082.
- [bootstrap-count.txt](bootstrap-count.txt): initial persistent counter, 0.
- [stlib.json](stlib.json): empty function library; handlers are inline.
- [secrets.local.json](secrets.local.json): empty local secret-provider fixture,
  **not real credentials**.
- [run-example.sh](run-example.sh): prepares a fresh temporary DataPath, copies
  fixtures, builds the real server, and `exec`s it. It is an example launcher,
  not a new server CLI or publishing command.
- [validate_examples.py](validate_examples.py): repeatable real-server checks
  using temporary data, without adding Go source or requiring NSQD.

Callbacks read the counter from an explicit global. Temporary `setq(func(...))`
handler values are defined next to their `listen` calls because hook program
execution creates a fresh working scope; it does not preserve all bootstrap
locals. See the canonical hook semantics rather than assuming JavaScript-style
declaration syntax or lexical scope across program executions.

## HTTP-only production start

```sh
bash examples/listeners/run-example.sh http production
```

The launcher prints the temporary DataPath. Keep that terminal running; from a
second terminal:

```sh
curl --fail http://127.0.0.1:18080/health
curl --fail -H 'Content-Type: application/json' \
  -d '{"input":"hello"}' http://127.0.0.1:18081/echo
```

Health returns `{"service":"listener-example","bootstrap_runs":1}` (JSON key
order is not significant). Echo returns HTTP 200 with `request` and
`bootstrap_runs:1`; `request.body.input` is `hello`. Malformed JSON returns 400;
GET on the POST-only echo service returns 405. Stop with Ctrl-C; inspect the exit
hook log messages and confirm ports close. This is a recommended manual check,
not a claim of completed signal smoke validation.

## Debug bootstrap-once demonstration

Stop the production process, then start a **fresh** example data directory:

```sh
bash examples/listeners/run-example.sh http debug
```

Repeat both application requests. Also request Dev REST:

```sh
curl --fail http://127.0.0.1:8090/health
curl --fail http://127.0.0.1:18080/health
```

The application still reports `bootstrap_runs:1`; inspect the printed DataPath's
counter file, which must contain `1`. Bootstrap increments this file **before**
binding, so duplicate execution in a fresh runtime also leaves evidence rather
than simply resetting an in-memory counter. Creating a session or attaching IDE
tools must not increment it. If you deliberately reuse the same directory across
process restarts, each new process increments it; compare to the previous value
instead of expecting 1. It is a demonstration counter, not a crash-safe global
exactly-once transaction or a multi-process counter. Do not run two hosts against
the same example directory.

The launcher disables MCP. For optional network MCP, launch the built server
with the same printed configuration, Dev REST enabled, `CHARIOT_MCP_ENABLED=true`,
and `CHARIOT_MCP_TRANSPORT=http` (or `ws`), not `stdio`. Leave arbitrary MCP
execution disabled unless intentionally using trusted local development. Parent
host/session/MCP package tests passed; the launcher itself does not exercise
every surface. See the [final validation record](../../docs/notes/Chariot%20Listener%20Refactor%20Plan.md#final-validation-record).

## Full HTTP + NSQ application

Start an external NSQD with TCP port 4150 and HTTP port 4151 reachable from the
application. The example does not install or start NSQD. Then:

```sh
CHARIOT_NSQ_ADDR=127.0.0.1:4150 \
  bash examples/listeners/run-example.sh nsq production
```

Repeat the HTTP requests above. Before publishing a message, attach an NSQ
consumer to topic `listener_example_responses` using channel `example_observer`
(for example, with the standard NSQ distribution's `nsq_tail` utility). Then use
the real NSQD HTTP publish endpoint:

```sh
curl --fail -X POST \
  'http://127.0.0.1:4151/pub?topic=listener_example_requests' \
  -d '{"messageType":"echo","responseTopicKey":"example_reply","input":"hello"}'
```

The response consumer receives JSON with `handled:true`, the original `message`,
and `bootstrap_runs:1`. `example_reply` is registered by bootstrap and maps to
`listener_example_responses`. The original payload routes to
`listener_example_dead_letters` on configured structural failures or exhausted
transient retries. Start a dead-letter consumer when testing these cases. An
NSQD publish acknowledgement alone does not prove Chariot handled the message
or published a response. Without NSQD, use the HTTP-only composition; do not
silently skip the NSQ call in a purported full application.

## DevTest Start / Exit

Use the running debug host above. The launcher already copies
[devtest.ch](devtest.ch) into its configured listener-scripts directory. In an
existing deployment, copy it into `CHARIOT_LISTENER_SCRIPTS_DIR` or upload it
through Charioteer's listener script editor instead.

In Charioteer, create a definition with name `echo-test`, primary script file
`devtest.ch`, and no registry hooks, then select **Start**. Resource-local hooks
are already in the script. Equivalently, using an authenticated Dev REST session
(substitute your session token locally):

```sh
curl --fail -H "Authorization: Bearer $CHARIOT_AUTH_TOKEN" \
  -H 'Content-Type: application/json' \
  -d '{"name":"echo-test","script":"Echo test","script_file":"devtest.ch","auto_start_devtest":false}' \
  http://127.0.0.1:8090/api/listeners
curl --fail -X POST -H "Authorization: Bearer $CHARIOT_AUTH_TOKEN" \
  http://127.0.0.1:8090/api/listeners/echo-test/start
curl --fail http://127.0.0.1:18082/devtest
curl --fail -X POST -H "Authorization: Bearer $CHARIOT_AUTH_TOKEN" \
  http://127.0.0.1:8090/api/listeners/echo-test/stop
```

Use your deployment's supported login flow to obtain the session; this example
does not invent credentials or a token CLI. Responses use the API's
`result`/`data` envelope. Start must report real resources and health; Exit/Stop
must close 18082 while the application's 18080/18081 remain live. Stop before
editing/deleting a live definition. After restart, definitions load stopped;
**auto-start is disabled**, even if the inert `auto_start_devtest` preference is
stored as true. `script` is only a label; `script_file` is required. A script
that creates no running listener fails Start. Isolation is resource ownership,
not a security sandbox or a deep copy of all host state.

## Promote and publish deliberately

1. Validate DevTest requests, failures, lifecycle and dependencies.
2. Incorporate the tested handler and listener calls into the production
   bootstrap. Change the DevTest loopback bind to the intended production bind
   and choose a nonconflicting production port; do not start from registry state.
3. Supply the function library, data/agent artifacts and external dependencies
   explicitly. Validate bootstrap rollback and debug bootstrap-once behavior.
4. Use the actual repository helper with a **real exported agent artifact**:

```sh
bash scripts/publish-agent-container.sh \
  --agent "$EXPORTED_AGENT_FILE" \
  --onstart examples/listeners/bootstrap-http-only.ch \
  --profile production --agent-name listener-example \
  --image your-registry/listener-example --tag production
```

This helper builds only. Review the [known helper limitations and required
overrides](../../docs/Containerizing%20a%20Chariot%20Agent.md) first; it is not a
turnkey publish path for these fixtures. To run its output after a successful
build, prepare a directory with the counter/library fixtures (and safe secret
configuration), then bypass its currently incompatible shell entrypoint:

```sh
docker run --rm --entrypoint /usr/local/bin/chariot-server \
  -p 127.0.0.1:18080:18080 -p 127.0.0.1:18081:18081 \
  -v "$EXAMPLE_DATA_DIR:/example-data" \
  -e CHARIOT_DATA_PATH=/app -e CHARIOT_BOOTSTRAP=onstart.chariot \
  -e CHARIOT_TREE_PATH=/example-data -e CHARIOT_FUNCTION_LIB=stlib.json \
  -e CHARIOT_SECRET_PROVIDER=file \
  -e CHARIOT_SECRET_FILE_PATH=/example-data/secrets.local.json \
  -e CHARIOT_HEADLESS=true -e CHARIOT_DEV_REST_ENABLED=false \
  -e CHARIOT_MCP_ENABLED=false \
  --mount "type=bind,src=$EXAMPLE_DATA_DIR/bootstrap-count.txt,dst=/app/bootstrap-count.txt" \
  your-registry/listener-example:production
docker push your-registry/listener-example:production
```

`EXAMPLE_DATA_DIR` must be an absolute path to a writable prepared directory,
not a registry definition. Reset its counter to 0 before a fresh demonstration.
Mounting it at `/app` would hide the script copied by the helper; the separate
mounts above avoid that. For debug build with `--profile debug`, explicitly enable
Dev REST and map its configured port to loopback. Push only after image-level
startup, requests, shutdown and dependency checks succeed. These container
commands are supported steps to perform, not a claim that an image was built,
pushed, or smoke-tested in this documentation task.

## Validation record

Build and run the repeatable fixture checks from the repository root:

```sh
(cd services/go-chariot && go build -o /tmp/chariot-wp8-validation ./cmd)
python3 examples/listeners/validate_examples.py /tmp/chariot-wp8-validation
bash -n examples/listeners/run-example.sh
```

Executed for WP8: server build and shell syntax passed; real-server
checks passed for both HTTP listeners, request/response JSON, strict malformed
JSON rejection, Dev REST with bootstrap counter 1, a second process counter 2,
DevTest script execution, the repaired decision handler, and full-bootstrap
rollback with a deliberately unreachable NSQD. The check runs the DevTest script
as a host bootstrap, **not** through the supervisor API. That validator alone
does not prove NSQ delivery or supervisor API acceptance.

Separately, parent validation passed all affected go-chariot unit/integration
packages (`go test ./chariot ./internal/... ./mcp ./tests ./cmd`), affected race
tests, `go vet`, all Charioteer tests, the Visual DSL production build, and the
example validator. Connected `TestListenNSQEndToEnd` and the host API
create/start/request/shutdown test also passed. These are completed gates, not
remaining handoff requests; see the
[final validation record](../../docs/notes/Chariot%20Listener%20Refactor%20Plan.md#final-validation-record).

`listenerStop(id)` returns `true` for an accepted asynchronous graceful-stop
request, not completion. The listener stays visible as `stopping` while active
handlers and exit hooks drain; after completion `listenerStatus(id)` returns
`DBNull`, without a tombstone. Unknown or already-removed IDs error on stop.
Go's waiting `StopListener(ctx, id)` must run outside `WithExecution`.

Child process termination is automated test cleanup; no manual SIGINT/SIGTERM
smoke or container image build/push was performed. Publishing-helper limitations
remain unresolved, and shutdown deadlines cannot terminate arbitrary blocked
host code or release ownership before it drains. Passing automated host/API and
NSQ tests does not establish container readiness or manual signal behavior.