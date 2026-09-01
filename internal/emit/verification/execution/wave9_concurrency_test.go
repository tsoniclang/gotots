package emit_test

import (
	"context"
	"fmt"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/tsoniclang/gotots/internal/emit"
	"github.com/tsoniclang/gotots/internal/load"
)

func TestWaveNineSerialExecutionCompilesWithoutAsyncArtifacts(t *testing.T) {
	program, err := load.Load(context.Background(), load.Request{
		Directory: waveNineConcurrencyDirectory(),
		Pattern:   ".",
	})
	if err != nil {
		t.Fatal(err)
	}
	scope := program.Roots()[0].Types().Scope()
	var roots []emit.Root
	for _, name := range []string{
		"Audit",
		"WhollySynchronous",
		"Buffered",
		"CloseDrain",
		"DirectionAndMeasure",
		"ChannelRange",
		"SelectDefault",
		"SelectReceive",
		"SelectSend",
		"SelectDelayedTarget",
		"Transport",
		"AggregateClosures",
		"Recursive",
		"ValueRecursive",
		"DirectSynchronous",
		"DiscardedReceive",
		"ChannelIdentityAndCopy",
		"GenericChannel",
		"GoroutineEvaluation",
		"GoroutineForms",
		"SelectEvaluation",
		"SelectControl",
		"DeferCooperative",
		"GenericConstraintChannel",
		"GenericConstraintForward",
		"GenericConstraintSynchronous",
		"DeferRecoverCooperative",
		"GenericInterfaceAudit",
	} {
		root, rootError := emit.NewRoot(scope.Lookup(name))
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
	for _, forbidden := range []string{
		"async ",
		"await ",
		"Promise<",
		"Awaitable<",
		"GoScheduler",
		"$cooperative_",
	} {
		if strings.Contains(artifacts.printed, forbidden) {
			t.Fatalf("serial artifacts contain %q", forbidden)
		}
	}
	for _, required := range []string{
		"export class GoChannel<T>",
		"serial channel send would block",
		"serial channel receive would block",
		"serial select would block",
		"export function WhollySynchronous(): gostring",
		"export function Unbuffered(): int32",
	} {
		if !strings.Contains(artifacts.printed, required) {
			t.Fatalf("serial artifacts lack %q", required)
		}
	}
	if artifacts.executableBytes > 133_000 || artifacts.executableLargest > 30_000 {
		t.Fatalf(
			"serial executable artifact bounds exceeded: total=%d largest=%d canonical=%d/%d",
			artifacts.executableBytes,
			artifacts.executableLargest,
			artifacts.bytes,
			artifacts.largest,
		)
	}
	waveThreeTypecheck(t, workingDirectory, artifacts.paths)
	targetOutput := executeSerialWaveNineTypeScript(
		t,
		workingDirectory,
		artifacts,
	)
	goOutput := executeSerialWaveNineGo(t, workingDirectory)
	if targetOutput != goOutput {
		t.Fatalf("serial execution mismatch:\nGo: %s\nTypeScript: %s", goOutput, targetOutput)
	}
}

func TestWaveNineSynchronousCallableIsRootStable(t *testing.T) {
	program, err := load.Load(context.Background(), load.Request{
		Directory: waveNineConcurrencyDirectory(),
		Pattern:   ".",
	})
	if err != nil {
		t.Fatal(err)
	}
	scope := program.Roots()[0].Types().Scope()
	synchronousRoot, err := emit.NewRoot(scope.Lookup("WhollySynchronous"))
	if err != nil {
		t.Fatal(err)
	}
	synchronousOnly, err := emit.CompileWithOptions(
		program,
		[]emit.Root{synchronousRoot},
		waveNineOptions(),
	)
	if err != nil {
		t.Fatal(err)
	}
	auditRoot, err := emit.NewRoot(scope.Lookup("Audit"))
	if err != nil {
		t.Fatal(err)
	}
	withChannels, err := emit.CompileWithOptions(
		program,
		[]emit.Root{auditRoot, synchronousRoot},
		waveNineOptions(),
	)
	if err != nil {
		t.Fatal(err)
	}
	onlyText := strings.TrimSpace(waveNineFunctionText(
		t,
		materializeArtifacts(t, synchronousOnly, t.TempDir()).printed,
		"WhollySynchronous",
	))
	withChannelsText := strings.TrimSpace(waveNineFunctionText(
		t,
		materializeArtifacts(t, withChannels, t.TempDir()).printed,
		"WhollySynchronous",
	))
	if onlyText != withChannelsText {
		t.Fatalf(
			"unrelated channel roots changed synchronous bytes\nonly:\n%s\nwith channels:\n%s",
			onlyText,
			withChannelsText,
		)
	}
	for _, forbidden := range []string{"async", "await", "Promise", "Awaitable"} {
		if strings.Contains(onlyText, forbidden) {
			t.Fatalf("synchronous callable contains %q:\n%s", forbidden, onlyText)
		}
	}
}

func TestImmediateFunctionLiteralAndDeferRemainSynchronous(t *testing.T) {
	program, err := load.Load(context.Background(), load.Request{
		Directory: filepath.Join(
			repositoryRoot(),
			"testdata",
			"constructs",
			"concurrency",
			"immediate-literal",
		),
		Pattern: ".",
	})
	if err != nil {
		t.Fatal(err)
	}
	root, err := emit.NewRoot(
		program.Roots()[0].Types().Scope().Lookup("ImmediateLiteralABIIsolation"),
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
	workingDirectory := t.TempDir()
	artifacts := materializeArtifacts(t, emission, workingDirectory)
	for _, forbidden := range []string{"async ", "await ", "Promise<", "Awaitable<"} {
		if strings.Contains(artifacts.printed, forbidden) {
			t.Fatalf("immediate literal artifacts contain %q", forbidden)
		}
	}
	for _, required := range []string{
		"export function ImmediateLiteralABIIsolation(): gostring",
		"__gotots_deferred_0",
		"($go$recovery: GoRecovery): void =>",
	} {
		if !strings.Contains(artifacts.printed, required) {
			t.Fatalf("immediate literal artifacts lack %q", required)
		}
	}
	runner := filepath.Join(workingDirectory, "runner.ts")
	writeProgramFile(t, runner, `import "./program.js";
import { ImmediateLiteralABIIsolation } from "`+artifacts.sourceModule+`";

console.log(ImmediateLiteralABIIsolation());
`)
	writeProgramFile(
		t,
		filepath.Join(workingDirectory, "package.json"),
		"{\"type\":\"module\"}\n",
	)
	paths := append(artifacts.paths, runner)
	waveThreeTypecheck(t, workingDirectory, paths)
	if output := runProgram(
		t,
		workingDirectory,
		"node",
		filepath.Join(workingDirectory, "out", "runner.js"),
	); output != "literalimmediateproviderbodydeferred\n" {
		t.Fatalf("immediate-literal output = %q", output)
	}
}

func waveNineFunctionText(t *testing.T, printed, name string) string {
	t.Helper()
	start := strings.Index(printed, "export function "+name)
	if start < 0 {
		t.Fatalf("generated output lacks function %s", name)
	}
	rest := printed[start:]
	end := strings.Index(rest[len("export "):], "\nexport ")
	factEnd := strings.Index(rest, "\nattribute<")
	artifactEnd := strings.Index(rest, "\n\n// ")
	if end >= 0 {
		end += len("export ")
	}
	if artifactEnd >= 0 && (end < 0 || artifactEnd < end) {
		end = artifactEnd
	}
	if factEnd >= 0 && (end < 0 || factEnd < end) {
		end = factEnd
	}
	if end >= 0 {
		return rest[:end]
	}
	return rest
}

func executeSerialWaveNineTypeScript(
	t *testing.T,
	workingDirectory string,
	artifacts waveFourArtifacts,
) string {
	t.Helper()
	packageModule := artifacts.packageModules["wave9concurrency"]
	if packageModule == "" {
		t.Fatal("wave 9 package assembly module is absent")
	}
	runner := filepath.Join(workingDirectory, "serial-runner.ts")
	writeProgramFile(t, runner, `import "./program.js";
import * as values from "`+packageModule+`";

console.log([
    values.WhollySynchronous(),
    values.Buffered(),
    values.CloseDrain(),
    values.DirectionAndMeasure(),
    values.ChannelRange(),
    values.SelectDefault(),
    values.SelectReceive(),
    values.SelectSend(),
    values.AggregateClosures(),
    values.Recursive(),
    values.ValueRecursive(),
    values.DirectSynchronous(),
    values.DiscardedReceive(),
    values.ChannelIdentityAndCopy(),
    values.GenericChannel(),
    values.SelectControl(),
].join(" "));
`)
	writeProgramFile(
		t,
		filepath.Join(workingDirectory, "package.json"),
		"{\"type\":\"module\"}\n",
	)
	paths := append(artifacts.paths, runner)
	waveThreeTypecheck(t, workingDirectory, paths)
	return runProgram(
		t,
		workingDirectory,
		"node",
		filepath.Join(workingDirectory, "out", "serial-runner.js"),
	)
}

func executeSerialWaveNineGo(t *testing.T, workingDirectory string) string {
	t.Helper()
	modulePath, err := filepath.Abs(waveNineConcurrencyDirectory())
	if err != nil {
		t.Fatal(err)
	}
	runnerDirectory := filepath.Join(workingDirectory, "go-serial-runner")
	writeProgramFile(t, filepath.Join(runnerDirectory, "go.mod"), fmt.Sprintf(`module example.com/serial-runner

go 1.26.4

require example.com/wave9concurrency v0.0.0

replace example.com/wave9concurrency => %s
`, modulePath))
	writeProgramFile(t, filepath.Join(runnerDirectory, "main.go"), `package main

import (
    "fmt"

    values "example.com/wave9concurrency"
)

func main() {
    fmt.Println(
        values.WhollySynchronous(),
        values.Buffered(),
        values.CloseDrain(),
        values.DirectionAndMeasure(),
        values.ChannelRange(),
        values.SelectDefault(),
        values.SelectReceive(),
        values.SelectSend(),
        values.AggregateClosures(),
        values.Recursive(),
        values.ValueRecursive(),
        values.DirectSynchronous(),
        values.DiscardedReceive(),
        values.ChannelIdentityAndCopy(),
        values.GenericChannel(),
        values.SelectControl(),
    )
}
`)
	return runProgram(
		t,
		runnerDirectory,
		filepath.Join(runtime.GOROOT(), "bin", "go"),
		"run",
		".",
	)
}
