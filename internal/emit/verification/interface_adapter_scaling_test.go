package emit_test

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tsoniclang/gotots/internal/emit"
	"github.com/tsoniclang/gotots/internal/load"
	"github.com/tsoniclang/gotots/internal/output"
)

type zeroMethodAdapterMeasurement struct {
	count        int
	bytes        int
	nodes        int
	factoryBytes int
}

func TestZeroMethodInterfaceAdaptersShareBoundedCommonMachinery(t *testing.T) {
	small := measureZeroMethodAdapters(t, 8)
	large := measureZeroMethodAdapters(t, 32)
	countDelta := large.count - small.count
	byteDelta := large.bytes - small.bytes
	nodeDelta := large.nodes - small.nodes
	if small.factoryBytes != large.factoryBytes {
		t.Fatalf(
			"adapter factory bytes changed with implementer count: %d -> %d",
			small.factoryBytes,
			large.factoryBytes,
		)
	}
	// Per-adapter growth contains only the executable adapter declaration; the
	// shared factory remains constant as implementer count grows.
	if countDelta != 24 || byteDelta/countDelta > 2_100 ||
		nodeDelta/countDelta > 210 {
		t.Fatalf(
			"zero-method adapter growth is not bounded: counts=%d/%d bytes=%d/%d nodes=%d/%d",
			small.count,
			large.count,
			small.bytes,
			large.bytes,
			small.nodes,
			large.nodes,
		)
	}
	t.Logf(
		"zero-method adapters counts=%d/%d bytes=%d/%d nodes=%d/%d factory=%d",
		small.count,
		large.count,
		small.bytes,
		large.bytes,
		small.nodes,
		large.nodes,
		large.factoryBytes,
	)
}

func measureZeroMethodAdapters(
	t *testing.T,
	count int,
) zeroMethodAdapterMeasurement {
	t.Helper()
	directory := t.TempDir()
	writeProgramFile(
		t,
		filepath.Join(directory, "go.mod"),
		"module example.com/interfaceadapterscaling\n\ngo 1.26.4\n",
	)
	var source strings.Builder
	source.WriteString("package interfaceadapterscaling\n\n")
	for index := 0; index < count; index++ {
		fmt.Fprintf(&source, "type value%d int32\n", index)
	}
	source.WriteString("\nfunc Audit() []any {\n\treturn []any{")
	for index := 0; index < count; index++ {
		fmt.Fprintf(&source, "value%d(%d),", index, index)
	}
	source.WriteString("}\n}\n")
	writeProgramFile(t, filepath.Join(directory, "source.go"), source.String())
	program, err := load.Load(context.Background(), load.Request{
		Directory: directory,
		Pattern:   ".",
	})
	if err != nil {
		t.Fatal(err)
	}
	root, err := emit.NewRoot(program.Roots()[0].Types().Scope().Lookup("Audit"))
	if err != nil {
		t.Fatal(err)
	}
	emission, err := emit.Compile(program, []emit.Root{root})
	if err != nil {
		t.Fatal(err)
	}
	targetDirectory := t.TempDir()
	artifacts := materializeArtifacts(t, emission, targetDirectory)
	waveThreeTypecheck(t, targetDirectory, artifacts.paths)
	measurement := zeroMethodAdapterMeasurement{count: count}
	for _, size := range artifacts.sizes {
		switch {
		case strings.HasPrefix(size.path, output.InterfaceAdapterSupportRoot+"/"):
			measurement.bytes += size.bytes
			measurement.nodes += size.nodes
		case size.path == "runtime/interface-value.ts":
			measurement.factoryBytes = size.bytes
		}
	}
	if measurement.bytes == 0 || measurement.nodes == 0 ||
		measurement.factoryBytes == 0 {
		t.Fatalf("adapter scaling artifacts are incomplete: %#v", measurement)
	}
	if calls := strings.Count(
		artifacts.printed,
		"= createGoInterfaceAdapter<",
	); calls != count {
		t.Fatalf("adapter factory calls = %d, want %d", calls, count)
	}
	if factories := strings.Count(
		artifacts.printed,
		"function createGoInterfaceAdapter<T>(",
	); factories != 1 {
		t.Fatalf("adapter factory definitions = %d, want one", factories)
	}
	if strings.Contains(
		artifacts.printed,
		"export class $goInterfaceAdapter$",
	) {
		t.Fatal("zero-method adapter was emitted as a repeated class")
	}
	return measurement
}
