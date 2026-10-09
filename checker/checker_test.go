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
		[]string{"not a function: x"},
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
