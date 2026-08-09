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
				"goSliceLiteralWith",
				"goSliceAppendWith",
				"goSliceAppendSliceWith",
				"goSliceCopyWith",
			} {
				if strings.Contains(printed.runtime, helper) {
					t.Fatalf(
						"aggregate runtime retains semantic callback helper %s:\n%s",
						helper,
						printed.runtime,
					)
				}
			}
			if strings.Count(printed.runtime, "function goSliceAllocate") != 1 {
				t.Fatalf("aggregate storage allocator is not emitted exactly once:\n%s", printed.runtime)
			}
			for _, shared := range []string{
				"const resolvedCapacity = globalThis.Number(capacity ?? numericLength);",
				"const nextCapacity = RuntimeSlice.$grownCapacity(this.capacity, newLength);",
				"return this.slice(0, length, null);",
				"return addressOf<T>(backing[this.offset + numericIndex]);",
			} {
				if !strings.Contains(printed.runtime, shared) {
					t.Fatalf("aggregate slice runtime lacks shared operation %q:\n%s", shared, printed.runtime)
				}
			}
			if strings.Count(printed.runtime, "while (nextCapacity < length)") != 1 {
				t.Fatalf("aggregate slice runtime duplicates capacity growth:\n%s", printed.runtime)
			}
			for _, forbidden := range []string{
				": any",
				": unknown",
				".call(",
				".apply(",
				".bind(",
				"import(",
				"zero: () =>",
				"copyValue:",
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
console.log(values.AppendSpreadCopiesValues());
console.log(values.AppendSpreadOverlapSnapshotsValues());
console.log(values.CopyDistinctCopiesValues());
console.log(values.CopyOverlapSnapshotsValues());
console.log(values.ArrayElementsCopyOnAppend());
console.log(values.ElidedNestedLiterals());
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

func TestScalarSliceArtifactHasNoSemanticCallbackSurface(t *testing.T) {
	emission := compileFixture(t)
	directory := t.TempDir()
	_, _, printed := materialize(t, directory, emission)
	for _, forbidden := range []string{
		"goSliceMakeWith",
		"goSliceNilWith",
		"goSliceLiteralWith",
		"goSliceAppendWith",
		"goSliceCopyWith",
		"static makeWith",
		"static nilWith",
		"static literalWith",
		"appendWith(",
		"static copyWith",
	} {
		if strings.Contains(printed.runtime, forbidden) {
			t.Fatalf("scalar slice runtime contains %q:\n%s", forbidden, printed.runtime)
		}
	}
}

func TestAppendSpreadSurfaceIsDemandedByTheExactValueFamily(t *testing.T) {
	without := compileSliceSource(t, `package demand

func Use(values []int32) []int32 {
	return append(values, 1)
}
`)
	withScalarSpread := compileSliceSource(t, `package demand

func Use(values []int32) []int32 {
	return append(values, values...)
}
`)
	withAggregateSpread := compileSliceSource(t, `package demand

type Box struct {
	Value int32
}

func Use(values []Box) []Box {
	return append(values, values...)
}
`)
	_, _, withoutPrinted := materialize(t, t.TempDir(), without)
	_, _, scalarPrinted := materialize(t, t.TempDir(), withScalarSpread)
	_, _, aggregatePrinted := materialize(t, t.TempDir(), withAggregateSpread)
	for _, fragment := range []string{
		"appendSlice(zero: T, source: RuntimeSlice<T>): RuntimeSlice<T>",
		"export function goSliceAppendSlice<T>",
	} {
		if strings.Contains(withoutPrinted.runtime, fragment) {
			t.Fatalf("ordinary append emitted undemanded %q", fragment)
		}
		if strings.Count(scalarPrinted.runtime, fragment) != 1 {
			t.Fatalf(
				"scalar spread count(%q) = %d, want one:\n%s",
				fragment,
				strings.Count(scalarPrinted.runtime, fragment),
				scalarPrinted.runtime,
			)
		}
		if strings.Contains(aggregatePrinted.runtime, fragment) {
			t.Fatalf("aggregate spread emitted scalar runtime operation %q", fragment)
		}
	}
	if !strings.Contains(aggregatePrinted.runtime, "static $allocate<T>") {
		t.Fatalf(
			"aggregate spread did not demand structural storage:\n%s",
			aggregatePrinted.runtime,
		)
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
	fmt.Println(values.AppendSpreadCopiesValues())
	fmt.Println(values.AppendSpreadOverlapSnapshotsValues())
	fmt.Println(values.CopyDistinctCopiesValues())
	fmt.Println(values.CopyOverlapSnapshotsValues())
	fmt.Println(values.ArrayElementsCopyOnAppend())
	fmt.Println(values.ElidedNestedLiterals())
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
