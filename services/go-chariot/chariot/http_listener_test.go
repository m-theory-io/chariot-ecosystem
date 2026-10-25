package chariot

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestListenNamedHandlerRequestAndInventory(t *testing.T) {
	rt := newListenerTestRuntime(t)
	if err := rt.SaveFunction("echoRequest", `function echoRequest(req) { return(req) }`, ""); err != nil {
		t.Fatalf("save handler: %v", err)
	}

	port := availableListenerTestPort(t)
	result, err := rt.ExecProgram(fmt.Sprintf(`listen(%d, "echoRequest", map("bind_address", "127.0.0.1", "methods", array("POST"), "base_path", "/decision"))`, port))
	if err != nil {
		t.Fatalf("listen failed: %v", err)
	}
	resultMap, ok := result.(*MapValue)
	if !ok {
		t.Fatalf("listen result type = %T, want *MapValue", result)
	}
	if status, _ := resultMap.Get("status"); status != Str("listening") {
		t.Fatalf("listen status = %v, want listening", status)
	}
	if address, _ := resultMap.Get("address"); !strings.HasPrefix(string(address.(Str)), "127.0.0.1:") {
		t.Fatalf("listen address = %v, want loopback bind", address)
	}
	if id, _ := resultMap.Get("id"); !strings.HasPrefix(string(id.(Str)), "http:127.0.0.1:") {
		t.Fatalf("listen id = %v, want HTTP listener ID", id)
	}

	request, err := http.NewRequest(http.MethodPost, fmt.Sprintf("http://127.0.0.1:%d/decision/order?id=42", port), strings.NewReader(`{"amount":12.5}`))
	if err != nil {
		t.Fatalf("create request: %v", err)
	}
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("X-Test", "listener")
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatalf("listener was not reachable after listen returned: %v", err)
	}
	defer response.Body.Close()

	var payload map[string]interface{}
	if err := json.NewDecoder(response.Body).Decode(&payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if payload["method"] != http.MethodPost {
		t.Fatalf("method = %v, want POST", payload["method"])
	}
	if payload["path"] != "/decision/order" {
		t.Fatalf("path = %v, want /decision/order", payload["path"])
	}
	query, ok := payload["query"].(map[string]interface{})
	if !ok || query["id"] != "42" {
		t.Fatalf("query = %#v, want id=42", payload["query"])
	}
	body, ok := payload["body"].(map[string]interface{})
	if !ok || body["amount"] != 12.5 {
		t.Fatalf("body = %#v, want decoded JSON", payload["body"])
	}
	if payload["raw_body"] != `{"amount":12.5}` {
		t.Fatalf("raw_body = %v", payload["raw_body"])
	}

	statuses := rt.ListListeners()
	if len(statuses) != 1 {
		t.Fatalf("listener count = %d, want 1", len(statuses))
	}
	listenerStatus := statuses[0]
	if listenerStatus.Kind != ListenerHTTP || listenerStatus.Port != port || listenerStatus.State != "running" {
		t.Fatalf("unexpected listener status: %+v", listenerStatus)
	}
	if !strings.HasPrefix(listenerStatus.Address, "127.0.0.1:") {
		t.Fatalf("listener address = %q, want loopback bind", listenerStatus.Address)
	}
	if listenerStatus.RequestCount != 1 || listenerStatus.InFlightCount != 0 || listenerStatus.FailureCount != 0 {
		t.Fatalf("unexpected listener metrics: %+v", listenerStatus)
	}
	if listenerStatus.LastActiveAt.IsZero() || listenerStatus.LastSuccessAt.IsZero() {
		t.Fatalf("activity timestamps were not recorded: %+v", listenerStatus)
	}
}

func TestListenInlineHandlerAndNoContent(t *testing.T) {
	rt := newListenerTestRuntime(t)
	port := availableListenerTestPort(t)
	_, err := rt.ExecProgram(fmt.Sprintf(`setq(handler, func(req) { return(DBNull) })
listen(%d, handler)`, port))
	if err != nil {
		t.Fatalf("listen with inline handler failed: %v", err)
	}

	response, err := http.Get(fmt.Sprintf("http://127.0.0.1:%d/", port))
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusNoContent {
		t.Fatalf("status = %d, want 204", response.StatusCode)
	}
}

func TestListenBindFailureIsSynchronous(t *testing.T) {
	occupied, err := net.Listen("tcp", ":0")
	if err != nil {
		t.Fatalf("occupy port: %v", err)
	}
	defer occupied.Close()
	port := occupied.Addr().(*net.TCPAddr).Port

	rt := newListenerTestRuntime(t)
	if err := rt.SaveFunction("handler", `function handler(req) { return("ok") }`, ""); err != nil {
		t.Fatalf("save handler: %v", err)
	}
	_, err = rt.ExecProgram(fmt.Sprintf(`listen(%d, "handler")`, port))
	if err == nil {
		t.Fatal("listen succeeded on an occupied port")
	}
	if !strings.Contains(err.Error(), "failed to bind listener") {
		t.Fatalf("bind error = %v, want synchronous bind failure", err)
	}
	if got := len(rt.ListListeners()); got != 0 {
		t.Fatalf("listener count after bind failure = %d, want 0", got)
	}
}

