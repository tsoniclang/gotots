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

func TestIntegerConstantsPrintAndStrictTypecheck(t *testing.T) {
	loaded := loadIntegerConstantsProject(t)
	workingDirectory := t.TempDir()
	outputPath := filepath.Join(workingDirectory, "constants.ts")
	targetFile := emitIntegerConstants(t, loaded, outputPath)
	printed := printTargetFile(t, targetFile, workingDirectory)

	expected, err := os.ReadFile(filepath.Join(integerConstantsProjectDirectory(), "expected.ts"))
	if err != nil {
		t.Fatal(err)
	}
	if printed != string(expected) {
		t.Fatalf("printed TypeScript:\n%s\nwant:\n%s", printed, expected)
	}
	if strings.Contains(printed, "9223372036854775807") ||
		strings.Contains(printed, "9223372036854776000") ||
		strings.Contains(printed, "n as int64") {
		t.Fatalf("printed TypeScript contains an inexact or bigint wide literal:\n%s", printed)
	}
	writeFile(t, outputPath, printed)
	strictTypecheckIntegerConstants(t, workingDirectory, outputPath)
}

func TestIntegerConstantsCreateBoundedTypedTargetTrees(t *testing.T) {
	loaded := loadIntegerConstantsProject(t)
	targetFile := emitIntegerConstants(t, loaded, filepath.Join(t.TempDir(), "constants.ts"))
	statements := targetFile.Statements()
	if len(statements) != 5 {
		t.Fatalf("target statements = %d, want import plus four functions", len(statements))
	}
	maximum := statements[3].(tsgo.FunctionDeclaration)
	maximumReturn := maximum.Body().(tsgo.Block).Statements()[0].(tsgo.ReturnStatement)
	sum, ok := maximumReturn.Expression().(tsgo.BinaryExpression)
	if !ok || sum.OperatorToken().Kind() != tsgo.SyntaxKindPlusToken {
		t.Fatalf("maximum expression = %T, want bounded binary sum", maximumReturn.Expression())
	}
	product, ok := sum.Left().(tsgo.BinaryExpression)
	if !ok || product.OperatorToken().Kind() != tsgo.SyntaxKindAsteriskToken {
		t.Fatalf("maximum left = %T, want bounded binary product", sum.Left())
	}
	minimum := statements[4].(tsgo.FunctionDeclaration)
	minimumReturn := minimum.Body().(tsgo.Block).Statements()[0].(tsgo.ReturnStatement)
	if minimumReturn.Expression().Kind() != tsgo.SyntaxKindBinaryExpression {
		t.Fatalf("minimum expression kind = %d, want binary expression", minimumReturn.Expression().Kind())
	}
}

func TestIntegerConstantsUseGoConstantValueNotLiteralSpelling(t *testing.T) {
	loaded := loadIntegerConstantsProject(t)
	maximum := loaded.Files()[0].Syntax().Decls[2].(*ast.FuncDecl)
	literal := maximum.Body.List[0].(*ast.ReturnStmt).Results[0].(*ast.BasicLit)
	literal.Value = "9223372036854776000"

	targetFile := emitIntegerConstants(t, loaded, filepath.Join(t.TempDir(), "constants.ts"))
	targetMaximum := targetFile.Statements()[3].(tsgo.FunctionDeclaration)
	targetReturn := targetMaximum.Body().(tsgo.Block).Statements()[0].(tsgo.ReturnStatement)
	sum := targetReturn.Expression().(tsgo.BinaryExpression)
	low := sum.Right().(tsgo.AsExpression).Expression().(tsgo.NumericLiteral)
	if low.Text() != "4294967295" {
		t.Fatalf("low chunk = %q, want exact semantic chunk 4294967295", low.Text())
	}
}

func TestIntegerConstantsRejectNonIntegerSyntaxMutation(t *testing.T) {
	loaded := loadIntegerConstantsProject(t)
	small := loaded.Files()[0].Syntax().Decls[0].(*ast.FuncDecl)
	literal := small.Body.List[0].(*ast.ReturnStmt).Results[0].(*ast.BasicLit)
	literal.Kind = token.FLOAT

	compiler := emit.New(loaded)
	_, err := compiler.EmitFile(
		loaded.Files()[0].Syntax(),
		filepath.Join(t.TempDir(), "constants.ts"),
	)
	var unsupported *api.UnsupportedError
	if !errors.As(err, &unsupported) {
		t.Fatalf("error = %v, want *api.UnsupportedError", err)
	}
	if unsupported.Category != api.CategoryExpression ||
		unsupported.Construct != "*ast.BasicLit" ||
		unsupported.Role != api.RoleReturnResult {
		t.Fatalf("unsupported error = %#v", unsupported)
	}
}

func loadIntegerConstantsProject(t *testing.T) *load.Package {
	t.Helper()
	loaded, err := load.One(context.Background(), load.Request{
		Directory: integerConstantsProjectDirectory(),
		Pattern:   ".",
	})
	if err != nil {
		t.Fatal(err)
	}
	return loaded
}

func emitIntegerConstants(
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

func printTargetFile(
	t *testing.T,
	targetFile tsgo.SourceFile,
	workingDirectory string,
) string {
	t.Helper()
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
	return printed
}

func strictTypecheckIntegerConstants(t *testing.T, workingDirectory, outputPath string) {
	t.Helper()
	writeFile(t, filepath.Join(workingDirectory, "package.json"), "{\"type\":\"module\"}\n")
	installTsonicCoreTypes(t, workingDirectory)
	toolPath := strings.TrimSpace(
		run(t, repositoryRoot(), filepath.Join(runtime.GOROOT(), "bin", "go"), "tool", "-n", "tsgo"),
	)
	run(t, workingDirectory,
		toolPath,
		"--target", "es2022",
		"--module", "nodenext",
		"--moduleResolution", "nodenext",
		"--strict",
		"--noEmit",
		outputPath,
	)
}

func executeIntegerConstantsGo(t *testing.T, workingDirectory string) string {
	t.Helper()
	modulePath, err := filepath.Abs(integerConstantsProjectDirectory())
	if err != nil {
		t.Fatal(err)
	}
	runnerDirectory := filepath.Join(workingDirectory, "go-runner")
	if err := os.MkdirAll(runnerDirectory, 0o755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(runnerDirectory, "go.mod"), fmt.Sprintf(`module example.com/runner

go 1.26.4

require example.com/integerconstants v0.0.0

replace example.com/integerconstants => %s
`, filepath.ToSlash(modulePath)))
	writeFile(t, filepath.Join(runnerDirectory, "main.go"), `package main

import (
	"fmt"

	constants "example.com/integerconstants"
)

func main() {
	fmt.Println(constants.Small())
	fmt.Println(constants.BeyondSafe())
	fmt.Println(constants.Maximum())
	fmt.Println(constants.Minimum())
}
`)
	return run(t, runnerDirectory, filepath.Join(runtime.GOROOT(), "bin", "go"), "run", ".")
}

func integerConstantsProjectDirectory() string {
	return filepath.Join(repositoryRoot(), "testdata", "projects", "integer-constants")
}
