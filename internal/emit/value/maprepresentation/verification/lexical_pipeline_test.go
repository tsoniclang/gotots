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

func TestLexicalAggregateMapsCompileAndExecuteDifferentially(t *testing.T) {
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
			loaded, err := load.One(context.Background(), load.Request{
				Directory: lexicalMapProjectDirectory(),
				Pattern:   ".",
			})
			if err != nil {
				t.Fatal(err)
			}
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
			assertLexicalMapArtifacts(t, emission, artifacts)
			goOutput := executeLexicalMapGo(t, workingDirectory)
			targetOutput := executeLexicalMapTypeScript(
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

func assertLexicalMapArtifacts(
	t *testing.T,
	emission emit.ProgramEmission,
	artifacts materialized,
) {
	t.Helper()
	assemblyPath := ""
	for _, file := range emission.Files() {
		if strings.HasPrefix(file.OutputPath(), "support/maps/") {
			t.Fatalf(
				"lexical map escaped to support file %s",
				file.OutputPath(),
			)
		}
		if file.Kind() == emit.TargetFilePackageAssembly {
			assemblyPath = file.OutputPath()
		}
	}
	if assemblyPath == "" {
		t.Fatal("lexical fixture has no package assembly")
	}
	source := readFile(t, artifacts.file(t, "source.ts"))
	assembly := readFile(t, artifacts.file(t, assemblyPath))
	if strings.Count(source, "class $goMap_") != 2 {
		t.Fatalf(
			"source lexical map classes = %d, want two:\n%s",
			strings.Count(source, "class $goMap_"),
			source,
		)
	}
	if strings.Count(assembly, "class $goMap_") != 1 {
		t.Fatalf(
			"initializer lexical map classes = %d, want one:\n%s",
			strings.Count(assembly, "class $goMap_"),
			assembly,
		)
	}
	nestedBlockStart := strings.Index(source, "export function NestedBlock")
	nestedLiteralStart := strings.Index(
		source,
		"export function NestedFunctionLiteral",
	)
	if nestedBlockStart < 0 ||
		nestedLiteralStart <= nestedBlockStart ||
		!strings.Contains(
			source[nestedBlockStart:nestedLiteralStart],
			"{\n        class Key",
		) ||
		!strings.Contains(
			source[nestedBlockStart:nestedLiteralStart],
			"\n        class $goMap_",
		) {
		t.Fatal("nested-block map class was not inserted after its local type")
	}
	if strings.Contains(
		source[nestedBlockStart:nestedLiteralStart],
		"{\n    class $goMap_",
	) {
		t.Fatal("nested-block map class was hoisted to function scope")
	}
	if !strings.Contains(
		source[nestedLiteralStart:],
		"(): int32 => {\n        class Key",
	) ||
		!strings.Contains(
			source[nestedLiteralStart:],
			"class $goMap_",
		) {
		t.Fatal("nested function-literal map class escaped its lexical body")
	}
	if !strings.Contains(assembly, "(): int32 => {\n        class Key") ||
		!strings.Contains(assembly, "class $goMap_") {
		t.Fatal("package initializer map class escaped its function literal")
	}
}

func executeLexicalMapGo(t *testing.T, workingDirectory string) string {
	t.Helper()
	modulePath, err := filepath.Abs(lexicalMapProjectDirectory())
	if err != nil {
		t.Fatal(err)
	}
	runnerDirectory := filepath.Join(workingDirectory, "go-runner")
	writeFile(t, filepath.Join(runnerDirectory, "go.mod"), fmt.Sprintf(`module example.com/runner

go 1.26.4

require example.com/lexicalmap v0.0.0

replace example.com/lexicalmap => %s
`, filepath.ToSlash(modulePath)))
	writeFile(t, filepath.Join(runnerDirectory, "main.go"), `package main

import (
	"fmt"

	values "example.com/lexicalmap"
)

func main() {
	fmt.Println(values.PackageVariableLiteral())
	fmt.Println(values.NestedBlock())
	fmt.Println(values.NestedFunctionLiteral())
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

func executeLexicalMapTypeScript(
	t *testing.T,
	artifacts materialized,
	workingDirectory string,
) string {
	t.Helper()
	runnerPath := filepath.Join(workingDirectory, "runner.ts")
	writeFile(t, runnerPath, `import {
    NestedBlock,
    NestedFunctionLiteral,
    PackageVariableLiteral,
} from "`+artifacts.module(t, "source.ts")+`";
import "./program.js";

console.log(String(PackageVariableLiteral()));
console.log(String(NestedBlock()));
console.log(String(NestedFunctionLiteral()));
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

func lexicalMapProjectDirectory() string {
	return filepath.Join(
		repositoryRoot(),
		"testdata",
		"constructs",
		"value",
		"map",
		"lexical",
	)
}
