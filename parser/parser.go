package parser

import (
	"fmt"
	"ruli/ast"
	"ruli/lexer"
	"strconv"
)

const (
	LOWEST = iota
	LOGICALOR
	LOGICALAND
	EQUALS
	LESSGREATER
	SUM
	PRODUCT
	CALL
	INDEX
)

var precedences = map[lexer.TokenType]int{
	lexer.OR:       LOGICALOR,
	lexer.AND:      LOGICALAND,
	lexer.EQ:       EQUALS,
	lexer.NOT_EQ:   EQUALS,
	lexer.LT:       LESSGREATER,
	lexer.GT:       LESSGREATER,
	lexer.LE:       LESSGREATER,
	lexer.GE:       LESSGREATER,
	lexer.PLUS:     SUM,
	lexer.MINUS:    SUM,
	lexer.ASTERISK: PRODUCT,
	lexer.SLASH:    PRODUCT,
	lexer.MOD:      PRODUCT,
	lexer.LPAREN:   CALL,
	lexer.LBRACKET: INDEX,
	lexer.DOT:      INDEX,
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

	// struct リテラル (ident{...}) の許可。
	// if / for の条件部・switch の値では `if flag { }` との曖昧性を避けるため
	// 一時的に無効化する (Go と同じ規則。括弧内なら有効)
	allowStructLiteral bool
}

// Errors はパース中に記録されたエラーを返す
func (p *Parser) Errors() []ParseError {
	return p.errors
}

func New(l *lexer.Lexer) *Parser {
	p := &Parser{
		l:                  l,
		prefixParseFns:     make(map[lexer.TokenType]prefixParseFn),
		infixParseFns:      make(map[lexer.TokenType]infixParseFn),
		allowStructLiteral: true,
	}

	p.registerPrefix(lexer.INT_LIT, p.parseIntegerLiteral)
	p.registerPrefix(lexer.REAL_LIT, p.parseRealLiteral)
	p.registerPrefix(lexer.IDENT, p.parseIdentifier)
	p.registerPrefix(lexer.STRING_LIT, p.parseStringLiteral)
	p.registerPrefix(lexer.LBRACKET, p.parseArrayLiteral)
	p.registerPrefix(lexer.TRUE, p.parseBooleanLiteral)
	p.registerPrefix(lexer.FALSE, p.parseBooleanLiteral)
	p.registerPrefix(lexer.LPAREN, p.parseParenthesized)
	p.registerPrefix(lexer.MINUS, p.parsePrefixExpression)
	p.registerPrefix(lexer.BANG, p.parsePrefixExpression)

	p.registerInfix(lexer.PLUS, p.parseInfixExpression)
	p.registerInfix(lexer.MINUS, p.parseInfixExpression)
	p.registerInfix(lexer.ASTERISK, p.parseInfixExpression)
	p.registerInfix(lexer.SLASH, p.parseInfixExpression)
	p.registerInfix(lexer.MOD, p.parseInfixExpression)
	p.registerInfix(lexer.EQ, p.parseInfixExpression)
	p.registerInfix(lexer.NOT_EQ, p.parseInfixExpression)
	p.registerInfix(lexer.LT, p.parseInfixExpression)
	p.registerInfix(lexer.GT, p.parseInfixExpression)
	p.registerInfix(lexer.LE, p.parseInfixExpression)
	p.registerInfix(lexer.GE, p.parseInfixExpression)
	p.registerInfix(lexer.AND, p.parseLogicalExpression)
	p.registerInfix(lexer.OR, p.parseLogicalExpression)
	p.registerInfix(lexer.LPAREN, p.parseCallExpression)
	p.registerInfix(lexer.LBRACKET, p.parseIndexExpression)
	p.registerInfix(lexer.DOT, p.parseDotSelector)

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

// syncTo は panic モード復帰: 指定トークン (または EOF) に到達するまで読み進める。
// エラーのカスケードを防ぐため、区切りに到達したところから解析を再開する
func (p *Parser) syncTo(tokens ...lexer.TokenType) {
	set := make(map[lexer.TokenType]bool, len(tokens))
	for _, t := range tokens {
		set[t] = true
	}

	for !set[p.curToken.Type] && p.curToken.Type != lexer.EOF {
		p.nextToken()
	}
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

	case lexer.TYPE:
		return p.parseTypeDeclStatement()

	case lexer.SWITCH:
		return p.parseSwitchStatement()

	case lexer.IDENT:

		// 複数代入先: x, ok := f() / x, ok = f()
		if p.peekToken.Type == lexer.COMMA {
			return p.parseMultiStatement()
		}

		switch p.peekToken.Type {

		case lexer.DECLARE, lexer.COLON:
			return p.parseVarDeclStatement()

		case lexer.INC, lexer.DEC:
			return p.parseIncDecStatement()

		}

	}

	return p.parseAssignOrExpressionStatement()
}

