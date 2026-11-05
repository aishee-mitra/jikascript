package parser

import (
	"fmt"
	"strconv"
	"strings"

	"jikascript/internal/ast"
	"jikascript/internal/lexer"
)

// Parser converts tokens into an AST.
type Parser struct {
	tokens []lexer.Token
	pos    int
	cur    lexer.Token
}

// ParseProgram parses a complete source file.
func ParseProgram(source string) (*ast.Program, error) {
	lex := lexer.New(source)
	tokens := []lexer.Token{}
	for {
		tok := lex.NextToken()
		tokens = append(tokens, tok)
		if tok.Type == lexer.EOF {
			break
		}
	}
	return New(tokens).Parse()
}

// New creates a parser from already tokenized source.
func New(tokens []lexer.Token) *Parser {
	p := &Parser{tokens: tokens}
	if len(tokens) > 0 {
		p.cur = tokens[0]
	}
	return p
}

// Parse parses the token stream into a program.
func (p *Parser) Parse() (*ast.Program, error) {
	program := &ast.Program{Statements: []ast.Stmt{}}
	for !p.isEOF() {
		stmt, err := p.parseStatement()
		if err != nil {
			return nil, err
		}
		if stmt != nil {
			program.Statements = append(program.Statements, stmt)
		}
		p.consumeSemicolonIfPresent()
	}
	return program, nil
}

func (p *Parser) parseStatement() (ast.Stmt, error) {
	tok := p.cur
	if tok.Type == lexer.EOF {
		return nil, nil
	}

	switch {
	case tok.Type == lexer.KEYWORD && tok.Literal == "let":
		return p.parseVarDecl(false)
	case tok.Type == lexer.KEYWORD && tok.Literal == "const":
		return p.parseVarDecl(true)
	case tok.Type == lexer.KEYWORD && tok.Literal == "function":
		return p.parseFunctionDecl()
	case tok.Type == lexer.KEYWORD && tok.Literal == "return":
		return p.parseReturnStmt()
	case tok.Type == lexer.KEYWORD && tok.Literal == "if":
		return p.parseIfStmt()
	case tok.Type == lexer.KEYWORD && tok.Literal == "while":
		return p.parseWhileStmt()
	case tok.Type == lexer.KEYWORD && tok.Literal == "do":
		return p.parseDoWhileStmt()
	case tok.Type == lexer.KEYWORD && tok.Literal == "for":
		return p.parseForStmt()
	case tok.Type == lexer.KEYWORD && tok.Literal == "switch":
		return p.parseSwitchStmt()
	case tok.Type == lexer.KEYWORD && tok.Literal == "break":
		return p.parseBreakStmt()
	case tok.Type == lexer.KEYWORD && tok.Literal == "continue":
		return p.parseContinueStmt()
	case tok.Type == lexer.KEYWORD && tok.Literal == "task":
		return p.parseTaskDecl()
	case tok.Type == lexer.KEYWORD && tok.Literal == "scheduler":
		return p.parseSchedulerStmt()
	default:
		return p.parseExpressionStmt()
	}
}

func (p *Parser) parseWhileStmt() (ast.Stmt, error) {
	pos := ast.Position{Line: p.cur.Line, Column: p.cur.Column}
	p.next() // while
	if p.cur.Type != lexer.PUNCT || p.cur.Literal != "(" {
		return nil, fmt.Errorf("line %d:%d: expected '(' after while", p.cur.Line, p.cur.Column)
	}
	p.next()
	cond, err := p.parseExpression()
	if err != nil {
		return nil, err
	}
	if p.cur.Type != lexer.PUNCT || p.cur.Literal != ")" {
		return nil, fmt.Errorf("line %d:%d: expected ')' after while condition", p.cur.Line, p.cur.Column)
	}
	p.next()
	body, err := p.parseBlock()
	if err != nil {
		return nil, err
	}
	return &ast.WhileStmt{Cond: cond, Body: body, Pos: pos}, nil
}

