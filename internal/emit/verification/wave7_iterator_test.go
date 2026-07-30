package emit_test

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/tsoniclang/gotots/internal/emit"
	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/load"
)

func TestWaveSevenIteratorRangesExecuteDifferentially(t *testing.T) {
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
			program, err := load.Load(context.Background(), load.Request{
				Directory: waveSevenGenericDirectory(),
				Pattern:   ".",
			})
			if err != nil {
				t.Fatal(err)
			}
			scope := program.Roots()[0].Types().Scope()
			var roots []emit.Root
			for _, name := range []string{
				"AuditIteratorRanges",
				"BreaksBadIterator",
				"CallsYieldAfterExit",
				"IteratorDeferredReturn",
				"IteratorMultipleReturn",
				"IteratorNamedReturn",
				"IteratorNestedReturn",
				"IteratorReturnBoundary",
				"IteratorSelectiveReturn",
				"RangesNilIterator",
				"ReturnsBadIterator",
			} {
				root, rootErr := emit.NewRoot(scope.Lookup(name))
				if rootErr != nil {
					t.Fatal(rootErr)
				}
				roots = append(roots, root)
			}
			emission, err := emit.CompileWithOptions(
				program,
				roots,
				testCase.options,
			)
			if err != nil {
				t.Fatal(err)
			}
			workingDirectory := t.TempDir()
			artifacts := materializeArtifacts(t, emission, workingDirectory)
			assertIteratorRangeShape(t, artifacts.printed)
			sourceModule := sourceModuleForExport(
				t,
				artifacts,
				workingDirectory,
				"AuditIteratorRanges",
			)
			runner := filepath.Join(workingDirectory, "runner.ts")
			writeProgramFile(t, runner, `import "./program.js";
import {
    AuditIteratorRanges,
    BreaksBadIterator,
    CallsYieldAfterExit,
    IteratorDeferredReturn,
    IteratorMultipleReturn,
    IteratorNamedReturn,
    IteratorNestedReturn,
    IteratorReturnBoundary,
    IteratorSelectiveReturn,
    RangesNilIterator,
    ReturnsBadIterator,
} from "`+sourceModule+`";

const values = AuditIteratorRanges();
const output: string[] = [];
for (let index = 0; index < values.length; index++) {
    output.push(String(values.get(index)));
}
const panics = (action: () => void): boolean => {
    try {
        action();
        return false;
    } catch {
        return true;
    }
};
console.log(output.join(" "));
console.log(String(IteratorReturnBoundary()));
console.log(String(IteratorSelectiveReturn()));
console.log(IteratorMultipleReturn().map(String).join(" "));
console.log(String(IteratorNamedReturn()));
console.log(String(IteratorNestedReturn()));
console.log(String(IteratorDeferredReturn()));
console.log(String(panics(BreaksBadIterator)));
console.log(String(panics(ReturnsBadIterator)));
console.log(String(panics(CallsYieldAfterExit)));
console.log(String(panics(() => { RangesNilIterator(undefined); })));
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
			goOutput := executeWaveSevenIteratorGo(t, workingDirectory)
			if targetOutput != goOutput {
				t.Fatalf(
					"iterator range differs\nTypeScript:\n%s\nGo:\n%s",
					targetOutput,
					goOutput,
				)
			}
		})
	}
}

func TestWaveSevenIteratorLabelControlFailsAtExactBoundary(t *testing.T) {
	program, err := load.Load(context.Background(), load.Request{
		Directory: waveSevenGenericDirectory(),
		Pattern:   ".",
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"IteratorLabelBoundary"} {
		t.Run(name, func(t *testing.T) {
			root, rootErr := emit.NewRoot(
				program.Roots()[0].Types().Scope().Lookup(name),
			)
			if rootErr != nil {
				t.Fatal(rootErr)
			}
			_, compileErr := emit.Compile(
				program,
				[]emit.Root{root},
			)
			var unsupported *api.UnsupportedError
			if !errors.As(compileErr, &unsupported) ||
				unsupported.Category != api.CategoryStatement {
				t.Fatalf(
					"boundary error = %#v, want statement UnsupportedError",
					compileErr,
				)
			}
		})
	}
}

func assertIteratorRangeShape(t *testing.T, printed string) {
	t.Helper()
	for _, required := range []string{
		"__gotots_range_state_",
		"range function continued iteration after function for loop body returned false",
		"range function continued iteration after loop body panic",
		"range function continued iteration after whole loop exit",
		"range function recovered a loop body panic and did not resume panicking",
		"export function GenericIteratorSum<T>",
		"export function GenericIteratorCopy<T>",
	} {
		if !strings.Contains(printed, required) {
			t.Fatalf("iterator artifact lacks %q:\n%s", required, printed)
		}
	}
	for _, forbidden := range []string{
		".call(",
		".apply(",
		".bind(",
		" as any",
		" as unknown",
	} {
		if strings.Contains(printed, forbidden) {
			t.Fatalf("iterator artifact contains %q:\n%s", forbidden, printed)
		}
	}
	if strings.Count(printed, "export function GenericIteratorSum<T>") != 1 {
		t.Fatalf("generic iterator body was duplicated:\n%s", printed)
	}
	if count := strings.Count(
		printed,
		"let __gotots_range_return_",
	); count != 8 {
		t.Fatalf(
			"iterator-return carrier declarations = %d, want 8:\n%s",
			count,
			printed,
		)
	}
}

func executeWaveSevenIteratorGo(
	t *testing.T,
	workingDirectory string,
) string {
	t.Helper()
	modulePath, err := filepath.Abs(waveSevenGenericDirectory())
	if err != nil {
		t.Fatal(err)
	}
	runnerDirectory := filepath.Join(workingDirectory, "go-iterator-wave7")
	writeProgramFile(t, filepath.Join(runnerDirectory, "go.mod"), fmt.Sprintf(
		`module example.com/runner

go 1.26.4

require example.com/wave7generics v0.0.0

replace example.com/wave7generics => %s
`,
		filepath.ToSlash(modulePath),
	))
	writeProgramFile(t, filepath.Join(runnerDirectory, "main.go"), `package main

import (
	"fmt"

	values "example.com/wave7generics"
)

func main() {
	for index, value := range values.AuditIteratorRanges() {
		if index != 0 {
			fmt.Print(" ")
		}
		fmt.Print(value)
	}
	fmt.Println()
	fmt.Println(values.IteratorReturnBoundary())
	fmt.Println(values.IteratorSelectiveReturn())
	fmt.Println(values.IteratorMultipleReturn())
	fmt.Println(values.IteratorNamedReturn())
	fmt.Println(values.IteratorNestedReturn())
	fmt.Println(values.IteratorDeferredReturn())
	fmt.Println(panics(values.BreaksBadIterator))
	fmt.Println(panics(func() { values.ReturnsBadIterator() }))
	fmt.Println(panics(values.CallsYieldAfterExit))
	fmt.Println(panics(func() { values.RangesNilIterator(nil) }))
}

func panics(action func()) (result bool) {
	defer func() {
		result = recover() != nil
	}()
	action()
	return false
}
`)
	return runProgram(
		t,
		runnerDirectory,
		filepath.Join(runtime.GOROOT(), "bin", "go"),
		"run",
		".",
	)
}
