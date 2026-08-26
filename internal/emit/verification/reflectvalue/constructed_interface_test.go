package reflectvalue_test

import (
	"strings"
	"testing"
)

// TestReflectDeepEqualCanonicalizesWithNativeEvidence covers deep comparison and projection
// tail of the reflection family: DeepEqual over structs, slices, maps,
// pointers, distinct types, and nil; Interface returning an exact copy of
// a located struct; and Indirect unwrapping pointers.
func TestReflectDeepEqualCanonicalizesWithNativeEvidence(t *testing.T) {
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

func Compare() string {
	left := Options{Name: "strict", Count: 3}
	right := Options{Name: "strict", Count: 3}
	different := Options{Name: "loose", Count: 3}
	leftSlice := []int{1, 2, 3}
	rightSlice := []int{1, 2, 3}
	shorter := []int{1, 2}
	leftMap := map[string]int{"a": 1, "b": 2}
	rightMap := map[string]int{"b": 2, "a": 1}
	missing := map[string]int{"a": 1}
	source := &Options{Name: "view", Count: 9}
	view := reflect.ValueOf(source).Elem()
	snapshot := view.Interface().(Options)
	source.Name = "mutated"
	indirect := reflect.Indirect(reflect.ValueOf(&left))
	return fmt.Sprintf(
		"%t %t %t %t %t %t %t %t %t %q %s %d",
		reflect.DeepEqual(left, right),
		reflect.DeepEqual(left, different),
		reflect.DeepEqual(leftSlice, rightSlice),
		reflect.DeepEqual(leftSlice, shorter),
		reflect.DeepEqual(leftMap, rightMap),
		reflect.DeepEqual(leftMap, missing),
		reflect.DeepEqual(&left, &right),
		reflect.DeepEqual(left, leftSlice),
		reflect.DeepEqual(nil, nil),
		snapshot.Name,
		indirect.Kind().String(),
		int(indirect.Field(1).Int()),
	)
}
`
	typescriptRunner := `const facts = Compare();
console.log(facts);
`
	goRunner := `package main

import (
	"fmt"

	fixture "example.com/reflectvalue"
)

func main() {
	fmt.Println(fixture.Compare())
}
`
	verifyReflectCanonical(
		t,
		source,
		"Compare",
		"reflectvalue",
		typescriptRunner,
		goRunner,
	)
}

// TestReflectTypeAssertCanonicalizesWithNativeEvidence covers reflect.TypeAssert over concrete
// scalar, string, and pointer type arguments with both hit and miss
// outcomes, matching the Go comma-ok semantics exactly.
func TestReflectTypeAssertCanonicalizesWithNativeEvidence(t *testing.T) {
	source := `package reflectvalue

import (
	"fmt"
	"reflect"
)

func Assert() string {
	number := reflect.ValueOf(42)
	text := reflect.ValueOf("go")
	pointer := reflect.ValueOf(new(int))
	gotNumber, numberOK := reflect.TypeAssert[int](number)
	missText, missOK := reflect.TypeAssert[string](number)
	gotText, textOK := reflect.TypeAssert[string](text)
	gotPointer, pointerOK := reflect.TypeAssert[*int](pointer)
	missPointer, missPointerOK := reflect.TypeAssert[*int](text)
	return fmt.Sprintf(
		"%d %t %q %t %q %t %t %t %t %t",
		gotNumber, numberOK,
		missText, missOK,
		gotText, textOK,
		gotPointer != nil, pointerOK,
		missPointer == nil, missPointerOK,
	)
}
`
	typescriptRunner := `const facts = Assert();
console.log(facts);
`
	goRunner := `package main

import (
	"fmt"

	fixture "example.com/reflectvalue"
)

func main() {
	fmt.Println(fixture.Assert())
}
`
	verifyReflectCanonical(
		t,
		source,
		"Assert",
		"reflectvalue",
		typescriptRunner,
		goRunner,
	)
}

func TestReflectConstructedValuesExposeExactInterfaceContracts(t *testing.T) {
	source := `package reflectvalue

import (
	"fmt"
	"reflect"
)

type Label string

func (label Label) Text() string {
	return string(label)
}

func (label Label) Unused() string {
	return "unused"
}

func (label *Label) Decode(text []byte) error {
	*label = Label(string(text))
	return nil
}

type Text interface {
	Text() string
}

type Decoder interface {
	Decode([]byte) error
}

func Constructed() string {
	zero := reflect.Zero(reflect.TypeOf(Label("")))
	text, textOK := reflect.TypeAssert[Text](zero)
	fresh := reflect.New(reflect.TypeOf(Label("")))
	decoder, decoderOK := reflect.TypeAssert[Decoder](fresh)
	_ = decoder.Decode([]byte("built"))
	return fmt.Sprintf(
		"%t %q %t %q",
		textOK, text.Text(),
		decoderOK, fresh.Elem().Interface().(Label),
	)
}
`
	verifyReflectCanonicalInspect(
		t,
		source,
		"Constructed",
		"reflectvalue",
		"console.log(Constructed());\n",
		`package main

import (
	"fmt"

	fixture "example.com/reflectvalue"
)

func main() {
	fmt.Println(fixture.Constructed())
}
`,
		func(artifacts renderedArtifacts) {
			for _, expectation := range []struct {
				adapter   string
				method    string
				forbidden string
			}{
				{
					adapter: "$goInterfaceAdapter$Named_reflectvalue$Label",
					method:  "Text(",
				},
				{
					adapter:   "$goInterfaceAdapter$PointerTo_Named_reflectvalue$Label",
					method:    "Decode(",
					forbidden: "Unused(",
				},
			} {
				start := strings.Index(
					artifacts.printed,
					"export class "+expectation.adapter,
				)
				if start < 0 {
					start = strings.Index(
						artifacts.printed,
						"export const "+expectation.adapter,
					)
				}
				if start < 0 {
					t.Fatalf("constructed adapter %q is absent", expectation.adapter)
				}
				end := len(artifacts.printed)
				if relative := strings.Index(
					artifacts.printed[start+1:],
					"\nexport ",
				); relative >= 0 {
					end = start + 1 + relative
				}
				if !strings.Contains(
					artifacts.printed[start:end],
					expectation.method,
				) {
					t.Fatalf(
						"constructed adapter %q lacks %q",
						expectation.adapter,
						expectation.method,
					)
				}
				if expectation.forbidden != "" && strings.Contains(
					artifacts.printed[start:end],
					expectation.forbidden,
				) {
					t.Fatalf(
						"constructed adapter %q contains unasserted %q",
						expectation.adapter,
						expectation.forbidden,
					)
				}
			}
		},
	)
}
