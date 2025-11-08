# Charioteer SSE Proxy Handlers - Fix

## Issue

When trying to use the SSE log streaming feature from the Charioteer frontend, the async execution was failing with a JSON parse error:

```
Network Error: JSON.parse: unexpected non-whitespace character after JSON data at line 1 column 5 of the JSON data
```

## Root Cause

The Charioteer service acts as a proxy between the browser and the go-chariot backend. While we added the SSE streaming endpoints to go-chariot and the frontend JavaScript code to call them, we **forgot to add proxy handlers in Charioteer** to forward these new async endpoints to the backend.

Without the proxy handlers:
- `/api/execute-async` → ❌ 404 Not Found
- `/api/logs/:execId` → ❌ 404 Not Found  
- `/api/result/:execId` → ❌ 404 Not Found

The 404 response body (HTML error page) was being parsed as JSON, causing the parse error.

## Solution

Added three new proxy handlers to `services/charioteer/main.go`:

### 1. `executeAsyncHandler` - Proxy async execution requests

```go
func executeAsyncHandler(w http.ResponseWriter, r *http.Request) {
    // Read request body
    body, err := io.ReadAll(r.Body)
    // ...
    
    // Forward to backend
    req, err := http.NewRequest("POST", getBackendURL()+"/api/execute-async", bytes.NewBuffer(body))
    // ...
    
    // Copy Authorization header and forward request
    req.Header.Set("Authorization", authHeader)
    req.Header.Set("Content-Type", "application/json")
    
    // Return response
    io.Copy(w, resp.Body)
}
```

### 2. `streamLogsHandler` - Proxy SSE log streams

```go
func streamLogsHandler(w http.ResponseWriter, r *http.Request) {
    // Extract execution ID from path
    execID := extractExecIDFromPath(r.URL.Path)
    
    // Forward to backend SSE endpoint
    backendURL := getBackendURL() + "/api/logs/" + execID
    req, err := http.NewRequest("GET", backendURL, nil)
    // ...
    
    // Set SSE headers
    w.Header().Set("Content-Type", "text/event-stream")
    w.Header().Set("Cache-Control", "no-cache")
    w.Header().Set("Connection", "keep-alive")
    w.Header().Set("X-Accel-Buffering", "no")
    
    // Stream response from backend to client
    flusher, ok := w.(http.Flusher)
    buf := make([]byte, 4096)
    for {
        n, err := resp.Body.Read(buf)
        if n > 0 {
            w.Write(buf[:n])
            flusher.Flush()  // Flush each chunk immediately for SSE
        }
        if err != nil {
            return
        }
    }
}
```

### 3. `getResultHandler` - Proxy result polling requests

```go
func getResultHandler(w http.ResponseWriter, r *http.Request) {
    // Extract execution ID from path
    execID := extractExecIDFromPath(r.URL.Path)
    
    // Forward to backend
    backendURL := getBackendURL() + "/api/result/" + execID
    req, err := http.NewRequest("GET", backendURL, nil)
    // ...
    
    // Copy response
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(resp.StatusCode)
    io.Copy(w, resp.Body)
}
```

### Route Registration

Added routes for both root and prefixed paths:

```go
// Root paths (direct access)
http.HandleFunc("/api/execute-async", authMiddleware(executeAsyncHandler))
http.HandleFunc("/api/logs/", authMiddleware(streamLogsHandler))
http.HandleFunc("/api/result/", authMiddleware(getResultHandler))

// Prefixed paths (proxy hosting)
http.HandleFunc("/charioteer/api/execute-async", authMiddleware(executeAsyncHandler))
http.HandleFunc("/charioteer/api/logs/", authMiddleware(streamLogsHandler))
http.HandleFunc("/charioteer/api/result/", authMiddleware(getResultHandler))
```

## Architecture

