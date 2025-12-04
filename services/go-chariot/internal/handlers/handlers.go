package handlers

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/bhouse1273/chariot-ecosystem/services/go-chariot/chariot"
	cfg "github.com/bhouse1273/chariot-ecosystem/services/go-chariot/configs"
	"github.com/bhouse1273/chariot-ecosystem/services/go-chariot/internal/listeners"
	"go.uber.org/zap"

	"github.com/labstack/echo/v4"
)

// Add this to your handlers.go or appropriate file
type ResultJSON struct {
	Result string      `json:"result"`
	Data   interface{} `json:"data"`
}

// Handlers holds all HTTP handlers and their dependencies
type Handlers struct {
	sessionManager   *chariot.SessionManager
	bootstrapRuntime *chariot.Runtime   // Global runtime for system operations
	startTime        time.Time          // Service start time for uptime metrics
	bootstrapLoaded  bool               // Indicates whether bootstrap script loaded successfully
	listenerManager  *listeners.Manager // Manages configured listeners
	execManager      *ExecutionManager  // Manages async script executions with log streaming
}

// NewHandlers creates a new Handlers instance with dependencies
func NewHandlers(sessionManager *chariot.SessionManager) *Handlers {
	// Create a bootstrap runtime for system operations like user authentication
	bootstrapRuntime := chariot.NewRuntime()

	// Register all standard functions in the bootstrap runtime
	chariot.RegisterAll(bootstrapRuntime)

	// Load bootstrap script that includes usersAgent
	var bootstrapLoaded bool
	if cfg.ChariotConfig.Bootstrap != "" {
		fullPath, err := chariot.GetSecureFilePath(cfg.ChariotConfig.Bootstrap, "data")
		if err != nil {
			cfg.ChariotLogger.Error("Failed to get secure file path for bootstrap", zap.String("bootstrap", cfg.ChariotConfig.Bootstrap), zap.Error(err))
		} else {
			cfg.ChariotLogger.Info("Loading bootstrap script (handlers)", zap.String("path", fullPath))
			content, err := os.ReadFile(fullPath)
			if err != nil {
				cfg.ChariotLogger.Error("Failed to read bootstrap script", zap.String("path", fullPath), zap.Error(err))
			} else {
				if _, err := bootstrapRuntime.ExecProgram(string(content)); err != nil {
					cfg.ChariotLogger.Error("Failed to execute bootstrap script in handlers", zap.String("path", fullPath), zap.Error(err))
				} else {
					cfg.ChariotLogger.Info("Bootstrap script executed (handlers)", zap.String("path", fullPath))
					bootstrapLoaded = true
				}
			}
		}
	} else {
		cfg.ChariotLogger.Warn("Bootstrap filename not configured; skipping handlers bootstrap")
	}

	// Pass bootstrap runtime to session manager so new sessions inherit globals
	sessionManager.SetBootstrapRuntime(bootstrapRuntime)

	// Initialize a listeners manager using the bootstrap runtime
	lman := listeners.NewManager(bootstrapRuntime)
	if err := lman.Load(); err != nil {
		cfg.ChariotLogger.Warn("Failed to load listeners registry", zap.Error(err))
	}
	// In REST mode, do NOT auto-start listeners. Headless mode is responsible for starting
	// listeners with auto_start=true (handled in cmd/main.go).

	return &Handlers{
		sessionManager:   sessionManager,
		bootstrapRuntime: bootstrapRuntime,
		startTime:        time.Now(),
		bootstrapLoaded:  bootstrapLoaded,
		listenerManager:  lman,
		execManager:      NewExecutionManager(),
	}
}

// Listener APIs
type listenerCreateReq struct {
	Name      string `json:"name"`
	Script    string `json:"script"`
	OnStart   string `json:"on_start"`
	OnExit    string `json:"on_exit"`
	AutoStart bool   `json:"auto_start"`
}

func (h *Handlers) ListListeners(c echo.Context) error {
	ls := h.listenerManager.List()
	return c.JSON(http.StatusOK, ResultJSON{Result: "OK", Data: ls})
}

