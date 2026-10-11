package main

import (
	"strings"
	"testing"

	"ruli"
)

func newTestSession() *replSession {
	return newReplSession(ruli.New())
}

func TestReplEcho(t *testing.T) {
	s := newTestSession()

	_, out := s.processLine("1 + 2")
	if out != "= 3" {
		t.Fatalf("wrong echo. got=%q", out)
	}

	_, out = s.processLine("x := 5")
	if out != "" {
		t.Fatalf("declaration should not echo. got=%q", out)
	}

	_, out = s.processLine("x * 2")
	if out != "= 10" {
		t.Fatalf("wrong echo. got=%q", out)
	}
}

func TestReplErrors(t *testing.T) {
	s := newTestSession()

	// 未完の式はエラーではなく継続待ち
	quit, out := s.processLine("1 +")
	_ = quit
	if out != "" || s.pending == "" {
		t.Fatalf("incomplete input should wait. out=%q pending=%q", out, s.pending)
	}

	quit, out = s.processLine("2")
	_ = quit
	if out != "= 3" {
		t.Fatalf("completed echo. got=%q", out)
	}

	// 完成した構文のエラーは表示され、REPL は続行できる
	_, out = s.processLine("y := @")
	if !strings.Contains(out, "@") || quit {
		t.Fatalf("expected error echo. got=%q", out)
	}

	_, out = s.processLine("2 * 3")
	if out != "= 6" {
		t.Fatalf("wrong echo after error. got=%q", out)
	}
}

func TestReplMultiLine(t *testing.T) {
	s := newTestSession()

	// if ブロックの開き括弧 → 継続待ち
	quit, out := s.processLine("x := 2")
	_ = quit
	if out != "" {
		t.Fatalf("no echo expected. got=%q", out)
	}

	quit, out = s.processLine("if x == 2 {")
	if quit || out != "" {
		t.Fatalf("expected continuing. quit=%v out=%q", quit, out)
	}
	if s.pending == "" {
		t.Fatal("pending not accumulated")
	}

	quit, out = s.processLine("\tprint(\"inside\")")
	if quit || out != "" {
		t.Fatalf("still continuing. quit=%v out=%q", quit, out)
	}

	quit, out = s.processLine("}")
	if quit || out != "" {
		t.Fatalf("closed block should not echo. quit=%v out=%q", quit, out)
	}
	if s.pending != "" {
		t.Fatalf("pending should be reset. got=%q", s.pending)
	}

	// ブロック内で宣言した変数は外から見えない (スコープ)
	quit, out = s.processLine("x + 1")
	if out != "= 3" {
		t.Fatalf("wrong echo. got=%q", out)
	}
}

func TestReplDotCommands(t *testing.T) {
	s := newTestSession()

	_, out := s.processLine(".help")
	if !strings.Contains(out, ".exit") {
		t.Fatalf("help should mention .exit. got=%q", out)
	}

	quit, out := s.processLine(".exit")
	if !quit || out != "" {
		t.Fatalf(".exit should quit. quit=%v out=%q", quit, out)
	}

	quit, out = s.processLine(".quit")
	if !quit || out != "" {
		t.Fatalf(".quit should quit. quit=%v out=%q", quit, out)
	}
}
