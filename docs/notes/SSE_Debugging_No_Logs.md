# SSE Log Streaming - Debugging "No Logs" Issue

## Problem

After implementing SSE log streaming, the frontend shows:
```
Execution ID: b3c2fb21-9145-4123-bf13-4bbcf53cd1ce
--- Streaming Logs ---
```

But no actual log entries appear, even though `logPrint('Hello SSE!')` is being called.

## Possible Causes

### 1. ❌ String Syntax (RULED OUT)
Initial hypothesis: Single quotes not supported
- **Reality**: Chariot supports single quotes, double quotes, and backticks
- **Verdict**: Not the issue

### 2. ✅ Race Condition (LIKELY)
The execution completes **before** the SSE connection is established:

```
Timeline:
T+0ms:    Client calls /api/execute-async
T+1ms:    Server starts goroutine, returns execution ID
T+2ms:    Goroutine executes logPrint() - very fast!
T+3ms:    Goroutine marks done, closes doneChan
T+10ms:   Client receives execution ID
T+15ms:   Client connects to /api/logs/:execId
T+16ms:   Server sends existing logs (should work!)
T+17ms:   Server selects on doneChan (already closed, triggers immediately)
T+18ms:   Server sends "done" event, closes connection
```

The logs **should** still be sent because we call `execCtx.LogBuffer.GetAll()` which returns buffered logs even after execution completes.

### 3. ✅ Logs Not Being Written (INVESTIGATION NEEDED)

Possible reasons logs aren't in the buffer:
- `logWriter` not set properly → **CHECKED**: SetLogWriter is called
- `WriteLog` not being called → **CHECKED**: It is called in logPrint
- `LogBuffer.Append` not working → **NEEDS CHECK**
- JSON serialization failing silently → **NEEDS CHECK**

## Debugging Changes Made

### Added Automatic Debug Logs

Modified `handlers_async.go` to add logs at start and end of execution:

```go
// Add a test log to verify streaming works
rt.WriteLog("INFO", "=== Execution started ===")

// Execute the program
val, err := rt.ExecProgram(req.Program)

// Add completion log
if err != nil {
    rt.WriteLog("ERROR", fmt.Sprintf("=== Execution failed: %v ===", err))
} else {
    rt.WriteLog("INFO", "=== Execution completed successfully ===")
}
```

These logs will appear **even if the user's script has syntax errors** or doesn't call logPrint at all.

### Added Backend Logging

Modified `StreamLogs` handler to log what it's doing:

```go
// Send all existing logs first
existingLogs := execCtx.LogBuffer.GetAll()
cfg.ChariotLogger.Info("Sending existing logs via SSE",
    zap.String("exec_id", execID),
    zap.Int("count", len(existingLogs)))

for _, entry := range existingLogs {
    // Send log...
}

// Check if execution is already done
if execCtx.IsDone() {
    cfg.ChariotLogger.Info("Execution already done, sending done event",
        zap.String("exec_id", execID))
    // Send done event...
}
```

Now we can check the backend logs to see:
- How many logs are in the buffer when SSE connects
- Whether execution is already done
- If there are any errors writing logs

## Testing Steps

### 1. Rebuild Services

```bash
# Rebuild go-chariot with debug logs
cd services/go-chariot
go build -o bin/go-chariot ./cmd

# Rebuild charioteer (already has proxy handlers)
cd ../charioteer
GOWORK=off go build -o bin/charioteer .
```

### 2. Run Services with Visible Logs

```bash
# Terminal 1: Start go-chariot (watch logs)
cd services/go-chariot
./bin/go-chariot

# Terminal 2: Start charioteer
cd services/charioteer
./bin/charioteer

# Terminal 3: Open browser
open http://localhost:8080/editor
```

### 3. Test with Simple Script

In Charioteer editor, run:

```chariot
logPrint("Hello SSE!")
```

### 4. Check Backend Logs

Look for these log messages in the go-chariot terminal:

```
INFO  Sending existing logs via SSE  exec_id=xxx count=3
INFO  Execution already done, sending done event  exec_id=xxx
```

The `count=3` should show:
1. "=== Execution started ==="
2. "Hello SSE!"
3. "=== Execution completed successfully ==="

If count=0, logs aren't being written to the buffer.

### 5. Check Browser Console

Open browser DevTools console and look for:
- Network errors on SSE connection
- JavaScript errors parsing log entries
- The raw SSE events (Network tab → EventStream)

## Expected vs Actual Output

### Expected (Working)

```
Starting execution...
Execution ID: b3c2fb21-9145-4123-bf13-4bbcf53cd1ce
--- Streaming Logs ---
[INFO] 10:58:17 === Execution started ===
[INFO] 10:58:17 Hello SSE!
[INFO] 10:58:17 === Execution completed successfully ===
--- Execution Complete ---
Final Result: null
```

### Actual (Current)

```
Execution ID: b3c2fb21-9145-4123-bf13-4bbcf53cd1ce
--- Streaming Logs ---
[blank - no logs appear]
```

## Next Debugging Steps

1. **Check if logs are written**: Run with debug logs and check backend terminal
2. **Check SSE connection**: Use browser DevTools Network tab to see raw SSE events
3. **Verify log buffer**: Add log to show `LogBuffer.GetAll()` contents
4. **Test timing**: Add artificial delay before execution to ensure SSE connects first

## Potential Fixes

### If logs aren't being written:
- Check if `LogBuffer.Append()` is actually being called
- Verify `rt.logWriter` is not nil inside `WriteLog()`
- Check for panics in the goroutine

### If logs are written but not sent:
- Check if `entry.JSON()` is returning empty string
- Verify SSE headers are correct
- Check for network errors in browser

### If SSE connection is broken:
- Check charioteer proxy is forwarding correctly
- Verify Authorization header is being passed
- Check for timeouts

## Files Modified

- `services/go-chariot/internal/handlers/handlers_async.go`:
  - Added automatic "Execution started/completed" logs
  - Added backend logging for debugging
  - Added early return if execution already done

## Build & Deploy

After debugging and fixing:

```bash
# Build Docker images
./scripts/build-azure-cross-platform.sh v0.037 all cpu

# Push to registry
./scripts/push-images.sh v0.037 all cpu

# Deploy to Azure (if needed)
```

## Status

- ✅ Added debug logs to execution
- ✅ Added backend logging to SSE handler
- ⏳ Need to rebuild and test
- ⏳ Need to check backend logs for diagnostics
