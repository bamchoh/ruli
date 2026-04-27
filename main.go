package main

import (
	"fmt"
	"ruli/lexer"
	"ruli/parser"
)

func main() {
	input := "x := 10 + 20 - 30 * 40 / 50"

	l := lexer.New(input)
	p := parser.New(l)

	program := p.ParseProgram()

	fmt.Println(program.Statements[0])

	var tok lexer.Token
	for tok.Type != lexer.EOF {
		tok = l.NextToken()
		fmt.Println(tok.Type, tok.Literal)
	}
}
