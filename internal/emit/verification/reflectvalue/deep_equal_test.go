package reflectvalue_test

import "testing"

// TestReflectDeepEqualCanonicalizesWithNativeEvidence covers deep comparison and projection
// tail of the reflection family: DeepEqual over structs, slices, maps,
// pointers, distinct types, and nil; Interface returning an exact copy of
// a located struct; and Indirect unwrapping pointers.
func TestReflectDeepEqualCanonicalizesWithNativeEvidence(t *testing.T) {
	source := `package reflectvalue

import (
	"fmt"
	"reflect"
)

type Options struct {
	Name    string
	Count   int
	Verbose bool
}

func Compare() string {
	left := Options{Name: "strict", Count: 3}
	right := Options{Name: "strict", Count: 3}
	different := Options{Name: "loose", Count: 3}
	leftSlice := []int{1, 2, 3}
	rightSlice := []int{1, 2, 3}
	shorter := []int{1, 2}
	leftMap := map[string]int{"a": 1, "b": 2}
	rightMap := map[string]int{"b": 2, "a": 1}
	missing := map[string]int{"a": 1}
	source := &Options{Name: "view", Count: 9}
	view := reflect.ValueOf(source).Elem()
	snapshot := view.Interface().(Options)
	source.Name = "mutated"
	indirect := reflect.Indirect(reflect.ValueOf(&left))
	return fmt.Sprintf(
		"%t %t %t %t %t %t %t %t %t %q %s %d",
		reflect.DeepEqual(left, right),
		reflect.DeepEqual(left, different),
		reflect.DeepEqual(leftSlice, rightSlice),
		reflect.DeepEqual(leftSlice, shorter),
		reflect.DeepEqual(leftMap, rightMap),
		reflect.DeepEqual(leftMap, missing),
		reflect.DeepEqual(&left, &right),
		reflect.DeepEqual(left, leftSlice),
		reflect.DeepEqual(nil, nil),
		snapshot.Name,
		indirect.Kind().String(),
		int(indirect.Field(1).Int()),
	)
}
`
	typescriptRunner := `const facts = Compare();
console.log(facts);
`
	goRunner := `package main

import (
	"fmt"

	fixture "example.com/reflectvalue"
)

func main() {
	fmt.Println(fixture.Compare())
}
`
	verifyReflectCanonical(
		t,
		source,
		"Compare",
		"reflectvalue",
		typescriptRunner,
		goRunner,
	)
}
