package main

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"flag"
	"fmt"
	"html/template"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/websocket"
)

// Configuration variables
var (
	backendURL         = flag.String("backend", "", "URL of the Chariot backend server")
	port               = flag.String("port", "8080", "Port to run the web server on")
	timeoutSeconds     = flag.Int("timeout", 300, "Timeout in seconds for backend requests")
	libraryName        = flag.String("library", "stdlib.json", "Name of the library to use for function execution")
	insecureSkipVerify = flag.Bool("insecure", true, "Skip TLS certificate verification for backend (dev only)")
	certPath           = flag.String("certpath", ".certs", "cert file folder")
	useSSL             = flag.Bool("ssl", false, "Use HTTPS with TLS certs (default false for dev)")
	mcpDataDir         = flag.String("mcp-data", "", "Directory for MCP settings and fixtures")
	mcpStore           *MCPStore
	agentFixtureStore  *AgentFixtureStore
)

// ResultJSON provides a standardized JSON response format
type ResultJSON struct {
	Result string      `json:"result"`
	Data   interface{} `json:"data"`
}

type ExecRequestData struct {
	Program  string `json:"program"`
	Filename string `json:"filename,omitempty"`
}

type contextKey string

// sendSuccess sends a successful ResultJSON response
func sendSuccess(w http.ResponseWriter, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(ResultJSON{
		Result: "OK",
		Data:   data,
	}); err != nil {
		log.Printf("encode success response error: %v", err)
	}
}

// sendError sends an error ResultJSON response
func sendError(w http.ResponseWriter, statusCode int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	if err := json.NewEncoder(w).Encode(ResultJSON{
		Result: "ERROR",
		Data:   message,
	}); err != nil {
		log.Printf("encode error response error: %v", err)
	}
}

// getBackendURL returns the backend URL from flag, environment variable, or default
func getBackendURL() string {
	if *backendURL != "" {
		return *backendURL
	}
	if env := os.Getenv("CHARIOT_BACKEND_URL"); env != "" {
		return env
	}
	return "https://localhost:8087"
}

func getMCPDataDir() string {
	if *mcpDataDir != "" {
		return *mcpDataDir
	}
	if env := os.Getenv("CHARIOTEER_MCP_DATA_DIR"); env != "" {
		return env
	}
	return "data"
}

// getPort returns the port from flag, environment variable, or default
func getPort() string {
	if *port != "8080" {
		return *port
	}
	if env := os.Getenv("CHARIOT_PORT"); env != "" {
		return env
	}
	return "8080"
}

// getTimeout returns the timeout duration from flag, environment variable, or default
func getTimeout() time.Duration {
	if *timeoutSeconds != 300 {
		return time.Duration(*timeoutSeconds) * time.Second
	}
	if env := os.Getenv("CHARIOT_TIMEOUT"); env != "" {
		if seconds, err := strconv.Atoi(env); err == nil && seconds > 0 {
			return time.Duration(seconds) * time.Second
		}
	}
	return 300 * time.Second
}

// Helper to create an HTTP client with optional TLS skip
func getHTTPClient() *http.Client {
	if strings.HasPrefix(getBackendURL(), "https://") && *insecureSkipVerify {
		return &http.Client{
			Timeout: getTimeout(),
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
			},
		}
	}
	return &http.Client{Timeout: getTimeout()}
}

// ---- Listener API proxy helpers ----
func proxyToBackendJSON(w http.ResponseWriter, r *http.Request, method, path string, body []byte) {
	client := getHTTPClient()
	req, err := http.NewRequest(method, getBackendURL()+path, bytes.NewBuffer(body))
	if err != nil {
		sendError(w, http.StatusInternalServerError, "Failed to create backend request: "+err.Error())
		return
	}
	req.Header.Set("Content-Type", "application/json")
	// Forward auth from cookie or header
	token := r.Header.Get("Authorization")
	if token == "" {
		if c, err := r.Cookie("chariot_token"); err == nil {
			token = c.Value
		}
	}
	if token != "" {
		req.Header.Set("Authorization", token)
	}
	resp, err := client.Do(req)
	if err != nil {
		sendError(w, http.StatusServiceUnavailable, "Failed to contact backend: "+err.Error())
		return
	}
	defer resp.Body.Close()
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(resp.StatusCode)
	if _, err := io.Copy(w, resp.Body); err != nil {
		log.Printf("proxy error copying body: %v", err)
	}
}

func appendQuery(path string, r *http.Request) string {
	if raw := r.URL.RawQuery; raw != "" {
		if strings.Contains(path, "?") {
			return path + "&" + raw
		}
		return path + "?" + raw
	}
	return path
}

func proxyDebugRequest(w http.ResponseWriter, r *http.Request, method, backendPath string, forwardBody bool) {
	var body []byte
	if forwardBody {
		if r.Body != nil {
			data, err := io.ReadAll(r.Body)
			if err != nil {
				sendError(w, http.StatusBadRequest, "failed to read request body")
				return
			}
			body = data
		}
	}

	proxyToBackendJSON(w, r, method, appendQuery(backendPath, r), body)
}

func listenersListHandler(w http.ResponseWriter, r *http.Request) {
	proxyToBackendJSON(w, r, http.MethodGet, "/api/listeners", nil)
}

// agentsListHandler proxies to backend /api/agents to list agents
func agentsListHandler(w http.ResponseWriter, r *http.Request) {
	proxyToBackendJSON(w, r, http.MethodGet, "/api/agents", nil)
}

func listenersCreateHandler(w http.ResponseWriter, r *http.Request) {
	body, _ := io.ReadAll(r.Body)
	proxyToBackendJSON(w, r, http.MethodPost, "/api/listeners", body)
}

func listenersDeleteHandler(w http.ResponseWriter, r *http.Request) {
	name := r.URL.Query().Get("name")
	if name == "" {
		sendError(w, http.StatusBadRequest, "missing name")
		return
	}
	proxyToBackendJSON(w, r, http.MethodDelete, "/api/listeners/"+url.PathEscape(name), nil)
}

func listenersStartHandler(w http.ResponseWriter, r *http.Request) {
	name := r.URL.Query().Get("name")
	if name == "" {
		sendError(w, http.StatusBadRequest, "missing name")
		return
	}
	proxyToBackendJSON(w, r, http.MethodPost, "/api/listeners/"+url.PathEscape(name)+"/start", nil)
}

func listenersStopHandler(w http.ResponseWriter, r *http.Request) {
	name := r.URL.Query().Get("name")
	if name == "" {
		sendError(w, http.StatusBadRequest, "missing name")
		return
	}
	proxyToBackendJSON(w, r, http.MethodPost, "/api/listeners/"+url.PathEscape(name)+"/stop", nil)
}

