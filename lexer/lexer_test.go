package lexer

import "testing"

func TestEmptyFile(t *testing.T) {
	input := ""
	l := New(input)

	tok := l.NextToken()

	if tok.Type != EOF {
		t.Fatalf("expected EOF, got %s", tok.Type)
	}
}
