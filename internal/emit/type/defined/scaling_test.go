package defined_test

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

func TestDefinedDeclarationsScaleLinearly(t *testing.T) {
	counts := []int{1, 2, 4}
	sourceBytes := make([]int, len(counts))
	targetBytes := make([]int, len(counts))
	targetNodes := make([]int, len(counts))
	client, err := tsgo.StartClient(repositoryRoot(), t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = client.Close() })
	for index, count := range counts {
		source, target := compileDefinedScaling(t, count)
		printed, err := client.PrintNode(target, tsgo.PrintOptions{})
		if err != nil {
			t.Fatal(err)
		}
		encoded, err := tsgo.EncodeSourceFile(target)
		if err != nil {
			t.Fatal(err)
		}
		sourceBytes[index] = len(source)
		targetBytes[index] = len(printed)
		targetNodes[index] = definedEncodedNodeCount(t, encoded)
		assertDefinedAliasCount(t, target, count)
	}
	assertAffineDoubling(t, "source bytes", sourceBytes)
	assertAffineDoubling(t, "target bytes", targetBytes)
	assertAffineDoubling(t, "target AST nodes", targetNodes)
	t.Logf(
		"defined scaling types=%v source=%v target=%v nodes=%v",
		counts,
		sourceBytes,
		targetBytes,
		targetNodes,
	)
}

func compileDefinedScaling(
	t *testing.T,
	count int,
) (string, tsgo.SourceFile) {
	t.Helper()
	directory := t.TempDir()
	writeDefinedFile(
		t,
		filepath.Join(directory, "go.mod"),
		"module example.com/definedscaling\n\ngo 1.26.4\n",
	)
	var source strings.Builder
	source.WriteString("package definedscaling\n\n")
	for index := range count {
		fmt.Fprintf(&source, "type T%d int32\n", index)
	}
	writeDefinedFile(
		t,
		filepath.Join(directory, "source.go"),
		source.String(),
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
	for _, file := range emission.Files() {
		if file.Kind() == emit.TargetFileSource {
			return source.String(), file.SourceFile()
		}
	}
	t.Fatal("defined scaling target source is absent")
	return "", nil
}

func assertDefinedAliasCount(
	t *testing.T,
	source tsgo.SourceFile,
	want int,
) {
	t.Helper()
	count := 0
	for _, statement := range source.Statements() {
		_, ok := statement.(tsgo.TypeAliasDeclaration)
		if !ok {
			continue
		}
		count++
	}
	if count != want {
		t.Fatalf("defined scaling aliases = %d, want %d", count, want)
	}
}

func assertAffineDoubling(
	t *testing.T,
	name string,
	values []int,
) {
	t.Helper()
	if len(values) != 3 {
		t.Fatalf("%s samples = %d, want 3", name, len(values))
	}
	firstDelta := values[1] - values[0]
	secondDelta := values[2] - values[1]
	if firstDelta <= 0 || secondDelta != 2*firstDelta {
		t.Fatalf("%s = %v, want exact affine doubling", name, values)
	}
}

func definedEncodedNodeCount(t *testing.T, encoded []byte) int {
	t.Helper()
	const (
		headerSize       = 44
		nodesOffsetField = 40
		nodeWidth        = 28
	)
	if len(encoded) < headerSize {
		t.Fatalf("encoded AST is only %d bytes", len(encoded))
	}
	nodesOffset := int(binary.LittleEndian.Uint32(
		encoded[nodesOffsetField:headerSize],
	))
	if nodesOffset < headerSize ||
		nodesOffset > len(encoded) ||
		(len(encoded)-nodesOffset)%nodeWidth != 0 {
		t.Fatalf("encoded AST node offset %d is invalid", nodesOffset)
	}
	return (len(encoded) - nodesOffset) / nodeWidth
}
