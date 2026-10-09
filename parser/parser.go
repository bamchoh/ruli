package parser

import (
	"fmt"
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

// ParseError はパースエラーを行列位置とともに保持する
type ParseError struct {
	Message string

	Line   int
	Column int
}

func (e *ParseError) Error() string {
	return fmt.Sprintf("%d:%d: %s", e.Line, e.Column, e.Message)
}

type Parser struct {
	l *lexer.Lexer

	curToken  lexer.Token
	peekToken lexer.Token

	prefixParseFns map[lexer.TokenType]prefixParseFn
	infixParseFns  map[lexer.TokenType]infixParseFn

	errors []ParseError
}

// Errors はパース中に記録されたエラーを返す
func (p *Parser) Errors() []ParseError {
	return p.errors
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
	p.registerPrefix(lexer.TRUE, p.parseBooleanLiteral)
	p.registerPrefix(lexer.FALSE, p.parseBooleanLiteral)
	p.registerPrefix(lexer.LPAREN, p.parseParenthesized)

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

// errorAt は指定トークンの行列位置を添えてパースエラーを記録する
func (p *Parser) errorAt(token lexer.Token, format string, args ...interface{}) {
	p.errors = append(p.errors, ParseError{
		Message: fmt.Sprintf(format, args...),
		Line:    token.Line,
		Column:  token.Column,
	})
}

func (p *Parser) errorAtPeek(format string, args ...interface{}) {
	p.errorAt(p.peekToken, format, args...)
}

// expectPeek は peekToken が指定トークンなら true を返す。
// 異なる場合はエラーを記録し false を返す (トークンは消費しない)
func (p *Parser) expectPeek(tokType lexer.TokenType) bool {
	if p.peekToken.Type == tokType {
		return true
	}

	p.errorAtPeek("expected '%s', got '%s'", tokType, tokenLabel(p.peekToken))
	return false
}

// tokenLabel はエラーメッセージ表示用にトークンを整形する
// (EOF は literal が空なので TokenType で代替する)
func tokenLabel(tok lexer.Token) string {
	if tok.Literal != "" {
		return tok.Literal
	}
	return string(tok.Type)
}

func (p *Parser) ParseProgram() *ast.Program {
	program := &ast.Program{
		Token: p.curToken,
	}

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
		return &ast.BreakStatement{Token: p.curToken}

	case lexer.CONTINUE:
		return &ast.ContinueStatement{Token: p.curToken}

	case lexer.FUNC:
		return p.parseFunctionStatement()

	case lexer.RETURN:
		return p.parseReturnStatement()

	case lexer.IDENT:

		switch p.peekToken.Type {

		case lexer.DECLARE, lexer.COLON:
			return p.parseVarDeclStatement()

		case lexer.INC, lexer.DEC:
			return p.parseIncDecStatement()

		}

	}

	return p.parseAssignOrExpressionStatement()
}

// parseAssignOrExpressionStatement は式を 1 つパースする。
// 直後に = が続く場合は代入文として構築する (x = 10 / a[0] = 10)
func (p *Parser) parseAssignOrExpressionStatement() ast.Statement {
	expr := p.parseExpression(LOWEST)

	if p.peekToken.Type != lexer.ASSIGN {
		return &ast.ExpressionStatement{Expression: expr}
	}

	p.nextToken() // =
	p.nextToken() // value first token

	return &ast.AssignStatement{
		Left:  expr,
		Value: p.parseExpression(LOWEST),
	}
}

func (p *Parser) parseVarDeclStatement() *ast.VarDeclStatement {

	stmt := &ast.VarDeclStatement{
		Name: &ast.Identifier{
			Token: p.curToken,
			Value: p.curToken.Literal,
		},
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

		if stmt.Type == nil {
			p.errorAt(p.curToken, "unknown type '%s'", tokenLabel(p.curToken))
		}

		if p.peekToken.Type == lexer.ASSIGN {
			p.nextToken() // =
			p.nextToken() // expr first token

			stmt.Value = p.parseExpression(LOWEST)
		}

		return stmt
	}

	return stmt
}

func (p *Parser) parseType() ast.TypeNode {

	switch p.curToken.Type {

	case lexer.INT:
		return &ast.BasicType{Token: p.curToken, Name: "INT"}

	case lexer.REAL:
		return &ast.BasicType{Token: p.curToken, Name: "REAL"}

	case lexer.BOOL:
		return &ast.BasicType{Token: p.curToken, Name: "BOOL"}

	case lexer.IDENT:
		// distinct type / struct名 将来対応
		return &ast.BasicType{Token: p.curToken, Name: p.curToken.Literal}
	}

	return nil
}

func (p *Parser) parseExpression(precedence int) ast.Expression {

	prefix := p.prefixParseFns[p.curToken.Type]
	if prefix == nil {
		p.errorAt(p.curToken, "unexpected token '%s'", tokenLabel(p.curToken))
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
		p.errorAt(p.curToken, "invalid integer literal '%s'", p.curToken.Literal)
		return nil
	}
	return &ast.IntegerLiteral{
		Token: p.curToken,
		Value: int(value),
	}
}

// castableTypes はキャスト可能な型名 (SPEC 6.3: REAL(a) のような明示キャスト)
var castableTypes = map[string]bool{
	"INT": true, "SINT": true, "USINT": true, "BYTE": true,
	"UINT": true, "WORD": true, "DINT": true, "UDINT": true,
	"DWORD": true, "LINT": true, "ULINT": true, "LWORD": true,
	"REAL": true, "LREAL": true, "BOOL": true,
}

func (p *Parser) parseIdentifier() ast.Expression {
	if _, ok := castableTypes[p.curToken.Literal]; ok &&
		p.peekToken.Type == lexer.LPAREN {
		// 型名 + ( のキャスト: REAL(x)
		return p.parseCast()
	}

	return &ast.Identifier{
		Token: p.curToken,
		Value: p.curToken.Literal,
	}
}

func (p *Parser) parseCast() ast.Expression {
	token := p.curToken
	target := p.curToken.Literal

	p.nextToken() // (
	p.nextToken() // value first token

	value := p.parseExpression(LOWEST)

	if p.expectPeek(lexer.RPAREN) {
		p.nextToken() // consume )
	}

	return &ast.CastExpression{
		Token: token,
		Type:  target,
		Value: value,
	}
}

// parseParenthesized は ( expr ) をパースする
func (p *Parser) parseParenthesized() ast.Expression {
	p.nextToken() // expr first token

	exp := p.parseExpression(LOWEST)

	if p.expectPeek(lexer.RPAREN) {
		p.nextToken() // consume )
	}

	return exp
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

	stmt := &ast.IfStatement{
		Token: p.curToken,
	}

	p.nextToken() // condition first token

	stmt.Condition = p.parseExpression(LOWEST)

	if !p.expectPeek(lexer.LBRACE) {
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
				Token: p.curToken,
				Statements: []ast.Statement{
					p.parseIfStatement(),
				},
			}
			return stmt
		}

		if !p.expectPeek(lexer.LBRACE) {
			return stmt
		}

		p.nextToken() // {
		p.nextToken() // first stmt in block
		stmt.Alternative = p.parseBlockStatement()
	}

	return stmt
}

