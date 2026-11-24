// parser.go
// Implements the lexer and parser for Chariot scripts.
package chariot

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"unicode"
)

// TokenType identifies the kind of token.
type TokenType int

const (
	TOK_EOF TokenType = iota
	TOK_IDENT
	TOK_NUMBER
	TOK_STRING
	TOK_LPAREN   // (
	TOK_RPAREN   // )
	TOK_LBRACE   // {
	TOK_RBRACE   // }
	TOK_COMMA    // ,
	TOK_LBRACKET // [
	TOK_RBRACKET // ]
	TOK_DOT      // .
	TOK_INVALID
	TOK_SEMICOLON // legacy statement separator, never an argument separator
)

// Token holds the type and literal text.
type Token struct {
	Type   TokenType
	Text   string
	Line   int
	Column int
	// NewlineBefore records a line break in skipped whitespace/comments, not
	// inside the preceding token (which may itself be a multiline string).
	NewlineBefore bool
}

// Lexer splits source into a stream of Tokens.
type Lexer struct {
	src  string
	pos  int
	line int // Track current line number
	col  int // Track current column
}

// NewLexer creates a new Lexer for the given source.
func NewLexer(src string) *Lexer {
	return &Lexer{src: src, line: 1, col: 1}
}

// getLineCol returns the current line and column
func (lx *Lexer) getLineCol() (int, int) {
	return lx.line, lx.col
}

// Next returns the next Token from the input.
func (lx *Lexer) Next() (tok Token) {
	// Track all consumed bytes, including comments and multiline strings. Token
	// positions describe the start, not the end of the lexer lookahead.
	start := lx.pos
	defer func() {
		for _, c := range lx.src[start:lx.pos] {
			if c == '\n' {
				lx.line++
				lx.col = 1
			} else {
				lx.col++
			}
		}
	}()
	s := lx.src
	// Skip whitespace and comments iteratively, including consecutive comments.
	for {
		for lx.pos < len(s) && unicode.IsSpace(rune(s[lx.pos])) {
			lx.pos++
		}
		if lx.pos+1 >= len(s) || s[lx.pos:lx.pos+2] != "//" {
			break
		}
		lx.pos += 2
		for lx.pos < len(s) && s[lx.pos] != '\n' {
			lx.pos++
		}
	}
	line, col := lx.line, lx.col
	for _, c := range s[start:lx.pos] {
		if c == '\n' {
			line++
			col = 1
		} else {
			col++
		}
	}
	newlineBefore := strings.ContainsAny(s[start:lx.pos], "\r\n")
	defer func() {
		tok.Line, tok.Column = line, col
		tok.NewlineBefore = newlineBefore
	}()
	if lx.pos >= len(s) {
		return Token{Type: TOK_EOF}
	}
	c := s[lx.pos]
	switch {
	case c == '`':
		// Handle backtick strings
		return lx.tokenizeBacktickString()
	case c == '/':
		lx.pos++
		return Token{Type: TOK_INVALID, Text: "/"}
	case isLetter(rune(c)):
		start := lx.pos
		for lx.pos < len(s) && (isLetter(rune(s[lx.pos])) || isDigit(s[lx.pos])) {
			lx.pos++
		}
		return Token{Type: TOK_IDENT, Text: s[start:lx.pos]}
	case isDigit(c) || (c == '-' && lx.pos+1 < len(s) && isDigit(s[lx.pos+1])):
		begin := lx.pos
		if c == '-' {
			lx.pos++
		}
		for lx.pos < len(s) && isDigit(s[lx.pos]) {
			lx.pos++
		}
		if lx.pos < len(s) && s[lx.pos] == '.' {
			// A dot followed by an identifier starts member access. Retain
			// legacy trailing-decimal syntax (1.) and negative decimals.
			if lx.pos+1 == len(s) || !isLetter(rune(s[lx.pos+1])) {
				lx.pos++
				for lx.pos < len(s) && isDigit(s[lx.pos]) {
					lx.pos++
				}
			}
		}
		if lx.pos < len(s) && (s[lx.pos] == 'e' || s[lx.pos] == 'E') {
			lx.pos++
			if lx.pos < len(s) && (s[lx.pos] == '+' || s[lx.pos] == '-') {
				lx.pos++
			}
			for lx.pos < len(s) && isDigit(s[lx.pos]) {
				lx.pos++
			}
		}
		return Token{Type: TOK_NUMBER, Text: s[begin:lx.pos]}

	case c == '\'':
		// Single-quoted string literal
		lx.pos++ // skip opening quote
		content := lx.parseStringContent('\'')
		if lx.pos < len(s) && s[lx.pos] == '\'' {
			lx.pos++ // skip closing quote
		} else {
			return Token{Type: TOK_INVALID, Text: "unterminated string"}
		}
		return Token{Type: TOK_STRING, Text: content}

	case c == '"':
		// Double-quoted string literal
		lx.pos++ // skip opening quote
		content := lx.parseStringContent('"')
		if lx.pos < len(s) && s[lx.pos] == '"' {
			lx.pos++ // skip closing quote
		} else {
			return Token{Type: TOK_INVALID, Text: "unterminated string"}
		}
		return Token{Type: TOK_STRING, Text: content}
	case c == '(':
		lx.pos++
		return Token{Type: TOK_LPAREN}
	case c == ')':
		lx.pos++
		return Token{Type: TOK_RPAREN}
	case c == '{':
		lx.pos++
		return Token{Type: TOK_LBRACE}
	case c == '}':
		lx.pos++
		return Token{Type: TOK_RBRACE}
	case c == ',':
		lx.pos++
		return Token{Type: TOK_COMMA}
	case c == '[':
		lx.pos++
		return Token{Type: TOK_LBRACKET}
	case c == ']':
		lx.pos++
		return Token{Type: TOK_RBRACKET}
	case c == '.':
		lx.pos++
		return Token{Type: TOK_DOT, Text: "."}
	case c == ';':
		lx.pos++
		return Token{Type: TOK_SEMICOLON, Text: ";"}
	default:
		lx.pos++
		return Token{Type: TOK_INVALID, Text: string(c)}
	}
}

