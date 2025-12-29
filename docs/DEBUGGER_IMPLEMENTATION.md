# Chariot Debugger Implementation

## Overview
A complete interactive debugger for the Chariot language with backend infrastructure and integrated web UI.

## Architecture

### Backend Components

#### 1. Core Debugger (`services/go-chariot/chariot/debugger.go`)
- **Debugger struct**: Thread-safe debugger with mutex-protected state
- **Features**:
  - Breakpoint management (add/remove/toggle by file:line)
  - Stepping modes: over, into, out
  - Call stack tracking with depth management
  - Real-time event streaming via channels
  - Pause/continue/stop operations coordinated through a `resumeChan`
  - Pending-event queue (max 200) so late WebSocket subscribers receive buffered events
  - `executionActive` flag plus `MarkRunning/MarkStopped/ForceStop` helpers to keep UI in sync with runtime lifecycle
- **State machine**: stopped → running → paused → stepping
- **Thread safety**: All operations protected by sync.RWMutex

#### 2. AST Position Tracking (`services/go-chariot/chariot/ast.go`)
- **SourcePos struct**: Tracks File, Line, Col for every AST node
- **Implementation**: Each node implements `GetPos()` so the debugger can report precise file/line information from runtime callbacks

#### 3. Runtime Integration (`services/go-chariot/chariot/runtime.go`)
- **Debugger field**: *Debugger attached to each runtime session on demand
- **Scope access**: GetCurrentScope(), GetGlobalScope() for variable inspection
- **Execution hooks**:
  - `Block.Exec()` consults `Debugger.ShouldBreak()` before each statement
  - Runtime surfaces `MarkRunning`, `MarkStopped`, and `ForceStop` so handlers can manage state transitions and unblock paused goroutines

#### 4. Debug API (`services/go-chariot/internal/handlers/handlers_debug.go`)
- **Access pattern**: Every request must include the authenticated `Authorization` header plus a `session={sessionId}` query parameter so the backend can route to the correct runtime.
- **REST Endpoints**:
  - `POST /api/debug/breakpoint`: Manage breakpoints (`action` supports add, remove, enable, disable, clear)
  - `POST /api/debug/step`: Step over/into/out
  - `POST /api/debug/continue`: Resume execution
  - `POST /api/debug/pause`: Pause execution
  - `GET /api/debug/state`: Get current debug state
  - `GET /api/debug/variables`: List variables in the current scope (or ancestor scope via `level` query param)
- **WebSocket**:
  - `WS /api/debug/events`: Real-time event streaming
  - Events: breakpoint, step, error, stopped

#### 5. Route Configuration (`services/go-chariot/internal/routes/routes.go`)
- All debug endpoints wired under `/api/debug` group
- Session-based authentication required

#### 6. Scope Helpers (`services/go-chariot/chariot/scope.go`)
- `AllVars()`: Get all variables in current scope
- `AllVarsWithParents()`: Get variables including parent scopes

### Frontend Components (Charioteer)

#### 1. CSS Styling (inline template in `services/charioteer/main.go`)
- **Debug panel styles**: `.debug-panel`, `.debug-section`, `.debug-controls`, `.debug-status`
- **Button styles**: `.debug-button` variants for play/pause/step with hover/disabled states
- **Data lists**: `.breakpoint-item`, `.callstack-item`, `.variable-item` share monospace formatting for clarity
- **Status indicators**: `.debug-status-icon` colors for running/paused/stepping/stopped plus `.breakpoint-remove` affordances
- **Monaco decorations**: `.breakpoint-line`, `.breakpoint-glyph`, `.debug-current-line`, `.debug-current-glyph` highlight breakpoints and the active line

#### 2. HTML Structure (rendered from `main.go` template)
- **Status bar**: Debug icon + state text that mirrors backend `DebugState`
- **Control buttons**: Continue (▶), Pause (⏸), Step Over (⤵), Step Into (↓), Step Out (↑)
- **Collapsible sections**: Breakpoints, Call Stack, and Variables panels toggle independently via `.debug-section-header`

