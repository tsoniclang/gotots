package reflectvalue_test

import (
	"strings"
	"testing"
)

func TestReflectBlankStructFieldsCanonicalizeWithNativeEvidence(t *testing.T) {
	source := `package reflectvalue

import (
	"fmt"
	"reflect"
)

type Marker[T any] struct {
	_ [0]T
	Value int
}

type Failure interface {
	Error() string
}

type InterfaceMarker struct {
	_ Failure
	Value string
}

func BlankFieldFacts() string {
	arrayValue := reflect.ValueOf(Marker[string]{Value: 7})
	arrayBlank := arrayValue.Field(0)
	interfaceValue := reflect.ValueOf(InterfaceMarker{Value: "ok"})
	interfaceBlank := interfaceValue.Field(0)
	return fmt.Sprintf(
		"array=%d/%s/%s/%t/%d interface=%d/%s/%s/%t/%t/%s",
		arrayValue.NumField(),
		arrayBlank.Type().String(),
		arrayBlank.Kind().String(),
		arrayBlank.CanSet(),
		arrayValue.Field(1).Int(),
		interfaceValue.NumField(),
		interfaceBlank.Type().String(),
		interfaceBlank.Kind().String(),
		interfaceBlank.CanSet(),
		interfaceBlank.IsZero(),
		interfaceValue.Field(1).String(),
	)
}
`
	typescriptRunner := `const facts = BlankFieldFacts();
console.log(facts);
`
	goRunner := `package main

import (
	"fmt"

	fixture "example.com/reflectvalue"
)

func main() {
	fmt.Println(fixture.BlankFieldFacts())
}
`
	verifyReflectCanonicalInspect(
		t,
		source,
		"BlankFieldFacts",
		"reflectvalue",
		typescriptRunner,
		goRunner,
		func(artifacts renderedArtifacts) {
			for _, required := range []string{
				"settable: false",
				"reflect: Value.Set using unaddressable value",
				".$storageOf(instance).Value",
			} {
				if !strings.Contains(artifacts.printed, required) {
					var storageLines []string
					for _, line := range strings.Split(artifacts.printed, "\n") {
						if strings.Contains(line, "$storageOf") {
							storageLines = append(storageLines, line)
						}
					}
					t.Fatalf(
						"blank-field reflection artifact lacks %q:\n%s",
						required,
						strings.Join(storageLines, "\n"),
					)
				}
			}
		},
	)
}
