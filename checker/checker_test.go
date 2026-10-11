package checker

import (
	"testing"

	"ruli/ast"
	"ruli/lexer"
	"ruli/parser"
)

func parseInput(input string) *ast.Program {
	l := lexer.New(input)
	p := parser.New(l)
	return p.ParseProgram()
}

func testCheckErrors(input string, expectedErrs []string, t *testing.T) {
	l := lexer.New(input)
	p := parser.New(l)
	program := p.ParseProgram()

	errs := Check(program)
	if len(errs) != len(expectedErrs) {
		t.Fatalf("wrong number of errors. got=%d (%v), want=%d", len(errs), errs, len(expectedErrs))
	}

	for i, want := range expectedErrs {
		if errs[i].Message != want {
			t.Fatalf("error %d message not as expected. got=%s, want=%s", i, errs[i].Message, want)
		}
	}
}

func TestCheckValidProgram(t *testing.T) {
	for _, input := range []string{
		`x := 10`,
		`x : INT = 10`,
		`flag := true
if flag {
	print(1)
} else {
	print(2)
}`,
		`sum := 0
for i := 0; i < 3; i++ {
	sum = sum + i
}`,
		`nums := [1, 2, 3]
len(nums)
nums[0] + 1`,
		`func add(a: INT, b: INT) INT {
	return a + b
}
add(1, 2)`,
		// SPEC 2.1: 宣言収集により順不同の呼び出しが可能
		`f(1)
func f(a: INT) INT {
	return a
}`,
		// 再帰
		`func fact(n: INT) INT {
	if n == 1 {
		return 1
	}
	return n * fact(n - 1)
}`,
	} {
		if errs := Check(parseInput(input)); len(errs) > 0 {
			t.Fatalf("unexpected errors (%q): %v", input, errs)
		}
	}
}

func TestCheckTypeMismatchBinary(t *testing.T) {
	testCheckErrors(
		`x := 1 + "a"`,
		[]string{"type mismatch: INT + STRING"},
		t,
	)
}

func TestCheckUndefinedVariable(t *testing.T) {
	testCheckErrors(
		`x := y`,
		[]string{"undefined variable: y"},
		t,
	)
}

func TestCheckAssignUndefined(t *testing.T) {
	testCheckErrors(
		`x = 1`,
		[]string{"undefined variable: x"},
		t,
	)
}

func TestCheckVarDeclMismatch(t *testing.T) {
	testCheckErrors(
		`x : INT = true`,
		[]string{"type mismatch: declared INT, got BOOL"},
		t,
	)
}

func TestCheckIfConditionNotBool(t *testing.T) {
	testCheckErrors(
		`x := 1
if x { }`,
		[]string{"condition must be BOOL, got INT"},
		t,
	)
}

func TestCheckFunctionArity(t *testing.T) {
	testCheckErrors(
		`func f(a: INT) INT {
	return a
}
f(1, 2)`,
		[]string{"wrong number of arguments. got=2, want=1"},
		t,
	)
}

func TestCheckFunctionArgType(t *testing.T) {
	testCheckErrors(
		`func f(a: INT) INT {
	return a
}
f(true)`,
		[]string{"type mismatch: argument 1: expected INT, got BOOL"},
		t,
	)
}

func TestCheckReturnMismatch(t *testing.T) {
	testCheckErrors(
		`func f() INT {
	return true
}`,
		[]string{"type mismatch: declared INT, got BOOL"},
		t,
	)
}

func TestCheckUndefinedFunction(t *testing.T) {
	testCheckErrors(
		`foo()`,
		[]string{"undefined function: foo"},
		t,
	)
}

func TestCheckCallNonFunction(t *testing.T) {
	testCheckErrors(
		`x := 1
x()`,
		[]string{"not a function: INT"},
		t,
	)
}

func TestCheckIncDec(t *testing.T) {
	testCheckErrors(
		`x++`,
		[]string{"undefined variable: x"},
		t,
	)
}

func TestCheckLenNonArray(t *testing.T) {
	testCheckErrors(
		`len(1)`,
		[]string{"argument to `len` not supported, got INT"},
		t,
	)
}

