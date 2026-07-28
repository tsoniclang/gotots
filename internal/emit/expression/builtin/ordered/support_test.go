package ordered_test

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
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func printOrdered(
	t *testing.T,
	workingDirectory string,
	emission emit.ProgramEmission,
) ([]string, string, string) {
	t.Helper()
	client, err := tsgo.StartClient(repositoryRoot(), workingDirectory)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = client.Close() })
	var targetPaths []string
	var sourceModule string
	var printed strings.Builder
	for _, file := range emission.Files() {
		text, err := client.PrintNode(file.SourceFile(), tsgo.PrintOptions{})
		if err != nil {
			t.Fatal(err)
		}
		printed.WriteString(text)
		path := filepath.Join(
			workingDirectory,
			filepath.FromSlash(file.OutputPath()),
		)
		writeFile(t, path, text)
		targetPaths = append(targetPaths, path)
		if file.Kind() == emit.TargetFileSource &&
			file.PackageName() == "orderedbuiltins" {
			sourceModule = "./" +
				strings.TrimSuffix(file.OutputPath(), ".ts") +
				".js"
		}
	}
	if sourceModule == "" {
		t.Fatal("ordered source module is absent")
	}
	return targetPaths, sourceModule, printed.String()
}

func runOrderedGo(t *testing.T, workingDirectory string) string {
	t.Helper()
	modulePath, err := filepath.Abs(orderedFixtureDirectory())
	if err != nil {
		t.Fatal(err)
	}
	runnerDirectory := filepath.Join(workingDirectory, "go-runner")
	writeFile(t, filepath.Join(runnerDirectory, "go.mod"), `module example.com/runner

go 1.26.4

require example.com/orderedbuiltins v0.0.0

replace example.com/orderedbuiltins => `+filepath.ToSlash(modulePath)+`
`)
	writeFile(t, filepath.Join(runnerDirectory, "main.go"), `package main

import (
	"fmt"
	"math"
	"strconv"

	values "example.com/orderedbuiltins"
)

func scalar(value float64) string {
	switch {
	case math.IsInf(value, 1):
		return "Infinity"
	case math.IsInf(value, -1):
		return "-Infinity"
	case math.IsNaN(value):
		return "NaN"
	case value == 0 && math.Signbit(value):
		return "-0"
	default:
		return strconv.FormatFloat(value, 'f', -1, 64)
	}
}

func main() {
	fmt.Println(values.MaxInt32(-4, 9, 3))
	fmt.Println(values.MinUint64(8, 2, 5))
	fmt.Println(scalar(float64(values.MaxFloat32(float32(math.Copysign(0, -1)), 0, -1))))
	fmt.Println(scalar(values.MinFloat64(math.Copysign(0, -1), 0, 1)))
	fmt.Println(scalar(float64(values.MaxFloat32(float32(math.NaN()), 1, 2))))
	fmt.Println(scalar(values.MinFloat64(math.Inf(1), 7, math.Inf(-1))))
	fmt.Printf("%x\n", values.MaxString("a", string([]byte{0xff}), "z"))
	fmt.Printf("%x\n", values.MinString("a", string([]byte{0xff}), "z"))
	fmt.Println(values.One(12))
	fmt.Println(values.Mixed(4))
	fmt.Println(values.ConstantInteger())
	fmt.Println(scalar(float64(values.ConstantFloat())))
	fmt.Println(values.ConstantString())
	value, order := values.OrderedMax()
	fmt.Println(value, order)
	value, order = values.OrderedWithPrerequisite()
	fmt.Println(value, order)
}
`)
	return runCommand(
		t,
		runnerDirectory,
		filepath.Join(runtime.GOROOT(), "bin", "go"),
		"run",
		".",
	)
}

func runOrderedTypeScript(
	t *testing.T,
	workingDirectory string,
	targetPaths []string,
	sourceModule string,
	suffix string,
) string {
	t.Helper()
	runner := `import * as values from "` + sourceModule + `";
const scalar = (value: number): string =>
    Object.is(value, -0) ? "-0" : String(value);
const hex = (value: string): string =>
    Array.from(value, byte => byte.charCodeAt(0).toString(16).padStart(2, "0")).join("");
console.log(String(values.MaxInt32(-4` + suffix + `, 9` + suffix + `, 3` + suffix + `)));
console.log(String(values.MinUint64(8` + suffix + `, 2` + suffix + `, 5` + suffix + `)));
console.log(scalar(values.MaxFloat32(-0, 0, -1)));
console.log(scalar(values.MinFloat64(-0, 0, 1)));
console.log(scalar(values.MaxFloat32(NaN, 1, 2)));
console.log(scalar(values.MinFloat64(Infinity, 7, -Infinity)));
console.log(hex(values.MaxString("a", String.fromCharCode(255), "z")));
console.log(hex(values.MinString("a", String.fromCharCode(255), "z")));
console.log(String(values.One(12` + suffix + `)));
console.log(String(values.Mixed(4` + suffix + `)));
console.log(String(values.ConstantInteger()));
console.log(scalar(values.ConstantFloat()));
console.log(values.ConstantString());
const [value, order] = values.OrderedMax();
console.log(String(value), order);
const [prerequisiteValue, prerequisiteOrder] = values.OrderedWithPrerequisite();
console.log(String(prerequisiteValue), prerequisiteOrder);
`
	runnerPath := filepath.Join(workingDirectory, "runner.ts")
	writeFile(t, runnerPath, runner)
	writeFile(
		t,
		filepath.Join(workingDirectory, "package.json"),
		"{\"type\":\"module\"}\n",
	)
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
	if err := tsgo.Compile(
		ctx,
		repositoryRoot(),
		workingDirectory,
		arguments,
	); err != nil {
		t.Fatalf("ordered program failed strict typecheck: %v", err)
	}
	return runCommand(
		t,
		workingDirectory,
		"node",
		filepath.Join(outputDirectory, "runner.js"),
	)
}

func orderedFixtureDirectory() string {
	return filepath.Join(
		repositoryRoot(),
		"testdata",
		"constructs",
		"expression",
		"ordered-builtins",
	)
}

func repositoryRoot() string {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		panic("resolve ordered-builtins repository root")
	}
	return filepath.Clean(
		filepath.Join(
			filepath.Dir(file),
			"..",
			"..",
			"..",
			"..",
			"..",
		),
	)
}

func runCommand(
	t *testing.T,
	directory string,
	name string,
	arguments ...string,
) string {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	command := exec.CommandContext(ctx, name, arguments...)
	command.Dir = directory
	command.Env = append(os.Environ(), "GOMEMLIMIT=1GiB")
	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf(
			"%s %s: %v\n%s",
			name,
			strings.Join(arguments, " "),
			err,
			output,
		)
	}
	return string(output)
}

func writeFile(t *testing.T, path string, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}
