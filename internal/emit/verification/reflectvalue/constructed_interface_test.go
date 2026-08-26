package reflectvalue_test

import (
	"strings"
	"testing"
)

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
