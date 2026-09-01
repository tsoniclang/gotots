package pointer_test

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tsoniclang/gotots/internal/load"
)

func TestDirectNamedStructPointersAndGenericFacetsMatchGo(
	t *testing.T,
) {
	source := `package pointerarchitecture

type Box struct { Value int32 }

func replace[T any](pointer *T, value T) T {
	previous := *pointer
	*pointer = value
	return previous
}

func NewBox(value int32) *Box {
	return &Box{Value: value}
}

func overwriteBox(pointer *Box, value Box) {
	*pointer = value
}

func Direct(value int32) (int32, int32, bool) {
	box := Box{Value: value}
	pointer := &box
	alias := pointer
	previous := replace(pointer, Box{Value: value + 2})
	overwriteBox(pointer, Box{Value: pointer.Value + 1})
	return previous.Value, pointer.Value, pointer == alias
}

func DirectMap(value int32) (int32, bool) {
	pointer := &Box{Value: value}
	alias := pointer
	values := map[*Box]int32{pointer: 1}
	return values[alias], pointer == alias
}

func Boolean(value bool) (bool, bool) {
	pointer := &value
	previous := replace(pointer, !value)
	return previous, value
}
`
	typescript := compilePointerArchitecture(
		t,
		map[string]string{"source.go": source},
	)
	for _, required := range []string{
		"function NewBox(value: int32): Pointer<Box> | undefined",
		"allocatePointer<Box>(new Box(value))",
		"storePointer((pointer ?? GoPanic.raiseRuntime",
		"pointer: Pointer<T> | undefined",
		"export class Box",
	} {
		if !strings.Contains(typescript, required) {
			t.Fatalf("direct pointer output lacks %q:\n%s", required, typescript)
		}
	}
	for _, forbidden := range []string{
		"GoPointer<Box",
		"GoPointer.cell<Box",
		"type Box$Storage",
		"Box.$storageOf(",
		"goPointerHash",
		": any",
		" as any",
		"unknown",
	} {
		if strings.Contains(typescript, forbidden) {
			t.Fatalf("direct pointer output contains %q:\n%s", forbidden, typescript)
		}
	}
}

func TestStableNamedStructLocationsPreserveCanonicalStorage(
	t *testing.T,
) {
	source := `package pointerarchitecture

type Child struct { Value int32 }
type Parent struct { Child Child }

var Global = Parent{Child: Child{Value: 1}}

func Nested(value int32) (int32, bool) {
	parent := Parent{Child: Child{Value: value}}
	pointer := &parent.Child
	parent = Parent{Child: Child{Value: value + 1}}
	pointer.Value++
	return parent.Child.Value, pointer == &parent.Child
}

func Field(value int32) (int32, bool) {
	parent := Parent{Child: Child{Value: value}}
	pointer := &parent.Child
	parent.Child = Child{Value: value + 2}
	pointer.Value++
	return parent.Child.Value, pointer == &parent.Child
}

func Package(value int32) (int32, bool) {
	pointer := &Global.Child
	Global = Parent{Child: Child{Value: value}}
	pointer.Value++
	return Global.Child.Value, pointer == &Global.Child
}
`
	typescript := compilePointerArchitecture(
		t,
		map[string]string{"source.go": source},
	)
	for _, required := range []string{
		"let pointer: Pointer<Child> | undefined",
		"projectPointer<Child$Storage, Child>(addressOf<Child$Storage>(storeTarget.Child)",
		"loadPointer<Child>",
		"equalPointer<Child>",
	} {
		if !strings.Contains(typescript, required) {
			t.Fatalf("stable struct-pointer output lacks %q:\n%s", required, typescript)
		}
	}
	for _, forbidden := range []string{
		"GoPointer<Child",
		"GoPointer.cell<Child",
		": any",
		" as any",
		"unknown",
	} {
		if strings.Contains(typescript, forbidden) {
			t.Fatalf("stable struct-pointer output contains %q:\n%s", forbidden, typescript)
		}
	}
}

