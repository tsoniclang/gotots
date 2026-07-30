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

func TestCooperativeIteratorCallbackPropagatesThroughCallableABIs(
	t *testing.T,
) {
	directory := filepath.Join(
		repositoryRoot(),
		"testdata",
		"constructs",
		"concurrency",
		"iterator-callback",
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
	cooperative := waveNineFunctionText(
		t,
		artifacts.printed,
		"CooperativeAudit",
	)
	for _, required := range []string{
		"export async function CooperativeAudit(",
		"async ($argument0: int32): Promise<bool> =>",
		"await __gotots_range_",
	} {
		if !strings.Contains(cooperative, required) {
			t.Fatalf(
				"cooperative iterator artifact lacks %q:\n%s",
				required,
				cooperative,
			)
		}
	}
	synchronous := waveNineFunctionText(
		t,
		artifacts.printed,
		"SynchronousAudit",
	)
	for _, forbidden := range []string{"async", "Promise<", "await "} {
		if strings.Contains(synchronous, forbidden) {
			t.Fatalf(
				"synchronous iterator artifact contains %q:\n%s",
				forbidden,
				synchronous,
			)
		}
	}
	runner := filepath.Join(workingDirectory, "runner.ts")
	writeProgramFile(t, runner, `import "./program.js";
import { CooperativeAudit, SynchronousAudit } from "`+artifacts.sourceModule+`";
import { GoScheduler } from "./runtime/channel.js";

await GoScheduler.run(async () => {
    console.log(String(await CooperativeAudit()) + " " + String(SynchronousAudit()));
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

require example.com/iteratorcallback v0.0.0

replace example.com/iteratorcallback => %s
`,
		filepath.ToSlash(directory),
	))
	writeProgramFile(t, filepath.Join(goRunner, "main.go"), `package main

import (
	"fmt"

	values "example.com/iteratorcallback"
)

func main() {
	fmt.Println(values.CooperativeAudit(), values.SynchronousAudit())
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
			"cooperative iterator output differs\nTypeScript: %q\nGo: %q",
			targetOutput,
			goOutput,
		)
	}
}
