package handlers

import (
	"encoding/json"
	"net/http"
	"time"

	ch "github.com/bhouse1273/chariot-ecosystem/services/go-chariot/chariot"
	"github.com/gorilla/websocket"
	"github.com/labstack/echo/v4"
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
	// Reuse dashboard upgrader policy
	conn, err := wsUpgrader.Upgrade(c.Response(), c.Request(), nil)
	if err != nil {
		return err
	}
	defer conn.Close()

	chEvents := make(chan ch.AgentEvent, 128)
	unsubscribe := ch.RegisterAgentEventSink(chEvents)
	defer unsubscribe()

	// Simple writer loop
	for ev := range chEvents {
		payload, _ := json.Marshal(ev)
		if err := conn.WriteMessage(websocket.TextMessage, payload); err != nil {
			break
		}
	}
	return nil
}
