package chariot

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/bhouse1273/chariot-ecosystem/services/go-chariot/logs"
	"github.com/nsqio/go-nsq"
)

func awaitLifecycle(t *testing.T, done <-chan struct{}) {
	t.Helper()
	select {
	case <-done:
	case <-time.After(3 * time.Second):
		t.Fatal("lifecycle operation did not complete")
	}
}

func blockedHTTPListener(t *testing.T, rt *Runtime, exit string) (*runtimeListener, func(), <-chan struct{}) {
	t.Helper()
	entered, release, requestDone := make(chan struct{}), make(chan struct{}), make(chan struct{})
	var once sync.Once
	unblock := func() { once.Do(func() { close(release) }) }
	t.Cleanup(unblock)
	rt.Register("blockRequest", func(...Value) (Value, error) {
		close(entered)
		<-release
		return Str("drained"), nil
	})
	if err := rt.SaveFunction("blockedHandler", `function blockedHandler(req) { return(blockRequest()) }`, ""); err != nil {
		t.Fatal(err)
	}
	l, err := startRuntimeListener(rt, listenerOptions{
		Port: availableListenerTestPort(t), BindAddress: "127.0.0.1",
		HandlerName: "blockedHandler", OnExitProgram: exit,
	})
	if err != nil {
		t.Fatal(err)
	}
	go func() {
		defer close(requestDone)
		client := &http.Client{Timeout: 3 * time.Second}
		response, err := client.Get("http://" + l.netListener.Addr().String())
		if err == nil {
			_ = response.Body.Close()
		}
	}()
	awaitLifecycle(t, entered)
	return l, unblock, requestDone
}

func TestHTTPConcurrentShutdownWaitsForDrainAndExit(t *testing.T) {
	rt := newListenerTestRuntime(t)
	var exits atomic.Int32
	exitEntered, exitRelease := make(chan struct{}), make(chan struct{})
	var exitOnce sync.Once
	releaseExit := func() { exitOnce.Do(func() { close(exitRelease) }) }
	t.Cleanup(releaseExit)
	rt.Register("drainedExit", func(...Value) (Value, error) {
		exits.Add(1)
		statuses := rt.ListListeners()
		if len(statuses) != 1 || statuses[0].InFlightCount != 0 {
			t.Errorf("hook ran before drain or lost ownership: %+v", statuses)
		}
		close(exitEntered)
		<-exitRelease
		return DBNull, nil
	})
	l, release, requestDone := blockedHTTPListener(t, rt, "drainedExit()")
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	results := make(chan error, 12)
	for i := 0; i < cap(results); i++ {
		go func() { results <- l.Shutdown(ctx) }()
	}
	// Accept-loop completion must not imply listener completion.
	deadline := time.Now().Add(time.Second)
	for !l.serverIsDraining() {
		if time.Now().After(deadline) {
			t.Fatal("server did not enter drain")
		}
		time.Sleep(time.Millisecond)
	}
	select {
	case <-l.done:
		t.Fatal("done closed while handler was active")
	case err := <-results:
		t.Fatalf("shutdown returned before drain: %v", err)
	default:
	}
	if statuses := rt.ListListeners(); len(statuses) != 1 || statuses[0].InFlightCount != 1 || exits.Load() != 0 {
		t.Fatalf("active ownership lost: %+v exits=%d", statuses, exits.Load())
	}
	release()
	awaitLifecycle(t, requestDone)
	awaitLifecycle(t, exitEntered)
	if len(rt.ListListeners()) != 1 {
		t.Fatal("ownership lost while exit hook was active")
	}
	releaseExit()
	for i := 0; i < cap(results); i++ {
		if err := <-results; err != nil {
			t.Fatal(err)
		}
	}
	if exits.Load() != 1 || len(rt.ListListeners()) != 0 {
		t.Fatalf("exit calls=%d listeners=%+v", exits.Load(), rt.ListListeners())
	}
}

func (l *runtimeListener) serverIsDraining() bool {
	l.handlersMu.Lock()
	defer l.handlersMu.Unlock()
	return l.draining
}

