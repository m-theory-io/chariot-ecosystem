# Chariot Listener Functions

Application listeners are created by an application's bootstrap runtime and remain owned by that runtime until explicitly stopped or the application host shuts down.

## HTTP listeners

### `listen(port, handler [, options])`

`handler` is either a named Chariot function string or an inline function. A successful return guarantees that the socket is bound.

Canonical options are:

| Option | Type | Behavior |
|---|---|---|
| `bind_address` | string | Host or IP without a port; empty means all interfaces |
| `methods` | string array | Allowed standard HTTP methods |
| `base_path` | string | Exact path or slash-delimited descendants only |
| `read_timeout_ms` | non-negative integer | HTTP server read timeout |
| `write_timeout_ms` | non-negative integer | HTTP server write timeout |
| `idle_timeout_ms` | non-negative integer | HTTP server idle timeout |
| `max_body_bytes` | positive integer | Maximum body size; default 1 MiB; overflow returns 413 |
| `strict_json` | boolean | Reject malformed JSON bodies with 400 when true |
| `on_start` | string | Program invoked after bind; failure rolls startup back |
| `on_exit` | string | Program invoked once when the listener exits |

Unknown keys, aliases, invalid methods, wrong types, fractional numbers, negative durations, and malformed bind addresses are rejected.

```chariot
listen(8080, "handleOrder", map(
  "bind_address", "127.0.0.1",
  "methods", array("POST"),
  "base_path", "/orders",
  "max_body_bytes", 1048576,
  "strict_json", true
))
```

The request map has these fields:

| Field | Meaning |
|---|---|
| `method`, `path`, `query_string` | HTTP method, URL path, raw query string |
| `remote_addr`, `host`, `scheme` | Peer address, request host, `http` or `https` |
| `timestamp` | UTC RFC3339Nano arrival timestamp |
| `query` | Present when nonempty; query-name/value map |
| `headers` | Present when nonempty; header-name/value map |
| `raw_body` | Present for a nonempty body, within the configured byte limit |
| `body` | Decoded JSON when the content type contains `application/json` |
| `body_error` | JSON decode error, only with `strict_json=false` (the default) |
| `form`, `form_error` | Parsed form values or form parse error when applicable |

Repeated query, header, or form values become arrays; single values are strings.
Oversize bodies return 413; strict malformed JSON returns 400 before dispatch.
Outside `base_path` returns 404; disallowed methods return 405. Permitted
`OPTIONS` requests return 204 without calling the handler. With `methods`
omitted there is no method filter. Ports must be integers from 1 through 65535;
port 0 is not supported. HTTP read/write/idle timeouts default to zero.

### Handler results

Define function values using actual Chariot syntax:

```chariot
setq(handler, func(req) {
  return(map("request", req))
})
listen(8080, handler)
```

Ordinary return values are JSON with HTTP 200; `DBNull` returns 204 with no
body. Handler execution errors return 500. Use `return(value)`, not JavaScript
`return value` or `let` declarations.

### Hooks are programs, not callback names inferred from variables

Both transports accept **strings** for `on_start` and `on_exit`. A string matching
a registered function-library name invokes that function with one numeric
argument: the HTTP port, or 0 for NSQ. Otherwise the string is evaluated as
Chariot program text. For a function value visible to that program, pass an
explicit call such as `"started(8080)"`, not the bare variable name `"started"`.
A bare variable expression is not a callback invocation. Program execution
starts a fresh working scope: local `setq` variables from bootstrap are not
available by name inside a later hook program. Use a registered library function,
explicit global state declared with `declareGlobal(name, type, value)`, or a
self-contained program such as `"logPrint('started')"`. Define temporary handler
values next to their listener call; keep callback state global when it must
survive nested hook execution. This is current interpreter behavior, not lexical
scope preservation. Hook filenames are not read by these options; use DevTest
hook-file fields for that workflow.

HTTP start hooks run after bind/registration; NSQ start hooks run after consumer
connection. Start-hook errors fail startup and roll back the resource. Exit
hooks run at most once after the resource drains. Keep hooks bounded: shutdown
deadlines cannot forcibly terminate arbitrary Chariot or host code.

### `httpResponse(status, body)`

### `httpResponse(status, headers, body)`

Creates explicit transport metadata without treating ordinary maps specially. Header values must be strings or arrays of strings. If no content type is supplied for a body, it defaults to `application/json`.

```chariot
httpResponse(201, map(
  "Location", "/orders/42",
  "Content-Type", "application/json"
), map("id", 42))
```

An ordinary `map("status", 201)` remains a JSON object returned with HTTP 200.

