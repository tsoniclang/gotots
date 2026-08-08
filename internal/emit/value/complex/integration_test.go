package complex_test

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
	runtimefixture "github.com/tsoniclang/gotots/internal/testfixture/gototsruntime"
)

func TestComplexFamilyExecutesDifferentially(t *testing.T) {
	emission := compileComplex(t)
	workingDirectory := t.TempDir()
	targetPaths, sourceModule, printed := printComplex(
		t,
		workingDirectory,
		emission,
	)
	for _, forbidden := range []string{
		" as any",
		" as unknown",
		"any",
		"unknown",
		".call(",
		".apply(",
		".bind(",
	} {
		if strings.Contains(printed, forbidden) {
			t.Fatalf("complex artifact contains %q:\n%s", forbidden, printed)
		}
	}
	goOutput := runComplexGo(t, workingDirectory)
	targetOutput := runComplexTypeScript(
		t,
		workingDirectory,
		targetPaths,
		sourceModule,
	)
	if targetOutput != goOutput {
		t.Fatalf(
			"TypeScript output differs from Go\nTypeScript:\n%s\nGo:\n%s",
			targetOutput,
			goOutput,
		)
	}
}

func compileComplex(t *testing.T) emit.ProgramEmission {
	t.Helper()
	loaded, err := load.One(context.Background(), load.Request{
		Directory: complexFixtureDirectory(),
		Pattern:   ".",
	})
	if err != nil {
		t.Fatal(err)
	}
	roots, err := emit.ExportedAPIRoots(loaded)
	if err != nil {
		t.Fatal(err)
	}
	emission, err := emit.Compile(loaded.Program(), roots)
	if err != nil {
		t.Fatalf("complex compile failed: %v", err)
	}
	return emission
}

func printComplex(
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
			file.PackageName() == "complexvalues" {
			sourceModule = "./" +
				strings.TrimSuffix(file.OutputPath(), ".ts") +
				".js"
		}
	}
	if sourceModule == "" {
		t.Fatal("complex source module is absent")
	}
	return targetPaths, sourceModule, printed.String()
}

