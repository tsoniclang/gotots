package emit_test

import (
	"context"
	"go/types"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tsoniclang/gotots/internal/emit"
	"github.com/tsoniclang/gotots/internal/load"
)

func waveNineFunctionWithPrefix(
	t *testing.T,
	printed string,
	prefix string,
) string {
	t.Helper()
	start := strings.Index(printed, "function "+prefix)
	if start < 0 {
		t.Fatalf("generated output lacks function prefix %s:\n%s", prefix, printed)
	}
	start = strings.LastIndex(printed[:start], "export ")
	if start < 0 {
		t.Fatalf("generated function prefix %s is not exported", prefix)
	}
	rest := printed[start:]
	next := strings.Index(rest[len("export "):], "\nexport ")
	if next < 0 {
		return rest
	}
	return rest[:len("export ")+next]
}

func waveNineClassMemberText(
	t *testing.T,
	printed string,
	className string,
	marker string,
) string {
	t.Helper()
	classStart := strings.Index(printed, "export class "+className)
	if classStart < 0 {
		t.Fatalf("generated output lacks class %s", className)
	}
	memberOffset := strings.Index(printed[classStart:], marker)
	if memberOffset < 0 {
		t.Fatalf("generated class %s lacks member marker %q", className, marker)
	}
	memberStart := classStart + memberOffset + 1
	memberEnd := strings.Index(printed[memberStart:], "\n    }\n")
	if memberEnd < 0 {
		t.Fatalf("generated class %s member %q has no boundary", className, marker)
	}
	return printed[memberStart : memberStart+memberEnd+len("\n    }")]
}

func genericCallbackFixture(
	t *testing.T,
) (string, string, emit.ProgramEmission, waveFourArtifacts) {
	directory := filepath.Join(
		repositoryRoot(),
		"testdata",
		"constructs",
		"concurrency",
		"generic-callback",
	)
	directory, err := filepath.Abs(directory)
	if err != nil {
		t.Fatal(err)
	}
	program, err := load.Load(context.Background(), load.Request{
		Directory: directory,
		Pattern:   ".",
	})
	if err != nil {
		t.Fatal(err)
	}
	roots, err := emit.ExportedAPIRoots(program.Roots()[0])
	if err != nil {
		t.Fatal(err)
	}
	selectedRoots := roots[:0]
	for _, root := range roots {
		function, ok := root.Object().(*types.Func)
		if ok && function.Signature().Recv() == nil &&
			function.Signature().TypeParams().Len() != 0 {
			continue
		}
		selectedRoots = append(selectedRoots, root)
	}
	roots = selectedRoots
	emission, err := emit.CompileWithOptions(
		program,
		roots,
		waveNineOptions(),
	)
	if err != nil {
		t.Fatal(err)
	}
	workingDirectory := t.TempDir()
	artifacts := materializeArtifacts(t, emission, workingDirectory)
	return directory, workingDirectory, emission, artifacts
}
