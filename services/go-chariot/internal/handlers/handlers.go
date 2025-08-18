package handlers

import (
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/bhouse1273/chariot-ecosystem/services/go-chariot/chariot"
	cfg "github.com/bhouse1273/chariot-ecosystem/services/go-chariot/configs"
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
	bootstrapRuntime *chariot.Runtime // Global runtime for system operations
	startTime        time.Time        // Service start time for uptime metrics
	bootstrapLoaded  bool             // Indicates whether bootstrap script loaded successfully
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
			cfg.ChariotLogger.Error("Failed to get secure file path for bootstrap", zap.Error(err))
		} else {
			content, err := os.ReadFile(fullPath)
			if err != nil {
				cfg.ChariotLogger.Error("Failed to read bootstrap script", zap.Error(err))
			} else {
				if _, err := bootstrapRuntime.ExecProgram(string(content)); err != nil {
					cfg.ChariotLogger.Error("Failed to load bootstrap script in handlers", zap.Error(err))
				} else {
					bootstrapLoaded = true
				}
			}
		}
	}

	return &Handlers{
		sessionManager:   sessionManager,
		bootstrapRuntime: bootstrapRuntime,
		startTime:        time.Now(),
		bootstrapLoaded:  bootstrapLoaded,
	}
}

func (h *Handlers) Execute(c echo.Context) error {
	// Incoming JSON: {"program": "your chariot code here"}
	type Request struct {
		Program string `json:"program"`
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
			bodyStart := paramsEnd + 1
			params = strings.TrimSpace(params[:paramsEnd])
			// Remove closing paren from Program using the already computed bodyStart
			req.Program = strings.TrimSpace(req.Program[:bodyStart])
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

	// 2. Execute
	val, err := session.Runtime.ExecProgram(req.Program)
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
	if !h.authenticateUser(username, password) {
		return c.JSON(http.StatusUnauthorized, ResultJSON{
			Result: "ERROR",
			Data:   "Invalid credentials",
		})

	}

	// Generate session token (use a proper token generator)
	token := generateSecureToken()

	// Create new session
	session := h.sessionManager.NewSession(username, cfg.ChariotLogger, token)
	session.Authenticated = true

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

// Session authentication middleware
func (h *Handlers) SessionAuth(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		token := c.Request().Header.Get("Authorization")
		cfg.ChariotLogger.Debug("SessionAuth middleware called", zap.String("token", token))
		if token == "" {
			return c.JSON(http.StatusUnauthorized, ResultJSON{
				Result: "ERROR",
				Data:   "Authentication required (empty token)",
			})
		}

		session, err := h.sessionManager.GetSession(token)
		if err != nil {
			return c.JSON(http.StatusUnauthorized, ResultJSON{
				Result: "ERROR",
				Data:   "Invalid or expired session",
			})
		}

		// Store session in context
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
