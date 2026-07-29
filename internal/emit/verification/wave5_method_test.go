package emit_test

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"go/ast"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/tsoniclang/gotots/internal/emit"
	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/load"
)

func TestWaveFiveSelectionUsesCheckerIdentityNotSpelling(t *testing.T) {
	program := loadWaveFiveMethods(t)
	roots, err := emit.ExportedAPIRoots(program.Roots()[0])
	if err != nil {
		t.Fatal(err)
	}
	baseline, err := emit.Compile(program, roots)
	if err != nil {
		t.Fatal(err)
	}
	selected := waveFivePromotedSelector(t, program, "Read")
	selected.Sel.Name = "SpellingMustNotSelectPromotion"
	mutated, err := emit.Compile(program, roots)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(
		encodeWaveThreeProgram(t, baseline),
		encodeWaveThreeProgram(t, mutated),
	) {
		t.Fatal("selection output changed after source-spelling-only mutation")
	}
}

func TestWaveFiveRejectsMismatchedSelectionIdentity(t *testing.T) {
	program := loadWaveFiveMethods(t)
	target := waveFivePromotedSelector(t, program, "Read")
	replacement := waveFivePromotedSelector(t, program, "Name")
	info := program.Roots()[0].TypesInfo()
	info.Selections[target] = info.Selections[replacement]
	roots, err := emit.ExportedAPIRoots(program.Roots()[0])
	if err != nil {
		t.Fatal(err)
	}
	_, err = emit.Compile(program, roots)
	var unsupported *api.UnsupportedError
	if !errors.As(err, &unsupported) ||
		unsupported.Category != api.CategoryExpression ||
		unsupported.Construct != "*ast.SelectorExpr" {
		t.Fatalf(
			"mismatched selection error = %#v, want selector expression boundary",
			err,
		)
	}
}

func TestWaveFiveMethodsPrintTypecheckAndMatchGo(t *testing.T) {
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
				Directory: waveFiveMethodDirectory(),
				Pattern:   ".",
			})
			if err != nil {
				t.Fatal(err)
			}
			roots, err := emit.ExportedAPIRoots(program.Roots()[0])
			if err != nil {
				t.Fatal(err)
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
			artifacts := materializeWaveFour(
				t,
				emission,
				workingDirectory,
			)
			if artifacts.bytes > 40_000 || artifacts.largest > 24_000 {
				t.Fatalf(
					"Wave 5 artifact bounds exceeded: total=%d largest=%d",
					artifacts.bytes,
					artifacts.largest,
				)
			}
			t.Logf(
				"Wave 5 artifacts total=%d largest=%d",
				artifacts.bytes,
				artifacts.largest,
			)
			assertWaveFiveShape(t, artifacts.printed)
			runner := filepath.Join(workingDirectory, "runner.ts")
			writeProgramFile(t, runner, `import "./program.js";
import {
    Audit,
    NilPointerMethodValue,
    NilPromotedPointerMethod,
    PanicPromotedField,
    PanicValueMethod,
} from "`+artifacts.sourceModule+`";

const values = Audit();
const output: string[] = [];
for (let index = 0; index < values.length; index++) {
    output.push(String(values.get(index)));
}
output.push(String(NilPointerMethodValue()));
output.push(String(NilPromotedPointerMethod()));
for (const action of [PanicValueMethod, PanicPromotedField]) {
    try {
        action();
        output.push("no-panic");
    } catch {
        output.push("panic");
    }
}
console.log(output.join(" "));
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
			goOutput := executeWaveFiveGo(t, workingDirectory)
			if targetOutput != goOutput {
				t.Fatalf(
					"Wave 5 output differs\nTypeScript:\n%s\nGo:\n%s",
					targetOutput,
					goOutput,
				)
			}
		})
	}
}

func loadWaveFiveMethods(t *testing.T) *load.Program {
	t.Helper()
	program, err := load.Load(context.Background(), load.Request{
		Directory: waveFiveMethodDirectory(),
		Pattern:   ".",
	})
	if err != nil {
		t.Fatal(err)
	}
	return program
}

func waveFivePromotedSelector(
	t *testing.T,
	program *load.Program,
	name string,
) *ast.SelectorExpr {
	t.Helper()
	root := program.Roots()[0]
	var result *ast.SelectorExpr
	for _, file := range root.Files() {
		ast.Inspect(file.Syntax(), func(node ast.Node) bool {
			selector, ok := node.(*ast.SelectorExpr)
			if !ok || selector.Sel.Name != name {
				return true
			}
			selected := root.TypesInfo().Selections[selector]
			if selected == nil || len(selected.Index()) < 2 {
				return true
			}
			result = selector
			return false
		})
		if result != nil {
			break
		}
	}
	if result == nil {
		t.Fatalf("promoted selector %s is absent", name)
	}
	return result
}

func assertWaveFiveShape(t *testing.T, printed string) {
	t.Helper()
	for _, required := range []string{
		"export function Base_Read",
		"export function Base_Add",
		"export function Derived_Name",
		"=> Base_Read(",
		"=> Base_Add(",
		"return Base_Name(base);",
		"__go_constructor",
	} {
		if !strings.Contains(printed, required) {
			t.Fatalf("Wave 5 artifacts lack %q:\n%s", required, printed)
		}
	}
	for _, forbidden := range []string{
		"extends Base",
		".bind(",
		".call(",
		".apply(",
	} {
		if strings.Contains(printed, forbidden) {
			t.Fatalf("Wave 5 artifacts contain %q:\n%s", forbidden, printed)
		}
	}
}

func executeWaveFiveGo(t *testing.T, workingDirectory string) string {
	t.Helper()
	modulePath, err := filepath.Abs(waveFiveMethodDirectory())
	if err != nil {
		t.Fatal(err)
	}
	runnerDirectory := filepath.Join(workingDirectory, "go-runner-wave5")
	writeProgramFile(t, filepath.Join(runnerDirectory, "go.mod"), fmt.Sprintf(
		`module example.com/runner

go 1.26.4

require example.com/wave5methods v0.0.0

replace example.com/wave5methods => %s
`,
		filepath.ToSlash(modulePath),
	))
	writeProgramFile(t, filepath.Join(runnerDirectory, "main.go"), `package main

import (
	"fmt"

	values "example.com/wave5methods"
)

func panics(action func()) (result bool) {
	defer func() {
		result = recover() != nil
	}()
	action()
	return false
}

func main() {
	for index, value := range values.Audit() {
		if index != 0 {
			fmt.Print(" ")
		}
		fmt.Print(value)
	}
	fmt.Print(" ", values.NilPointerMethodValue())
	fmt.Print(" ", values.NilPromotedPointerMethod())
	for _, action := range []func(){
		func() { values.PanicValueMethod() },
		func() { values.PanicPromotedField() },
	} {
		if panics(action) {
			fmt.Print(" panic")
		} else {
			fmt.Print(" no-panic")
		}
	}
	fmt.Println()
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

func waveFiveMethodDirectory() string {
	return filepath.Join(
		repositoryRoot(),
		"testdata",
		"constructs",
		"method",
		"wave5",
	)
}
