package reflectvalue_test

import (
	"strings"
	"testing"
)

func TestReflectNewDefinedScalarLocationCanonicalizesWithNativeEvidence(t *testing.T) {
	source := `package reflectvalue

import (
	"fmt"
	"reflect"
)

type DefinedText string
type DefinedCount uint32
type definedHolder struct {
	count DefinedCount
}

func storeDefinedCount(target *DefinedCount, value DefinedCount) {
	*target = value
}

func DefinedScalarFacts() string {
	text := reflect.New(reflect.TypeOf(DefinedText("")))
	text.Elem().SetString("named")
	count := reflect.New(reflect.TypeOf(DefinedCount(0)))
	count.Elem().SetUint(37)
	holder := definedHolder{}
	storeDefinedCount(&holder.count, DefinedCount(41))
	return fmt.Sprintf("%s %d %d", text.Elem().Interface().(DefinedText), count.Elem().Interface().(DefinedCount), holder.count)
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
	verifyReflectCanonicalInspect(
		t,
		source,
		"DefinedScalarFacts",
		"reflectvalue",
		typescriptRunner,
		goRunner,
		func(artifacts renderedArtifacts) {
			for _, required := range []string{
				"allocatePointer(new DefinedText",
				"loadPointer(instance)",
				"allocatePointer(0 * DefinedCount__from_reflectvalue.$goType)",
			} {
				if !strings.Contains(artifacts.printed, required) {
					t.Fatalf(
						"defined scalar reflection artifact lacks %q:\n%s",
						required,
						artifacts.printed,
					)
				}
			}
			if strings.Contains(
				artifacts.printed,
				"new DefinedCount__from_reflectvalue(",
			) {
				t.Fatal("native numeric reflection artifact restored a wrapper")
			}
		},
	)
}
