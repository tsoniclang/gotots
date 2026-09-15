package conversion_test

import (
	"strings"
	"testing"

	"github.com/tsoniclang/gotots/internal/emit"
)

func TestRawBlankFieldsRetainSelectedSlots(test *testing.T) {
	loaded := loadMemoryStorageCase(test, `
type Record struct { _ *uint32; Field0 uint32; _ uint16; Tail struct{} }
func Convert(value *Record) *Record { return (*Record)(unsafe.Pointer(value)) }
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
	for _, required := range []string{
		"$blank0: field<Pointer<uint32> | undefined>()",
		"$blank2: field<uint16>()",
		".$blank0, 0, 8, memoryLayout<Pointer<uint32> | undefined>",
		".Field0, 8, 4, memoryLayout<uint32>",
		".$blank2, 12, 2, memoryLayout<uint16>",
		".Tail, 14, 1, memoryLayout<",
		"memoryLayout<Record$Storage>",
		", 16, 8, 16, memoryField(",
	} {
		if !strings.Contains(printed, required) {
			test.Fatalf("physical blank-field output lacks %q", required)
		}
	}
}

func TestRawEmptyStructStorageExcludesNominalBrands(test *testing.T) {
	loaded := loadMemoryStorageCase(test, `
func Convert(value *struct{}) *struct{} { return (*struct{})(unsafe.Pointer(value)) }
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
	if strings.Contains(printed, "memoryLayout<GoEmptyStruct>") {
		test.Fatal("a nominal Go value must not be used as a fieldless data layout")
	}
	for _, required := range []string{"memoryLayout<{}>", "$storageOf", "$fromStorage", "bindMemoryRecord"} {
		if !strings.Contains(printed, required) {
			test.Fatalf("empty storage conversion lacks %q", required)
		}
	}
}

func TestGenericEmptyArrayStorageMatchesItsConcreteLayout(test *testing.T) {
	loaded := loadMemoryStorageCase(test, `
type Generic[T any] struct { _ [0]T; Values [2]T }
type Record struct { Value Generic[struct{}] }
func Convert(value *Record) *Record { return (*Record)(unsafe.Pointer(value)) }
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
	for _, required := range []string{"GoContainerStorage<T>", "Generic<GoEmptyStruct>", "GoArray<{}, 0>", "GoArray<{}, 2>"} {
		if !strings.Contains(printed, required) {
			test.Fatalf("generic empty array storage lacks %q", required)
		}
	}
}
