package chariot

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/nsqio/go-nsq"
)

// Exercise the actual MCP execution shape, not a direct builtin call: the
// non-reentrant outer guard must remain held until ExecProgram has returned.
func TestGuardedListenerStopAcceptsIdleListenerWithExitHook(t *testing.T) {
	for _, kind := range []ListenerKind{ListenerHTTP, ListenerNSQ} {
		t.Run(string(kind), func(t *testing.T) {
			rt := newListenerTestRuntime(t)
			exitEntered, exitRelease := make(chan struct{}), make(chan struct{})
			guardRelease, accepted := make(chan struct{}), make(chan struct{})
			var exitOnce, guardOnce sync.Once
			releaseExit := func() { exitOnce.Do(func() { close(exitRelease) }) }
			releaseGuard := func() { guardOnce.Do(func() { close(guardRelease) }) }
			t.Cleanup(releaseExit)
			t.Cleanup(releaseGuard)
			var exits atomic.Int32
			rt.Register("stopRequestExit", func(...Value) (Value, error) {
				exits.Add(1)
				close(exitEntered)
				<-exitRelease
				return DBNull, nil
			})

			var id string
			var done <-chan struct{}
			if kind == ListenerHTTP {
				l, err := startRuntimeListener(rt, listenerOptions{
					Port: availableListenerTestPort(t), BindAddress: "127.0.0.1",
					HandlerName: "unused", OnExitProgram: "stopRequestExit()",
				})
				if err != nil {
					t.Fatal(err)
				}
				id, done = l.id(), l.done
			} else {
				// A real go-nsq consumer without a broker is sufficient to test
				// StopChan, runtime ownership, and guarded lifecycle hooks.
				consumer, err := nsq.NewConsumer("stop_request", "idle", nsq.NewConfig())
				if err != nil {
					t.Fatal(err)
				}
				consumer.AddHandler(nsq.HandlerFunc(func(*nsq.Message) error { return nil }))
				l := &nsqListener{runtime: rt, consumer: consumer, metrics: newListenerMetrics(), done: make(chan struct{}),
					opts: nsqListenerOptions{Topic: "stop_request", Channel: "idle", OnExitProgram: "stopRequestExit()"}}
				if err := registerRuntimeNSQListener(rt, l); err != nil {
					t.Fatal(err)
				}
				l.metrics.setState("running")
				go l.waitForStop()
				id, done = l.id(), l.done
			}

			result := make(chan error, 1)
			go func() {
				result <- rt.WithExecution(func() error {
					for i := 0; i < 2; i++ {
						value, err := rt.ExecProgram(fmt.Sprintf(`listenerStop(%q)`, id))
						if err != nil || value != Bool(true) {
							return fmt.Errorf("accepted result=%v error=%v", value, err)
						}
					}
					status, err := rt.ExecProgram(fmt.Sprintf(`listenerStatus(%q)`, id))
					if err != nil {
						return err
					}
					if m, ok := status.(*MapValue); !ok || m.Values["state"] != Str("stopping") {
						return fmt.Errorf("accepted stop status=%v", status)
					}
					close(accepted)
					<-guardRelease
					return nil
				})
			}()
			select {
			case <-accepted:
			case err := <-result:
				t.Fatalf("guarded stop failed: %v", err)
			case <-time.After(2 * time.Second):
				t.Fatal("guarded listenerStop waited for its own exit hook")
			}
			select {
			case <-exitEntered:
				t.Fatal("exit hook ran while the requesting execution still owned the guard")
			default:
			}
			releaseGuard()
			if err := <-result; err != nil {
				t.Fatal(err)
			}
			awaitLifecycle(t, exitEntered)

			// Concurrent requests must not repeat shutdown/hooks or cancel the
			// first request's context. A Go stop still waits for the hook.
			var requests sync.WaitGroup
			for i := 0; i < 12; i++ {
				requests.Add(1)
				go func() {
					defer requests.Done()
					if err := rt.RequestStopListener(id); err != nil {
						t.Errorf("repeat accepted stop: %v", err)
					}
				}()
			}
			requests.Wait()
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()
			waiting := make(chan error, 1)
			go func() { waiting <- rt.StopListener(ctx, id) }()
			select {
			case err := <-waiting:
				t.Fatalf("Go stop returned before hook completion: %v", err)
			case <-time.After(20 * time.Millisecond):
			}
			if statuses := rt.ListListeners(); len(statuses) != 1 || statuses[0].State != "stopping" || statuses[0].InFlightCount != 0 {
				t.Fatalf("outstanding exit hook disappeared: %+v", statuses)
			}
			releaseExit()
			if err := <-waiting; err != nil {
				t.Fatal(err)
			}
			awaitLifecycle(t, done)
			if exits.Load() != 1 || len(rt.ListListeners()) != 0 {
				t.Fatalf("exit calls=%d listeners=%+v", exits.Load(), rt.ListListeners())
			}
			if err := rt.WithExecution(func() error {
				_, err := rt.ExecProgram(fmt.Sprintf(`listenerStop(%q)`, id))
				if err == nil || !strings.Contains(err.Error(), "not found") {
					return fmt.Errorf("removed ID must fail synchronously: %v", err)
				}
				return nil
			}); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestHTTPHandlerListenerStopAcceptsSelfAndOther(t *testing.T) {
	for _, self := range []bool{true, false} {
		t.Run(fmt.Sprintf("self=%v", self), func(t *testing.T) {
			rt := newListenerTestRuntime(t)
			var exits atomic.Int32
			rt.Register("handlerStopExit", func(...Value) (Value, error) {
				exits.Add(1)
				return DBNull, nil
			})
			port := availableListenerTestPort(t)
			id := fmt.Sprintf("http:127.0.0.1:%d", port)
			if !self {
				target, err := startRuntimeListener(rt, listenerOptions{
					Port: availableListenerTestPort(t), BindAddress: "127.0.0.1",
					HandlerName: "unused", OnExitProgram: "handlerStopExit()",
				})
				if err != nil {
					t.Fatal(err)
				}
				id = target.id()
			}
			rt.Register("observeAcceptedStop", func(...Value) (Value, error) {
				for _, status := range rt.ListListeners() {
					if status.ID == id {
						if status.State != "stopping" || (self && status.InFlightCount != 1) || exits.Load() != 0 {
							return nil, fmt.Errorf("incorrect outstanding stop: %+v exits=%d", status, exits.Load())
						}
						return Bool(true), nil
					}
				}
				return nil, errors.New("target ownership lost before handler returned")
			})
			if err := rt.SaveFunction("stoppingHandler", fmt.Sprintf(`function stoppingHandler(req) {
				listenerStop(%q)
				listenerStop(%q)
				return(observeAcceptedStop())
			}`, id, id), ""); err != nil {
				t.Fatal(err)
			}
			exitProgram := ""
			if self {
				exitProgram = "handlerStopExit()"
			}
			if _, err := startRuntimeListener(rt, listenerOptions{
				Port: port, BindAddress: "127.0.0.1", HandlerName: "stoppingHandler", OnExitProgram: exitProgram,
			}); err != nil {
				t.Fatal(err)
			}
			client := &http.Client{Timeout: 2 * time.Second}
			response, err := client.Get(fmt.Sprintf("http://127.0.0.1:%d/", port))
			if err != nil {
				t.Fatalf("stop handler failed to return its response: %v", err)
			}
			defer response.Body.Close()
			var accepted bool
			if err := json.NewDecoder(response.Body).Decode(&accepted); err != nil || !accepted || response.StatusCode != http.StatusOK {
				t.Fatalf("accepted response=%v status=%d error=%v", accepted, response.StatusCode, err)
			}
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()
			if err := rt.ShutdownListeners(ctx); err != nil {
				t.Fatalf("wait for handler stop: %v", err)
			}
			if exits.Load() != 1 {
				t.Fatalf("exit hook count=%d", exits.Load())
			}
		})
	}
}