// isLetter reports whether r is an acceptable identifier start or part.
func isLetter(r rune) bool {
	return unicode.IsLetter(r) || r == '_'
}

// isDigit reports whether b is a decimal digit.
func isDigit(b byte) bool {
	return '0' <= b && b <= '9'
}

// Parser holds state for recursive descent parsing.
type Parser struct {
	lx              *Lexer
	cur             Token
	currentPosition int
	filename        string // Track source filename for debugging
}

// NewParser constructs a Parser from source text.
func NewParser(src string) *Parser {
	p := &Parser{lx: NewLexer(src), filename: "main.ch"} // Default filename
	p.next()
	return p
}

// NewParserWithFilename creates a parser with a specific filename for debugging
func NewParserWithFilename(src string, filename string) *Parser {
	p := &Parser{lx: NewLexer(src), filename: filename}
	p.next()
	return p
}

// getCurrentPos returns the current source position
func (p *Parser) getCurrentPos() SourcePos {
	return SourcePos{File: p.filename, Line: p.cur.Line, Column: p.cur.Column}
}

// next reads the next token into cur.
func (p *Parser) next() {
	p.cur = p.lx.Next()
}

// ParseCode parses a code string and returns a Node representation
func (p *Parser) ParseCode(code string) (Node, error) {
	// Create a new parser with the code
	parser := NewParser(code)

	// Parse the program
	block, err := parser.parseProgram()
	if err != nil {
		return nil, err
	}

	// Return the parsed Block as Node
	return block, nil
}