func listenersItemHandler(w http.ResponseWriter, r *http.Request) {
	name := strings.TrimPrefix(r.URL.Path, "/charioteer/api/listeners/")
	name = strings.Trim(name, "/")
	if name == "" {
		sendError(w, http.StatusBadRequest, "missing listener name")
		return
	}
	switch r.Method {
	case http.MethodPut:
		body, _ := io.ReadAll(r.Body)
		proxyToBackendJSON(w, r, http.MethodPut, "/api/listeners/"+url.PathEscape(name), body)
	case http.MethodDelete:
		proxyToBackendJSON(w, r, http.MethodDelete, "/api/listeners/"+url.PathEscape(name), nil)
	default:
		sendError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

// sessionProfileHandler proxies to backend /api/session/profile
func sessionProfileHandler(w http.ResponseWriter, r *http.Request) {
	proxyToBackendJSON(w, r, http.MethodGet, "/api/session/profile", nil)
}

// filesListProxyHandler proxies file list and save requests to backend /api/files
func filesListProxyHandler(w http.ResponseWriter, r *http.Request) {
	scope := r.URL.Query().Get("scope")
	path := "/api/files"
	if scope != "" {
		path += "?scope=" + url.QueryEscape(scope)
	}

	switch r.Method {
	case http.MethodGet:
		// List files
		proxyToBackendJSON(w, r, http.MethodGet, path, nil)
	case http.MethodPost:
		// Save file
		body, err := io.ReadAll(r.Body)
		if err != nil {
			sendError(w, http.StatusBadRequest, "failed to read request body")
			return
		}
		proxyToBackendJSON(w, r, http.MethodPost, path, body)
	default:
		sendError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

// fileGetProxyHandler proxies file get/delete requests to backend /api/files/:name
func fileGetProxyHandler(w http.ResponseWriter, r *http.Request) {
	// Extract filename from path: /api/files/:name or /charioteer/api/files/:name
	var name string
	if strings.HasPrefix(r.URL.Path, "/charioteer/api/files/") {
		name = strings.TrimPrefix(r.URL.Path, "/charioteer/api/files/")
	} else if strings.HasPrefix(r.URL.Path, "/api/files/") {
		name = strings.TrimPrefix(r.URL.Path, "/api/files/")
	}

	if name == "" {
		sendError(w, http.StatusBadRequest, "filename required in path")
		return
	}

	scope := r.URL.Query().Get("scope")
	path := "/api/files/" + url.PathEscape(name)
	if scope != "" {
		path += "?scope=" + url.QueryEscape(scope)
	}

	switch r.Method {
	case http.MethodGet:
		proxyToBackendJSON(w, r, http.MethodGet, path, nil)
	case http.MethodDelete:
		proxyToBackendJSON(w, r, http.MethodDelete, path, nil)
	default:
		sendError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func listenerScriptsCollectionHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		proxyToBackendJSON(w, r, http.MethodGet, "/api/listener-scripts", nil)
	case http.MethodPost:
		body, err := io.ReadAll(r.Body)
		if err != nil {
			sendError(w, http.StatusBadRequest, "failed to read request body")
			return
		}
		proxyToBackendJSON(w, r, http.MethodPost, "/api/listener-scripts", body)
	default:
		sendError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func listenerScriptItemHandler(w http.ResponseWriter, r *http.Request) {
	var name string
	if strings.HasPrefix(r.URL.Path, "/charioteer/api/listener-scripts/") {
		name = strings.TrimPrefix(r.URL.Path, "/charioteer/api/listener-scripts/")
	} else if strings.HasPrefix(r.URL.Path, "/api/listener-scripts/") {
		name = strings.TrimPrefix(r.URL.Path, "/api/listener-scripts/")
	}
	if name == "" {
		sendError(w, http.StatusBadRequest, "script name required")
		return
	}
	backendPath := "/api/listener-scripts/" + url.PathEscape(name)
	switch r.Method {
	case http.MethodGet:
		proxyToBackendJSON(w, r, http.MethodGet, backendPath, nil)
	case http.MethodDelete:
		proxyToBackendJSON(w, r, http.MethodDelete, backendPath, nil)
	default:
		sendError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

// ---- WebSocket proxy support ----
// We use gorilla/websocket for client/server WS in charioteer as well to proxy to backend
// without relying on the http reverse proxy. This keeps the Authorization header on upgrade.
// Minimal inline proxy without external deps besides stdlib.

// dashboardWSProxyHandler proxies WebSocket connections to the backend /api/dashboard/stream
func dashboardWSProxyHandler(w http.ResponseWriter, r *http.Request) {
	// Browsers cannot set custom headers on WebSocket upgrade. Accept token from query string.
	// Fallbacks: Authorization header (for non-browser clients) or cookie named "chariot_token".
	token := r.URL.Query().Get("token")
	if token == "" {
		token = r.Header.Get("Authorization")
	}
	if token == "" {
		if c, err := r.Cookie("chariot_token"); err == nil {
			token = c.Value
		}
	}
	if token == "" {
		sendError(w, http.StatusUnauthorized, "Authorization token required")
		return
	}

	// Build backend WS URL from backend HTTP URL
	backend, err := url.Parse(getBackendURL())
	if err != nil {
		sendError(w, http.StatusInternalServerError, "Invalid backend URL")
		return
	}
	scheme := "ws"
	if backend.Scheme == "https" {
		scheme = "wss"
	}
	target := &url.URL{Scheme: scheme, Host: backend.Host, Path: "/api/dashboard/stream"}

	// Perform a simple bidirectional proxy using gorilla/websocket client and Upgrader
	// Use separate connections: clientConn (server->browser) and backendConn (server->backend)

	// Upgrade incoming request
	upgrader := websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin:     func(r *http.Request) bool { return true },
	}
	clientConn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WS proxy upgrade failed: %v", err)
		return
	}
	defer clientConn.Close()

	// Dial backend
	header := http.Header{}
	header.Set("Authorization", token)
	// Configure WS dialer (allow skipping TLS verify for dev if backend is https)
	dialer := *websocket.DefaultDialer
	if backend.Scheme == "https" && *insecureSkipVerify {
		dialer.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	}
	backendConn, _, err := dialer.Dial(target.String(), header)
	if err != nil {
		log.Printf("WS proxy dial backend failed: %v", err)
		if err := clientConn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseTryAgainLater, "backend unavailable")); err != nil {
			log.Printf("WS proxy failed to write close message: %v", err)
		}
		return
	}
	defer backendConn.Close()

	// Pump data between connections
	errc := make(chan error, 2)
	go func() { // browser -> backend
		for {
			mt, msg, err := clientConn.ReadMessage()
			if err != nil {
				errc <- err
				return
			}
			if err := backendConn.WriteMessage(mt, msg); err != nil {
				errc <- err
				return
			}
		}
	}()
	go func() { // backend -> browser
		for {
			mt, msg, err := backendConn.ReadMessage()
			if err != nil {
				errc <- err
				return
			}
			if err := clientConn.WriteMessage(mt, msg); err != nil {
				errc <- err
				return
			}
		}
	}()

	// Wait for one side to close
	<-errc
}

// agentsWSProxyHandler proxies WebSocket connections to the backend /ws/agents
func agentsWSProxyHandler(w http.ResponseWriter, r *http.Request) {
	// Token via query/header/cookie, same approach as dashboard proxy
	token := r.URL.Query().Get("token")
	if token == "" {
		token = r.Header.Get("Authorization")
	}
	if token == "" {
		if c, err := r.Cookie("chariot_token"); err == nil {
			token = c.Value
		}
	}
	if token == "" {
		sendError(w, http.StatusUnauthorized, "Authorization token required")
		return
	}

	backend, err := url.Parse(getBackendURL())
	if err != nil {
		sendError(w, http.StatusInternalServerError, "Invalid backend URL")
		return
	}
	scheme := "ws"
	if backend.Scheme == "https" {
		scheme = "wss"
	}
	target := &url.URL{Scheme: scheme, Host: backend.Host, Path: "/ws/agents"}

	// Upgrade incoming client first
	upgrader := websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin:     func(r *http.Request) bool { return true },
	}
	clientConn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("Agents WS proxy upgrade failed: %v", err)
		return
	}
	defer clientConn.Close()

	// Dial backend with Authorization header
	header := http.Header{}
	header.Set("Authorization", token)
	d := *websocket.DefaultDialer
	if backend.Scheme == "https" && *insecureSkipVerify {
		d.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	}
	backendConn, _, err := d.Dial(target.String(), header)
	if err != nil {
		log.Printf("Agents WS proxy dial backend failed: %v", err)
		// Signal close to client
		_ = clientConn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseTryAgainLater, "backend unavailable"))
		return
	}
	defer backendConn.Close()

	// Optional: lightweight ping/pong support in proxy
	clientConn.SetReadLimit(512)
	clientConn.SetPongHandler(func(string) error { return nil })
	backendConn.SetReadLimit(512)
	backendConn.SetPongHandler(func(string) error { return nil })

	// Pipe data both ways
	errc := make(chan error, 2)
	go func() { // browser -> backend
		for {
			mt, msg, err := clientConn.ReadMessage()
			if err != nil {
				errc <- err
				return
			}
			if err := backendConn.WriteMessage(mt, msg); err != nil {
				errc <- err
				return
			}
		}
	}()
	go func() { // backend -> browser
		for {
			mt, msg, err := backendConn.ReadMessage()
			if err != nil {
				errc <- err
				return
			}
			if err := clientConn.WriteMessage(mt, msg); err != nil {
				errc <- err
				return
			}
		}
	}()

	// Wait until one side closes
	<-errc
}

