package chariot

import (
	"fmt"
	"io/ioutil"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	cfg "github.com/bhouse1273/go-chariot/configs"
)

// FunctionExtractor analyzes TreeNodes to extract all function calls
type FunctionExtractor struct {
	FoundFunctions map[string]bool
	FunctionRegex  *regexp.Regexp
}

// NewFunctionExtractor creates a new function extractor
func NewFunctionExtractor() *FunctionExtractor {
	// Regex to match function calls: functionName(...)
	funcRegex := regexp.MustCompile(`\b([a-zA-Z_][a-zA-Z0-9_]*)\s*\(`)

	return &FunctionExtractor{
		FoundFunctions: make(map[string]bool),
		FunctionRegex:  funcRegex,
	}
}

// ExtractFromTreeNode recursively extracts function calls from a TreeNode
func (fe *FunctionExtractor) ExtractFromTreeNode(node TreeNode) error {
	if node == nil {
		return nil
	}

	// Extract from node name
	if name := node.Name(); name != "" {
		fe.ExtractFromText(name)
	}

	// Extract from all attributes
	for attrName, attrValue := range node.GetAttributes() {
		fe.ExtractFromText(attrName)
		fe.ExtractFromValue(attrValue)
	}

	// Extract from text content (for XML nodes)
	if textContent := fe.GetTextContent(node); textContent != "" {
		fe.ExtractFromText(textContent)
	}

	// Extract from metadata
	if metaHolder, ok := node.(interface{ GetAllMetadata() map[string]interface{} }); ok {
		for key, value := range metaHolder.GetAllMetadata() {
			fe.ExtractFromText(key)
			fe.ExtractFromInterface(value)
		}
	}

	// Recursively process children
	for _, child := range node.GetChildren() {
		if err := fe.ExtractFromTreeNode(child); err != nil {
			return err
		}
	}

	return nil
}

// ExtractFromTreeFile loads a tree file and extracts function calls
func (fe *FunctionExtractor) ExtractFromTreeFile(filename string) error {
	// Use the existing global tree serializer service
	serializer := getTreeSerializerService()

	// Determine format from file extension
	ext := strings.ToLower(filepath.Ext(filename))
	var format string
	switch ext {
	case ".json":
		format = "json"
	case ".xml":
		format = "xml"
	case ".yaml", ".yml":
		format = "yaml"
	case ".gob":
		format = "gob"
	default:
		return fmt.Errorf("unsupported file format: %s", ext)
	}
	_ = format

	// Load the tree using existing serializer
	tree, err := serializer.LoadAsync(filename, nil, 30*time.Second)
	if err != nil {
		// If tree loading fails (e.g., for simple YAML/text files),
		// try to read as plain text and extract functions
		return fe.ExtractFromPlainTextFile(filename)
	}

	return fe.ExtractFromTreeNode(tree)
}

// ExtractFromPlainTextFile reads a file as plain text and extracts functions
func (fe *FunctionExtractor) ExtractFromPlainTextFile(filename string) error {
	// Get the base tree path from config
	var basePath string
	if cfg.ChariotConfig.TreePath != "" {
		basePath = cfg.ChariotConfig.TreePath
	} else {
		basePath = "." // fallback to current directory
	}

	// Resolve the full file path
	fullPath := filepath.Join(basePath, filename)

	// Read the file content
	content, err := ioutil.ReadFile(fullPath)
	if err != nil {
		return fmt.Errorf("failed to read file %s: %v", filename, err)
	}

	// Extract functions from the text content
	fe.ExtractFromText(string(content))
	return nil
}

// ExtractFromMultipleFiles processes multiple tree files
func (fe *FunctionExtractor) ExtractFromMultipleFiles(filenames []string) error {
	for _, filename := range filenames {
		if err := fe.ExtractFromTreeFile(filename); err != nil {
			return err
		}
	}
	return nil
}

// ExtractFromDirectory processes all tree files in a directory
func (fe *FunctionExtractor) ExtractFromDirectory(dirPath string) error {
	// Get the base tree path from config, similar to tree serializer
	var basePath string
	if cfg.ChariotConfig.TreePath != "" {
		basePath = cfg.ChariotConfig.TreePath
	} else {
		basePath = "." // fallback to current directory
	}

	// Resolve the full directory path
	fullDirPath := filepath.Join(basePath, dirPath)

	files, err := filepath.Glob(filepath.Join(fullDirPath, "*.json"))
	if err != nil {
		return err
	}

	yamlFiles, err := filepath.Glob(filepath.Join(fullDirPath, "*.yaml"))
	if err != nil {
		return err
	}
	files = append(files, yamlFiles...)

	ymlFiles, err := filepath.Glob(filepath.Join(fullDirPath, "*.yml"))
	if err != nil {
		return err
	}
	files = append(files, ymlFiles...)

	// Convert full paths back to relative paths for the serializer
	relativeFiles := make([]string, len(files))
	for i, file := range files {
		if rel, err := filepath.Rel(basePath, file); err == nil {
			relativeFiles[i] = rel
		} else {
			relativeFiles[i] = file // fallback to absolute path
		}
	}

	return fe.ExtractFromMultipleFiles(relativeFiles)
}

