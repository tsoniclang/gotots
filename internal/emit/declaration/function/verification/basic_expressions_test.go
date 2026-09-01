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

func TestBasicExpressionsPrintTypecheckAndExecuteDifferentially(t *testing.T) {
	loaded := loadBasicExpressionsProject(t)
	workingDirectory := t.TempDir()
	outputPath := filepath.Join(workingDirectory, "basic-expressions.ts")
	targetFile := emitBasicExpressions(t, loaded)
	printed := printExecutableTargetFile(t, targetFile, workingDirectory)

	expected, err := os.ReadFile(filepath.Join(basicExpressionsProjectDirectory(), "expected.ts"))
	if err != nil {
		t.Fatal(err)
	}
	if printed != string(expected) {
		t.Fatalf("printed TypeScript:\n%s\nwant:\n%s", printed, expected)
	}
	writeFile(t, outputPath, printed)

	goOutput := executeBasicExpressionsGo(t, workingDirectory)
	typeScriptOutput := executeBasicExpressionsTypeScript(t, loaded, workingDirectory)
	if typeScriptOutput != goOutput {
		t.Fatalf("TypeScript output = %q, Go output = %q", typeScriptOutput, goOutput)
	}
}

func TestBasicExpressionsCreateExactTargetTrees(t *testing.T) {
	loaded := loadBasicExpressionsProject(t)
	targetFile := emitBasicExpressions(t, loaded)

	arithmetic := targetFunction(t, targetFile, "Arithmetic")
	arithmeticReturn := targetReturn(t, arithmetic)
	product, ok := arithmeticReturn.Expression().(tsgo.BinaryExpression)
	if !ok || product.OperatorToken().Kind() != tsgo.SyntaxKindAsteriskToken {
		t.Fatalf("Arithmetic expression = %T, want direct multiplication", arithmeticReturn.Expression())
	}
	group, ok := product.Left().(tsgo.ParenthesizedExpression)
	if !ok {
		t.Fatalf("Arithmetic left = %T, want source parenthesized subtraction", product.Left())
	}
	difference, ok := group.Expression().(tsgo.BinaryExpression)
	if !ok || difference.OperatorToken().Kind() != tsgo.SyntaxKindMinusToken {
		t.Fatalf("Arithmetic grouped expression = %T, want subtraction", group.Expression())
	}

	shortCircuit := targetFunction(t, targetFile, "ShortCircuitAnd")
	logical, ok := targetReturn(t, shortCircuit).Expression().(tsgo.BinaryExpression)
	if !ok || logical.OperatorToken().Kind() != tsgo.SyntaxKindAmpersandAmpersandToken {
		t.Fatalf("ShortCircuitAnd expression = %T, want logical and", targetReturn(t, shortCircuit).Expression())
	}
	if logical.Left().Kind() != tsgo.SyntaxKindFalseKeyword {
		t.Fatalf("ShortCircuitAnd literal kind = %d, want false", logical.Left().Kind())
	}
}

func TestBasicExpressionBoundaryMutationsFailClosed(t *testing.T) {
	testCases := []struct {
		name      string
		mutate    func(*ast.File)
		construct string
		role      api.Role
	}{
		{
			name: "integer logical operator remains unsupported",
			mutate: func(file *ast.File) {
				function := sourceFunction(t, file, "WrapMultiply")
				function.Body.List[0].(*ast.ReturnStmt).
					Results[0].(*ast.BinaryExpr).
					Op = token.LAND
			},
			construct: "*ast.BinaryExpr",
			role:      api.RoleReturnResult,
		},
		{
			name: "boolean bitwise remains unsupported",
			mutate: func(file *ast.File) {
				function := sourceFunction(t, file, "Logic")
				function.Body.List[0].(*ast.ReturnStmt).
					Results[0].(*ast.BinaryExpr).
					Op = token.XOR
			},
			construct: "*ast.BinaryExpr",
			role:      api.RoleReturnResult,
		},
		{
			name: "parenthesized child must retain typed evidence",
			mutate: func(file *ast.File) {
				function := sourceFunction(t, file, "Arithmetic")
				product := function.Body.List[0].(*ast.ReturnStmt).
					Results[0].(*ast.BinaryExpr)
				product.X.(*ast.ParenExpr).X = ast.NewIdent("missing")
			},
			construct: "*ast.ParenExpr",
			role:      api.RoleBinaryLeft,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			loaded := loadBasicExpressionsProject(t)
			testCase.mutate(loaded.Files()[0].Syntax())
			_, err := emit.CompileFile(loaded, loaded.Files()[0].Syntax())
			var unsupported *api.UnsupportedError
			if !errors.As(err, &unsupported) {
				t.Fatalf("error = %v, want *api.UnsupportedError", err)
			}
			if unsupported.Category != api.CategoryExpression ||
				unsupported.Construct != testCase.construct ||
				unsupported.Role != testCase.role {
				t.Fatalf("unsupported error = %#v", unsupported)
			}
		})
	}
}

func TestBooleanConstantsUseGoObjectIdentity(t *testing.T) {
	loaded := loadBasicExpressionsProject(t)
	source := sourceFunction(t, loaded.Files()[0].Syntax(), "ShortCircuitAnd")
	logical := source.Body.List[0].(*ast.ReturnStmt).
		Results[0].(*ast.BinaryExpr)
	literal := logical.X.(*ast.Ident)
	literal.Name = "forgedSourceSpelling"

	targetFile := emitBasicExpressions(t, loaded)
	targetLogical := targetReturn(
		t,
		targetFunction(t, targetFile, "ShortCircuitAnd"),
	).Expression().(tsgo.BinaryExpression)
	if targetLogical.Left().Kind() != tsgo.SyntaxKindFalseKeyword {
		t.Fatalf("mutated literal kind = %d, want semantic false object", targetLogical.Left().Kind())
	}
}

