package integer_test

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

func TestBigIntDivisionByZeroUsesSharedGoPanicDifferentially(t *testing.T) {
	loaded := loadIntegerFamily(t)
	options := integerOptions(emit.IntegerRepresentationBigInt)
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
		!strings.Contains(integerRuntime, "GoPanic.raiseRuntime") ||
		strings.Count(panicRuntime, "export class GoPanic") != 1 ||
		!strings.Contains(panicRuntime, "export class GoRuntimePanicValue") ||
		strings.Contains(panicRuntime, "export class GoPanic<") {
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

func TestNumberIntegerDivisionAndRemainderExecuteDifferentially(t *testing.T) {
	loaded, err := load.One(context.Background(), load.Request{
		Directory: integerBoundaryDirectory(),
		Pattern:   ".",
	})
	if err != nil {
		t.Fatal(err)
	}
	var roots []emit.Root
	for _, name := range []string{
		"NumberDivide",
		"NumberRemainder",
		"NumberCompound",
	} {
		root, rootErr := emit.NewRoot(loaded.Types().Scope().Lookup(name))
		if rootErr != nil {
			t.Fatal(rootErr)
		}
		roots = append(roots, root)
	}
	emission, err := emit.Compile(loaded.Program(), roots)
	if err != nil {
		t.Fatal(err)
	}
	workingDirectory := t.TempDir()
	artifacts := materializeIntegerFamily(t, emission, workingDirectory)
	integerRuntime := readIntegerArtifact(
		t,
		filepath.Join(workingDirectory, "runtime", "integer.ts"),
	)
	for _, expected := range []struct {
		text  string
		count int
	}{
		{"export function goNumberIntegerDivide", 1},
		{"export function goNumberIntegerRemainder", 1},
		{"Math.trunc(left / right)", 1},
		{`GoPanic.raiseRuntime("integer divide by zero")`, 2},
		{"return result === 0 ? 0 : result;", 2},
	} {
		if strings.Count(integerRuntime, expected.text) != expected.count {
			t.Fatalf(
				"number integer runtime count(%q) != %d:\n%s",
				expected.text,
				expected.count,
				integerRuntime,
			)
		}
	}
	runner := filepath.Join(workingDirectory, "runner.ts")
	writeFile(t, runner, `import {
    NumberCompound,
    NumberDivide,
    NumberRemainder,
} from "`+artifacts.module(t, "source.ts")+`";

const panics = (operation: () => void): boolean => {
    try {
        operation();
        return false;
    } catch {
        return true;
    }
};

console.log(NumberDivide(-17, 5));
console.log(NumberRemainder(-17, 5));
console.log(NumberDivide(17, -5));
console.log(NumberRemainder(17, -5));
console.log(NumberCompound(-17, 5).join(" "));
console.log(Object.is(NumberDivide(-1, 2), -0));
console.log(Object.is(NumberRemainder(-5, 5), -0));
console.log(panics(() => { NumberDivide(1, 0); }));
console.log(panics(() => { NumberRemainder(1, 0); }));
`)
	targetOutput := executeMaterializedTypeScript(
		t,
		workingDirectory,
		artifacts,
		runner,
	)
	goOutput := executeNumberIntegerOperationsGo(t, workingDirectory)
	if targetOutput != goOutput {
		t.Fatalf(
			"number integer TypeScript output = %q, Go output = %q",
			targetOutput,
			goOutput,
		)
	}
}

func executeNumberIntegerOperationsGo(
	t *testing.T,
	workingDirectory string,
) string {
	t.Helper()
	modulePath, err := filepath.Abs(integerBoundaryDirectory())
	if err != nil {
		t.Fatal(err)
	}
	runnerDirectory := filepath.Join(workingDirectory, "go-number-runner")
	writeFile(t, filepath.Join(runnerDirectory, "go.mod"), fmt.Sprintf(`module example.com/runner

go 1.26.4

require example.com/integerboundaries v0.0.0

replace example.com/integerboundaries => %s
`, filepath.ToSlash(modulePath)))
	writeFile(t, filepath.Join(runnerDirectory, "main.go"), `package main

import (
	"fmt"
	"math"
	values "example.com/integerboundaries"
)

func panics(operation func()) (result bool) {
	defer func() {
		result = recover() != nil
	}()
	operation()
	return false
}

func main() {
	fmt.Println(values.NumberDivide(-17, 5))
	fmt.Println(values.NumberRemainder(-17, 5))
	fmt.Println(values.NumberDivide(17, -5))
	fmt.Println(values.NumberRemainder(17, -5))
	fmt.Println(values.NumberCompound(-17, 5))
	fmt.Println(math.Signbit(float64(values.NumberDivide(-1, 2))))
	fmt.Println(math.Signbit(float64(values.NumberRemainder(-5, 5))))
	fmt.Println(panics(func() { values.NumberDivide(1, 0) }))
	fmt.Println(panics(func() { values.NumberRemainder(1, 0) }))
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