// parseMultiStatement は x, ok := f() / x, ok = f() をパースする (SPEC 5.4)
func (p *Parser) parseMultiStatement() ast.Statement {
	names := []*ast.Identifier{{
		Token: p.curToken,
		Value: p.curToken.Literal,
	}}

	for p.peekToken.Type == lexer.COMMA {
		p.nextToken() // ,
		p.nextToken() // next identifier
		names = append(names, &ast.Identifier{
			Token: p.curToken,
			Value: p.curToken.Literal,
		})
	}

	switch p.peekToken.Type {
	case lexer.DECLARE:
		p.nextToken() // :=
		p.nextToken() // value first token
		return &ast.MultiVarDeclStatement{
			Token:  names[0].Token,
			Names:  names,
			Values: p.parseExpression(LOWEST),
		}

	case lexer.ASSIGN:
		p.nextToken() // =
		p.nextToken() // value first token
		lefts := make([]ast.Expression, 0, len(names))
		for _, name := range names {
			lefts = append(lefts, name)
		}
		return &ast.MultiAssignStatement{
			Token: names[0].Token,
			Lefts: lefts,
			Value: p.parseExpression(LOWEST),
		}
	}

	p.errorAtPeek("expected ':=' or '=', got '%s'", tokenLabel(p.peekToken))
	return nil
}

// parseAssignOrExpressionStatement は式を 1 つパースする。
// 直後に = が続く場合は代入文、+= 等なら複合代入文として構築する
func (p *Parser) parseAssignOrExpressionStatement() ast.Statement {
	expr := p.parseExpression(LOWEST)

	switch p.peekToken.Type {
	case lexer.ASSIGN:
		p.nextToken() // =
		p.nextToken() // value first token

		return &ast.AssignStatement{
			Left:  expr,
			Value: p.parseExpression(LOWEST),
		}

	case lexer.PLUS_ASSIGN, lexer.MINUS_ASSIGN, lexer.MUL_ASSIGN,
		lexer.DIV_ASSIGN, lexer.MOD_ASSIGN:
		// 複合代入 (SPEC 6.2): x += 1 / a[0] *= 2 / m.speed -= 1
		opToken := p.peekToken
		p.nextToken() // 演算子
		p.nextToken() // value first token

		return &ast.CompoundAssignStatement{
			Token:    expr.GetToken(),
			Target:   expr,
			Operator: opToken.Literal,
			Value:    p.parseExpression(LOWEST),
		}
	}

	return &ast.ExpressionStatement{Expression: expr}
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

		// parseType が独自のエラーメッセージを出す場合 (ref の誤用など) は追記しない
		if stmt.Type == nil && p.curToken.Type != lexer.REF {
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

	case lexer.REF:
		// ref はパラメータの型の一部でのみ有効 (SPEC 5.5)
		p.errorAt(p.curToken, "ref is only allowed in function parameters")
		return nil

	case lexer.INT:
		return &ast.BasicType{Token: p.curToken, Name: "INT"}

	case lexer.REAL:
		return &ast.BasicType{Token: p.curToken, Name: "REAL"}

	case lexer.BOOL:
		return &ast.BasicType{Token: p.curToken, Name: "BOOL"}

	case lexer.IDENT:
		// distinct type / struct名 将来対応
		return &ast.BasicType{Token: p.curToken, Name: p.curToken.Literal}

	case lexer.LBRACKET:
		return p.parseArrayType()
	}

	return nil
}

