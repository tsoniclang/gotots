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
)

func TestUnnamedReceiversPrintTypecheckAndMatchGo(t *testing.T) {
	directory := filepath.Join(
		repositoryRoot(),
		"testdata",
		"constructs",
		"declaration",
		"function",
		"unnamed-receiver",
	)
	directory, err := filepath.Abs(directory)
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
	emission, err := emit.Compile(program, roots)
	if err != nil {
		t.Fatal(err)
	}
	workingDirectory := t.TempDir()
	artifacts := materializeArtifacts(t, emission, workingDirectory)
	for _, required := range []string{
		"Value(): int32",
		"static Pointer($0: GoPointer<Token",
		"Token.$fromStorage(value$storage.value).Value()",
		"Token.Pointer(value$storage)",
	} {
		if !strings.Contains(artifacts.printed, required) {
			t.Fatalf("unnamed receiver artifact lacks %q:\n%s", required, artifacts.printed)
		}
	}
	for _, forbidden := range []string{
		"export function Token_Value",
		"export function Token_Pointer",
	} {
		if strings.Contains(artifacts.printed, forbidden) {
			t.Fatalf("unnamed receiver artifact contains %q", forbidden)
		}
	}
	runner := filepath.Join(workingDirectory, "runner.ts")
	writeProgramFile(t, runner, `import "./program.js";
import { Audit } from "`+artifacts.sourceModule+`";

const values = Audit();
console.log(String(values.get(0)) + " " + String(values.get(1)));
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

require example.com/unnamedreceiver v0.0.0

replace example.com/unnamedreceiver => %s
`,
		filepath.ToSlash(directory),
	))
	writeProgramFile(t, filepath.Join(goRunner, "main.go"), `package main

import (
	"fmt"

	values "example.com/unnamedreceiver"
)

func main() {
	result := values.Audit()
	fmt.Println(result[0], result[1])
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
			"unnamed receiver output differs\nTypeScript: %q\nGo: %q",
			targetOutput,
			goOutput,
		)
	}
}
