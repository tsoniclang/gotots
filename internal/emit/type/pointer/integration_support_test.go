package pointer_test

import (
	"context"
	"errors"
	"go/ast"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/tsoniclang/gotots/internal/emit"
	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/load"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

type materializedProgram struct {
	targetPaths []string
	modules     map[string]string
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

func compileTemporaryFunctionSource(t *testing.T, source string) error {
	t.Helper()
	directory := t.TempDir()
	writeFile(
		t,
		filepath.Join(directory, "go.mod"),
		"module example.com/pointerboundary\n\ngo 1.26.4\n",
	)
	writeFile(t, filepath.Join(directory, "source.go"), source)
	loaded, err := load.One(context.Background(), load.Request{
		Directory: directory,
		Pattern:   ".",
	})
	if err != nil {
		t.Fatal(err)
	}
	return compileLoadedPackage(t, loaded)
}

func compileLoadedPackage(t *testing.T, loaded *load.Package) error {
	t.Helper()
	roots, err := emit.ExportedAPIRoots(loaded)
	if err != nil {
		t.Fatal(err)
	}
	_, err = emit.Compile(loaded.Program(), roots)
	return err
}

func assertUnsupported(
	t *testing.T,
	err error,
	role api.Role,
	category api.Category,
	construct string,
) {
	t.Helper()
	var unsupported *api.UnsupportedError
	if !errors.As(err, &unsupported) {
		t.Fatalf("error = %v, want *api.UnsupportedError", err)
	}
	if unsupported.Role != role ||
		unsupported.Category != category ||
		unsupported.Construct != construct {
		t.Fatalf("unsupported = %#v", unsupported)
	}
}

func sourceFunction(
	t *testing.T,
	source *ast.File,
	name string,
) *ast.FuncDecl {
	t.Helper()
	for _, declaration := range source.Decls {
		function, ok := declaration.(*ast.FuncDecl)
		if ok && function.Name.Name == name {
			return function
		}
	}
	t.Fatalf("source function %q is absent", name)
	return nil
}

func targetFunction(
	t *testing.T,
	source tsgo.SourceFile,
	name string,
) tsgo.FunctionDeclaration {
	t.Helper()
	for _, statement := range source.Statements() {
		function, ok := statement.(tsgo.FunctionDeclaration)
		if ok && function.Name().Text() == name {
			return function
		}
	}
	t.Fatalf("target function %q is absent", name)
	return nil
}

func targetReturn(
	t *testing.T,
	function tsgo.FunctionDeclaration,
) tsgo.ReturnStatement {
	t.Helper()
	statements := function.Body().(tsgo.Block).Statements()
	for index := len(statements) - 1; index >= 0; index-- {
		if statement, ok := statements[index].(tsgo.ReturnStatement); ok {
			return statement
		}
	}
	t.Fatalf("target function %q has no return", function.Name().Text())
	return nil
}

func assertDoublingDeltas(t *testing.T, label string, values []int) {
	t.Helper()
	if len(values) != 3 {
		t.Fatalf("%s sample count = %d, want three", label, len(values))
	}
	firstDelta := values[1] - values[0]
	secondDelta := values[2] - values[1]
	if firstDelta <= 0 ||
		secondDelta*10 < firstDelta*17 ||
		secondDelta*10 > firstDelta*23 {
		t.Fatalf(
			"%s = %v; doubling deltas %d/%d are not linear",
			label,
			values,
			firstDelta,
			secondDelta,
		)
	}
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

func repositoryRoot() string {
	return filepath.Join("..", "..", "..", "..")
}