// Update ExtractFromInterface to handle serialized AST nodes:

// ExtractFromInterface extracts function calls from interface{} values
func (fe *FunctionExtractor) ExtractFromInterface(value interface{}) {
	switch v := value.(type) {
	case string:
		fe.ExtractFromText(v)
	case []interface{}:
		for _, item := range v {
			fe.ExtractFromInterface(item)
		}
	case map[string]interface{}:
		for key, val := range v {
			fe.ExtractFromText(key)
			fe.ExtractFromInterface(val)
		}

		// Handle serialized AST nodes
		if nodeType, exists := v["_node_type"]; exists {
			fe.ExtractFromSerializedNode(nodeType.(string), v)
		}
	case TreeNode:
		fe.ExtractFromTreeNode(v)
	}
}

// ExtractFromValue extracts function calls from a Chariot Value
func (fe *FunctionExtractor) ExtractFromValue(value Value) {
	switch v := value.(type) {
	case Str:
		fe.ExtractFromText(string(v))
	case *ArrayValue:
		for i := 0; i < v.Length(); i++ {
			fe.ExtractFromValue(v.Get(i))
		}
	case TreeNode:
		fe.ExtractFromTreeNode(v)
	case *FunctionValue:
		// Extract from function body - Body is a single Node
		if v.Body != nil {
			fe.ExtractFromNode(v.Body)
		}
	case map[string]Value:
		// Handle maps of Values (like JSON attributes)
		for key, val := range v {
			fe.ExtractFromText(key)
			fe.ExtractFromValue(val)
		}
	default:
		// Try to handle as interface{} for other types
		if iface, ok := value.(interface{}); ok {
			fe.ExtractFromInterface(iface)
		}
	}
}

// Also add the helper method to extract from AST nodes:
func (fe *FunctionExtractor) ExtractFromNode(node Node) {
	switch n := node.(type) {
	case *Block:
		// Extract from all statements in the block
		for _, stmt := range n.Stmts {
			fe.ExtractFromNode(stmt)
		}
	case *FuncCall:
		// Extract the function name
		fe.FoundFunctions[n.Name] = true
		// Extract from arguments
		for _, arg := range n.Args {
			fe.ExtractFromNode(arg)
		}
	case *VarRef:
		// Variable references might be function names in dynamic calls
		// But we'll be conservative and not extract these
	case *Literal:
		// Extract from string literals that might contain function calls
		if str, ok := n.Val.(Str); ok {
			fe.ExtractFromText(string(str))
		}
	case *IfNode:
		// Extract from condition and branches
		fe.ExtractFromNode(n.Condition)
		for _, stmt := range n.TrueBranch {
			fe.ExtractFromNode(stmt)
		}
		for _, stmt := range n.FalseBranch {
			fe.ExtractFromNode(stmt)
		}
	case *WhileNode:
		// Extract from condition and body
		fe.ExtractFromNode(n.Condition)
		for _, stmt := range n.Body {
			fe.ExtractFromNode(stmt)
		}
	case *SwitchNode:
		// Extract from test expression and cases
		if n.TestExpr != nil {
			fe.ExtractFromNode(n.TestExpr)
		}
		for _, caseNode := range n.Cases {
			fe.ExtractFromNode(caseNode.Condition)
			fe.ExtractFromNode(caseNode.Body)
		}
		if n.DefaultCase != nil {
			fe.ExtractFromNode(n.DefaultCase.Body)
		}
	case *FunctionDefNode:
		// Extract from function body
		fe.ExtractFromNode(n.Body)
	case *FunctionCallNode:
		// Extract from function expression and arguments
		fe.ExtractFromNode(n.FuncExpr)
		for _, arg := range n.Args {
			fe.ExtractFromNode(arg)
		}
	case *ArrayLiteralNode:
		// Extract from array elements
		for _, elem := range n.Elements {
			fe.ExtractFromNode(elem)
		}
	}
}