func (h *Handlers) CreateListener(c echo.Context) error {
	var req listenerCreateReq
	if err := c.Bind(&req); err != nil || req.Name == "" {
		return c.JSON(http.StatusBadRequest, ResultJSON{Result: "ERROR", Data: "invalid request"})
	}

	// Convert selected files to stdlib functions and set hook names
	toAdd := make(map[string]*chariot.FunctionValue)
	processFile := func(fname string) (string, error) {
		if fname == "" {
			return "", nil
		}
		base := filepath.Base(fname)
		name := strings.TrimSuffix(base, filepath.Ext(base))
		// Files are under data/files; secure resolve
		fullRel := filepath.Join("files", fname)
		fullPath, err := chariot.GetSecureFilePath(fullRel, "data")
		if err != nil {
			return "", fmt.Errorf("resolve file: %w", err)
		}
		content, err := os.ReadFile(fullPath)
		if err != nil {
			return "", fmt.Errorf("read file: %w", err)
		}
		if err := h.bootstrapRuntime.SaveFunction(name, string(content), ""); err != nil {
			return "", fmt.Errorf("parse function: %w", err)
		}
		if fn, ok := h.bootstrapRuntime.GetFunction(name); ok {
			toAdd[name] = fn
		} else {
			return "", fmt.Errorf("function not found after save: %s", name)
		}
		return name, nil
	}

	if newName, err := processFile(req.OnStart); err != nil {
		return c.JSON(http.StatusBadRequest, ResultJSON{Result: "ERROR", Data: fmt.Sprintf("on_start: %v", err)})
	} else if newName != "" {
		req.OnStart = newName
	}
	if newName, err := processFile(req.OnExit); err != nil {
		return c.JSON(http.StatusBadRequest, ResultJSON{Result: "ERROR", Data: fmt.Sprintf("on_exit: %v", err)})
	} else if newName != "" {
		req.OnExit = newName
	}

	if len(toAdd) > 0 {
		funcs := make(map[string]*chariot.FunctionValue)
		if cfg.ChariotConfig.FunctionLib != "" {
			if existing, err := chariot.LoadFunctionsFromFile(cfg.ChariotConfig.FunctionLib); err == nil {
				for k, v := range existing {
					funcs[k] = v
				}
			}
			for k, v := range toAdd {
				funcs[k] = v
			}
			if err := chariot.SaveFunctionsToFile(funcs, cfg.ChariotConfig.FunctionLib); err != nil {
				return c.JSON(http.StatusInternalServerError, ResultJSON{Result: "ERROR", Data: fmt.Sprintf("save stdlib: %v", err)})
			}
			for name, fn := range toAdd {
				h.bootstrapRuntime.RegisterFunction(name, fn)
			}
		}
	}

	l, err := h.listenerManager.Create(req.Name, req.Script, req.OnStart, req.OnExit, req.AutoStart)
	if err != nil {
		return c.JSON(http.StatusBadRequest, ResultJSON{Result: "ERROR", Data: err.Error()})
	}
	return c.JSON(http.StatusOK, ResultJSON{Result: "OK", Data: l})
}

// SaveFunctionLibraryHandler saves multiple functions into the shared stdlib file
// Expects JSON: { "functions": { "name": { /* FunctionValue map form */ } } }
func (h *Handlers) SaveFunctionLibraryHandler(c echo.Context) error {
	// Parse body
	var req struct {
		Functions map[string]map[string]interface{} `json:"functions"`
	}
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, ResultJSON{Result: "ERROR", Data: "invalid request"})
	}
	if len(req.Functions) == 0 {
		return c.JSON(http.StatusBadRequest, ResultJSON{Result: "ERROR", Data: "no functions provided"})
	}
	// Merge with existing library (load, then overwrite keys)
	funcs := make(map[string]*chariot.FunctionValue)
	if cfg.ChariotConfig.FunctionLib != "" {
		if existing, err := chariot.LoadFunctionsFromFile(cfg.ChariotConfig.FunctionLib); err == nil {
			for k, v := range existing {
				funcs[k] = v
			}
		}
	}
	// Convert incoming maps to FunctionValue via deserializer
	for name, m := range req.Functions {
		if fv, err := chariot.MapToFunctionValue(m); err == nil {
			funcs[name] = fv
		} else {
			return c.JSON(http.StatusBadRequest, ResultJSON{Result: "ERROR", Data: fmt.Sprintf("invalid function '%s': %v", name, err)})
		}
	}
	// Save back to stdlib file
	if cfg.ChariotConfig.FunctionLib == "" {
		return c.JSON(http.StatusInternalServerError, ResultJSON{Result: "ERROR", Data: "function_lib not configured"})
	}
	if err := chariot.SaveFunctionsToFile(funcs, cfg.ChariotConfig.FunctionLib); err != nil {
		return c.JSON(http.StatusInternalServerError, ResultJSON{Result: "ERROR", Data: err.Error()})
	}
	// Also refresh bootstrap runtime registered functions for immediate availability
	for name, fn := range funcs {
		h.bootstrapRuntime.RegisterFunction(name, fn)
	}
	return c.JSON(http.StatusOK, ResultJSON{Result: "OK", Data: "library saved"})
}

func (h *Handlers) DeleteListener(c echo.Context) error {
	name := c.Param("name")
	if name == "" {
		return c.JSON(http.StatusBadRequest, ResultJSON{Result: "ERROR", Data: "missing name"})
	}
	if err := h.listenerManager.Delete(name); err != nil {
		return c.JSON(http.StatusBadRequest, ResultJSON{Result: "ERROR", Data: err.Error()})
	}
	return c.JSON(http.StatusOK, ResultJSON{Result: "OK", Data: map[string]string{"deleted": name}})
}

