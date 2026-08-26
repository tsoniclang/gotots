package emit_test

import (
	"context"
	"go/types"
	"path/filepath"
	"testing"

	"github.com/tsoniclang/gotots/internal/emit"
	"github.com/tsoniclang/gotots/internal/load"
)

func serialGenericCallbackFixture(
	t *testing.T,
) (string, string, emit.ProgramEmission, waveFourArtifacts) {
	t.Helper()
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
	emission, err := emit.CompileWithOptions(
		program,
		selectedRoots,
		waveNineOptions(),
	)
	if err != nil {
		t.Fatal(err)
	}
	workingDirectory := t.TempDir()
	artifacts := materializeArtifacts(t, emission, workingDirectory)
	return directory, workingDirectory, emission, artifacts
}
