package reflectvalue_test

import (
	"strings"
	"testing"
)

func TestReflectInterfaceFieldLocationCanonicalizesWithNativeEvidence(t *testing.T) {
	source := `package reflectvalue

import (
	"fmt"
	"reflect"
)

type Named interface {
	Name() string
}

type Label struct {
	Text string
}

func (label Label) Name() string {
	return label.Text
}

type Record struct {
	Value Named
}

type GenericRecord[T any] struct {
	Value Named
	Marker T
}

type Source struct {
	Value Label
}

func wrongValueRejected(field reflect.Value) (rejected bool) {
	defer func() {
		rejected = recover() != nil
	}()
	field.Set(reflect.ValueOf(1))
	return false
}

func InterfaceFieldFacts() string {
	record := &Record{}
	field := reflect.ValueOf(record).Elem().Field(0)
	before := fmt.Sprintf(
		"%t %s %t %t %t",
		field.IsValid(),
		field.Kind().String(),
		field.IsNil(),
		field.IsZero(),
		field.CanSet(),
	)

	source := &Source{Value: Label{Text: "first"}}
	field.Set(reflect.ValueOf(source).Elem().Field(0))
	source.Value.Text = "changed"
	after := fmt.Sprintf(
		"%s %t %t %s %s %t",
		field.Kind().String(),
		field.IsNil(),
		field.IsZero(),
		field.Elem().Kind().String(),
		field.Interface().(Named).Name(),
		wrongValueRejected(field),
	)

	field.Set(reflect.Zero(field.Type()))
	final := fmt.Sprintf(
		"%t %t %t",
		field.IsValid(),
		field.IsNil(),
		field.IsZero(),
	)

	generic := &GenericRecord[int]{}
	genericField := reflect.ValueOf(generic).Elem().Field(0)
	genericField.Set(reflect.ValueOf(source).Elem().Field(0))
	genericResult := fmt.Sprintf(
		"%t %s",
		genericField.CanSet(),
		generic.Value.Name(),
	)
	return before + " | " + after + " | " + final + " | " + genericResult
}
`
	typescriptRunner := `const facts = InterfaceFieldFacts();
console.log(facts);
`
	goRunner := `package main

import (
	"fmt"

	fixture "example.com/reflectvalue"
)

func main() {
	fmt.Println(fixture.InterfaceFieldFacts())
}
`
	verifyReflectCanonicalInspect(
		t,
		source,
		"InterfaceFieldFacts",
		"reflectvalue",
		typescriptRunner,
		goRunner,
		func(artifacts renderedArtifacts) {
			for _, required := range []string{
				"value === undefined",
				"reflect: Value.Set received a value outside the interface contract",
				"zero: (): GoInterfaceValue | undefined => undefined",
				".$storageOf(instance).Value",
			} {
				if !strings.Contains(artifacts.printed, required) {
					t.Fatalf("interface-field reflection artifact lacks %q", required)
				}
			}
		},
	)
}

func TestReflectProviderInterfaceFieldCanonicalizesWithNativeEvidence(t *testing.T) {
	source := `package reflectvalue

import (
	"fmt"
	"reflect"
)

type TypeHolder struct {
	Value reflect.Type
}

func ProviderInterfaceFieldFacts() string {
	holder := &TypeHolder{Value: reflect.TypeOf(int32(0))}
	field := reflect.ValueOf(holder).Elem().Field(0)
	before := field.Interface().(reflect.Type).String()
	field.Set(reflect.ValueOf(reflect.TypeOf("")))
	return fmt.Sprintf("%s %s", before, holder.Value.String())
}
`
	typescriptRunner := `const facts = ProviderInterfaceFieldFacts();
console.log(facts);
`
	goRunner := `package main

import (
	"fmt"

	fixture "example.com/reflectvalue"
)

func main() {
	fmt.Println(fixture.ProviderInterfaceFieldFacts())
}
`
	verifyReflectCanonicalInspect(
		t,
		source,
		"ProviderInterfaceFieldFacts",
		"reflectvalue",
		typescriptRunner,
		goRunner,
		func(artifacts renderedArtifacts) {
			if !strings.Contains(artifacts.printed, "$candidate is reflect.Type") {
				t.Fatal("sealed provider interface field lacks its typed admission guard")
			}
		},
	)
}
