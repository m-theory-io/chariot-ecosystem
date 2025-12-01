// Project: Chariot
// ast.go
// Defines the AST nodes and execution logic for Chariot scripts.
package chariot

import (
	"errors"
	"fmt"
	"strings"
)

// Define custom error types for break/continue flow control
type BreakError struct{}

func (e *BreakError) Error() string { return "break" }

type ContinueError struct{}

func (e *ContinueError) Error() string { return "continue" }

// ReturnError represents an early return with a value from a block
type ReturnError struct {
	Value Value
}

func (e *ReturnError) Error() string { return "return" }

// ExitRequest represents a request to terminate program execution
type ExitRequest struct {
	Code int
}

func (e *ExitRequest) Error() string { return fmt.Sprintf("exit with code %d", e.Code) }

// ArrayLiteralNode represents an array literal [elem1, elem2, ...]
type ArrayLiteralNode struct {
	Elements []Node
	Pos      SourcePos
}

func (a *ArrayLiteralNode) GetPos() SourcePos    { return a.Pos }
func (a *ArrayLiteralNode) SetPos(pos SourcePos) { a.Pos = pos }

// Exec evaluates each element and creates an ArrayValue
func (a *ArrayLiteralNode) Exec(rt *Runtime) (Value, error) {
	array := &ArrayValue{
		Elements: make([]Value, 0, len(a.Elements)),
	}

	for _, elemNode := range a.Elements {
		elemValue, err := elemNode.Exec(rt)
		if err != nil {
			return nil, err
		}
		array.Append(elemValue)
	}

	return array, nil
}

func (a *ArrayLiteralNode) ToMap() map[string]interface{} {
	elements := make([]interface{}, len(a.Elements))
	for i, elem := range a.Elements {
		elements[i] = elem.ToMap()
	}
	return map[string]interface{}{
		"_node_type": "ArrayLiteralNode",
		"elements":   elements,
	}
}

func (a *ArrayLiteralNode) ToString() string {
	var sb strings.Builder
	sb.WriteString("[")
	for i, elem := range a.Elements {
		if tvar := elem.ToString(); tvar == "" {
			return ""
		} else {
			sb.WriteString(tvar)
		}
		if i < len(a.Elements)-1 {
			sb.WriteString(", ")
		}
	}
	sb.WriteString("]")
	return sb.String()
}

// FunctionDefNode represents a function definition
type FunctionDefNode struct {
	Parameters []string
	Body       Node
	Source     string
	Position   int // Source position for error reporting (deprecated, use Pos)
	Pos        SourcePos
}

func (f *FunctionDefNode) GetPos() SourcePos    { return f.Pos }
func (f *FunctionDefNode) SetPos(pos SourcePos) { f.Pos = pos }

// Exec creates a FunctionValue
func (f *FunctionDefNode) Exec(rt *Runtime) (Value, error) {
	return &FunctionValue{
		Body:       f.Body,
		Parameters: f.Parameters,
		SourceCode: f.Source,
		IsParsed:   true,            // Already parsed
		Scope:      rt.currentScope, // Capture closure
	}, nil
}

func (f *FunctionDefNode) ToMap() map[string]interface{} {
	return map[string]interface{}{
		"_node_type": "FunctionDefNode",
		"parameters": f.Parameters,
		"body":       f.Body.ToMap(),
		"source":     f.Source,
		"position":   f.Position,
	}
}

// ToString returns the function definition as a string.
func (f *FunctionDefNode) ToString() string {
	var sb strings.Builder
	sb.WriteString("function(")
	for i, param := range f.Parameters {
		sb.WriteString(param)
		if i < len(f.Parameters)-1 {
			sb.WriteString(", ")
		}
	}
	sb.WriteString(") {\n")
	if bodyStr := f.Body.ToString(); bodyStr == "" {
		return ""
	} else {
		sb.WriteString("    ")
		sb.WriteString(bodyStr)
		sb.WriteString("\n")

	}
	sb.WriteString("}\n")
	return sb.String()
}

// FunctionCallNode represents a function call
type FunctionCallNode struct {
	FuncExpr Node   // Expression that should evaluate to a FunctionValue
	Args     []Node // Argument expressions
	Pos      SourcePos
}

