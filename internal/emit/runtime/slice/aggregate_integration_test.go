package slice_test

import (
	"context"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tsoniclang/gotots/internal/emit"
	"github.com/tsoniclang/gotots/internal/load"
)

func TestAggregateSliceOperationsPrintTypecheckAndMatchGo(t *testing.T) {
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
			emission := compileAggregateSliceFixture(t, testCase.options)
			directory := t.TempDir()
			paths, module, printed := materialize(t, directory, emission)
			for _, helper := range []string{
				"goSliceMakeWith",
				"goSliceAppendWith",
				"goSliceCopyWith",
			} {
				if strings.Count(printed.runtime, "function "+helper) != 1 {
					t.Fatalf(
						"aggregate runtime helper %s is not emitted exactly once:\n%s",
						helper,
						printed.runtime,
					)
				}
			}
			for _, forbidden := range []string{
				": any",
				": unknown",
				".call(",
				".apply(",
				".bind(",
				"import(",
			} {
				if strings.Contains(printed.source, forbidden) ||
					strings.Contains(printed.runtime, forbidden) {
					t.Fatalf("aggregate slice artifact contains %q", forbidden)
				}
			}
			runner := filepath.Join(directory, "runner.ts")
			writeFile(t, runner, `import "./program.js";
import * as values from "`+module+`";

console.log(values.MakeZerosAreFresh());
console.log(values.CapacityZerosAreFresh());
console.log(values.LiteralCopiesValues());
console.log(values.SparseLiteralZerosAreFresh());
console.log(values.AppendReuseAliasesBackingAndCopiesArgument());
console.log(values.AppendReallocationCopiesExisting());
console.log(values.AppendTailZerosAreFresh());
console.log(values.CopyDistinctCopiesValues());
console.log(values.CopyOverlapSnapshotsValues());
console.log(values.AddressTargetsBackingElement());
console.log(values.ArrayElementsCopyOnAppend());
`)
			writeFile(t, filepath.Join(directory, "package.json"), "{\"type\":\"module\"}\n")
			paths = append(paths, runner)
			typecheck(t, directory, paths)
			targetOutput := run(
				t,
				directory,
				"node",
				filepath.Join(directory, "out", "runner.js"),
			)
			goOutput := executeAggregateSliceGo(t, directory)
			if targetOutput != goOutput {
				t.Fatalf(
					"aggregate-slice output differs\nTypeScript:\n%s\nGo:\n%s",
					targetOutput,
					goOutput,
				)
			}
		})
	}
}

func TestScalarSliceArtifactHasNoAggregateOperationSurface(t *testing.T) {
	emission := compileFixture(t)
	directory := t.TempDir()
	_, _, printed := materialize(t, directory, emission)
	for _, forbidden := range []string{
		"goSliceMakeWith",
		"goSliceAppendWith",
		"goSliceCopyWith",
		"static makeWith",
		"appendWith(",
		"static copyWith",
	} {
		if strings.Contains(printed.runtime, forbidden) {
			t.Fatalf("scalar slice runtime contains %q:\n%s", forbidden, printed.runtime)
		}
	}
}

func compileAggregateSliceFixture(
	t *testing.T,
	options emit.Options,
) emit.ProgramEmission {
	t.Helper()
	program, err := load.Load(context.Background(), load.Request{
		Directory: aggregateSliceDirectory(),
		Pattern:   ".",
	})
	if err != nil {
		t.Fatal(err)
	}
	roots, err := emit.ExportedAPIRoots(program.Roots()[0])
	if err != nil {
		t.Fatal(err)
	}
	emission, err := emit.CompileWithOptions(program, roots, options)
	if err != nil {
		t.Fatal(err)
	}
	return emission
}

func executeAggregateSliceGo(t *testing.T, directory string) string {
	t.Helper()
	modulePath, err := filepath.Abs(aggregateSliceDirectory())
	if err != nil {
		t.Fatal(err)
	}
	runnerDirectory := filepath.Join(directory, "go-runner-aggregate-slice")
	writeFile(t, filepath.Join(runnerDirectory, "go.mod"), `module example.com/runner

go 1.26.4

require example.com/aggregateslice v0.0.0

replace example.com/aggregateslice => `+
		filepath.ToSlash(modulePath)+"\n")
	writeFile(t, filepath.Join(runnerDirectory, "main.go"), `package main

import (
	"fmt"
	values "example.com/aggregateslice"
)

func main() {
	fmt.Println(values.MakeZerosAreFresh())
	fmt.Println(values.CapacityZerosAreFresh())
	fmt.Println(values.LiteralCopiesValues())
	fmt.Println(values.SparseLiteralZerosAreFresh())
	fmt.Println(values.AppendReuseAliasesBackingAndCopiesArgument())
	fmt.Println(values.AppendReallocationCopiesExisting())
	fmt.Println(values.AppendTailZerosAreFresh())
	fmt.Println(values.CopyDistinctCopiesValues())
	fmt.Println(values.CopyOverlapSnapshotsValues())
	fmt.Println(values.AddressTargetsBackingElement())
	fmt.Println(values.ArrayElementsCopyOnAppend())
}
`)
	return run(t, runnerDirectory, "go", "run", ".")
}

func aggregateSliceDirectory() string {
	return filepath.Join(
		repositoryRoot(),
		"testdata",
		"constructs",
		"value",
		"slice",
		"aggregate",
	)
}
