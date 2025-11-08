package handlers

import (
	"sync"
	"time"

	"github.com/bhouse1273/chariot-ecosystem/services/go-chariot/chariot"
	"github.com/google/uuid"
)

// ExecutionManager manages all active and recent script executions
type ExecutionManager struct {
	contexts sync.Map // map[string]*ExecutionContext
	mu       sync.RWMutex
}

// NewExecutionManager creates a new execution manager
func NewExecutionManager() *ExecutionManager {
	mgr := &ExecutionManager{}
	// Start cleanup goroutine to remove old executions
	go mgr.cleanupLoop()
	return mgr
}

// Create creates a new execution context
func (m *ExecutionManager) Create(userID, program string) *ExecutionContext {
	ctx := &ExecutionContext{
		ID:        uuid.New().String(),
		UserID:    userID,
		Program:   program,
		StartedAt: time.Now(),
		LogBuffer: NewLogBuffer(1000), // Max 1000 log entries
		Done:      false,
		doneChan:  make(chan struct{}),
	}
	m.contexts.Store(ctx.ID, ctx)
	return ctx
}

// Get retrieves an execution context by ID
func (m *ExecutionManager) Get(execID string) *ExecutionContext {
	val, ok := m.contexts.Load(execID)
	if !ok {
		return nil
	}
	return val.(*ExecutionContext)
}

// Remove removes an execution context
func (m *ExecutionManager) Remove(execID string) {
	m.contexts.Delete(execID)
}

// cleanupLoop removes executions older than 5 minutes that are completed
func (m *ExecutionManager) cleanupLoop() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		now := time.Now()
		m.contexts.Range(func(key, value interface{}) bool {
			ctx := value.(*ExecutionContext)
			ctx.mu.RLock()
			isDone := ctx.Done
			completedAt := ctx.CompletedAt
			ctx.mu.RUnlock()

			// Remove if completed more than 5 minutes ago
			if isDone && !completedAt.IsZero() && now.Sub(completedAt) > 5*time.Minute {
				m.contexts.Delete(key)
			}
			return true
		})
	}
}

// ExecutionContext holds the state of a single script execution
type ExecutionContext struct {
	ID          string
	UserID      string
	Program     string
	StartedAt   time.Time
	CompletedAt time.Time

	LogBuffer *LogBuffer
	Result    interface{}
	Error     error
	Done      bool
	doneChan  chan struct{}

	mu sync.RWMutex
}

// MarkDone marks the execution as complete
func (ctx *ExecutionContext) MarkDone(result interface{}, err error) {
	ctx.mu.Lock()
	defer ctx.mu.Unlock()

	ctx.Result = result
	ctx.Error = err
	ctx.Done = true
	ctx.CompletedAt = time.Now()
	close(ctx.doneChan)
}

// IsDone returns whether the execution is complete
func (ctx *ExecutionContext) IsDone() bool {
	ctx.mu.RLock()
	defer ctx.mu.RUnlock()
	return ctx.Done
}

// DoneChan returns a channel that closes when execution completes
func (ctx *ExecutionContext) DoneChan() <-chan struct{} {
	return ctx.doneChan
}

// GetResult returns the result and error (safe for concurrent access)
func (ctx *ExecutionContext) GetResult() (interface{}, error) {
	ctx.mu.RLock()
	defer ctx.mu.RUnlock()
	return ctx.Result, ctx.Error
}

// LogBuffer is a thread-safe circular buffer for log entries
type LogBuffer struct {
	entries     []chariot.LogEntry
	maxSize     int
	subscribers []chan chariot.LogEntry
	mu          sync.RWMutex
}

// NewLogBuffer creates a new log buffer
func NewLogBuffer(maxSize int) *LogBuffer {
	return &LogBuffer{
		entries:     make([]chariot.LogEntry, 0, maxSize),
		maxSize:     maxSize,
		subscribers: make([]chan chariot.LogEntry, 0),
	}
}

// Append adds a log entry and notifies subscribers
func (lb *LogBuffer) Append(entry chariot.LogEntry) {
	lb.mu.Lock()
	defer lb.mu.Unlock()

	// Add to buffer (circular)
	if len(lb.entries) >= lb.maxSize {
		// Remove oldest entry
		lb.entries = lb.entries[1:]
	}
	lb.entries = append(lb.entries, entry)

	// Notify all subscribers (non-blocking)
	for _, ch := range lb.subscribers {
		select {
		case ch <- entry:
		default:
			// Subscriber too slow, skip
		}
	}
}

// GetAll returns all buffered log entries
func (lb *LogBuffer) GetAll() []chariot.LogEntry {
	lb.mu.RLock()
	defer lb.mu.RUnlock()

	// Return a copy to avoid race conditions
	result := make([]chariot.LogEntry, len(lb.entries))
	copy(result, lb.entries)
	return result
}

// Subscribe creates a new subscriber channel for real-time log streaming
func (lb *LogBuffer) Subscribe() chan chariot.LogEntry {
	lb.mu.Lock()
	defer lb.mu.Unlock()

	ch := make(chan chariot.LogEntry, 100) // Buffer to handle bursts
	lb.subscribers = append(lb.subscribers, ch)
	return ch
}

// Unsubscribe removes a subscriber channel
func (lb *LogBuffer) Unsubscribe(ch chan chariot.LogEntry) {
	lb.mu.Lock()
	defer lb.mu.Unlock()

	for i, sub := range lb.subscribers {
		if sub == ch {
			lb.subscribers = append(lb.subscribers[:i], lb.subscribers[i+1:]...)
			close(ch)
			break
		}
	}
}

// LogEntry represents a single log message
// DEPRECATED: Use chariot.LogEntry instead
type LogEntry = chariot.LogEntry
