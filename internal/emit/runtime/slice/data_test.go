package slice_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestUnsafeSliceDataSelectsExactRetainedLocations(test *testing.T) {
	emission := compileSliceSource(test, `package slicevalues
import "unsafe"
type Named []uint32
type Record struct { Value uint32 }
func Data(values Named) *uint32 { return unsafe.SliceData(values) }
func PointerData(values []*uint32) **uint32 { return unsafe.SliceData(values) }
func RecordData(values []Record) *Record { return unsafe.SliceData(values) }
func ComplexData(values []complex128) *complex128 { return unsafe.SliceData(values) }
`)
	directory := test.TempDir()
	paths, _, printed := materialize(test, directory, emission)
	writeFile(test, filepath.Join(directory, "package.json"), "{\"type\":\"module\"}\n")
	typecheck(test, directory, paths)
	for _, required := range []string{
		"goSliceData<uint32>",
		"goSliceData<Pointer<uint32> | undefined>",
		"projectPointer<Record$Storage, Record>",
		"projectPointer<typeof GoComplex128Storage, GoComplex128>",
	} {
		if !strings.Contains(printed.source, required) {
			test.Fatalf("slice data output lacks %q", required)
		}
	}
	for _, required := range []string{"if (value.isNil())", "if (value.capacity === 0)", "allocatePointer<T>(zero)", "value.slice(0, 1, null).address(0)"} {
		if !strings.Contains(printed.runtime, required) {
			test.Fatalf("slice data runtime lacks %q", required)
		}
	}
	for _, forbidden := range []string{"memoryLayout", "toRawPointer", " as any", " as unknown"} {
		if strings.Contains(printed.source+printed.runtime, forbidden) {
			test.Fatalf("slice data required %q", forbidden)
		}
	}
}

func TestUnsafeSliceDataAliasFixtureIsStrict(test *testing.T) {
	source, err := os.ReadFile(filepath.Join(repositoryRoot(), "testdata", "constructs", "value", "slicedata", "source.go"))
	if err != nil {
		test.Fatal(err)
	}
	emission := compileSliceSource(test, string(source))
	directory := test.TempDir()
	paths, _, printed := materialize(test, directory, emission)
	writeFile(test, filepath.Join(directory, "package.json"), "{\"type\":\"module\"}\n")
	typecheck(test, directory, paths)
	if strings.Count(printed.runtime, "export function goSliceData<T>") != 1 {
		test.Fatal("slice data helper is not demanded exactly once")
	}
}
