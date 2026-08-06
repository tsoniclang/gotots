package reflectvalue_test

import (
	"strings"
	"testing"
)

func TestReflectNewDefinedScalarLocationMatchesGo(t *testing.T) {
	source := `package reflectvalue

import (
	"fmt"
	"reflect"
)

type DefinedText string
type DefinedCount uint32

func DefinedScalarFacts() string {
	text := reflect.New(reflect.TypeOf(DefinedText("")))
	text.Elem().SetString("named")
	count := reflect.New(reflect.TypeOf(DefinedCount(0)))
	count.Elem().SetUint(37)
	return fmt.Sprintf("%s %d", text.Elem().Interface().(DefinedText), count.Elem().Interface().(DefinedCount))
}
`
	typescriptRunner := `const facts = await DefinedScalarFacts();
console.log(facts);
`
	goRunner := `package main

import (
	"fmt"

	fixture "example.com/reflectvalue"
)

func main() {
	fmt.Println(fixture.DefinedScalarFacts())
}
`
	runReflectDifferentialInspect(
		t,
		source,
		"DefinedScalarFacts",
		"reflectvalue",
		typescriptRunner,
		goRunner,
		func(artifacts renderedArtifacts) {
			for _, required := range []string{
				"GoPointer.cell(new DefinedText",
				"GoPointer.cell(new DefinedCount",
			} {
				if !strings.Contains(artifacts.printed, required) {
					t.Fatalf("defined scalar reflection artifact lacks %q", required)
				}
			}
		},
	)
}