func (c *FunctionCallNode) GetPos() SourcePos    { return c.Pos }
func (c *FunctionCallNode) SetPos(pos SourcePos) { c.Pos = pos }

// Exec evaluates the function with arguments
func (c *FunctionCallNode) Exec(rt *Runtime) (Value, error) {
	// Evaluate function expression
	fnVal, err := c.FuncExpr.Exec(rt)
	if err != nil {
		return nil, err
	}

	// Check if it's a function
	fn, ok := fnVal.(*FunctionValue)
	if !ok {
		return nil, fmt.Errorf("not a function: %v", fnVal)
	}

	// Evaluate arguments
	args := make([]Value, len(c.Args))
	for i, argExpr := range c.Args {
		argVal, err := argExpr.Exec(rt)
		if err != nil {
			return nil, err
		}
		args[i] = argVal
	}

	// Execute function
	return executeFunctionValue(rt, fn, args)
}

func (c *FunctionCallNode) ToMap() map[string]interface{} {
	args := make([]interface{}, len(c.Args))
	for i, arg := range c.Args {
		args[i] = arg.ToMap()
	}
	return map[string]interface{}{
		"_node_type": "FunctionCallNode",
		"funcExpr":   c.FuncExpr.ToMap(),
		"args":       args,
	}
}

// ToString returns the function call as a string.
func (c *FunctionCallNode) ToString() string {
	var sb strings.Builder
	if funcStr := c.FuncExpr.ToString(); funcStr == "" {
		return ""
	} else {
		sb.WriteString(funcStr)
	}
	sb.WriteString("(")
	for i, arg := range c.Args {
		if argStr := arg.ToString(); argStr == "" {
			return ""
		} else {
			sb.WriteString(argStr)
		}
		if i < len(c.Args)-1 {
			sb.WriteString(", ")
		}
	}
	sb.WriteString(")")
	return sb.String()
}

// WhileNode represents a while loop in the AST
type WhileNode struct {
	Condition Node
	Body      []Node
	Position  int // deprecated, use Pos
	Pos       SourcePos
}

func (w *WhileNode) GetPos() SourcePos    { return w.Pos }
func (w *WhileNode) SetPos(pos SourcePos) { w.Pos = pos }

// Exec executes a while loop
func (w *WhileNode) Exec(rt *Runtime) (Value, error) {
	var last Value
	for {
		// Evaluate condition
		condValue, err := w.Condition.Exec(rt)
		if err != nil {
			return nil, err
		}

		// Check if condition is truthy
		if !boolify(condValue) {
			break
		}

		// Execute body as a block with proper scope
		bodyBlock := &Block{Stmts: w.Body}
		blockResult, err := bodyBlock.Exec(rt)

		// Handle special control flow errors
		if err != nil {
			if _, ok := err.(*BreakError); ok {
				return last, nil // Exit the loop completely
			}
			if _, ok := err.(*ContinueError); ok {
				continue // Go to next iteration
			}
			return nil, err // Propagate other errors
		}

		last = blockResult

	}

	return last, nil
}

func (w *WhileNode) ToMap() map[string]interface{} {
	body := make([]interface{}, len(w.Body))
	for i, stmt := range w.Body {
		body[i] = stmt.ToMap()
	}
	return map[string]interface{}{
		"_node_type": "WhileNode",
		"condition":  w.Condition.ToMap(),
		"body":       body,
		"position":   w.Position,
	}
}

// ToString returns the while loop as a string.
func (w *WhileNode) ToString() string {
	var sb strings.Builder
	sb.WriteString("while (")
	if condStr := w.Condition.ToString(); condStr == "" {
		return ""
	} else {
		sb.WriteString(condStr)
	}
	sb.WriteString(") {\n")
	for _, stmt := range w.Body {
		if stmtStr := stmt.ToString(); stmtStr == "" {
			return ""
		} else {
			sb.WriteString("    ")
			sb.WriteString(stmtStr)
			sb.WriteString("\n")
		}
	}
	sb.WriteString("}")
	return sb.String()
}

// BreakNode represents a break statement
type BreakNode struct {
	Position int // deprecated, use Pos
	Pos      SourcePos
}

