# SSE-Based Log Streaming

## Overview

The go-chariot service now supports real-time log streaming from Chariot script execution using Server-Sent Events (SSE). This enables the Charioteer UI to display script logs as they occur, providing immediate feedback during long-running operations.

## Architecture

### Components

1. **ExecutionManager** (`internal/handlers/execution_context.go`)
   - Thread-safe registry of active and recent executions
   - Automatic cleanup of completed executions (5-minute TTL)
   - UUID-based execution IDs for tracking

2. **ExecutionContext** (`internal/handlers/execution_context.go`)
   - Per-execution state tracking (ID, UserID, Program, Result, Error)
   - Embedded LogBuffer for capturing logs
   - Done channel for signaling completion

3. **LogBuffer** (`internal/handlers/execution_context.go`)
   - Circular buffer (max 1000 entries) with pub/sub
   - Thread-safe log appending and retrieval
   - Multiple subscriber support for SSE streaming

4. **Runtime LogWriter** (`chariot/runtime.go`)
   - LogWriter interface for dependency injection
   - LogEntry struct with Timestamp, Level, Message
   - Integration with logPrint() Chariot function

### Flow

```
Client                 go-chariot                Runtime              LogBuffer
  |                        |                        |                     |
  |-- POST /api/execute-async ---------------------->|                     |
  |<----- 200 OK (execution_id) --------------------|                     |
  |                        |                        |                     |
  |                        |-- Start goroutine ---->|                     |
  |                        |                        |-- SetLogWriter ---->|
  |                        |                        |                     |
  |-- GET /api/logs/:id (SSE) -------------------->|                     |
  |                        |                        |                     |
  |                        |-- Subscribe ---------->|-------------------->|
  |<----- data: {existing logs} --------------------|<--------------------|
  |                        |                        |                     |
  |                        |                        |-- logPrint() ------>|
  |<----- data: {new log} --------------------------|<--------------------|
  |                        |                        |                     |
  |                        |                        |-- Done ------------->|
  |<----- event: done ------------------------------|<--------------------|
  |                        |                        |                     |
  |-- GET /api/result/:id ----------------------->|                     |
  |<----- 200 OK (final result) -------------------|                     |
```

## API Endpoints

### 1. POST /api/execute-async

Start asynchronous script execution.

**Request:**
```json
{
  "program": "logPrint(\"Hello\"); let x = 1 + 2; logPrint(\"Result: \" + x);"
}
```

**Response (200 OK):**
```json
{
  "execution_id": "550e8400-e29b-41d4-a716-446655440000"
}
```

**Response (400 Bad Request):**
```json
{
  "error": "Missing program in request"
}
```

---

### 2. GET /api/logs/:execId

Stream logs in real-time via Server-Sent Events.

**Headers:**
- `Content-Type: text/event-stream`
- `Cache-Control: no-cache`
- `Connection: keep-alive`

**SSE Events:**

```
data: {"timestamp":"2024-01-20T10:30:00Z","level":"INFO","message":"Starting test"}

data: {"timestamp":"2024-01-20T10:30:01Z","level":"INFO","message":"Result is: 3"}

data: {"timestamp":"2024-01-20T10:30:02Z","level":"INFO","message":"Test complete"}

event: done
data: {}
```

**Response (404 Not Found):**
```json
{
  "error": "Execution not found"
}
```

---

### 3. GET /api/result/:execId

Poll for execution result (non-streaming).

**Response (200 OK - Completed):**
```json
{
  "execution_id": "550e8400-e29b-41d4-a716-446655440000",
  "status": "completed",
  "result": {
    "value": 3,
    "type": "number"
  },
  "logs": [
    {
      "timestamp": "2024-01-20T10:30:00Z",
      "level": "INFO",
      "message": "Starting test"
    },
    {
      "timestamp": "2024-01-20T10:30:01Z",
      "level": "INFO",
      "message": "Result is: 3"
    }
  ]
}
```

