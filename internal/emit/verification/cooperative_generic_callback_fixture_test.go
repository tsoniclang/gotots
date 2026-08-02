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

func genericConcreteCallName(t *testing.T, source string, owner string) string {
	t.Helper()
	marker := owner + "$concrete_"
	start := strings.Index(source, marker)
	if start < 0 {
		t.Fatalf("%s contains no %s call:\n%s", owner, marker, source)
	}
	end := strings.IndexByte(source[start:], '(')
	if end < 0 {
		t.Fatalf("%s concrete call has no argument list:\n%s", owner, source)
	}
	return source[start : start+end]
}