func (p *Parser) parseDoWhileStmt() (ast.Stmt, error) {
	pos := ast.Position{Line: p.cur.Line, Column: p.cur.Column}
	p.next() // do
	body, err := p.parseBlock()
	if err != nil {
		return nil, err
	}
	if p.cur.Type != lexer.KEYWORD || p.cur.Literal != "while" {
		return nil, fmt.Errorf("line %d:%d: expected 'while' after do block", p.cur.Line, p.cur.Column)
	}
	p.next()
	if p.cur.Type != lexer.PUNCT || p.cur.Literal != "(" {
		return nil, fmt.Errorf("line %d:%d: expected '(' after while", p.cur.Line, p.cur.Column)
	}
	p.next()
	cond, err := p.parseExpression()
	if err != nil {
		return nil, err
	}
	if p.cur.Type != lexer.PUNCT || p.cur.Literal != ")" {
		return nil, fmt.Errorf("line %d:%d: expected ')' after do-while condition", p.cur.Line, p.cur.Column)
	}
	p.next()
	return &ast.DoWhileStmt{Body: body, Cond: cond, Pos: pos}, nil
}

func (p *Parser) parseForStmt() (ast.Stmt, error) {
	pos := ast.Position{Line: p.cur.Line, Column: p.cur.Column}
	p.next() // for
	if p.cur.Type != lexer.PUNCT || p.cur.Literal != "(" {
		return nil, fmt.Errorf("line %d:%d: expected '(' after for", p.cur.Line, p.cur.Column)
	}
	p.next()
	var init ast.Stmt
	if !(p.cur.Type == lexer.PUNCT && p.cur.Literal == ";") {
		var err error
		init, err = p.parseStatement()
		if err != nil {
			return nil, err
		}
	}
	if p.cur.Type != lexer.PUNCT || p.cur.Literal != ";" {
		return nil, fmt.Errorf("line %d:%d: expected ';' after for init", p.cur.Line, p.cur.Column)
	}
	p.next()
	var cond ast.Expr
	if !(p.cur.Type == lexer.PUNCT && p.cur.Literal == ";") {
		var err error
		cond, err = p.parseExpression()
		if err != nil {
			return nil, err
		}
	}
	if p.cur.Type != lexer.PUNCT || p.cur.Literal != ";" {
		return nil, fmt.Errorf("line %d:%d: expected ';' after for condition", p.cur.Line, p.cur.Column)
	}
	p.next()
	var post ast.Expr
	if !(p.cur.Type == lexer.PUNCT && p.cur.Literal == ")") {
		var err error
		post, err = p.parseExpression()
		if err != nil {
			return nil, err
		}
	}
	if p.cur.Type != lexer.PUNCT || p.cur.Literal != ")" {
		return nil, fmt.Errorf("line %d:%d: expected ')' after for clause", p.cur.Line, p.cur.Column)
	}
	p.next()
	body, err := p.parseBlock()
	if err != nil {
		return nil, err
	}
	return &ast.ForStmt{Init: init, Cond: cond, Post: post, Body: body, Pos: pos}, nil
}

