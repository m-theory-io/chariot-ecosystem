package chariot

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"
)

func TestNSQListenerInventory(t *testing.T) {
	rt := newListenerTestRuntime(t)
	listener := &nsqListener{
		runtime: rt,
		opts: nsqListenerOptions{
			Topic:       "inventory_topic",
			Channel:     "inventory_channel",
			NSQDAddress: "127.0.0.1:4150",
		},
		metrics: newListenerMetrics(),
		done:    make(chan struct{}),
	}
	if err := registerRuntimeNSQListener(rt, listener); err != nil {
		t.Fatalf("register NSQ listener: %v", err)
	}
	t.Cleanup(func() { unregisterRuntimeNSQListener(rt, listenerKey(listener.opts.Topic, listener.opts.Channel)) })
	listener.metrics.setState("running")

	statuses := rt.ListListeners()
	if len(statuses) != 1 {
		t.Fatalf("listener count = %d, want 1", len(statuses))
	}
	status := statuses[0]
	if status.ID != "nsq:inventory_topic:inventory_channel" || status.Kind != ListenerNSQ || status.State != "running" {
		t.Fatalf("unexpected NSQ listener status: %+v", status)
	}
	unregisterRuntimeNSQListener(rt, listenerKey(listener.opts.Topic, listener.opts.Channel))
}

func TestCloseRuntimeStopsOwnedListeners(t *testing.T) {
	rt := NewRuntime()
	RegisterAll(rt)
	id := fmt.Sprintf("listener-close-%d", time.Now().UnixNano())
	RegisterRuntime(id, rt)

	if err := rt.SaveFunction("handler", `function handler(req) { return("ok") }`, ""); err != nil {
		t.Fatalf("save handler: %v", err)
	}
	rt.Register("failExit", func(args ...Value) (Value, error) {
		return nil, errors.New("runtime exit rejected")
	})
	port := availableListenerTestPort(t)
	if _, err := rt.ExecProgram(fmt.Sprintf(`listen(%d, "handler", map("on_exit", "failExit()"))`, port)); err != nil {
		t.Fatalf("listen: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := CloseRuntime(ctx, id); err == nil || !strings.Contains(err.Error(), "runtime exit rejected") {
		t.Fatalf("close runtime error = %v, want exit-hook error", err)
	}
	if _, exists := GetRuntimeByID(id); exists {
		t.Fatal("runtime remained registered after CloseRuntime")
	}
	if got := len(rt.ListListeners()); got != 0 {
		t.Fatalf("owned listener count after CloseRuntime = %d, want 0", got)
	}
}
