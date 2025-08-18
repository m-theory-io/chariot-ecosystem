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
	api.GET("/functions", h.ListFunctions)
	api.POST("/function/save", h.SaveFunctionHandler)
}
