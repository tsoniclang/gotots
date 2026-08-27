package emit_test

import (
	"context"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tsoniclang/gotots/internal/emit"
	"github.com/tsoniclang/gotots/internal/load"
)

func TestPromiseTypeIntrinsicReservesOnlyItsTargetTypeName(t *testing.T) {
	project := t.TempDir()
	writeProgramFile(
		t,
		filepath.Join(project, "go.mod"),
		"module example.com/promiseshadow\n\ngo 1.26.4\n",
	)
	writeProgramFile(t, filepath.Join(project, "source.go"), `package promiseshadow

type Promise[T any] struct {
	Value T
}

type Object struct{}

type Reader interface {
	Read() int32
}

func Wrap(value int32) Promise[int32] {
	return Promise[int32]{Value: value}
}

func Wait(values <-chan int32) int32 {
	return <-values
}
`)
	program, err := load.Load(context.Background(), load.Request{
		Directory: project,
		Pattern:   ".",
	})
	if err != nil {
		t.Fatal(err)
	}
	roots, err := emit.ExportedAPIRoots(program.Roots()[0])
	if err != nil {
		t.Fatal(err)
	}
	options := emit.DefaultOptions()
	emission, err := emit.CompileWithOptions(program, roots, options)
	if err != nil {
		t.Fatal(err)
	}
	workingDirectory := t.TempDir()
	artifacts := materializeArtifacts(t, emission, workingDirectory)
	for _, required := range []string{
		"export class Promise__shadow_1<T>",
		"export class Object__shadow_1",
		"export function Wrap(value: int32): Promise__shadow_1<int32>",
		"export function Wait(values: GoReceiveChannel<int32> | undefined): int32",
	} {
		if !strings.Contains(artifacts.printed, required) {
			t.Fatalf("Promise target-type boundary lacks %q:\n%s", required, artifacts.printed)
		}
	}
	if strings.Contains(artifacts.printed, "globalThis.Promise") {
		t.Fatalf("Promise type uses a verbose qualified name:\n%s", artifacts.printed)
	}
	source := strings.Join(
		artifacts.printedByKind[emit.TargetFileSource],
		"\n",
	)
	support := strings.Join(
		artifacts.printedByKind[emit.TargetFileSupport],
		"\n",
	)
	if !strings.Contains(source, "export class Object__shadow_1") ||
		!strings.Contains(source, "globalThis.Object.freeze") {
		t.Fatalf("source host value is not collision-safe:\n%s", source)
	}
	if !strings.Contains(support, "Object.freeze") ||
		strings.Contains(support, "globalThis.Object.freeze") {
		t.Fatalf("isolated support host value is not direct:\n%s", support)
	}
	waveThreeTypecheck(t, workingDirectory, artifacts.paths)
}
