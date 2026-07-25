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
	"github.com/tsoniclang/gotots/internal/load"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func TestPackageWideCallsPrintTypecheckAndExecuteDifferentially(t *testing.T) {
	loaded := loadBoolMultifileProject(t)
	workingDirectory := t.TempDir()
	targetFiles := emitBoolMultifile(t, loaded, workingDirectory)

	client, err := tsgo.StartClient(repositoryRoot(), workingDirectory)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := client.Close(); err != nil {
			t.Errorf("close TS-Go client: %v", err)
		}
	})
	for baseName, targetFile := range targetFiles {
		printed, err := client.PrintNode(targetFile, tsgo.PrintOptions{})
		if err != nil {
			t.Fatal(err)
		}
		expected, err := os.ReadFile(filepath.Join(
			boolMultifileProjectDirectory(),
			"expected-"+baseName+".ts",
		))
		if err != nil {
			t.Fatal(err)
		}
		if printed != string(expected) {
			t.Fatalf("%s TypeScript:\n%s\nwant:\n%s", baseName, printed, expected)
		}
		if err := os.WriteFile(
			filepath.Join(workingDirectory, baseName+".ts"),
			[]byte(printed),
			0o644,
		); err != nil {
			t.Fatal(err)
		}
	}

	goOutput := executeBoolMultifileGo(t, workingDirectory)
	typeScriptOutput := executeBoolMultifileTypeScript(t, workingDirectory)
	if typeScriptOutput != goOutput {
		t.Fatalf("TypeScript output = %q, Go output = %q", typeScriptOutput, goOutput)
	}
}

func TestPackageWideImportsAreExactAndDeduplicated(t *testing.T) {
	loaded := loadBoolMultifileProject(t)
	targetFiles := emitBoolMultifile(t, loaded, t.TempDir())
	entry := targetFiles["entry"]
	statements := entry.Statements()
	if len(statements) != 5 {
		t.Fatalf("entry statements = %d, want two imports plus three functions", len(statements))
	}
	valueImport, ok := statements[1].(tsgo.ImportDeclaration)
	if !ok {
		t.Fatalf("entry statement 1 = %T, want tsgo.ImportDeclaration", statements[1])
	}
	module := valueImport.ModuleSpecifier().(tsgo.StringLiteral)
	if module.Text() != "./logic.js" {
		t.Fatalf("value import module = %q, want ./logic.js", module.Text())
	}
	clause := valueImport.ImportClause()
	if clause.PhaseModifier() != 0 {
		t.Fatalf("value import phase = %d, want ordinary value import", clause.PhaseModifier())
	}
	bindings := clause.NamedBindings().(tsgo.NamedImports).Elements()
	if len(bindings) != 1 || bindings[0].Name().Text() != "flip" {
		t.Fatalf("value import bindings = %v, want one flip", bindings)
	}
	logicFunction := targetFiles["logic"].Statements()[2].(tsgo.FunctionDeclaration)
	if modifiers := logicFunction.Modifiers(); len(modifiers) != 1 ||
		modifiers[0].Kind() != tsgo.SyntaxKindExportKeyword {
		t.Fatalf("unexported Go function target modifiers = %v, want module export", modifiers)
	}
}

func TestPackageWideImportUsesGoObjectOwnership(t *testing.T) {
	loaded := loadBoolMultifileProject(t)
	var entryFile *ast.File
	for _, sourceFile := range loaded.Files() {
		if filepath.Base(sourceFile.Path()) == "entry.go" {
			entryFile = sourceFile.Syntax()
			break
		}
	}
	if entryFile == nil {
		t.Fatal("entry.go syntax not found")
	}
	runFunction := entryFile.Decls[0].(*ast.FuncDecl)
	call := runFunction.Body.List[0].(*ast.ReturnStmt).Results[0].(*ast.CallExpr)
	call.Fun.(*ast.Ident).Name = "identity"

	compiler := emit.New(loaded)
	targetFile, err := compiler.EmitFile(entryFile, filepath.Join(t.TempDir(), "entry.ts"))
	if err != nil {
		t.Fatal(err)
	}
	valueImport := targetFile.Statements()[1].(tsgo.ImportDeclaration)
	imported := valueImport.ImportClause().NamedBindings().(tsgo.NamedImports).Elements()
	if len(imported) != 1 || imported[0].Name().Text() != "flip" {
		t.Fatalf("value imports = %v, want declaration-owned flip", imported)
	}
	targetRun := targetFile.Statements()[2].(tsgo.FunctionDeclaration)
	targetReturn := targetRun.Body().(tsgo.Block).Statements()[0].(tsgo.ReturnStatement)
	targetCall := targetReturn.Expression().(tsgo.CallExpression)
	if callee := targetCall.Expression().(tsgo.Identifier).Text(); callee != "flip" {
		t.Fatalf("callee = %q, want declaration-owned flip", callee)
	}
}

