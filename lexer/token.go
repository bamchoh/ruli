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
	COLON   = ":"
	ASSIGN  = "="
	DECLARE = ":="

	PLUS     = "+"
	MINUS    = "-"
	BANG     = "!"
	ASTERISK = "*"
	SLASH    = "/"

	SEMICOLON = ";"

	INT  = "INT"
	REAL = "REAL"
	BOOL = "BOOL"

	EQ     = "=="
	NOT_EQ = "!="
	LT     = "<"
	GT     = ">"

	IF   = "IF"
	ELSE = "ELSE"

	LBRACE = "{"
	RBRACE = "}"
	LPAREN = "("
	RPAREN = ")"
	COMMA  = ","

	INC = "++"
	DEC = "--"
	FOR = "FOR"
)

var keywords = map[string]TokenType{
	"if":   IF,
	"else": ELSE,
	"for":  FOR,
}

func LookupIdent(ident string) TokenType {
	if tok, ok := keywords[ident]; ok {
		return tok
	}
	return IDENT
}
