package function_test

import (
	"context"
	"errors"
	"fmt"
	"go/ast"
	"go/token"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/tsoniclang/gotots/internal/emit"
	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/load"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func TestLocalVariablesPrintTypecheckAndExecuteDifferentially(t *testing.T) {
	loaded := loadLocalVariablesProject(t)
	workingDirectory := t.TempDir()
	outputPath := filepath.Join(workingDirectory, "local-variables.ts")
	targetFile := emitLocalVariables(t, loaded)
	printed := printTargetFile(t, targetFile, workingDirectory)

	expected, err := os.ReadFile(filepath.Join(localVariablesProjectDirectory(), "expected.ts"))
	if err != nil {
		t.Fatal(err)
	}
	if printed != string(expected) {
		t.Fatalf("printed TypeScript:\n%s\nwant:\n%s", printed, expected)
	}
	writeFile(t, outputPath, printed)

	goOutput := executeLocalVariablesGo(t, workingDirectory)
	typeScriptOutput := executeLocalVariablesTypeScript(t, loaded, workingDirectory)
	if typeScriptOutput != goOutput {
		t.Fatalf("TypeScript output = %q, Go output = %q", typeScriptOutput, goOutput)
	}
}

func TestLocalVariablesCreateExactScopedTargetTree(t *testing.T) {
	loaded := loadLocalVariablesProject(t)
	targetFile := emitLocalVariables(t, loaded)
	function := targetFunction(t, targetFile, "Compute")
	body := function.Body().(tsgo.Block).Statements()
	if len(body) != 2 {
		t.Fatalf("function statements = %d, want outer declaration and lexical block", len(body))
	}
	outer := body[0].(tsgo.VariableStatement)
	if name := outer.DeclarationList().Declarations()[0].Name().(tsgo.Identifier).Text(); name != "base" {
		t.Fatalf("outer name = %q, want base", name)
	}
	inner := body[1].(tsgo.Block).Statements()
	if len(inner) != 8 {
		t.Fatalf(
			"inner statements = %d, want declarations, parallel assignment, and return",
			len(inner),
		)
	}
	shadow := inner[0].(tsgo.VariableStatement)
	shadowDeclaration := shadow.DeclarationList().Declarations()[0]
	if name := shadowDeclaration.Name().(tsgo.Identifier).Text(); name != "base__shadow_1" {
		t.Fatalf("shadow name = %q, want base__shadow_1", name)
	}
	initializer := shadowDeclaration.Initializer().(tsgo.BinaryExpression)
	if name := initializer.Left().(tsgo.Identifier).Text(); name != "base" {
		t.Fatalf("shadow initializer reference = %q, want outer base", name)
	}
	pair := inner[1].(tsgo.VariableStatement).DeclarationList().Declarations()
	if len(pair) != 2 {
		t.Fatalf("pair declarations = %d, want 2", len(pair))
	}
	unicode := inner[2].(tsgo.VariableStatement).DeclarationList().Declarations()[0]
	if name := unicode.Name().(tsgo.Identifier).Text(); name != "__u3c0_" {
		t.Fatalf("portable Unicode name = %q, want __u3c0_", name)
	}
	for index, expected := range []string{"assignmentValue", "assignmentValue2"} {
		capture := inner[index+3].(tsgo.VariableStatement).
			DeclarationList().
			Declarations()[0]
		if name := capture.Name().(tsgo.Identifier).Text(); name != expected {
			t.Fatalf("capture %d = %q, want %q", index, name, expected)
		}
	}
	lateOuter := targetFunction(t, targetFile, "LateOuter").
		Body().(tsgo.Block).
		Statements()
	earlyName := lateOuter[0].(tsgo.Block).
		Statements()[0].(tsgo.VariableStatement).
		DeclarationList().
		Declarations()[0].
		Name().(tsgo.Identifier).
		Text()
	lateName := lateOuter[1].(tsgo.VariableStatement).
		DeclarationList().
		Declarations()[0].
		Name().(tsgo.Identifier).
		Text()
	if earlyName != "value__shadow_1" || lateName != "value" {
		t.Fatalf("child-before-parent names = %q, %q", earlyName, lateName)
	}
}

func TestLocalVariablesUseGoObjectIdentityAcrossShadowing(t *testing.T) {
	loaded := loadLocalVariablesProject(t)
	function := loaded.Files()[0].Syntax().Decls[0].(*ast.FuncDecl)
	block := function.Body.List[1].(*ast.BlockStmt)
	group := block.List[0].(*ast.DeclStmt).Decl.(*ast.GenDecl)
	shadow := group.Specs[0].(*ast.ValueSpec)
	outerReference := shadow.Values[0].(*ast.BinaryExpr).X.(*ast.Ident)
	outerReference.Name = "forgedSourceSpelling"

	targetFile := emitLocalVariables(t, loaded)
	targetDeclaration := targetFunction(t, targetFile, "Compute")
	targetBlock := targetDeclaration.Body().(tsgo.Block).Statements()[1].(tsgo.Block)
	declaration := targetBlock.Statements()[0].(tsgo.VariableStatement).
		DeclarationList().
		Declarations()[0]
	initializer := declaration.Initializer().(tsgo.BinaryExpression)
	if name := initializer.Left().(tsgo.Identifier).Text(); name != "base" {
		t.Fatalf("shadow initializer reference = %q, want outer base", name)
	}
}