func (b *BreakNode) GetPos() SourcePos    { return b.Pos }
func (b *BreakNode) SetPos(pos SourcePos) { b.Pos = pos }

// Exec implements breaking out of a loop
func (b *BreakNode) Exec(rt *Runtime) (Value, error) {
	return nil, &BreakError{}
}

// ContinueNode represents a continue statement
type ContinueNode struct {
	Position int // deprecated, use Pos
	Pos      SourcePos
}

func (c *ContinueNode) GetPos() SourcePos    { return c.Pos }
func (c *ContinueNode) SetPos(pos SourcePos) { c.Pos = pos }

// Exec implements continuing to the next loop iteration
func (c *ContinueNode) Exec(rt *Runtime) (Value, error) {
	return nil, &ContinueError{}
}

// IfNode represents an if/else condition in the AST
type IfNode struct {
	Condition   Node   // The condition to evaluate
	TrueBranch  []Node // Statements to execute when condition is true
	FalseBranch []Node // Statements to execute when condition is false (optional)
	Position    int    // Source position for error reporting (deprecated, use Pos)
	Pos         SourcePos
}

func (i *IfNode) GetPos() SourcePos    { return i.Pos }
func (i *IfNode) SetPos(pos SourcePos) { i.Pos = pos }

// Exec executes an if/else statement
func (i *IfNode) Exec(rt *Runtime) (Value, error) {
	// Evaluate condition
	condValue, err := i.Condition.Exec(rt)
	if err != nil {
		return nil, err
	}

	// Determine which branch to execute based on condition
	if boolify(condValue) {
		// Create new scope for the true branch
		branchScope := NewScope(rt.currentScope)
		prevScope := rt.currentScope
		rt.currentScope = branchScope

		// Execute true branch statements
		var result Value
		for _, stmt := range i.TrueBranch {
			result, err = stmt.Exec(rt)
			if err != nil {
				// Restore scope before returning error
				rt.currentScope = prevScope
				// Check for special control flow errors
				if _, ok := err.(*BreakError); ok {
					return nil, err // Let enclosing loop handle break
				}
				if _, ok := err.(*ContinueError); ok {
					return nil, err // Let enclosing loop handle continue
				}
				return nil, err // Propagate other errors
			}
		}

		// Restore previous scope
		rt.currentScope = prevScope
		return result, nil

	} else if len(i.FalseBranch) > 0 {
		// Create new scope for the false branch
		branchScope := NewScope(rt.currentScope)
		prevScope := rt.currentScope
		rt.currentScope = branchScope

		// Execute false branch statements
		var result Value
		for _, stmt := range i.FalseBranch {
			result, err = stmt.Exec(rt)
			if err != nil {
				// Restore scope before returning error
				rt.currentScope = prevScope
				// Check for special control flow errors
				if _, ok := err.(*BreakError); ok {
					return nil, err // Let enclosing loop handle break
				}
				if _, ok := err.(*ContinueError); ok {
					return nil, err // Let enclosing loop handle continue
				}
				return nil, err // Propagate other errors
			}
		}

		// Restore previous scope
		rt.currentScope = prevScope
		return result, nil
	}

	// No branch executed
	return DBNull, nil
}

func (n *IfNode) ToMap() map[string]interface{} {
	trueBranch := make([]interface{}, len(n.TrueBranch))
	for i, stmt := range n.TrueBranch {
		trueBranch[i] = stmt.ToMap()
	}
	falseBranch := make([]interface{}, len(n.FalseBranch))
	for i, stmt := range n.FalseBranch {
		falseBranch[i] = stmt.ToMap()
	}
	return map[string]interface{}{
		"_node_type":  "IfNode",
		"condition":   n.Condition.ToMap(),
		"trueBranch":  trueBranch,
		"falseBranch": falseBranch,
		"position":    n.Position,
	}
}

