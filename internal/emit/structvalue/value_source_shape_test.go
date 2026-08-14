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
	var constructor tsgo.ConstructorDeclaration
	for _, member := range class.Members() {
		switch selected := member.(type) {
		case tsgo.PropertyDeclaration:
			name := targetName(selected.Name())
			if name != "$goType" && name != "then" {
				t.Fatalf("Record gained separate field declaration %q", targetName(selected.Name()))
			}
		case tsgo.ConstructorDeclaration:
			constructor = selected
		case tsgo.MethodDeclaration:
			t.Fatalf("ordinary Record gained support method %q", targetName(selected.Name()))
		}
	}
	if constructor == nil || len(constructor.Parameters()) != 2 {
		t.Fatalf("Record constructor = %T with %d parameters, want two source-named parameters", constructor, constructorParameterCount(constructor))
	}
	parameters := []string{
		targetName(constructor.Parameters()[0].Name()),
		targetName(constructor.Parameters()[1].Name()),
	}
	if fmt.Sprint(parameters) != "[First Second]" {
		t.Fatalf("Record constructor parameters = %v, want declaration names", parameters)
	}

	statements := targetFunction(t, source, "Build").Body().(tsgo.Block).Statements()
	if len(statements) != 1 {
		t.Fatalf("Build statements = %d, want one direct return", len(statements))
	}
	result, ok := statements[0].(tsgo.ReturnStatement).Expression().(tsgo.NewExpression)
	if !ok {
		t.Fatalf("Build result = %T, want direct new Record", statements[0].(tsgo.ReturnStatement).Expression())
	}
	if targetName(result.Expression()) != "Record" || len(result.Arguments()) != 2 {
		t.Fatal("Build does not construct Record through its direct constructor")
	}
	if got := []string{
		targetName(result.Arguments()[0]),
		targetName(result.Arguments()[1]),
	}; fmt.Sprint(got) != "[first second]" {
		t.Fatalf("Build arguments = %v, want direct declaration order", got)
	}
}

