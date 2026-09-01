package function_test

import (
	"context"
	"errors"
	"fmt"
	"go/ast"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/tsoniclang/gotots/internal/emit"
	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/load"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func TestStructuredControlPrintsTypechecksAndExecutesDifferentially(t *testing.T) {
	loaded := loadStructuredControlProject(t)
	workingDirectory := t.TempDir()
	outputPath := filepath.Join(workingDirectory, "structured-control.ts")
	targetFile := emitStructuredControl(t, loaded)
	printed := printExecutableTargetFile(t, targetFile, workingDirectory)

	expected, err := os.ReadFile(filepath.Join(structuredControlProjectDirectory(), "expected.ts"))
	if err != nil {
		t.Fatal(err)
	}
	if printed != string(expected) {
		t.Fatalf("printed TypeScript:\n%s\nwant:\n%s", printed, expected)
	}
	writeFile(t, outputPath, printed)

	goOutput := executeStructuredControlGo(t, workingDirectory)
	typeScriptOutput := executeStructuredControlTypeScript(t, loaded, workingDirectory)
	if typeScriptOutput != goOutput {
		t.Fatalf("TypeScript output = %q, Go output = %q", typeScriptOutput, goOutput)
	}
}

func TestStructuredControlCreatesScopedExactTargetTree(t *testing.T) {
	loaded := loadStructuredControlProject(t)
	targetFile := emitStructuredControl(t, loaded)
	classify := targetFunction(t, targetFile, "Classify")
	classifyBody := classify.Body().(tsgo.Block).Statements()
	if len(classifyBody) != 1 {
		t.Fatalf("Classify statements = %d, want one initializer scope", len(classifyBody))
	}
	scope, ok := classifyBody[0].(tsgo.Block)
	if !ok {
		t.Fatalf("Classify statement = %T, want scoped block", classifyBody[0])
	}
	if len(scope.Statements()) != 2 {
		t.Fatalf("initializer scope statements = %d, want declaration and if", len(scope.Statements()))
	}
	condition := scope.Statements()[1].(tsgo.IfStatement)
	if _, ok := condition.ElseStatement().(tsgo.IfStatement); !ok {
		t.Fatalf("alternate = %T, want nested if", condition.ElseStatement())
	}

	sum := targetFunction(t, targetFile, "Sum")
	sumLoop := sum.Body().(tsgo.Block).Statements()[2].(tsgo.ForStatement)
	if sumLoop.Initializer() != nil || sumLoop.Incrementor() != nil ||
		sumLoop.Condition() == nil {
		t.Fatal("condition-only loop target children are not exact")
	}
	once := targetFunction(t, targetFile, "Once")
	onceLoop := once.Body().(tsgo.Block).Statements()[1].(tsgo.ForStatement)
	if onceLoop.Initializer() != nil ||
		onceLoop.Condition() != nil ||
		onceLoop.Incrementor() != nil {
		t.Fatal("infinite loop target has a fabricated clause")
	}
}

func TestStructuredControlChildRoleMutationsFailClosed(t *testing.T) {
	for _, testCase := range []struct {
		name      string
		mutate    func(*ast.IfStmt)
		construct string
		role      api.Role
	}{
		{
			name: "initializer",
			mutate: func(source *ast.IfStmt) {
				source.Init = &ast.ReturnStmt{}
			},
			construct: "*ast.ReturnStmt",
			role:      api.RoleIfInitializer,
		},
		{
			name: "alternate",
			mutate: func(source *ast.IfStmt) {
				source.Else = &ast.ReturnStmt{}
			},
			construct: "*ast.ReturnStmt",
			role:      api.RoleIfElse,
		},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			loaded := loadStructuredControlProject(t)
			function := loaded.Files()[0].Syntax().Decls[0].(*ast.FuncDecl)
			source := function.Body.List[0].(*ast.IfStmt)
			testCase.mutate(source)

			_, err := emit.CompileFile(loaded, loaded.Files()[0].Syntax())
			var unsupported *api.UnsupportedError
			if !errors.As(err, &unsupported) {
				t.Fatalf("error = %v, want *api.UnsupportedError", err)
			}
			if unsupported.Category != api.CategoryStatement ||
				unsupported.Construct != testCase.construct ||
				unsupported.Role != testCase.role {
				t.Fatalf("unsupported error = %#v", unsupported)
			}
		})
	}
}

func loadStructuredControlProject(t *testing.T) *load.Package {
	t.Helper()
	loaded, err := load.One(context.Background(), load.Request{
		Directory: structuredControlProjectDirectory(),
		Pattern:   ".",
	})
	if err != nil {
		t.Fatal(err)
	}
	return loaded
}

func emitStructuredControl(
	t *testing.T,
	loaded *load.Package,
) tsgo.SourceFile {
	t.Helper()
	return compileSourceFile(t, loaded, loaded.Files()[0].Syntax())
}

func executeStructuredControlGo(t *testing.T, workingDirectory string) string {
	t.Helper()
	modulePath, err := filepath.Abs(structuredControlProjectDirectory())
	if err != nil {
		t.Fatal(err)
	}
	runnerDirectory := filepath.Join(workingDirectory, "go-runner")
	if err := os.MkdirAll(runnerDirectory, 0o755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(runnerDirectory, "go.mod"), fmt.Sprintf(`module example.com/runner

go 1.26.4

require example.com/structuredcontrol v0.0.0

replace example.com/structuredcontrol => %s
`, filepath.ToSlash(modulePath)))
	writeFile(t, filepath.Join(runnerDirectory, "main.go"), `package main

import (
	"fmt"

	control "example.com/structuredcontrol"
)

func main() {
	fmt.Println(control.Classify(-5))
	fmt.Println(control.Classify(-2147483648))
	fmt.Println(control.Classify(0))
	fmt.Println(control.Classify(9))
	fmt.Println(control.Classify(2147483647))
	fmt.Println(control.Sum(0))
	fmt.Println(control.Sum(5))
	fmt.Println(control.Once())
}
`)
	return run(t, runnerDirectory, filepath.Join(runtime.GOROOT(), "bin", "go"), "run", ".")
}

func executeStructuredControlTypeScript(
	t *testing.T,
	loaded *load.Package,
	workingDirectory string,
) string {
	t.Helper()
	artifacts := materializeExportedProgram(t, loaded, workingDirectory)
	runnerPath := filepath.Join(workingDirectory, "runner.ts")
	writeFile(t, runnerPath, `import { Classify, Once, Sum } from "`+
		artifacts.module(t, "source.ts")+`";

console.log(Classify(-5));
console.log(Classify(-2147483648));
console.log(Classify(0));
console.log(Classify(9));
console.log(Classify(2147483647));
console.log(Sum(0));
console.log(Sum(5));
console.log(Once());
`)
	return executeMaterializedTypeScript(t, workingDirectory, artifacts, runnerPath)
}

func structuredControlProjectDirectory() string {
	return filepath.Join(repositoryRoot(), "testdata", "projects", "structured-control")
}
