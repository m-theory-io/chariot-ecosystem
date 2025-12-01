package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	ch "github.com/bhouse1273/chariot-ecosystem/services/go-chariot/chariot"
	cfg "github.com/bhouse1273/chariot-ecosystem/services/go-chariot/configs"
	"github.com/gorilla/websocket"
	"github.com/labstack/echo/v4"
	"go.uber.org/zap"
)

type agentStartReq struct {
	Name          string `json:"name"`
	PlanVar       string `json:"planVar"`
	MaxConcurrent int    `json:"maxConcurrent"`
	PollSeconds   int    `json:"pollSeconds"`
}

func (h *Handlers) ListAgents(c echo.Context) error {
	names := ch.DefaultAgentNames()
	return c.JSON(http.StatusOK, ResultJSON{Result: "OK", Data: map[string]any{"agents": names}})
}

func (h *Handlers) StartAgent(c echo.Context) error {
	var req agentStartReq
	if err := c.Bind(&req); err != nil || req.Name == "" || req.PlanVar == "" {
		return c.JSON(http.StatusBadRequest, ResultJSON{Result: "ERROR", Data: "invalid request"})
	}
	// Look up plan by variable name in bootstrap runtime
	v, _ := h.bootstrapRuntime.GetVariable(req.PlanVar)
	pl, ok := v.(*ch.Plan)
	if !ok || pl == nil {
		return c.JSON(http.StatusBadRequest, ResultJSON{Result: "ERROR", Data: "plan variable not found"})
	}
	maxC := req.MaxConcurrent
	if maxC <= 0 {
		maxC = 1
	}
	poll := req.PollSeconds
	if poll <= 0 {
		poll = 3
	}
	if err := ch.DefaultAgentStart(req.Name, h.bootstrapRuntime, pl, maxC, time.Duration(poll)*time.Second); err != nil {
		return c.JSON(http.StatusBadRequest, ResultJSON{Result: "ERROR", Data: err.Error()})
	}
	return c.JSON(http.StatusOK, ResultJSON{Result: "OK", Data: map[string]any{"started": req.Name}})
}

func (h *Handlers) StopAgent(c echo.Context) error {
	name := c.Param("name")
	if name == "" {
		return c.JSON(http.StatusBadRequest, ResultJSON{Result: "ERROR", Data: "missing name"})
	}
	ch.DefaultAgentStop(name)
	return c.JSON(http.StatusOK, ResultJSON{Result: "OK", Data: map[string]any{"stopped": name}})
}

func (h *Handlers) PublishAgent(c echo.Context) error {
	name := c.Param("name")
	if name == "" {
		return c.JSON(http.StatusBadRequest, ResultJSON{Result: "ERROR", Data: "missing name"})
	}
	if ok := ch.DefaultAgentPublish(name); !ok {
		return c.JSON(http.StatusNotFound, ResultJSON{Result: "ERROR", Data: "agent not found"})
	}
	return c.JSON(http.StatusOK, ResultJSON{Result: "OK", Data: map[string]any{"published": name}})
}

type beliefReq struct {
	Key   string      `json:"key"`
	Value interface{} `json:"value"`
}

func (h *Handlers) PutBelief(c echo.Context) error {
	name := c.Param("name")
	var req beliefReq
	if err := c.Bind(&req); err != nil || name == "" || req.Key == "" {
		return c.JSON(http.StatusBadRequest, ResultJSON{Result: "ERROR", Data: "invalid request"})
	}
	val := toChariotValue(req.Value)
	if ok := ch.DefaultAgentBelief(name, req.Key, val); !ok {
		return c.JSON(http.StatusNotFound, ResultJSON{Result: "ERROR", Data: "agent not found"})
	}
	return c.JSON(http.StatusOK, ResultJSON{Result: "OK", Data: map[string]any{"belief": req.Key}})
}

// CreateAgent creates and starts a new BDI agent (plan by name, not variable)
func (h *Handlers) CreateAgent(c echo.Context) error {
	var req struct {
		Name          string  `json:"name"`
		Plan          string  `json:"plan"`
		MaxConcurrent int     `json:"maxConcurrent"`
		PollSeconds   float64 `json:"pollSeconds"`
	}
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, ResultJSON{Result: "error", Data: "invalid request"})
	}

	if req.Name == "" || req.Plan == "" {
		return c.JSON(http.StatusBadRequest, ResultJSON{Result: "error", Data: "name and plan are required"})
	}

	if req.MaxConcurrent <= 0 {
		req.MaxConcurrent = 1
	}
	if req.PollSeconds <= 0 {
		req.PollSeconds = 3.0
	}

	// Get the plan from bootstrap runtime
	planVal, ok := h.bootstrapRuntime.GlobalScope().Get(req.Plan)
	if !ok || planVal == nil {
		return c.JSON(http.StatusNotFound, ResultJSON{Result: "error", Data: fmt.Sprintf("plan '%s' not found", req.Plan)})
	}

	plan, ok := planVal.(*ch.Plan)
	if !ok {
		return c.JSON(http.StatusBadRequest, ResultJSON{Result: "error", Data: fmt.Sprintf("'%s' is not a plan", req.Plan)})
	}

	pollEvery := time.Duration(req.PollSeconds * float64(time.Second))
	err := ch.DefaultAgentStart(req.Name, h.bootstrapRuntime, plan, req.MaxConcurrent, pollEvery)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, ResultJSON{Result: "error", Data: err.Error()})
	}

	cfg.ChariotLogger.Info("Agent created", zap.String("name", req.Name), zap.String("plan", req.Plan))
	return c.JSON(http.StatusOK, ResultJSON{Result: "success", Data: map[string]string{"agent": req.Name}})
}

