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

func TestWaveNineChannelCallsAreConstantAndSelectIsLinear(
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
	counts := []int{4, 8, 16}
	selectBytes := make([]int, len(counts))
	selectNodes := make([]int, len(counts))
	var constantCallSite string
	var constantCallNodes int
	var constantRuntime string
	for index, count := range counts {
		measurement := compileWaveNineScaling(t, client, count)
		selectBytes[index] = len(measurement.selectText)
		selectNodes[index] = measurement.selectNodes
		if alternatives := strings.Count(
			measurement.selectText,
			"GoChannel.$selectReceive",
		); alternatives != count {
			t.Fatalf(
				"select alternatives at %d cases = %d, want %d",
				count,
				alternatives,
				count,
			)
		}
		if index == 0 {
			constantCallSite = measurement.callText
			constantCallNodes = measurement.callNodes
			constantRuntime = measurement.runtimeText
			continue
		}
		if measurement.callText != constantCallSite ||
			measurement.callNodes != constantCallNodes {
			t.Fatalf(
				"one send/receive site changed at %d unrelated select cases",
				count,
			)
		}
		if measurement.runtimeText != constantRuntime {
			t.Fatalf(
				"channel/scheduler runtime changed at %d select cases",
				count,
			)
		}
	}
	if strings.Count(constantCallSite, "GoChannel.send") != 1 ||
		strings.Count(constantCallSite, "GoChannel.receive") != 1 {
		t.Fatalf("constant channel site is not one send/receive:\n%s", constantCallSite)
	}
	assertWaveFourLinearDoubling(t, "Wave 9 select bytes", selectBytes)
	assertWaveFourLinearDoubling(t, "Wave 9 select AST nodes", selectNodes)
	t.Logf(
		"Wave 9 scaling cases=%v select-bytes=%v select-nodes=%v call-bytes/nodes=%d/%d runtime-bytes=%d",
		counts,
		selectBytes,
		selectNodes,
		len(constantCallSite),
		constantCallNodes,
		len(constantRuntime),
	)
}

type waveNineScalingMeasurement struct {
	callText    string
	callNodes   int
	selectText  string
	selectNodes int
	runtimeText string
}

func compileWaveNineScaling(
	t *testing.T,
	client *tsgo.Client,
	count int,
) waveNineScalingMeasurement {
	t.Helper()
	directory := t.TempDir()
	writeProgramFile(
		t,
		filepath.Join(directory, "go.mod"),
		"module example.com/wave9scaling\n\ngo 1.26.4\n",
	)
	var source strings.Builder
	source.WriteString(`package wave9scaling

func ChannelCall() int32 {
	values := make(chan int32, 1)
	values <- 1
	return <-values
}

func SelectScale(
`)
	for index := range count {
		fmt.Fprintf(&source, "\tchannel%d chan int32,\n", index)
	}
	source.WriteString(") int32 {\n\tselect {\n")
	for index := range count {
		fmt.Fprintf(
			&source,
			"\tcase <-channel%d:\n\t\treturn %d\n",
			index,
			index,
		)
	}
	source.WriteString("\tdefault:\n\t\treturn -1\n\t}\n}\n")
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
	emission, err := emit.CompileWithOptions(
		program,
		roots,
		waveNineOptions(),
	)
	if err != nil {
		t.Fatal(err)
	}
	var measurement waveNineScalingMeasurement
	for _, file := range emission.Files() {
		printed, err := client.PrintNode(
			file.SourceFile(),
			tsgo.PrintOptions{},
		)
		if err != nil {
			t.Fatal(err)
		}
		switch file.OutputPath() {
		case "runtime/channel.ts":
			measurement.runtimeText = printed
		default:
			if file.Kind() != emit.TargetFileSource {
				continue
			}
			for _, statement := range file.SourceFile().Statements() {
				function, ok := statement.(tsgo.FunctionDeclaration)
				if !ok {
					continue
				}
				encoded, err := tsgo.EncodeNode(function)
				if err != nil {
					t.Fatal(err)
				}
				switch function.Name().Text() {
				case "ChannelCall":
					measurement.callText = waveNineFunctionText(
						t,
						printed,
						"ChannelCall",
					)
					measurement.callNodes =
						waveFourEncodedNodes(t, encoded)
				case "SelectScale":
					measurement.selectText = waveNineFunctionText(
						t,
						printed,
						"SelectScale",
					)
					measurement.selectNodes =
						waveFourEncodedNodes(t, encoded)
				}
			}
		}
	}
	if measurement.callText == "" ||
		measurement.callNodes == 0 ||
		measurement.selectText == "" ||
		measurement.selectNodes == 0 ||
		measurement.runtimeText == "" {
		t.Fatal("Wave 9 scaling artifacts are incomplete")
	}
	return measurement
}
