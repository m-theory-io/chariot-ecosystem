// Standalone HTTP-only composition root. Requires bootstrap-count.txt containing 0.
// The persistent counter detects a second execution even in a fresh runtime.
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