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
	"strings"
	"testing"

	"github.com/tsoniclang/gotots/internal/emit"
	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/load"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func TestLoopControlPrintsTypechecksAndExecutesDifferentially(t *testing.T) {
	loaded := loadLoopControlProject(t)
	workingDirectory := t.TempDir()
	outputPath := filepath.Join(workingDirectory, "loop.ts")
	targetFile := emitLoopControl(t, loaded, outputPath)
	printed := printTargetFile(t, targetFile, workingDirectory)

	expected, err := os.ReadFile(filepath.Join(loopControlProjectDirectory(), "expected.ts"))
	if err != nil {
		t.Fatal(err)
	}
	if printed != string(expected) {
		t.Fatalf("printed TypeScript:\n%s\nwant:\n%s", printed, expected)
	}
	writeFile(t, outputPath, printed)
	goOutput := executeLoopControlGo(t, workingDirectory)
	typeScriptOutput := executeLoopControlTypeScript(t, workingDirectory, outputPath)
	if typeScriptOutput != goOutput {
		t.Fatalf("TypeScript output = %q, Go output = %q", typeScriptOutput, goOutput)
	}
}

func TestLoopControlCreatesExactTargetTree(t *testing.T) {
	loaded := loadLoopControlProject(t)
	targetFile := emitLoopControl(t, loaded, filepath.Join(t.TempDir(), "loop.ts"))
	function := targetFile.Statements()[1].(tsgo.FunctionDeclaration)
	statements := function.Body().(tsgo.Block).Statements()
	if len(statements) != 3 {
		t.Fatalf("function statements = %d, want declaration, loop, return", len(statements))
	}
	loop, ok := statements[1].(tsgo.ForStatement)
	if !ok {
		t.Fatalf("statement 1 = %T, want tsgo.ForStatement", statements[1])
	}
	if _, ok := loop.Initializer().(tsgo.VariableDeclarationList); !ok {
		t.Fatalf("loop initializer = %T, want variable declaration list", loop.Initializer())
	}
	if loop.Condition().Kind() != tsgo.SyntaxKindBinaryExpression {
		t.Fatalf("loop condition kind = %d, want binary expression", loop.Condition().Kind())
	}
	if loop.Incrementor().Kind() != tsgo.SyntaxKindPostfixUnaryExpression {
		t.Fatalf("loop incrementor kind = %d, want postfix unary expression", loop.Incrementor().Kind())
	}
	body := loop.Statement().(tsgo.Block)
	firstIf := body.Statements()[0].(tsgo.IfStatement)
	continueStatement := firstIf.ThenStatement().(tsgo.Block).Statements()[0]
	if continueStatement.Kind() != tsgo.SyntaxKindContinueStatement {
		t.Fatalf("continue kind = %d, want continue statement", continueStatement.Kind())
	}
	secondIf := body.Statements()[2].(tsgo.IfStatement)
	breakStatement := secondIf.ThenStatement().(tsgo.Block).Statements()[0]
	if breakStatement.Kind() != tsgo.SyntaxKindBreakStatement {
		t.Fatalf("break kind = %d, want break statement", breakStatement.Kind())
	}
}

func TestLoopControlRejectsBranchOutsideLoopContext(t *testing.T) {
	for _, branchToken := range []token.Token{token.BREAK, token.CONTINUE} {
		t.Run(branchToken.String(), func(t *testing.T) {
			loaded := loadLoopControlProject(t)
			function := loaded.Files()[0].Syntax().Decls[0].(*ast.FuncDecl)
			function.Body.List = append(
				[]ast.Stmt{&ast.BranchStmt{Tok: branchToken}},
				function.Body.List...,
			)

			compiler := emit.New(loaded)
			_, err := compiler.EmitFile(
				loaded.Files()[0].Syntax(),
				filepath.Join(t.TempDir(), "loop.ts"),
			)
			assertUnsupportedStatement(t, err, "*ast.BranchStmt")
		})
	}
}

