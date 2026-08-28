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

func TestNestedNamedStructStorageZeroIsDirectAndDifferential(t *testing.T) {
	project := t.TempDir()
	writeProgramFile(
		t,
		filepath.Join(project, "go.mod"),
		"module example.com/storagezero\n\ngo 1.26.4\n",
	)
	writeProgramFile(t, filepath.Join(project, "source.go"), `package storagezero

type Leaf struct { Count int32 }
type Inner[T any] struct {
	Value T
	Leaf Leaf
}
type Outer[T any] struct { Inner Inner[T] }
type WithBlank struct {
	_ Leaf
	Value int32
}

var Stored Outer[int32]

func CopyBlank(value WithBlank) WithBlank { return value }

func Run() int32 {
	values := []Outer[int32]{{}, {}}
	values[1].Inner.Value = 7
	sparse := Outer[int32]{}
	blankValues := []WithBlank{{Value: 3}}
	blankValues[0].Value++
	copied := CopyBlank(blankValues[0])
	return Stored.Inner.Value + sparse.Inner.Leaf.Count +
		values[0].Inner.Leaf.Count + values[1].Inner.Value + copied.Value
}
`)
	program, err := load.Load(context.Background(), load.Request{
		Directory: project,
		Pattern:   ".",
	})
	if err != nil {
		t.Fatal(err)
	}
	root, err := emit.NewRoot(program.Roots()[0].Types().Scope().Lookup("Run"))
	if err != nil {
		t.Fatal(err)
	}
	emission, err := emit.Compile(program, []emit.Root{root})
	if err != nil {
		t.Fatal(err)
	}
	workingDirectory := t.TempDir()
	artifacts := materializeArtifacts(t, emission, workingDirectory)
	for _, required := range []string{
		"static $zeroStorage",
		"Outer.$zeroStorage",
		"Inner.$zeroStorage",
		"Leaf.$zeroStorage",
	} {
		if !strings.Contains(artifacts.printed, required) {
			t.Fatalf("direct storage zero lacks %q:\n%s", required, artifacts.printed)
		}
	}
	for _, forbidden := range []string{
		"Outer.$storageOf(Outer.$zero",
		"Inner.$storageOf(Inner.$zero",
		"Leaf.$storageOf(Leaf.$zero",
	} {
		if strings.Contains(artifacts.printed, forbidden) {
			t.Fatalf("direct storage zero retained %q:\n%s", forbidden, artifacts.printed)
		}
	}
	runner := filepath.Join(workingDirectory, "runner.ts")
	writeProgramFile(t, runner, `import "./program.js";
import { Run } from "`+artifacts.sourceModule+`";

console.log(Run());
`)
	writeProgramFile(
		t,
		filepath.Join(workingDirectory, "package.json"),
		"{\"type\":\"module\"}\n",
	)
	paths := append(artifacts.paths, runner)
	waveThreeTypecheck(t, workingDirectory, paths)
	targetOutput := runProgram(
		t,
		workingDirectory,
		"node",
		filepath.Join(workingDirectory, "out", "runner.js"),
	)
	goOutput := executeStorageZeroGo(t, project, workingDirectory)
	if targetOutput != goOutput {
		t.Fatalf("TypeScript output = %q, Go output = %q", targetOutput, goOutput)
	}
}

func executeStorageZeroGo(
	t *testing.T,
	project string,
	workingDirectory string,
) string {
	t.Helper()
	runner := filepath.Join(workingDirectory, "go-runner")
	writeProgramFile(t, filepath.Join(runner, "go.mod"), fmt.Sprintf(`module example.com/runner

go 1.26.4

require example.com/storagezero v0.0.0
replace example.com/storagezero => %s
`, filepath.ToSlash(project)))
	writeProgramFile(t, filepath.Join(runner, "main.go"), `package main

import (
	"fmt"
	"example.com/storagezero"
)

func main() { fmt.Println(storagezero.Run()) }
`)
	return runProgram(
		t,
		runner,
		filepath.Join(runtime.GOROOT(), "bin", "go"),
		"run",
		".",
	)
}
