package main

import (
	"ruli/evaluator"
	"ruli/lexer"
	"ruli/object"
	"ruli/parser"
)

func main() {
	input := `
温度: INT = 10
print(温度)
	`

	l := lexer.New(input)
	p := parser.New(l)
	program := p.ParseProgram()

	env := object.NewEnvironment()
	evaluator.Eval(program, env)
}
