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

	"github.com/tsoniclang/gotots/internal/load"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func TestPackageConstantsPrintTypecheckAndExecuteDifferentially(t *testing.T) {
	loaded := loadPackageConstantsProject(t)
	workingDirectory := t.TempDir()
	targetFiles := emitPackageConstantsProject(t, loaded, workingDirectory)
	for name, targetFile := range targetFiles {
		printed := printTargetFile(t, targetFile, workingDirectory)
		expected, err := os.ReadFile(filepath.Join(
			packageConstantsProjectDirectory(),
			"expected-"+name+".ts",
		))
		if err != nil {
			t.Fatal(err)
		}
		if printed != string(expected) {
			t.Fatalf("%s TypeScript:\n%s\nwant:\n%s", name, printed, expected)
		}
		writeFile(t, filepath.Join(workingDirectory, name+".ts"), printed)
	}

	goOutput := runPackageConstantsGo(t, workingDirectory)
	targetOutput := runPackageConstantsTypeScript(t, loaded, workingDirectory)
	if targetOutput != goOutput {
		t.Fatalf("TypeScript output = %q, Go output = %q", targetOutput, goOutput)
	}
}

func TestPackageConstantsCreateOwnedExportedDeclarationsAndImports(t *testing.T) {
	loaded := loadPackageConstantsProject(t)
	targetFiles := emitPackageConstantsProject(t, loaded, t.TempDir())
	constants := targetFiles["constants"].Statements()
	if len(constants) != 3 {
		t.Fatalf("constant statements = %d, want type import and two declarations", len(constants))
	}
	for index, name := range []string{"Base", "Enabled"} {
		statement := constants[index+1].(tsgo.VariableStatement)
		if statement.Modifiers()[0].Kind() != tsgo.SyntaxKindExportKeyword {
			t.Fatalf("%s is not exported", name)
		}
		if statement.DeclarationList().Flags()&tsgo.NodeFlagsConst == 0 {
			t.Fatalf("%s is not const", name)
		}
		declaration := statement.DeclarationList().Declarations()[0]
		if declaration.Name().(tsgo.Identifier).Text() != name {
			t.Fatalf("declaration name = %q, want %q", declaration.Name(), name)
		}
	}

	use := targetFiles["use"].Statements()
	valueImport := use[1].(tsgo.ImportDeclaration)
	if valueImport.ModuleSpecifier().(tsgo.StringLiteral).Text() != "./constants.js" {
		t.Fatalf("constant import module = %q", valueImport.ModuleSpecifier())
	}
}

func TestPackageConstantSpellingMutationKeepsObjectOwnedReference(t *testing.T) {
	loaded := loadPackageConstantsProject(t)
	use := sourceFileNamed(t, loaded, "use.go")
	addBase := use.Decls[0].(*ast.FuncDecl)
	reference := addBase.Body.List[0].(*ast.ReturnStmt).
		Results[0].(*ast.BinaryExpr).X.(*ast.Ident)
	reference.Name = "forged"

	targetFiles := emitPackageConstantsProject(t, loaded, t.TempDir())
	targetUse := targetFiles["use"].Statements()[2].(tsgo.FunctionDeclaration)
	targetReturn := targetUse.Body().(tsgo.Block).Statements()[0].(tsgo.ReturnStatement)
	sum := targetReturn.Expression().(tsgo.BinaryExpression)
	targetReference := sum.Left().(tsgo.Identifier)
	if targetReference.Text() != "Base" {
		t.Fatalf("target reference = %q, want Base", targetReference.Text())
	}
}

func loadPackageConstantsProject(t *testing.T) *load.Package {
	t.Helper()
	loaded, err := load.One(context.Background(), load.Request{
		Directory: packageConstantsProjectDirectory(),
		Pattern:   ".",
	})
	if err != nil {
		t.Fatal(err)
	}
	return loaded
}

func emitPackageConstantsProject(
	t *testing.T,
	loaded *load.Package,
	workingDirectory string,
) map[string]tsgo.SourceFile {
	t.Helper()
	targets := make(map[string]tsgo.SourceFile, len(loaded.Files()))
	for _, file := range loaded.Files() {
		name := strings.TrimSuffix(filepath.Base(file.Path()), ".go")
		target := compileSourceFile(t, loaded, file.Syntax())
		targets[name] = target
	}
	return targets
}

func sourceFileNamed(t *testing.T, loaded *load.Package, name string) *ast.File {
	t.Helper()
	for _, file := range loaded.Files() {
		if filepath.Base(file.Path()) == name {
			return file.Syntax()
		}
	}
	t.Fatalf("source file %s not found", name)
	return nil
}

func runPackageConstantsGo(t *testing.T, workingDirectory string) string {
	t.Helper()
	modulePath, err := filepath.Abs(packageConstantsProjectDirectory())
	if err != nil {
		t.Fatal(err)
	}
	runnerDirectory := filepath.Join(workingDirectory, "go-runner")
	if err := os.MkdirAll(runnerDirectory, 0o755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(runnerDirectory, "go.mod"), fmt.Sprintf(`module example.com/runner

go 1.26.4

require example.com/packageconstants v0.0.0

replace example.com/packageconstants => %s
`, filepath.ToSlash(modulePath)))
	writeFile(t, filepath.Join(runnerDirectory, "main.go"), `package main

import (
	"fmt"

	constants "example.com/packageconstants"
)

func main() {
	fmt.Println(constants.AddBase(2))
	fmt.Println(constants.IsEnabled())
}
`)
	return run(
		t,
		runnerDirectory,
		filepath.Join(runtime.GOROOT(), "bin", "go"),
		"run",
		".",
	)
}

func runPackageConstantsTypeScript(
	t *testing.T,
	loaded *load.Package,
	workingDirectory string,
) string {
	t.Helper()
	artifacts := materializeExportedProgram(t, loaded, workingDirectory)
	runner := filepath.Join(workingDirectory, "runner.ts")
	writeFile(t, runner, `import { AddBase, IsEnabled } from "`+
		artifacts.module(t, "use.ts")+`";

console.log(AddBase(2));
console.log(IsEnabled());
`)
	return executeMaterializedTypeScript(t, workingDirectory, artifacts, runner)
}

func packageConstantsProjectDirectory() string {
	return filepath.Join(repositoryRoot(), "testdata", "projects", "package-constants")
}
