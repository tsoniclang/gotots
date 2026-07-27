package pointer_test

import (
	"bytes"
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

func TestAddressableStorageSitesScaleLinearlyAndLeavePlainArtifactStable(
	t *testing.T,
) {
	counts := []int{2, 4, 8}
	sourceBytes := make([]int, len(counts))
	targetBytes := make([]int, len(counts))
	targetNodes := make([]int, len(counts))
	var runtimeSource string
	var plainArtifact []byte
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
	for index, count := range counts {
		measurement := measureAddressableStorageScale(t, client, count)
		sourceBytes[index] = measurement.sourceBytes
		targetBytes[index] = measurement.targetBytes
		targetNodes[index] = measurement.targetNodes
		if measurement.cells != count {
			t.Fatalf(
				"addressable cells = %d, want %d",
				measurement.cells,
				count,
			)
		}
		if measurement.ordinaryCells != 0 {
			t.Fatalf(
				"ordinary locals wrapped as cells = %d",
				measurement.ordinaryCells,
			)
		}
		if index == 0 {
			runtimeSource = measurement.runtimeSource
			plainArtifact = measurement.plainArtifact
			continue
		}
		if measurement.runtimeSource != runtimeSource {
			t.Fatal("pointer runtime grew with address-site count")
		}
		if !bytes.Equal(measurement.plainArtifact, plainArtifact) {
			t.Fatal("unaddressed function artifact changed with address-site count")
		}
	}
	assertDoublingDeltas(t, "addressable Go source bytes", sourceBytes)
	assertDoublingDeltas(t, "addressable TypeScript bytes", targetBytes)
	assertDoublingDeltas(t, "addressable TS-Go AST nodes", targetNodes)
	t.Logf(
		"addressable scaling sites=%v source=%v target=%v nodes=%v runtime=%d",
		counts,
		sourceBytes,
		targetBytes,
		targetNodes,
		len(runtimeSource),
	)
}

type addressableStorageScale struct {
	sourceBytes   int
	targetBytes   int
	targetNodes   int
	cells         int
	ordinaryCells int
	runtimeSource string
	plainArtifact []byte
}

func measureAddressableStorageScale(
	t *testing.T,
	client *tsgo.Client,
	count int,
) addressableStorageScale {
	t.Helper()
	directory := t.TempDir()
	writeFile(
		t,
		filepath.Join(directory, "go.mod"),
		"module example.com/addressablescaling\n\ngo 1.26.4\n",
	)
	var source strings.Builder
	source.WriteString(`package addressablescaling

func Plain(value int32) int32 {
	ordinary := value + 1
	return ordinary
}

`)
	for index := range count {
		fmt.Fprintf(
			&source,
			"func Address%d(value int32) int32 { ordinary := value + 1; addressed := value; pointer := &addressed; *pointer++; return ordinary + addressed }\n",
			index,
		)
	}
	writeFile(t, filepath.Join(directory, "source.go"), source.String())
	loaded, err := load.One(context.Background(), load.Request{
		Directory: directory,
		Pattern:   ".",
	})
	if err != nil {
		t.Fatal(err)
	}
	emission := compileAddressablePointerProject(t, loaded)
	measurement := addressableStorageScale{sourceBytes: source.Len()}
	for _, file := range emission.Files() {
		printed, err := client.PrintNode(file.SourceFile(), tsgo.PrintOptions{})
		if err != nil {
			t.Fatal(err)
		}
		if file.OutputPath() == "runtime/pointer.ts" {
			measurement.runtimeSource = printed
		}
		if file.Kind() != emit.TargetFileSource {
			continue
		}
		measurement.targetBytes += len(printed)
		measurement.cells += strings.Count(printed, "GoPointer.cell")
		measurement.ordinaryCells += strings.Count(printed, "ordinary$storage")
		encoded, err := tsgo.EncodeSourceFile(file.SourceFile())
		if err != nil {
			t.Fatal(err)
		}
		measurement.targetNodes += pointerEncodedNodeCount(t, encoded)
		for _, statement := range file.SourceFile().Statements() {
			function, ok := statement.(tsgo.FunctionDeclaration)
			if !ok || function.Name().Text() != "Plain" {
				continue
			}
			measurement.plainArtifact, err = tsgo.EncodeNode(function)
			if err != nil {
				t.Fatal(err)
			}
		}
	}
	if measurement.runtimeSource == "" || len(measurement.plainArtifact) == 0 {
		t.Fatal("scaling measurement lacks runtime or plain function artifact")
	}
	return measurement
}

func pointerEncodedNodeCount(t *testing.T, encoded []byte) int {
	t.Helper()
	const (
		headerSize       = 44
		nodesOffsetField = 40
		nodeWidth        = 28
	)
	if len(encoded) < headerSize {
		t.Fatalf("encoded TS-Go AST is shorter than its protocol header")
	}
	nodesOffset := int(binary.LittleEndian.Uint32(
		encoded[nodesOffsetField:headerSize],
	))
	if nodesOffset < headerSize ||
		nodesOffset > len(encoded) ||
		(len(encoded)-nodesOffset)%nodeWidth != 0 {
		t.Fatalf("encoded TS-Go AST has invalid node offset %d", nodesOffset)
	}
	return (len(encoded) - nodesOffset) / nodeWidth
}
