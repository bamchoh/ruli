package main

import (
	"fmt"
	"ruli/lexer"
)

func main() {
	l := lexer.New("")

	tok := l.NextToken()
	fmt.Println(tok.Type, tok.Literal)
}
