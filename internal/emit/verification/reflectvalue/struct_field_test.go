package reflectvalue_test

import (
	"regexp"
	"strings"
	"testing"
)

// TestReflectStructFieldMutationCanonicalizesWithNativeEvidence covers the addressable value
// location model: pointer Elem, struct NumField/Field, settability, field
// mutation through original storage, IsZero, and SetString all match Go
// exactly — the exact shape of the TS-Go compiler-options merge that first
// blocked the generated product.
func TestReflectStructFieldMutationCanonicalizesWithNativeEvidence(t *testing.T) {
	source := `package reflectvalue

import (
	"fmt"
	"reflect"
)

type Options struct {
	Name    string
	Count   int
	Verbose bool
}

func Merge() string {
	source := &Options{Name: "strict", Count: 3}
	target := &Options{Verbose: true}
	sourceValue := reflect.ValueOf(source).Elem()
	targetValue := reflect.ValueOf(target).Elem()
	moved := 0
	for index := 0; index < sourceValue.NumField(); index++ {
		field := sourceValue.Field(index)
		if field.IsZero() {
			continue
		}
		targetValue.Field(index).Set(field)
		moved++
	}
	targetValue.Field(0).SetString("renamed")
	rvalue := reflect.ValueOf(*source)
	return fmt.Sprintf(
		"%d %q %d %t %t %t %t %d",
		moved,
		target.Name, target.Count, target.Verbose,
		targetValue.CanSet(),
		targetValue.Field(0).CanSet(),
		rvalue.Field(0).CanSet(),
		rvalue.NumField(),
	)
}
`
	typescriptRunner := `const facts = Merge();
console.log(facts);
`
	goRunner := `package main

import (
	"fmt"

	fixture "example.com/reflectvalue"
)

func main() {
	fmt.Println(fixture.Merge())
}
`
	verifyReflectCanonicalInspect(
		t,
		source,
		"Merge",
		"reflectvalue",
		typescriptRunner,
		goRunner,
		func(artifacts renderedArtifacts) {
			if !strings.Contains(
				artifacts.printed,
				"ReflectTypeMetadataOperations.$registerStruct(",
			) {
				t.Fatalf(
					"struct reflection does not use the typed common owner (%d bytes)",
					len(artifacts.printed),
				)
			}
			if strings.Contains(
				artifacts.printed,
				"switch (index)",
			) {
				t.Fatalf(
					"struct reflection repeats per-type index dispatch (%d bytes)",
					len(artifacts.printed),
				)
			}
			if !regexp.MustCompile(
				`(?s)\.\$registerStruct\(\s*[^,]+,\s*\(\)\s*=>`,
			).MatchString(artifacts.printed) {
				t.Fatalf("struct reflection resolves its adapter eagerly")
			}
			if !regexp.MustCompile(
				`(?s)\.\$registerStruct\([^;]+,\s*fields\s*=>`,
			).MatchString(artifacts.printed) {
				t.Fatalf("struct reflection materializes field facts eagerly")
			}
			if fields := strings.Count(
				artifacts.printed,
				"fields.valueProperty(",
			); fields != 3 {
				t.Fatalf("direct reflected property facts = %d, want 3", fields)
			}
			if regexp.MustCompile(
				`(?s)fields\.value\([^;]+instance\s*=>\s*\([^)]*\.(?:Name|Count|Verbose)\)`,
			).MatchString(artifacts.printed) {
				t.Fatal("direct reflected fields retain generated getter callbacks")
			}
		},
	)
}

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
