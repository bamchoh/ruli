package lexer

import "testing"

func TestAssignStatement(t *testing.T) {
	input := `x := 10
	y := 10 + 20 - 30 * 40 / 50
	z : INT = 10
	a = 10
	if x > 5 {
		y := x * 2
	} else {
		y := x / 2
	}
	`

	l := New(input)

	tests := []struct {
		expectedType    TokenType
		expectedLiteral string
	}{
		{IDENT, "x"},
		{DECLARE, ":="},
		{INT_LIT, "10"},
		{IDENT, "y"},
		{DECLARE, ":="},
		{INT_LIT, "10"},
		{PLUS, "+"},
		{INT_LIT, "20"},
		{MINUS, "-"},
		{INT_LIT, "30"},
		{ASTERISK, "*"},
		{INT_LIT, "40"},
		{SLASH, "/"},
		{INT_LIT, "50"},
		{IDENT, "z"},
		{COLON, ":"},
		{IDENT, "INT"},
		{ASSIGN, "="},
		{INT_LIT, "10"},
		{IDENT, "a"},
		{ASSIGN, "="},
		{INT_LIT, "10"},
		{IF, "if"},
		{IDENT, "x"},
		{GT, ">"},
		{INT_LIT, "5"},
		{LBRACE, "{"},
		{IDENT, "y"},
		{DECLARE, ":="},
		{IDENT, "x"},
		{ASTERISK, "*"},
		{INT_LIT, "2"},
		{RBRACE, "}"},
		{ELSE, "else"},
		{LBRACE, "{"},
		{IDENT, "y"},
		{DECLARE, ":="},
		{IDENT, "x"},
		{SLASH, "/"},
		{INT_LIT, "2"},
		{RBRACE, "}"},
		{EOF, ""},
	}

	for i, tt := range tests {
		tok := l.NextToken()

		if tok.Type != tt.expectedType {
			t.Fatalf("tests[%d] type wrong. expected=%q got=%q",
				i, tt.expectedType, tok.Type)
		}

		if tok.Literal != tt.expectedLiteral {
			t.Fatalf("tests[%d] literal wrong. expected=%q got=%q",
				i, tt.expectedLiteral, tok.Literal)
		}
	}
}
