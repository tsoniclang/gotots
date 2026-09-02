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
	"github.com/tsoniclang/gotots/internal/target/tsgo"
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

func TestWaveFiveMethodsPrintAndTypecheck(t *testing.T) {
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
			artifacts := materializeArtifacts(
				t,
				emission,
				workingDirectory,
			)
			// Canonical facts are counted here and erased only by the executable
			// TypeScript target, whose output has an independent budget.
			if artifacts.bytes > 80_000 || artifacts.largest > 55_000 {
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
			waveThreeTypecheck(t, workingDirectory, artifacts.paths)
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
		"export class Base",
		"Read(): int32",
		"static Add(base:",
		"static Name(derived:",
		".Read()",
		"Base.Add(receiver",
		"return Base.Name(base);",
		"__go_constructor",
	} {
		if !strings.Contains(printed, required) {
			t.Fatalf("Wave 5 artifacts lack %q:\n%s", required, printed)
		}
	}
	for _, forbidden := range []string{
		"export function Base_Read",
		"export function Base_Add",
		"export function Derived_Name",
		"=> Base_Read(",
		"=> Base_Add(",
		"extends Base",
		".bind(",
		".call(",
		".apply(",
		"goPointerHash",
		"class GoMapHash",
		"Read($go$recovery",
		"DeferredCallableRegistry",
		"GoDeferredRegistry",
		"deferred-registry.js",
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

func TestWaveFivePromotionScalesWithEmbeddingDepth(t *testing.T) {
	depths := []int{4, 8, 16}
	sourceBytes := make([]int, len(depths))
	targetBytes := make([]int, len(depths))
	targetNodes := make([]int, len(depths))
	for index, depth := range depths {
		source, target, nodes := compileWaveFiveScaling(t, depth)
		sourceBytes[index] = len(source)
		targetBytes[index] = len(target)
		targetNodes[index] = nodes
		if definitions := strings.Count(target, "static Add("); definitions != 1 {
			t.Fatalf(
				"promotion depth %d emits %d Add definitions, want one",
				depth,
				definitions,
			)
		}
		if calls := strings.Count(target, "T0.Add("); calls != 1 {
			t.Fatalf(
				"promotion depth %d emits %d T0.Add calls, want one",
				depth,
				calls,
			)
		}
		if strings.Contains(target, "T0_Add") {
			t.Fatalf("promotion depth %d retained a top-level receiver twin", depth)
		}
		if strings.Contains(target, "switch (") ||
			strings.Contains(target, "extends T") {
			t.Fatalf(
				"promotion depth %d introduced dispatch/inheritance:\n%s",
				depth,
				target,
			)
		}
	}
	assertWaveFourLinearDoubling(t, "Wave 5 source bytes", sourceBytes)
	assertWaveFourLinearDoubling(t, "Wave 5 target bytes", targetBytes)
	assertWaveFourLinearDoubling(t, "Wave 5 target AST nodes", targetNodes)
	t.Logf(
		"Wave 5 depth=%v source=%v target=%v nodes=%v",
		depths,
		sourceBytes,
		targetBytes,
		targetNodes,
	)
}

func compileWaveFiveScaling(
	t *testing.T,
	depth int,
) (string, string, int) {
	t.Helper()
	directory := t.TempDir()
	writeProgramFile(
		t,
		filepath.Join(directory, "go.mod"),
		"module example.com/wave5scaling\n\ngo 1.26.4\n",
	)
	var source strings.Builder
	source.WriteString(
		"package wave5scaling\n\n" +
			"type T0 struct { Value int32 }\n" +
			"func (value *T0) Add(delta int32) { value.Value += delta }\n\n",
	)
	for index := 1; index <= depth; index++ {
		fmt.Fprintf(
			&source,
			"type T%d struct { T%d }\n",
			index,
			index-1,
		)
	}
	fmt.Fprintf(
		&source,
		"\nfunc Scale(value *T%d) int32 {\n"+
			"\tvalue.Add(1)\n"+
			"\treturn value.Value\n"+
			"}\n",
		depth,
	)
	writeProgramFile(
		t,
		filepath.Join(directory, "source.go"),
		source.String(),
	)
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
	client, err := tsgo.StartClient(repositoryRoot(), t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := client.Close(); err != nil {
			t.Errorf("close TS-Go client: %v", err)
		}
	})
	for _, file := range emission.Files() {
		if file.Kind() != emit.TargetFileSource {
			continue
		}
		printed, err := client.PrintNode(
			file.SourceFile(),
			tsgo.PrintOptions{},
		)
		if err != nil {
			t.Fatal(err)
		}
		encoded, err := tsgo.EncodeSourceFile(file.SourceFile())
		if err != nil {
			t.Fatal(err)
		}
		return source.String(), printed, waveFourEncodedNodes(t, encoded)
	}
	t.Fatal("Wave 5 scaling source artifact is absent")
	return "", "", 0
}

func TestUnnamedReceiversPrintAndTypecheck(t *testing.T) {
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
		"static Pointer($0: Pointer__from_tsonic_core<Token> | undefined)",
		"value.Value()",
		"Token.Pointer(addressOf<Token>(value))",
	} {
		if !strings.Contains(artifacts.printed, required) {
			t.Fatalf("unnamed receiver artifact lacks %q:\n%s", required, artifacts.printed)
		}
	}
	for _, forbidden := range []string{
		"export function Token_Value",
		"export function Token_Pointer",
		"Token$Storage",
		"GoPointer<Token",
		"Token.$copy(value)",
	} {
		if strings.Contains(artifacts.printed, forbidden) {
			t.Fatalf("unnamed receiver artifact contains %q", forbidden)
		}
	}
	waveThreeTypecheck(t, workingDirectory, artifacts.paths)
}
