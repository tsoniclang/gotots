package conversion_test

import (
	"context"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tsoniclang/gotots/internal/emit"
	"github.com/tsoniclang/gotots/internal/load"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func TestArrayStorageKeepsAddressAndAssignmentOwners(test *testing.T) {
	fixture := filepath.Join(repositoryRoot(), "testdata", "constructs", "value", "arraystorage")
	loaded, err := load.One(context.Background(), load.Request{Directory: fixture, Pattern: "."})
	if err != nil {
		test.Fatal(err)
	}
	roots, err := emit.ExportedAPIRoots(loaded)
	if err != nil {
		test.Fatal(err)
	}
	emission, err := emit.Compile(loaded.Program(), roots)
	if err != nil {
		test.Fatal(err)
	}
	strictTypecheckEmission(test, emission)
	client, err := tsgo.StartClient(repositoryRoot(), test.TempDir())
	if err != nil {
		test.Fatal(err)
	}
	test.Cleanup(func() { _ = client.Close() })
	var printed strings.Builder
	for _, file := range emission.Files() {
		text, printErr := client.PrintNode(file.SourceFile(), tsgo.PrintOptions{})
		if printErr != nil {
			test.Fatal(printErr)
		}
		printed.WriteString(text)
	}
	for _, required := range []string{
		"goSliceArrayPointer<uint32, 2>(goArraySlice(",
		"goSliceArrayPointer<uint32, 0>(goArraySlice(",
		"static $assign(",
		"Record.$assign(",
		"projectPointer<GoArray<uint32, 2>, Pair>",
		"addressOf<T | undefined>(backing[location[1]])",
		"$go$assign$T0_T0_to_T0",
	} {
		if !strings.Contains(printed.String(), required) {
			test.Fatalf("array storage output lacks %q", required)
		}
	}
	for _, forbidden := range []string{"array assignment is unsupported", " as any", " as unknown"} {
		if strings.Contains(printed.String(), forbidden) {
			test.Fatalf("array storage output contains %q", forbidden)
		}
	}
}