func TestPackageWideImportsAreCheckoutIndependent(t *testing.T) {
	relocatedProject := filepath.Join(t.TempDir(), "relocated")
	if err := os.MkdirAll(relocatedProject, 0o755); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"go.mod", "entry.go", "logic.go"} {
		source, err := os.ReadFile(filepath.Join(boolMultifileProjectDirectory(), name))
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(relocatedProject, name), source, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	loaded, err := load.One(context.Background(), load.Request{
		Directory: relocatedProject,
		Pattern:   ".",
	})
	if err != nil {
		t.Fatal(err)
	}
	targetFiles := emitBoolMultifile(t, loaded, t.TempDir())
	entryImport := targetFiles["entry"].Statements()[1].(tsgo.ImportDeclaration)
	module := entryImport.ModuleSpecifier().(tsgo.StringLiteral).Text()
	if module != "./logic.js" {
		t.Fatalf("relocated import module = %q, want ./logic.js", module)
	}
}

func loadBoolMultifileProject(t *testing.T) *load.Package {
	t.Helper()
	loaded, err := load.One(context.Background(), load.Request{
		Directory: boolMultifileProjectDirectory(),
		Pattern:   ".",
	})
	if err != nil {
		t.Fatal(err)
	}
	return loaded
}

func emitBoolMultifile(
	t *testing.T,
	loaded *load.Package,
	workingDirectory string,
) map[string]tsgo.SourceFile {
	t.Helper()
	compiler := emit.New(loaded)
	files := loaded.Files()
	targetFiles := make(map[string]tsgo.SourceFile, len(files))
	for _, sourceFile := range files {
		baseName := strings.TrimSuffix(filepath.Base(sourceFile.Path()), ".go")
		targetFile, err := compiler.EmitFile(
			sourceFile.Syntax(),
			filepath.Join(workingDirectory, baseName+".ts"),
		)
		if err != nil {
			t.Fatal(err)
		}
		targetFiles[baseName] = targetFile
	}
	return targetFiles
}

func executeBoolMultifileGo(t *testing.T, workingDirectory string) string {
	t.Helper()
	modulePath, err := filepath.Abs(boolMultifileProjectDirectory())
	if err != nil {
		t.Fatal(err)
	}
	runnerDirectory := filepath.Join(workingDirectory, "go-runner")
	if err := os.MkdirAll(runnerDirectory, 0o755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(runnerDirectory, "go.mod"), fmt.Sprintf(`module example.com/runner

go 1.26.4

require example.com/boolmulti v0.0.0

replace example.com/boolmulti => %s
`, filepath.ToSlash(modulePath)))
	writeFile(t, filepath.Join(runnerDirectory, "main.go"), `package main

import (
	"fmt"

	multi "example.com/boolmulti"
)

func main() {
	fmt.Println(multi.Run(false))
	fmt.Println(multi.Run(true))
	fmt.Println(multi.Again(false))
}
`)
	return run(t, runnerDirectory, filepath.Join(runtime.GOROOT(), "bin", "go"), "run", ".")
}

func executeBoolMultifileTypeScript(t *testing.T, workingDirectory string) string {
	t.Helper()
	writeFile(t, filepath.Join(workingDirectory, "package.json"), "{\"type\":\"module\"}\n")
	installTsonicCoreTypes(t, workingDirectory)
	runnerPath := filepath.Join(workingDirectory, "runner.ts")
	writeFile(t, runnerPath, `import { Again, Run } from "./entry.js";

console.log(Run(false));
console.log(Run(true));
console.log(Again(false));
`)
	outputDirectory := filepath.Join(workingDirectory, "out")
	toolPath := strings.TrimSpace(
		run(t, repositoryRoot(), filepath.Join(runtime.GOROOT(), "bin", "go"), "tool", "-n", "tsgo"),
	)
	run(t, workingDirectory,
		toolPath,
		"--target", "es2022",
		"--module", "nodenext",
		"--moduleResolution", "nodenext",
		"--strict",
		"--outDir", outputDirectory,
		filepath.Join(workingDirectory, "entry.ts"),
		filepath.Join(workingDirectory, "logic.ts"),
		runnerPath,
	)
	return run(t, workingDirectory, "node", filepath.Join(outputDirectory, "runner.js"))
}

func boolMultifileProjectDirectory() string {
	return filepath.Join(repositoryRoot(), "testdata", "projects", "bool-multifile")
}
