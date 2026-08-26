package reflectvalue_test

import (
	"strings"
	"testing"
)

func TestReflectInterfaceSliceCanonicalizesWithNativeEvidence(t *testing.T) {
	source := `package reflectvalue

import (
	"fmt"
	"reflect"
)

type Named interface {
	Name() string
}

type Item struct {
	Text string
}

func (item Item) Name() string {
	return item.Text
}

func InterfaceSliceFacts() string {
	values := []Named{Item{Text: "first"}, nil}
	view := reflect.ValueOf(&values).Elem()
	first := view.Index(0)
	nilElement := view.Index(1)

	replacement := Item{Text: "second"}
	first.Set(reflect.ValueOf(replacement))
	replacement.Text = "changed"

	appended := reflect.Append(
		view,
		reflect.ValueOf(Item{Text: "third"}),
		reflect.Zero(view.Type().Elem()),
	)
	made := reflect.MakeSlice(view.Type(), 1, 3)
	made.Index(0).Set(reflect.ValueOf(Item{Text: "fourth"}))

	return fmt.Sprintf(
		"%s/%s/%t/%t %s/%s/%t %s",
		first.Kind().String(),
		first.Elem().Kind().String(),
		nilElement.IsNil(),
		nilElement.IsZero(),
		values[0].Name(),
		appended.Index(2).Interface().(Named).Name(),
		appended.Index(3).IsNil(),
		made.Index(0).Interface().(Named).Name(),
	)
}
`
	typescriptRunner := `const facts = InterfaceSliceFacts();
console.log(facts);
`
	goRunner := `package main

import (
	"fmt"

	fixture "example.com/reflectvalue"
)

func main() {
	fmt.Println(fixture.InterfaceSliceFacts())
}
`
	verifyReflectCanonicalInspect(
		t,
		source,
		"InterfaceSliceFacts",
		"reflectvalue",
		typescriptRunner,
		goRunner,
		func(artifacts renderedArtifacts) {
			for _, required := range []string{
				"value === undefined",
				"outside the interface contract",
				"GoInterfaceValue | undefined",
				"RuntimeSlice.make<",
			} {
				if !strings.Contains(artifacts.printed, required) {
					t.Fatalf("interface-slice reflection artifact lacks %q", required)
				}
			}
		},
	)
}
