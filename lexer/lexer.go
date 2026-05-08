package lexer

import (
	"strings"
	"unicode"
)

type Lexer struct {
	input        []rune
	position     int
	readPosition int
	ch           rune
}

func New(input string) *Lexer {
	l := &Lexer{
		input: []rune(input),
	}
	l.readChar()
	return l
}

func (l *Lexer) NextToken() Token {
	var tok Token

	l.skipWhitespace()

	if l.ch == '#' {
		l.skipComment()
		return l.NextToken()
	}

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

	case '[':
		tok = Token{Type: LBRACKET, Literal: "["}

	case ']':
		tok = Token{Type: RBRACKET, Literal: "]"}

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
	if l.readPosition >= len(l.input) {
		l.ch = 0
	} else {
		l.ch = l.input[l.readPosition]
	}

	l.position = l.readPosition
	l.readPosition++
}

func (l *Lexer) skipWhitespace() {
	for l.ch == ' ' || l.ch == '\t' || l.ch == '\n' || l.ch == '\r' {
		l.readChar()
	}
}

func (l *Lexer) peekChar() rune {
	if l.readPosition >= len(l.input) {
		return 0
	}
	return l.input[l.readPosition]
}

func (l *Lexer) readIdentifier() string {
	start := l.position

	for isLetter(l.ch) || isDigit(l.ch) {
		l.readChar()
	}

	return string(l.input[start:l.position])
}

func (l *Lexer) readNumber() string {
	start := l.position

	for isDigit(l.ch) {
		l.readChar()
	}

	return string(l.input[start:l.position])
}

func isLetter(ch rune) bool {
	return unicode.IsLetter(ch) || ch == '_'
}

func isDigit(ch rune) bool {
	return '0' <= ch && ch <= '9'
}

func (l *Lexer) readString() string {
	var result strings.Builder

	for {
		l.readChar()

		if l.ch == 0 || l.ch == '"' {
			break
		}

		if l.ch == '\\' {
			l.readChar()

			switch l.ch {
			case 'n':
				result.WriteRune('\n')
			case 't':
				result.WriteRune('\t')
			case 'r':
				result.WriteRune('\r')
			case '"':
				result.WriteRune('"')
			case '\\':
				result.WriteRune('\\')
			default:
				// 未知エスケープはそのまま入れる
				result.WriteRune(l.ch)
			}
			continue
		}

		result.WriteRune(l.ch)
	}

	return result.String()
}

func (l *Lexer) skipComment() {
	for l.ch != '\n' && l.ch != '\r' && l.ch != 0 {
		l.readChar()
	}
}
