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

type waveSixScale struct {
	implementers int
	adapters     int
	callBytes    int
	printedBytes int
	wireBytes    int
}

func TestWaveSixInterfaceDispatchIsIndependentOfImplementerCount(
	t *testing.T,
) {
	client, err := tsgo.StartClient(repositoryRoot(), t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := client.Close(); err != nil {
			t.Errorf("close TS-Go client: %v", err)
		}
	})

	var measurements []waveSixScale
	for _, count := range []int{4, 8, 16} {
		measurement := measureWaveSixScale(t, client, count)
		t.Logf(
			"implementers=%d adapters=%d call=%dB printed=%dB wire=%dB",
			measurement.implementers,
			measurement.adapters,
			measurement.callBytes,
			measurement.printedBytes,
			measurement.wireBytes,
		)
		if measurement.adapters != count {
			t.Fatalf(
				"%d implementers emitted %d adapters",
				count,
				measurement.adapters,
			)
		}
		measurements = append(measurements, measurement)
	}
	if measurements[0].callBytes != measurements[1].callBytes ||
		measurements[1].callBytes != measurements[2].callBytes {
		t.Fatalf(
			"interface call size depends on implementers: %v",
			measurements,
		)
	}
	assertWaveSixLinearGrowth(
		t,
		"printed TypeScript",
		measurements,
		func(value waveSixScale) int { return value.printedBytes },
	)
	assertWaveSixLinearGrowth(
		t,
		"encoded TS-Go AST",
		measurements,
		func(value waveSixScale) int { return value.wireBytes },
	)
}

func measureWaveSixScale(
	t *testing.T,
	client *tsgo.Client,
	implementers int,
) waveSixScale {
	t.Helper()
	projectDirectory := t.TempDir()
	writeProgramFile(
		t,
		filepath.Join(projectDirectory, "go.mod"),
		"module example.com/wave6scale\n\ngo 1.26.4\n",
	)
	var source strings.Builder
	source.WriteString("package wave6scale\n\n")
	source.WriteString("type Reader interface { Read() int32 }\n\n")
	for index := range implementers {
		fmt.Fprintf(&source, "type V%d struct{}\n", index)
		fmt.Fprintf(
			&source,
			"func (value V%d) Read() int32 { return %d }\n",
			index,
			index,
		)
	}
	source.WriteString(
		"\nfunc Call(value Reader) int32 { return value.Read() }\n",
	)
	source.WriteString("func Audit() int32 { return ")
	for index := range implementers {
		if index != 0 {
			source.WriteString(" + ")
		}
		fmt.Fprintf(&source, "Call(V%d{})", index)
	}
	source.WriteString(" }\n")
	writeProgramFile(
		t,
		filepath.Join(projectDirectory, "source.go"),
		source.String(),
	)
	program, err := load.Load(context.Background(), load.Request{
		Directory: projectDirectory,
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
	result := waveSixScale{implementers: implementers}
	var printed strings.Builder
	for _, file := range emission.Files() {
		wire, err := tsgo.EncodeSourceFile(file.SourceFile())
		if err != nil {
			t.Fatal(err)
		}
		result.wireBytes += len(wire)
		target, err := client.PrintNode(file.SourceFile(), tsgo.PrintOptions{})
		if err != nil {
			t.Fatal(err)
		}
		result.printedBytes += len(target)
		printed.WriteString(target)
		printed.WriteByte('\n')
	}
	target := printed.String()
	result.adapters = strings.Count(
		target,
		"export class $goInterfaceAdapter_",
	)
	call := targetFunctionText(t, target, "Call")
	result.callBytes = len(call)
	if strings.Contains(call, "switch") ||
		strings.Contains(call, "$is(") ||
		!strings.Contains(call, ".Read(") {
		t.Fatalf("interface call is not direct O(1) dispatch:\n%s", call)
	}
	return result
}

func assertWaveSixLinearGrowth(
	t *testing.T,
	name string,
	values []waveSixScale,
	measure func(waveSixScale) int,
) {
	t.Helper()
	first := measure(values[1]) - measure(values[0])
	second := measure(values[2]) - measure(values[1])
	if first <= 0 || second <= 0 || second > first*23/10 {
		t.Fatalf(
			"%s growth is not linear: deltas %d, %d",
			name,
			first,
			second,
		)
	}
}
