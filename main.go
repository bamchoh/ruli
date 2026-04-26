package main

import (
	"fmt"
	"ruli/ast"
	"ruli/lexer"
	"ruli/parser"
)

func main() {
	input := "x := 10 + 20"

	l := lexer.New(input)
	p := parser.New(l)

	program := p.ParseProgram()

	stmt := program.Statements[0].(*ast.AssignStatement)

	expr := stmt.Value.(*ast.BinaryExpression)

	fmt.Println(expr.String())
}
