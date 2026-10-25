// Runnable transport example, not a business decision engine.
// Replace this echo result with your validated agent/rule evaluation.
setq(decisionHandler, func(req) {
  return(httpResponse(200, map("service", "decision-example", "request", req)))
})

listen(8050, decisionHandler, map(
  "methods", array("POST"),
  "base_path", "/decision",
  "strict_json", true,
  "on_start", "logPrint('Decision listener started')",
  "on_exit", "logPrint('Decision listener exited')"
))