// ToString returns the if/else statement as a string.
// ToString returns the if/else statement as a string.
func (n *IfNode) ToString() string {
	var sb strings.Builder
	sb.WriteString("if (")
	sb.WriteString(n.Condition.ToString())
	sb.WriteString(") {\n")
	for _, stmt := range n.TrueBranch {
		sb.WriteString("    ") // Indent the statement
		sb.WriteString(stmt.ToString())
		sb.WriteString("\n")
	}
	sb.WriteString("}") // Closing brace at same level as "if"
	if len(n.FalseBranch) > 0 {
		sb.WriteString(" else {\n")
		for _, stmt := range n.FalseBranch {
			sb.WriteString("    ") // Indent the statement
			sb.WriteString(stmt.ToString())
			sb.WriteString("\n")
		}
		sb.WriteString("}") // Closing brace at same level as "else"
	}
	return sb.String()
}

// SwitchNode represents a switch statement
type SwitchNode struct {
	TestExpr    Node         // nil for switch() form, expression for switch(expr) form
	Cases       []*CaseNode  // List of case statements
	DefaultCase *DefaultNode // Optional default case
	Position    int          // Source position for error reporting (deprecated, use Pos)
	Pos         SourcePos
}

func (s *SwitchNode) GetPos() SourcePos    { return s.Pos }
func (s *SwitchNode) SetPos(pos SourcePos) { s.Pos = pos }

// Exec executes a switch statement
func (s *SwitchNode) Exec(rt *Runtime) (Value, error) {
	var testValue Value
	var err error

	// Evaluate test expression if present
	if s.TestExpr != nil {
		testValue, err = s.TestExpr.Exec(rt)
		if err != nil {
			return nil, err
		}
	}

	// Try each case
	for _, caseNode := range s.Cases {
		var matched bool

		if testValue != nil {
			// Flavor 1: switch(testValue) - compare with case value
			caseValue, err := caseNode.Condition.Exec(rt)
			if err != nil {
				return nil, err
			}

			// Use equal function for comparison
			if equalFunc, ok := rt.funcs["equal"]; ok {
				result, err := equalFunc(testValue, caseValue)
				if err != nil {
					return nil, err
				}

				if boolResult, ok := result.(Bool); ok {
					matched = bool(boolResult)
				}
			} else {
				// Fallback to direct comparison
				matched = compareValues(testValue, caseValue)
			}
		} else {
			// Flavor 2: switch() - evaluate case expression
			caseResult, err := caseNode.Condition.Exec(rt)
			if err != nil {
				return nil, err
			}

			matched = boolify(caseResult)
		}

		if matched {
			return caseNode.Body.Exec(rt)
		}
	}

	// No cases matched, try default
	if s.DefaultCase != nil {
		return s.DefaultCase.Body.Exec(rt)
	}

	return DBNull, nil
}

func (s *SwitchNode) ToMap() map[string]interface{} {
	cases := make([]interface{}, len(s.Cases))
	for i, caseNode := range s.Cases {
		cases[i] = caseNode.ToMap()
	}

	result := map[string]interface{}{
		"_node_type": "SwitchNode",
		"cases":      cases,
		"position":   s.Position,
	}

	if s.TestExpr != nil {
		result["testExpr"] = s.TestExpr.ToMap()
	}

	if s.DefaultCase != nil {
		result["defaultCase"] = s.DefaultCase.ToMap()
	}

	return result
}

func (s *SwitchNode) ToString() string {
	var sb strings.Builder
	sb.WriteString("switch")

	if s.TestExpr != nil {
		sb.WriteString("(")
		sb.WriteString(s.TestExpr.ToString())
		sb.WriteString(")")
	} else {
		sb.WriteString("()")
	}

	sb.WriteString(" {\n")

	for _, caseNode := range s.Cases {
		sb.WriteString("    ")
		sb.WriteString(caseNode.ToString())
		sb.WriteString("\n")
	}

	if s.DefaultCase != nil {
		sb.WriteString("    ")
		sb.WriteString(s.DefaultCase.ToString())
		sb.WriteString("\n")
	}

	sb.WriteString("}") // <-- Add this line to close the switch block

	return sb.String()
}

// CaseNode represents a case within a switch
type CaseNode struct {
	Condition Node // The condition to match against
	Body      Node // The block to execute if matched
	Pos       SourcePos
}

func (c *CaseNode) GetPos() SourcePos    { return c.Pos }
func (c *CaseNode) SetPos(pos SourcePos) { c.Pos = pos }

