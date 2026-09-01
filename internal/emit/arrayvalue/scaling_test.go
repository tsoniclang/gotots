package arrayvalue_test

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tsoniclang/gotots/internal/emit"
	"github.com/tsoniclang/gotots/internal/load"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func TestArrayUseSitesScaleLinearlyAtOneTwoAndFour(t *testing.T) {
	counts := []int{1, 2, 4}
	targetBytes := make([]int, len(counts))
	sourceBytes := make([]int, len(counts))
	topLevelNodes := make([]int, len(counts))
	for index, count := range counts {
		source, emission := compileScalingArrays(t, count)
		sourceBytes[index] = len(source)
		directory := t.TempDir()
		target := materializeArrayProgram(t, directory, emission)
		writeFile(
			t,
			filepath.Join(directory, "package.json"),
			"{\"type\":\"module\"}\n",
		)
		if err := compileTypeScript(t, directory, target.paths); err != nil {
			t.Fatal(err)
		}
		for _, printed := range target.printed {
			targetBytes[index] += len(printed)
		}
		topLevelNodes[index] = assertScalingAST(t, emission, count)
	}
	firstDelta := targetBytes[1] - targetBytes[0]
	secondDelta := targetBytes[2] - targetBytes[1]
	if firstDelta <= 0 ||
		secondDelta*10 < firstDelta*18 ||
		secondDelta*10 > firstDelta*22 {
		t.Fatalf(
			"target bytes = %v, deltas %d/%d are not linear",
			targetBytes,
			firstDelta,
			secondDelta,
		)
	}
	if topLevelNodes[1]-topLevelNodes[0] != 1 ||
		topLevelNodes[2]-topLevelNodes[1] != 2 {
		t.Fatalf("top-level target AST nodes = %v", topLevelNodes)
	}
	t.Logf(
		"array scaling source bytes=%v target bytes=%v top-level AST nodes=%v",
		sourceBytes,
		targetBytes,
		topLevelNodes,
	)
}

func compileScalingArrays(
	t *testing.T,
	count int,
) (string, emit.ProgramEmission) {
	t.Helper()
	directory := t.TempDir()
	writeFile(
		t,
		filepath.Join(directory, "go.mod"),
		"module example.com/arrayscaling\n\ngo 1.26.4\n",
	)
	var source strings.Builder
	source.WriteString("package arrayscaling\n\n")
	for index := range count {
		fmt.Fprintf(&source, `func F%d(value int32) int {
array := [2]int32{value, value + 1}
copy := array
copy[0] = value + 2
if copy == array {
		return len(copy) + cap(array)
}
return len(copy)
}

`, index)
	}
	writeFile(t, filepath.Join(directory, "source.go"), source.String())
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
	emission, err := emit.CompileWithOptions(program, roots, arrayNumberOptions())
	if err != nil {
		t.Fatal(err)
	}
	return source.String(), emission
}

func assertScalingAST(
	t *testing.T,
	emission emit.ProgramEmission,
	count int,
) int {
	t.Helper()
	functions := 0
	runtimeDefinitions := 0
	var bodyWidth int
	for _, file := range emission.Files() {
		if file.OutputPath() == "runtime/array.ts" {
			for _, statement := range file.SourceFile().Statements() {
				class, ok := statement.(tsgo.ClassDeclaration)
				if ok && class.Name().Text() == "GoArray" {
					runtimeDefinitions++
				}
			}
		}
		if file.Kind() != emit.TargetFileSource {
			continue
		}
		for _, statement := range file.SourceFile().Statements() {
			function, ok := statement.(tsgo.FunctionDeclaration)
			if !ok {
				continue
			}
			functions++
			width := len(function.Body().(tsgo.Block).Statements())
			if bodyWidth == 0 {
				bodyWidth = width
			} else if width != bodyWidth {
				t.Fatalf("array use-site AST widths = %d and %d", bodyWidth, width)
			}
		}
	}
	if functions != count || runtimeDefinitions != 1 {
		t.Fatalf(
			"array scaling functions/runtime = %d/%d, want %d/1",
			functions,
			runtimeDefinitions,
			count,
		)
	}
	return functions + runtimeDefinitions
}
