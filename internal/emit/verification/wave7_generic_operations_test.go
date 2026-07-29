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

func TestWaveSevenBigIntGenericArithmeticExecutesDifferentially(t *testing.T) {
	program, err := load.Load(context.Background(), load.Request{
		Directory: waveSevenGenericDirectory(),
		Pattern:   ".",
	})
	if err != nil {
		t.Fatal(err)
	}
	root, err := emit.NewRoot(
		program.Roots()[0].Types().Scope().Lookup("AuditBigIntOperations"),
	)
	if err != nil {
		t.Fatal(err)
	}
	options := emit.Options{
		IntegerRepresentation: emit.IntegerRepresentationBigInt,
		EvaluationOrder:       emit.EvaluationOrderPreserveGo,
	}
	emission, err := emit.CompileWithOptions(
		program,
		[]emit.Root{root},
		options,
	)
	if err != nil {
		t.Fatal(err)
	}
	workingDirectory := t.TempDir()
	artifacts := materializeArtifacts(t, emission, workingDirectory)
	for _, required := range []string{
		"export function Arithmetic<T>",
		"$go$binary_divide_",
		"$go$binary_remainder_",
		"goIntegerDivide",
		"goIntegerRemainder",
	} {
		if !strings.Contains(artifacts.printed, required) {
			t.Fatalf("generic arithmetic artifacts lack %q", required)
		}
	}
	sourceModule := sourceModuleForExport(
		t,
		artifacts,
		workingDirectory,
		"AuditBigIntOperations",
	)
	runner := filepath.Join(workingDirectory, "runner.ts")
	writeProgramFile(t, runner, `import "./program.js";
import { AuditBigIntOperations } from "`+sourceModule+`";

console.log(AuditBigIntOperations().toString());
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
	goOutput := executeWaveSevenScalarGo(
		t,
		workingDirectory,
		"AuditBigIntOperations",
	)
	if targetOutput != goOutput {
		t.Fatalf(
			"generic arithmetic differs\nTypeScript:\n%s\nGo:\n%s",
			targetOutput,
			goOutput,
		)
	}
}

func executeWaveSevenScalarGo(
	t *testing.T,
	workingDirectory string,
	function string,
) string {
	t.Helper()
	modulePath, err := filepath.Abs(waveSevenGenericDirectory())
	if err != nil {
		t.Fatal(err)
	}
	runnerDirectory := filepath.Join(workingDirectory, "go-scalar-wave7")
	writeProgramFile(t, filepath.Join(runnerDirectory, "go.mod"), fmt.Sprintf(
		`module example.com/runner

go 1.26.4

require example.com/wave7generics v0.0.0

replace example.com/wave7generics => %s
`,
		filepath.ToSlash(modulePath),
	))
	writeProgramFile(t, filepath.Join(runnerDirectory, "main.go"), fmt.Sprintf(
		`package main

import (
	"fmt"

	values "example.com/wave7generics"
)

func main() {
	fmt.Println(values.%s())
}
`,
		function,
	))
	return runProgram(
		t,
		runnerDirectory,
		filepath.Join(runtime.GOROOT(), "bin", "go"),
		"run",
		".",
	)
}
