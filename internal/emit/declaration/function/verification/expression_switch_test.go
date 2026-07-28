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

func TestExpressionSwitchPrintsTypechecksAndExecutesDifferentially(t *testing.T) {
	loaded := loadExpressionSwitchProject(t)
	workingDirectory := t.TempDir()
	outputPath := filepath.Join(workingDirectory, "expression-switch.ts")
	targetFile := emitExpressionSwitch(t, loaded)
	printed := printTargetFile(t, targetFile, workingDirectory)

	expected, err := os.ReadFile(filepath.Join(expressionSwitchProjectDirectory(), "expected.ts"))
	if err != nil {
		t.Fatal(err)
	}
	if printed != string(expected) {
		t.Fatalf("printed TypeScript:\n%s\nwant:\n%s", printed, expected)
	}
	writeFile(t, outputPath, printed)

	goOutput := executeExpressionSwitchGo(t, workingDirectory)
	typeScriptOutput := executeExpressionSwitchTypeScript(t, loaded, workingDirectory)
	if typeScriptOutput != goOutput {
		t.Fatalf("TypeScript output = %q, Go output = %q", typeScriptOutput, goOutput)
	}
}

func TestExpressionSwitchCreatesScopedExactTargetTree(t *testing.T) {
	loaded := loadExpressionSwitchProject(t)
	targetFile := emitExpressionSwitch(t, loaded)
	function := targetFile.Statements()[1].(tsgo.FunctionDeclaration)
	body := function.Body().(tsgo.Block).Statements()
	if len(body) != 3 {
		t.Fatalf("function statements = %d, want declaration, switch scope, return", len(body))
	}
	scope, ok := body[1].(tsgo.Block)
	if !ok {
		t.Fatalf("switch owner = %T, want scoped block", body[1])
	}
	scoped := scope.Statements()
	if len(scoped) != 2 {
		t.Fatalf("switch scope statements = %d, want initializer and switch", len(scoped))
	}
	targetSwitch, ok := scoped[1].(tsgo.SwitchStatement)
	if !ok {
		t.Fatalf("scoped statement = %T, want switch", scoped[1])
	}
	clauses := targetSwitch.CaseBlock().Clauses()
	if len(clauses) != 4 {
		t.Fatalf("target clauses = %d, want 4 labels", len(clauses))
	}
	if len(clauses[1].(tsgo.CaseClause).Statements()) != 0 {
		t.Fatal("first shared-body label unexpectedly owns statements")
	}
	for _, index := range []int{0, 2, 3} {
		var statements []tsgo.Statement
		switch clause := clauses[index].(type) {
		case tsgo.CaseClause:
			statements = clause.Statements()
		case tsgo.DefaultClause:
			statements = clause.Statements()
		default:
			t.Fatalf("clause %d = %T", index, clauses[index])
		}
		if len(statements) != 1 {
			t.Fatalf("clause %d statements = %d, want one lexical block", index, len(statements))
		}
		clauseBlock := statements[0].(tsgo.Block).Statements()
		if clauseBlock[len(clauseBlock)-1].Kind() != tsgo.SyntaxKindBreakStatement {
			t.Fatalf("clause %d does not end with synthesized break", index)
		}
	}
}

func TestExpressionSwitchChildBoundaryMutationsFailClosed(t *testing.T) {
	for _, testCase := range []struct {
		name      string
		mutate    func(*ast.SwitchStmt)
		category  api.Category
		construct string
		role      api.Role
	}{
		{
			name: "initializer",
			mutate: func(source *ast.SwitchStmt) {
				source.Init = &ast.ReturnStmt{}
			},
			category:  api.CategoryStatement,
			construct: "*ast.ReturnStmt",
			role:      api.RoleSwitchInitializer,
		},
		{
			name: "clause",
			mutate: func(source *ast.SwitchStmt) {
				source.Body.List[0] = &ast.ReturnStmt{}
			},
			category:  api.CategoryStatement,
			construct: "*ast.ReturnStmt",
			role:      api.RoleSwitchClause,
		},
		{
			name: "case expression",
			mutate: func(source *ast.SwitchStmt) {
				clause := source.Body.List[0].(*ast.CaseClause)
				clause.List[0] = &ast.Ident{Name: "untypedMutation"}
			},
			category:  api.CategoryExpression,
			construct: "*ast.Ident",
			role:      api.RoleSwitchCaseExpression,
		},
		{
			name: "fallthrough",
			mutate: func(source *ast.SwitchStmt) {
				clause := source.Body.List[0].(*ast.CaseClause)
				clause.Body[1] = &ast.BranchStmt{Tok: token.FALLTHROUGH}
			},
			category:  api.CategoryStatement,
			construct: "*ast.BranchStmt",
			role:      api.RoleSwitchCaseStatement,
		},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			loaded := loadExpressionSwitchProject(t)
			function := loaded.Files()[0].Syntax().Decls[0].(*ast.FuncDecl)
			source := function.Body.List[1].(*ast.SwitchStmt)
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

func loadExpressionSwitchProject(t *testing.T) *load.Package {
	t.Helper()
	loaded, err := load.One(context.Background(), load.Request{
		Directory: expressionSwitchProjectDirectory(),
		Pattern:   ".",
	})
	if err != nil {
		t.Fatal(err)
	}
	return loaded
}

func emitExpressionSwitch(
	t *testing.T,
	loaded *load.Package,
) tsgo.SourceFile {
	t.Helper()
	return compileSourceFile(t, loaded, loaded.Files()[0].Syntax())
}

func executeExpressionSwitchGo(t *testing.T, workingDirectory string) string {
	t.Helper()
	modulePath, err := filepath.Abs(expressionSwitchProjectDirectory())
	if err != nil {
		t.Fatal(err)
	}
	runnerDirectory := filepath.Join(workingDirectory, "go-runner")
	if err := os.MkdirAll(runnerDirectory, 0o755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(runnerDirectory, "go.mod"), fmt.Sprintf(`module example.com/runner

go 1.26.4

require example.com/expressionswitch v0.0.0

replace example.com/expressionswitch => %s
`, filepath.ToSlash(modulePath)))
	writeFile(t, filepath.Join(runnerDirectory, "main.go"), `package main

import (
	"fmt"

	expressionswitch "example.com/expressionswitch"
)

func main() {
	fmt.Println(expressionswitch.Classify(0))
	fmt.Println(expressionswitch.Classify(1))
	fmt.Println(expressionswitch.Classify(2))
	fmt.Println(expressionswitch.Classify(9))
	fmt.Println(expressionswitch.Classify(2147483647))
	fmt.Println(expressionswitch.Classify(-2147483648))
}
`)
	return run(t, runnerDirectory, filepath.Join(runtime.GOROOT(), "bin", "go"), "run", ".")
}

func executeExpressionSwitchTypeScript(
	t *testing.T,
	loaded *load.Package,
	workingDirectory string,
) string {
	t.Helper()
	artifacts := materializeExportedProgram(t, loaded, workingDirectory)
	runnerPath := filepath.Join(workingDirectory, "runner.ts")
	writeFile(t, runnerPath, `import { Classify } from "`+artifacts.module(t, "source.ts")+`";

console.log(Classify(0));
console.log(Classify(1));
console.log(Classify(2));
console.log(Classify(9));
console.log(Classify(2147483647));
console.log(Classify(-2147483648));
`)
	return executeMaterializedTypeScript(t, workingDirectory, artifacts, runnerPath)
}

func expressionSwitchProjectDirectory() string {
	return filepath.Join(repositoryRoot(), "testdata", "projects", "expression-switch")
}
