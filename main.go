package main

import (
	"ruli/evaluator"
	"ruli/lexer"
	"ruli/parser"
)

func main() {
	input := `
for i := 0; i < 10; i++ {
    if i == 3 {
        continue
    }

    if i == 7 {
        break
    }

    print(i)
}	
	`

	l := lexer.New(input)
	p := parser.New(l)
	program := p.ParseProgram()

	env := evaluator.NewEnvironment()
	evaluator.Eval(program, env)
}