func (l *Lexer) tokenizeBacktickString() Token {
	l.pos++ // skip opening backtick

	var content strings.Builder

	for l.pos < len(l.src) {
		ch := l.src[l.pos]

		if ch == '`' {
			// Found closing backtick
			l.pos++
			return Token{
				Type: TOK_STRING,
				Text: content.String(),
			}
		}

		// Add character as-is (no escape processing)
		content.WriteByte(ch)
		l.pos++
	}

	// If we get here, we have an unterminated backtick string
	// For now, return what we have (you might want to handle this differently)
	return Token{
		Type: TOK_INVALID,
		Text: "unterminated backtick string",
	}
}

/*
// Helper function to convert a Block to a TreeNode
func convertBlockToTreeNode(block *Block) TreeNode {
	programNode := NewTreeNode("program")
	programNode.SetAttribute("type", Str("program"))

	// Add each statement as a child node
	for _, stmt := range block.Stmts {
		programNode.AddChild(convertNodeToTreeNode(stmt))
	}

	return programNode
}

// Helper function to convert a Node to a TreeNode
func convertNodeToTreeNode(node Node) TreeNode {
	switch n := node.(type) {
	case *FuncCall:
		callNode := NewTreeNode("call")
		callNode.SetAttribute("type", Str("call"))
		callNode.SetAttribute("name", Str(n.Name))

		// Add arguments as children
		for _, arg := range n.Args {
			callNode.AddChild(convertNodeToTreeNode(arg))
		}

		return callNode

	case *VarRef:
		varNode := NewTreeNode("variable")
		varNode.SetAttribute("type", Str("variable"))
		varNode.SetAttribute("name", Str(n.Name))
		return varNode

	case *Literal:
		literalNode := NewTreeNode("literal")
		literalNode.SetAttribute("type", Str("literal"))
		literalNode.SetAttribute("value", n.Val)
		return literalNode

	case *Block:
		blockNode := NewTreeNode("block")
		blockNode.SetAttribute("type", Str("block"))

		// Add statements as children
		for _, stmt := range n.Stmts {
			blockNode.AddChild(convertNodeToTreeNode(stmt))
		}

		return blockNode

	default:
		// Create a placeholder for unsupported node types
		unknownNode := NewTreeNode("unknown")
		unknownNode.SetAttribute("type", Str("unknown"))
		return unknownNode
	}
}
*/

var parserDebug = func() bool {
	switch strings.ToLower(os.Getenv("CH_PARSER_DEBUG")) {
	case "1", "true", "yes", "on":
		return true
	default:
		return false
	}
}()

// parseProgram parses a sequence of expressions until EOF.
func (p *Parser) parseProgram() (*Block, error) {
	blk := &Block{}
	for p.cur.Type != TOK_EOF {
		if p.cur.Type == TOK_SEMICOLON {
			p.next()
			continue
		}
		// Capture position before parsing expression
		pos := p.getCurrentPos()
		if parserDebug {
			fmt.Printf("DEBUG PARSER: About to parse expr at %s:%d:%d, current token: %v\n", pos.File, pos.Line, pos.Column, p.cur.Type)
		}

		expr, err := p.parseExpr()
		if err != nil {
			return nil, err
		}

		// Set position on the parsed node if it supports it
		if posNode, ok := expr.(interface{ SetPos(SourcePos) }); ok {
			posNode.SetPos(pos)
			if parserDebug {
				fmt.Printf("DEBUG PARSER: Set position on %T to %s:%d:%d\n", expr, pos.File, pos.Line, pos.Column)
			}
		} else if parserDebug {
			fmt.Printf("DEBUG PARSER: Node %T does not support SetPos\n", expr)
		}

		blk.Stmts = append(blk.Stmts, expr)
	}
	if parserDebug {
		fmt.Printf("DEBUG PARSER: Finished parsing, total statements: %d\n", len(blk.Stmts))
	}
	return blk, nil
}