func getTLSKey() (string, error) {
	if *certPath != "" {
		keyPath := fmt.Sprintf("%s/charioteer.key", *certPath)
		log.Printf("Checking for TLS key at %s", keyPath)
		if _, err := os.Stat(keyPath); err == nil {
			return keyPath, nil
		}
		return "", fmt.Errorf("TLS key file not found at %s", keyPath)
	}
	return "", fmt.Errorf("TLS key path is not set")
}

func getTLSCert() (string, error) {
	if *certPath != "" {
		certPath := fmt.Sprintf("%s/charioteer.crt", *certPath)
		log.Printf("Checking for TLS certificate at %s", certPath)
		if _, err := os.Stat(certPath); err == nil {
			return certPath, nil
		}
		return "", fmt.Errorf("TLS certificate file not found at %s", certPath)
	}
	return "", fmt.Errorf("TLS certificate path is not set")
}

var editorTemplate = template.Must(template.ParseFiles("templates/editor.html"))

type EditorData struct {
	InitialCode string
	LocalMCPURL string
}

type DashboardData struct {
	BackendURL string
}

func editorHandler(w http.ResponseWriter, r *http.Request) {
	// Set proper content type
	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	// Parse template
	tmpl, err := template.New("editor").Parse(editorTemplate.Tree.Root.String())
	if err != nil {
		http.Error(w, "Template parsing error: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Prepare template data
	data := EditorData{
		InitialCode: `// Chariot Script Example
    declare(x, 'N', 100)
    setq(result, add(x, 100))
    result`,
		LocalMCPURL: defaultLocalChariotMCPURL(),
	}

	// Execute template
	err = tmpl.Execute(w, data)
	if err != nil {
		http.Error(w, "Template execution error: "+err.Error(), http.StatusInternalServerError)
		return
	}
}

// Handler to list files in a directory
func listFilesHandler(w http.ResponseWriter, r *http.Request) {
	folder := r.URL.Query().Get("folder")
	if folder == "" {
		sendError(w, http.StatusBadRequest, "folder parameter required")
		return
	}

	// Read the directory
	files, err := os.ReadDir(folder)
	if err != nil {
		sendError(w, http.StatusInternalServerError, "Unable to read directory: "+err.Error())
		return
	}

	// Filter for .ch files
	var fileNames []string
	for _, file := range files {
		if !file.IsDir() {
			name := file.Name()
			// Skip macOS metadata files and other hidden files
			if strings.HasPrefix(name, "._") || strings.HasPrefix(name, ".") {
				continue
			}

			// Add files with .ch extension or all files if you prefer
			if strings.HasSuffix(name, ".ch") {
				fileNames = append(fileNames, name)
			}
		}
	}

	// Return as ResultJSON
	sendSuccess(w, fileNames)
}

// Handler to get file content
func getFileHandler(w http.ResponseWriter, r *http.Request) {
	filePath := r.URL.Query().Get("path")
	if filePath == "" {
		sendError(w, http.StatusBadRequest, "path parameter required")
		return
	}

	// Security check - ensure path is within allowed directory
	if strings.Contains(filePath, "..") {
		sendError(w, http.StatusBadRequest, "Invalid path")
		return
	}

	// Read the file
	content, err := os.ReadFile(filePath)
	if err != nil {
		sendError(w, http.StatusNotFound, "Unable to read file: "+err.Error())
		return
	}

	// Return file content as ResultJSON
	sendSuccess(w, string(content))
}

// Handler to execute code
func executeHandler(w http.ResponseWriter, r *http.Request) {

	// Log what the proxy receives from the browser
	log.Printf("PROXY DEBUG: Received Authorization header: '%s'", r.Header.Get("Authorization"))
	log.Printf("PROXY DEBUG: All headers from browser: %+v", r.Header)

	requestData := ExecRequestData{}

	// Parse JSON request body
	if err := json.NewDecoder(r.Body).Decode(&requestData); err != nil {
		sendError(w, http.StatusBadRequest, "Invalid request body: "+err.Error())
		return
	}

	ctx := context.Background()

	if authHeader := r.Header.Get("Authorization"); authHeader != "" {
		ctx = context.WithValue(ctx, contextKey("auth"), authHeader)
	} else {
		sendError(w, http.StatusUnauthorized, "Authorization header required")
		return
	}

	// Begin snip
	responseBody, statusCode, err := callExecute(ctx, &requestData)
	// End snip

	if err != nil {
		sendError(w, statusCode, "Failed to execute code: "+err.Error())
		return
	}

	// Forward the response back to the client directly
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	if _, err := w.Write(*responseBody); err != nil {
		log.Printf("error writing execute response: %v", err)
	}
}

// Handler to execute code asynchronously (proxy to go-chariot)
func executeAsyncHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		sendError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	// Read request body
	body, err := io.ReadAll(r.Body)
	if err != nil {
		sendError(w, http.StatusBadRequest, "Failed to read request body: "+err.Error())
		return
	}

	log.Printf("Proxying execute-async request: %s", string(body))

	// Forward to backend
	req, err := http.NewRequest("POST", getBackendURL()+"/api/execute-async", bytes.NewBuffer(body))
	if err != nil {
		sendError(w, http.StatusInternalServerError, "Failed to create backend request: "+err.Error())
		return
	}

	// Copy Authorization header
	if authHeader := r.Header.Get("Authorization"); authHeader != "" {
		req.Header.Set("Authorization", authHeader)
	}
	req.Header.Set("Content-Type", "application/json")

	// Make request to backend
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		sendError(w, http.StatusBadGateway, "Failed to reach backend: "+err.Error())
		return
	}
	defer resp.Body.Close()

	// Read response
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		sendError(w, http.StatusInternalServerError, "Failed to read backend response: "+err.Error())
		return
	}

	log.Printf("Backend execute-async response (status %d): %s", resp.StatusCode, string(respBody))

	// Copy response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(resp.StatusCode)
	w.Write(respBody)
}

