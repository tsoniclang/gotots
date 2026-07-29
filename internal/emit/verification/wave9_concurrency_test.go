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

func TestWaveNineConcurrencyCompilesThroughPublicPipeline(t *testing.T) {
	program, err := load.Load(context.Background(), load.Request{
		Directory: waveNineConcurrencyDirectory(),
		Pattern:   ".",
	})
	if err != nil {
		t.Fatal(err)
	}
	auditRoot, err := emit.NewRoot(
		program.Roots()[0].Types().Scope().Lookup("Audit"),
	)
	if err != nil {
		t.Fatal(err)
	}
	synchronousRoot, err := emit.NewRoot(
		program.Roots()[0].Types().Scope().Lookup("WhollySynchronous"),
	)
	if err != nil {
		t.Fatal(err)
	}
	emission, err := emit.CompileWithOptions(
		program,
		[]emit.Root{auditRoot, synchronousRoot},
		waveNineOptions(),
	)
	if err != nil {
		t.Fatal(err)
	}
	workingDirectory := t.TempDir()
	artifacts := materializeArtifacts(t, emission, workingDirectory)
	if artifacts.bytes > 120_000 || artifacts.largest > 32_000 {
		t.Fatalf(
			"Wave 9 artifact bounds exceeded: total=%d largest=%d",
			artifacts.bytes,
			artifacts.largest,
		)
	}
	if artifacts.nodes > 24_000 {
		t.Fatalf(
			"Wave 9 artifact AST bound exceeded: nodes=%d",
			artifacts.nodes,
		)
	}
	assertWaveNineArtifactShape(t, artifacts.printed)
	runner := filepath.Join(workingDirectory, "runner.ts")
	writeProgramFile(t, runner, `import "./program.js";
import { Audit } from "`+artifacts.sourceModule+`";
import { GoScheduler } from "./runtime/channel.js";

await GoScheduler.run(async () => {
    const values = await Audit();
    console.log(values.join(" "));
});
`)
	writeProgramFile(
		t,
		filepath.Join(workingDirectory, "package.json"),
		"{\"type\":\"module\"}\n",
	)
	paths := append(artifacts.paths, runner)
	waveThreeTypecheck(t, workingDirectory, paths)
	targetOutput := runProgram(
		t,
		workingDirectory,
		"node",
		filepath.Join(workingDirectory, "out", "runner.js"),
	)
	goOutput := executeWaveNineGo(t, workingDirectory)
	if targetOutput != goOutput {
		t.Fatalf(
			"Wave 9 output differs\nTypeScript:\n%s\nGo:\n%s",
			targetOutput,
			goOutput,
		)
	}
	t.Logf(
		"Wave 9 matrix: files=%d bytes=%d nodes=%d largest=%d",
		len(artifacts.paths),
		artifacts.bytes,
		artifacts.nodes,
		artifacts.largest,
	)
	limit := min(20, len(artifacts.sizes))
	for index, artifact := range artifacts.sizes[:limit] {
		t.Logf(
			"Wave 9 artifact %02d: %s bytes=%d nodes=%d",
			index+1,
			artifact.path,
			artifact.bytes,
			artifact.nodes,
		)
	}
}

func TestWaveNineLeavesUnrelatedCallableABIByteStable(t *testing.T) {
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
	withConcurrency, err := emit.CompileWithOptions(
		program,
		[]emit.Root{auditRoot, synchronousRoot},
		waveNineOptions(),
	)
	if err != nil {
		t.Fatal(err)
	}
	synchronousArtifacts := materializeArtifacts(
		t,
		synchronousOnly,
		t.TempDir(),
	)
	concurrentArtifacts := materializeArtifacts(
		t,
		withConcurrency,
		t.TempDir(),
	)
	synchronousFunction := waveNineFunctionText(
		t,
		synchronousArtifacts.printed,
		"WhollySynchronous",
	)
	concurrentFunction := waveNineFunctionText(
		t,
		concurrentArtifacts.printed,
		"WhollySynchronous",
	)
	synchronousFunction = strings.TrimRight(synchronousFunction, "\n")
	concurrentFunction = strings.TrimRight(concurrentFunction, "\n")
	if synchronousFunction != concurrentFunction {
		t.Fatalf(
			"unrelated cooperative contracts changed synchronous bytes\nonly:\n%s\nwith concurrency:\n%s",
			synchronousFunction,
			concurrentFunction,
		)
	}
	for _, forbidden := range []string{"async", "Promise<", "GoScheduler"} {
		if strings.Contains(synchronousFunction, forbidden) {
			t.Fatalf(
				"synchronous callable contract contains %q:\n%s",
				forbidden,
				synchronousFunction,
			)
		}
	}
}

