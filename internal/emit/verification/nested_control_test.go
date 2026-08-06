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

func TestNestedBreakTargetsPrintTypecheckAndExecuteDifferentially(t *testing.T) {
	project := t.TempDir()
	writeProgramFile(t, filepath.Join(project, "go.mod"), `module example.com/nestedcontrol

go 1.26.4
`)
	writeProgramFile(t, filepath.Join(project, "source.go"), `package nestedcontrol

func LoopInsideSwitch(value int) int {
	total := 0
	switch value {
	case 0:
		fallthrough
	case 1:
		for {
			total++
			break
		}
		total += 10
	}
	return total
}

func SwitchInsideLoop() int {
	total := 0
outer:
	for index := 0; index < 2; index++ {
		switch index {
		case 0:
			total++
			break
		}
		total += 10
		if total > 100 {
			break outer
		}
	}
	return total
}

func Audit() (int, int) { return LoopInsideSwitch(0), SwitchInsideLoop() }
`)

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
	if strings.Contains(
		artifacts.printed,
		"break __gotots_control_target_0;\n                }\n                total",
	) {
		t.Fatalf("inner loop break escaped to the outer switch:\n%s", artifacts.printed)
	}

	runner := filepath.Join(workingDirectory, "runner.ts")
	writeProgramFile(t, runner, `import "./program.js";
import { Audit } from "`+sourceModuleForExport(
		t,
		artifacts,
		workingDirectory,
		"Audit",
	)+`";

const values = Audit();
console.log(String(values[0]) + "," + String(values[1]));
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
		`module example.com/nestedcontrol-runner

go 1.26.4

require example.com/nestedcontrol v0.0.0

replace example.com/nestedcontrol => %s
`, filepath.ToSlash(project)))
	writeProgramFile(t, filepath.Join(goRunner, "main.go"), `package main

import (
	"fmt"

	"example.com/nestedcontrol"
)

func main() {
	left, right := nestedcontrol.Audit()
	fmt.Printf("%d,%d\n", left, right)
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
			"nested control differs\nTypeScript: %q\nGo: %q",
			targetOutput,
			goOutput,
		)
	}
}
