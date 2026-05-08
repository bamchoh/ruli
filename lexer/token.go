package lexer

type TokenType string

type Token struct {
	Type    TokenType
	Literal string

	Line   int
	Column int
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

	LBRACE   = "{"
	RBRACE   = "}"
	LPAREN   = "("
	RPAREN   = ")"
	LBRACKET = "["
	RBRACKET = "]"
	COMMA    = ","

	INC = "++"
	DEC = "--"
	FOR = "FOR"

	BREAK    = "BREAK"
	CONTINUE = "CONTINUE"

	FUNC   = "FUNC"
	RETURN = "RETURN"

	STRING_LIT = "STRING_LIT"
)

var keywords = map[string]TokenType{
	"if":       IF,
	"else":     ELSE,
	"for":      FOR,
	"break":    BREAK,
	"continue": CONTINUE,
	"func":     FUNC,
	"return":   RETURN,
}

func LookupIdent(ident string) TokenType {
	if tok, ok := keywords[ident]; ok {
		return tok
	}
	return IDENT
}
