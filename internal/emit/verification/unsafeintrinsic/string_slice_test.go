package unsafeintrinsic_test

import (
	"context"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tsoniclang/gotots/internal/emit"
	"github.com/tsoniclang/gotots/internal/load"
)

func TestUnsafeStringFromSliceUsesOneBoundedRuntimeOperation(t *testing.T) {
	project := t.TempDir()
	writeProgramFile(
		t,
		filepath.Join(project, "go.mod"),
		"module example.com/unsafestring\n\ngo 1.26.4\n",
	)
	writeProgramFile(t, filepath.Join(project, "source.go"), `package unsafestring

import "unsafe"

func Build(bytes []byte, offset int, length int) string {
	return unsafe.String(&bytes[offset], length)
}
`)
	program, err := load.Load(context.Background(), load.Request{
		Directory: project,
		Pattern:   ".",
	})
	if err != nil {
		t.Fatal(err)
	}
	root, err := emit.NewRoot(program.Roots()[0].Types().Scope().Lookup("Build"))
	if err != nil {
		t.Fatal(err)
	}
	emission, err := emit.Compile(program, []emit.Root{root})
	if err != nil {
		t.Fatal(err)
	}
	artifacts := materializeArtifacts(t, emission, t.TempDir())
	for _, required := range []string{
		"GoString.fromRegion(",
		"goSliceElementRegion<uint8>(",
		"let logicalResult",
		"value.$arrayLocation(0)",
		"return goRegionView<T>(location, index)",
		"index >= value.sourceLength()",
	} {
		if !strings.Contains(artifacts.printed, required) {
			t.Fatalf("unsafe string output lacks %q:\n%s", required, artifacts.printed)
		}
	}
	for _, forbidden := range []string{
		"GoPointer",
		"GoUnsafePointer",
		"goPointerRegion",
		"goUnsafeString<",
		"offsetRawPointer(",
		"reinterpretRawPointer<",
		"toRawPointer<",
	} {
		if strings.Contains(artifacts.printed, forbidden) {
			t.Fatalf("unsafe string output retains %q:\n%s", forbidden, artifacts.printed)
		}
	}
}
