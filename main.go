package main

import (
	"ruli/evaluator"
	"ruli/lexer"
	"ruli/object"
	"ruli/parser"
)

func main() {
	input := `
print("test")

x := "こんにちわ"
print(x)

print("a" + "b")
print("abc" == x)
	`

	l := lexer.New(input)
	p := parser.New(l)
	program := p.ParseProgram()

	env := object.NewEnvironment()
	evaluator.Eval(program, env)
}
