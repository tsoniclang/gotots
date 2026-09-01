package emit_test

import (
	"context"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tsoniclang/gotots/internal/emit"
	"github.com/tsoniclang/gotots/internal/load"
)

func TestGenericContainerStorageBindsExactTargetFacets(t *testing.T) {
	directory := filepath.Join(
		repositoryRoot(),
		"testdata",
		"constructs",
		"generic",
		"container-storage",
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
	root, err := emit.NewRoot(
		program.Roots()[0].Types().Scope().Lookup("Audit"),
	)
	if err != nil {
		t.Fatal(err)
	}
	emission, err := emit.Compile(program, []emit.Root{root})
	if err != nil {
		t.Fatal(err)
	}
	workingDirectory := t.TempDir()
	artifacts := materializeArtifacts(t, emission, workingDirectory)
	for index, size := range artifacts.sizes {
		if index == 20 {
			break
		}
		t.Logf("artifact %s bytes=%d nodes=%d", size.path, size.bytes, size.nodes)
	}
	ordinaryBytes := artifacts.bytes
	concretizationBytes := 0
	concretizations := 0
	capabilityBytes := 0
	capabilities := 0
	for _, size := range artifacts.sizes {
		switch {
		case strings.HasPrefix(
			size.path,
			"support/generics/concretizations/",
		):
			ordinaryBytes -= size.bytes
			concretizationBytes += size.bytes
			concretizations++
		case strings.HasPrefix(
			size.path,
			"support/generics/capabilities/",
		):
			ordinaryBytes -= size.bytes
			capabilityBytes += size.bytes
			capabilities++
		}
	}
	for _, required := range []string{
		"class Bag<T>",
		"class Arena<T>",
		"RuntimeSlice<T>",
		"RuntimeSlice<GoContainerStorage<T>>",
		"Bag.$zero<PlainItem>()",
		"Arena.$zero<Item>()",
		"class Item implements GoContainerStoredValue<Item$Storage>",
		"Pointer<Item>",
		"Pointer<int32>",
		"addressOf<",
		"loadPointer<",
		"function ArrayAddress$kernel<T>",
		"function ArrayAddress$",
		"RuntimeSlice.literal<Item$Storage>([Item.$storageOf(",
	} {
		if !strings.Contains(artifacts.printed, required) {
			t.Fatalf(
				"generic container-storage artifact lacks %q",
				required,
			)
		}
	}
	for _, forbidden := range []string{
		"T$Storage",
		"T$ContainerStorage",
		"T$Pointer",
		"RuntimeSlice<PlainItem$Storage>",
		"GoPointer",
		"runtime/pointer",
	} {
		if strings.Contains(artifacts.printed, forbidden) {
			t.Fatalf("generic container-storage artifact exposes %q", forbidden)
		}
	}
	writeProgramFile(
		t,
		filepath.Join(workingDirectory, "package.json"),
		"{\"type\":\"module\"}\n",
	)
	waveThreeTypecheck(
		t,
		workingDirectory,
		artifacts.paths,
	)
	// Canonical facts are counted here and erased only by the executable
	// TypeScript target, whose output has an independent budget.
	if ordinaryBytes > 75_000 ||
		concretizations != 7 || concretizationBytes > 23_000 ||
		capabilities != 0 || capabilityBytes != 0 ||
		artifacts.bytes > 100_000 ||
		artifacts.nodes > 12_500 ||
		artifacts.largest > 45_000 {
		t.Fatalf(
			"generic container-storage artifact bounds exceeded: ordinary=%d concretizations=%d/%d capabilities=%d/%d total=%d nodes=%d largest=%d",
			ordinaryBytes,
			concretizations,
			concretizationBytes,
			capabilities,
			capabilityBytes,
			artifacts.bytes,
			artifacts.nodes,
			artifacts.largest,
		)
	}
	t.Logf(
		"generic container-storage artifacts files=%d bytes=%d nodes=%d largest=%d",
		len(artifacts.paths),
		artifacts.bytes,
		artifacts.nodes,
		artifacts.largest,
	)
}
