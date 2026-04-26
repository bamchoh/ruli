package parser

import (
	"ruli/ast"
	"ruli/lexer"
)

const (
	LOWEST  = iota
	SUM     // + -
	PRODUCT // * /
)

var precedences = map[lexer.TokenType]int{
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
		if p.peekToken.Type == lexer.DECLARE {
			return p.parseAssignStatement()
		}
	}

	return nil
}

func (p *Parser) parseAssignStatement() *ast.AssignStatement {

	stmt := &ast.AssignStatement{
		Name: p.curToken.Literal,
	}

	p.nextToken() // skip IDENT
	p.nextToken() // skip :=

	stmt.Value = p.parseExpression(LOWEST)

	return stmt
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
	return &ast.IntegerLiteral{
		Value: p.curToken.Literal,
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