func TestNamedConstructionOrderMatchesGoWithoutUnneededCaptures(t *testing.T) {
	project, emission := compileValueSourceProgramWithOptions(t, `package valuesource

var trace int32

type Record struct {
	First int32
	Second int32
}

func mark(value int32) int32 {
	trace = trace*10 + value
	return value
}

func Run(__gotots_field_0 int32) int32 {
	trace = 0
	value := Record{Second: mark(2), First: mark(1)}
	return trace*100 + value.First*10 + value.Second
}
`, emit.Options{
		IntegerRepresentation: emit.IntegerRepresentationNumber,
		EvaluationOrder:       emit.EvaluationOrderPreserveGo,
	})
	source := structTargetSource(t, emission)
	run := targetFunction(t, source, "Run")
	statements := run.Body().(tsgo.Block).Statements()
	if len(statements) != 5 {
		t.Fatalf("Run statements = %d, want assignment, two captures, construction, return", len(statements))
	}
	for index, statement := range statements[1:3] {
		declaration, ok := statement.(tsgo.VariableStatement)
		if !ok {
			t.Fatalf("capture %d = %T, want variable statement", index, statement)
		}
		name := targetName(declaration.DeclarationList().Declarations()[0].Name())
		if name == "__gotots_field_0" {
			t.Fatal("generated field capture masks the visible source parameter")
		}
	}
	declaration, ok := statements[3].(tsgo.VariableStatement)
	if !ok {
		t.Fatalf("Run construction = %T, want direct variable declaration", statements[3])
	}
	construction, ok := declaration.DeclarationList().Declarations()[0].Initializer().(tsgo.NewExpression)
	if !ok {
		t.Fatalf("Run initializer = %T, want direct new Record", declaration.DeclarationList().Declarations()[0].Initializer())
	}
	if len(construction.Arguments()) != 2 {
		t.Fatalf("Run constructor arguments = %d, want two", len(construction.Arguments()))
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
console.log(String(Run(0)));
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

func main() { fmt.Println(valuesource.Run(0)) }
`)
	goOutput := runProgram(t, goRunner, "go", "run", ".")
	if targetOutput != goOutput {
		t.Fatalf("named construction output differs: TypeScript %q, Go %q", targetOutput, goOutput)
	}
}

func TestLaterStorageDemandReconstructsEarlierCompositeConstruction(
	t *testing.T,
) {
	_, emission := compileValueSourceProgram(t, `package valuesource

type Record struct { Value int32 }
type RecordPointer *Record

func Build() int32 {
	value := Record{Value: 4}
	pointer := RecordPointer(&value)
	pointer.Value = 9
	return value.Value
}
`)
	source := structTargetSource(t, emission)
	constructor := classConstructor(t, targetClass(t, source, "Record"))
	if len(constructor.Parameters()) != 1 {
		t.Fatalf("storage constructor parameters = %d, want one", len(constructor.Parameters()))
	}
	storageType, ok := constructor.Parameters()[0].Type().(tsgo.TypeReferenceNode)
	if !ok {
		t.Fatalf("storage constructor type = %T, want type reference", constructor.Parameters()[0].Type())
	}
	if got := storageType.TypeName().(tsgo.Identifier).Text(); got != "Record$Storage" {
		t.Fatalf("storage constructor type = %q, want Record$Storage", got)
	}
	statements := targetFunction(t, source, "Build").Body().(tsgo.Block).Statements()
	declaration := statements[0].(tsgo.VariableStatement).
		DeclarationList().Declarations()[0]
	construction, ok := declaration.Initializer().(tsgo.CallExpression)
	if !ok || len(construction.Arguments()) != 1 {
		t.Fatalf("reconstructed composite = %T, want one from-storage argument", declaration.Initializer())
	}
	member, ok := construction.Expression().(tsgo.PropertyAccessExpression)
	if !ok || targetName(member.Expression()) != "Record" ||
		targetName(member.Name()) != "$fromStorage" {
		t.Fatal("reconstructed composite did not use the public storage factory")
	}
	if _, ok := construction.Arguments()[0].(tsgo.ObjectLiteralExpression); !ok {
		t.Fatalf("reconstructed composite argument = %T, want storage object", construction.Arguments()[0])
	}

	workingDirectory := t.TempDir()
	targetPaths, _ := materializeStructProgramWithGolden(
		t,
		workingDirectory,
		emission,
		false,
	)
	compileStructTypeScript(t, workingDirectory, targetPaths)
}

func TestEmptyStorageBackedCompositeUsesPublicFactory(t *testing.T) {
	_, emission := compileValueSourceProgram(t, `package valuesource

type Empty struct{}
type EmptyPointer *Empty

func Build() Empty {
	value := Empty{}
	pointer := EmptyPointer(&value)
	return *pointer
}
`)
	source := structTargetSource(t, emission)
	statements := targetFunction(t, source, "Build").Body().(tsgo.Block).Statements()
	declaration := statements[0].(tsgo.VariableStatement).
		DeclarationList().Declarations()[0]
	construction, ok := declaration.Initializer().(tsgo.CallExpression)
	if !ok || len(construction.Arguments()) != 1 {
		t.Fatalf("empty composite = %T, want one from-storage argument", declaration.Initializer())
	}
	member, ok := construction.Expression().(tsgo.PropertyAccessExpression)
	if !ok || targetName(member.Expression()) != "Empty" ||
		targetName(member.Name()) != "$fromStorage" {
		t.Fatal("empty composite did not use the public storage factory")
	}
	object, ok := construction.Arguments()[0].(tsgo.ObjectLiteralExpression)
	if !ok || len(object.Properties()) != 0 {
		t.Fatalf("empty composite storage = %T, want empty object", construction.Arguments()[0])
	}

	workingDirectory := t.TempDir()
	targetPaths, _ := materializeStructProgramWithGolden(
		t,
		workingDirectory,
		emission,
		false,
	)
	compileStructTypeScript(t, workingDirectory, targetPaths)
}

func compileValueSourceProgram(
	t *testing.T,
	source string,
) (string, emit.ProgramEmission) {
	return compileValueSourceProgramWithOptions(t, source, emit.DefaultOptions())
}

func compileValueSourceProgramWithOptions(
	t *testing.T,
	source string,
	options emit.Options,
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
	emission, err := emit.CompileWithOptions(program, roots, options)
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