func TestLocalVariablesBoundaryMutationsFailClosed(t *testing.T) {
	for _, testCase := range []struct {
		name      string
		mutate    func(*ast.DeclStmt)
		category  api.Category
		construct string
		role      api.Role
	}{
		{
			name: "declaration token",
			mutate: func(source *ast.DeclStmt) {
				source.Decl.(*ast.GenDecl).Tok = token.CONST
			},
			category:  api.CategoryDeclaration,
			construct: "*ast.Ident",
			role:      api.RoleLocalDeclaration,
		},
		{
			name: "non-value spec",
			mutate: func(source *ast.DeclStmt) {
				source.Decl.(*ast.GenDecl).Specs[0] = &ast.TypeSpec{
					Name: &ast.Ident{Name: "NotAValue"},
				}
			},
			category:  api.CategoryStatement,
			construct: "*ast.TypeSpec",
			role:      api.RoleLocalDeclaration,
		},
		{
			name: "missing type and initializer",
			mutate: func(source *ast.DeclStmt) {
				spec := source.Decl.(*ast.GenDecl).Specs[0].(*ast.ValueSpec)
				spec.Type = nil
				spec.Values = nil
			},
			category:  api.CategoryStatement,
			construct: "*ast.ValueSpec",
			role:      api.RoleLocalDeclaration,
		},
		{
			name: "blank target",
			mutate: func(source *ast.DeclStmt) {
				source.Decl.(*ast.GenDecl).Specs[0].(*ast.ValueSpec).Names[0].Name = "_"
			},
			category:  api.CategoryStatement,
			construct: "*ast.Ident",
			role:      api.RoleLocalDeclaration,
		},
		{
			name: "initializer evidence",
			mutate: func(source *ast.DeclStmt) {
				source.Decl.(*ast.GenDecl).Specs[0].(*ast.ValueSpec).Values[0] =
					&ast.SelectorExpr{
						X:   &ast.Ident{Name: "missing"},
						Sel: &ast.Ident{Name: "Value"},
					}
			},
			category:  api.CategoryExpression,
			construct: "*ast.SelectorExpr",
			role:      api.RoleLocalValue,
		},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			loaded := loadLocalVariablesProject(t)
			function := loaded.Files()[0].Syntax().Decls[0].(*ast.FuncDecl)
			source := function.Body.List[0].(*ast.DeclStmt)
			testCase.mutate(source)

			_, err := emit.CompileFile(loaded, loaded.Files()[0].Syntax())
			var unsupported *api.UnsupportedError
			if !errors.As(err, &unsupported) {
				t.Fatalf("error = %v, want *api.UnsupportedError", err)
			}
			if unsupported.Category != testCase.category ||
				unsupported.Construct != testCase.construct ||
				unsupported.Role != testCase.role {
				t.Fatalf("unsupported error = %#v", unsupported)
			}
		})
	}
}

func loadLocalVariablesProject(t *testing.T) *load.Package {
	t.Helper()
	loaded, err := load.One(context.Background(), load.Request{
		Directory: localVariablesProjectDirectory(),
		Pattern:   ".",
	})
	if err != nil {
		t.Fatal(err)
	}
	return loaded
}

func emitLocalVariables(
	t *testing.T,
	loaded *load.Package,
) tsgo.SourceFile {
	t.Helper()
	return compileSourceFile(t, loaded, loaded.Files()[0].Syntax())
}

func executeLocalVariablesGo(t *testing.T, workingDirectory string) string {
	t.Helper()
	modulePath, err := filepath.Abs(localVariablesProjectDirectory())
	if err != nil {
		t.Fatal(err)
	}
	runnerDirectory := filepath.Join(workingDirectory, "go-runner")
	if err := os.MkdirAll(runnerDirectory, 0o755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(runnerDirectory, "go.mod"), fmt.Sprintf(`module example.com/runner

go 1.26.4

require example.com/localvariables v0.0.0

replace example.com/localvariables => %s
`, filepath.ToSlash(modulePath)))
	writeFile(t, filepath.Join(runnerDirectory, "main.go"), `package main

import (
	"fmt"

	localvariables "example.com/localvariables"
)

func main() {
	fmt.Println(localvariables.Compute(0))
	fmt.Println(localvariables.Compute(5))
	fmt.Println(localvariables.Compute(20))
	fmt.Println(localvariables.LateOuter(0))
	fmt.Println(localvariables.LateOuter(5))
}
`)
	return run(t, runnerDirectory, filepath.Join(runtime.GOROOT(), "bin", "go"), "run", ".")
}

func executeLocalVariablesTypeScript(
	t *testing.T,
	loaded *load.Package,
	workingDirectory string,
) string {
	t.Helper()
	artifacts := materializeExportedProgram(t, loaded, workingDirectory)
	runnerPath := filepath.Join(workingDirectory, "runner.ts")
	writeFile(t, runnerPath, `import { Compute, LateOuter } from "`+
		artifacts.module(t, "source.ts")+`";

console.log(Compute(0));
console.log(Compute(5));
console.log(Compute(20));
console.log(LateOuter(0));
console.log(LateOuter(5));
`)
	return executeMaterializedTypeScript(t, workingDirectory, artifacts, runnerPath)
}

func localVariablesProjectDirectory() string {
	return filepath.Join(repositoryRoot(), "testdata", "projects", "local-variables")
}
