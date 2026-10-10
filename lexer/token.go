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
	MOD      = "%"

	// 複合代入 (SPEC 6.2)
	PLUS_ASSIGN  = "+="
	MINUS_ASSIGN = "-="
	MUL_ASSIGN   = "*="
	DIV_ASSIGN   = "/="
	MOD_ASSIGN   = "%="

	SEMICOLON = ";"

	INT  = "INT"
	REAL = "REAL"
	BOOL = "BOOL"

	EQ     = "=="
	NOT_EQ = "!="
	LT     = "<"
	GT     = ">"

	// SPEC 6.2 の残りの演算子
	LE = "<="
	GE = ">="

	AND = "&&"
	OR  = "||"

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
	DOT = "."

	BREAK    = "BREAK"
	CONTINUE = "CONTINUE"

	FUNC   = "FUNC"
	RETURN = "RETURN"

	STRING_LIT = "STRING_LIT"
	TRUE       = "TRUE"
	FALSE      = "FALSE"

	TYPE   = "TYPE"
	STRUCT = "STRUCT"

	// 将来対応の予約語 (SPEC 7.4 / 8.1 / 10)。
	// トークンとしては存在するが prefix パーサが無く、使用するとパースエラーになる
	NIL     = "NIL"
	REF     = "REF"
	DEFER   = "DEFER"
	SWITCH  = "SWITCH"
	CASE    = "CASE"
	DEFAULT = "DEFAULT"
)

var keywords = map[string]TokenType{
	"if":       IF,
	"else":     ELSE,
	"for":      FOR,
	"break":    BREAK,
	"continue": CONTINUE,
	"func":     FUNC,
	"return":   RETURN,
	"true":     TRUE,
	"false":    FALSE,
	"type":     TYPE,
	"struct":   STRUCT,
	"nil":      NIL,
	"ref":      REF,
	"defer":    DEFER,
	"switch":   SWITCH,
	"case":     CASE,
	"default":  DEFAULT,
}

func LookupIdent(ident string) TokenType {
	if tok, ok := keywords[ident]; ok {
		return tok
	}
	return IDENT
}