func TestRuntimeListenerStopAndShutdownAreIdempotent(t *testing.T) {
	rt := newListenerTestRuntime(t)
	if err := rt.SaveFunction("handler", `function handler(req) { return("ok") }`, ""); err != nil {
		t.Fatalf("save handler: %v", err)
	}

	ports := []int{availableListenerTestPort(t), availableListenerTestPort(t)}
	for _, port := range ports {
		if _, err := rt.ExecProgram(fmt.Sprintf(`listen(%d, "handler")`, port)); err != nil {
			t.Fatalf("listen on %d: %v", port, err)
		}
	}
	statuses := rt.ListListeners()
	if len(statuses) != 2 {
		t.Fatalf("listener count = %d, want 2", len(statuses))
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := rt.StopListener(ctx, statuses[0].ID); err != nil {
		t.Fatalf("stop one listener: %v", err)
	}
	if got := len(rt.ListListeners()); got != 1 {
		t.Fatalf("listener count after stop = %d, want 1", got)
	}
	if err := rt.ShutdownListeners(ctx); err != nil {
		t.Fatalf("shutdown listeners: %v", err)
	}
	if err := rt.ShutdownListeners(ctx); err != nil {
		t.Fatalf("second shutdown should be idempotent: %v", err)
	}
	if got := len(rt.ListListeners()); got != 0 {
		t.Fatalf("listener count after shutdown = %d, want 0", got)
	}
	if err := rt.StopListener(ctx, "http:missing"); err == nil {
		t.Fatal("stopping an unknown listener succeeded")
	}
}

func TestListenerLifecycleHooksFailStartupAndExitOnce(t *testing.T) {
	rt := newListenerTestRuntime(t)
	rt.Register("failStart", func(args ...Value) (Value, error) {
		return nil, errors.New("start rejected")
	})
	port := availableListenerTestPort(t)
	_, err := startRuntimeListener(rt, listenerOptions{
		Port:           port,
		BindAddress:    "127.0.0.1",
		HandlerName:    "unused",
		OnStartProgram: "failStart()",
	})
	if err == nil || !strings.Contains(err.Error(), "start rejected") {
		t.Fatalf("onStart error = %v, want start rejected", err)
	}
	if got := len(rt.ListListeners()); got != 0 {
		t.Fatalf("listener count after onStart failure = %d, want 0", got)
	}
	rebound, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", port))
	if err != nil {
		t.Fatalf("port was not released after onStart failure: %v", err)
	}
	_ = rebound.Close()

	var exitCalls atomic.Int32
	rt.Register("recordExit", func(args ...Value) (Value, error) {
		exitCalls.Add(1)
		return DBNull, nil
	})
	port = availableListenerTestPort(t)
	listener, err := startRuntimeListener(rt, listenerOptions{
		Port:          port,
		HandlerName:   "unused",
		OnExitProgram: "recordExit()",
	})
	if err != nil {
		t.Fatalf("start listener: %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := listener.Shutdown(ctx); err != nil {
		t.Fatalf("first shutdown: %v", err)
	}
	if err := listener.Shutdown(ctx); err != nil {
		t.Fatalf("second shutdown: %v", err)
	}
	if got := exitCalls.Load(); got != 1 {
		t.Fatalf("onExit calls = %d, want 1", got)
	}

	var failedExitCalls atomic.Int32
	rt.Register("failExit", func(args ...Value) (Value, error) {
		failedExitCalls.Add(1)
		return nil, errors.New("exit rejected")
	})
	failedExitListener, err := startRuntimeListener(rt, listenerOptions{
		Port:          availableListenerTestPort(t),
		HandlerName:   "unused",
		OnExitProgram: "failExit()",
	})
	if err != nil {
		t.Fatalf("start listener with failing exit hook: %v", err)
	}
	if err := failedExitListener.Shutdown(ctx); err == nil || !strings.Contains(err.Error(), "exit rejected") {
		t.Fatalf("shutdown error = %v, want exit hook error", err)
	}
	if err := failedExitListener.Shutdown(ctx); err == nil || !strings.Contains(err.Error(), "exit rejected") {
		t.Fatalf("second shutdown error = %v, want retained exit hook error", err)
	}
	if got := failedExitCalls.Load(); got != 1 {
		t.Fatalf("failing onExit calls = %d, want 1", got)
	}
}

func newListenerTestRuntime(t *testing.T) *Runtime {
	t.Helper()
	rt := NewRuntime()
	RegisterAll(rt)
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		if err := rt.ShutdownListeners(ctx); err != nil {
			t.Errorf("cleanup listeners: %v", err)
		}
	})
	return rt
}

func availableListenerTestPort(t *testing.T) int {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("allocate port: %v", err)
	}
	port := listener.Addr().(*net.TCPAddr).Port
	if err := listener.Close(); err != nil {
		t.Fatalf("release allocated port: %v", err)
	}
	return port
}