func (h *Handlers) StartListener(c echo.Context) error {
	name := c.Param("name")
	if name == "" {
		return c.JSON(http.StatusBadRequest, ResultJSON{Result: "ERROR", Data: "missing name"})
	}
	l, err := h.listenerManager.Start(name, cfg.ChariotConfig.Port)
	if err != nil {
		return c.JSON(http.StatusBadRequest, ResultJSON{Result: "ERROR", Data: err.Error()})
	}
	return c.JSON(http.StatusOK, ResultJSON{Result: "OK", Data: l})
}

func (h *Handlers) StopListener(c echo.Context) error {
	name := c.Param("name")
	if name == "" {
		return c.JSON(http.StatusBadRequest, ResultJSON{Result: "ERROR", Data: "missing name"})
	}
	l, err := h.listenerManager.Stop(name, cfg.ChariotConfig.Port)
	if err != nil {
		return c.JSON(http.StatusBadRequest, ResultJSON{Result: "ERROR", Data: err.Error()})
	}
	return c.JSON(http.StatusOK, ResultJSON{Result: "OK", Data: l})
}

func (h *Handlers) Execute(c echo.Context) error {
	// Incoming JSON: {"program": "your chariot code here", "filename": "optional.ch"}
	type Request struct {
		Program  string `json:"program"`
		Filename string `json:"filename,omitempty"`
	}
	var req Request
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, ResultJSON{
			Result: "ERROR",
			Data:   "Invalid request format",
		})
	}

	// Validate program field
	if req.Program == "" {
		return c.JSON(http.StatusBadRequest, ResultJSON{
			Result: "ERROR",
			Data:   "Missing program field",
		})
	}

	// Basic validation
	if len(req.Program) < 5 {
		return c.JSON(http.StatusBadRequest, ResultJSON{
			Result: "ERROR",
			Data:   "Program is too short",
		})
	}

	// Refactor program if function definition is detected
	// from function myFunc() { ... } to
	// (setq myFunc func() { ... })
	// (call myFunc, arg1, arg2, ...)
	if strings.HasPrefix(req.Program, "function") {
		// Remove any leading/trailing whitespace
		req.Program = strings.TrimSpace(req.Program)
		// Extract the name portion as the string between "function" and the first parenthesis
		parts := strings.SplitN(req.Program, "(", 2)
		if len(parts) < 2 {
			return c.JSON(http.StatusBadRequest, ResultJSON{
				Result: "ERROR",
				Data:   "Invalid function definition format",
			})
		}
		funcName := strings.TrimSpace(parts[0][len("function"):])
		if funcName == "" {
			return c.JSON(http.StatusBadRequest, ResultJSON{
				Result: "ERROR",
				Data:   "Function name cannot be empty",
			})
		}
		// Extract the parameters between the parens if any
		params := strings.TrimSpace(parts[1])
		if strings.HasPrefix(params, ")") {
			// No parameters, just an empty set of parentheses
			params = ""
		} else {
			// Look for first )
			paramsEnd := strings.Index(params, ")")
			if paramsEnd == -1 {
				return c.JSON(http.StatusBadRequest, ResultJSON{
					Result: "ERROR",
					Data:   "Invalid function parameters format",
				})
			}
			params = strings.TrimSpace(params[:paramsEnd])
			// Remove closing paren from Program using the already computed bodyStart
			bodyStart := strings.Index(req.Program, "{")
			req.Program = strings.TrimSpace(req.Program[bodyStart:])
		}
		// Extract the body -- the braces plus all lines betwee them
		bodyStart := strings.Index(req.Program, "{")
		bodyEnd := strings.LastIndex(req.Program, "}")
		if bodyStart == -1 || bodyEnd == -1 || bodyEnd <= bodyStart {
			return c.JSON(http.StatusBadRequest, ResultJSON{
				Result: "ERROR",
				Data:   "Invalid function body format",
			})
		}
		body := req.Program[bodyStart+1 : bodyEnd]
		// Remove function name from Program
		req.Program = strings.Replace(req.Program, funcName, "", 1)
		// Replace "function" with "func" for compatibility
		req.Program = strings.Replace(req.Program, "function", "func", 1)
		// Wrap the function definition in a setq, wrap setq in call, inject params after setq in call args
		req.Program = fmt.Sprintf("setq(%s, func(%s) {\n%s\n})\n", funcName, params, body)
		// Wrap the whole thing in a call -- we already removed funcName from Program
		req.Program = fmt.Sprintf("call(setq(%s, %s)\n", funcName, req.Program)
	}

	// Get session from context
	session := c.Get("session").(*chariot.Session)

	// Initialize debugger if not already present
	if session.Runtime.Debugger == nil {
		session.Runtime.Debugger = chariot.NewDebugger()
	}

	// Use provided filename or default to "main.ch"
	filename := req.Filename
	if filename == "" {
		filename = "main.ch"
	}

	// Log debugging info
	fmt.Printf("\n========== EXECUTION START ==========\n")
	fmt.Printf("DEBUG: Executing program with filename: %s\n", filename)
	fmt.Printf("DEBUG: Program length: %d characters\n", len(req.Program))
	fmt.Printf("DEBUG: Program content:\n%s\n", req.Program)
	fmt.Printf("DEBUG: --- END PROGRAM ---\n")
	if session.Runtime.Debugger != nil {
		breakpoints := session.Runtime.Debugger.GetBreakpoints()
		fmt.Printf("DEBUG: Active breakpoints: %d\n", len(breakpoints))
		for _, bp := range breakpoints {
			fmt.Printf("DEBUG:   - %s:%d\n", bp.File, bp.Line)
		}
	}
	fmt.Printf("=====================================\n\n")

	// Check if debugging is active (has breakpoints)
	hasBreakpoints := false
	if session.Runtime.Debugger != nil {
		breakpoints := session.Runtime.Debugger.GetBreakpoints()
		hasBreakpoints = len(breakpoints) > 0
	}

	// Don't use debug mode for system calls like inspectRuntime()
	isSystemCall := req.Program == "inspectRuntime()" || req.Program == "listFunctions()"

	// If debugging is active and this is user code, run in background and return immediately
	if hasBreakpoints && !isSystemCall {
		fmt.Printf("DEBUG: Running in background due to active breakpoints\n")
		go func() {
			val, err := session.Runtime.ExecProgramWithFilename(req.Program, filename)
			if err != nil {
				fmt.Printf("DEBUG: Execution error: %v\n", err)
				// Send error event
				if session.Runtime.Debugger != nil {
					session.Runtime.Debugger.SendEvent(chariot.DebugEvent{
						Type:    "error",
						Message: err.Error(),
					})
				}
				return
			}
			fmt.Printf("DEBUG: Execution completed successfully: %v\n", val)
			// Send completion event
			if session.Runtime.Debugger != nil {
				session.Runtime.Debugger.SendEvent(chariot.DebugEvent{
					Type: "stopped",
				})
			}
		}()

		return c.JSON(http.StatusOK, ResultJSON{
			Result: "OK",
			Data:   "Execution started in debug mode",
		})
	}

	// Normal synchronous execution when not debugging
	val, err := session.Runtime.ExecProgramWithFilename(req.Program, filename)
	if err != nil {
		return c.JSON(http.StatusBadRequest, ResultJSON{
			Result: "ERROR",
			Data:   fmt.Sprintf("Execution error: %v", err),
		})
	}

	// 3. Convert Chariot Value to proper JSON-serializable format
	result := convertValueToJSON(val)
	resultJSON := ResultJSON{
		Result: "OK",
		Data:   result,
	}
	return c.JSON(http.StatusOK, resultJSON)
}

