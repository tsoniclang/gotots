package emit_test

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

type waveEightScale struct {
	sourceBytes int
	targetBytes int
	targetNodes int
	runtime     string
}

func TestWaveEightControlGrowthIsLinearAndRuntimeIsConstant(t *testing.T) {
	client, err := tsgo.StartClient(repositoryRoot(), t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := client.Close(); err != nil {
			t.Errorf("close TS-Go client: %v", err)
		}
	})
	counts := []int{1, 2, 4}
	measurements := make([]waveEightScale, 0, len(counts))
	for _, count := range counts {
		measurement := measureWaveEightScale(t, client, count)
		measurements = append(measurements, measurement)
	}
	sourceBytes := waveEightScaleValues(
		measurements,
		func(value waveEightScale) int { return value.sourceBytes },
	)
	targetBytes := waveEightScaleValues(
		measurements,
		func(value waveEightScale) int { return value.targetBytes },
	)
	targetNodes := waveEightScaleValues(
		measurements,
		func(value waveEightScale) int { return value.targetNodes },
	)
	assertWaveFourLinearDoubling(t, "Wave 8 source bytes", sourceBytes)
	assertWaveFourLinearDoubling(t, "Wave 8 target bytes", targetBytes)
	assertWaveFourLinearDoubling(t, "Wave 8 target AST nodes", targetNodes)
	if measurements[0].runtime != measurements[1].runtime ||
		measurements[1].runtime != measurements[2].runtime {
		t.Fatal("Wave 8 runtime support grew with callable count")
	}
	t.Logf(
		"Wave 8 scaling callables=%v source=%v target=%v nodes=%v runtime=%d",
		counts,
		sourceBytes,
		targetBytes,
		targetNodes,
		len(measurements[0].runtime),
	)
}

func measureWaveEightScale(
	t *testing.T,
	client *tsgo.Client,
	count int,
) waveEightScale {
	t.Helper()
	source, emission := compileWaveEightScale(t, count)
	result := waveEightScale{sourceBytes: len(source)}
	var printed strings.Builder
	var runtime strings.Builder
	for _, file := range emission.Files() {
		target, err := client.PrintNode(file.SourceFile(), tsgo.PrintOptions{})
		if err != nil {
			t.Fatal(err)
		}
		encoded, err := tsgo.EncodeSourceFile(file.SourceFile())
		if err != nil {
			t.Fatal(err)
		}
		result.targetBytes += len(target)
		result.targetNodes += waveFourEncodedNodes(t, encoded)
		printed.WriteString(target)
		printed.WriteByte('\n')
		if strings.HasPrefix(file.OutputPath(), "runtime/") {
			runtime.WriteString(file.OutputPath())
			runtime.WriteByte('\n')
			runtime.WriteString(target)
		}
	}
	target := printed.String()
	for name, got := range map[string]int{
		"defer registrations": strings.Count(target, ".push("),
		"goto states":         strings.Count(target, "let __gotots_goto_state_"),
		"source body stores":  strings.Count(target, "result += value;"),
	} {
		if got != count {
			t.Fatalf("%s at %d callables = %d, want %d", name, count, got, count)
		}
	}
	result.runtime = runtime.String()
	return result
}

func compileWaveEightScale(
	t *testing.T,
	count int,
) (string, emit.ProgramEmission) {
	t.Helper()
	directory := t.TempDir()
	writeProgramFile(
		t,
		filepath.Join(directory, "go.mod"),
		"module example.com/wave8scale\n\ngo 1.26.4\n",
	)
	var source strings.Builder
	source.WriteString("package wave8scale\n\n")
	for index := range count {
		fmt.Fprintf(
			&source,
			"func Scale%d(value int) (result int) {\n"+
				"\tdefer func(captured int) { result += captured }(value)\n"+
				"\tgoto check%d\n"+
				"body%d:\n"+
				"\tresult += value\n"+
				"\tvalue--\n"+
				"check%d:\n"+
				"\tif value > 0 { goto body%d }\n"+
				"\treturn\n"+
				"}\n\n",
			index,
			index,
			index,
			index,
			index,
		)
	}
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
	return source.String(), emission
}

func waveEightScaleValues(
	values []waveEightScale,
	selectValue func(waveEightScale) int,
) []int {
	result := make([]int, 0, len(values))
	for _, value := range values {
		result = append(result, selectValue(value))
	}
	return result
}
