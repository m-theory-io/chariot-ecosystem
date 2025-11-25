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
  - Pause/continue/stop operations
- **State machine**: stopped → running → paused → stepping
- **Thread safety**: All operations protected by sync.RWMutex

#### 2. AST Position Tracking (`services/go-chariot/chariot/ast.go`)
- **SourcePos struct**: Tracks File, Line, Col for all AST nodes
- **Implementation**: Added GetPos() method to all 14 node types:
  - Block, IfNode, ForNode, WhileNode
  - AssignmentNode, ExprNode, CallNode, IndexNode
  - UnaryNode, BinaryNode, LiteralNode, IdentifierNode
  - ReturnNode, BreakNode

#### 3. Runtime Integration (`services/go-chariot/chariot/runtime.go`)
- **Debugger field**: *Debugger added to Runtime struct
- **Scope access**: GetCurrentScope(), GetGlobalScope() for variable inspection
- **Execution hooks**: Block.Exec() checks ShouldBreak() before each statement

#### 4. Debug API (`services/go-chariot/internal/handlers/handlers_debug.go`)
- **REST Endpoints**:
  - `POST /api/debug/breakpoint`: Add/remove breakpoints
  - `POST /api/debug/step`: Step over/into/out
  - `POST /api/debug/continue`: Resume execution
  - `POST /api/debug/pause`: Pause execution
  - `GET /api/debug/state`: Get current debug state
  - `GET /api/debug/variables`: List all variables in scope
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

#### 1. CSS Styling (`main.go` ~line 755-970)
- **Debug panel styles**: `.debug-panel`, `.debug-section`, `.debug-controls`
- **Button styles**: `.debug-button` with hover/active/disabled states
- **Item styles**: `.breakpoint-item`, `.callstack-item`, `.variable-item`
- **Status indicators**: Color-coded icons (running=green, paused=yellow, stepping=cyan, stopped=red)
- **Monaco decorations**: `.breakpoint-line`, `.breakpoint-glyph`, `.debug-current-line`, `.debug-current-glyph`

#### 2. HTML Structure (`main.go` ~line 1178-1235)
- **Status bar**: Debug icon + state text
- **Control buttons**: Continue (▶), Pause (⏸), Step Over (⤵), Step Into (↓), Step Out (↑)
- **Collapsible sections**:
  - Breakpoints list with remove buttons
  - Call stack with function names and locations
  - Variables inspector with name/value pairs

#### 3. JavaScript Logic (`main.go` ~line 1495-1915)
- **State management**:
  - `debugSocket`: WebSocket connection
  - `debugState`: Current state (stopped/running/paused/stepping)
  - `breakpoints`: Map of file:line → {file, line, enabled}
  - `currentDebugLine`: Currently highlighted line
  - `debugDecorations`: Monaco editor decorations

- **UI rendering functions**:
  - `updateDebugStatus()`: Update status bar and button states
  - `renderBreakpoints()`: Display breakpoint list
  - `renderCallStack()`: Display call stack frames
  - `renderVariables()`: Display variable values
  - `toggleDebugSection()`: Collapse/expand sections

- **Breakpoint management**:
  - `toggleBreakpoint(line)`: Add or remove breakpoint
  - `addBreakpoint(line)`: Add breakpoint and sync to backend
  - `removeBreakpoint(line)`: Remove breakpoint and sync to backend
  - `updateEditorBreakpoints()`: Update Monaco decorations

- **Debug control actions**:
  - `debugContinue()`: Resume execution
  - `debugPause()`: Pause execution
  - `debugStepOver()`: Step to next line (same level)
  - `debugStepInto()`: Step into function
  - `debugStepOut()`: Step out of function

- **WebSocket integration**:
  - `connectDebugSocket()`: Establish WebSocket connection
  - `handleDebugEvent(event)`: Process debug events from backend
  - `fetchDebugState()`: Get current call stack and variables

- **Event binding**:
  - `bindDebugHandlers()`: Wire up button clicks and keyboard shortcuts
  - **Keyboard shortcuts**:
    - F5: Continue
    - F10: Step Over
    - F11: Step Into
    - Shift+F11: Step Out

#### 4. Monaco Integration (`main.go` ~line 2052-2060)
- **Glyph margin enabled**: `glyphMargin: true`
- **Click handler**: Toggle breakpoint on glyph margin click
- **Decorations**: Breakpoints shown with red circle, current line with yellow arrow

#### 5. Lifecycle Integration
- **Login**: `connectDebugSocket()` called after successful authentication (line 2214)
- **Logout**: Close WebSocket and reset debug state (line 2617-2633)
- **Initialization**: `bindDebugHandlers()` called in `initializeEventHandlers()` (line 2819)

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
  "action": "add" | "remove"
}
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

## Implementation Notes

### Go Template String Escaping
JavaScript template literals (backticks) must be escaped in Go's raw string literals:
```go
// Wrong: html := `const x = \`${value}\`;`
// Right: html := ` + "`" + `const x = ${value}` + "`" + `;
```

### Thread Safety
All debugger state modifications are protected by `sync.RWMutex` to ensure safe concurrent access from WebSocket handlers and runtime execution.

### Event Streaming
Debug events are sent via buffered channel (size 100) to prevent blocking runtime execution. WebSocket handler drains the channel and broadcasts to connected clients.

### Breakpoint Resolution
Breakpoints are indexed by `file:line` key to support multiple files and avoid line number conflicts.

## Future Enhancements
- [ ] Conditional breakpoints (break if expression is true)
- [ ] Watch expressions (monitor variable changes)
- [ ] Breakpoint hit counts
- [ ] Log points (log without breaking)
- [ ] Multi-session debugging
- [ ] Persistent breakpoints (save/load with file)
- [ ] Source mapping for generated code
- [ ] Remote debugging support
