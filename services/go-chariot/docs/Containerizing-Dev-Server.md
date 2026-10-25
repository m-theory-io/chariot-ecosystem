# Containerizing go-chariot Server (Development Mode)

Use the debug profile in the canonical
[container guide](../../../docs/Containerizing%20a%20Chariot%20Agent.md) for
bootstrap/path configuration, Dev REST exposure, shutdown, and publishing-helper
limitations. The server defaults to Dev REST port 8087; the examples explicitly
select 8090.

The [runnable listener examples](../../../examples/listeners/README.md) cover
HTTP-only and HTTP/NSQ applications, debug bootstrap-once checks, and DevTest
Start/Exit. Listener and DevTest contracts are in
[Listener Functions](../../../docs/ListenerFunctions.md).

This is a navigation link, not a maintained duplicate. The obsolete infrastructure
walkthrough has been removed; it used unsupported startup settings and unverified
build/deployment examples. Image build/push and manual signal smoke remain
separate validation gates.
