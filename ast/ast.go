package ast

import "fmt"

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
	Name  string
	Value Expression
}

func (as *AssignStatement) statementNode() {}
func (as *AssignStatement) String() string {
	return as.Name + " := " + as.Value.String()
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
