package reflectvalue_test

import "testing"

// TestReflectConstructorsAndSetsMatchGo proves the constructor and scalar
// mutation family: Zero and New construction, PointerTo and SliceOf
// descriptor composition, Convert between scalar kinds, typed scalar Sets
// through addressable locations, SetZero, CanInt classification, and Addr
// round-tripping back to the located value.
func TestReflectConstructorsAndSetsCanonicalizeWithNativeEvidence(t *testing.T) {
	source := `package reflectvalue

import (
	"fmt"
	"reflect"
)

func Build() string {
	zero := reflect.Zero(reflect.TypeOf(int64(0)))
	fresh := reflect.New(reflect.TypeOf(""))
	fresh.Elem().SetString("built")
	var count int
	holder := reflect.ValueOf(&count).Elem()
	holder.SetInt(41)
	holder.SetZero()
	holder.SetInt(7)
	var ratio float64
	reflect.ValueOf(&ratio).Elem().SetFloat(2.5)
	var flag bool
	reflect.ValueOf(&flag).Elem().SetBool(true)
	var wide uint16
	reflect.ValueOf(&wide).Elem().SetUint(9)
	converted := reflect.ValueOf(int32(66)).Convert(reflect.TypeOf(int64(0)))
	text := reflect.ValueOf(65).Convert(reflect.TypeOf(""))
	pointerType := reflect.PointerTo(reflect.TypeOf(0))
	sliceType := reflect.SliceOf(reflect.TypeOf(0))
	address := holder.Addr()
	address.Elem().SetInt(12)
	return fmt.Sprintf(
		"%d %t %q %d %g %t %d %d %q %s %s %t %t %d %t",
		zero.Int(), zero.IsZero(),
		fresh.Elem().String(),
		count, ratio, flag, int(wide),
		converted.Int(), text.String(),
		pointerType.Kind().String(), sliceType.Kind().String(),
		holder.CanInt(), zero.CanInt(),
		count, text.CanInt(),
	)
}
`
	typescriptRunner := `const facts = Build();
console.log(facts);
`
	goRunner := `package main

import (
	"fmt"

	fixture "example.com/reflectvalue"
)

func main() {
	fmt.Println(fixture.Build())
}
`
	verifyReflectCanonical(
		t,
		source,
		"Build",
		"reflectvalue",
		typescriptRunner,
		goRunner,
	)
}