func TestCloseRuntimeDeadlineForceClosesButRetainsActiveOwnership(t *testing.T) {
	rt := newListenerTestRuntime(t)
	id := fmt.Sprintf("hardening-%d", time.Now().UnixNano())
	RegisterRuntime(id, rt)
	t.Cleanup(func() { _ = CloseRuntime(context.Background(), id) })
	var exits atomic.Int32
	rt.Register("countExit", func(...Value) (Value, error) { exits.Add(1); return DBNull, nil })
	l, release, requestDone := blockedHTTPListener(t, rt, "countExit()")
	ctx, cancel := context.WithTimeout(context.Background(), 25*time.Millisecond)
	defer cancel()
	if err := CloseRuntime(ctx, id); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("close error=%v", err)
	}
	// The request must lose its socket even though its host function ignores cancellation.
	awaitLifecycle(t, requestDone)
	if registered, ok := GetRuntimeByID(id); !ok || registered != rt {
		t.Fatal("live runtime ownership was dropped")
	}
	statuses := rt.ListListeners()
	if len(statuses) != 1 || statuses[0].State != "failed" || statuses[0].InFlightCount != 1 || exits.Load() != 0 {
		t.Fatalf("failed live resource not visible: %+v exits=%d", statuses, exits.Load())
	}
	release()
	awaitLifecycle(t, l.done)
	if err := l.Shutdown(context.Background()); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("terminal error not retained: %v", err)
	}
	if err := CloseRuntime(context.Background(), id); err != nil {
		t.Fatal(err)
	}
	if _, ok := GetRuntimeByID(id); ok || exits.Load() != 1 {
		t.Fatalf("registry retained drained runtime or hook repeated: registered=%v exits=%d", ok, exits.Load())
	}
}

func TestHTTPAcceptFailureRetainsOwnershipUntilHandlerExits(t *testing.T) {
	rt := newListenerTestRuntime(t)
	l, release, requestDone := blockedHTTPListener(t, rt, "")
	_ = l.netListener.Close()
	awaitLifecycle(t, requestDone)
	statuses := rt.ListListeners()
	if len(statuses) != 1 || statuses[0].State != "failed" {
		t.Fatalf("accept failure disappeared before drain: %+v", statuses)
	}
	release()
	awaitLifecycle(t, l.done)
	if err := l.Shutdown(context.Background()); err == nil {
		t.Fatal("accept failure not retained")
	}
}

type lifecycleCloser struct{ calls atomic.Int32 }

func (c *lifecycleCloser) Close() error { c.calls.Add(1); return nil }

func TestSessionDisposalDrainsListenersBeforeHooksAndResources(t *testing.T) {
	sm := NewSessionManager(time.Minute, time.Minute)
	t.Cleanup(sm.Stop)
	logger := logs.NewZapLogger()
	defer logger.Sync()
	s := sm.NewSession("user", logger, "session")
	resource := &lifecycleCloser{}
	s.AddResource("resource", resource)
	var starts, exits atomic.Int32
	s.Runtime.Register("sessionStart", func(...Value) (Value, error) { starts.Add(1); return DBNull, nil })
	s.Runtime.Register("sessionExit", func(...Value) (Value, error) {
		exits.Add(1)
		if len(s.Runtime.ListListeners()) != 0 || resource.calls.Load() != 0 {
			t.Error("session hook ran before drain or after resource teardown")
		}
		return DBNull, nil
	})
	s.SetOnStart("sessionStart()")
	s.SetOnExit("sessionExit()")
	s.Run()
	s.Run()
	awaitLifecycle(t, s.startDone)
	l, release, requestDone := blockedHTTPListener(t, s.Runtime, "")
	ctx, cancel := context.WithTimeout(context.Background(), 25*time.Millisecond)
	defer cancel()
	if err := sm.EndSessionContext(ctx, s.ID); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("end session error=%v", err)
	}
	awaitLifecycle(t, requestDone)
	if _, ok := sm.LookupSession(s.ID); !ok || s.Runtime == nil || resource.calls.Load() != 0 || exits.Load() != 0 {
		t.Fatal("session lost live ownership or cleaned up before drain")
	}
	release()
	awaitLifecycle(t, l.done)
	awaitLifecycle(t, s.disposeDone)
	_ = sm.EndSessionContext(context.Background(), s.ID)
	s.Cleanup()
	if sm.GetActiveSessions() != 0 || starts.Load() != 1 || exits.Load() != 1 || resource.calls.Load() != 1 {
		t.Fatalf("sessions=%d starts=%d exits=%d closes=%d", sm.GetActiveSessions(), starts.Load(), exits.Load(), resource.calls.Load())
	}
}

func TestSessionManagerConcurrentStopCleansOwnedListeners(t *testing.T) {
	sm := NewSessionManager(time.Minute, time.Minute)
	logger := logs.NewZapLogger()
	defer logger.Sync()
	for _, token := range []string{"one", "two"} {
		s := sm.NewSession("user", logger, token)
		if _, err := startRuntimeListener(s.Runtime, listenerOptions{Port: availableListenerTestPort(t), HandlerName: "unused"}); err != nil {
			t.Fatal(err)
		}
	}
	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() { defer wg.Done(); sm.Stop() }()
	}
	wg.Wait()
	if sm.GetActiveSessions() != 0 {
		t.Fatal("manager stop retained disposed sessions")
	}
	if s := sm.NewSession("user", logger, "late"); s != nil {
		t.Fatal("manager admitted a session after stop")
	}
}

