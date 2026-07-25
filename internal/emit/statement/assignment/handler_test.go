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
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func TestParallelIdentifierAssignmentPrintsTypechecksAndExecutesDifferentially(
	t *testing.T,
) {
	loaded := loadProject(t)
	workingDirectory := t.TempDir()
	outputPath := filepath.Join(workingDirectory, "assignment.ts")
	targetFile := emitProject(t, loaded, outputPath)

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
	typeScriptOutput := executeTypeScript(t, workingDirectory, outputPath)
	if typeScriptOutput != goOutput {
		t.Fatalf("TypeScript output = %q, Go output = %q", typeScriptOutput, goOutput)
	}
}

func TestParallelAssignmentCreatesCapturesBeforeStores(t *testing.T) {
	loaded := loadProject(t)
	targetFile := emitProject(
		t,
		loaded,
		filepath.Join(t.TempDir(), "assignment.ts"),
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
	if identifierText(firstCapture.Name()) != "$assign0" ||
		identifierText(firstCapture.Initializer()) != "right" ||
		identifierText(secondCapture.Name()) != "$assign1" ||
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
		identifierText(firstStore.Right()) != "$assign0" ||
		identifierText(secondStore.Left()) != "right" ||
		identifierText(secondStore.Right()) != "$assign1" {
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

		_, err := emit.New(loaded).EmitFile(
			loaded.Files()[0].Syntax(),
			filepath.Join(t.TempDir(), "assignment.ts"),
		)
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
	outputPath string,
) tsgo.SourceFile {
	t.Helper()
	targetFile, err := emit.New(loaded).EmitFile(loaded.Files()[0].Syntax(), outputPath)
	if err != nil {
		t.Fatal(err)
	}
	return targetFile
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
}
`)
	return run(t, runnerDirectory, filepath.Join(runtime.GOROOT(), "bin", "go"), "run", ".")
}

func executeTypeScript(
	t *testing.T,
	workingDirectory string,
	outputPath string,
) string {
	t.Helper()
	writeFile(t, filepath.Join(workingDirectory, "package.json"), "{\"type\":\"module\"}\n")
	installCoreTypes(t, workingDirectory)
	runnerPath := filepath.Join(workingDirectory, "runner.ts")
	writeFile(t, runnerPath, `import { Declare, Rotate, Shadow, SwapLeft } from "./assignment.js";

console.log(SwapLeft(3, 9));
console.log(Rotate(4, 7));
console.log(Declare(11, 13));
console.log(Shadow(17));
`)
	outputDirectory := filepath.Join(workingDirectory, "out")
	toolPath := strings.TrimSpace(run(
		t,
		repositoryRoot(),
		filepath.Join(runtime.GOROOT(), "bin", "go"),
		"tool",
		"-n",
		"tsgo",
	))
	run(
		t,
		workingDirectory,
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

func installCoreTypes(t *testing.T, workingDirectory string) {
	t.Helper()
	moduleDirectory := filepath.Join(workingDirectory, "node_modules", "@tsonic", "core")
	if err := os.MkdirAll(moduleDirectory, 0o755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(moduleDirectory, "package.json"), `{
  "name": "@tsonic/core",
  "type": "module",
  "exports": {
    "./types.js": {
      "types": "./types.d.ts",
      "default": "./types.js"
    }
  }
}
`)
	writeFile(t, filepath.Join(moduleDirectory, "types.d.ts"), `export type bool = boolean;
export type int32 = number;
export type int64 = number;
`)
	writeFile(t, filepath.Join(moduleDirectory, "types.js"), "export {};\n")
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