func (p *Parser) parseSwitchStmt() (ast.Stmt, error) {
	pos := ast.Position{Line: p.cur.Line, Column: p.cur.Column}
	p.next() // switch
	if p.cur.Type != lexer.PUNCT || p.cur.Literal != "(" {
		return nil, fmt.Errorf("line %d:%d: expected '(' after switch", p.cur.Line, p.cur.Column)
	}
	p.next()
	expr, err := p.parseExpression()
	if err != nil {
		return nil, err
	}
	if p.cur.Type != lexer.PUNCT || p.cur.Literal != ")" {
		return nil, fmt.Errorf("line %d:%d: expected ')' after switch expression", p.cur.Line, p.cur.Column)
	}
	p.next()
	if p.cur.Type != lexer.PUNCT || p.cur.Literal != "{" {
		return nil, fmt.Errorf("line %d:%d: expected '{' after switch", p.cur.Line, p.cur.Column)
	}
	p.next()

	cases := []ast.SwitchCase{}
	for !(p.cur.Type == lexer.PUNCT && p.cur.Literal == "}") && !p.isEOF() {
		casePos := ast.Position{Line: p.cur.Line, Column: p.cur.Column}
		var caseExpr ast.Expr
		switch {
		case p.cur.Type == lexer.KEYWORD && p.cur.Literal == "case":
			p.next()
			var err error
			caseExpr, err = p.parseExpression()
			if err != nil {
				return nil, err
			}
		case p.cur.Type == lexer.KEYWORD && p.cur.Literal == "default":
			p.next()
		default:
			return nil, fmt.Errorf("line %d:%d: expected 'case' or 'default' in switch", p.cur.Line, p.cur.Column)
		}
		if p.cur.Type != lexer.PUNCT || p.cur.Literal != ":" {
			return nil, fmt.Errorf("line %d:%d: expected ':' in switch case", p.cur.Line, p.cur.Column)
		}
		p.next()

		body := []ast.Stmt{}
		for !(p.cur.Type == lexer.KEYWORD && (p.cur.Literal == "case" || p.cur.Literal == "default")) &&
			!(p.cur.Type == lexer.PUNCT && p.cur.Literal == "}") &&
			!p.isEOF() {
			stmt, err := p.parseStatement()
			if err != nil {
				return nil, err
			}
			if stmt != nil {
				body = append(body, stmt)
			}
			p.consumeSemicolonIfPresent()
		}

		cases = append(cases, ast.SwitchCase{Value: caseExpr, Body: body, Pos: casePos})
	}
	if p.cur.Type != lexer.PUNCT || p.cur.Literal != "}" {
		return nil, fmt.Errorf("line %d:%d: expected '}' after switch", p.cur.Line, p.cur.Column)
	}
	p.next()
	return &ast.SwitchStmt{Expr: expr, Cases: cases, Pos: pos}, nil
}

func (p *Parser) parseBreakStmt() (ast.Stmt, error) {
	pos := ast.Position{Line: p.cur.Line, Column: p.cur.Column}
	p.next()
	return &ast.BreakStmt{Pos: pos}, nil
}

func (p *Parser) parseContinueStmt() (ast.Stmt, error) {
	pos := ast.Position{Line: p.cur.Line, Column: p.cur.Column}
	p.next()
	return &ast.ContinueStmt{Pos: pos}, nil
}

func (p *Parser) parseVarDecl(constDecl bool) (ast.Stmt, error) {
	pos := ast.Position{Line: p.cur.Line, Column: p.cur.Column}
	p.next() // let/const
	if p.cur.Type != lexer.IDENT {
		return nil, fmt.Errorf("line %d:%d: expected variable name", p.cur.Line, p.cur.Column)
	}
	name := p.cur.Literal
	p.next()

	vartype := ""
	arrayCapacity := 0
	if p.cur.Type == lexer.PUNCT && p.cur.Literal == ":" {
		p.next()
		if p.cur.Type != lexer.IDENT && p.cur.Type != lexer.KEYWORD {
			return nil, fmt.Errorf("line %d:%d: expected type annotation", p.cur.Line, p.cur.Column)
		}
		vartype = p.cur.Literal
		p.next()
		if p.cur.Type == lexer.PUNCT && p.cur.Literal == "[" {
			p.next()
			if p.cur.Type != lexer.NUMBER {
				return nil, fmt.Errorf("line %d:%d: expected numeric array capacity", p.cur.Line, p.cur.Column)
			}
			capValue, err := strconv.Atoi(p.cur.Literal)
			if err != nil || capValue <= 0 {
				return nil, fmt.Errorf("line %d:%d: invalid array capacity %q", p.cur.Line, p.cur.Column, p.cur.Literal)
			}
			arrayCapacity = capValue
			p.next()
			if p.cur.Type != lexer.PUNCT || p.cur.Literal != "]" {
				return nil, fmt.Errorf("line %d:%d: expected ']' after array capacity", p.cur.Line, p.cur.Column)
			}
			p.next()
		}
	}

	if p.cur.Type != lexer.OPERATOR || p.cur.Literal != "=" {
		return nil, fmt.Errorf("line %d:%d: expected '=' in declaration", p.cur.Line, p.cur.Column)
	}
	p.next()
	value, err := p.parseExpression()
	if err != nil {
		return nil, err
	}
	if constDecl {
		return &ast.ConstDecl{Name: name, Type: vartype, ArrayCapacity: arrayCapacity, Value: value, Pos: pos}, nil
	}
	return &ast.VarDecl{Name: name, Type: vartype, ArrayCapacity: arrayCapacity, Value: value, Pos: pos}, nil
}

