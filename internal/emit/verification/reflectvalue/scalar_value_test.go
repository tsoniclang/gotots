package reflectvalue_test

import (
	"strings"
	"testing"
)

// TestReflectValueScalarReadsMatchGo proves the canonical reflection value
// model over scalar kinds: ValueOf, Kind, Type identity, scalar reads,
// validity, and the invalid-versus-typed-nil distinction all match Go
// exactly through the generated value-operation metadata.
func TestReflectValueScalarReadsCanonicalizeWithNativeEvidence(t *testing.T) {
	source := `package reflectvalue

import (
	"fmt"
	"reflect"
)

func Facts() string {
	number := reflect.ValueOf(42)
	unsigned := reflect.ValueOf(uint16(7))
	float := reflect.ValueOf(3.5)
	flag := reflect.ValueOf(true)
	text := reflect.ValueOf("go")
	var nothing any
	invalid := reflect.ValueOf(nothing)
	var nilPointer *int
	typedNil := reflect.ValueOf(nilPointer)
	pointer := reflect.ValueOf(new(int))
	return fmt.Sprintf(
		"%s %d %s %d %s %g %s %t %s %q %t %t %t %s %t %t %t",
		number.Kind().String(), number.Int(),
		unsigned.Kind().String(), unsigned.Uint(),
		float.Kind().String(), float.Float(),
		flag.Kind().String(), flag.Bool(),
		text.Kind().String(), text.String(),
		number.IsValid(), invalid.IsValid(), typedNil.IsValid(),
		typedNil.Kind().String(), typedNil.IsNil(), pointer.IsNil(),
		number.Type() == reflect.TypeOf(7),
	)
}
`
	typescriptRunner := `const facts = Facts();
console.log(facts);
`
	goRunner := `package main

import (
	"fmt"

	fixture "example.com/reflectvalue"
)

func main() {
	fmt.Println(fixture.Facts())
}
`
	verifyReflectCanonical(
		t,
		source,
		"Facts",
		"reflectvalue",
		typescriptRunner,
		goRunner,
	)
}

func TestReflectRawPointerCanonicalizesStorageAndNamedValues(test *testing.T) {
	source := `package reflectvalue

import (
  "reflect"
  "unsafe"
)

type Raw unsafe.Pointer

func Check() bool {
  var value uint32 = 7
  pointer := &value
  reflected := reflect.ValueOf(pointer).UnsafePointer()
  *(*uint32)(reflected) = 42
  direct := unsafe.Pointer(pointer)
  var missing *uint32
  var rawMissing unsafe.Pointer
  named := Raw(direct)
  return reflected == direct && value == 42 &&
    reflect.ValueOf(named).UnsafePointer() == direct &&
    reflect.ValueOf(direct).UnsafePointer() == direct &&
    reflect.ValueOf(missing).UnsafePointer() == nil &&
    reflect.ValueOf(rawMissing).UnsafePointer() == nil
}

`
	verifyReflectCanonicalInspect(test, source, "Check", "reflectvalue",
		`console.log(Check());`,
		`package main
import ("fmt"; fixture "example.com/reflectvalue")
func main() { fmt.Println(fixture.Check()) }
`, func(artifacts renderedArtifacts) {
			if !strings.Contains(artifacts.printed, "toRawPointer<uint32>") || !strings.Contains(artifacts.printed, "unsafePointer:") {
				test.Fatal("reflection did not emit its canonical storage callback")
			}
			if strings.Contains(artifacts.printed, "ProviderRawPointer") {
				test.Fatal("retired private raw-pointer carrier survived")
			}
		})
}
