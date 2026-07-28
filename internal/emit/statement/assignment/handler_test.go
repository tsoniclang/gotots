package assignment_test

import (
	"context"
	"errors"
	"fmt"
	"go/ast"
	"go/token"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/tsoniclang/gotots/internal/emit"
	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/load"
	"github.com/tsoniclang/gotots/internal/output"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func TestParallelIdentifierAssignmentPrintsTypechecksAndExecutesDifferentially(
	t *testing.T,
) {
	loaded := loadProject(t)
	workingDirectory := t.TempDir()
	outputPath := filepath.Join(workingDirectory, "assignment.ts")
	targetFile := emitProject(t, loaded)

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
	expected, err := os.ReadFile(filepath.Join(projectDirectory(), "expected.ts"))
	if err != nil {
		t.Fatal(err)
	}
	if printed != string(expected) {
		t.Fatalf("printed TypeScript:\n%s\nwant:\n%s", printed, expected)
	}
	writeFile(t, outputPath, printed)

	goOutput := executeGo(t, workingDirectory)
	typeScriptOutput := executeTypeScript(t, loaded, workingDirectory)
	if typeScriptOutput != goOutput {
		t.Fatalf("TypeScript output = %q, Go output = %q", typeScriptOutput, goOutput)
	}
}

func TestParallelAssignmentCreatesCapturesBeforeStores(t *testing.T) {
	loaded := loadProject(t)
	targetFile := emitProject(
		t,
		loaded,
	)
	function := targetFile.Statements()[1].(tsgo.FunctionDeclaration)
	statements := function.Body().(tsgo.Block).Statements()
	if len(statements) != 5 {
		t.Fatalf("SwapLeft statements = %d, want two captures, two stores, return", len(statements))
	}
	for index := 0; index < 2; index++ {
		declaration, ok := statements[index].(tsgo.VariableStatement)
		if !ok {
			t.Fatalf("statement %d = %T, want capture declaration", index, statements[index])
		}
		if declaration.DeclarationList().Flags()&tsgo.NodeFlagsConst == 0 {
			t.Fatalf("capture %d is not const", index)
		}
	}
	firstCapture := statements[0].(tsgo.VariableStatement).
		DeclarationList().Declarations()[0]
	secondCapture := statements[1].(tsgo.VariableStatement).
		DeclarationList().Declarations()[0]
	if identifierText(firstCapture.Name()) != "__gotots_assign_0" ||
		identifierText(firstCapture.Initializer()) != "right" ||
		identifierText(secondCapture.Name()) != "__gotots_assign_1" ||
		identifierText(secondCapture.Initializer()) != "left" {
		t.Fatal("right sides were not captured in source order before stores")
	}
	for index := 2; index < 4; index++ {
		if _, ok := statements[index].(tsgo.ExpressionStatement); !ok {
			t.Fatalf("statement %d = %T, want assignment store", index, statements[index])
		}
	}
	firstStore := statements[2].(tsgo.ExpressionStatement).
		Expression().(tsgo.BinaryExpression)
	secondStore := statements[3].(tsgo.ExpressionStatement).
		Expression().(tsgo.BinaryExpression)
	if identifierText(firstStore.Left()) != "left" ||
		identifierText(firstStore.Right()) != "__gotots_assign_0" ||
		identifierText(secondStore.Left()) != "right" ||
		identifierText(secondStore.Right()) != "__gotots_assign_1" {
		t.Fatal("stores do not consume captures in left-to-right target order")
	}
}

func TestParallelAssignmentRejectsUnownedTargetAndMultiResultCases(t *testing.T) {
	for _, mutate := range []func(*ast.AssignStmt){
		func(source *ast.AssignStmt) {
			source.Lhs[0] = &ast.IndexExpr{X: source.Lhs[0], Index: source.Rhs[0]}
		},
		func(source *ast.AssignStmt) {
			source.Rhs = source.Rhs[:1]
		},
	} {
		loaded := loadProject(t)
		function := loaded.Files()[0].Syntax().Decls[0].(*ast.FuncDecl)
		mutate(function.Body.List[0].(*ast.AssignStmt))

		_, err := emit.CompileFile(loaded, loaded.Files()[0].Syntax())
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
}

func TestCompoundIdentifierAssignmentUsesDirectTargetOperation(t *testing.T) {
	loaded := loadProject(t)
	targetFile := emitProject(
		t,
		loaded,
	)
	function := targetFile.Statements()[5].(tsgo.FunctionDeclaration)
	statement := function.Body().(tsgo.Block).Statements()[0].(tsgo.ExpressionStatement)
	operation := statement.Expression().(tsgo.BinaryExpression)
	if operation.OperatorToken().Kind() != tsgo.SyntaxKindPlusEqualsToken ||
		identifierText(operation.Left()) != "total" {
		t.Fatal("compound assignment is not a direct owned +=")
	}
	if identifierText(operation.Right()) != "delta" {
		t.Fatal("compound assignment changed its right operand")
	}
}

func TestCompoundAssignmentBoundaryMutationsFailClosed(t *testing.T) {
	for name, mutate := range map[string]func(*ast.AssignStmt){
		"operator": func(source *ast.AssignStmt) {
			source.Tok = token.SUB_ASSIGN
		},
	} {
		t.Run(name, func(t *testing.T) {
			loaded := loadProject(t)
			function := loaded.Files()[0].Syntax().Decls[4].(*ast.FuncDecl)
			mutate(function.Body.List[0].(*ast.AssignStmt))
			_, err := emit.CompileFile(loaded, loaded.Files()[0].Syntax())
			var unsupported *api.UnsupportedError
			if !errors.As(err, &unsupported) ||
				unsupported.Construct != "*ast.AssignStmt" ||
				unsupported.Role != api.RoleBlockStatement {
				t.Fatalf("error = %#v, want owned assignment failure", err)
			}
		})
	}
}

func TestPrimitiveAccessorCompoundFailsAtAssignmentOwner(t *testing.T) {
	directory := t.TempDir()
	writeFile(
		t,
		filepath.Join(directory, "go.mod"),
		"module example.com/compoundboundary\n\ngo 1.26.4\n",
	)
	writeFile(t, filepath.Join(directory, "source.go"), `package compoundboundary

func F(values []int32, delta int32) {
	values[0] += delta
}
`)
	loaded, err := load.One(context.Background(), load.Request{
		Directory: directory,
		Pattern:   ".",
	})
	if err != nil {
		t.Fatal(err)
	}
	_, err = emit.CompileFile(loaded, loaded.Files()[0].Syntax())
	var unsupported *api.UnsupportedError
	if !errors.As(err, &unsupported) ||
		unsupported.Construct != "*ast.AssignStmt" ||
		unsupported.Role != api.RoleBlockStatement {
		t.Fatalf("error = %#v, want owned assignment failure", err)
	}
}

func identifierText(node tsgo.Node) string {
	return node.(tsgo.Identifier).Text()
}

func loadProject(t *testing.T) *load.Package {
	t.Helper()
	loaded, err := load.One(context.Background(), load.Request{
		Directory: projectDirectory(),
		Pattern:   ".",
	})
	if err != nil {
		t.Fatal(err)
	}
	return loaded
}

func emitProject(
	t *testing.T,
	loaded *load.Package,
) tsgo.SourceFile {
	t.Helper()
	source := loaded.Files()[0].Syntax()
	emission, err := emit.CompileFile(loaded, source)
	if err != nil {
		t.Fatal(err)
	}
	owned, ok := loaded.FileForSyntax(source)
	if !ok {
		t.Fatal("source syntax is not package-owned")
	}
	expectedPath, err := output.SourcePath(loaded, owned)
	if err != nil {
		t.Fatal(err)
	}
	for _, file := range emission.Files() {
		if file.Kind() == emit.TargetFileSource && file.OutputPath() == expectedPath {
			return file.SourceFile()
		}
	}
	t.Fatalf("complete emission has no source artifact %s", expectedPath)
	return nil
}

func executeGo(t *testing.T, workingDirectory string) string {
	t.Helper()
	modulePath, err := filepath.Abs(projectDirectory())
	if err != nil {
		t.Fatal(err)
	}
	runnerDirectory := filepath.Join(workingDirectory, "go-runner")
	if err := os.MkdirAll(runnerDirectory, 0o755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(runnerDirectory, "go.mod"), fmt.Sprintf(`module example.com/runner

go 1.26.4

require example.com/parallelassignment v0.0.0

replace example.com/parallelassignment => %s
`, filepath.ToSlash(modulePath)))
	writeFile(t, filepath.Join(runnerDirectory, "main.go"), `package main

import (
	"fmt"

	assignment "example.com/parallelassignment"
)

func main() {
	fmt.Println(assignment.SwapLeft(3, 9))
	fmt.Println(assignment.Rotate(4, 7))
	fmt.Println(assignment.Declare(11, 13))
	fmt.Println(assignment.Shadow(17))
	fmt.Println(assignment.Accumulate(19, 23))
}
`)
	return run(t, runnerDirectory, filepath.Join(runtime.GOROOT(), "bin", "go"), "run", ".")
}

func executeTypeScript(
	t *testing.T,
	loaded *load.Package,
	workingDirectory string,
) string {
	t.Helper()
	writeFile(t, filepath.Join(workingDirectory, "package.json"), "{\"type\":\"module\"}\n")
	targetPaths, sourceModule := materializeProject(t, loaded, workingDirectory)
	runnerPath := filepath.Join(workingDirectory, "runner.ts")
	writeFile(t, runnerPath, `import { Accumulate, Declare, Rotate, Shadow, SwapLeft } from "`+
		sourceModule+`";

console.log(SwapLeft(3, 9));
console.log(Rotate(4, 7));
console.log(Declare(11, 13));
console.log(Shadow(17));
console.log(Accumulate(19, 23));
`)
	outputDirectory := filepath.Join(workingDirectory, "out")
	arguments := []string{
		"--target", "es2022",
		"--module", "nodenext",
		"--moduleResolution", "nodenext",
		"--strict",
		"--outDir", outputDirectory,
	}
	arguments = append(arguments, targetPaths...)
	arguments = append(arguments, runnerPath)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := tsgo.Compile(
		ctx,
		repositoryRoot(),
		workingDirectory,
		arguments,
	); err != nil {
		t.Fatal(err)
	}
	return run(t, workingDirectory, "node", filepath.Join(outputDirectory, "runner.js"))
}

func materializeProject(
	t *testing.T,
	loaded *load.Package,
	workingDirectory string,
) ([]string, string) {
	t.Helper()
	roots, err := emit.ExportedAPIRoots(loaded)
	if err != nil {
		t.Fatal(err)
	}
	emission, err := emit.Compile(loaded.Program(), roots)
	if err != nil {
		t.Fatal(err)
	}
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
	var sourceModule string
	for _, file := range emission.Files() {
		printed, err := client.PrintNode(file.SourceFile(), tsgo.PrintOptions{})
		if err != nil {
			t.Fatal(err)
		}
		targetPath := filepath.Join(workingDirectory, filepath.FromSlash(file.OutputPath()))
		writeFile(t, targetPath, printed)
		targetPaths = append(targetPaths, targetPath)
		if file.Kind() == emit.TargetFileSource {
			if sourceModule != "" {
				t.Fatal("assignment fixture emitted multiple source modules")
			}
			sourceModule = "./" + strings.TrimSuffix(file.OutputPath(), ".ts") + ".js"
		}
	}
	if sourceModule == "" {
		t.Fatal("assignment fixture emitted no source module")
	}
	return targetPaths, sourceModule
}

func run(t *testing.T, directory, name string, arguments ...string) string {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	command := exec.CommandContext(ctx, name, arguments...)
	command.Dir = directory
	command.Env = append(os.Environ(), "GOMEMLIMIT=1GiB")
	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("%s %s: %v\n%s", name, strings.Join(arguments, " "), err, output)
	}
	return string(output)
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func projectDirectory() string {
	return filepath.Join(repositoryRoot(), "testdata", "projects", "parallel-assignment")
}

func repositoryRoot() string {
	return filepath.Join("..", "..", "..", "..")
}
