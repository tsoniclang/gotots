package main

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// failingWriter fails every write, standing in for a broken output sink.
type failingWriter struct{}

func (failingWriter) Write([]byte) (int, error) {
	return 0, errors.New("output sink failed")
}

// TestRunPropagatesWriterFailure proves rendering fails closed: a failing
// writer surfaces its error from run instead of being discarded.
func TestRunPropagatesWriterFailure(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "sample.go")
	if err := os.WriteFile(path, []byte(sample), 0o644); err != nil {
		t.Fatalf("write sample: %v", err)
	}
	err := run([]string{"inspect", "constructs", path}, failingWriter{})
	if err == nil {
		t.Fatal("run succeeded with a failing writer")
	}
	if !strings.Contains(err.Error(), "output sink failed") {
		t.Fatalf("run error = %v, want the writer's failure", err)
	}
}

const sample = `package p

func F(x int) int { return x + 1 }
`

// TestRunInspectConstructs proves the supported path produces an inventory.
func TestRunInspectConstructs(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "sample.go")
	if err := os.WriteFile(path, []byte(sample), 0o644); err != nil {
		t.Fatalf("write sample: %v", err)
	}
	var out bytes.Buffer
	if err := run([]string{"inspect", "constructs", path}, &out); err != nil {
		t.Fatalf("run: %v", err)
	}
	if !strings.Contains(out.String(), "FuncDecl") {
		t.Errorf("inventory output missing FuncDecl:\n%s", out.String())
	}
}

// TestRunFailsClosed proves every unsupported invocation returns a typed
// UnsupportedCommandError rather than doing partial work.
func TestRunFailsClosed(t *testing.T) {
	cases := [][]string{
		{},
		{"generate"},
		{"inspect"},
		{"inspect", "constructs"},
		{"inspect", "types", "x.go"},
	}
	for _, args := range cases {
		var out bytes.Buffer
		err := run(args, &out)
		if err == nil {
			t.Errorf("run(%q) succeeded, want fail-closed error", args)
			continue
		}
		var unsupported *UnsupportedCommandError
		if !errors.As(err, &unsupported) {
			t.Errorf("run(%q) error = %T, want *UnsupportedCommandError", args, err)
		}
		if out.Len() != 0 {
			t.Errorf("run(%q) wrote output on failure: %q", args, out.String())
		}
	}
}