func (h *Handlers) ListFunctions(c echo.Context) error {
	// Get the authenticated session
	session := c.Get("session").(*chariot.Session)

	// List functions from the session's runtime
	functions := session.Runtime.ListFunctions()

	// Convert to JSON-serializable format
	functionList := make([]string, 0, functions.Length())
	for i := 0; i < functions.Length(); i++ {
		functionList = append(functionList, string(functions.Get(i).(chariot.Str)))
	}

	return c.JSON(http.StatusOK, ResultJSON{
		Result: "OK",
		Data:   functionList,
	})
}

// ListGlobalVariables lists all global variables from the session's runtime
func (h *Handlers) ListGlobalVariables(c echo.Context) error {
	// Get the authenticated session
	session := c.Get("session").(*chariot.Session)

	// List global variables from the session's runtime
	globalVars := session.Runtime.ListGlobalVariables()

	// Convert to JSON-serializable format
	varMap := make(map[string]interface{})
	for name, value := range globalVars {
		varMap[name] = convertValueToJSON(value)
	}

	return c.JSON(http.StatusOK, ResultJSON{
		Result: "OK",
		Data:   varMap,
	})
}

// GetData retrieves user-specific data and available resources
func (h *Handlers) GetData(c echo.Context) error {
	// Get the authenticated session
	session := c.Get("session").(*chariot.Session)

	// Prepare response with user context data
	response := map[string]interface{}{
		"user": map[string]interface{}{
			"id":           session.UserID,
			"lastAccessed": session.LastAccessed,
		},
		"resources": make(map[string]interface{}),
	}

	// Add available templates
	templates, err := listAvailableTemplates(session)
	if err == nil {
		response["templates"] = templates
	}

	// Add session-specific resources
	sessionResources := make(map[string]interface{})
	for name := range session.Resources {
		// Just show resource names, not the actual resources
		sessionResources[name] = true
	}
	response["resources"] = sessionResources

	// Add recent execution results if stored in session
	if results, ok := session.GetData("recentResults"); ok {
		response["recentResults"] = results
	}

	return c.JSON(http.StatusOK, ResultJSON{
		Result: "OK",
		Data:   response,
	})
}

