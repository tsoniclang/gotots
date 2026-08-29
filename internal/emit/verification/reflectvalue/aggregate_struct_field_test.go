package reflectvalue_test

import (
	"strings"
	"testing"
)

func TestReflectAggregateStructFieldCanonicalizesWithNativeEvidence(t *testing.T) {
	source := `package reflectvalue

import (
	"fmt"
	"reflect"
)

type Mode int

type Inner struct {
	Count int
}

type Options struct {
	_      struct{}
	Mode   Mode
	Names  []string
	Index  map[string]int
	Child  *Inner
	Inline Inner
}

func MergeAggregate() string {
	source := &Options{
		Mode:   2,
		Names:  []string{"before"},
		Index:  map[string]int{"key": 3},
		Child:  &Inner{Count: 4},
		Inline: Inner{Count: 5},
	}
	target := &Options{}
	sourceValue := reflect.ValueOf(source).Elem()
	targetValue := reflect.ValueOf(target).Elem()
	moved := 0
	for index := 0; index < sourceValue.NumField(); index++ {
		sourceField := sourceValue.Field(index)
		targetField := targetValue.Field(index)
		if sourceField.IsZero() || !targetField.CanSet() {
			continue
		}
		targetField.Set(sourceField)
		moved++
	}
	source.Names[0] = "after"
	source.Index["key"] = 9
	source.Child.Count = 10
	source.Inline.Count = 11
	whole := &Options{}
	reflect.ValueOf(whole).Elem().Set(targetValue)
	target.Mode = 7
	target.Names[0] = "shared"
	target.Inline.Count = 12
	return fmt.Sprintf(
		"%d %d %q %d %d %d %d %q %d",
		moved,
		target.Mode,
		target.Names[0],
		target.Index["key"],
		target.Child.Count,
		target.Inline.Count,
		whole.Mode,
		whole.Names[0],
		whole.Inline.Count,
	)
}
`
	typescriptRunner := `const facts = MergeAggregate();
console.log(facts);
`
	goRunner := `package main

import (
	"fmt"

	fixture "example.com/reflectvalue"
)

func main() {
	fmt.Println(fixture.MergeAggregate())
}
`
	verifyReflectCanonicalInspect(
		t,
		source,
		"MergeAggregate",
		"reflectvalue",
		typescriptRunner,
		goRunner,
		func(artifacts renderedArtifacts) {
			for _, required := range []string{
				"elements.value(",
				".$copy(",
				"storePointer(pointer,",
				"fields.readonlyValue(",
			} {
				if !strings.Contains(artifacts.printed, required) {
					t.Fatalf("aggregate reflection artifact lacks %q", required)
				}
			}
			if strings.Contains(
				artifacts.printed,
				"outside the generated location model",
			) {
				t.Fatal("aggregate reflection artifact kept the old boundary")
			}
		},
	)
}
