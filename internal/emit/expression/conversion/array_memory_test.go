package conversion_test

import (
	"strings"
	"testing"

	"github.com/tsoniclang/gotots/internal/emit"
)

func TestRawArrayUsesBackingAndExactFixedStorage(t *testing.T) {
	loaded := loadMemoryStorageCase(t, `
func Convert(value *[2]uint32) *[2]uint32 { return (*[2]uint32)(unsafe.Pointer(value)) }
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
	for _, required := range []string{"FixedArray<uint32, 2>", "memoryArrayLayout<uint32, 2>", "goArrayLocation<uint32, 2>", "reinterpretRawPointer<FixedArray<uint32, 2>>", "goArrayFromRegion<uint32, 2>"} {
		if !strings.Contains(printed, required) {
			t.Fatalf("physical array output lacks %q", required)
		}
	}
	if strings.Contains(printed, "memoryLayout<GoArray") || strings.Contains(printed, "projectPointer<GoArray<uint32, 2>, FixedArray") {
		t.Fatal("logical array window was misidentified as physical array storage")
	}
}

func TestNestedArrayFieldsRetainBothExtents(t *testing.T) {
	loaded := loadMemoryStorageCase(t, `
type Matrix struct { Values [2][3]uint32 }
func Convert(value *Matrix) *Matrix { return (*Matrix)(unsafe.Pointer(value)) }
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
	for _, required := range []string{"FixedArray<FixedArray<uint32, 3>, 2>", "memoryArrayLayout<FixedArray<uint32, 3>, 2>", "memoryArrayLayout<uint32, 3>", "field<GoArray<", "field<FixedArray<"} {
		if !strings.Contains(printed, required) {
			t.Fatalf("nested array output lost %q", required)
		}
	}
	if strings.Contains(printed, "goArrayStorage") {
		t.Fatal("physical array views must not reintroduce copied backing storage")
	}
}

func TestHugeZeroSizedArrayExtentIsNotRoundedOrExpanded(t *testing.T) {
	loaded := loadMemoryStorageCase(t, `
func Convert() unsafe.Pointer {
    value := new([9007199254740993]struct{})
    return unsafe.Pointer(value)
}
`)
	root, err := emit.NewRoot(loaded.Types().Scope().Lookup("Convert"))
	if err != nil {
		t.Fatal(err)
	}
	emission, err := emit.Compile(loaded.Program(), []emit.Root{root})
	if err != nil {
		t.Fatal(err)
	}
	_, _, printed := printConversions(t, t.TempDir(), emission)
	if !strings.Contains(printed, "9007199254740993n") || !strings.Contains(printed, "defaultValue<FixedArray<") {
		t.Fatal("huge zero-sized storage lacks an exact extent and canonical zero allocation")
	}
	if strings.Contains(printed, "9007199254740992") || len(printed) > 100000 {
		t.Fatal("huge zero-sized storage was rounded or expanded")
	}
}

func TestZeroLengthRawArrayPreservesItsLocationWithoutAnElement(test *testing.T) {
	loaded := loadMemoryStorageCase(test, `
func Convert(value *[0]uint32) *[0]uint32 { return (*[0]uint32)(unsafe.Pointer(value)) }
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
	for _, required := range []string{"FixedArray<uint32, 0>", "memoryArrayLayout<uint32, 0>", "projectPointer<GoArray<uint32, 0>, FixedArray<uint32, 0>>"} {
		if !strings.Contains(printed, required) {
			test.Fatalf("empty array location lacks %q", required)
		}
	}
	if strings.Contains(printed, "goArrayLocation<uint32, 0>") || strings.Contains(printed, "addressOf<uint32>(") {
		test.Fatal("empty array location fabricated a first element")
	}
}
