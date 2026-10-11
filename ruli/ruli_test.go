package ruli_test

import (
	"bytes"
	"strings"
	"testing"

	"ruli"
	"ruli/object"
)

func TestRunString(t *testing.T) {
	buf := &bytes.Buffer{}
	s := ruli.New()
	s.SetOutput(buf)

	if err := s.RunString("total := 0\nfor i := 0; i < 3; i++ {\n\ttotal += i\n}\nprint(total)"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if got := buf.String(); got != "3\n" {
		t.Fatalf("wrong output. got=%q", got)
	}
}

func TestEnvironmentPersists(t *testing.T) {
	s := ruli.New()

	if err := s.RunString("count := 0"); err != nil {
		t.Fatal(err)
	}
	if err := s.RunString("count = count + 41"); err != nil {
		t.Fatal(err)
	}
	if err := s.RunString("print(count)"); err != nil {
		t.Fatal(err)
	}

	// env をホストから直接読むこともできる
	v, ok := s.Env().Get("count")
	if !ok {
		t.Fatal("count not in env")
	}
	if got := v.Inspect(); got != "41" {
		t.Fatalf("wrong value. got=%s", got)
	}
}

func TestParseErrorReport(t *testing.T) {
	err := ruli.New().RunString("func f(a) BOOL { return true }")
	if err == nil {
		t.Fatal("expected error")
	}
	if got := err.Error(); got != "1:9: parameter 'a' requires a type" {
		t.Fatalf("wrong error. got=%q", got)
	}
}

func TestTypeErrorReport(t *testing.T) {
	err := ruli.New().RunString("x := 1 + \"a\"")
	if err == nil {
		t.Fatal("expected error")
	}
	if got := err.Error(); got != "1:6: type mismatch: INT + STRING" {
		t.Fatalf("wrong error. got=%q", got)
	}
}

func TestRuntimeErrorReport(t *testing.T) {
	err := ruli.New().RunString("x := 1 / 0")
	if err == nil {
		t.Fatal("expected error")
	}
	if got := err.Error(); got != "ERROR: line 1, column 6: division by zero" {
		t.Fatalf("wrong error. got=%q", got)
	}
}

func TestHostFunction(t *testing.T) {
	buf := &bytes.Buffer{}
	s := ruli.New()
	s.SetOutput(buf)

	// 引数と戻り値を持つホスト関数
	s.SetFunction("greet", "STRING", func(args ...ruli.Value) (ruli.Value, error) {
		return ruli.NewStringValue("hello " + args[0].Inspect()), nil
	})

	// 戻り値なし (VOID)
	s.SetFunction("notify", "", func(args ...ruli.Value) (ruli.Value, error) {
		return ruli.NewNullValue(), nil
	})

	if err := s.RunString("x := greet(42)\nprint(x)\nnotify(x)"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if got := buf.String(); got != "hello 42\n" {
		t.Fatalf("wrong output. got=%q", got)
	}
}

func TestHostFunctionReturnTypeChecked(t *testing.T) {
	s := ruli.New()
	s.SetFunction("doubled", "INT", func(args ...ruli.Value) (ruli.Value, error) {
		return ruli.NewIntValue(2 * args[0].(*object.Integer).Value), nil
	})

	if err := s.RunString("x := doubled(21)"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// 宣言された戻り値型は型検査に使われる (VOID ではなく INT)
	if err := s.RunString("doubled(1) + \"x\""); err == nil {
		t.Fatal("expected type error")
	}
}

func TestHostFunctionInvalidReturnType(t *testing.T) {
	s := ruli.New()
	if err := s.SetFunction("weird", "NOPE", func(args ...ruli.Value) (ruli.Value, error) {
		return ruli.NewNullValue(), nil
	}); err == nil {
		t.Fatal("expected error for invalid return type")
	} else if got := err.Error(); got != "unknown type: NOPE" {
		t.Fatalf("wrong error. got=%q", got)
	}
}

func TestHostFunctionValueConstructors(t *testing.T) {
	buf := &bytes.Buffer{}
	s := ruli.New()
	s.SetOutput(buf)

	s.SetFunction("info", "", func(args ...ruli.Value) (ruli.Value, error) {
		return ruli.NewArrayValue([]ruli.Value{
			ruli.NewIntValue(1),
			ruli.NewRealValue(2.5),
			ruli.NewBoolValue(true),
			ruli.NewStringValue("x"),
			ruli.NewStructValue(map[string]ruli.Value{
				"name": ruli.NewStringValue("m"),
			}),
			ruli.NewNullValue(),
		}), nil
	})

	if err := s.RunString("print(info())"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// 全エレメントが並んで出力される
	out := buf.String()
	for _, want := range []string{"1", "2.500000", "true", "x", "{name: m}"} {
		if !strings.Contains(out, want) {
			t.Fatalf("output missing %q. got=%q", want, out)
		}
	}
}

func TestHostFunctionArityIsUnchecked(t *testing.T) {
	// 引数の個数検証はホスト側の責務
	buf := &bytes.Buffer{}
	s := ruli.New()
	s.SetOutput(buf)

	s.SetFunction("add", "INT", func(args ...ruli.Value) (ruli.Value, error) {
		total := 0
		for _, a := range args {
			if i, ok := a.(*object.Integer); ok {
				total += i.Value
			}
		}
		return ruli.NewIntValue(total), nil
	})

	if err := s.RunString("print(add(1, 2, 3))"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := buf.String(); got != "6\n" {
		t.Fatalf("wrong output. got=%q", got)
	}
}

func TestEvalStringReturnsValue(t *testing.T) {
	s := ruli.New()

	v, err := s.EvalString("1 + 2")
	if err != nil {
		t.Fatal(err)
	}
	if got := v.Inspect(); got != "3" {
		t.Fatalf("wrong value. got=%s", got)
	}

	// 宣言は Null
	v, err = s.EvalString("x := 5")
	if err != nil {
		t.Fatal(err)
	}
	if _, isNull := v.(*object.Null); !isNull {
		t.Fatalf("declaration should return Null. got=%T", v)
	}

	// break は Null に畳まれる
	v, err = s.EvalString("break")
	if err != nil {
		t.Fatal(err)
	}
	if _, isNull := v.(*object.Null); !isNull {
		t.Fatalf("break should flatten to Null. got=%T", v)
	}

	// トップレベル return はその値
	src := "func f(a: INT) INT { return a * 2 }\nf(21)"
	v, err = s.EvalString(src)
	if err != nil {
		t.Fatal(err)
	}
	if got := v.Inspect(); got != "42" {
		t.Fatalf("wrong return value. got=%s", got)
	}
}
