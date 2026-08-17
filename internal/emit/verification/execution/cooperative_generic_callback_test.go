package emit_test

import (
	"fmt"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestGenericCallableTransportUsesOneAwaitableABI(t *testing.T) {
	directory, workingDirectory, emission, artifacts :=
		genericCallbackFixture(t)

	apply := waveNineFunctionText(t, artifacts.printed, "Apply")
	for _, required := range []string{
		"export async function Apply$kernel<T>(",
		"predicate: (($0: T) => Awaitable<bool>) | undefined",
		"return await (__gotots_callee_",
	} {
		if !strings.Contains(apply, required) {
			t.Fatalf("canonical generic callable lacks %q:\n%s", required, apply)
		}
	}
	if count := strings.Count(
		artifacts.printed,
		"export async function Apply$kernel<T>(",
	); count != 1 {
		t.Fatalf("generic Apply kernel count = %d, want one", count)
	}
	for _, forbidden := range []string{
		"$cooperative_",
		"$goCapability_",
		"support/generics/capabilities/",
		"instanceof Promise",
		".then ===",
		"$Value =",
	} {
		if strings.Contains(artifacts.printed, forbidden) {
			t.Fatalf("generic callable output retained forbidden %q", forbidden)
		}
	}
	for _, function := range []string{
		"SynchronousApply",
		"CooperativeApply",
		"SynchronousFunctionValue",
		"CooperativeFunctionValue",
		"SynchronousMethodValue",
		"CooperativeMethodValue",
		"IndependentSynchronous",
		"SynchronousSequence",
		"CooperativeSequence",
		"SynchronousGenericInterfaceMethod",
		"CooperativeGenericInterfaceMethod",
	} {
		target := waveNineFunctionText(t, artifacts.printed, function)
		if !strings.Contains(target, "async") ||
			!strings.Contains(target, "await ") {
			t.Fatalf(
				"transported callable did not propagate through %s:\n%s",
				function,
				target,
			)
		}
	}
	direct := waveNineFunctionText(t, artifacts.printed, "IsSeven")
	for _, forbidden := range []string{"async", "await ", "Promise<", "Awaitable<"} {
		if strings.Contains(direct, forbidden) {
			t.Fatalf("direct synchronous callable contains %q:\n%s", forbidden, direct)
		}
	}
	named := waveNineFunctionText(
		t,
		artifacts.printed,
		"NamedSynchronousApply",
	)
	if !strings.Contains(named, "InvokeIntPredicate(IsSeven)") ||
		strings.Contains(named, "=> IsSeven(") {
		t.Fatalf(
			"synchronous provider was not transported directly through Awaitable ABI:\n%s",
			named,
		)
	}
	filter := waveNineFunctionText(t, artifacts.printed, "FilterSequence")
	if strings.Contains(filter, "export async function") ||
		!strings.Contains(filter, "new Sequence(async (") ||
		!strings.Contains(filter, "Awaitable<") {
		t.Fatalf(
			"returned callable did not isolate its canonical ABI from its provider:\n%s",
			filter,
		)
	}
	clone := waveNineFunctionText(
		t,
		artifacts.printed,
		"CloneSynchronousStoredCallback",
	)
	if !strings.Contains(clone, "async") || !strings.Contains(clone, "await ") {
		t.Fatalf("stored callable invocation did not propagate Awaitable:\n%s", clone)
	}
	if packageAssemblyExports(emission.Files(), "genericcallback", "Apply$kernel") {
		t.Fatal("package assembly publishes private generic callable kernel")
	}

	runner := filepath.Join(workingDirectory, "runner.ts")
	writeProgramFile(t, runner, `import "./program.js";
import {
    CooperativeApply,
    CooperativeLexicalResult,
    SynchronousApply,
    CooperativeFunctionValue,
    SynchronousFunctionValue,
    CooperativeNested,
    CooperativeMethod,
    CooperativeMethodValue,
    SynchronousMethodValue,
    CooperativeMethodExpression,
    CooperativeResult,
    CooperativeSequence,
    CooperativeBoolSequence,
    InitializedSynchronousCallback,
    InitializedCooperativeCallback,
    IndependentPackageInitializer,
    IndependentSynchronous,
    SynchronousSequence,
    CooperativeGenericProfileWithNamedCallback,
    CooperativeNestedGenericMethod,
    CooperativeRecursiveGenericMethod,
    CooperativeGenericInterfaceMethod,
    SynchronousGenericInterfaceMethod,
    CooperativeStoredCallback,
    SynchronousStoredCallback,
    CloneSynchronousStoredCallback,
} from "`+sourceModuleForExport(
		t,
		artifacts,
		workingDirectory,
		"CooperativeApply",
	)+`";
import { GoScheduler } from "./runtime/channel.js";

await GoScheduler.run(async () => {
    console.log([
        await CooperativeApply(),
        await CooperativeLexicalResult(),
        await SynchronousApply(),
        await CooperativeFunctionValue(),
        await SynchronousFunctionValue(),
        await CooperativeNested(),
        await CooperativeMethod(),
        await CooperativeMethodValue(),
        await SynchronousMethodValue(),
        await CooperativeMethodExpression(),
        await CooperativeResult(),
        await CooperativeSequence(),
        await CooperativeBoolSequence(),
        await InitializedSynchronousCallback(),
        await InitializedCooperativeCallback(),
        await IndependentPackageInitializer(),
        await IndependentSynchronous(),
        await SynchronousSequence(),
        await CooperativeGenericProfileWithNamedCallback(),
        await CooperativeNestedGenericMethod(),
        await CooperativeRecursiveGenericMethod(),
        await CooperativeGenericInterfaceMethod(),
        await SynchronousGenericInterfaceMethod(),
        await CooperativeStoredCallback(),
        await SynchronousStoredCallback(),
        await CloneSynchronousStoredCallback(),
    ].map(String).join(" "));
});
`)
	writeProgramFile(
		t,
		filepath.Join(workingDirectory, "package.json"),
		"{\"type\":\"module\"}\n",
	)
	waveThreeTypecheck(
		t,
		workingDirectory,
		append(artifacts.paths, runner),
	)
	goRunner := filepath.Join(workingDirectory, "go-runner")
	writeProgramFile(t, filepath.Join(goRunner, "go.mod"), fmt.Sprintf(
		`module example.com/runner

go 1.26.4

require example.com/genericcallback v0.0.0

replace example.com/genericcallback => %s
`,
		filepath.ToSlash(directory),
	))
	writeProgramFile(t, filepath.Join(goRunner, "main.go"), `package main

import (
	"fmt"

	values "example.com/genericcallback"
)

func main() {
	fmt.Println(
		values.CooperativeApply(),
		values.CooperativeLexicalResult(),
		values.SynchronousApply(),
		values.CooperativeFunctionValue(),
		values.SynchronousFunctionValue(),
		values.CooperativeNested(),
		values.CooperativeMethod(),
		values.CooperativeMethodValue(),
		values.SynchronousMethodValue(),
		values.CooperativeMethodExpression(),
		values.CooperativeResult(),
		values.CooperativeSequence(),
		values.CooperativeBoolSequence(),
		values.InitializedSynchronousCallback(),
		values.InitializedCooperativeCallback(),
		values.IndependentPackageInitializer(),
		values.IndependentSynchronous(),
		values.SynchronousSequence(),
		values.CooperativeGenericProfileWithNamedCallback(),
		values.CooperativeNestedGenericMethod(),
		values.CooperativeRecursiveGenericMethod(),
		values.CooperativeGenericInterfaceMethod(),
		values.SynchronousGenericInterfaceMethod(),
		values.CooperativeStoredCallback(),
		values.SynchronousStoredCallback(),
		values.CloneSynchronousStoredCallback(),
	)
}
`)
	goOutput := runProgram(
		t,
		goRunner,
		filepath.Join(runtime.GOROOT(), "bin", "go"),
		"run",
		".",
	)
	requireNativeGoEvidence(t, goOutput)
}