func TestCarrierTransitionParameterizedIdentityAndSliceAliasMatchGo(
	t *testing.T,
) {
	source := `package pointerarchitecture

type Box[T any] struct { Value T }
type Left struct { Value int32 ` + "`json:\"left\"`" + ` }
type Right struct { Value int32 ` + "`json:\"right\"`" + ` }

func replace[T any](pointer *T, value T) T {
	previous := *pointer
	*pointer = value
	return previous
}

func DirectInt(value int32) *Box[int32] {
	return &Box[int32]{Value: value}
}

func DirectIntValue(value int32) int32 {
	return DirectInt(value).Value
}

func SliceAlias(value int32) (int32, int32, bool) {
	values := []Box[int32]{{Value: value}}
	alias := values[:]
	pointer := &values[0]
	previous := replace(pointer, Box[int32]{Value: value + 2})
	pointer.Value++
	return previous.Value, alias[0].Value, pointer == &alias[0]
}

func Scalar(value int32) (int32, int32) {
	pointer := &value
	previous := replace(pointer, value + 3)
	return previous, value
}

func Convert(value int32) (int32, bool) {
	left := &Left{Value: value}
	right := (*Right)(left)
	right.Value++
	return left.Value, left == (*Left)(right)
}

func StringPointer(value string) *Box[string] {
	return &Box[string]{Value: value}
}

func StringValue(value string) string {
	return StringPointer(value).Value
}

func CarrierMap(value int32) (int32, bool) {
	values := []Box[int32]{{Value: value}}
	pointer := &values[0]
	alias := pointer
	entries := map[*Box[int32]]int32{pointer: 1}
	return entries[alias], pointer == alias
}
`
	typescript := compilePointerArchitecture(
		t,
		map[string]string{"source.go": source},
	)
	for _, required := range []string{
		"Pointer<Box<int32>> | undefined",
		"function DirectInt(value: int32): Pointer<Box<int32>> | undefined",
		"RuntimeSlice.literal<Box$Storage<int32>>",
		"goSliceAddress<Box$Storage<int32>>",
		"projectPointer<Left, Right>",
		"hashPointer<Box",
		"function StringPointer(value: gostring): Pointer<Box<gostring>> | undefined",
	} {
		if !strings.Contains(typescript, required) {
			t.Fatalf("carrier pointer output lacks %q:\n%s", required, typescript)
		}
	}
	for _, forbidden := range []string{
		"GoPointer.view<Left, Right",
		"GoPointer.optionalStorage",
		"indexView",
		"AddressView",
		"goSliceAddressView",
		": any",
		" as any",
		"unknown",
	} {
		if strings.Contains(typescript, forbidden) {
			t.Fatalf("carrier pointer output contains %q:\n%s", forbidden, typescript)
		}
	}
}

func TestGenericPointerReceiverOnStructFieldUsesDeclarationABI(
	t *testing.T,
) {
	source := `package pointerarchitecture

type Shelf[T any] struct { Value T }

func (s *Shelf[T]) Put(value T) {
	s.Value = value
}

func (s *Shelf[T]) IsNil() bool {
	return s == nil
}

type Store struct {
	Shelf Shelf[int32]
}

func GenericField(value int32) int32 {
	store := Store{}
	store.Shelf.Put(value)
	return store.Shelf.Value
}

func GenericPointer(value int32) (int32, bool) {
	store := Store{}
	pointer := &store.Shelf
	pointer.Put(value)
	return store.Shelf.Value, pointer.IsNil()
}

func GenericNil() bool {
	var pointer *Shelf[int32]
	return pointer.IsNil()
}
`
	typescript := compilePointerArchitecture(
		t,
		map[string]string{"source.go": source},
	)
	if strings.Contains(typescript, "GoPointer") {
		t.Fatalf(
			"direct generic receiver acquired an interior pointer:\n%s",
			typescript,
		)
	}
	if !strings.Contains(
		typescript,
		"static Put$kernel<T>(s: Pointer<Shelf<T>> | undefined",
	) || !strings.Contains(
		typescript,
		"Shelf$Put$int32",
	) || !strings.Contains(
		typescript,
		"addressOf<Shelf<int32>>",
	) {
		t.Fatalf(
			"generic receiver did not use its declaration ABI:\n%s",
			typescript,
		)
	}
	if strings.Contains(typescript, "GoPointer") {
		t.Fatalf(
			"generic receiver retained a carrier-to-logical bridge:\n%s",
			typescript,
		)
	}
}

