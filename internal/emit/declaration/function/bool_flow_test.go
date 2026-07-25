package function_test

import (
	"context"
	"errors"
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
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

func TestBoolFlowPrintsTypechecksAndExecutesDifferentially(t *testing.T) {
	loaded := loadBoolFlowProject(t)
	workingDirectory := t.TempDir()
	outputPath := filepath.Join(workingDirectory, "bool-flow.ts")
	targetFile := emitBoolFlow(t, loaded, outputPath)

	client, err := tsgo.StartClient(repositoryRoot(), workingDirectory)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := client.Close(); err != nil {
			t.Errorf("close TS-Go client: %v", err)
		}
	})
	printed, err := client.PrintNode(targetFile, tsgo.PrintOptions{})
	if err != nil {
		t.Fatal(err)
	}
	expected, err := os.ReadFile(filepath.Join(boolFlowProjectDirectory(), "expected.ts"))
	if err != nil {
		t.Fatal(err)
	}
	if printed != string(expected) {
		t.Fatalf("printed TypeScript:\n%s\nwant:\n%s", printed, expected)
	}
	if err := os.WriteFile(outputPath, []byte(printed), 0o644); err != nil {
		t.Fatal(err)
	}

	goOutput := executeBoolFlowGo(t, workingDirectory)
	typeScriptOutput := executeBoolFlowTypeScript(t, workingDirectory, outputPath)
	if typeScriptOutput != goOutput {
		t.Fatalf("TypeScript output = %q, Go output = %q", typeScriptOutput, goOutput)
	}
}

func TestBoolFlowCreatesExactTargetTree(t *testing.T) {
	loaded := loadBoolFlowProject(t)
	targetFile := emitBoolFlow(t, loaded, filepath.Join(t.TempDir(), "bool-flow.ts"))
	statements := targetFile.Statements()
	if len(statements) != 4 {
		t.Fatalf("target statements = %d, want import plus three functions", len(statements))
	}
	runFunction, ok := statements[1].(tsgo.FunctionDeclaration)
	if !ok {
		t.Fatalf("run declaration = %T, want tsgo.FunctionDeclaration", statements[1])
	}
	body := runFunction.Body().(tsgo.Block)
	if len(body.Statements()) != 3 {
		t.Fatalf("Run statements = %d, want 3", len(body.Statements()))
	}
	if _, ok := body.Statements()[0].(tsgo.VariableStatement); !ok {
		t.Fatalf("Run statement 0 = %T, want tsgo.VariableStatement", body.Statements()[0])
	}
	ifStatement, ok := body.Statements()[1].(tsgo.IfStatement)
	if !ok {
		t.Fatalf("Run statement 1 = %T, want tsgo.IfStatement", body.Statements()[1])
	}
	if ifStatement.Expression().Kind() != tsgo.SyntaxKindPrefixUnaryExpression {
		t.Fatalf("if condition kind = %d, want prefix unary", ifStatement.Expression().Kind())
	}
	sameFunction := statements[3].(tsgo.FunctionDeclaration)
	sameReturn := sameFunction.Body().(tsgo.Block).Statements()[0].(tsgo.ReturnStatement)
	equality := sameReturn.Expression().(tsgo.BinaryExpression)
	if equality.OperatorToken().Kind() != tsgo.SyntaxKindEqualsEqualsEqualsToken {
		t.Fatalf("equality operator = %d, want strict equality", equality.OperatorToken().Kind())
	}
}

func TestBoolFlowRejectsCompoundAssignmentVariant(t *testing.T) {
	loaded := loadBoolFlowProject(t)
	runFunction := loaded.Files()[0].Syntax().Decls[0].(*ast.FuncDecl)
	ifStatement := runFunction.Body.List[1].(*ast.IfStmt)
	assignment := ifStatement.Body.List[0].(*ast.AssignStmt)
	assignment.Tok = token.ADD_ASSIGN

	compiler := emit.New(loaded)
	_, err := compiler.EmitFile(loaded.Files()[0].Syntax(), filepath.Join(t.TempDir(), "bool-flow.ts"))
	var unsupported *api.UnsupportedError
	if !errors.As(err, &unsupported) {
		t.Fatalf("error = %v, want *api.UnsupportedError", err)
	}
	if unsupported.Category != api.CategoryStatement ||
		unsupported.Construct != "*ast.AssignStmt" ||
		unsupported.Role != api.RoleBlockStatement {
		t.Fatalf("unsupported error = %#v", unsupported)
	}
}

func TestBoolFlowCallsUseGoObjectIdentity(t *testing.T) {
	loaded := loadBoolFlowProject(t)
	runFunction := loaded.Files()[0].Syntax().Decls[0].(*ast.FuncDecl)
	ifStatement := runFunction.Body.List[1].(*ast.IfStmt)
	assignment := ifStatement.Body.List[0].(*ast.AssignStmt)
	call := assignment.Rhs[0].(*ast.CallExpr)
	call.Fun.(*ast.Ident).Name = "forgedSourceSpelling"

	targetFile := emitBoolFlow(t, loaded, filepath.Join(t.TempDir(), "bool-flow.ts"))
	runTarget := targetFile.Statements()[1].(tsgo.FunctionDeclaration)
	targetIf := runTarget.Body().(tsgo.Block).Statements()[1].(tsgo.IfStatement)
	thenBlock := targetIf.ThenStatement().(tsgo.Block)
	targetAssignment := thenBlock.Statements()[0].(tsgo.ExpressionStatement)
	binary := targetAssignment.Expression().(tsgo.BinaryExpression)
	targetCall := binary.Right().(tsgo.CallExpression)
	callee := targetCall.Expression().(tsgo.Identifier)
	if callee.Text() != "Flip" {
		t.Fatalf("callee = %q, want Flip", callee.Text())
	}
}

