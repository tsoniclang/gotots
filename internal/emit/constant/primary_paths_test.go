package constant_test

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/tsoniclang/gotots/internal/emit"
	"github.com/tsoniclang/gotots/internal/load"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
	runtimefixture "github.com/tsoniclang/gotots/internal/testfixture/gototsruntime"
)

func loadConstantRoots(t *testing.T) *load.Package {
	t.Helper()
	dir := filepath.Join(repositoryRoot(), "testdata", "constructs", "value", "constant-roots")
	loaded, err := load.One(context.Background(), load.Request{Directory: dir, Pattern: "."})
	if err != nil {
		t.Fatal(err)
	}
	return loaded
}

// TestCompileFileEmitsUntypedConstantRootsWithoutCrashing proves the primary
// whole-file entrypoint no longer fails on untyped-constant roots: CompileFile
// schedules every package declaration — including untyped constants — as a
// typed coverage root. A coverage-only constant contributes no runtime binding;
// projections arrive only from represented uses. The whole program
// strict-typechecks.
func TestCompileFileEmitsUntypedConstantRootsWithoutCrashing(t *testing.T) {
	loaded := loadConstantRoots(t)
	sourceFile := loaded.Files()[0].Syntax()

	emission, err := emit.CompileFile(loaded, sourceFile)
	if err != nil {
		t.Fatalf("CompileFile over untyped-constant roots failed: %v", err)
	}
	typecheckProgram(t, emission)
}

// TestExportedAPIRootsEmitsUntypedConstantRootsWithoutCrashing proves the same
// for the exported-Go-API entrypoint: unused exported untyped constants retain
// their compile-time-only disposition, while represented uses still demand
// projections. The program strict-typechecks.
func TestExportedAPIRootsEmitsUntypedConstantRootsWithoutCrashing(t *testing.T) {
	loaded := loadConstantRoots(t)
	roots, err := emit.ExportedAPIRoots(loaded)
	if err != nil {
		t.Fatalf("ExportedAPIRoots failed: %v", err)
	}
	emission, err := emit.Compile(loaded.Program(), roots)
	if err != nil {
		t.Fatalf("Compile over exported untyped-constant roots failed: %v", err)
	}
	typecheckProgram(t, emission)
}

// typecheckProgram materializes every emitted file and strict-typechecks the
// whole program with --noEmit, proving the generated TypeScript is internally
// consistent without executing it.
func typecheckProgram(t *testing.T, emission emit.ProgramEmission) {
	t.Helper()
	workingDirectory := t.TempDir()
	client, err := tsgo.StartClient(repositoryRoot(), workingDirectory)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := client.Close(); err != nil {
			t.Errorf("close TS-Go client: %v", err)
		}
	})
	var targetPaths []string
	for _, file := range emission.Files() {
		printed, err := client.PrintNode(file.SourceFile(), tsgo.PrintOptions{})
		if err != nil {
			t.Fatal(err)
		}
		targetPath := filepath.Join(workingDirectory, filepath.FromSlash(file.OutputPath()))
		writeFile(t, targetPath, printed)
		targetPaths = append(targetPaths, targetPath)
	}
	writeFile(t, filepath.Join(workingDirectory, "package.json"), "{\"type\":\"module\"}\n")
	arguments := []string{
		"--noEmit",
		"--target", "es2022",
		"--module", "nodenext",
		"--moduleResolution", "nodenext",
		"--strict",
	}
	arguments = append(arguments, targetPaths...)
	if err := runtimefixture.InstallResolution(workingDirectory, filepath.Join(workingDirectory, "out")); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := tsgo.Compile(ctx, repositoryRoot(), workingDirectory, arguments); err != nil {
		t.Fatalf("generated program failed strict typecheck: %v", err)
	}
}
