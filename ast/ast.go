package ast

import (
	"bytes"
	"fmt"
	"ruli/lexer"
	"strings"
)

type Node interface {
	String() string
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
	Statements []Statement
}

func (p *Program) String() string {
	return ""
}

type AssignStatement struct {
	Left  Expression
	Value Expression
}

func (as *AssignStatement) statementNode() {}
func (as *AssignStatement) String() string {
	return as.Left.String() + " = " + as.Value.String()
}

type VarDeclStatement struct {
	Name  string
	Type  TypeNode   // 型推論なら nil
	Value Expression // 初期値なしなら nil
}

func (vs *VarDeclStatement) statementNode() {}

func (vs *VarDeclStatement) String() string {
	return fmt.Sprintf("var %s : %s = %v", vs.Name, vs.Type.String(), vs.Value)
}

type Identifier struct {
	Token lexer.Token
	Value string
}

func (i *Identifier) expressionNode() {}
func (i *Identifier) String() string {
	return i.Value
}

type IntegerLiteral struct {
	Value int
}

func (i *IntegerLiteral) expressionNode() {}
func (i *IntegerLiteral) String() string {
	return fmt.Sprintf("%d", i.Value)
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

type BasicType struct {
	Name string
}

func (b *BasicType) typeNode() {}
func (b *BasicType) String() string {
	return b.Name
}

type BlockStatement struct {
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

type IfStatement struct {
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

type IncDecStatement struct {
	Name     string
	Operator string
}

func (is *IncDecStatement) statementNode() {}

func (is *IncDecStatement) String() string {
	return is.Name + is.Operator
}

type ForStatement struct {
	Init      Statement
	Condition Expression
	Post      Statement
	Body      *BlockStatement
}

func (fs *ForStatement) statementNode() {}

func (fs *ForStatement) String() string {
	return "for"
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

type CallExpression struct {
	Token     lexer.Token
	Function  Expression
	Arguments []Expression
}

func (ce *CallExpression) expressionNode() {}

func (ce *CallExpression) String() string {
	return "call"
}

type BreakStatement struct{}

func (bs *BreakStatement) statementNode() {}

func (bs *BreakStatement) String() string {
	return "break"
}

type ContinueStatement struct{}

func (cs *ContinueStatement) statementNode() {}

func (cs *ContinueStatement) String() string {
	return "continue"
}

type Parameter struct {
	Name string
	Type string
}

type FunctionStatement struct {
	Name       string
	Parameters []Parameter
	ReturnType string
	Body       *BlockStatement
}

func (fs *FunctionStatement) statementNode() {}

func (fs *FunctionStatement) String() string {
	return "func " + fs.Name
}

type ReturnStatement struct {
	Value Expression
}

func (rs *ReturnStatement) statementNode() {}

func (rs *ReturnStatement) String() string {
	return "return"
}

type StringLiteral struct {
	Value string
}

func (sl *StringLiteral) expressionNode() {}

func (sl *StringLiteral) String() string {
	return `"` + sl.Value + `"`
}

type ArrayLiteral struct {
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

type IndexExpression struct {
	Token lexer.Token
	Left  Expression
	Index Expression
}

func (ie *IndexExpression) expressionNode() {}

func (ie *IndexExpression) String() string {
	return "(" + ie.Left.String() + "[" + ie.Index.String() + "])"
}
