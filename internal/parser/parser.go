package parser

import (
	"fmt"
	"strings"

	"microjs/internal/ast"
	"microjs/internal/lexer"
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
	case tok.Type == lexer.KEYWORD && tok.Literal == "task":
		return p.parseTaskDecl()
	case tok.Type == lexer.KEYWORD && tok.Literal == "scheduler":
		return p.parseSchedulerStmt()
	default:
		return p.parseExpressionStmt()
	}
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
	if p.cur.Type == lexer.PUNCT && p.cur.Literal == ":" {
		p.next()
		if p.cur.Type != lexer.IDENT && p.cur.Type != lexer.KEYWORD {
			return nil, fmt.Errorf("line %d:%d: expected type annotation", p.cur.Line, p.cur.Column)
		}
		vartype = p.cur.Literal
		p.next()
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
		return &ast.ConstDecl{Name: name, Type: vartype, Value: value, Pos: pos}, nil
	}
	return &ast.VarDecl{Name: name, Type: vartype, Value: value, Pos: pos}, nil
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
	return p.parseBinaryOrAssignment()
}

func (p *Parser) parseBinaryOrAssignment() (ast.Expr, error) {
	left, err := p.parseComparison()
	if err != nil {
		return nil, err
	}
	if p.cur.Type == lexer.OPERATOR && (p.cur.Literal == "=" || p.cur.Literal == "+=" || p.cur.Literal == "-=" || p.cur.Literal == "*=" || p.cur.Literal == "/=") {
		start := ast.Position{Line: p.cur.Line, Column: p.cur.Column}
		op := p.cur.Literal
		p.next()
		right, err := p.parseBinaryOrAssignment()
		if err != nil {
			return nil, err
		}
		return &ast.BinaryExpr{Left: left, Op: op, Right: right, Pos: start}, nil
	}
	return left, nil
}

func (p *Parser) parseComparison() (ast.Expr, error) {
	left, err := p.parseAdditive()
	if err != nil {
		return nil, err
	}
	for p.cur.Type == lexer.OPERATOR && (p.cur.Literal == "==" || p.cur.Literal == "!=" || p.cur.Literal == "<" || p.cur.Literal == "<=" || p.cur.Literal == ">" || p.cur.Literal == ">=" || p.cur.Literal == "&&" || p.cur.Literal == "||") {
		op := p.cur.Literal
		p.next()
		right, err := p.parseAdditive()
		if err != nil {
			return nil, err
		}
		left = &ast.BinaryExpr{Left: left, Op: op, Right: right, Pos: ast.Position{Line: p.cur.Line, Column: p.cur.Column}}
	}
	return left, nil
}

func (p *Parser) parseAdditive() (ast.Expr, error) {
	left, err := p.parseMultiplicative()
	if err != nil {
		return nil, err
	}
	for p.cur.Type == lexer.OPERATOR && (p.cur.Literal == "+" || p.cur.Literal == "-") {
		op := p.cur.Literal
		p.next()
		right, err := p.parseMultiplicative()
		if err != nil {
			return nil, err
		}
		left = &ast.BinaryExpr{Left: left, Op: op, Right: right, Pos: ast.Position{Line: p.cur.Line, Column: p.cur.Column}}
	}
	return left, nil
}

func (p *Parser) parseMultiplicative() (ast.Expr, error) {
	left, err := p.parseUnary()
	if err != nil {
		return nil, err
	}
	for p.cur.Type == lexer.OPERATOR && (p.cur.Literal == "*" || p.cur.Literal == "/" || p.cur.Literal == "%") {
		op := p.cur.Literal
		p.next()
		right, err := p.parseUnary()
		if err != nil {
			return nil, err
		}
		left = &ast.BinaryExpr{Left: left, Op: op, Right: right, Pos: ast.Position{Line: p.cur.Line, Column: p.cur.Column}}
	}
	return left, nil
}

func (p *Parser) parseUnary() (ast.Expr, error) {
	if p.cur.Type == lexer.OPERATOR && (p.cur.Literal == "!" || p.cur.Literal == "+" || p.cur.Literal == "-" || p.cur.Literal == "++" || p.cur.Literal == "--") {
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
	case lexer.KEYWORD:
		if p.cur.Literal == "true" || p.cur.Literal == "false" {
			lit := &ast.BoolLiteral{Value: p.cur.Literal == "true", Pos: ast.Position{Line: p.cur.Line, Column: p.cur.Column}}
			p.next()
			return lit, nil
		}
		if p.cur.Literal == "pin" || p.cur.Literal == "scheduler" || p.cur.Literal == "start" {
			ident := &ast.Identifier{Name: p.cur.Literal, Pos: ast.Position{Line: p.cur.Line, Column: p.cur.Column}}
			p.next()
			return ident, nil
		}
		return nil, fmt.Errorf("line %d:%d: unexpected keyword '%s' in expression", p.cur.Line, p.cur.Column, p.cur.Literal)
	case lexer.PUNCT:
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

func (p *Parser) DebugString() string {
	var parts []string
	for _, tok := range p.tokens {
		parts = append(parts, fmt.Sprintf("%s:%s", tok.Type, tok.Literal))
	}
	return strings.Join(parts, " ")
}
