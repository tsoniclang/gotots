package float_test

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
		panic("resolve float repository root")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", "..", "..", ".."))
}

func floatDirectory() string {
	return filepath.Join(repositoryRoot(), "testdata", "constructs", "value", "float")
}

// TestFloatValueFamilyExecutesDifferentially proves float32/float64 constants,
// literals, and zero values materialize as exact TypeScript number literals that
// strict-typecheck and execute identically to Go. It also pins the float32
// rounding: 0.1 at float32 emits the float64 that equals Go's float32(0.1), not
// the shorter float32 spelling that would denote a different number.
func TestFloatValueFamilyExecutesDifferentially(t *testing.T) {
	loaded, err := load.One(context.Background(), load.Request{Directory: floatDirectory(), Pattern: "."})
	if err != nil {
		t.Fatal(err)
	}
	roots, err := emit.ExportedAPIRoots(loaded)
	if err != nil {
		t.Fatal(err)
	}
	emission, err := emit.Compile(loaded.Program(), roots)
	if err != nil {
		t.Fatalf("float compile failed: %v", err)
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
			t.Fatalf("float artifact contains %q:\n%s", forbidden, printed.String())
		}
	}
	if !strings.Contains(printed.String(), "0.10000000149011612") {
		t.Fatalf("float32(0.1) must emit its exact float64 value:\n%s", printed.String())
	}
	if sourceModule == "" {
		t.Fatal("float source module absent")
	}

	goOutput := executeFloatGo(t, workingDirectory)
	tsOutput := executeFloatTS(t, workingDirectory, targetPaths, sourceModule)
	if tsOutput != goOutput {
		t.Fatalf("TypeScript output = %q, Go output = %q", tsOutput, goOutput)
	}
}

func executeFloatGo(t *testing.T, workingDirectory string) string {
	t.Helper()
	modulePath, err := filepath.Abs(floatDirectory())
	if err != nil {
		t.Fatal(err)
	}
	runnerDirectory := filepath.Join(workingDirectory, "go-runner")
	writeFile(t, filepath.Join(runnerDirectory, "go.mod"), `module example.com/runner

go 1.26.4

require example.com/float v0.0.0

replace example.com/float => `+filepath.ToSlash(modulePath)+`
`)
	writeFile(t, filepath.Join(runnerDirectory, "main.go"), `package main

import (
	"fmt"

	values "example.com/float"
)

func main() {
	fmt.Println(values.Constant())
	fmt.Println(float64(values.RoundedConstant()))
	fmt.Println(values.NegativeConstant())
	fmt.Println(values.TypedConstant())
	fmt.Println(values.Literal())
	fmt.Println(values.WholeLiteral())
	fmt.Println(values.Zero())
	fmt.Println(values.Large())
	fmt.Println(values.Subnormal())
	fmt.Println(float64(values.Rounded()))
	fmt.Println(float64(values.LocalConstant()))
}
`)
	return run(t, runnerDirectory, filepath.Join(runtime.GOROOT(), "bin", "go"), "run", ".")
}

func executeFloatTS(t *testing.T, workingDirectory string, targetPaths []string, sourceModule string) string {
	t.Helper()
	runner := `import * as values from "` + sourceModule + `";
console.log(String(values.Constant()));
console.log(String(values.RoundedConstant()));
console.log(String(values.NegativeConstant()));
console.log(String(values.TypedConstant()));
console.log(String(values.Literal()));
console.log(String(values.WholeLiteral()));
console.log(String(values.Zero()));
console.log(String(values.Large()));
console.log(String(values.Subnormal()));
console.log(String(values.Rounded()));
console.log(String(values.LocalConstant()));
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
		t.Fatalf("float program failed strict typecheck: %v", err)
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