// SetBelief sets a belief on an agent (JSON body with name, not URL param)
func (h *Handlers) SetBelief(c echo.Context) error {
	var req struct {
		Name  string      `json:"name"`
		Key   string      `json:"key"`
		Value interface{} `json:"value"`
	}
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, ResultJSON{Result: "error", Data: "invalid request"})
	}

	if req.Name == "" || req.Key == "" {
		return c.JSON(http.StatusBadRequest, ResultJSON{Result: "error", Data: "name and key are required"})
	}

	// Convert JSON value to Chariot Value
	val, err := ch.JSONToValue(req.Value)
	if err != nil {
		return c.JSON(http.StatusBadRequest, ResultJSON{Result: "error", Data: fmt.Sprintf("invalid value: %v", err)})
	}

	if !ch.DefaultAgentBelief(req.Name, req.Key, val) {
		return c.JSON(http.StatusNotFound, ResultJSON{Result: "error", Data: fmt.Sprintf("agent '%s' not found", req.Name)})
	}

	cfg.ChariotLogger.Info("Agent belief set", zap.String("name", req.Name), zap.String("key", req.Key))
	return c.JSON(http.StatusOK, ResultJSON{Result: "success", Data: map[string]interface{}{"agent": req.Name, "key": req.Key}})
}

// GetBeliefs returns all beliefs for an agent
func (h *Handlers) GetBeliefs(c echo.Context) error {
	name := c.Param("name")
	if name == "" {
		return c.JSON(http.StatusBadRequest, ResultJSON{Result: "error", Data: "name is required"})
	}

	beliefs := ch.DefaultAgentGetBeliefs(name)
	if beliefs == nil {
		return c.JSON(http.StatusNotFound, ResultJSON{Result: "error", Data: fmt.Sprintf("agent '%s' not found", name)})
	}

	// Convert Chariot Values to JSON-serializable format
	result := make(map[string]interface{})
	for k, v := range beliefs {
		result[k] = ch.ValueToJSON(v)
	}

	return c.JSON(http.StatusOK, ResultJSON{Result: "success", Data: result})
}

// GetAgentInfo returns detailed info about an agent
func (h *Handlers) GetAgentInfo(c echo.Context) error {
	name := c.Param("name")
	if name == "" {
		return c.JSON(http.StatusBadRequest, ResultJSON{Result: "error", Data: "name is required"})
	}

	info := ch.DefaultAgentGetInfo(name)
	if info == nil {
		return c.JSON(http.StatusNotFound, ResultJSON{Result: "error", Data: fmt.Sprintf("agent '%s' not found", name)})
	}

	return c.JSON(http.StatusOK, ResultJSON{Result: "success", Data: info})
}

