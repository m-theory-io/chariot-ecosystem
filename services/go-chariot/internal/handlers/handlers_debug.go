package handlers

import (
	"net/http"
	"strconv"

	"github.com/bhouse1273/chariot-ecosystem/services/go-chariot/chariot"
	"github.com/gorilla/websocket"
	"github.com/labstack/echo/v4"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow all origins for now
	},
}

// DebugBreakpointRequest represents a request to add/remove a breakpoint
type DebugBreakpointRequest struct {
	File      string `json:"file"`
	Line      int    `json:"line"`
	Condition string `json:"condition,omitempty"`
	Action    string `json:"action"` // "add", "remove", "enable", "disable"
}

// DebugStepRequest represents a step operation request
type DebugStepRequest struct {
	Mode string `json:"mode"` // "over", "into", "out"
}

// DebugStateResponse contains the current debugger state
type DebugStateResponse struct {
	State       string                   `json:"state"`
	Breakpoints []*chariot.Breakpoint    `json:"breakpoints"`
	CallStack   []chariot.StackFrame     `json:"callStack"`
	CurrentFile string                   `json:"currentFile"`
	CurrentLine int                      `json:"currentLine"`
	Variables   map[string]chariot.Value `json:"variables,omitempty"`
}

// DebugBreakpoint handles adding/removing/toggling breakpoints
func (h *Handlers) DebugBreakpoint(c echo.Context) error {
	sessionID := c.QueryParam("session")
	if sessionID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "session parameter required"})
	}

	session, err := h.sessionManager.GetSession(sessionID)
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "session not found"})
	}

	// Ensure debugger is enabled for this session
	if session.Runtime.Debugger == nil {
		session.Runtime.Debugger = chariot.NewDebugger()
	}

	var req DebugBreakpointRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}

	debugger := session.Runtime.Debugger

	switch req.Action {
	case "add":
		bp := debugger.SetBreakpoint(req.File, req.Line, req.Condition)
		return c.JSON(http.StatusOK, bp)

	case "remove":
		removed := debugger.RemoveBreakpoint(req.File, req.Line)
		return c.JSON(http.StatusOK, map[string]bool{"removed": removed})

	case "enable":
		debugger.EnableBreakpoint(req.File, req.Line, true)
		return c.JSON(http.StatusOK, map[string]string{"status": "enabled"})

	case "disable":
		debugger.EnableBreakpoint(req.File, req.Line, false)
		return c.JSON(http.StatusOK, map[string]string{"status": "disabled"})

	default:
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid action: must be add, remove, enable, or disable"})
	}
}

// DebugStep handles stepping operations (over, into, out)
func (h *Handlers) DebugStep(c echo.Context) error {
	sessionID := c.QueryParam("session")
	if sessionID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "session parameter required"})
	}

	session, err := h.sessionManager.GetSession(sessionID)
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "session not found"})
	}

	if session.Runtime.Debugger == nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "debugger not initialized"})
	}

	var req DebugStepRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}

	debugger := session.Runtime.Debugger

	switch req.Mode {
	case "over":
		debugger.StepOver()
	case "into":
		debugger.StepInto()
	case "out":
		debugger.StepOut()
	default:
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid step mode: must be over, into, or out"})
	}

	return c.JSON(http.StatusOK, map[string]string{"status": "stepping"})
}

// DebugContinue resumes execution
func (h *Handlers) DebugContinue(c echo.Context) error {
	sessionID := c.QueryParam("session")
	if sessionID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "session parameter required"})
	}

	session, err := h.sessionManager.GetSession(sessionID)
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "session not found"})
	}

	if session.Runtime.Debugger == nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "debugger not initialized"})
	}

	session.Runtime.Debugger.Continue()

	return c.JSON(http.StatusOK, map[string]string{"status": "running"})
}

// DebugPause pauses execution at the next statement
func (h *Handlers) DebugPause(c echo.Context) error {
	sessionID := c.QueryParam("session")
	if sessionID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "session parameter required"})
	}

	session, err := h.sessionManager.GetSession(sessionID)
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "session not found"})
	}

	if session.Runtime.Debugger == nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "debugger not initialized"})
	}

	session.Runtime.Debugger.Pause()

	return c.JSON(http.StatusOK, map[string]string{"status": "paused"})
}

// DebugState returns the current debugger state
func (h *Handlers) DebugState(c echo.Context) error {
	sessionID := c.QueryParam("session")
	if sessionID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "session parameter required"})
	}

	session, err := h.sessionManager.GetSession(sessionID)
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "session not found"})
	}

	if session.Runtime.Debugger == nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "debugger not initialized"})
	}

	debugger := session.Runtime.Debugger
	file, line := debugger.GetCurrentPosition()

	// Get current scope variables
	var variables map[string]chariot.Value
	if session.Runtime.GetCurrentScope() != nil {
		variables = session.Runtime.GetCurrentScope().AllVars()
	}

	response := DebugStateResponse{
		State:       string(debugger.GetState()),
		Breakpoints: debugger.GetBreakpoints(),
		CallStack:   debugger.GetCallStack(),
		CurrentFile: file,
		CurrentLine: line,
		Variables:   variables,
	}

	return c.JSON(http.StatusOK, response)
}

// DebugEvents establishes a WebSocket connection for real-time debug events
func (h *Handlers) DebugEvents(c echo.Context) error {
	sessionID := c.QueryParam("session")
	if sessionID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "session parameter required"})
	}

	session, err := h.sessionManager.GetSession(sessionID)
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "session not found"})
	}

	// Ensure debugger is enabled
	if session.Runtime.Debugger == nil {
		session.Runtime.Debugger = chariot.NewDebugger()
	}

	// Upgrade HTTP connection to WebSocket
	ws, err := upgrader.Upgrade(c.Response(), c.Request(), nil)
	if err != nil {
		return err
	}
	defer ws.Close()

	debugger := session.Runtime.Debugger
	eventChannel := debugger.GetEventChannel()

	// Send events to WebSocket client
	for event := range eventChannel {
		if err := ws.WriteJSON(event); err != nil {
			// Client disconnected or error writing
			break
		}
	}

	return nil
}

// DebugVariables returns variables in the current scope
func (h *Handlers) DebugVariables(c echo.Context) error {
	sessionID := c.QueryParam("session")
	if sessionID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "session parameter required"})
	}

	session, err := h.sessionManager.GetSession(sessionID)
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "session not found"})
	}

	// Get scope level from query parameter (default: current)
	scopeLevel := 0
	if levelStr := c.QueryParam("level"); levelStr != "" {
		var parseErr error
		scopeLevel, parseErr = strconv.Atoi(levelStr)
		if parseErr != nil {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid scope level"})
		}
	}

	scope := session.Runtime.GetCurrentScope()

	// Navigate to requested scope level
	for i := 0; i < scopeLevel && scope != nil; i++ {
		scope = scope.GetParent()
	}

	if scope == nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "scope level not found"})
	}

	variables := scope.AllVars()

	return c.JSON(http.StatusOK, variables)
}