// ExtractFromSerializedNode extracts function calls from serialized AST nodes
func (fe *FunctionExtractor) ExtractFromSerializedNode(nodeType string, data map[string]interface{}) {
	switch nodeType {
	case "FuncCall":
		// Extract function name
		if name, exists := data["name"]; exists {
			if nameStr, ok := name.(string); ok {
				fe.FoundFunctions[nameStr] = true
			}
		}
		// Extract from arguments
		if args, exists := data["args"]; exists {
			fe.ExtractFromInterface(args)
		}
	case "Block":
		// Extract from statements
		if stmts, exists := data["stmts"]; exists {
			fe.ExtractFromInterface(stmts)
		}
	case "IfNode":
		// Extract from condition and branches
		if condition, exists := data["condition"]; exists {
			fe.ExtractFromInterface(condition)
		}
		if trueBranch, exists := data["trueBranch"]; exists {
			fe.ExtractFromInterface(trueBranch)
		}
		if falseBranch, exists := data["falseBranch"]; exists {
			fe.ExtractFromInterface(falseBranch)
		}
	case "WhileNode":
		// Extract from condition and body
		if condition, exists := data["condition"]; exists {
			fe.ExtractFromInterface(condition)
		}
		if body, exists := data["body"]; exists {
			fe.ExtractFromInterface(body)
		}
	case "SwitchNode":
		// Extract from test expression and cases
		if testExpr, exists := data["testExpr"]; exists {
			fe.ExtractFromInterface(testExpr)
		}
		if cases, exists := data["cases"]; exists {
			fe.ExtractFromInterface(cases)
		}
		if defaultCase, exists := data["defaultCase"]; exists {
			fe.ExtractFromInterface(defaultCase)
		}
	case "CaseNode":
		// Extract from condition and body
		if condition, exists := data["condition"]; exists {
			fe.ExtractFromInterface(condition)
		}
		if body, exists := data["body"]; exists {
			fe.ExtractFromInterface(body)
		}
	case "DefaultNode":
		// Extract from body
		if body, exists := data["body"]; exists {
			fe.ExtractFromInterface(body)
		}
	case "FunctionDefNode":
		// Extract from body
		if body, exists := data["body"]; exists {
			fe.ExtractFromInterface(body)
		}
	case "FunctionCallNode":
		// Extract from function expression and arguments
		if funcExpr, exists := data["funcExpr"]; exists {
			fe.ExtractFromInterface(funcExpr)
		}
		if args, exists := data["args"]; exists {
			fe.ExtractFromInterface(args)
		}
	case "ArrayLiteralNode":
		// Extract from elements
		if elements, exists := data["elements"]; exists {
			fe.ExtractFromInterface(elements)
		}
	case "Literal":
		// Extract from string literals
		if val, exists := data["val"]; exists {
			if valStr, ok := val.(string); ok {
				fe.ExtractFromText(valStr)
			}
		}
	}
}

// ExtractFromText extracts function calls from text content
func (fe *FunctionExtractor) ExtractFromText(text string) {
	matches := fe.FunctionRegex.FindAllStringSubmatch(text, -1)
	for _, match := range matches {
		if len(match) > 1 {
			funcName := match[1]
			// Filter out common non-function words
			if fe.IsLikelyFunction(funcName) {
				fe.FoundFunctions[funcName] = true
			}
		}
	}
}

// IsLikelyFunction filters out common words that aren't functions
func (fe *FunctionExtractor) IsLikelyFunction(name string) bool {
	// Skip common keywords that aren't functions
	keywords := map[string]bool{
		"else": true, "break": true, "continue": true,
		"true": true, "false": true, "null": true,
	}

	// Skip single letters (likely variables)
	if len(name) <= 1 {
		return false
	}

	// Skip if it's a keyword
	if keywords[strings.ToLower(name)] {
		return false
	}

	// Skip if it looks like a property access (contains dots)
	if strings.Contains(name, ".") {
		return false
	}

	return true
}

// getTextContent extracts text content from various node types
func (fe *FunctionExtractor) GetTextContent(node TreeNode) string {
	// Try to get text content if the node supports it
	if textNode, ok := node.(interface{ GetText() string }); ok {
		return textNode.GetText()
	}

	// For JSON nodes, check if there's a text attribute
	if textAttr, exists := node.GetAttribute("text"); exists {
		if textStr, ok := textAttr.(Str); ok {
			return string(textStr)
		}
	}

	return ""
}

// GetExtractedFunctions returns the list of extracted function names
func (fe *FunctionExtractor) GetExtractedFunctions() []string {
	var functions []string
	for funcName := range fe.FoundFunctions {
		functions = append(functions, funcName)
	}
	sort.Strings(functions)
	return functions
}

// GetExtractedFunctionCount returns the number of unique functions found
func (fe *FunctionExtractor) GetExtractedFunctionCount() int {
	return len(fe.FoundFunctions)
}

// Reset clears all extracted functions
func (fe *FunctionExtractor) Reset() {
	fe.FoundFunctions = make(map[string]bool)
}

// ValidateExtractedFunctions checks if all extracted functions exist in the master registry
func (fe *FunctionExtractor) ValidateExtractedFunctions() ([]string, []string, error) {
	var valid []string
	var invalid []string

	for funcName := range fe.FoundFunctions {
		if HasFunction(funcName) {
			valid = append(valid, funcName)
		} else {
			invalid = append(invalid, funcName)
		}
	}

	sort.Strings(valid)
	sort.Strings(invalid)

	return valid, invalid, nil
}
