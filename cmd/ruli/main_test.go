package main

import (
	"os"
	"path/filepath"
	"testing"
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
	if err := run(writeScript(t, "x := 1 + 2\nprint(x)\n")); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRunParseError(t *testing.T) {
	err := run(writeScript(t, "print(\"a\", \"b\""))
	if err == nil {
		t.Fatal("expected error")
	}
	if got := err.Error(); got != "1:15: expected ')', got 'EOF'" {
		t.Fatalf("wrong error. got=%q", got)
	}
}

func TestRunMultipleParseErrors(t *testing.T) {
	err := run(writeScript(t, "a := @\nb := @"))
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
		t.Fatalf("expected 2 error lines, got %d (%q)", count, err)
	}
}

func TestRunRuntimeError(t *testing.T) {
	err := run(writeScript(t, "x := 1 / 0"))
	if err == nil {
		t.Fatal("expected error")
	}
	if got := err.Error(); got != "ERROR: line 1, column 6: division by zero" {
		t.Fatalf("wrong error. got=%q", got)
	}
}

func TestRunMissingFile(t *testing.T) {
	if err := run(writeScript(t, "") + "-missing"); err == nil {
		t.Fatal("expected error")
	}
}
