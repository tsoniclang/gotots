package emit_test

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tsoniclang/gotots/internal/emit"
	"github.com/tsoniclang/gotots/internal/load"
)

func TestTypeSwitchCaseBindingIsMutableWhenGoAssignsIt(t *testing.T) {
	directory := t.TempDir()
	writeProgramFile(t, filepath.Join(directory, "go.mod"),
		"module example.com/type-switch-binding\n\ngo 1.26.4\n")
	writeProgramFile(t, filepath.Join(directory, "source.go"), `package binding

type Values []int32

func Rewrite(value any) int32 {
	switch current := value.(type) {
	case Values:
		current = Values{9}
		return current[0]
	default:
		return 0
	}
}
`)
	program, err := load.Load(context.Background(), load.Request{
		Directory: directory,
		Pattern:   ".",
	})
	if err != nil {
		t.Fatal(err)
	}
	root, err := emit.NewRoot(program.Roots()[0].Types().Scope().Lookup("Rewrite"))
	if err != nil {
		t.Fatal(err)
	}
	emission, err := emit.Compile(program, []emit.Root{root})
	if err != nil {
		t.Fatal(err)
	}
	workingDirectory := t.TempDir()
	artifacts := materializeArtifacts(t, emission, workingDirectory)
	if !strings.Contains(artifacts.printed, "let current") ||
		strings.Contains(artifacts.printed, "const current") {
		t.Fatalf("mutable type-switch binding:\n%s", artifacts.printed)
	}
	writeProgramFile(t, filepath.Join(workingDirectory, "package.json"),
		"{\"type\":\"module\"}\n")
	waveThreeTypecheck(t, workingDirectory, artifacts.paths)
	if _, err := os.Stat(filepath.Join(workingDirectory, "out")); err != nil {
		t.Fatal(err)
	}
}
