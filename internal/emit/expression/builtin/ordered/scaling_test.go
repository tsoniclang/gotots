package ordered_test

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

func TestOrderedBuiltinScalesWithArguments(t *testing.T) {
	counts := []int{8, 16, 32}
	targetBytes := make([]int, len(counts))
	for index, count := range counts {
		directory := t.TempDir()
		writeFile(t, filepath.Join(directory, "go.mod"), `module example.com/orderedbuiltins

go 1.26.4
`)
		writeFile(
			t,
			filepath.Join(directory, "source.go"),
			orderedScalingSource(count),
		)
		loaded, err := load.One(context.Background(), load.Request{
			Directory: directory,
			Pattern:   ".",
		})
		if err != nil {
			t.Fatal(err)
		}
		roots, err := emit.ExportedAPIRoots(loaded)
		if err != nil {
			t.Fatal(err)
		}
		options := emit.DefaultOptions()
		options.IntegerRepresentation = emit.IntegerRepresentationBigInt
		emission, err := emit.CompileWithOptions(
			loaded.Program(),
			roots,
			options,
		)
		if err != nil {
			t.Fatal(err)
		}
		value := returnExpression(t, emission, "Maximum")
		if calls := orderedRuntimeCalls(value); calls != count-1 {
			t.Fatalf(
				"%d arguments emitted %d helper calls, want %d",
				count,
				calls,
				count-1,
			)
		}
		assertRuntimeDefinitionCount(t, emission, "runtime/integer.ts", 1)
		_, _, printed := printOrdered(t, t.TempDir(), emission)
		targetBytes[index] = len(printed)
	}
	for index := 1; index < len(counts); index++ {
		previousPerArgument := float64(targetBytes[index-1]) /
			float64(counts[index-1])
		currentPerArgument := float64(targetBytes[index]) /
			float64(counts[index])
		if currentPerArgument > previousPerArgument*1.1 {
			t.Fatalf(
				"target bytes/argument grew %.2f -> %.2f",
				previousPerArgument,
				currentPerArgument,
			)
		}
	}
	t.Logf(
		"ordered scaling arguments=%v target-bytes=%v",
		counts,
		targetBytes,
	)
}

func orderedScalingSource(count int) string {
	var source strings.Builder
	source.WriteString("package orderedbuiltins\n\nfunc Maximum(\n")
	for index := 0; index < count; index++ {
		fmt.Fprintf(&source, "\tvalue%d int64,\n", index)
	}
	source.WriteString(") int64 {\n\treturn max(\n")
	for index := 0; index < count; index++ {
		fmt.Fprintf(&source, "\t\tvalue%d,\n", index)
	}
	source.WriteString("\t)\n}\n")
	return source.String()
}

func orderedRuntimeCalls(value tsgo.Expression) int {
	call, ok := value.(tsgo.CallExpression)
	if !ok {
		return 0
	}
	callee, ok := call.Expression().(tsgo.Identifier)
	if !ok || callee.Text() != "goIntegerMax" {
		return 0
	}
	count := 1
	for _, argument := range call.Arguments() {
		count += orderedRuntimeCalls(argument)
	}
	return count
}
