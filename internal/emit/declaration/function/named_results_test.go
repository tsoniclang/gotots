package function_test

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/tsoniclang/gotots/internal/load"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func TestNamedResultsPrintTypecheckAndExecuteDifferentially(t *testing.T) {
	loaded := loadNamedResultsProject(t)
	workingDirectory := t.TempDir()
	targetFile := compileSourceFile(t, loaded, loaded.Files()[0].Syntax())
	printed := printTargetFile(t, targetFile, workingDirectory)
	if printed == "" {
		t.Fatal("named-result target is empty")
	}

	goOutput := executeNamedResultsGo(t, workingDirectory)
	typeScriptOutput := executeNamedResultsTypeScript(t, loaded, workingDirectory)
	if typeScriptOutput != goOutput {
		t.Fatalf("TypeScript output = %q, Go output = %q", typeScriptOutput, goOutput)
	}
}

func TestNamedResultsOwnZerosAndBareReturns(t *testing.T) {
	loaded := loadNamedResultsProject(t)
	targetFile := compileSourceFile(t, loaded, loaded.Files()[0].Syntax())

	next := targetFunction(t, targetFile, "Next")
	statements := next.Body().(tsgo.Block).Statements()
	if len(statements) != 5 {
		t.Fatalf("Next statements = %d, want two zeros, two stores, and return", len(statements))
	}
	for index := range 2 {
		declaration, ok := statements[index].(tsgo.VariableStatement)
		if !ok {
			t.Fatalf("Next statement %d = %T, want named-result declaration", index, statements[index])
		}
		variables := declaration.DeclarationList().Declarations()
		if len(variables) != 1 || variables[0].Type() == nil || variables[0].Initializer() == nil {
			t.Fatalf("Next result declaration %d is incomplete", index)
		}
	}
	result, ok := statements[4].(tsgo.ReturnStatement)
	if !ok {
		t.Fatalf("Next final statement = %T, want ReturnStatement", statements[4])
	}
	tuple, ok := result.Expression().(tsgo.ArrayLiteralExpression)
	if !ok || len(tuple.Elements()) != 2 {
		t.Fatalf("Next return = %T, want two-element tuple", result.Expression())
	}

	nested := targetFunction(t, targetFile, "Nested")
	declaration := nested.Body().(tsgo.Block).Statements()[1].(tsgo.VariableStatement)
	literal := declaration.DeclarationList().Declarations()[0].
		Initializer().(tsgo.FunctionExpression)
	literalStatements := literal.Body().(tsgo.Block).Statements()
	if len(literalStatements) != 3 {
		t.Fatalf("nested literal statements = %d, want own zero, store, and return", len(literalStatements))
	}
}

func loadNamedResultsProject(t *testing.T) *load.Package {
	t.Helper()
	loaded, err := load.One(context.Background(), load.Request{
		Directory: namedResultsProjectDirectory(),
		Pattern:   ".",
	})
	if err != nil {
		t.Fatal(err)
	}
	return loaded
}

func executeNamedResultsGo(t *testing.T, workingDirectory string) string {
	t.Helper()
	modulePath, err := filepath.Abs(namedResultsProjectDirectory())
	if err != nil {
		t.Fatal(err)
	}
	runnerDirectory := filepath.Join(workingDirectory, "go-runner")
	if err := os.MkdirAll(runnerDirectory, 0o755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(runnerDirectory, "go.mod"), fmt.Sprintf(`module example.com/runner

go 1.26.4

require example.com/namedresults v0.0.0

replace example.com/namedresults => %s
`, filepath.ToSlash(modulePath)))
	writeFile(t, filepath.Join(runnerDirectory, "main.go"), `package main

import (
	"fmt"

	results "example.com/namedresults"
)

func main() {
	fmt.Println(results.Next(5))
	fmt.Println(results.Next(-2))
	fmt.Println(results.Single(7))
	fmt.Println(results.Explicit(8))
	fmt.Println(results.Nested(9))
	fmt.Println(results.ZeroBox().Value)
}
`)
	return run(t, runnerDirectory, filepath.Join(runtime.GOROOT(), "bin", "go"), "run", ".")
}

func executeNamedResultsTypeScript(
	t *testing.T,
	loaded *load.Package,
	workingDirectory string,
) string {
	t.Helper()
	artifacts := materializeExportedProgram(t, loaded, workingDirectory)
	runnerPath := filepath.Join(workingDirectory, "runner.ts")
	writeFile(t, runnerPath, `import {
    Explicit,
    Nested,
    Next,
	    Single,
	    ZeroBox,
	} from "`+artifacts.module(t, "source.ts")+`";

console.log(...Next(5));
console.log(...Next(-2));
console.log(Single(7));
console.log(Explicit(8));
console.log(Nested(9));
console.log(ZeroBox().Value);
`)
	return executeMaterializedTypeScript(t, workingDirectory, artifacts, runnerPath)
}

func namedResultsProjectDirectory() string {
	return filepath.Join(
		repositoryRoot(),
		"testdata",
		"constructs",
		"declaration",
		"function",
		"named-results",
	)
}
