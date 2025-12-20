package chariot

import (
	"fmt"
	"sync"
)

const maxPendingEvents = 200

// DebugState represents the current state of the debugger
type DebugState string

const (
	DebugStateRunning  DebugState = "running"
	DebugStatePaused   DebugState = "paused"
	DebugStateStepping DebugState = "stepping"
	DebugStateStopped  DebugState = "stopped"
)

// StepMode defines how the debugger should step through execution
type StepMode string

const (
	StepModeNone StepMode = "none"
	StepModeOver StepMode = "over" // Execute current line, stop at next line in same scope
	StepModeInto StepMode = "into" // Step into function calls
	StepModeOut  StepMode = "out"  // Continue until current function returns
)

// Breakpoint represents a breakpoint location in source code
type Breakpoint struct {
	ID        string `json:"id"`
	File      string `json:"file"`
	Line      int    `json:"line"`
	Enabled   bool   `json:"enabled"`
	Condition string `json:"condition,omitempty"` // Optional condition expression
}

// StackFrame represents a single frame in the call stack
type StackFrame struct {
	FunctionName string           `json:"functionName"`
	File         string           `json:"file"`
	Line         int              `json:"line"`
	Scope        map[string]Value `json:"scope"` // Variables visible in this frame
}

// DebugEvent represents an event that occurred during debugging
type DebugEvent struct {
	Type      string       `json:"type"` // "breakpoint", "step", "error", "stopped"
	File      string       `json:"file,omitempty"`
	Line      int          `json:"line,omitempty"`
	Message   string       `json:"message,omitempty"`
	CallStack []StackFrame `json:"callStack,omitempty"`
}

// Debugger manages debugging state and operations for a Chariot runtime
type Debugger struct {
	mu               sync.RWMutex
	state            DebugState
	stepMode         StepMode
	breakpoints      map[string]*Breakpoint // key: "file:line"
	callStack        []StackFrame
	currentLine      int
	currentFile      string
	stepDepth        int // Tracks call depth for step over/out
	eventSubscribers map[int]chan DebugEvent
	pendingEvents    []DebugEvent
	nextSubscriberID int
	resumeChan       chan struct{} // Signal to resume execution
	lastPausedFile   string        // Track last paused position
	lastPausedLine   int           // Track last paused line
	executionActive  bool          // Indicates whether a debug run is in-flight
}

// NewDebugger creates a new debugger instance
func NewDebugger() *Debugger {
	return &Debugger{
		state:            DebugStateRunning,
		stepMode:         StepModeNone,
		breakpoints:      make(map[string]*Breakpoint),
		callStack:        make([]StackFrame, 0),
		eventSubscribers: make(map[int]chan DebugEvent),
		resumeChan:       make(chan struct{}),
	}
}

// SetBreakpoint adds a breakpoint at the specified location
func (d *Debugger) SetBreakpoint(file string, line int, condition string) *Breakpoint {
	d.mu.Lock()
	defer d.mu.Unlock()

	key := fmt.Sprintf("%s:%d", file, line)
	bp := &Breakpoint{
		ID:        key,
		File:      file,
		Line:      line,
		Enabled:   true,
		Condition: condition,
	}
	d.breakpoints[key] = bp
	return bp
}

// RemoveBreakpoint removes a breakpoint at the specified location
func (d *Debugger) RemoveBreakpoint(file string, line int) bool {
	d.mu.Lock()
	defer d.mu.Unlock()

	key := fmt.Sprintf("%s:%d", file, line)
	if _, exists := d.breakpoints[key]; exists {
		delete(d.breakpoints, key)
		return true
	}
	return false
}

// EnableBreakpoint enables/disables a breakpoint
func (d *Debugger) EnableBreakpoint(file string, line int, enabled bool) bool {
	d.mu.Lock()
	defer d.mu.Unlock()

	key := fmt.Sprintf("%s:%d", file, line)
	if bp, exists := d.breakpoints[key]; exists {
		bp.Enabled = enabled
		return true
	}
	return false
}

// GetBreakpoints returns all breakpoints
func (d *Debugger) GetBreakpoints() []*Breakpoint {
	d.mu.RLock()
	defer d.mu.RUnlock()

	breakpoints := make([]*Breakpoint, 0, len(d.breakpoints))
	for _, bp := range d.breakpoints {
		breakpoints = append(breakpoints, bp)
	}
	return breakpoints
}

// ClearBreakpoints removes all breakpoints or just those for a specific file
func (d *Debugger) ClearBreakpoints(file string) int {
	d.mu.Lock()
	defer d.mu.Unlock()

	// Clear everything when no file specified
	if file == "" {
		removed := len(d.breakpoints)
		d.breakpoints = make(map[string]*Breakpoint)
		return removed
	}

	removed := 0
	for key, bp := range d.breakpoints {
		if bp.File == file {
			delete(d.breakpoints, key)
			removed++
		}
	}
	return removed
}

