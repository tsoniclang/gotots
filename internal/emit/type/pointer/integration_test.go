package pointer_test

import (
	"context"
	"fmt"
	"go/ast"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tsoniclang/gotots/internal/emit"
	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/load"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func TestScalarPointersUseCanonicalMarkersWithoutWrappingOrdinaryLocals(t *testing.T) {
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
	callee, ok := created.Expression().(tsgo.Identifier)
	if !ok || callee.Text() != "allocatePointer" {
		t.Fatalf("pointer constructor = %T, want allocatePointer", created.Expression())
	}
	if len(created.TypeArguments()) != 1 || len(created.Arguments()) != 1 {
		t.Fatalf(
			"pointer construction = %d type arguments, %d values; want one and one",
			len(created.TypeArguments()),
			len(created.Arguments()),
		)
	}
	store, ok := newBody[1].(tsgo.ExpressionStatement).
		Expression().(tsgo.CallExpression)
	if !ok {
		t.Fatalf("pointer store = %T, want CallExpression", newBody[1])
	}
	storeCallee, ok := store.Expression().(tsgo.Identifier)
	if !ok || storeCallee.Text() != "storePointer" {
		t.Fatalf("pointer store callee = %T, want storePointer", store.Expression())
	}

	read := targetReturn(t, targetFunction(t, source, "Read"))
	readValue, ok := read.Expression().(tsgo.CallExpression)
	if !ok {
		t.Fatalf("pointer read = %T, want CallExpression", read.Expression())
	}
	readCallee, ok := readValue.Expression().(tsgo.Identifier)
	if !ok || readCallee.Text() != "loadPointer" {
		t.Fatalf("pointer read callee = %T, want loadPointer", readValue.Expression())
	}

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
				callee, selected := call.Expression().(tsgo.Identifier)
				if selected && callee.Text() == "allocatePointer" {
					t.Fatal("pointer support wrapped an unrelated ordinary local")
				}
			}
		}
	}
}

func TestScalarPointerMarkersPrintAndTypecheck(t *testing.T) {
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
		`import type { Pointer } from "@tsonic/core/types.js"`,
		`import { allocatePointer, equalPointer, loadPointer, storePointer } from "@tsonic/core/lang.js"`,
		"allocatePointer<int32>(0)",
		"loadPointer<int32>((pointer ?? GoPanic.raiseRuntime",
		"storePointer((pointer ?? GoPanic.raiseRuntime",
		"equalPointer<int32>(original, alias)",
		"!(original === undefined)",
		"let assigned: Pointer<int32> | undefined",
		"export function Read(pointer: Pointer<int32> | undefined): int32",
	} {
		if !strings.Contains(target, required) {
			t.Fatalf("pointer artifact lacks %q:\n%s", required, target)
		}
	}
	for _, forbidden := range []string{
		": any",
		" as any",
		"unknown",
		".call(",
		".apply(",
		".bind(",
		"!.",
		"GoPointer",
		"goPointer",
	} {
		if strings.Contains(target, forbidden) {
			t.Fatalf("pointer artifact contains %q:\n%s", forbidden, target)
		}
	}

	typecheckMaterializedTypeScript(t, workingDirectory, artifacts)
}

func TestScalarPointerMarkersDoNotRequestAGoToTSPointerRuntime(t *testing.T) {
	loaded := loadScalarPointerProject(t)
	emission := compileScalarPointerProject(t, loaded)
	for _, file := range emission.Files() {
		if file.OutputPath() == "runtime/pointer.ts" {
			t.Fatal("canonical pointer markers requested the retired GoToTS pointer runtime")
		}
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
		if strings.Count(target, "allocatePointer<int32>(0)") != count {
			t.Fatalf(
				"pointer constructors = %d, want %d",
				strings.Count(target, "allocatePointer<int32>(0)"),
				count,
			)
		}
		if strings.Contains(target, "GoPointer") ||
			strings.Contains(target, "runtime/pointer") {
			t.Fatal("pointer scaling restored the retired GoToTS pointer runtime")
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
