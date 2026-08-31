package reflectvalue_test

import (
	"strings"
	"testing"
)

func TestReflectInterfaceFieldLocationCanonicalizesWithNativeEvidence(t *testing.T) {
	source := `package reflectvalue

import (
	"fmt"
	"reflect"
)

type Named interface {
	Name() string
}

type Label struct {
	Text string
}

func (label Label) Name() string {
	return label.Text
}

type Record struct {
	Value Named
}

type GenericRecord[T any] struct {
	Value Named
	Marker T
}

type Source struct {
	Value Label
}

func wrongValueRejected(field reflect.Value) (rejected bool) {
	defer func() {
		rejected = recover() != nil
	}()
	field.Set(reflect.ValueOf(1))
	return false
}

func InterfaceFieldFacts() string {
	record := &Record{}
	field := reflect.ValueOf(record).Elem().Field(0)
	before := fmt.Sprintf(
		"%t %s %t %t %t",
		field.IsValid(),
		field.Kind().String(),
		field.IsNil(),
		field.IsZero(),
		field.CanSet(),
	)

	source := &Source{Value: Label{Text: "first"}}
	field.Set(reflect.ValueOf(source).Elem().Field(0))
	source.Value.Text = "changed"
	after := fmt.Sprintf(
		"%s %t %t %s %s %t",
		field.Kind().String(),
		field.IsNil(),
		field.IsZero(),
		field.Elem().Kind().String(),
		field.Interface().(Named).Name(),
		wrongValueRejected(field),
	)

	field.Set(reflect.Zero(field.Type()))
	final := fmt.Sprintf(
		"%t %t %t",
		field.IsValid(),
		field.IsNil(),
		field.IsZero(),
	)

	generic := &GenericRecord[int]{}
	genericField := reflect.ValueOf(generic).Elem().Field(0)
	genericField.Set(reflect.ValueOf(source).Elem().Field(0))
	genericResult := fmt.Sprintf(
		"%t %s",
		genericField.CanSet(),
		generic.Value.Name(),
	)
	return before + " | " + after + " | " + final + " | " + genericResult
}
`
	typescriptRunner := `const facts = InterfaceFieldFacts();
console.log(facts);
`
	goRunner := `package main

import (
	"fmt"

	fixture "example.com/reflectvalue"
)

func main() {
	fmt.Println(fixture.InterfaceFieldFacts())
}
`
	verifyReflectCanonicalInspect(
		t,
		source,
		"InterfaceFieldFacts",
		"reflectvalue",
		typescriptRunner,
		goRunner,
		func(artifacts renderedArtifacts) {
			for _, required := range []string{
				"fields.interfaceValue(",
				"value === undefined",
				"zero: (): GoInterfaceValue | undefined => undefined",
				".$storageOf(instance).Value",
			} {
				if !strings.Contains(artifacts.printed, required) {
					t.Fatalf("interface-field reflection artifact lacks %q", required)
				}
			}
		},
	)
}

func TestReflectProviderInterfaceFieldCanonicalizesWithNativeEvidence(t *testing.T) {
	source := `package reflectvalue

import (
	"fmt"
	"reflect"
)

type TypeHolder struct {
	Value reflect.Type
}

func ProviderInterfaceFieldFacts() string {
	holder := &TypeHolder{Value: reflect.TypeOf(int32(0))}
	field := reflect.ValueOf(holder).Elem().Field(0)
	before := field.Interface().(reflect.Type).String()
	field.Set(reflect.ValueOf(reflect.TypeOf("")))
	return fmt.Sprintf("%s %s", before, holder.Value.String())
}
`
	typescriptRunner := `const facts = ProviderInterfaceFieldFacts();
console.log(facts);
`
	goRunner := `package main

import (
	"fmt"

	fixture "example.com/reflectvalue"
)

func main() {
	fmt.Println(fixture.ProviderInterfaceFieldFacts())
}
`
	verifyReflectCanonicalInspect(
		t,
		source,
		"ProviderInterfaceFieldFacts",
		"reflectvalue",
		typescriptRunner,
		goRunner,
		func(artifacts renderedArtifacts) {
			if !strings.Contains(artifacts.printed, "$candidate is reflect.Type") {
				t.Fatal("sealed provider interface field lacks its typed admission guard")
			}
		},
	)
}

func TestReflectPointerElementKindsCanonicalizeWithNativeEvidence(t *testing.T) {
	source := `package reflectvalue

import (
	"fmt"
	"reflect"
)

func PointerElementFacts() string {
	var slot any = "before"
	interfaceElement := reflect.ValueOf(&slot).Elem()
	before := interfaceElement.Elem().String()
	interfaceElement.Set(reflect.ValueOf(int64(41)))

	value := 7
	pointer := &value
	pointerElement := reflect.ValueOf(&pointer).Elem()
	pointerElement.Elem().SetInt(9)

	array := [2]int{1, 2}
	arrayElement := reflect.ValueOf(&array).Elem()
	replacementArray := [2]int{4, 5}
	arrayElement.Set(reflect.ValueOf(replacementArray))
	replacementArray[0] = 99

	values := []int{1}
	sliceElement := reflect.ValueOf(&values).Elem()
	sliceElement.Set(reflect.ValueOf([]int{2, 3}))

	entries := map[string]int{"before": 1}
	mapElement := reflect.ValueOf(&entries).Elem()
	mapElement.Set(reflect.ValueOf(map[string]int{"after": 2}))

	function := func() int { return 3 }
	functionElement := reflect.ValueOf(&function).Elem()
	functionElement.Set(reflect.ValueOf(func() int { return 5 }))

	channel := make(chan int, 1)
	replacementChannel := make(chan int, 1)
	replacementChannel <- 6
	channelElement := reflect.ValueOf(&channel).Elem()
	channelElement.Set(reflect.ValueOf(replacementChannel))

	anonymous := struct{ Count int }{Count: 1}
	anonymousElement := reflect.ValueOf(&anonymous).Elem()
	anonymousElement.Field(0).SetInt(4)

	var nilSlot any
	nilElement := reflect.ValueOf(&nilSlot).Elem()
	var absent *int
	invalidNilPointer := !reflect.ValueOf(absent).Elem().IsValid()
	return fmt.Sprintf(
		"%s %d %d %v %v %d %d %d %d %t %t %t %t",
		before, slot.(int64), value, array, values, entries["after"],
		function(), <-channel, anonymous.Count, interfaceElement.CanSet(),
		nilElement.IsNil(), nilElement.IsZero(), invalidNilPointer,
	)
}
`
	typescriptRunner := `console.log(PointerElementFacts());
`
	goRunner := `package main

import (
	"fmt"

	fixture "example.com/reflectvalue"
)

func main() {
	fmt.Println(fixture.PointerElementFacts())
}
`
	verifyReflectCanonicalInspect(
		t,
		source,
		"PointerElementFacts",
		"reflectvalue",
		typescriptRunner,
		goRunner,
		func(artifacts renderedArtifacts) {
			for _, required := range []string{
				"$goReflectType$PointerTo_Interface_void",
				"$goReflectType$PointerTo_PointerTo_int",
				"$goReflectType$PointerTo_Array2Of_int",
				"$goReflectType$PointerTo_SliceOf_int",
				"$goReflectType$PointerTo_MapOf_string_To_int",
				"$goReflectType$PointerTo_void_to_int",
				"$goReflectType$PointerTo_ChannelOf_int",
				"loadPointer(pointer)",
				"storePointer(pointer",
			} {
				if !strings.Contains(artifacts.printed, required) {
					t.Fatalf("pointer-element artifact lacks %q", required)
				}
			}
		},
	)
}
