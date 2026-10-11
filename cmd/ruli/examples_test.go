package main

import (
	"io"
	"os"
	"path/filepath"
	"testing"

	"ruli"
)

// TestExamplesRun は examples/ の全スクリプトがエラーなく走ることを保証する
func TestExamplesRun(t *testing.T) {
	matches, err := filepath.Glob(filepath.Join("..", "..", "examples", "*.ruli"))
	if err != nil {
		t.Fatal(err)
	}
	if len(matches) == 0 {
		t.Fatal("no examples found")
	}

	for _, path := range matches {
		path := path
		t.Run(filepath.Base(path), func(t *testing.T) {
			s := ruli.New()
			s.SetOutput(io.Discard)

			if _, err := os.Stat(path); err != nil {
				t.Fatal(err)
			}

			if err := s.RunFile(path); err != nil {
				t.Fatalf("%s failed: %v", path, err)
			}
		})
	}
}