func TestCheckReturnOutsideFunction(t *testing.T) {
	testCheckErrors(
		`return 1`,
		[]string{"return outside function"},
		t,
	)
}

func TestCheckErrorHasPosition(t *testing.T) {
	errs := Check(parseInput("x := y"))
	if len(errs) != 1 {
		t.Fatalf("wrong number of errors. got=%d", len(errs))
	}
	if errs[0].Line != 1 || errs[0].Column != 6 {
		t.Fatalf("wrong position. got=(%d, %d), want=(%d, %d)", errs[0].Line, errs[0].Column, 1, 6)
	}
	if errs[0].Error() != "1:6: undefined variable: y" {
		t.Fatalf("wrong formatted message. got=%s", errs[0].Error())
	}
}

func TestCheckCastValid(t *testing.T) {
	for _, input := range []string{
		`x := REAL(1)`,
		`x := DINT(REAL(1) / REAL(2))`,
		`x : DINT = 5`,
		`x : WORD = 5`,
		`x := REAL(1) + REAL(2)`,
		`f : BOOL = BOOL(true)`,
	} {
		if errs := Check(parseInput(input)); len(errs) > 0 {
			t.Fatalf("unexpected errors (%q): %v", input, errs)
		}
	}
}

func TestCheckCastToRealFromBool(t *testing.T) {
	testCheckErrors(
		`x := REAL(true)`,
		[]string{"invalid cast: BOOL to REAL"},
		t,
	)
}

func TestCheckCastToBoolFromInt(t *testing.T) {
	testCheckErrors(
		`x := BOOL(1)`,
		[]string{"invalid cast: INT to BOOL"},
		t,
	)
}

func TestCheckCastRealIntMix(t *testing.T) {
	// SPEC 6.3: INT + REAL もキャスト後もエラー
	testCheckErrors(
		`x := REAL(1) + 5`,
		[]string{"type mismatch: REAL + INT"},
		t,
	)
}

func TestCheckStructValid(t *testing.T) {
	input := `type Motor struct {
	speed: INT
	enable: BOOL
}
type Drive struct {
	gear: WORD
	motor: Motor
}

m : Motor
m.speed = 10
m.enable = m.speed == 10

d : Drive
d.motor = m
d.motor.speed = 1

f(m.speed)
func f(a: INT) INT {
	return a
}`
	if errs := Check(parseInput(input)); len(errs) > 0 {
		t.Fatalf("unexpected errors: %v", errs)
	}
}

func TestCheckStructUndefinedType(t *testing.T) {
	testCheckErrors(
		`m : Nope`,
		[]string{"unknown type: Nope"},
		t,
	)
}

func TestCheckStructUndefinedField(t *testing.T) {
	testCheckErrors(
		`type Motor struct {
	speed: INT
}

m : Motor
m.nope = 1`,
		[]string{"undefined field: nope"},
		t,
	)
}

func TestCheckStructFieldTypeMismatch(t *testing.T) {
	testCheckErrors(
		`type Motor struct {
	speed: INT
}

m : Motor
m.speed = true`,
		[]string{"type mismatch: INT = BOOL"},
		t,
	)
}

func TestCheckStructCompare(t *testing.T) {
	testCheckErrors(
		`type Motor struct {
	speed: INT
}

a : Motor
b : Motor
a == b`,
		[]string{"cannot compare STRUCT"},
		t,
	)
}

func TestCheckStructForwardReference(t *testing.T) {
	// 宣言収集 (SPEC 2.1) により後続の struct も参照できる
	input := `type Drive struct {
	motor: Motor
}
type Motor struct {
	speed: INT
}

m : Motor
d : Drive
d.motor = m`
	if errs := Check(parseInput(input)); len(errs) > 0 {
		t.Fatalf("unexpected errors: %v", errs)
	}
}

func TestCheckLiteralOutOfRange(t *testing.T) {
	testCheckErrors(
		`x : SINT = 128`,
		[]string{"value 128 out of range for SINT (-128..127)"},
		t,
	)
}

