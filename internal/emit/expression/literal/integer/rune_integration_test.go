package integer_test

import (
	"context"
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
	loaded, err := load.One(context.Background(), load.Request{Directory: runeDirectory(), Pattern: "."})
	if err != nil {
		t.Fatal(err)
	}
	roots, err := emit.ExportedAPIRoots(loaded)
	if err != nil {
		t.Fatal(err)
	}
	emission, err := emit.Compile(loaded.Program(), roots)
	if err != nil {
		t.Fatalf("rune compile failed: %v", err)
	}

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
	for _, forbidden := range []string{" as ", "any", "unknown", ".call(", ".apply(", ".bind("} {
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
