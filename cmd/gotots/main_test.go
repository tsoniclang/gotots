package main

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeSampleModule(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	for rel, content := range map[string]string{
		"go.mod":  "module cli.example/sample\n\ngo 1.26\n",
		"main.go": "package main\n\nfunc main() { _ = add(1, 2) }\n\nfunc add(a, b int) int { return a + b }\n",
	} {
		if err := os.WriteFile(filepath.Join(dir, rel), []byte(content), 0o644); err != nil {
			t.Fatalf("write: %v", err)
		}
	}
	return dir
}

// TestRunInspectConstructs proves the supported path renders occurrence
// records and exact denominators.
func TestRunInspectConstructs(t *testing.T) {
	dir := writeSampleModule(t)
	var out bytes.Buffer
	if err := run([]string{"inspect", "constructs", "-dir", dir}, &out); err != nil {
		t.Fatalf("run: %v", err)
	}
	rendered := out.String()
	for _, needle := range []string{
		"package cli.example/sample::cli.example/sample",
		"cli.example/sample::main.go#",
		"kind=FuncDecl",
		"edge=File.Decls role=declaration",
		"denominators: goVersion=",
		"unknownConstructs=0",
	} {
		if !strings.Contains(rendered, needle) {
			t.Errorf("output missing %q:\n%s", needle, rendered)
		}
	}
}

// TestRunFailsClosed proves every unsupported invocation returns a typed
// UnsupportedCommandError and writes nothing.
func TestRunFailsClosed(t *testing.T) {
	cases := [][]string{
		{},
		{"generate"},
		{"inspect"},
		{"inspect", "types", "x"},
		{"inspect", "constructs", "-bogus"},
		{"inspect", "constructs", "-dir"},
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

// failingWriter fails every write, standing in for a broken output sink.
type failingWriter struct{}

func (failingWriter) Write([]byte) (int, error) {
	return 0, errors.New("output sink failed")
}

// TestRunPropagatesWriterFailure proves rendering fails closed: a failing
// writer surfaces its error from run instead of being discarded.
func TestRunPropagatesWriterFailure(t *testing.T) {
	dir := writeSampleModule(t)
	err := run([]string{"inspect", "constructs", "-dir", dir}, failingWriter{})
	if err == nil {
		t.Fatal("run succeeded with a failing writer")
	}
	if !strings.Contains(err.Error(), "output sink failed") {
		t.Fatalf("run error = %v, want the writer's failure", err)
	}
}
