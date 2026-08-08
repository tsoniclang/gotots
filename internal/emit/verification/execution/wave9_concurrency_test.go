package emit_test

import (
	"context"
	"path/filepath"
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
	assertWaveNineGenericArtifactBudget(t, artifacts.sizes)
	if artifacts.bytes > 146_000 || artifacts.largest > 32_000 {
		t.Fatalf(
			"Wave 9 artifact bounds exceeded: total=%d largest=%d",
			artifacts.bytes,
			artifacts.largest,
		)
	}
	if artifacts.nodes > 25_600 {
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
	goOutput := executeWaveNineGo(t, workingDirectory)
	requireNativeGoEvidence(t, goOutput)
}

func assertWaveNineGenericArtifactBudget(
	t *testing.T,
	artifacts []artifactSize,
) {
	t.Helper()
	concretizations := 0
	concretizationBytes := 0
	capabilities := 0
	capabilityBytes := 0
	for _, artifact := range artifacts {
		switch {
		case strings.HasPrefix(
			artifact.path,
			"support/generics/concretizations/",
		):
			concretizations++
			concretizationBytes += artifact.bytes
		case strings.HasPrefix(
			artifact.path,
			"support/generics/capabilities/",
		):
			capabilities++
			capabilityBytes += artifact.bytes
		}
	}
	// Canonical pointer markers carry pointer intent directly, so the fixture
	// owns seven generic capabilities rather than the former two extra
	// generated pointer-representation capabilities.
	if concretizations != 7 || concretizationBytes > 6_200 ||
		capabilities != 7 || capabilityBytes > 5_000 {
		t.Fatalf(
			"Wave 9 generic artifact bounds exceeded: concretizations=%d/%d capabilities=%d/%d",
			concretizations,
			concretizationBytes,
			capabilities,
			capabilityBytes,
		)
	}
}

func TestWaveNineKeepsTransportedCallableABIByteStableAcrossRoots(t *testing.T) {
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
	for _, required := range []string{
		"export async function WhollySynchronous(): Promise<gostring>",
		"(($0: gostring) => Awaitable<gostring>) | undefined",
		"return await (__gotots_callee_",
	} {
		if !strings.Contains(synchronousFunction, required) {
			t.Fatalf(
				"transported callable contract lacks %q:\n%s",
				required,
				synchronousFunction,
			)
		}
	}
	if strings.Contains(synchronousFunction, "$cooperative_") {
		t.Fatalf("transported callable retained a profile variant:\n%s", synchronousFunction)
	}
}

func TestImmediateFunctionLiteralBypassesFirstClassCallableABI(t *testing.T) {
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
		program.Roots()[0].Types().Scope().
			Lookup("ImmediateLiteralABIIsolation"),
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
	packageAssemblies := artifacts.printedByKind[emit.TargetFilePackageAssembly]
	if len(packageAssemblies) != 1 {
		t.Fatalf("package assemblies = %d, want one", len(packageAssemblies))
	}
	packageAssembly := packageAssemblies[0]
	if strings.Contains(packageAssembly, "async function $initialize") ||
		strings.Contains(packageAssembly, "await (") {
		t.Fatalf(
			"immediate synchronous literal inherited its callable ABI:\n%s",
			packageAssembly,
		)
	}
	if !strings.Contains(
		artifacts.printed,
		"__gotots_defers_0.push(async ($go$recovery: GoRecovery): Promise<void> => {",
	) {
		t.Fatalf(
			"deferred direct literal lacks its recovery-owned defer envelope:\n%s",
			artifacts.printed,
		)
	}
	if strings.Contains(
		artifacts.printed,
		" = function ($go$recovery: GoRecovery): void {",
	) {
		t.Fatalf(
			"non-recovering direct literal acquired a recovery parameter:\n%s",
			artifacts.printed,
		)
	}
	if strings.Contains(
		artifacts.printed,
		"=> Promise<void>) | undefined = function",
	) {
		t.Fatalf(
			"deferred direct literal inherited its first-class ABI:\n%s",
			artifacts.printed,
		)
	}
	runner := filepath.Join(workingDirectory, "runner.ts")
	writeProgramFile(t, runner, `import "./program.js";
import { ImmediateLiteralABIIsolation } from "`+artifacts.sourceModule+`";
import { GoScheduler } from "./runtime/channel.js";

await GoScheduler.run(async () => {
    console.log(await ImmediateLiteralABIIsolation());
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
	if targetOutput != "literalimmediateproviderbodydeferred\n" {
		t.Fatalf("immediate-literal output = %q", targetOutput)
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
		"async (): Promise<void> =>",
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
		"GoArray.literal<(($0: GoReceiveChannel<int32> | undefined) => Awaitable<int32>) | undefined",
		"RuntimeSlice.literal<(($0: GoReceiveChannel<int32> | undefined) => Awaitable<int32>) | undefined",
		"mapping.lookup(0)",
		"let asserted: (($0: GoReceiveChannel<int32> | undefined) => Awaitable<int32>) | undefined",
		"identity$concrete_",
		"let methodValue: (() => Awaitable<int32>) | undefined",
		"await goInterfaceNonNil<Reader>(",
		"[receiveOne, synchronousOne]",
	} {
		if !strings.Contains(transport, required) {
			t.Fatalf("Transport lacks %q:\n%s", required, transport)
		}
	}
	if strings.Contains(
		transport,
		"async ($argument0: GoReceiveChannel<int32> | undefined): Promise<int32>",
	) {
		t.Fatalf("synchronous callable transport retained an async wrapper:\n%s", transport)
	}
	if concretizations := strings.Count(
		transport,
		"identity$concrete_",
	); concretizations != 2 {
		t.Fatalf(
			"Transport uses %d identity concretizations, want callable and slice",
			concretizations,
		)
	}
	if strings.Contains(transport, "identity<") {
		t.Fatalf("Transport bypasses exact generic concretization:\n%s", transport)
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
	deferred := waveNineFunctionText(t, printed, "deferredOnly")
	for _, required := range []string{
		"export async function deferredOnly(",
		"await __gotots_deferred_",
	} {
		if !strings.Contains(deferred, required) {
			t.Fatalf(
				"cooperative defer lacks %q:\n%s",
				required,
				deferred,
			)
		}
	}
	deferredRecover := waveNineFunctionText(
		t,
		printed,
		"cooperativeDeferredRecover",
	)
	for _, required := range []string{
		"export async function cooperativeDeferredRecover(",
		"Promise<void>",
		"await __gotots_deferred_",
	} {
		if !strings.Contains(deferredRecover, required) {
			t.Fatalf(
				"cooperative deferred recover lacks %q:\n%s",
				required,
				deferredRecover,
			)
		}
	}
	genericConstraint := waveNineFunctionText(
		t,
		printed,
		"pullConstraint",
	)
	for _, required := range []string{
		"export async function pullConstraint$kernel<",
		"return await $go$constraint_method_",
	} {
		if !strings.Contains(genericConstraint, required) {
			t.Fatalf(
				"cooperative constraint method lacks %q:\n%s",
				required,
				genericConstraint,
			)
		}
	}
	forwardLeaf := waveNineFunctionText(t, printed, "forwardLeaf")
	for _, required := range []string{
		"export async function forwardLeaf$kernel<",
		"return await $go$constraint_method_",
	} {
		if !strings.Contains(forwardLeaf, required) {
			t.Fatalf(
				"cooperative generic leaf lacks %q:\n%s",
				required,
				forwardLeaf,
			)
		}
	}
	forwardBridge := waveNineFunctionText(t, printed, "forwardBridge")
	for _, required := range []string{
		"export async function forwardBridge$kernel<",
		"return await forwardLeaf$kernel<T>(",
	} {
		if !strings.Contains(forwardBridge, required) {
			t.Fatalf(
				"cooperative generic forwarding lacks %q:\n%s",
				required,
				forwardBridge,
			)
		}
	}
	staticConstraint := waveNineFunctionText(t, printed, "readStatic")
	for _, required := range []string{
		"export async function readStatic$kernel<",
		"($0: T) => Awaitable<int64>",
		"return await $go$constraint_method_",
	} {
		if !strings.Contains(staticConstraint, required) {
			t.Fatalf(
				"transported generic constraint lacks %q:\n%s",
				required,
				staticConstraint,
			)
		}
	}
	if !strings.Contains(printed, "PackageReceiver") ||
		!strings.Contains(printed, "Awaitable<int32>") {
		t.Fatal("function-valued package storage lacks its cooperative ABI")
	}
}
