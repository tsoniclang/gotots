package function_test

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

func TestAddConstructCreatesExactTargetTree(t *testing.T) {
	loaded := loadAddProject(t)
	targetFile := emitAdd(t, loaded)

	statements := targetFile.Statements()
	if len(statements) != 2 {
		t.Fatalf("target statements = %d, want 2", len(statements))
	}
	assertPrimitiveImportDeclaration(t, statements[0], "int32")
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
	assertIntParameter(t, parameters[0], "left", "int32")
	assertIntParameter(t, parameters[1], "right", "int32")
	assertPrimitiveType(t, function.Type(), "int32")
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
	addition, ok := returnStatement.Expression().(tsgo.BinaryExpression)
	if !ok {
		t.Fatalf("return expression = %T, want direct binary addition", returnStatement.Expression())
	}
	if addition.OperatorToken().Kind() != tsgo.SyntaxKindPlusToken {
		t.Fatalf("binary operator = %d, want plus", addition.OperatorToken().Kind())
	}
	assertIdentifier(t, addition.Left(), "left")
	assertIdentifier(t, addition.Right(), "right")
}

func TestAddConstructPrintsTypechecksAndExecutesDifferentially(t *testing.T) {
	loaded := loadAddProject(t)
	workingDirectory := t.TempDir()
	outputPath := filepath.Join(workingDirectory, "add.ts")
	targetFile := emitAdd(t, loaded)

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
	typeScriptOutput := executeTypeScript(t, loaded, workingDirectory)
	if typeScriptOutput != goOutput {
		t.Fatalf("TypeScript output = %q, Go output = %q", typeScriptOutput, goOutput)
	}
}

func TestAddRejectsIncoherentLogicalOperator(t *testing.T) {
	loaded := loadAddProject(t)
	function := loaded.Files()[0].Syntax().Decls[0].(*ast.FuncDecl)
	binary := function.Body.List[0].(*ast.ReturnStmt).Results[0].(*ast.BinaryExpr)
	binary.Op = token.LAND

	_, err := emit.CompileFile(loaded, loaded.Files()[0].Syntax())
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
	function := loaded.Files()[0].Syntax().Decls[0].(*ast.FuncDecl)
	binary := function.Body.List[0].(*ast.ReturnStmt).Results[0].(*ast.BinaryExpr)
	binary.X.(*ast.Ident).Name = "forgedSourceSpelling"

	targetFile := emitAdd(t, loaded)
	targetFunction := targetFile.Statements()[1].(tsgo.FunctionDeclaration)
	targetBody := targetFunction.Body().(tsgo.Block)
	targetReturn := targetBody.Statements()[0].(tsgo.ReturnStatement)
	targetBinary := targetReturn.Expression().(tsgo.BinaryExpression)
	assertIdentifier(t, targetBinary.Left(), "left")
}

func assertPrimitiveImportDeclaration(
	t *testing.T,
	statement tsgo.Statement,
	targetName string,
) {
	t.Helper()
	declaration, ok := statement.(tsgo.ImportDeclaration)
	if !ok {
		t.Fatalf("first target statement = %T, want tsgo.ImportDeclaration", statement)
	}
	module, ok := declaration.ModuleSpecifier().(tsgo.StringLiteral)
	if !ok || module.Text() != "../../../runtime/scalars.js" {
		t.Fatalf(
			"type import module = %T, want ../../../runtime/scalars.js",
			declaration.ModuleSpecifier(),
		)
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

func emitAdd(t *testing.T, loaded *load.Package) tsgo.SourceFile {
	t.Helper()
	return compileSourceFile(t, loaded, loaded.Files()[0].Syntax())
}

func compileSourceFile(
	t *testing.T,
	loaded *load.Package,
	source *ast.File,
) tsgo.SourceFile {
	t.Helper()
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

func executeTypeScript(
	t *testing.T,
	loaded *load.Package,
	workingDirectory string,
) string {
	t.Helper()
	artifacts := materializeExportedProgram(t, loaded, workingDirectory)
	runnerPath := filepath.Join(workingDirectory, "runner.ts")
	writeFile(t, runnerPath, `import { Add } from "`+artifacts.module(t, "source.ts")+`";

console.log(Add(20, 22).toString());
console.log(Add(-7, 2).toString());
`)
	return executeMaterializedTypeScript(t, workingDirectory, artifacts, runnerPath)
}

type materializedProgram struct {
	targetPaths          []string
	modules              map[string]string
	initializationModule string
}

func materializeExportedProgram(
	t *testing.T,
	loaded *load.Package,
	workingDirectory string,
) materializedProgram {
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
	result := materializedProgram{modules: make(map[string]string)}
	for _, file := range emission.Files() {
		printed, err := client.PrintNode(file.SourceFile(), tsgo.PrintOptions{})
		if err != nil {
			t.Fatal(err)
		}
		targetPath := filepath.Join(
			workingDirectory,
			filepath.FromSlash(file.OutputPath()),
		)
		writeFile(t, targetPath, printed)
		result.targetPaths = append(result.targetPaths, targetPath)
		if file.Kind() == emit.TargetFileSource {
			base := filepath.Base(file.OutputPath())
			if result.modules[base] != "" {
				t.Fatalf("multiple emitted source modules use basename %q", base)
			}
			result.modules[base] = "./" +
				strings.TrimSuffix(file.OutputPath(), ".ts") + ".js"
		} else if file.Kind() == emit.TargetFileProgramInitialization {
			if result.initializationModule != "" {
				t.Fatal("multiple program-initialization modules were emitted")
			}
			result.initializationModule = "./" +
				strings.TrimSuffix(file.OutputPath(), ".ts") + ".js"
		}
	}
	return result
}

func (p materializedProgram) module(t *testing.T, base string) string {
	t.Helper()
	module := p.modules[base]
	if module == "" {
		t.Fatalf("emitted source module %q is absent", base)
	}
	return module
}

func (p materializedProgram) initialization(t *testing.T) string {
	t.Helper()
	if p.initializationModule == "" {
		t.Fatal("program-initialization module is absent")
	}
	return p.initializationModule
}

func executeMaterializedTypeScript(
	t *testing.T,
	workingDirectory string,
	artifacts materializedProgram,
	runnerPath string,
) string {
	t.Helper()
	writeFile(t, filepath.Join(workingDirectory, "package.json"), "{\"type\":\"module\"}\n")
	outputDirectory := filepath.Join(workingDirectory, "out")
	arguments := []string{
		"--target", "es2022",
		"--module", "nodenext",
		"--moduleResolution", "nodenext",
		"--strict",
		"--outDir", outputDirectory,
	}
	arguments = append(arguments, artifacts.targetPaths...)
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

func addProjectDirectory() string {
	return filepath.Join(repositoryRoot(), "testdata", "constructs", "declaration", "function", "add")
}

func repositoryRoot() string {
	return filepath.Join("..", "..", "..", "..", "..")
}
