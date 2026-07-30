package emit_test

import (
	"context"
	"fmt"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/tsoniclang/gotots/internal/emit"
	"github.com/tsoniclang/gotots/internal/load"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func TestTargetBindingsOrderEagerDependenciesAndProtectIntrinsics(
	t *testing.T,
) {
	directory, err := filepath.Abs(targetBindingDirectory())
	if err != nil {
		t.Fatal(err)
	}
	program, err := load.Load(context.Background(), load.Request{
		Directory: directory,
		Pattern:   ".",
	})
	if err != nil {
		t.Fatal(err)
	}
	roots, err := emit.ExportedAPIRoots(program.Roots()[0])
	if err != nil {
		t.Fatal(err)
	}
	options := emit.DefaultOptions()
	options.IntegerRepresentation = emit.IntegerRepresentationBigInt
	emission, err := emit.CompileWithOptions(program, roots, options)
	if err != nil {
		t.Fatal(err)
	}
	assertTargetBindingAST(t, emission)

	workingDirectory := t.TempDir()
	artifacts := materializeArtifacts(t, emission, workingDirectory)
	if !strings.Contains(artifacts.printed, "globalThis.Number(") {
		t.Fatalf(
			"target intrinsic is not globally anchored:\n%s",
			artifacts.printed,
		)
	}
	if strings.Contains(artifacts.printed, "function globalThis(") {
		t.Fatalf(
			"Go binding captured the target global anchor:\n%s",
			artifacts.printed,
		)
	}
	runner := filepath.Join(workingDirectory, "runner.ts")
	writeProgramFile(t, runner, `import "./program.js";
import { Audit } from "`+artifacts.sourceModule+`";

console.log(String(Audit(5n)));
`)
	writeProgramFile(
		t,
		filepath.Join(workingDirectory, "package.json"),
		"{\"type\":\"module\"}\n",
	)
	waveThreeTypecheck(
		t,
		workingDirectory,
		append(artifacts.paths, runner),
	)
	targetOutput := runProgram(
		t,
		workingDirectory,
		"node",
		filepath.Join(workingDirectory, "out", "runner.js"),
	)
	goRunner := filepath.Join(workingDirectory, "go-runner")
	writeProgramFile(t, filepath.Join(goRunner, "go.mod"), fmt.Sprintf(
		`module example.com/runner

go 1.26.4

require example.com/targetbinding v0.0.0

replace example.com/targetbinding => %s
`,
		filepath.ToSlash(directory),
	))
	writeProgramFile(t, filepath.Join(goRunner, "main.go"), `package main

import (
	"fmt"

	values "example.com/targetbinding"
)

func main() {
	fmt.Println(values.Audit(5))
}
`)
	goOutput := runProgram(
		t,
		goRunner,
		filepath.Join(runtime.GOROOT(), "bin", "go"),
		"run",
		".",
	)
	if targetOutput != goOutput {
		t.Fatalf(
			"target-binding differential differs\nTypeScript: %q\nGo: %q",
			targetOutput,
			goOutput,
		)
	}
}

func assertTargetBindingAST(t *testing.T, emission emit.ProgramEmission) {
	t.Helper()
	for _, file := range emission.Files() {
		if file.Kind() != emit.TargetFileSource {
			continue
		}
		classIndex := -1
		constantIndex := -1
		for index, statement := range file.SourceFile().Statements() {
			switch statement := statement.(type) {
			case tsgo.ClassDeclaration:
				if statement.Name().Text() == "Number" {
					classIndex = index
				}
			case tsgo.VariableStatement:
				for _, declaration := range statement.
					DeclarationList().
					Declarations() {
					name, ok := declaration.Name().(tsgo.Identifier)
					if ok && name.Text() == "Before" {
						constantIndex = index
					}
				}
			}
		}
		if classIndex < 0 || constantIndex < 0 {
			t.Fatalf(
				"target declarations absent: class=%d constant=%d",
				classIndex,
				constantIndex,
			)
		}
		if classIndex >= constantIndex {
			t.Fatalf(
				"eager dependency order = class %d, constant %d",
				classIndex,
				constantIndex,
			)
		}
		return
	}
	t.Fatal("target-binding fixture emitted no source file")
}

func targetBindingDirectory() string {
	return filepath.Join(
		repositoryRoot(),
		"testdata",
		"constructs",
		"naming",
		"target-binding",
	)
}
