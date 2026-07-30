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

func TestCooperativeStructuredForClausesStayInTheEnclosingCallable(
	t *testing.T,
) {
	directory := filepath.Join(
		repositoryRoot(),
		"testdata",
		"constructs",
		"concurrency",
		"for-clause",
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
	for _, function := range []string{
		"CooperativeCondition",
		"CooperativePost",
	} {
		target := waveNineFunctionText(t, artifacts.printed, function)
		if !strings.Contains(target, "export async function "+function+"(") {
			t.Fatalf("structured for function is not cooperative:\n%s", target)
		}
		for _, forbidden := range []string{
			"__gotots_for_condition_",
			"__gotots_for_post_",
		} {
			if strings.Contains(target, forbidden) {
				t.Fatalf(
					"structured for function retains %q callback:\n%s",
					forbidden,
					target,
				)
			}
		}
	}
	post := waveNineFunctionText(t, artifacts.printed, "CooperativePost")
	for _, required := range []string{
		"let __gotots_for_first_",
		"if (__gotots_for_first_",
		"else {",
		"await next(",
	} {
		if !strings.Contains(post, required) {
			t.Fatalf("structured post lacks %q:\n%s", required, post)
		}
	}
	runner := filepath.Join(workingDirectory, "runner.ts")
	writeProgramFile(t, runner, `import "./program.js";
import { CooperativeCondition, CooperativePost } from "`+artifacts.sourceModule+`";
import { GoScheduler } from "./runtime/channel.js";

await GoScheduler.run(async () => {
    console.log(String(await CooperativeCondition()) + " " + String(await CooperativePost()));
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

require example.com/cooperativeforclause v0.0.0

replace example.com/cooperativeforclause => %s
`,
		filepath.ToSlash(directory),
	))
	writeProgramFile(t, filepath.Join(goRunner, "main.go"), `package main

import (
	"fmt"

	values "example.com/cooperativeforclause"
)

func main() {
	fmt.Println(values.CooperativeCondition(), values.CooperativePost())
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
			"cooperative for-clause output differs\nTypeScript: %q\nGo: %q",
			targetOutput,
			goOutput,
		)
	}
}
