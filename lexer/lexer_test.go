package lexer

import "testing"

func TestAssignStatement(t *testing.T) {
	input := `x := 10
	y := 10 + 20 - 30 * 40 / 50
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
