package function_test

import (
	"context"
	"errors"
	"fmt"
	"go/ast"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tsoniclang/gotots/internal/emit"
	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/load"
)

func TestCallableUnsupportedNeighborsFailAtTypedOwners(t *testing.T) {
	testCases := []struct {
		name      string
		source    string
		role      api.Role
		category  api.Category
		construct string
	}{
		{
			name: "generic declaration",
			source: `package boundary

func Identity[T any](value T) T {
	return value
}
`,
			role:      api.RoleFileDeclaration,
			category:  api.CategoryDeclaration,
			construct: "*ast.FuncDecl",
		},
		{
			name: "expression form new",
			source: `package boundary

func NewValue(value int32) *int32 {
	return new(value)
}
`,
			role:      api.RoleReturnResult,
			category:  api.CategoryExpression,
			construct: "*ast.CallExpr",
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			err := compileTemporaryFunctionSource(t, testCase.source)
			var unsupported *api.UnsupportedError
			if !errors.As(err, &unsupported) {
				t.Fatalf("error = %v, want *api.UnsupportedError", err)
			}
			if unsupported.Role != testCase.role ||
				unsupported.Category != testCase.category ||
				unsupported.Construct != testCase.construct {
				t.Fatalf("unsupported = %#v", unsupported)
			}
		})
	}
}

func TestCallableSyntaxAndTypeMutationsFailAtSignatureOwner(t *testing.T) {
	t.Run("named parameter loses syntax binding", func(t *testing.T) {
		loaded := loadCallableValuesProject(t)
		apply := sourceFunction(t, loaded.Files()[0].Syntax(), "Apply")
		apply.Type.Params.List[0].Names = nil

		err := compileLoadedPackage(t, loaded)
		assertUnsupported(
			t,
			err,
			api.RoleParameterType,
			api.CategoryType,
			"*ast.Field",
		)
	})

	t.Run("result type loses checker fact", func(t *testing.T) {
		loaded := loadCallableValuesProject(t)
		offset := sourceFunction(t, loaded.Files()[0].Syntax(), "Offset")
		resultType := offset.Type.Results.List[0].Type
		delete(loaded.TypesInfo().Types, resultType)

		err := compileLoadedPackage(t, loaded)
		assertUnsupported(
			t,
			err,
			api.RoleResultType,
			api.CategoryType,
			"*ast.Field",
		)
	})

	t.Run("nested result points at outer result", func(t *testing.T) {
		loaded := loadNamedResultsProject(t)
		nested := sourceFunction(t, loaded.Files()[0].Syntax(), "Nested")
		outerName := nested.Type.Results.List[0].Names[0]
		declaration := nested.Body.List[0].(*ast.AssignStmt)
		literal := declaration.Rhs[0].(*ast.FuncLit)
		innerName := literal.Type.Results.List[0].Names[0]
		loaded.TypesInfo().Defs[innerName] = loaded.TypesInfo().Defs[outerName]

		err := compileLoadedPackage(t, loaded)
		assertUnsupported(
			t,
			err,
			api.RoleCallableResult,
			api.CategoryType,
			"*ast.Ident",
		)
	})

	t.Run("capture loses object identity", func(t *testing.T) {
		loaded := loadCallableValuesProject(t)
		offset := sourceFunction(t, loaded.Files()[0].Syntax(), "Offset")
		literal := offset.Body.List[0].(*ast.ReturnStmt).Results[0].(*ast.FuncLit)
		result := literal.Body.List[0].(*ast.ReturnStmt)
		capture := result.Results[0].(*ast.BinaryExpr).Y.(*ast.Ident)
		delete(loaded.TypesInfo().Uses, capture)

		err := compileLoadedPackage(t, loaded)
		assertUnsupported(
			t,
			err,
			api.RoleBinaryRight,
			api.CategoryExpression,
			"*ast.Ident",
		)
	})
}

func compileTemporaryFunctionSource(t *testing.T, source string) error {
	t.Helper()
	directory := t.TempDir()
	writeFile(
		t,
		filepath.Join(directory, "go.mod"),
		"module example.com/callableboundary\n\ngo 1.26.4\n",
	)
	writeFile(t, filepath.Join(directory, "source.go"), source)
	loaded, err := load.One(context.Background(), load.Request{
		Directory: directory,
		Pattern:   ".",
	})
	if err != nil {
		t.Fatal(err)
	}
	return compileLoadedPackage(t, loaded)
}