func (c *CaseNode) ToMap() map[string]interface{} {
	return map[string]interface{}{
		"_node_type": "CaseNode",
		"condition":  c.Condition.ToMap(),
		"body":       c.Body.ToMap(),
	}
}

func (c *CaseNode) ToString() string {
	var sb strings.Builder
	sb.WriteString("case(")
	sb.WriteString(c.Condition.ToString())
	sb.WriteString(") {\n")

	// If body is a block, print each statement indented
	if block, ok := c.Body.(*Block); ok {
		for _, stmt := range block.Stmts {
			sb.WriteString("        ") // 2 levels of indent
			sb.WriteString(stmt.ToString())
			sb.WriteString("\n")
		}
	} else {
		// Otherwise, just print the body indented
		sb.WriteString("        ")
		sb.WriteString(c.Body.ToString())
		sb.WriteString("\n")
	}

	sb.WriteString("    }")
	return sb.String()
}

// DefaultNode represents the default case in a switch
type DefaultNode struct {
	Body Node // The block to execute by default
	Pos  SourcePos
}

func (d *DefaultNode) GetPos() SourcePos    { return d.Pos }
func (d *DefaultNode) SetPos(pos SourcePos) { d.Pos = pos }

func (d *DefaultNode) ToMap() map[string]interface{} {
	return map[string]interface{}{
		"_node_type": "DefaultNode",
		"body":       d.Body.ToMap(),
	}
}

func (d *DefaultNode) ToString() string {
	var sb strings.Builder
	sb.WriteString("default() {\n")
	if block, ok := d.Body.(*Block); ok {
		for _, stmt := range block.Stmts {
			sb.WriteString("        ")
			sb.WriteString(stmt.ToString())
			sb.WriteString("\n")
		}
	} else {
		sb.WriteString("        ")
		sb.WriteString(d.Body.ToString())
		sb.WriteString("\n")
	}
	sb.WriteString("    }")
	return sb.String()
}

// Node is an AST node that can be executed.
// SourcePos captures source file position for debugging
type SourcePos struct {
	File   string
	Line   int
	Column int
}

type Node interface {
	Exec(rt *Runtime) (Value, error)
	ToMap() map[string]interface{} // For serialization to JSON/YAML
	ToString() string              // For obtaining source code representation
	GetPos() SourcePos             // For debugger breakpoint mapping
}

// VarRef represents a reference to a runtime variable.
type VarRef struct {
	Name string
	Pos  SourcePos
}

func (v *VarRef) GetPos() SourcePos    { return v.Pos }
func (v *VarRef) SetPos(pos SourcePos) { v.Pos = pos }

// Exec resolves the variable value from the runtime.
func (v *VarRef) Exec(rt *Runtime) (Value, error) {
	// 1. Check current scope chain (walks up to globalScope automatically)
	if rt.currentScope != nil {
		if val, ok := rt.currentScope.Get(v.Name); ok {
			return val, nil
		}
	}

	// 2. Check host objects
	if val, ok := rt.objects[v.Name]; ok {
		return val, nil
	}

	// 3. Check named lists
	if val, ok := rt.lists[v.Name]; ok {
		// Convert list map to MapValue for access
		mapVal := NewMap()
		for k, v := range val {
			mapVal.Set(k, v)
		}
		return mapVal, nil
	}

	// 4. Check named nodes
	if val, ok := rt.nodes[v.Name]; ok {
		return val, nil
	}

	// 5. Check user-defined functions (treat as first-class values)
	if fn, ok := rt.functions[v.Name]; ok {
		return fn, nil
	}

	return nil, fmt.Errorf("variable '%s' not defined", v.Name)
}

// VarRef node
func (v *VarRef) ToMap() map[string]interface{} {
	return map[string]interface{}{
		"_node_type": "VarRef",
		"name":       v.Name,
	}
}

// ToString returns the variable reference as a string.
func (v *VarRef) ToString() string {
	return v.Name
}

// Literal represents a numeric, string, or boolean literal.
type Literal struct {
	Val Value
	Pos SourcePos
}

func (l *Literal) GetPos() SourcePos    { return l.Pos }
func (l *Literal) SetPos(pos SourcePos) { l.Pos = pos }

