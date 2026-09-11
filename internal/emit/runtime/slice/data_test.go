package slice_test

import (
	"os"
	"path/filepath"
	"regexp"
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
func ArrayData(values [][1024]uint32) *[1024]uint32 { return unsafe.SliceData(values) }
func AggregateData(values [][2]Record) *[2]Record { return unsafe.SliceData(values) }
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
	for _, required := range []string{"return value.$data(zero)", "if (this.isNil())", "if (this.capacity === 0)", "zero: () => T", "allocatePointer<T>(zero())", "this.slice(0, 1, null).address(0)"} {
		if !strings.Contains(printed.runtime, required) {
			test.Fatalf("slice data runtime lacks %q", required)
		}
	}
	if !strings.Contains(printed.source, "values, ():") {
		test.Fatal("slice data constructs an unused aggregate zero before selecting its address")
	}
	if strings.Count(printed.runtime, "zero()") != 1 ||
		!regexp.MustCompile(`if \(this.capacity === 0\)\s*(?:\{\s*)?return allocatePointer<T>\(zero\(\)\);`).MatchString(printed.runtime) {
		test.Fatal("zero construction is not confined to the unspecified empty address")
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