#### 3. JavaScript Logic (embedded `<script>` in `main.go`)
- **State management**:
  - `debugSocket`, `debugState`, `currentDebugLine`, and `debugDecorations` mirror backend execution state
  - `breakpoints`: `Map` keyed by line number for the currently loaded file; the backend keeps the canonical `file:line` map
  - `debugSocketShouldReconnect`, `DEBUG_SOCKET_RETRY_MS`, `DEBUG_SOCKET_READY_TIMEOUT_MS`, and `debugSocketReadyResolvers` coordinate connection retries and waiters
  - Session-aware globals (`sessionId`, `authToken`, `sandboxProfile`, etc.) inform breakpoint sync and scope filtering
- **UI rendering functions**: `updateDebugStatus`, `renderBreakpoints`, `renderCallStack`, `renderVariables`, and `toggleDebugSection` keep the panel in sync with runtime events
- **Breakpoint management**:
  - `toggleBreakpoint`, `addBreakpoint`, `removeBreakpoint`, and `updateEditorBreakpoints` manage client-side state
  - `clearBreakpointsOnServer` and `syncBreakpointsWithServer` keep the server authoritative, removing stale entries when scopes/files change
- **Execution guard**: `runCode()` waits on `waitForDebugSocketReady()` whenever breakpoints exist so the first instruction cannot outrun the WebSocket subscription
- **Debug control actions**: `debugContinue`, `debugPause`, `debugStepOver`, `debugStepInto`, and `debugStepOut` each issue authenticated `fetch` calls to the matching REST endpoints and optimistically update the status bar
- **WebSocket integration**:
  - `shouldConnectDebugSocket`, `ensureDebugSocketConnected`, `scheduleDebugSocketReconnect`, `connectDebugSocket`, and `settleDebugSocketWaiters` manage lifecycle and reconnection logic
  - `handleDebugEvent` (plus `fetchDebugState` fallback) updates the call stack, variables, and highlighted line when events arrive
- **Event binding**:
  - `bindDebugHandlers` wires toolbar buttons + keyboard shortcuts (F5/F10/F11/Shift+F11)
  - Monaco’s glyph-margin click handler toggles breakpoints via these helpers

#### 4. Monaco Integration
- **Glyph margin enabled**: `glyphMargin: true` plus a dedicated click handler to open/clear breakpoints
- **Decorations**: Breakpoints render as red circles; the paused line renders as a yellow arrow via Monaco decorations

#### 5. Lifecycle Integration
- **Login/bootstrap**: After authentication the client fetches session profile info, refreshes file lists for the selected scope, and (if breakpoints exist) calls `ensureDebugSocketConnected`
- **Logout**: Shuts down the WebSocket, clears breakpoint/variable UI, and resets state to `stopped`
- **Initialization**: `bindAuthHandlers`, `initializeEditor`, and `initializeEventHandlers` run once the DOM is ready so the debugger toolbar works before the first login

## Usage

### Setting Breakpoints
1. Click on the line number in the Monaco editor glyph margin
2. Breakpoint appears as a red circle in the gutter
3. Breakpoint listed in the "Breakpoints" section with file and line number
4. Click the ✖ button to remove a breakpoint

### Running with Debugger
1. Set breakpoints in your code
2. Click the Run button or press F5
3. Execution pauses at the first breakpoint
4. Debug status shows "Paused at file:line"
5. Current line highlighted in yellow
6. Call stack and variables populated

### Stepping Through Code
- **Continue (F5)**: Resume execution until next breakpoint
- **Step Over (F10)**: Execute current line, don't enter functions
- **Step Into (F11)**: Step into function calls
- **Step Out (Shift+F11)**: Run until return from current function

### Inspecting State
- **Call Stack**: Shows function call hierarchy with file:line locations
- **Variables**: Displays all variables in current scope with values
- **Status Bar**: Shows current debug state (running/paused/stepping/stopped)

## API Reference

### REST Endpoints