func (p *Parser) parseBlockStatement() *ast.BlockStatement {

	block := &ast.BlockStatement{
		Token: p.curToken,
	}

	for p.curToken.Type != lexer.RBRACE && p.curToken.Type != lexer.EOF {

		stmt := p.parseStatement()
		if stmt != nil {
			block.Statements = append(block.Statements, stmt)
		}

		p.nextToken()
	}

	if p.curToken.Type == lexer.EOF {
		p.errorAt(p.curToken, "expected '}', got EOF")
	}

	return block
}

func (p *Parser) parseIncDecStatement() *ast.IncDecStatement {
	stmt := &ast.IncDecStatement{
		Token: p.curToken,
		Name:  p.curToken.Literal,
	}

	p.nextToken() // ++ or --

	stmt.Operator = p.curToken.Literal

	return stmt
}

func (p *Parser) parseForStatement() *ast.ForStatement {

	stmt := &ast.ForStatement{
		Token: p.curToken,
	}

	p.nextToken() // init first token
	stmt.Init = p.parseStatement()

	if !p.expectPeek(lexer.SEMICOLON) {
		return stmt
	}

	p.nextToken() // ;
	p.nextToken() // condition first token
	stmt.Condition = p.parseExpression(LOWEST)

	if !p.expectPeek(lexer.SEMICOLON) {
		return stmt
	}

	p.nextToken() // ;
	p.nextToken() // post first token
	stmt.Post = p.parseStatement()

	if !p.expectPeek(lexer.LBRACE) {
		return stmt
	}

	p.nextToken() // {
	p.nextToken() // first stmt in body

	stmt.Body = p.parseBlockStatement()

	return stmt
}

func (p *Parser) parseCallExpression(function ast.Expression) ast.Expression {

	exp := &ast.CallExpression{
		Token:    p.curToken,
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

	if p.expectPeek(end) {
		p.nextToken() // consume end token
	}

	return list
}

func (p *Parser) parseReturnStatement() *ast.ReturnStatement {
	stmt := &ast.ReturnStatement{
		Token: p.curToken,
	}

	p.nextToken()
	stmt.Value = p.parseExpression(LOWEST)

	return stmt
}

func (p *Parser) parseFunctionStatement() *ast.FunctionStatement {

	stmt := &ast.FunctionStatement{
		Token: p.curToken,
	}

	p.nextToken() // function name
	stmt.Name = p.curToken.Literal

	if !p.expectPeek(lexer.LPAREN) {
		return stmt
	}

	p.nextToken() // (
	stmt.Parameters = p.parseFunctionParameters()

	// 戻り値型省略可
	if p.peekToken.Type == lexer.IDENT {
		p.nextToken()
		stmt.ReturnType = p.curToken.Literal
	}

	if !p.expectPeek(lexer.LBRACE) {
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
		Token: p.curToken,
		Value: p.curToken.Literal,
	}
}

func (p *Parser) parseBooleanLiteral() ast.Expression {
	return &ast.BooleanLiteral{
		Token: p.curToken,
		Value: p.curToken.Type == lexer.TRUE,
	}
}

func (p *Parser) parseArrayLiteral() ast.Expression {
	array := &ast.ArrayLiteral{
		Token: p.curToken,
	}

	array.Elements = p.parseExpressionList(lexer.RBRACKET)

	return array
}

func (p *Parser) parseIndexExpression(left ast.Expression) ast.Expression {
	exp := &ast.IndexExpression{
		Token: p.curToken,
		Left:  left,
	}

	p.nextToken()

	exp.Index = p.parseExpression(LOWEST)

	if !p.expectPeek(lexer.RBRACKET) {
		return exp
	}

	p.nextToken() // consume ]

	return exp
}
