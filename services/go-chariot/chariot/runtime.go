package chariot

import (
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"reflect"
	"regexp"
	"sync"
	"time"

	cfg "github.com/bhouse1273/chariot-ecosystem/services/go-chariot/configs"
	"go.uber.org/zap"
)

// LogWriter is an interface for capturing logs during script execution
type LogWriter interface {
	Append(entry LogEntry)
}

// LogEntry represents a single log entry
type LogEntry struct {
	Timestamp time.Time `json:"timestamp"`
	Level     string    `json:"level"`
	Message   string    `json:"message"`
}

// JSON returns the JSON representation of the log entry
func (e LogEntry) JSON() string {
	data, _ := json.Marshal(e)
	return string(data)
}

var (
	globalNameFilter []string = []string{
		"True", "False", "Null", "DBNull", "true", "false", "null",
	}
)

type ScriptError struct {
	Message  string
	Line     int
	Column   int
	Source   string
	Severity string // "error", "warning", etc.
}

// Position represents a location in source code
type Position struct {
	Line   int
	Column int
	File   string
}

// Runtime orchestrates execution of AST nodes and host objects.
type Runtime struct {
	funcs           map[string]func(...Value) (Value, error) // Registered functions
	objects         map[string]interface{}                   // Host objects bound to names
	lists           map[string]map[string]Value              // Named lists (like arrays)
	nodes           map[string]TreeNode                      // Named nodes for easy access
	functions       map[string]*FunctionValue                // user-defined functions
	currentPosition Position                                 // Current position in the source code
	scriptErrors    []ScriptError                            // Replace string array with structured errors

	// Logging
	logWriter LogWriter // Optional log writer for capturing script execution logs

	// Tables and related tracking
	currentTable string                        // default table if none named
	tables       map[string][]map[string]Value // Table data
	cursors      map[string]int                // Current row position for each table
	keyColumns   map[string]string             // Primary key column name for each table

	// Document handling
	document       TreeNode
	defaultDocPath string // Default path for document saving/loading

	timeOffset int // Hour offset from UTC for time calculations                                 // Main Chariot document

	namespaces map[string]Value // Namespace handlers for different contexts

	// Value-added features
	Parser            *Parser // Parser for Chariot code
	currentScope      *Scope  // Current scope for variable resolution
	globalScope       *Scope
	DefaultTemplateID string

	// Debugger
	Debugger *Debugger // Optional debugger for breakpoints and stepping
}

// NewRuntime creates an empty runtime environment.
func NewRuntime() *Runtime {
	rt := &Runtime{
		// Existing initializations
		funcs:             make(map[string]func(...Value) (Value, error)),
		objects:           make(map[string]interface{}),
		lists:             make(map[string]map[string]Value),
		nodes:             make(map[string]TreeNode),
		functions:         make(map[string]*FunctionValue),
		DefaultTemplateID: "new-item",
		timeOffset:        0, // Default time offset
		namespaces:        make(map[string]Value),

		tables:     make(map[string][]map[string]Value),
		keyColumns: make(map[string]string),

		// Initializa document
		document: NewTreeNode("document"),
		// Set default document path
		defaultDocPath: "./chariot_document.xml",

		// Initialize script errors
		scriptErrors: make([]ScriptError, 0),

		// Initialize parser
		Parser: NewParser(""),
	}
	// Initialize globalScope
	rt.globalScope = NewScope(nil)
	// Initialize currentScope with globalScope as parent
	rt.currentScope = NewScope(rt.globalScope)

	// Add built-in constants to global scope rather than vars
	rt.globalScope.Set("True", Bool(true))
	rt.globalScope.Set("False", Bool(false))
	rt.globalScope.Set("Null", DBNull)
	rt.globalScope.Set("DBNull", DBNull)
	rt.globalScope.Set("true", Bool(true))
	rt.globalScope.Set("false", Bool(false))
	rt.globalScope.Set("null", DBNull)

	// Load configured function library
	if cfg.ChariotConfig.FunctionLib != "" {
		if flib, err := LoadFunctionsFromFile(cfg.ChariotConfig.FunctionLib); err == nil {
			rt.functions = flib
		} else {
			rt.AddScriptError(fmt.Sprintf("Failed to load function library: %v", err))
		}
	}

	return rt
}

