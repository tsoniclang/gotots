package reflectvalue_test

import (
	"strings"
	"testing"
)

// TestReflectAddressInterfacesCanonicalizeWithNativeEvidence proves that
// addresses produced from pointer elements, struct fields, and slice elements
// recover the exact pointer interface, preserve location identity, and mutate
// the same storage observed by ordinary Go code.
func TestReflectAddressInterfacesCanonicalizeWithNativeEvidence(t *testing.T) {
	source := `package reflectvalue

import (
	"fmt"
	"reflect"
)

type Label string

func (label *Label) UnmarshalText(text []byte) error {
	*label = Label(string(text))
	return nil
}

type Holder struct {
	Value Label
}

func Addresses() string {
	direct := Label("direct")
	directValue := reflect.ValueOf(&direct).Elem()
	directPointer, directOK := reflect.TypeAssert[*Label](directValue.Addr())
	directAgain, directAgainOK := reflect.TypeAssert[*Label](directValue.Addr())
	interfacePointer, interfaceOK := directValue.Addr().Interface().(*Label)
	recoveredType := reflect.ValueOf(directValue.Addr().Interface()).Type()
	_ = directPointer.UnmarshalText([]byte("changed"))

	holder := Holder{Value: "field"}
	fieldValue := reflect.ValueOf(&holder).Elem().Field(0)
	fieldPointer, fieldOK := reflect.TypeAssert[*Label](fieldValue.Addr())
	_ = fieldPointer.UnmarshalText([]byte("updated"))

	values := []Label{"slice"}
	sliceValue := reflect.ValueOf(values).Index(0)
	slicePointer, sliceOK := reflect.TypeAssert[*Label](sliceValue.Addr())
	_ = slicePointer.UnmarshalText([]byte("indexed"))

	return fmt.Sprintf(
		"%t %t %t %t %t %t %s/%s %q %t %q %t %q",
		directOK, directAgainOK,
		directPointer == directAgain,
		interfaceOK, interfacePointer == directPointer,
		recoveredType == directValue.Addr().Type(),
		directValue.Addr().Type().Kind(), directValue.Addr().Elem().Kind(), direct,
		fieldOK, holder.Value,
		sliceOK, values[0],
	)
}
`
	typescriptRunner := `const facts = Addresses();
console.log(facts);
`
	goRunner := `package main

import (
	"fmt"

	fixture "example.com/reflectvalue"
)

func main() {
	fmt.Println(fixture.Addresses())
}
`
	verifyReflectCanonical(
		t,
		source,
		"Addresses",
		"reflectvalue",
		typescriptRunner,
		goRunner,
	)
}

func TestReflectAddressDoesNotDemandChildPointerReflectionClosure(t *testing.T) {
	source := `package reflectvalue

import "reflect"

type Field int

type Record struct {
	Value Field
}

func Audit() int {
	record := Record{Value: 42}
	return int(reflect.ValueOf(&record).Elem().Field(0).Int())
}
`
	verifyReflectCanonicalInspect(
		t,
		source,
		"Audit",
		"reflectvalue",
		"console.log(Audit());\n",
		`package main

import (
	"fmt"

	fixture "example.com/reflectvalue"
)

func main() {
	fmt.Println(fixture.Audit())
}
`,
		func(artifacts renderedArtifacts) {
			const adapter = "$goInterfaceAdapter$PointerTo_Named_reflectvalue$Field"
			if !strings.Contains(artifacts.printed, adapter) {
				t.Fatalf("address callback does not retain %q", adapter)
			}
			const descriptor = "$goReflectType$PointerTo_Named_reflectvalue$Field"
			if strings.Contains(artifacts.printed, descriptor) {
				t.Fatalf("address callback eagerly retains %q", descriptor)
			}
		},
	)
}
