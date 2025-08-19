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
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
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
	json.NewEncoder(w).Encode(ResultJSON{
		Result: "OK",
		Data:   data,
	})
}

// sendError sends an error ResultJSON response
func sendError(w http.ResponseWriter, statusCode int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(ResultJSON{
		Result: "ERROR",
		Data:   message,
	})
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
        .monaco-editor .token.keyword.chariot.node { color: #a29bfe !important; }
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
                <button id="runButton" class="run-button" disabled>▶ Run</button>
                
                <div class="auth-section">
                    <div id="loginSection">
                        <input type="text" id="usernameInput" placeholder="Username" class="auth-input">
                        <input type="password" id="passwordInput" placeholder="Password" class="auth-input">
                        <button id="loginButton" class="auth-button">Login</button>
                    </div>
                    <div id="loggedInSection" style="display: none;">
                        <span class="user-info">Logged in as: <span id="currentUserSpan"></span></span>
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
            </div>
        </div>
    </div>

    <script src="https://cdn.jsdelivr.net/npm/monaco-editor@0.45.0/min/vs/loader.js"></script>
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
            [/\b(addTo|array|lastIndex|removeAt|reverse|setAt|slice)\b(?=\s*\()/, 'keyword.chariot.array'],
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
            [/\b(addChild|childCount|clear|cloneNode|create|csvNode|findByName|firstChild|getAttribute|getChildAt|getChildByName|getDepth|getLevel|getName|getParent|getPath|getRoot|getSiblings|getText|hasAttribute|isLeaf|isRoot|jsonNode|lastChild|list|mapNode|nodeToString|queryNode|removeAttribute|removeChild|setAttribute|setAttributes|setChildByName|setName|setText|traverseNode|xmlNode|yamlNode)\b(?=\s*\()/, 'keyword.chariot.node'],
            [/\b(generateCreateTable|sqlBegin|sqlConnect|sqlClose|sqlCommit|sqlExecute|sqlListTables|sqlQuery|sqlRollback)\b(?=\s*\()/, 'keyword.chariot.sql'],
            [/\b(append|ascii|atPos|char|charAt|concat|digits|format|hasPrefix|hasSuffix|interpolate|join|lastPos|lower|occurs|padLeft|padRight|repeat|replace|right|split|sprintf|string|strlen|substr|substring|trim|trimLeft|trimRight|upper)\b(?=\s*\()/, 'keyword.chariot.string'],
            [/\b(exit|getEnv|hasEnv|listen|logPrint|platform|sleep|timeFormat|timestamp)\b(?=\s*\()/, 'keyword.chariot.system'],
            [/\b(newTree|treeFind|treeGetMetadata|treeLoad|treeLoadSecure|treeSave|treeSaveSecure|treeSearch||treeToYAML|treeToXML|treeValidateSecure|treeWalk)\b(?=\s*\()/, 'keyword.chariot.tree'],
            [/\b(boolean|call|declare|declareGlobal|deleteFunction|destroy|empty|exists|func|function|getFunction|getVariable|hasMeta|inspectRuntime|isNull|isNumeric|listFunctions|loadFunctions|mapValue|merge|offerVar|offerVariable|registerFunction|saveFunctions|setValue|setq|toBool|toMapValue|toNumber|toString|typeOf|valueOf)\b(?=\s*\()/, 'keyword.chariot.value'],
            [/\bfunction\b/, 'keyword.control.chariot'], // Always highlight 'function' as a keyword
            [/[a-zA-Z_$][\w$]*/, 'identifier'], 
        ];

        // Wait for DOM to be ready
        document.addEventListener('DOMContentLoaded', function() {
            initializeEditor();
        });
        
        // Pin Monaco to a specific version for stability
        require.config({ paths: { vs: 'https://cdn.jsdelivr.net/npm/monaco-editor@0.45.0/min/vs' } });
        
        let editor;
        let fileEditorContent = '';         // Last content loaded in Files tab
        let fileEditorFileName = '';        // Last file loaded in Files tab
        let functionEditorContent = '';     // Last content loaded in Function Library tab
        let functionEditorFunctionName = ''; // Last function loaded in Function Library tab
        let currentFileName = '';
        let currentTab = 'output';
        let isResizing = false;
        let authToken = null;
        let currentUser = null;
        let isFileModified = false;
        let originalContent = '';
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
                }
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
            const functionSelect = document.getElementById('functionSelect');
            const newFunctionButton = document.getElementById('newFunctionButton');
            const saveFunctionButton = document.getElementById('saveFunctionButton');
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
                        result.data.forEach(fn => {
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
        function initializeEventHandlers() {
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
            
            // Login functionality
            const loginButton = document.getElementById('loginButton');
            const logoutButton = document.getElementById('logoutButton');
            const usernameInput = document.getElementById('usernameInput');
            const passwordInput = document.getElementById('passwordInput');
            
            if (loginButton) {
                loginButton.addEventListener('click', login);
                console.log('DEBUG: Login button handler added');
            }
            
            if (logoutButton) {
                logoutButton.addEventListener('click', logout);
                console.log('DEBUG: Logout button handler added');
            }
            
            // Enter key in password field
            if (passwordInput) {
                passwordInput.addEventListener('keypress', function(e) {
                    if (e.key === 'Enter') {
                        login();
                    }
                });
            }
            
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
                runButton.addEventListener('click', runCode);
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
            
            // Keyboard shortcuts
            document.addEventListener('keydown', function(e) {
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
            console.log('DEBUG: Keyboard shortcuts added');
            // Toolbar tab switching
            const filesTab = document.getElementById('filesTab');
            const functionsTab = document.getElementById('functionsTab');
            const fileToolbar = document.getElementById('fileToolbar');
            const functionsToolbar = document.getElementById('functionsToolbar');

            if (filesTab && functionsTab && fileToolbar && functionsToolbar) {
                function showToolbar(selected) {
                    // Hide all toolbars
                    fileToolbar.classList.remove('active');
                    functionsToolbar.classList.remove('active');
                    // Remove active from both tabs
                    filesTab.classList.remove('active');
                    functionsTab.classList.remove('active');

                    if (selected === 'files') {
                        // Save current function editor state
                        if (currentTab === 'functions') {
                            functionEditorContent = editor.getValue();
                            functionEditorFunctionName = document.getElementById('functionSelect').value;
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
                    } else {
                        // Save current file editor state
                        if (currentTab === 'files') {
                            fileEditorContent = editor.getValue();
                            fileEditorFileName = currentFileName;
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
                    }
                }
                filesTab.addEventListener('click', function() {
                    showToolbar('files');
                });
                functionsTab.addEventListener('click', function() {
                    showToolbar('functions');
                });
            }
        }        
        // Tab switching
        function switchTab(tabName) {
            document.querySelectorAll('.tab').forEach(tab => {
                tab.classList.remove('active');
            });
            document.querySelector('[data-tab="' + tabName + '"]').classList.add('active');
            
            currentTab = tabName;
            updateTabContent();
        }
        
        // Update tab content based on current tab
        function updateTabContent() {
            const content = document.getElementById('tabContent');
            // Content will be updated by specific functions (showOutput, showProblems, etc.)
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

        // Escape HTML for safe display
        function escapeHtml(text) {
            const div = document.createElement('div');
            div.textContent = text;
            return div.innerHTML;
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

    </script>
</body>
</html>`

type EditorData struct {
	InitialCode string
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
	w.Write(*responseBody)
}

// Add authentication middleware
func authMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		token := r.Header.Get("Authorization")
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

	// Forward the response back to the client directly
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(resp.StatusCode)
	w.Write(responseBody)
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

	// Forward the response back to the client directly
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(resp.StatusCode)
	w.Write(responseBody)
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
	io.Copy(w, resp.Body)
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
	http.HandleFunc("/charioteer/login", loginHandler)   // Implement loginHandler to handle login requests
	http.HandleFunc("/charioteer/logout", logoutHandler) // Implement logoutHandler to handle logout requests

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