func TestCheckLiteralOutOfRangeNegative(t *testing.T) {
	testCheckErrors(
		`x : INT = -32769`,
		[]string{"value -32769 out of range for INT (-32768..32767)"},
		t,
	)
}

func TestCheckCastLiteralOutOfRange(t *testing.T) {
	testCheckErrors(
		`x := WORD(70000)`,
		[]string{"value 70000 out of range for WORD (0..65535)"},
		t,
	)
}

func TestCheckLiteralInRange(t *testing.T) {
	for _, input := range []string{
		`x : SINT = -128`,
		`x : ULINT = 9223372036854775807`,
		`x := WORD(65535)`,
	} {
		if errs := Check(parseInput(input)); len(errs) > 0 {
			t.Fatalf("unexpected errors (%q): %v", input, errs)
		}
	}
}

func TestCheckPrefixExpression(t *testing.T) {
	testCheckErrors(
		`x := -true`,
		[]string{"operand of - must be numeric, got BOOL"},
		t,
	)
	testCheckErrors(
		`x := !1`,
		[]string{"operand of ! must be BOOL, got INT"},
		t,
	)
	valid := parseInput("x := -5\ny := !true\nz : SINT = -128")
	if errs := Check(valid); len(errs) > 0 {
		t.Fatalf("unexpected errors: %v", errs)
	}
}

func TestCheckMultiReturnValid(t *testing.T) {
	input := `func div(a: INT, b: INT) (INT, BOOL) {
	return a / b, b != 0
}

x, ok := div(10, 2)
x, ok = div(15, 3)
div(10, 2)
print(x)`
	if errs := Check(parseInput(input)); len(errs) > 0 {
		t.Fatalf("unexpected errors: %v", errs)
	}
}

func TestCheckMultiReturnSingleVar(t *testing.T) {
	// SPEC 5.4: 複数戻り値を単一変数へ代入は禁止
	testCheckErrors(
		`func div(a: INT, b: INT) (INT, BOOL) {
	return a / b, b != 0
}
y := div(10, 2)`,
		[]string{"cannot assign 2 return values to 1 variables"},
		t,
	)
}

func TestCheckReturnCount(t *testing.T) {
	testCheckErrors(
		`func div(a: INT, b: INT) (INT, BOOL) {
	return a
}`,
		[]string{"wrong number of return values. got=1, want=2"},
		t,
	)
}

func TestCheckArrayTypeValid(t *testing.T) {
	input := `values : [10]INT
matrix : [3][3]INT

values[0] = 5
values[9] = -1
matrix[2][1] = 7
n := len(values)
small := [1, 2, 3]
x : [3]INT = small

buf : [4]SINT
buf[0] = -128

func f(v: [4]INT) INT {
	return v[0]
}
x2 := f([1, 2, 3, 4])

func g() [4]INT {
	return [1, 2, 3, 4]
}
y := g()`
	if errs := Check(parseInput(input)); len(errs) > 0 {
		t.Fatalf("unexpected errors: %v", errs)
	}
}

func TestCheckArraySizeMismatch(t *testing.T) {
	// SPEC 4.4: サイズは型の一部
	testCheckErrors(
		`a := [1, 2, 3]
x : [10]INT = a`,
		[]string{"type mismatch: declared [10]INT, got [3]INT"},
		t,
	)
}

func TestCheckArrayElementWidth(t *testing.T) {
	testCheckErrors(
		`values : [10]SINT
values[0] = 300`,
		[]string{"value 300 out of range for SINT (-128..127)"},
		t,
	)
}

func TestCheckArrayStaticBounds(t *testing.T) {
	// SPEC 4.7 を静的に強化: 既知サイズへのリテラルインデックス
	testCheckErrors(
		`values := [1, 2, 3]
values[3]`,
		[]string{"index out of range: 3 (array size 3)"},
		t,
	)
	testCheckErrors(
		`values := [1, 2, 3]
x := values[5]`,
		[]string{"index out of range: 5 (array size 3)"},
		t,
	)
}

