package reflectvalue_test

import "testing"

// TestReflectSliceOperationsMatchGo proves the slice value family: Len,
// Cap, Index element locations aliasing the original backing array,
// element mutation through Set, element zero evidence, Append growth, and
// MakeSlice construction all match Go exactly.
func TestReflectSliceOperationsCanonicalizeWithNativeEvidence(t *testing.T) {
	source := `package reflectvalue

import (
	"fmt"
	"reflect"
)

func Checksum() string {
	values := []int{4, 0, 15}
	view := reflect.ValueOf(values)
	total := 0
	zeros := 0
	for index := 0; index < view.Len(); index++ {
		element := view.Index(index)
		if element.IsZero() {
			zeros++
			continue
		}
		total += int(element.Int())
	}
	view.Index(1).Set(reflect.ValueOf(23))
	grown := reflect.Append(view, reflect.ValueOf(8), reflect.ValueOf(16))
	made := reflect.MakeSlice(reflect.TypeOf(values), 2, 5)
	return fmt.Sprintf(
		"%d %d %d %d %d %d %t %d %d %d %d %d %t %t",
		total, zeros,
		values[0], values[1], values[2],
		view.Cap(),
		view.Index(0).CanSet(),
		grown.Len(), grown.Cap(),
		int(grown.Index(4).Int()),
		made.Len(), made.Cap(),
		made.Index(1).IsZero(),
		grown.Index(0).CanSet(),
	)
}
`
	typescriptRunner := `const facts = await Checksum();
console.log(facts);
`
	goRunner := `package main

import (
	"fmt"

	fixture "example.com/reflectvalue"
)

func main() {
	fmt.Println(fixture.Checksum())
}
`
	verifyReflectCanonical(
		t,
		source,
		"Checksum",
		"reflectvalue",
		typescriptRunner,
		goRunner,
	)
}