// ClearCaches
func (rt *Runtime) ClearCaches() {
	rt.objects = make(map[string]interface{})
	rt.lists = make(map[string]map[string]Value)
	rt.tables = make(map[string][]map[string]Value)
	rt.cursors = make(map[string]int)
	rt.namespaces = make(map[string]Value)
	rt.currentTable = ""
	// Note: We do NOT clear scopes here - they persist across cache clears
}

// SetLogWriter sets the log writer for capturing logs during script execution
func (rt *Runtime) SetLogWriter(writer LogWriter) {
	rt.logWriter = writer
}

// WriteLog writes a log entry if a log writer is configured
func (rt *Runtime) WriteLog(level, message string) {
	cfg.ChariotLogger.Debug("WriteLog called",
		zap.String("level", level),
		zap.String("message", message),
		zap.Bool("has_log_writer", rt.logWriter != nil))

	if rt.logWriter != nil {
		rt.logWriter.Append(LogEntry{
			Timestamp: time.Now(),
			Level:     level,
			Message:   message,
		})
	}
}

// GetFunction retrieves a registered user-defined function by name
func (rt *Runtime) GetFunction(name string) (*FunctionValue, bool) {
	if fn, exists := rt.functions[name]; exists {
		return fn, true
	}
	return nil, false
}

// GetRegisteredFunctions returns a copy of all registered built-in functions
func (rt *Runtime) GetRegisteredFunctions() map[string]func(...Value) (Value, error) {
	// Return a copy to prevent external modification
	funclist := make(map[string]func(...Value) (Value, error))
	for name, fn := range rt.funcs {
		funclist[name] = fn
	}
	return funclist
}

// GetCurrentScope returns the current scope for variable resolution
func (rt *Runtime) GetCurrentScope() *Scope {
	return rt.currentScope
}

// GetGlobalScope returns the global scope
func (rt *Runtime) GetGlobalScope() *Scope {
	return rt.globalScope
}

// ResetCurrentScope discards any variables from the previous execution while
// preserving global/built-in symbols so each run starts with a clean slate.
func (rt *Runtime) ResetCurrentScope() {
	rt.currentScope = NewScope(rt.globalScope)
}

// ExecuteCodeWithScope parses and executes Chariot code with the given scope
func (rt *Runtime) ExecuteCodeWithScope(code string, scope *Scope) (Value, error) {
	// Parse code into AST
	ast, err := rt.Parser.ParseCode(code)
	if err != nil {
		return nil, fmt.Errorf("parse error: %v", err)
	}

	// Execute the AST
	return rt.ExecuteASTWithScope(ast, scope)
}

// ExecuteASTWithScope executes an AST node with the given scope
func (rt *Runtime) ExecuteASTWithScope(node Node, scope *Scope) (Value, error) {
	// Save previous scope
	prevScope := rt.currentScope

	// Set current scope for execution
	rt.currentScope = scope

	var result Value
	var err error

	// Handle type based on what we received
	if treeNode, isTreeNode := node.(TreeNode); isTreeNode {
		// If it's a TreeNode, use the evaluator
		evaluator := NewEvaluator(rt, scope)
		result, err = evaluator.Evaluate(treeNode)
	} else {
		// If it's a regular Node, use direct execution
		result, err = node.Exec(rt)
	}

	// Restore previous scope
	rt.currentScope = prevScope

	return result, err
}

/*
func (rt *Runtime) ExecAST(ast *Block, env map[string]any) (Value, error) {
    v, err := rt.execBlock(ast, env) // existing internal executor reused
    if err != nil { return false, err }
    b, ok := v.(bool)
    if !ok {
        return false, fmt.Errorf("expected bool result, got %T", v)
    }
    return b, nil
}
*/
// Add these methods to runtime.go after line 187

// SetVariable sets a variable in the current scope
func (rt *Runtime) SetVariable(name string, value Value) {
	rt.currentScope.Set(name, value)
}

// SetVariableInScope updates a variable in the correct scope
// The variable must exist somewhere in the scope chain
func (rt *Runtime) SetVariableInScope(name string, value Value) bool {
	// Start with current scope
	scope := rt.currentScope

	for scope != nil {
		// Check if variable exists in this scope
		if entry, exists := scope.vars[name]; exists {
			// Update it in this scope
			scope.vars[name] = ScopeEntry{
				Value:    value,
				TypeCode: entry.TypeCode,
				IsTyped:  entry.IsTyped,
			}
			return true
		}

		// Move up to parent scope
		scope = scope.parent
	}

	// Not found in any scope
	return false
}

