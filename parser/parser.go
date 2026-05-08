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
	CALL
	INDEX
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
	lexer.LPAREN:   CALL,
	lexer.LBRACKET: INDEX,
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
	p.registerPrefix(lexer.STRING_LIT, p.parseStringLiteral)
	p.registerPrefix(lexer.LBRACKET, p.parseArrayLiteral)

	p.registerInfix(lexer.PLUS, p.parseInfixExpression)
	p.registerInfix(lexer.MINUS, p.parseInfixExpression)
	p.registerInfix(lexer.ASTERISK, p.parseInfixExpression)
	p.registerInfix(lexer.SLASH, p.parseInfixExpression)
	p.registerInfix(lexer.EQ, p.parseInfixExpression)
	p.registerInfix(lexer.NOT_EQ, p.parseInfixExpression)
	p.registerInfix(lexer.LT, p.parseInfixExpression)
	p.registerInfix(lexer.GT, p.parseInfixExpression)
	p.registerInfix(lexer.LPAREN, p.parseCallExpression)
	p.registerInfix(lexer.LBRACKET, p.parseIndexExpression)

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

		for p.curToken.Type == lexer.SEMICOLON {
			p.nextToken()
		}

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

	case lexer.IF:
		return p.parseIfStatement()

	case lexer.FOR:
		return p.parseForStatement()

	case lexer.BREAK:
		return &ast.BreakStatement{}

	case lexer.CONTINUE:
		return &ast.ContinueStatement{}

	case lexer.FUNC:
		return p.parseFunctionStatement()

	case lexer.RETURN:
		return p.parseReturnStatement()

	case lexer.IDENT:

		if p.peekToken.Type == lexer.LBRACKET {
			return p.parseIndexedAssignStatement()
		}

		switch p.peekToken.Type {

		case lexer.DECLARE, lexer.COLON:
			return p.parseVarDeclStatement()

		case lexer.ASSIGN:
			return p.parseAssignStatement()

		case lexer.INC, lexer.DEC:
			return p.parseIncDecStatement()

		}

	}

	return p.parseExpressionStatement()
}

func (p *Parser) parseExpressionStatement() *ast.ExpressionStatement {
	stmt := &ast.ExpressionStatement{}
	stmt.Expression = p.parseExpression(LOWEST)
	return stmt
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
		Left: &ast.Identifier{
			Value: p.curToken.Literal,
		},
	}
	p.nextToken() // skip =
	p.nextToken() // first expr token
	stmt.Value = p.parseExpression(LOWEST)
	return stmt
}

