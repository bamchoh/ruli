package main

import (
	"fmt"
	"ruli/evaluator"
	"ruli/lexer"
	"ruli/parser"
)

func main() {
	input := `x := 0; y := 20;for i := 0; i < 5; i++ { x = x + 1 }`

	l := lexer.New(input)
	p := parser.New(l)
	program := p.ParseProgram()

	env := evaluator.NewEnvironment()
	evaluator.Eval(program, env)

	for _, name := range []string{"x", "y"} {
		v, _ := env.Get(name)
		fmt.Println(name, "=", v.Inspect(), v.Type())
	}
}
