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
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func TestGenericCallableProfilesDoNotWidenOtherInstantiations(
	t *testing.T,
) {
	directory := filepath.Join(
		repositoryRoot(),
		"testdata",
		"constructs",
		"concurrency",
		"generic-callback",
	)
	directory, err := filepath.Abs(directory)
	if err != nil {
		t.Fatal(err)
	}
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
	workingDirectory := t.TempDir()
	artifacts := materializeArtifacts(t, emission, workingDirectory)

	apply := waveNineFunctionText(t, artifacts.printed, "Apply")
	for _, required := range []string{
		"export function Apply<T>(",
		"=> bool",
		"return __gotots_callee_",
	} {
		if !strings.Contains(apply, required) {
			t.Fatalf(
				"synchronous generic callback provider lacks %q:\n%s",
				required,
				apply,
			)
		}
	}
	for _, forbidden := range []string{"async", "Promise<", "await "} {
		if strings.Contains(apply, forbidden) {
			t.Fatalf(
				"synchronous generic callback provider contains %q:\n%s",
				forbidden,
				apply,
			)
		}
	}
	cooperativeApply := waveNineFunctionWithPrefix(
		t,
		artifacts.printed,
		"Apply$cooperative_",
	)
	for _, required := range []string{
		"export async function Apply$cooperative_",
		"=> Promise<bool>",
		"return await __gotots_callee_",
	} {
		if !strings.Contains(cooperativeApply, required) {
			t.Fatalf(
				"cooperative generic callback variant lacks %q:\n%s",
				required,
				cooperativeApply,
			)
		}
	}
	base := waveNineFunctionText(t, artifacts.printed, "ApplyFirst")
	for _, forbidden := range []string{"async", "Promise<", "await "} {
		if strings.Contains(base, forbidden) {
			t.Fatalf(
				"synchronous generic provider ApplyFirst contains %q:\n%s",
				forbidden,
				base,
			)
		}
	}
	variant := waveNineFunctionWithPrefix(
		t,
		artifacts.printed,
		"ApplyFirst$cooperative_",
	)
	if !strings.Contains(variant, "async") ||
		!strings.Contains(variant, "await ") {
		t.Fatalf(
			"cooperative generic provider variant ApplyFirst is incomplete:\n%s",
			variant,
		)
	}
	base = waveNineClassMemberText(
		t,
		artifacts.printed,
		"Box",
		"\n    Apply(",
	)
	for _, forbidden := range []string{"async", "Promise<", "await "} {
		if strings.Contains(base, forbidden) {
			t.Fatalf(
				"synchronous generic provider Box.Apply contains %q:\n%s",
				forbidden,
				base,
			)
		}
	}
	variant = waveNineClassMemberText(
		t,
		artifacts.printed,
		"Box",
		"\n    async Apply$cooperative_",
	)
	if !strings.Contains(variant, "await ") {
		t.Fatalf(
			"cooperative generic provider variant Box.Apply is incomplete:\n%s",
			variant,
		)
	}
	if strings.Contains(artifacts.printed, "Box_Apply") {
		t.Fatal("generic receiver method retained a top-level receiver twin")
	}
	synchronous := waveNineFunctionText(
		t,
		artifacts.printed,
		"SynchronousApply",
	)
	for _, required := range []string{
		"export function SynchronousApply(",
		"function (value: gostring): bool {",
		"return Apply<gostring>(",
	} {
		if !strings.Contains(synchronous, required) {
			t.Fatalf(
				"synchronous generic instantiation lacks %q:\n%s",
				required,
				synchronous,
			)
		}
	}
	for _, forbidden := range []string{"async", "Promise<", "await "} {
		if strings.Contains(synchronous, forbidden) {
			t.Fatalf(
				"synchronous generic instantiation contains %q:\n%s",
				forbidden,
				synchronous,
			)
		}
	}
	for _, function := range []string{
		"CooperativeApply",
		"CooperativeFunctionValue",
		"CooperativeNested",
		"CooperativeMethod",
		"CooperativeMethodValue",
		"CooperativeMethodExpression",
		"CooperativeResult",
		"CooperativeGenericProfileWithNamedCallback",
		"CooperativeNestedGenericMethod",
		"CooperativeRecursiveGenericMethod",
		"CooperativeGenericInterfaceMethod",
	} {
		target := waveNineFunctionText(t, artifacts.printed, function)
		if !strings.Contains(target, "async") ||
			!strings.Contains(target, "await ") {
			t.Fatalf(
				"generic callable correspondence is absent from %s:\n%s",
				function,
				target,
			)
		}
	}
	namedCallbackProfile := waveNineFunctionWithPrefix(
		t,
		artifacts.printed,
		"GenericProfileWithNamedCallback$cooperative_",
	)
	if !strings.Contains(namedCallbackProfile, "async ($argument0: int32)") {
		t.Fatalf(
			"generic profile did not adapt a synchronous named callback:\n%s",
			namedCallbackProfile,
		)
	}
	for _, function := range []string{
		"SynchronousFunctionValue",
		"SynchronousMethodValue",
	} {
		target := waveNineFunctionText(t, artifacts.printed, function)
		for _, forbidden := range []string{"async", "Promise<", "await "} {
			if strings.Contains(target, forbidden) {
				t.Fatalf(
					"synchronous generic value %s contains %q:\n%s",
					function,
					forbidden,
					target,
				)
			}
		}
	}
	makeReceiver := waveNineFunctionText(
		t,
		artifacts.printed,
		"MakeReceiver",
	)
	for _, required := range []string{
		"=> Promise<T>",
		"return async function",
	} {
		if !strings.Contains(makeReceiver, required) {
			t.Fatalf(
				"intrinsically cooperative result lacks %q:\n%s",
				required,
				makeReceiver,
			)
		}
	}
	for _, forbidden := range []string{
		"MakeReceiver$cooperative_",
		"instanceof Promise",
		".then ===",
	} {
		if strings.Contains(artifacts.printed, forbidden) {
			t.Fatalf(
				"generic callable output contains forbidden %q",
				forbidden,
			)
		}
	}
	filterSequence := waveNineFunctionText(
		t,
		artifacts.printed,
		"FilterSequence",
	)
	for _, forbidden := range []string{"async function", "Promise<", "await "} {
		if strings.Contains(filterSequence, forbidden) {
			t.Fatalf(
				"synchronous generic returned literal contains %q:\n%s",
				forbidden,
				filterSequence,
			)
		}
	}
	cooperativeFilterSequence := waveNineFunctionWithPrefix(
		t,
		artifacts.printed,
		"FilterSequence$cooperative_",
	)
	for _, required := range []string{"async function", "Promise<", "await "} {
		if !strings.Contains(cooperativeFilterSequence, required) {
			t.Fatalf(
				"cooperative generic returned literal lacks %q:\n%s",
				required,
				cooperativeFilterSequence,
			)
		}
	}
	for _, required := range []string{
		"export class Sequence<T, $Value = ",
		"constructor(public readonly $value: $Value)",
	} {
		if !strings.Contains(artifacts.printed, required) {
			t.Fatalf("named callable value lacks %q", required)
		}
	}
	if strings.Contains(artifacts.printed, "Sequence<T> & {") {
		t.Fatal("named callable value retained a result intersection repair")
	}
	for _, required := range []string{
		"export interface MutableValue<T>",
		"Change($argument0: (($0: T, $go$recovery?: GoRecovery) => Promise<void>)",
		"async Change($argument0: (($0: int32, $go$recovery?: GoRecovery) => Promise<void>)",
		"async Change($argument0: (($0: gostring, $go$recovery?: GoRecovery) => Promise<void>)",
	} {
		if !strings.Contains(artifacts.printed, required) {
			t.Fatalf(
				"generic interface callable correspondence lacks %q:\n%s",
				required,
				artifacts.printed,
			)
		}
	}
	if count := strings.Count(
		artifacts.printed,
		"function Apply$cooperative_",
	); count != 1 {
		t.Fatalf("Apply cooperative profile count = %d, want 1", count)
	}
	if count := strings.Count(
		artifacts.printed,
		"function FilterSequence$cooperative_",
	); count != 2 {
		t.Fatalf("FilterSequence cooperative profile count = %d, want 2", count)
	}
	applyProfileName := cooperativeFunctionName(cooperativeApply)
	if applyProfileName == "" {
		t.Fatalf("cooperative Apply function has no target name:\n%s", cooperativeApply)
	}
	if !packageAssemblyExports(emission.Files(), "genericcallback", applyProfileName) {
		t.Fatalf(
			"package assembly does not export %s",
			applyProfileName,
		)
	}
	for _, required := range []string{
		"Apply<gostring>(",
		"await Apply$cooperative_",
	} {
		if !strings.Contains(artifacts.printed, required) {
			t.Fatalf(
				"package initialization lacks %q:\n%s",
				required,
				artifacts.printed,
			)
		}
	}
	if count := strings.Count(
		artifacts.printed,
		"export async function $initialize(): Promise<void>",
	); count != 2 {
		t.Fatalf("cooperative package initializer count = %d, want 2", count)
	}
	programInitialization := strings.Join(
		artifacts.printedByKind[emit.TargetFileProgramInitialization],
		"\n",
	)
	if count := strings.Count(programInitialization, "await "); count != 2 {
		t.Fatalf(
			"program cooperative package await count = %d, want 2:\n%s",
			count,
			programInitialization,
		)
	}
	independent := waveNineFunctionText(
		t,
		artifacts.printed,
		"IndependentSynchronous",
	)
	for _, forbidden := range []string{"async", "Promise<", "await "} {
		if strings.Contains(independent, forbidden) {
			t.Fatalf(
				"unrelated callable ABI contains %q:\n%s",
				forbidden,
				independent,
			)
		}
	}

	runner := filepath.Join(workingDirectory, "runner.ts")
	writeProgramFile(t, runner, `import "./program.js";
import {
    CooperativeApply,
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
        await SynchronousApply(),
        await CooperativeFunctionValue(),
        SynchronousFunctionValue(),
        await CooperativeNested(),
        await CooperativeMethod(),
        await CooperativeMethodValue(),
        SynchronousMethodValue(),
        await CooperativeMethodExpression(),
        await CooperativeResult(),
        await CooperativeSequence(),
        await CooperativeBoolSequence(),
        InitializedSynchronousCallback(),
        InitializedCooperativeCallback(),
        IndependentPackageInitializer(),
        IndependentSynchronous(),
        SynchronousSequence(),
        await CooperativeGenericProfileWithNamedCallback(),
        await CooperativeNestedGenericMethod(),
        await CooperativeRecursiveGenericMethod(),
        await CooperativeGenericInterfaceMethod(),
        await SynchronousGenericInterfaceMethod(),
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
	targetOutput := runProgram(
		t,
		workingDirectory,
		"node",
		filepath.Join(workingDirectory, "out", "runner.js"),
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
	if targetOutput != goOutput {
		t.Fatalf(
			"generic callback output differs\nTypeScript: %q\nGo: %q",
			targetOutput,
			goOutput,
		)
	}
}

func waveNineFunctionWithPrefix(
	t *testing.T,
	printed string,
	prefix string,
) string {
	t.Helper()
	start := strings.Index(printed, "function "+prefix)
	if start < 0 {
		t.Fatalf("generated output lacks function prefix %s:\n%s", prefix, printed)
	}
	start = strings.LastIndex(printed[:start], "export ")
	if start < 0 {
		t.Fatalf("generated function prefix %s is not exported", prefix)
	}
	rest := printed[start:]
	next := strings.Index(rest[len("export "):], "\nexport ")
	if next < 0 {
		return rest
	}
	return rest[:len("export ")+next]
}

func waveNineClassMemberText(
	t *testing.T,
	printed string,
	className string,
	marker string,
) string {
	t.Helper()
	classStart := strings.Index(printed, "export class "+className)
	if classStart < 0 {
		t.Fatalf("generated output lacks class %s", className)
	}
	memberOffset := strings.Index(printed[classStart:], marker)
	if memberOffset < 0 {
		t.Fatalf("generated class %s lacks member marker %q", className, marker)
	}
	memberStart := classStart + memberOffset + 1
	memberEnd := strings.Index(printed[memberStart:], "\n    }\n")
	if memberEnd < 0 {
		t.Fatalf("generated class %s member %q has no boundary", className, marker)
	}
	return printed[memberStart : memberStart+memberEnd+len("\n    }")]
}

func cooperativeFunctionName(function string) string {
	start := strings.Index(function, "function ")
	if start < 0 {
		return ""
	}
	start += len("function ")
	end := strings.IndexByte(function[start:], '<')
	if end < 0 {
		end = strings.IndexByte(function[start:], '(')
	}
	if end < 0 {
		return ""
	}
	return function[start : start+end]
}

func packageAssemblyExports(
	files []emit.TargetFile,
	packageName string,
	name string,
) bool {
	for _, file := range files {
		if file.Kind() != emit.TargetFilePackageAssembly ||
			file.PackageName() != packageName {
			continue
		}
		for _, statement := range file.SourceFile().Statements() {
			declaration, ok := statement.(tsgo.ExportDeclaration)
			if !ok {
				continue
			}
			exports, ok := declaration.ExportClause().(tsgo.NamedExports)
			if !ok {
				continue
			}
			for _, specifier := range exports.Elements() {
				identifier, ok := specifier.Name().(tsgo.Identifier)
				if ok && identifier.Text() == name {
					return true
				}
			}
		}
	}
	return false
}