func runComplexGo(t *testing.T, workingDirectory string) string {
	t.Helper()
	modulePath, err := filepath.Abs(complexFixtureDirectory())
	if err != nil {
		t.Fatal(err)
	}
	runnerDirectory := filepath.Join(workingDirectory, "go-runner")
	writeFile(t, filepath.Join(runnerDirectory, "go.mod"), `module example.com/runner

go 1.26.4

require example.com/complexvalues v0.0.0

replace example.com/complexvalues => `+filepath.ToSlash(modulePath)+`
`)
	writeFile(t, filepath.Join(runnerDirectory, "main.go"), `package main

import (
	"fmt"
	"math"
	"strconv"

	values "example.com/complexvalues"
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
		return strconv.FormatFloat(value, 'g', -1, 64)
	}
}

func show(value complex128) string {
	return scalar(real(value)) + "," + scalar(imag(value))
}

func main() {
	a64 := values.Construct64(1.5, -2.25)
	b64 := values.Construct64(-3.75, 4.5)
	a128 := values.Construct128(1.5, -2.25)
	b128 := values.Construct128(-3.75, 4.5)
	fmt.Println(show(complex128(values.Constant64())))
	fmt.Println(show(values.Constant128()))
	fmt.Println(show(values.Imaginary128()))
	fmt.Println(show(values.Folded128()))
	fmt.Println(show(complex128(values.Zero64())))
	fmt.Println(show(values.Zero128()))
	fmt.Println(show(complex128(values.Add64(a64, b64))))
	fmt.Println(show(complex128(values.Subtract64(a64, b64))))
	fmt.Println(show(complex128(values.Multiply64(a64, b64))))
	fmt.Println(show(complex128(values.Divide64(a64, b64))))
	fmt.Println(show(complex128(values.Negate64(a64))))
	fmt.Println(show(complex128(values.Identity64(a64))))
	fmt.Println(values.Equal64(a64, a64), values.NotEqual64(a64, b64))
	fmt.Println(show(values.Add128(a128, b128)))
	fmt.Println(show(values.Subtract128(a128, b128)))
	fmt.Println(show(values.Multiply128(a128, b128)))
	fmt.Println(show(values.Divide128(a128, b128)))
	fmt.Println(show(values.Negate128(a128)))
	fmt.Println(show(values.Identity128(a128)))
	fmt.Println(values.Equal128(a128, a128), values.NotEqual128(a128, b128))
	fmt.Println(scalar(float64(values.Real64(a64))), scalar(float64(values.Imag64(a64))))
	fmt.Println(scalar(values.Real128(a128)), scalar(values.Imag128(a128)))
	fmt.Println(scalar(values.ConstantReal()), scalar(values.ConstantImag()))
	fmt.Println(show(values.ConstructInOrder()), values.ObservedOrder())
	fmt.Println(show(complex128(values.Construct64(0.1, -0.1))))
	zero128 := values.Construct128(0, 0)
	fmt.Println(show(values.Divide128(a128, zero128)))
	signedZero128 := values.Construct128(math.Copysign(0, -1), 0)
	fmt.Println(show(values.Divide128(a128, signedZero128)))
	large128 := values.Construct128(1e308, 1e308)
	fmt.Println(show(values.Divide128(large128, large128)))
	inf128 := values.Construct128(math.Inf(1), 1)
	fmt.Println(show(values.Divide128(inf128, b128)))
	fmt.Println(show(values.Divide128(a128, inf128)))
	nan128 := values.Construct128(math.NaN(), 1)
	fmt.Println(show(values.Divide128(nan128, b128)))
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

func runComplexTypeScript(
	t *testing.T,
	workingDirectory string,
	targetPaths []string,
	sourceModule string,
) string {
	t.Helper()
	runner := `import * as values from "` + sourceModule + `";
const scalar = (value: number): string =>
    Object.is(value, -0) ? "-0" : String(value);
const show = (value: { readonly real: number; readonly imag: number }): string =>
    scalar(value.real) + "," + scalar(value.imag);
const a64 = values.Construct64(1.5, -2.25);
const b64 = values.Construct64(-3.75, 4.5);
const a128 = values.Construct128(1.5, -2.25);
const b128 = values.Construct128(-3.75, 4.5);
console.log(show(values.Constant64()));
console.log(show(values.Constant128()));
console.log(show(values.Imaginary128()));
console.log(show(values.Folded128()));
console.log(show(values.Zero64()));
console.log(show(values.Zero128()));
console.log(show(values.Add64(a64, b64)));
console.log(show(values.Subtract64(a64, b64)));
console.log(show(values.Multiply64(a64, b64)));
console.log(show(values.Divide64(a64, b64)));
console.log(show(values.Negate64(a64)));
console.log(show(values.Identity64(a64)));
console.log(values.Equal64(a64, a64), values.NotEqual64(a64, b64));
console.log(show(values.Add128(a128, b128)));
console.log(show(values.Subtract128(a128, b128)));
console.log(show(values.Multiply128(a128, b128)));
console.log(show(values.Divide128(a128, b128)));
console.log(show(values.Negate128(a128)));
console.log(show(values.Identity128(a128)));
console.log(values.Equal128(a128, a128), values.NotEqual128(a128, b128));
console.log(scalar(values.Real64(a64)), scalar(values.Imag64(a64)));
console.log(scalar(values.Real128(a128)), scalar(values.Imag128(a128)));
console.log(scalar(values.ConstantReal()), scalar(values.ConstantImag()));
console.log(show(values.ConstructInOrder()), values.ObservedOrder());
console.log(show(values.Construct64(0.1, -0.1)));
const zero128 = values.Construct128(0, 0);
console.log(show(values.Divide128(a128, zero128)));
const signedZero128 = values.Construct128(-0, 0);
console.log(show(values.Divide128(a128, signedZero128)));
const large128 = values.Construct128(1e308, 1e308);
console.log(show(values.Divide128(large128, large128)));
const inf128 = values.Construct128(Infinity, 1);
console.log(show(values.Divide128(inf128, b128)));
console.log(show(values.Divide128(a128, inf128)));
const nan128 = values.Construct128(NaN, 1);
console.log(show(values.Divide128(nan128, b128)));
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
	if err := runtimefixture.InstallResolution(workingDirectory, outputDirectory); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := tsgo.Compile(
		ctx,
		repositoryRoot(),
		workingDirectory,
		arguments,
	); err != nil {
		t.Fatalf("complex program failed strict typecheck: %v", err)
	}
	return runCommand(
		t,
		workingDirectory,
		"node",
		filepath.Join(outputDirectory, "runner.js"),
	)
}

func complexFixtureDirectory() string {
	return filepath.Join(
		repositoryRoot(),
		"testdata",
		"constructs",
		"value",
		"complex",
	)
}

func repositoryRoot() string {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		panic("resolve complex repository root")
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
