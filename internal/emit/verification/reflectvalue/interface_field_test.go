package reflectvalue_test

import (
	"strings"
	"testing"
)

func TestReflectInterfaceFieldLocationMatchesGo(t *testing.T) {
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
	return before + " | " + after + " | " + final
}
`
	typescriptRunner := `const facts = await InterfaceFieldFacts();
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
	runReflectDifferentialInspect(
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
			} {
				if !strings.Contains(artifacts.printed, required) {
					t.Fatalf("interface-field reflection artifact lacks %q", required)
				}
			}
		},
	)
}
