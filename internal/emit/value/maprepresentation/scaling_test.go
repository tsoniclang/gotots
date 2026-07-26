package maprepresentation_test

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

func TestMapSitesScaleLinearlyWithOneRuntimeDefinition(t *testing.T) {
	type measurement struct {
		sourceBytes  int
		targetBytes  int
		runtimeBytes int
	}
	counts := []int{4, 8, 16}
	measurements := make([]measurement, 0, len(counts))
	var canonicalRuntime string
	for _, count := range counts {
		directory := t.TempDir()
		source := scalingSource(count)
		writeFile(t, filepath.Join(directory, "go.mod"), "module example.com/mapscaling\n\ngo 1.26.4\n")
		writeFile(t, filepath.Join(directory, "source.go"), source)
		loaded, err := load.One(context.Background(), load.Request{
			Directory: directory,
			Pattern:   ".",
		})
		if err != nil {
			t.Fatal(err)
		}
		emission := compileExported(t, loaded)
		runtimeFiles := 0
		var sourceFile emit.TargetFile
		for _, file := range emission.Files() {
			switch {
			case strings.HasSuffix(file.OutputPath(), "runtime/map.ts"):
				runtimeFiles++
			case file.Kind() == emit.TargetFileSource:
				sourceFile = file
			}
		}
		if runtimeFiles != 1 || sourceFile.SourceFile() == nil {
			t.Fatalf(
				"%d sites emitted %d runtime map files and source=%v",
				count,
				runtimeFiles,
				sourceFile.SourceFile() != nil,
			)
		}
		if got := countFunctions(sourceFile); got != count {
			t.Fatalf("%d sites emitted %d functions", count, got)
		}
		artifacts := materialize(t, emission, t.TempDir())
		target := readFile(t, artifacts.file(t, "source.ts"))
		runtimeSource := readFile(t, artifacts.file(t, "runtime/map.ts"))
		if got := strings.Count(target, ".store("); got != count {
			t.Fatalf("%d sites emitted %d stores", count, got)
		}
		if got := strings.Count(target, ".lookup("); got != count {
			t.Fatalf("%d sites emitted %d lookups", count, got)
		}
		if canonicalRuntime == "" {
			canonicalRuntime = runtimeSource
		} else if runtimeSource != canonicalRuntime {
			t.Fatalf("%d sites changed the shared runtime definition", count)
		}
		measurements = append(measurements, measurement{
			sourceBytes:  len(source),
			targetBytes:  len(target),
			runtimeBytes: len(runtimeSource),
		})
	}
	firstDelta := measurements[1].targetBytes - measurements[0].targetBytes
	secondDelta := measurements[2].targetBytes - measurements[1].targetBytes
	if firstDelta <= 0 ||
		secondDelta <= 0 ||
		secondDelta > firstDelta*2+64 {
		t.Fatalf(
			"target growth is not linear: bytes=%v deltas=%d,%d",
			measurements,
			firstDelta,
			secondDelta,
		)
	}
	for index := 1; index < len(measurements); index++ {
		if measurements[index].runtimeBytes != measurements[0].runtimeBytes {
			t.Fatalf("runtime bytes vary by site count: %v", measurements)
		}
	}
	t.Logf("map scaling counts=%v measurements=%+v", counts, measurements)
}

func scalingSource(count int) string {
	var source strings.Builder
	source.WriteString("package mapscaling\n\n")
	for index := range count {
		fmt.Fprintf(
			&source,
			"func F%d(values map[int32]int32) int32 { values[%d] = %d; return values[%d] }\n",
			index,
			index,
			index+1,
			index,
		)
	}
	return source.String()
}

func countFunctions(file emit.TargetFile) int {
	count := 0
	for _, statement := range file.SourceFile().Statements() {
		if _, ok := statement.(tsgo.FunctionDeclaration); ok {
			count++
		}
	}
	return count
}