func TestCanonicalExecutionGuardSerializesTransportsAndOuterWork(t *testing.T) {
	rt := newListenerTestRuntime(t)
	var active atomic.Int32
	rt.Register("guardedWork", func(...Value) (Value, error) {
		if active.Add(1) != 1 {
			t.Error("shared runtime execution overlapped")
		}
		defer active.Add(-1)
		// A nested interpreter call must not try to reacquire the outer guard.
		return rt.ExecProgram(`"nested"`)
	})
	if err := rt.SaveFunction("guarded", `function guarded(req) { return(guardedWork()) }`, ""); err != nil {
		t.Fatal(err)
	}
	httpListener := &runtimeListener{runtime: rt, opts: listenerOptions{HandlerName: "guarded"}}
	nsqListener := &nsqListener{runtime: rt}
	var wg sync.WaitGroup
	for i := 0; i < 30; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			var err error
			switch i % 3 {
			case 0:
				_, err = httpListener.invokeHandler(DBNull)
			case 1:
				_, err = nsqListener.invokeHandler(&nsqHandlerBinding{HandlerName: "guarded"}, DBNull)
			default:
				err = rt.WithExecution(func() error { _, err := rt.ExecProgram(`guardedWork()`); return err })
			}
			if err != nil {
				t.Errorf("guarded execution: %v", err)
			}
		}(i)
	}
	done := make(chan struct{})
	go func() { wg.Wait(); close(done) }()
	awaitLifecycle(t, done)
}

func TestExpiredSessionCleansListenersWithoutRun(t *testing.T) {
	sm := NewSessionManager(time.Minute, time.Hour)
	defer sm.Stop()
	logger := logs.NewZapLogger()
	defer logger.Sync()
	s := sm.NewSession("user", logger, "expired", -time.Second)
	var exits atomic.Int32
	s.Runtime.Register("expiredExit", func(...Value) (Value, error) { exits.Add(1); return DBNull, nil })
	s.SetOnExit("expiredExit()")
	if _, err := startRuntimeListener(s.Runtime, listenerOptions{Port: availableListenerTestPort(t), HandlerName: "unused"}); err != nil {
		t.Fatal(err)
	}
	sm.CleanupExpiredSessions()
	s.Run() // Must not resurrect a disposed session or run hooks again.
	s.Cleanup()
	late := &lifecycleCloser{}
	s.AddResource("late", late)
	if sm.GetActiveSessions() != 0 || len(s.Runtime.ListListeners()) != 0 || exits.Load() != 1 || late.calls.Load() != 1 {
		t.Fatal("expiry/direct cleanup leaked resources or repeated hooks")
	}
}

func TestNSQConcurrentShutdownOwnsListenerThroughExitHook(t *testing.T) {
	rt := newListenerTestRuntime(t)
	consumer, err := nsq.NewConsumer("lifecycle", "hardening", nsq.NewConfig())
	if err != nil {
		t.Fatal(err)
	}
	consumer.AddHandler(nsq.HandlerFunc(func(*nsq.Message) error { return nil }))
	entered, release := make(chan struct{}), make(chan struct{})
	var once sync.Once
	unblock := func() { once.Do(func() { close(release) }) }
	defer unblock()
	var exits atomic.Int32
	rt.Register("nsqExit", func(...Value) (Value, error) {
		exits.Add(1)
		close(entered)
		<-release
		return nil, errors.New("NSQ exit failed")
	})
	l := &nsqListener{runtime: rt, consumer: consumer, metrics: newListenerMetrics(), done: make(chan struct{}),
		opts: nsqListenerOptions{Topic: "lifecycle", Channel: "hardening", OnExitProgram: "nsqExit()"}}
	if err := registerRuntimeNSQListener(rt, l); err != nil {
		t.Fatal(err)
	}
	go l.waitForStop()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	results := make(chan error, 8)
	for i := 0; i < cap(results); i++ {
		go func() { results <- l.Shutdown(ctx) }()
	}
	awaitLifecycle(t, entered)
	if len(rt.ListListeners()) != 1 {
		t.Fatal("NSQ ownership dropped during exit hook")
	}
	unblock()
	for i := 0; i < cap(results); i++ {
		if err := <-results; err == nil {
			t.Fatal("NSQ exit failure was lost")
		}
	}
	if exits.Load() != 1 || len(rt.ListListeners()) != 0 || l.status().State != "failed" {
		t.Fatalf("NSQ finalization: exits=%d status=%+v", exits.Load(), l.status())
	}
}
