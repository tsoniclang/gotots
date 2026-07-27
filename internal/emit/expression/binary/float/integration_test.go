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
		panic("resolve float-operators repository root")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", "..", "..", "..", ".."))
}

func operatorsDirectory() string {
	return filepath.Join(repositoryRoot(), "testdata", "constructs", "expression", "float-operators")
}

// TestFloatOperatorsExecuteDifferentially proves float64 arithmetic, ordering,
// equality, and unary negation carry IEEE-754 semantics to TypeScript exactly:
// division by zero yields ±Infinity/NaN (never a panic), NaN compares false
// under every ordering and unequal to itself, and signed zeros compare equal.
func TestFloatOperatorsExecuteDifferentially(t *testing.T) {
	loaded, err := load.One(context.Background(), load.Request{Directory: operatorsDirectory(), Pattern: "."})
	if err != nil {
		t.Fatal(err)
	}
	roots, err := emit.ExportedAPIRoots(loaded)
	if err != nil {
		t.Fatal(err)
	}
	emission, err := emit.Compile(loaded.Program(), roots)
	if err != nil {
		t.Fatalf("float-operators compile failed: %v", err)
	}

	workingDirectory := t.TempDir()
	client, err := tsgo.StartClient(repositoryRoot(), workingDirectory)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = client.Close() })

	var targetPaths []string
	sourceModule := ""
	for _, file := range emission.Files() {
		text, err := client.PrintNode(file.SourceFile(), tsgo.PrintOptions{})
		if err != nil {
			t.Fatal(err)
		}
		targetPath := filepath.Join(workingDirectory, filepath.FromSlash(file.OutputPath()))
		writeFile(t, targetPath, text)
		targetPaths = append(targetPaths, targetPath)
		if file.Kind() == emit.TargetFileSource && filepath.Base(file.OutputPath()) == "source.ts" {
			sourceModule = "./" + strings.TrimSuffix(file.OutputPath(), ".ts") + ".js"
		}
	}

	goOutput := runOperatorsGo(t, workingDirectory)
	tsOutput := runOperatorsTS(t, workingDirectory, targetPaths, sourceModule)
	if tsOutput != goOutput {
		t.Fatalf("TypeScript output = %q, Go output = %q", tsOutput, goOutput)
	}
}

func runOperatorsGo(t *testing.T, workingDirectory string) string {
	t.Helper()
	modulePath, err := filepath.Abs(operatorsDirectory())
	if err != nil {
		t.Fatal(err)
	}
	runnerDirectory := filepath.Join(workingDirectory, "go-runner")
	writeFile(t, filepath.Join(runnerDirectory, "go.mod"), `module example.com/runner

go 1.26.4

require example.com/floatops v0.0.0

replace example.com/floatops => `+filepath.ToSlash(modulePath)+`
`)
	writeFile(t, filepath.Join(runnerDirectory, "main.go"), `package main

import (
	"fmt"
	"math"
	"strconv"

	values "example.com/floatops"
)

func js(f float64) string {
	switch {
	case math.IsInf(f, 1):
		return "Infinity"
	case math.IsInf(f, -1):
		return "-Infinity"
	case math.IsNaN(f):
		return "NaN"
	default:
		return strconv.FormatFloat(f, 'g', -1, 64)
	}
}

func main() {
	nan := values.Div(0, 0)
	fmt.Println(js(values.Add(1.5, 2.5)))
	fmt.Println(js(values.Sub(5, 3)))
	fmt.Println(js(values.Mul(2.5, 4)))
	fmt.Println(js(values.Div(7, 2)))
	fmt.Println(js(values.Div(1, 0)))
	fmt.Println(js(values.Div(-1, 0)))
	fmt.Println(js(nan))
	fmt.Println(js(values.Negate(3.5)))
	fmt.Println(js(values.Identity(-2.25)))
	fmt.Println(js(values.ConstantNeg()))
	fmt.Println(values.Less(1, 2), values.Less(nan, 1))
	fmt.Println(values.LessEqual(2, 2), values.GreaterEqual(2, 2))
	fmt.Println(values.Greater(3, 2), values.Greater(nan, nan))
	fmt.Println(values.Equal(0, math.Copysign(0, -1)), values.Equal(nan, nan))
	fmt.Println(values.NotEqual(nan, nan), values.NotEqual(1, 1))
}
`)
	return runCommand(t, runnerDirectory, filepath.Join(runtime.GOROOT(), "bin", "go"), "run", ".")
}

func runOperatorsTS(t *testing.T, workingDirectory string, targetPaths []string, sourceModule string) string {
	t.Helper()
	runner := `import * as values from "` + sourceModule + `";
const nan = values.Div(0, 0);
console.log(String(values.Add(1.5, 2.5)));
console.log(String(values.Sub(5, 3)));
console.log(String(values.Mul(2.5, 4)));
console.log(String(values.Div(7, 2)));
console.log(String(values.Div(1, 0)));
console.log(String(values.Div(-1, 0)));
console.log(String(nan));
console.log(String(values.Negate(3.5)));
console.log(String(values.Identity(-2.25)));
console.log(String(values.ConstantNeg()));
console.log(values.Less(1, 2), values.Less(nan, 1));
console.log(values.LessEqual(2, 2), values.GreaterEqual(2, 2));
console.log(values.Greater(3, 2), values.Greater(nan, nan));
console.log(values.Equal(0, -0), values.Equal(nan, nan));
console.log(values.NotEqual(nan, nan), values.NotEqual(1, 1));
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
		t.Fatalf("float-operators program failed strict typecheck: %v", err)
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