// parseExpr handles variable refs, literals, function calls, and blocks.
func (p *Parser) parseExpr() (Node, error) {
	pos := p.getCurrentPos()
	node, err := p.parsePrimary()
	if err != nil {
		return nil, fmt.Errorf("%s:%d:%d: %w", pos.File, pos.Line, pos.Column, err)
	}
	if n, ok := node.(interface{ SetPos(SourcePos) }); ok {
		n.SetPos(pos)
	}
	for {
		switch p.cur.Type {
		case TOK_DOT:
			p.next()
			if p.cur.Type != TOK_IDENT {
				return nil, fmt.Errorf("%s:%d:%d: expected member name after '.'", p.filename, p.cur.Line, p.cur.Column)
			}
			node = &MemberAccessNode{Receiver: node, Name: p.cur.Text, Pos: pos}
			p.next()
		case TOK_LPAREN:
			// A completed expression cannot acquire a call across a newline.
			// This also applies inside argument lists; the enclosing parser keeps
			// consuming arguments/grouping rather than ending the outer call.
			if p.cur.NewlineBefore {
				return node, nil
			}
			args, err := p.parseCallArgs()
			if err != nil {
				return nil, err
			}
			node = &FunctionCallNode{FuncExpr: node, Args: args, Pos: pos}
		default:
			return node, nil
		}
	}
}

func (p *Parser) parseCallArgs() ([]Node, error) {
	p.next() // '('
	args := []Node{}
	for p.cur.Type != TOK_RPAREN {
		if p.cur.Type == TOK_EOF {
			return nil, fmt.Errorf("%s:%d:%d: expected ')'", p.filename, p.cur.Line, p.cur.Column)
		}
		arg, err := p.parseExpr()
		if err != nil {
			return nil, err
		}
		args = append(args, arg)
		if p.cur.Type == TOK_COMMA {
			p.next()
		}
		// Legacy Chariot accepts whitespace-separated arguments. Preserve it;
		// unsupported punctuation still fails in parseExpr instead of vanishing.
	}
	p.next()
	return args, nil
}

func (p *Parser) parsePrimary() (Node, error) {
	if p.cur.Type == TOK_INVALID {
		return nil, fmt.Errorf("invalid token %q", p.cur.Text)
	}
	if p.cur.Type == TOK_LPAREN {
		p.next()
		node, err := p.parseExpr()
		if err != nil {
			return nil, err
		}
		if p.cur.Type != TOK_RPAREN {
			return nil, fmt.Errorf("expected ')' after grouped expression")
		}
		p.next()
		return node, nil
	}
	// identifier: variable or function call
	if p.cur.Type == TOK_IDENT {
		ident := p.cur.Text

		// Check for control flow keywords
		switch ident {
		case "if":
			p.next() // consume "if"
			return p.parseIfStatement()
		case "while":
			p.next() // consume "while"
			return p.parseWhileStatement()
		case "switch":
			p.next() // consume "switch"
			return p.parseSwitch()
		case "func":
			// Function definition found - do NOT consume "func" here
			return p.parseFunction()
		}

		p.next()
		// function call?
		if p.cur.Type == TOK_LPAREN && !p.cur.NewlineBefore {
			args, err := p.parseCallArgs()
			if err != nil {
				return nil, err
			}
			// optional block for constructs like while
			// Declaration/member names are symbols, not variable reads.
			nameIndex := -1
			if ident == "field" || ident == "requires" {
				nameIndex = 0
			}
			if ident == "method" {
				nameIndex = 1
			}
			if nameIndex >= 0 && len(args) > nameIndex {
				if name, ok := args[nameIndex].(*VarRef); ok {
					args[nameIndex] = &Literal{Val: Str(name.Name)}
				}
			}
			if p.cur.Type == TOK_LBRACE {
				blk, err := p.parseBlock()
				if err != nil {
					return nil, err
				}
				args = append(args, blk)
			}
			return &FuncCall{Name: ident, Args: args}, nil
		}
		// bare identifier => variable reference
		return &VarRef{Name: ident}, nil
	}
	// number literal
	if p.cur.Type == TOK_NUMBER {
		f, err := strconv.ParseFloat(p.cur.Text, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid number %q", p.cur.Text)
		}
		node := &Literal{Val: Number(f)}
		p.next()
		return node, nil
	}
	// string literal
	if p.cur.Type == TOK_STRING {
		node := &Literal{Val: Str(p.cur.Text)}
		p.next()
		return node, nil
	}
	// Check for array literal
	if p.cur.Type == TOK_LBRACKET {
		return p.parseArrayLiteral()
	}

	return nil, fmt.Errorf("unexpected token %v", p.cur)
}