// Handler to stream logs via SSE (proxy to go-chariot)
func streamLogsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		sendError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	// Extract execution ID from path
	// Path can be /api/logs/:execId or /charioteer/api/logs/:execId
	pathParts := strings.Split(strings.TrimPrefix(r.URL.Path, "/"), "/")
	var execID string
	for i, part := range pathParts {
		if part == "logs" && i+1 < len(pathParts) {
			execID = pathParts[i+1]
			break
		}
	}

	if execID == "" {
		sendError(w, http.StatusBadRequest, "Missing execution ID")
		return
	}

	// Get token from query parameter (EventSource doesn't support custom headers)
	token := r.URL.Query().Get("token")
	if token == "" {
		// Fallback to Authorization header if present
		token = r.Header.Get("Authorization")
	}

	// Forward to backend SSE endpoint
	backendURL := getBackendURL() + "/api/logs/" + execID
	req, err := http.NewRequest("GET", backendURL, nil)
	if err != nil {
		sendError(w, http.StatusInternalServerError, "Failed to create backend request: "+err.Error())
		return
	}

	// Set Authorization header for backend
	if token != "" {
		req.Header.Set("Authorization", token)
	}

	log.Printf("SSE proxy: Forwarding request to backend for exec %s", execID)

	// Make request to backend
	client := &http.Client{Timeout: 0} // No timeout for SSE streaming
	resp, err := client.Do(req)
	if err != nil {
		sendError(w, http.StatusBadGateway, "Failed to reach backend: "+err.Error())
		return
	}
	defer resp.Body.Close()

	// If backend returned error (not 200), forward it
	if resp.StatusCode != http.StatusOK {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(resp.StatusCode)
		io.Copy(w, resp.Body)
		return
	}

	// Set SSE headers
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")
	w.WriteHeader(http.StatusOK)

	// Stream response from backend to client
	flusher, ok := w.(http.Flusher)
	if !ok {
		log.Printf("streaming not supported")
		return
	}

	// Copy SSE events from backend to client
	buf := make([]byte, 4096)
	totalBytes := 0
	for {
		n, err := resp.Body.Read(buf)
		if n > 0 {
			totalBytes += n
			log.Printf("SSE proxy: received %d bytes from backend (total: %d) for exec %s", n, totalBytes, execID)
			if _, writeErr := w.Write(buf[:n]); writeErr != nil {
				log.Printf("error writing SSE data: %v", writeErr)
				return
			}
			flusher.Flush()
		}
		if err != nil {
			if err != io.EOF {
				log.Printf("error reading SSE stream: %v", err)
			} else {
				log.Printf("SSE stream completed for exec %s (total bytes: %d)", execID, totalBytes)
			}
			return
		}
	}
}

// Handler to get execution result (proxy to go-chariot)
func getResultHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		sendError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	// Extract execution ID from path
	// Path can be /api/result/:execId or /charioteer/api/result/:execId
	pathParts := strings.Split(strings.TrimPrefix(r.URL.Path, "/"), "/")
	var execID string
	for i, part := range pathParts {
		if part == "result" && i+1 < len(pathParts) {
			execID = pathParts[i+1]
			break
		}
	}

	if execID == "" {
		sendError(w, http.StatusBadRequest, "Missing execution ID")
		return
	}

	// Forward to backend
	backendURL := getBackendURL() + "/api/result/" + execID
	req, err := http.NewRequest("GET", backendURL, nil)
	if err != nil {
		sendError(w, http.StatusInternalServerError, "Failed to create backend request: "+err.Error())
		return
	}

	// Copy Authorization header
	if authHeader := r.Header.Get("Authorization"); authHeader != "" {
		req.Header.Set("Authorization", authHeader)
	}

	// Make request to backend
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		sendError(w, http.StatusBadGateway, "Failed to reach backend: "+err.Error())
		return
	}
	defer resp.Body.Close()

	// Copy response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(resp.StatusCode)
	if _, err := io.Copy(w, resp.Body); err != nil {
		log.Printf("error copying result response: %v", err)
	}
}

// Add authentication middleware
func authMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		token := r.Header.Get("Authorization")

		// For dashboard route and SSE endpoints, also check URL parameter
		// (EventSource doesn't support custom headers)
		if token == "" && (r.URL.Path == "/charioteer/dashboard" ||
			strings.Contains(r.URL.Path, "/api/logs/") ||
			strings.Contains(r.URL.Path, "/charioteer/api/logs/")) {
			urlToken := r.URL.Query().Get("token")
			if urlToken != "" {
				token = urlToken
			}
		}

		// if token == "" || !strings.HasPrefix(token, "Bearer ") {
		if token == "" {
			sendError(w, http.StatusUnauthorized, "Unauthorized")
			return
		}

		// Validate token here
		if !validateToken(strings.TrimPrefix(token, "Bearer ")) {
			sendError(w, http.StatusUnauthorized, "Invalid token")
			return
		}

		next(w, r)
	}
}

// callExecute
func callExecute(ctx context.Context, requestData *ExecRequestData) (*[]byte, int, error) {
	content, err := json.Marshal(requestData)
	if err != nil {
		return nil, http.StatusInternalServerError, fmt.Errorf("failed to marshal request data: %w", err)
	}

	// Create request with proper headers
	req, err := http.NewRequest("POST", getBackendURL()+"/api/execute", bytes.NewBuffer(content))
	if err != nil {
		return nil, http.StatusInternalServerError, fmt.Errorf("failed to create request: %w", err)
	}

	// Set Content-Type
	req.Header.Set("Content-Type", "application/json")

	// Extract auth token from context
	authToken, ok := ctx.Value(contextKey("auth")).(string)
	if ok {
		req.Header.Set("Authorization", authToken)
	}

	// Make the request
	client := getHTTPClient()
	resp, err := client.Do(req)
	if err != nil {
		return nil, http.StatusInternalServerError, fmt.Errorf("failed to execute code: %w", err)
	}
	defer resp.Body.Close()

	// Read response from Chariot server
	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, http.StatusInternalServerError, fmt.Errorf("failed to read response: %w", err)
	}
	return &responseBody, resp.StatusCode, nil
}

// Dummy token validation function (replace with real validation as needed)
func validateToken(token string) bool {
	// For demonstration, accept any non-empty token
	return token != ""
}

func loginHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		sendError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	// Set CORS headers if needed
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	// Read the request body from the client
	body, err := io.ReadAll(r.Body)
	if err != nil {
		sendError(w, http.StatusBadRequest, "Failed to read request body")
		return
	}
	defer r.Body.Close()

	// Forward the request to the Chariot server
	req, err := http.NewRequest("POST", getBackendURL()+"/login", bytes.NewBuffer(body))
	if err != nil {
		sendError(w, http.StatusInternalServerError, "Failed to create request")
		return
	}

	// Copy headers from original request
	req.Header.Set("Content-Type", "application/json")

	// Make the request to the Chariot server
	client := getHTTPClient()

	resp, err := client.Do(req)
	if err != nil {
		log.Printf("Failed to connect to Chariot server: %v", err)
		sendError(w, http.StatusServiceUnavailable, "Chariot server unavailable")
		return
	}
	defer resp.Body.Close()

	// Read response from Chariot server
	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		sendError(w, http.StatusInternalServerError, "Failed to read response")
		return
	}

	// If login succeeded and a token is present, set an HttpOnly cookie for WS auth
	if resp.StatusCode == http.StatusOK {
		var parsed struct {
			Result string `json:"result"`
			Data   struct {
				Token string `json:"token"`
			} `json:"data"`
		}
		if err := json.Unmarshal(responseBody, &parsed); err == nil && strings.EqualFold(parsed.Result, "OK") && parsed.Data.Token != "" {
			cookie := &http.Cookie{
				Name:     "chariot_token",
				Value:    parsed.Data.Token,
				Path:     "/",
				HttpOnly: true,
				// Secure when behind TLS or reverse proxy indicating HTTPS
				Secure:   r.TLS != nil || strings.EqualFold(r.Header.Get("X-Forwarded-Proto"), "https"),
				SameSite: http.SameSiteLaxMode,
				// Session cookie; optionally set MaxAge if desired
			}
			http.SetCookie(w, cookie)
		}
	}

	// Forward the response back to the client directly
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(resp.StatusCode)
	if _, err := w.Write(responseBody); err != nil {
		log.Printf("error writing login response: %v", err)
	}
}

func logoutHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		sendError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	// Set CORS headers if needed
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	// Read the request body from the client (if any)
	body, err := io.ReadAll(r.Body)
	if err != nil {
		sendError(w, http.StatusBadRequest, "Failed to read request body")
		return
	}
	defer r.Body.Close()

	// Forward the request to the Chariot server
	req, err := http.NewRequest("POST", getBackendURL()+"/logout", bytes.NewBuffer(body))
	if err != nil {
		sendError(w, http.StatusInternalServerError, "Failed to create request")
		return
	}

	// Copy important headers from original request
	req.Header.Set("Content-Type", "application/json")

	// Forward Authorization header if present
	if authHeader := r.Header.Get("Authorization"); authHeader != "" {
		req.Header.Set("Authorization", authHeader)
	}

	// Make the request to the Chariot server
	client := getHTTPClient()

	resp, err := client.Do(req)
	if err != nil {
		log.Printf("Failed to connect to Chariot server for logout: %v", err)
		sendError(w, http.StatusServiceUnavailable, "Chariot server unavailable")
		return
	}
	defer resp.Body.Close()

	// Read response from Chariot server
	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		sendError(w, http.StatusInternalServerError, "Failed to read response")
		return
	}

	// Clear the auth cookie regardless of backend response
	expired := &http.Cookie{
		Name:     "chariot_token",
		Value:    "",
		Path:     "/",
		Expires:  time.Unix(0, 0),
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   r.TLS != nil || strings.EqualFold(r.Header.Get("X-Forwarded-Proto"), "https"),
		SameSite: http.SameSiteLaxMode,
	}
	http.SetCookie(w, expired)

	// Forward the response back to the client directly
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(resp.StatusCode)
	if _, err := w.Write(responseBody); err != nil {
		log.Printf("error writing logout response: %v", err)
	}
}

// Handler to serve the dashboard page
func dashboardHandler(w http.ResponseWriter, r *http.Request) {
	// Set proper content type
	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	// Parse template
	tmpl, err := template.ParseFiles("templates/dashboard.html")
	if err != nil {
		http.Error(w, "Template parsing error: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Prepare template data
	data := DashboardData{
		BackendURL: getBackendURL(),
	}

	// Execute template
	err = tmpl.Execute(w, data)
	if err != nil {
		http.Error(w, "Template execution error: "+err.Error(), http.StatusInternalServerError)
		return
	}
}

// Handler to proxy dashboard API requests to go-chariot
func dashboardAPIHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		sendError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	// Set CORS headers
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	// Forward request to go-chariot backend
	backendURL := getBackendURL() + "/api/dashboard/status"

	// Create request with proper headers
	req, err := http.NewRequest("GET", backendURL, nil)
	if err != nil {
		sendError(w, http.StatusInternalServerError, "Failed to create request: "+err.Error())
		return
	}

	// Get auth token from request header and forward it
	authToken := r.Header.Get("Authorization")
	if authToken != "" {
		req.Header.Set("Authorization", authToken)
	}

	client := &http.Client{
		Timeout: time.Duration(*timeoutSeconds) * time.Second,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: *insecureSkipVerify},
		},
	}

	resp, err := client.Do(req)
	if err != nil {
		sendError(w, http.StatusInternalServerError, "Failed to connect to backend: "+err.Error())
		return
	}
	defer resp.Body.Close()

	// Read response from go-chariot server
	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		sendError(w, http.StatusInternalServerError, "Failed to read response")
		return
	}

	// Check if go-chariot returned success
	if resp.StatusCode != http.StatusOK {
		sendError(w, resp.StatusCode, "Backend error: "+string(responseBody))
		return
	}

	// Parse the go-chariot response to validate it's valid JSON
	var dashboardData interface{}
	if err := json.Unmarshal(responseBody, &dashboardData); err != nil {
		sendError(w, http.StatusInternalServerError, "Invalid response from backend")
		return
	}

	// Wrap the response in the expected format for the frontend
	wrappedResponse := map[string]interface{}{
		"result": "OK",
		"data":   dashboardData,
	}

	// Send the wrapped response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(wrappedResponse); err != nil {
		log.Printf("error encoding dashboard wrapped response: %v", err)
	}
}

// Handler to save file content
func saveFileHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		sendError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	var requestData struct {
		Path    string `json:"path"`
		Content string `json:"content"`
	}

	// Parse JSON request body
	if err := json.NewDecoder(r.Body).Decode(&requestData); err != nil {
		sendError(w, http.StatusBadRequest, "Invalid request body: "+err.Error())
		return
	}

	// Security check - ensure path is within allowed directory and doesn't contain dangerous sequences
	if strings.Contains(requestData.Path, "..") {
		sendError(w, http.StatusBadRequest, "Invalid path")
		return
	}

	// Ensure the directory exists
	dir := strings.TrimSuffix(requestData.Path, "/"+strings.Split(requestData.Path, "/")[len(strings.Split(requestData.Path, "/"))-1])
	if err := os.MkdirAll(dir, 0755); err != nil {
		sendError(w, http.StatusInternalServerError, "Unable to create directory: "+err.Error())
		return
	}

	// Write the file
	if err := os.WriteFile(requestData.Path, []byte(requestData.Content), 0644); err != nil {
		sendError(w, http.StatusInternalServerError, "Unable to save file: "+err.Error())
		return
	}

	// Return success response
	sendSuccess(w, "File saved successfully")
}

// Handler to rename a file
func renameFileHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		sendError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	var requestData struct {
		OldPath string `json:"oldPath"`
		NewPath string `json:"newPath"`
	}

	// Parse JSON request body
	if err := json.NewDecoder(r.Body).Decode(&requestData); err != nil {
		sendError(w, http.StatusBadRequest, "Invalid request body: "+err.Error())
		return
	}

	// Security check - ensure paths are within allowed directory and don't contain dangerous sequences
	if strings.Contains(requestData.OldPath, "..") || strings.Contains(requestData.NewPath, "..") {
		sendError(w, http.StatusBadRequest, "Invalid path")
		return
	}

	// Check if old file exists
	if _, err := os.Stat(requestData.OldPath); os.IsNotExist(err) {
		sendError(w, http.StatusNotFound, "Source file does not exist")
		return
	}

	// Check if new file already exists
	if _, err := os.Stat(requestData.NewPath); err == nil {
		sendError(w, http.StatusConflict, "Destination file already exists")
		return
	}

	// Ensure the destination directory exists
	dir := strings.TrimSuffix(requestData.NewPath, "/"+strings.Split(requestData.NewPath, "/")[len(strings.Split(requestData.NewPath, "/"))-1])
	if err := os.MkdirAll(dir, 0755); err != nil {
		sendError(w, http.StatusInternalServerError, "Unable to create destination directory: "+err.Error())
		return
	}

	// Rename the file
	if err := os.Rename(requestData.OldPath, requestData.NewPath); err != nil {
		sendError(w, http.StatusInternalServerError, "Unable to rename file: "+err.Error())
		return
	}

	// Return success response
	sendSuccess(w, "File renamed successfully")
}

// Handler to delete a file
func deleteFileHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		sendError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	var requestData struct {
		Path string `json:"path"`
	}

	// Parse JSON request body
	if err := json.NewDecoder(r.Body).Decode(&requestData); err != nil {
		sendError(w, http.StatusBadRequest, "Invalid request body: "+err.Error())
		return
	}

	// Security check - ensure path is within allowed directory and doesn't contain dangerous sequences
	if strings.Contains(requestData.Path, "..") {
		sendError(w, http.StatusBadRequest, "Invalid path")
		return
	}

	// Check if file exists
	if _, err := os.Stat(requestData.Path); os.IsNotExist(err) {
		sendError(w, http.StatusNotFound, "File does not exist")
		return
	}

	// Delete the file
	if err := os.Remove(requestData.Path); err != nil {
		sendError(w, http.StatusInternalServerError, "Unable to delete file: "+err.Error())
		return
	}

	// Return success response
	sendSuccess(w, "File deleted successfully")
}

