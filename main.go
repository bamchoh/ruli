package main

import (
	"ruli/evaluator"
	"ruli/lexer"
	"ruli/object"
	"ruli/parser"
)

func main() {
	input := `
nums := "123"

for i := 0; i < len(nums); i++ {
	print(nums[i])
}
	`

	l := lexer.New(input)
	p := parser.New(l)
	program := p.ParseProgram()

	env := object.NewEnvironment()
	evaluator.Eval(program, env)
}