func (p *Parser) parseSwitch() (Node, error) {
	startPos := p.currentPosition

	// Parse optional test expression: switch(expr) or switch()
	if p.cur.Type != TOK_LPAREN {
		return nil, errors.New("expected '(' after 'switch'")
	}
	p.next() // consume '('

	var testExpr Node
	if p.cur.Type != TOK_RPAREN {
		expr, err := p.parseExpr()
		if err != nil {
			return nil, err
		}
		testExpr = expr
	}

	if p.cur.Type != TOK_RPAREN {
		return nil, errors.New("expected ')' after switch expression")
	}
	p.next() // consume ')'

	// Parse the block
	if p.cur.Type != TOK_LBRACE {
		return nil, errors.New("expected '{' after switch(...)")
	}
	p.next() // consume '{'

	var cases []*CaseNode
	var defaultCase *DefaultNode

	for p.cur.Type != TOK_RBRACE && p.cur.Type != TOK_EOF {
		if p.cur.Type == TOK_IDENT {
			switch p.cur.Text {
			case "case":
				p.next() // consume "case"
				caseNode, err := p.parseCase()
				if err != nil {
					return nil, err
				}
				cases = append(cases, caseNode)

			case "default":
				if defaultCase != nil {
					return nil, errors.New("multiple default cases in switch")
				}
				p.next() // consume "default"
				defNode, err := p.parseDefault()
				if err != nil {
					return nil, err
				}
				defaultCase = defNode

			default:
				return nil, fmt.Errorf("only 'case' and 'default' are allowed in switch block, got '%s'", p.cur.Text)
			}
		} else {
			return nil, errors.New("expected 'case' or 'default' in switch block")
		}
	}

	if p.cur.Type != TOK_RBRACE {
		return nil, errors.New("expected '}' after switch block")
	}
	p.next() // consume '}'

	if len(cases) == 0 && defaultCase == nil {
		return nil, errors.New("switch block must contain at least one case or default")
	}

	return &SwitchNode{
		TestExpr:    testExpr,
		Cases:       cases,
		DefaultCase: defaultCase,
		Position:    startPos,
	}, nil
}

func (p *Parser) parseCase() (*CaseNode, error) {
	// Parse case(expr) { ... }
	if p.cur.Type != TOK_LPAREN {
		return nil, errors.New("expected '(' after 'case'")
	}
	p.next() // consume '('

	expr, err := p.parseExpr()
	if err != nil {
		return nil, err
	}

	if p.cur.Type != TOK_RPAREN {
		return nil, errors.New("expected ')' after case expression")
	}
	p.next() // consume ')'

	body, err := p.parseBlock()
	if err != nil {
		return nil, err
	}

	return &CaseNode{
		Condition: expr,
		Body:      body,
	}, nil
}

func (p *Parser) parseDefault() (*DefaultNode, error) {
	// Parse default() { ... }
	if p.cur.Type != TOK_LPAREN {
		return nil, errors.New("expected '(' after 'default'")
	}
	p.next() // consume '('

	if p.cur.Type != TOK_RPAREN {
		return nil, errors.New("expected ')' after 'default'")
	}
	p.next() // consume ')'

	body, err := p.parseBlock()
	if err != nil {
		return nil, err
	}

	return &DefaultNode{
		Body: body,
	}, nil
}

