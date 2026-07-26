package function_test

import (
	"context"
	"errors"
	"fmt"
	"go/ast"
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

func TestZeroResultCallsPrintTypecheckAndExecuteDifferentially(t *testing.T) {
	loaded := loadZeroResultProject(t)
	workingDirectory := t.TempDir()
	outputPath := filepath.Join(workingDirectory, "void-calls.ts")
	targetFile := emitZeroResultProject(t, loaded, outputPath)
	printed := printTargetFile(t, targetFile, workingDirectory)

	expected, err := os.ReadFile(filepath.Join(zeroResultProjectDirectory(), "expected.ts"))
	if err != nil {
		t.Fatal(err)
	}
	if printed != string(expected) {
		t.Fatalf("printed TypeScript:\n%s\nwant:\n%s", printed, expected)
	}
	writeFile(t, outputPath, printed)

	goOutput := executeZeroResultGo(t, workingDirectory)
	typeScriptOutput := executeZeroResultTypeScript(t, workingDirectory, outputPath)
	if typeScriptOutput != goOutput {
		t.Fatalf("TypeScript output = %q, Go output = %q", typeScriptOutput, goOutput)
	}
}

func TestZeroResultCallsCreateExactTargetTree(t *testing.T) {
	loaded := loadZeroResultProject(t)
	targetFile := emitZeroResultProject(
		t,
		loaded,
		filepath.Join(t.TempDir(), "void-calls.ts"),
	)
	statements := targetFile.Statements()
	if len(statements) != 4 {
		t.Fatalf("target statements = %d, want import and three functions", len(statements))
	}
	touch := statements[1].(tsgo.FunctionDeclaration)
	if touch.Type().Kind() != tsgo.SyntaxKindVoidKeyword {
		t.Fatalf("Touch result kind = %d, want void keyword", touch.Type().Kind())
	}
	touchIf := touch.Body().(tsgo.Block).Statements()[0].(tsgo.IfStatement)
	bareReturn := touchIf.ThenStatement().(tsgo.Block).
		Statements()[0].(tsgo.ReturnStatement)
	if bareReturn.Expression() != nil {
		t.Fatalf("bare return expression = %T, want nil", bareReturn.Expression())
	}
	runFunction := statements[3].(tsgo.FunctionDeclaration)
	runStatements := runFunction.Body().(tsgo.Block).Statements()
	if len(runStatements) != 3 {
		t.Fatalf("Run statements = %d, want two calls and return", len(runStatements))
	}
	for index := 0; index < 2; index++ {
		if _, ok := runStatements[index].(tsgo.ExpressionStatement); !ok {
			t.Fatalf("Run statement %d = %T, want call statement", index, runStatements[index])
		}
	}
}

func TestZeroResultCallableMutationsFailAtOwningContext(t *testing.T) {
	t.Run("zero-result-call-as-value", func(t *testing.T) {
		loaded := loadZeroResultProject(t)
		identity := loaded.Files()[0].Syntax().Decls[1].(*ast.FuncDecl)
		runFunction := loaded.Files()[0].Syntax().Decls[2].(*ast.FuncDecl)
		touchCall := runFunction.Body.List[0].(*ast.ExprStmt).X
		identity.Body.List[0].(*ast.ReturnStmt).Results = []ast.Expr{touchCall}

		_, err := emit.CompileFile(loaded, loaded.Files()[0].Syntax())
		assertUnsupportedCallable(
			t,
			err,
			api.CategoryExpression,
			"*ast.CallExpr",
			api.RoleReturnResult,
		)
	})

	t.Run("bare-return-from-value-function", func(t *testing.T) {
		loaded := loadZeroResultProject(t)
		identity := loaded.Files()[0].Syntax().Decls[1].(*ast.FuncDecl)
		identity.Body.List[0].(*ast.ReturnStmt).Results = nil

		_, err := emit.CompileFile(loaded, loaded.Files()[0].Syntax())
		assertUnsupportedCallable(
			t,
			err,
			api.CategoryStatement,
			"*ast.ReturnStmt",
			api.RoleBlockStatement,
		)
	})
}

func assertUnsupportedCallable(
	t *testing.T,
	err error,
	category api.Category,
	construct string,
	role api.Role,
) {
	t.Helper()
	var unsupported *api.UnsupportedError
	if !errors.As(err, &unsupported) {
		t.Fatalf("error = %v, want *api.UnsupportedError", err)
	}
	if unsupported.Category != category ||
		unsupported.Construct != construct ||
		unsupported.Role != role {
		t.Fatalf("unsupported error = %#v", unsupported)
	}
}

func loadZeroResultProject(t *testing.T) *load.Package {
	t.Helper()
	loaded, err := load.One(context.Background(), load.Request{
		Directory: zeroResultProjectDirectory(),
		Pattern:   ".",
	})
	if err != nil {
		t.Fatal(err)
	}
	return loaded
}

func emitZeroResultProject(
	t *testing.T,
	loaded *load.Package,
	outputPath string,
) tsgo.SourceFile {
	t.Helper()
	targetFile, err := emit.CompileFile(loaded, loaded.Files()[0].Syntax())
	if err != nil {
		t.Fatal(err)
	}
	return targetFile
}

func executeZeroResultGo(t *testing.T, workingDirectory string) string {
	t.Helper()
	modulePath, err := filepath.Abs(zeroResultProjectDirectory())
	if err != nil {
		t.Fatal(err)
	}
	runnerDirectory := filepath.Join(workingDirectory, "go-runner")
	if err := os.MkdirAll(runnerDirectory, 0o755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(runnerDirectory, "go.mod"), fmt.Sprintf(`module example.com/runner

go 1.26.4

require example.com/voidcalls v0.0.0

replace example.com/voidcalls => %s
`, filepath.ToSlash(modulePath)))
	writeFile(t, filepath.Join(runnerDirectory, "main.go"), `package main

import (
	"fmt"

	calls "example.com/voidcalls"
)

func main() {
	fmt.Println(calls.Run(-1))
	fmt.Println(calls.Run(7))
}
`)
	return run(t, runnerDirectory, filepath.Join(runtime.GOROOT(), "bin", "go"), "run", ".")
}

func executeZeroResultTypeScript(
	t *testing.T,
	workingDirectory string,
	outputPath string,
) string {
	t.Helper()
	writeFile(t, filepath.Join(workingDirectory, "package.json"), "{\"type\":\"module\"}\n")
	installTsonicCoreTypes(t, workingDirectory)
	runnerPath := filepath.Join(workingDirectory, "runner.ts")
	writeFile(t, runnerPath, `import { Run } from "./void-calls.js";

console.log(Run(-1));
console.log(Run(7));
`)
	outputDirectory := filepath.Join(workingDirectory, "out")
	toolPath := strings.TrimSpace(
		run(t, repositoryRoot(), filepath.Join(runtime.GOROOT(), "bin", "go"), "tool", "-n", "tsgo"),
	)
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

func zeroResultProjectDirectory() string {
	return filepath.Join(repositoryRoot(), "testdata", "projects", "void-calls")
}