func compileLoadedPackage(t *testing.T, loaded *load.Package) error {
	t.Helper()
	roots, err := emit.ExportedAPIRoots(loaded)
	if err != nil {
		t.Fatal(err)
	}
	_, err = emit.Compile(loaded.Program(), roots)
	return err
}

func assertUnsupported(
	t *testing.T,
	err error,
	role api.Role,
	category api.Category,
	construct string,
) {
	t.Helper()
	var unsupported *api.UnsupportedError
	if !errors.As(err, &unsupported) {
		t.Fatalf("error = %v, want *api.UnsupportedError", err)
	}
	if unsupported.Role != role ||
		unsupported.Category != category ||
		unsupported.Construct != construct {
		t.Fatalf("unsupported = %#v", unsupported)
	}
}

func TestVariadicCallablesPrintTypecheckAndExecuteDifferentially(t *testing.T) {
	project := loadVariadicProject(t)
	workingDirectory := t.TempDir()
	artifacts := materializeExportedProgram(t, project.loaded, workingDirectory)
	source := readMaterializedSource(t, artifacts, "source.ts")
	for _, forbidden := range []string{".call(", ".apply(", ".bind(", ": any", ": unknown"} {
		if strings.Contains(source, forbidden) {
			t.Fatalf("variadic source contains %q:\n%s", forbidden, source)
		}
	}
	if !strings.Contains(source, "values: RuntimeSlice<int32>") {
		t.Fatalf("variadic signature does not use the runtime slice ABI:\n%s", source)
	}
	if strings.Contains(source, "...values: RuntimeSlice<int32>") {
		t.Fatalf("variadic source used an invalid TypeScript rest parameter:\n%s", source)
	}
	if !strings.Contains(source, "RuntimeSlice.literal<int32>") {
		t.Fatalf("ordinary variadic arguments were not materialized as a runtime slice:\n%s", source)
	}
	if !strings.Contains(source, "RuntimeSlice.nil<int32>") {
		t.Fatalf("empty scalar variadic calls did not preserve a nil slice:\n%s", source)
	}
	for _, helper := range []string{"goSliceLiteralWith", "goSliceNilWith"} {
		if !strings.Contains(source, helper) {
			t.Fatalf("aggregate variadic path does not request %s:\n%s", helper, source)
		}
	}
	runnerPath := filepath.Join(workingDirectory, "runner.ts")
	writeFile(t, runnerPath, `import {
    Use,
    UseAggregate,
    UseAggregateEmpty,
    UseEmpty,
} from "`+artifacts.module(t, "source.ts")+`";

console.log(Use());
console.log(UseAggregate());
console.log(UseEmpty());
console.log(UseAggregateEmpty());
`)
	targetOutput := executeMaterializedTypeScript(t, workingDirectory, artifacts, runnerPath)
	goOutput := runVariadicGo(t, project.directory)
	if targetOutput != goOutput {
		t.Fatalf("variadic TypeScript output = %q, Go output = %q", targetOutput, goOutput)
	}
}

func TestVariadicArgumentsPreserveTheDeclaredSliceParameter(t *testing.T) {
	project := loadVariadicProject(t)
	emission, err := emit.Compile(
		project.loaded.Program(),
		variadicExportedRoots(t, project.loaded),
	)
	if err != nil {
		t.Fatal(err)
	}
	source := ""
	for _, file := range emission.Files() {
		if file.Kind() == emit.TargetFileSource {
			source += printTargetFile(t, file.SourceFile(), t.TempDir())
		}
	}
	if strings.Count(source, "values: RuntimeSlice<int32>") < 3 {
		t.Fatalf("variadic declarations and function types were not represented consistently:\n%s", source)
	}
}

func TestVariadicDeclarationMutationFailsAtSignatureOwner(t *testing.T) {
	project := loadVariadicProject(t)
	sum := sourceFunction(t, project.loaded.Files()[0].Syntax(), "Sum")
	sum.Type.Params.List[len(sum.Type.Params.List)-1].Type = ast.NewIdent("int32")
	_, err := emit.CompileFile(project.loaded, project.loaded.Files()[0].Syntax())
	var unsupported *api.UnsupportedError
	if !errors.As(err, &unsupported) ||
		unsupported.Category != api.CategoryType ||
		unsupported.Role != api.RoleParameterType ||
		unsupported.Construct != "*ast.Field" {
		t.Fatalf("variadic declaration mutation error = %#v, want parameter field unsupported", err)
	}
}