func assertWaveNineArtifactShape(t *testing.T, printed string) {
	t.Helper()
	for _, required := range []string{
		"export class GoChannel<T>",
		"export class GoScheduler",
		"await GoScheduler.block",
		"GoScheduler.spawn",
		"GoScheduler.block(goSelect",
	} {
		if !strings.Contains(printed, required) {
			t.Fatalf("Wave 9 artifacts lack %q:\n%s", required, printed)
		}
	}
	for _, forbidden := range []string{
		"CallableFacetVariable",
		"CallableFacetStructField",
		"CallableFacetCallResult",
		": any",
		": unknown",
		" as any",
		" as unknown",
		".call(",
		".apply(",
		".bind(",
	} {
		if strings.Contains(printed, forbidden) {
			t.Fatalf("Wave 9 artifacts contain %q:\n%s", forbidden, printed)
		}
	}
	unbuffered := waveNineFunctionText(t, printed, "Unbuffered")
	for _, required := range []string{
		"export async function Unbuffered(): Promise<int32>",
		"async function (): Promise<void>",
		"GoScheduler.spawn(async (): Promise<void>",
		"await __gotots_callee_",
	} {
		if !strings.Contains(unbuffered, required) {
			t.Fatalf("Unbuffered lacks %q:\n%s", required, unbuffered)
		}
	}
	transport := waveNineFunctionText(t, printed, "Transport")
	for _, required := range []string{
		"export async function Transport(): Promise<int32>",
		"GoArray.literal<(($0: GoReceiveChannel<int32> | undefined, $go$recovery?: GoRecovery) => Promise<int32>) | undefined",
		"RuntimeSlice.literal<(($0: GoReceiveChannel<int32> | undefined, $go$recovery?: GoRecovery) => Promise<int32>) | undefined",
		"mapping.lookup(0)",
		"let asserted: (($0: GoReceiveChannel<int32> | undefined, $go$recovery?: GoRecovery) => Promise<int32>) | undefined",
		"identity<(($0: GoReceiveChannel<int32> | undefined, $go$recovery?: GoRecovery) => Promise<int32>) | undefined>",
		"let methodValue: (($go$recovery?: GoRecovery) => Promise<int32>) | undefined",
		"await goInterfaceNonNil<Reader>(",
		"async ($argument0: GoReceiveChannel<int32> | undefined): Promise<int32>",
	} {
		if !strings.Contains(transport, required) {
			t.Fatalf("Transport lacks %q:\n%s", required, transport)
		}
	}
	synchronous := waveNineFunctionText(t, printed, "synchronousOne")
	if strings.Contains(synchronous, "async") ||
		strings.Contains(synchronous, "Promise<") {
		t.Fatalf("synchronous provider acquired async tax:\n%s", synchronous)
	}
	directSynchronous := waveNineFunctionText(
		t,
		printed,
		"DirectSynchronous",
	)
	for _, forbidden := range []string{"async", "Promise<", "await "} {
		if strings.Contains(directSynchronous, forbidden) {
			t.Fatalf(
				"direct synchronous call acquired %q:\n%s",
				forbidden,
				directSynchronous,
			)
		}
	}
	selectDefault := waveNineFunctionText(t, printed, "SelectDefault")
	if !strings.Contains(
		selectDefault,
		"export function SelectDefault(): int32",
	) || !strings.Contains(selectDefault, "goSelectReady(") {
		t.Fatalf(
			"default-bearing select lacks its synchronous ready path:\n%s",
			selectDefault,
		)
	}
	for _, forbidden := range []string{
		"async",
		"Promise<",
		"await ",
		"GoScheduler",
		"goSelect(",
	} {
		if strings.Contains(selectDefault, forbidden) {
			t.Fatalf(
				"default-bearing select acquired %q:\n%s",
				forbidden,
				selectDefault,
			)
		}
	}
	channelCopy := waveNineFunctionText(
		t,
		printed,
		"ChannelIdentityAndCopy",
	)
	for _, required := range []string{
		"return Payload.$copy(value);",
		"mapping.lookup(projected)",
		"projected === undefined",
	} {
		if !strings.Contains(channelCopy, required) {
			t.Fatalf(
				"channel identity/copy lacks %q:\n%s",
				required,
				channelCopy,
			)
		}
	}
	goroutineEvaluation := waveNineFunctionText(
		t,
		printed,
		"GoroutineEvaluation",
	)
	for _, required := range []string{
		"const __gotots_callee_",
		"const __gotots_argument_",
		"GoScheduler.spawn(async (): Promise<void>",
	} {
		if !strings.Contains(goroutineEvaluation, required) {
			t.Fatalf(
				"goroutine evaluation lacks %q:\n%s",
				required,
				goroutineEvaluation,
			)
		}
	}
	goroutineForms := waveNineFunctionText(t, printed, "GoroutineForms")
	if spawns := strings.Count(
		goroutineForms,
		"GoScheduler.spawn",
	); spawns != 4 {
		t.Fatalf("goroutine forms emit %d spawn sites, want four", spawns)
	}
	for _, function := range []string{
		"DiscardedReceive",
		"GenericChannel",
		"SelectEvaluation",
		"SelectControl",
	} {
		target := waveNineFunctionText(t, printed, function)
		if !strings.Contains(target, "await GoScheduler.block") {
			t.Fatalf("%s did not propagate cooperative execution:\n%s", function, target)
		}
	}
	valueRecursive := waveNineFunctionText(t, printed, "ValueRecursive")
	if !strings.Contains(valueRecursive, "return await valueCycleA(") {
		t.Fatalf(
			"recursive callable propagation lacks direct await:\n%s",
			valueRecursive,
		)
	}
	if !strings.Contains(printed, "PackageReceiver") ||
		!strings.Contains(printed, "Promise<int32>") {
		t.Fatal("function-valued package storage lacks its cooperative ABI")
	}
}

func waveNineFunctionText(t *testing.T, printed, name string) string {
	t.Helper()
	start := strings.Index(printed, "export function "+name+"(")
	if start < 0 {
		start = strings.Index(printed, "export async function "+name+"(")
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

func executeWaveNineGo(t *testing.T, workingDirectory string) string {
	t.Helper()
	modulePath, err := filepath.Abs(waveNineConcurrencyDirectory())
	if err != nil {
		t.Fatal(err)
	}
	runnerDirectory := filepath.Join(workingDirectory, "go-runner-wave9")
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
	fmt.Println(values.Audit())
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

func waveNineConcurrencyDirectory() string {
	return filepath.Join(
		repositoryRoot(),
		"testdata",
		"constructs",
		"concurrency",
		"wave9",
	)
}
