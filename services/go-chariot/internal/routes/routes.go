package routes

import (
	"github.com/bhouse1273/chariot-ecosystem/services/go-chariot/internal/handlers"
	"github.com/labstack/echo/v4"
)

// RegisterRoutes sets up all API routes
func RegisterRoutes(e *echo.Echo, h *handlers.Handlers) {

	// Public routes here
	e.GET("/", func(c echo.Context) error {
		return c.String(200, "Hello from go-chariot!")
	})
	e.GET("/health", h.Health)
	e.GET("/ready", h.Ready)
	e.POST("/login", h.HandleLogin)
	e.POST("/logout", h.HandleLogout)

	// Protected routes
	api := e.Group("/api")
	api.Use(h.SessionAuth)
	api.GET("/data", h.GetData)
	api.POST("/execute", h.Execute)
	api.POST("/execute-async", h.ExecuteAsync)
	api.GET("/logs/:execId", h.StreamLogs)
	api.GET("/result/:execId", h.GetResult)
	api.GET("/functions", h.ListFunctions)
	api.GET("/global-variables", h.ListGlobalVariables)
	api.POST("/function/save", h.SaveFunctionHandler)
	api.POST("/functions/save-library", h.SaveFunctionLibraryHandler)

	// Diagrams API
	diagrams := api.Group("/diagrams")
	diagrams.GET("", h.ListDiagrams)           // GET /api/diagrams
	diagrams.GET("/:name", h.GetDiagram)       // GET /api/diagrams/:name
	diagrams.POST("", h.SaveDiagram)           // POST /api/diagrams
	diagrams.DELETE("/:name", h.DeleteDiagram) // DELETE /api/diagrams/:name

	// Listener registry APIs
	listeners := api.Group("/listeners")
	listeners.GET("", h.ListListeners)              // GET /api/listeners
	listeners.POST("", h.CreateListener)            // POST /api/listeners
	listeners.DELETE("/:name", h.DeleteListener)    // DELETE /api/listeners/:name
	listeners.POST("/:name/start", h.StartListener) // POST /api/listeners/:name/start
	listeners.POST("/:name/stop", h.StopListener)   // POST /api/listeners/:name/stop

	// Agents APIs
	agents := api.Group("/agents")
	agents.GET("", h.ListAgents)
	agents.POST("/create", h.CreateAgent)      // POST /api/agents/create
	agents.POST("/stop", h.StopAgent)          // POST /api/agents/stop
	agents.POST("/publish", h.PublishAgent)    // POST /api/agents/publish
	agents.POST("/belief", h.SetBelief)        // POST /api/agents/belief
	agents.GET("/:name/beliefs", h.GetBeliefs) // GET /api/agents/:name/beliefs
	agents.GET("/:name/info", h.GetAgentInfo)  // GET /api/agents/:name/info
	agents.POST("/run-once", h.RunPlanOnce)    // POST /api/agents/run-once
	// Legacy routes for compatibility
	agents.POST("/start", h.StartAgent)
	agents.POST("/:name/stop", h.StopAgent)
	agents.POST("/:name/publish", h.PublishAgent)
	agents.PUT("/:name/beliefs", h.PutBelief)

	// Protected dashboard routes (require authentication)
	dashboard := e.Group("/dashboard")
	dashboard.Use(h.SessionAuth)
	dashboard.GET("", h.HandleDashboard)  // /dashboard
	dashboard.GET("/", h.HandleDashboard) // /dashboard/ (with trailing slash)

	// Protected dashboard API routes
	dashboardAPI := e.Group("/api/dashboard")
	dashboardAPI.Use(h.SessionAuth)
	dashboardAPI.GET("/status", h.HandleDashboardAPI)
	// WebSocket stream: auth is performed inside handler with non-extending lookup
	e.GET("/api/dashboard/stream", h.HandleDashboardWS)

	// Agents WS stream (canonical path under /ws)
	e.GET("/ws/agents", h.HandleAgentsWS)

	// Debug API routes
	debug := api.Group("/debug")
	debug.POST("/breakpoint", h.DebugBreakpoint)
	debug.POST("/step", h.DebugStep)
	debug.POST("/continue", h.DebugContinue)
	debug.POST("/pause", h.DebugPause)
	debug.GET("/state", h.DebugState)
	debug.GET("/variables", h.DebugVariables)
	// WebSocket for debug events
	e.GET("/api/debug/events", h.DebugEvents)
}
