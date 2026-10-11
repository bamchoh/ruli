package main

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"ruli"
)

func writeScript(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "script.ruli")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestRunValidScript(t *testing.T) {
	buf := &bytes.Buffer{}
	s := ruli.New()
	s.SetOutput(buf)

	if err := s.RunFile(writeScript(t, "x := 1 + 2\nprint(x)\n")); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if got := buf.String(); got != "3\n" {
		t.Fatalf("wrong output. got=%q", got)
	}
}

func TestRunParseError(t *testing.T) {
	err := ruli.New().RunFile(writeScript(t, "print(\"a\", \"b\""))
	if err == nil {
		t.Fatal("expected error")
	}
	if got := err.Error(); got != "1:15: expected ',' or ')', got EOF" {
		t.Fatalf("wrong error. got=%q", got)
	}
}

func TestRunMultipleParseErrors(t *testing.T) {
	err := ruli.New().RunFile(writeScript(t, "a := @\nb := @"))
	if err == nil {
		t.Fatal("expected error")
	}
	count := 1
	for _, c := range err.Error() {
		if c == '\n' {
			count++
		}
	}
	if count != 2 {
		t.Fatalf("expected 2 error lines, got %d (%q)", count, err.Error())
	}
}

func TestRunRuntimeError(t *testing.T) {
	err := ruli.New().RunFile(writeScript(t, "x := 1 / 0"))
	if err == nil {
		t.Fatal("expected error")
	}
	if got := err.Error(); got != "ERROR: line 1, column 6: division by zero" {
		t.Fatalf("wrong error. got=%q", got)
	}
}

func TestRunMissingFile(t *testing.T) {
	err := ruli.New().RunFile(writeScript(t, "") + "-missing")
	if err == nil {
		t.Fatal("expected error")
	}
	if !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("expected not-exist error, got %v", err)
	}
}

func TestRunFixedWidthTypesAndCasts(t *testing.T) {
	s := ruli.New()
	script := "w : WORD = 5\nd : DINT = w\nr := REAL(d)"
	if err := s.RunString(script); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
