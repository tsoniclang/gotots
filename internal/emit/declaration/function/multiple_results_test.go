package function_test

import (
	"context"
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

func TestMultipleResultsPrintTypecheckAndExecuteDifferentially(t *testing.T) {
	loaded := loadMultipleResultsProject(t)
	workingDirectory := t.TempDir()
	outputPath := filepath.Join(workingDirectory, "multiple-results.ts")
	targetFile := emitMultipleResultsProject(t, loaded, outputPath)
	printed := printTargetFile(t, targetFile, workingDirectory)

	expected, err := os.ReadFile(filepath.Join(multipleResultsProjectDirectory(), "expected.ts"))
	if err != nil {
		t.Fatal(err)
	}
	if printed != string(expected) {
		t.Fatalf("printed TypeScript:\n%s\nwant:\n%s", printed, expected)
	}
	writeFile(t, outputPath, printed)

	goOutput := executeMultipleResultsGo(t, workingDirectory)
	typeScriptOutput := executeMultipleResultsTypeScript(t, workingDirectory, outputPath)
	if typeScriptOutput != goOutput {
		t.Fatalf("TypeScript output = %q, Go output = %q", typeScriptOutput, goOutput)
	}
}

func TestMultipleResultsCreateDirectTupleTreeAndSingleEvaluation(t *testing.T) {
	loaded := loadMultipleResultsProject(t)
	targetFile := emitMultipleResultsProject(
		t,
		loaded,
		filepath.Join(t.TempDir(), "multiple-results.ts"),
	)
	statements := targetFile.Statements()
	if len(statements) != 10 {
		t.Fatalf("target statements = %d, want import and nine functions", len(statements))
	}

	pair := statements[1].(tsgo.FunctionDeclaration)
	tupleType, ok := pair.Type().(tsgo.TupleTypeNode)
	if !ok || len(tupleType.Elements()) != 2 {
		t.Fatalf("Pair result = %T, want two-element tuple", pair.Type())
	}
	pairReturn := pair.Body().(tsgo.Block).Statements()[0].(tsgo.ReturnStatement)
	literal, ok := pairReturn.Expression().(tsgo.ArrayLiteralExpression)
	if !ok || len(literal.Elements()) != 2 {
		t.Fatalf("Pair return = %T, want two-element array literal", pairReturn.Expression())
	}

	forward := statements[2].(tsgo.FunctionDeclaration)
	forwardReturn := forward.Body().(tsgo.Block).Statements()[0].(tsgo.ReturnStatement)
	if _, ok := forwardReturn.Expression().(tsgo.CallExpression); !ok {
		t.Fatalf("Forward return = %T, want direct call", forwardReturn.Expression())
	}

	consume := statements[3].(tsgo.FunctionDeclaration)
	consumeStatements := consume.Body().(tsgo.Block).Statements()
	if calls := countDirectCalls(consumeStatements, "Pair"); calls != 1 {
		t.Fatalf("Consume Pair calls = %d, want exactly one", calls)
	}
	capture := consumeStatements[0].(tsgo.VariableStatement).
		DeclarationList().Declarations()[0]
	if capture.Name().(tsgo.Identifier).Text() != "__gotots_results_0" {
		t.Fatalf("capture name = %q", capture.Name().(tsgo.Identifier).Text())
	}
	if _, ok := capture.Type().(tsgo.TupleTypeNode); !ok {
		t.Fatalf("capture type = %T, want tuple", capture.Type())
	}

	keepFirst := statements[5].(tsgo.FunctionDeclaration)
	keepStatements := keepFirst.Body().(tsgo.Block).Statements()
	if len(keepStatements) != 3 {
		t.Fatalf("KeepFirst statements = %d, want capture, declaration, return", len(keepStatements))
	}

	addPair := statements[9].(tsgo.FunctionDeclaration)
	addPairStatements := addPair.Body().(tsgo.Block).Statements()
	if len(addPairStatements) != 2 {
		t.Fatalf("AddPair statements = %d, want capture and return", len(addPairStatements))
	}
	if calls := countDirectCalls(addPairStatements, "Numbers"); calls != 1 {
		t.Fatalf("AddPair Numbers calls = %d, want exactly one", calls)
	}
}

func TestMultipleResultNamedReturnRemainsExplicitlyUnsupported(t *testing.T) {
	loaded := loadMultipleResultsProject(t)
	pair := loaded.Files()[0].Syntax().Decls[0].(*ast.FuncDecl)
	pair.Type.Results.List[0].Names = []*ast.Ident{ast.NewIdent("left")}

	_, err := emit.CompileFile(loaded, loaded.Files()[0].Syntax())
	assertUnsupportedCallable(
		t,
		err,
		api.CategoryDeclaration,
		"*ast.Field",
		api.RoleResultType,
	)
}

func TestMultipleResultArityMutationFailsAtCallOwner(t *testing.T) {
	loaded := loadMultipleResultsProject(t)
	consume := loaded.Files()[0].Syntax().Decls[2].(*ast.FuncDecl)
	multipleCall := consume.Body.List[0].(*ast.AssignStmt).Rhs[0]
	consume.Body.List[len(consume.Body.List)-1].(*ast.ReturnStmt).Results[0] = multipleCall

	_, err := emit.CompileFile(loaded, loaded.Files()[0].Syntax())
	assertUnsupportedCallable(
		t,
		err,
		api.CategoryExpression,
		"*ast.CallExpr",
		api.RoleReturnResult,
	)
}

func countDirectCalls(statements []tsgo.Statement, name string) int {
	count := 0
	for _, statement := range statements {
		declaration, ok := statement.(tsgo.VariableStatement)
		if !ok {
			continue
		}
		for _, target := range declaration.DeclarationList().Declarations() {
			call, ok := target.Initializer().(tsgo.CallExpression)
			if !ok {
				continue
			}
			identifier, ok := call.Expression().(tsgo.Identifier)
			if ok && identifier.Text() == name {
				count++
			}
		}
	}
	return count
}

func loadMultipleResultsProject(t *testing.T) *load.Package {
	t.Helper()
	loaded, err := load.One(context.Background(), load.Request{
		Directory: multipleResultsProjectDirectory(),
		Pattern:   ".",
	})
	if err != nil {
		t.Fatal(err)
	}
	return loaded
}

func emitMultipleResultsProject(
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

func executeMultipleResultsGo(t *testing.T, workingDirectory string) string {
	t.Helper()
	modulePath, err := filepath.Abs(multipleResultsProjectDirectory())
	if err != nil {
		t.Fatal(err)
	}
	runnerDirectory := filepath.Join(workingDirectory, "go-runner")
	if err := os.MkdirAll(runnerDirectory, 0o755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(runnerDirectory, "go.mod"), fmt.Sprintf(`module example.com/runner

go 1.26.4

require example.com/multipleresults v0.0.0

replace example.com/multipleresults => %s
`, filepath.ToSlash(modulePath)))
	writeFile(t, filepath.Join(runnerDirectory, "main.go"), `package main

import (
	"fmt"

	results "example.com/multipleresults"
)

func main() {
	fmt.Println(results.Pair(3))
	fmt.Println(results.Forward(-2))
	fmt.Println(results.Consume(4))
	fmt.Println(results.Consume(-4))
	fmt.Println(results.Reassign(7))
	fmt.Println(results.KeepFirst(9))
	fmt.Println(results.Discard(11))
	fmt.Println(results.AddPair(5))
}
`)
	return run(t, runnerDirectory, filepath.Join(runtime.GOROOT(), "bin", "go"), "run", ".")
}

func executeMultipleResultsTypeScript(
	t *testing.T,
	workingDirectory string,
	outputPath string,
) string {
	t.Helper()
	writeFile(t, filepath.Join(workingDirectory, "package.json"), "{\"type\":\"module\"}\n")
	installTsonicCoreTypes(t, workingDirectory)
	runnerPath := filepath.Join(workingDirectory, "runner.ts")
	writeFile(t, runnerPath, `import {
    Consume,
    Discard,
    AddPair,
    Forward,
    KeepFirst,
    Pair,
    Reassign,
} from "./multiple-results.js";

console.log(...Pair(3));
console.log(...Forward(-2));
console.log(Consume(4));
console.log(Consume(-4));
console.log(Reassign(7));
console.log(KeepFirst(9));
console.log(Discard(11));
console.log(AddPair(5));
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

func multipleResultsProjectDirectory() string {
	return filepath.Join(repositoryRoot(), "testdata", "projects", "multiple-results")
}
