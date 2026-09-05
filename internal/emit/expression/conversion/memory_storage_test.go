package conversion_test

import (
	"context"
	"errors"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tsoniclang/gotots/internal/emit"
	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/load"
)

func TestRawAggregateLayoutSelectsPhysicalStorage(t *testing.T) {
	loaded := loadMemoryStorageCase(t, `
type Pair struct { First uint32; Second uint32 }
func Convert(value *Pair) *Pair { return (*Pair)(unsafe.Pointer(value)) }
`)
	root, err := emit.NewRoot(loaded.Types().Scope().Lookup("Convert"))
	if err != nil {
		t.Fatal(err)
	}
	emission, err := emit.Compile(loaded.Program(), []emit.Root{root})
	if err != nil {
		t.Fatal(err)
	}
	strictTypecheckEmission(t, emission)
	_, _, printed := printConversions(t, t.TempDir(), emission)
	for _, required := range []string{"memoryLayout<Pair$Storage>", "projectPointer<Pair, Pair$Storage>", "projectPointer<Pair$Storage, Pair>", "Pair.$storageOf", "Pair.$fromStorage", ".Second, 4, 4)"} {
		if !strings.Contains(printed, required) {
			t.Fatalf("physical memory output lacks %q", required)
		}
	}
	if strings.Contains(printed, "memoryLayout<Pair>(") {
		t.Fatal("logical wrapper acquired the physical layout")
	}
}

func TestRawMemoryRejectsUnrepresentedDescriptorFamilies(t *testing.T) {
	for _, spelling := range []string{"[2]uint32", "[]uint32", "string", "complex128", "interface{}", "map[int]int", "chan int", "func()", "struct{ Nested struct{ Value uint32 } }", "struct{ _ uint32; Value uint32 }"} {
		t.Run(spelling, func(t *testing.T) {
			loaded := loadMemoryStorageCase(t, "func Convert(value *"+spelling+") unsafe.Pointer { return unsafe.Pointer(value) }")
			root, err := emit.NewRoot(loaded.Types().Scope().Lookup("Convert"))
			if err != nil {
				t.Fatal(err)
			}
			_, err = emit.Compile(loaded.Program(), []emit.Root{root})
			var unsupported *api.UnsupportedError
			if !errors.As(err, &unsupported) || unsupported.Category != api.CategoryExpression {
				t.Fatalf("unrepresented physical descriptor = %v, want source expression boundary", err)
			}
		})
	}
}

func TestRawScalarLayoutRetainsSelected386Alignment(t *testing.T) {
	directory := t.TempDir()
	writeFile(t, filepath.Join(directory, "go.mod"), "module example.com/memory386\n\ngo 1.26.4\n")
	writeFile(t, filepath.Join(directory, "source.go"), "package conversion\nimport \"unsafe\"\nfunc Convert(value *uint64) unsafe.Pointer { return unsafe.Pointer(value) }\n")
	profile, err := load.NewBuildProfile("linux", "386", false, nil)
	if err != nil {
		t.Fatal(err)
	}
	loaded, err := load.One(context.Background(), load.Request{Directory: directory, Pattern: ".", BuildProfile: profile})
	if err != nil {
		t.Fatal(err)
	}
	root, err := emit.NewRoot(loaded.Types().Scope().Lookup("Convert"))
	if err != nil {
		t.Fatal(err)
	}
	options := emit.DefaultOptions()
	options.IntegerRepresentation = emit.IntegerRepresentationBigInt
	emission, err := emit.CompileWithOptions(loaded.Program(), []emit.Root{root}, options)
	if err != nil {
		t.Fatal(err)
	}
	strictTypecheckEmission(t, emission)
	_, _, printed := printConversions(t, t.TempDir(), emission)
	if !strings.Contains(printed, "little32") || !strings.Contains(printed, "8, 4, 8)") {
		t.Fatal("selected 386 source alignment was replaced with scalar width or host ABI")
	}
}

func loadMemoryStorageCase(t *testing.T, declarations string) *load.Package {
	t.Helper()
	directory := t.TempDir()
	writeFile(t, filepath.Join(directory, "go.mod"), "module example.com/memorystorage\n\ngo 1.26.4\n")
	writeFile(t, filepath.Join(directory, "source.go"), "package conversion\nimport \"unsafe\"\n"+declarations)
	loaded, err := load.One(context.Background(), load.Request{Directory: directory, Pattern: "."})
	if err != nil {
		t.Fatal(err)
	}
	return loaded
}