func TestCheckSwitchValid(t *testing.T) {
	input := `x := 2
s := ""
switch x {
case 1:
	s = "one"
case 2, 3:
	s = "two"
default:
	s = "other"
}
switch s {
case "one":
	print(1)
}`
	if errs := Check(parseInput(input)); len(errs) > 0 {
		t.Fatalf("unexpected errors: %v", errs)
	}
}

func TestCheckSwitchCaseTypeMismatch(t *testing.T) {
	testCheckErrors(
		`x := 1
switch x {
case true:
	print(1)
}`,
		[]string{"case value type mismatch: expected INT, got BOOL"},
		t,
	)
}

func TestCheckSwitchStructValue(t *testing.T) {
	testCheckErrors(
		`type M struct {
	s: INT
}

m : M
switch m {
case 1:
}`,
		[]string{"switch value must be INT, REAL, STRING or BOOL, got STRUCT"},
		t,
	)
}

func TestCheckCompoundAssignValid(t *testing.T) {
	input := `x := 1
x += 2
x -= 3
x *= 4
x /= 2
x %= 7

buf := [1, 2, 3]
buf[0] += 5

s := "a"
s += "b"

w : WORD = 5
w += 10`
	if errs := Check(parseInput(input)); len(errs) > 0 {
		t.Fatalf("unexpected errors: %v", errs)
	}
}

func TestCheckCompoundAssignUndefined(t *testing.T) {
	testCheckErrors(
		`x += 1`,
		[]string{"undefined variable: x"},
		t,
	)
}

func TestCheckCompoundAssignMismatch(t *testing.T) {
	testCheckErrors(
		`x := 1
x += "a"`,
		[]string{"type mismatch: INT + STRING"},
		t,
	)
	testCheckErrors(
		`flag := true
flag += 1`,
		[]string{"type mismatch: BOOL + INT"},
		t,
	)
}

func TestCheckCompoundAssignStaticBounds(t *testing.T) {
	// 変数宣言と同じ静的検査 (未定義インデックスは実行時へ)
	testCheckErrors(
		`buf := [1, 2, 3]
buf[3] += 1`,
		[]string{"index out of range: 3 (array size 3)"},
		t,
	)
}

func TestCheckLogicalValid(t *testing.T) {
	input := `flag := true
x := 1
ok := flag && x == 1 || !flag`
	if errs := Check(parseInput(input)); len(errs) > 0 {
		t.Fatalf("unexpected errors: %v", errs)
	}
}

func TestCheckLogicalNonBool(t *testing.T) {
	testCheckErrors(
		`flag := 1 && true`,
		[]string{"operand of && must be BOOL, got INT (left)"},
		t,
	)
	testCheckErrors(
		`flag := true && 1`,
		[]string{"operand of && must be BOOL, got INT (right)"},
		t,
	)
}

func TestCheckComparisonOperators(t *testing.T) {
	for _, input := range []string{
		`x := 1 <= 2`,
		`x := 2 >= 1`,
		`x := REAL(1) <= REAL(2)`,
	} {
		if errs := Check(parseInput(input)); len(errs) > 0 {
			t.Fatalf("unexpected errors (%q): %v", input, errs)
		}
	}
}

func TestCheckDistinctTypeValid(t *testing.T) {
	input := `type TEMP INT
type RATIO REAL

t : TEMP
t = TEMP(30)
v := INT(t) + 10

r : RATIO
r = RATIO(REAL(1))

func f(t: TEMP) BOOL {
	return INT(t) > 20
}
f(t)`
	if errs := Check(parseInput(input)); len(errs) > 0 {
		t.Fatalf("unexpected errors: %v", errs)
	}
}

func TestCheckDistinctTypeImplicitBase(t *testing.T) {
	// SPEC 4.2: 基本型との暗黙変換なし
	testCheckErrors(
		`type TEMP INT
t : TEMP = 30`,
		[]string{"type mismatch: declared TEMP, got INT"},
		t,
	)
}

func TestCheckDistinctTypeArithmetic(t *testing.T) {
	testCheckErrors(
		`type TEMP INT
t : TEMP
t + 1`,
		[]string{"type mismatch: TEMP + INT"},
		t,
	)
}

