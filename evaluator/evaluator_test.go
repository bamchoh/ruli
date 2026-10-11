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
		`func f(x: [3]INT) INT {
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

func testEvalReal(input string, expected float64, t *testing.T) {
	evaluated := testEval(input)
	if isError(evaluated) {
		t.Fatalf("unexpected error: %s", evaluated.Inspect())
	}
	result, ok := evaluated.(*object.Real)
	if !ok {
		t.Fatalf("object is not Real. got=%T (%+v)", evaluated, evaluated)
	}
	if result.Value != expected {
		t.Fatalf("object has wrong value. got=%f, want=%f", result.Value, expected)
	}
}

func TestCastNumberToReal(t *testing.T) {
	testEvalReal(`REAL(1) + REAL(2)`, 3.0, t)
}

func TestCastTruncatesToInteger(t *testing.T) {
	// 実数 → 整数は 0 方向へ切り捨て
	testEvalInteger(`DINT(REAL(7) / REAL(2))`, 3, t)
}

func TestStructZeroInitAndFields(t *testing.T) {
	testEvalInteger(
		`type Motor struct {
	speed: INT
	enable: BOOL
}
m : Motor
m.speed = 5
m.speed`,
		5,
		t,
	)
}

func TestStructCopyOnBind(t *testing.T) {
	// SPEC 4.3: struct は値渡し
	testEvalInteger(
		`type Motor struct {
	speed: INT
}

m : Motor
m.speed = 5
n := m
n.speed = 99
m.speed`,
		5,
		t,
	)
}

func TestStructCopyOnFunctionArg(t *testing.T) {
	testEvalInteger(
		`type Motor struct {
	speed: INT
}

func set(x: Motor) INT {
	x.speed = 99
	return x.speed
}

m : Motor
m.speed = 5
set(m)
m.speed`,
		5,
		t,
	)
}

func TestTypedDeclZeroValues(t *testing.T) {
	// 型だけの宣言はゼロ値で初期化される
	testEvalInteger(`x : INT
x`, 0, t)
}

func testEvalBool(input string, expected bool, t *testing.T) {
	evaluated := testEval(input)
	if isError(evaluated) {
		t.Fatalf("unexpected error: %s", evaluated.Inspect())
	}
	result, ok := evaluated.(*object.Boolean)
	if !ok {
		t.Fatalf("object is not Boolean. got=%T (%+v)", evaluated, evaluated)
	}
	if result.Value != expected {
		t.Fatalf("object has wrong value. got=%t, want=%t", result.Value, expected)
	}
}

func TestPrefixExpression(t *testing.T) {
	testEvalInteger(`-5 + 3`, -2, t)
	testEvalReal(`-REAL(2)`, -2.0, t)
	testEvalBool(`!false`, true, t)
	testEvalBool(`flag := true
!flag`, false, t)
}

func TestRuntimeWidthCheckOnAssign(t *testing.T) {
	// 変数値の代入でも実行時幅チェックが効く
	testEvalError(
		`y := 200
x : SINT = 0
x = y`,
		"value 200 out of range for SINT (-128..127)",
		t,
	)
}

func TestRuntimeWidthCheckOnIncDec(t *testing.T) {
	testEvalError(
		`x : INT = 32767
x++`,
		"value 32768 out of range for INT (-32768..32767)",
		t,
	)
}

func TestRuntimeWidthCheckOnNegation(t *testing.T) {
	testEvalError(
		`x : SINT = -128
x--`,
		"value -129 out of range for SINT (-128..127)",
		t,
	)
}

func TestRuntimeWidthCheckOnCast(t *testing.T) {
	// 実数値経由のキャストは実行時に検査される
	testEvalError(
		`WORD(REAL(1) * REAL(70000))`,
		"value 70000 out of range for WORD (0..65535)",
		t,
	)
}

func TestRuntimeWidthCheckOnStructField(t *testing.T) {
	testEvalError(
		`type Motor struct {
	speed: SINT
}

m : Motor
y := 200
m.speed = y`,
		"value 200 out of range for SINT (-128..127)",
		t,
	)
}

func TestUnaryMinusLiteralWithinRange(t *testing.T) {
	// 負数リテラルの宣言・代入は可能
	testEvalInteger(
		`x : SINT = -128
x`,
		-128,
		t,
	)
}

func TestMultiReturnValues(t *testing.T) {
	testEvalInteger(
		`func div(a: INT, b: INT) (INT, BOOL) {
	return a / b, b != 0
}
x, ok := div(10, 2)
x`,
		5,
		t,
	)
	testEvalBool(
		`func div(a: INT, b: INT) (INT, BOOL) {
	return a / b, b != 0
}
x, ok := div(10, 2)
ok`,
		true,
		t,
	)
}

func TestMultiReturnInvalidBinding(t *testing.T) {
	// SPEC 5.4: 複数戻り値の単一束縛はランタイムエラー
	(testEvalError)(
		`func f() (INT, BOOL) {
	return 1, true
}
y := f()`,
		"cannot assign 2 values to 1 variables",
		t,
	)
}

func TestModOperator(t *testing.T) {
	testEvalInteger(`17 % 5`, 2, t)

	evaluated := testEval(`17 % 0`)
	if !isError(evaluated) {
		t.Fatal("expected error for 17 % 0")
	}
	if msg := evaluated.(*object.Error).Message; msg != "division by zero" {
		t.Fatalf("wrong error. got=%s", msg)
	}
}

func TestArrayZeroInit(t *testing.T) {
	// SPEC 4.4: 固定長配列はゼロ値で初期化される
	testEvalInteger(
		`values : [10]INT
