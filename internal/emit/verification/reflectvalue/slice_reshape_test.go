package reflectvalue_test

import "testing"

// TestReflectSliceReshapeMatchesGo proves the addressable slice value
// family: Elem over a pointer-to-slice storage cell, SetLen re-slicing,
// Grow preserving length while extending capacity, Bytes aliasing, and
// SetBytes replacing the whole header through original storage. Capacity
// is asserted only where the product growth profile provably agrees with
// the Go growth policy.
func TestReflectSliceReshapeMatchesGo(t *testing.T) {
	source := `package reflectvalue

import (
	"fmt"
	"reflect"
)

func Reshape() string {
	buffer := []byte{1, 2, 3, 4}
	pointer := &buffer
	view := reflect.ValueOf(pointer).Elem()
	view.SetLen(2)
	shortLen, shortCap := view.Len(), view.Cap()
	view.Grow(6)
	grownEnough := view.Cap() >= 8
	stableLen := view.Len()
	raw := view.Bytes()
	rawFirst := int(raw[0])
	view.SetBytes([]byte{9, 8, 7})
	return fmt.Sprintf(
		"%d %d %t %d %d %d %d %d %d %t",
		shortLen, shortCap, grownEnough, stableLen, rawFirst,
		len(buffer), int(buffer[0]), int(buffer[2]),
		view.Cap(), view.CanSet(),
	)
}
`
	typescriptRunner := `const facts = await Reshape();
console.log(facts);
`
	goRunner := `package main

import (
	"fmt"

	fixture "example.com/reflectvalue"
)

func main() {
	fmt.Println(fixture.Reshape())
}
`
	runReflectDifferential(
		t,
		source,
		"Reshape",
		"reflectvalue",
		typescriptRunner,
		goRunner,
	)
}