func TestCheckDistinctTypeCastBack(t *testing.T) {
	// 基本型への明示キャストは可能
	input := "type TEMP INT\nt : TEMP = TEMP(5)\nv := INT(t) + 1"
	if errs := Check(parseInput(input)); len(errs) > 0 {
		t.Fatalf("unexpected errors: %v", errs)
	}
}

func TestCheckCastToStruct(t *testing.T) {
	testCheckErrors(
		`type M struct {
	s: INT
}

m : M
x := M(1)`,
		[]string{"invalid cast: INT to M"},
		t,
	)
}

func TestCheckRefValid(t *testing.T) {
	input := `type Motor struct {
	speed: INT
}

m : Motor

func stop(m: ref Motor) BOOL {
	m.speed = 0
	return true
}
stop(m)


nums := [1, 2, 3]
func zero(a: ref [3]INT) BOOL {
	a[0] = 0
	return true
}
zero(nums)

func addTen(n: ref INT) BOOL {
	n += 10
	return true
}
x := 1
addTen(x)`
	if errs := Check(parseInput(input)); len(errs) > 0 {
		t.Fatalf("unexpected errors: %v", errs)
	}
}

func TestCheckRefArgMustBeVariable(t *testing.T) {
	// ref 引数は代入可能な変数のみ (IEC の IN_OUT)
	testCheckErrors(
		`func f(n: ref INT) BOOL {
	return true
}
f(1)`,
		[]string{"ref argument must be a variable: 1"},
		t,
	)
	testCheckErrors(
		`func f(n: ref INT) BOOL {
	return true
}
f(1 + 2)`,
		[]string{"ref argument must be a variable: (1 + 2)"},
		t,
	)
}

func TestCheckRefArgType(t *testing.T) {
	testCheckErrors(
		`type M struct {
	s: INT
}

func f(m: ref M) BOOL {
	return true
}

y := 1
f(y)`,
		[]string{"type mismatch: argument 1: expected STRUCT, got INT"},
		t,
	)
}

func TestCheckRealLiteral(t *testing.T) {
	for _, input := range []string{
		`x := 3.5`,
		`x : REAL = 1.5`,
		`x : LREAL = 1.5`,
		`x := 1.5 + REAL(2)`,
		`x := 3.5 < 2.0`,
	} {
		if errs := Check(parseInput(input)); len(errs) > 0 {
			t.Fatalf("unexpected errors (%q): %v", input, errs)
		}
	}
}

func TestCheckRealLiteralMixed(t *testing.T) {
	// SPEC 6.3: 実数リテラルと整数の混在もキャスト必須
	testCheckErrors(
		`x := 1.5 + 2`,
		[]string{"type mismatch: REAL + INT"},
		t,
	)
	testCheckErrors(
		`x : INT = 1.5`,
		[]string{"type mismatch: declared INT, got REAL"},
		t,
	)
}

func TestCheckStructLiteralValid(t *testing.T) {
	input := `type Motor struct {
	speed: INT
	enable: BOOL
}
type Conf struct {
	gain: REAL
}

m : Motor = Motor{speed: 10, enable: true}
m2 := Motor{speed: 5}
m3 := Motor{}
c := Conf{gain: 0.75}`
	if errs := Check(parseInput(input)); len(errs) > 0 {
		t.Fatalf("unexpected errors: %v", errs)
	}
}

func TestCheckStructLiteralErrors(t *testing.T) {
	base := `type Motor struct {
	speed: SINT
}

`
	testCheckErrors(
		base+`m := Motor{nope: 1}`,
		[]string{"undefined field: nope"},
		t,
	)
	testCheckErrors(
		base+`m := Motor{speed: 300}`,
		[]string{"value 300 out of range for SINT (-128..127)"},
		t,
	)
	testCheckErrors(
		base+`m := Motor{speed: true}`,
		[]string{"type mismatch: SINT : BOOL"},
		t,
	)
	testCheckErrors(
		base+`m := Motor{speed: 1, speed: 2}`,
		[]string{"duplicate field: speed"},
		t,
	)
	testCheckErrors(
		`m := Nope{x: 1}`,
		[]string{"unknown type: Nope"},
		t,
	)
}

