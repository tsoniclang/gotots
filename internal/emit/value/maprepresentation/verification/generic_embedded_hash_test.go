package maprepresentation_test

import (
	"context"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/tsoniclang/gotots/internal/load"
)

func TestEmbeddedGenericHashProjectsConcreteCapability(t *testing.T) {
	project := t.TempDir()
	writeFile(t, filepath.Join(project, "go.mod"), `module example.com/embeddedhash

go 1.26.4
`)
	writeFile(t, filepath.Join(project, "source.go"), `package embeddedhash

type entry[K comparable, V any] struct {
	Key K
	Value V
}

type Wrapped[K comparable, V any] struct {
	entry[K, V]
}

type Item struct {
	Value int32
}

func Facts() string {
	item := Item{Value: 7}
	left := Wrapped[string, Item]{
		entry: entry[string, Item]{Key: "key", Value: item},
	}
	right := Wrapped[string, Item]{
		entry: entry[string, Item]{Key: "key", Value: item},
	}
	values := map[Wrapped[string, Item]]string{left: "found"}
	if left != right {
		return "not equal"
	}
	return values[right]
}
`)
	loaded, err := load.One(context.Background(), load.Request{
		Directory: project,
		Pattern:   ".",
	})
	if err != nil {
		t.Fatal(err)
	}
	emission, err := compileExportedResult(loaded)
	if err != nil {
		t.Fatal(err)
	}
	workingDirectory := t.TempDir()
	artifacts := materialize(t, emission, workingDirectory)
	_, sourceArtifact := uniqueArtifactContaining(
		t,
		artifacts,
		"export class Wrapped<K, V>",
		"static $hash<K, V>",
	)
	for _, required := range []string{
		"static $hash<K, V>($go$hash$",
		": ($0: entry<K, V>) => uint32, $source: Wrapped<K, V>): number {",
		"entry.$fromStorage<K, V>($source.$storage.entry)",
	} {
		if !strings.Contains(sourceArtifact, required) {
			t.Fatalf(
				"embedded generic hash artifact lacks %q:\n%s",
				required,
				sourceArtifact,
			)
		}
	}
	for _, forbidden := range []string{
		": any",
		" as any",
		"unknown",
		".call(",
		".apply(",
		".bind(",
	} {
		if strings.Contains(sourceArtifact, forbidden) {
			t.Fatalf(
				"embedded generic hash artifact contains %q:\n%s",
				forbidden,
				sourceArtifact,
			)
		}
	}
	runnerPath := filepath.Join(workingDirectory, "runner.ts")
	writeFile(t, runnerPath, `import "./program.js";
import { Facts } from "`+artifacts.module(t, "source.ts")+`";

console.log(Facts());
`)
	writeFile(
		t,
		filepath.Join(workingDirectory, "package.json"),
		"{\"type\":\"module\"}\n",
	)
	strictTypecheckWithRunner(t, artifacts, workingDirectory, runnerPath)
	targetOutput := run(
		t,
		workingDirectory,
		"node",
		filepath.Join(workingDirectory, "out", "runner.js"),
	)

	runnerDirectory := filepath.Join(project, "cmd", "compare")
	writeFile(t, filepath.Join(runnerDirectory, "main.go"), `package main

import (
	"fmt"

	fixture "example.com/embeddedhash"
)

func main() {
	fmt.Println(fixture.Facts())
}
`)
	goOutput := run(
		t,
		runnerDirectory,
		filepath.Join(runtime.GOROOT(), "bin", "go"),
		"run",
		".",
	)
	if targetOutput != goOutput {
		t.Fatalf(
			"embedded generic hash differential:\nGo:\n%s\nTypeScript:\n%s",
			goOutput,
			targetOutput,
		)
	}
}
