package emit_test

import (
	"fmt"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestGenericCallableTransportIsUniformlySynchronous(t *testing.T) {
	directory, workingDirectory, emission, artifacts :=
		serialGenericCallbackFixture(t)
	for _, forbidden := range []string{
		"async ",
		"await ",
		"Promise<",
		"Awaitable<",
		"GoScheduler",
		"$cooperative_",
		"instanceof Promise",
		".then ===",
	} {
		if strings.Contains(artifacts.printed, forbidden) {
			t.Fatalf("synchronous generic callback output contains %q", forbidden)
		}
	}
	apply := waveNineFunctionText(t, artifacts.printed, "Apply$kernel")
	for _, required := range []string{
		"export function Apply$kernel<T>(",
		"predicate: (($0: T) => bool) | undefined",
	} {
		if !strings.Contains(apply, required) {
			t.Fatalf("synchronous generic callable lacks %q:\n%s", required, apply)
		}
	}
	if count := strings.Count(
		artifacts.printed,
		"export function Apply$kernel<T>(",
	); count != 1 {
		t.Fatalf("generic Apply kernel count = %d, want one", count)
	}
	if packageAssemblyExports(emission.Files(), "genericcallback", "Apply$kernel") {
		t.Fatal("package assembly publishes private generic callable kernel")
	}

	sourceModule := sourceModuleForExport(
		t,
		artifacts,
		workingDirectory,
		"ChannelApply",
	)
	runner := filepath.Join(workingDirectory, "runner.ts")
	writeProgramFile(t, runner, `import "./program.js";
import * as values from "`+sourceModule+`";

console.log([
    values.ChannelApply(),
    values.ChannelLexicalResult(),
    values.PlainApply(),
    values.ChannelFunctionValue(),
    values.PlainFunctionValue(),
    values.ChannelNested(),
    values.ChannelMethod(),
    values.ChannelMethodValue(),
    values.PlainMethodValue(),
    values.ChannelMethodExpression(),
    values.ChannelResult(),
    values.ChannelSequence(),
    values.ChannelBoolSequence(),
    values.InitializedPlainCallback(),
    values.InitializedChannelCallback(),
    values.IndependentPackageInitializer(),
    values.IndependentPlain(),
    values.PlainSequence(),
	values.ChannelGenericProfileWithNamedCallback(),
	values.ChannelNestedGenericMethod(),
].map(String).join(" "));
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
		values.ChannelApply(),
		values.ChannelLexicalResult(),
		values.PlainApply(),
		values.ChannelFunctionValue(),
		values.PlainFunctionValue(),
		values.ChannelNested(),
		values.ChannelMethod(),
		values.ChannelMethodValue(),
		values.PlainMethodValue(),
		values.ChannelMethodExpression(),
		values.ChannelResult(),
		values.ChannelSequence(),
		values.ChannelBoolSequence(),
		values.InitializedPlainCallback(),
		values.InitializedChannelCallback(),
        values.IndependentPackageInitializer(),
		values.IndependentPlain(),
		values.PlainSequence(),
		values.ChannelGenericProfileWithNamedCallback(),
		values.ChannelNestedGenericMethod(),
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
		t.Fatalf("generic callback mismatch:\nGo: %s\nTypeScript: %s", goOutput, targetOutput)
	}
}
