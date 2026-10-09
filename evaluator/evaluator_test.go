package evaluator

import (
	"testing"

	"ruli/lexer"
	"ruli/object"
	"ruli/parser"
)

func testEval(input string) object.Object {
	l := lexer.New(input)
	p := parser.New(l)
	program := p.ParseProgram()
	env := object.NewEnvironment()
	return Eval(program, env)
}

func testEvalInteger(input string, expected int, t *testing.T) {
	evaluated := testEval(input)
	if isError(evaluated) {
		t.Fatalf("unexpected error: %s", evaluated.Inspect())
	}
	result, ok := evaluated.(*object.Integer)
	if !ok {
		t.Fatalf("object is not Integer. got=%T (%+v)", evaluated, evaluated)
	}
	if result.Value != expected {
		t.Fatalf("object has wrong value. got=%d, want=%d", result.Value, expected)
	}
}

func testEvalError(input string, wantMessage string, t *testing.T) {
	evaluated := testEval(input)
	if !isError(evaluated) {
		t.Fatalf(
			"expected error, got %T (%+v)",
			evaluated,
			evaluated,
		)
	}
	if msg := evaluated.(*object.Error).Message; msg != wantMessage {
		t.Fatalf("wrong error message. got=%s, want=%s", msg, wantMessage)
	}
}

func TestIntegerArithmetic(t *testing.T) {
	tests := []struct {
		input    string
		expected int
	}{
		{"5 + 3", 8},
		{"5 - 3", 2},
		{"5 * 3", 15},
		{"6 / 3", 2},
	}
	for _, tt := range tests {
		testEvalInteger(tt.input, tt.expected, t)
	}
}

func TestDivisionByZero(t *testing.T) {
	evaluated := testEval("1 / 0")
	if !isError(evaluated) {
		t.Fatalf("expected error, got %T (%+v)", evaluated, evaluated)
	}
	if msg := evaluated.(*object.Error).Message; msg != "division by zero" {
		t.Fatalf("wrong error message. got=%s", msg)
	}
}

func TestBinaryExpressionTypeMismatch(t *testing.T) {
	testEvalError(
		`1 + "a"`,
		"type mismatch: INTEGER + STRING",
		t,
	)
}

func TestUnsupportedOperator(t *testing.T) {
	testEvalError(
		`"a" < "b"`,
		"unsupported operator: <",
		t,
	)
}

func TestIncDecUndefinedVariable(t *testing.T) {
	testEvalError(
		`x++`,
		"identifier not found: x",
		t,
	)
}

func TestIncDecNonInteger(t *testing.T) {
	testEvalError(
		`x := true
x++`,
		"operand of ++ must be an integer, got BOOL",
		t,
	)
}

func TestWrongNumberOfArguments(t *testing.T) {
	testEvalError(
		`func add(a: INT, b: INT) INT {
	return a + b
}
add(1)`,
		"wrong number of arguments. got=1, want=2",
		t,
	)
}

func TestErrorHasPosition(t *testing.T) {
	evaluated := testEval("1 / 0")
	errObj, ok := evaluated.(*object.Error)
	if !ok {
		t.Fatalf("object is not Error. got=%T", evaluated)
	}
	// 二項演算のエラー位置は演算子トークン(+ = 1行目2列目)に基づく
	if errObj.Line != 1 || errObj.Column != 2 {
		t.Fatalf(
			"wrong position. got=(%d, %d), want=(%d, %d)",
			errObj.Line,
			errObj.Column,
			1,
			2,
		)
	}
}