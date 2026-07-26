package structvalue_test

import (
	"context"
	"fmt"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/tsoniclang/gotots/internal/emit"
	"github.com/tsoniclang/gotots/internal/load"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func TestNamedStructValuesCrossFilesAndPackagesTypecheckAndExecute(t *testing.T) {
	projectDirectory := t.TempDir()
	writeProgramFile(
		t,
		filepath.Join(projectDirectory, "go.mod"),
		"module example.com/records\n\ngo 1.26.4\n",
	)
	writeProgramFile(t, filepath.Join(projectDirectory, "model", "point.go"), `package model

type Point struct {
	X int32
}

func (point Point) WithX(next int32) Point {
	point.X = next
	return point
}
`)
	writeProgramFile(t, filepath.Join(projectDirectory, "api", "box.go"), `package api

import "example.com/records/model"

type Box struct {
	Point model.Point
}

func New(value int32) Box {
	return Box{Point: model.Point{X: value}}
}

func Move(box Box, next int32) Box {
	box.Point = box.Point.WithX(next)
	return box
}

func Run() int32 {
	original := New(4)
	changed := Move(original, 9)
	return changed.Point.X*10 + original.Point.X
}
`)
	program, err := load.Load(context.Background(), load.Request{
		Directory: projectDirectory,
		Pattern:   "./api",
	})
	if err != nil {
		t.Fatal(err)
	}
	roots, err := emit.ExportedAPIRoots(program.Roots()[0])
	if err != nil {
		t.Fatal(err)
	}
	emission, err := emit.Compile(program, roots)
	if err != nil {
		t.Fatal(err)
	}

	workingDirectory := t.TempDir()
	client, err := tsgo.StartClient(repositoryRoot(), workingDirectory)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := client.Close(); err != nil {
			t.Errorf("close TS-Go client: %v", err)
		}
	})
	var targetPaths []string
	var apiModule string
	for _, file := range emission.Files() {
		printed, err := client.PrintNode(file.SourceFile(), tsgo.PrintOptions{})
		if err != nil {
			t.Fatal(err)
		}
		targetPath := filepath.Join(
			workingDirectory,
			filepath.FromSlash(file.OutputPath()),
		)
		writeProgramFile(t, targetPath, printed)
		targetPaths = append(targetPaths, targetPath)
		if file.PackageName() == "api" {
			if !strings.Contains(printed, `import { Point`) ||
				strings.Count(printed, "export class Point") != 0 {
				t.Fatal("api module does not reference the one model-owned Point definition")
			}
			apiModule = "./" + strings.TrimSuffix(file.OutputPath(), ".ts") + ".js"
		}
	}
	if apiModule == "" {
		t.Fatal("cross-package emission has no api module")
	}
	runnerPath := filepath.Join(workingDirectory, "runner.ts")
	writeProgramFile(t, runnerPath, `import { Run } from "`+apiModule+`";

console.log(Run());
`)
	writeProgramFile(t, filepath.Join(workingDirectory, "package.json"), "{\"type\":\"module\"}\n")
	targetPaths = append(targetPaths, runnerPath)
	compileStructTypeScript(t, workingDirectory, targetPaths)
	targetOutput := runProgram(
		t,
		workingDirectory,
		"node",
		filepath.Join(workingDirectory, "out", "runner.js"),
	)

	goRunner := filepath.Join(workingDirectory, "go-runner")
	writeProgramFile(t, filepath.Join(goRunner, "go.mod"), fmt.Sprintf(`module example.com/runner

go 1.26.4

require example.com/records v0.0.0

replace example.com/records => %s
`, filepath.ToSlash(projectDirectory)))
	writeProgramFile(t, filepath.Join(goRunner, "main.go"), `package main

import (
	"fmt"

	"example.com/records/api"
)

func main() {
	fmt.Println(api.Run())
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
		t.Fatalf("TypeScript output = %q, Go output = %q", targetOutput, goOutput)
	}
}