## Runtime lifecycle

- `listenerList()` returns status maps for all live listeners owned by the current runtime.
- `listenerStatus(id)` returns one status map while the listener is owned,
  including during `stopping`. After completion it returns `DBNull`; there are
  no completed-listener tombstones. Unknown IDs also return `DBNull`.
- `listenerStop(id)` **requests graceful stop** and returns `true` when the
  request is accepted, not when shutdown completes. The stop is asynchronous,
  so a handler may request its own stop and still return its response. Repeated
  requests while the listener is still owned are accepted without repeating
  shutdown or exit hooks. Unknown or already-removed IDs return an error.

Listener IDs are process-local resource identities. Status maps include transport identity, state, timestamps, request/failure/in-flight counters, and the latest error when present.

The language status keys are `id`, `kind`, `state`, `started_at`,
`request_count`, `failure_count`, `in_flight_count`, and, when applicable,
`address`, `port`, `topic`, `channel`, `last_active_at`, `last_success_at`,
`last_error`. Copy the actual ID from inventory; do not hard-code wildcard bind
addresses. An accepted stop makes `state` equal to `stopping`; the listener stays
visible in `listenerList()` and `listenerStatus(id)` until active handlers and
exit hooks finish. Completion removes it from inventory, rather than retaining
a terminal status record.

Go callers use `RequestStopListener(id)` for request-only semantics or
`StopListener(ctx, id)` to request stop **and wait** for completion. Call the
waiting API outside `WithExecution`: handlers and hooks may need that same
non-reentrant execution guard to finish. A wait deadline returns an error but
does not release ownership or terminate blocked Chariot or host code. Session
clones do not own bootstrap listeners. All listener callbacks sharing a runtime
execute serially.

## NSQ listeners

### `listenNSQ(topic, channel, handlers [, options])`

`handlers` maps normalized `messageType` values to named or inline Chariot functions. NSQ may deliver concurrently, but Chariot serializes HTTP and NSQ callbacks across every listener sharing a runtime. Production therefore uses actor-style runtime state rather than a runtime pool.

Canonical options are:

| Option | Type | Default |
|---|---|---|
| `nsqd_address` | string | `CHARIOT_NSQ_ADDR` |
| `lookupd_addresses` | string array | none |
| `concurrency` | positive integer | 1 |
| `max_in_flight` | positive integer | 200 |
| `max_attempts` | integer 1–65535 | 5 |
| `backoff_strategy` | `exponential` or `full_jitter` | `exponential` |
| `backoff_multiplier_ms` | non-negative integer | 1000 |
| `max_backoff_ms` | non-negative integer | 120000 |
| `dead_letter_topic` | string | disabled |
| `decode_error_policy` | policy | `dead_letter` |
| `missing_message_type_policy` | policy | `dead_letter` |
| `missing_handler_policy` | policy | `dead_letter` |
| `missing_response_policy` | policy | `dead_letter` |
| `unknown_response_policy` | policy | `dead_letter` |
| `response_required` | boolean | false |
| `response_topic_field` | string | `responseTopicKey` |
| `response_nsqd_address` | string | `nsqd_address` |
| `on_start`, `on_exit` | string | empty |

A policy is `ack`, `requeue`, or `dead_letter`:

- malformed messages, missing `messageType`, and missing handlers use their configured structural-failure policy;
- handler, encoding, and response-publish failures requeue until `max_attempts`;
- exhausted transient failures publish the original payload to `dead_letter_topic` when configured, then acknowledge;
- dead-letter publish failure returns an error so NSQ requeues rather than losing the message;
- a missing response key is acknowledged when `response_required` is false;
- unknown response-topic keys use `unknown_response_policy` because they indicate invalid routing configuration.

```chariot
listenNSQ("decision_requests", "decision_workers", map(
  "lease", "handleLeaseDecision"
), map(
  "nsqd_address", "nsqd:4150",
  "max_attempts", 5,
  "backoff_strategy", "full_jitter",
  "dead_letter_topic", "decision_dead_letters",
  "response_required", true
))
```

Unknown option names and legacy aliases are rejected. Update `nsqd` to `nsqd_address`, `lookupd` to `lookupd_addresses`, `maxinflight` to `max_in_flight`, and `response_nsqd` to `response_nsqd_address`.

### Response topics and message payloads

Handlers receive the decoded JSON object (not an HTTP request map), augmented
with `_raw` (original message text) and `_nsq` (`attempts`, UTC RFC3339Nano
`timestamp`, and hex `id`); these metadata keys overwrite payload keys of the
same name.
`messageType` is trimmed and case-normalized for dispatch. A binding may be a
function value, a registered function-name string, or a map containing `handler`
and optional `responseTopicKey`:

