package lexer

type TokenType string

type Token struct {
	Type    TokenType
	Literal string
}

const (
	// special
	EOF     = "EOF"
	ILLEGAL = "ILLEGAL"

	// identifiers + literals
	IDENT   = "IDENT"
	INT_LIT = "INT_LIT"

	// operators
	ASSIGN  = "="
	DECLARE = ":="
)
