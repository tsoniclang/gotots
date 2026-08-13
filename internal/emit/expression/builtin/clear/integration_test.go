package clear_test

import (
	"errors"
	"fmt"
	"go/ast"
	"go/types"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/tsoniclang/gotots/internal/emit"
	"github.com/tsoniclang/gotots/internal/emit/api"
)

func TestClearBuiltinsPrintTypecheckAndExecuteDifferentially(t *testing.T) {
	workingDirectory := t.TempDir()
	emission := compileClear(t)
	artifacts := materializeClear(t, emission, workingDirectory)
	printed := strings.Join(
		readClearArtifacts(t, artifacts.paths),
		"\n",
	)
	for _, forbidden := range []string{
		" as any",
		" as unknown",
		".call(",
		".apply(",
		".bind(",
		"goSliceClearWith",
	} {
		if strings.Contains(printed, forbidden) {
			t.Fatalf("clear artifact contains %q:\n%s", forbidden, printed)
		}
	}
	for _, required := range []string{
		"goSliceClear(values, 0)",
		"values.clear()",
		"clear()",
		"export function clearGeneric$kernel<C>",
		"$go$clear$",
		"export function clearGeneric$SliceOf_Named_clearvalues$Box",
		"export function clearGeneric$MapOf_int32_To_Named_clearvalues$Box",
	} {
		if !strings.Contains(printed, required) {
			t.Fatalf("clear artifact lacks %q:\n%s", required, printed)
		}
	}
	if strings.Contains(printed, "export function clearGeneric<C>") {
		t.Fatalf("clear artifact restored a public hidden-capability ABI:\n%s", printed)
	}
	runnerPath := filepath.Join(workingDirectory, "runner.ts")
	writeClearFile(t, runnerPath, fmt.Sprintf(`import {
	    ClearAggregateMap,
	    ClearAggregateSlice,
	    ClearGenericMap,
	    ClearGenericSlice,
	    ClearNilValues,
    ClearScalarMap,
    ClearScalarSlice,
} from %q;

console.log(String(ClearScalarSlice()));
console.log(String(ClearAggregateSlice()));
console.log(String(ClearScalarMap()));
console.log(String(ClearAggregateMap()));
console.log(String(ClearGenericSlice()));
console.log(String(ClearGenericMap()));
console.log(String(ClearNilValues()));
`, artifacts.module))
	writeClearFile(
		t,
		filepath.Join(workingDirectory, "package.json"),
		"{\"type\":\"module\"}\n",
	)
	strictClearTypecheck(t, workingDirectory, artifacts.paths, runnerPath)
	goOutput := runClearGo(t, workingDirectory)
	targetOutput := runClear(
		t,
		workingDirectory,
		"node",
		filepath.Join(workingDirectory, "out", "runner.js"),
	)
	if targetOutput != goOutput {
		t.Fatalf(
			"TypeScript output differs from Go\nTypeScript:\n%s\nGo:\n%s",
			targetOutput,
			goOutput,
		)
	}
}

func TestClearRuntimeSurfacesFollowValueContracts(t *testing.T) {
	without := printClearEmission(t, compileClearSource(t, `package demand

func Use() int {
	values := []int32{1}
	entries := map[int32]int32{1: 2}
	return len(values) + len(entries)
}
`))
	with := printClearEmission(t, compileClearSource(t, `package demand

func Use() int32 {
	values := []int32{1}
	entries := map[int32]int32{1: 2}
	clear(values)
	clear(entries)
	return values[0] + entries[1]
}
`))
	withoutSlice := without["runtime/slice.ts"]
	withSlice := with["runtime/slice.ts"]
	if withoutSlice == "" || withSlice == "" {
		t.Fatal("slice runtime artifact is absent from demand pair")
	}
	for _, fragment := range []string{
		"clear(zero: T): void",
		"export function goSliceClear<T>",
	} {
		if strings.Contains(withoutSlice, fragment) {
			t.Fatalf("slice runtime contains undemanded %q:\n%s", fragment, withoutSlice)
		}
		if strings.Count(withSlice, fragment) != 1 {
			t.Fatalf(
				"slice runtime count(%q) = %d, want one:\n%s",
				fragment,
				strings.Count(withSlice, fragment),
				withSlice,
			)
		}
	}
	withoutMap := without["runtime/map.ts"]
	withMap := with["runtime/map.ts"]
	for name, source := range map[string]string{
		"without clear use": withoutMap,
		"with clear use":    withMap,
	} {
		if source == "" {
			t.Fatalf("map runtime is absent %s", name)
		}
		if strings.Count(source, "clear(): void {") != 1 ||
			strings.Count(source, "clear(): void;") != 1 {
			t.Fatalf(
				"map runtime %s lacks one implementation and one interface contract:\n%s",
				name,
				source,
			)
		}
		if strings.Contains(source, "goMapClear") {
			t.Fatalf("map runtime %s retained the superseded clear helper:\n%s", name, source)
		}
	}
	var withoutProgram strings.Builder
	var withProgram strings.Builder
	for path, source := range without {
		if path != "runtime/map.ts" {
			withoutProgram.WriteString(source)
		}
	}
	for path, source := range with {
		if path != "runtime/map.ts" {
			withProgram.WriteString(source)
		}
	}
	if strings.Contains(withoutProgram.String(), ".clear();") {
		t.Fatalf("source without clear contains a map clear call:\n%s", withoutProgram.String())
	}
	if !strings.Contains(withProgram.String(), ".clear();") {
		t.Fatalf("source with clear lacks a direct map clear call:\n%s", withProgram.String())
	}
}