// Execute executes code in the current scope
func (rt *Runtime) Execute(code string) (Value, error) {
	return rt.ExecuteCodeWithScope(code, rt.currentScope)
}

// ExecuteWithVariables executes code with temporary variables in current scope
func (rt *Runtime) ExecuteWithVariables(code string, variables map[string]Value) (Value, error) {
	// Create temporary scope with current scope as parent
	tempScope := NewScope(rt.currentScope)

	// Set all variables in temp scope
	for name, value := range variables {
		tempScope.Set(name, value)
	}

	// Execute with temp scope
	return rt.ExecuteCodeWithScope(code, tempScope)
}

// GetVariableFromScope gets a variable from the specified scope
func (rt *Runtime) GetVariableFromScope(scope *Scope, name string) (Value, bool) {
	return scope.Get(name)
}

// Register makes a built-in function available.
func (rt *Runtime) Register(name string, fn func(...Value) (Value, error)) {
	rt.funcs[name] = fn
}

// RegisterFunction registers a user-defined function with the runtime
func (rt *Runtime) RegisterFunction(name string, fn *FunctionValue) {
	if rt.functions == nil {
		rt.functions = make(map[string]*FunctionValue)
	}
	rt.functions[name] = fn
}

// SaveFunction saves a user-defined function to the runtime
func (rt *Runtime) SaveFunction(name string, code string, formatted_source string) error {
	// 1. Transform pretty-printed format if needed
	re := regexp.MustCompile(`(?s)^function\s+(\w+)\s*\(([^)]*)\)\s*\{(.*)\}$`)
	if matches := re.FindStringSubmatch(code); len(matches) == 4 {
		// matches[1] = function name, matches[2] = params, matches[3] = body
		// Use the supplied name (not matches[1]) for overwrite safety
		params := matches[2]
		body := matches[3]
		code = fmt.Sprintf("setq(%s, func(%s) {%s})", name, params, body)
	}

	// 2. Parse the code
	ast, err := rt.Parser.ParseCode(code)
	if err != nil {
		return err
	}

	// 3. Extract FunctionDefNode and build FunctionValue
	if block, ok := ast.(*Block); ok && len(block.Stmts) == 1 {
		if setqCall, ok := block.Stmts[0].(*FuncCall); ok && setqCall.Name == "setq" && len(setqCall.Args) == 2 {
			if fnDef, ok := setqCall.Args[1].(*FunctionDefNode); ok {
				fn := &FunctionValue{
					Parameters:      fnDef.Parameters,
					Body:            fnDef.Body,
					SourceCode:      code,
					FormattedSource: formatted_source,
					Scope:           nil,
				}
				rt.functions[name] = fn
				return nil
			}
		}
		// Fallback: direct FunctionDefNode as statement
		if fnDef, ok := block.Stmts[0].(*FunctionDefNode); ok {
			fn := &FunctionValue{
				Parameters:      fnDef.Parameters,
				Body:            fnDef.Body,
				SourceCode:      code,
				FormattedSource: formatted_source,
				Scope:           nil,
			}
			rt.functions[name] = fn
			return nil
		}
	}
	// If the AST is directly a FunctionDefNode
	if fnDef, ok := ast.(*FunctionDefNode); ok {
		fn := &FunctionValue{
			Parameters:      fnDef.Parameters,
			Body:            fnDef.Body,
			SourceCode:      code,
			FormattedSource: formatted_source,
			Scope:           nil,
		}
		rt.functions[name] = fn
		return nil
	}

	return fmt.Errorf("provided code does not define a function")
}

// SetCurrentPosition updates the runtime's position tracker
func (rt *Runtime) SetCurrentPosition(line, column int, file string) {
	rt.currentPosition = Position{
		Line:   line,
		Column: column,
		File:   file,
	}
}

// ExecProgram parses and executes source, returning the last value.
func (rt *Runtime) ExecProgram(src string) (Value, error) {
	return rt.ExecProgramWithFilename(src, "main.ch")
}

// ExecProgramWithFilename parses and executes source code with a specific filename for debugging
func (rt *Runtime) ExecProgramWithFilename(src string, filename string) (Value, error) {
	ast, err := NewParserWithFilename(src, filename).parseProgram()
	if err != nil {
		return nil, err
	}

	// Ensure each execution starts with a fresh working scope so stale
	// variables from earlier runs do not leak into new debugger sessions.
	rt.ResetCurrentScope()

	// Execute with a proper scope
	return ast.Exec(rt)
}