// SaveFunction handler - saves a user-defined function
func (h *Handlers) SaveFunctionHandler(c echo.Context) error {
	// Get the authenticated session
	session := c.Get("session").(*chariot.Session)

	// Parse the request body for function definition
	var req struct {
		Name            string `json:"name"`
		Code            string `json:"code"`
		FormattedSource string `json:"formatted_source"`
	}
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, ResultJSON{
			Result: "ERROR",
			Data:   "Invalid request format",
		})
	}

	// Validate function name and code
	if req.Name == "" || req.Code == "" {
		return c.JSON(http.StatusBadRequest, ResultJSON{
			Result: "ERROR",
			Data:   "Function name and code are required",
		})
	}

	// Save the function in the session's runtime
	if err := session.Runtime.SaveFunction(req.Name, req.Code, req.FormattedSource); err != nil {
		return c.JSON(http.StatusInternalServerError, ResultJSON{
			Result: "ERROR",
			Data:   fmt.Sprintf("Failed to save function: %v", err),
		})
	}

	return c.JSON(http.StatusOK, ResultJSON{
		Result: "OK",
		Data:   fmt.Sprintf("Function '%s' saved successfully", req.Name),
	})
}

// Login handler - creates a new session
func (h *Handlers) HandleLogin(c echo.Context) error {
	if c.Request().Method != "POST" {
		return c.JSON(http.StatusMethodNotAllowed, ResultJSON{
			Result: "ERROR",
			Data:   "Method not allowed",
		})
	}

	var username, password string

	// Check Content-Type to determine how to parse the request
	contentType := c.Request().Header.Get("Content-Type")

	if strings.Contains(contentType, "application/json") {
		// Parse JSON body
		var loginReq struct {
			Username string `json:"username"`
			Password string `json:"password"`
		}

		if err := c.Bind(&loginReq); err != nil {
			return c.JSON(http.StatusBadRequest, ResultJSON{
				Result: "ERROR",
				Data:   "Invalid JSON format",
			})
		}

		// Validate credentials
		if loginReq.Username == "" || loginReq.Password == "" {
			return c.JSON(http.StatusBadRequest, ResultJSON{
				Result: "ERROR",
				Data:   "Username and password required",
			})
		}
		username = loginReq.Username
		password = loginReq.Password

	} else {
		// Parse form data (existing behavior)
		if err := c.Request().ParseForm(); err != nil {
			return c.JSON(http.StatusBadRequest, ResultJSON{
				Result: "ERROR",
				Data:   "Unable to parse form data: " + err.Error(),
			})
		}

		username = c.Request().FormValue("username")
		password = c.Request().FormValue("password")
	}

	// Validate credentials
	if username == "" || password == "" {
		return c.JSON(http.StatusBadRequest, ResultJSON{
			Result: "ERROR",
			Data:   "Username and password required",
		})
	}

	// Verify credentials (implement your authentication logic)
	/*
		if !h.authenticateUser(username, password) {
			return c.JSON(http.StatusUnauthorized, ResultJSON{
				Result: "ERROR",
				Data:   "Invalid credentials",
			})
		}
	*/

	// Generate session token (use a proper token generator)
	token := generateSecureToken()

	// Create new session
	session := h.sessionManager.NewSession(username, cfg.ChariotLogger, token)
	session.Authenticated = true

	// Ensure user's sandbox directories exist
	if err := cfg.EnsureSandboxDirectories(username); err != nil {
		log.Printf("Warning: Failed to create sandbox directories for user %s: %v", username, err)
		// Don't fail login, just log the warning
	}

	// Success response
	return c.JSON(http.StatusOK, ResultJSON{
		Result: "OK",
		Data: map[string]string{
			"token": token,
			"user":  username,
		},
	})
}

// Logout handler - terminates the session
func (h *Handlers) HandleLogout(c echo.Context) error {
	token := c.Request().Header.Get("Authorization")
	if token == "" {
		return c.JSON(http.StatusBadRequest, ResultJSON{
			Result: "ERROR",
			Data:   "No authentication token provided",
		})
	}

	if err := h.sessionManager.EndSession(token); err != nil {
		return c.JSON(http.StatusNotFound, ResultJSON{
			Result: "ERROR",
			Data:   "Session not found",
		})
	}

	return c.JSON(http.StatusOK, ResultJSON{
		Result: "OK",
		Data: map[string]string{
			"message": "Successfully logged out",
		},
	})
}

