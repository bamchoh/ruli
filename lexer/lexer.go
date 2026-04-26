package lexer

type Lexer struct {
	input string
	pos   int
}

func New(input string) *Lexer {
	return &Lexer{input: input}
}

func (l *Lexer) NextToken() Token {
	if l.pos >= len(l.input) {
		return Token{
			Type:    EOF,
			Literal: "",
		}
	}

	ch := l.input[l.pos]
	l.pos++

	return Token{
		Type:    TokenType(ch),
		Literal: string(ch),
	}
}
