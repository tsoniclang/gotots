package reflectvalue_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestReflectSliceOperationsMatchGo proves the slice value family: Len,
// Cap, Index element locations aliasing the original backing array,
// element mutation through Set, element zero evidence, Append growth, and
// MakeSlice construction all match Go exactly.
func TestReflectSliceOperationsCanonicalizeWithNativeEvidence(t *testing.T) {
	source := `package reflectvalue

import (
	"fmt"
	"reflect"
)

func Checksum() string {
	values := []int{4, 0, 15}
	view := reflect.ValueOf(values)
	total := 0
	zeros := 0
	for index := 0; index < view.Len(); index++ {
		element := view.Index(index)
		if element.IsZero() {
			zeros++
			continue
		}
		total += int(element.Int())
	}
	view.Index(1).Set(reflect.ValueOf(23))
	grown := reflect.Append(view, reflect.ValueOf(8), reflect.ValueOf(16))
	made := reflect.MakeSlice(reflect.TypeOf(values), 2, 5)
	return fmt.Sprintf(
		"%d %d %d %d %d %d %t %d %d %d %d %d %t %t",
		total, zeros,
		values[0], values[1], values[2],
		view.Cap(),
		view.Index(0).CanSet(),
		grown.Len(), grown.Cap(),
		int(grown.Index(4).Int()),
		made.Len(), made.Cap(),
		made.Index(1).IsZero(),
		grown.Index(0).CanSet(),
	)
}

`
	typescriptRunner := `const facts = Checksum();
console.log(facts);
`
	goRunner := `package main

import (
	"fmt"

	fixture "example.com/reflectvalue"
)

func main() {
	fmt.Println(fixture.Checksum())
}
`
	verifyReflectCanonical(
		t,
		source,
		"Checksum",
		"reflectvalue",
		typescriptRunner,
		goRunner,
	)
}

func TestReflectPointerAndAggregateContainersUseCompleteDescriptors(t *testing.T) {
	source := `package reflectvalue

import (
	"fmt"
	"reflect"
)

type Cell struct {
	Count int
}

type Key struct {
	ID int
}

func ContainerFacts() string {
	first := "first"
	second := "second"
	pointers := []*string{&first}
	pointerView := reflect.ValueOf(&pointers).Elem()
	appendedPointers := reflect.Append(pointerView, reflect.ValueOf(&second))
	madePointers := reflect.MakeSlice(reflect.TypeOf(pointers), 2, 4)
	madePointers.Index(0).Set(reflect.ValueOf(&first))
	pointerView.Grow(4)

	cells := []Cell{{Count: 1}, {Count: 2}}
	cellView := reflect.ValueOf(&cells).Elem()
	replacement := Cell{Count: 3}
	cellView.Index(0).Set(reflect.ValueOf(replacement))
	replacement.Count = 30
	appendedCells := reflect.Append(cellView, reflect.ValueOf(Cell{Count: 4}))
	madeCells := reflect.MakeSlice(reflect.TypeOf(cells), 2, 5)
	madeCells.Index(0).Set(reflect.ValueOf(Cell{Count: 5}))
	madeCells.Index(1).Field(0).SetInt(6)

	cellMap := map[Key]Cell{}
	mapView := reflect.ValueOf(cellMap)
	mapSource := Cell{Count: 7}
	mapView.SetMapIndex(reflect.ValueOf(Key{ID: 1}), reflect.ValueOf(mapSource))
	mapSource.Count = 70
	madeMap := reflect.MakeMap(reflect.TypeOf(cellMap))
	madeMap.SetMapIndex(
		reflect.ValueOf(Key{ID: 2}),
		reflect.ValueOf(Cell{Count: 8}),
	)

	pointerMap := map[string]*string{}
	pointerMapView := reflect.ValueOf(pointerMap)
	pointerMapView.SetMapIndex(reflect.ValueOf("value"), reflect.ValueOf(&second))

	return fmt.Sprintf(
		"%d/%d/%s/%t %d/%d/%d/%d %d/%d/%s",
		appendedPointers.Len(),
		madePointers.Cap(),
		*madePointers.Index(0).Interface().(*string),
		madePointers.Index(1).IsNil(),
		cells[0].Count,
		appendedCells.Index(2).Interface().(Cell).Count,
		madeCells.Index(0).Interface().(Cell).Count,
		madeCells.Index(1).Interface().(Cell).Count,
		mapView.MapIndex(reflect.ValueOf(Key{ID: 1})).Interface().(Cell).Count,
		madeMap.MapIndex(reflect.ValueOf(Key{ID: 2})).Interface().(Cell).Count,
		*pointerMapView.MapIndex(reflect.ValueOf("value")).Interface().(*string),
	)
}
`
	typescriptRunner := `const facts = ContainerFacts();
console.log(facts);
`
	goRunner := `package main

import (
	"fmt"

	fixture "example.com/reflectvalue"
)

func main() {
	fmt.Println(fixture.ContainerFacts())
}
`
	verifyReflectCanonicalInspect(
		t,
		source,
		"ContainerFacts",
		"reflectvalue",
		typescriptRunner,
		goRunner,
		func(artifacts renderedArtifacts) {
			for _, required := range []string{
				"const incoming = values.map(",
				".$grownCapacity(",
				".$copy(",
				"mapIndex:",
				"mapStore:",
				"RuntimeSlice.make<",
			} {
				if !strings.Contains(artifacts.printed, required) {
					t.Fatalf("container reflection artifact lacks %q", required)
				}
			}
		},
	)
}

func TestReflectContainerDescriptorsHaveNoScalarOnlyRoute(t *testing.T) {
	root := filepath.Join(
		repositoryRoot(),
		"internal",
		"emit",
		"declaration",
		"reflectiontype",
	)
	for _, name := range []string{"value_slices.go", "value_maps.go"} {
		content, err := os.ReadFile(filepath.Join(root, name))
		if err != nil {
			t.Fatal(err)
		}
		for _, forbidden := range []string{
			"types.IsBoolean",
			"types.IsString",
			"types.IsInteger",
			"types.IsFloat",
			"reducedSliceProperties",
			"scalarZeroExpression",
		} {
			if strings.Contains(string(content), forbidden) {
				t.Fatalf("%s retains scalar-only reflection route %q", name, forbidden)
			}
		}
	}
}
