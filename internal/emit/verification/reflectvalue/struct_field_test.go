package reflectvalue_test

import "testing"

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
	typescriptRunner := `const facts = await Merge();
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
	verifyReflectCanonical(
		t,
		source,
		"Merge",
		"reflectvalue",
		typescriptRunner,
		goRunner,
	)
}