// ParseProgram parses source code, returning the AST.
func (rt *Runtime) ParseProgram(src string) (*Block, error) {

	ast, err := NewParser(src).parseProgram()
	if err != nil {
		return nil, err
	}

	// Return abstract syntax tree
	return ast, nil
}

// convertToGoValue converts a Chariot Value to a Go value of the expected type.
func (rt *Runtime) convertToGoValue(val Value, targetType reflect.Type) (reflect.Value, error) {
	// Handle nil
	if val == nil {
		return reflect.Zero(targetType), nil
	}

	// Handle common types
	switch v := val.(type) {
	case Str:
		if targetType.Kind() == reflect.String {
			return reflect.ValueOf(string(v)), nil
		}

	case Number:
		switch targetType.Kind() {
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			return reflect.ValueOf(int64(v)).Convert(targetType), nil
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			return reflect.ValueOf(uint64(v)).Convert(targetType), nil
		case reflect.Float32, reflect.Float64:
			return reflect.ValueOf(float64(v)).Convert(targetType), nil
		}

	case Bool:
		if targetType.Kind() == reflect.Bool {
			return reflect.ValueOf(bool(v)), nil
		}

	case TreeNode:
		// If target expects TreeNode interface
		if targetType.Implements(reflect.TypeOf((*TreeNode)(nil)).Elem()) {
			return reflect.ValueOf(v), nil
		}
	}

	// Try direct conversion if types are compatible
	origValue := reflect.ValueOf(val)
	if origValue.Type().ConvertibleTo(targetType) {
		return origValue.Convert(targetType), nil
	}

	return reflect.Value{}, fmt.Errorf("cannot convert %T to %v", val, targetType)
}

// convertToChariotValue converts a Go value to a Chariot Value.
func (rt *Runtime) convertToChariotValue(val reflect.Value) (Value, error) {
	// Handle nil
	if !val.IsValid() || (val.Kind() == reflect.Ptr && val.IsNil()) {
		return nil, nil
	}

	// Extract interface values
	if val.Kind() == reflect.Interface && !val.IsNil() {
		val = val.Elem()
	}

	// Convert based on Go type
	switch val.Kind() {
	case reflect.String:
		return Str(val.String()), nil

	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return Number(val.Int()), nil

	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return Number(val.Uint()), nil

	case reflect.Float32, reflect.Float64:
		return Number(val.Float()), nil

	case reflect.Bool:
		return Bool(val.Bool()), nil

	case reflect.Slice, reflect.Array:
		// Convert to array of Values
		length := val.Len()
		elements := make([]Value, length)
		for i := 0; i < length; i++ {
			elem, err := rt.convertToChariotValue(val.Index(i))
			if err != nil {
				return nil, err
			}
			elements[i] = elem
		}
		return &ArrayValue{Elements: elements}, nil

	case reflect.Map:
		// For maps, we could convert to TreeNode or keep as map values
		// Let's go with TreeNode for consistency
		mapNode := NewMapNode("map_result")
		iter := val.MapRange()
		for iter.Next() {
			keyStr := fmt.Sprintf("%v", iter.Key().Interface())
			valValue, err := rt.convertToChariotValue(iter.Value())
			if err != nil {
				return nil, err
			}
			mapNode.Set(keyStr, valValue)
		}
		return mapNode, nil
	}

	// Check if it's a TreeNode
	if treeNode, ok := val.Interface().(TreeNode); ok {
		return treeNode, nil
	}

	// For other types, store as host object and return reference
	objName := fmt.Sprintf("host_obj_%d", len(rt.objects))
	rt.objects[objName] = val.Interface()

	// Return reference node
	refNode := NewTreeNode(objName)
	refNode.SetAttribute("type", Str("host_reference"))

	return refNode, nil
}

// SetTimeOffset sets the global time offset in hours
func (rt *Runtime) SetTimeOffset(hours int) {
	rt.timeOffset = hours
}

// Namespace handling methods

// GetVariable retrieves a variable value from the runtime
// Returns the value and true if found, DBNull and false if not found
func (rt *Runtime) GetVariable(name string) (Value, bool) {
	// First check current scope
	if value, exists := rt.CurrentScope().Get(name); exists {
		return value, true
	}

	// Then check global scope
	if value, exists := rt.GlobalScope().Get(name); exists {
		return value, true
	}

	// Not found
	return DBNull, false
}