func TestClearBuiltinIdentityMutationFailsClosed(t *testing.T) {
	loaded, err := loadClearProject()
	if err != nil {
		t.Fatal(err)
	}
	mutations := 0
	for identifier, object := range loaded.TypesInfo().Uses {
		if object == types.Universe.Lookup("clear") {
			loaded.TypesInfo().Uses[identifier] = types.Universe.Lookup("len")
			mutations++
		}
	}
	if mutations != 7 {
		t.Fatalf("clear identity mutations = %d, want seven", mutations)
	}
	roots, err := emit.ExportedAPIRoots(loaded)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := emit.Compile(loaded.Program(), roots); err == nil {
		t.Fatal("spelling mutation compiled through clear owner")
	}
}

func TestClearMissingBuiltinIdentityFactFailsAtBuiltinOwner(t *testing.T) {
	loaded, err := loadClearProject()
	if err != nil {
		t.Fatal(err)
	}
	removed := 0
	for _, file := range loaded.Files() {
		ast.Inspect(file.Syntax(), func(node ast.Node) bool {
			call, ok := node.(*ast.CallExpr)
			if !ok || len(call.Args) != 1 {
				return true
			}
			identifier, ok := call.Fun.(*ast.Ident)
			if !ok || loaded.TypesInfo().Uses[identifier] != types.Universe.Lookup("clear") {
				return true
			}
			delete(loaded.TypesInfo().Uses, identifier)
			removed++
			return true
		})
	}
	if removed != 7 {
		t.Fatalf("clear type facts removed = %d, want seven", removed)
	}
	roots, err := emit.ExportedAPIRoots(loaded)
	if err != nil {
		t.Fatal(err)
	}
	_, err = emit.Compile(loaded.Program(), roots)
	var unsupported *api.UnsupportedError
	if !errors.As(err, &unsupported) {
		t.Fatalf("error = %v, want typed unsupported at clear owner", err)
	}
}

func readClearArtifacts(t *testing.T, paths []string) []string {
	t.Helper()
	contents := make([]string, 0, len(paths))
	for _, path := range paths {
		content, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		contents = append(contents, string(content))
	}
	return contents
}

func runClearGo(t *testing.T, workingDirectory string) string {
	t.Helper()
	modulePath, err := filepath.Abs(clearFixtureDirectory())
	if err != nil {
		t.Fatal(err)
	}
	runnerDirectory := filepath.Join(workingDirectory, "go-runner")
	writeClearFile(t, filepath.Join(runnerDirectory, "go.mod"), fmt.Sprintf(`module example.com/runner

go 1.26.4

require example.com/clearvalues v0.0.0

replace example.com/clearvalues => %s
`, filepath.ToSlash(modulePath)))
	writeClearFile(t, filepath.Join(runnerDirectory, "main.go"), `package main

import (
    "fmt"
    clearvalues "example.com/clearvalues"
)

func main() {
    fmt.Println(clearvalues.ClearScalarSlice())
    fmt.Println(clearvalues.ClearAggregateSlice())
    fmt.Println(clearvalues.ClearScalarMap())
    fmt.Println(clearvalues.ClearAggregateMap())
    fmt.Println(clearvalues.ClearGenericSlice())
    fmt.Println(clearvalues.ClearGenericMap())
    fmt.Println(clearvalues.ClearNilValues())
}
`)
	return runClear(t, runnerDirectory, filepath.Join(runtime.GOROOT(), "bin", "go"), "run", ".")
}
