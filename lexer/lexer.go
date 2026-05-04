package lexer

type Lexer struct {
	input string
	pos   int
	ch    byte
}

func New(input string) *Lexer {
	l := &Lexer{input: input}
	l.readChar()
	return l
}

func (l *Lexer) NextToken() Token {
	var tok Token

	l.skipWhitespace()

	switch l.ch {

	case '=':
		if l.peekChar() == '=' {
			ch := l.ch
			l.readChar()
			tok = Token{Type: EQ, Literal: string(ch) + string(l.ch)}
		} else {
			tok = Token{Type: ASSIGN, Literal: "="}
		}

	case '!':
		if l.peekChar() == '=' {
			ch := l.ch
			l.readChar()
			tok = Token{Type: NOT_EQ, Literal: string(ch) + string(l.ch)}
		} else {
			tok = Token{Type: ILLEGAL, Literal: "!"}
		}

	case '<':
		tok = Token{Type: LT, Literal: "<"}

	case '>':
		tok = Token{Type: GT, Literal: ">"}

	case '+':
		if l.peekChar() == '+' {
			ch := l.ch
			l.readChar()
			tok = Token{Type: INC, Literal: string(ch) + string(l.ch)}
		} else {
			tok = Token{Type: PLUS, Literal: "+"}
		}

	case '-':
		if l.peekChar() == '-' {
			ch := l.ch
			l.readChar()
			tok = Token{Type: DEC, Literal: string(ch) + string(l.ch)}
		} else {
			tok = Token{Type: MINUS, Literal: "-"}
		}

	case '*':
		tok = Token{Type: ASTERISK, Literal: "*"}
	case '/':
		tok = Token{Type: SLASH, Literal: "/"}

	case ':':
		if l.peekChar() == '=' {
			ch := l.ch
			l.readChar()
			tok = Token{Type: DECLARE, Literal: string(ch) + string(l.ch)}
		} else {
			tok = Token{Type: COLON, Literal: string(l.ch)}
		}

	case '{':
		tok = Token{Type: LBRACE, Literal: "{"}

	case '}':
		tok = Token{Type: RBRACE, Literal: "}"}

	case '(':
		tok = Token{Type: LPAREN, Literal: "("}

	case ')':
		tok = Token{Type: RPAREN, Literal: ")"}

	case ',':
		tok = Token{Type: COMMA, Literal: ","}

	case ';':
		tok = Token{Type: SEMICOLON, Literal: ";"}

	case '"':
		tok = Token{Type: STRING_LIT, Literal: l.readString()}

	case 0:
		tok = Token{Type: EOF, Literal: ""}

	default:
		if isLetter(l.ch) {
			lit := l.readIdentifier()
			tok.Type = LookupIdent(lit)
			tok = Token{Type: tok.Type, Literal: lit}
			return tok
		} else if isDigit(l.ch) {
			return Token{Type: INT_LIT, Literal: l.readNumber()}
		} else {
			tok = Token{Type: ILLEGAL, Literal: string(l.ch)}
		}
	}

	l.readChar()
	return tok
}

func (l *Lexer) readChar() {
	if l.pos >= len(l.input) {
		l.ch = 0
	} else {
		l.ch = l.input[l.pos]
	}
	l.pos++
}

func (l *Lexer) skipWhitespace() {
	for l.ch == ' ' || l.ch == '\t' || l.ch == '\n' || l.ch == '\r' {
		l.readChar()
	}
}

func (l *Lexer) peekChar() byte {
	if l.pos >= len(l.input) {
		return 0
	}
	return l.input[l.pos]
}

func (l *Lexer) readIdentifier() string {
	start := l.pos - 1

	for isLetter(l.ch) || isDigit(l.ch) || l.ch == '_' {
		l.readChar()
	}

	return l.input[start : l.pos-1]
}

func (l *Lexer) readNumber() string {
	start := l.pos - 1

	for isDigit(l.ch) {
		l.readChar()
	}

	return l.input[start : l.pos-1]
}

func isLetter(ch byte) bool {
	return ('a' <= ch && ch <= 'z') ||
		('A' <= ch && ch <= 'Z') ||
		ch == '_'
}

func isDigit(ch byte) bool {
	return '0' <= ch && ch <= '9'
}

func (l *Lexer) readString() string {
	l.readChar() // consume the opening quote
	start := l.pos - 1

	for l.ch != '"' && l.ch != 0 {
		l.readChar()
	}

	return l.input[start : l.pos-1]
}