// SessionProfile returns information about the authenticated session and storage scopes.
func (h *Handlers) SessionProfile(c echo.Context) error {
	sess, ok := c.Get("session").(*chariot.Session)
	if !ok || sess == nil {
		return c.JSON(http.StatusUnauthorized, ResultJSON{Result: "ERROR", Data: "session not found"})
	}
	username := sess.Username
	if username == "" {
		username = sess.UserID
	}
	// Auto-create sandbox directories if enabled
	if cfg.ChariotConfig.SandboxEnabled {
		if err := cfg.EnsureSandboxDirectories(username); err != nil {
			cfg.ChariotLogger.Warn("Failed to ensure sandbox directories", zap.String("username", username), zap.Error(err))
		}
	}
	profile := map[string]interface{}{
		"user_id":               sess.UserID,
		"username":              username,
		"sandbox_enabled":       cfg.ChariotConfig.SandboxEnabled,
		"sandbox_scope_default": string(cfg.DefaultStorageScope()),
		"sandbox_scopes":        []cfg.StorageScope{cfg.StorageScopeSandbox, cfg.StorageScopeGlobal},
		"sandbox_key":           cfg.SanitizeSandboxKey(username),
	}
	return c.JSON(http.StatusOK, ResultJSON{Result: "OK", Data: profile})
}

// Session authentication middleware
func (h *Handlers) SessionAuth(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		r := c.Request()
		// 1) Trust oauth2-proxy via nginx when present
		if r.Header.Get("X-Proxy-Auth") == "oauth2" {
			user := r.Header.Get("X-User")
			if user == "" {
				user = r.Header.Get("X-Email")
			}
			if user == "" {
				user = "oauth2-user"
			}
			derivedToken := "proxy-" + user
			if sess, ok := h.sessionManager.LookupSession(derivedToken); ok {
				c.Set("session", sess)
				return next(c)
			}
			// Create a per-user session keyed by derived token
			sess := h.sessionManager.NewSession(user, cfg.ChariotLogger, derivedToken)
			sess.Authenticated = true
			c.Set("session", sess)
			return next(c)
		}

		// 2) Authorization header path (supports optional Bearer prefix)
		authz := strings.TrimSpace(r.Header.Get("Authorization"))
		if strings.HasPrefix(strings.ToLower(authz), "bearer ") {
			authz = strings.TrimSpace(authz[7:])
		}
		cfg.ChariotLogger.Debug("SessionAuth middleware called", zap.String("token", authz))
		if authz == "" {
			return c.JSON(http.StatusUnauthorized, ResultJSON{Result: "ERROR", Data: "Authentication required (empty token)"})
		}
		session, err := h.sessionManager.GetSession(authz)
		if err != nil {
			return c.JSON(http.StatusUnauthorized, ResultJSON{Result: "ERROR", Data: "Invalid or expired session"})
		}
		c.Set("session", session)
		return next(c)
	}
}

func (h *Handlers) authenticateUser(username, password string) bool {
	// Use the bootstrap runtime to access usersAgent for authentication
	if h.bootstrapRuntime == nil {
		cfg.ChariotLogger.Error("Bootstrap runtime not available for authentication")
		return false
	}

	// Execute authentication script using the bootstrap runtime
	authScript := fmt.Sprintf(`
if(exists('usersAgent')) {
	setq(users, getChildByName(usersAgent, 'users'))
	setq(authResult, authenticateUser(users, '%s', '%s'))
	if(unequal(authResult, null)) {
		true
	} else {
		false
	}
} else {
	false
}
	`, username, password)

	result, err := h.bootstrapRuntime.ExecProgram(authScript)
	if err != nil {
		cfg.ChariotLogger.Error("Authentication script failed", zap.Error(err))
		return false
	}

	// Convert result to boolean
	if boolResult, ok := result.(chariot.Bool); ok {
		return bool(boolResult)
	}
	if strResult, ok := result.(chariot.Str); ok {
		return string(strResult) == "true"
	}

	return false
}

func generateSecureToken() string {
	// Implement your secure token generation logic here
	// For demonstration, we return a simple string
	return "secure-token"
}

// Helper function to list available templates for user
func listAvailableTemplates(session *chariot.Session) ([]map[string]interface{}, error) {
	// This would connect to your template store
	// For now returning stub data
	return []map[string]interface{}{
		{
			"id":          "template1",
			"name":        "Basic Report",
			"description": "Generates a simple report from data",
		},
		{
			"id":          "template2",
			"name":        "Data Analysis",
			"description": "Analyzes data points and generates statistics",
		},
	}, nil
}