// GetVariableString is a convenience method that returns the string representation
func (rt *Runtime) GetVariableString(name string) (string, bool) {
	value, exists := rt.GetVariable(name)
	if !exists {
		return "", false
	}
	return fmt.Sprintf("%v", value), true
}

// DeleteFunction removes a user-defined function from the runtime
func (rt *Runtime) DeleteFunction(name string) {
	if _, exists := rt.functions[name]; exists {
		delete(rt.functions, name)
	} else {
		rt.AddScriptError(fmt.Sprintf("Function '%s' does not exist", name))
	}
}

// GetVariableNative converts a Chariot value to native Go types for easier testing
func (rt *Runtime) GetVariableNative(name string) (interface{}, bool) {
	value, exists := rt.GetVariable(name)
	if !exists {
		return nil, false
	}

	return convertValueToNative(value), true
}

// ListVariables returns all variable names in current and global scopes
func (rt *Runtime) ListVariables() map[string]Value {
	variables := make(map[string]Value)

	// Add global variables first
	for name, value := range rt.GlobalScope().vars {
		variables[name] = value
	}

	// Add/override with current scope variables
	for name, value := range rt.CurrentScope().vars {
		variables[name] = value
	}

	return variables
}

// ListVariables returns all variable names in current and global scopes
func (rt *Runtime) ListGlobalVariables() map[string]Value {
	variables := make(map[string]Value)

	// Add global variables first
	for name, entry := range rt.GlobalScope().vars {
		if !contains(globalNameFilter, name) {
			variables[name] = entry.Value // Unwrap ScopeEntry to get raw Value
		}
	}

	return variables
}

// ListLocalVariables returns all variable names in the current scope
func (rt *Runtime) ListLocalVariables() map[string]Value {
	variables := make(map[string]Value)

	// Add current scope variables
	for name, entry := range rt.CurrentScope().vars {
		variables[name] = entry.Value // Unwrap ScopeEntry to get raw Value
	}

	return variables
}

// ListObjects returns all host objects registered in the runtime
func (rt *Runtime) ListObjects() map[string]interface{} {
	objects := make(map[string]interface{})
	// Add all registered objects
	for name, obj := range rt.objects {
		objects[name] = obj
	}
	return objects
}

// ListLists returns all named lists in the runtime
func (rt *Runtime) ListLists() map[string]map[string]Value {
	lists := make(map[string]map[string]Value)
	// Add all named lists
	for name, list := range rt.lists {
		lists[name] = make(map[string]Value)
		for key, value := range list {
			lists[name][key] = value
		}
	}
	return lists
}

// ListNamespaces returns all registered namespaces in the runtime
func (rt *Runtime) ListNamespaces() map[string]Value {
	namespaces := make(map[string]Value)
	// Add all registered namespaces
	for name, ns := range rt.namespaces {
		namespaces[name] = ns
	}
	return namespaces
}

// ListTables returns all registered tables in the runtime
func (rt *Runtime) ListTables() map[string][]map[string]Value {
	tables := make(map[string][]map[string]Value)
	// Add all registered tables
	for name, table := range rt.tables {
		tables[name] = make([]map[string]Value, len(table))
		for i, row := range table {
			tables[name][i] = make(map[string]Value)
			for key, value := range row {
				tables[name][i][key] = value
			}
		}
	}
	return tables
}

// ListKeyColumns returns all key columns for registered tables
func (rt *Runtime) ListKeyColumns() map[string]string {
	keyColumns := make(map[string]string)
	// Add all key columns
	for table, key := range rt.keyColumns {
		keyColumns[table] = key
	}
	return keyColumns
}

// ListNodes returns all named nodes in the runtime
func (rt *Runtime) ListNodes() map[string]TreeNode {
	nodes := make(map[string]TreeNode)
	// Add all named nodes
	for name, node := range rt.nodes {
		nodes[name] = node.Clone() // Clone to avoid external modifications
	}
	return nodes
}

func (rt *Runtime) ListFunctions() *ArrayValue {
	functions := make([]Value, 0, len(rt.functions))
	for name := range rt.functions {
		functions = append(functions, Str(name))
	}
	return &ArrayValue{Elements: functions}
}

// ListUserFunctionsMap returns a shallow copy of the runtime's user-defined functions map
func (rt *Runtime) ListUserFunctionsMap() map[string]*FunctionValue {
	out := make(map[string]*FunctionValue, len(rt.functions))
	for k, v := range rt.functions {
		out[k] = v
	}
	return out
}

