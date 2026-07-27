package stringvalue_test

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

func TestStringUseSitesScaleLinearlyAndRuntimeStaysConstant(t *testing.T) {
	counts := []int{1, 2, 4}
	sourceBytes := make([]int, len(counts))
	targetBytes := make([]int, len(counts))
	encodedBytes := make([]int, len(counts))
	targetNodes := make([]int, len(counts))
	var runtimeSource string
	for index, count := range counts {
		source, emission := compileScalingStrings(t, count)
		printedSource, printedRuntime := printScalingStrings(t, emission)
		encoded, err := tsgo.EncodeSourceFile(
			targetFile(t, emission, emit.TargetFileSource),
		)
		if err != nil {
			t.Fatal(err)
		}
		sourceBytes[index] = len(source)
		targetBytes[index] = len(printedSource)
		encodedBytes[index] = len(encoded)
		targetNodes[index] = encodedNodeCount(t, encoded)
		if index == 0 {
			runtimeSource = printedRuntime
		} else if printedRuntime != runtimeSource {
			t.Fatal("string runtime grew with use-site count")
		}
	}
	assertLinearDoubling(t, "source bytes", sourceBytes)
	assertLinearDoubling(t, "target bytes", targetBytes)
	assertLinearDoubling(t, "encoded AST bytes", encodedBytes)
	assertLinearDoubling(t, "target AST nodes", targetNodes)
	t.Logf(
		"string scaling sites=%v source=%v target=%v encoded=%v nodes=%v runtime=%d",
		counts,
		sourceBytes,
		targetBytes,
		encodedBytes,
		targetNodes,
		len(runtimeSource),
	)
}

func encodedNodeCount(t *testing.T, encoded []byte) int {
	t.Helper()
	const (
		headerSize       = 44
		nodesOffsetField = 40
		nodeWidth        = 28
	)
	if len(encoded) < headerSize {
		t.Fatalf("encoded TS-Go AST = %d bytes, shorter than protocol header", len(encoded))
	}
	nodesOffset := int(binary.LittleEndian.Uint32(encoded[nodesOffsetField:headerSize]))
	if nodesOffset < headerSize ||
		nodesOffset > len(encoded) ||
		(len(encoded)-nodesOffset)%nodeWidth != 0 {
		t.Fatalf("encoded TS-Go AST has invalid node section offset %d", nodesOffset)
	}
	return (len(encoded) - nodesOffset) / nodeWidth
}

func compileScalingStrings(t *testing.T, count int) (string, emit.ProgramEmission) {
	t.Helper()
	directory := t.TempDir()
	writeFile(t, filepath.Join(directory, "go.mod"), "module example.com/scalingstrings\n\ngo 1.26.4\n")
	var source strings.Builder
	source.WriteString("package scalingstrings\n\n")
	for index := range count {
		fmt.Fprintf(
			&source,
			"func Index%d(value string, at int) byte { return value[at] }\n",
			index,
		)
		fmt.Fprintf(
			&source,
			"func Slice%d(value string, low int, high int) string { return value[low:high] }\n",
			index,
		)
	}
	writeFile(t, filepath.Join(directory, "source.go"), source.String())
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

func printScalingStrings(
	t *testing.T,
	emission emit.ProgramEmission,
) (string, string) {
	t.Helper()
	workingDirectory := t.TempDir()
	client, err := tsgo.StartClient(repositoryRoot(), workingDirectory)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := client.Close(); err != nil {
			t.Errorf("close TS-Go client: %v", err)
		}
	})
	var source string
	var runtimeSource string
	for _, file := range emission.Files() {
		if file.Kind() != emit.TargetFileSource &&
			file.OutputPath() != "runtime/string.ts" {
			continue
		}
		printed, err := client.PrintNode(file.SourceFile(), tsgo.PrintOptions{})
		if err != nil {
			t.Fatal(err)
		}
		if file.Kind() == emit.TargetFileSource {
			source = printed
		} else {
			runtimeSource = printed
		}
	}
	if source == "" || runtimeSource == "" {
		t.Fatal("scaling fixture lacks source or string runtime output")
	}
	return source, runtimeSource
}

func assertLinearDoubling(t *testing.T, measure string, values []int) {
	t.Helper()
	first := values[1] - values[0]
	second := values[2] - values[1]
	if first <= 0 || second < first*19/10 || second > first*21/10 {
		t.Fatalf("%s = %v, want constant per-site growth", measure, values)
	}
}