len(values)`,
		10,
		t,
	)
	testEvalInteger(
		`values : [10]INT
values[9]`,
		0,
		t,
	)
}

func TestNestedArrayZeroInit(t *testing.T) {
	testEvalInteger(
		`matrix : [3][3]INT
len(matrix[2])`,
		3,
		t,
	)
	testEvalInteger(
		`matrix : [3][3]INT
matrix[2][2]`,
		0,
		t,
	)
}

func TestArrayWidthCheckingOnDeclElement(t *testing.T) {
	// 宣言リテラルの初回要素の幅チェックは実行時にも行われる
	testEvalError(
		`y := 5
x : SINT = y * 100`,
		"value 500 out of range for SINT (-128..127)",
		t,
	)
}

func TestSwitchStatement(t *testing.T) {
	testEvalInteger(
		`x := 3
switch x {
case 1:
	x = 10
case 2, 3:
	x = 20
default:
	x = 30
}
x`,
		20,
		t,
	)
}

func TestSwitchDefault(t *testing.T) {
	testEvalInteger(
		`x := 99
switch x {
case 1:
	x = 10
default:
	x = 30
}
x`,
		30,
		t,
	)
}

func TestSwitchNoMatchNoDefault(t *testing.T) {
	testEvalInteger(
		`x := 5
y := 1
switch x {
case 1:
	y = 10
}
y`,
		1,
		t,
	)
}

func TestSwitchBreakExitsSwitchOnly(t *testing.T) {
	// break は switch のみ抜け、外側の for は継続する
	testEvalInteger(
		`total := 0
for i := 0; i < 5; i++ {
	switch i {
	case 1:
		break
	default:
		total = total + 100
	}
	total = total + i
}
total`,
		410,
		t,
	)
}

func TestSwitchReturnPropagates(t *testing.T) {
	testEvalInteger(
		`func f(x: INT) INT {
	switch x {
	case 5:
		return 100
	}
	return 0
}
f(5)`,
		100,
		t,
	)
}

func TestSwitchCaseScope(t *testing.T) {
	// case 本体はスコープを持つ
	testEvalError(
		`x := 1
switch x {
case 1:
	y := 10
}
y`,
		"identifier not found: y",
		t,
	)
}

func TestCompoundAssignEvals(t *testing.T) {
	testEvalInteger(
		`x := 10
x += 5
x -= 3
x *= 4
x /= 3
x %= 5
x`,
		1,
		t,
	)
}

func TestCompoundAssignString(t *testing.T) {
	evaluated := testEval(`s := "ab"
s += "cd"
s`)
	if isError(evaluated) {
		t.Fatalf("unexpected error: %s", evaluated.Inspect())
	}
	if got := evaluated.(*object.String).Value; got != "abcd" {
		t.Fatalf("wrong value. got=%s, want=abcd", got)
	}
}

func TestCompoundAssignWidthCheck(t *testing.T) {
	testEvalError(
		`w : SINT = 100
w += 100`,
		"value 200 out of range for SINT (-128..127)",
		t,
	)
}

func TestCompoundAssignOnArrayElement(t *testing.T) {
	testEvalInteger(
		`buf := [1, 2, 3]
buf[1] *= 10
buf[1]`,
		20,
		t,
	)
}

func TestCompoundAssignOnStructField(t *testing.T) {
	testEvalInteger(
		`type Motor struct {
	total: INT
}

m : Motor
m.total += 7
m.total`,
		7,
		t,
	)
}

func TestShortCircuitAnd(t *testing.T) {
	// 左辺 false なら右辺の関数は呼ばれない
	testEvalInteger(
		`count := 0
func bump() BOOL {
	count++
	return true
}
flag := false
result := flag && bump()
count`,
		0,
		t,
	)
}

func TestShortCircuitOr(t *testing.T) {
	// 左辺 true なら右辺の関数は呼ばれない
	testEvalInteger(
		`count := 0
func bump() BOOL {
	count++
	return true
}
flag := true
result := flag || bump()
result2 := !flag || bump()
count`,
		1,
		t,
	)
}

func TestLogicalOperators(t *testing.T) {
	testEvalBool(`true && true`, true, t)
	testEvalBool(`true && false`, false, t)
	testEvalBool(`false || true`, true, t)
	testEvalBool(`false || false`, false, t)
	testEvalBool(`true || false && true`, true, t)
}

func TestLessEqualsGreaterEquals(t *testing.T) {
	testEvalBool(`5 <= 5`, true, t)
	testEvalBool(`5 <= 4`, false, t)
	testEvalBool(`5 >= 5`, true, t)
	testEvalBool(`4 >= 5`, false, t)
	testEvalBool(`REAL(1) <= REAL(2)`, true, t)
	testEvalBool(`REAL(2) >= REAL(3)`, false, t)
}

func TestRefStructPropagates(t *testing.T) {
	// ref 引数の struct 変更は呼び出し元に反映される
	testEvalInteger(
		`type Motor struct {
	speed: INT
}

