package pointer_test

import (
	"context"
	"fmt"
	"go/ast"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/tsoniclang/gotots/internal/emit"
	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/load"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func TestPointerCopyAndOrdinaryLocalsUseOnlyRequiredStorage(t *testing.T) {
	loaded := loadScalarPointerProject(t)
	emission := compileScalarPointerProject(t, loaded)
	source := targetFileByPathSuffix(t, emission, "/source.ts").SourceFile()

	newValue := targetFunction(t, source, "NewValue")
	if _, ok := newValue.Type().(tsgo.UnionTypeNode); !ok {
		t.Fatalf("NewValue result = %T, want pointer union", newValue.Type())
	}
	newBody := newValue.Body().(tsgo.Block).Statements()
	declaration := newBody[0].(tsgo.VariableStatement).
		DeclarationList().Declarations()[0]
	created, ok := declaration.Initializer().(tsgo.CallExpression)
	if !ok {
		t.Fatalf("new pointer initializer = %T, want CallExpression", declaration.Initializer())
	}
	callee, ok := created.Expression().(tsgo.PropertyAccessExpression)
	if !ok ||
		callee.Expression().(tsgo.Identifier).Text() != "GoPointer" ||
		callee.Name().(tsgo.Identifier).Text() != "cell" {
		t.Fatalf("pointer constructor = %T, want GoPointer.cell", created.Expression())
	}
	if len(created.TypeArguments()) != 2 || len(created.Arguments()) != 1 {
		t.Fatalf(
			"pointer construction = %d type arguments, %d values; want two and one",
			len(created.TypeArguments()),
			len(created.Arguments()),
		)
	}
	store := newBody[1].(tsgo.ExpressionStatement).Expression().(tsgo.BinaryExpression)
	assertPointerCellAccess(t, store.Left(), "pointer")

	read := targetReturn(t, targetFunction(t, source, "Read"))
	assertPointerCellAccess(t, read.Expression(), "pointer")

	alias := targetFunction(t, source, "Alias").Body().(tsgo.Block).Statements()
	aliasDeclaration := alias[1].(tsgo.VariableStatement).
		DeclarationList().Declarations()[0]
	aliasReference, ok := aliasDeclaration.Initializer().(tsgo.Identifier)
	if !ok || aliasReference.Text() != "original" {
		t.Fatalf(
			"pointer copy = %T, want direct original reference",
			aliasDeclaration.Initializer(),
		)
	}
	if call, wrapped := aliasDeclaration.Initializer().(tsgo.CallExpression); wrapped {
		callee, selected := call.Expression().(tsgo.PropertyAccessExpression)
		if selected &&
			callee.Expression().(tsgo.Identifier).Text() == "GoPointer" &&
			callee.Name().(tsgo.Identifier).Text() == "cell" {
			t.Fatal("pointer copy allocated a fresh cell")
		}
	}

	ordinary := targetFunction(t, source, "Ordinary")
	for _, statement := range ordinary.Body().(tsgo.Block).Statements() {
		variables, ok := statement.(tsgo.VariableStatement)
		if !ok {
			continue
		}
		for _, declaration := range variables.DeclarationList().Declarations() {
			if call, wrapped := declaration.Initializer().(tsgo.CallExpression); wrapped {
				callee, selected := call.Expression().(tsgo.PropertyAccessExpression)
				if selected &&
					callee.Expression().(tsgo.Identifier).Text() == "GoPointer" &&
					callee.Name().(tsgo.Identifier).Text() == "cell" {
					t.Fatal("pointer support wrapped an unrelated ordinary local")
				}
			}
		}
	}
}

