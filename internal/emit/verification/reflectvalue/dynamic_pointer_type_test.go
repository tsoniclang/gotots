package reflectvalue_test

import (
	"strings"
	"testing"
)

// TestDynamicPointerTypeCompositionCanonicalizesWithNativeEvidence covers PointerTo
// one canonical descriptor from a runtime-flowing Type without requiring a
// statically mentioned pointer type. It also proves value- and pointer-method
// implementation sets and repeated pointer composition.
func TestDynamicPointerTypeCompositionCanonicalizesWithNativeEvidence(t *testing.T) {
	source := `package reflectvalue

import (
	"fmt"
	"reflect"
)

type Marker interface { Mark() }

type ValueMethod struct{}
func (ValueMethod) Mark() {}

type PointerMethod struct{}
func (*PointerMethod) Mark() {}

func DynamicPointerFacts() string {
	marker := reflect.TypeOf((*Marker)(nil)).Elem()
	value := reflect.TypeOf(ValueMethod{})
	pointerOnly := reflect.TypeOf(PointerMethod{})
	valuePointer := reflect.PointerTo(value)
	pointerOnlyPointer := reflect.PointerTo(pointerOnly)
	doublePointer := reflect.PointerTo(valuePointer)
	again := reflect.PointerTo(value)
	return fmt.Sprintf(
		"%s %s %t %t %t %t %t %d %d",
		valuePointer.String(), doublePointer.String(),
		valuePointer == again, valuePointer.Elem() == value,
		value.Implements(marker), valuePointer.Implements(marker),
		pointerOnlyPointer.Implements(marker),
		valuePointer.Size(), valuePointer.Align(),
	)
}
`
	typescriptRunner := `const facts = await DynamicPointerFacts();
console.log(facts);
`
	goRunner := `package main

import (
	"fmt"

	fixture "example.com/reflectvalue"
)

func main() {
	fmt.Println(fixture.DynamicPointerFacts())
}
`
	verifyReflectCanonicalInspect(
		t,
		source,
		"DynamicPointerFacts",
		"reflectvalue",
		typescriptRunner,
		goRunner,
		func(artifacts renderedArtifacts) {
			for _, required := range []string{
				"methodSet:",
				"pointerMethodSet:",
				"pointerInheritsMethods: true",
			} {
				if !strings.Contains(artifacts.printed, required) {
					t.Fatalf(
						"dynamic pointer artifact lacks %q",
						required,
					)
				}
			}
		},
	)
}