// ShouldBreak checks if execution should pause at the current location
func (d *Debugger) ShouldBreak(file string, line int, rt *Runtime) bool {
	d.mu.Lock()
	defer d.mu.Unlock()

	// DEBUG: Log what we're checking
	key := fmt.Sprintf("%s:%d", file, line)
	fmt.Printf("DEBUG DEBUGGER: ShouldBreak checking %s (have %d breakpoints)\n", key, len(d.breakpoints))
	for bpKey, bp := range d.breakpoints {
		fmt.Printf("DEBUG DEBUGGER:   Breakpoint: %s (enabled=%v)\n", bpKey, bp.Enabled)
	}

	// Check breakpoints
	if bp, exists := d.breakpoints[key]; exists && bp.Enabled {
		fmt.Printf("DEBUG DEBUGGER: MATCH! Breaking at %s\n", key)
		// TODO: Evaluate condition if present
		if bp.Condition == "" {
			return true
		}
		// For now, break regardless of condition
		return true
	}

	// Check step mode
	switch d.stepMode {
	case StepModeOver:
		// Break if we're at the same depth or shallower
		if len(d.callStack) <= d.stepDepth {
			d.stepMode = StepModeNone
			return true
		}
	case StepModeInto:
		// Always break on next line
		d.stepMode = StepModeNone
		return true
	case StepModeOut:
		// Break when we return from current function
		if len(d.callStack) < d.stepDepth {
			d.stepMode = StepModeNone
			return true
		}
	}

	return false
}

// Pause pauses execution and stores the current position
func (d *Debugger) Pause() {
	d.mu.Lock()
	d.state = DebugStatePaused
	// Store current position for step detection
	d.lastPausedFile = d.currentFile
	d.lastPausedLine = d.currentLine
	d.mu.Unlock()

	fmt.Printf("DEBUG DEBUGGER: Paused at %s:%d, waiting for resume signal...\n", d.currentFile, d.currentLine)
	// Block until Continue() or step function is called
	<-d.resumeChan
	fmt.Printf("DEBUG DEBUGGER: Resumed execution\n")
}

// HasMovedToNewLine checks if execution has moved to a different line since last pause
func (d *Debugger) HasMovedToNewLine(file string, line int) bool {
	d.mu.RLock()
	defer d.mu.RUnlock()

	// First pause - no previous position stored
	if d.lastPausedFile == "" {
		return true
	}

	// Check if file or line has changed
	moved := (file != d.lastPausedFile || line != d.lastPausedLine)
	if moved {
		fmt.Printf("DEBUG DEBUGGER: Moved from %s:%d to %s:%d\n", d.lastPausedFile, d.lastPausedLine, file, line)
	} else {
		fmt.Printf("DEBUG DEBUGGER: Still at same line %s:%d\n", file, line)
	}
	return moved
}

// Continue resumes execution
func (d *Debugger) Continue() {
	d.mu.Lock()
	d.state = DebugStateRunning
	d.stepMode = StepModeNone
	d.mu.Unlock()

	fmt.Printf("DEBUG DEBUGGER: Continue called, sending resume signal\n")
	// Send resume signal (non-blocking in case nothing is waiting)
	select {
	case d.resumeChan <- struct{}{}:
	default:
	}
}

// StepOver steps over the current line
func (d *Debugger) StepOver() {
	d.mu.Lock()
	d.state = DebugStateStepping
	d.stepMode = StepModeOver
	d.stepDepth = len(d.callStack)
	d.mu.Unlock()

	fmt.Printf("DEBUG DEBUGGER: StepOver called, sending resume signal\n")
	// Send resume signal to continue to next line
	select {
	case d.resumeChan <- struct{}{}:
	default:
	}
}

// StepInto steps into function calls
func (d *Debugger) StepInto() {
	d.mu.Lock()
	d.state = DebugStateStepping
	d.stepMode = StepModeInto
	d.mu.Unlock()

	fmt.Printf("DEBUG DEBUGGER: StepInto called, sending resume signal\n")
	// Send resume signal to continue
	select {
	case d.resumeChan <- struct{}{}:
	default:
	}
}

// StepOut steps out of the current function
func (d *Debugger) StepOut() {
	d.mu.Lock()
	d.state = DebugStateStepping
	d.stepMode = StepModeOut
	d.stepDepth = len(d.callStack)
	d.mu.Unlock()

	fmt.Printf("DEBUG DEBUGGER: StepOut called, sending resume signal\n")
	// Send resume signal to continue
	select {
	case d.resumeChan <- struct{}{}:
	default:
	}
}

// GetState returns the current debug state
func (d *Debugger) GetState() DebugState {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.state
}

// IsExecutionActive reports whether a debug-enabled execution is currently running
func (d *Debugger) IsExecutionActive() bool {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.executionActive
}

// MarkRunning marks the debugger as actively executing code
func (d *Debugger) MarkRunning() {
	d.mu.Lock()
	d.state = DebugStateRunning
	d.stepMode = StepModeNone
	d.lastPausedFile = ""
	d.lastPausedLine = 0
	d.executionActive = true
	d.mu.Unlock()
}

