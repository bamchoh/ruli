package ast

import (
	"bytes"
	"fmt"
	"ruli/lexer"
	"strings"
)

type Node interface {
	String() string
	GetToken() lexer.Token
}

type Statement interface {
	Node
	statementNode()
}

type Expression interface {
	Node
	expressionNode()
}

type TypeNode interface {
	Node
	typeNode()
}

type Program struct {
	Token      lexer.Token
	Statements []Statement
}

func (p *Program) String() string {
	return ""
}

func (p *Program) GetToken() lexer.Token {
	return p.Token
}

type AssignStatement struct {
	Left  Expression
	Value Expression
}

func (as *AssignStatement) statementNode() {}
func (as *AssignStatement) String() string {
	return as.Left.String() + " = " + as.Value.String()
}

func (as *AssignStatement) GetToken() lexer.Token {
	return as.Left.GetToken()
}

type VarDeclStatement struct {
	Name  *Identifier
	Type  TypeNode   // 型推論なら nil
	Value Expression // 初期値なしなら nil
}

func (vs *VarDeclStatement) statementNode() {}

func (vs *VarDeclStatement) String() string {

	var out strings.Builder

	out.WriteString(vs.Name.String())

	if vs.Type != nil {
		out.WriteString(": ")
		out.WriteString(vs.Type.String())
	}

	if vs.Value != nil {
		out.WriteString(" = ")
		out.WriteString(vs.Value.String())
	}

	return out.String()
}

func (vs *VarDeclStatement) GetToken() lexer.Token {
	return vs.Name.GetToken()
}

type Identifier struct {
	Token lexer.Token
	Value string
}

func (i *Identifier) expressionNode() {}
func (i *Identifier) String() string {
	return i.Value
}
func (i *Identifier) GetToken() lexer.Token {
	return i.Token
}

type IntegerLiteral struct {
	Token lexer.Token
	Value int
}

func (i *IntegerLiteral) expressionNode() {}
func (i *IntegerLiteral) String() string {
	return fmt.Sprintf("%d", i.Value)
}

func (i *IntegerLiteral) GetToken() lexer.Token {
	return i.Token
}

type BinaryExpression struct {
	Left     Expression
	Operator string
	Right    Expression
}

func (be *BinaryExpression) expressionNode() {}

func (be *BinaryExpression) String() string {
	return fmt.Sprintf("(%s %s %s)",
		be.Left.String(),
		be.Operator,
		be.Right.String())
}

func (be *BinaryExpression) GetToken() lexer.Token {
	return be.Left.GetToken()
}

type BasicType struct {
	Token lexer.Token
	Name  string
}

func (b *BasicType) typeNode() {}
func (b *BasicType) String() string {
	return b.Name
}

func (b *BasicType) GetToken() lexer.Token {
	return b.Token
}

type BlockStatement struct {
	Token      lexer.Token
	Statements []Statement
}

func (bs *BlockStatement) statementNode() {}

func (bs *BlockStatement) String() string {
	var out bytes.Buffer

	for _, s := range bs.Statements {
		out.WriteString(s.String())
	}

	return out.String()
}

func (bs *BlockStatement) GetToken() lexer.Token {
	return bs.Token
}

type IfStatement struct {
	Token       lexer.Token
	Condition   Expression
	Consequence *BlockStatement
	Alternative *BlockStatement
}

func (ie *IfStatement) statementNode() {}

func (ie *IfStatement) String() string {
	var out bytes.Buffer

	out.WriteString("if")
	out.WriteString(ie.Condition.String())
	out.WriteString(" ")
	out.WriteString(ie.Consequence.String())

	if ie.Alternative != nil {
		out.WriteString("else ")
		out.WriteString(ie.Alternative.String())
	}

	return out.String()
}

func (ie *IfStatement) GetToken() lexer.Token {
	return ie.Token
}

type IncDecStatement struct {
	Token    lexer.Token
	Name     string
	Operator string
}

func (is *IncDecStatement) statementNode() {}

func (is *IncDecStatement) String() string {
	return is.Name + is.Operator
}

func (is *IncDecStatement) GetToken() lexer.Token {
	return is.Token
}

type ForStatement struct {
	Token     lexer.Token
	Init      Statement
	Condition Expression
	Post      Statement
	Body      *BlockStatement
}

func (fs *ForStatement) statementNode() {}

func (fs *ForStatement) String() string {
	return "for"
}

func (fs *ForStatement) GetToken() lexer.Token {
	return fs.Token
}

type ExpressionStatement struct {
	Expression Expression
}

func (es *ExpressionStatement) statementNode() {}

func (es *ExpressionStatement) String() string {
	if es.Expression != nil {
		return es.Expression.String()
	}
	return ""
}

func (es *ExpressionStatement) GetToken() lexer.Token {
	return es.Expression.GetToken()
}

type CallExpression struct {
	Token     lexer.Token
	Function  Expression
	Arguments []Expression
}

func (ce *CallExpression) expressionNode() {}

func (ce *CallExpression) String() string {
	return "call"
}

func (ce *CallExpression) GetToken() lexer.Token {
	return ce.Token
}

type BreakStatement struct {
	Token lexer.Token
}

func (bs *BreakStatement) statementNode() {}

func (bs *BreakStatement) String() string {
	return "break"
}

func (bs *BreakStatement) GetToken() lexer.Token {
	return bs.Token
}

type ContinueStatement struct {
	Token lexer.Token
}

func (cs *ContinueStatement) statementNode() {}

func (cs *ContinueStatement) String() string {
	return "continue"
}

func (cs *ContinueStatement) GetToken() lexer.Token {
	return cs.Token
}

type Parameter struct {
	Name string
	Type string
}

type FunctionStatement struct {
	Token      lexer.Token
	Name       string
	Parameters []Parameter
	ReturnType string
	Body       *BlockStatement
}

func (fs *FunctionStatement) statementNode() {}

func (fs *FunctionStatement) String() string {
	return "func " + fs.Name
}

func (fs *FunctionStatement) GetToken() lexer.Token {
	return fs.Token
}

type ReturnStatement struct {
	Token lexer.Token
	Value Expression
}

func (rs *ReturnStatement) statementNode() {}

func (rs *ReturnStatement) String() string {
	return "return"
}

func (rs *ReturnStatement) GetToken() lexer.Token {
	return rs.Token
}

type StringLiteral struct {
	Token lexer.Token
	Value string
}

func (sl *StringLiteral) expressionNode() {}

func (sl *StringLiteral) String() string {
	return `"` + sl.Value + `"`
}

func (sl *StringLiteral) GetToken() lexer.Token {
	return sl.Token
}

type ArrayLiteral struct {
	Token    lexer.Token
	Elements []Expression
}

func (al *ArrayLiteral) expressionNode() {}

func (al *ArrayLiteral) String() string {
	var out strings.Builder

	out.WriteString("[")

	for i, e := range al.Elements {
		if i > 0 {
			out.WriteString(", ")
		}
		out.WriteString(e.String())
	}

	out.WriteString("]")

	return out.String()
}

func (al *ArrayLiteral) GetToken() lexer.Token {
	return al.Token
}

type IndexExpression struct {
	Token lexer.Token
	Left  Expression
	Index Expression
}

func (ie *IndexExpression) expressionNode() {}

func (ie *IndexExpression) String() string {
	return "(" + ie.Left.String() + "[" + ie.Index.String() + "])"
}

func (ie *IndexExpression) GetToken() lexer.Token {
	return ie.Token
}
