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

func TestWaveNineSerialTerminationBoundariesAreExplicit(t *testing.T) {
	testCases := []struct {
		name     string
		function string
		want     string
	}{
		{"send-closed", "PanicSendClosed", "panic:send on closed channel"},
		{"close-nil", "PanicCloseNil", "panic:close of nil channel"},
		{"close-closed", "PanicCloseClosed", "panic:close of closed channel"},
		{
			"nil-receive",
			"DeadlockNilReceive",
			"panic:serial channel receive would block",
		},
		{
			"goroutine-panic",
			"PanicGoroutine",
			"panic:send on closed channel",
		},
		{
			"blocked-goroutine",
			"ReturnWithBlockedGoroutine",
			"panic:serial channel receive would block",
		},
	}
	program, err := load.Load(context.Background(), load.Request{
		Directory: waveNineConcurrencyDirectory(),
		Pattern:   ".",
	})
	if err != nil {
		t.Fatal(err)
	}
	scope := program.Roots()[0].Types().Scope()
	roots := make([]emit.Root, 0, len(testCases))
	for _, testCase := range testCases {
		root, rootError := emit.NewRoot(scope.Lookup(testCase.function))
		if rootError != nil {
			t.Fatal(rootError)
		}
		roots = append(roots, root)
	}
	emission, err := emit.CompileWithOptions(
		program,
		roots,
		waveNineOptions(),
	)
	if err != nil {
		t.Fatal(err)
	}
	workingDirectory := t.TempDir()
	artifacts := materializeArtifacts(t, emission, workingDirectory)
	writeProgramFile(
		t,
		filepath.Join(workingDirectory, "package.json"),
		"{\"type\":\"module\"}\n",
	)
	runners := make([]string, 0, len(testCases))
	for _, testCase := range testCases {
		runner := filepath.Join(workingDirectory, "runner-"+testCase.name+".ts")
		writeProgramFile(t, runner, waveNineSerialTargetRunner(
			artifacts.sourceModule,
			testCase.function,
		))
		runners = append(runners, runner)
	}
	waveThreeTypecheck(
		t,
		workingDirectory,
		append(artifacts.paths, runners...),
	)
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			output := runProgram(
				t,
				workingDirectory,
				"node",
				filepath.Join(
					workingDirectory,
					"out",
					"runner-"+testCase.name+".js",
				),
			)
			if strings.TrimSpace(output) != testCase.want {
				t.Fatalf("serial outcome = %q, want %q", output, testCase.want)
			}
		})
	}
}

func waveNineSerialTargetRunner(module, function string) string {
	return `import "./program.js";
import { ` + function + ` } from "` + module + `";
import { GoPanic, GoRuntimePanicValue } from "./runtime/panic.js";

try {
    ` + function + `();
    console.log("return");
} catch (failure) {
    console.log(
        failure instanceof GoPanic &&
            failure.value instanceof GoRuntimePanicValue
            ? "panic:" + failure.value.message
            : "wrong failure",
    );
}
`
}

func waveNineOptions() emit.Options {
	return emit.DefaultOptions()
}

func TestWaveNineSerialPreemptionBoundaryAddsNoYieldHeuristic(t *testing.T) {
	program, err := load.Load(context.Background(), load.Request{
		Directory: waveNineConcurrencyDirectory(),
		Pattern:   ".",
	})
	if err != nil {
		t.Fatal(err)
	}
	root, err := emit.NewRoot(
		program.Roots()[0].Types().Scope().Lookup("RequiresPreemption"),
	)
	if err != nil {
		t.Fatal(err)
	}
	emission, err := emit.CompileWithOptions(
		program,
		[]emit.Root{root},
		waveNineOptions(),
	)
	if err != nil {
		t.Fatal(err)
	}
	artifacts := materializeArtifacts(t, emission, t.TempDir())
	target := waveNineFunctionText(t, artifacts.printed, "RequiresPreemption")
	if !strings.Contains(target, "for (;;)") {
		t.Fatalf("serial preemption boundary lost its source loop:\n%s", target)
	}
	for _, forbidden := range []string{
		"async",
		"await",
		"Promise",
		"GoScheduler",
		"setTimeout",
		"queueMicrotask",
		"setImmediate",
		"yield",
	} {
		if strings.Contains(target, forbidden) {
			t.Fatalf("serial preemption boundary acquired %q:\n%s", forbidden, target)
		}
	}
}

func TestWaveNineChannelCallsAreConstantAndSelectIsLinear(t *testing.T) {
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
			t.Fatalf("one send/receive site changed at %d select cases", count)
		}
		if measurement.runtimeText != constantRuntime {
			t.Fatalf("channel runtime changed at %d select cases", count)
		}
	}
	if strings.Count(constantCallSite, "GoChannel.send") != 1 ||
		strings.Count(constantCallSite, "GoChannel.receive") != 1 {
		t.Fatalf("constant channel site is not one send/receive:\n%s", constantCallSite)
	}
	assertWaveFourLinearDoubling(t, "Wave 9 select bytes", selectBytes)
	assertWaveFourLinearDoubling(t, "Wave 9 select AST nodes", selectNodes)
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
	writeProgramFile(t, filepath.Join(directory, "source.go"), source.String())
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
		printed, printError := client.PrintNode(
			file.SourceFile(),
			tsgo.PrintOptions{},
		)
		if printError != nil {
			t.Fatal(printError)
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
				encoded, encodeError := tsgo.EncodeNode(function)
				if encodeError != nil {
					t.Fatal(encodeError)
				}
				switch function.Name().Text() {
				case "ChannelCall":
					measurement.callText = waveNineFunctionText(
						t,
						printed,
						"ChannelCall",
					)
					measurement.callNodes = waveFourEncodedNodes(t, encoded)
				case "SelectScale":
					measurement.selectText = waveNineFunctionText(
						t,
						printed,
						"SelectScale",
					)
					measurement.selectNodes = waveFourEncodedNodes(t, encoded)
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
