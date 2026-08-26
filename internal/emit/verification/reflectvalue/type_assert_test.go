package reflectvalue_test

import "testing"

// TestReflectTypeAssertCanonicalizesWithNativeEvidence covers reflect.TypeAssert over concrete
// scalar, string, and pointer type arguments with both hit and miss
// outcomes, matching the Go comma-ok semantics exactly.
func TestReflectTypeAssertCanonicalizesWithNativeEvidence(t *testing.T) {
	source := `package reflectvalue

import (
	"fmt"
	"reflect"
)

func Assert() string {
	number := reflect.ValueOf(42)
	text := reflect.ValueOf("go")
	pointer := reflect.ValueOf(new(int))
	gotNumber, numberOK := reflect.TypeAssert[int](number)
	missText, missOK := reflect.TypeAssert[string](number)
	gotText, textOK := reflect.TypeAssert[string](text)
	gotPointer, pointerOK := reflect.TypeAssert[*int](pointer)
	missPointer, missPointerOK := reflect.TypeAssert[*int](text)
	return fmt.Sprintf(
		"%d %t %q %t %q %t %t %t %t %t",
		gotNumber, numberOK,
		missText, missOK,
		gotText, textOK,
		gotPointer != nil, pointerOK,
		missPointer == nil, missPointerOK,
	)
}
`
	typescriptRunner := `const facts = Assert();
console.log(facts);
`
	goRunner := `package main

import (
	"fmt"

	fixture "example.com/reflectvalue"
)

func main() {
	fmt.Println(fixture.Assert())
}
`
	verifyReflectCanonical(
		t,
		source,
		"Assert",
		"reflectvalue",
		typescriptRunner,
		goRunner,
	)
}