func (rt *Runtime) LoadConfiguredFunctionLib() error {
	libFile := cfg.ChariotConfig.FunctionLib
	if libFile == "" {
		return nil // No function library specified
	}
	// Use TreePath for consistency
	fullPath := filepath.Join(cfg.ChariotConfig.TreePath, libFile)
	functions, err := LoadFunctionsFromFile(fullPath)
	if err != nil {
		return err
	}
	for name, fn := range functions {
		rt.RegisterFunction(name, fn)
	}
	return nil
}

// SetGlobalVariable allows the host to set global variables
func (rt *Runtime) SetGlobalVariable(name string, value Value) {
	rt.GlobalScope().Set(name, value)
}

func (rt *Runtime) SetListValue(key string, value Value) (Value, error) {
	// Implementation for List value setting
	if rt.lists == nil {
		rt.lists = make(map[string]map[string]Value)
	}
	if _, exists := rt.lists[key]; !exists {
		rt.lists[key] = make(map[string]Value)
	}
	rt.lists[key][key] = value
	return value, nil
}

func (rt *Runtime) SetArrayValue(index string, value Value) (Value, error) {
	// Implementation for Array value setting
	if rt.lists == nil {
		rt.lists = make(map[string]map[string]Value)
	}
	if _, exists := rt.lists[index]; !exists {
		rt.lists[index] = make(map[string]Value)
	}
	rt.lists[index][index] = value
	return value, nil
}

// Evaluator handles AST evaluation
type Evaluator struct {
	runtime *Runtime
	scope   *Scope
}

// NewEvaluator creates a new AST evaluator
func NewEvaluator(rt *Runtime, scope *Scope) *Evaluator {
	return &Evaluator{
		runtime: rt,
		scope:   scope,
	}
}

// Evaluate evaluates an AST node and returns its value
func (e *Evaluator) Evaluate(node TreeNode) (Value, error) {
	// Implementation depends on your AST structure
	// This would handle different node types like:
	// - Function calls
	// - Variable references
	// - Literals
	// - Operators
	// - Control structures

	// Example implementation for a simplified AST
	nodeType, hasType := node.GetAttribute("type")
	if !hasType {
		return nil, errors.New("node has no type attribute")
	}

	switch nodeType {
	case Str("literal"):
		// Return the value directly
		val, _ := node.GetAttribute("value")
		return val, nil

	case Str("variable"):
		// Get variable from scope
		name, _ := node.GetAttribute("name")
		nameStr, _ := name.(Str)
		val, exists := e.scope.Get(string(nameStr))
		if !exists {
			return nil, fmt.Errorf("undefined variable: %s", nameStr)
		}
		return val, nil

	case Str("call"):
		// Function call
		fnName, _ := node.GetAttribute("name")
		fnNameStr, _ := fnName.(Str)

		// Evaluate arguments
		argNodes := node.GetChildren()
		args := make([]Value, len(argNodes))
		for i, argNode := range argNodes {
			argVal, err := e.Evaluate(argNode)
			if err != nil {
				return nil, err
			}
			args[i] = argVal
		}

		// Call the function
		fn, exists := e.runtime.functions[string(fnNameStr)]
		if !exists {
			return nil, fmt.Errorf("undefined function: %s", fnNameStr)
		}

		return executeFunctionValue(e.runtime, fn, args)
	}

	return nil, fmt.Errorf("unsupported node type: %v", nodeType)
}

func (rt *Runtime) RunProgram(entry string, port int) error {
	// Execute a registered stdlib function by name if present; otherwise treat as code string
	if entry == "" {
		return nil
	}
	if fn, ok := rt.functions[entry]; ok {
		// Call with one argument: port (optional for functions that accept it)
		_, err := executeFunctionValue(rt, fn, []Value{Number(port)})
		return err
	}
	// Fallback: attempt to execute raw code string
	_, err := rt.ExecProgram(entry)
	return err
}

// SetDefaultDocPath changes the default path for document operations
func (rt *Runtime) SetDefaultDocPath(path string) {
	rt.defaultDocPath = path
}

// AddScriptError adds an error message to the runtime's error collection
func (rt *Runtime) AddScriptError(errMsg string) {
	rt.scriptErrors = append(rt.scriptErrors, ScriptError{
		Message:  errMsg,
		Line:     rt.currentPosition.Line,
		Column:   rt.currentPosition.Column,
		Source:   rt.currentPosition.File,
		Severity: "error",
	})
}

