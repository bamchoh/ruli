package parser

import (
	"ruli/ast"
	"ruli/lexer"
	"strconv"
)

const (
	LOWEST = iota
	EQUALS
	LESSGREATER
	SUM
	PRODUCT
)

var precedences = map[lexer.TokenType]int{
	lexer.EQ:       EQUALS,
	lexer.NOT_EQ:   EQUALS,
	lexer.LT:       LESSGREATER,
	lexer.GT:       LESSGREATER,
	lexer.PLUS:     SUM,
	lexer.MINUS:    SUM,
	lexer.ASTERISK: PRODUCT,
	lexer.SLASH:    PRODUCT,
}

type (
	prefixParseFn func() ast.Expression
	infixParseFn  func(ast.Expression) ast.Expression
)

type Parser struct {
	l *lexer.Lexer

	curToken  lexer.Token
	peekToken lexer.Token

	prefixParseFns map[lexer.TokenType]prefixParseFn
	infixParseFns  map[lexer.TokenType]infixParseFn
}

func New(l *lexer.Lexer) *Parser {
	p := &Parser{
		l:              l,
		prefixParseFns: make(map[lexer.TokenType]prefixParseFn),
		infixParseFns:  make(map[lexer.TokenType]infixParseFn),
	}

	p.registerPrefix(lexer.INT_LIT, p.parseIntegerLiteral)
	p.registerPrefix(lexer.IDENT, p.parseIdentifier)

	p.registerInfix(lexer.PLUS, p.parseInfixExpression)
	p.registerInfix(lexer.MINUS, p.parseInfixExpression)
	p.registerInfix(lexer.ASTERISK, p.parseInfixExpression)
	p.registerInfix(lexer.SLASH, p.parseInfixExpression)
	p.registerInfix(lexer.EQ, p.parseInfixExpression)
	p.registerInfix(lexer.NOT_EQ, p.parseInfixExpression)
	p.registerInfix(lexer.LT, p.parseInfixExpression)
	p.registerInfix(lexer.GT, p.parseInfixExpression)

	p.nextToken()
	p.nextToken()

	return p
}

func (p *Parser) nextToken() {
	p.curToken = p.peekToken
	p.peekToken = p.l.NextToken()
}

func (p *Parser) ParseProgram() *ast.Program {
	program := &ast.Program{}

	for p.curToken.Type != lexer.EOF {

		stmt := p.parseStatement()
		if stmt != nil {
			program.Statements = append(program.Statements, stmt)
		}

		p.nextToken()
	}

	return program
}

func (p *Parser) parseStatement() ast.Statement {

	switch p.curToken.Type {

	case lexer.IDENT:

		switch p.peekToken.Type {

		case lexer.DECLARE, lexer.COLON:
			return p.parseVarDeclStatement()

		case lexer.ASSIGN:
			return p.parseAssignStatement()
		}
	}

	return nil
}

func (p *Parser) parseVarDeclStatement() *ast.VarDeclStatement {

	stmt := &ast.VarDeclStatement{
		Name: p.curToken.Literal,
	}

	// x := 10
	if p.peekToken.Type == lexer.DECLARE {
		p.nextToken() // :=
		p.nextToken() // first expr token

		stmt.Value = p.parseExpression(LOWEST)
		return stmt
	}

	// x: INT = 10
	// x: INT
	if p.peekToken.Type == lexer.COLON {
		p.nextToken() // :
		p.nextToken() // type ident

		stmt.Type = p.parseType()

		if p.peekToken.Type == lexer.ASSIGN {
			p.nextToken() // =
			p.nextToken() // expr first token

			stmt.Value = p.parseExpression(LOWEST)
		}

		return stmt
	}

	return stmt
}

func (p *Parser) parseAssignStatement() *ast.AssignStatement {
	stmt := &ast.AssignStatement{
		Name: p.curToken.Literal,
	}
	p.nextToken() // skip =
	p.nextToken() // first expr token
	stmt.Value = p.parseExpression(LOWEST)
	return stmt
}

func (p *Parser) parseType() ast.TypeNode {

	switch p.curToken.Type {

	case lexer.INT:
		return &ast.BasicType{Name: "INT"}

	case lexer.REAL:
		return &ast.BasicType{Name: "REAL"}

	case lexer.BOOL:
		return &ast.BasicType{Name: "BOOL"}

	case lexer.IDENT:
		// distinct type / struct名 将来対応
		return &ast.BasicType{Name: p.curToken.Literal}
	}

	return nil
}

func (p *Parser) parseExpression(precedence int) ast.Expression {

	prefix := p.prefixParseFns[p.curToken.Type]
	if prefix == nil {
		return nil
	}

	leftExp := prefix()

	for p.peekToken.Type != lexer.SEMICOLON &&
		precedence < p.peekPrecedence() {

		infix := p.infixParseFns[p.peekToken.Type]
		if infix == nil {
			return leftExp
		}

		p.nextToken()
		leftExp = infix(leftExp)
	}

	return leftExp
}

func (p *Parser) parseIntegerLiteral() ast.Expression {
	value, err := strconv.ParseInt(p.curToken.Literal, 0, 64)
	if err != nil {
		return nil
	}
	return &ast.IntegerLiteral{
		Value: int(value),
	}
}

func (p *Parser) parseIdentifier() ast.Expression {
	return &ast.Identifier{
		Value: p.curToken.Literal,
	}
}

func (p *Parser) parseInfixExpression(left ast.Expression) ast.Expression {

	expr := &ast.BinaryExpression{
		Left:     left,
		Operator: p.curToken.Literal,
	}

	precedence := p.curPrecedence()
	p.nextToken()

	expr.Right = p.parseExpression(precedence)

	return expr
}

func (p *Parser) peekPrecedence() int {
	if p, ok := precedences[p.peekToken.Type]; ok {
		return p
	}
	return LOWEST
}

func (p *Parser) curPrecedence() int {
	if p, ok := precedences[p.curToken.Type]; ok {
		return p
	}
	return LOWEST
}

func (p *Parser) registerPrefix(tokenType lexer.TokenType, fn prefixParseFn) {
	p.prefixParseFns[tokenType] = fn
}

func (p *Parser) registerInfix(tokenType lexer.TokenType, fn infixParseFn) {
	p.infixParseFns[tokenType] = fn
}
