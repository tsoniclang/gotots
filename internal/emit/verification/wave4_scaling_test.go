package emit_test

import (
	"context"
	"encoding/binary"
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tsoniclang/gotots/internal/emit"
	"github.com/tsoniclang/gotots/internal/load"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func TestWaveFourRangeAndSwitchScaleWithSourceSyntax(t *testing.T) {
	counts := []int{4, 8, 16}
	sourceBytes := make([]int, len(counts))
	targetBytes := make([]int, len(counts))
	targetNodes := make([]int, len(counts))
	for index, count := range counts {
		source, target, nodes := compileWaveFourScaling(t, count)
		sourceBytes[index] = len(source)
		targetBytes[index] = len(target)
		targetNodes[index] = nodes
		if loops := strings.Count(
			target,
			"for (let __gotots_range_index_",
		); loops != 1 {
			t.Fatalf("range loops at %d cases = %d, want one", count, loops)
		}
		if checks := strings.Count(
			target,
			"let __gotots_switch_match_",
		); checks != count {
			t.Fatalf(
				"switch checks at %d cases = %d, want %d",
				count,
				checks,
				count,
			)
		}
	}
	assertWaveFourLinearDoubling(t, "source bytes", sourceBytes)
	assertWaveFourLinearDoubling(t, "target bytes", targetBytes)
	assertWaveFourLinearDoubling(t, "target AST nodes", targetNodes)
	t.Logf(
		"Wave 4 scaling cases=%v source=%v target=%v nodes=%v",
		counts,
		sourceBytes,
		targetBytes,
		targetNodes,
	)
}

func compileWaveFourScaling(
	t *testing.T,
	count int,
) (string, string, int) {
	t.Helper()
	directory := t.TempDir()
	writeProgramFile(
		t,
		filepath.Join(directory, "go.mod"),
		"module example.com/wave4scaling\n\ngo 1.26.4\n",
	)
	var source strings.Builder
	fmt.Fprintf(
		&source,
		"package wave4scaling\n\nfunc Scale(value [2]int32) int32 {\n"+
			"\tvar total int32\n"+
			"\tfor index := range [%d]int32{} { total += int32(index) }\n"+
			"\tswitch value {\n",
		count,
	)
	for index := range count {
		fmt.Fprintf(
			&source,
			"\tcase [2]int32{%d, %d}: total += %d\n",
			index,
			index+1,
			index,
		)
	}
	source.WriteString("\tdefault: total--\n\t}\n\treturn total\n}\n")
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
		printed, err := client.PrintNode(file.SourceFile(), tsgo.PrintOptions{})
		if err != nil {
			t.Fatal(err)
		}
		encoded, err := tsgo.EncodeSourceFile(file.SourceFile())
		if err != nil {
			t.Fatal(err)
		}
		return source.String(), printed, waveFourEncodedNodes(t, encoded)
	}
	t.Fatal("Wave 4 scaling source artifact is absent")
	return "", "", 0
}

func waveFourEncodedNodes(t *testing.T, encoded []byte) int {
	t.Helper()
	const (
		headerSize       = 44
		nodesOffsetField = 40
		nodeWidth        = 28
	)
	if len(encoded) < headerSize {
		t.Fatalf("encoded target is %d bytes, want protocol header", len(encoded))
	}
	nodesOffset := int(binary.LittleEndian.Uint32(
		encoded[nodesOffsetField:headerSize],
	))
	if nodesOffset < headerSize ||
		nodesOffset > len(encoded) ||
		(len(encoded)-nodesOffset)%nodeWidth != 0 {
		t.Fatalf("encoded target has invalid node offset %d", nodesOffset)
	}
	return (len(encoded) - nodesOffset) / nodeWidth
}

func assertWaveFourLinearDoubling(
	t *testing.T,
	name string,
	values []int,
) {
	t.Helper()
	first := values[1] - values[0]
	second := values[2] - values[1]
	if first <= 0 || second*10 < first*17 || second*10 > first*23 {
		t.Fatalf(
			"%s = %v; doubling deltas %d/%d are not linear",
			name,
			values,
			first,
			second,
		)
	}
}
