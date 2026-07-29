package maprepresentation_test

import (
	"context"
	"fmt"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/tsoniclang/gotots/internal/emit"
	"github.com/tsoniclang/gotots/internal/load"
)

func TestDefinedMapsUseNilUnionAndExecuteDifferentially(t *testing.T) {
	for _, testCase := range []struct {
		name    string
		options emit.Options
	}{
		{name: "number", options: emit.DefaultOptions()},
		{
			name: "bigint",
			options: emit.Options{
				IntegerRepresentation: emit.IntegerRepresentationBigInt,
				EvaluationOrder:       emit.EvaluationOrderPreserveGo,
			},
		},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			loaded := loadDefinedMapProject(t)
			roots, err := emit.ExportedAPIRoots(loaded)
			if err != nil {
				t.Fatal(err)
			}
			emission, err := emit.CompileWithOptions(
				loaded.Program(),
				roots,
				testCase.options,
			)
			if err != nil {
				t.Fatal(err)
			}
			workingDirectory := t.TempDir()
			artifacts := materialize(t, emission, workingDirectory)
			assertDefinedMapArtifacts(t, artifacts)
			goOutput := executeDefinedMapGo(t, workingDirectory)
			targetOutput := executeDefinedMapTypeScript(
				t,
				artifacts,
				workingDirectory,
			)
			if targetOutput != goOutput {
				t.Fatalf(
					"TypeScript output differs from Go\nTypeScript:\n%s\nGo:\n%s",
					targetOutput,
					goOutput,
				)
			}
		})
	}
}

func assertDefinedMapArtifacts(t *testing.T, artifacts materialized) {
	t.Helper()
	source := readFile(t, artifacts.file(t, "source.ts"))
	t.Logf("defined map source bytes=%d", len(source))
	if len(source) > 12_000 {
		t.Fatalf(
			"defined map source = %d bytes, want at most 12000",
			len(source),
		)
	}
	for _, required := range []string{
		"Values | undefined",
		"export type Alias = Values | undefined",
		"export type PlainAlias = GoMapValue<",
		"export class Other",
		"GoPointer.cell<Values | undefined, Values | undefined>(void 0)",
		"Values.$wrapMap(GoMap.make",
		"Other.$wrapMap(",
	} {
		if !strings.Contains(source, required) {
			t.Fatalf("defined map artifact lacks %q:\n%s", required, source)
		}
	}
	zeroStart := strings.Index(source, "export function ZeroState")
	makeStart := strings.Index(source, "export function MakeAliases")
	if zeroStart < 0 || makeStart <= zeroStart {
		t.Fatalf("defined map functions are absent:\n%s", source)
	}
	zeroBody := source[zeroStart:makeStart]
	if strings.Contains(zeroBody, "new Values(") ||
		strings.Contains(zeroBody, "Values.$wrapMap(") ||
		!strings.Contains(zeroBody, "void 0") {
		t.Fatalf("defined map zero allocates a nominal wrapper:\n%s", zeroBody)
	}
	conversionStart := strings.Index(source, "export function Conversions")
	if conversionStart <= makeStart {
		t.Fatalf("defined map conversion function is absent:\n%s", source)
	}
	makeBody := source[makeStart:conversionStart]
	if strings.Count(makeBody, "Values.$wrapMap(") != 1 {
		t.Fatalf("defined map make wrapper calls = %d, want one:\n%s",
			strings.Count(makeBody, "Values.$wrapMap("),
			makeBody,
		)
	}
	if strings.Count(source, "new Values(") != 1 {
		t.Fatalf(
			"defined map Values constructors = %d, want the guarded family owner only:\n%s",
			strings.Count(source, "new Values("),
			source,
		)
	}
	for _, forbidden := range []string{
		"export class Alias",
		"export class PlainAlias",
		"any",
		"unknown",
		".call(",
		".apply(",
		".bind(",
	} {
		if strings.Contains(source, forbidden) {
			t.Fatalf("defined map artifact contains %q:\n%s", forbidden, source)
		}
	}
}

func executeDefinedMapGo(t *testing.T, workingDirectory string) string {
	t.Helper()
	modulePath, err := filepath.Abs(definedMapProjectDirectory())
	if err != nil {
		t.Fatal(err)
	}
	runnerDirectory := filepath.Join(workingDirectory, "go-runner")
	writeFile(t, filepath.Join(runnerDirectory, "go.mod"), fmt.Sprintf(`module example.com/runner

go 1.26.4

require example.com/definedmap v0.0.0

replace example.com/definedmap => %s
`, filepath.ToSlash(modulePath)))
	writeFile(t, filepath.Join(runnerDirectory, "main.go"), `package main

import (
	"fmt"

	values "example.com/definedmap"
)

func nilWriteFails() (failed bool) {
	defer func() { failed = recover() != nil }()
	values.NilWrite()
	return false
}

func main() {
	fmt.Println(values.ZeroState())
	fmt.Println(values.MakeAliases())
	fmt.Println(values.Conversions())
	fmt.Println(values.NilOperations())
	fmt.Println(values.PlainAliasBehavior())
	fmt.Println(nilWriteFails())
}
`)
	return run(
		t,
		runnerDirectory,
		filepath.Join(runtime.GOROOT(), "bin", "go"),
		"run",
		".",
	)
}

func executeDefinedMapTypeScript(
	t *testing.T,
	artifacts materialized,
	workingDirectory string,
) string {
	t.Helper()
	runnerPath := filepath.Join(workingDirectory, "runner.ts")
	writeFile(t, runnerPath, `import {
    Conversions,
    MakeAliases,
    NilOperations,
    NilWrite,
    PlainAliasBehavior,
    ZeroState,
} from "`+artifacts.module(t, "source.ts")+`";
import "./program.js";

function show(value: number | bigint | boolean): string {
    return typeof value === "bigint" ? value.toString() : String(value);
}
function print(...values: Array<number | bigint | boolean>): void {
    console.log(...values.map(show));
}

print(...ZeroState());
print(...MakeAliases());
print(...Conversions());
print(...NilOperations());
print(...PlainAliasBehavior());
let nilWriteFailed = false;
try { NilWrite(); } catch { nilWriteFailed = true; }
console.log(nilWriteFailed);
`)
	writeFile(
		t,
		filepath.Join(workingDirectory, "package.json"),
		"{\"type\":\"module\"}\n",
	)
	strictTypecheckWithRunner(t, artifacts, workingDirectory, runnerPath)
	return run(
		t,
		workingDirectory,
		"node",
		filepath.Join(workingDirectory, "out", "runner.js"),
	)
}

func loadDefinedMapProject(t *testing.T) *load.Package {
	t.Helper()
	loaded, err := load.One(context.Background(), load.Request{
		Directory: definedMapProjectDirectory(),
		Pattern:   ".",
	})
	if err != nil {
		t.Fatal(err)
	}
	return loaded
}

func definedMapProjectDirectory() string {
	return filepath.Join(
		repositoryRoot(),
		"testdata",
		"constructs",
		"value",
		"map",
		"defined",
	)
}