func (p *Parser) parseIfStatement() (Node, error) {
	startPos := p.currentPosition

	// Parse condition in parentheses
	if err := p.consume("(", "Expected '(' after 'if'"); err != nil {
		return nil, err
	}

	condition, err := p.parseExpr()
	if err != nil {
		return nil, err
	}

	if err := p.consume(")", "Expected ')' after condition"); err != nil {
		return nil, err
	}

	// Parse true branch
	trueBlock, err := p.parseBlock()
	if err != nil {
		return nil, err
	}

	// Create IfNode first - MOVED THIS UP
	ifNode := &IfNode{
		Condition:  condition,
		TrueBranch: trueBlock.Stmts,
		Position:   startPos,
	}

	// Check for else
	if p.cur.Type == TOK_IDENT && p.cur.Text == "else" {
		p.next() // consume else

		// Check for else-if pattern
		if p.cur.Type == TOK_IDENT && p.cur.Text == "if" {
			p.next() // consume "if"
			// Parse the else-if condition and block recursively
			elseIfNode, err := p.parseIfStatement()
			if err != nil {
				return nil, err
			}

			// Now this is valid because ifNode exists
			ifNode.FalseBranch = []Node{elseIfNode}
		} else {
			// Regular else block
			falseBlock, err := p.parseBlock()
			if err != nil {
				return nil, err
			}
			ifNode.FalseBranch = falseBlock.Stmts
		}
	}

	return ifNode, nil
}

func (p *Parser) parseWhileStatement() (Node, error) {
	startPos := p.currentPosition

	// Parse condition in parentheses
	if err := p.consume("(", "Expected '(' after 'while'"); err != nil {
		return nil, err
	}

	condition, err := p.parseExpr()
	if err != nil {
		return nil, err
	}

	if err := p.consume(")", "Expected ')' after condition"); err != nil {
		return nil, err
	}

	// Don't consume the opening brace here, let parseBlock handle it
	block, err := p.parseBlock()
	if err != nil {
		return nil, err
	}

	return &WhileNode{
		Condition: condition,
		Body:      block.Stmts,
		Position:  startPos,
	}, nil
}

func (p *Parser) parseBlock() (*Block, error) {
	if p.cur.Type != TOK_LBRACE {
		return nil, fmt.Errorf("expected '{', got %s", p.cur.Text)
	}
	p.next() // Skip '{'

	blk := &Block{}
	for p.cur.Type != TOK_RBRACE && p.cur.Type != TOK_EOF {
		if p.cur.Type == TOK_SEMICOLON {
			p.next()
			continue
		}
		stmt, err := p.parseExpr()
		if err != nil {
			return nil, err
		}
		blk.Stmts = append(blk.Stmts, stmt)
	}

	if p.cur.Type != TOK_RBRACE {
		return nil, fmt.Errorf("expected '}', got EOF")
	}
	p.next() // Skip '}'

	return blk, nil
}

// consume checks if the current token matches the expected text/type and advances
func (p *Parser) consume(expected string, errorMsg string) error {
	// Check for token type matches based on the expected character
	if expected == "(" && p.cur.Type != TOK_LPAREN ||
		expected == ")" && p.cur.Type != TOK_RPAREN ||
		expected == "{" && p.cur.Type != TOK_LBRACE ||
		expected == "}" && p.cur.Type != TOK_RBRACE ||
		(expected != "(" && expected != ")" && expected != "{" && expected != "}" && p.cur.Text != expected) {
		return fmt.Errorf("%s, got %s", errorMsg, p.cur.Text)
	}

	p.next() // Advance to next token
	return nil
}

