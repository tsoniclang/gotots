package reflectvalue_test

import (
	"strings"
	"testing"
)

// TestReflectAddressInterfacesCanonicalizeWithNativeEvidence proves that
// addresses produced from pointer elements, struct fields, and slice elements
// recover the exact pointer interface and asserted method contracts. Native Go
// evidence independently fixes the expected identity and mutation behavior.
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

type Decoder interface {
	Decode([]byte) error
}

func (label *Label) Decode(text []byte) error {
	*label = Label(string(text))
	return nil
}

type Holder struct {
	Value Label
}

type GenericLabel[T ~string] struct {
	Value T
}

func (label *GenericLabel[T]) Decode(text []byte) error {
	label.Value = T(string(text))
	return nil
}

type GenericHolder struct {
	Value GenericLabel[string]
}

func Addresses() string {
	direct := Label("direct")
	directValue := reflect.ValueOf(&direct).Elem()
	directPointer, directOK := reflect.TypeAssert[*Label](directValue.Addr())
	directAgain, directAgainOK := reflect.TypeAssert[*Label](directValue.Addr())
	interfacePointer, interfaceOK := directValue.Addr().Interface().(*Label)
	directDecoder, directDecoderOK := directValue.Addr().Interface().(Decoder)
	reflectedDecoder, reflectedDecoderOK := reflect.TypeAssert[Decoder](directValue.Addr())
	recoveredType := reflect.ValueOf(directValue.Addr().Interface()).Type()
	_ = directDecoder.Decode([]byte("interface"))
	_ = reflectedDecoder.Decode([]byte("reflected"))

	holder := Holder{Value: "field"}
	fieldValue := reflect.ValueOf(&holder).Elem().Field(0)
	fieldPointer, fieldOK := reflect.TypeAssert[*Label](fieldValue.Addr())
	fieldDecoder, fieldDecoderOK := fieldValue.Addr().Interface().(Decoder)
	_ = fieldDecoder.Decode([]byte("updated"))

	values := []Label{"slice"}
	sliceValue := reflect.ValueOf(values).Index(0)
	slicePointer, sliceOK := reflect.TypeAssert[*Label](sliceValue.Addr())
	sliceDecoder, sliceDecoderOK := reflect.TypeAssert[Decoder](sliceValue.Addr())
	_ = sliceDecoder.Decode([]byte("indexed"))

	genericHolder := GenericHolder{Value: GenericLabel[string]{Value: "generic"}}
	genericValue := reflect.ValueOf(&genericHolder).Elem().Field(0)
	genericDecoder, genericDecoderOK := reflect.TypeAssert[Decoder](genericValue.Addr())
	_ = genericDecoder.Decode([]byte("instantiated"))

	return fmt.Sprintf(
		"%t %t %t %t %t %t %t %t %s/%s %q %t %t %t %q %t %t %t %q %t %q",
		directOK, directAgainOK,
		directPointer == directAgain,
		interfaceOK, interfacePointer == directPointer,
		directDecoderOK, reflectedDecoderOK,
		recoveredType == directValue.Addr().Type(),
		directValue.Addr().Type().Kind(), directValue.Addr().Elem().Kind(), direct,
		fieldOK, fieldPointer != nil, fieldDecoderOK, holder.Value,
		sliceOK, slicePointer != nil, sliceDecoderOK, values[0],
		genericDecoderOK, genericHolder.Value.Value,
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
	verifyReflectCanonicalInspect(
		t,
		source,
		"Addresses",
		"reflectvalue",
		typescriptRunner,
		goRunner,
		func(artifacts renderedArtifacts) {
			const adapter = "$goInterfaceAdapter$PointerTo_Named_reflectvalue$GenericLabelOf_string"
			start := strings.Index(artifacts.printed, "export class "+adapter)
			if start < 0 {
				start = strings.Index(artifacts.printed, "export const "+adapter)
			}
			if start < 0 {
				t.Fatalf("address-only generic adapter %q is absent", adapter)
			}
			end := len(artifacts.printed)
			if relative := strings.Index(artifacts.printed[start+1:], "\nexport "); relative >= 0 {
				end = start + 1 + relative
			}
			if !strings.Contains(artifacts.printed[start:end], "Decode(") {
				t.Fatalf("address-only generic adapter %q lacks its asserted interface contract", adapter)
			}
		},
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
