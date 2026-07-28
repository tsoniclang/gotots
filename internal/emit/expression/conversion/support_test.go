package conversion_test

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

func printConversions(
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
			file.PackageName() == "conversion" {
			sourceModule = "./" +
				strings.TrimSuffix(file.OutputPath(), ".ts") +
				".js"
		}
	}
	if sourceModule == "" {
		t.Fatal("conversion source module is absent")
	}
	return targetPaths, sourceModule, printed.String()
}

func runConversionGo(t *testing.T, workingDirectory string) string {
	t.Helper()
	modulePath, err := filepath.Abs(conversionFixtureDirectory())
	if err != nil {
		t.Fatal(err)
	}
	runnerDirectory := filepath.Join(workingDirectory, "go-runner")
	writeFile(t, filepath.Join(runnerDirectory, "go.mod"), `module example.com/runner

go 1.26.4

require example.com/conversion v0.0.0

replace example.com/conversion => `+filepath.ToSlash(modulePath)+`
`)
	writeFile(t, filepath.Join(runnerDirectory, "main.go"), `package main

import (
	"fmt"
	"strconv"

	values "example.com/conversion"
)

func main() {
	fmt.Println(values.NarrowSigned(130))
	fmt.Println(values.NarrowUnsigned(-1))
	fmt.Println(values.Sign32(-1))
	fmt.Println(values.Sign64EvaluatesOnce())
	fmt.Println(values.Widen(-8))
	fmt.Println(strconv.FormatFloat(values.IntegerToFloat64(9007199254740991), 'f', -1, 64))
	fmt.Println(strconv.FormatFloat(float64(values.UnsignedToFloat32(16777217)), 'f', -1, 64))
	fmt.Println(values.FloatToInt8(130.9))
	fmt.Println(values.FloatToUint32(-1.9))
	fmt.Println(values.FloatToInt64(-123.9))
	fmt.Println(strconv.FormatFloat(values.WidenFloat(1.5), 'f', -1, 64))
	fmt.Println(strconv.FormatFloat(float64(values.NarrowFloat(16777217)), 'f', -1, 64))
	wideReal, wideImag := values.WidenComplex(1.5, -2.25)
	fmt.Println(wideReal, wideImag)
	narrowReal, narrowImag := values.NarrowComplex(1.5, -2.25)
	fmt.Println(narrowReal, narrowImag)
	fmt.Println(values.ComplexEvaluatesOnce())
	fmt.Println(values.ConstantInteger())
	fmt.Println(strconv.FormatFloat(float64(values.ConstantFloat()), 'f', -1, 64))
	fmt.Println(real(values.ConstantComplex()), imag(values.ConstantComplex()))
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

func runConversionTypeScript(
	t *testing.T,
	workingDirectory string,
	targetPaths []string,
	sourceModule string,
	suffix string,
) string {
	t.Helper()
	runner := `import * as values from "` + sourceModule + `";
const show = (value: number | bigint): string => String(value);
console.log(show(values.NarrowSigned(130` + suffix + `)));
console.log(show(values.NarrowUnsigned(-1` + suffix + `)));
console.log(show(values.Sign32(-1` + suffix + `)));
console.log(show(values.Sign64EvaluatesOnce()));
console.log(show(values.Widen(-8` + suffix + `)));
console.log(show(values.IntegerToFloat64(9007199254740991` + suffix + `)));
console.log(show(values.UnsignedToFloat32(16777217` + suffix + `)));
console.log(show(values.FloatToInt8(130.9)));
console.log(show(values.FloatToUint32(-1.9)));
console.log(show(values.FloatToInt64(-123.9)));
console.log(show(values.WidenFloat(1.5)));
console.log(show(values.NarrowFloat(16777217)));
const [wideReal, wideImag] = values.WidenComplex(1.5, -2.25);
console.log(wideReal, wideImag);
const [narrowReal, narrowImag] = values.NarrowComplex(1.5, -2.25);
console.log(narrowReal, narrowImag);
console.log(show(values.ComplexEvaluatesOnce()));
console.log(show(values.ConstantInteger()));
console.log(show(values.ConstantFloat()));
const constantComplex = values.ConstantComplex();
console.log(constantComplex.real, constantComplex.imag);
`
	return executeConversionTypeScript(
		t,
		workingDirectory,
		targetPaths,
		runner,
	)
}

func executeConversionTypeScript(
	t *testing.T,
	workingDirectory string,
	targetPaths []string,
	runner string,
) string {
	t.Helper()
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
		t.Fatalf("conversion program failed strict typecheck: %v", err)
	}
	return runCommand(
		t,
		workingDirectory,
		"node",
		filepath.Join(outputDirectory, "runner.js"),
	)
}

func conversionFixtureDirectory() string {
	return filepath.Join(
		repositoryRoot(),
		"testdata",
		"constructs",
		"expression",
		"conversion",
	)
}

func repositoryRoot() string {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		panic("resolve conversion repository root")
	}
	return filepath.Clean(
		filepath.Join(filepath.Dir(file), "..", "..", "..", ".."),
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
