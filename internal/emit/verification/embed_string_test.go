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

func TestEmbeddedStringPrintsTypechecksAndExecutesExactBytes(t *testing.T) {
	project := t.TempDir()
	writeProgramFile(t, filepath.Join(project, "go.mod"), `module example.com/embedstring

go 1.26.4
`)
	writeProgramFile(t, filepath.Join(project, "source.go"), `package embedstring

import _ "embed"

//go:embed payload.bin
var payload string

func Audit() string { return payload }
`)
	if err := os.WriteFile(
		filepath.Join(project, "payload.bin"),
		[]byte{0x00, 0x22, 0x5c, 0x7f, 0x80, 0xff, 0x0a},
		0o644,
	); err != nil {
		t.Fatal(err)
	}

	program, err := load.Load(context.Background(), load.Request{
		Directory: project,
		Pattern:   ".",
	})
	if err != nil {
		t.Fatal(err)
	}
	root, err := emit.NewRoot(
		program.Roots()[0].Types().Scope().Lookup("Audit"),
	)
	if err != nil {
		t.Fatal(err)
	}
	emission, err := emit.Compile(program, []emit.Root{root})
	if err != nil {
		t.Fatal(err)
	}
	workingDirectory := t.TempDir()
	artifacts := materializeArtifacts(t, emission, workingDirectory)
	if strings.Contains(artifacts.printed, "$state.payload = \"\";") {
		t.Fatalf("embedded package variable retained its zero value:\n%s", artifacts.printed)
	}

	runner := filepath.Join(workingDirectory, "runner.ts")
	writeProgramFile(t, runner, `import "./program.js";
import { Audit } from "`+sourceModuleForExport(
		t,
		artifacts,
		workingDirectory,
		"Audit",
	)+`";

console.log(Array.from(Audit(), (value) => value.charCodeAt(0).toString(16).padStart(2, "0")).join(""));
`)
	writeProgramFile(
		t,
		filepath.Join(workingDirectory, "package.json"),
		"{\"type\":\"module\"}\n",
	)
	waveThreeTypecheck(t, workingDirectory, append(artifacts.paths, runner))
	targetOutput := runProgram(
		t,
		workingDirectory,
		"node",
		filepath.Join(workingDirectory, "out", "runner.js"),
	)

	goRunner := filepath.Join(workingDirectory, "go-runner")
	writeProgramFile(t, filepath.Join(goRunner, "go.mod"), fmt.Sprintf(
		`module example.com/embedstring-runner

go 1.26.4

require example.com/embedstring v0.0.0

replace example.com/embedstring => %s
`, filepath.ToSlash(project)))
	writeProgramFile(t, filepath.Join(goRunner, "main.go"), `package main

import (
	"fmt"

	"example.com/embedstring"
)

func main() { fmt.Printf("%x\n", []byte(embedstring.Audit())) }
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
			"embedded bytes differ\nTypeScript: %q\nGo: %q",
			targetOutput,
			goOutput,
		)
	}
}
