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

func TestWaveFivePromotionScalesWithEmbeddingDepth(t *testing.T) {
	depths := []int{4, 8, 16}
	sourceBytes := make([]int, len(depths))
	targetBytes := make([]int, len(depths))
	targetNodes := make([]int, len(depths))
	for index, depth := range depths {
		source, target, nodes := compileWaveFiveScaling(t, depth)
		sourceBytes[index] = len(source)
		targetBytes[index] = len(target)
		targetNodes[index] = nodes
		if calls := strings.Count(target, "T0_Add("); calls != 2 {
			t.Fatalf(
				"promotion depth %d emits %d T0_Add sites, want declaration and one call",
				depth,
				calls,
			)
		}
		if strings.Contains(target, "switch (") ||
			strings.Contains(target, "extends T") {
			t.Fatalf(
				"promotion depth %d introduced dispatch/inheritance:\n%s",
				depth,
				target,
			)
		}
	}
	assertWaveFourLinearDoubling(t, "Wave 5 source bytes", sourceBytes)
	assertWaveFourLinearDoubling(t, "Wave 5 target bytes", targetBytes)
	assertWaveFourLinearDoubling(t, "Wave 5 target AST nodes", targetNodes)
	t.Logf(
		"Wave 5 depth=%v source=%v target=%v nodes=%v",
		depths,
		sourceBytes,
		targetBytes,
		targetNodes,
	)
}

func compileWaveFiveScaling(
	t *testing.T,
	depth int,
) (string, string, int) {
	t.Helper()
	directory := t.TempDir()
	writeProgramFile(
		t,
		filepath.Join(directory, "go.mod"),
		"module example.com/wave5scaling\n\ngo 1.26.4\n",
	)
	var source strings.Builder
	source.WriteString(
		"package wave5scaling\n\n" +
			"type T0 struct { Value int32 }\n" +
			"func (value *T0) Add(delta int32) { value.Value += delta }\n\n",
	)
	for index := 1; index <= depth; index++ {
		fmt.Fprintf(
			&source,
			"type T%d struct { T%d }\n",
			index,
			index-1,
		)
	}
	fmt.Fprintf(
		&source,
		"\nfunc Scale(value *T%d) int32 {\n"+
			"\tvalue.Add(1)\n"+
			"\treturn value.Value\n"+
			"}\n",
		depth,
	)
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
	client, err := tsgo.StartClient(repositoryRoot(), t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := client.Close(); err != nil {
			t.Errorf("close TS-Go client: %v", err)
		}
	})
	for _, file := range emission.Files() {
		if file.Kind() != emit.TargetFileSource {
			continue
		}
		printed, err := client.PrintNode(
			file.SourceFile(),
			tsgo.PrintOptions{},
		)
		if err != nil {
			t.Fatal(err)
		}
		encoded, err := tsgo.EncodeSourceFile(file.SourceFile())
		if err != nil {
			t.Fatal(err)
		}
		return source.String(), printed, waveFourEncodedNodes(t, encoded)
	}
	t.Fatal("Wave 5 scaling source artifact is absent")
	return "", "", 0
}