func sourceFunction(t *testing.T, file *ast.File, name string) *ast.FuncDecl {
	t.Helper()
	for _, declaration := range file.Decls {
		function, ok := declaration.(*ast.FuncDecl)
		if ok && function.Name.Name == name {
			return function
		}
	}
	t.Fatalf("source function %s not found", name)
	return nil
}

func targetFunction(
	t *testing.T,
	file tsgo.SourceFile,
	name string,
) tsgo.FunctionDeclaration {
	t.Helper()
	for _, statement := range file.Statements() {
		function, ok := statement.(tsgo.FunctionDeclaration)
		if ok && function.Name().Text() == name {
			return function
		}
	}
	t.Fatalf("target function %s not found", name)
	return nil
}

func targetReturn(
	t *testing.T,
	function tsgo.FunctionDeclaration,
) tsgo.ReturnStatement {
	t.Helper()
	statements := function.Body().(tsgo.Block).Statements()
	if len(statements) != 1 {
		t.Fatalf("%s body statements = %d, want one", function.Name().Text(), len(statements))
	}
	result, ok := statements[0].(tsgo.ReturnStatement)
	if !ok {
		t.Fatalf("%s body statement = %T, want return", function.Name().Text(), statements[0])
	}
	return result
}

func loadBasicExpressionsProject(t *testing.T) *load.Package {
	t.Helper()
	loaded, err := load.One(context.Background(), load.Request{
		Directory: basicExpressionsProjectDirectory(),
		Pattern:   ".",
	})
	if err != nil {
		t.Fatal(err)
	}
	return loaded
}

func emitBasicExpressions(
	t *testing.T,
	loaded *load.Package,
) tsgo.SourceFile {
	t.Helper()
	return compileSourceFile(t, loaded, loaded.Files()[0].Syntax())
}

func executeBasicExpressionsGo(t *testing.T, workingDirectory string) string {
	t.Helper()
	modulePath, err := filepath.Abs(basicExpressionsProjectDirectory())
	if err != nil {
		t.Fatal(err)
	}
	runnerDirectory := filepath.Join(workingDirectory, "go-runner")
	if err := os.MkdirAll(runnerDirectory, 0o755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(runnerDirectory, "go.mod"), fmt.Sprintf(`module example.com/runner

go 1.26.4

require example.com/basicexpressions v0.0.0

replace example.com/basicexpressions => %s
`, filepath.ToSlash(modulePath)))
	writeFile(t, filepath.Join(runnerDirectory, "main.go"), `package main

import (
	"fmt"

	expressions "example.com/basicexpressions"
)

func main() {
	fmt.Println(expressions.Arithmetic(10))
	fmt.Println(expressions.Arithmetic(-4))
	fmt.Println(expressions.WrapAdd(40))
	fmt.Println(expressions.WrapAdd(100))
	fmt.Println(expressions.WrapSubtract(40))
	fmt.Println(expressions.WrapSubtract(-100))
	fmt.Println(expressions.WrapMultiply(21))
	fmt.Println(expressions.WrapMultiply(1000))
	fmt.Println(expressions.Increment(100))
	fmt.Println(expressions.Decrement(-100))
	fmt.Println(expressions.Compare(-2147483648, 2147483647))
	fmt.Println(expressions.Compare(7, 7))
	fmt.Println(expressions.Logic(false, false))
	fmt.Println(expressions.Logic(false, true))
	fmt.Println(expressions.Logic(true, false))
	fmt.Println(expressions.Logic(true, true))
	fmt.Println(expressions.ShortCircuitAnd())
	fmt.Println(expressions.ShortCircuitOr())
}
`)
	return run(t, runnerDirectory, filepath.Join(runtime.GOROOT(), "bin", "go"), "run", ".")
}

func executeBasicExpressionsTypeScript(
	t *testing.T,
	loaded *load.Package,
	workingDirectory string,
) string {
	t.Helper()
	artifacts := materializeExportedProgram(t, loaded, workingDirectory)
	runnerPath := filepath.Join(workingDirectory, "runner.ts")
	writeFile(t, runnerPath, `import {
    Arithmetic,
    Compare,
    Decrement,
    Increment,
    Logic,
    ShortCircuitAnd,
    ShortCircuitOr,
    WrapAdd,
    WrapMultiply,
    WrapSubtract,
	} from "`+artifacts.module(t, "source.ts")+`";

console.log(Arithmetic(10));
console.log(Arithmetic(-4));
console.log(WrapAdd(40));
console.log(WrapAdd(100));
console.log(WrapSubtract(40));
console.log(WrapSubtract(-100));
console.log(WrapMultiply(21));
console.log(WrapMultiply(1000));
console.log(Increment(100));
console.log(Decrement(-100));
console.log(...Compare(-2147483648, 2147483647));
console.log(...Compare(7, 7));
console.log(Logic(false, false));
console.log(Logic(false, true));
console.log(Logic(true, false));
console.log(Logic(true, true));
console.log(ShortCircuitAnd());
console.log(ShortCircuitOr());
`)
	return executeMaterializedTypeScript(t, workingDirectory, artifacts, runnerPath)
}

func basicExpressionsProjectDirectory() string {
	return filepath.Join(repositoryRoot(), "testdata", "projects", "basic-expressions")
}