func stop(m: ref Motor) BOOL {
	m.speed = 0
	return true
}

m : Motor
m.speed = 42
stop(m)
m.speed`,
		0,
		t,
	)
}

func TestRefArrayPropagates(t *testing.T) {
	testEvalInteger(
		`buf := [7, 8, 9]
func zero(a: ref [3]INT) BOOL {
	a[0] = 0
	return true
}
zero(buf)
buf[0]`,
		0,
		t,
	)
}

func TestRefScalarWriteBack(t *testing.T) {
	// スカラー ref は終了時に書き戻される (VAR_IN_OUT)
	testEvalInteger(
		`func addTen(n: ref INT) BOOL {
	n += 10
	return true
}
x := 5
addTen(x)
x`,
		15,
		t,
	)
	testEvalInteger(
		`func bump(n: ref INT) BOOL {
	n++
	return true
}
y := 5
bump(y)
y`,
		6,
		t,
	)
}

func TestValuePassStillCopies(t *testing.T) {
	// ref なしのパラメータは今までどおりコピー
	testEvalInteger(
		`type Motor struct {
	speed: INT
}

func broken(m: Motor) BOOL {
	m.speed = 99
	return true
}

m : Motor
broken(m)
m.speed`,
		0,
		t,
	)
}

func TestRealLiterals(t *testing.T) {
	testEvalReal(`2.5 + 2.0`, 4.5, t)
	testEvalReal(`-0.75`, -0.75, t)
	testEvalReal(`2.5 / 5.0`, 0.5, t)
	testEvalBool(`3.5 < 2.0`, false, t)
	testEvalBool(`1.5 <= 1.5`, true, t)
	testEvalInteger(`DINT(2.9)`, 2, t)
	// ランタイムの型名はオブジェクト型で出る (静的検査では INT)
	testEvalError(
		`x := 1.5 + 2`,
		"type mismatch: REAL + INTEGER",
		t,
	)
}

func TestStructLiteralRuntime(t *testing.T) {
	testEvalInteger(
		`type Motor struct {
	speed: INT
	enable: BOOL
}

m : Motor = Motor{speed: 10, enable: true}
m.speed`,
		10,
		t,
	)
	testEvalBool(
		`type Motor struct {
	speed: INT
	enable: BOOL
}

m2 := Motor{speed: 5}
m2.enable`,
		false,
		t,
	)
	// 空リテラルはゼロ値の struct
	testEvalInteger(
		`type Motor struct {
	speed: INT
	enable: BOOL
}

m3 := Motor{}
m3.speed`,
		0,
		t,
	)
	// リテラルでも値渡し (コピー)
	testEvalInteger(
		`type Motor struct {
	speed: INT
	enable: BOOL
}

m := Motor{speed: 10, enable: true}
n := m
n.speed = 99
m.speed`,
		10,
		t,
	)
}

func TestStructLiteralRuntimeErrors(t *testing.T) {
	testEvalError(
		`type Motor struct {
	speed: INT
	enable: BOOL
}

m := Motor{nope: 1}
m.speed`,
		"undefined field: nope",
		t,
	)
	testEvalError(
		`m := Nope{x: 1}`,
		"undefined struct type: Nope",
		t,
	)
}

func TestMethods(t *testing.T) {
	motor := `type Motor struct {
	speed: INT
}

func (m: Motor) isStopped() BOOL {
	return m.speed == 0
}

func (m: ref Motor) stop() BOOL {
	m.speed = 0
	return true
}

func (m: ref Motor) setSpeed(v: INT) BOOL {
	m.speed = v
	return v > 0
}

m : Motor
`

	testEvalInteger(
		motor+`m.setSpeed(5)
m.speed`,
		5,
		t,
	)
	testEvalBool(
		motor+`m.setSpeed(5)
m.isStopped()`,
		false,
		t,
	)
	testEvalInteger(
		motor+`m.setSpeed(5)
m.stop()
m.speed`,
		0,
		t,
	)
	testEvalBool(
		motor+`m.isStopped()`,
		true,
		t,
	)
	// 値渡しレシーバでは外に反映されない
	testEvalInteger(
		motor+`func (m: Motor) break_(v: INT) BOOL {
	m.speed = v
	return true
}

m.setSpeed(5)
m.break_(99)
m.speed`,
		5,
		t,
	)
	// ネストしたフィールドからの呼び出し
	testEvalInteger(
		`type Motor struct {
	speed: INT
}
type Drive struct {
	motor: Motor
}
func (m: ref Motor) stop() BOOL {
	m.speed = 0
	return true
}
func (m: ref Motor) setSpeed(v: INT) BOOL {
	m.speed = v
	return true
}

d : Drive
d.motor.setSpeed(7)
d.motor.stop()
d.motor.speed`,
		0,
		t,
	)
}
