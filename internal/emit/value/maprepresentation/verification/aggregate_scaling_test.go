package maprepresentation_test

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tsoniclang/gotots/internal/emit"
	"github.com/tsoniclang/gotots/internal/load"
)

func TestAggregateMapUseGrowthKeepsOneStaticShapeOwner(t *testing.T) {
	counts := []int{2, 4, 8}
	var canonicalOwner string
	var targetBytes []int
	for _, count := range counts {
		directory := t.TempDir()
		writeFile(
			t,
			filepath.Join(directory, "go.mod"),
			"module example.com/aggregatescaling\n\ngo 1.26.4\n",
		)
		writeFile(
			t,
			filepath.Join(directory, "source.go"),
			aggregateScalingSource(count),
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
		emission, err := emit.Compile(loaded.Program(), roots)
		if err != nil {
			t.Fatal(err)
		}
		artifacts := materialize(t, emission, t.TempDir())
		generated := ""
		for _, file := range emission.Files() {
			if !strings.HasPrefix(file.OutputPath(), "support/maps/") {
				continue
			}
			if generated != "" {
				t.Fatalf("%d uses emitted more than one map-shape owner", count)
			}
			generated = readFile(t, artifacts.file(t, file.OutputPath()))
		}
		if generated == "" ||
			strings.Count(generated, "export class $goMap_") != 1 {
			t.Fatalf("%d uses emitted no exact static map owner", count)
		}
		if canonicalOwner == "" {
			canonicalOwner = generated
		} else if generated != canonicalOwner {
			t.Fatalf("%d uses changed or duplicated map-shape logic", count)
		}
		target := readFile(t, artifacts.file(t, "source.ts"))
		targetBytes = append(targetBytes, len(target))
	}
	firstDelta := targetBytes[1] - targetBytes[0]
	secondDelta := targetBytes[2] - targetBytes[1]
	if firstDelta <= 0 ||
		secondDelta <= 0 ||
		secondDelta > firstDelta*2+64 {
		t.Fatalf(
			"aggregate map use growth is not linear: bytes=%v deltas=%d,%d",
			targetBytes,
			firstDelta,
			secondDelta,
		)
	}
	t.Logf(
		"aggregate map uses=%v source-bytes=%v owner-bytes=%d",
		counts,
		targetBytes,
		len(canonicalOwner),
	)
}

func aggregateScalingSource(count int) string {
	var source strings.Builder
	source.WriteString(`package aggregatescaling

type Key [2]int32

`)
	for index := range count {
		fmt.Fprintf(
			&source,
			"func F%d() int32 { values := map[Key]int32{{%d, %d}: %d}; return values[Key{%d, %d}] }\n",
			index,
			index,
			index+1,
			index+2,
			index,
			index+1,
		)
	}
	return source.String()
}