// Exec returns the literal value.
func (l *Literal) Exec(_ *Runtime) (Value, error) {
	return l.Val, nil
}

// Literal node
func (l *Literal) ToMap() map[string]interface{} {
	return map[string]interface{}{
		"_node_type": "Literal",
		"val":        l.Val,
	}
}

// ToString returns the literal value as a string.
func (l *Literal) ToString() string {
	switch v := l.Val.(type) {
	case Number:
		return fmt.Sprintf("%g", v)
	case Str:
		return fmt.Sprintf("%q", string(v))
	case Bool:
		return fmt.Sprintf("%t", v)
	default:
		return fmt.Sprintf("%v", v) // Fallback for other types
	}
}

// Block represents a sequence of statements to execute.
type Block struct {
	Stmts []Node
	Pos   SourcePos
}

func (b *Block) GetPos() SourcePos    { return b.Pos }
func (b *Block) SetPos(pos SourcePos) { b.Pos = pos }

// Exec runs each statement in order, returning the last value.
func (b *Block) Exec(rt *Runtime) (Value, error) {
	// Execute statements in the current scope
	// Do NOT create a new scope here - let functions/control structures create their own
	var last Value
	if parserDebug {
		fmt.Printf("DEBUG BLOCK.EXEC: Executing block with %d statements, debugger=%v\n", len(b.Stmts), rt.Debugger != nil)
	}
	for _, stmt := range b.Stmts {
		// Debugger support: check breakpoint and update position
		if rt.Debugger != nil {
			pos := stmt.GetPos()
			// DEBUG: Log position information
			if parserDebug {
				fmt.Printf("DEBUG: Statement %T has position %s:%d:%d\n", stmt, pos.File, pos.Line, pos.Column)
				if pos.File != "" && pos.Line > 0 {
					fmt.Printf("DEBUG: Executing statement at %s:%d\n", pos.File, pos.Line)
				}
			}
			rt.Debugger.UpdatePosition(pos.File, pos.Line)

			// Wait while paused at breakpoint
			if rt.Debugger.ShouldBreak(pos.File, pos.Line, rt) {
				// Send debug event with current position BEFORE pausing
				rt.Debugger.SendEvent(DebugEvent{
					Type: "breakpoint",
					File: pos.File,
					Line: pos.Line,
				})
				// Now pause and wait for resume signal
				rt.Debugger.Pause()
			}

			// Handle stepping - only pause if we've moved to a new line
			state := rt.Debugger.GetState()
			if state == DebugStateStepping {
				// Check if we're at a different line than where we paused
				if rt.Debugger.HasMovedToNewLine(pos.File, pos.Line) {
					// Send step event BEFORE pausing
					rt.Debugger.SendEvent(DebugEvent{
						Type: "step",
						File: pos.File,
						Line: pos.Line,
					})
					// Now pause and wait for resume signal
					rt.Debugger.Pause()
				}
			}
		}

		v, err := stmt.Exec(rt)
		if err != nil {
			return nil, err
		}
		last = v
	}

	return last, nil
}

func (b *Block) ToMap() map[string]interface{} {
	stmts := make([]interface{}, len(b.Stmts))
	for i, stmt := range b.Stmts {
		stmts[i] = stmt.ToMap()
	}
	return map[string]interface{}{
		"_node_type": "Block",
		"stmts":      stmts,
	}
}

func (b *Block) ToString() string {
	var sb strings.Builder
	for _, stmt := range b.Stmts {
		// Use ToString to get the source code representation
		if tvar := stmt.ToString(); tvar == "" {
			sb.WriteString("ERROR")
		} else {
			sb.WriteString(tvar)
		}
		sb.WriteString("\n")
	}
	return sb.String()
}

// FuncCall represents a function or method invocation.
type FuncCall struct {
	Name string
	Args []Node
	Pos  SourcePos
}

func (f *FuncCall) GetPos() SourcePos    { return f.Pos }
func (f *FuncCall) SetPos(pos SourcePos) { f.Pos = pos }

