package pointer_test

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
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

func Direct(value int32) (int32, int32, bool) {
	box := Box{Value: value}
	pointer := &box
	alias := pointer
	previous := replace(pointer, Box{Value: value + 2})
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
	typescript, goOutput, tsOutput := compileAndRunPointerArchitecture(
		t,
		source,
		`import { Boolean, Direct, DirectMap, NewBox } from "__SOURCE__";

console.log(...Direct(40));
console.log(...DirectMap(41));
console.log(NewBox(7)?.Value);
console.log(...Boolean(true));
`,
		`fmt.Println(pointer.Direct(40))
fmt.Println(pointer.DirectMap(41))
fmt.Println(pointer.NewBox(7).Value)
fmt.Println(pointer.Boolean(true))
`,
	)
	if tsOutput != goOutput {
		t.Fatalf("TypeScript output = %q, Go output = %q", tsOutput, goOutput)
	}
	for _, required := range []string{
		"function NewBox(value: int32): Box | undefined",
		"static $assign(",
		"GoPointerType<T>",
		"class Box implements GoPointerRepresentedValue<Box>",
	} {
		if !strings.Contains(typescript, required) {
			t.Fatalf("direct pointer output lacks %q:\n%s", required, typescript)
		}
	}
	for _, forbidden := range []string{
		"GoPointer<Box",
		"Box$Storage",
		"GoPointer.cell<Box",
		"goPointerHash",
		"any",
		"unknown",
	} {
		if strings.Contains(typescript, forbidden) {
			t.Fatalf("direct pointer output contains %q:\n%s", forbidden, typescript)
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
	typescript, goOutput, tsOutput := compileAndRunPointerArchitecture(
		t,
		source,
		`import {
    CarrierMap,
    Convert,
    DirectIntValue,
    Scalar,
    SliceAlias,
    StringValue,
} from "__SOURCE__";

console.log(...SliceAlias(40));
console.log(...Scalar(50));
console.log(...Convert(60));
console.log(...CarrierMap(61));
console.log(DirectIntValue(62));
console.log(StringValue("ok"));
`,
		`fmt.Println(pointer.SliceAlias(40))
fmt.Println(pointer.Scalar(50))
fmt.Println(pointer.Convert(60))
fmt.Println(pointer.CarrierMap(61))
fmt.Println(pointer.DirectIntValue(62))
fmt.Println(pointer.StringPointer("ok").Value)
`,
	)
	if tsOutput != goOutput {
		t.Fatalf("TypeScript output = %q, Go output = %q", tsOutput, goOutput)
	}
	for _, required := range []string{
		"GoPointer<Box<int32",
		"function DirectInt(value: int32): GoPointer<Box<int32",
		"RuntimeSlice.literal<Box$Storage<int32>>",
		"goSliceAddress<Box<int32",
		"GoPointer.view<Left, Right",
		"goPointerHash",
		"function StringPointer(value: gostring): GoPointer<Box<gostring>, Box$Storage<gostring>>",
	} {
		if !strings.Contains(typescript, required) {
			t.Fatalf("carrier pointer output lacks %q:\n%s", required, typescript)
		}
	}
	for _, forbidden := range []string{
		"GoPointer.optionalStorage",
		"indexView",
		"AddressView",
		"goSliceAddressView",
		"any",
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
	typescript, goOutput, tsOutput := compileAndRunPointerArchitecture(
		t,
		source,
		`import { GenericField, GenericNil, GenericPointer } from "__SOURCE__";

console.log(GenericField(42));
console.log(...GenericPointer(43));
console.log(GenericNil());
`,
		`fmt.Println(pointer.GenericField(42))
fmt.Println(pointer.GenericPointer(43))
fmt.Println(pointer.GenericNil())
`,
	)
	if tsOutput != goOutput {
		t.Fatalf("TypeScript output = %q, Go output = %q", tsOutput, goOutput)
	}
	if strings.Contains(typescript, "GoPointer.field<Shelf") {
		t.Fatalf(
			"direct generic receiver acquired an interior pointer:\n%s",
			typescript,
		)
	}
	if !strings.Contains(
		typescript,
		"static Put$kernel<T>(s: GoPointer<Shelf<T>, Shelf$Storage<T>>",
	) || !strings.Contains(
		typescript,
		"Put$concrete_",
	) || !strings.Contains(
		typescript,
		"GoPointer.objectField<Shelf<int32>",
	) {
		t.Fatalf(
			"generic receiver did not use its declaration ABI:\n%s",
			typescript,
		)
	}
	if strings.Contains(typescript, "GoPointer.optionalStorage(") {
		t.Fatalf(
			"generic receiver retained a carrier-to-logical bridge:\n%s",
			typescript,
		)
	}
}

func TestForeignGenericMethodOriginAndAdapterUseDeclarationABI(
	t *testing.T,
) {
	typescript, goOutput, tsOutput :=
		compileAndRunPointerArchitectureFiles(
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
			`import { ForeignAdapter, ForeignGeneric } from "__SOURCE__";

console.log(ForeignGeneric("key", 44));
const ready = ForeignAdapter();
if (ready === undefined) throw new Error("unexpected nil");
console.log(ready.Ready());
`,
			`fmt.Println(pointer.ForeignGeneric("key", 44))
fmt.Println(pointer.ForeignAdapter().Ready())
`,
		)
	if tsOutput != goOutput {
		t.Fatalf("TypeScript output = %q, Go output = %q", tsOutput, goOutput)
	}
	if strings.Contains(typescript, "GoPointer.optionalStorage(") {
		t.Fatalf(
			"foreign generic receiver retained a carrier-to-logical bridge:\n%s",
			typescript,
		)
	}
	if !strings.Contains(
		typescript,
		"static Set$kernel<K, V>(ledger: GoPointer<Ledger<K, V>, Ledger$Storage<K, V>>",
	) {
		t.Fatalf("foreign generic method lacks the family carrier ABI:\n%s", typescript)
	}
	if !strings.Contains(typescript, "class $goInterfaceAdapter_") {
		t.Fatalf("foreign generic adapter was not emitted:\n%s", typescript)
	}
}

func compileAndRunPointerArchitecture(
	t *testing.T,
	source string,
	runner string,
	goBody string,
) (string, string, string) {
	t.Helper()
	return compileAndRunPointerArchitectureFiles(
		t,
		map[string]string{"source.go": source},
		runner,
		goBody,
	)
}

func compileAndRunPointerArchitectureFiles(
	t *testing.T,
	sources map[string]string,
	runner string,
	goBody string,
) (string, string, string) {
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
	runnerPath := filepath.Join(targetDirectory, "runner.ts")
	writeFile(
		t,
		runnerPath,
		strings.ReplaceAll(runner, "__SOURCE__", sourceModule),
	)
	tsOutput := executeMaterializedTypeScript(
		t,
		targetDirectory,
		artifacts,
		runnerPath,
	)
	goOutput := runPointerArchitectureGo(t, directory, goBody)
	return generated.String(), goOutput, tsOutput
}

func runPointerArchitectureGo(
	t *testing.T,
	moduleDirectory string,
	body string,
) string {
	t.Helper()
	runnerDirectory := filepath.Join(moduleDirectory, "runner")
	writeFile(t, filepath.Join(runnerDirectory, "go.mod"), fmt.Sprintf(`module example.com/runner

go 1.26.4

require example.com/pointerarchitecture v0.0.0

replace example.com/pointerarchitecture => %s
`, filepath.ToSlash(moduleDirectory)))
	writeFile(t, filepath.Join(runnerDirectory, "main.go"), `package main

import (
	"fmt"

	pointer "example.com/pointerarchitecture"
)

func main() {
`+body+`}
`)
	return run(
		t,
		runnerDirectory,
		filepath.Join(runtime.GOROOT(), "bin", "go"),
		"run",
		".",
	)
}
