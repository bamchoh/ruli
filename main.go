package main

import (
	"fmt"
	"ruli/evaluator"
	"ruli/lexer"
	"ruli/parser"
)

func main() {
	input := `x := 10
y := x > 5
z := x == 10
w := x < 3`

	l := lexer.New(input)
	p := parser.New(l)
	program := p.ParseProgram()

	env := evaluator.NewEnvironment()
	evaluator.Eval(program, env)

	for _, name := range []string{"x", "y", "z", "w"} {
		v, _ := env.Get(name)
		fmt.Println(name, "=", v.Inspect(), v.Type())
	}
}
