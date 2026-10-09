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
	// 二項演算のエラー位置は左オペランドの始端 (1 行 1 列) に基づく
	if errObj.Line != 1 || errObj.Column != 1 {
		t.Fatalf(
			"wrong position. got=(%d, %d), want=(%d, %d)",
			errObj.Line,
			errObj.Column,
			1,
			2,
		)
	}
}

func TestBlockScopeShadowing(t *testing.T) {
	// SPEC 3.5: ブロック内の := は外側を汚さない
	testEvalInteger(
		`x := 10
if true {
	x := 20
}
x`,
		10,
		t,
	)
}

func TestAssignThroughOuterScope(t *testing.T) {
	testEvalInteger(
		`x := 10
if true {
	x = 20
}
x`,
		20,
		t,
	)
}

func TestAssignUndefinedVariable(t *testing.T) {
	testEvalError(
		`y := 1
x = 10`,
		"undefined variable: x",
		t,
	)
}

func TestForInitScopeDoesNotLeak(t *testing.T) {
	testEvalError(
		`for i := 0; i < 2; i++ {
}
i`,
		"identifier not found: i",
		t,
	)
}

func TestIncDecThroughBlockScope(t *testing.T) {
	testEvalInteger(
		`x := 10
for i := 0; i < 3; i++ {
	x++
}
x`,
		13,
		t,
	)
}

func TestFunctionReadsOuterScope(t *testing.T) {
	// SPEC 12.1: 関数から外側スコープの read/write は許可
	testEvalInteger(
		`x := 10
func inc() INT {
	x++
	return x
}
inc()
x`,
		11,
		t,
	)
}

func TestFunctionParamScope(t *testing.T) {
	testEvalInteger(
		`func twice(a: INT) INT {
	return a * 2
}
twice(21)`,
		42,
		t,
	)
}

func TestArrayCopyOnVarDecl(t *testing.T) {
	// SPEC 4.4: 束縛は配列を複製する
	testEvalInteger(
		`a := [1, 2, 3]
b := a
b[0] = 99
a[0]`,
		1,
		t,
	)
}

func TestArrayCopyOnAssign(t *testing.T) {
	testEvalInteger(
		`a := [1]
b := [5]
b = a
b[0] = 99
a[0]`,
		1,
		t,
	)
}

func TestArrayCopyOnFunctionArg(t *testing.T) {
	// SPEC 5.2: 引数は完全値渡し
	testEvalInteger(
		`func f(x) INT {
	x[0] = 99
	return x[0]
}
a := [1, 2, 3]
f(a)
a[0]`,
		1,
		t,
	)
}

func TestNestedArrayCopy(t *testing.T) {
	testEvalInteger(
		`m := [[1, 2], [3, 4]]
n := m
n[1][0] = 99
m[1][0]`,
		3,
		t,
	)
}

func TestArrayElementMutationIsShared(t *testing.T) {
	// 同じ配列オブジェクトを参照する要素/外側参照からの変更は共有が許可される
	// (SPEC 12.1: 外側スコープの書き込み)
	testEvalInteger(
		`a := [1, 2, 3]
func inc() INT {
	a[0] = 50
	return a[0]
}
inc()
a[0]`,
		50,
		t,
	)
}