func TestScalarPointersPrintTypecheckAndExecuteDifferentially(t *testing.T) {
	loaded := loadScalarPointerProject(t)
	workingDirectory := t.TempDir()
	artifacts := materializeExportedProgram(t, loaded, workingDirectory)
	sourcePath := filepath.Join(
		workingDirectory,
		filepath.FromSlash(strings.TrimPrefix(artifacts.module(t, "source.ts"), "./")),
	)
	sourcePath = strings.TrimSuffix(sourcePath, ".js") + ".ts"
	printed, err := os.ReadFile(sourcePath)
	if err != nil {
		t.Fatal(err)
	}
	target := string(printed)
	for _, required := range []string{
		"GoPointer.cell<int32, int32>(0)",
		"GoPointer.dereference<int32, int32>(pointer).value",
		"GoPointer.equal(original, alias)",
		"!(original === undefined)",
		"let assigned: GoPointer<int32, int32> | undefined",
	} {
		if !strings.Contains(target, required) {
			t.Fatalf("pointer artifact lacks %q:\n%s", required, target)
		}
	}
	for _, forbidden := range []string{
		"any",
		"unknown",
		".call(",
		".apply(",
		".bind(",
		"!.",
		"new GoPointer<int32, int32>(original)",
	} {
		if strings.Contains(target, forbidden) {
			t.Fatalf("pointer artifact contains %q:\n%s", forbidden, target)
		}
	}

	runnerPath := filepath.Join(workingDirectory, "runner.ts")
	writeFile(t, runnerPath, `import {
    Alias,
    Bool,
    Distinct,
    IsNil,
    NewValue,
    NewZero,
    NilRead,
    Read,
    Reset,
    SetShared,
    SharedIsNil,
    SharedValue,
    Wide,
    Zero,
} from "`+artifacts.module(t, "source.ts")+`";

console.log(NewZero());
console.log(Bool(true));
console.log(Wide(43));
console.log(...Alias(37));
console.log(IsNil(Zero()));
console.log(Reset(NewValue(1)));
console.log(Distinct());
console.log(SharedIsNil());
const pointer = NewValue(41);
SetShared(pointer);
console.log(SharedValue());
console.log(Read(pointer));
try {
    NilRead();
    console.log("nil-succeeded");
} catch {
    console.log("nil-failed");
}
`)
	typeScriptOutput := executeMaterializedTypeScript(
		t,
		workingDirectory,
		artifacts,
		runnerPath,
	)
	goOutput := executeScalarPointerGo(t, workingDirectory)
	if typeScriptOutput != goOutput {
		t.Fatalf("TypeScript output = %q, Go output = %q", typeScriptOutput, goOutput)
	}
}

func TestScalarPointerRuntimeRequestExactJoinsOneDefinition(t *testing.T) {
	loaded := loadScalarPointerProject(t)
	emission := compileScalarPointerProject(t, loaded)
	runtimeFiles := 0
	runtimeClasses := 0
	for _, file := range emission.Files() {
		if file.OutputPath() != "runtime/pointer.ts" {
			continue
		}
		runtimeFiles++
		for _, statement := range file.SourceFile().Statements() {
			class, ok := statement.(tsgo.ClassDeclaration)
			if ok && class.Name().Text() == "GoPointer" {
				runtimeClasses++
			}
		}
	}
	if runtimeFiles != 1 || runtimeClasses != 1 {
		t.Fatalf(
			"pointer runtime = %d files / %d definitions, want 1 / 1",
			runtimeFiles,
			runtimeClasses,
		)
	}
}

func TestScalarPointerTypeEvidenceMutationIsObserved(t *testing.T) {
	loaded := loadScalarPointerProject(t)
	nilRead := sourceFunction(t, loaded.Files()[0].Syntax(), "NilRead")
	dereference := nilRead.Body.List[1].(*ast.ReturnStmt).Results[0].(*ast.StarExpr)
	delete(loaded.TypesInfo().Types, dereference.X)
	delete(loaded.TypesInfo().Uses, dereference.X.(*ast.Ident))
	err := compileLoadedPackage(t, loaded)
	assertUnsupported(
		t,
		err,
		api.RoleReturnResult,
		api.CategoryExpression,
		"*ast.StarExpr",
	)
}

func TestScalarPointerUseSitesScaleLinearly(t *testing.T) {
	counts := []int{4, 8, 16}
	sourceBytes := make([]int, len(counts))
	targetBytes := make([]int, len(counts))
	for index, count := range counts {
		source, target := compilePointerScaling(t, count)
		sourceBytes[index] = len(source)
		targetBytes[index] = len(target)
		if strings.Count(target, "GoPointer.cell<int32, int32>(0)") != count {
			t.Fatalf(
				"pointer constructors = %d, want %d",
				strings.Count(target, "GoPointer.cell<int32, int32>(0)"),
				count,
			)
		}
		if strings.Count(target, "export class GoPointer") != 1 {
			t.Fatal("pointer runtime definition count changed with use sites")
		}
	}
	assertDoublingDeltas(t, "pointer Go source bytes", sourceBytes)
	assertDoublingDeltas(t, "pointer TypeScript bytes", targetBytes)
	t.Logf(
		"pointer scaling counts=%v source-bytes=%v target-bytes=%v",
		counts,
		sourceBytes,
		targetBytes,
	)
}

func loadScalarPointerProject(t *testing.T) *load.Package {
	t.Helper()
	loaded, err := load.One(context.Background(), load.Request{
		Directory: scalarPointerProjectDirectory(),
		Pattern:   ".",
	})
	if err != nil {
		t.Fatal(err)
	}
	return loaded
}

func compileScalarPointerProject(
	t *testing.T,
	loaded *load.Package,
) emit.ProgramEmission {
	t.Helper()
	roots, err := emit.ExportedAPIRoots(loaded)
	if err != nil {
		t.Fatal(err)
	}
	emission, err := emit.Compile(loaded.Program(), roots)
	if err != nil {
		t.Fatal(err)
	}
	return emission
}

