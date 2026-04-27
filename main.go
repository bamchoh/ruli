package main

import (
	"fmt"
	"ruli/evaluator"
	"ruli/lexer"
	"ruli/parser"
)

func main() {
	input := `
x := 10
x = x + 20
y = x * 2`

	l := lexer.New(input)
	p := parser.New(l)
	program := p.ParseProgram()

	env := evaluator.NewEnvironment()
	evaluator.Eval(program, env)

	fmt.Println(env.Get("x"))
	fmt.Println(env.Get("y"))
}
