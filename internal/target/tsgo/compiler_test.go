package tsgo

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestCompilerDiagnosticsFailClosedWhenToolExitsSuccessfully(t *testing.T) {
	workingDirectory := t.TempDir()
	sourcePath := filepath.Join(workingDirectory, "invalid.ts")
	if err := os.WriteFile(
		sourcePath,
		[]byte("const value: string = 1;\n"),
		0o644,
	); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	err := CompileWithTool(
		ctx,
		selectedTool(t),
		workingDirectory,
		[]string{"--strict", "--noEmit", sourcePath},
	)
	var compilerError *CompilerError
	if !errors.As(err, &compilerError) ||
		!strings.Contains(compilerError.Diagnostics, "TS2322") {
		t.Fatalf("compiler error = %#v, want fail-closed TS2322 diagnostics", err)
	}
}

func TestCompilerAcceptsSilentStrictSuccess(t *testing.T) {
	workingDirectory := t.TempDir()
	sourcePath := filepath.Join(workingDirectory, "valid.ts")
	if err := os.WriteFile(
		sourcePath,
		[]byte("const value: string = \"ok\";\n"),
		0o644,
	); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := CompileWithTool(
		ctx,
		selectedTool(t),
		workingDirectory,
		[]string{"--strict", "--noEmit", sourcePath},
	); err != nil {
		t.Fatal(err)
	}
}