func (p *Parser) parseFunctionDecl() (ast.Stmt, error) {
	pos := ast.Position{Line: p.cur.Line, Column: p.cur.Column}
	p.next() // function
	if p.cur.Type != lexer.IDENT {
		return nil, fmt.Errorf("line %d:%d: expected function name", p.cur.Line, p.cur.Column)
	}
	name := p.cur.Literal
	p.next()
	if p.cur.Type != lexer.PUNCT || p.cur.Literal != "(" {
		return nil, fmt.Errorf("line %d:%d: expected '(' after function name", p.cur.Line, p.cur.Column)
	}
	p.next()
	params := []string{}
	if !(p.cur.Type == lexer.PUNCT && p.cur.Literal == ")") {
		for {
			if p.cur.Type != lexer.IDENT {
				return nil, fmt.Errorf("line %d:%d: expected parameter name", p.cur.Line, p.cur.Column)
			}
			params = append(params, p.cur.Literal)
			p.next()
			if p.cur.Type == lexer.PUNCT && p.cur.Literal == "," {
				p.next()
				continue
			}
			break
		}
	}
	if p.cur.Type != lexer.PUNCT || p.cur.Literal != ")" {
		return nil, fmt.Errorf("line %d:%d: expected ')' after parameter list", p.cur.Line, p.cur.Column)
	}
	p.next()
	body, err := p.parseBlock()
	if err != nil {
		return nil, err
	}
	return &ast.FunctionDecl{Name: name, Params: params, Body: body, Pos: pos}, nil
}

func (p *Parser) parseReturnStmt() (ast.Stmt, error) {
	pos := ast.Position{Line: p.cur.Line, Column: p.cur.Column}
	p.next() // return
	var value ast.Expr
	if !(p.cur.Type == lexer.PUNCT && p.cur.Literal == ";") {
		var err error
		value, err = p.parseExpression()
		if err != nil {
			return nil, err
		}
	}
	return &ast.ReturnStmt{Value: value, Pos: pos}, nil
}

func (p *Parser) parseIfStmt() (ast.Stmt, error) {
	pos := ast.Position{Line: p.cur.Line, Column: p.cur.Column}
	p.next() // if
	if p.cur.Type != lexer.PUNCT || p.cur.Literal != "(" {
		return nil, fmt.Errorf("line %d:%d: expected '(' after if", p.cur.Line, p.cur.Column)
	}
	p.next()
	cond, err := p.parseExpression()
	if err != nil {
		return nil, err
	}
	if p.cur.Type != lexer.PUNCT || p.cur.Literal != ")" {
		return nil, fmt.Errorf("line %d:%d: expected ')' after if condition", p.cur.Line, p.cur.Column)
	}
	p.next()
	thenBlock, err := p.parseBlock()
	if err != nil {
		return nil, err
	}
	var elseBlock []ast.Stmt
	if p.cur.Type == lexer.KEYWORD && p.cur.Literal == "else" {
		p.next()
		if p.cur.Type == lexer.KEYWORD && p.cur.Literal == "if" {
			stmt, err := p.parseIfStmt()
			if err != nil {
				return nil, err
			}
			elseBlock = []ast.Stmt{stmt}
		} else {
			elseBlock, err = p.parseBlock()
			if err != nil {
				return nil, err
			}
		}
	}
	return &ast.IfStmt{Cond: cond, Then: thenBlock, Else: elseBlock, Pos: pos}, nil
}

