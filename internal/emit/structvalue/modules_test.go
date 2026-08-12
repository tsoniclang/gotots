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
	hidden int32
	X      int32
}

func Seed(x, hidden int32) Point { return Point{X: x, hidden: hidden} }
func Hidden(point Point) int32 { return point.hidden }

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
	return Box{Point: model.Seed(value, 3)}
}

func NewWithZeroHidden(value int32) Box {
	return Box{Point: model.Point{X: value}}
}

func Move(box Box, next int32) Box {
	box.Point = box.Point.WithX(next)
	return box
}

func Run() int32 {
	original := New(4)
	changed := Move(original, 9)
	zeroed := NewWithZeroHidden(2)
	return changed.Point.X*10000 + original.Point.X*1000 +
		model.Hidden(changed.Point)*100 + model.Hidden(original.Point)*10 +
		model.Hidden(zeroed.Point)
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
	modelCopyMethods := 0
	var modelConstructorFields []string
	apiPointDefinitions := 0
	apiPointImports := 0
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
		for _, statement := range file.SourceFile().Statements() {
			switch statement := statement.(type) {
			case tsgo.ClassDeclaration:
				if statement.Name().Text() != "Point" {
					continue
				}
				if file.PackageName() == "model" {
					for _, member := range statement.Members() {
						if constructor, ok := member.(tsgo.ConstructorDeclaration); ok {
							input := constructor.Parameters()[0].Type().(tsgo.TypeLiteralNode)
							modelConstructorFields = typeLiteralMemberNames(input)
						}
						method, ok := member.(tsgo.MethodDeclaration)
						if ok && targetName(method.Name()) == "$copy" {
							modelCopyMethods++
						}
					}
				}
				if file.PackageName() == "api" {
					apiPointDefinitions++
				}
			case tsgo.ImportDeclaration:
				if file.PackageName() != "api" ||
					statement.ImportClause() == nil {
					continue
				}
				bindings, ok := statement.ImportClause().
					NamedBindings().(tsgo.NamedImports)
				if !ok {
					continue
				}
				for _, binding := range bindings.Elements() {
					exported := binding.Name().Text()
					if binding.PropertyName() != nil {
						exported = binding.PropertyName().(tsgo.Identifier).Text()
					}
					if exported == "Point" {
						apiPointImports++
					}
				}
			}
		}
		if file.Kind() == emit.TargetFileSource &&
			file.PackageName() == "api" {
			if strings.Count(printed, "export class Point") != 0 {
				t.Fatal("api module does not reference the one model-owned Point definition")
			}
			if !strings.Contains(printed, ".$zero()") ||
				strings.Contains(printed, ".$make(") {
				t.Fatal("external inaccessible-field composite bypassed zero-then-visible-field construction")
			}
		}
		if file.Kind() == emit.TargetFilePackageAssembly &&
			file.PackageName() == "api" {
			apiModule = "./" +
				strings.TrimSuffix(file.OutputPath(), ".ts") +
				".js"
		}
	}
	if modelCopyMethods != 1 ||
		apiPointDefinitions != 0 ||
		apiPointImports != 1 ||
		fmt.Sprint(modelConstructorFields) != "[hidden X]" {
		t.Fatalf(
			"Point ownership = copy methods %d, api definitions %d, api imports %d, constructor fields %v; want 1/0/1/[hidden X]",
			modelCopyMethods,
			apiPointDefinitions,
			apiPointImports,
			modelConstructorFields,
		)
	}
	if apiModule == "" {
		t.Fatal("cross-package emission has no api module")
	}
	runnerPath := filepath.Join(workingDirectory, "runner.ts")
	writeProgramFile(t, runnerPath, `import "./program.js";
import { Run } from "`+apiModule+`";

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
