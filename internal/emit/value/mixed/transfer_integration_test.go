package mixed_test

import (
	"context"
	"fmt"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/tsoniclang/gotots/internal/emit"
	"github.com/tsoniclang/gotots/internal/load"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
	runtimefixture "github.com/tsoniclang/gotots/internal/testfixture/gototsruntime"
)

func TestAssignableDefinedValuesTransferAtTypedBoundaries(t *testing.T) {
	sourceDirectory := t.TempDir()
	writeFile(
		t,
		filepath.Join(sourceDirectory, "go.mod"),
		"module example.com/transfer\n\ngo 1.26.4\n",
	)
	writeFile(t, filepath.Join(sourceDirectory, "api.go"), `package api

type Bytes []byte
type Counts map[string]int32
type Cell *int32
type Callback func(int32) int32
type Pair [2]int32
type Key string

func changeBytes(value []byte) {
	value[0]++
}

func asBytes(value []byte) Bytes {
	return value
}

func changeCounts(value map[string]int32) {
	value["value"]++
}

func asCounts(value map[string]int32) Counts {
	return value
}

func changeCell(value *int32) {
	*value++
}

func asCell(value *int32) Cell {
	return value
}

func call(value func(int32) int32) int32 {
	return value(4)
}

func asCallback(value func(int32) int32) Callback {
	return value
}

func changePair(value [2]int32) [2]int32 {
	value[0]++
	return value
}

func asPair(value [2]int32) Pair {
	return value
}

func keyLength(value Key) int32 {
	return int32(len(value))
}

func rangeDefinedKey() int32 {
	values := map[Key]int32{Key("abc"): 1}
	for key := range values {
		return keyLength(key)
	}
	return 0
}

func channelTransfer() int32 {
	namedValues := make(chan Bytes, 1)
	raw := []byte{15}
	namedValues <- raw
	var receivedRaw []byte = <-namedValues

	rawValues := make(chan []byte, 1)
	named := Bytes{16}
	select {
	case rawValues <- named:
	}
	var receivedNamed Bytes
	select {
	case receivedNamed = <-rawValues:
	}
	return int32(receivedRaw[0] + receivedNamed[0])
}

func PointerExercise() int32 {
	number := int32(9)
	cell := asCell(&number)
	changeCell(cell)
	return *cell
}

func Exercise() int32 {
	bytes := Bytes{1}
	changeBytes(bytes)
	rawBytes := []byte{3}
	namedBytes := asBytes(rawBytes)
	namedBytes[0]++

	counts := Counts{"value": 5}
	changeCounts(counts)
	rawCounts := map[string]int32{"value": 7}
	namedCounts := asCounts(rawCounts)
	namedCounts["value"]++

	callback := Callback(func(value int32) int32 { return value + 10 })
	rawCallbackResult := call(callback)
	namedCallback := asCallback(func(value int32) int32 { return value + 20 })
	namedCallbackResult := namedCallback(4)

	pair := Pair{11, 12}
	changedPair := changePair(pair)
	rawPair := [2]int32{13, 14}
	namedPair := asPair(rawPair)
	namedPair[0]++

	return int32(bytes[0]) +
		int32(rawBytes[0]) +
		counts["value"] +
		rawCounts["value"] +
		rawCallbackResult +
		namedCallbackResult +
		pair[0] +
		changedPair[0] +
		rawPair[0] +
		namedPair[0] +
		rangeDefinedKey() +
		channelTransfer()
}
`)
	program, err := load.Load(context.Background(), load.Request{
		Directory: sourceDirectory,
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
	emission, err := emit.CompileWithOptions(program, roots, options)
	if err != nil {
		t.Fatal(err)
	}

	targetDirectory := t.TempDir()
	artifacts := materialize(t, emission, targetDirectory)
	outputPath := strings.TrimPrefix(
		strings.TrimSuffix(artifacts.apiModule, ".js"),
		"./",
	) + ".ts"
	source := artifacts.printed[outputPath]
	if !strings.Contains(source, ".$value") ||
		!strings.Contains(source, "new Bytes(") ||
		!strings.Contains(source, "new Counts(") ||
		!strings.Contains(source, "new Cell(") ||
		!strings.Contains(source, "new Callback(") ||
		!strings.Contains(source, "new Pair(") ||
		!strings.Contains(source, "new Key(") {
		t.Fatalf("defined transfer projection/construction is incomplete:\n%s", source)
	}

	runnerPath := filepath.Join(targetDirectory, "runner.ts")
	writeFile(t, runnerPath, `import * as values from "`+artifacts.apiModule+`";
console.log(String(values.Exercise()));
`)
	writeFile(
		t,
		filepath.Join(targetDirectory, "package.json"),
		"{\"type\":\"module\"}\n",
	)
	typecheck(t, targetDirectory, artifacts.paths, runnerPath)
	targetOutput := run(
		t,
		targetDirectory,
		"node",
		filepath.Join(targetDirectory, "out", "runner.js"),
	)
	goOutput := executeTransferGo(t, sourceDirectory, targetDirectory)
	if targetOutput != goOutput {
		t.Fatalf(
			"TypeScript output = %q, Go output = %q",
			targetOutput,
			goOutput,
		)
	}
	assertTransferMutationsFail(
		t,
		targetDirectory,
		artifacts.paths,
		runnerPath,
		filepath.Join(targetDirectory, filepath.FromSlash(outputPath)),
		source,
	)
}

func assertTransferMutationsFail(
	t *testing.T,
	workingDirectory string,
	paths []string,
	runnerPath string,
	sourcePath string,
	source string,
) {
	t.Helper()
	for _, mutation := range []struct {
		name string
		from string
		to   string
	}{
		{
			name: "source projection",
			from: "changeBytes(bytes.$value);",
			to:   "changeBytes(bytes);",
		},
		{
			name: "destination construction",
			from: "return new Bytes(value);",
			to:   "return value;",
		},
	} {
		t.Run("mutation/"+mutation.name, func(t *testing.T) {
			mutated := strings.Replace(source, mutation.from, mutation.to, 1)
			if mutated == source {
				t.Fatalf("mutation %q changed no generated source", mutation.name)
			}
			writeFile(t, sourcePath, mutated)
			t.Cleanup(func() {
				writeFile(t, sourcePath, source)
			})
			if err := transferTypecheck(
				workingDirectory,
				paths,
				runnerPath,
			); err == nil {
				t.Fatalf("mutation %q passed strict typechecking", mutation.name)
			}
			writeFile(t, sourcePath, source)
		})
	}
}

func transferTypecheck(
	workingDirectory string,
	paths []string,
	runnerPath string,
) error {
	arguments := []string{
		"--target", "es2022",
		"--module", "nodenext",
		"--moduleResolution", "nodenext",
		"--strict",
		"--noEmit",
	}
	arguments = append(arguments, paths...)
	arguments = append(arguments, runnerPath)
	if err := runtimefixture.InstallResolution(workingDirectory, filepath.Join(workingDirectory, "out")); err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	return tsgo.Compile(
		ctx,
		repositoryRoot(),
		workingDirectory,
		arguments,
	)
}

func executeTransferGo(
	t *testing.T,
	sourceDirectory string,
	targetDirectory string,
) string {
	t.Helper()
	runnerDirectory := filepath.Join(targetDirectory, "go-transfer-runner")
	writeFile(t, filepath.Join(runnerDirectory, "go.mod"), fmt.Sprintf(`module example.com/runner

go 1.26.4

require example.com/transfer v0.0.0

replace example.com/transfer => %s
`, filepath.ToSlash(sourceDirectory)))
	writeFile(t, filepath.Join(runnerDirectory, "main.go"), `package main

import (
	"fmt"
	values "example.com/transfer"
)

func main() {
	fmt.Println(values.Exercise())
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