func (p *Parser) parseTaskDecl() (ast.Stmt, error) {
	pos := ast.Position{Line: p.cur.Line, Column: p.cur.Column}
	p.next() // task
	if p.cur.Type != lexer.PUNCT || p.cur.Literal != "(" {
		return nil, fmt.Errorf("line %d:%d: expected '(' after task", p.cur.Line, p.cur.Column)
	}
	p.next()
	nameExpr, err := p.parseExpression()
	if err != nil {
		return nil, err
	}
	if p.cur.Type != lexer.PUNCT || p.cur.Literal != "," {
		return nil, fmt.Errorf("line %d:%d: expected ',' after task name", p.cur.Line, p.cur.Column)
	}
	p.next()
	intervalExpr, err := p.parseExpression()
	if err != nil {
		return nil, err
	}
	if p.cur.Type != lexer.PUNCT || p.cur.Literal != "," {
		return nil, fmt.Errorf("line %d:%d: expected ',' after task interval", p.cur.Line, p.cur.Column)
	}
	p.next()
	callback, err := p.parseArrowFunction()
	if err != nil {
		return nil, err
	}
	if p.cur.Type != lexer.PUNCT || p.cur.Literal != ")" {
		return nil, fmt.Errorf("line %d:%d: expected ')' after task callback", p.cur.Line, p.cur.Column)
	}
	p.next()
	name, ok := extractStringName(nameExpr)
	if !ok {
		return nil, fmt.Errorf("line %d:%d: task name must be a string literal", p.cur.Line, p.cur.Column)
	}
	return &ast.TaskDecl{Name: name, Interval: intervalExpr, Callback: callback, Pos: pos}, nil
}

func extractStringName(expr ast.Expr) (string, bool) {
	if lit, ok := expr.(*ast.StringLiteral); ok {
		return lit.Value, true
	}
	return "", false
}

func (p *Parser) parseSchedulerStmt() (ast.Stmt, error) {
	pos := ast.Position{Line: p.cur.Line, Column: p.cur.Column}
	p.next() // scheduler
	if p.cur.Type != lexer.PUNCT || p.cur.Literal != "." {
		return nil, fmt.Errorf("line %d:%d: expected '.' after scheduler", p.cur.Line, p.cur.Column)
	}
	p.next()
	if !(p.cur.Type == lexer.IDENT || p.cur.Type == lexer.KEYWORD) || p.cur.Literal != "start" {
		return nil, fmt.Errorf("line %d:%d: expected start() call", p.cur.Line, p.cur.Column)
	}
	p.next()
	if p.cur.Type != lexer.PUNCT || p.cur.Literal != "(" {
		return nil, fmt.Errorf("line %d:%d: expected '(' after scheduler.start", p.cur.Line, p.cur.Column)
	}
	p.next()
	if p.cur.Type != lexer.PUNCT || p.cur.Literal != ")" {
		return nil, fmt.Errorf("line %d:%d: expected ')' after scheduler.start", p.cur.Line, p.cur.Column)
	}
	p.next()
	return &ast.ExprStmt{Expr: &ast.CallExpr{Function: &ast.SelectorExpr{Object: &ast.Identifier{Name: "scheduler", Pos: pos}, Field: "start", Pos: pos}, Args: nil, Pos: pos}, Pos: pos}, nil
}

func (p *Parser) parseArrowFunction() (*ast.ArrowFunctionExpr, error) {
	pos := ast.Position{Line: p.cur.Line, Column: p.cur.Column}
	params := []string{}
	if p.cur.Type == lexer.PUNCT && p.cur.Literal == "(" {
		p.next()
		if !(p.cur.Type == lexer.PUNCT && p.cur.Literal == ")") {
			for {
				if p.cur.Type != lexer.IDENT {
					return nil, fmt.Errorf("line %d:%d: expected parameter name", p.cur.Line, p.cur.Column)
				}
				params = append(params, p.cur.Literal)
				p.next()
				if p.cur.Type == lexer.PUNCT && p.cur.Literal == "," {
					p.next()
					continue
				}
				break
			}
		}
		if p.cur.Type != lexer.PUNCT || p.cur.Literal != ")" {
			return nil, fmt.Errorf("line %d:%d: expected ')' after arrow params", p.cur.Line, p.cur.Column)
		}
		p.next()
	} else if p.cur.Type == lexer.IDENT {
		params = append(params, p.cur.Literal)
		p.next()
	}
	if p.cur.Type != lexer.OPERATOR || p.cur.Literal != "=>" {
		return nil, fmt.Errorf("line %d:%d: expected '=>' in arrow function", p.cur.Line, p.cur.Column)
	}
	p.next()
	if p.cur.Type == lexer.PUNCT && p.cur.Literal == "{" {
		body, err := p.parseBlock()
		if err != nil {
			return nil, err
		}
		return &ast.ArrowFunctionExpr{Params: params, Body: body, Pos: pos}, nil
	}
	value, err := p.parseExpression()
	if err != nil {
		return nil, err
	}
	return &ast.ArrowFunctionExpr{Params: params, Body: []ast.Stmt{&ast.ReturnStmt{Value: value, Pos: pos}}, Pos: pos}, nil
}