// Helper function to convert Chariot Values to JSON-serializable format
func convertValueToJSON(val interface{}) interface{} {
	if val == nil {
		return nil
	}

	switch v := val.(type) {
	case chariot.Str:
		s := string(v)
		// Optionally, convert "true"/"false" strings to bool
		if s == "true" {
			return true
		}
		if s == "false" {
			return false
		}
		return s
	case chariot.Number:
		return float64(v)
	case chariot.Bool:
		return bool(v)
	case bool:
		return v
	case *chariot.ArrayValue:
		result := make([]interface{}, len(v.Elements))
		for i, elem := range v.Elements {
			result[i] = convertValueToJSON(elem)
		}
		return result
	case *chariot.MapValue:
		result := make(map[string]interface{})
		for k, mv := range v.Values {
			result[k] = convertValueToJSON(mv)
		}
		return result
	case *chariot.JSONNode:
		return v.GetJSONValue()
	case *chariot.MapNode:
		return v.ToMap()
	case *chariot.CouchbaseNode:
		if results := v.GetQueryResults(); results != nil {
			return convertValueToJSON(results)
		}
		return []interface{}{}
	case chariot.TreeNode:
		result := make(map[string]interface{})
		for k, mv := range v.GetAttributes() {
			result[k] = convertValueToJSON(mv)
		}
		for _, mv := range v.GetChildren() {
			result[mv.Name()] = convertValueToJSON(mv)
		}
		return result
	case map[string]interface{}:
		result := make(map[string]interface{})
		for k, v2 := range v {
			result[k] = convertValueToJSON(v2)
		}
		return result
	case []interface{}:
		result := make([]interface{}, len(v))
		for i, v2 := range v {
			result[i] = convertValueToJSON(v2)
		}
		return result
	default:
		// Optionally, handle string "true"/"false" here as well
		if s, ok := v.(string); ok {
			if s == "true" {
				return true
			}
			if s == "false" {
				return false
			}
			return s
		}
		return fmt.Sprintf("%v", val)
	}
}

// File management handlers with scope support

// ListFiles returns a list of files in the specified scope
func (h *Handlers) ListFiles(c echo.Context) error {
	sess, ok := c.Get("session").(*chariot.Session)
	if !ok || sess == nil {
		return c.JSON(http.StatusUnauthorized, ResultJSON{Result: "ERROR", Data: "session required"})
	}
	username := sess.Username
	if username == "" {
		username = sess.UserID
	}

	// Parse scope from query param, default to user's default scope
	scopeRaw := c.QueryParam("scope")
	scope := cfg.ResolveStorageScope(scopeRaw)

	cfg.ChariotLogger.Info("ListFiles request",
		zap.String("user", username),
		zap.String("scopeRaw", scopeRaw),
		zap.String("resolvedScope", string(scope)),
		zap.Bool("sandboxEnabled", cfg.ChariotConfig.SandboxEnabled),
	)

	// Get base directory for data/files
	baseDir, err := cfg.EnsureStorageBase(cfg.StorageKindData, scope, username)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, ResultJSON{Result: "ERROR", Data: err.Error()})
	}

	filesDir := filepath.Join(baseDir, "files")
	cfg.ChariotLogger.Info("ListFiles directory",
		zap.String("filesDir", filesDir),
	)

	if err := os.MkdirAll(filesDir, 0o755); err != nil {
		return c.JSON(http.StatusInternalServerError, ResultJSON{Result: "ERROR", Data: err.Error()})
	}

	entries, err := os.ReadDir(filesDir)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, ResultJSON{Result: "ERROR", Data: err.Error()})
	}

	var files []string
	for _, entry := range entries {
		if !entry.IsDir() && filepath.Ext(entry.Name()) == ".ch" {
			files = append(files, entry.Name())
		}
	}

	// Set response header indicating actual scope used
	c.Response().Header().Set("X-Chariot-Scope", string(scope))
	return c.JSON(http.StatusOK, ResultJSON{Result: "OK", Data: files})
}

// GetFile retrieves file content from the specified scope
func (h *Handlers) GetFile(c echo.Context) error {
	sess, ok := c.Get("session").(*chariot.Session)
	if !ok || sess == nil {
		return c.JSON(http.StatusUnauthorized, ResultJSON{Result: "ERROR", Data: "session required"})
	}
	username := sess.Username
	if username == "" {
		username = sess.UserID
	}

	fileName := c.Param("name")
	if fileName == "" {
		return c.JSON(http.StatusBadRequest, ResultJSON{Result: "ERROR", Data: "file name required"})
	}

	scopeRaw := c.QueryParam("scope")
	scope := cfg.ResolveStorageScope(scopeRaw)

	baseDir, err := cfg.EnsureStorageBase(cfg.StorageKindData, scope, username)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, ResultJSON{Result: "ERROR", Data: err.Error()})
	}

	filePath := filepath.Join(baseDir, "files", fileName)
	content, err := os.ReadFile(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return c.JSON(http.StatusNotFound, ResultJSON{Result: "ERROR", Data: "file not found"})
		}
		return c.JSON(http.StatusInternalServerError, ResultJSON{Result: "ERROR", Data: err.Error()})
	}

	c.Response().Header().Set("X-Chariot-Scope", string(scope))
	return c.JSON(http.StatusOK, ResultJSON{Result: "OK", Data: string(content)})
}

