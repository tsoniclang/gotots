package arrayvalue_test

import (
	"context"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tsoniclang/gotots/internal/emit"
	"github.com/tsoniclang/gotots/internal/load"
)

func TestAggregateArrayZeroCopyLiteralEqualityAndAddressMatchGo(
	t *testing.T,
) {
	for _, testCase := range []struct {
		name    string
		options emit.Options
		suffix  string
	}{
		{name: "number", options: emit.DefaultOptions()},
		{
			name: "bigint",
			options: emit.Options{
				IntegerRepresentation: emit.IntegerRepresentationBigInt,
				EvaluationOrder:       emit.EvaluationOrderPreserveGo,
			},
			suffix: "n",
		},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			emission := compileAggregateArrayFixture(t, testCase.options)
			directory := t.TempDir()
			target := materializeArrayProgram(t, directory, emission)
			runtime := target.printed["runtime/array.ts"]
			for _, helper := range []string{
				"goArrayZeroWith",
				"goArrayLiteralWith",
				"goArrayCopyWith",
			} {
				if strings.Count(runtime, "function "+helper) != 1 {
					t.Fatalf(
						"aggregate runtime helper %s is not emitted exactly once:\n%s",
						helper,
						runtime,
					)
				}
			}
			for path, artifact := range target.printed {
				for _, forbidden := range []string{
					": any",
					": unknown",
					".call(",
					".apply(",
					".bind(",
					"import(",
				} {
					if strings.Contains(artifact, forbidden) {
						t.Fatalf("%s contains forbidden %q:\n%s", path, forbidden, artifact)
					}
				}
			}
			runner := filepath.Join(directory, "runner.ts")
			writeFile(t, runner, aggregateArrayRunner(target, testCase.suffix))
			writeFile(t, filepath.Join(directory, "package.json"), "{\"type\":\"module\"}\n")
			target.paths = append(target.paths, runner)
			if err := compileTypeScript(t, directory, target.paths); err != nil {
				t.Fatal(err)
			}
			typeScriptOutput := run(
				t,
				directory,
				"node",
				filepath.Join(directory, "out", "runner.js"),
			)
			goOutput := runAggregateArrayGo(t, directory)
			if typeScriptOutput != goOutput {
				t.Fatalf(
					"aggregate-array output differs\nTypeScript:\n%s\nGo:\n%s",
					typeScriptOutput,
					goOutput,
				)
			}
		})
	}
}

func TestScalarArrayArtifactHasNoAggregateOperationSurface(t *testing.T) {
	directory := t.TempDir()
	target := materializeArrayProgram(t, directory, compileArrayFixture(t))
	runtime := target.printed["runtime/array.ts"]
	for _, forbidden := range []string{
		"goArrayZeroWith",
		"goArrayLiteralWith",
		"goArrayCopyWith",
	} {
		if strings.Contains(runtime, forbidden) {
			t.Fatalf("scalar array runtime contains %q:\n%s", forbidden, runtime)
		}
	}
}

func compileAggregateArrayFixture(
	t *testing.T,
	options emit.Options,
) emit.ProgramEmission {
	t.Helper()
	program, err := load.Load(context.Background(), load.Request{
		Directory: aggregateArrayDirectory(),
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

func aggregateArrayRunner(
	target materializedProgram,
	suffix string,
) string {
	return `import "` + target.programInit + `";
import * as values from "` + target.sourceModule + `";

console.log(values.ZeroFresh().map(String).join(" "));
console.log(values.CopyIsDeep().map(String).join(" "));
console.log(values.NamedCopyIsDeep().map(String).join(" "));
console.log(values.NestedCopyIsDeep().map(String).join(" "));
console.log(values.SparseLiteralZerosAreFresh().map(String).join(" "));
const left = values.NewBoxes(1` + suffix + `, 2` + suffix + `);
const right = values.NewBoxes(1` + suffix + `, 2` + suffix + `);
console.log(values.Equal(left, right));
const pointed = values.PointerStore(left);
console.log(String(values.First(pointed)), String(values.Second(pointed)));
const [original, copied] = values.CallIsolation(right);
console.log(
    String(values.First(original)),
    String(values.First(copied)),
    String(values.Second(original)),
);
`
}

func runAggregateArrayGo(t *testing.T, directory string) string {
	t.Helper()
	runnerDirectory := filepath.Join(directory, "go-runner-aggregate-array")
	writeFile(t, filepath.Join(runnerDirectory, "go.mod"), `module example.com/runner

go 1.26.4

require example.com/aggregatearray v0.0.0

replace example.com/aggregatearray => `+
		filepath.ToSlash(aggregateArrayDirectory())+"\n")
	writeFile(t, filepath.Join(runnerDirectory, "main.go"), `package main

import (
	"fmt"
	values "example.com/aggregatearray"
)

func main() {
	fmt.Println(values.ZeroFresh())
	fmt.Println(values.CopyIsDeep())
	fmt.Println(values.NamedCopyIsDeep())
	fmt.Println(values.NestedCopyIsDeep())
	fmt.Println(values.SparseLiteralZerosAreFresh())
	left := values.Boxes{{Value: 1}, {Value: 2}}
	right := values.Boxes{{Value: 1}, {Value: 2}}
	fmt.Println(values.Equal(left, right))
	pointed := values.PointerStore(left)
	fmt.Println(values.First(pointed), values.Second(pointed))
	original, copied := values.CallIsolation(right)
	fmt.Println(values.First(original), values.First(copied), values.Second(original))
}
`)
	return run(t, runnerDirectory, "go", "run", ".")
}

func aggregateArrayDirectory() string {
	return filepath.Join(
		repositoryRoot(),
		"testdata",
		"constructs",
		"value",
		"array",
		"aggregate",
	)
}