// MarkStopped marks the debugger as idle after execution ends
func (d *Debugger) MarkStopped() {
	d.mu.Lock()
	d.state = DebugStateStopped
	d.executionActive = false
	d.stepMode = StepModeNone
	d.mu.Unlock()
}

// UpdatePosition updates the current execution position
func (d *Debugger) UpdatePosition(file string, line int) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.currentFile = file
	d.currentLine = line
}

// PushStackFrame adds a new frame to the call stack
func (d *Debugger) PushStackFrame(frame StackFrame) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.callStack = append(d.callStack, frame)
}

// PopStackFrame removes the top frame from the call stack
func (d *Debugger) PopStackFrame() {
	d.mu.Lock()
	defer d.mu.Unlock()
	if len(d.callStack) > 0 {
		d.callStack = d.callStack[:len(d.callStack)-1]
	}
}

// GetCallStack returns the current call stack
func (d *Debugger) GetCallStack() []StackFrame {
	d.mu.RLock()
	defer d.mu.RUnlock()

	// Return a copy to prevent external modification
	stack := make([]StackFrame, len(d.callStack))
	copy(stack, d.callStack)
	return stack
}

// GetCurrentPosition returns the current file and line
func (d *Debugger) GetCurrentPosition() (string, int) {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.currentFile, d.currentLine
}

// SendEvent sends a debug event to listeners
func (d *Debugger) SendEvent(event DebugEvent) {
	fmt.Printf("DEBUG DEBUGGER: SendEvent called with type=%s, file=%s, line=%d\n", event.Type, event.File, event.Line)
	d.mu.Lock()
	if len(d.eventSubscribers) == 0 {
		if len(d.pendingEvents) >= maxPendingEvents {
			d.pendingEvents = d.pendingEvents[1:]
		}
		d.pendingEvents = append(d.pendingEvents, event)
		d.mu.Unlock()
		fmt.Printf("DEBUG DEBUGGER: No subscribers, queued event (%d pending)\n", len(d.pendingEvents))
		return
	}
	subs := make([]chan DebugEvent, 0, len(d.eventSubscribers))
	for _, ch := range d.eventSubscribers {
		subs = append(subs, ch)
	}
	d.mu.Unlock()
	for _, ch := range subs {
		select {
		case ch <- event:
		default:
			fmt.Printf("DEBUG DEBUGGER: WARNING - Subscriber channel full, event dropped!\n")
		}
	}
}

// SubscribeEvents registers a listener for debug events and returns a channel plus an unsubscribe function
func (d *Debugger) SubscribeEvents() (<-chan DebugEvent, func()) {
	d.mu.Lock()
	id := d.nextSubscriberID
	d.nextSubscriberID++
	ch := make(chan DebugEvent, 100)
	if d.eventSubscribers == nil {
		d.eventSubscribers = make(map[int]chan DebugEvent)
	}
	d.eventSubscribers[id] = ch
	pending := make([]DebugEvent, len(d.pendingEvents))
	copy(pending, d.pendingEvents)
	d.pendingEvents = nil
	unsubscribe := func() {
		d.mu.Lock()
		defer d.mu.Unlock()
		if sub, ok := d.eventSubscribers[id]; ok {
			delete(d.eventSubscribers, id)
			close(sub)
		}
	}
	d.mu.Unlock()
	for _, evt := range pending {
		select {
		case ch <- evt:
		default:
			fmt.Printf("DEBUG DEBUGGER: WARNING - subscriber backlog full while replaying pending events\n")
		}
	}
	return ch, unsubscribe
}

// RequeueEvent stores a debug event so that the next subscriber receives it immediately
func (d *Debugger) RequeueEvent(event DebugEvent) {
	d.mu.Lock()
	defer d.mu.Unlock()
	if len(d.pendingEvents) >= maxPendingEvents {
		d.pendingEvents = d.pendingEvents[1:]
	}
	d.pendingEvents = append(d.pendingEvents, event)
}

// Stop stops the debugger and cleans up resources
func (d *Debugger) Stop() {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.state = DebugStateStopped
	d.executionActive = false
	for id, ch := range d.eventSubscribers {
		close(ch)
		delete(d.eventSubscribers, id)
	}
}

// ForceStop aborts any in-progress execution, clears state, and emits a stopped event
func (d *Debugger) ForceStop(reason string) {
	if d == nil {
		return
	}
	d.mu.Lock()
	previousState := d.state
	d.state = DebugStateStopped
	d.stepMode = StepModeNone
	d.callStack = d.callStack[:0]
	d.currentFile = ""
	d.currentLine = 0
	d.stepDepth = 0
	d.lastPausedFile = ""
	d.lastPausedLine = 0
	d.executionActive = false
	d.mu.Unlock()

	// Unblock any paused goroutines waiting on resume
	select {
	case d.resumeChan <- struct{}{}:
	default:
	}

	if reason != "" && previousState != DebugStateStopped {
		d.SendEvent(DebugEvent{
			Type:    "stopped",
			Message: reason,
		})
	}
}