// processEscapeSequences handles escape sequences in string content
func processEscapeSequences(s string) string {
	if !strings.Contains(s, "\\") {
		return s // No escapes, return as-is for performance
	}

	var result strings.Builder
	result.Grow(len(s)) // Pre-allocate for efficiency

	for i := 0; i < len(s); i++ {
		if s[i] == '\\' && i+1 < len(s) {
			// Handle escape sequences
			i++ // Skip backslash
			switch s[i] {
			case '"':
				result.WriteByte('"') // Escaped double quote
			case '\'':
				result.WriteByte('\'') // Escaped single quote
			case '\\':
				result.WriteByte('\\') // Escaped backslash
			case 'n':
				result.WriteByte('\n') // Newline
			case 't':
				result.WriteByte('\t') // Tab
			case 'r':
				result.WriteByte('\r') // Carriage return
			case 'b':
				result.WriteByte('\b') // Backspace
			case 'f':
				result.WriteByte('\f') // Form feed
			case '0':
				result.WriteByte('\000') // Null character
			default:
				// Unknown escape sequence - preserve both characters
				result.WriteByte('\\')
				result.WriteByte(s[i])
			}
		} else {
			result.WriteByte(s[i])
		}
	}

	return result.String()
}

// parseStringContent safely parses string content between quotes, handling escapes
func (lx *Lexer) parseStringContent(quote byte) string {
	var content strings.Builder
	s := lx.src

	for lx.pos < len(s) && s[lx.pos] != quote {
		if s[lx.pos] == '\\' && lx.pos+1 < len(s) {
			// Found escape sequence - consume both characters
			content.WriteByte(s[lx.pos]) // backslash
			lx.pos++
			if lx.pos < len(s) {
				content.WriteByte(s[lx.pos]) // escaped character
				lx.pos++
			}
		} else {
			content.WriteByte(s[lx.pos])
			lx.pos++
		}
	}

	return processEscapeSequences(content.String())
}

// Example function syntax: func(x, y) { return x + y; }
func (p *Parser) parseFunction() (Node, error) {
	startPos := p.currentPosition

	// Parse 'func' keyword (we assume it's already consumed if we're in this method)
	if p.cur.Type != TOK_IDENT || p.cur.Text != "func" {
		return nil, errors.New("expected 'func' keyword")
	}
	p.next() // consume "func"

	// Parse parameter list
	if p.cur.Type != TOK_LPAREN {
		return nil, errors.New("expected '(' after func")
	}
	p.next() // consume "("

	var params []string
	if p.cur.Type != TOK_RPAREN {
		for {
			if p.cur.Type != TOK_IDENT {
				return nil, errors.New("parameter name expected")
			}
			params = append(params, p.cur.Text)
			p.next() // consume parameter name

			if p.cur.Type == TOK_RPAREN {
				break
			}

			if p.cur.Type != TOK_COMMA {
				return nil, errors.New("expected ',' between parameters")
			}
			p.next() // consume ","
		}
	}
	p.next() // consume ")"

	// Parse function body
	if p.cur.Type != TOK_LBRACE {
		return nil, errors.New("expected '{' for function body")
	}

	body, err := p.parseBlock()
	if err != nil {
		return nil, err
	}

	// Capture source by creating an identifier for the function
	// This is a placeholder, you might want to enhance this
	sourceDesc := fmt.Sprintf("func(%s)", strings.Join(params, ","))

	return &FunctionDefNode{
		Parameters: params,
		Body:       body,
		Source:     sourceDesc,
		Position:   startPos,
	}, nil
}

func (p *Parser) parseArrayLiteral() (Node, error) {
	// Consume the opening '['
	p.next()

	elements := []Node{}

	// Parse elements until we see closing bracket
	for p.cur.Type != TOK_RBRACKET {
		// Parse the array element
		elem, err := p.parseExpr()
		if err != nil {
			return nil, err
		}
		elements = append(elements, elem)

		// If next token is a comma, consume it and continue
		if p.cur.Type == TOK_COMMA {
			p.next()
		} else if p.cur.Type != TOK_RBRACKET {
			// If not a comma and not closing bracket, that's an error
			return nil, fmt.Errorf("expected ',' or ']', got %v", p.cur)
		}
	}

	// Consume the closing ']'
	p.next()

	return &ArrayLiteralNode{Elements: elements}, nil
}