// parseArrayType は [10]INT / [3][3]INT のような固定長配列型をパースする (SPEC 4.4)
func (p *Parser) parseArrayType() ast.TypeNode {
	token := p.curToken // [

	p.nextToken() // size literal

	if p.curToken.Type != lexer.INT_LIT {
		p.errorAt(p.curToken, "array size must be an integer literal, got '%s'", tokenLabel(p.curToken))
		return nil
	}

	if p.curToken.Literal == "" {
		return nil
	}

	size, err := strconv.Atoi(p.curToken.Literal)
	if err != nil {
		p.errorAt(p.curToken, "array size must be an integer literal: %s", p.curToken.Literal)
		return nil
	}
	if size <= 0 {
		p.errorAt(p.curToken, "array size must be positive, got %d", size)
		return nil
	}

	if !p.expectPeek(lexer.RBRACKET) {
		return nil
	}

	p.nextToken() // ]
	p.nextToken() // element type first token

	elem := p.parseType()
	if elem == nil {
		p.errorAt(p.curToken, "unknown element type '%s'", tokenLabel(p.curToken))
		return nil
	}

	return &ast.ArrayType{
		Token: token,
		Size:  size,
		Elem:  elem,
	}
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

func (p *Parser) parseRealLiteral() ast.Expression {
	value, err := strconv.ParseFloat(p.curToken.Literal, 64)
	if err != nil {
		p.errorAt(p.curToken, "invalid real literal '%s'", p.curToken.Literal)
		return nil
	}
	return &ast.RealLiteral{
		Token: p.curToken,
		Value: value,
	}
}

// typeNames は有効な型名 (SPEC 4.1 / 4.3 の基本型)
var typeNames = map[string]bool{
	"INT": true, "SINT": true, "USINT": true, "BYTE": true,
	"UINT": true, "WORD": true, "DINT": true, "UDINT": true,
	"DWORD": true, "LINT": true, "ULINT": true, "LWORD": true,
	"REAL": true, "LREAL": true, "BOOL": true, "STRING": true,
}

func resolveTypeName(tok lexer.Token) (string, bool) {
	return tok.Literal, typeNames[tok.Literal]
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

	ident := &ast.Identifier{
		Token: p.curToken,
		Value: p.curToken.Literal,
	}

	if p.allowStructLiteral && p.peekToken.Type == lexer.LBRACE {
		// 構造体リテラル: Motor{speed: 10}
		return p.parseStructLiteral(ident)
	}

	return ident
}

// parseDotSelector はフィールドアクセスの postfix (.field) をパースする。
// m.speed だけでなく Motor{...}.speed や f(x).y にも有効
func (p *Parser) parseDotSelector(left ast.Expression) ast.Expression {
	p.nextToken() // field name

	if p.curToken.Type != lexer.IDENT {
		p.errorAt(p.curToken, "expected field name, got '%s'", tokenLabel(p.curToken))
		return left
	}

	return &ast.SelectorExpression{
		Token:  left.GetToken(),
		Object: left,
		Field:  p.curToken.Literal,
	}
}

// parseStructLiteral は Motor{speed: 10, enable: true} をパースする。
// 未指定フィールドはゼロ値、区切りは , または ;
func (p *Parser) parseStructLiteral(ident *ast.Identifier) ast.Expression {
	lit := &ast.StructLiteral{
		Token: ident.Token,
		Name:  ident.Value,
	}

	p.nextToken() // {
	p.nextToken() // first field (or })

	for p.curToken.Type != lexer.RBRACE && p.curToken.Type != lexer.EOF {
		fieldName := p.curToken.Literal

		if !p.expectPeek(lexer.COLON) {
			return lit
		}

		p.nextToken() // :
		p.nextToken() // value first token

		value := p.parseExpression(LOWEST)

		lit.Fields = append(lit.Fields, ast.StructLiteralField{
			Name:  fieldName,
			Value: value,
		})

		// 区切り: , / ; で次のフィールド、} で終了
		switch p.peekToken.Type {
		case lexer.COMMA, lexer.SEMICOLON:
			p.nextToken() // separator
			p.nextToken() // next field name
		case lexer.RBRACE:
			p.nextToken() // }
		default:
			p.errorAtPeek("expected ',' or '}', got '%s'", tokenLabel(p.peekToken))
			return lit
		}
	}

	if p.curToken.Type == lexer.EOF {
		p.errorAt(p.curToken, "expected '}', got EOF")
	}

	return lit
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

// parseParenthesized は ( expr ) をパースする。
// 条件部で struct リテラルを書きたい場合は括弧で括る (Go と同じ規則)
func (p *Parser) parseParenthesized() ast.Expression {
	prev := p.allowStructLiteral
	p.allowStructLiteral = true

	p.nextToken() // expr first token

	exp := p.parseExpression(LOWEST)

	p.allowStructLiteral = prev

	if p.expectPeek(lexer.RPAREN) {
		p.nextToken() // consume )
	}

	return exp
}

// parsePrefixExpression は -x / !flag をパースする。
// オペランドは PRODUCT 相当の優先度で読む ( -a * b は (-a) * b )
func (p *Parser) parsePrefixExpression() ast.Expression {
	stmt := &ast.PrefixExpression{
		Token:    p.curToken,
		Operator: p.curToken.Literal,
	}

	p.nextToken()
	stmt.Right = p.parseExpression(PRODUCT)

	return stmt
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

// parseLogicalExpression は && / || をパースする (short-circuit 用に BinaryExpression と区別)
func (p *Parser) parseLogicalExpression(left ast.Expression) ast.Expression {
	expr := &ast.LogicalExpression{
		Token:    p.curToken,
		Operator: p.curToken.Literal,
		Left:     left,
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

	// 条件部では struct リテラルを禁止 (if flag { } との曖昧性の解消)
	prev := p.allowStructLiteral
	p.allowStructLiteral = false

	p.nextToken() // condition first token

	stmt.Condition = p.parseExpression(LOWEST)

	p.allowStructLiteral = prev

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

	// 節の境界 (;)。欠けていてもカスケードしない: 次節の先頭から解析を続ける
	if p.peekToken.Type == lexer.SEMICOLON {
		p.nextToken() // ;
		p.nextToken() // condition first token
	} else {
		p.errorAtPeek("expected ';', got '%s'", tokenLabel(p.peekToken))
		p.nextToken() // condition first token
	}

	// for 条件部でも struct リテラルを禁止 (ブロック {} との曖昧性の解消)
	prev := p.allowStructLiteral
	p.allowStructLiteral = false
	stmt.Condition = p.parseExpression(LOWEST)
	p.allowStructLiteral = prev

	if p.peekToken.Type == lexer.SEMICOLON {
		p.nextToken() // ;
		p.nextToken() // post first token
	} else {
		p.errorAtPeek("expected ';', got '%s'", tokenLabel(p.peekToken))
		p.nextToken() // post first token
	}
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

// parseExpressionList は ( 要素 ) / [ 要素 ] のような要素リストをパースする。
// 区切りが欠けた場合は panic モード復帰する (カスケード防止)
func (p *Parser) parseExpressionList(end lexer.TokenType) []ast.Expression {
	var list []ast.Expression

	if p.peekToken.Type == end {
		p.nextToken()
		return list
	}

	p.nextToken()
	list = append(list, p.parseExpression(LOWEST))

	for {
		switch {
		case p.peekToken.Type == lexer.COMMA:
			p.nextToken() // ,
			if p.peekToken.Type == end {
				// 末尾カンマ
				p.nextToken()
				return list
			}
			p.nextToken() // next element first token
			list = append(list, p.parseExpression(LOWEST))

		case p.peekToken.Type == end:
			p.nextToken() // end を消費して終了
			return list

		default:
			if p.peekToken.Type == lexer.EOF {
				p.errorAtPeek("expected ',' or '%s', got EOF", string(end))
				return list
			}
			p.errorAtPeek("expected ',' or '%s', got '%s'", string(end), tokenLabel(p.peekToken))

			// panic モード復帰: 区切りまでスキップして解析を再開する
			p.syncTo(lexer.COMMA, end)

			switch p.curToken.Type {
			case lexer.COMMA:
				p.nextToken() // , を消費 → cur は次の要素の先頭
				if p.curToken.Type == end {
					p.nextToken()
					return list
				}
				list = append(list, p.parseExpression(LOWEST))

			case end:
				p.nextToken()
				return list

			default:
				return list // EOF
			}
		}
	}
}

// parseTypeDeclStatement は type Motor struct { ... } をパースする
func (p *Parser) parseTypeDeclStatement() *ast.TypeDeclStatement {
	stmt := &ast.TypeDeclStatement{
		Token: p.curToken,
	}

	p.nextToken() // type name
	stmt.Name = p.curToken.Literal

	if p.peekToken.Type != lexer.STRUCT {
		// distinct 型 (SPEC 4.2): type TEMP INT
		if p.peekToken.Type == lexer.IDENT || p.peekToken.Type == lexer.LBRACKET {
			p.nextToken() // base type first token
			stmt.Base = p.parseType()
			if stmt.Base == nil {
				p.errorAt(p.curToken, "unknown base type '%s'", tokenLabel(p.curToken))
			}
		} else {
			p.errorAtPeek("expected 'struct' or a base type, got '%s'", tokenLabel(p.peekToken))
		}
		return stmt
	}

	p.nextToken() // struct
	p.nextToken() // {
	p.nextToken() // first field or }

	for p.curToken.Type != lexer.RBRACE && p.curToken.Type != lexer.EOF {
		field := ast.StructField{
			Name: p.curToken.Literal,
		}

		if !p.expectPeek(lexer.COLON) {
			return stmt
		}

		p.nextToken() // :
		p.nextToken() // type name

		field.Type = p.parseType()
		if field.Type == nil {
			p.errorAt(p.curToken, "unknown type '%s'", tokenLabel(p.curToken))
		}

		stmt.Fields = append(stmt.Fields, field)

		// 次のフィールド (または }) へ
		p.nextToken()
	}

	if p.curToken.Type == lexer.EOF {
		p.errorAt(p.curToken, "expected '}', got EOF")
	}

	return stmt
}

// parseSwitchStatement は switch 文 (SPEC 7.4) をパースする。fallthrough なし
func (p *Parser) parseSwitchStatement() *ast.SwitchStatement {
	stmt := &ast.SwitchStatement{
		Token: p.curToken,
	}

	p.nextToken() // value first token

	// switch 値でも struct リテラルを禁止 (case ブロック {} との曖昧性の解消)
	prev := p.allowStructLiteral
	p.allowStructLiteral = false
	stmt.Value = p.parseExpression(LOWEST)
	p.allowStructLiteral = prev

	if !p.expectPeek(lexer.LBRACE) {
		return stmt
	}

	p.nextToken() // {
	p.nextToken() // case / default / }

	for p.curToken.Type != lexer.RBRACE && p.curToken.Type != lexer.EOF {
		switch p.curToken.Type {
		case lexer.CASE:
			sc := p.parseSwitchCase()
			if sc != nil {
				stmt.Cases = append(stmt.Cases, sc)
			}

		case lexer.DEFAULT:
			if stmt.Default != nil {
				p.errorAt(p.curToken, "duplicate default case")
			}

			if !p.expectPeek(lexer.COLON) {
				return stmt
			}

			p.nextToken() // :
			p.nextToken() // first stmt in body (or case/})

			if stmt.Default == nil {
				stmt.Default = &ast.SwitchCase{Token: p.curToken, Body: p.parseCaseBody()}
			} else {
				p.parseCaseBody() // 重複 default は解析して捨てる
			}

		default:
			p.errorAt(
				p.curToken,
				"unexpected token '%s' in switch, expected 'case' or 'default'",
				tokenLabel(p.curToken),
			)
			return stmt
		}
	}

	if p.curToken.Type == lexer.EOF {
		p.errorAt(p.curToken, "expected '}', got EOF")
	}

	return stmt
}

// parseSwitchCase は case 1, 2: から本体までをパースする
func (p *Parser) parseSwitchCase() *ast.SwitchCase {
	sc := &ast.SwitchCase{
		Token: p.curToken,
	}

	p.nextToken() // first value
	sc.Values = append(sc.Values, p.parseExpression(LOWEST))

	for p.peekToken.Type == lexer.COMMA {
		p.nextToken() // ,
		p.nextToken() // next value
		sc.Values = append(sc.Values, p.parseExpression(LOWEST))
	}

	if !p.expectPeek(lexer.COLON) {
		// panic モード復帰: : または次の case / default / } までスキップする
		p.syncTo(lexer.COLON, lexer.CASE, lexer.DEFAULT, lexer.RBRACE)
		if p.curToken.Type == lexer.COLON {
			p.nextToken() // :
			p.nextToken() // first stmt in body
			sc.Body = p.parseCaseBody()
		}
		return sc
	}

	p.nextToken() // :
	p.nextToken() // first stmt in body

	sc.Body = p.parseCaseBody()

	return sc
}

// parseCaseBody は case 本体の文を CASE / DEFAULT / RBRACE まで解析する
func (p *Parser) parseCaseBody() *ast.BlockStatement {
	body := &ast.BlockStatement{
		Token: p.curToken,
	}

	for p.curToken.Type != lexer.CASE && p.curToken.Type != lexer.DEFAULT &&
		p.curToken.Type != lexer.RBRACE && p.curToken.Type != lexer.EOF {

		stmt := p.parseStatement()
		if stmt != nil {
			body.Statements = append(body.Statements, stmt)
		}

		p.nextToken()
	}

	return body
}

func (p *Parser) parseReturnStatement() *ast.ReturnStatement {
	stmt := &ast.ReturnStatement{
		Token: p.curToken,
	}

	p.nextToken()
	stmt.Values = append(stmt.Values, p.parseExpression(LOWEST))

	// 複数戻り値 (SPEC 5.3): return a % b, b != 0
	for p.peekToken.Type == lexer.COMMA {
		p.nextToken() // ,
		p.nextToken() // next value first token
		stmt.Values = append(stmt.Values, p.parseExpression(LOWEST))
	}

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

	// 戻り値型省略可 (配列型 [4]INT も可)
	if p.peekToken.Type == lexer.IDENT || p.peekToken.Type == lexer.LBRACKET {
		p.nextToken()
		stmt.ReturnTypes = []ast.TypeNode{p.parseType()}
	}

	// 複数戻り値型 (SPEC 5.3): (INT, BOOL)
	if p.peekToken.Type == lexer.LPAREN {
		p.nextToken() // (
		p.nextToken() // first return type

		for p.curToken.Type != lexer.RPAREN && p.curToken.Type != lexer.EOF {
			if p.curToken.Type == lexer.COMMA {
				p.nextToken()
				continue
			}

			rt := p.parseType()
			if rt == nil {
				p.errorAt(p.curToken, "unknown type '%s'", tokenLabel(p.curToken))
			}
			stmt.ReturnTypes = append(stmt.ReturnTypes, rt)
			p.nextToken() // next return type or )
		}

		if p.curToken.Type == lexer.EOF {
			p.errorAt(p.curToken, "expected ')', got EOF")
		}
	}

	if !p.expectPeek(lexer.LBRACE) {
		return stmt
	}

	p.nextToken() // {
	p.nextToken() // first stmt in body

	stmt.Body = p.parseBlockStatement()

	return stmt
}

// parseFunctionParameters はパラメータリストをパースする。
// ループの invariant は「cur がパラメータ名の先頭」
func (p *Parser) parseFunctionParameters() []ast.Parameter {
	var params []ast.Parameter

	if p.peekToken.Type == lexer.RPAREN {
		p.nextToken()
		return params
	}

	p.nextToken() // first param name

	for {
		switch {
		case p.curToken.Type == lexer.RPAREN:
			p.nextToken() // ) を消費して終了
			return params

		case p.curToken.Type == lexer.EOF:
			p.errorAt(p.curToken, "expected ')', got EOF")
			return params
		}

		param := ast.Parameter{
			Name: p.curToken.Literal,
		}

		// パラメータは必ず型付き (SPEC 5.1)
		hadTypeError := false
		if p.peekToken.Type != lexer.COLON {
			p.errorAtPeek("parameter '%s' requires a type", param.Name)
			hadTypeError = true
		} else {
			p.nextToken() // colon
			p.nextToken() // type first token (または ref)

			if p.curToken.Type == lexer.REF {
				// ref はパラメータ型の一部 (SPEC 5.5 / IEC VAR_IN_OUT): m: ref Motor
				if isTypeStartToken(p.peekToken.Type) {
					param.Ref = true
					p.nextToken() // type first token

					param.Type = p.parseType()
					if param.Type == nil {
						p.errorAt(p.curToken, "unknown type '%s'", tokenLabel(p.curToken))
						hadTypeError = true
					}
				} else {
					p.errorAtPeek("ref parameter requires a type")
					hadTypeError = true
				}
			} else {
				param.Type = p.parseType()
				if param.Type == nil {
					p.errorAt(p.curToken, "unknown type '%s'", tokenLabel(p.curToken))
					hadTypeError = true
				}
			}
		}

		params = append(params, param)

		// 型エラーの復帰: 同じ位置で区切りエラーを重複報告しない
		if hadTypeError {
			p.syncTo(lexer.COMMA, lexer.RPAREN)

			switch p.curToken.Type {
			case lexer.COMMA:
				p.nextToken() // , を消費 → cur は次のパラメータ名
				continue
			case lexer.RPAREN:
				p.nextToken()
				return params
			default:
				return params // EOF
			}
		}

		// 区切り / 復帰 (panic モード: , か ) までスキップして解析を再開)
		switch {
		case p.peekToken.Type == lexer.COMMA:
			p.nextToken() // ,
			if p.peekToken.Type == lexer.RPAREN {
				// 末尾カンマ
				p.nextToken()
				return params
			}
			p.nextToken() // next param name

		case p.peekToken.Type == lexer.RPAREN:
			p.nextToken()
			return params

		default:
			if p.peekToken.Type == lexer.EOF {
				p.errorAtPeek("expected ',' or ')', got EOF")
				return params
			}
			p.errorAtPeek("expected ',' or ')', got '%s'", tokenLabel(p.peekToken))
			p.syncTo(lexer.COMMA, lexer.RPAREN)

			switch p.curToken.Type {
			case lexer.COMMA:
				p.nextToken() // , を消費 → cur は次のパラメータ名
				continue
			case lexer.RPAREN:
				p.nextToken()
				return params
			default:
				return params // EOF
			}
		}
	}
}

// isTypeStartToken は型の開始になり得るトークンか (IDENT / 配列型の [ )
func isTypeStartToken(t lexer.TokenType) bool {
	return t == lexer.IDENT || t == lexer.LBRACKET
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
