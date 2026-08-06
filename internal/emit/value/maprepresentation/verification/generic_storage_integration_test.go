package maprepresentation_test

import (
	"context"
	"fmt"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/tsoniclang/gotots/internal/load"
)

func TestNamedMapGenericStorageFacetsTypecheckAndExecuteDifferentially(
	t *testing.T,
) {
	project := t.TempDir()
	writeFile(t, filepath.Join(project, "go.mod"), `module example.com/genericnamedmap

go 1.26.4
`)
	writeFile(t, filepath.Join(project, "source.go"), `package genericnamedmap

import "example.com/genericnamedmap/generic"

type Values map[string]int32

func (values Values) Clone() Values {
	result := make(Values, len(values))
	for key, value := range values {
		result[key] = value
	}
	return result
}

func RoundTrip(value int32) int32 {
	holder := generic.NewHolder[string, Values]()
	result := holder.RoundTrip("value", Values{"answer": value}).Value()
	return result["answer"]
}
`)
	writeFile(t, filepath.Join(project, "generic", "holder.go"), `package generic

type Cloneable[T any] interface {
	Clone() T
}

type Holder[K comparable, V Cloneable[V]] struct {
	values map[K]V
}

type Entry[K comparable, V Cloneable[V]] struct {
	key K
	value V
}

func NewHolder[K comparable, V Cloneable[V]]() *Holder[K, V] {
	return &Holder[K, V]{values: make(map[K]V)}
}

func (holder *Holder[K, V]) RoundTrip(key K, value V) *Entry[K, V] {
	holder.values[key] = value
	return &Entry[K, V]{key: key, value: holder.values[key]}
}

func (entry *Entry[K, V]) Value() V {
	return entry.value
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
	runnerPath := filepath.Join(workingDirectory, "runner.ts")
	writeFile(t, runnerPath, `import { RoundTrip } from "`+
		artifacts.module(t, "source.ts")+`";
import "./program.js";

console.log(RoundTrip(41));
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
	goOutput := runGenericNamedMapGo(t, project, workingDirectory)
	if targetOutput != goOutput {
		t.Fatalf(
			"TypeScript output = %q, Go output = %q",
			targetOutput,
			goOutput,
		)
	}
	assertNamedMapGenericStorageFacets(t, artifacts, workingDirectory)
}

func assertNamedMapGenericStorageFacets(
	t *testing.T,
	artifacts materialized,
	workingDirectory string,
) {
	t.Helper()
	projectionPath, projection := uniqueArtifactContaining(
		t,
		artifacts,
		"$0: Values__from_genericnamedmap",
		"): GoMapValue<gostring, int32>",
		"return $0.$value;",
	)
	wrappingPath, wrapping := uniqueArtifactContaining(
		t,
		artifacts,
		"$0: GoMapValue<gostring, int32>",
		"): Values__from_genericnamedmap",
		"return new Values__from_genericnamedmap($0);",
	)
	for _, testCase := range []struct {
		name   string
		path   string
		source string
		from   string
		to     string
	}{
		{
			name:   "logical map returned as storage",
			path:   projectionPath,
			source: projection,
			from:   "return $0.$value;",
			to:     "return $0;",
		},
		{
			name:   "storage map returned as logical",
			path:   wrappingPath,
			source: wrapping,
			from:   "return new Values__from_genericnamedmap($0);",
			to:     "return $0;",
		},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			mutated := strings.Replace(testCase.source, testCase.from, testCase.to, 1)
			if mutated == testCase.source {
				t.Fatalf("mutation source %q is absent", testCase.from)
			}
			writeFile(t, testCase.path, mutated)
			if err := strictTypecheckResult(
				artifacts,
				workingDirectory,
				"",
			); err == nil {
				t.Fatal("storage projection mutation passed strict typechecking")
			}
			writeFile(t, testCase.path, testCase.source)
		})
	}
}

func uniqueArtifactContaining(
	t *testing.T,
	artifacts materialized,
	required ...string,
) (string, string) {
	t.Helper()
	var matchedPath string
	var matchedSource string
	for _, path := range artifacts.paths {
		source := readFile(t, path)
		matches := true
		for _, text := range required {
			if !strings.Contains(source, text) {
				matches = false
				break
			}
		}
		if !matches {
			continue
		}
		if matchedPath != "" {
			t.Fatalf(
				"artifact requirements %q match both %s and %s",
				required,
				matchedPath,
				path,
			)
		}
		matchedPath = path
		matchedSource = source
	}
	if matchedPath == "" {
		t.Fatalf("no artifact contains %q", required)
	}
	return matchedPath, matchedSource
}

func runGenericNamedMapGo(
	t *testing.T,
	project string,
	workingDirectory string,
) string {
	t.Helper()
	runnerDirectory := filepath.Join(workingDirectory, "go-generic-runner")
	writeFile(t, filepath.Join(runnerDirectory, "go.mod"), fmt.Sprintf(`module example.com/runner

go 1.26.4

require example.com/genericnamedmap v0.0.0

replace example.com/genericnamedmap => %s
`, filepath.ToSlash(project)))
	writeFile(t, filepath.Join(runnerDirectory, "main.go"), `package main

import (
	"fmt"

	values "example.com/genericnamedmap"
)

func main() {
	fmt.Println(values.RoundTrip(41))
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
