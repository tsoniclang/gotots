package structvalue_test

import (
	"context"
	"fmt"
	"path/filepath"
	"testing"

	"github.com/tsoniclang/gotots/internal/emit"
	"github.com/tsoniclang/gotots/internal/load"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func TestOrdinaryStructUsesNamedConstructionWithoutUniversalFactory(
	t *testing.T,
) {
	_, emission := compileValueSourceProgram(t, `package valuesource

type Record struct {
	First int32
	Second bool
}

func Build(first int32, second bool) Record {
	return Record{Second: second, First: first}
}
`)
	source := structTargetSource(t, emission)
	class := targetClass(t, source, "Record")
	var properties []string
	var constructor tsgo.ConstructorDeclaration
	for _, member := range class.Members() {
		switch selected := member.(type) {
		case tsgo.PropertyDeclaration:
			if targetName(selected.Name()) != "$goType" {
				properties = append(properties, targetName(selected.Name()))
			}
		case tsgo.ConstructorDeclaration:
			constructor = selected
		case tsgo.MethodDeclaration:
			t.Fatalf("ordinary Record gained support method %q", targetName(selected.Name()))
		}
	}
	if fmt.Sprint(properties) != "[First Second]" {
		t.Fatalf("Record properties = %v, want named declaration fields", properties)
	}
	if constructor == nil || len(constructor.Parameters()) != 1 {
		t.Fatalf("Record constructor = %T with %d parameters, want one named object", constructor, constructorParameterCount(constructor))
	}
	parameterType, ok := constructor.Parameters()[0].Type().(tsgo.TypeLiteralNode)
	if !ok {
		t.Fatalf("Record constructor parameter = %T, want named object type", constructor.Parameters()[0].Type())
	}
	if got := typeLiteralMemberNames(parameterType); fmt.Sprint(got) != "[First Second]" {
		t.Fatalf("Record constructor fields = %v, want declaration names", got)
	}

	statements := targetFunction(t, source, "Build").Body().(tsgo.Block).Statements()
	if len(statements) != 1 {
		t.Fatalf("Build statements = %d, want one direct return", len(statements))
	}
	result, ok := statements[0].(tsgo.ReturnStatement).Expression().(tsgo.NewExpression)
	if !ok {
		t.Fatalf("Build result = %T, want direct new Record", statements[0].(tsgo.ReturnStatement).Expression())
	}
	if targetName(result.Expression()) != "Record" || len(result.Arguments()) != 1 {
		t.Fatal("Build does not construct Record from one named object")
	}
	object, ok := result.Arguments()[0].(tsgo.ObjectLiteralExpression)
	if !ok {
		t.Fatalf("Build constructor input = %T, want object literal", result.Arguments()[0])
	}
	if got := objectPropertyNames(object); fmt.Sprint(got) != "[Second First]" {
		t.Fatalf("Build property order = %v, want Go source order", got)
	}
}

func TestNamedConstructionOrderMatchesGoWithoutUnneededCaptures(t *testing.T) {
	project, emission := compileValueSourceProgram(t, `package valuesource

var trace int32

type Record struct {
	First int32
	Second int32
}

func mark(value int32) int32 {
	trace = trace*10 + value
	return value
}

func Run() int32 {
	trace = 0
	value := Record{Second: mark(2), First: mark(1)}
	return trace*100 + value.First*10 + value.Second
}
`)
	source := structTargetSource(t, emission)
	run := targetFunction(t, source, "Run")
	statements := run.Body().(tsgo.Block).Statements()
	if len(statements) != 3 {
		t.Fatalf("Run statements = %d, want assignment, direct construction, return", len(statements))
	}
	declaration, ok := statements[1].(tsgo.VariableStatement)
	if !ok {
		t.Fatalf("Run construction = %T, want direct variable declaration", statements[1])
	}
	construction, ok := declaration.DeclarationList().Declarations()[0].Initializer().(tsgo.NewExpression)
	if !ok {
		t.Fatalf("Run initializer = %T, want direct new Record", declaration.DeclarationList().Declarations()[0].Initializer())
	}
	object := construction.Arguments()[0].(tsgo.ObjectLiteralExpression)
	if got := objectPropertyNames(object); fmt.Sprint(got) != "[Second First]" {
		t.Fatalf("Run construction order = %v, want source order", got)
	}

	workingDirectory := t.TempDir()
	targetPaths, module := materializeStructProgramWithGolden(
		t,
		workingDirectory,
		emission,
		false,
	)
	runner := filepath.Join(workingDirectory, "runner.ts")
	writeProgramFile(t, runner, `import { Run } from "`+module+`";
console.log(String(Run()));
`)
	compileStructTypeScript(t, workingDirectory, append(targetPaths, runner))
	targetOutput := runProgram(
		t,
		workingDirectory,
		"node",
		filepath.Join(workingDirectory, "out", "runner.js"),
	)
	goRunner := filepath.Join(workingDirectory, "go-runner-value-source")
	writeProgramFile(t, filepath.Join(goRunner, "go.mod"), `module example.com/runner

go 1.26.4

require example.com/value-source v0.0.0

replace example.com/value-source => `+filepath.ToSlash(project)+"\n")
	writeProgramFile(t, filepath.Join(goRunner, "main.go"), `package main

import (
	"fmt"
	valuesource "example.com/value-source"
)

func main() { fmt.Println(valuesource.Run()) }
`)
	goOutput := runProgram(t, goRunner, "go", "run", ".")
	if targetOutput != goOutput {
		t.Fatalf("named construction output differs: TypeScript %q, Go %q", targetOutput, goOutput)
	}
}

func compileValueSourceProgram(
	t *testing.T,
	source string,
) (string, emit.ProgramEmission) {
	t.Helper()
	directory := t.TempDir()
	writeProgramFile(
		t,
		filepath.Join(directory, "go.mod"),
		"module example.com/value-source\n\ngo 1.26.4\n",
	)
	writeProgramFile(t, filepath.Join(directory, "source.go"), source)
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
	emission, err := emit.Compile(program, roots)
	if err != nil {
		t.Fatal(err)
	}
	return directory, emission
}

func constructorParameterCount(constructor tsgo.ConstructorDeclaration) int {
	if constructor == nil {
		return 0
	}
	return len(constructor.Parameters())
}

func typeLiteralMemberNames(source tsgo.TypeLiteralNode) []string {
	result := make([]string, 0, len(source.Members()))
	for _, member := range source.Members() {
		property, ok := member.(tsgo.PropertySignatureDeclaration)
		if ok {
			result = append(result, targetName(property.Name()))
		}
	}
	return result
}

func objectPropertyNames(source tsgo.ObjectLiteralExpression) []string {
	result := make([]string, 0, len(source.Properties()))
	for _, member := range source.Properties() {
		property, ok := member.(tsgo.PropertyAssignment)
		if ok {
			result = append(result, targetName(property.Name()))
		}
	}
	return result
}