func TestVariadicSpreadMutationFailsAtCallOwner(t *testing.T) {
	project := loadVariadicProject(t)
	forward := sourceFunction(t, project.loaded.Files()[0].Syntax(), "Forward")
	call := forward.Body.List[0].(*ast.ReturnStmt).Results[0].(*ast.CallExpr)
	call.Ellipsis = token.NoPos
	_, err := emit.CompileFile(project.loaded, project.loaded.Files()[0].Syntax())
	var unsupported *api.UnsupportedError
	if !errors.As(err, &unsupported) ||
		unsupported.Category != api.CategoryExpression ||
		unsupported.Role != api.RoleCallArgument {
		t.Fatalf("variadic spread mutation error = %#v, want call argument unsupported", err)
	}
}

type variadicProject struct {
	directory string
	loaded    *load.Package
}

func loadVariadicProject(t *testing.T) variadicProject {
	t.Helper()
	directory := t.TempDir()
	writeFile(t, filepath.Join(directory, "go.mod"), "module example.com/variadic\n\ngo 1.26.4\n")
	writeFile(t, filepath.Join(directory, "source.go"), `package variadic

type Accumulator struct {
	Base int32
}

func Sum(prefix int32, values ...int32) int32 {
	return prefix + values[0] + values[1]
}

func Forward(values ...int32) int32 {
	return Sum(values[0], values[1:]...)
}

func SumAll(values ...int32) int32 {
	return values[0] + values[1]
}

func EmptyVariadic(values ...int32) bool {
	return values == nil
}

func UseEmpty() bool {
	return EmptyVariadic()
}

func Aggregate(values ...Accumulator) int32 {
	return values[0].Base
}

func AggregateNil(values ...Accumulator) bool {
	return values == nil
}

func UseAggregateEmpty() bool {
	return AggregateNil()
}

func UseAggregate() int32 {
	return Aggregate(Accumulator{Base: 7}) + Aggregate([]Accumulator{{Base: 8}}...)
}

func (a Accumulator) Add(values ...int32) int32 {
	return a.Base + values[0]
}

func Apply(transform func(...int32) int32) int32 {
	return transform(3, 4)
}

func Use() int32 {
	accumulator := Accumulator{Base: 10}
	return Sum(1, 2, 3) + Forward(4, 5, 6) + accumulator.Add(7) + Apply(SumAll)
}
`)
	loaded, err := load.One(context.Background(), load.Request{
		Directory: directory,
		Pattern:   ".",
	})
	if err != nil {
		t.Fatal(err)
	}
	return variadicProject{directory: directory, loaded: loaded}
}

func variadicExportedRoots(t *testing.T, loaded *load.Package) []emit.Root {
	t.Helper()
	roots, err := emit.ExportedAPIRoots(loaded)
	if err != nil {
		t.Fatal(err)
	}
	return roots
}

func readMaterializedSource(
	t *testing.T,
	artifacts materializedProgram,
	name string,
) string {
	t.Helper()
	path := ""
	for _, targetPath := range artifacts.targetPaths {
		if filepath.Base(targetPath) == name {
			path = targetPath
			break
		}
	}
	if path == "" {
		t.Fatalf("materialized source %q is absent", name)
	}
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(content)
}

func runVariadicGo(t *testing.T, projectDirectory string) string {
	t.Helper()
	workingDirectory := t.TempDir()
	absPath, err := filepath.Abs(projectDirectory)
	if err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(workingDirectory, "go.mod"), fmt.Sprintf(`module example.com/runner

go 1.26.4

require example.com/variadic v0.0.0

replace example.com/variadic => %s
`, filepath.ToSlash(absPath)))
	writeFile(t, filepath.Join(workingDirectory, "main.go"), `package main

import (
	"fmt"

	"example.com/variadic"
)

func main() {
	fmt.Println(variadic.Use())
	fmt.Println(variadic.UseAggregate())
	fmt.Println(variadic.UseEmpty())
	fmt.Println(variadic.UseAggregateEmpty())
}
`)
	return run(t, workingDirectory, "go", "run", ".")
}
