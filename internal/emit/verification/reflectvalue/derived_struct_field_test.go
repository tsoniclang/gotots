package reflectvalue_test

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestReflectDerivedStructFieldUsesCanonicalStorage(t *testing.T) {
	source := `package reflectvalue

import (
	"fmt"
	"reflect"

	"example.com/reflectvalue/basis"
)

type Derived basis.Record

func DerivedFieldFacts() string {
	value := Derived(basis.Make(41))
	field := reflect.ValueOf(value).Field(0)
	return fmt.Sprintf("%s %d %t", field.Kind().String(), field.Int(), field.CanSet())
}
`
	typescriptRunner := `const facts = await DerivedFieldFacts();
console.log(facts);
`
	goRunner := `package main

import (
	"fmt"

	fixture "example.com/reflectvalue"
)

func main() {
	fmt.Println(fixture.DerivedFieldFacts())
}
`
	verifyReflectCanonicalProjectInspect(
		t,
		source,
		"DerivedFieldFacts",
		"reflectvalue",
		typescriptRunner,
		goRunner,
		func(project string) {
			writeProgramFile(
				t,
				filepath.Join(project, "basis", "basis.go"),
				`package basis

type Record struct {
	value int64
}

func Make(value int64) Record {
	return Record{value: value}
}
`,
			)
		},
		func(artifacts renderedArtifacts) {
			if !strings.Contains(artifacts.printed, ".$storageOf(instance).value") {
				t.Fatal("derived struct reflection did not use canonical field storage")
			}
		},
	)
}