// Function Library Handlers

// List all function names in the runtime
func listFunctionsHandler(w http.ResponseWriter, r *http.Request) {
	// Implementation here -- format call to callExecute
	requestData := ExecRequestData{
		Program: "listFunctions()",
	}

	// Get auth header from request
	ctx := context.Background()
	if authHeader := r.Header.Get("Authorization"); authHeader != "" {
		ctx = context.WithValue(ctx, contextKey("auth"), authHeader)
	} else {
		sendError(w, http.StatusUnauthorized, "Authorization header required")
		return
	}

	response, statusCode, err := callExecute(ctx, &requestData)
	if err != nil {
		sendError(w, statusCode, "Failed to list functions: "+err.Error())
		return
	}
	log.Printf("DEBUG: Backend response for listFunctions: %s", string(*response))
	// Parse the backend response
	var backendResp ResultJSON
	if err := json.Unmarshal(*response, &backendResp); err != nil {
		sendError(w, http.StatusInternalServerError, "Failed to parse backend response: "+err.Error())
		return
	}

	sendSuccess(w, backendResp.Data)
}

// Get source code for a function
func getFunctionHandler(w http.ResponseWriter, r *http.Request) {
	// Implementation here
	functionName := r.URL.Query().Get("name")
	if functionName == "" {
		sendError(w, http.StatusBadRequest, "Function name parameter required")
		return
	}
	requestData := ExecRequestData{
		Program: fmt.Sprintf("getFunction('%s')", functionName),
	}
	// Get auth header from request
	ctx := context.Background()
	if authHeader := r.Header.Get("Authorization"); authHeader != "" {
		ctx = context.WithValue(ctx, contextKey("auth"), authHeader)
	} else {
		sendError(w, http.StatusUnauthorized, "Authorization header required")
		return
	}
	response, statusCode, err := callExecute(ctx, &requestData)
	if err != nil {
		sendError(w, statusCode, "Failed to get function source: "+err.Error())
		return
	}
	log.Printf("DEBUG: Backend response for getFunctionSource: %s", string(*response))
	// Parse the backend response
	var backendResp ResultJSON
	if err := json.Unmarshal(*response, &backendResp); err != nil {
		sendError(w, http.StatusInternalServerError, "Failed to parse backend response: "+err.Error())
		return
	}

	sendSuccess(w, backendResp.Data)
}