func TestForeignGenericMethodOriginAndAdapterUseDeclarationABI(
	t *testing.T,
) {
	typescript := compilePointerArchitecture(
		t,
		map[string]string{
			"ledger.go": `package pointerarchitecture

type Ledger[K comparable, V any] struct { Value V }

func (ledger *Ledger[K, V]) Set(key K, value V) {
	ledger.Value = value
}

func (ledger *Ledger[K, V]) Ready() bool {
	return ledger != nil
}
`,
			"source.go": `package pointerarchitecture

type ReadyContract interface { Ready() bool }

type Registry[T comparable] struct {
	Ledger Ledger[T, int32]
}

func ForeignGeneric(key string, value int32) int32 {
	registry := Registry[string]{}
	pointer := &registry.Ledger
	pointer.Set(key, value)
	return registry.Ledger.Value
}

func ForeignAdapter() ReadyContract {
	registry := Registry[string]{}
	return &registry.Ledger
}
`,
		},
	)
	if strings.Contains(typescript, "GoPointer") {
		t.Fatalf(
			"foreign generic receiver retained a carrier-to-logical bridge:\n%s",
			typescript,
		)
	}
	if !strings.Contains(
		typescript,
		"static Set$kernel<K, V>(ledger: Pointer<Ledger<K, V>> | undefined",
	) {
		t.Fatalf("foreign generic method lacks the direct family ABI:\n%s", typescript)
	}
	if !strings.Contains(typescript, "class $goInterfaceAdapter$") {
		t.Fatalf("foreign generic adapter was not emitted:\n%s", typescript)
	}
}

func compilePointerArchitecture(
	t *testing.T,
	sources map[string]string,
) string {
	t.Helper()
	directory := t.TempDir()
	writeFile(
		t,
		filepath.Join(directory, "go.mod"),
		"module example.com/pointerarchitecture\n\ngo 1.26.4\n",
	)
	for name, source := range sources {
		writeFile(t, filepath.Join(directory, name), source)
	}
	loaded, err := load.One(context.Background(), load.Request{
		Directory: directory,
		Pattern:   ".",
	})
	if err != nil {
		t.Fatal(err)
	}
	targetDirectory := filepath.Join(directory, "target")
	if err := os.MkdirAll(targetDirectory, 0o700); err != nil {
		t.Fatal(err)
	}
	artifacts := materializeExportedProgram(t, loaded, targetDirectory)
	sourceModule := artifacts.module(t, "source.ts")
	sourcePath := filepath.Join(
		targetDirectory,
		filepath.FromSlash(strings.TrimPrefix(sourceModule, "./")),
	)
	sourcePath = strings.TrimSuffix(sourcePath, ".js") + ".ts"
	printed, err := os.ReadFile(sourcePath)
	if err != nil {
		t.Fatal(err)
	}
	var generated strings.Builder
	generated.Write(printed)
	for _, targetPath := range artifacts.targetPaths {
		if targetPath == sourcePath {
			continue
		}
		content, readErr := os.ReadFile(targetPath)
		if readErr != nil {
			t.Fatal(readErr)
		}
		generated.Write(content)
	}
	typecheckMaterializedTypeScript(t, targetDirectory, artifacts)
	return generated.String()
}
