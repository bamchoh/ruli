package main

import (
	"fmt"
	"ruli/lexer"
)

func main() {
	l := lexer.New("x := 10")

	var tok lexer.Token
	for tok.Type != lexer.EOF {
		tok = l.NextToken()
		fmt.Println(tok.Type, tok.Literal)
	}
}
