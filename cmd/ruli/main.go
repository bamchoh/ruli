package main

import (
	"fmt"
	"os"
	"ruli/evaluator"
	"ruli/lexer"
	"ruli/object"
	"ruli/parser"
)

func main() {
	input := `flag := false
if flag {
	print("Flag is true")
} else {
	print("Flag is false")
}`

	l := lexer.New(input)
	p := parser.New(l)
	program := p.ParseProgram()

	env := object.NewEnvironment()
	result := evaluator.Eval(program, env)
	if errObj, ok := result.(*object.Error); ok {
		fmt.Fprintln(os.Stderr, errObj.Inspect())
		os.Exit(1)
	}
}