```chariot
registerResponseTopic("reply", "decision_responses")
setq(handleMessage, func(message) {
  return(map("handled", true, "input", message))
})
listenNSQ("decision_requests", "decision_workers", map(
  "echo", map("handler", handleMessage, "responseTopicKey", "reply")
), map("nsqd_address", "127.0.0.1:4150", "response_required", true))
```

Alternatively put `"responseTopicKey":"reply"` in each incoming message.
The configured `response_topic_field` is checked first; a nonempty message key
overrides the binding's fallback key. Keys resolve through
`registerResponseTopic(key, topic)` / `listResponseTopics()` or the process
configuration `CHARIOT_NSQ_RESPONSE_TOPICS` (comma-separated `key:topic` pairs).
The handler result is JSON-encoded and published directly, without an extra
response envelope. A routing key is a registered logical key, not an arbitrary
destination topic. No response is published when routing is absent and optional.
Response-topic registration is process-wide, not private to a DevTest clone.

Topic and channel arguments must be nonempty strings. NSQ requires a reachable
daemon; use the [HTTP-only bootstrap](../examples/listeners/bootstrap-http-only.ch)
when it is unavailable. See the [runnable workflow](../examples/listeners/README.md)
and [container guide](Containerizing%20a%20Chariot%20Agent.md).

## DevTest supervisor: configuration and migration

Production starts only from `CHARIOT_BOOTSTRAP`; the registry is not a second
composition root. The v2 snapshot persists **configuration only**:

```json
{
  "version": 2,
  "listeners": {
    "echo-test": {
      "name": "echo-test",
      "script": "Echo test",
      "script_file": "devtest.ch",
      "on_start": "",
      "on_start_file": "",
      "on_exit": "",
      "on_exit_file": "",
      "auto_start_devtest": false
    }
  }
}
```

- Loading v1 retains definition fields, ignores persisted live status and legacy
  `auto_start`, and forces `auto_start_devtest=false`. The next successful
  create/update/delete saves v2. Start/Stop do not rewrite configuration.
- **Auto-start is disabled in both production and DevTest.** The v2
  `auto_start_devtest` field is an inert stored preference, even if true. Start
  explicitly. There is no supported auto-start enable flag.
- `script` is a display label, not executable source or a file fallback.
  `script_file` is required for new/updated definitions and successful Start.
  Repair legacy definitions lacking it through Update before starting them.
- Script and hook-file fields accept single `.ch` filenames (extension added
  when omitted), not paths. Start reads regular files confined to
  `CHARIOT_LISTENER_SCRIPTS_DIR` (default `CHARIOT_DATA_PATH/listeners`), with
  symlink escapes rejected. A relative scripts-dir config is joined to DataPath.
- Start clones the application runtime's library/state **without rerunning
  bootstrap or inheriting listener ownership**, then executes `script_file`.
  Every created listener must be running, and at least one must exist. Hook-only
  definitions and scripts that merely return a value fail Start.
- Registry `on_start` / `on_exit` use the program-string rules above (registered
  functions receive the configured Dev REST port). A nonempty `on_start_file` or
  `on_exit_file` takes precedence: its source is compiled as a named function
  under the filename stem. Use valid saved-function source for those files.
  For example, a hook file may contain
  `setq(startHook, func(port) { logPrint('test application started') })`.
  The registry start hook runs after primary resources exist; the registry exit
  hook runs after resource-local exits. Identical trimmed hook identities are
  suppressed to avoid running the same resource and registry hook twice.
- API status is derived live: `starting`, `running`, `stopping`, `stopped`, or
  `error`, plus `resources`, activity, health and errors. Restart loads stopped
  definitions, never resurrected running status. Stop before updating/deleting
  a live or still-draining instance. A Stop deadline does not release ownership
  while arbitrary code is still executing.
- Clones provide **lifecycle isolation, not a security sandbox**. Global values
  may be shallow-copied and host objects borrowed. Do not use untrusted scripts;
  borrowed bootstrap connections must not be closed by test disposal. Newly
  created owned closers are disposed after listeners and hooks drain.

With authenticated Dev REST, use GET/POST `/api/listeners`, PUT/DELETE
`/api/listeners/:name`, and POST `/api/listeners/:name/start` or `/stop`.
Script CRUD uses `/api/listener-scripts`. Charioteer's Start/Exit controls call
these lifecycle operations. The registry list is distinct from `listenerList()`:
one lists test applications; the other lists resources in the current runtime.
