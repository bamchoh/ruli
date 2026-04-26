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
	return as.Name + " := ..."
}

type Identifier struct {
	Value string
}

func (i *Identifier) expressionNode() {}
func (i *Identifier) String() string {
	return i.Value
}

type IntegerLiteral struct {
	Value string
}

func (i *IntegerLiteral) expressionNode() {}
func (i *IntegerLiteral) String() string {
	return i.Value
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