**Response (202 Accepted - Running):**
```json
{
  "execution_id": "550e8400-e29b-41d4-a716-446655440000",
  "status": "running",
  "logs": [
    {
      "timestamp": "2024-01-20T10:30:00Z",
      "level": "INFO",
      "message": "Starting test"
    }
  ]
}
```

**Response (500 Internal Server Error - Failed):**
```json
{
  "execution_id": "550e8400-e29b-41d4-a716-446655440000",
  "status": "error",
  "error": "undefined variable: foo",
  "logs": [...]
}
```

**Response (404 Not Found):**
```json
{
  "error": "Execution not found"
}
```

---

## Frontend Integration

### JavaScript Example

```javascript
async function executeScriptWithStreaming(program) {
  // 1. Start execution
  const response = await fetch('/api/execute-async', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ program })
  });
  
  const { execution_id } = await response.json();
  
  // 2. Connect to SSE stream
  const eventSource = new EventSource(`/api/logs/${execution_id}`);
  
  // 3. Handle log messages
  eventSource.onmessage = (event) => {
    const log = JSON.parse(event.data);
    console.log(`[${log.level}] ${log.message}`);
    appendToOutputTab(log);
  };
  
  // 4. Handle completion
  eventSource.addEventListener('done', async () => {
    eventSource.close();
    
    // 5. Fetch final result
    const result = await fetch(`/api/result/${execution_id}`).then(r => r.json());
    console.log('Final result:', result);
    displayResult(result);
  });
  
  // 6. Handle errors
  eventSource.onerror = (error) => {
    console.error('SSE error:', error);
    eventSource.close();
  };
}
```

### React Hook Example

```typescript
import { useEffect, useState } from 'react';

interface LogEntry {
  timestamp: string;
  level: string;
  message: string;
}

interface ExecutionResult {
  execution_id: string;
  status: 'running' | 'completed' | 'error';
  result?: any;
  error?: string;
  logs: LogEntry[];
}

export function useScriptExecution(program: string | null) {
  const [logs, setLogs] = useState<LogEntry[]>([]);
  const [result, setResult] = useState<ExecutionResult | null>(null);
  const [isRunning, setIsRunning] = useState(false);
  
  useEffect(() => {
    if (!program) return;
    
    let eventSource: EventSource | null = null;
    
    const execute = async () => {
      setIsRunning(true);
      setLogs([]);
      
      // Start execution
      const response = await fetch('/api/execute-async', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ program })
      });
      
      const { execution_id } = await response.json();
      
      // Stream logs
      eventSource = new EventSource(`/api/logs/${execution_id}`);
      
      eventSource.onmessage = (event) => {
        const log = JSON.parse(event.data);
        setLogs(prev => [...prev, log]);
      };
      
      eventSource.addEventListener('done', async () => {
        eventSource?.close();
        
        // Fetch final result
        const finalResult = await fetch(`/api/result/${execution_id}`)
          .then(r => r.json());
        
        setResult(finalResult);
        setIsRunning(false);
      });
      
      eventSource.onerror = () => {
        eventSource?.close();
        setIsRunning(false);
      };
    };
    
    execute();
    
    return () => {
      eventSource?.close();
    };
  }, [program]);
  
  return { logs, result, isRunning };
}
```

---

## Testing

### Manual Test (curl)

```bash
# 1. Login
curl -c /tmp/cookies.txt -X POST http://localhost:8087/login \
  -H "Content-Type: application/json" \
  -d '{"username":"admin","password":"admin"}'

# 2. Start execution
EXEC_ID=$(curl -s -b /tmp/cookies.txt -X POST http://localhost:8087/api/execute-async \
  -H "Content-Type: application/json" \
  -d '{"program":"logPrint(\"Test\"); 1+2"}' | jq -r .execution_id)

# 3. Stream logs
curl -N -b /tmp/cookies.txt "http://localhost:8087/api/logs/$EXEC_ID"

# 4. Get result (in another terminal)
curl -b /tmp/cookies.txt "http://localhost:8087/api/result/$EXEC_ID"
```

### Automated Test Script

Run `./test-sse.sh` in the `services/go-chariot` directory:

```bash
cd services/go-chariot
./test-sse.sh
```