func (p *Parser) parseIndexedAssignStatement() *ast.AssignStatement {

	left := p.parseExpression(LOWEST)

	if p.peekToken.Type != lexer.ASSIGN {
		return nil
	}

	stmt := &ast.AssignStatement{
		Left: left,
	}

	p.nextToken() // =
	p.nextToken() // first expr

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

func (p *Parser) parseIfStatement() *ast.IfStatement {

	stmt := &ast.IfStatement{}

	p.nextToken() // condition first token

	stmt.Condition = p.parseExpression(LOWEST)

	if p.peekToken.Type != lexer.LBRACE {
		return stmt
	}

	p.nextToken() // {
	p.nextToken() // first stmt inside block

	stmt.Consequence = p.parseBlockStatement()

	if p.peekToken.Type == lexer.ELSE {
		p.nextToken() // else

		if p.peekToken.Type == lexer.IF {
			p.nextToken()

			stmt.Alternative = &ast.BlockStatement{
				Statements: []ast.Statement{
					p.parseIfStatement(),
				},
			}
			return stmt
		}

		if p.peekToken.Type != lexer.LBRACE {
			return stmt
		}

		p.nextToken() // {
		p.nextToken() // {
		stmt.Alternative = p.parseBlockStatement()
	}

	return stmt
}

func (p *Parser) parseBlockStatement() *ast.BlockStatement {

	block := &ast.BlockStatement{}

	for p.curToken.Type != lexer.RBRACE && p.curToken.Type != lexer.EOF {

		stmt := p.parseStatement()
		if stmt != nil {
			block.Statements = append(block.Statements, stmt)
		}

		p.nextToken()
	}

	return block
}

func (p *Parser) parseIncDecStatement() *ast.IncDecStatement {
	stmt := &ast.IncDecStatement{
		Name: p.curToken.Literal,
	}

	p.nextToken() // ++ or --

	stmt.Operator = p.curToken.Literal

	return stmt
}

func (p *Parser) parseForStatement() *ast.ForStatement {

	stmt := &ast.ForStatement{}

	p.nextToken() // init first token
	stmt.Init = p.parseStatement()

	if p.peekToken.Type != lexer.SEMICOLON {
		return stmt
	}

	p.nextToken() // ;
	p.nextToken() // condition first token
	stmt.Condition = p.parseExpression(LOWEST)

	if p.peekToken.Type != lexer.SEMICOLON {
		return stmt
	}

	p.nextToken() // ;
	p.nextToken() // post first token
	stmt.Post = p.parseStatement()

	if p.peekToken.Type != lexer.LBRACE {
		return stmt
	}

	p.nextToken() // {
	p.nextToken() // first stmt in body

	stmt.Body = p.parseBlockStatement()

	return stmt
}

func (p *Parser) parseCallExpression(function ast.Expression) ast.Expression {

	exp := &ast.CallExpression{
		Function: function,
	}

	exp.Arguments = p.parseExpressionList(lexer.RPAREN)

	return exp
}

func (p *Parser) parseExpressionList(end lexer.TokenType) []ast.Expression {
	var list []ast.Expression

	if p.peekToken.Type == end {
		p.nextToken()
		return list
	}

	p.nextToken()
	list = append(list, p.parseExpression(LOWEST))

	for p.peekToken.Type == lexer.COMMA {
		p.nextToken()
		p.nextToken()
		list = append(list, p.parseExpression(LOWEST))
	}

	if p.peekToken.Type == end {
		p.nextToken()
	}

	return list
}

func (p *Parser) parseReturnStatement() *ast.ReturnStatement {
	stmt := &ast.ReturnStatement{}

	p.nextToken()
	stmt.Value = p.parseExpression(LOWEST)

	return stmt
}

func (p *Parser) parseFunctionStatement() *ast.FunctionStatement {

	stmt := &ast.FunctionStatement{}

	p.nextToken() // function name
	stmt.Name = p.curToken.Literal

	if p.peekToken.Type != lexer.LPAREN {
		return stmt
	}

	p.nextToken() // (
	stmt.Parameters = p.parseFunctionParameters()

	// 戻り値型省略可
	if p.peekToken.Type == lexer.IDENT {
		p.nextToken()
		stmt.ReturnType = p.curToken.Literal
	}

	if p.peekToken.Type != lexer.LBRACE {
		return stmt
	}

	p.nextToken() // {
	p.nextToken() // first stmt in body

	stmt.Body = p.parseBlockStatement()

	return stmt
}

func (p *Parser) parseFunctionParameters() []ast.Parameter {
	var params []ast.Parameter

	if p.peekToken.Type == lexer.RPAREN {
		p.nextToken()
		return params
	}

	for {
		p.nextToken() // param name

		param := ast.Parameter{
			Name: p.curToken.Literal,
		}

		if p.peekToken.Type == lexer.COLON {
			p.nextToken() // :
			p.nextToken() // type
			param.Type = p.curToken.Literal
		}

		params = append(params, param)

		if p.peekToken.Type == lexer.COMMA {
			p.nextToken()
			continue
		}

		if p.peekToken.Type == lexer.RPAREN {
			p.nextToken()
			break
		}
	}

	return params
}

func (p *Parser) parseStringLiteral() ast.Expression {
	return &ast.StringLiteral{
		Value: p.curToken.Literal,
	}
}

func (p *Parser) parseArrayLiteral() ast.Expression {
	array := &ast.ArrayLiteral{}

	array.Elements = p.parseExpressionList(lexer.RBRACKET)

	return array
}

func (p *Parser) parseIndexExpression(left ast.Expression) ast.Expression {
	exp := &ast.IndexExpression{
		Left: left,
	}

	p.nextToken()

	exp.Index = p.parseExpression(LOWEST)

	if p.peekToken.Type != lexer.RBRACKET {
		return nil
	}

	p.nextToken()

	return exp
}
