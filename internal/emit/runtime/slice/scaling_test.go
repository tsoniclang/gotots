package slice_test

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tsoniclang/gotots/internal/emit"
	"github.com/tsoniclang/gotots/internal/load"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func TestSliceUseSitesScaleLinearlyAndRuntimeStaysConstant(t *testing.T) {
	counts := []int{8, 16, 32}
	sourceSizes := make([]int, 0, len(counts))
	var runtimeSource string
	for _, count := range counts {
		source := scaleSource(count)
		printed := printGenerated(t, source)
		if strings.Count(printed.source, ".set(") != count ||
			strings.Count(printed.source, ".get(") != count {
			t.Fatalf(
				"%d sites emitted set/get counts %d/%d",
				count,
				strings.Count(printed.source, ".set("),
				strings.Count(printed.source, ".get("),
			)
		}
		sourceSizes = append(sourceSizes, len(printed.source))
		if runtimeSource == "" {
			runtimeSource = printed.runtime
		} else if printed.runtime != runtimeSource {
			t.Fatal("runtime slice module grew or changed with source use-site count")
		}
	}
	firstDelta := sourceSizes[1] - sourceSizes[0]
	secondDelta := sourceSizes[2] - sourceSizes[1]
	if firstDelta <= 0 ||
		secondDelta < firstDelta*19/10 ||
		secondDelta > firstDelta*21/10 {
		t.Fatalf(
			"source sizes %v are not linear under doubled use sites",
			sourceSizes,
		)
	}
}

func TestSparseKeyedSliceLiteralDoesNotExpandWithLargestIndex(t *testing.T) {
	printed := printGenerated(t, `package scale
func Scale() int32 {
	values := []int32{1000000: 7}
	return values[1000000]
}
`)
	if len(printed.source) > 2500 {
		t.Fatalf(
			"sparse keyed literal emitted %d bytes, want bounded per explicit element",
			len(printed.source),
		)
	}
	for _, fragment := range []string{
		"RuntimeSlice.make<int32>(1000001, null, 0)",
		".set(1000000, 7)",
		".get(1000000)",
	} {
		if !strings.Contains(printed.source, fragment) {
			t.Fatalf("sparse keyed literal lacks %q:\n%s", fragment, printed.source)
		}
	}
}

func scaleSource(count int) string {
	var source strings.Builder
	source.WriteString("package scale\nfunc Scale(values []int32) int32 {\n")
	for index := range count {
		fmt.Fprintf(&source, "values[%d] = %d\n", index, index)
	}
	source.WriteString("var total int32\n")
	for index := range count {
		fmt.Fprintf(&source, "total = total + values[%d]\n", index)
	}
	source.WriteString("return total\n}\n")
	return source.String()
}

func printGenerated(t *testing.T, source string) printedFiles {
	t.Helper()
	directory := t.TempDir()
	if err := os.WriteFile(
		filepath.Join(directory, "go.mod"),
		[]byte("module example.com/scale\n\ngo 1.26.4\n"),
		0o644,
	); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(
		filepath.Join(directory, "source.go"),
		[]byte(source),
		0o644,
	); err != nil {
		t.Fatal(err)
	}
	program, err := load.Load(context.Background(), load.Request{
		Directory: directory,
		Pattern:   ".",
	})
	if err != nil {
		t.Fatal(err)
	}
	root, err := emit.NewRoot(program.Roots()[0].Types().Scope().Lookup("Scale"))
	if err != nil {
		t.Fatal(err)
	}
	emission, err := emit.CompileWithOptions(
		program,
		[]emit.Root{root},
		emit.Options{
			IntegerRepresentation: emit.IntegerRepresentationNumber,
			EvaluationOrder:       emit.EvaluationOrderDirect,
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	client, err := tsgo.StartClient(repositoryRoot(), directory)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := client.Close(); err != nil {
			t.Errorf("close TS-Go client: %v", err)
		}
	}()
	var printed printedFiles
	for _, file := range emission.Files() {
		target, err := client.PrintNode(file.SourceFile(), tsgo.PrintOptions{})
		if err != nil {
			t.Fatal(err)
		}
		if file.Kind() == emit.TargetFileSource {
			printed.source += target
		}
		if file.OutputPath() == "runtime/slice.ts" {
			printed.runtime = target
		}
	}
	if printed.source == "" || printed.runtime == "" {
		t.Fatalf(
			"generated source/runtime sizes = %d/%d",
			len(printed.source),
			len(printed.runtime),
		)
	}
	return printed
}