---

## Configuration

### Execution Retention

Completed executions are retained for **5 minutes** after completion. This is configured in `NewExecutionManager()`:

```go
const cleanupInterval = 5 * time.Minute
const retentionDuration = 5 * time.Minute
```

To adjust retention, modify these constants in `internal/handlers/execution_context.go`.

### Log Buffer Size

The log buffer has a maximum of **1000 entries** per execution:

```go
const maxLogEntries = 1000
```

To adjust buffer size, modify this constant in `internal/handlers/execution_context.go`.

---

## Backward Compatibility

The synchronous `/api/execute` endpoint remains unchanged and available for clients that don't need real-time log streaming. Both endpoints can be used simultaneously.

**Synchronous Execution:**
```bash
curl -X POST /api/execute -d '{"program":"1+2"}'
# Returns: {"result":3}
```

**Asynchronous Execution:**
```bash
curl -X POST /api/execute-async -d '{"program":"1+2"}'
# Returns: {"execution_id":"..."}
```

---

## Limitations

1. **Session Management**: Execution contexts are in-memory only. If the server restarts, all active executions are lost.

2. **Scalability**: Each SSE connection holds an open HTTP connection. For high-concurrency scenarios, consider:
   - Connection pooling
   - Redis-backed execution storage
   - Load balancing with sticky sessions

3. **Log Retention**: Logs older than 5 minutes after completion are automatically deleted. For audit requirements, consider persisting logs to a database.

4. **Multi-Server Deployment**: ExecutionManager is in-memory. In a multi-server setup, clients must connect to the same server instance for streaming (use sticky sessions or Redis pub/sub).

---

## Future Enhancements

1. **Persistent Storage**: Store execution contexts in Redis or database for multi-server support
2. **WebSocket Support**: Add WebSocket alternative for bidirectional communication
3. **Execution History**: Add endpoint to list recent executions (`GET /api/executions`)
4. **Cancellation**: Add endpoint to cancel running executions (`POST /api/execution/:id/cancel`)
5. **Progress Reporting**: Add execution progress percentage to logs
6. **Log Levels**: Add endpoint to filter logs by level (`/api/logs/:id?level=error`)

---

## Troubleshooting

### Logs Not Streaming

**Symptom**: SSE endpoint returns 404 or no logs appear

**Causes**:
1. Execution ID not found (check if execution was started successfully)
2. Runtime.logWriter not set (check ExecuteAsync sets it before running)
3. logPrint() not called in script (add explicit logPrint calls for testing)

**Debug**:
```bash
# Check if execution exists
curl /api/result/$EXEC_ID

# Check server logs for errors
docker logs go-chariot 2>&1 | grep -i error
```

### SSE Connection Closes Immediately

**Symptom**: EventSource closes right after connecting

**Causes**:
1. Authentication failure (SessionAuth middleware rejects request)
2. Execution already completed before SSE connection established
3. Server not sending proper SSE headers

**Debug**:
```bash
# Test with verbose curl
curl -v -N /api/logs/$EXEC_ID

# Check headers
# Should see: Content-Type: text/event-stream
```

### Memory Leak

**Symptom**: Server memory grows over time

**Causes**:
1. ExecutionManager cleanup not running
2. LogBuffer subscribers not unsubscribed
3. Too many concurrent executions

**Debug**:
```bash
# Check Go runtime stats
curl /api/debug/pprof/heap > heap.prof
go tool pprof heap.prof
```

---

## Related Files

- `services/go-chariot/internal/handlers/execution_context.go` - ExecutionManager, ExecutionContext, LogBuffer
- `services/go-chariot/internal/handlers/handlers_async.go` - ExecuteAsync, StreamLogs, GetResult handlers
- `services/go-chariot/chariot/runtime.go` - LogWriter interface, SetLogWriter, WriteLog
- `services/go-chariot/chariot/system_funcs.go` - logPrint() integration with WriteLog
- `services/go-chariot/internal/routes/routes.go` - SSE endpoint registration
- `services/go-chariot/test-sse.sh` - Automated test script
