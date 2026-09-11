package conversion_test

import (
	"strings"
	"testing"

	"github.com/tsoniclang/gotots/internal/emit"
)

func TestStringAndSliceRawStorageRetainsDescriptorWords(test *testing.T) {
	for _, sourceType := range []string{"string", "[]uint32"} {
		test.Run(sourceType, func(test *testing.T) {
			loaded := loadMemoryStorageCase(test, "func Convert(value *"+sourceType+") *"+sourceType+" { return (*"+sourceType+")(unsafe.Pointer(value)) }")
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
			for _, required := range []string{"field<RawPointer | undefined>()", "field<int64>()", "memoryField", "projectPointer", "toRawPointer", "reinterpretRawPointer"} {
				if !strings.Contains(printed, required) {
					test.Fatalf("descriptor output lacks %q", required)
				}
			}
		})
	}
}

func TestOrdinaryDescriptorFieldsDoNotDemandMemoryProjection(test *testing.T) {
	loaded := loadMemoryStorageCase(test, `
var _ unsafe.Pointer
type Holder struct { Text string; Values []uint32; Array [2]string }
func Update(value *Holder, text string, values []uint32) string {
    value.Text = text
    value.Values = values
    value.Array[0] = text
    return value.Text
}
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
	for _, forbidden := range []string{"memoryLayout", "memoryArrayLayout", "toRawPointer", "reinterpretRawPointer", "goArrayStorage"} {
		if strings.Contains(printed, forbidden) {
			test.Fatalf("ordinary storage accidentally requests %q", forbidden)
		}
	}
}

func TestPhysicalRecordBindsOriginalFieldLocations(test *testing.T) {
	loaded := loadMemoryStorageCase(test, `
type Holder struct { Count uint32; Text string; Values []uint32; Array [2]uint32 }
func Convert(value *Holder) *Holder { return (*Holder)(unsafe.Pointer(value)) }
func CountAddress(value *Holder) *uint32 { return &Convert(value).Count }
`)
	roots, err := emit.ExportedAPIRoots(loaded)
	if err != nil {
		test.Fatal(err)
	}
	emission, err := emit.Compile(loaded.Program(), roots)
	if err != nil {
		test.Fatal(err)
	}
	strictTypecheckEmission(test, emission)
	_, _, printed := printConversions(test, test.TempDir(), emission)
	for _, required := range []string{"bindMemoryField(", "bindMemoryRecord(", "viewPointer<", "addressOf<uint32>(", ".Count, 0, 4, memoryLayout<uint32>", "FixedArray<uint32, 2>", "field<gostring>()", "field<RawPointer | undefined>()"} {
		if !strings.Contains(printed, required) {
			test.Fatalf("physical/logical location relation lacks %q", required)
		}
	}
	for _, forbidden := range []string{"locatedRecord", " as unknown", " as any", "memoryLayout<Holder>("} {
		if strings.Contains(printed, forbidden) {
			test.Fatalf("physical/logical location relation contains %q", forbidden)
		}
	}
}
