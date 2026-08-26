package emit_test

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/tsoniclang/gotots/internal/emit"
	"github.com/tsoniclang/gotots/internal/load"
)

func TestWaveSevenGenericAssertionsCompileThroughPublicPipeline(t *testing.T) {
	for _, testCase := range []struct {
		name    string
		options emit.Options
	}{
		{name: "number", options: emit.DefaultOptions()},
		{
			name: "bigint",
			options: emit.Options{
				IntegerRepresentation: emit.IntegerRepresentationBigInt,
				EvaluationOrder:       emit.EvaluationOrderPreserveGo,
			},
		},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			program, err := load.Load(context.Background(), load.Request{
				Directory: waveSevenGenericDirectory(),
				Pattern:   ".",
			})
			if err != nil {
				t.Fatal(err)
			}
			root, err := emit.NewRoot(
				program.Roots()[0].Types().Scope().Lookup(
					"AuditGenericAssertions",
				),
			)
			if err != nil {
				t.Fatal(err)
			}
			emission, err := emit.CompileWithOptions(
				program,
				[]emit.Root{root},
				testCase.options,
			)
			if err != nil {
				t.Fatal(err)
			}
			workingDirectory := t.TempDir()
			artifacts := materializeArtifacts(t, emission, workingDirectory)
			for _, required := range []string{
				"export function AssertValue$kernel<T>",
				"export function MustAssertValue$kernel<T>",
				"export function TypeSwitchValue$kernel<T>",
				"export function AssertValue$int32",
				"export function MustAssertValue$int32",
				"export function TypeSwitchValue$int32",
				"$go$interface_assert$",
				"$go$interface_assert_ok$",
			} {
				if !strings.Contains(artifacts.printed, required) {
					t.Fatalf(
						"generic assertion artifacts lack %q:\n%s",
						required,
						artifacts.printed,
					)
				}
			}
			runner := filepath.Join(workingDirectory, "runner.ts")
			writeProgramFile(t, runner, `import "./program.js";
import { AuditGenericAssertions } from "`+artifacts.sourceModule+`";

const values = AuditGenericAssertions();
console.log(Array.from({ length: values.length }, (_, index) =>
    String(values.get(index))).join(" "));
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
			goOutput := executeWaveSevenGenericGo(
				t,
				workingDirectory,
				"AuditGenericAssertions",
			)
			if targetOutput != goOutput {
				t.Fatalf(
					"generic assertion output differs\nTypeScript:\n%s\nGo:\n%s",
					targetOutput,
					goOutput,
				)
			}
		})
	}
}

func sourceModuleForExport(
	t *testing.T,
	artifacts waveFourArtifacts,
	workingDirectory string,
	name string,
) string {
	t.Helper()
	var selected string
	for _, path := range artifacts.paths {
		content, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		printed := string(content)
		if !strings.Contains(
			printed,
			"export function "+name+"(",
		) {
			continue
		}
		if selected != "" {
			t.Fatalf("multiple source modules export %s", name)
		}
		selected = path
	}
	if selected == "" {
		t.Fatalf("no source module exports %s", name)
	}
	relative, err := filepath.Rel(workingDirectory, selected)
	if err != nil {
		t.Fatal(err)
	}
	return "./" + strings.TrimSuffix(filepath.ToSlash(relative), ".ts") + ".js"
}

func assertWaveSevenGenericFoundationShape(t *testing.T, printed string) {
	t.Helper()
	for _, required := range []string{
		"export function Identity$kernel<",
		"export function Add$kernel<",
		"export function Zero$kernel<",
		"export function ZeroFromNew$kernel<",
		"export function Equal$kernel<",
		"export function Twice$kernel<",
		"export function Add$int32",
		"export function Zero$int32",
		"export function ZeroFromNew$int32",
		"export function Equal$int32",
		"export function Twice$int32",
		"export function Identity$int32",
	} {
		if !strings.Contains(printed, required) {
			t.Fatalf("Wave 7 generic artifacts lack %q:\n%s", required, printed)
		}
	}
	for _, forbidden := range []string{
		"export function Identity<T>",
		"export function Add<T>",
		"export function Zero<T>",
		"export function Equal<T>",
		"class GoValueOps",
		"interface GoValueOps",
		"Record<string",
		"switch (typeof",
	} {
		if strings.Contains(printed, forbidden) {
			t.Fatalf(
				"Wave 7 generic artifacts contain %q:\n%s",
				forbidden,
				printed,
			)
		}
	}
	if strings.Count(printed, "export function Add$int32") != 1 {
		t.Fatalf("generic Add body was duplicated:\n%s", printed)
	}
	zeroFromNew := concreteFunctionText(t, printed, "ZeroFromNew")
	if !strings.Contains(zeroFromNew, "ZeroFromNew$kernel<") ||
		!strings.Contains(zeroFromNew, "(): int32 =>") ||
		!strings.Contains(zeroFromNew, "return 0") {
		t.Fatalf(
			"generic *new(T) wrapper did not delegate with one inline zero operation:\n%s",
			zeroFromNew,
		)
	}
	zeroFromNewKernel := genericKernelText(t, printed, "ZeroFromNew")
	if !strings.Contains(zeroFromNewKernel, "$go$zero$") ||
		strings.Contains(zeroFromNewKernel, "return 0") ||
		strings.Contains(zeroFromNewKernel, "GoPointer") {
		t.Fatalf(
			"generic *new(T) kernel bypassed its selected zero capability:\n%s",
			zeroFromNewKernel,
		)
	}
	if strings.Contains(printed, "GoPointer.field<T, Box") {
		t.Fatalf(
			"direct generic pointer-receiver field assignment formed an interior pointer:\n%s",
			printed,
		)
	}
	sameStorage := concreteFunctionText(t, printed, "SameSliceStorage")
	if !strings.Contains(sameStorage, "SameSliceStorage$kernel<") ||
		strings.Contains(sameStorage, "GoPointer.equal") {
		t.Fatalf(
			"generic pointer-equality wrapper did not delegate to its declaration-owned kernel:\n%s",
			sameStorage,
		)
	}
	sameStorageKernel := genericKernelText(t, printed, "SameSliceStorage")
	if !strings.Contains(sameStorageKernel, "$go$equal$") ||
		strings.Contains(sameStorageKernel, "GoPointer.equal") {
		t.Fatalf(
			"generic pointer-equality kernel bypassed its selected capability:\n%s",
			sameStorageKernel,
		)
	}
	twice := concreteFunctionText(t, printed, "Twice")
	if !strings.Contains(twice, "Twice$kernel<") {
		t.Fatalf(
			"generic forwarding wrapper did not delegate to its declaration-owned kernel:\n%s",
			twice,
		)
	}
	twiceKernel := genericKernelText(t, printed, "Twice")
	if !strings.Contains(twiceKernel, "Add$kernel<") ||
		strings.Contains(twiceKernel, "Add<T>") {
		t.Fatalf(
			"open generic forwarding did not select the callee kernel:\n%s",
			twiceKernel,
		)
	}
}

func genericKernelText(
	t *testing.T,
	printed string,
	name string,
) string {
	t.Helper()
	startMarker := "export function " + name + "$kernel"
	start := strings.Index(printed, startMarker)
	if start < 0 {
		t.Fatalf("Wave 7 artifacts lack generic kernel %s", name)
	}
	remainder := printed[start+len(startMarker):]
	end := strings.Index(remainder, "\nexport function ")
	artifactEnd := strings.Index(remainder, "\n\n// ")
	if end < 0 || artifactEnd >= 0 && artifactEnd < end {
		end = artifactEnd
	}
	if end < 0 {
		return printed[start:]
	}
	return printed[start : start+len(startMarker)+end]
}

func concreteFunctionText(
	t *testing.T,
	printed string,
	name string,
) string {
	t.Helper()
	startMarker := "export function " + name + "$int32"
	start := strings.Index(printed, startMarker)
	if start < 0 {
		t.Fatalf("Wave 7 artifacts lack concrete function %s", name)
	}
	remainder := printed[start+len(startMarker):]
	end := strings.Index(remainder, "\nexport function ")
	artifactEnd := strings.Index(remainder, "\n\n// ")
	if end < 0 || artifactEnd >= 0 && artifactEnd < end {
		end = artifactEnd
	}
	if end < 0 {
		return printed[start:]
	}
	return printed[start : start+len(startMarker)+end]
}

func targetGenericFunctionText(
	t *testing.T,
	printed string,
	name string,
) string {
	t.Helper()
	startMarker := "export function " + name + "<"
	start := strings.Index(printed, startMarker)
	if start < 0 {
		t.Fatalf("Wave 7 artifacts lack generic function %s", name)
	}
	remaining := printed[start+len(startMarker):]
	end := strings.Index(remaining, "\nexport function ")
	if end < 0 {
		return printed[start:]
	}
	return printed[start : start+len(startMarker)+end]
}

func executeWaveSevenGenericGo(
	t *testing.T,
	workingDirectory string,
	function string,
) string {
	t.Helper()
	modulePath, err := filepath.Abs(waveSevenGenericDirectory())
	if err != nil {
		t.Fatal(err)
	}
	runnerDirectory := filepath.Join(workingDirectory, "go-runner-wave7")
	writeProgramFile(t, filepath.Join(runnerDirectory, "go.mod"), fmt.Sprintf(
		`module example.com/runner

go 1.26.4

require example.com/wave7generics v0.0.0

replace example.com/wave7generics => %s
`,
		filepath.ToSlash(modulePath),
	))
	writeProgramFile(t, filepath.Join(runnerDirectory, "main.go"), fmt.Sprintf(`package main

import (
	"fmt"

	values "example.com/wave7generics"
)

func main() {
	for index, value := range values.%s() {
		if index != 0 {
			fmt.Print(" ")
		}
		fmt.Print(value)
	}
	fmt.Println()
}
`, function))
	return runProgram(
		t,
		runnerDirectory,
		filepath.Join(runtime.GOROOT(), "bin", "go"),
		"run",
		".",
	)
}