// RunPlanOnce executes a plan once with custom variables (no persistent agent)
func (h *Handlers) RunPlanOnce(c echo.Context) error {
	var req struct {
		Plan      string                 `json:"plan"`
		Mode      string                 `json:"mode"` // "bdi" or "dry-run"
		VarsMap   map[string]interface{} `json:"varsMap"`
		AgentName string                 `json:"agentName"`
	}
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, ResultJSON{Result: "error", Data: "invalid request"})
	}

	if req.Plan == "" {
		return c.JSON(http.StatusBadRequest, ResultJSON{Result: "error", Data: "plan is required"})
	}

	if req.Mode == "" {
		req.Mode = "bdi"
	}

	// Build the Chariot code to execute
	// Note: req.Plan is the plan variable name - retrieve it from bootstrap globals
	var code strings.Builder
	// First, get the plan object from the bootstrap runtime's global scope
	code.WriteString("setq(__planToRun, getVariable(\"" + req.Plan + "\"))\n")
	code.WriteString("runPlanOnceEx(__planToRun, \"" + req.Mode + "\"")
	if len(req.VarsMap) > 0 {
		// Convert varsMap to Chariot map literal
		code.WriteString(", map(")
		first := true
		for k, v := range req.VarsMap {
			if !first {
				code.WriteString(", ")
			}
			first = false
			code.WriteString(fmt.Sprintf("'%s', ", k))
			// Simple JSON value serialization
			switch val := v.(type) {
			case string:
				code.WriteString(fmt.Sprintf("'%s'", val))
			case float64:
				code.WriteString(fmt.Sprintf("%v", val))
			case bool:
				if val {
					code.WriteString("true")
				} else {
					code.WriteString("false")
				}
			default:
				code.WriteString(fmt.Sprintf("'%v'", val))
			}
		}
		code.WriteString(")")
	}
	code.WriteString(")")

	// Optionally hydrate a named agent's beliefs before executing the plan
	if req.AgentName != "" {
		if info := ch.DefaultAgentGetInfo(req.AgentName); info == nil {
			return c.JSON(http.StatusNotFound, ResultJSON{Result: "error", Data: fmt.Sprintf("agent '%s' not found", req.AgentName)})
		}
		if len(req.VarsMap) > 0 {
			for k, v := range req.VarsMap {
				val := toChariotValue(v)
				if !ch.DefaultAgentBelief(req.AgentName, k, val) {
					return c.JSON(http.StatusInternalServerError, ResultJSON{Result: "error", Data: fmt.Sprintf("failed to set belief '%s' on agent '%s'", k, req.AgentName)})
				}
			}
			cfg.ChariotLogger.Info("RunPlanOnce applied beliefs", zap.String("agent", req.AgentName), zap.Int("count", len(req.VarsMap)))
		}
	}

	// Get session from context to use its runtime (which has all bootstrap globals)
	session := c.Get("session").(*ch.Session)

	// Execute in the session's runtime (where the plan is defined)
	res, err := session.Runtime.ExecProgram(code.String())
	if err != nil {
		cfg.ChariotLogger.Error("RunPlanOnce error", zap.String("plan", req.Plan), zap.Error(err))
		return c.JSON(http.StatusInternalServerError, ResultJSON{Result: "error", Data: err.Error()})
	}

	cfg.ChariotLogger.Info("Plan executed once", zap.String("plan", req.Plan), zap.String("mode", req.Mode))
	return c.JSON(http.StatusOK, ResultJSON{Result: "success", Data: ch.ValueToJSON(res)})
}

// toChariotValue: best-effort conversion from JSON types to chariot.Value
func toChariotValue(v interface{}) ch.Value {
	switch t := v.(type) {
	case nil:
		return nil
	case bool:
		return ch.Bool(t)
	case float64:
		return ch.Number(t)
	case string:
		return ch.Str(t)
	case map[string]interface{}:
		mv := ch.NewMap()
		for k, vv := range t {
			mv.Values[k] = toChariotValue(vv)
		}
		return mv
	case []interface{}:
		arr := ch.NewArray()
		for _, vv := range t {
			arr.Append(toChariotValue(vv))
		}
		return arr
	default:
		b, _ := json.Marshal(v)
		return ch.Str(string(b))
	}
}

// WebSocket: stream agent events
func (h *Handlers) HandleAgentsWS(c echo.Context) error {
	// Upgrade to WebSocket (same Upgrader settings as dashboard)
	conn, err := wsUpgrader.Upgrade(c.Response(), c.Request(), nil)
	if err != nil {
		return err
	}
	defer conn.Close()

	// Subscribe to agent events
	chEvents := make(chan ch.AgentEvent, 128)
	unsubscribe := ch.RegisterAgentEventSink(chEvents)
	defer unsubscribe()

	// Improve stability: handle control frames and keep-alive pings
	conn.SetReadLimit(512)
	conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	conn.SetPongHandler(func(string) error {
		conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	// Reader goroutine: drain incoming messages to process pings/close frames
	done := make(chan struct{})
	go func() {
		defer close(done)
		for {
			if _, _, err := conn.ReadMessage(); err != nil {
				return
			}
		}
	}()

	// Send initial hello so clients immediately see something
	_ = conn.WriteMessage(websocket.TextMessage, []byte(`{"type":"hello","result":"OK","service":"agents"}`))

	// Periodic ping to keep intermediaries happy
	ping := time.NewTicker(30 * time.Second)
	defer ping.Stop()

	// Visible JSON heartbeat in addition to WS ping
	heartbeat := time.NewTicker(15 * time.Second)
	defer heartbeat.Stop()

	for {
		select {
		case ev, ok := <-chEvents:
			if !ok {
				return nil
			}
			payload, _ := json.Marshal(ev)
			if err := conn.WriteMessage(websocket.TextMessage, payload); err != nil {
				return nil
			}
		case <-ping.C:
			_ = conn.WriteControl(websocket.PingMessage, []byte("ping"), time.Now().Add(5*time.Second))
		case <-heartbeat.C:
			// Non-blocking best-effort heartbeat
			_ = conn.WriteMessage(websocket.TextMessage, []byte(`{"type":"heartbeat","ts":`+time.Now().UTC().Format("\"2006-01-02T15:04:05Z07:00\"")+`}`))
		case <-done:
			return nil
		}
	}
}
