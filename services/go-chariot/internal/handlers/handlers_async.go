package handlers

import (
	"fmt"
	"net/http"
	"time"

	"github.com/bhouse1273/chariot-ecosystem/services/go-chariot/chariot"
	cfg "github.com/bhouse1273/chariot-ecosystem/services/go-chariot/configs"
	"github.com/labstack/echo/v4"
	"go.uber.org/zap"
)

// ExecuteAsync starts a script execution asynchronously and returns an execution ID
// The client can then stream logs via /logs/:execId and poll for result via /result/:execId
func (h *Handlers) ExecuteAsync(c echo.Context) error {
	// Incoming JSON: {"program": "your chariot code here"}
	type Request struct {
		Program string `json:"program"`
	}
	var req Request
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, ResultJSON{
			Result: "ERROR",
			Data:   "Invalid request format",
		})
	}

	// Validate program field
	if req.Program == "" {
		return c.JSON(http.StatusBadRequest, ResultJSON{
			Result: "ERROR",
			Data:   "Missing program field",
		})
	}

	// Basic validation
	if len(req.Program) < 5 {
		return c.JSON(http.StatusBadRequest, ResultJSON{
			Result: "ERROR",
			Data:   "Program is too short",
		})
	}

	// Get session from context
	session := c.Get("session").(*chariot.Session)

	// Create execution context
	execCtx := h.execManager.Create(session.UserID, req.Program)

	// Start execution in background goroutine
	go func() {
		defer func() {
			if r := recover(); r != nil {
				cfg.ChariotLogger.Error("Panic in async execution",
					zap.String("exec_id", execCtx.ID),
					zap.Any("panic", r))
				execCtx.MarkDone(nil, fmt.Errorf("execution panic: %v", r))
			}
		}()

		// Use the session's runtime (which has bootstrap globals/objects)
		rt := session.Runtime

		// Hook the runtime's logger to write to the execution context
		rt.SetLogWriter(execCtx.LogBuffer)

		// Add a test log to verify streaming works
		rt.WriteLog("INFO", "=== Execution started ===")

		// Execute the program
		val, err := rt.ExecProgram(req.Program)

		// Add completion log
		if err != nil {
			rt.WriteLog("ERROR", fmt.Sprintf("=== Execution failed: %v ===", err))
		} else {
			rt.WriteLog("INFO", "=== Execution completed successfully ===")
		}

		// Convert result to JSON-serializable format
		var result interface{}
		if err == nil {
			result = convertValueToJSON(val)
		}

		// Mark execution as complete
		execCtx.MarkDone(result, err)

		cfg.ChariotLogger.Info("Async execution completed",
			zap.String("exec_id", execCtx.ID),
			zap.Bool("success", err == nil))
	}()

	return c.JSON(http.StatusOK, ResultJSON{
		Result: "OK",
		Data: map[string]string{
			"execution_id": execCtx.ID,
		},
	})
}

// StreamLogs streams log entries for a given execution via Server-Sent Events (SSE)
func (h *Handlers) StreamLogs(c echo.Context) error {
	execID := c.Param("execId")
	if execID == "" {
		return c.JSON(http.StatusBadRequest, ResultJSON{
			Result: "ERROR",
			Data:   "Missing execution ID",
		})
	}

	execCtx := h.execManager.Get(execID)
	if execCtx == nil {
		return c.JSON(http.StatusNotFound, ResultJSON{
			Result: "ERROR",
			Data:   "Execution not found",
		})
	}

	// Set SSE headers
	c.Response().Header().Set("Content-Type", "text/event-stream")
	c.Response().Header().Set("Cache-Control", "no-cache")
	c.Response().Header().Set("Connection", "keep-alive")
	c.Response().Header().Set("X-Accel-Buffering", "no") // Disable nginx buffering
	c.Response().WriteHeader(http.StatusOK)

	// Subscribe to log buffer
	subscriber := execCtx.LogBuffer.Subscribe()
	defer execCtx.LogBuffer.Unsubscribe(subscriber)

	// Send all existing logs first
	existingLogs := execCtx.LogBuffer.GetAll()
	cfg.ChariotLogger.Info("Sending existing logs via SSE",
		zap.String("exec_id", execID),
		zap.Int("count", len(existingLogs)))

	for _, entry := range existingLogs {
		if _, err := fmt.Fprintf(c.Response(), "data: %s\n\n", entry.JSON()); err != nil {
			cfg.ChariotLogger.Warn("Failed to write SSE log entry", zap.Error(err))
			return err
		}
		c.Response().Flush()
	}

	// Check if execution is already done
	if execCtx.IsDone() {
		cfg.ChariotLogger.Info("Execution already done, sending done event",
			zap.String("exec_id", execID))
		if _, err := fmt.Fprintf(c.Response(), "event: done\ndata: {}\n\n"); err != nil {
			cfg.ChariotLogger.Warn("Failed to write SSE done event", zap.Error(err))
		}
		c.Response().Flush()
		return nil
	}

	// Stream new logs as they arrive until execution completes or client disconnects
	for {
		select {
		case entry, ok := <-subscriber:
			if !ok {
				// Channel closed, subscriber unsubscribed
				return nil
			}
			if _, err := fmt.Fprintf(c.Response(), "data: %s\n\n", entry.JSON()); err != nil {
				cfg.ChariotLogger.Warn("Failed to write SSE log entry", zap.Error(err))
				return err
			}
			c.Response().Flush()

		case <-execCtx.DoneChan():
			// Execution completed, send final event and close
			if _, err := fmt.Fprintf(c.Response(), "event: done\ndata: {}\n\n"); err != nil {
				cfg.ChariotLogger.Warn("Failed to write SSE done event", zap.Error(err))
			}
			c.Response().Flush()
			return nil

		case <-c.Request().Context().Done():
			// Client disconnected
			cfg.ChariotLogger.Debug("Client disconnected from log stream",
				zap.String("exec_id", execID))
			return nil
		}
	}
}

// GetResult returns the result of an execution (polling endpoint)
func (h *Handlers) GetResult(c echo.Context) error {
	execID := c.Param("execId")
	if execID == "" {
		return c.JSON(http.StatusBadRequest, ResultJSON{
			Result: "ERROR",
			Data:   "Missing execution ID",
		})
	}

	execCtx := h.execManager.Get(execID)
	if execCtx == nil {
		return c.JSON(http.StatusNotFound, ResultJSON{
			Result: "ERROR",
			Data:   "Execution not found",
		})
	}

	// Check if execution is complete
	if !execCtx.IsDone() {
		return c.JSON(http.StatusAccepted, ResultJSON{
			Result: "PENDING",
			Data: map[string]interface{}{
				"execution_id": execID,
				"status":       "running",
				"started_at":   execCtx.StartedAt.Format(time.RFC3339),
			},
		})
	}

	// Get result and error
	result, err := execCtx.GetResult()
	if err != nil {
		return c.JSON(http.StatusOK, ResultJSON{
			Result: "ERROR",
			Data:   fmt.Sprintf("Execution error: %v", err),
		})
	}

	return c.JSON(http.StatusOK, ResultJSON{
		Result: "OK",
		Data:   result,
	})
}
