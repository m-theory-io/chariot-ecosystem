// Full composition root: two HTTP listeners and one NSQ consumer.
// Requires bootstrap-count.txt containing 0 and reachable CHARIOT_NSQ_ADDR.
declareGlobal(bootstrapRuns, "N", add(toNumber(readFile("bootstrap-count.txt")), 1))
writeFile("bootstrap-count.txt", toString(bootstrapRuns))

setq(healthHandler, func(req) {
  return(map("service", "listener-example", "bootstrap_runs", bootstrapRuns))
})
listen(18080, healthHandler, map(
  "bind_address", "0.0.0.0",
  "methods", array("GET"),
  "base_path", "/health",
  "on_start", "logPrint('health listener started')",
  "on_exit", "logPrint('health listener exited')"
))
// Define this after the first hook program; callback state belongs in globals.
setq(echoHandler, func(req) {
  return(httpResponse(200, map("request", req, "bootstrap_runs", bootstrapRuns)))
})
listen(18081, echoHandler, map(
  "bind_address", "0.0.0.0",
  "methods", array("POST"),
  "base_path", "/echo",
  "strict_json", true,
  "max_body_bytes", 1048576,
  "on_start", "logPrint('echo listener started')",
  "on_exit", "logPrint('echo listener exited')"
))

registerResponseTopic("example_reply", "listener_example_responses")
setq(messageHandler, func(message) {
  return(map("handled", true, "message", message, "bootstrap_runs", bootstrapRuns))
})
listenNSQ("listener_example_requests", "listener_example_workers", map(
  "echo", messageHandler
), map(
  "nsqd_address", getEnv("CHARIOT_NSQ_ADDR"),
  "max_attempts", 5,
  "dead_letter_topic", "listener_example_dead_letters",
  "response_required", true,
  "on_start", "logPrint('NSQ listener started')",
  "on_exit", "logPrint('NSQ listener exited')"
))