func targetFileByPathSuffix(
	t *testing.T,
	emission emit.ProgramEmission,
	suffix string,
) emit.TargetFile {
	t.Helper()
	for _, file := range emission.Files() {
		if strings.HasSuffix(file.OutputPath(), suffix) {
			return file
		}
	}
	t.Fatalf("target file ending in %q is absent", suffix)
	return emit.TargetFile{}
}

func assertPointerCellAccess(
	t *testing.T,
	expression tsgo.Expression,
	pointerName string,
) {
	t.Helper()
	access, ok := expression.(tsgo.PropertyAccessExpression)
	if !ok || access.Name().(tsgo.Identifier).Text() != "value" {
		t.Fatalf("pointer access = %T, want .value", expression)
	}
	call, ok := access.Expression().(tsgo.CallExpression)
	if !ok {
		t.Fatalf("pointer receiver = %T, want guard call", access.Expression())
	}
	callee, ok := call.Expression().(tsgo.PropertyAccessExpression)
	if !ok ||
		callee.Expression().(tsgo.Identifier).Text() != "GoPointer" ||
		callee.Name().(tsgo.Identifier).Text() != "dereference" ||
		len(call.TypeArguments()) != 2 ||
		len(call.Arguments()) != 1 {
		t.Fatalf("pointer guard = %T, want GoPointer.dereference", call.Expression())
	}
	identifier, ok := call.Arguments()[0].(tsgo.Identifier)
	if !ok || identifier.Text() != pointerName {
		t.Fatalf("pointer guard argument = %T, want %s", call.Arguments()[0], pointerName)
	}
}

func executeScalarPointerGo(t *testing.T, workingDirectory string) string {
	t.Helper()
	modulePath, err := filepath.Abs(scalarPointerProjectDirectory())
	if err != nil {
		t.Fatal(err)
	}
	runnerDirectory := filepath.Join(workingDirectory, "go-runner")
	writeFile(t, filepath.Join(runnerDirectory, "go.mod"), fmt.Sprintf(`module example.com/runner

go 1.26.4

require example.com/scalarpointer v0.0.0

replace example.com/scalarpointer => %s
`, filepath.ToSlash(modulePath)))
	writeFile(t, filepath.Join(runnerDirectory, "main.go"), `package main

import (
    "fmt"

    pointer "example.com/scalarpointer"
)

func nilResult() (result string) {
    defer func() {
        if recover() != nil {
            result = "nil-failed"
        }
    }()
    pointer.NilRead()
    return "nil-succeeded"
}

func main() {
    fmt.Println(pointer.NewZero())
    fmt.Println(pointer.Bool(true))
    fmt.Println(pointer.Wide(43))
    fmt.Println(pointer.Alias(37))
    fmt.Println(pointer.IsNil(pointer.Zero()))
    fmt.Println(pointer.Reset(pointer.NewValue(1)))
    fmt.Println(pointer.Distinct())
    fmt.Println(pointer.SharedIsNil())
    value := pointer.NewValue(41)
    pointer.SetShared(value)
    fmt.Println(pointer.SharedValue())
    fmt.Println(pointer.Read(value))
    fmt.Println(nilResult())
}
`)
	return run(
		t,
		runnerDirectory,
		filepath.Join(runtime.GOROOT(), "bin", "go"),
		"run",
		".",
	)
}

func compilePointerScaling(t *testing.T, count int) (string, string) {
	t.Helper()
	directory := t.TempDir()
	writeFile(
		t,
		filepath.Join(directory, "go.mod"),
		"module example.com/pointerscaling\n\ngo 1.26.4\n",
	)
	var source strings.Builder
	source.WriteString("package pointerscaling\n\n")
	for index := range count {
		fmt.Fprintf(
			&source,
			"func Pointer%d(value int32) int32 { pointer := new(int32); *pointer = value; return *pointer }\n",
			index,
		)
	}
	writeFile(t, filepath.Join(directory, "source.go"), source.String())
	loaded, err := load.One(context.Background(), load.Request{
		Directory: directory,
		Pattern:   ".",
	})
	if err != nil {
		t.Fatal(err)
	}
	workingDirectory := t.TempDir()
	artifacts := materializeExportedProgram(t, loaded, workingDirectory)
	var target strings.Builder
	for _, path := range artifacts.targetPaths {
		printed, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		target.Write(printed)
	}
	return source.String(), target.String()
}

func scalarPointerProjectDirectory() string {
	return filepath.Join(
		repositoryRoot(),
		"testdata",
		"constructs",
		"value",
		"pointer",
		"scalar",
	)
}