```
┌─────────────┐         ┌──────────────┐         ┌──────────────┐
│   Browser   │         │  Charioteer  │         │  go-chariot  │
│  (Frontend) │         │    (Proxy)   │         │  (Backend)   │
└──────┬──────┘         └──────┬───────┘         └──────┬───────┘
       │                       │                        │
       │ POST /api/execute-async                        │
       ├──────────────────────>│                        │
       │                       │ POST /api/execute-async│
       │                       ├───────────────────────>│
       │                       │                        │
       │                       │   { execution_id }     │
       │   { execution_id }    │<───────────────────────┤
       │<──────────────────────┤                        │
       │                       │                        │
       │ GET /api/logs/:id (SSE)                        │
       ├──────────────────────>│                        │
       │                       │ GET /api/logs/:id      │
       │                       ├───────────────────────>│
       │                       │                        │
       │ data: {log entry}     │ data: {log entry}      │
       │<──────────────────────┤<───────────────────────┤
       │ (streaming...)        │ (streaming...)         │
       │                       │                        │
       │ GET /api/result/:id   │                        │
       ├──────────────────────>│                        │
       │                       │ GET /api/result/:id    │
       │                       ├───────────────────────>│
       │                       │                        │
       │   { result: ... }     │   { result: ... }      │
       │<──────────────────────┤<───────────────────────┤
```

## Key Implementation Details

### SSE Streaming

The `streamLogsHandler` must:
1. **Not set a timeout** on the HTTP client (`Timeout: 0`)
2. **Flush immediately** after writing each chunk
3. **Copy SSE headers** exactly (Content-Type, Cache-Control, Connection)
4. **Disable buffering** (X-Accel-Buffering: no) for nginx

### Path Extraction

Both `/api/logs/:execId` and `/charioteer/api/logs/:execId` must work:

```go
pathParts := strings.Split(strings.TrimPrefix(r.URL.Path, "/"), "/")
var execID string
for i, part := range pathParts {
    if part == "logs" && i+1 < len(pathParts) {
        execID = pathParts[i+1]
        break
    }
}
```

### Authorization

All proxy handlers copy the Authorization header from the incoming request:

```go
if authHeader := r.Header.Get("Authorization"); authHeader != "" {
    req.Header.Set("Authorization", authHeader)
}
```

## Files Modified

- `services/charioteer/main.go`:
  - Added `executeAsyncHandler()` (~50 lines)
  - Added `streamLogsHandler()` (~80 lines)
  - Added `getResultHandler()` (~50 lines)
  - Registered 6 new routes (3 root + 3 prefixed)

## Testing

After implementing these handlers, the SSE streaming works correctly:

```javascript
// 1. Start async execution
const response = await fetch('/api/execute-async', {
    method: 'POST',
    body: JSON.stringify({ program: 'logPrint("Hello SSE!")' })
});
const { execution_id } = await response.json();

// 2. Stream logs via SSE
const eventSource = new EventSource(`/api/logs/${execution_id}`);
eventSource.onmessage = (event) => {
    const log = JSON.parse(event.data);
    console.log(`[${log.level}] ${log.message}`);
};

// 3. Get final result
const result = await fetch(`/api/result/${execution_id}`).then(r => r.json());
console.log('Result:', result);
```

## Lesson Learned

When adding new backend endpoints, always remember to:
1. ✅ Add backend handlers (go-chariot)
2. ✅ Add frontend code (Charioteer JavaScript)
3. ⚠️ **Add proxy handlers** (Charioteer Go) if using a proxy architecture!

The proxy layer is easy to forget because it's transparent when it works, but critical when it doesn't.

## Build Status

- ✅ Charioteer builds successfully with new handlers
- ✅ No compilation errors
- ✅ Ready for deployment

## Next Steps

1. Rebuild charioteer Docker image:
   ```bash
   ./scripts/build-azure-cross-platform.sh v0.037 charioteer cpu
   ```

2. Push to registry:
   ```bash
   ./scripts/push-images.sh v0.037 charioteer cpu
   ```

3. Deploy to Azure and test SSE streaming end-to-end