func TestLoopControlRejectsUnsupportedPostVariant(t *testing.T) {
	loaded := loadLoopControlProject(t)
	function := loaded.Files()[0].Syntax().Decls[0].(*ast.FuncDecl)
	loop := function.Body.List[1].(*ast.ForStmt)
	loop.Post.(*ast.IncDecStmt).Tok = token.ADD

	compiler := emit.New(loaded)
	_, err := compiler.EmitFile(
		loaded.Files()[0].Syntax(),
		filepath.Join(t.TempDir(), "loop.ts"),
	)
	var unsupported *api.UnsupportedError
	if !errors.As(err, &unsupported) {
		t.Fatalf("error = %v, want *api.UnsupportedError", err)
	}
	if unsupported.Category != api.CategoryStatement ||
		unsupported.Construct != "*ast.IncDecStmt" ||
		unsupported.Role != api.RoleForPost {
		t.Fatalf("unsupported error = %#v", unsupported)
	}
}

func TestLoopControlRejectsUnsupportedInitializerVariant(t *testing.T) {
	loaded := loadLoopControlProject(t)
	function := loaded.Files()[0].Syntax().Decls[0].(*ast.FuncDecl)
	loop := function.Body.List[1].(*ast.ForStmt)
	loop.Init.(*ast.AssignStmt).Tok = token.ASSIGN

	compiler := emit.New(loaded)
	_, err := compiler.EmitFile(
		loaded.Files()[0].Syntax(),
		filepath.Join(t.TempDir(), "loop.ts"),
	)
	var unsupported *api.UnsupportedError
	if !errors.As(err, &unsupported) {
		t.Fatalf("error = %v, want *api.UnsupportedError", err)
	}
	if unsupported.Category != api.CategoryStatement ||
		unsupported.Construct != "*ast.AssignStmt" ||
		unsupported.Role != api.RoleForInitializer {
		t.Fatalf("unsupported error = %#v", unsupported)
	}
}

func TestLoopControlRejectsLabeledBranchUntilTargetIdentityExists(t *testing.T) {
	loaded := loadLoopControlProject(t)
	function := loaded.Files()[0].Syntax().Decls[0].(*ast.FuncDecl)
	loop := function.Body.List[1].(*ast.ForStmt)
	breakStatement := loop.Body.List[2].(*ast.IfStmt).Body.List[0].(*ast.BranchStmt)
	breakStatement.Label = &ast.Ident{Name: "outer"}

	compiler := emit.New(loaded)
	_, err := compiler.EmitFile(
		loaded.Files()[0].Syntax(),
		filepath.Join(t.TempDir(), "loop.ts"),
	)
	assertUnsupportedStatement(t, err, "*ast.BranchStmt")
}

func TestLoopControlRejectsUnsupportedConditionOperator(t *testing.T) {
	loaded := loadLoopControlProject(t)
	function := loaded.Files()[0].Syntax().Decls[0].(*ast.FuncDecl)
	loop := function.Body.List[1].(*ast.ForStmt)
	loop.Cond.(*ast.BinaryExpr).Op = token.SHL

	compiler := emit.New(loaded)
	_, err := compiler.EmitFile(
		loaded.Files()[0].Syntax(),
		filepath.Join(t.TempDir(), "loop.ts"),
	)
	var unsupported *api.UnsupportedError
	if !errors.As(err, &unsupported) {
		t.Fatalf("error = %v, want *api.UnsupportedError", err)
	}
	if unsupported.Category != api.CategoryExpression ||
		unsupported.Construct != "*ast.BinaryExpr" ||
		unsupported.Role != api.RoleForCondition {
		t.Fatalf("unsupported error = %#v", unsupported)
	}
}