// Save/update a function -- forwards to dev server /api/function/save
func saveFunctionHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		sendError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	var req struct {
		Name string   `json:"name"`
		Code string   `json:"code"`
		Args []string `json:"args,omitempty"`
		Body string   `json:"body,omitempty"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendError(w, http.StatusBadRequest, "Invalid request body: "+err.Error())
		return
	}

	// Prefer "code" if present, otherwise reconstruct from name/args/body
	code := req.Code
	if code == "" && req.Name != "" {
		code = "function " + req.Name + "(" + strings.Join(req.Args, ", ") + ") {\n" + req.Body + "\n}"
	}
	if req.Name == "" || code == "" {
		sendError(w, http.StatusBadRequest, "Missing function name or code")
		return
	}
	// Prepare JSON for backend
	payload := map[string]string{
		"name":             req.Name,
		"code":             code,
		"formatted_source": code, // Include formatted code if needed
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		sendError(w, http.StatusInternalServerError, "Failed to marshal payload: "+err.Error())
		return
	}

	// Prepare request to dev server
	backendReq, err := http.NewRequest("POST", getBackendURL()+"/api/function/save", bytes.NewBuffer(payloadBytes))
	if err != nil {
		sendError(w, http.StatusInternalServerError, "Failed to create backend request: "+err.Error())
		return
	}
	backendReq.Header.Set("Content-Type", "application/json")

	// Forward Authorization header if present
	if authHeader := r.Header.Get("Authorization"); authHeader != "" {
		backendReq.Header.Set("Authorization", authHeader)
	}

	client := getHTTPClient()
	resp, err := client.Do(backendReq)
	if err != nil {
		sendError(w, http.StatusServiceUnavailable, "Failed to contact backend: "+err.Error())
		return
	}
	defer resp.Body.Close()

	// Forward backend response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(resp.StatusCode)
	if _, err := io.Copy(w, resp.Body); err != nil {
		log.Printf("error copying backend response: %v", err)
	}
}

// Delete a function - delegates deleteFunction(<name>) to callExecute
func deleteFunctionHandler(w http.ResponseWriter, r *http.Request) {
	// Implementation here
	functionName := r.URL.Query().Get("name")
	if functionName == "" {
		sendError(w, http.StatusBadRequest, "Function name parameter required")
		return
	}
	requestData := ExecRequestData{
		Program: fmt.Sprintf("deleteFunction('%s')", functionName),
	}
	// Get auth header from request
	ctx := context.Background()
	if authHeader := r.Header.Get("Authorization"); authHeader != "" {
		ctx = context.WithValue(ctx, contextKey("auth"), authHeader)
	} else {
		sendError(w, http.StatusUnauthorized, "Authorization header required")
		return
	}

	response, statusCode, err := callExecute(ctx, &requestData)
	if err != nil {
		sendError(w, statusCode, "Failed to delete function: "+err.Error())
		return
	}

	// Parse backend response
	var backendResp ResultJSON
	if err := json.Unmarshal(*response, &backendResp); err != nil {
		sendError(w, http.StatusInternalServerError, "Failed to parse backend response: "+err.Error())
		return
	}

	sendSuccess(w, backendResp.Data)
}

// Save the entire function library to the backend configured file name
func saveLibraryHandler(w http.ResponseWriter, r *http.Request) {
	// Implementation here - use the callExecute function to save the library
	requestData := ExecRequestData{
		Program: fmt.Sprintf("saveFunctions('%s')", *libraryName),
	}

	// Get auth header from request
	ctx := context.Background()
	if authHeader := r.Header.Get("Authorization"); authHeader != "" {
		ctx = context.WithValue(ctx, contextKey("auth"), authHeader)
	} else {
		sendError(w, http.StatusUnauthorized, "Authorization header required")
		return
	}

	response, statusCode, err := callExecute(ctx, &requestData)
	if err != nil {
		sendError(w, statusCode, "Failed to save library: "+err.Error())
		return
	}

	// Parse backend response
	var backendResp ResultJSON
	if err := json.Unmarshal(*response, &backendResp); err != nil {
		sendError(w, http.StatusInternalServerError, "Failed to parse backend response: "+err.Error())
		return
	}

	sendSuccess(w, backendResp.Data)
}

// cleanupMetadataFiles removes macOS metadata files from the files directory
func cleanupMetadataFiles(directory string) {
	files, err := os.ReadDir(directory)
	if err != nil {
		log.Printf("Warning: Could not read directory for cleanup: %v", err)
		return
	}

	var deletedCount int
	for _, file := range files {
		if !file.IsDir() {
			name := file.Name()
			// Remove macOS metadata files and other unwanted hidden files
			if strings.HasPrefix(name, "._") ||
				strings.HasPrefix(name, ".DS_Store") ||
				strings.HasPrefix(name, ".Spotlight-") ||
				strings.HasPrefix(name, ".Trashes") {
				filePath := filepath.Join(directory, name)
				if err := os.Remove(filePath); err != nil {
					log.Printf("Warning: Could not remove metadata file %s: %v", name, err)
				} else {
					deletedCount++
					log.Printf("Cleaned up metadata file: %s", name)
				}
			}
		}
	}

	if deletedCount > 0 {
		log.Printf("Cleaned up %d metadata files from %s", deletedCount, directory)
	}
}

func runtimeInspectHandler(w http.ResponseWriter, r *http.Request) {
	// Example: call backend to get runtime info
	requestData := ExecRequestData{
		Program: "inspectRuntime()", // You must implement this in your backend
	}
	ctx := context.Background()
	if authHeader := r.Header.Get("Authorization"); authHeader != "" {
		ctx = context.WithValue(ctx, contextKey("auth"), authHeader)
	} else {
		sendError(w, http.StatusUnauthorized, "Authorization header required")
		return
	}
	response, statusCode, err := callExecute(ctx, &requestData)
	if err != nil {
		sendError(w, statusCode, "Failed to inspect runtime: "+err.Error())
		return
	}
	var backendResp ResultJSON
	if err := json.Unmarshal(*response, &backendResp); err != nil {
		sendError(w, http.StatusInternalServerError, "Failed to parse backend response: "+err.Error())
		return
	}
	sendSuccess(w, backendResp.Data)
}

func loadLibraryHandler(w http.ResponseWriter, r *http.Request) {
	// Implementation here
}

func debugBreakpointHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		sendError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	proxyDebugRequest(w, r, http.MethodPost, "/api/debug/breakpoint", true)
}

func debugStateHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		sendError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	proxyDebugRequest(w, r, http.MethodGet, "/api/debug/state", false)
}

func debugContinueHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		sendError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	proxyDebugRequest(w, r, http.MethodPost, "/api/debug/continue", false)
}

func debugPauseHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		sendError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	proxyDebugRequest(w, r, http.MethodPost, "/api/debug/pause", false)
}

func debugStepHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		sendError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	proxyDebugRequest(w, r, http.MethodPost, "/api/debug/step", true)
}

// healthHandler provides a simple health check endpoint
func healthHandler(w http.ResponseWriter, r *http.Request) {
	health := map[string]interface{}{
		"status":    "ok",
		"service":   "charioteer",
		"timestamp": time.Now().Unix(),
	}
	sendSuccess(w, health)
}

// Serve the chariot-codegen IIFE bundle from local filesystem
func codegenJSHandler(w http.ResponseWriter, r *http.Request) {
	// Try workspace path first
	paths := []string{
		filepath.Join("..", "..", "packages", "chariot-codegen", "dist", "index.global.js"),
		filepath.Join("packages", "chariot-codegen", "dist", "index.global.js"),
		filepath.Join(".", "index.global.js"),
	}
	for _, p := range paths {
		if _, err := os.Stat(p); err == nil {
			http.ServeFile(w, r, p)
			return
		}
	}
	http.Error(w, "codegen bundle not found", http.StatusNotFound)
}

func main() {
	flag.Parse()

	// Clean up metadata files on startup
	cleanupMetadataFiles("files")

	var err error
	mcpStore, err = newMCPStore(getMCPDataDir())
	if err != nil {
		log.Fatalf("failed to initialize MCP store: %v", err)
	}
	agentFixtureStore, err = newAgentFixtureStore(getMCPDataDir())
	if err != nil {
		log.Fatalf("failed to initialize agent fixture store: %v", err)
	}

	// Protected routes -- proxy file operations to backend
	http.HandleFunc("/api/session/profile", authMiddleware(sessionProfileHandler))
	http.HandleFunc("/api/files/", authMiddleware(fileGetProxyHandler))  // Handles /api/files/:name
	http.HandleFunc("/api/files", authMiddleware(filesListProxyHandler)) // Handles /api/files (list/save)
	http.HandleFunc("/api/execute", authMiddleware(executeHandler))
	http.HandleFunc("/api/execute-async", authMiddleware(executeAsyncHandler))
	http.HandleFunc("/api/logs/", authMiddleware(streamLogsHandler))
	http.HandleFunc("/api/result/", authMiddleware(getResultHandler))
	// Protected routes -- function library operations
	http.HandleFunc("/api/functions", authMiddleware(listFunctionsHandler))
	http.HandleFunc("/api/function", authMiddleware(getFunctionHandler))
	http.HandleFunc("/api/function/save", authMiddleware(saveFunctionHandler))
	http.HandleFunc("/api/function/delete", authMiddleware(deleteFunctionHandler))
	http.HandleFunc("/api/library/save", authMiddleware(saveLibraryHandler))
	http.HandleFunc("/api/library/load", authMiddleware(loadLibraryHandler))
	http.HandleFunc("/api/runtime/inspect", authMiddleware(runtimeInspectHandler))
	http.HandleFunc("/api/debug/breakpoint", authMiddleware(debugBreakpointHandler))
	http.HandleFunc("/api/debug/state", authMiddleware(debugStateHandler))
	http.HandleFunc("/api/debug/continue", authMiddleware(debugContinueHandler))
	http.HandleFunc("/api/debug/pause", authMiddleware(debugPauseHandler))
	http.HandleFunc("/api/debug/step", authMiddleware(debugStepHandler))
	http.HandleFunc("/api/mcp/settings", authMiddleware(mcpSettingsHandler))
	http.HandleFunc("/api/mcp/fixtures", authMiddleware(mcpFixturesHandler))
	http.HandleFunc("/api/mcp/fixtures/", authMiddleware(mcpFixtureItemHandler))
	http.HandleFunc("/api/mcp/tools", authMiddleware(mcpListToolsHandler))
	http.HandleFunc("/api/mcp/execute", authMiddleware(mcpExecuteHandler))
	http.HandleFunc("/api/agents/fixtures", authMiddleware(agentFixturesHandler))
	http.HandleFunc("/api/agents/fixtures/", authMiddleware(agentFixtureItemHandler))

	// Prefixed API routes for proxy path support
	http.HandleFunc("/charioteer/api/session/profile", authMiddleware(sessionProfileHandler))
	http.HandleFunc("/charioteer/api/files/", authMiddleware(fileGetProxyHandler))  // Handles /charioteer/api/files/:name
	http.HandleFunc("/charioteer/api/files", authMiddleware(filesListProxyHandler)) // Handles /charioteer/api/files (list/save)
	http.HandleFunc("/charioteer/api/execute", authMiddleware(executeHandler))
	http.HandleFunc("/charioteer/api/execute-async", authMiddleware(executeAsyncHandler))
	http.HandleFunc("/charioteer/api/logs/", authMiddleware(streamLogsHandler))
	http.HandleFunc("/charioteer/api/result/", authMiddleware(getResultHandler))
	http.HandleFunc("/charioteer/api/functions", authMiddleware(listFunctionsHandler))
	http.HandleFunc("/charioteer/api/function", authMiddleware(getFunctionHandler))
	http.HandleFunc("/charioteer/api/function/save", authMiddleware(saveFunctionHandler))
	http.HandleFunc("/charioteer/api/function/delete", authMiddleware(deleteFunctionHandler))
	http.HandleFunc("/charioteer/api/library/save", authMiddleware(saveLibraryHandler))
	http.HandleFunc("/charioteer/api/library/load", authMiddleware(loadLibraryHandler))
	http.HandleFunc("/charioteer/api/runtime/inspect", authMiddleware(runtimeInspectHandler))
	http.HandleFunc("/charioteer/api/debug/breakpoint", authMiddleware(debugBreakpointHandler))
	http.HandleFunc("/charioteer/api/debug/state", authMiddleware(debugStateHandler))
	http.HandleFunc("/charioteer/api/debug/continue", authMiddleware(debugContinueHandler))
	http.HandleFunc("/charioteer/api/debug/pause", authMiddleware(debugPauseHandler))
	http.HandleFunc("/charioteer/api/debug/step", authMiddleware(debugStepHandler))
	http.HandleFunc("/charioteer/api/mcp/settings", authMiddleware(mcpSettingsHandler))
	http.HandleFunc("/charioteer/api/mcp/fixtures", authMiddleware(mcpFixturesHandler))
	http.HandleFunc("/charioteer/api/mcp/fixtures/", authMiddleware(mcpFixtureItemHandler))
	http.HandleFunc("/charioteer/api/mcp/tools", authMiddleware(mcpListToolsHandler))
	http.HandleFunc("/charioteer/api/mcp/execute", authMiddleware(mcpExecuteHandler))
	http.HandleFunc("/charioteer/api/agents/fixtures", authMiddleware(agentFixturesHandler))
	http.HandleFunc("/charioteer/api/agents/fixtures/", authMiddleware(agentFixtureItemHandler))

	// Public routes
	http.HandleFunc("/charioteer/health", healthHandler)
	http.HandleFunc("/charioteer/editor", editorHandler)
	http.HandleFunc("/charioteer/dashboard", authMiddleware(dashboardHandler))
	http.HandleFunc("/charioteer/login", loginHandler)   // Implement loginHandler to handle login requests
	http.HandleFunc("/charioteer/logout", logoutHandler) // Implement logoutHandler to handle logout requests

	// Serve shared codegen bundle (both root and prefixed for proxy hosting)
	http.HandleFunc("/chariot-codegen.js", codegenJSHandler)
	http.HandleFunc("/charioteer/chariot-codegen.js", codegenJSHandler)

	// Dashboard API proxy route
	http.HandleFunc("/charioteer/api/dashboard/status", authMiddleware(dashboardAPIHandler))
	http.HandleFunc("/charioteer/api/agents", authMiddleware(agentsListHandler))

	// Agent management proxy routes -> go-chariot backend
	http.HandleFunc("/charioteer/api/agents/plans", authMiddleware(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			proxyToBackendJSON(w, r, http.MethodGet, "/api/agents/plans", nil)
		case http.MethodPost:
			body, _ := io.ReadAll(r.Body)
			proxyToBackendJSON(w, r, http.MethodPost, "/api/agents/plans", body)
		default:
			sendError(w, http.StatusMethodNotAllowed, "method not allowed")
		}
	}))

	http.HandleFunc("/charioteer/api/agents/create", authMiddleware(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			sendError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}
		body, _ := io.ReadAll(r.Body)
		proxyToBackendJSON(w, r, http.MethodPost, "/api/agents/create", body)
	}))

	http.HandleFunc("/charioteer/api/agents/stop", authMiddleware(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			sendError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}
		body, _ := io.ReadAll(r.Body)
		proxyToBackendJSON(w, r, http.MethodPost, "/api/agents/stop", body)
	}))

	http.HandleFunc("/charioteer/api/agents/publish", authMiddleware(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			sendError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}
		body, _ := io.ReadAll(r.Body)
		proxyToBackendJSON(w, r, http.MethodPost, "/api/agents/publish", body)
	}))

	http.HandleFunc("/charioteer/api/agents/belief", authMiddleware(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			sendError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}
		body, _ := io.ReadAll(r.Body)
		proxyToBackendJSON(w, r, http.MethodPost, "/api/agents/belief", body)
	}))

	http.HandleFunc("/charioteer/api/agents/run-once", authMiddleware(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			sendError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}
		body, _ := io.ReadAll(r.Body)
		proxyToBackendJSON(w, r, http.MethodPost, "/api/agents/run-once", body)
	}))

	// Agent info/beliefs routes with path parameters
	http.HandleFunc("/charioteer/api/agents/", authMiddleware(func(w http.ResponseWriter, r *http.Request) {
		// Parse path to extract agent name and sub-path
		// Format: /charioteer/api/agents/:name/beliefs or /charioteer/api/agents/:name/info
		path := strings.TrimPrefix(r.URL.Path, "/charioteer/api/agents/")
		parts := strings.SplitN(path, "/", 2)

		if len(parts) < 2 {
			sendError(w, http.StatusNotFound, "invalid agent path")
			return
		}

		agentName := parts[0]
		subPath := parts[1]

		if subPath == "beliefs" && r.Method == http.MethodGet {
			proxyToBackendJSON(w, r, http.MethodGet, "/api/agents/"+url.PathEscape(agentName)+"/beliefs", nil)
		} else if subPath == "info" && r.Method == http.MethodGet {
			proxyToBackendJSON(w, r, http.MethodGet, "/api/agents/"+url.PathEscape(agentName)+"/info", nil)
		} else {
			sendError(w, http.StatusNotFound, "unknown agent endpoint")
		}
	}))

	// Diagrams proxy endpoints -> go-chariot backend
	http.HandleFunc("/charioteer/api/diagrams", authMiddleware(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodPost {
			sendError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}
		var body []byte
		if r.Body != nil {
			b, _ := io.ReadAll(r.Body)
			body = b
		}
		proxyToBackendJSON(w, r, r.Method, "/api/diagrams", body)
	}))
	http.HandleFunc("/charioteer/api/diagrams/", authMiddleware(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodDelete {
			sendError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}
		name := strings.TrimPrefix(r.URL.Path, "/charioteer/api/diagrams/")
		if name == "" {
			sendError(w, http.StatusBadRequest, "diagram name required")
			return
		}
		proxyToBackendJSON(w, r, r.Method, "/api/diagrams/"+url.PathEscape(name), nil)
	}))
	// Listener API proxy routes
	http.HandleFunc("/charioteer/api/listeners", authMiddleware(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			listenersListHandler(w, r)
		case http.MethodPost:
			listenersCreateHandler(w, r)
		default:
			sendError(w, http.StatusMethodNotAllowed, "method not allowed")
		}
	}))
	http.HandleFunc("/charioteer/api/listeners/", authMiddleware(listenersItemHandler))
	http.HandleFunc("/charioteer/api/listener/delete", authMiddleware(listenersDeleteHandler))
	http.HandleFunc("/charioteer/api/listener/start", authMiddleware(listenersStartHandler))
	http.HandleFunc("/charioteer/api/listener/stop", authMiddleware(listenersStopHandler))
	http.HandleFunc("/charioteer/api/listener-scripts", authMiddleware(listenerScriptsCollectionHandler))
	http.HandleFunc("/charioteer/api/listener-scripts/", authMiddleware(listenerScriptItemHandler))
	// WebSocket proxy for dashboard stream (token passed as query param)
	http.HandleFunc("/charioteer/ws/dashboard", dashboardWSProxyHandler)
	// WebSocket proxy for agents stream (token passed as query param)
	http.HandleFunc("/charioteer/ws/agents", agentsWSProxyHandler)

	log.Println("Current working directory:", func() string { dir, _ := os.Getwd(); return dir }())
	log.Println("Chariot Editor server starting on :" + getPort())
	log.Println("Backend server URL:", getBackendURL())
	log.Println("Visit: https://localhost:" + getPort() + "/editor")

	if *useSSL {
		tlsKey, err := getTLSKey()
		if err != nil {
			log.Fatal("Failed to get TLS key:", err)
		}
		tlsCert, err := getTLSCert()
		if err != nil {
			log.Fatal("Failed to get TLS certificate:", err)
		}
		log.Println("Starting HTTPS server with TLS certs")
		log.Fatal(http.ListenAndServeTLS(":"+getPort(), tlsCert, tlsKey, nil))
	} else {
		log.Println("Starting HTTP server (no TLS)")
		log.Fatal(http.ListenAndServe(":"+getPort(), nil))
	}
}
