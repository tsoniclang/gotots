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

func TestDefinedMapsUseSelectedRepresentationsAndExecuteDifferentially(
	t *testing.T,
) {
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

func TestNativeDefinedMapKeyNominalityMutationFailsStrictTypecheck(
	t *testing.T,
) {
	loaded := loadDefinedMapProject(t)
	roots, err := emit.ExportedAPIRoots(loaded)
	if err != nil {
		t.Fatal(err)
	}
	emission, err := emit.Compile(loaded.Program(), roots)
	if err != nil {
		t.Fatal(err)
	}
	workingDirectory := t.TempDir()
	artifacts := materialize(t, emission, workingDirectory)
	specializationPath := definedKeySpecializationPath(t, artifacts)
	original := readFile(t, specializationPath)
	strictTypecheck(t, artifacts, workingDirectory)
	for _, testCase := range []struct {
		name        string
		method      string
		replacement string
	}{
		{
			name:        "raw number returned as nominal key",
			method:      "$copyKey",
			replacement: "return 0;",
		},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			mutated := replaceMethodReturn(
				t,
				original,
				testCase.method,
				testCase.replacement,
			)
			writeFile(t, specializationPath, mutated)
			if err := strictTypecheckResult(
				artifacts,
				workingDirectory,
				"",
			); err == nil {
				t.Fatalf("%s mutation passed strict typechecking", testCase.method)
			}
			writeFile(t, specializationPath, original)
		})
	}
}

func assertDefinedMapArtifacts(t *testing.T, artifacts materialized) {
	t.Helper()
	source := readFile(t, artifacts.file(t, "source.ts"))
	specialization := definedKeySpecialization(t, artifacts)
	t.Logf("defined map source bytes=%d", len(source))
	if len(source) > 12_000 {
		t.Fatalf(
			"defined map source = %d bytes, want at most 12000",
			len(source),
		)
	}
	for _, required := range []string{
		"let values: Values = new Values(GoMap.nil",
		"export type Alias = Values;",
		"export type PlainAlias = GoMapValue<",
		"export class Other",
		"Pointer<Values> | undefined = allocatePointer<Values>(new Values(GoMap.nil",
		"loadPointer<Values>",
		"let values: Values = new Values(GoMap.make",
		"let other: Other = new Other(values.$value)",
		"GoMapValue<Count, int32>",
		".store(key, ",
		".lookup(key)",
		"export function DefinedKeyZero(): bool",
		".nil();",
	} {
		if !strings.Contains(source, required) {
			t.Fatalf("defined map artifact lacks %q:\n%s", required, source)
		}
	}
	for _, required := range []string{
		"private static $copyKey($key: Count",
		"): Count",
		"return $key;",
		"lookup(key: Count",
		"store(key: Count",
		"value: int32): void",
		"keys(): Count",
		"[] {",
	} {
		if !strings.Contains(specialization, required) {
			t.Fatalf(
				"defined-key map specialization lacks %q:\n%s",
				required,
				specialization,
			)
		}
	}
	zeroStart := strings.Index(source, "export function ZeroState")
	makeStart := strings.Index(source, "export function MakeAliases")
	if zeroStart < 0 || makeStart <= zeroStart {
		t.Fatalf("defined map functions are absent:\n%s", source)
	}
	zeroBody := source[zeroStart:makeStart]
	if strings.Count(zeroBody, "new Values(GoMap.nil") != 2 {
		t.Fatalf("defined map zero wrapper count differs:\n%s", zeroBody)
	}
	conversionStart := strings.Index(source, "export function Conversions")
	if conversionStart <= makeStart {
		t.Fatalf("defined map conversion function is absent:\n%s", source)
	}
	for _, forbidden := range []string{
		"Values | undefined",
		"$wrapMap",
		"$readMap",
		"$storeMap",
		"key.$value",
		"export class Alias",
		"export class PlainAlias",
		"GoMap.nil<Count",
		"$projectKey",
		"$reifyKey",
		"new Count",
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

func definedKeySpecialization(t *testing.T, artifacts materialized) string {
	t.Helper()
	return readFile(t, definedKeySpecializationPath(t, artifacts))
}

func definedKeySpecializationPath(
	t *testing.T,
	artifacts materialized,
) string {
	t.Helper()
	var matches []string
	for _, path := range artifacts.paths {
		if strings.Contains(filepath.ToSlash(path), "/support/maps/") {
			matches = append(matches, path)
		}
	}
	if len(matches) != 1 {
		t.Fatalf(
			"defined-key map specialization files = %d, want one",
			len(matches),
		)
	}
	return matches[0]
}

func replaceMethodReturn(
	t *testing.T,
	source string,
	method string,
	replacement string,
) string {
	t.Helper()
	start := strings.Index(source, "static "+method+"(")
	if start < 0 {
		t.Fatalf("specialization method %q is absent", method)
	}
	body := strings.Index(source[start:], "{")
	if body < 0 {
		t.Fatalf("specialization method %q has no body", method)
	}
	body += start
	returnStart := strings.Index(source[body:], "return ")
	if returnStart < 0 {
		t.Fatalf("specialization method %q has no return", method)
	}
	returnStart += body
	returnEnd := strings.Index(source[returnStart:], ";")
	if returnEnd < 0 {
		t.Fatalf("specialization method %q return has no terminator", method)
	}
	returnEnd += returnStart + 1
	return source[:returnStart] + replacement + source[returnEnd:]
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
	fmt.Println(values.MakeAliases())
	fmt.Println(values.Conversions())
	fmt.Println(values.NilOperations())
	fmt.Println(values.PlainAliasBehavior())
	fmt.Println(values.DefinedKeyBehavior())
	fmt.Println(values.DefinedKeyZero())
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
    DefinedKeyBehavior,
    DefinedKeyZero,
    MakeAliases,
    NilOperations,
    NilWrite,
    PlainAliasBehavior,
} from "`+artifacts.module(t, "source.ts")+`";
import "./program.js";

function show(value: number | bigint | boolean): string {
    return typeof value === "bigint" ? value.toString() : String(value);
}
function print(...values: Array<number | bigint | boolean>): void {
    console.log(...values.map(show));
}

print(...MakeAliases());
print(...Conversions());
print(...NilOperations());
print(...PlainAliasBehavior());
print(...DefinedKeyBehavior());
console.log(DefinedKeyZero());
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
