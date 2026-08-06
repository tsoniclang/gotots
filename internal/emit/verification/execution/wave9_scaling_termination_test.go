package emit_test

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/tsoniclang/gotots/internal/emit"
	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/load"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func waveNineFunctionText(t *testing.T, printed, name string) string {
	t.Helper()
	start := strings.Index(printed, "export function "+name)
	if start < 0 {
		start = strings.Index(printed, "export async function "+name)
	}
	if start < 0 {
		t.Fatalf("Wave 9 artifacts lack function %s", name)
	}
	end := len(printed) - start
	for _, marker := range []string{"\nexport ", "\n// "} {
		candidate := strings.Index(printed[start:], marker)
		if candidate >= 0 && candidate < end {
			end = candidate
		}
	}
	if end == len(printed)-start {
		return printed[start:]
	}
	return printed[start : start+end]
}

func TestWaveNineTerminationMatchesGo(t *testing.T) {
	testCases := []struct {
		name     string
		function string
		want     string
	}{
		{"send-closed", "PanicSendClosed", "panic:send on closed channel"},
		{"close-nil", "PanicCloseNil", "panic:close of nil channel"},
		{"close-closed", "PanicCloseClosed", "panic:close of closed channel"},
		{
			"nil-receive-deadlock",
			"DeadlockNilReceive",
			"panic:all goroutines are asleep - deadlock!",
		},
		{
			"uncaught-goroutine-panic",
			"PanicGoroutine",
			"panic:send on closed channel",
		},
		{
			"main-return",
			"ReturnWithBlockedGoroutine",
			"return",
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
		root, rootErr := emit.NewRoot(scope.Lookup(testCase.function))
		if rootErr != nil {
			t.Fatal(rootErr)
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
	targetRunners := make([]string, 0, len(testCases))
	for _, testCase := range testCases {
		runner := filepath.Join(
			workingDirectory,
			"runner-"+testCase.name+".ts",
		)
		writeProgramFile(t, runner, waveNineTargetRunner(
			artifacts.sourceModule,
			testCase.function,
		))
		targetRunners = append(targetRunners, runner)
	}
	waveThreeTypecheck(
		t,
		workingDirectory,
		append(artifacts.paths, targetRunners...),
	)
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			targetOutput, targetErr := runWaveNineCommand(
				workingDirectory,
				"node",
				filepath.Join(
					workingDirectory,
					"out",
					"runner-"+testCase.name+".js",
				),
			)
			targetOutcome := waveNineOutcome(targetOutput, targetErr)
			goOutput, goErr := executeWaveNineFailureGo(
				t,
				testCase.name,
				testCase.function,
			)
			goOutcome := waveNineOutcome(goOutput, goErr)
			if targetOutcome != testCase.want ||
				goOutcome != testCase.want {
				t.Fatalf(
					"termination differs: target=%q Go=%q want=%q\n"+
						"target output:\n%s\nGo output:\n%s",
					targetOutcome,
					goOutcome,
					testCase.want,
					targetOutput,
					goOutput,
				)
			}
		})
	}
}