func (p *Parser) parseExpressionStmt() (ast.Stmt, error) {
	expr, err := p.parseExpression()
	if err != nil {
		return nil, err
	}
	return &ast.ExprStmt{Expr: expr, Pos: ast.Position{Line: p.cur.Line, Column: p.cur.Column}}, nil
}

func (p *Parser) parseBlock() ([]ast.Stmt, error) {
	if p.cur.Type != lexer.PUNCT || p.cur.Literal != "{" {
		return nil, fmt.Errorf("line %d:%d: expected '{'", p.cur.Line, p.cur.Column)
	}
	p.next()
	stmts := []ast.Stmt{}
	for !(p.cur.Type == lexer.PUNCT && p.cur.Literal == "}") && !p.isEOF() {
		stmt, err := p.parseStatement()
		if err != nil {
			return nil, err
		}
		if stmt != nil {
			stmts = append(stmts, stmt)
		}
		p.consumeSemicolonIfPresent()
	}
	if p.cur.Type != lexer.PUNCT || p.cur.Literal != "}" {
		return nil, fmt.Errorf("line %d:%d: expected '}'", p.cur.Line, p.cur.Column)
	}
	p.next()
	return stmts, nil
}

func (p *Parser) parseExpression() (ast.Expr, error) {
	return p.parseAssignment()
}

func (p *Parser) parseAssignment() (ast.Expr, error) {
	left, err := p.parseLogicalOr()
	if err != nil {
		return nil, err
	}
	if p.cur.Type == lexer.OPERATOR && isAssignmentOperator(p.cur.Literal) {
		start := ast.Position{Line: p.cur.Line, Column: p.cur.Column}
		op := p.cur.Literal
		p.next()
		right, err := p.parseAssignment()
		if err != nil {
			return nil, err
		}
		return &ast.BinaryExpr{Left: left, Op: op, Right: right, Pos: start}, nil
	}
	return left, nil
}

func (p *Parser) parseLogicalOr() (ast.Expr, error) {
	return p.parseBinaryLeftAssoc(p.parseLogicalAnd, []string{"||"})
}

func (p *Parser) parseLogicalAnd() (ast.Expr, error) {
	return p.parseBinaryLeftAssoc(p.parseBitwiseOr, []string{"&&"})
}

func (p *Parser) parseBitwiseOr() (ast.Expr, error) {
	return p.parseBinaryLeftAssoc(p.parseBitwiseXor, []string{"|"})
}

func (p *Parser) parseBitwiseXor() (ast.Expr, error) {
	return p.parseBinaryLeftAssoc(p.parseBitwiseAnd, []string{"^"})
}

func (p *Parser) parseBitwiseAnd() (ast.Expr, error) {
	return p.parseBinaryLeftAssoc(p.parseEquality, []string{"&"})
}

func (p *Parser) parseEquality() (ast.Expr, error) {
	return p.parseBinaryLeftAssoc(p.parseRelational, []string{"==", "!=", "===", "!=="})
}

func (p *Parser) parseRelational() (ast.Expr, error) {
	return p.parseBinaryLeftAssoc(p.parseShift, []string{"<", "<=", ">", ">="})
}

