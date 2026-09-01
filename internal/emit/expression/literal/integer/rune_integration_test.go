package integer_test

import (
	"bytes"
	"context"
	"go/ast"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/tsoniclang/gotots/internal/emit"
	"github.com/tsoniclang/gotots/internal/load"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
	runtimefixture "github.com/tsoniclang/gotots/internal/testfixture/gototsruntime"
)

func repositoryRoot() string {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		panic("resolve rune repository root")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", "..", "..", "..", ".."))
}

func runeDirectory() string {
	return filepath.Join(repositoryRoot(), "testdata", "constructs", "value", "rune")
}

// TestRuneLiteralExecutesDifferentially proves rune literals — ASCII, escape,
// multi-byte, astral-plane, and a rune constant — materialize as their exact
// int32 code point and execute identically to Go, including a rune literal used
// at an int32 target.
func TestRuneLiteralExecutesDifferentially(t *testing.T) {
	emission := compileRune(t, loadRune(t))

	workingDirectory := t.TempDir()
	client, err := tsgo.StartClient(repositoryRoot(), workingDirectory)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = client.Close() })

	var targetPaths []string
	sourceModule := ""
	var printed strings.Builder
	for _, file := range emission.Files() {
		text, err := client.PrintNode(file.SourceFile(), tsgo.PrintOptions{})
		if err != nil {
			t.Fatal(err)
		}
		printed.WriteString(text)
		targetPath := filepath.Join(workingDirectory, filepath.FromSlash(file.OutputPath()))
		writeFile(t, targetPath, text)
		targetPaths = append(targetPaths, targetPath)
		if file.Kind() == emit.TargetFileSource && filepath.Base(file.OutputPath()) == "source.ts" {
			sourceModule = "./" + strings.TrimSuffix(file.OutputPath(), ".ts") + ".js"
		}
	}
	for _, forbidden := range []string{
		" as any", " as unknown", ": any", ": unknown",
		".call(", ".apply(", ".bind(",
	} {
		if strings.Contains(printed.String(), forbidden) {
			t.Fatalf("rune artifact contains %q:\n%s", forbidden, printed.String())
		}
	}
	if !strings.Contains(printed.String(), "127881") { // '🎉' == U+1F389
		t.Fatalf("astral rune must emit its exact code point:\n%s", printed.String())
	}

	goOutput := runRuneGo(t, workingDirectory)
	tsOutput := runRuneTS(t, workingDirectory, targetPaths, sourceModule)
	if tsOutput != goOutput {
		t.Fatalf("TypeScript output = %q, Go output = %q", tsOutput, goOutput)
	}
}

func TestRuneValuesComeFromCheckerEvidence(t *testing.T) {
	baseline := compileRune(t, loadRune(t))
	mutatedInput := loadRune(t)
	mutated := false
	for _, file := range mutatedInput.Files() {
		ast.Inspect(file.Syntax(), func(node ast.Node) bool {
			literal, ok := node.(*ast.BasicLit)
			if ok && literal.Value == "'A'" {
				// This deliberately poisons syntax after type checking. The
				// retained checker value is still U+0041; a source-spelling
				// materializer would incorrectly emit U+005A.
				literal.Value = "'Z'"
				mutated = true
				return false
			}
			return true
		})
	}
	if !mutated {
		t.Fatal("rune literal mutation target was absent")
	}
	changedSyntax := compileRune(t, mutatedInput)
	assertRuneProgramASTEqual(t, baseline, changedSyntax)

	source := runeSourceFile(t, baseline)
	for _, name := range []string{"ASCII", "EscapedASCII"} {
		value := runeFunctionReturn(t, source, name)
		literal, ok := value.(tsgo.NumericLiteral)
		if !ok || literal.Text() != "65" {
			t.Fatalf("%s returns %T %v, want canonical numeric rune 65", name, value, value)
		}
	}
}

func loadRune(t *testing.T) *load.Package {
	t.Helper()
	loaded, err := load.One(
		context.Background(),
		load.Request{Directory: runeDirectory(), Pattern: "."},
	)
	if err != nil {
		t.Fatal(err)
	}
	return loaded
}

func compileRune(t *testing.T, loaded *load.Package) emit.ProgramEmission {
	t.Helper()
	roots, err := emit.ExportedAPIRoots(loaded)
	if err != nil {
		t.Fatal(err)
	}
	emission, err := emit.Compile(loaded.Program(), roots)
	if err != nil {
		t.Fatalf("rune compile failed: %v", err)
	}
	return emission
}

