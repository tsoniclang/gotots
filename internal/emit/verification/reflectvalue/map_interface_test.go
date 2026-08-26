package reflectvalue_test

import (
	"strings"
	"testing"
)

func TestReflectInterfaceMapCanonicalizesWithNativeEvidence(t *testing.T) {
	source := `package reflectvalue

import (
	"fmt"
	"reflect"
)

type Key interface {
	Code() int
}

type Number int

func (number Number) Code() int {
	return int(number)
}

type Named interface {
	Name() string
}

type Item struct {
	Text string
}

func (item Item) Name() string {
	return item.Text
}

func InterfaceMapFacts() string {
	keys := map[Key]int{Number(1): 10, nil: 20}
	keyView := reflect.ValueOf(keys)
	nilKey := reflect.Zero(keyView.Type().Key())
	keyView.SetMapIndex(nilKey, reflect.ValueOf(21))
	keyView.SetMapIndex(reflect.ValueOf(Number(2)), reflect.ValueOf(30))
	var deleteValue reflect.Value
	keyView.SetMapIndex(reflect.ValueOf(Number(1)), deleteValue)

	keyTotal := 0
	nilKeySeen := false
	keyIterator := keyView.MapRange()
	for keyIterator.Next() {
		key := keyIterator.Key()
		if key.IsNil() {
			nilKeySeen = true
		} else {
			keyTotal += key.Interface().(Key).Code()
		}
		keyTotal += int(keyIterator.Value().Int())
	}

	values := map[string]Named{
		"item": Item{Text: "first"},
		"nil":  nil,
	}
	valueView := reflect.ValueOf(values)
	valueView.SetMapIndex(
		reflect.ValueOf("second"),
		reflect.ValueOf(Item{Text: "second"}),
	)
	valueView.SetMapIndex(
		reflect.ValueOf("nil"),
		reflect.Zero(valueView.Type().Elem()),
	)
	valueView.SetMapIndex(reflect.ValueOf("item"), deleteValue)

	nilValue := valueView.MapIndex(reflect.ValueOf("nil"))
	second := valueView.MapIndex(reflect.ValueOf("second"))
	nilValues := 0
	valueIterator := valueView.MapRange()
	for valueIterator.Next() {
		if valueIterator.Value().IsNil() {
			nilValues++
		}
	}

	return fmt.Sprintf(
		"%d/%t/%d/%t %d/%t/%t/%s/%d",
		keyTotal,
		nilKeySeen,
		keyView.Len(),
		keyView.MapIndex(nilKey).IsValid(),
		valueView.Len(),
		nilValue.IsValid(),
		nilValue.IsNil(),
		second.Interface().(Named).Name(),
		nilValues,
	)
}
`
	typescriptRunner := `const facts = InterfaceMapFacts();
console.log(facts);
`
	goRunner := `package main

import (
	"fmt"

	fixture "example.com/reflectvalue"
)

func main() {
	fmt.Println(fixture.InterfaceMapFacts())
}
`
	verifyReflectCanonicalInspect(
		t,
		source,
		"InterfaceMapFacts",
		"reflectvalue",
		typescriptRunner,
		goRunner,
		func(artifacts renderedArtifacts) {
			for _, required := range []string{
				"deleteEntry",
				"outside the interface contract",
			} {
				if !strings.Contains(artifacts.printed, required) {
					t.Fatalf("interface-map reflection artifact lacks %q", required)
				}
			}
		},
	)
}
