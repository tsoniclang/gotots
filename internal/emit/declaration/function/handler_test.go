package function_test

import (
	"context"
	"errors"
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
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

func TestAddConstructCreatesExactTargetTree(t *testing.T) {
	loaded := loadAddProject(t)
	targetFile := emitAdd(t, loaded, filepath.Join(t.TempDir(), "add.ts"))

	statements := targetFile.Statements()
	if len(statements) != 2 {
		t.Fatalf("target statements = %d, want 2", len(statements))
	}
	assertIntTypeImport(t, statements[0], primitiveName(loaded))
	function, ok := statements[1].(tsgo.FunctionDeclaration)
	if !ok {
		t.Fatalf("target declaration = %T, want tsgo.FunctionDeclaration", statements[0])
	}
	if function.Name().Text() != "Add" {
		t.Fatalf("function name = %q, want Add", function.Name().Text())
	}
	if modifiers := function.Modifiers(); len(modifiers) != 1 ||
		modifiers[0].Kind() != tsgo.SyntaxKindExportKeyword {
		t.Fatalf("function modifiers = %v, want one export keyword", modifiers)
	}
	parameters := function.Parameters()
	if len(parameters) != 2 {
		t.Fatalf("parameters = %d, want 2", len(parameters))
	}
	assertIntParameter(t, parameters[0], "left", primitiveName(loaded))
	assertIntParameter(t, parameters[1], "right", primitiveName(loaded))
	assertPrimitiveType(t, function.Type(), primitiveName(loaded))
	body, ok := function.Body().(tsgo.Block)
	if !ok {
		t.Fatalf("function body = %T, want tsgo.Block", function.Body())
	}
	bodyStatements := body.Statements()
	if len(bodyStatements) != 1 {
		t.Fatalf("body statements = %d, want 1", len(bodyStatements))
	}
	returnStatement, ok := bodyStatements[0].(tsgo.ReturnStatement)
	if !ok {
		t.Fatalf("body statement = %T, want tsgo.ReturnStatement", bodyStatements[0])
	}
	binary, ok := returnStatement.Expression().(tsgo.BinaryExpression)
	if !ok {
		t.Fatalf("return expression = %T, want tsgo.BinaryExpression", returnStatement.Expression())
	}
	if binary.OperatorToken().Kind() != tsgo.SyntaxKindPlusToken {
		t.Fatalf("binary operator = %d, want plus", binary.OperatorToken().Kind())
	}
	assertIdentifier(t, binary.Left(), "left")
	assertIdentifier(t, binary.Right(), "right")
}

func TestAddConstructPrintsTypechecksAndExecutesDifferentially(t *testing.T) {
	loaded := loadAddProject(t)
	workingDirectory := t.TempDir()
	outputPath := filepath.Join(workingDirectory, "add.ts")
	targetFile := emitAdd(t, loaded, outputPath)

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
	expected, err := os.ReadFile(filepath.Join(addProjectDirectory(), "expected.ts"))
	if err != nil {
		t.Fatal(err)
	}
	if printed != string(expected) {
		t.Fatalf("printed TypeScript:\n%s\nwant:\n%s", printed, expected)
	}
	if err := os.WriteFile(outputPath, []byte(printed), 0o644); err != nil {
		t.Fatal(err)
	}

	goOutput := executeGo(t, workingDirectory)
	typeScriptOutput := executeTypeScript(t, workingDirectory, outputPath)
	if typeScriptOutput != goOutput {
		t.Fatalf("TypeScript output = %q, Go output = %q", typeScriptOutput, goOutput)
	}
}

func TestAddRejectsUnsupportedBinaryOperator(t *testing.T) {
	loaded := loadAddProject(t)
	function := loaded.Syntax()[0].Decls[0].(*ast.FuncDecl)
	binary := function.Body.List[0].(*ast.ReturnStmt).Results[0].(*ast.BinaryExpr)
	binary.Op = token.SUB

	compiler := emit.New(loaded)
	_, err := compiler.EmitFile(loaded.Syntax()[0], filepath.Join(t.TempDir(), "add.ts"))
	var unsupported *api.UnsupportedError
	if !errors.As(err, &unsupported) {
		t.Fatalf("error = %v, want *api.UnsupportedError", err)
	}
	if unsupported.Category != api.CategoryExpression ||
		unsupported.Construct != "*ast.BinaryExpr" ||
		unsupported.Role != api.RoleReturnResult {
		t.Fatalf("unsupported error = %#v", unsupported)
	}
}

func TestAddReferencesUseGoObjectIdentity(t *testing.T) {
	loaded := loadAddProject(t)
	function := loaded.Syntax()[0].Decls[0].(*ast.FuncDecl)
	binary := function.Body.List[0].(*ast.ReturnStmt).Results[0].(*ast.BinaryExpr)
	binary.X.(*ast.Ident).Name = "forgedSourceSpelling"

	targetFile := emitAdd(t, loaded, filepath.Join(t.TempDir(), "add.ts"))
	targetFunction := targetFile.Statements()[1].(tsgo.FunctionDeclaration)
	targetBody := targetFunction.Body().(tsgo.Block)
	targetReturn := targetBody.Statements()[0].(tsgo.ReturnStatement)
	targetBinary := targetReturn.Expression().(tsgo.BinaryExpression)
	assertIdentifier(t, targetBinary.Left(), "left")
}

func assertIntTypeImport(t *testing.T, statement tsgo.Statement, targetName string) {
	t.Helper()
	declaration, ok := statement.(tsgo.ImportDeclaration)
	if !ok {
		t.Fatalf("first target statement = %T, want tsgo.ImportDeclaration", statement)
	}
	module, ok := declaration.ModuleSpecifier().(tsgo.StringLiteral)
	if !ok || module.Text() != "@tsonic/core/types.js" {
		t.Fatalf("type import module = %T, want @tsonic/core/types.js", declaration.ModuleSpecifier())
	}
	clause := declaration.ImportClause()
	if clause.PhaseModifier() != tsgo.ImportPhaseModifierSyntaxKindTypeKeyword {
		t.Fatalf("import phase = %d, want type", clause.PhaseModifier())
	}
	bindings, ok := clause.NamedBindings().(tsgo.NamedImports)
	if !ok {
		t.Fatalf("named bindings = %T, want tsgo.NamedImports", clause.NamedBindings())
	}
	elements := bindings.Elements()
	if len(elements) != 1 || elements[0].Name().Text() != targetName {
		t.Fatalf("type import elements = %v, want one %s", elements, targetName)
	}
}

func assertIntParameter(
	t *testing.T,
	parameter tsgo.ParameterDeclaration,
	name string,
	targetType string,
) {
	t.Helper()
	identifier, ok := parameter.Name().(tsgo.Identifier)
	if !ok {
		t.Fatalf("parameter name = %T, want tsgo.Identifier", parameter.Name())
	}
	if identifier.Text() != name {
		t.Fatalf("parameter name = %q, want %q", identifier.Text(), name)
	}
	assertPrimitiveType(t, parameter.Type(), targetType)
}

func assertPrimitiveType(t *testing.T, typeNode tsgo.TypeNode, targetName string) {
	t.Helper()
	reference, ok := typeNode.(tsgo.TypeReferenceNode)
	if !ok {
		t.Fatalf("target type = %T, want tsgo.TypeReferenceNode", typeNode)
	}
	name, ok := reference.TypeName().(tsgo.Identifier)
	if !ok || name.Text() != targetName {
		t.Fatalf("target type name = %T, want %s", reference.TypeName(), targetName)
	}
}

func assertIdentifier(t *testing.T, expression tsgo.Expression, name string) {
	t.Helper()
	identifier, ok := expression.(tsgo.Identifier)
	if !ok {
		t.Fatalf("expression = %T, want tsgo.Identifier", expression)
	}
	if identifier.Text() != name {
		t.Fatalf("identifier = %q, want %q", identifier.Text(), name)
	}
}

func loadAddProject(t *testing.T) *load.Package {
	t.Helper()
	loaded, err := load.One(context.Background(), load.Request{
		Directory: addProjectDirectory(),
		Pattern:   ".",
	})
	if err != nil {
		t.Fatal(err)
	}
	return loaded
}

func emitAdd(t *testing.T, loaded *load.Package, outputPath string) tsgo.SourceFile {
	t.Helper()
	compiler := emit.New(loaded)
	targetFile, err := compiler.EmitFile(loaded.Syntax()[0], outputPath)
	if err != nil {
		t.Fatal(err)
	}
	return targetFile
}

func executeGo(t *testing.T, workingDirectory string) string {
	t.Helper()
	modulePath, err := filepath.Abs(addProjectDirectory())
	if err != nil {
		t.Fatal(err)
	}
	runnerDirectory := filepath.Join(workingDirectory, "go-runner")
	if err := os.MkdirAll(runnerDirectory, 0o755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(runnerDirectory, "go.mod"), fmt.Sprintf(`module example.com/runner

go 1.26.4

require example.com/addconstruct v0.0.0

replace example.com/addconstruct => %s
`, filepath.ToSlash(modulePath)))
	writeFile(t, filepath.Join(runnerDirectory, "main.go"), `package main

import (
	"fmt"

	add "example.com/addconstruct"
)

func main() {
	fmt.Println(add.Add(20, 22))
	fmt.Println(add.Add(-7, 2))
}
`)
	return run(t, runnerDirectory, filepath.Join(runtime.GOROOT(), "bin", "go"), "run", ".")
}

func executeTypeScript(t *testing.T, workingDirectory, outputPath string) string {
	t.Helper()
	writeFile(t, filepath.Join(workingDirectory, "package.json"), "{\"type\":\"module\"}\n")
	installTsonicCoreTypes(t, workingDirectory)
	runnerPath := filepath.Join(workingDirectory, "runner.ts")
	writeFile(t, runnerPath, `import { Add } from "./add.js";

console.log(Add(20n, 22n).toString());
console.log(Add(-7n, 2n).toString());
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

func installTsonicCoreTypes(t *testing.T, workingDirectory string) {
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
	writeFile(t, filepath.Join(moduleDirectory, "types.d.ts"), `export type int32 = number;
export type int64 = bigint;
`)
	writeFile(t, filepath.Join(moduleDirectory, "types.js"), "export {};\n")
}

func primitiveName(loaded *load.Package) string {
	if loaded.TypesSizes().Sizeof(types.Typ[types.Int]) == 4 {
		return "int32"
	}
	return "int64"
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

func addProjectDirectory() string {
	return filepath.Join(repositoryRoot(), "testdata", "constructs", "declaration", "function", "add")
}

func repositoryRoot() string {
	return filepath.Join("..", "..", "..", "..")
}
