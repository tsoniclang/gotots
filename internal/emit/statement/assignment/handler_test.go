package assignment_test

import (
	"context"
	"errors"
	"fmt"
	"go/ast"
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
	runtimefixture "github.com/tsoniclang/gotots/internal/testfixture/gototsruntime"
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
	printed, err := client.PrintNode(
		assignmentExecutableSource(targetFile),
		tsgo.PrintOptions{},
	)
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
	function := assignmentFunctionByName(t, targetFile, "SwapLeft")
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

func TestParallelAssignmentBoundaryMutationsFailAtTheirOwners(t *testing.T) {
	for _, testCase := range []struct {
		mutate    func(*ast.AssignStmt)
		category  api.Category
		role      api.Role
		construct string
	}{
		{
			mutate: func(source *ast.AssignStmt) {
				source.Lhs[0] = &ast.IndexExpr{
					X:     source.Lhs[0],
					Index: source.Rhs[0],
				}
			},
			category:  api.CategoryExpression,
			role:      api.RoleAssignmentTarget,
			construct: "*ast.IndexExpr",
		},
		{
			mutate: func(source *ast.AssignStmt) {
				source.Rhs = source.Rhs[:1]
			},
			category:  api.CategoryStatement,
			role:      api.RoleBlockStatement,
			construct: "*ast.AssignStmt",
		},
	} {
		loaded := loadProject(t)
		function := loaded.Files()[0].Syntax().Decls[0].(*ast.FuncDecl)
		testCase.mutate(function.Body.List[0].(*ast.AssignStmt))

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
	}
}

func TestCompoundIdentifierAssignmentUsesDirectTargetOperation(t *testing.T) {
	loaded := loadProject(t)
	targetFile := emitProject(
		t,
		loaded,
	)
	function := assignmentFunctionByName(t, targetFile, "Accumulate")
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

func assignmentFunctionByName(
	t *testing.T,
	source tsgo.SourceFile,
	name string,
) tsgo.FunctionDeclaration {
	t.Helper()
	for _, statement := range source.Statements() {
		function, ok := statement.(tsgo.FunctionDeclaration)
		if ok && function.Name().Text() == name {
			return function
		}
	}
	t.Fatalf("target function %q is absent", name)
	return nil
}

func assignmentExecutableSource(source tsgo.SourceFile) tsgo.SourceFile {
	statements := make([]tsgo.Statement, 0, len(source.Statements()))
	for _, statement := range source.Statements() {
		if declaration, ok := statement.(tsgo.ImportDeclaration); ok {
			module, moduleOK := declaration.ModuleSpecifier().(tsgo.StringLiteral)
			if moduleOK && (strings.HasSuffix(module.Text(), "/source-fact.js") ||
				module.Text() == "@tsonic/core/lang.js") {
				continue
			}
		}
		if _, fact := statement.(tsgo.ExpressionStatement); fact {
			continue
		}
		statements = append(statements, statement)
	}
	factory := tsgo.NewFactory()
	return factory.SourceFile(
		statements,
		source.EndOfFileToken(),
		source.SourceData(),
	)
}

func TestPrimitiveAccessorCompoundAndIncrementAreOwned(t *testing.T) {
	directory := t.TempDir()
	writeFile(
		t,
		filepath.Join(directory, "go.mod"),
		"module example.com/compoundboundary\n\ngo 1.26.4\n",
	)
	writeFile(t, filepath.Join(directory, "source.go"), `package compoundboundary

func F(values []int32, delta int32) int32 {
	values[0] += delta
	values[0]--
	return values[0]
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
	if err != nil {
		t.Fatalf("accessor update was rejected: %v", err)
	}
}

func TestSingleBlankAssignmentEvaluatesRuntimeValuesAndErasesConstants(
	t *testing.T,
) {
	directory := t.TempDir()
	writeFile(
		t,
		filepath.Join(directory, "go.mod"),
		"module example.com/blankassignment\n\ngo 1.26.4\n",
	)
	writeFile(t, filepath.Join(directory, "source.go"), `package blankassignment

func Value() int32 { return 3 }

func Use() int32 {
	_ = 1 + 2
	_ = Value()
	return 4
}
`)
	loaded, err := load.One(context.Background(), load.Request{
		Directory: directory,
		Pattern:   ".",
	})
	if err != nil {
		t.Fatal(err)
	}
	roots, err := emit.ExportedAPIRoots(loaded)
	if err != nil {
		t.Fatal(err)
	}
	emission, err := emit.Compile(loaded.Program(), roots)
	if err != nil {
		t.Fatal(err)
	}
	var use tsgo.FunctionDeclaration
	for _, file := range emission.Files() {
		if file.Kind() != emit.TargetFileSource {
			continue
		}
		for _, statement := range file.SourceFile().Statements() {
			function, ok := statement.(tsgo.FunctionDeclaration)
			if ok && function.Name().Text() == "Use" {
				use = function
			}
		}
	}
	if use == nil {
		t.Fatal("Use target function is absent")
	}
	statements := use.Body().(tsgo.Block).Statements()
	if len(statements) != 2 {
		t.Fatalf(
			"Use statements = %d, want runtime evaluation and return",
			len(statements),
		)
	}
	evaluated, ok := statements[0].(tsgo.ExpressionStatement)
	if !ok {
		t.Fatalf("blank runtime evaluation = %T, want expression", statements[0])
	}
	call, ok := evaluated.Expression().(tsgo.CallExpression)
	if !ok {
		t.Fatalf("blank runtime value = %T, want call", evaluated.Expression())
	}
	callee, ok := call.Expression().(tsgo.Identifier)
	if !ok || callee.Text() != "Value" {
		t.Fatalf("blank runtime callee = %T/%v, want Value", call.Expression(), call.Expression())
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
	if err := runtimefixture.InstallResolution(workingDirectory, outputDirectory); err != nil {
		t.Fatal(err)
	}
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