// AddError adds an error object with source information to the runtime's error collection
func (rt *Runtime) AddError(err error, source string) {
	rt.scriptErrors = append(rt.scriptErrors, ScriptError{
		Message:  fmt.Sprintf("%s: %v", source, err),
		Line:     rt.currentPosition.Line,
		Column:   rt.currentPosition.Column,
		Source:   source,
		Severity: "error",
	})
}

// GetLastError returns the most recently added error message
func (rt *Runtime) GetLastError() string {
	if len(rt.scriptErrors) > 0 {
		// Extract just the message from the last ScriptError
		return rt.scriptErrors[len(rt.scriptErrors)-1].Message
	}
	return ""
}

// GetAllScriptErrors returns all accumulated script errors
func (rt *Runtime) GetAllScriptErrors() []ScriptError {
	return rt.scriptErrors
}

// HasErrors returns true if there are any script errors
func (rt *Runtime) HasErrors() bool {
	return len(rt.scriptErrors) > 0
}

func (rt *Runtime) CurrentScope() *Scope {
	return rt.currentScope
}

func (rt *Runtime) GlobalScope() *Scope {
	return rt.globalScope
}

// CloneRuntime creates a deep copy of a runtime for isolated testing
func (rt *Runtime) CloneRuntime() *Runtime {
	clone := &Runtime{
		funcs:             make(map[string]func(...Value) (Value, error)),
		objects:           make(map[string]interface{}),
		lists:             make(map[string]map[string]Value),
		nodes:             make(map[string]TreeNode),
		functions:         make(map[string]*FunctionValue),
		tables:            make(map[string][]map[string]Value),
		keyColumns:        make(map[string]string),
		cursors:           make(map[string]int),
		namespaces:        make(map[string]Value),
		DefaultTemplateID: rt.DefaultTemplateID,
		document:          rt.document.Clone(),
		defaultDocPath:    rt.defaultDocPath,
		timeOffset:        rt.timeOffset,
		Parser:            NewParser(""),
	}

	// Clone script errors
	clone.scriptErrors = make([]ScriptError, len(rt.scriptErrors))
	for i, err := range rt.scriptErrors {
		clone.scriptErrors[i] = ScriptError{
			Message:  err.Message,
			Line:     err.Line,
			Column:   err.Column,
			Source:   err.Source,
			Severity: err.Severity,
		}
	}

	// Clone scope chain
	clone.globalScope = NewScope(nil)
	for k, v := range rt.globalScope.vars {
		clone.globalScope.Set(k, v)
	}

	clone.currentScope = NewScope(clone.globalScope)
	for k, v := range rt.currentScope.vars {
		clone.currentScope.Set(k, v)
	}

	// Copy functions
	for k, v := range rt.funcs {
		clone.funcs[k] = v
	}

	// Copy user-defined functions and REBIND closure scope to the clone's global scope
	if rt.functions != nil {
		for name, fn := range rt.functions {
			clone.functions[name] = cloneFunctionValueWithScope(fn, clone.globalScope)
		}
	}

	// Clone nodes
	for k, v := range rt.nodes {
		clone.nodes[k] = v.Clone()
	}

	// Deep clone lists
	for listName, list := range rt.lists {
		cloneList := make(map[string]Value)
		for k, v := range list {
			cloneList[k] = v // Values themselves should be immutable
		}
		clone.lists[listName] = cloneList
	}

	// Deep clone tables
	for tableName, table := range rt.tables {
		cloneTable := make([]map[string]Value, len(table))
		for i, row := range table {
			cloneRow := make(map[string]Value)
			for k, v := range row {
				cloneRow[k] = v // Values themselves should be immutable
			}
			cloneTable[i] = cloneRow
		}
		clone.tables[tableName] = cloneTable
	}

	// Clone cursors
	for k, v := range rt.cursors {
		clone.cursors[k] = v
	}

	// Clone keyColumns
	for k, v := range rt.keyColumns {
		clone.keyColumns[k] = v
	}

	// Clone namespaces
	for k, v := range rt.namespaces {
		clone.namespaces[k] = v // Assuming these are immutable
	}

	// Objects are trickier since they're interface{}
	// For now let's do a shallow copy, but consider a deeper strategy
	for k, v := range rt.objects {
		clone.objects[k] = v
	}

	// Register all functions with the cloned runtime
	RegisterAll(clone)

	return clone
}

