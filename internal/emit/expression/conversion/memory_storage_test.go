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
	for _, required := range []string{"const Pair$Storage:", "= struct({", "First: field<uint32>()", "Second: field<uint32>()", "type Pair$Storage = typeof Pair$Storage;", "memoryLayout<Pair$Storage>", "projectPointer<Pair, Pair$Storage>", "projectPointer<Pair$Storage, Pair>", "Pair.$storageOf", "Pair.$fromStorage", ".Second, 4, 4, memoryLayout<uint32>"} {
		if !strings.Contains(printed, required) {
			t.Fatalf("physical memory output lacks %q", required)
		}
	}
	if strings.Contains(printed, "memoryLayout<Pair>(") {
		t.Fatal("logical wrapper acquired the physical layout")
	}
}

func TestRawMemoryRejectsUnrepresentedDescriptorFamilies(t *testing.T) {
	for _, spelling := range []string{"[2]uint32", "[]uint32", "string", "complex128", "interface{}", "map[int]int", "chan int", "func()"} {
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

func TestRawNestedLayoutRetainsPhysicalChildren(test *testing.T) {
	loaded := loadMemoryStorageCase(test, `
type Pair struct { First uint32; Second uint32 }
type Outer struct { Tag uint32; Inner Pair }
func Convert(value *Outer) *Outer { return (*Outer)(unsafe.Pointer(value)) }
`)
	root, err := emit.NewRoot(loaded.Types().Scope().Lookup("Convert"))
	if err != nil {
		test.Fatal(err)
	}
	emission, err := emit.Compile(loaded.Program(), []emit.Root{root})
	if err != nil {
		test.Fatal(err)
	}
	strictTypecheckEmission(test, emission)
	_, _, printed := printConversions(test, test.TempDir(), emission)
	for _, required := range []string{"memoryLayout<Outer$Storage>", ".Inner, 4, 4, memoryLayout<Pair$Storage>", ".Second, 4, 4, memoryLayout<uint32>"} {
		if !strings.Contains(printed, required) {
			test.Fatalf("nested physical memory output lacks %q", required)
		}
	}
}

func TestRawClosedGenericLayoutRetainsConcreteFieldContracts(test *testing.T) {
	loaded := loadMemoryStorageCase(test, `
type Box[Element any] struct { Value Element }
func Convert(value *Box[uint32]) *Box[uint32] { return (*Box[uint32])(unsafe.Pointer(value)) }
`)
	root, err := emit.NewRoot(loaded.Types().Scope().Lookup("Convert"))
	if err != nil {
		test.Fatal(err)
	}
	emission, err := emit.Compile(loaded.Program(), []emit.Root{root})
	if err != nil {
		test.Fatal(err)
	}
	strictTypecheckEmission(test, emission)
	_, _, printed := printConversions(test, test.TempDir(), emission)
	if !strings.Contains(printed, "Value: field<uint32>()") ||
		!strings.Contains(printed, ".Value, 0, 4, memoryLayout<uint32>") {
		test.Fatal("closed generic memory lacks its concrete neutral value-record and child-layout contracts")
	}
	if strings.Contains(printed, "memoryLayout<Box$Storage<uint32>>") {
		test.Fatal("physical memory depends on an unproven generic storage alias")
	}
}

func TestRawNestedGenericLayoutRetainsConcreteFieldContracts(test *testing.T) {
	loaded := loadMemoryStorageCase(test, `
type Box[Element any] struct { Value Element }
type Outer[Element any] struct { Inner Box[Element]; Next *Outer[Element] }
func Convert(value *Outer[uint32]) *Outer[uint32] { return (*Outer[uint32])(unsafe.Pointer(value)) }
`)
	root, err := emit.NewRoot(loaded.Types().Scope().Lookup("Convert"))
	if err != nil {
		test.Fatal(err)
	}
	emission, err := emit.Compile(loaded.Program(), []emit.Root{root})
	if err != nil {
		test.Fatal(err)
	}
	strictTypecheckEmission(test, emission)
	_, _, printed := printConversions(test, test.TempDir(), emission)
	for _, required := range []string{"Value: field<uint32>()", "Inner: field<", "Next: field<Pointer<Outer__from_conversion<uint32>> | undefined>()", ".Value, 0, 4, memoryLayout<uint32>"} {
		if !strings.Contains(printed, required) {
			for _, line := range strings.Split(printed, "\n") {
				if strings.Contains(line, "Next:") {
					test.Log(line)
				}
			}
			test.Fatalf("nested generic storage lost %q", required)
		}
	}
	for _, forbidden := range []string{"memoryLayout<Box$Storage<uint32>>", "memoryLayout<Outer$Storage<uint32>>"} {
		if strings.Contains(printed, forbidden) {
			test.Fatalf("closed generic memory retains unproven alias %q", forbidden)
		}
	}
}

func TestOrdinaryGenericFieldUpdateDoesNotRequestRawMemory(test *testing.T) {
	loaded := loadMemoryStorageCase(test, `
type Box[Element any] struct { Value Element }
func Update(value *Box[uint32]) uint32 { value.Value = 9; return value.Value }
func Unselected(value *Box[uint32]) unsafe.Pointer { return unsafe.Pointer(value) }
`)
	root, err := emit.NewRoot(loaded.Types().Scope().Lookup("Update"))
	if err != nil {
		test.Fatal(err)
	}
	emission, err := emit.Compile(loaded.Program(), []emit.Root{root})
	if err != nil {
		test.Fatal(err)
	}
	strictTypecheckEmission(test, emission)
	_, _, printed := printConversions(test, test.TempDir(), emission)
	if !strings.Contains(printed, ".Value = 9") {
		test.Fatal("ordinary generic field update lost its direct property assignment")
	}
	for _, forbidden := range []string{"memoryLayout", "toRawPointer", "fromRawPointer", "memoryField"} {
		if strings.Contains(printed, forbidden) {
			test.Fatalf("ordinary field update requests %q", forbidden)
		}
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
	loaded, err := load.One(context.Background(), load.Request{Directory: directory, Pattern: ".", BuildProfile: profile,
		ToolCacheRoot: filepath.Join(repositoryRoot(), ".temp", "cache", "toolchain")})
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
	loaded, err := load.One(context.Background(), load.Request{Directory: directory, Pattern: ".",
		ToolCacheRoot: filepath.Join(repositoryRoot(), ".temp", "cache", "toolchain")})
	if err != nil {
		t.Fatal(err)
	}
	return loaded
}