#### Add/Remove Breakpoint
```
POST /api/debug/breakpoint?session={sessionId}
Content-Type: application/json
Authorization: {authToken}

{
  "file": "example.ch",
  "line": 10,
  "action": "add" | "remove" | "enable" | "disable" | "clear"
}

// When action == "clear", omit "line" to clear every file or set "file" to limit the removal scope.
```

#### Step
```
POST /api/debug/step?session={sessionId}
Content-Type: application/json
Authorization: {authToken}

{
  "mode": "over" | "into" | "out"
}
```

#### Continue
```
POST /api/debug/continue?session={sessionId}
Authorization: {authToken}
```

#### Pause
```
POST /api/debug/pause?session={sessionId}
Authorization: {authToken}
```

#### Get State
```
GET /api/debug/state?session={sessionId}
Authorization: {authToken}

Response:
{
  "state": "paused",
  "file": "example.ch",
  "line": 10,
  "callStack": [
    {"functionName": "main", "file": "example.ch", "line": 10}
  ],
  "variables": {
    "x": 42,
    "name": "test"
  }
}
```

#### List Variables
```
GET /api/debug/variables?session={sessionId}&level={scopeLevel}
Authorization: {authToken}

// level is optional; 0 = current scope, 1 = parent, etc.
```

### WebSocket Events

#### Connection
```
WS /api/debug/events?session={sessionId}
```

#### Event Types
```javascript
// Breakpoint hit
{
  "type": "breakpoint",
  "file": "example.ch",
  "line": 10,
  "timestamp": "2024-01-01T12:00:00Z"
}

// Step completed
{
  "type": "step",
  "file": "example.ch",
  "line": 11,
  "timestamp": "2024-01-01T12:00:00Z"
}

// Error occurred
{
  "type": "error",
  "message": "Runtime error",
  "timestamp": "2024-01-01T12:00:00Z"
}

// Execution stopped
{
  "type": "stopped",
  "timestamp": "2024-01-01T12:00:00Z"
}
```

## Testing

### Manual Testing Steps
1. Build Charioteer: `cd services/charioteer && go build -o charioteer main.go`
2. Run Charioteer: `./charioteer`
3. Log in to the UI
4. Open a .ch file
5. Click line numbers to set breakpoints
6. Verify breakpoints appear in list
7. Click Run button
8. Verify execution pauses at breakpoint
9. Check call stack populated
10. Check variables displayed
11. Test step over/into/out
12. Test continue
13. Remove breakpoint and verify removed

### Edge Cases
- Multiple breakpoints in same file
- Breakpoints in different files
- Stepping through nested function calls
- Pausing during execution
- WebSocket reconnection on connection loss
- Session timeout handling
- Switching file or scope selections should remove stale breakpoints on both the client and server

## Implementation Notes

### Go Template String Escaping
JavaScript template literals (backticks) must be escaped in Go's raw string literals:
```go
// Wrong: html := `const x = \`${value}\`;`
// Right: html := ` + "`" + `const x = ${value}` + "`" + `;
```

### Thread Safety
All debugger state modifications are protected by `sync.RWMutex`, and the dedicated `resumeChan` ensures only one goroutine controls pause/continue semantics at a time.

### Event Streaming
Each subscriber receives a buffered channel (size 100) to avoid blocking runtime execution. When no subscribers are connected the debugger stores up to 200 `pendingEvents`, replaying them to the next WebSocket client before resuming live streaming.

### Breakpoint Resolution
The backend indexes breakpoints by `file:line` to support multiple files and avoid collisions. The Charioteer UI keeps a per-file `Map` keyed only by line number and syncs it with the canonical backend list whenever the active file or scope changes.

## Future Enhancements
- [ ] Conditional breakpoints (break if expression is true)
- [ ] Watch expressions (monitor variable changes)
- [ ] Breakpoint hit counts
- [ ] Log points (log without breaking)
- [ ] Multi-session debugging
- [ ] Persistent breakpoints (save/load with file)
- [ ] Source mapping for generated code
- [ ] Remote debugging support