func TestLoopControlPostUsesGoObjectIdentity(t *testing.T) {
	loaded := loadLoopControlProject(t)
	function := loaded.Files()[0].Syntax().Decls[0].(*ast.FuncDecl)
	loop := function.Body.List[1].(*ast.ForStmt)
	loop.Post.(*ast.IncDecStmt).X.(*ast.Ident).Name = "forgedSourceSpelling"

	targetFile := emitLoopControl(t, loaded, filepath.Join(t.TempDir(), "loop.ts"))
	targetFunction := targetFile.Statements()[1].(tsgo.FunctionDeclaration)
	targetLoop := targetFunction.Body().(tsgo.Block).Statements()[1].(tsgo.ForStatement)
	increment := targetLoop.Incrementor().(tsgo.PostfixUnaryExpression)
	if name := increment.Operand().(tsgo.Identifier).Text(); name != "current" {
		t.Fatalf("post operand = %q, want current", name)
	}
}

func assertUnsupportedStatement(t *testing.T, err error, construct string) {
	t.Helper()
	var unsupported *api.UnsupportedError
	if !errors.As(err, &unsupported) {
		t.Fatalf("error = %v, want *api.UnsupportedError", err)
	}
	if unsupported.Category != api.CategoryStatement ||
		unsupported.Construct != construct ||
		unsupported.Role != api.RoleBlockStatement {
		t.Fatalf("unsupported error = %#v", unsupported)
	}
}

func loadLoopControlProject(t *testing.T) *load.Package {
	t.Helper()
	loaded, err := load.One(context.Background(), load.Request{
		Directory: loopControlProjectDirectory(),
		Pattern:   ".",
	})
	if err != nil {
		t.Fatal(err)
	}
	return loaded
}

func emitLoopControl(
	t *testing.T,
	loaded *load.Package,
	outputPath string,
) tsgo.SourceFile {
	t.Helper()
	compiler := emit.New(loaded)
	targetFile, err := compiler.EmitFile(loaded.Files()[0].Syntax(), outputPath)
	if err != nil {
		t.Fatal(err)
	}
	return targetFile
}

func executeLoopControlGo(t *testing.T, workingDirectory string) string {
	t.Helper()
	modulePath, err := filepath.Abs(loopControlProjectDirectory())
	if err != nil {
		t.Fatal(err)
	}
	runnerDirectory := filepath.Join(workingDirectory, "go-runner")
	if err := os.MkdirAll(runnerDirectory, 0o755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(runnerDirectory, "go.mod"), fmt.Sprintf(`module example.com/runner

go 1.26.4

require example.com/loopcontrol v0.0.0

replace example.com/loopcontrol => %s
`, filepath.ToSlash(modulePath)))
	writeFile(t, filepath.Join(runnerDirectory, "main.go"), `package main

import (
	"fmt"

	loop "example.com/loopcontrol"
)

func main() {
	fmt.Println(loop.Sum(0))
	fmt.Println(loop.Sum(1))
	fmt.Println(loop.Sum(5))
	fmt.Println(loop.Sum(10))
}
`)
	return run(t, runnerDirectory, filepath.Join(runtime.GOROOT(), "bin", "go"), "run", ".")
}

func executeLoopControlTypeScript(
	t *testing.T,
	workingDirectory string,
	outputPath string,
) string {
	t.Helper()
	writeFile(t, filepath.Join(workingDirectory, "package.json"), "{\"type\":\"module\"}\n")
	installTsonicCoreTypes(t, workingDirectory)
	runnerPath := filepath.Join(workingDirectory, "runner.ts")
	writeFile(t, runnerPath, `import { Sum } from "./loop.js";

console.log(Sum(0));
console.log(Sum(1));
console.log(Sum(5));
console.log(Sum(10));
`)
	outputDirectory := filepath.Join(workingDirectory, "out")
	toolPath := strings.TrimSpace(
		run(t, repositoryRoot(), filepath.Join(runtime.GOROOT(), "bin", "go"), "tool", "-n", "tsgo"),
	)
	run(t, workingDirectory,
		toolPath,
		"--target", "es2022",
		"--module", "nodenext",
		"--moduleResolution", "nodenext",
		"--strict",
		"--outDir", outputDirectory,
		outputPath,
		runnerPath,
	)
	return run(t, workingDirectory, "node", filepath.Join(outputDirectory, "runner.js"))
}

func loopControlProjectDirectory() string {
	return filepath.Join(repositoryRoot(), "testdata", "projects", "loop-control")
}
