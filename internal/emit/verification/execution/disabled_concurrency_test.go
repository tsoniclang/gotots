package emit_test

import (
	"context"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tsoniclang/gotots/internal/emit"
	"github.com/tsoniclang/gotots/internal/load"
)

func TestWaveNineDisabledProfileIsSynchronous(t *testing.T) {
	directory := t.TempDir()
	writeProgramFile(
		t,
		filepath.Join(directory, "go.mod"),
		"module example.com/concurrencyprofile\n\ngo 1.26.4\n",
	)
	writeProgramFile(t, filepath.Join(directory, "source.go"), `package concurrencyprofile

func Run() int32 {
	values := make(chan int32, 1)
	go func() { values <- 1 }()
	select {
	case value := <-values:
		return value
	default:
		return -1
	}
}

func Block() {
	values := make(chan int32)
	values <- 1
}
`)
	program, err := load.Load(context.Background(), load.Request{
		Directory: directory,
		Pattern:   ".",
	})
	if err != nil {
		t.Fatal(err)
	}
	root, err := emit.NewRoot(
		program.Roots()[0].Types().Scope().Lookup("Run"),
	)
	if err != nil {
		t.Fatal(err)
	}
	blockRoot, err := emit.NewRoot(
		program.Roots()[0].Types().Scope().Lookup("Block"),
	)
	if err != nil {
		t.Fatal(err)
	}
	emission, err := emit.Compile(program, []emit.Root{root, blockRoot})
	if err != nil {
		t.Fatal(err)
	}
	workingDirectory := t.TempDir()
	artifacts := materializeArtifacts(t, emission, workingDirectory)
	for _, forbidden := range []string{
		"Promise<",
		"async ",
		"await ",
		"GoScheduler",
	} {
		if strings.Contains(artifacts.printed, forbidden) {
			t.Fatalf(
				"disabled concurrency emitted %q:\n%s",
				forbidden,
				artifacts.printed,
			)
		}
	}
	runner := filepath.Join(workingDirectory, "runner.ts")
	writeProgramFile(t, runner, `import "./program.js";
import { Block, Run } from "`+artifacts.sourceModule+`";
import { GoPanic, GoRuntimePanicValue } from "./runtime/panic.js";
let boundary = "missing";
try {
    Block();
} catch (failure) {
    boundary = failure instanceof GoPanic && failure.value instanceof GoRuntimePanicValue
        ? failure.value.Error()
        : "non-panic";
}
console.log(String(Run()) + "|" + boundary);
`)
	writeProgramFile(
		t,
		filepath.Join(workingDirectory, "package.json"),
		"{\"type\":\"module\"}\n",
	)
	paths := append(artifacts.paths, runner)
	waveThreeTypecheck(t, workingDirectory, paths)
	if output := runProgram(
		t,
		workingDirectory,
		"node",
		filepath.Join(workingDirectory, "out", "runner.js"),
	); output != "1|synchronous channel send would block\n" {
		t.Fatalf("disabled concurrency output = %q", output)
	}
}