// Exec handles built-ins, control-flow functions, and host binding calls.
func (f *FuncCall) Exec(rt *Runtime) (Value, error) {
	// Special handling for declare and declareGlobal - don't evaluate first arg
	if f.Name == "declare" || f.Name == "declareGlobal" || f.Name == "setq" {
		if len(f.Args) < 2 {
			return nil, fmt.Errorf("%s requires at least 2 arguments", f.Name)
		}

		// First argument should be a VarRef (variable name) - don't evaluate it
		varRef, ok := f.Args[0].(*VarRef)
		if !ok {
			return nil, fmt.Errorf("%s: first argument must be a variable name", f.Name)
		}

		// Evaluate remaining arguments normally
		vals := make([]Value, len(f.Args))
		vals[0] = Str(varRef.Name) // Convert variable name to string

		for i := 1; i < len(f.Args); i++ {
			v, err := f.Args[i].Exec(rt)
			if err != nil {
				return nil, err
			}
			vals[i] = v
		}

		// Call the registered function
		if h, ok := rt.funcs[f.Name]; ok {
			return h(vals...)
		}
		return nil, fmt.Errorf("undefined function '%s'", f.Name)
	}
	// Special handling for createTransform - first arg is naked symbol name
	if f.Name == "createTransform" {
		if len(f.Args) < 1 {
			return nil, fmt.Errorf("createTransform requires 1 argument: transformName")
		}

		// First argument should be a VarRef (transform name) - don't evaluate it
		varRef, ok := f.Args[0].(*VarRef)
		if !ok {
			return nil, fmt.Errorf("createTransform: first argument must be a transform name")
		}

		// Convert variable name to string and call the registered function
		vals := []Value{Str(varRef.Name)}
		if h, ok := rt.funcs[f.Name]; ok {
			return h(vals...)
		}
		return nil, fmt.Errorf("undefined function '%s'", f.Name)
	}
	// Special-case `while(condition) { block }`
	if f.Name == "while" {
		if len(f.Args) != 2 {
			return nil, errors.New("while requires 2 args: condition and block")
		}
		condNode := f.Args[0]
		blockNode, ok := f.Args[1].(*Block)
		if !ok {
			return nil, errors.New("while: second arg must be a block")
		}
		var last Value
		for {
			cv, err := condNode.Exec(rt)
			if err != nil {
				return nil, err
			}
			if !boolify(cv) {
				break
			}
			last, err = blockNode.Exec(rt)
			if err != nil {
				return nil, err
			}
		}
		return last, nil
	}

	// Evaluate all arguments for other calls
	vals := make([]Value, len(f.Args))
	for i, arg := range f.Args {
		v, err := arg.Exec(rt)
		if err != nil {
			return nil, err
		}
		vals[i] = v
	}

	// Host object method: obj.Method()
	if parts := strings.SplitN(f.Name, ".", 2); len(parts) == 2 {
		if obj, ok := rt.objects[parts[0]]; ok {
			return rt.CallHostMethod(obj, parts[1], vals)
		}
	}

	// Built-in function dispatch
	if h, ok := rt.funcs[f.Name]; ok {
		return h(vals...)
	}
	// UDF function call
	if fn, ok := rt.functions[f.Name]; ok {
		return rt.funcs["call"](fn, vals)
	}
	return nil, fmt.Errorf("undefined function '%s'", f.Name)
}

// FuncCall node
func (f *FuncCall) ToMap() map[string]interface{} {
	args := make([]interface{}, len(f.Args))
	for i, arg := range f.Args {
		args[i] = arg.ToMap()
	}
	return map[string]interface{}{
		"_node_type": "FuncCall",
		"name":       f.Name,
		"args":       args,
	}
}

func (f *FuncCall) ToString() string {
	args := make([]string, len(f.Args))
	for i, arg := range f.Args {
		if tvar := arg.ToString(); tvar == "" {
			args[i] = "ERROR"
		} else {
			args[i] = tvar
		}
	}
	return fmt.Sprintf("%s(%s)", f.Name, strings.Join(args, ", ")) // Remove the semicolon
}

// boolify converts a Value to a Go bool.
func boolify(v Value) bool {
	switch b := v.(type) {
	case Bool:
		return bool(b)
	case Number:
		return b != 0
	case Str:
		return string(b) != ""
	}
	return false
}

func (n Number) Abs() Number {
	if n < 0 {
		return -n
	}
	return n
}
