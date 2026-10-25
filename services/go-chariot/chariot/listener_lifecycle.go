package chariot

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"sync"
	"sync/atomic"
	"time"

	cfg "github.com/bhouse1273/chariot-ecosystem/services/go-chariot/configs"
)

func listenerShutdownGrace() time.Duration {
	if cfg.ChariotConfig.ShutdownTimeout > 0 {
		return time.Duration(cfg.ChariotConfig.ShutdownTimeout) * time.Second
	}
	return 30 * time.Second
}

// ListenerKind identifies the transport used by a live runtime listener.
type ListenerKind string

const (
	ListenerHTTP ListenerKind = "http"
	ListenerNSQ  ListenerKind = "nsq"
)

// ListenerStatus is an immutable snapshot of a listener owned by a Runtime.
type ListenerStatus struct {
	ID            string       `json:"id"`
	Kind          ListenerKind `json:"kind"`
	State         string       `json:"state"`
	Address       string       `json:"address,omitempty"`
	Port          int          `json:"port,omitempty"`
	Topic         string       `json:"topic,omitempty"`
	Channel       string       `json:"channel,omitempty"`
	StartedAt     time.Time    `json:"started_at"`
	LastActiveAt  time.Time    `json:"last_active_at,omitempty"`
	LastSuccessAt time.Time    `json:"last_success_at,omitempty"`
	LastError     string       `json:"last_error,omitempty"`
	RequestCount  uint64       `json:"request_count"`
	FailureCount  uint64       `json:"failure_count"`
	InFlightCount int64        `json:"in_flight_count"`
}

type listenerMetrics struct {
	mu            sync.RWMutex
	state         string
	startedAt     time.Time
	lastActiveAt  time.Time
	lastSuccessAt time.Time
	lastError     string
	terminalError error
	requestCount  atomic.Uint64
	failureCount  atomic.Uint64
	inFlightCount atomic.Int64
}

func newListenerMetrics() listenerMetrics {
	return listenerMetrics{state: "starting", startedAt: time.Now().UTC()}
}

func (m *listenerMetrics) setState(state string) {
	m.mu.Lock()
	m.state = state
	m.mu.Unlock()
}

func (m *listenerMetrics) recordStart() {
	m.requestCount.Add(1)
	m.inFlightCount.Add(1)
	m.mu.Lock()
	m.lastActiveAt = time.Now().UTC()
	m.mu.Unlock()
}

func (m *listenerMetrics) recordSuccess() {
	m.inFlightCount.Add(-1)
	m.mu.Lock()
	m.lastSuccessAt = time.Now().UTC()
	m.mu.Unlock()
}

func (m *listenerMetrics) recordFailure(err error) {
	m.failureCount.Add(1)
	m.inFlightCount.Add(-1)
	m.mu.Lock()
	if err != nil {
		m.lastError = err.Error()
	}
	m.mu.Unlock()
}

func (m *listenerMetrics) recordError(err error) {
	m.failureCount.Add(1)
	m.mu.Lock()
	if err != nil {
		m.lastError = err.Error()
	}
	m.mu.Unlock()
}

func (m *listenerMetrics) recordTerminalError(err error) {
	if err == nil {
		return
	}
	m.recordError(err)
	m.mu.Lock()
	m.terminalError = errors.Join(m.terminalError, err)
	m.mu.Unlock()
}

func (m *listenerMetrics) terminalErr() error {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.terminalError
}

func (m *listenerMetrics) snapshot(status ListenerStatus) ListenerStatus {
	m.mu.RLock()
	status.State = m.state
	status.StartedAt = m.startedAt
	status.LastActiveAt = m.lastActiveAt
	status.LastSuccessAt = m.lastSuccessAt
	status.LastError = m.lastError
	m.mu.RUnlock()
	status.RequestCount = m.requestCount.Load()
	status.FailureCount = m.failureCount.Load()
	status.InFlightCount = m.inFlightCount.Load()
	return status
}

// ListListeners returns live status snapshots for listeners owned by the runtime.
func (rt *Runtime) ListListeners() []ListenerStatus {
	statuses := append(listRuntimeHTTPListenerStatuses(rt), listRuntimeNSQListenerStatuses(rt)...)
	sort.Slice(statuses, func(i, j int) bool { return statuses[i].ID < statuses[j].ID })
	return statuses
}

// RequestStopListener accepts an asynchronous graceful stop of an owned listener.
// Unknown IDs fail synchronously. A nil error means accepted, not completed:
// ListListeners retains the listener through handler drain and on_exit execution.
// Repeated requests while it is owned are accepted without repeating shutdown or
// hooks. After completion the ID is no longer owned and requests fail as unknown.
// The first stop initiator owns the shutdown context; language requests use the
// configured shutdown grace independently of the requesting execution's lifetime.
// This operation is safe inside WithExecution and never releases its guard.
func (rt *Runtime) RequestStopListener(id string) error {
	type stoppable interface {
		requestStop(context.Context) bool
		Shutdown(context.Context) error
	}
	accept := func(listener stoppable) {
		ctx, cancel := context.WithTimeout(context.Background(), listenerShutdownGrace())
		if !listener.requestStop(ctx) {
			cancel()
			return
		}
		go func() {
			defer cancel()
			// Shutdown records deadline/transport failures in listener status and
			// retains ownership until the actual drain and hook have completed.
			_ = listener.Shutdown(ctx)
		}()
	}
	for _, listener := range runtimeHTTPListeners(rt) {
		if listener.id() == id {
			accept(listener)
			return nil
		}
	}
	for _, listener := range runtimeNSQListenerSnapshot(rt) {
		if listener.id() == id {
			accept(listener)
			return nil
		}
	}
	return fmt.Errorf("listener %q not found", id)
}

// StopListener gracefully stops one listener owned by the runtime, waiting for
// handler drain and on_exit completion or ctx expiry. Never call it while holding
// WithExecution (including from a handler/hook); use RequestStopListener instead.
func (rt *Runtime) StopListener(ctx context.Context, id string) error {
	if found, err := stopRuntimeHTTPListener(ctx, rt, id); found {
		return err
	}
	if found, err := stopRuntimeNSQListener(ctx, rt, id); found {
		return err
	}
	return fmt.Errorf("listener %q not found", id)
}

// ShutdownListeners gracefully stops every listener owned by the runtime.
// It is safe to call more than once.
func (rt *Runtime) ShutdownListeners(ctx context.Context) error {
	// Start every transport's shutdown even if another one consumes the budget.
	type ownedListener interface {
		Shutdown(context.Context) error
		id() string
	}
	var listeners []ownedListener
	for _, listener := range runtimeHTTPListeners(rt) {
		listeners = append(listeners, listener)
	}
	for _, listener := range runtimeNSQListenerSnapshot(rt) {
		listeners = append(listeners, listener)
	}
	results := make(chan error, len(listeners))
	for _, listener := range listeners {
		go func(listener ownedListener) {
			err := listener.Shutdown(ctx)
			if err != nil {
				err = fmt.Errorf("stop %s: %w", listener.id(), err)
			}
			results <- err
		}(listener)
	}
	var shutdownErrors []error
	for range listeners {
		if err := <-results; err != nil {
			shutdownErrors = append(shutdownErrors, err)
		}
	}
	return errors.Join(shutdownErrors...)
}
