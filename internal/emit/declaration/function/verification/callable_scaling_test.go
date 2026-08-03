package function_test

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tsoniclang/gotots/internal/load"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func TestCallableConstructionAndOutputScaleLinearly(t *testing.T) {
	counts := []int{4, 8, 16}
	sourceBytes := make([]int, len(counts))
	targetBytes := make([]int, len(counts))
	runArtifacts := make([]string, len(counts))

	client, err := tsgo.StartClient(repositoryRoot(), t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := client.Close(); err != nil {
			t.Errorf("close TS-Go client: %v", err)
		}
	})

	for index, count := range counts {
		source, target := compileCallableScaling(t, count)
		printed, err := client.PrintNode(target, tsgo.PrintOptions{})
		if err != nil {
			t.Fatal(err)
		}
		runPrinted, err := client.PrintNode(
			targetFunction(t, target, "Run"),
			tsgo.PrintOptions{},
		)
		if err != nil {
			t.Fatal(err)
		}

		assertCallableScalingTree(t, target, count)
		sourceBytes[index] = len(source)
		targetBytes[index] = len(printed)
		runArtifacts[index] = runPrinted
	}

	assertDoublingDeltas(t, "Go source bytes", sourceBytes)
	assertDoublingDeltas(t, "TypeScript target bytes", targetBytes)
	for index := 1; index < len(runArtifacts); index++ {
		if runArtifacts[index] != runArtifacts[0] {
			t.Fatal("one function-value call site grew with possible function count")
		}
	}
	t.Logf(
		"callable scaling counts=%v source-bytes=%v target-bytes=%v",
		counts,
		sourceBytes,
		targetBytes,
	)
}

func compileCallableScaling(t *testing.T, count int) (string, tsgo.SourceFile) {
	t.Helper()
	directory := t.TempDir()
	writeFile(
		t,
		filepath.Join(directory, "go.mod"),
		"module example.com/callablescaling\n\ngo 1.26.4\n",
	)
	source := callableScalingSource(count)
	writeFile(t, filepath.Join(directory, "source.go"), source)
	loaded, err := load.One(context.Background(), load.Request{
		Directory: directory,
		Pattern:   ".",
	})
	if err != nil {
		t.Fatal(err)
	}
	return source, compileSourceFile(t, loaded, loaded.Files()[0].Syntax())
}

func callableScalingSource(count int) string {
	var source strings.Builder
	source.WriteString("package callablescaling\n\n")
	for index := range count {
		fmt.Fprintf(
			&source,
			"func Candidate%d(value int32) int32 { return value + %d }\n",
			index,
			index,
		)
	}
	source.WriteString("\nfunc Choose(index int32) func(int32) int32 {\n\tswitch index {\n")
	for index := range count {
		fmt.Fprintf(&source, "\tcase %d:\n\t\treturn Candidate%d\n", index, index)
	}
	source.WriteString("\tdefault:\n\t\treturn Candidate0\n\t}\n}\n\n")
	source.WriteString(`func Run(index, value int32) int32 {
	callback := Choose(index)
	return callback(value)
}

func Nested(seed int32) int32 {
	current := seed
`)
	for index := range count {
		fmt.Fprintf(
			&source,
			"\ttransform%d := func(value int32) int32 { return value + current }\n",
			index,
		)
	}
	source.WriteString("\treturn ")
	for index := range count {
		if index != 0 {
			source.WriteString(" + ")
		}
		fmt.Fprintf(&source, "transform%d(seed)", index)
	}
	source.WriteString("\n}\n\nfunc Parameters(\n")
	for index := range count {
		fmt.Fprintf(&source, "\ttransform%d func(int32) int32,\n", index)
	}
	source.WriteString("\tvalue int32,\n) int32 {\n\treturn ")
	for index := range count {
		if index != 0 {
			source.WriteString(" + ")
		}
		fmt.Fprintf(&source, "transform%d(value)", index)
	}
	source.WriteString("\n}\n")
	return source.String()
}

func assertCallableScalingTree(t *testing.T, source tsgo.SourceFile, count int) {
	t.Helper()
	parameters := targetFunction(t, source, "Parameters")
	if len(parameters.Parameters()) != count+1 {
		t.Fatalf(
			"Parameters target parameters = %d, want %d",
			len(parameters.Parameters()),
			count+1,
		)
	}

	nested := targetFunction(t, source, "Nested")
	literals := 0
	for _, statement := range nested.Body().(tsgo.Block).Statements() {
		variables, ok := statement.(tsgo.VariableStatement)
		if !ok {
			continue
		}
		for _, declaration := range variables.DeclarationList().Declarations() {
			if _, ok := declaration.Initializer().(tsgo.ArrowFunction); ok {
				literals++
			}
		}
	}
	if literals != count {
		t.Fatalf("Nested arrow functions = %d, want %d", literals, count)
	}

	run := targetFunction(t, source, "Run")
	runStatements := run.Body().(tsgo.Block).Statements()
	result, ok := runStatements[len(runStatements)-1].(tsgo.ReturnStatement)
	if !ok {
		t.Fatalf("Run final statement = %T, want ReturnStatement", runStatements[len(runStatements)-1])
	}
	call, ok := result.Expression().(tsgo.CallExpression)
	if !ok || len(call.Arguments()) != 1 {
		t.Fatalf("Run result = %T, want one-argument direct call", result.Expression())
	}
	callee, ok := call.Expression().(tsgo.Identifier)
	if !ok || !strings.HasPrefix(callee.Text(), "__gotots_callee_") {
		t.Fatalf("Run callee = %T, want captured callable", call.Expression())
	}
	if len(runStatements) != 5 {
		t.Fatalf("Run statements = %d, want callback, callee, argument, guard, return", len(runStatements))
	}
	if _, ok := runStatements[len(runStatements)-2].(tsgo.IfStatement); !ok {
		t.Fatalf("Run guard = %T, want IfStatement", runStatements[len(runStatements)-2])
	}
}

func assertDoublingDeltas(t *testing.T, label string, values []int) {
	t.Helper()
	if len(values) != 3 {
		t.Fatalf("%s sample count = %d, want three", label, len(values))
	}
	firstDelta := values[1] - values[0]
	secondDelta := values[2] - values[1]
	if firstDelta <= 0 ||
		secondDelta*10 < firstDelta*17 ||
		secondDelta*10 > firstDelta*23 {
		t.Fatalf(
			"%s = %v; doubling deltas %d/%d are not linear",
			label,
			values,
			firstDelta,
			secondDelta,
		)
	}
}