func waveNineTargetRunner(module, function string) string {
	return `import "./program.js";
import { ` + function + ` } from "` + module + `";
import { GoScheduler } from "./runtime/channel.js";
import { GoPanic, GoRuntimePanicValue } from "./runtime/panic.js";

try {
    await GoScheduler.run(async () => {
        await ` + function + `();
    });
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

func executeWaveNineFailureGo(
	t *testing.T,
	name string,
	function string,
) (string, error) {
	t.Helper()
	modulePath, err := filepath.Abs(waveNineConcurrencyDirectory())
	if err != nil {
		t.Fatal(err)
	}
	runnerDirectory := filepath.Join(t.TempDir(), "go-runner-"+name)
	writeProgramFile(t, filepath.Join(runnerDirectory, "go.mod"), fmt.Sprintf(
		`module example.com/runner

go 1.26.4

require example.com/wave9concurrency v0.0.0

replace example.com/wave9concurrency => %s
`,
		filepath.ToSlash(modulePath),
	))
	writeProgramFile(t, filepath.Join(runnerDirectory, "main.go"), `package main

import (
	"fmt"

	values "example.com/wave9concurrency"
)

func main() {
	values.`+function+`()
	fmt.Println("return")
}
`)
	return runWaveNineCommand(
		runnerDirectory,
		filepath.Join(runtime.GOROOT(), "bin", "go"),
		"run",
		".",
	)
}

func runWaveNineCommand(
	directory string,
	name string,
	arguments ...string,
) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	command := exec.CommandContext(ctx, name, arguments...)
	command.Dir = directory
	command.Env = append(os.Environ(), "GOMEMLIMIT=1GiB")
	output, err := command.CombinedOutput()
	return string(output), err
}

func waveNineOutcome(output string, runError error) string {
	for _, failure := range []string{
		"all goroutines are asleep - deadlock!",
		"send on closed channel",
		"close of nil channel",
		"close of closed channel",
	} {
		if strings.Contains(output, failure) {
			return "panic:" + failure
		}
	}
	if runError == nil && strings.Contains(output, "return") {
		return "return"
	}
	return "unclassified"
}

func TestWaveNineRequiresExplicitCooperativeProfile(t *testing.T) {
	directory := t.TempDir()
	writeProgramFile(
		t,
		filepath.Join(directory, "go.mod"),
		"module example.com/concurrencyprofile\n\ngo 1.26.4\n",
	)
	writeProgramFile(t, filepath.Join(directory, "source.go"), `package concurrencyprofile

func Run() int32 {
	values := make(chan int32, 1)
	values <- 1
	return <-values
}
`)
	program, err := load.Load(context.Background(), load.Request{
		Directory: directory,
		Pattern:   ".",
	})
	if err != nil {
		t.Fatal(err)
	}
	root, err := emit.NewRoot(
		program.Roots()[0].Types().Scope().Lookup("Run"),
	)
	if err != nil {
		t.Fatal(err)
	}
	_, err = emit.Compile(program, []emit.Root{root})
	var unsupported *api.UnsupportedError
	if !errors.As(err, &unsupported) ||
		unsupported.Category != api.CategoryExpression ||
		unsupported.Construct != "*ast.CallExpr" ||
		unsupported.Role != api.RoleLocalValue {
		t.Fatalf(
			"default concurrency error = %#v, want make-expression UnsupportedError",
			err,
		)
	}
	if _, err := emit.CompileWithOptions(
		program,
		[]emit.Root{root},
		waveNineOptions(),
	); err != nil {
		t.Fatalf("explicit cooperative profile: %v", err)
	}
}

func waveNineOptions() emit.Options {
	options := emit.DefaultOptions()
	options.ConcurrencySemantics = emit.ConcurrencySemanticsCooperative
	return options
}

func TestConcurrencySemanticsSelectionIsClosed(t *testing.T) {
	if emit.DefaultOptions().ConcurrencySemantics !=
		emit.ConcurrencySemanticsDisabled {
		t.Fatal("default options silently select cooperative concurrency")
	}
	for source, want := range map[string]emit.ConcurrencySemantics{
		"disabled":    emit.ConcurrencySemanticsDisabled,
		"cooperative": emit.ConcurrencySemanticsCooperative,
	} {
		got, err := emit.ParseConcurrencySemantics(source)
		if err != nil || got != want {
			t.Fatalf("parse %q = %s, %v; want %s", source, got, err, want)
		}
	}
	got, err := emit.ParseConcurrencySemantics("preemptive")
	if err == nil || got != emit.ConcurrencySemanticsInvalid {
		t.Fatalf("parse preemptive = %s, %v; want typed rejection", got, err)
	}
	options := emit.DefaultOptions()
	options.ConcurrencySemantics = emit.ConcurrencySemanticsInvalid
	_, err = emit.CompileWithOptions(nil, nil, options)
	var optionsError *emit.OptionsError
	if !errors.As(err, &optionsError) ||
		optionsError.Field != "concurrency semantics" {
		t.Fatalf("invalid concurrency option error = %#v", err)
	}
}

func TestWaveNinePreemptionBoundaryAddsNoYieldHeuristic(t *testing.T) {
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
	if !strings.Contains(target, "GoScheduler.spawn") ||
		!strings.Contains(target, "for (;;)") {
		t.Fatalf("preemption boundary shape changed:\n%s", target)
	}
	for _, forbidden := range []string{
		"setTimeout",
		"queueMicrotask",
		"setImmediate",
		"yield",
	} {
		if strings.Contains(target, forbidden) {
			t.Fatalf(
				"preemption boundary acquired %q heuristic:\n%s",
				forbidden,
				target,
			)
		}
	}
}

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
