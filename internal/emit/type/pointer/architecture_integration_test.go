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

func Replace[T any](pointer *T, value T) T {
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
	previous := Replace(pointer, Box{Value: value + 2})
	return previous.Value, pointer.Value, pointer == alias
}

func DirectMap(value int32) (int32, bool) {
	pointer := &Box{Value: value}
	alias := pointer
	values := map[*Box]int32{pointer: 1}
	return values[alias], pointer == alias
}
`
	typescript, goOutput, tsOutput := compileAndRunPointerArchitecture(
		t,
		source,
		`import { Direct, DirectMap, NewBox } from "__SOURCE__";

console.log(...Direct(40));
console.log(...DirectMap(41));
console.log(NewBox(7)?.Value);
`,
		`fmt.Println(pointer.Direct(40))
fmt.Println(pointer.DirectMap(41))
fmt.Println(pointer.NewBox(7).Value)
`,
	)
	if tsOutput != goOutput {
		t.Fatalf("TypeScript output = %q, Go output = %q", tsOutput, goOutput)
	}
	for _, required := range []string{
		"function NewBox(value: int32): Box | undefined",
		"static $assign(",
		"T$Pointer",
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

func Replace[T any](pointer *T, value T) T {
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
	previous := Replace(pointer, Box[int32]{Value: value + 2})
	pointer.Value++
	return previous.Value, alias[0].Value, pointer == &alias[0]
}

func Scalar(value int32) (int32, int32) {
	pointer := &value
	previous := Replace(pointer, value + 3)
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
		"RuntimeSlice.literal<Box$Storage<int32, int32>>",
		"goSliceAddress<Box<int32",
		"GoPointer.view<Left, Right",
		"goPointerHash",
		"function StringPointer(value: gostring): Box<gostring, gostring> | undefined",
	} {
		if !strings.Contains(typescript, required) {
			t.Fatalf("carrier pointer output lacks %q:\n%s", required, typescript)
		}
	}
	for _, forbidden := range []string{
		"GoPointer<Box<string",
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

func compileAndRunPointerArchitecture(
	t *testing.T,
	source string,
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
	writeFile(t, filepath.Join(directory, "source.go"), source)
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
