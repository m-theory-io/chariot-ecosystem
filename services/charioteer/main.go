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
	libraryName        = flag.String("library", "stlib.json", "Name of the library to use for function execution")
	insecureSkipVerify = flag.Bool("insecure", true, "Skip TLS certificate verification for backend (dev only)")
	certPath           = flag.String("certpath", ".certs", "cert file folder")
	useSSL             = flag.Bool("ssl", false, "Use HTTPS with TLS certs (default false for dev)")
)

// ResultJSON provides a standardized JSON response format
type ResultJSON struct {
	Result string      `json:"result"`
	Data   interface{} `json:"data"`
}

type ExecRequestData struct {
	Program string `json:"program"`
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

const editorTemplate = `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Charioteer Code Editor</title>
    <style>
        body { 
            margin: 0; 
            padding: 0; 
            font-family: 'Segoe UI', Tahoma, Geneva, Verdana, sans-serif;
            height: 100vh;
            overflow: hidden;
        }
        // Tabs for switching toolbar modes
        .toolbar-tabs {
            display: flex;
            gap: 4px;
            margin-right: 16px;
        }

        .toolbar-tab {
            background: #232326;
            color: #ccc;
            border: none;
            border-radius: 3px 3px 0 0;
            padding: 6px 18px 4px 18px;
            font-size: 14px;
            cursor: pointer;
            outline: none;
            border-bottom: 2px solid transparent;
            transition: background 0.15s, color 0.15s;
        }

        .toolbar-tab.active {
            background: #1e1e1e;
            color: #fff;
            border-bottom: 2px solid #007acc;
            font-weight: bold;
        }

        .toolbar-section {
            display: none;
        }
        .toolbar-section.active {
            display: flex;
        }        
        .toolbar {
            background-color: #2d2d30;
            color: white;
            padding: 10px;
            display: flex;
            align-items: center;
            gap: 15px;
            height: 50px;
            box-sizing: border-box;
            flex-shrink: 0;
            min-width: 0; /* Allow flexbox shrinking */
            overflow: hidden; /* Hide overflow instead of wrapping */
        }
        
        .file-selector {
            display: flex;
            align-items: center;
            gap: 8px;           /* Even spacing between label and select */
            margin-left: 0;     /* Remove any left margin */
            min-width: 0;
            padding-left: 0;    /* Remove any left padding */
        }

        .file-selector label {
            font-size: 14px;
            white-space: nowrap;
            min-width: 40px;    /* Optional: keep label width consistent */
            margin-right: 0;    /* Remove any right margin */
        }

        .file-selector select {
            background-color: #3c3c3c;
            color: white;
            border: 1px solid #555;
            border-radius: 3px;
            padding: 5px 10px;
            font-size: 14px;
            min-width: 150px;
            max-width: 200px;
            flex-shrink: 1;
            margin-right: 12px; /* Add space to the right of the dropdown */
        }
        
        .save-buttons {
            display: flex;
            align-items: center;
            gap: 8px;
            flex-shrink: 0; /* Don't shrink save buttons */
        }
        
        .toolbar-button {
            background-color: #4a4a4a;
            color: white;
            border: none;
            border-radius: 3px;
            padding: 6px 12px;
            font-size: 13px;
            cursor: pointer;
        }
        
        .toolbar-button:hover:not(:disabled) {
            background-color: #5a5a5a;
        }
        
        .toolbar-button:disabled {
            background-color: #3a3a3a;
            color: #888;
            cursor: not-allowed;
        }
        
        .auth-section {
            display: flex;
            align-items: center;
            gap: 10px;
            margin-left: auto;
            flex-shrink: 0;
            min-width: 0; /* Allow flexbox shrinking */
        }

        #functionsToolbar {
            align-items: center;
            gap: 8px;
            margin-left: 16px;
        }

        #functionSelect {
            background-color: #3c3c3c;
            color: white;
            border: 1px solid #555;
            border-radius: 3px;
            padding: 5px 10px;
            font-size: 14px;
            min-width: 150px;
            max-width: 200px;
            flex-shrink: 1;
            margin-right: 12px; /* Space before first button */
        }        
        
        #loginSection {
            display: flex;
            align-items: center;
            gap: 8px;
            flex-wrap: nowrap;
        }
        
        #loggedInSection {
            display: flex;
            align-items: center;
            gap: 10px;
            flex-wrap: nowrap;
            min-width: 0; /* Allow flexbox shrinking */
        }
        
        .auth-input {
            background-color: #3c3c3c;
            color: white;
            border: 1px solid #555;
            border-radius: 3px;
            padding: 5px 8px;
            font-size: 13px;
            width: 100px; /* Reduced from 120px */
            min-width: 80px; /* Minimum width */
            flex-shrink: 1; /* Allow shrinking */
        }
        
        .auth-button {
            background-color: #28a745;
            color: white;
            border: none;
            border-radius: 3px;
            padding: 6px 12px;
            font-size: 13px;
            cursor: pointer;
            flex-shrink: 0; /* Don't shrink buttons */
            white-space: nowrap;
        }
        
        .auth-button.logout {
            background-color: #dc3545;
        }
        
        .user-info {
            color: #4ec9b0;
            font-size: 13px;
            white-space: nowrap;
            overflow: hidden;
            text-overflow: ellipsis;
            max-width: 150px; /* Limit width to prevent overflow */
            flex-shrink: 1; /* Allow shrinking */
        }

        /* Session expiration warning modal */
        .session-warning-modal {
            position: fixed;
            top: 0;
            left: 0;
            width: 100%;
            height: 100%;
            background-color: rgba(0, 0, 0, 0.7);
            display: flex;
            justify-content: center;
            align-items: center;
            z-index: 10000;
            font-family: 'Segoe UI', Tahoma, Geneva, Verdana, sans-serif;
        }

        .session-warning-dialog {
            background-color: #2d2d30;
            color: #d4d4d4;
            border: 2px solid #007acc;
            border-radius: 8px;
            padding: 24px;
            max-width: 400px;
            width: 90%;
            box-shadow: 0 8px 32px rgba(0, 0, 0, 0.5);
            animation: modalSlideIn 0.3s ease-out;
        }

        @keyframes modalSlideIn {
            from {
                opacity: 0;
                transform: translateY(-30px);
            }
            to {
                opacity: 1;
                transform: translateY(0);
            }
        }

        .session-warning-title {
            font-size: 18px;
            font-weight: 600;
            margin-bottom: 16px;
            color: #f0ad4e;
            display: flex;
            align-items: center;
            gap: 8px;
        }

        .session-warning-message {
            font-size: 14px;
            line-height: 1.5;
            margin-bottom: 20px;
            color: #cccccc;
        }

        .session-warning-countdown {
            font-size: 16px;
            font-weight: 600;
            margin-bottom: 20px;
            color: #f0ad4e;
            text-align: center;
            padding: 8px;
            background-color: rgba(240, 173, 78, 0.1);
            border-radius: 4px;
        }

        .session-warning-buttons {
            display: flex;
            justify-content: space-between;
            gap: 12px;
        }

        .session-warning-button {
            flex: 1;
            padding: 10px 16px;
            border: none;
            border-radius: 4px;
            font-size: 14px;
            font-weight: 500;
            cursor: pointer;
            transition: all 0.2s ease;
        }

        .session-warning-button.extend {
            background-color: #28a745;
            color: white;
        }

        .session-warning-button.extend:hover {
            background-color: #218838;
        }

        .session-warning-button.logout {
            background-color: #dc3545;
            color: white;
        }

        .session-warning-button.logout:hover {
            background-color: #c82333;
        }        

        .run-button {
            background-color: #007acc;
            color: white;
            border: none;
            border-radius: 3px;
            padding: 8px 16px;
            font-size: 14px;
            cursor: pointer;
        }
        
        .run-button:disabled {
            background-color: #555;
            cursor: not-allowed;
        }

        .left-panel {
            width: 260px;
            background: #232326;
            color: #d4d4d4;
            border-right: 1px solid #333;
            padding: 0;
            overflow-y: auto;
            flex-shrink: 0;
            display: flex;
            flex-direction: column;
            min-width: 200px; /* Minimum width for resizing */
            max-width: 600px; /* Maximum width for resizing */
        }

        /* Add left panel splitter styles */
        .left-splitter {
            width: 4px;
            background-color: #3c3c3c;
            cursor: col-resize;
            flex-shrink: 0;
        }
        
        .left-splitter:hover {
            background-color: #007acc;
        }

        .left-panel div[style*="font-weight:bold"] {
            font-size: 13px !important;
            color: #f5f5f5 !important; /* ivory/white */
            font-weight: 500 !important;
            margin-bottom: 2px !important;
            letter-spacing: 0.5px;
        }

        .main-container {
            display: flex;
            flex-direction: row;
            height: 100vh;
            position: relative;
            min-height: 0;
        }

        .center-panel {
            display: flex;
            flex-direction: column;
            flex: 1 1 0%;
            min-width: 0;
            min-height: 0;
        }

        .editor-container {
            flex: 1 1 0%;
            min-width: 0;
            min-height: 0;
            background-color: #1e1e1e;
        }
        
        .splitter {
            height: 4px;
            background-color: #3c3c3c;
            cursor: row-resize;
            flex-shrink: 0;
        }
        
        .splitter:hover {
            background-color: #007acc;
        }
        
        .bottom-panel {
            background-color: #252526;
            border-top: 1px solid #3c3c3c;
            height: 250px;
            display: flex;
            flex-direction: column;
            flex-shrink: 0;
        }
        
        .tab-bar {
            background-color: #2d2d30;
            display: flex;
            align-items: center;
            height: 35px;
            border-bottom: 1px solid #3c3c3c;
            flex-shrink: 0;
        }
        
        .tab {
            background-color: #2d2d30;
            color: #ccc;
            border: none;
            padding: 8px 16px;
            cursor: pointer;
            font-size: 13px;
            border-right: 1px solid #3c3c3c;
        }
        
        .tab.active {
            background-color: #1e1e1e;
            color: white;
        }
        
        .tab-content {
            flex: 1;
            padding: 10px;
            overflow-y: auto;
            background-color: #1e1e1e;
            color: #d4d4d4;
            font-family: 'Consolas', 'Monaco', 'Courier New', monospace;
            font-size: 13px;
            white-space: pre-wrap;
        }

        /* Tree viewer styles */
        .left-panel .tree-view {
            font-family: 'Consolas', 'Monaco', monospace;
            font-size: 13px;
            line-height: 1.5;
            padding: 10px;
            user-select: text;
        }
        .tree-key {
            color: #4ec9b0;
            cursor: pointer;
        }
        .tree-toggle {
            cursor: pointer;
            color: #ffd700;
            margin-right: 4px;
            font-weight: bold;
        }
        .tree-collapsed > .tree-children {
            display: none;
        }
        .tree-leaf {
            color: #d4d4d4;
        }
        
        .tree-node-type {
            color: #ffb86c;
            font-style: italic;
        }

        .tree-function {
            color: #ffb86c;
            cursor: pointer;
            text-decoration: underline;
        }
        
        .tree-function:hover {
            color: #ffd700;
        }
        
        .output-success { color: #4ec9b0; }
        .output-error { color: #f44747; }
        .output-info { color: #569cd6; }
        .loading { color: #ffcc02; }

        /* Enhanced bracket highlighting */
        .monaco-editor .bracket-match {
            background-color: rgba(0, 122, 204, 0.3) !important;
            border: 1px solid #007acc !important;
            border-radius: 2px !important;
        }

        .monaco-editor .bracket-highlight {
            background-color: rgba(0, 122, 204, 0.2) !important;
            box-shadow: 0 0 0 1px rgba(0, 122, 204, 0.6) !important;
            border-radius: 2px !important;
        }

        /* Bracket pair colorization */
        .monaco-editor .bracket-highlighting-0 { color: #FFD700 !important; }
        .monaco-editor .bracket-highlighting-1 { color: #DA70D6 !important; }
        .monaco-editor .bracket-highlighting-2 { color: #87CEEB !important; }
        .monaco-editor .bracket-highlighting-3 { color: #98FB98 !important; }
        .monaco-editor .bracket-highlighting-4 { color: #F0E68C !important; }
        .monaco-editor .bracket-highlighting-5 { color: #FF6347 !important; }

        /* Bracket guides */
        .monaco-editor .bracket-pair-guide {
            border-left: 1px solid rgba(0, 122, 204, 0.3) !important;
        }

        .monaco-editor .bracket-pair-guide.active {
            border-left: 1px solid rgba(0, 122, 204, 0.8) !important;
        }        

        /* Chariot syntax highlighting colors for Monaco editor */
        .monaco-editor .token.keyword.control.chariot { color: #c586c0 !important; }
        .monaco-editor .token.keyword.chariot.array { color: #4fc1ff !important; }
        .monaco-editor .token.keyword.chariot.bdi { color: #4fc1ff !important; }
        .monaco-editor .token.keyword.chariot.comparison { color: #ffd700 !important; }
        .monaco-editor .token.keyword.chariot.couchbase { color: #ff6b6b !important; }
        .monaco-editor .token.keyword.chariot.crypto { color: #FF6B6B !important; }
        .monaco-editor .token.keyword.chariot.date { color: #4ecdc4 !important; }
        .monaco-editor .token.keyword.chariot.dispatcher { color: #95e1d3 !important; }
        .monaco-editor .token.keyword.chariot.etl { color: #a8e6cf !important; }
        .monaco-editor .token.keyword.chariot.file { color: #ffd3a5 !important; }
        .monaco-editor .token.keyword.chariot.flow { color: #c586c0 !important; }
        .monaco-editor .token.keyword.chariot.host { color: #fd79a8 !important; }
        .monaco-editor .token.keyword.chariot.json { color: #fdcb6e !important; }
        .monaco-editor .token.keyword.chariot.math { color: #6c5ce7 !important; }
        .monaco-editor .token.keyword.chariot.knapsack { color: #a29bfe !important; }
        .monaco-editor .token.keyword.chariot.rl { color: #a29bfe !important; }
        .monaco-editor .token.keyword.chariot.node { color: #a29bfe !important; }
        .monaco-editor .token.keyword.chariot.csv { color: #fdcb6e !important; }
        .monaco-editor .token.keyword.chariot.sql { color: #fd79a8 !important; }
        .monaco-editor .token.keyword.chariot.string { color: #00b894 !important; }
        .monaco-editor .token.keyword.chariot.system { color: #e17055 !important; }
        .monaco-editor .token.keyword.chariot.value { color: #0984e3 !important; }
        .monaco-editor .token.keyword.function.user { color: #ffb86c !important; }

        /* Responsive design for narrow screens */
        @media (max-width: 1200px) {
            .toolbar {
                gap: 10px; /* Reduce gap */
            }
            
            .auth-input {
                width: 80px; /* Further reduce input width */
                min-width: 60px;
            }
            
            .file-selector select {
                min-width: 120px;
                max-width: 150px;
            }
            
            .user-info {
                max-width: 120px; /* Further reduce user info width */
                font-size: 12px;
            }
        }
        
        @media (max-width: 900px) {
            .toolbar {
                gap: 8px;
                padding: 8px;
            }
            
            .auth-input {
                width: 70px;
                min-width: 50px;
                font-size: 12px;
                padding: 4px 6px;
            }
            
            .auth-button {
                padding: 4px 8px;
                font-size: 12px;
            }
            
            .file-selector label {
                display: none; /* Hide "File:" label on very narrow screens */
            }
            
            .user-info {
                max-width: 100px;
                font-size: 11px;
            }
        }

        .file-action {
            margin-left: 10px; /* Add some separation from save buttons */
        }
        
        .file-action.delete {
            background-color: #dc3545; /* Red background for delete */
        }
        
        .file-action.delete:hover:not(:disabled) {
            background-color: #c82333; /* Darker red on hover */
        }

    </style>
</head>
<body>
    <div class="main-container">
        <div class="left-panel" id="leftPanel">
            <!-- Runtime inspection UI will go here -->
        </div>
        <div class="left-splitter" id="leftSplitter"></div>
        <div class="center-panel" id="centerPanel">    
            <div class="toolbar">
                <div class="toolbar-tabs">
                    <button id="filesTab" class="toolbar-tab active">Files</button>
                    <button id="functionsTab" class="toolbar-tab">Function Library</button>
                    <button id="diagramsTab" class="toolbar-tab">Diagrams</button>
                    <button id="dashboardTab" class="toolbar-tab">Dashboard</button>
                    <button id="agentsTab" class="toolbar-tab">Agents</button>
                </div>
                <div id="fileToolbar" class="toolbar-section active">
                    <div class="file-selector">
                        <label for="fileSelect">File:</label>
                        <select id="fileSelect" disabled>
                            <option value="">Select a file...</option>
                        </select>
                    </div>
                    
                    <div class="save-buttons">
                        <button id="newButton" class="toolbar-button">📄 New</button>
                        <button id="saveButton" class="toolbar-button" disabled>💾 Save</button>
                        <button id="saveAsButton" class="toolbar-button" disabled>💾 Save As...</button>
                        <button id="renameButton" class="toolbar-button file-action" disabled>📝 Rename</button>
                        <button id="deleteButton" class="toolbar-button file-action delete" disabled>🗑️ Delete</button>
                    </div>
                    
                </div>
                <div id="functionsToolbar" class="toolbar-section">
                    <select id="functionSelect" disabled>
                        <option value="">Select a function...</option>
                    </select>
                    <button id="newFunctionButton" class="toolbar-button" disabled>📄 New</button>
                    <button id="saveFunctionButton" class="toolbar-button" disabled>💾 Save</button>
                    <button id="saveAsFunctionButton" class="toolbar-button" disabled>💾 Save As...</button>
                    <button id="deleteFunctionButton" class="toolbar-button file-action delete" disabled>🗑️ Delete</button>
                    <button id="saveLibraryButton" class="toolbar-button" disabled>💾 Save Library</button>
                </div>
                <div id="dashboardToolbar" class="toolbar-section">
                    <button id="refreshDashboardButton" class="toolbar-button" onclick="refreshDashboardData()">🔄 Refresh</button>
                </div>
                <div id="agentsToolbar" class="toolbar-section">
                    <button id="refreshAgentsButton" class="toolbar-button">🔄 Refresh</button>
                </div>
                <div id="diagramsToolbar" class="toolbar-section">
                    <div class="file-selector">
                        <label for="diagramSelect">Diagram:</label>
                        <select id="diagramSelect" disabled>
                            <option value="">Select a diagram...</option>
                        </select>
                    </div>
                    <div class="save-buttons">
                        <button id="refreshDiagramsButton" class="toolbar-button">🔄 Refresh</button>
                        <button id="saveDiagramButton" class="toolbar-button" disabled>💾 Save</button>
                        <button id="saveAsDiagramButton" class="toolbar-button" disabled>💾 Save As...</button>
                        <button id="deleteDiagramButton" class="toolbar-button file-action delete" disabled>🗑️ Delete</button>
                    </div>
                </div>
                <div class="run-controls" style="display: flex; align-items: center; gap: 8px;">
                    <button id="runButton" class="run-button" disabled>▶ Run</button>
                    <label style="display: flex; align-items: center; gap: 4px; font-size: 13px; cursor: pointer;">
                        <input type="checkbox" id="streamingToggle" checked style="cursor: pointer;">
                        <span>Stream Logs</span>
                    </label>
                </div>
                
                <div class="auth-section">
                    <div id="loginSection">
                        <input type="text" id="usernameInput" placeholder="Username" class="auth-input">
                        <input type="password" id="passwordInput" placeholder="Password" class="auth-input">
                        <button id="loginButton" class="auth-button">Login</button>
                    </div>
                    <div id="loggedInSection" style="display: none;">
                        <span class="user-info"><span id="currentUserSpan"></span></span>
                        <button id="logoutButton" class="auth-button logout">Logout</button>
                    </div>
                </div>
            </div>
            <div class="editor-container" id="editorContainer"></div>
            <div class="splitter" id="splitter"></div>
            <div class="bottom-panel" id="bottomPanel">
                <div class="tab-bar">
                    <button class="tab active" data-tab="output">Output</button>
                    <button class="tab" data-tab="problems">Problems</button>
                </div>
                <div class="tab-content" id="outputContent">Please log in to use the editor...</div>
                <div class="tab-content" id="problemsContent" style="display:none;"></div>
            </div>
        </div>
    </div>

    <script src="https://cdn.jsdelivr.net/npm/monaco-editor@0.45.0/min/vs/loader.js"></script>
    <script src="chariot-codegen.js"></script>
    <script>
        // Configuration
        const CHARIOT_FILES_FOLDER = 'files';
        const SESSION_DURATION_MINUTES = 30; // 30 minutes session duration
        const WARNING_BEFORE_MINUTES = 3; // Show warning 3 minutes before expiration
        const LOGOUT_BEFORE_SECONDS = 30; // Auto-logout 30 seconds before expiration        
        // Chariot base rules
        const CHARIOT_MONARCH_BASE_RULES = [
            // Comments
            [/\/\/.*$/, 'comment'],
            [/\/\*/, 'comment', '@comment'],

            // Strings
            [/"([^"\\]|\\.)*$/, 'string.invalid'],
            [/"/, 'string', '@string'],
            [/'([^'\\]|\\.)*$/, 'string.invalid'],
            [/'/, 'string', '@string_single'],

            // Numbers
            [/\d*\.\d+([eE][-+]?\d+)?/, 'number.float'],
            [/\d+/, 'number'],

            // True keywords (special parser handling, not function calls)
            [/\belse|break|continue\b(?!\s*\()/, 'keyword.control.chariot'],

            // Special control flow constructs (create special AST nodes, not FuncCall)
            [/\b(if|while|func|switch|case|default)\b(?=\s*\()/, 'keyword.control.chariot'],

            // Chariot specific functions (only when followed by parens)
			[/\b(findUser|createUser|updateUser|deleteUser|authenticateUser|setUserPassword|generateToken|validateDisplayName)\b(?=\s*\()/, 'keyword.auth'],
            [/\b(addTo|array|lastIndex|range|removeAt|reverse|setAt|slice)\b(?=\s*\()/, 'keyword.chariot.array'],
            [/\b(agentBelief|agentNew|agentRegister|agentStart|agentStartNamed|agentStopNamed|agentList|agentPublish|agentStop|belief|plan|runPlanOnce|runPlanOnceBDI|runPlanOnceEx)\b(?=\s*\()/, 'keyword.chariot.bdi'],
            [/\b(and|bigger|biggerEq|equal|iif|not|or|unequal|smaller|smallerEq)\b(?=\s*\()/, 'keyword.chariot.comparison'],
            [/\b(cbClose|cbConnect|cbGet|cbInsert|cbOpenBucket|cbQuery|cbRemove|cbReplace|cbSetScope|cbUpsert|newID)\b(?=\s*\()/, 'keyword.chariot.couchbase'],
            [/\b(date|dateAdd|dateDiff|dateSchedule|day|dayCount|dayOfWeek|endOfMonth|formatDate|isBusinessDay|isDate|isEndOfMonth|julianDay|month|nextBusinessDay|now|today|utcTime|year|yearFraction)\b(?=\s*\()/, 'keyword.chariot.date'],
            [/\b(apply|clone|contains|getAllMeta|getAt|getAttributes|getMeta|getProp|indexOf|setMeta|setProp|length)\b(?=\s*\()/, 'keyword.chariot.dispatcher'],
            [/\b(addMapping|addMappingWithTransform|createTransform|doETL|etlStatus|getTransform|listTransforms|registerTransform)\b(?=\s*\()/, 'keyword.chariot.etl'],
            [/\b(deleteFile|convertJSONFileToYAML|convertYAMLFileToJSON|fileExists|getFileSize|jsonToYAML|jsonToYAMLNode|listFiles|loadCSV|loadCSVRaw|loadJSON|loadJSONRaw|loadYAML|loadYAMLMultiDoc|loadYAMLRaw|loadXML|loadXMLRaw|parseXMLString|readFile|saveCSV|saveCSVRaw|saveJSON|saveJSONRaw|saveYAML|saveYAMLMultiDoc|saveYAMLRaw|saveXML|saveXMLRaw|writeFile|yamlToJSON|yamlToJSONNode)\b(?=\s*\()/, 'keyword.chariot.file'],
            [/\b(encrypt|decrypt|sign|verify|hash256|hash512|generateKey|generateRSAKey|randomBytes)\b(?=\s*\()/, 'keyword.chariot.crypto'],
            [/\b(break|continue)\b(?=\s*\()/, 'keyword.chariot.flow'],
            [/\b(callMethod||getHostObject|hostObject)\b(?=\s*\()/, 'keyword.chariot.host'],
            [/\b(parseJSON|parseJSONValue|toJSON|toSimpleJSON)\b(?=\s*\()/, 'keyword.chariot.json'],
            [/\b(abs|add|amortize|apr|avg|balloon|ceil|ceiling|cos|depreciation|div|e|exp|floor|fv|int|irr|ln|loanBalance|log|log10|log2|max|min|mod|mul|nper|npv|pct|pi|pmt|pow|pv|random|randomSeed|randomString|rate|round|sin|sqrt|sub|sum|tan)\b(?=\s*\()/, 'keyword.chariot.math'],
            [/\b(parseJSON|parseJSONValue|toJSON|toSimpleJSON)\b(?=\s*\()/, 'keyword.chariot.json'],
            [/\b(knapsack|knapsackConfig)\b(?=\s*\()/, 'keyword.chariot.knapsack'],
            [/\b(rlInit|rlScore|rlLearn|rlClose|rlSelectBest|extractRLFeatures|rlExplore|nbaDecision)\b(?=\s*\()/, 'keyword.chariot.rl'],
            [/\b(addChild|childCount|clear|cloneNode|create|csvNode|findByName|firstChild|getAttribute|getChildAt|getChildByName|getDepth|getLevel|getName|getParent|getPath|getRoot|getSiblings|getText|hasAttribute|isLeaf|isRoot|jsonNode|lastChild|list|mapNode|nodeToString|queryNode|removeAttribute|removeChild|setAttribute|setAttributes|setChildByName|setName|setText|traverseNode|xmlNode|yamlNode)\b(?=\s*\()/, 'keyword.chariot.node'],
            [/\b(csvHeaders|csvRowCount|csvColumnCount|csvGetRow|csvGetCell|csvToCSV|csvLoad)\b(?=\s*\()/, 'keyword.chariot.csv'],
            [/\b(generateCreateTable|sqlBegin|sqlConnect|sqlClose|sqlCommit|sqlExecute|sqlListTables|sqlQuery|sqlRollback)\b(?=\s*\()/, 'keyword.chariot.sql'],
            [/\b(append|ascii|atPos|char|charAt|concat|digits|format|hasPrefix|hasSuffix|interpolate|join|lastPos|lower|occurs|padLeft|padRight|repeat|replace|right|split|sprintf|string|strlen|substr|substring|trim|trimLeft|trimRight|upper)\b(?=\s*\()/, 'keyword.chariot.string'],
            [/\b(exit|getEnv|hasEnv|listen|logPrint|mcpCallTool|mcpConnect|mcpClose|mcpListTools|platform|sleep|timeFormat|timestamp)\b(?=\s*\()/, 'keyword.chariot.system'],
            [/\b(newTree|treeFind|treeGetMetadata|treeLoad|treeLoadSecure|treeSave|treeSaveSecure|treeSearch||treeToYAML|treeToXML|treeValidateSecure|treeWalk)\b(?=\s*\()/, 'keyword.chariot.tree'],
            [/\b(boolean|call|declare|declareGlobal|deleteFunction|destroy|empty|exists|func|function|getFunction|getVariable|hasMeta|inspectRuntime|isNull|isNumeric|listFunctions|loadFunctions|mapValue|merge|offerVar|offerVariable|registerFunction|saveFunctions|setValue|setq|symbol|toBool|toMapValue|toNumber|toString|typeOf|valueOf)\b(?=\s*\()/, 'keyword.chariot.value'],
            [/\bfunction\b/, 'keyword.control.chariot'], // Always highlight 'function' as a keyword
            [/[a-zA-Z_$][\w$]*/, 'identifier'], 
        ];

        // Wait for DOM to be ready (and handle already-loaded state)
        (function initWhenReady() {
            const start = function() {
                try { bindAuthHandlers(); } catch (e) { console.warn('bindAuthHandlers error', e); }
                initializeEditor();
            };
            if (document.readyState === 'loading') {
                document.addEventListener('DOMContentLoaded', start);
            } else {
                start();
            }
        })();
        
        // Pin Monaco to a specific version for stability
        require.config({ paths: { vs: 'https://cdn.jsdelivr.net/npm/monaco-editor@0.45.0/min/vs' } });
        // Ensure auth/login handlers are attached at least once
        let authHandlersInitialized = false;
        function bindAuthHandlers() {
            if (authHandlersInitialized) return;
            const loginButton = document.getElementById('loginButton');
            const logoutButton = document.getElementById('logoutButton');
            const passwordInput = document.getElementById('passwordInput');
            if (loginButton) {
                loginButton.addEventListener('click', login);
            }
            if (logoutButton) {
                logoutButton.addEventListener('click', logout);
            }
            if (passwordInput) {
                passwordInput.addEventListener('keypress', function(e) { if (e.key === 'Enter') login(); });
            }
            authHandlersInitialized = true;
        }
        
        let editor;
        let fileEditorContent = '';         // Last content loaded in Files tab
        let fileEditorFileName = '';        // Last file loaded in Files tab
        let functionEditorContent = '';     // Last content loaded in Function Library tab
    let functionEditorFunctionName = ''; // Last function loaded in Function Library tab
        let dashboardContent = '';          // Last dashboard HTML content
        let dashboardLoaded = false;        // Track if dashboard has been loaded
        let currentFileName = '';
        let currentTab = 'output';
    let dashboardAutoRefresh = null;    // Timer for auto-refreshing dashboard when visible
    let currentBottomTab = 'output';    // Tracks the bottom panel tab (output|problems)
    // Throttle WS updates to avoid overwhelming UI
    let dashboardWSUpdateTimer = null;
    let pendingDashboardData = null;
    // Persist listener row selections across updates
    const selectedListeners = new Set();
    let isResizing = false;
        let authToken = null;
        let currentUser = null;
        let isFileModified = false;
        let originalContent = '';
    // Ensure toolbar handlers for Function Library are only initialized once across editor rebuilds
    let functionsToolbarInitialized = false;
    let diagramsToolbarInitialized = false;
    // Diagrams state
    let currentDiagramName = '';
    let currentDiagramJSON = null; // last loaded JSON for selected diagram
        // Session management variables
        let sessionTimer = null;
        let warningTimer = null;
        let logoutTimer = null;
        let sessionWarningShown = false;        
        
        // Initialize the editor
        function initializeEditor() {
            // Initialize Monaco Editor
            require(['vs/editor/editor.main'], function () {
                setupMonacoEditor();
            });
        }
        
        // Initialize splitter for resizing
        function initializeSplitter() {
            const splitter = document.getElementById('splitter');
            const bottomPanel = document.getElementById('bottomPanel');
            
            splitter.addEventListener('mousedown', function(e) {
                isResizing = true;
                document.addEventListener('mousemove', handleResize);
                document.addEventListener('mouseup', stopResize);
                e.preventDefault();
            });
            
            function handleResize(e) {
                if (!isResizing) return;
                
                const containerHeight = window.innerHeight - 50;
                const mouseY = e.clientY - 50;
                const newBottomHeight = containerHeight - mouseY;
                
                if (newBottomHeight >= 100 && newBottomHeight <= containerHeight - 200) {
                    bottomPanel.style.height = newBottomHeight + 'px';
                    if (editor) {
                        editor.layout();
                    }
                }
            }
            
            function stopResize() {
                isResizing = false;
                document.removeEventListener('mousemove', handleResize);
                document.removeEventListener('mouseup', stopResize);
            }
        }


        // Initialize left panel splitter for resizing
        function initializeLeftSplitter() {
            const leftSplitter = document.getElementById('leftSplitter');
            const leftPanel = document.getElementById('leftPanel');
            let isLeftResizing = false;
            
            leftSplitter.addEventListener('mousedown', function(e) {
                isLeftResizing = true;
                document.addEventListener('mousemove', handleLeftResize);
                document.addEventListener('mouseup', stopLeftResize);
                e.preventDefault();
            });
            
            function handleLeftResize(e) {
                if (!isLeftResizing) return;
                
                const newWidth = e.clientX;
                
                if (newWidth >= 200 && newWidth <= 600) {
                    leftPanel.style.width = newWidth + 'px';
                    if (editor) {
                        editor.layout();
                    }
                }
            }
            
            function stopLeftResize() {
                isLeftResizing = false;
                document.removeEventListener('mousemove', handleLeftResize);
                document.removeEventListener('mouseup', stopLeftResize);
            }
        }

        // Helper function to get auth headers
        function getAuthHeaders() {
            const headers = {};
            if (authToken) {
                headers['Authorization'] = authToken;
            } else {
                // Check localStorage for token
                const savedToken = localStorage.getItem('chariot_token');
                if (savedToken) {
                    headers['Authorization'] = savedToken;
                    authToken = savedToken; // Update current token
                }
            }
            console.log('DEBUG: getAuthHeaders', headers);
            return headers;
        }
        
        // Helper function to get auth headers with JSON content type
        function getAuthHeadersWithJSON() {
            const headers = getAuthHeaders();
            headers['Content-Type'] = 'application/json';
            console.log('DEBUG: getAuthHeadersWithJSON', headers);
            return headers;
        }
        
        // Helper function to get the correct API path (with /charioteer prefix when needed)
        function getAPIPath(path) {
            // Check if we're being accessed through a proxy path (like /charioteer/)
            const currentPath = window.location.pathname;
            if (currentPath.startsWith('/charioteer/')) {
                // If the path doesn't already start with /charioteer/, add it
                if (!path.startsWith('/charioteer/')) {
                    return '/charioteer' + path;
                }
            }
            return path;
        }
        
        // Set up Monaco editor (called from initializeEditor)
        function setupMonacoEditor() {
            // Dispose of existing editor instance completely before creating new one
            if (editor) {
                try {
                    const model = editor.getModel();
                    if (model) model.dispose();
                    editor.dispose();
                } catch (e) {
                    console.warn('Editor disposal error:', e);
                }
                editor = null;
            }
            
            // Register Chariot language
            monaco.languages.register({ id: 'chariot' });
            
            // Set up Chariot syntax highlighting with NO user functions initially
            setChariotTokenizer([]);
            
            // Create editor with empty content
            editor = monaco.editor.create(document.getElementById('editorContainer'), {
                value: '',
                language: 'chariot',
                theme: 'vs-dark',
                automaticLayout: true,
                fontSize: 14,
                minimap: { enabled: true },
                scrollBeyondLastLine: false,
                wordWrap: 'on',
                // Add bracket matching configuration
                matchBrackets: 'always',
                renderLineHighlight: 'gutter',
                bracketPairColorization: {
                    enabled: true
                },
                guides: {
                    bracketPairs: true,
                    bracketPairsHorizontal: false,
                    highlightActiveBracketPair: true
                },
                // Disable any conflicting keybindings
                cursorBlinking: 'smooth',
                cursorSmoothCaretAnimation: false
            });
            
            // Add event listener for content changes            
            // Initialize UI
            trackFileChanges(); // Add this line
            
            // Initialize event handlers
            initializeEventHandlers();
 
            // Initialize splitters
            initializeSplitter();
            initializeLeftSplitter();

            // Set initial UI state (logged out)
            updateAuthUI(false);
            
            // Check for existing token
            checkExistingAuth();

            initializeFunctionLibraryToolbar();
        }
        
        // Check for existing authentication
        function checkExistingAuth() {
            const savedToken = localStorage.getItem('chariot_token');
            const savedUser = localStorage.getItem('chariot_user');
            
            if (savedToken && savedUser) {
                authToken = savedToken;
                currentUser = savedUser;

                // Start session management for existing session
                startSessionManagement();                

                updateAuthUI(true);
                fetchUserFunctions().then(functionNames => {
                    setChariotTokenizer(functionNames);
                });
                loadFileList();
            }
        }
        
        function setChariotTokenizer(userFunctions) {
            // Clone the base rules
            let rules = CHARIOT_MONARCH_BASE_RULES.slice();

            // Insert user function rule before the identifier rule
            if (userFunctions && userFunctions.length > 0) {
                const escaped = userFunctions.map(fn => fn.replace(/[.*+?^${}()|[\]\\]/g, '\\$&'));
                // Build the regex as a string, not a RegExp object!
                const userFuncRegexStr = "\\b(" + escaped.join('|') + ")\\b(?=\\s*\\()";
                // Find the index of the identifier rule
                const idx = rules.findIndex(rule => Array.isArray(rule) && rule[1] === 'identifier');
                if (idx !== -1) {
                    rules.splice(idx, 0, [userFuncRegexStr, 'keyword.function.user']);
                }
            }

            monaco.languages.setMonarchTokensProvider('chariot', {
                tokenizer: {
                    root: rules,
                    comment: [
                        [/[^/*]+/, 'comment'],
                        [/\*\//, 'comment', '@pop'],
                        [/[/*]/, 'comment']
                    ],
                    
                    string: [
                        [/[^\\"]+/, 'string'],
                        [/\\./, 'string.escape'],
                        [/"/, 'string', '@pop']
                    ],
                    
                    string_single: [
                        [/[^\\']+/, 'string'],
                        [/\\./, 'string.escape'],
                        [/'/, 'string', '@pop']
                    ]
                }
            });
            // Configure bracket pairs for Chariot language
            monaco.languages.setLanguageConfiguration('chariot', {
                brackets: [
                    ['(', ')'],
                    ['[', ']'],
                    ['{', '}']
                ],
                autoClosingPairs: [
                    { open: '(', close: ')' },
                    { open: '[', close: ']' },
                    { open: '{', close: '}' },
                    { open: '"', close: '"' },
                    { open: "'", close: "'" }
                ],
                surroundingPairs: [
                    { open: '(', close: ')' },
                    { open: '[', close: ']' },
                    { open: '{', close: '}' },
                    { open: '"', close: '"' },
                    { open: "'", close: "'" }
                ],
                colorizedBracketPairs: [
                    ['(', ')'],
                    ['[', ']'],
                    ['{', '}']
                ]
            });

        }

        // Update authentication UI
        function updateAuthUI(isLoggedIn) {
            const loginSection = document.getElementById('loginSection');
            const loggedInSection = document.getElementById('loggedInSection');
            const fileSelect = document.getElementById('fileSelect');
            const runButton = document.getElementById('runButton');
            const currentUserSpan = document.getElementById('currentUserSpan');

            // Check if elements exist before trying to use them
            if (!loginSection || !loggedInSection || !fileSelect || !runButton) {
                console.log('DEBUG: Some UI elements not found, deferring updateAuthUI');
                console.log('DEBUG: loginSection:', !!loginSection, 'loggedInSection:', !!loggedInSection, 'fileSelect:', !!fileSelect, 'runButton:', !!runButton);
                return;
            }

            updateSaveButtonStates();            

            if (isLoggedIn) {
                loginSection.style.display = 'none';
                loggedInSection.style.display = 'flex';
                fileSelect.disabled = false;
                // Don't enable run button here - it should only be enabled when there's code
                console.log('DEBUG: Login successful, Run button remains disabled until code is present');
                if (currentUserSpan) {
                    currentUserSpan.textContent = currentUser;
                }
                showOutput('Ready', 'success');
            } else {
                currentFileName = '';
                originalContent = '';
                loginSection.style.display = 'flex';
                loggedInSection.style.display = 'none';
                fileSelect.disabled = true;
                runButton.disabled = true;
                showOutput('Please log in to use the editor', 'info');
            }
        }
        
        function initializeFunctionLibraryToolbar() {
            if (functionsToolbarInitialized) return;
            const functionSelect = document.getElementById('functionSelect');
            const newFunctionButton = document.getElementById('newFunctionButton');
            const saveFunctionButton = document.getElementById('saveFunctionButton');
            const saveAsFunctionButton = document.getElementById('saveAsFunctionButton');
            const deleteFunctionButton = document.getElementById('deleteFunctionButton');
            const saveLibraryButton = document.getElementById('saveLibraryButton');

            // Populate dropdown when Function Library tab is activated
            document.getElementById('functionsTab').addEventListener('click', async function() {
                await loadFunctionList();
                // Restore previous selection if available
                const functionSelect = document.getElementById('functionSelect');
                if (functionEditorFunctionName && functionSelect) {
                    functionSelect.value = functionEditorFunctionName;
                }
            });

            // New function handler
            newFunctionButton.addEventListener('click', newFunction);

            // Save function handler
            saveFunctionButton.addEventListener('click', saveFunction);

            // Save As function handler
            saveAsFunctionButton.addEventListener('click', saveAsFunction);

            // Save library handler
            saveLibraryButton.addEventListener('click', saveLibrary);

            // Delete function handler
            deleteFunctionButton.addEventListener('click', async function() {
                if (!authToken) {
                    showOutput('Please log in first', 'error');
                    return;
                }
                const functionName = functionSelect.value;
                if (!functionName) {
                    showOutput('No function selected to delete', 'error');
                    return;
                }
                if (!confirm('Are you sure you want to delete the function "' + functionName + '"? This cannot be undone.')) {
                    return;
                }
                deleteFunctionButton.disabled = true;
                deleteFunctionButton.textContent = '🗑️ Deleting...';
                try {
                    const response = await fetch('/charioteer/api/function/delete?name=' + encodeURIComponent(functionName), {
                        method: 'DELETE',
                        headers: getAuthHeaders()
                    });
                    const result = await response.json();
                    if (response.ok && result.result === "OK") {
                        showOutput('Function deleted: ' + functionName, 'success');
                        await loadFunctionList();
                        functionSelect.value = ''; // Clear selection
                        editor.setValue(''); // Clear editor
                    } else {
                        showOutput('Delete failed: ' + (result.data || response.statusText), 'error');
                    }
                } catch (e) {
                    showOutput('Delete error: ' + e.message, 'error');
                } finally {
                    deleteFunctionButton.disabled = false;
                    deleteFunctionButton.textContent = '🗑️ Delete';
                }
            });

            // Enable/disable buttons based on selection
            functionSelect.addEventListener('change', async function() {
                const hasSelection = !!functionSelect.value;
                saveFunctionButton.disabled = !hasSelection;
                deleteFunctionButton.disabled = !hasSelection;

                if (hasSelection) {
                    await loadFunctionSource(functionSelect.value);
                }
            });

            // Initially disable all buttons
            newFunctionButton.disabled = false; // Enable new function button if logged in
            saveFunctionButton.disabled = true;
            saveAsFunctionButton.disabled = true; // Enable as needed
            deleteFunctionButton.disabled = true;
            saveLibraryButton.disabled = false; // Enable as needed
            functionsToolbarInitialized = true;
        }

        function newFunction() {
            if (!authToken) {
                showOutput('Please log in first', 'error');
                return;
            }
            // Clear editor and reset state for new function
            editor.setValue('');
            currentFileName = '';
            originalContent = '';
            isFileModified = false;
            functionEditorContent = '';
            functionEditorFunctionName = '';
            updateSaveButtonStates();
            updateRunButtonState();
        }

        // Load function list from backend and populate dropdown
        async function loadFunctionList() {
            const functionSelect = document.getElementById('functionSelect');
            functionSelect.innerHTML = '<option value="">Select a function...</option>';
            functionSelect.disabled = true;

            try {
                const response = await fetch('/charioteer/api/functions', { headers: getAuthHeaders() });
                if (response.ok) {
                    const result = await response.json();
                    if (result.result === "OK" && Array.isArray(result.data)) {
                        const unique = Array.from(new Set(result.data));
                        unique.forEach(fn => {
                            const option = document.createElement('option');
                            option.value = fn;
                            option.textContent = fn;
                            functionSelect.appendChild(option);
                        });
                        functionSelect.disabled = false;
                    }
                }
            } catch (e) {
                console.error('Failed to load function list:', e);
            }
        }

        // Fetch and display the selected function's source code
        async function loadFunctionSource(functionName) {
            try {
                const response = await fetch('/charioteer/api/function?name=' + encodeURIComponent(functionName), {
                    headers: getAuthHeaders()
                });
                if (response.ok) {
                    const result = await response.json();
                    if (result.result === "OK" && result.data && typeof result.data === "object") {
                        // Assume result.data = { name: "foo", args: ["a", "b"], body: "..." }
                        const fn = result.data;
                        // Format as Chariot function definition
                        let code = "function " + fn.name + "(" + (fn.args || []).join(', ') + ") {\n" + fn.body + "\n}";
                        if (editor) {
                            editor.setValue(code);
                            currentFileName = ''; // Not a file
                            originalContent = code;
                            isFileModified = false;
                            functionEditorContent = code;
                            functionEditorFunctionName = functionName;
                            updateSaveButtonStates();
                            updateRunButtonState();
                        }
                    } else if (result.result === "OK" && typeof result.data === "string") {
                        // If backend just returns the raw function code as a string
                        if (editor) {
                            editor.setValue(result.data);
                            currentFileName = '';
                            originalContent = result.data;
                            isFileModified = false;
                            functionEditorContent = result.data;
                            functionEditorFunctionName = functionName;
                            updateSaveButtonStates();
                            updateRunButtonState();
                        }
                    } else {
                        showOutput('Failed to load function: ' + (result.data || 'Unknown error'), 'error');
                    }
                } else {
                    showOutput('Failed to load function: ' + response.statusText, 'error');
                }
            } catch (e) {
                showOutput('Error loading function: ' + e.message, 'error');
            }
        }

        // Save/update a function
        async function saveFunction() {
            if (!authToken) {
                showOutput('Please log in first', 'error');
                return;
            }
            const code = editor.getValue();
            // Simple regex to extract function name, args, and body
            const match = code.match(/function\s+([a-zA-Z_][\w]*)\s*\(([^)]*)\)\s*\{([\s\S]*)\}$/);
            if (!match) {
                showOutput('Invalid function format. Please use: function name(args) { ... }', 'error');
                return;
            }
            const name = match[1].trim();
            const args = match[2].split(',').map(s => s.trim()).filter(Boolean);
            let body = match[3];
            // Remove the last closing brace if present (since [\s\S]*$ is greedy)
            if (body.endsWith('}')) {
                body = body.slice(0, -1);
            }
            body = body.replace(/^\s*\n/, ''); // Remove leading newline

            const saveFunctionButton = document.getElementById('saveFunctionButton');
            saveFunctionButton.disabled = true;
            saveFunctionButton.textContent = '💾 Saving...';

            try {
                const response = await fetch('/charioteer/api/function/save', {
                    method: 'POST',
                    headers: getAuthHeadersWithJSON(),
                    body: JSON.stringify({
                        name: name,
                        code: code // send the full code string
                    })
                });
                const result = await response.json();
                if (response.ok && result.result === "OK") {
                    showOutput('Function saved: ' + name, 'success');
                    await loadFunctionList();
                    document.getElementById('functionSelect').value = name;
                } else {
                    showOutput('Save failed: ' + (result.data || response.statusText), 'error');
                }
            } catch (e) {
                showOutput('Save error: ' + e.message, 'error');
            } finally {
                saveFunctionButton.disabled = false;
                saveFunctionButton.textContent = '💾 Save';
            }
        }

        // Save as new file
        async function saveAsFunction() {
            if (!authToken) {
                showOutput('Please log in first', 'error');
                return;
            }
            
            const content = editor.getValue();
            if (!content.trim()) {
                showOutput('No content to save', 'error');
                return;
            }
            
            // Prompt for function name
            const functionName = prompt('Enter function name:', 'newFunction');
            if (!functionName) {
                return; // User cancelled
            }

            // Validate function name
            if (!/^[a-zA-Z_][\w]*$/.test(functionName)) {
                showOutput('Invalid function name. Please use a valid identifier.', 'error');
                return;
            }

            const saveAsButton = document.getElementById('saveAsButton');
            saveAsButton.disabled = true;
            saveAsButton.textContent = '💾 Saving...';

            try {
                const response = await fetch('/charioteer/api/function/save', {
                    method: 'POST',
                    headers: getAuthHeadersWithJSON(),
                    body: JSON.stringify({
                        name: functionName,
                        code: content
                    })
                });

                if (response.status === 401) {
                    logout();
                    return;
                }

                if (response.ok) {
                    // Switch to the new function
                    currentFunctionName = functionName;
                    originalContent = content;
                    isFileModified = false;

                    // Refresh function list and select the new function
                    await loadFunctionList();
                    document.getElementById('functionSelect').value = functionName;

                    updateSaveButtonStates();
                    showOutput('Function saved as: ' + functionName, 'success');
                } else {
                    const error = await response.text();
                    showOutput('Save As failed: ' + error, 'error');
                }

            } catch (error) {
                showOutput('Save As error: ' + error.message, 'error');
            } finally {
                saveAsButton.disabled = false;
                saveAsButton.textContent = '💾 Save As...';
            }
        }

        // Save the entire function library
        async function saveLibrary() {
            if (!authToken) {
                showOutput('Please log in first', 'error');
                return;
            }
            const saveLibraryButton = document.getElementById('saveLibraryButton');
            saveLibraryButton.disabled = true;
            saveLibraryButton.textContent = '💾 Saving...';

            try {
                const response = await fetch('/charioteer/api/library/save', {
                    method: 'POST',
                    headers: getAuthHeadersWithJSON()
                });
                const result = await response.json();
                if (response.ok && result.result === "OK") {
                    showOutput('Library saved successfully', 'success');
                } else {
                    showOutput('Save library failed: ' + (result.data || response.statusText), 'error');
                }
            } catch (e) {
                showOutput('Save library error: ' + e.message, 'error');
            } finally {
                saveLibraryButton.disabled = false;
                saveLibraryButton.textContent = '💾 Save Library';
            }
        }
         
        // Login functionality
        async function login() {
            console.log('DEBUG: Login function called');
            const username = document.getElementById('usernameInput').value.trim();
            const password = document.getElementById('passwordInput').value;
            
            console.log('DEBUG: Username:', username, 'Password length:', password.length);
            
            if (!username || !password) {
                showOutput('Please enter both username and password', 'error');
                return;
            }
            
            const loginButton = document.getElementById('loginButton');
            loginButton.disabled = true;
            loginButton.textContent = 'Logging in...';
            
            try {
                const response = await fetch(getAPIPath('/login'), {
                    method: 'POST',
                    headers: {
                        'Content-Type': 'application/json'
                    },
                    body: JSON.stringify({
                        username: username,
                        password: password
                    })
                });
                
                const result = await response.json();
                if (response.ok && result.result === "OK" && result.data && result.data.token) {
                    authToken = result.data.token;
                    currentUser = username;

                    // Save to localStorage
                    localStorage.setItem('chariot_token', authToken);
                    localStorage.setItem('chariot_user', currentUser);

                    // Start session management
                    startSessionManagement();                    
                    
                    updateAuthUI(true);
                    await fetchUserFunctions().then(functionNames => {
                        setChariotTokenizer(functionNames);
                    });                    
                    loadFileList();
                    updateLeftPanel();
                    
                    // Clear password field
                    document.getElementById('passwordInput').value = '';
                } else {
                    const errorMsg = result.result === "ERROR" ? result.data : 'Invalid credentials';
                    showOutput('Login failed: ' + errorMsg, 'error');
                }
            } catch (error) {
                showOutput('Login error: ' + error.message, 'error');
            } finally {
                loginButton.disabled = false;
                loginButton.textContent = 'Login';
            }
        }
        
        // Logout functionality
        function logout() {
            console.log('DEBUG: Logout function called');
            
            // Clear session timers
            clearSessionTimers();
            
            // Close any open warning dialog
            closeSessionWarning();
            
            authToken = null;
            currentUser = null;
            
            // Clear localStorage
            localStorage.removeItem('chariot_token');
            localStorage.removeItem('chariot_user');
            
            // Clear editor and file list
            if (editor) {
                editor.setValue('');
            }
            currentFileName = '';
            
            const fileSelect = document.getElementById('fileSelect');
            fileSelect.innerHTML = '<option value="">Select a file...</option>';
            
            updateAuthUI(false);
        }

        // Session management functions
        function startSessionManagement() {
            console.log('DEBUG: Starting session management');
            
            // Clear any existing timers
            clearSessionTimers();
            
            const sessionDurationMs = SESSION_DURATION_MINUTES * 60 * 1000;
            const warningTimeMs = sessionDurationMs - (WARNING_BEFORE_MINUTES * 60 * 1000);
            const logoutTimeMs = sessionDurationMs - (LOGOUT_BEFORE_SECONDS * 1000);
            
            console.log('DEBUG: Session timers set - Warning in:', warningTimeMs, 'ms, Logout in:', logoutTimeMs, 'ms');
            
            // Set warning timer
            warningTimer = setTimeout(() => {
                showSessionWarning();
            }, warningTimeMs);
            
            // Set logout timer
            logoutTimer = setTimeout(() => {
                showOutput('Session expired. You have been logged out.', 'error');
                logout();
            }, logoutTimeMs);
        }

        function clearSessionTimers() {
            if (warningTimer) {
                clearTimeout(warningTimer);
                warningTimer = null;
            }
            if (logoutTimer) {
                clearTimeout(logoutTimer);
                logoutTimer = null;
            }
            if (sessionTimer) {
                clearTimeout(sessionTimer);
                sessionTimer = null;
            }
        }

        function showSessionWarning() {
            if (sessionWarningShown) return;
            
            console.log('DEBUG: Showing session warning');
            sessionWarningShown = true;
            
            // Create modal HTML
            const modal = document.createElement('div');
            modal.className = 'session-warning-modal';
            modal.innerHTML = '' +
                '<div class="session-warning-dialog">' +
                    '<div class="session-warning-title">' +
                        '⚠️ Session Expiring Soon' +
                    '</div>' +
                    '<div class="session-warning-message">' +
                        'Your session will expire in approximately <strong>3 minutes</strong>. ' +
                        'If you don\'t extend your session, you will be automatically logged out in 30 seconds.' +
                    '</div>' +
                    '<div class="session-warning-countdown" id="sessionCountdown">' +
                        'Time remaining: <span id="countdownTime">03:00</span>' +
                    '</div>' +
                    '<div class="session-warning-buttons">' +
                        '<button class="session-warning-button extend" id="extendSessionButton">' +
                            'Extend Session' +
                        '</button>' +
                        '<button class="session-warning-button logout" id="logoutNowButton">' +
                            'Logout Now' +
                        '</button>' +
                    '</div>' +
                '</div>';
            
            document.body.appendChild(modal);
            
            // Add event listeners
            document.getElementById('extendSessionButton').addEventListener('click', () => {
                extendSession();
                closeSessionWarning();
            });
            
            document.getElementById('logoutNowButton').addEventListener('click', () => {
                closeSessionWarning();
                logout();
            });
            
            // Start countdown
            startWarningCountdown();
        }

        function startWarningCountdown() {
            const countdownElement = document.getElementById('countdownTime');
            if (!countdownElement) return;
            
            let remainingSeconds = WARNING_BEFORE_MINUTES * 60; // 3 minutes in seconds
            
            const countdownInterval = setInterval(() => {
                remainingSeconds--;
                
                if (remainingSeconds <= 0) {
                    clearInterval(countdownInterval);
                    return;
                }
                
                const minutes = Math.floor(remainingSeconds / 60);
                const seconds = remainingSeconds % 60;
                const timeString = minutes.toString().padStart(2, '0') + ':' + seconds.toString().padStart(2, '0');
                
                countdownElement.textContent = timeString;
                
                // Change color when less than 1 minute remaining
                if (remainingSeconds <= 60) {
                    countdownElement.style.color = '#dc3545';
                }
            }, 1000);
        }

        function extendSession() {
            console.log('DEBUG: Extending session');
            
            // Clear existing timers
            clearSessionTimers();
            
            // Restart session management
            startSessionManagement();
            
            // Reset warning flag
            sessionWarningShown = false;
            
            showOutput('Session extended successfully', 'success');
        }

        function extendSessionSilently() {
            console.log('DEBUG: Silently extending session');
            
            // Clear existing timers
            clearSessionTimers();
            
            // Restart session management
            startSessionManagement();
            
            // Reset warning flag
            sessionWarningShown = false;
        }

        function closeSessionWarning() {
            const modal = document.querySelector('.session-warning-modal');
            if (modal) {
                document.body.removeChild(modal);
            }
            sessionWarningShown = false;
        }

        // Add this function and call it after the editor is initialized
        // Ensure we only bind global UI handlers once across editor rebuilds
        let uiHandlersInitialized = false;
        function initializeEventHandlers() {
            if (uiHandlersInitialized) { return; }
            // Mark immediately to avoid duplicate bindings during early calls
            uiHandlersInitialized = true;
            console.log('DEBUG: Initializing event handlers');

            // Activity tracking for session extension
            const activityEvents = ['click', 'keydown', 'mousemove', 'scroll'];
            let lastActivityTime = Date.now();

            activityEvents.forEach(eventType => {
                document.addEventListener(eventType, () => {
                    const now = Date.now();
                    // Only extend session if there's been activity and user is logged in
                    if (authToken && now - lastActivityTime > 60000) { // Only extend every minute
                        lastActivityTime = now;
                        if (!sessionWarningShown) {
                            // Silently extend session on activity (only if warning not shown)
                            extendSessionSilently();
                        }
                    }
                });
            });            
            
            // Login functionality (also bound in bindAuthHandlers as a fallback)
            const loginButton = document.getElementById('loginButton');
            const logoutButton = document.getElementById('logoutButton');
            const usernameInput = document.getElementById('usernameInput');
            const passwordInput = document.getElementById('passwordInput');
            
            if (loginButton && !authHandlersInitialized) {
                loginButton.addEventListener('click', login);
                console.log('DEBUG: Login button handler added');
            }
            
            if (logoutButton && !authHandlersInitialized) {
                logoutButton.addEventListener('click', logout);
                console.log('DEBUG: Logout button handler added');
            }
            
            // Enter key in password field
            if (passwordInput && !authHandlersInitialized) {
                passwordInput.addEventListener('keypress', function(e) {
                    if (e.key === 'Enter') {
                        login();
                    }
                });
            }
            // Mark initialized to avoid double-binding later
            authHandlersInitialized = true;
            
            // File selection
            const fileSelect = document.getElementById('fileSelect');
            if (fileSelect) {
                fileSelect.addEventListener('change', function() {
                    const fileName = this.value;
                    if (fileName) {
                        loadFile(fileName);
                    }
                });
                console.log('DEBUG: File select handler added');
            }
            
            // Save buttons
            const newButton = document.getElementById('newButton');
            const saveButton = document.getElementById('saveButton');
            const saveAsButton = document.getElementById('saveAsButton');
            const runButton = document.getElementById('runButton');
            
            if (newButton) {
                newButton.addEventListener('click', newFile);
                console.log('DEBUG: New button handler added');
            }
            
            if (saveButton) {
                saveButton.addEventListener('click', saveFile);
                console.log('DEBUG: Save button handler added');
            }
            
            if (saveAsButton) {
                saveAsButton.addEventListener('click', saveAsFile);
                console.log('DEBUG: Save As button handler added');
            }
            
            // File action buttons
            const renameButton = document.getElementById('renameButton');
            const deleteButton = document.getElementById('deleteButton');
            
            if (renameButton) {
                renameButton.addEventListener('click', renameFile);
                console.log('DEBUG: Rename button handler added');
            }
            
            if (deleteButton) {
                deleteButton.addEventListener('click', deleteFile);
                console.log('DEBUG: Delete button handler added');
            }
            
            if (runButton) {
                runButton.addEventListener('click', function() {
                    const streamingToggle = document.getElementById('streamingToggle');
                    if (streamingToggle && streamingToggle.checked) {
                        runCodeAsync();
                    } else {
                        runCode();
                    }
                });
                console.log('DEBUG: Run button handler added');
            }
            
            // Tab functionality
            const tabs = document.querySelectorAll('.tab');
            tabs.forEach(tab => {
                tab.addEventListener('click', function() {
                    switchTab(this.dataset.tab);
                });
            });
            console.log('DEBUG: Tab handlers added');
            
            // Keyboard shortcuts (only register once)
            if (!window.chariotKeyboardHandlerRegistered) {
                document.addEventListener('keydown', function(e) {
                    // Skip if event is from Monaco editor
                    if (e.target.closest('.monaco-editor')) {
                        // Only handle Ctrl/Cmd+S, ignore arrow keys
                        if ((e.ctrlKey || e.metaKey) && e.key === 's') {
                            e.preventDefault();
                            if (e.shiftKey) {
                                saveAsFile();
                            } else {
                                saveFile();
                            }
                        }
                        return; // Don't interfere with Monaco's key handling
                    }
                    
                    // Original keyboard shortcut logic for non-Monaco elements
                    if (e.ctrlKey || e.metaKey) {
                        if (e.key === 's') {
                            e.preventDefault();
                            if (e.shiftKey) {
                                saveAsFile();
                            } else {
                                saveFile();
                            }
                        }
                    }
                });
                window.chariotKeyboardHandlerRegistered = true;
                console.log('DEBUG: Keyboard shortcuts added');
            }
            // Toolbar tab switching
            const filesTab = document.getElementById('filesTab');
            const functionsTab = document.getElementById('functionsTab');
            const dashboardTab = document.getElementById('dashboardTab');
            const agentsTab = document.getElementById('agentsTab');
            const diagramsTab = document.getElementById('diagramsTab');
            const fileToolbar = document.getElementById('fileToolbar');
            const functionsToolbar = document.getElementById('functionsToolbar');
            const dashboardToolbar = document.getElementById('dashboardToolbar');
            const agentsToolbar = document.getElementById('agentsToolbar');
            const diagramsToolbar = document.getElementById('diagramsToolbar');

            if (filesTab && functionsTab && dashboardTab && agentsTab && diagramsTab && fileToolbar && functionsToolbar && dashboardToolbar && agentsToolbar && diagramsToolbar) {
                // Helpers to manage dashboard auto-refresh lifecycle
                function stopDashboardAutoRefresh() {
                    if (dashboardAutoRefresh) {
                        clearInterval(dashboardAutoRefresh);
                        dashboardAutoRefresh = null;
                    }
                    // Also close WS if open
                    try { if (dashboardWS) { dashboardWS.close(); dashboardWS = null; } } catch (e) {}
                }

                function startDashboardAutoRefresh() {
                    // Ensure only one timer is running
                    stopDashboardAutoRefresh();
                    // Poll every 30 seconds to reflect session changes/timeouts
                    dashboardAutoRefresh = setInterval(() => {
                        if (currentTab === 'dashboard' && typeof fetchAndUpdateDashboard === 'function') {
                            try { fetchAndUpdateDashboard(); } catch (e) { /* ignore */ }
                        } else {
                            stopDashboardAutoRefresh();
                        }
                    }, 30000);
                }

                function showToolbar(selected) {
                    // Hide all toolbars
                    fileToolbar.classList.remove('active');
                    functionsToolbar.classList.remove('active');
                    dashboardToolbar.classList.remove('active');
                    agentsToolbar.classList.remove('active');
                    diagramsToolbar.classList.remove('active');
                    // Remove active from all tabs
                    filesTab.classList.remove('active');
                    functionsTab.classList.remove('active');
                    dashboardTab.classList.remove('active');
                    agentsTab.classList.remove('active');
                    diagramsTab.classList.remove('active');

                    if (selected === 'files') {
                        // Leaving dashboard: stop auto refresh
                        stopDashboardAutoRefresh();
                        // Leaving agents: stop WS and restore editor container
                        if (currentTab === 'agents') {
                            stopAgentsWS();
                            const editorContainer = document.getElementById('editorContainer');
                            if (editorContainer) {
                                editorContainer.innerHTML = '';
                                setupMonacoEditor();
                            }
                        }
                        // Save current function editor state
                        if (currentTab === 'functions') {
                            functionEditorContent = editor.getValue();
                            functionEditorFunctionName = document.getElementById('functionSelect').value;
                        }
                        // Save current dashboard state
                        if (currentTab === 'dashboard') {
                            const editorContainer = document.getElementById('editorContainer');
                            if (editorContainer) {
                                dashboardContent = editorContainer.innerHTML;
                            }
                        }
                        // Restore Monaco editor if coming from dashboard or agents
                        if (currentTab === 'dashboard' || currentTab === 'agents') {
                            // Clear dashboard content and restore editor container
                            const editorContainer = document.getElementById('editorContainer');
                            editorContainer.innerHTML = '';
                            setupMonacoEditor();
                        }
                        // Restore file editor state
                        fileToolbar.classList.add('active');
                        filesTab.classList.add('active');
                        if (fileEditorContent !== '') {
                            editor.setValue(fileEditorContent);
                            currentFileName = fileEditorFileName;
                            originalContent = fileEditorContent;
                            isFileModified = false;
                        } else {
                            editor.setValue('');
                            currentFileName = '';
                            originalContent = '';
                            isFileModified = false;
                        }
                        updateSaveButtonStates();
                        updateRunButtonState();
                        currentTab = 'files';
                    } else if (selected === 'functions') {
                        // Leaving dashboard: stop auto refresh
                        stopDashboardAutoRefresh();
                        // Leaving agents: stop WS and restore editor
                        if (currentTab === 'agents') {
                            stopAgentsWS();
                            const editorContainer = document.getElementById('editorContainer');
                            if (editorContainer) {
                                editorContainer.innerHTML = '';
                                setupMonacoEditor();
                            }
                        }
                        // Save current file editor state
                        if (currentTab === 'files') {
                            fileEditorContent = editor.getValue();
                            fileEditorFileName = currentFileName;
                        }
                        // Save current dashboard state
                        if (currentTab === 'dashboard') {
                            const editorContainer = document.getElementById('editorContainer');
                            if (editorContainer) {
                                dashboardContent = editorContainer.innerHTML;
                            }
                        }
                        // Restore Monaco editor if coming from dashboard or agents
                        if (currentTab === 'dashboard' || currentTab === 'agents') {
                            // Clear dashboard content and restore editor container
                            const editorContainer = document.getElementById('editorContainer');
                            editorContainer.innerHTML = '';
                            setupMonacoEditor();
                        }
                        // Restore function editor state
                        functionsToolbar.classList.add('active');
                        functionsTab.classList.add('active');
                        const functionSelect = document.getElementById('functionSelect');
                        if (functionEditorFunctionName && functionSelect) {
                            functionSelect.value = functionEditorFunctionName;
                        }
                        if (functionEditorContent !== '') {
                            editor.setValue(functionEditorContent);
                            originalContent = functionEditorContent;
                            isFileModified = false;
                            // Optionally, set functionSelect.value = functionEditorFunctionName;
                        } else {
                            editor.setValue('');
                            originalContent = '';
                            isFileModified = false;
                        }
                        updateSaveButtonStates();
                        updateRunButtonState();
                        currentTab = 'functions';
                    } else if (selected === 'dashboard') {
                        // Save current editor state
                        if (currentTab === 'files') {
                            fileEditorContent = editor.getValue();
                            fileEditorFileName = currentFileName;
                        } else if (currentTab === 'functions') {
                            functionEditorContent = editor.getValue();
                            functionEditorFunctionName = document.getElementById('functionSelect').value;
                        } else if (currentTab === 'agents') {
                            // Ensure we fully stop the Agents WS loop when switching to dashboard
                            stopAgentsWS();
                        }
                        // Show dashboard toolbar
                        dashboardToolbar.classList.add('active');
                        dashboardTab.classList.add('active');
                        
                        // Load or restore dashboard content
                        if (dashboardLoaded && dashboardContent) {
                            // Restore existing dashboard content
                            const editorContainer = document.getElementById('editorContainer');
                            if (editor) {
                                editor.getModel()?.dispose();
                                editor.dispose();
                                editor = null;
                            }
                            editorContainer.innerHTML = dashboardContent;
                            // Prefer WebSocket stream; it will fallback to polling on error
                            connectDashboardWS();
                            // Rebind listeners panel handlers after restoring DOM
                            bindListenersPanelHandlers();
                        } else {
                            // Load dashboard content for first time
                            loadDashboardContent();
                            // Prefer WebSocket stream; it will fallback to polling on error
                            connectDashboardWS();
                        }
                        
                        // Dashboard will replace the editor content entirely
                        originalContent = '';
                        isFileModified = false;
                        updateSaveButtonStates();
                        updateRunButtonState();
                        currentTab = 'dashboard';
                    } else if (selected === 'agents') {
                        // Ensure any dashboard auto-refresh/WS is stopped when entering Agents
                        stopDashboardAutoRefresh();
                        // Save current editor state when switching to Agents
                        if (currentTab === 'files') {
                            fileEditorContent = editor.getValue();
                            fileEditorFileName = currentFileName;
                        } else if (currentTab === 'functions') {
                            functionEditorContent = editor.getValue();
                            const fnSel = document.getElementById('functionSelect');
                            functionEditorFunctionName = fnSel ? fnSel.value : '';
                        }
                        // If coming from dashboard, ensure Monaco editor is torn down (agents replaces editor area)
                        if (currentTab === 'dashboard') {
                            const editorContainer = document.getElementById('editorContainer');
                            if (editor) {
                                editor.getModel()?.dispose();
                                editor.dispose();
                                editor = null;
                            }
                            editorContainer.innerHTML = '';
                        }
                        // Activate Agents toolbar/tab
                        agentsToolbar.classList.add('active');
                        agentsTab.classList.add('active');
                        // Load or restore Agents content
                        const editorElement = document.getElementById('editorContainer');
                        // reset reconnect state before enabling
                        if (agentsWSReconnectTimer) { try { clearTimeout(agentsWSReconnectTimer); } catch (e) {} agentsWSReconnectTimer = null; }
                        agentsWSBackoffMs = 1000;
                        agentsWSConnecting = false;
                        agentsWSReconnectEnabled = true; // allow WS to reconnect while Agents is active
                        if (agentsLoaded && agentsContent) {
                            if (editor) {
                                editor.getModel()?.dispose();
                                editor.dispose();
                                editor = null;
                            }
                            editorElement.innerHTML = agentsContent;
                            // Re-bind UI handlers for restored DOM
                            const clearBtn = document.getElementById('clearAgentsStreamButton');
                            if (clearBtn) {
                                clearBtn.addEventListener('click', function() {
                                    const s = document.getElementById('agentsStream');
                                    if (s) s.textContent = '';
                                });
                            }
                            const hbToggle = document.getElementById('toggleHeartbeats');
                            if (hbToggle) {
                                hbToggle.checked = agentsShowHeartbeats;
                                hbToggle.addEventListener('change', function() {
                                    agentsShowHeartbeats = hbToggle.checked;
                                });
                            }
                            try { fetchAndRenderAgents(); } catch (e) {}
                            connectAgentsWS();
                        } else {
                            // Ensure content is loaded before connecting WS to avoid races with DOM elements
                            try {
                                loadAgentsContent().then(() => {
                                    connectAgentsWS();
                                });
                            } catch (e) {
                                // Fallback: attempt WS connect regardless
                                connectAgentsWS();
                            }
                        }
                        originalContent = '';
                        isFileModified = false;
                        updateSaveButtonStates();
                        updateRunButtonState();
                        currentTab = 'agents';
                    } else if (selected === 'diagrams') {
                        // Leaving dashboard: stop auto refresh
                        stopDashboardAutoRefresh();
                        // Leaving agents: stop WS and restore editor
                        if (currentTab === 'agents') {
                            stopAgentsWS();
                            const editorContainer = document.getElementById('editorContainer');
                            if (editor) {
                                editor.getModel()?.dispose();
                                editor.dispose();
                                editor = null;
                            }
                            // also clear any pending reconnects
                            if (agentsWSReconnectTimer) { try { clearTimeout(agentsWSReconnectTimer); } catch (e) {} agentsWSReconnectTimer = null; }
                            if (editorContainer) {
                                editorContainer.innerHTML = '';
                                setupMonacoEditor();
                            }
                        }
                        // Save current editor states when switching away
                        if (currentTab === 'files') {
                            fileEditorContent = editor.getValue();
                            fileEditorFileName = currentFileName;
                        } else if (currentTab === 'functions') {
                            functionEditorContent = editor.getValue();
                            const fnSel = document.getElementById('functionSelect');
                            functionEditorFunctionName = fnSel ? fnSel.value : '';
                        }
                        // Restore Monaco if coming from dashboard
                        if (currentTab === 'dashboard') {
                            const editorContainer = document.getElementById('editorContainer');
                            if (editor) {
                                editor.getModel()?.dispose();
                                editor.dispose();
                                editor = null;
                            }
                            editorContainer.innerHTML = '';
                            setupMonacoEditor();
                        }
                        // Show diagrams toolbar and set active tab
                        diagramsToolbar.classList.add('active');
                        diagramsTab.classList.add('active');
                        // Ensure buttons reflect current editor state
                        updateSaveButtonStates();
                        updateRunButtonState();
                        currentTab = 'diagrams';
                        // Load diagrams list on first enter or refresh as needed
                        try { loadDiagramsList(); } catch (e) { /* ignore */ }
                    }
                }
                filesTab.addEventListener('click', function() {
                    showToolbar('files');
                });
                functionsTab.addEventListener('click', function() {
                    showToolbar('functions');
                });
                dashboardTab.addEventListener('click', function() {
                    showToolbar('dashboard');
                });
                agentsTab.addEventListener('click', function() {
                    showToolbar('agents');
                });
                diagramsTab.addEventListener('click', function() {
                    showToolbar('diagrams');
                });
            }
            // Diagrams toolbar handlers (initialize once)
            if (!diagramsToolbarInitialized) {
                const refreshDiagramsButton = document.getElementById('refreshDiagramsButton');
                const saveDiagramButton = document.getElementById('saveDiagramButton');
                const saveAsDiagramButton = document.getElementById('saveAsDiagramButton');
                const deleteDiagramButton = document.getElementById('deleteDiagramButton');
                const diagramSelect = document.getElementById('diagramSelect');

                if (diagramSelect) {
                    diagramSelect.addEventListener('change', async function() {
                        const hasSelection = !!diagramSelect.value;
                        currentDiagramName = hasSelection ? diagramSelect.value : '';
                        toggleDiagramActionButtons(hasSelection);
                        if (hasSelection) {
                            await generateFromSelectedDiagram();
                        } else if (editor) {
                            editor.setValue('');
                        }
                    });
                }

                if (refreshDiagramsButton) {
                    refreshDiagramsButton.addEventListener('click', async function() {
                        await loadDiagramsList();
                        // If a diagram is selected, re-generate from source to discard unsaved edits
                        const select = document.getElementById('diagramSelect');
                        if (select && select.value) {
                            await generateFromSelectedDiagram();
                        }
                    });
                }

                if (saveDiagramButton) {
                    saveDiagramButton.addEventListener('click', async function() {
                        await saveCurrentDiagram(false);
                    });
                }
                if (saveAsDiagramButton) {
                    saveAsDiagramButton.addEventListener('click', async function() {
                        await saveCurrentDiagram(true);
                    });
                }
                if (deleteDiagramButton) {
                    deleteDiagramButton.addEventListener('click', async function() {
                        await deleteCurrentDiagram();
                    });
                }

                diagramsToolbarInitialized = true;
            }

            // Agents toolbar handlers (initialize once)
            if (!window.agentsToolbarInitialized) {
                const refreshAgentsButton = document.getElementById('refreshAgentsButton');
                if (refreshAgentsButton) {
                    refreshAgentsButton.addEventListener('click', async function() {
                        await fetchAndRenderAgents();
                    });
                }
                window.agentsToolbarInitialized = true;
            }
        }        

        // Load diagrams list from backend and populate dropdown
        async function loadDiagramsList() {
            const select = document.getElementById('diagramSelect');
            if (!select) return;
            select.innerHTML = '<option value="">Select a diagram...</option>';
            select.disabled = true;
            toggleDiagramActionButtons(false);
            try {
                const resp = await fetch(getAPIPath('/api/diagrams'), { headers: getAuthHeaders() });
                if (!resp.ok) {
                    if (resp.status === 401) { logout(); return; }
                    const t = await resp.text();
                    showOutput('Failed to load diagrams: ' + t, 'error');
                    return;
                }
                const result = await resp.json();
                const list = Array.isArray(result) ? result : (result && result.result === 'OK' ? result.data : []);
                if (Array.isArray(list)) {
                    list.forEach(item => {
                        const name = typeof item === 'string' ? item : item.name;
                        if (!name) return;
                        const opt = document.createElement('option');
                        opt.value = name;
                        opt.textContent = name;
                        select.appendChild(opt);
                    });
                    select.disabled = false;
                } else {
                    showOutput('No diagrams available', 'info');
                }
            } catch (e) {
                showOutput('Error loading diagrams: ' + e.message, 'error');
            }
            // already marked at entry
        }

        // Fetch selected diagram JSON and generate Chariot code using shared codegen
        async function generateFromSelectedDiagram() {
            const select = document.getElementById('diagramSelect');
            if (!select || !select.value) {
                showOutput('Please select a diagram first', 'error');
                return;
            }
            if (!window.ChariotCodegen || typeof window.ChariotCodegen.generateChariotCodeFromDiagram !== 'function') {
                showOutput('Code generator not loaded. Ensure /chariot-codegen.js is available and built.', 'error');
                return;
            }
            const name = select.value;
            try {
                const resp = await fetch(getAPIPath('/api/diagrams/' + encodeURIComponent(name)), { headers: getAuthHeaders() });
                if (!resp.ok) {
                    if (resp.status === 401) { logout(); return; }
                    const t = await resp.text();
                    showOutput('Failed to load diagram: ' + t, 'error');
                    return;
                }
                // Backend returns raw diagram JSON (not wrapped)
                const diagram = await resp.json();
                currentDiagramJSON = diagram;
                let code = '';
                if (diagram && typeof diagram.code === 'string' && diagram.code.trim().length > 0) {
                    // Prefer user-authored/saved code when present
                    code = diagram.code;
                } else {
                    code = window.ChariotCodegen.generateChariotCodeFromDiagram(JSON.stringify(diagram));
                }
                code = stripEmbeddedDiagramMarker(code);
                if (editor) {
                    editor.setValue(code);
                    showOutput('Generated code from diagram: ' + name, 'success');
                    updateRunButtonState();
                }
            } catch (e) {
                showOutput('Error generating code: ' + e.message, 'error');
            }
        }

        // Remove embedded diagram payload marker from displayed code
        function stripEmbeddedDiagramMarker(code) {
            try {
                const lines = code.split(/\r?\n/);
                const filtered = lines.filter(l => l.indexOf('__VDSL_SOURCE__: base64:') === -1);
                return filtered.join('\n').trimEnd();
            } catch (_) { return code; }
        }

        function toggleDiagramActionButtons(enabled) {
            const saveBtn = document.getElementById('saveDiagramButton');
            const saveAsBtn = document.getElementById('saveAsDiagramButton');
            const delBtn = document.getElementById('deleteDiagramButton');
            if (saveBtn) saveBtn.disabled = !enabled;
            if (saveAsBtn) saveAsBtn.disabled = !enabled;
            if (delBtn) delBtn.disabled = !enabled;
        }

        async function saveCurrentDiagram(asNew) {
            if (!authToken) { showOutput('Please log in first', 'error'); return; }
            const select = document.getElementById('diagramSelect');
            if (!select) return;
            let name = currentDiagramName || select.value;
            if (asNew || !name) {
                const input = prompt('Enter diagram name:', name || 'new-diagram');
                if (!input) return;
                name = input.trim();
            }
            // Use last loaded diagram JSON as the source of truth. The embedded payload is hidden from display
            // and not required at save time until a true reverse code->diagram exists.
            const contentJSON = (function() {
                try {
                    // Clone to avoid mutating in-memory object
                    const clone = JSON.parse(JSON.stringify(currentDiagramJSON || {}));
                    if (editor) {
                        clone.code = editor.getValue();
                    }
                    return clone;
                } catch (_) {
                    return currentDiagramJSON;
                }
            })();
            if (!contentJSON) {
                showOutput('No diagram content available to save.', 'error');
                return;
            }
            try {
                const resp = await fetch(getAPIPath('/api/diagrams'), {
                    method: 'POST',
                    headers: getAuthHeadersWithJSON(),
                    body: JSON.stringify({ name, content: contentJSON })
                });
                if (resp.status === 401) { logout(); return; }
                if (resp.ok) {
                    showOutput('Diagram saved: ' + name, 'success');
                    await loadDiagramsList();
                    const sel = document.getElementById('diagramSelect');
                    if (sel) sel.value = name;
                    currentDiagramName = name;
                } else {
                    const t = await resp.text();
                    showOutput('Save failed: ' + t, 'error');
                }
            } catch (e) {
                showOutput('Save error: ' + e.message, 'error');
            }
        }

        async function deleteCurrentDiagram() {
            if (!authToken) { showOutput('Please log in first', 'error'); return; }
            const select = document.getElementById('diagramSelect');
            if (!select || !select.value) { showOutput('No diagram selected', 'error'); return; }
            const name = select.value;
            if (!confirm('Delete diagram "' + name + '"? This cannot be undone.')) return;
            try {
                const resp = await fetch(getAPIPath('/api/diagrams/' + encodeURIComponent(name)), {
                    method: 'DELETE',
                    headers: getAuthHeaders(),
                });
                if (resp.status === 401) { logout(); return; }
                if (resp.ok) {
                    showOutput('Diagram deleted: ' + name, 'success');
                    await loadDiagramsList();
                    const sel = document.getElementById('diagramSelect');
                    if (sel) sel.value = '';
                    currentDiagramName = '';
                    currentDiagramJSON = null;
                    if (editor) editor.setValue('');
                    toggleDiagramActionButtons(false);
                } else {
                    const t = await resp.text();
                    showOutput('Delete failed: ' + t, 'error');
                }
            } catch (e) {
                showOutput('Delete error: ' + e.message, 'error');
            }
        }
        // Tab switching
        function switchTab(tabName) {
            document.querySelectorAll('.tab').forEach(tab => {
                tab.classList.remove('active');
            });
            document.querySelector('[data-tab="' + tabName + '"]').classList.add('active');
            // Toggle visible content for bottom panel
            const out = document.getElementById('outputContent');
            const probs = document.getElementById('problemsContent');
            if (out && probs) {
                if (tabName === 'problems') {
                    out.style.display = 'none';
                    probs.style.display = 'block';
                } else {
                    out.style.display = 'block';
                    probs.style.display = 'none';
                    tabName = 'output';
                }
            }
            currentBottomTab = tabName;
            updateTabContent();
        }
        
        // Update tab content based on current tab
        function updateTabContent() {
            const content = document.getElementById('tabContent');
            // Content will be updated by specific functions (showOutput, showProblem, etc.)
        }

        // Recursively render a JS object as a tree
        async function updateLeftPanel() {
            const panel = document.getElementById('leftPanel');
            if (!panel) return;
            try {
                const response = await fetch('/charioteer/api/runtime/inspect', { headers: getAuthHeaders() });
                if (response.ok) {
                    const result = await response.json();
                    if (result.result === "OK" && result.data) {
                        let runtimeData = result.data;
                        panel.innerHTML = '<div class="tree-view">' + renderTree(runtimeData, null, []) + '</div>';
                        addTreeToggleHandlers(panel);
                    } else {
                        panel.innerHTML = '<div style="color:#f44747; padding:10px;">No runtime data available.</div>';
                    }
                } else {
                    panel.innerHTML = '<div style="color:#f44747; padding:10px;">Failed to load runtime info.</div>';
                }
            } catch (e) {
                panel.innerHTML = '<div style="color:#f44747; padding:10px;">Error loading runtime info.</div>';
            }
        }

        // Run code functionality
        async function runCode() {
            console.log('DEBUG: runCode called');
            if (!authToken) {
                // Check local storage for token
                const savedToken = localStorage.getItem('chariot_token');
                const savedUser = localStorage.getItem('chariot_user');

                if (savedToken && savedUser) {
                    authToken = savedToken;
                    currentUser = savedUser;
                    updateAuthUI(true);
                } else {
                    showOutput('Please log in first', 'error');
                    return;
                }
            }
            
            const code = editor.getValue();
            if (!code.trim()) {
                showOutput('No code to run', 'info');
                return;
            }
            
            const runButton = document.getElementById('runButton');
            runButton.disabled = true;
            runButton.textContent = '⏳ Running...';
            
            try {
                showOutput('Executing code...', 'loading');
                switchTab('output');

                headers = getAuthHeadersWithJSON();  // Use the version with Content-Type
                console.log('DEBUG: runCode headers', headers);
                
                const response = await fetch(getAPIPath('/api/execute'), {
                    method: 'POST',
                    headers: headers,
                    body: JSON.stringify({ program: code })
                });
                
                if (response.status === 401) {
                    showOutput('Authentication expired. Please log in again.', 'error');
                    logout();
                    return;
                }
                
                const result = await response.json();
                
                if (response.ok && result.result === "OK") {
                    showOutput('Result: ' + JSON.stringify(result.data, null, 2), 'success');
                    updateLeftPanel();
                } else {
                    const errorMsg = result.result === "ERROR" ? result.data : 'Execution failed';
                    showOutput('Error: ' + errorMsg, 'error');
                }
                
            } catch (error) {
                showOutput('Network Error: ' + error.message, 'error');
            } finally {
                runButton.disabled = false;
                runButton.textContent = '▶ Run';
            }
        }

        // Run code with streaming logs via SSE
        async function runCodeAsync() {
            if (!authToken) {
                // Try to get the token from the cookie
                const token = getCookie('chariot_token');
                if (token) {
                    authToken = token;
                    updateAuthUI(true);
                } else {
                    showOutput('Please log in first', 'error');
                    return;
                }
            }
            
            const code = editor.getValue();
            if (!code.trim()) {
                showOutput('No code to run', 'info');
                return;
            }
            
            const runButton = document.getElementById('runButton');
            runButton.disabled = true;
            runButton.textContent = '⏳ Running...';
            
            try {
                showOutput('Starting execution...', 'loading');
                switchTab('output');

                // Step 1: Start async execution
                const execResponse = await fetch(getAPIPath('/api/execute-async'), {
                    method: 'POST',
                    headers: getAuthHeadersWithJSON(),
                    body: JSON.stringify({ program: code })
                });
                
                if (execResponse.status === 401) {
                    showOutput('Authentication expired. Please log in again.', 'error');
                    logout();
                    return;
                }
                
                const execResult = await execResponse.json();
                
                if (!execResponse.ok || execResult.result !== "OK") {
                    const errorMsg = execResult.result === "ERROR" ? execResult.data : 'Failed to start execution';
                    showOutput('Error: ' + errorMsg, 'error');
                    return;
                }
                
                const executionId = execResult.data.execution_id;
                showOutput('Execution ID: ' + executionId + '\n--- Streaming Logs ---\n', 'info');
                
                // Step 2: Stream logs via SSE
                await streamExecutionLogs(executionId);
                
                // Step 3: Get final result
                await getExecutionResult(executionId);
                
                // Update left panel to reflect any changes
                updateLeftPanel();
                
            } catch (error) {
                showOutput('\nNetwork Error: ' + error.message, 'error');
            } finally {
                runButton.disabled = false;
                runButton.textContent = '▶ Run';
            }
        }

        // Stream execution logs via Server-Sent Events
        function streamExecutionLogs(executionId) {
            return new Promise((resolve, reject) => {
                // EventSource doesn't support custom headers, so pass token as query param
                const url = getAPIPath('/api/logs/' + executionId) + '?token=' + encodeURIComponent(authToken);
                const eventSource = new EventSource(url);
                
                eventSource.onmessage = (event) => {
                    try {
                        const logEntry = JSON.parse(event.data);
                        const timestamp = new Date(logEntry.timestamp).toLocaleTimeString();
                        const levelColor = {
                            'DEBUG': '#569cd6',
                            'INFO': '#4ec9b0',
                            'WARN': '#dcdcaa',
                            'ERROR': '#f44747'
                        }[logEntry.level] || '#cccccc';
                        
                        const logLine = '<span style="color:' + levelColor + '">[' + logEntry.level + ']</span> ' + 
                                       '<span style="color:#666">' + timestamp + '</span> ' + 
                                       logEntry.message;
                        appendToOutput(logLine);
                    } catch (err) {
                        console.error('Failed to parse log entry:', err);
                    }
                };
                
                eventSource.addEventListener('done', () => {
                    eventSource.close();
                    appendToOutput('\n--- Execution Complete ---\n', 'info');
                    resolve();
                });
                
                eventSource.onerror = (error) => {
                    eventSource.close();
                    console.error('SSE error:', error);
                    // Don't reject, just resolve to continue to result fetching
                    resolve();
                };
            });
        }

        // Get final execution result
        async function getExecutionResult(executionId) {
            try {
                const response = await fetch(getAPIPath('/api/result/' + executionId), {
                    headers: getAuthHeaders()
                });
                
                if (!response.ok) {
                    appendToOutput('\nFailed to fetch result (status: ' + response.status + ')', 'error');
                    return;
                }
                
                const result = await response.json();
                
                if (result.result === "OK") {
                    appendToOutput('\nFinal Result: ' + JSON.stringify(result.data, null, 2), 'success');
                } else if (result.result === "ERROR") {
                    appendToOutput('\nExecution Error: ' + result.data, 'error');
                } else if (result.result === "PENDING") {
                    appendToOutput('\nExecution still running...', 'info');
                }
            } catch (error) {
                appendToOutput('\nFailed to fetch result: ' + error.message, 'error');
            }
        }

        // Helper function to append to output without clearing
        function appendToOutput(text, type) {
            const outputContent = document.getElementById('outputContent');
            if (!outputContent) return;
            
            const line = document.createElement('div');
            if (type === 'error') {
                line.style.color = '#f44747';
            } else if (type === 'success') {
                line.style.color = '#4ec9b0';
            } else if (type === 'info') {
                line.style.color = '#569cd6';
            } else if (type === 'loading') {
                line.style.color = '#dcdcaa';
            }
            line.innerHTML = text;
            outputContent.appendChild(line);
            
            // Auto-scroll to bottom
            outputContent.scrollTop = outputContent.scrollHeight;
        }

        // Add expand/collapse handlers
        function addTreeToggleHandlers(panel) {
            panel.querySelectorAll('.tree-toggle').forEach(toggle => {
                toggle.parentElement.classList.add('tree-collapsed');
                toggle.addEventListener('click', function(e) {
                    e.stopPropagation();
                    this.parentElement.classList.toggle('tree-collapsed');
                    this.textContent = this.parentElement.classList.contains('tree-collapsed') ? '▶' : '▼';
                });
            });

            // Add click handlers for functions
            panel.querySelectorAll('.tree-function').forEach(funcElement => {
                funcElement.addEventListener('click', function(e) {
                    e.stopPropagation();
                    const functionText = this.textContent;
                    
                    // Extract function details from the tree structure
                    const attributeName = this.closest('div').querySelector('.tree-key').textContent;
                    const nodePath = findNodePath(this);
                    
                    loadTreeNodeFunctionForEditing(nodePath, attributeName, functionText);
                });
            });
        }

        // Find the path of the node in the tree view
        function findNodePath(element) {
            const path = [];
            let current = element.closest('div');
            
            while (current && current.closest('.tree-view')) {
                const keyElement = current.querySelector(':scope > .tree-key');
                if (keyElement) {
                    const keyText = keyElement.textContent;
                    // Skip attribute names, we want the node path
                    const parentDiv = current.parentElement?.parentElement;
                    if (parentDiv && parentDiv.querySelector(':scope > .tree-node-type')) {
                        path.unshift(keyText);
                    }
                }
                current = current.parentElement?.parentElement;
            }
            
            return path;
        }

        // Load TreeNode function attribute for editing
        async function loadTreeNodeFunctionForEditing(nodePath, attributeName, functionText) {
            try {
                // Try to extract function code from the function string representation
                let functionCode = functionText;
                
                // If it's a "Function(...)" representation, we need to get the actual source
                if (functionText.startsWith('Function(') && functionText.endsWith(')')) {
                    // For now, we'll create a placeholder - in a real implementation,
                    // you might need to call the backend to get the function source
                    functionCode = "func(profile) { \n    // Edit this function code\n    // Original: " + functionText + "\n    bigger(getAttribute(profile, 'age'), 18) \n}";
                }
                
                // Build the node access path
                let nodeAccessCode = 'agent'; // Start with the root node
                if (nodePath.length > 0) {
                    // Build path like: getChildAt(agent, 0) for first level, etc.
                    for (let i = 0; i < nodePath.length; i++) {
                        const childName = nodePath[i];
                        // This is simplified - you might need more sophisticated path building
                        // based on how your tree structure works
                        if (childName === 'offer') {
                            nodeAccessCode = "getChildAt(" + nodeAccessCode + ", 0)";
                        } else if (childName === 'rules') {
                            nodeAccessCode = "getChildAt(" + nodeAccessCode + ", 1)";
                        } else if (childName === 'handlers') {
                            nodeAccessCode = "getChildAt(" + nodeAccessCode + ", 2)";
                        }
                    }
                }
                
                // Create wrapper code that will update the function in the TreeNode
                const wrapperCode = "// TreeNode Function: " + attributeName + "\n" +
                    "// This code will update the function attribute in the TreeNode when executed\n\n" +
                    "// Get the node reference\n" +
                    "setq(targetNode, " + nodeAccessCode + ")\n\n" +
                    "// Define the updated function\n" +
                    "setq(updatedFunction, " + functionCode + ")\n\n" +
                    "// Set the attribute with the new function\n" +
                    "setAttribute(targetNode, '" + attributeName + "', updatedFunction)\n\n" +
                    "// Optional: Save the tree structure\n" +
                    "// treeSave(agent, 'decisionAgent1.json')\n\n" +
                    "// Return confirmation\n" +
                    "concat(\"Updated function '" + attributeName + "' in node\")";

                if (editor) {
                    editor.setValue(wrapperCode);
                    currentFileName = ''; // Not a file
                    originalContent = wrapperCode;
                    isFileModified = false;
                    
                    // Switch to Files tab to show the code
                    document.getElementById('filesTab').click();
                    
                    updateSaveButtonStates();
                    updateRunButtonState();
                    
                    showOutput("Loaded TreeNode function \"" + attributeName + "\" for editing. Run to update the node.", 'info');
                }
            } catch (e) {
                showOutput('Error loading TreeNode function: ' + e.message, 'error');
            }
        }

        // Recursively render a JS object as a tree
        function renderTree(obj, key = null, nodePath = []) {
            if (obj === null) {
                if (key !== null) {
                    return '<span class="tree-key">' + escapeHtml(key) + '</span>: <span class="tree-leaf">null</span>';
                }
                return '<span class="tree-leaf">null</span>';
            }
            if (typeof obj !== 'object') {
                // Check if this is a function representation
                const objStr = obj.toString();
                if (objStr.startsWith('Function(') && objStr.endsWith(')')) {
                    if (key !== null) {
                        return '<span class="tree-key">' + escapeHtml(key) + '</span>: <span class="tree-function" data-node-path="' + escapeHtml(JSON.stringify(nodePath)) + '" data-attribute="' + escapeHtml(key) + '">' + escapeHtml(objStr) + '</span>';
                    } else {
                        return '<span class="tree-function" data-node-path="' + escapeHtml(JSON.stringify(nodePath)) + '">' + escapeHtml(objStr) + '</span>';
                    }
                }
                
                if (key !== null) {
                    return '<span class="tree-key">' + escapeHtml(key) + '</span>: <span class="tree-leaf">' + escapeHtml(objStr) + '</span>';
                } else {
                    return '<span class="tree-leaf">' + escapeHtml(objStr) + '</span>';
                }
            }
            let html = '';
            if (Array.isArray(obj)) {
                const keyPart = key !== null ? escapeHtml(key) : '[Array]';
                html += '<div><span class="tree-toggle">▶</span><span class="tree-key">' + keyPart + '</span>: [<div class="tree-children" style="margin-left:16px;">';
                obj.forEach((item, idx) => {
                    html += '<div>' + renderTree(item, idx, [...nodePath, idx]) + '</div>';
                });
                html += '</div>]</div>';
            } else {
                // Check if this looks like a TreeNode/Node object
                const hasTypeAndName = obj.hasOwnProperty('type') && obj.hasOwnProperty('name');
                const hasNodeProperties = obj.hasOwnProperty('children') || obj.hasOwnProperty('attributes') || obj.hasOwnProperty('text');
                const isTreeNode = hasTypeAndName || hasNodeProperties;
                
                if (isTreeNode) {
                    const keyPart = key !== null ? escapeHtml(key) : 'Node';
                    const nodeName = obj.name || obj.type || 'unnamed';
                    const currentNodePath = key !== null ? [...nodePath, key] : nodePath;
                    html += '<div><span class="tree-toggle">▶</span><span class="tree-key">' + keyPart + '</span>: <span class="tree-node-type">Node(' + escapeHtml(nodeName) + ')</span><div class="tree-children" style="margin-left:16px;">';
                    for (const k in obj) {
                        html += '<div>' + renderTree(obj[k], k, currentNodePath) + '</div>';
                    }
                    html += '</div></div>';
                } else {
                    // Regular object
                    const keyPart = key !== null ? escapeHtml(key) : 'Object';
                    const currentPath = key !== null ? [...nodePath, key] : nodePath;
                    html += '<div><span class="tree-toggle">▶</span><span class="tree-key">' + keyPart + '</span>: {<div class="tree-children" style="margin-left:16px;">';
                    for (const k in obj) {
                        html += '<div>' + renderTree(obj[k], k, currentPath) + '</div>';
                    }
                    html += '</div>}</div>';
                }
            }
            return html;
        }

        // Problems tab logger
        function showProblem(text, level) {
            const content = document.getElementById('problemsContent');
            if (!content) return;
            const timestamp = new Date().toLocaleTimeString();
            let color = '#d4d4d4';
            if (level === 'error') color = '#f44747';
            else if (level === 'warn' || level === 'warning') color = '#d7ba7d';
            else if (level === 'info') color = '#4fc1ff';
            const line = document.createElement('div');
            line.style.padding = '6px 8px';
            line.style.borderBottom = '1px solid #3e3e42';
            line.innerHTML = '[' + timestamp + '] <span style="color:' + color + '">' + escapeHtml(text) + '</span>';
            content.appendChild(line);
            // Auto-scroll to latest
            content.scrollTop = content.scrollHeight;
        }
        
        // Update Run button state based on editor content
        function updateRunButtonState() {
            const runButton = document.getElementById('runButton');
            if (!runButton) return;
            
            const hasAuth = authToken !== null;
            const hasContent = editor && editor.getValue().trim() !== '';
            
            // Enable Run button only if authenticated AND there's code to run
            runButton.disabled = !(hasAuth && hasContent);
            
            console.log('DEBUG: Run button state updated - hasAuth:', hasAuth, 'hasContent:', hasContent, 'disabled:', runButton.disabled);
        }
        
        // Show output in the output tab
        function showOutput(text, type) {
            const content = document.getElementById('outputContent');
            if (!content) {
                console.log('DEBUG: outputContent element not found, message:', text);
                return;
            }
            
            const timestamp = new Date().toLocaleTimeString();
            
            let className = '';
            switch (type) {
                case 'success': className = 'output-success'; break;
                case 'error': className = 'output-error'; break;
                case 'info': className = 'output-info'; break;
                case 'loading': className = 'loading'; break;
            }
            
            content.innerHTML = '[' + timestamp + '] <span class="' + className + '">' + escapeHtml(text) + '</span>';
            content.scrollTop = content.scrollHeight;
        }
        
        // Escape HTML for safe display
        function escapeHtml(text) {
            const div = document.createElement('div');
            div.textContent = text;
            return div.innerHTML;
        }
        
        // Initialize splitter for resizing
        function initializeSplitter() {
            const splitter = document.getElementById('splitter');
            const bottomPanel = document.getElementById('bottomPanel');
            
            splitter.addEventListener('mousedown', function(e) {
                isResizing = true;
                document.addEventListener('mousemove', handleResize);
                document.addEventListener('mouseup', stopResize);
                e.preventDefault();
            });
            
            function handleResize(e) {
                if (!isResizing) return;
                
                const containerHeight = window.innerHeight - 50;
                const mouseY = e.clientY - 50;
                const newBottomHeight = containerHeight - mouseY;
                
                if (newBottomHeight >= 100 && newBottomHeight <= containerHeight - 200) {
                    bottomPanel.style.height = newBottomHeight + 'px';
                    if (editor) {
                        editor.layout();
                    }
                }
            }
            
            function stopResize() {
                isResizing = false;
                document.removeEventListener('mousemove', handleResize);
                document.removeEventListener('mouseup', stopResize);
            }
        }


        // Track file modifications
        function trackFileChanges() {
            if (editor) {
                editor.onDidChangeModelContent(() => {
                    const currentContent = editor.getValue();
                    const wasModified = isFileModified;
                    isFileModified = (currentContent !== originalContent);
                    
                    console.log('Content changed - isFileModified:', isFileModified, 'currentContent length:', currentContent.length, 'originalContent length:', originalContent.length);
                    
                    // Always update button states when content changes
                    updateSaveButtonStates();
                    updateRunButtonState(); // Update Run button state on any content change
                });
            }
        }

        // Update save button states
        function updateSaveButtonStates() {
            const saveButton = document.getElementById('saveButton');
            const saveAsButton = document.getElementById('saveAsButton');
            const renameButton = document.getElementById('renameButton');
            const deleteButton = document.getElementById('deleteButton');
            
            // Functions tab buttons
            const saveFunctionButton = document.getElementById('saveFunctionButton');
            const saveAsFunctionButton = document.getElementById('saveAsFunctionButton');
            const deleteFunctionButton = document.getElementById('deleteFunctionButton');
            
            // Check if main buttons exist
            if (!saveButton || !saveAsButton) {
                return;
            }
            
            const hasAuth = authToken !== null;
            const hasFile = currentFileName !== '';
            const hasContent = editor && editor.getValue().trim() !== '';
            const activeTab = document.querySelector('.toolbar-tab.active')?.textContent;
            
            // Debug logging
            console.log('updateSaveButtonStates - activeTab:', activeTab, 'hasAuth:', hasAuth, 'hasContent:', hasContent, 'isFileModified:', isFileModified);
            
            // Files tab buttons
            if (activeTab === 'Files') {
                // Save button: enabled if authenticated, has current file, and content is modified
                saveButton.disabled = !(hasAuth && hasFile && isFileModified);
                if (isFileModified && hasFile) {
                    saveButton.classList.add('modified');
                } else {
                    saveButton.classList.remove('modified');
                }
                
                // Save As button: enabled if authenticated and has content
                saveAsButton.disabled = !(hasAuth && hasContent);
                
                // Rename and Delete buttons: enabled only if authenticated and a file is loaded
                if (renameButton) {
                    renameButton.disabled = !(hasAuth && hasFile);
                }
                if (deleteButton) {
                    deleteButton.disabled = !(hasAuth && hasFile);
                }
            }
            
            // Functions tab buttons
            if (activeTab === 'Function Library') {
                console.log('In Function Library tab, functionEditorFunctionName:', functionEditorFunctionName);
                
                // Save Function button: enabled if authenticated, has function loaded, and content is modified
                if (saveFunctionButton) {
                    const hasFunction = functionEditorFunctionName !== '';
                    saveFunctionButton.disabled = !(hasAuth && hasFunction && isFileModified);
                    console.log('Save Function button - hasFunction:', hasFunction, 'disabled:', saveFunctionButton.disabled);
                }
                
                // Save As Function button: enabled if authenticated and has content
                if (saveAsFunctionButton) {
                    saveAsFunctionButton.disabled = !(hasAuth && hasContent);
                    console.log('Save As Function button - disabled:', saveAsFunctionButton.disabled);
                }
                
                // Delete Function button: enabled if authenticated and has function loaded
                if (deleteFunctionButton) {
                    const hasFunction = functionEditorFunctionName !== '';
                    deleteFunctionButton.disabled = !(hasAuth && hasFunction);
                }
            }
        }

        // Save current file
        async function saveFile() {
            if (!authToken) {
                showOutput('Please log in first', 'error');
                return;
            }
            
            if (!currentFileName) {
                showOutput('No file selected to save', 'error');
                return;
            }
            
            const content = editor.getValue();
            const saveButton = document.getElementById('saveButton');
            
            saveButton.disabled = true;
            saveButton.textContent = '💾 Saving...';
            
            try {
                const response = await fetch(getAPIPath('/api/save'), {
                    method: 'POST',
                    headers: getAuthHeadersWithJSON(),
                    body: JSON.stringify({
                        path: CHARIOT_FILES_FOLDER + '/' + currentFileName,
                        content: content
                    })
                });
                
                if (response.status === 401) {
                    logout();
                    return;
                }
                
                if (response.ok) {
                    originalContent = content;
                    isFileModified = false;
                    updateSaveButtonStates();
                    showOutput('File saved successfully: ' + currentFileName, 'success');
                } else {
                    const error = await response.text();
                    showOutput('Save failed: ' + error, 'error');
                }
                
            } catch (error) {
                showOutput('Save error: ' + error.message, 'error');
            } finally {
                saveButton.disabled = false;
                saveButton.textContent = '💾 Save';
            }
        }

        // Create new file
        function newFile() {
            if (!authToken) {
                showOutput('Please log in first', 'error');
                return;
            }
            
            // Clear the editor
            if (editor) {
                editor.setValue('// New Chariot Script\n');
                currentFileName = '';
                originalContent = '';
                isFileModified = true;
                updateSaveButtonStates();
                updateRunButtonState();
                
                // Clear file selection
                const fileSelect = document.getElementById('fileSelect');
                if (fileSelect) {
                    fileSelect.value = '';
                }
                
                showOutput('New file created. Use "Save As..." to save with a filename.', 'info');
            }
        }

        // Save as new file
        async function saveAsFile() {
            if (!authToken) {
                showOutput('Please log in first', 'error');
                return;
            }
            
            const content = editor.getValue();
            if (!content.trim()) {
                showOutput('No content to save', 'error');
                return;
            }
            
            // Prompt for filename
            const fileName = prompt('Enter filename (with .ch extension):', 'new_file.ch');
            if (!fileName) {
                return; // User cancelled
            }
            
            // Validate filename
            if (!fileName.endsWith('.ch')) {
                showOutput('Filename must end with .ch extension', 'error');
                return;
            }
            
            const saveAsButton = document.getElementById('saveAsButton');
            saveAsButton.disabled = true;
            saveAsButton.textContent = '💾 Saving...';
            
            try {
                const response = await fetch(getAPIPath('/api/save'), {
                    method: 'POST',
                    headers: getAuthHeadersWithJSON(),
                    body: JSON.stringify({
                        path: CHARIOT_FILES_FOLDER + '/' + fileName,
                        content: content
                    })
                });
                
                if (response.status === 401) {
                    logout();
                    return;
                }
                
                if (response.ok) {
                    // Switch to the new file
                    currentFileName = fileName;
                    originalContent = content;
                    isFileModified = false;
                    
                    // Refresh file list and select the new file
                    await loadFileList();
                    document.getElementById('fileSelect').value = fileName;
                    
                    updateSaveButtonStates();
                    showOutput('File saved as: ' + fileName, 'success');
                } else {
                    const error = await response.text();
                    showOutput('Save As failed: ' + error, 'error');
                }
                
            } catch (error) {
                showOutput('Save As error: ' + error.message, 'error');
            } finally {
                saveAsButton.disabled = false;
                saveAsButton.textContent = '💾 Save As...';
            }
        }

        // Save button event listener        
        // Load list of files from the server
        async function loadFileList() {
            console.log('DEBUG: loadFileList called, authToken:', !!authToken);
            
            if (!authToken) {
                console.log('DEBUG: No authToken, skipping file list load');
                return;
            }
            
            try {
                const url = getAPIPath('/api/files?folder=' + encodeURIComponent(CHARIOT_FILES_FOLDER));
                console.log('DEBUG: Fetching file list from:', url);
                
                const headers = getAuthHeaders();
                console.log('DEBUG: Using headers:', headers);
                
                const response = await fetch(url, {
                    headers: headers
                });
                
                console.log('DEBUG: Response status:', response.status);
                
                if (response.status === 401) {
                    console.log('DEBUG: Unauthorized, logging out');
                    logout();
                    return;
                }
                
                if (response.ok) {
                    const result = await response.json();
                    console.log('DEBUG: Received response:', result);
                    
                    if (result.result === "OK") {
                        const files = result.data;
                        console.log('DEBUG: Received files:', files);
                        
                        const fileSelect = document.getElementById('fileSelect');
                        fileSelect.innerHTML = '<option value="">Select a file...</option>';
                        
                        if (files && files.length > 0) {
                            files.forEach(file => {
                                const option = document.createElement('option');
                                option.value = file;
                                option.textContent = file;
                                fileSelect.appendChild(option);
                            });
                            fileSelect.disabled = false;
                            console.log('DEBUG: Added', files.length, 'files to dropdown');
                        } else {
                            console.log('DEBUG: No files found');
                        }
                    } else {
                        console.log('DEBUG: File list error:', result.data);
                        showOutput('Failed to load file list: ' + result.data, 'error');
                    }
                } else {
                    console.error('DEBUG: Failed to load files:', response.statusText);
                }
            } catch (error) {
                console.error('DEBUG: Error loading file list:', error);
            }
        }        
        // Populate the dropdown with files
        function populateFileDropdown(files) {
            const select = document.getElementById('fileSelect');
            
            select.innerHTML = '<option value="">Select a file...</option>';
            
            files.forEach(file => {
                const option = document.createElement('option');
                option.value = file;
                option.textContent = file;
                select.appendChild(option);
            });
        }
        
        // Get function list
        async function fetchUserFunctions() {
            try {
                const response = await fetch(getAPIPath('/api/functions'), {
                    headers: getAuthHeaders()
                });
                if (response.ok) {
                    const result = await response.json();
                    if (result.result === "OK" && Array.isArray(result.data)) {
                        return result.data;
                    }
                }
            } catch (e) {
                console.error('Failed to fetch user functions:', e);
            }
            return [];
        }

        // Update the loadFile function to track original content
        async function loadFile(fileName) {
            if (!authToken) return;
            
            try {
                const response = await fetch('/charioteer/api/file?path=' + encodeURIComponent(CHARIOT_FILES_FOLDER + '/' + fileName), {
                    headers: getAuthHeaders()
                });
                
                if (response.status === 401) {
                    logout();
                    return;
                }
                
                if (response.ok) {
                    const result = await response.json();
                    if (result.result === "OK") {
                        const content = result.data;
                        if (editor) {
                            editor.setValue(content);
                            currentFileName = fileName;
                            originalContent = content; // Track original content
                            isFileModified = false;
                            fileEditorContent = content;
                            fileEditorFileName = fileName;
                            updateSaveButtonStates();
                            updateRunButtonState(); // Update Run button state on file load
                        }
                    } else {
                        console.error('Failed to load file:', result.data);
                        alert('Failed to load file: ' + result.data);
                    }
                } else {
                    console.error('Failed to load file:', response.statusText);
                    alert('Failed to load file: ' + fileName);
                }
            } catch (error) {
                console.error('Error loading file:', error);
                alert('Error loading file: ' + fileName);
            }
        }

        // Rename current file
        async function renameFile() {
            if (!authToken) {
                showOutput('Please log in first', 'error');
                return;
            }
            
            if (!currentFileName) {
                showOutput('No file selected to rename', 'error');
                return;
            }
            
            // Prompt for new filename
            const newFileName = prompt('Enter new filename (with .ch extension):', currentFileName);
            if (!newFileName || newFileName === currentFileName) {
                return; // User cancelled or didn't change name
            }
            
            // Validate filename
            if (!newFileName.endsWith('.ch')) {
                showOutput('Filename must end with .ch extension', 'error');
                return;
            }
            
            const renameButton = document.getElementById('renameButton');
            renameButton.disabled = true;
            renameButton.textContent = '📝 Renaming...';
            
            try {
                const response = await fetch(getAPIPath('/api/rename'), {
                    method: 'POST',
                    headers: getAuthHeadersWithJSON(),
                    body: JSON.stringify({
                        oldPath: CHARIOT_FILES_FOLDER + '/' + currentFileName,
                        newPath: CHARIOT_FILES_FOLDER + '/' + newFileName
                    })
                });
                
                if (response.status === 401) {
                    logout();
                    return;
                }
                
                if (response.ok) {
                    const oldFileName = currentFileName;
                    currentFileName = newFileName;
                    
                    // Refresh file list and select the renamed file
                    await loadFileList();
                    document.getElementById('fileSelect').value = newFileName;
                    
                    updateSaveButtonStates();
                    showOutput('File renamed from "' + oldFileName + '" to "' + newFileName + '"', 'success');
                } else {
                    const error = await response.text();
                    showOutput('Rename failed: ' + error, 'error');
                }
                
            } catch (error) {
                showOutput('Rename error: ' + error.message, 'error');
            } finally {
                renameButton.disabled = false;
                renameButton.textContent = '📝 Rename';
            }
        }

        // Delete current file
        async function deleteFile() {
            if (!authToken) {
                showOutput('Please log in first', 'error');
                return;
            }
            
            if (!currentFileName) {
                showOutput('No file selected to delete', 'error');
                return;
            }
            
            // Confirm deletion
            if (!confirm('Are you sure you want to delete "' + currentFileName + '"? This action cannot be undone.')) {
                return;
            }
            
            const deleteButton = document.getElementById('deleteButton');
            deleteButton.disabled = true;
            deleteButton.textContent = '🗑️ Deleting...';
            
            try {
                const response = await fetch(getAPIPath('/api/delete'), {
                    method: 'POST',
                    headers: getAuthHeadersWithJSON(),
                    body: JSON.stringify({
                        path: CHARIOT_FILES_FOLDER + '/' + currentFileName
                    })
                });
                
                if (response.status === 401) {
                    logout();
                    return;
                }
                
                if (response.ok) {
                    const deletedFileName = currentFileName;
                    
                    // Clear editor and file selection
                    if (editor) {
                        editor.setValue('');
                    }
                    currentFileName = '';
                    originalContent = '';
                    isFileModified = false;
                    
                    // Refresh file list and reset selection
                    await loadFileList();
                    document.getElementById('fileSelect').value = '';
                    
                    updateSaveButtonStates();
                    updateRunButtonState();
                    showOutput('File deleted: "' + deletedFileName + '"', 'success');
                } else {
                    const error = await response.text();
                    showOutput('Delete failed: ' + error, 'error');
                }
                
            } catch (error) {
                showOutput('Delete error: ' + error.message, 'error');
            } finally {
                deleteButton.disabled = false;
                deleteButton.textContent = '🗑️ Delete';
            }
        }

        // Dashboard functions
        function refreshDashboardData() {
            // Show a message in the output panel about dashboard refresh
            showOutput('Refreshing dashboard data...', 'info');
            if (currentTab === 'dashboard' && dashboardLoaded) {
                fetchAndUpdateDashboard();
            }
        }

        // Load dashboard content into the editor area
        async function loadDashboardContent() {
            try {
                const editorElement = document.getElementById('editorContainer');
                if (!editorElement) {
                    showOutput('Editor container not found', 'error');
                    return;
                }

                // Hide Monaco editor when displaying dashboard
                if (editor) {
                    editor.getModel()?.dispose();
                    editor.dispose();
                    editor = null;
                }

                // Create dashboard HTML
                const dashboardHTML = '<div class="dashboard-container" style="padding: 20px; color: #d4d4d4; background-color: #1e1e1e; height: 100%; overflow-y: auto; font-family: \'Segoe UI\', Tahoma, Geneva, Verdana, sans-serif;">' +
                    
                    '<div id="dashboardError" style="display: none; background-color: #f44747; color: white; padding: 15px; border-radius: 4px; margin-bottom: 20px;"></div>' +
                    
                    '<div id="dashboardLoading" style="text-align: center; padding: 40px; color: #569cd6;">' +
                        '<p>Loading dashboard data...</p>' +
                    '</div>' +
                    
                    '<div id="dashboardContent" style="display: none;">' +
                        '<div class="metrics-grid" style="display: grid; grid-template-columns: repeat(auto-fit, minmax(300px, 1fr)); gap: 20px; margin-bottom: 30px;">' +
                            '<div class="metric-card" style="background: #2d2d30; border: 1px solid #3e3e42; border-radius: 8px; padding: 20px;">' +
                                '<h3 style="margin: 0 0 15px 0; color: #569cd6; font-size: 18px;">Active Sessions</h3>' +
                                '<div id="activeSessions" style="font-size: 24px; font-weight: bold; color: #4ec9b0; margin-bottom: 10px;">0</div>' +
                                '<div style="color: #cccccc; font-size: 14px;">Currently active user sessions</div>' +
                            '</div>' +
                            
                            '<div class="metric-card" style="background: #2d2d30; border: 1px solid #3e3e42; border-radius: 8px; padding: 20px;">' +
                                '<h3 style="margin: 0 0 15px 0; color: #569cd6; font-size: 18px;">Total Sessions</h3>' +
                                '<div id="totalSessions" style="font-size: 24px; font-weight: bold; color: #4ec9b0; margin-bottom: 10px;">0</div>' +
                                '<div style="color: #cccccc; font-size: 14px;">Total sessions since startup</div>' +
                            '</div>' +
                            
                            '<div class="metric-card" style="background: #2d2d30; border: 1px solid #3e3e42; border-radius: 8px; padding: 20px;">' +
                                '<h3 style="margin: 0 0 15px 0; color: #569cd6; font-size: 18px;">System Uptime</h3>' +
                                '<div id="uptime" style="font-size: 24px; font-weight: bold; color: #4ec9b0; margin-bottom: 10px;">Unknown</div>' +
                                '<div style="color: #cccccc; font-size: 14px;">Server uptime</div>' +
                            '</div>' +
                            
                            '<div class="metric-card" style="background: #2d2d30; border: 1px solid #3e3e42; border-radius: 8px; padding: 20px;">' +
                                '<h3 style="margin: 0 0 15px 0; color: #569cd6; font-size: 18px;">System Status</h3>' +
                                '<div id="systemStatus" style="font-size: 24px; font-weight: bold; color: #4ec9b0; margin-bottom: 10px;">Unknown</div>' +
                                '<div style="color: #cccccc; font-size: 14px;">Current system status</div>' +
                            '</div>' +
                        '</div>' +
                        
                        '<div class="sessions-section" style="background: #2d2d30; border: 1px solid #3e3e42; border-radius: 8px; padding: 20px;">' +
                            '<h3 style="margin: 0 0 20px 0; color: #569cd6; font-size: 18px;">Active Sessions</h3>' +
                            '<div style="overflow-x: auto;">' +
                                '<table style="width: 100%; border-collapse: collapse; color: #d4d4d4;">' +
                                    '<thead>' +
                                        '<tr style="border-bottom: 1px solid #3e3e42;">' +
                                            '<th style="text-align: left; padding: 12px; color: #569cd6;">Username</th>' +
                                            '<th style="text-align: left; padding: 12px; color: #569cd6;">Session ID</th>' +
                                            '<th style="text-align: left; padding: 12px; color: #569cd6;">Created</th>' +
                                            '<th style="text-align: left; padding: 12px; color: #569cd6;">Last Access</th>' +
                                            '<th style="text-align: left; padding: 12px; color: #569cd6;">Status</th>' +
                                        '</tr>' +
                                    '</thead>' +
                                    '<tbody id="sessionsTableBody">' +
                                        '<tr>' +
                                            '<td colspan="5" style="text-align: center; padding: 20px; color: #888;">Loading sessions...</td>' +
                                        '</tr>' +
                                    '</tbody>' +
                                '</table>' +
                            '</div>' +
                        '</div>' +
                    '</div>' +
                '</div>';

                // Set the dashboard HTML
                editorElement.innerHTML = dashboardHTML;
                
                // Save dashboard content for state management
                dashboardContent = dashboardHTML;
                dashboardLoaded = true;

                // Load dashboard data
                await fetchAndUpdateDashboard();

            } catch (error) {
                console.error('Error loading dashboard content:', error);
                showOutput('Failed to load dashboard: ' + error.message, 'error');
            }
        }

        // Agents UI state and helpers
    let agentsContent = '';
    let agentsLoaded = false;
    let agentsWS = null;
    let agentsWSBackoffMs = 1000;
    let agentsWSForcedPolling = false;
    let agentsWSReconnectEnabled = false; // only reconnect when Agents tab is active
    let agentsWSReconnectTimer = null;    // pending reconnect timer id
    let agentsWSConnecting = false;       // prevent concurrent connect attempts
    let agentsShowHeartbeats = false;     // UI toggle to show/hide heartbeat messages

        function stopAgentsWS() {
            // Disable reconnects and close any existing socket
            agentsWSReconnectEnabled = false;
            if (agentsWSReconnectTimer) { try { clearTimeout(agentsWSReconnectTimer); } catch (e) {} agentsWSReconnectTimer = null; }
            try { if (agentsWS) { agentsWS.onclose = null; agentsWS.onerror = null; agentsWS.close(); } } catch (e) {}
            agentsWS = null;
            agentsWSConnecting = false;
        }

        // Build and load the Agents UI into the editor area
        async function loadAgentsContent() {
            try {
                // Hide Monaco editor when displaying agents
                const editorElement = document.getElementById('editorContainer');
                if (editor) {
                    editor.getModel()?.dispose();
                    editor.dispose();
                    editor = null;
                }
                                const agentsHTML = '<div class="agents-container" style="padding: 20px; color: #d4d4d4; background-color: #1e1e1e; height: 100%; overflow-y: auto; font-family: \'Segoe UI\', Tahoma, Geneva, Verdana, sans-serif;">' +
                    '<div id="agentsError" style="display: none; background-color: #f44747; color: white; padding: 15px; border-radius: 4px; margin-bottom: 20px;"></div>' +
                    '<div id="agentsLoading" style="text-align: center; padding: 40px; color: #569cd6;">' +
                        '<p>Loading agents…</p>' +
                    '</div>' +
                    '<div id="agentsContent" style="display:none;">' +
                        '<div style="display:flex; gap:24px; align-items:flex-start;">' +
                            '<div style="flex:0 0 320px; background:#252526; border:1px solid #333; border-radius:6px; padding:12px;">' +
                                '<h3 style="margin-top:0; color:#569cd6;">Agents</h3>' +
                                '<div id="agentsList" style="max-height:50vh; overflow:auto; font-family: monospace;"></div>' +
                            '</div>' +
                            '<div style="flex:1; background:#252526; border:1px solid #333; border-radius:6px; padding:12px;">' +
                                                                '<div style="display:flex; align-items:center; justify-content:space-between; gap:12px;">' +
                                                                    '<h3 style="margin-top:0; color:#569cd6;">Agent events</h3>' +
                                                                    '<div style="display:flex; align-items:center; gap:10px;">' +
                                                                        '<label style="font-size:12px; color:#bbb; display:flex; align-items:center; gap:6px;">' +
                                                                             '<input type="checkbox" id="toggleHeartbeats" /> Show heartbeats' +
                                                                        '</label>' +
                                                                        '<button id="clearAgentsStreamButton" class="toolbar-button">🧹 Clear</button>' +
                                                                    '</div>' +
                                                                '</div>' +
                                '<pre id="agentsStream" style="height:50vh; overflow:auto; background:#1e1e1e; color:#d4d4d4; padding:12px; border-radius:4px; border:1px solid #333; white-space:pre-wrap; word-break:break-word;"></pre>' +
                            '</div>' +
                        '</div>' +
                    '</div>' +
                '</div>';

                // Inject and save state
                editorElement.innerHTML = agentsHTML;
                agentsContent = agentsHTML;
                agentsLoaded = true;

                // Bind clear button and toggle
                const clearBtn = document.getElementById('clearAgentsStreamButton');
                if (clearBtn) {
                    clearBtn.addEventListener('click', function() {
                        const s = document.getElementById('agentsStream');
                        if (s) s.textContent = '';
                    });
                }
                const hbToggle = document.getElementById('toggleHeartbeats');
                if (hbToggle) {
                    hbToggle.checked = agentsShowHeartbeats;
                    hbToggle.addEventListener('change', function() {
                        agentsShowHeartbeats = hbToggle.checked;
                    });
                }

                // Initial list load
                await fetchAndRenderAgents();
            } catch (error) {
                console.error('Error loading agents content:', error);
                showOutput('Failed to load agents: ' + error.message, 'error');
            }
        }

        function connectAgentsWS() {
            // Only connect/reconnect if enabled (Agents tab active)
            if (!agentsWSReconnectEnabled) return;
            // Prevent parallel attempts
            if (agentsWSConnecting) return;
            agentsWSConnecting = true;
            // Clear any pending reconnect timer to avoid stacked connects
            if (agentsWSReconnectTimer) { try { clearTimeout(agentsWSReconnectTimer); } catch (e) {} agentsWSReconnectTimer = null; }
            // Avoid duplicate sockets: if an open or connecting socket exists, don't disrupt it
            try {
                if (agentsWS) {
                    const rs = agentsWS.readyState; // 0=CONNECTING,1=OPEN,2=CLOSING,3=CLOSED
                    if (rs === 0 || rs === 1 || rs === 2) {
                        agentsWSConnecting = false;
                        return;
                    }
                    // CLOSED: allow a fresh connect
                }
            } catch (e) { /* ignore */ }
            const proto = (window.location.protocol === 'https:') ? 'wss' : 'ws';
            const basePath = window.location.pathname.startsWith('/charioteer/') ? '/charioteer' : '';
            const token = (authToken || localStorage.getItem('chariot_token') || '').trim();
            const qs = token ? ('?token=' + encodeURIComponent(token)) : '';
            const wsURL = proto + '://' + window.location.host + basePath + '/ws/agents' + qs;
            try {
                agentsWS = new WebSocket(wsURL);
                agentsWS.onopen = () => {
                    console.log('Agents WS connected');
                    agentsWSBackoffMs = 1000;
                    agentsWSForcedPolling = false;
                    agentsWSConnecting = false;
                    if (agentsWSReconnectTimer) { try { clearTimeout(agentsWSReconnectTimer); } catch (e) {} agentsWSReconnectTimer = null; }
                    const err = document.getElementById('agentsError');
                    if (err) { err.style.display = 'none'; }
                };
                agentsWS.onmessage = (evt) => {
                    const s = document.getElementById('agentsStream');
                    if (!s) return;
                    try {
                        const msg = JSON.parse(evt.data);
                        // Filter heartbeats unless explicitly enabled
                        if (msg && msg.type === 'heartbeat' && !agentsShowHeartbeats) {
                            return;
                        }
                        const line = (typeof msg === 'string') ? msg : JSON.stringify(msg);
                        s.textContent += (s.textContent ? '\n' : '') + line;
                        s.scrollTop = s.scrollHeight;
                    } catch (e) {
                        s.textContent += (s.textContent ? '\n' : '') + evt.data;
                        s.scrollTop = s.scrollHeight;
                    }
                    const loading = document.getElementById('agentsLoading');
                    const content = document.getElementById('agentsContent');
                    if (loading) loading.style.display = 'none';
                    if (content) content.style.display = 'block';
                };
                agentsWS.onclose = (ev) => {
                    console.log('Agents WS closed', ev && ev.code, ev && ev.reason);
                    agentsWSConnecting = false;
                    if (agentsWSReconnectEnabled && token && !agentsWSForcedPolling && currentTab === 'agents') {
                        const err = document.getElementById('agentsError');
                        if (err) { err.textContent = 'Realtime link lost, retrying…'; err.style.display = 'block'; }
                        if (agentsWSReconnectTimer) { try { clearTimeout(agentsWSReconnectTimer); } catch (e) {} }
                        agentsWSReconnectTimer = setTimeout(() => { agentsWSReconnectTimer = null; connectAgentsWS(); }, Math.min(agentsWSBackoffMs, 30000));
                        agentsWSBackoffMs = Math.min(agentsWSBackoffMs * 2, 30000);
                    }
                };
                agentsWS.onerror = (e) => {
                    console.log('Agents WS error', e);
                    agentsWSConnecting = false;
                    if (agentsWSReconnectEnabled && token && !agentsWSForcedPolling && currentTab === 'agents') {
                        const err = document.getElementById('agentsError');
                        if (err) { err.textContent = 'Realtime error, retrying…'; err.style.display = 'block'; }
                        try { agentsWS.close(); } catch (e) {}
                        if (agentsWSReconnectTimer) { try { clearTimeout(agentsWSReconnectTimer); } catch (e) {} }
                        agentsWSReconnectTimer = setTimeout(() => { agentsWSReconnectTimer = null; connectAgentsWS(); }, Math.min(agentsWSBackoffMs, 30000));
                        agentsWSBackoffMs = Math.min(agentsWSBackoffMs * 2, 30000);
                    }
                };
            } catch (e) {
                console.warn('Agents WS not available, using no-op');
                agentsWSConnecting = false;
            }
        }

    async function fetchAndRenderAgents() {
            const err = document.getElementById('agentsError');
            const loading = document.getElementById('agentsLoading');
            const content = document.getElementById('agentsContent');
            try {
                if (err) err.style.display = 'none';
                if (loading) loading.style.display = 'block';
                if (content) content.style.display = 'none';
                const resp = await fetch('/charioteer/api/agents', { headers: getAuthHeaders() });
                if (!resp.ok) throw new Error('Failed to fetch agents: ' + resp.statusText);
                const result = await resp.json();
                let agents = [];
                if (result) {
                    if (Array.isArray(result)) {
                        agents = result;
                    } else if (Array.isArray(result.agents)) {
                        agents = result.agents;
                    } else if (result.data) {
                        if (Array.isArray(result.data)) {
                            agents = result.data;
                        } else if (Array.isArray(result.data.agents)) {
                            agents = result.data.agents;
                        }
                    }
                }
                const listDiv = document.getElementById('agentsList');
                if (listDiv) {
                    if (!agents || agents.length === 0) {
                        listDiv.innerHTML = '<div style="color:#888;">No agents</div>';
                    } else {
                        listDiv.innerHTML = agents.map(a => '<div style="padding:4px 0;">' + (a && a.name ? a.name : a) + '</div>').join('');
                    }
                }
                if (loading) loading.style.display = 'none';
                if (content) content.style.display = 'block';
            } catch (e) {
                console.error('Agents API error:', e);
                if (err) { err.textContent = e.message; err.style.display = 'block'; }
                if (loading) loading.style.display = 'none';
                if (content) content.style.display = 'none';
            }
        }

        // (removed duplicate connectAgentsWS definition)

        // WebSocket client for dashboard stream
        let dashboardWS = null;
        let dashboardWSBackoffMs = 1000; // start at 1s, cap later
        let dashboardWSForcedPolling = false; // only used if we truly can't WS at all

        function showDashboardStatusBanner(text, kind) {
            const elId = 'dashboardError';
            let el = document.getElementById(elId);
            if (!el) {
                // Create hidden placeholder but we will not inject into DOM prominently
                el = document.createElement('div');
                el.id = elId;
                el.style.display = 'none';
            }
            // Hide when no text provided
            if (!text) {
                el.style.display = 'none';
                el.textContent = '';
                return;
            }
            // Route transient alerts to Problems tab instead of visible banner
            const level = (kind === 'warn' || kind === 'warning') ? 'warn' : (kind === 'error' ? 'error' : 'info');
            showProblem(text, level);
        }

    function connectDashboardWS() {
            try {
                // Determine WS URL based on current path and protocol
                const proto = (window.location.protocol === 'https:') ? 'wss' : 'ws';
                const basePath = window.location.pathname.startsWith('/charioteer/') ? '/charioteer' : '';
        const token = (authToken || localStorage.getItem('chariot_token') || '').trim();
        const qs = token ? ('?token=' + encodeURIComponent(token)) : '';
        const wsURL = proto + '://' + window.location.host + basePath + '/ws/dashboard' + qs;
                // Browser WebSocket cannot set custom headers; we rely on authMiddleware
                // which reads Authorization header from the initial HTTP request. To supply it,
                // we append it as a query parameter that our proxy ignores for security, but
                // our authMiddleware still checks request headers. As browsers cannot set headers,
                // we also support a cookie or localStorage-based transfer via a small fetch before WS.

                // Attempt direct connect; server-side proxy will read Authorization from initial HTTP upgrade
                dashboardWS = new WebSocket(wsURL);
                dashboardWS.onopen = () => {
                    console.log('Dashboard WS connected');
                    dashboardWSBackoffMs = 1000; // reset backoff on success
                    dashboardWSForcedPolling = false;
                    showDashboardStatusBanner('', '');
                };
                dashboardWS.onmessage = (evt) => {
                    try {
                        const msg = JSON.parse(evt.data);
                        if (msg && msg.result === 'OK') {
                            // Throttle UI updates
                            pendingDashboardData = msg.data;
                            if (!dashboardWSUpdateTimer) {
                                dashboardWSUpdateTimer = setTimeout(() => {
                                    dashboardWSUpdateTimer = null;
                                    const data = pendingDashboardData;
                                    pendingDashboardData = null;
                                    if (data) updateDashboardUI(data);
                                    const dl = document.getElementById('dashboardLoading');
                                    const dc = document.getElementById('dashboardContent');
                                    if (dl) dl.style.display = 'none';
                                    if (dc) dc.style.display = 'block';
                                }, 500);
                            }
                        }
                    } catch (e) {
                        console.warn('WS message parse error', e);
                    }
                };
                dashboardWS.onclose = (ev) => {
                    console.log('Dashboard WS closed', ev && ev.code, ev && ev.reason);
                    // If we have a token, prefer reconnect with backoff instead of polling
                    const token = (authToken || localStorage.getItem('chariot_token') || '').trim();
                    if (token && !dashboardWSForcedPolling) {
                        showDashboardStatusBanner('Realtime link lost, retrying…', 'warn');
                        setTimeout(() => connectDashboardWS(), Math.min(dashboardWSBackoffMs, 30000));
                        dashboardWSBackoffMs = Math.min(dashboardWSBackoffMs * 2, 30000);
                        return;
                    }
                    // No token? then use polling (read-only-ish view)
                    startDashboardAutoRefresh();
                };
                dashboardWS.onerror = (e) => {
                    console.log('Dashboard WS error', e);
                    // Try reconnect with backoff when token exists
                    const token = (authToken || localStorage.getItem('chariot_token') || '').trim();
                    if (token && !dashboardWSForcedPolling) {
                        showDashboardStatusBanner('Realtime error, retrying…', 'warn');
                        try { dashboardWS.close(); } catch (e) {}
                        setTimeout(() => connectDashboardWS(), Math.min(dashboardWSBackoffMs, 30000));
                        dashboardWSBackoffMs = Math.min(dashboardWSBackoffMs * 2, 30000);
                        return;
                    }
                    startDashboardAutoRefresh();
                };
            } catch (e) {
                console.warn('WS connect failed, fallback to polling', e);
                // If we truly cannot WS at all (e.g., environment restrictions), switch to polling
                dashboardWSForcedPolling = true;
                startDashboardAutoRefresh();
            }
        }

        // Fetch and update dashboard data (HTTP fallback)
        async function fetchAndUpdateDashboard() {
            try {
                const dashboardError = document.getElementById('dashboardError');
                const dashboardLoading = document.getElementById('dashboardLoading');
                const dashboardContent = document.getElementById('dashboardContent');

                if (dashboardError) dashboardError.style.display = 'none';
                if (dashboardLoading) dashboardLoading.style.display = 'block';
                if (dashboardContent) dashboardContent.style.display = 'none';

                // Get auth headers
                const headers = getAuthHeaders();
                headers['Content-Type'] = 'application/json';

                const response = await fetch('/charioteer/api/dashboard/status', {
                    method: 'GET',
                    headers: headers
                });

                if (!response.ok) {
                    throw new Error('Failed to fetch dashboard data: ' + response.statusText);
                }

                const result = await response.json();
                if (result.result !== 'OK') {
                    throw new Error('Dashboard API error: ' + (result.data || 'Unknown error'));
                }

                // Update dashboard with data
                console.log('Dashboard API response:', result.data);
                updateDashboardUI(result.data);

                if (dashboardLoading) dashboardLoading.style.display = 'none';
                if (dashboardContent) dashboardContent.style.display = 'block';

            } catch (error) {
                console.error('Error fetching dashboard data:', error);
                
                const dashboardError = document.getElementById('dashboardError');
                const dashboardLoading = document.getElementById('dashboardLoading');
                const dashboardContent = document.getElementById('dashboardContent');

                // Route error to Problems tab, keep banner hidden
                showProblem('Failed to load dashboard data: ' + error.message, 'error');
                if (dashboardError) { dashboardError.style.display = 'none'; }
                if (dashboardLoading) dashboardLoading.style.display = 'none';
                if (dashboardContent) dashboardContent.style.display = 'none';
            }
    }

        // Function to format Go duration string to human readable format
        function formatUptime(uptimeStr) {
            if (!uptimeStr || uptimeStr === 'Unknown') return 'Unknown';
            
            // Parse Go duration string like "2h33m30.783968579s"
            const timeUnits = {
                'h': 'hour',
                'm': 'minute', 
                's': 'second'
            };
            
            let result = [];
            let remaining = uptimeStr;
            
            // Extract hours
            const hourMatch = remaining.match(/(\d+)h/);
            if (hourMatch) {
                const hours = parseInt(hourMatch[1]);
                if (hours > 0) {
                    result.push(hours + ' ' + (hours === 1 ? 'hour' : 'hours'));
                }
                remaining = remaining.replace(/\d+h/, '');
            }
            
            // Extract minutes
            const minuteMatch = remaining.match(/(\d+)m/);
            if (minuteMatch) {
                const minutes = parseInt(minuteMatch[1]);
                if (minutes > 0) {
                    result.push(minutes + ' ' + (minutes === 1 ? 'minute' : 'minutes'));
                }
                remaining = remaining.replace(/\d+m/, '');
            }
            
            // Extract seconds (only show if less than 1 hour)
            const secondMatch = remaining.match(/(\d+(?:\.\d+)?)s/);
            if (secondMatch && result.length === 0) {
                const seconds = Math.floor(parseFloat(secondMatch[1]));
                if (seconds > 0) {
                    result.push(seconds + ' ' + (seconds === 1 ? 'second' : 'seconds'));
                }
            }
            
            return result.length > 0 ? result.slice(0, 2).join(', ') : 'Just started';
        }

        // Update dashboard UI with data
        function updateDashboardUI(data) {
            // Update metrics - map from actual API response structure
            const activeSessions = document.getElementById('activeSessions');
            const totalSessions = document.getElementById('totalSessions');
            const uptime = document.getElementById('uptime');
            const systemStatus = document.getElementById('systemStatus');

            // Map from go-chariot API response structure
            if (activeSessions) activeSessions.textContent = (data.session_stats && data.session_stats.active_count) || 0;
            if (totalSessions) totalSessions.textContent = (data.session_stats && data.session_stats.active_count) || 0; // Using active_count as total for now
            if (uptime) uptime.textContent = formatUptime((data.server_status && data.server_status.uptime) || 'Unknown');
            if (systemStatus) systemStatus.textContent = (data.server_status && data.server_status.status) || 'Unknown';

            // Update sessions table
            const tbody = document.getElementById('sessionsTableBody');
            if (tbody) {
                tbody.innerHTML = '';
                
                console.log('Active sessions data:', data.active_sessions);
                console.log('Active sessions length:', data.active_sessions ? data.active_sessions.length : 'undefined');

                if (data.active_sessions && data.active_sessions.length > 0) {
                    const fmt = (iso) => {
                        if (!iso) return 'Unknown';
                        const d = new Date(iso);
                        if (isNaN(d)) return 'Unknown';
                        return d.toLocaleString(undefined, { year: 'numeric', month: 'short', day: '2-digit', hour: 'numeric', minute: '2-digit' });
                    };

                    data.active_sessions.forEach(session => {
                        const row = document.createElement('tr');
                        row.style.borderBottom = '1px solid #3e3e42';
                        const sid = session.session_id || session.sessionId || session.id || 'Unknown';
                        const status = session.status || ((session.expires_at && new Date(session.expires_at) > new Date()) ? 'active' : 'expired');
                        row.innerHTML = 
                            '<td style="padding: 12px;">' + escapeHtml(session.username || session.user_id || 'Unknown') + '</td>' +
                            '<td style="padding: 12px;">' + escapeHtml(sid !== 'Unknown' ? (sid.substring(0, 8) + '...') : 'Unknown') + '</td>' +
                            '<td style="padding: 12px;">' + escapeHtml(fmt(session.created)) + '</td>' +
                            '<td style="padding: 12px;">' + escapeHtml(fmt(session.last_access || session.lastSeen || session.last_accessed)) + '</td>' +
                            '<td style="padding: 12px; color: ' + (status === 'active' ? '#4ec9b0' : '#f44747') + ';">' + 
                            (status === 'active' ? 'Active' : 'Expired') + '</td>';
                        tbody.appendChild(row);
                    });
                } else {
                    const row = document.createElement('tr');
                    row.innerHTML = '<td colspan="5" style="text-align: center; padding: 20px; color: #888;">No sessions found</td>';
                    tbody.appendChild(row);
                }
            }

                        // Render listeners grid under sessions
            const existing = document.getElementById('listenersSection');
            if (!existing) {
                const container = document.querySelector('.dashboard-container');
                if (container) {
                                        const section = document.createElement('div');
                                        section.id = 'listenersSection';
                                        // Match the Sessions panel styling exactly
                                        section.className = 'sessions-section';
                                        section.style.cssText = 'background: #2d2d30; border: 1px solid #3e3e42; border-radius: 8px; padding: 20px; margin-top: 20px;';
                                        section.innerHTML = ''+
                                            '<div style="display:flex; align-items:center; justify-content:space-between; margin: 0 0 20px 0;">' +
                                                '<h3 style="margin: 0; color: #569cd6; font-size: 18px;">Listeners</h3>' +
                                                '<div style="display:flex; gap:8px;">' +
                                                    '<button id="createListenerBtn" class="toolbar-button">Create</button>' +
                                                    '<button id="deleteListenerBtn" class="toolbar-button">Delete</button>' +
                                                '</div>' +
                                            '</div>' +
                                            '<div style="overflow-x: auto;">' +
                                                '<table style="width: 100%; border-collapse: collapse; color: #d4d4d4;">' +
                                                    '<thead>' +
                                                        '<tr style="border-bottom: 1px solid #3e3e42;">' +
                                                            '<th style="text-align: left; padding: 12px; vertical-align: middle; width: 34px;">' +
                                                                '<div style="display:flex; align-items:center;">' +
                                                                    '<input type="checkbox" id="listenersSelectAll" style="margin:0;"/>' +
                                                                '</div>' +
                                                            '</th>' +
                                                            '<th style="text-align: left; padding: 12px; color: #569cd6;">Name</th>' +
                                                            '<th style="text-align: left; padding: 12px; color: #569cd6;">Status</th>' +
                                                            '<th style="text-align: left; padding: 12px; color: #569cd6;">Auto Start</th>' +
                                                            '<th style="text-align: left; padding: 12px; color: #569cd6;">Health</th>' +
                                                            '<th style="text-align: left; padding: 12px; color: #569cd6;">Actions</th>' +
                                                        '</tr>' +
                                                    '</thead>' +
                                                    '<tbody id="listenersTableBody">' +
                                                    '</tbody>' +
                                                '</table>' +
                                            '</div>' +
                                                // Delete confirm modal
                                                '<div id="listenerDeleteConfirmOverlay" style="display:none; position:fixed; inset:0; background:rgba(0,0,0,0.5); z-index:10000; align-items:center; justify-content:center;">'+
                                                    '<div style="background:#252526; color:#d4d4d4; border:1px solid #3e3e42; border-radius:6px; width:420px; max-width:90vw; box-shadow:0 6px 18px rgba(0,0,0,0.4);">'+
                                                        '<div style="padding:14px 16px; border-bottom:1px solid #3e3e42; font-weight:600;">Confirm Delete</div>'+ 
                                                        '<div style="padding:16px;">Delete selected Listeners?</div>'+ 
                                                        '<div style="padding:12px 16px; border-top:1px solid #3e3e42; display:flex; justify-content:flex-end; gap:8px;">'+ 
                                                            '<button id="listenerDeleteCancel" class="toolbar-button">Cancel</button>'+ 
                                                            '<button id="listenerDeleteOk" class="toolbar-button" style="background-color:#dc3545;">OK</button>'+ 
                                                        '</div>'+ 
                                                    '</div>'+ 
                                                '</div>'+
                                                // Modal root
                                                '<div id="listenerModalOverlay" style="display:none; position:fixed; inset:0; background:rgba(0,0,0,0.5); z-index:9999; align-items:center; justify-content:center;">'+
                                                    '<div id="listenerModal" style="background:#252526; color:#d4d4d4; border:1px solid #3e3e42; border-radius:6px; width:520px; max-width:90vw; box-shadow:0 6px 18px rgba(0,0,0,0.4);">'+
                                                        '<div style="padding:12px 16px; border-bottom:1px solid #3e3e42; display:flex; justify-content:space-between; align-items:center;">'+
                                                            '<div id="listenerModalTitle" style="font-weight:600;">Create Listener</div>'+
                                                            '<button id="listenerModalClose" class="toolbar-button">Close</button>'+ 
                                                        '</div>'+ 
                                                        '<div style="padding:16px; display:flex; flex-direction:column; gap:10px;">'+
                                                            '<label style="display:flex; flex-direction:column; gap:4px;">Name<input id="listenerName" placeholder="orders-listener" style="padding:8px; background:#1e1e1e; color:#d4d4d4; border:1px solid #3e3e42; border-radius:4px;"/></label>'+ 
                                                            '<label style="display:flex; flex-direction:column; gap:4px;">On Start<select id="listenerOnStartSelect" style="padding:8px; background:#1e1e1e; color:#d4d4d4; border:1px solid #3e3e42; border-radius:4px;"></select></label>'+ 
                                                            '<label style="display:flex; flex-direction:column; gap:4px;">On Exit<select id="listenerOnExitSelect" style="padding:8px; background:#1e1e1e; color:#d4d4d4; border:1px solid #3e3e42; border-radius:4px;"></select></label>'+ 
                                                            '<label style="display:flex; align-items:center; gap:8px;"><input type="checkbox" id="listenerAutoStart"/> Auto Start</label>'+ 
                                                        '</div>'+ 
                                                        '<div style="padding:12px 16px; border-top:1px solid #3e3e42; display:flex; justify-content:flex-end; gap:8px;">'+ 
                                                            '<button id="listenerModalCancel" class="toolbar-button">Cancel</button>'+ 
                                                            '<button id="listenerModalSave" class="toolbar-button">Save</button>'+ 
                                                        '</div>'+ 
                                                    '</div>'+ 
                                                '</div>';
                                        container.appendChild(section);
                                        // Bind all Listeners panel handlers once elements exist
                                        bindListenersPanelHandlers();
                }
            }
            const lbody = document.getElementById('listenersTableBody');
            if (lbody) {
                lbody.innerHTML = '';
                const listeners = (data && data.listeners) || [];
                if (listeners.length === 0) {
                    const row = document.createElement('tr');
                    row.innerHTML = '<td colspan="6" style="text-align:center; padding:20px; color:#888;">No listeners</td>';
                    lbody.appendChild(row);
                } else {
                    listeners.forEach(ls => {
                        const row = document.createElement('tr');
                        row.style.borderBottom = '1px solid #3e3e42';
                        const status = ls.status || 'stopped';
                        const health = ls.is_healthy ? 'Healthy' : 'Unhealthy';
                        const isChecked = selectedListeners.has(ls.name || '');
                        row.innerHTML =
                            '<td style="padding:12px; width:34px; vertical-align:middle;">' +
                                '<div style="display:flex; align-items:center;">' +
                                    '<input type="checkbox" class="listenerRowChk" data-name="' + (ls.name || '') + '" style="margin:0;" ' + (isChecked ? 'checked' : '') + '/>' +
                                '</div>' +
                            '</td>'+
                            '<td style="padding:12px;">' + escapeHtml(ls.name || '') + '</td>'+
                            '<td style="padding:12px;">' + '<span style="color:' + (status==='running'?'#4ec9b0':'#f44747') + '">' + status + '</span></td>'+
                            '<td style="padding:12px;">' + (ls.auto_start ? 'Yes' : 'No') + '</td>'+ 
                            '<td style="padding:12px;">' + health + '</td>'+ 
                            '<td style="padding:12px;">' +
                                '<button class="toolbar-button" data-act="start">Start</button> ' +
                                '<button class="toolbar-button" data-act="stop">Exit</button>' +
                            '</td>';
                        // Wire actions
                        setTimeout(() => {
                            const startBtn = row.querySelector('button[data-act="start"]');
                            const stopBtn = row.querySelector('button[data-act="stop"]');
                            const chk = row.querySelector('input.listenerRowChk');
                            if (chk) {
                                chk.addEventListener('change', (ev) => {
                                    const name = chk.getAttribute('data-name') || '';
                                    if (chk.checked) selectedListeners.add(name); else selectedListeners.delete(name);
                                    // Update header checkbox tri-state
                                    updateListenersHeaderCheckboxState(listeners);
                                });
                            }
                            if (startBtn) startBtn.onclick = async () => {
                                const resp = await fetch('/charioteer/api/listener/start?name=' + encodeURIComponent(ls.name), { method:'POST', headers: getAuthHeaders() });
                                if (!resp.ok) { const t = await resp.text(); return alert('Start failed: ' + t); }
                                fetchAndUpdateDashboard();
                            };
                            if (stopBtn) stopBtn.onclick = async () => {
                                const resp = await fetch('/charioteer/api/listener/stop?name=' + encodeURIComponent(ls.name), { method:'POST', headers: getAuthHeaders() });
                                if (!resp.ok) { const t = await resp.text(); return alert('Stop failed: ' + t); }
                                fetchAndUpdateDashboard();
                            };
                        }, 0);
                        lbody.appendChild(row);
                    });
                    // Update header checkbox based on selection
                    updateListenersHeaderCheckboxState(listeners);
                }
            }
        }

        function updateListenersHeaderCheckboxState(listeners) {
            const selAll = document.getElementById('listenersSelectAll');
            if (!selAll) return;
            const names = (listeners || []).map(ls => ls.name || '');
            const allSelected = names.length > 0 && names.every(n => selectedListeners.has(n));
            const noneSelected = names.every(n => !selectedListeners.has(n));
            selAll.indeterminate = !allSelected && !noneSelected;
            selAll.checked = allSelected;
        }

        function bindListenersPanelHandlers() {
            const openModal = () => { const o = document.getElementById('listenerModalOverlay'); if (o) { o.style.display='flex'; }};
            const closeModal = () => { const o = document.getElementById('listenerModalOverlay'); if (o) { o.style.display='none'; }};
            const createBtn = document.getElementById('createListenerBtn');
            const deleteBtn = document.getElementById('deleteListenerBtn');
            const modalClose = document.getElementById('listenerModalClose');
            const modalCancel = document.getElementById('listenerModalCancel');
            const modalSave = document.getElementById('listenerModalSave');
            const selAll = document.getElementById('listenersSelectAll');
            const delOverlay = document.getElementById('listenerDeleteConfirmOverlay');
            const delCancel = document.getElementById('listenerDeleteCancel');
            const delOk = document.getElementById('listenerDeleteOk');

            if (modalClose) modalClose.onclick = closeModal;
            if (modalCancel) modalCancel.onclick = closeModal;
            if (modalSave) modalSave.onclick = async () => {
                const name = (document.getElementById('listenerName').value || '').trim();
                const onStartSelect = document.getElementById('listenerOnStartSelect');
                const onExitSelect = document.getElementById('listenerOnExitSelect');
                const autoStart = !!(document.getElementById('listenerAutoStart') && document.getElementById('listenerAutoStart').checked);
                const onStart = onStartSelect ? (onStartSelect.value || '').trim() : '';
                const onExit = onExitSelect ? (onExitSelect.value || '').trim() : '';
                if (!name) { alert('Name is required'); return; }
                const resp = await fetch('/charioteer/api/listeners', {
                    method: 'POST',
                    headers: Object.assign({'Content-Type':'application/json'}, getAuthHeaders()),
                    body: JSON.stringify({ name, on_start: onStart, on_exit: onExit, auto_start: autoStart })
                });
                if (!resp.ok) { const t = await resp.text(); alert('Create failed: ' + t); return; }
                closeModal();
                fetchAndUpdateDashboard();
            };

            if (createBtn) createBtn.onclick = async () => {
                const t = document.getElementById('listenerModalTitle'); if (t) t.textContent = 'Create Listener';
                const fields = ['listenerName'];
                fields.forEach(id => { const el = document.getElementById(id); if (el) el.value = ''; });
                // Populate dropdowns from Files list
                try {
                    const url = getAPIPath('/api/files?folder=' + encodeURIComponent(CHARIOT_FILES_FOLDER));
                    const response = await fetch(url, { headers: getAuthHeaders() });
                    if (response.ok) {
                        const result = await response.json();
                        const files = (result && result.result === 'OK') ? result.data : [];
                        const startSel = document.getElementById('listenerOnStartSelect');
                        const exitSel = document.getElementById('listenerOnExitSelect');
                        if (startSel) { startSel.innerHTML = '<option value="">Select a file...</option>'; }
                        if (exitSel) { exitSel.innerHTML = '<option value="">Select a file...</option>'; }
                        (files || []).forEach(file => {
                            const opt1 = document.createElement('option'); opt1.value = file; opt1.textContent = file; if (startSel) startSel.appendChild(opt1);
                            const opt2 = document.createElement('option'); opt2.value = file; opt2.textContent = file; if (exitSel) exitSel.appendChild(opt2);
                        });
                    }
                } catch (e) { /* ignore */ }
                openModal();
            };
            if (deleteBtn) deleteBtn.onclick = () => {
                const names = Array.from(selectedListeners);
                if (names.length === 0) { alert('Select one or more listeners to delete.'); return; }
                if (delOverlay) delOverlay.style.display = 'flex';
            };
            if (delCancel) delCancel.onclick = () => { if (delOverlay) delOverlay.style.display = 'none'; };
            if (delOk) delOk.onclick = async () => {
                const names = Array.from(selectedListeners);
                if (names.length === 0) { if (delOverlay) delOverlay.style.display = 'none'; return; }
                // Optimistically remove from UI first
                const lbody = document.getElementById('listenersTableBody');
                if (lbody) {
                    names.forEach(n => {
                        const row = lbody.querySelector('input.listenerRowChk[data-name="' + CSS.escape(n) + '"]');
                        if (row) {
                            const tr = row.closest('tr');
                            if (tr) tr.remove();
                        }
                    });
                }
                // Call backend to delete from registry JSON
                for (const n of names) {
                    const resp = await fetch('/charioteer/api/listener/delete?name=' + encodeURIComponent(n), { method: 'POST', headers: getAuthHeaders() });
                    if (!resp.ok) {
                        const t = await resp.text();
                        alert('Delete failed for ' + n + ': ' + t);
                        if (delOverlay) delOverlay.style.display = 'none';
                        return;
                    }
                    selectedListeners.delete(n);
                }
                if (delOverlay) delOverlay.style.display = 'none';
                // Refresh to reflect backend state
                fetchAndUpdateDashboard();
            };
            if (selAll) selAll.onclick = () => {
                const lbody = document.getElementById('listenersTableBody');
                const checks = lbody ? Array.from(lbody.querySelectorAll('.listenerRowChk')) : [];
                if (selAll.checked) {
                    checks.forEach(chk => { chk.checked = true; selectedListeners.add(chk.getAttribute('data-name') || ''); });
                } else {
                    checks.forEach(chk => { chk.checked = false; selectedListeners.delete(chk.getAttribute('data-name') || ''); });
                }
                // Update indeterminate explicitly
                selAll.indeterminate = false;
            };
        }

        function escapeHtml(text) {
            const div = document.createElement('div');
            div.textContent = text;
            return div.innerHTML;
        }

    </script>
</body>
</html>`

const dashboardTemplate = `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Charioteer Dashboard</title>
    <style>
        body { 
            margin: 0; 
            padding: 0; 
            font-family: 'Segoe UI', Tahoma, Geneva, Verdana, sans-serif;
            background-color: #1e1e1e;
            color: #d4d4d4;
        }
        .dashboard-container {
            padding: 20px;
            max-width: 1200px;
            margin: 0 auto;
        }
        .dashboard-header {
            text-align: center;
            margin-bottom: 30px;
        }
        .dashboard-header h1 {
            color: #569cd6;
            margin: 0;
        }
        .metrics-grid {
            display: grid;
            grid-template-columns: repeat(auto-fit, minmax(300px, 1fr));
            gap: 20px;
            margin-bottom: 30px;
        }
        .metric-card {
            background: #2d2d30;
            border: 1px solid #3e3e42;
            border-radius: 8px;
            padding: 20px;
        }
        .metric-card h3 {
            margin: 0 0 15px 0;
            color: #569cd6;
            font-size: 18px;
        }
        .metric-value {
            font-size: 24px;
            font-weight: bold;
            color: #4ec9b0;
            margin-bottom: 10px;
        }
        .metric-label {
            color: #cccccc;
            font-size: 14px;
        }
        .sessions-table {
            background: #2d2d30;
            border: 1px solid #3e3e42;
            border-radius: 8px;
            overflow: hidden;
        }
        .sessions-table h3 {
            margin: 0;
            padding: 20px;
            background: #383838;
            color: #569cd6;
        }
        .table-container {
            max-height: 400px;
            overflow-y: auto;
        }
        table {
            width: 100%;
            border-collapse: collapse;
        }
        th, td {
            padding: 12px;
            text-align: left;
            border-bottom: 1px solid #3e3e42;
        }
        th {
            background: #383838;
            color: #cccccc;
            font-weight: 600;
        }
        td {
            color: #d4d4d4;
        }
        .status-active {
            color: #4ec9b0;
        }
        .status-expired {
            color: #f48771;
        }
        .refresh-button {
            background: #0e639c;
            color: white;
            border: none;
            padding: 10px 20px;
            border-radius: 4px;
            cursor: pointer;
            font-size: 14px;
            margin-bottom: 20px;
        }
        .refresh-button:hover {
            background: #1177bb;
        }
        .error-message {
            background: #5a1d1d;
            border: 1px solid #be1100;
            color: #f48771;
            padding: 15px;
            border-radius: 4px;
            margin-bottom: 20px;
        }
        .loading {
            text-align: center;
            color: #569cd6;
            font-size: 18px;
            margin: 40px 0;
        }
    </style>
</head>
<body>
    <div class="dashboard-container">
        <div class="dashboard-header">
            <button class="refresh-button" onclick="refreshDashboard()">🔄 Refresh</button>
        </div>
        
        <div id="errorMessage" class="error-message" style="display: none;"></div>
        <div id="loading" class="loading">Loading dashboard data...</div>
        
        <div id="dashboardContent" style="display: none;">
            <div class="metrics-grid">
                <div class="metric-card">
                    <h3>Active Sessions</h3>
                    <div class="metric-value" id="activeSessions">-</div>
                    <div class="metric-label">Currently logged in users</div>
                </div>
                <div class="metric-card">
                    <h3>Total Sessions</h3>
                    <div class="metric-value" id="totalSessions">-</div>
                    <div class="metric-label">All sessions (active + expired)</div>
                </div>
                <div class="metric-card">
                    <h3>Server Uptime</h3>
                    <div class="metric-value" id="uptime">-</div>
                    <div class="metric-label">Since last restart</div>
                </div>
                <div class="metric-card">
                    <h3>System Status</h3>
                    <div class="metric-value" id="systemStatus">-</div>
                    <div class="metric-label">Current system state</div>
                </div>
            </div>
            
            <div class="sessions-table">
                <h3>Active Sessions</h3>
                <div class="table-container">
                    <table>
                        <thead>
                            <tr>
                                <th>Username</th>
                                <th>Session ID</th>
                                <th>Created</th>
                                <th>Last Access</th>
                                <th>Status</th>
                            </tr>
                        </thead>
                        <tbody id="sessionsTableBody">
                            <tr>
                                <td colspan="5" style="text-align: center;">Loading sessions...</td>
                            </tr>
                        </tbody>
                    </table>
                </div>
            </div>
        </div>
    </div>

    <script>
        const BACKEND_URL = '{{.BackendURL}}';
        let refreshInterval;
        let authToken = null;

        // Extract token from URL parameter
        function getTokenFromURL() {
            const urlParams = new URLSearchParams(window.location.search);
            return urlParams.get('token');
        }

        // Initialize auth token
        authToken = getTokenFromURL();

        async function fetchDashboardData() {
            try {
                const headers = {
                    'Content-Type': 'application/json'
                };
                
                // Add authorization header if we have a token
                if (authToken) {
                    headers['Authorization'] = authToken;
                }

                const response = await fetch('/charioteer/api/dashboard/status', {
                    method: 'GET',
                    headers: headers
                });

                if (!response.ok) {
                    throw new Error('Failed to fetch dashboard data: ' + response.statusText);
                }

                const result = await response.json();
                if (result.result !== 'OK') {
                    throw new Error('Dashboard API error: ' + (result.data || 'Unknown error'));
                }

                return result.data;
            } catch (error) {
                console.error('Error fetching dashboard data:', error);
                throw error;
            }
        }

        function updateDashboard(data) {
            // Update metrics
            document.getElementById('activeSessions').textContent = data.activeSessions || 0;
            document.getElementById('totalSessions').textContent = data.totalSessions || 0;
            document.getElementById('uptime').textContent = data.uptime || 'Unknown';
            document.getElementById('systemStatus').textContent = data.systemStatus || 'Unknown';

            // Update sessions table
            const tbody = document.getElementById('sessionsTableBody');
            tbody.innerHTML = '';

            if (data.sessions && data.sessions.length > 0) {
                data.sessions.forEach(session => {
                    const row = document.createElement('tr');
                    row.innerHTML = '<td>' + escapeHtml(session.username || 'Unknown') + '</td>' +
                                   '<td>' + escapeHtml(session.sessionId ? session.sessionId.substring(0, 8) + '...' : 'Unknown') + '</td>' +
                                   '<td>' + escapeHtml(session.created || 'Unknown') + '</td>' +
                                   '<td>' + escapeHtml(session.lastAccess || 'Unknown') + '</td>' +
                                   '<td class="' + (session.active ? 'status-active' : 'status-expired') + '">' + 
                                   (session.active ? 'Active' : 'Expired') + '</td>';
                    tbody.appendChild(row);
                });
            } else {
                const row = document.createElement('tr');
                row.innerHTML = '<td colspan="5" style="text-align: center;">No sessions found</td>';
                tbody.appendChild(row);
            }
        }

        function escapeHtml(text) {
            const div = document.createElement('div');
            div.textContent = text;
            return div.innerHTML;
        }

        function showError(message) {
            const errorDiv = document.getElementById('errorMessage');
            errorDiv.textContent = message;
            errorDiv.style.display = 'block';
            document.getElementById('loading').style.display = 'none';
            document.getElementById('dashboardContent').style.display = 'none';
        }

        function hideError() {
            document.getElementById('errorMessage').style.display = 'none';
        }

        async function refreshDashboard() {
            try {
                hideError();
                document.getElementById('loading').style.display = 'block';
                document.getElementById('dashboardContent').style.display = 'none';

                const data = await fetchDashboardData();
                updateDashboard(data);

                document.getElementById('loading').style.display = 'none';
                document.getElementById('dashboardContent').style.display = 'block';
            } catch (error) {
                showError('Failed to load dashboard data: ' + error.message);
            }
        }

        // Initialize dashboard
        document.addEventListener('DOMContentLoaded', function() {
            refreshDashboard();
            // Refresh every 30 seconds
            refreshInterval = setInterval(refreshDashboard, 30000);
        });

        // Cleanup interval when page unloads
        window.addEventListener('beforeunload', function() {
            if (refreshInterval) {
                clearInterval(refreshInterval);
            }
        });
    </script>
</body>
</html>`

type EditorData struct {
	InitialCode string
}

type DashboardData struct {
	BackendURL string
}

func editorHandler(w http.ResponseWriter, r *http.Request) {
	// Set proper content type
	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	// Parse template
	tmpl, err := template.New("editor").Parse(editorTemplate)
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
	tmpl, err := template.New("dashboard").Parse(dashboardTemplate)
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

	// Protected routes -- local file operations
	http.HandleFunc("/api/files", authMiddleware(listFilesHandler))
	http.HandleFunc("/api/file", authMiddleware(getFileHandler))
	http.HandleFunc("/api/save", authMiddleware(saveFileHandler))
	http.HandleFunc("/api/rename", authMiddleware(renameFileHandler))
	http.HandleFunc("/api/delete", authMiddleware(deleteFileHandler))
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

	// Prefixed API routes for proxy path support
	http.HandleFunc("/charioteer/api/files", authMiddleware(listFilesHandler))
	http.HandleFunc("/charioteer/api/file", authMiddleware(getFileHandler))
	http.HandleFunc("/charioteer/api/save", authMiddleware(saveFileHandler))
	http.HandleFunc("/charioteer/api/rename", authMiddleware(renameFileHandler))
	http.HandleFunc("/charioteer/api/delete", authMiddleware(deleteFileHandler))
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
	http.HandleFunc("/charioteer/api/listener/delete", authMiddleware(listenersDeleteHandler))
	http.HandleFunc("/charioteer/api/listener/start", authMiddleware(listenersStartHandler))
	http.HandleFunc("/charioteer/api/listener/stop", authMiddleware(listenersStopHandler))
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