func TestBoolFlowLiteralsUseGoObjectIdentity(t *testing.T) {
	loaded := loadBoolFlowProject(t)
	runFunction := loaded.Files()[0].Syntax().Decls[0].(*ast.FuncDecl)
	definition := runFunction.Body.List[0].(*ast.AssignStmt)
	literal := definition.Rhs[0].(*ast.Ident)
	literal.Name = "true"

	targetFile := emitBoolFlow(t, loaded, filepath.Join(t.TempDir(), "bool-flow.ts"))
	runTarget := targetFile.Statements()[1].(tsgo.FunctionDeclaration)
	targetDefinition := runTarget.Body().(tsgo.Block).Statements()[0].(tsgo.VariableStatement)
	initializer := targetDefinition.DeclarationList().Declarations()[0].Initializer()
	if initializer.Kind() != tsgo.SyntaxKindFalseKeyword {
		t.Fatalf("initializer kind = %d, want false keyword", initializer.Kind())
	}
}

func TestBoolFlowRejectsUnaryOperatorVariant(t *testing.T) {
	loaded := loadBoolFlowProject(t)
	runFunction := loaded.Files()[0].Syntax().Decls[0].(*ast.FuncDecl)
	ifStatement := runFunction.Body.List[1].(*ast.IfStmt)
	ifStatement.Cond.(*ast.UnaryExpr).Op = token.ADD

	compiler := emit.New(loaded)
	_, err := compiler.EmitFile(loaded.Files()[0].Syntax(), filepath.Join(t.TempDir(), "bool-flow.ts"))
	var unsupported *api.UnsupportedError
	if !errors.As(err, &unsupported) {
		t.Fatalf("error = %v, want *api.UnsupportedError", err)
	}
	if unsupported.Category != api.CategoryExpression ||
		unsupported.Construct != "*ast.UnaryExpr" ||
		unsupported.Role != api.RoleIfCondition {
		t.Fatalf("unsupported error = %#v", unsupported)
	}
}

func TestBoolFlowRejectsConversionAtOrdinaryCallOwner(t *testing.T) {
	loaded := loadBoolFlowProject(t)
	runFunction := loaded.Files()[0].Syntax().Decls[0].(*ast.FuncDecl)
	ifStatement := runFunction.Body.List[1].(*ast.IfStmt)
	call := ifStatement.Body.List[0].(*ast.AssignStmt).Rhs[0].(*ast.CallExpr)
	callee := call.Fun.(*ast.Ident)
	loaded.TypesInfo().Uses[callee] = types.Universe.Lookup("bool")

	compiler := emit.New(loaded)
	_, err := compiler.EmitFile(loaded.Files()[0].Syntax(), filepath.Join(t.TempDir(), "bool-flow.ts"))
	var unsupported *api.UnsupportedError
	if !errors.As(err, &unsupported) {
		t.Fatalf("error = %v, want *api.UnsupportedError", err)
	}
	if unsupported.Category != api.CategoryExpression ||
		unsupported.Construct != "*ast.CallExpr" ||
		unsupported.Role != api.RoleAssignmentValue {
		t.Fatalf("unsupported error = %#v", unsupported)
	}
}

func loadBoolFlowProject(t *testing.T) *load.Package {
	t.Helper()
	loaded, err := load.One(context.Background(), load.Request{
		Directory: boolFlowProjectDirectory(),
		Pattern:   ".",
	})
	if err != nil {
		t.Fatal(err)
	}
	return loaded
}

func emitBoolFlow(t *testing.T, loaded *load.Package, outputPath string) tsgo.SourceFile {
	t.Helper()
	compiler := emit.New(loaded)
	targetFile, err := compiler.EmitFile(loaded.Files()[0].Syntax(), outputPath)
	if err != nil {
		t.Fatal(err)
	}
	return targetFile
}

func executeBoolFlowGo(t *testing.T, workingDirectory string) string {
	t.Helper()
	modulePath, err := filepath.Abs(boolFlowProjectDirectory())
	if err != nil {
		t.Fatal(err)
	}
	runnerDirectory := filepath.Join(workingDirectory, "go-runner")
	if err := os.MkdirAll(runnerDirectory, 0o755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(runnerDirectory, "go.mod"), fmt.Sprintf(`module example.com/runner

go 1.26.4

require example.com/boolflow v0.0.0

replace example.com/boolflow => %s
`, filepath.ToSlash(modulePath)))
	writeFile(t, filepath.Join(runnerDirectory, "main.go"), `package main

import (
	"fmt"

	flow "example.com/boolflow"
)

func main() {
	fmt.Println(flow.Run(false))
	fmt.Println(flow.Run(true))
	fmt.Println(flow.Flip(false))
	fmt.Println(flow.Same(true, true))
	fmt.Println(flow.Same(true, false))
}
`)
	return run(t, runnerDirectory, filepath.Join(runtime.GOROOT(), "bin", "go"), "run", ".")
}

func executeBoolFlowTypeScript(t *testing.T, workingDirectory, outputPath string) string {
	t.Helper()
	writeFile(t, filepath.Join(workingDirectory, "package.json"), "{\"type\":\"module\"}\n")
	installTsonicCoreTypes(t, workingDirectory)
	runnerPath := filepath.Join(workingDirectory, "runner.ts")
	writeFile(t, runnerPath, `import { Flip, Run, Same } from "./bool-flow.js";

console.log(Run(false));
console.log(Run(true));
console.log(Flip(false));
console.log(Same(true, true));
console.log(Same(true, false));
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

func boolFlowProjectDirectory() string {
	return filepath.Join(repositoryRoot(), "testdata", "projects", "bool-flow")
}
