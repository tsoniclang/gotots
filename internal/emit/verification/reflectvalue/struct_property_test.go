package reflectvalue_test

import (
	"strings"
	"testing"
)

func TestReflectStructPropertyFactsPreserveValueCopies(t *testing.T) {
	source := `package reflectvalue

import (
	"fmt"
	"reflect"
)

type Child struct {
	Count int
}

type Outer struct {
	Child Child
	Total int
}

func PropertyFacts() string {
	source := &Outer{Child: Child{Count: 4}, Total: 5}
	target := &Outer{}
	sourceValue := reflect.ValueOf(source).Elem()
	targetValue := reflect.ValueOf(target).Elem()
	targetValue.Field(0).Set(sourceValue.Field(0))
	targetValue.Field(1).Set(sourceValue.Field(1))
	source.Child.Count = 9
	return fmt.Sprintf("%d %d", target.Child.Count, target.Total)
}
`
	typescriptRunner := `const facts = PropertyFacts();
console.log(facts);
`
	goRunner := `package main

import (
	"fmt"

	fixture "example.com/reflectvalue"
)

func main() {
	fmt.Println(fixture.PropertyFacts())
}
`
	verifyReflectCanonicalInspect(
		t,
		source,
		"PropertyFacts",
		"reflectvalue",
		typescriptRunner,
		goRunner,
		func(artifacts renderedArtifacts) {
			if !strings.Contains(
				artifacts.printed,
				"fields.copyingValueProperty(",
			) {
				t.Fatalf(
					"copying reflected property fact is absent:\n%s",
					artifacts.printed,
				)
			}
			if !strings.Contains(
				artifacts.printed,
				`"Child", value =>`,
			) {
				t.Fatal("copying reflected property does not retain its exact key")
			}
		},
	)
}