func runeSourceFile(
	t *testing.T,
	emission emit.ProgramEmission,
) tsgo.SourceFile {
	t.Helper()
	for _, file := range emission.Files() {
		if file.Kind() == emit.TargetFileSource &&
			file.PackageName() == "rune" {
			return file.SourceFile()
		}
	}
	t.Fatal("rune source artifact is absent")
	return nil
}

func runeFunctionReturn(
	t *testing.T,
	source tsgo.SourceFile,
	name string,
) tsgo.Expression {
	t.Helper()
	for _, statement := range source.Statements() {
		function, ok := statement.(tsgo.FunctionDeclaration)
		if !ok || function.Name().Text() != name {
			continue
		}
		for _, bodyStatement := range function.Body().(tsgo.Block).Statements() {
			if returned, ok := bodyStatement.(tsgo.ReturnStatement); ok {
				return returned.Expression()
			}
		}
		t.Fatalf("%s has no target return statement", name)
	}
	t.Fatalf("target function %s is absent", name)
	return nil
}

func assertRuneProgramASTEqual(
	t *testing.T,
	left emit.ProgramEmission,
	right emit.ProgramEmission,
) {
	t.Helper()
	leftFiles := left.Files()
	rightFiles := right.Files()
	if len(leftFiles) != len(rightFiles) {
		t.Fatalf("file counts differ: %d and %d", len(leftFiles), len(rightFiles))
	}
	for index := range leftFiles {
		if leftFiles[index].OutputPath() != rightFiles[index].OutputPath() {
			t.Fatalf(
				"file %d paths differ: %s and %s",
				index,
				leftFiles[index].OutputPath(),
				rightFiles[index].OutputPath(),
			)
		}
		leftBytes, err := tsgo.EncodeSourceFile(leftFiles[index].SourceFile())
		if err != nil {
			t.Fatal(err)
		}
		rightBytes, err := tsgo.EncodeSourceFile(rightFiles[index].SourceFile())
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(leftBytes, rightBytes) {
			t.Fatalf(
				"target AST for %s changed when only stale rune spelling changed",
				leftFiles[index].OutputPath(),
			)
		}
	}
}

func runRuneGo(t *testing.T, workingDirectory string) string {
	t.Helper()
	modulePath, err := filepath.Abs(runeDirectory())
	if err != nil {
		t.Fatal(err)
	}
	runnerDirectory := filepath.Join(workingDirectory, "go-runner")
	writeFile(t, filepath.Join(runnerDirectory, "go.mod"), `module example.com/runner

go 1.26.4

require example.com/rune v0.0.0

replace example.com/rune => `+filepath.ToSlash(modulePath)+`
`)
	writeFile(t, filepath.Join(runnerDirectory, "main.go"), `package main

import (
	"fmt"

	values "example.com/rune"
)

func main() {
	fmt.Println(values.ASCII())
	fmt.Println(values.EscapedASCII())
	fmt.Println(values.Newline())
	fmt.Println(values.Accented())
	fmt.Println(values.CJK())
	fmt.Println(values.Emoji())
	fmt.Println(values.Constant())
	fmt.Println(values.Widened())
}
`)
	return runCommand(t, runnerDirectory, filepath.Join(runtime.GOROOT(), "bin", "go"), "run", ".")
}

func runRuneTS(t *testing.T, workingDirectory string, targetPaths []string, sourceModule string) string {
	t.Helper()
	runner := `import * as values from "` + sourceModule + `";
console.log(String(values.ASCII()));
console.log(String(values.EscapedASCII()));
console.log(String(values.Newline()));
console.log(String(values.Accented()));
console.log(String(values.CJK()));
console.log(String(values.Emoji()));
console.log(String(values.Constant()));
console.log(String(values.Widened()));
`
	runnerPath := filepath.Join(workingDirectory, "runner.ts")
	writeFile(t, runnerPath, runner)
	writeFile(t, filepath.Join(workingDirectory, "package.json"), "{\"type\":\"module\"}\n")
	outputDirectory := filepath.Join(workingDirectory, "out")
	arguments := []string{
		"--target", "es2022",
		"--module", "nodenext",
		"--moduleResolution", "nodenext",
		"--strict",
		"--outDir", outputDirectory,
	}
	arguments = append(arguments, targetPaths...)
	arguments = append(arguments, runnerPath)
	if err := runtimefixture.InstallResolution(workingDirectory, outputDirectory); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := tsgo.Compile(ctx, repositoryRoot(), workingDirectory, arguments); err != nil {
		t.Fatalf("rune program failed strict typecheck: %v", err)
	}
	return runCommand(t, workingDirectory, "node", filepath.Join(outputDirectory, "runner.js"))
}

func runCommand(t *testing.T, directory, name string, arguments ...string) string {
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