func (p *Parser) parseShift() (ast.Expr, error) {
	return p.parseBinaryLeftAssoc(p.parseAdditive, []string{"<<", ">>"})
}

func (p *Parser) parseAdditive() (ast.Expr, error) {
	return p.parseBinaryLeftAssoc(p.parseMultiplicative, []string{"+", "-"})
}

func (p *Parser) parseMultiplicative() (ast.Expr, error) {
	return p.parseBinaryLeftAssoc(p.parseUnary, []string{"*", "/", "%"})
}

func (p *Parser) parseBinaryLeftAssoc(next func() (ast.Expr, error), ops []string) (ast.Expr, error) {
	left, err := next()
	if err != nil {
		return nil, err
	}
	for p.cur.Type == lexer.OPERATOR && containsOperator(ops, p.cur.Literal) {
		op := p.cur.Literal
		pos := ast.Position{Line: p.cur.Line, Column: p.cur.Column}
		p.next()
		right, err := next()
		if err != nil {
			return nil, err
		}
		left = &ast.BinaryExpr{Left: left, Op: op, Right: right, Pos: pos}
	}
	return left, nil
}

func (p *Parser) parseUnary() (ast.Expr, error) {
	if p.cur.Type == lexer.OPERATOR && (p.cur.Literal == "!" || p.cur.Literal == "+" || p.cur.Literal == "-" || p.cur.Literal == "++" || p.cur.Literal == "--" || p.cur.Literal == "~") {
		op := p.cur.Literal
		p.next()
		right, err := p.parseUnary()
		if err != nil {
			return nil, err
		}
		return &ast.UnaryExpr{Op: op, Expr: right, Pos: ast.Position{Line: p.cur.Line, Column: p.cur.Column}}, nil
	}
	primary, err := p.parsePrimary()
	if err != nil {
		return nil, err
	}
	return p.parsePostfix(primary)
}

func (p *Parser) parsePrimary() (ast.Expr, error) {
	switch p.cur.Type {
	case lexer.IDENT:
		ident := &ast.Identifier{Name: p.cur.Literal, Pos: ast.Position{Line: p.cur.Line, Column: p.cur.Column}}
		p.next()
		return ident, nil
	case lexer.NUMBER:
		lit := &ast.NumberLiteral{Value: p.cur.Literal, Pos: ast.Position{Line: p.cur.Line, Column: p.cur.Column}}
		p.next()
		return lit, nil
	case lexer.STRING:
		lit := &ast.StringLiteral{Value: p.cur.Literal, Pos: ast.Position{Line: p.cur.Line, Column: p.cur.Column}}
		p.next()
		return lit, nil
	case lexer.BOOL:
		lit := &ast.BoolLiteral{Value: p.cur.Literal == "true", Pos: ast.Position{Line: p.cur.Line, Column: p.cur.Column}}
		p.next()
		return lit, nil
	case lexer.KEYWORD:
		if p.cur.Literal == "pin" || p.cur.Literal == "scheduler" || p.cur.Literal == "start" {
			ident := &ast.Identifier{Name: p.cur.Literal, Pos: ast.Position{Line: p.cur.Line, Column: p.cur.Column}}
			p.next()
			return ident, nil
		}
		return nil, fmt.Errorf("line %d:%d: unexpected keyword '%s' in expression", p.cur.Line, p.cur.Column, p.cur.Literal)
	case lexer.PUNCT:
		if p.cur.Literal == "[" {
			pos := ast.Position{Line: p.cur.Line, Column: p.cur.Column}
			p.next()
			elements := []ast.Expr{}
			if !(p.cur.Type == lexer.PUNCT && p.cur.Literal == "]") {
				for {
					elem, err := p.parseExpression()
					if err != nil {
						return nil, err
					}
					elements = append(elements, elem)
					if p.cur.Type == lexer.PUNCT && p.cur.Literal == "," {
						p.next()
						continue
					}
					break
				}
			}
			if p.cur.Type != lexer.PUNCT || p.cur.Literal != "]" {
				return nil, fmt.Errorf("line %d:%d: expected ']' after array literal", p.cur.Line, p.cur.Column)
			}
			p.next()
			return &ast.ArrayLiteralExpr{Elements: elements, Pos: pos}, nil
		}
		if p.cur.Literal == "(" {
			p.next()
			expr, err := p.parseExpression()
			if err != nil {
				return nil, err
			}
			if p.cur.Type != lexer.PUNCT || p.cur.Literal != ")" {
				return nil, fmt.Errorf("line %d:%d: expected ')'", p.cur.Line, p.cur.Column)
			}
			p.next()
			return expr, nil
		}
		return nil, fmt.Errorf("line %d:%d: unexpected token %q in expression", p.cur.Line, p.cur.Column, p.cur.Literal)
	default:
		return nil, fmt.Errorf("line %d:%d: unexpected token %q in expression", p.cur.Line, p.cur.Column, p.cur.Literal)
	}
}