// cloneFunctionValueWithScope creates a shallow copy of a FunctionValue but points its closure to newScope.
// This preserves lexical semantics when moving functions across runtimes (e.g., bootstrap -> per-agent clone).
func cloneFunctionValueWithScope(src *FunctionValue, newScope *Scope) *FunctionValue {
	if src == nil {
		return nil
	}
	return &FunctionValue{
		Body:            src.Body,
		Parameters:      append([]string(nil), src.Parameters...),
		SourceCode:      src.SourceCode,
		FormattedSource: src.FormattedSource,
		IsParsed:        src.IsParsed,
		Scope:           newScope,
	}
}

// FindVariable searches for a variable in the current scope chain
// Returns the entry and true if found, empty entry and false if not
func (rt *Runtime) FindVariable(name string) (ScopeEntry, bool) {
	// Start with current scope
	scope := rt.currentScope

	for scope != nil {
		// Check if variable exists in this scope
		if entry, exists := scope.vars[name]; exists {
			return entry, true
		}

		// Move up to parent scope
		scope = scope.parent
	}

	return ScopeEntry{}, false
}

var (
	runtimeRegistry = make(map[string]*Runtime)
	runtimeMutex    sync.RWMutex
	defaultRuntime  *Runtime
)

// RegisterRuntime registers a runtime instance with an ID
func RegisterRuntime(id string, rt *Runtime) {
	runtimeMutex.Lock()
	defer runtimeMutex.Unlock()
	runtimeRegistry[id] = rt

	// Set as default if it's the first one
	if defaultRuntime == nil {
		defaultRuntime = rt
	}
}

// GetRuntime returns the default runtime or a specific runtime by ID
func GetRuntime(id ...string) *Runtime {
	runtimeMutex.RLock()
	defer runtimeMutex.RUnlock()

	if len(id) > 0 && id[0] != "" {
		return runtimeRegistry[id[0]]
	}

	return defaultRuntime
}

// GetRuntimeByID returns a specific runtime by ID
func GetRuntimeByID(id string) (*Runtime, bool) {
	runtimeMutex.RLock()
	defer runtimeMutex.RUnlock()
	rt, exists := runtimeRegistry[id]
	return rt, exists
}

// UnregisterRuntime removes a runtime from the registry
func UnregisterRuntime(id string) {
	runtimeMutex.Lock()
	defer runtimeMutex.Unlock()
	delete(runtimeRegistry, id)

	// Clear default if it was the default
	if defaultRuntime != nil {
		if rt, exists := runtimeRegistry[id]; exists && rt == defaultRuntime {
			defaultRuntime = nil
			// Set a new default if any exist
			for _, newDefault := range runtimeRegistry {
				defaultRuntime = newDefault
				break
			}
		}
	}
}

// ListRuntimes returns all registered runtime IDs
func ListRuntimes() []string {
	runtimeMutex.RLock()
	defer runtimeMutex.RUnlock()

	ids := make([]string, 0, len(runtimeRegistry))
	for id := range runtimeRegistry {
		ids = append(ids, id)
	}
	return ids
}

func executeFunctionValue(rt *Runtime, fn *FunctionValue, args []Value) (Value, error) {
	// Create new scope with proper parent
	var parentScope *Scope
	if fn.Scope != nil {
		// Use closure scope for functions created at runtime
		parentScope = fn.Scope
	} else {
		// Use current scope for deserialized functions
		parentScope = rt.currentScope
	}

	fnScope := NewScope(parentScope)

	// Bind arguments to parameters (KEEP THIS)
	for i, param := range fn.Parameters {
		if i < len(args) {
			fnScope.Set(param, args[i])
		} else {
			fnScope.Set(param, DBNull) // Default value for missing args
		}
	}

	// Save current scope and restore after execution (KEEP THIS)
	prevScope := rt.currentScope
	rt.currentScope = fnScope

	var result Value
	var err error

	// Extract statements from Body if it's a Block
	if block, ok := fn.Body.(*Block); ok {
		// Execute statements from the Block
		var last Value
		for _, stmt := range block.Stmts {
			last, err = stmt.Exec(rt)
			if err != nil {
				break
			}
		}
		result = last
	} else {
		// Single statement
		result, err = fn.Body.Exec(rt)
	}

	// Restore scope (KEEP THIS)
	rt.currentScope = prevScope

	// Handle return statements
	if err != nil {
		if retErr, ok := err.(*ReturnError); ok {
			// Return is successful - extract the value
			return retErr.Value, nil
		}
	}

	return result, err
}
