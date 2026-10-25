// Copy into the configured listener scripts directory; select it with script_file.
// Does not run bootstrap or use its production ports.
setq(devTestHandler, func(req) {
  return(httpResponse(200, map("mode", "devtest", "request", req)))
})
listen(18082, devTestHandler, map(
  "bind_address", "127.0.0.1",
  "methods", array("GET", "POST"),
  "base_path", "/devtest",
  "on_start", "logPrint('DevTest listener started')",
  "on_exit", "logPrint('DevTest listener exited')"
))