// SaveFile saves file content to the specified scope
func (h *Handlers) SaveFile(c echo.Context) error {
	sess, ok := c.Get("session").(*chariot.Session)
	if !ok || sess == nil {
		return c.JSON(http.StatusUnauthorized, ResultJSON{Result: "ERROR", Data: "session required"})
	}
	username := sess.Username
	if username == "" {
		username = sess.UserID
	}

	var req struct {
		Name    string `json:"name"`
		Content string `json:"content"`
	}
	if err := c.Bind(&req); err != nil || req.Name == "" {
		return c.JSON(http.StatusBadRequest, ResultJSON{Result: "ERROR", Data: "invalid request"})
	}

	scopeRaw := c.QueryParam("scope")
	scope := cfg.ResolveStorageScope(scopeRaw)

	cfg.ChariotLogger.Info("SaveFile request",
		zap.String("user", username),
		zap.String("fileName", req.Name),
		zap.String("scopeRaw", scopeRaw),
		zap.String("resolvedScope", string(scope)),
		zap.Bool("sandboxEnabled", cfg.ChariotConfig.SandboxEnabled),
	)

	baseDir, err := cfg.EnsureStorageBase(cfg.StorageKindData, scope, username)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, ResultJSON{Result: "ERROR", Data: err.Error()})
	}

	filesDir := filepath.Join(baseDir, "files")
	cfg.ChariotLogger.Info("SaveFile directory",
		zap.String("filesDir", filesDir),
	)

	if err := os.MkdirAll(filesDir, 0o755); err != nil {
		return c.JSON(http.StatusInternalServerError, ResultJSON{Result: "ERROR", Data: err.Error()})
	}

	filePath := filepath.Join(filesDir, req.Name)
	if err := os.WriteFile(filePath, []byte(req.Content), 0o644); err != nil {
		return c.JSON(http.StatusInternalServerError, ResultJSON{Result: "ERROR", Data: err.Error()})
	}

	cfg.ChariotLogger.Info("SaveFile success",
		zap.String("filePath", filePath),
	)
	c.Response().Header().Set("X-Chariot-Scope", string(scope))
	return c.JSON(http.StatusOK, ResultJSON{Result: "OK", Data: "file saved"})
}

// DeleteFile removes a file from the specified scope
func (h *Handlers) DeleteFile(c echo.Context) error {
	sess, ok := c.Get("session").(*chariot.Session)
	if !ok || sess == nil {
		return c.JSON(http.StatusUnauthorized, ResultJSON{Result: "ERROR", Data: "session required"})
	}
	username := sess.Username
	if username == "" {
		username = sess.UserID
	}

	fileName := c.Param("name")
	if fileName == "" {
		return c.JSON(http.StatusBadRequest, ResultJSON{Result: "ERROR", Data: "file name required"})
	}

	scopeRaw := c.QueryParam("scope")
	scope := cfg.ResolveStorageScope(scopeRaw)

	baseDir, err := cfg.EnsureStorageBase(cfg.StorageKindData, scope, username)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, ResultJSON{Result: "ERROR", Data: err.Error()})
	}

	filePath := filepath.Join(baseDir, "files", fileName)
	if err := os.Remove(filePath); err != nil {
		if os.IsNotExist(err) {
			return c.JSON(http.StatusNotFound, ResultJSON{Result: "ERROR", Data: "file not found"})
		}
		return c.JSON(http.StatusInternalServerError, ResultJSON{Result: "ERROR", Data: err.Error()})
	}

	c.Response().Header().Set("X-Chariot-Scope", string(scope))
	return c.JSON(http.StatusNoContent, nil)
}

// Health returns basic liveness information
func (h *Handlers) Health(c echo.Context) error {
	return c.JSON(http.StatusOK, map[string]interface{}{
		"status":          "ok",
		"uptime_seconds":  time.Since(h.startTime).Seconds(),
		"headless":        cfg.ChariotConfig.Headless,
		"bootstrapLoaded": h.bootstrapLoaded,
	})
}

// Ready returns readiness (e.g., bootstrap script loaded)
func (h *Handlers) Ready(c echo.Context) error {
	status := http.StatusOK
	if !h.bootstrapLoaded {
		status = http.StatusServiceUnavailable
	}
	return c.JSON(status, map[string]interface{}{
		"status":          map[bool]string{true: "ready", false: "initializing"}[h.bootstrapLoaded],
		"bootstrapLoaded": h.bootstrapLoaded,
	})
}