func (p *Parser) parsePostfix(expr ast.Expr) (ast.Expr, error) {
	for {
		switch {
		case p.cur.Type == lexer.PUNCT && p.cur.Literal == ".":
			p.next()
			if p.cur.Type != lexer.IDENT {
				return nil, fmt.Errorf("line %d:%d: expected identifier after '.'", p.cur.Line, p.cur.Column)
			}
			field := p.cur.Literal
			p.next()
			expr = &ast.SelectorExpr{Object: expr, Field: field, Pos: ast.Position{Line: p.cur.Line, Column: p.cur.Column}}
		case p.cur.Type == lexer.PUNCT && p.cur.Literal == "(":
			p.next()
			args := []ast.Expr{}
			if !(p.cur.Type == lexer.PUNCT && p.cur.Literal == ")") {
				for {
					arg, err := p.parseExpression()
					if err != nil {
						return nil, err
					}
					args = append(args, arg)
					if p.cur.Type == lexer.PUNCT && p.cur.Literal == "," {
						p.next()
						continue
					}
					break
				}
			}
			if p.cur.Type != lexer.PUNCT || p.cur.Literal != ")" {
				return nil, fmt.Errorf("line %d:%d: expected ')' after arguments", p.cur.Line, p.cur.Column)
			}
			p.next()
			expr = &ast.CallExpr{Function: expr, Args: args, Pos: ast.Position{Line: p.cur.Line, Column: p.cur.Column}}
		case p.cur.Type == lexer.PUNCT && p.cur.Literal == "[":
			indexPos := ast.Position{Line: p.cur.Line, Column: p.cur.Column}
			p.next()
			idxExpr, err := p.parseExpression()
			if err != nil {
				return nil, err
			}
			if p.cur.Type != lexer.PUNCT || p.cur.Literal != "]" {
				return nil, fmt.Errorf("line %d:%d: expected ']' after index expression", p.cur.Line, p.cur.Column)
			}
			p.next()
			expr = &ast.IndexExpr{Object: expr, Index: idxExpr, Pos: indexPos}
		default:
			return expr, nil
		}
	}
}

func (p *Parser) consumeSemicolonIfPresent() {
	if p.cur.Type == lexer.PUNCT && p.cur.Literal == ";" {
		p.next()
	}
}

func (p *Parser) isEOF() bool {
	return p.cur.Type == lexer.EOF
}

func (p *Parser) next() {
	if p.pos+1 < len(p.tokens) {
		p.pos++
		p.cur = p.tokens[p.pos]
		return
	}
	p.cur = lexer.Token{Type: lexer.EOF, Literal: "", Line: p.cur.Line, Column: p.cur.Column}
}

func containsOperator(ops []string, op string) bool {
	for _, candidate := range ops {
		if candidate == op {
			return true
		}
	}
	return false
}

func isAssignmentOperator(op string) bool {
	switch op {
	case "=", "+=", "-=", "*=", "/=", "%=", "&=", "|=", "^=", "<<=", ">>=":
		return true
	default:
		return false
	}
}

func (p *Parser) DebugString() string {
	var parts []string
	for _, tok := range p.tokens {
		parts = append(parts, fmt.Sprintf("%s:%s", tok.Type, tok.Literal))
	}
	return strings.Join(parts, " ")
}