func TestCheckMethodValid(t *testing.T) {
	input := `type Motor struct {
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
m.setSpeed(5)
m.stop()
q := m.isStopped()`
	if errs := Check(parseInput(input)); len(errs) > 0 {
		t.Fatalf("unexpected errors: %v", errs)
	}
}

func TestCheckMethodErrs(t *testing.T) {
	base := `type Motor struct {
	speed: INT
}

`
	testCheckErrors(
		base+`func (m: Motor) speed() BOOL { return true }`,
		[]string{"method conflicts with field: Motor.speed"},
		t,
	)
	testCheckErrors(
		base+`func (m: Motor) stop() BOOL { return true }
func (m: Motor) stop() BOOL { return false }`,
		[]string{"duplicate method: Motor.stop"},
		t,
	)
	testCheckErrors(
		`func (n: INT) reset(v: INT) INT { return v }`,
		[]string{"method receiver must be a struct, got INT"},
		t,
	)
	testCheckErrors(
		base+`m : Motor
m.restart()`,
		[]string{"undefined method: Motor.restart"},
		t,
	)
	testCheckErrors(
		base+`m : Motor
m.speed()`,
		[]string{"field is not a function: speed"},
		t,
	)
	testCheckErrors(
		base+`x := 1
x.foo()`,
		[]string{"no method 'foo' on INT"},
		t,
	)
	testCheckErrors(
		base+`func (m: Motor) set(v: INT) BOOL { return true }

m : Motor
m.set(true)`,
		[]string{"type mismatch: argument 1: expected INT, got BOOL"},
		t,
	)
	testCheckErrors(
		base+`func (m: Motor) set(v: INT) BOOL { return true }

m : Motor
m.set(1, 2)`,
		[]string{"wrong number of arguments. got=2, want=1"},
		t,
	)
}

func TestCheckStringExtensions(t *testing.T) {
	for _, input := range []string{
		`ok := "a" < "b"`,
		`ok := "a" <= "a"`,
		`ok := "b" >= "a"`,
		`x := 1e5`,
		`x : REAL = 2.5e-3`,
		`s := "hello"
c := s[0]`,
	} {
		if errs := Check(parseInput(input)); len(errs) > 0 {
			t.Fatalf("unexpected errors (%q): %v", input, errs)
		}
	}
}

func TestCheckStringIndexAssign(t *testing.T) {
	// 文字列へのインデックス代入は不可 (不変値)
	testCheckErrors(
		`s := "abc"
s[0] = "x"`,
		[]string{"not an array"},
		t,
	)
}

func TestCheckHigherOrderFunctions(t *testing.T) {
	input := `func apply(fn: FUNC(x: INT) INT, v: INT) INT {
	return fn(v)
}

func double(x: INT) INT {
	return x * 2
}

g := double
apply(g, 10)

func makeFn() FUNC(x: INT) INT {
	return double
}
h := makeFn()
h(2)

func invoke(fn: FUNC(f: FUNC() INT) (INT, BOOL)) { return 0 }`
	if errs := Check(parseInput(input)); len(errs) > 0 {
		t.Fatalf("unexpected errors: %v", errs)
	}
}

func TestCheckHigherOrderMismatch(t *testing.T) {
	// シグネチャ不一致は静的エラー (メッセージは署名表示)
	base := `func apply(fn: FUNC(x: INT) INT, v: INT) INT {
	return fn(v)
}

func wrong(x: BOOL) INT {
	return 0
}

func voidFn() {
	print(1)
}

`
	testCheckErrors(
		base+`apply(wrong, 1)`,
		[]string{"type mismatch: argument 1: expected FUNC(INT) INT, got FUNC(BOOL) INT"},
		t,
	)
	testCheckErrors(
		base+`apply(voidFn, 1)`,
		[]string{"type mismatch: argument 1: expected FUNC(INT) INT, got FUNC()"},
		t,
	)
	testCheckErrors(
		base+`x := 1
x()`,
		[]string{"not a function: INT"},
		t,
	)
}
