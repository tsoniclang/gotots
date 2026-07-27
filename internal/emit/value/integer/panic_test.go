package integer_test

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/tsoniclang/gotots/internal/emit"
)

func TestBigIntDivisionByZeroUsesSharedGoPanicDifferentially(t *testing.T) {
	loaded := loadIntegerFamily(t)
	options := emit.DefaultOptions()
	options.IntegerRepresentation = emit.IntegerRepresentationBigInt
	emission := compileIntegerFamily(
		t,
		loaded,
		options,
		"BigSigned",
	)
	workingDirectory := t.TempDir()
	artifacts := materializeIntegerFamily(t, emission, workingDirectory)
	integerRuntime := readIntegerArtifact(
		t,
		filepath.Join(workingDirectory, "runtime", "integer.ts"),
	)
	panicRuntime := readIntegerArtifact(
		t,
		filepath.Join(workingDirectory, "runtime", "panic.ts"),
	)
	if strings.Count(
		integerRuntime,
		`import { GoPanic } from "./panic.js";`,
	) != 1 ||
		!strings.Contains(integerRuntime, "GoPanic.raise") ||
		strings.Count(panicRuntime, "export class GoPanic<T>") != 1 {
		t.Fatalf(
			"shared panic artifacts are not exact:\n%s\n%s",
			integerRuntime,
			panicRuntime,
		)
	}
	runner := filepath.Join(workingDirectory, "runner.ts")
	writeFile(t, runner, `import { BigSigned } from "`+
		artifacts.module(t, "source.ts")+`";
import { GoPanic } from "./runtime/panic.js";

try {
    BigSigned(1n, 0n);
    console.log(false);
} catch (error) {
    console.log(error instanceof GoPanic);
}
`)
	targetOutput := executeMaterializedTypeScript(
		t,
		workingDirectory,
		artifacts,
		runner,
	)
	goOutput := executeIntegerDivideByZeroGo(t, workingDirectory)
	if targetOutput != goOutput || targetOutput != "true\n" {
		t.Fatalf(
			"TypeScript panic output = %q, Go output = %q",
			targetOutput,
			goOutput,
		)
	}
}

func executeIntegerDivideByZeroGo(
	t *testing.T,
	workingDirectory string,
) string {
	t.Helper()
	modulePath, err := filepath.Abs(integerFamilyDirectory())
	if err != nil {
		t.Fatal(err)
	}
	runnerDirectory := filepath.Join(workingDirectory, "go-zero-runner")
	writeFile(t, filepath.Join(runnerDirectory, "go.mod"), fmt.Sprintf(`module example.com/runner

go 1.26.4

require example.com/integerfamily v0.0.0

replace example.com/integerfamily => %s
`, filepath.ToSlash(modulePath)))
	writeFile(t, filepath.Join(runnerDirectory, "main.go"), `package main

import (
	"fmt"
	values "example.com/integerfamily"
)

func panics() (result bool) {
	defer func() {
		result = recover() != nil
	}()
	values.BigSigned(1, 0)
	return false
}

func main() {
	fmt.Println(panics())
}
`)
	return run(
		t,
		runnerDirectory,
		filepath.Join(runtime.GOROOT(), "bin", "go"),
		"run",
		".",
	)
}

func readIntegerArtifact(t *testing.T, path string) string {
	t.Helper()
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(content)
}
