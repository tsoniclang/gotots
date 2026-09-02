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

func ExplicitBreak(value int) int {
	total := 0
	switch value {
	case 0:
		total++
		fallthrough
	case 1:
		total += 10
		break
	}
	return total
}

func ContinueThroughSwitch() int {
	total := 0
	for index := 0; index < 3; index++ {
		switch index {
		case 0:
			fallthrough
		case 1:
			continue
		default:
			total += 10
		}
		total++
	}
	return total
}

func ArraySwitch(value [1]int) int {
	switch value {
	case [1]int{0}:
		fallthrough
	case [1]int{1}:
		return 1
	default:
		return 2
	}
}

func Audit() (int, int, int, int, int) {
	return LoopInsideSwitch(0), SwitchInsideLoop(), ExplicitBreak(0), ContinueThroughSwitch(), ArraySwitch([1]int{0})
}
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
	fallthroughTarget := targetFunctionText(
		t,
		artifacts.printed,
		"LoopInsideSwitch",
	)
	if strings.Contains(fallthroughTarget, "switchMatch") {
		t.Fatalf(
			"primitive fallthrough switch retained conditional matching:\n%s",
			fallthroughTarget,
		)
	}
	if !strings.Contains(fallthroughTarget, "switch (value)") {
		t.Fatalf(
			"primitive fallthrough switch is not native:\n%s",
			fallthroughTarget,
		)
	}
	arrayTarget := targetFunctionText(t, artifacts.printed, "ArraySwitch")
	if !strings.Contains(arrayTarget, "switchMatch") {
		t.Fatalf(
			"custom-equality fallthrough switch lost conditional matching:\n%s",
			arrayTarget,
		)
	}
	if strings.Contains(
		artifacts.printed,
		"break controlTarget;\n                }\n                total",
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
console.log(values.map(String).join(","));
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
	loopInside, switchInside, explicitBreak, continued, arraySwitch := nestedcontrol.Audit()
	fmt.Printf("%d,%d,%d,%d,%d\n", loopInside, switchInside, explicitBreak, continued, arraySwitch)
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
