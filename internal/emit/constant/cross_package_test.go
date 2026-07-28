package constant_test

import (
	"context"
	"fmt"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/tsoniclang/gotots/internal/emit"
	"github.com/tsoniclang/gotots/internal/load"
)

func crossPackageDirectory() string {
	return filepath.Join(repositoryRoot(), "testdata", "projects", "constant-cross-package")
}

// TestConstantCrossPackageProjectionsExecuteDifferentially proves an untyped
// constant declared in one package projects correctly into another package that
// uses it — both through a qualified selector (defs.Width) and through a
// dot-import (bare Width). Each cross-package use imports its projection by
// derived name from the provider's assembly, which re-exports exactly the
// projections its consumers demanded. The whole program strict-typechecks and
// executes identically to Go.
func TestConstantCrossPackageProjectionsExecuteDifferentially(t *testing.T) {
	program, err := load.Load(context.Background(), load.Request{
		Directory: crossPackageDirectory(),
		Pattern:   "./api",
	})
	if err != nil {
		t.Fatal(err)
	}
	loaded := program.Roots()[0]
	roots, err := emit.ExportedAPIRoots(loaded)
	if err != nil {
		t.Fatal(err)
	}
	emission, err := emit.Compile(loaded.Program(), roots)
	if err != nil {
		t.Fatalf("cross-package constant compile failed: %v", err)
	}

	workingDirectory := t.TempDir()
	artifacts := materializeConstantFamily(t, emission, workingDirectory)

	goOutput := executeCrossPackageGo(t, workingDirectory)
	targetOutput := executeCrossPackageTS(t, artifacts, workingDirectory)
	if targetOutput != goOutput {
		t.Fatalf("cross-package TypeScript output = %q, Go output = %q", targetOutput, goOutput)
	}
}

func executeCrossPackageGo(t *testing.T, workingDirectory string) string {
	t.Helper()
	modulePath, err := filepath.Abs(crossPackageDirectory())
	if err != nil {
		t.Fatal(err)
	}
	runnerDirectory := filepath.Join(workingDirectory, "cross-go")
	writeFile(t, filepath.Join(runnerDirectory, "go.mod"), fmt.Sprintf(`module example.com/runner

go 1.26.4

require example.com/constantcross v0.0.0

replace example.com/constantcross => %s
`, filepath.ToSlash(modulePath)))
	writeFile(t, filepath.Join(runnerDirectory, "main.go"), `package main

import (
	"fmt"

	"example.com/constantcross/api"
)

func main() {
	fmt.Println(api.Widths())
	fmt.Println(api.Enum())
	fmt.Println(api.Flags())
	fmt.Println(api.DotWidth())
	fmt.Println(api.DotLabel())
}
`)
	return run(t, runnerDirectory, filepath.Join(runtime.GOROOT(), "bin", "go"), "run", ".")
}

func executeCrossPackageTS(
	t *testing.T,
	artifacts materializedProgram,
	workingDirectory string,
) string {
	t.Helper()
	runner := `import { Widths, Enum, Flags } from "` + artifacts.module(t, "api.ts") + `";
import { DotWidth, DotLabel } from "` + artifacts.module(t, "dotuse.ts") + `";

const row = (value: readonly unknown[]): string =>
	value.map((entry) => String(entry)).join(" ");

console.log(row(Widths()));
console.log(row(Enum()));
console.log(row(Flags()));
console.log(String(DotWidth()));
console.log(String(DotLabel()));
`
	runnerPath := filepath.Join(workingDirectory, "runner.ts")
	writeFile(t, runnerPath, runner)
	return executeMaterializedTypeScript(t, workingDirectory, artifacts, runnerPath)
}
