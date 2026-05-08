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

	line   int
	column int
}

func New(input string) *Lexer {
	l := &Lexer{
		input:  []rune(input),
		line:   1,
		column: 0,
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
			tok = l.newToken(EQ, string(ch)+string(l.ch))
		} else {
			tok = l.newToken(ASSIGN, "=")
		}

	case '!':
		if l.peekChar() == '=' {
			ch := l.ch
			l.readChar()
			tok = l.newToken(NOT_EQ, string(ch)+string(l.ch))
		} else {
			tok = l.newToken(ILLEGAL, "!")
		}

	case '<':
		tok = l.newToken(LT, "<")

	case '>':
		tok = l.newToken(GT, ">")

	case '+':
		if l.peekChar() == '+' {
			ch := l.ch
			l.readChar()
			tok = l.newToken(INC, string(ch)+string(l.ch))
		} else {
			tok = l.newToken(PLUS, "+")
		}

	case '-':
		if l.peekChar() == '-' {
			ch := l.ch
			l.readChar()
			tok = l.newToken(DEC, string(ch)+string(l.ch))
		} else {
			tok = l.newToken(MINUS, "-")
		}

	case '*':
		tok = l.newToken(ASTERISK, "*")
	case '/':
		tok = l.newToken(SLASH, "/")

	case ':':
		if l.peekChar() == '=' {
			ch := l.ch
			l.readChar()
			tok = l.newToken(DECLARE, string(ch)+string(l.ch))
		} else {
			tok = l.newToken(COLON, string(l.ch))
		}

	case '[':
		tok = l.newToken(LBRACKET, "[")

	case ']':
		tok = l.newToken(RBRACKET, "]")

	case '{':
		tok = l.newToken(LBRACE, "{")

	case '}':
		tok = l.newToken(RBRACE, "}")

	case '(':
		tok = l.newToken(LPAREN, "(")

	case ')':
		tok = l.newToken(RPAREN, ")")

	case ',':
		tok = l.newToken(COMMA, ",")

	case ';':
		tok = l.newToken(SEMICOLON, ";")

	case '"':
		tok = l.newToken(STRING_LIT, l.readString())

	case 0:
		tok = l.newToken(EOF, "")
	default:
		if isLetter(l.ch) {
			lit := l.readIdentifier()
			tok.Type = LookupIdent(lit)
			tok = l.newToken(tok.Type, lit)
			return tok
		} else if isDigit(l.ch) {
			return l.newToken(INT_LIT, l.readNumber())
		} else {
			tok = l.newToken(ILLEGAL, string(l.ch))
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

	if l.ch == '\n' {
		l.line++
		l.column = 0
	} else {
		l.column++
	}
}

func (l *Lexer) newToken(t TokenType, lit string) Token {
	return Token{
		Type:    t,
		Literal: lit,
		Line:    l.line,
		Column:  l.column,
	}
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
