package structvalue_test

import (
	"context"
	"errors"
	"fmt"
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

func TestNamedStructValuesPrintTypecheckAndExecuteDifferentially(t *testing.T) {
	emission := compileStructFixture(t)

	workingDirectory := t.TempDir()
	targetPaths, module := materializeStructProgram(t, workingDirectory, emission)
	runnerPath := filepath.Join(workingDirectory, "runner.ts")
	writeProgramFile(t, runnerPath, `import {
    AssignResult,
    CompositeArgument,
    CompositeField,
    CompositeSecondArgument,
    CopyResult,
    EmptyEqual,
    EqualDifferentResult,
    EqualSameResult,
    ExplicitVarCopyResult,
    GroupedResult,
    MethodResult,
    MultipleResultIsolated,
    NotEqual,
    OmittedComposite,
    ParameterResult,
    ParallelAssignment,
    PositionalComposite,
    PrimitiveZero,
    ReservedValue,
    ZeroIsFresh,
} from "`+module+`";

console.log(ZeroIsFresh());
console.log(CopyResult());
console.log(AssignResult());
console.log(ParameterResult());
console.log(EqualSameResult());
console.log(EqualDifferentResult());
console.log(MethodResult());
console.log(MultipleResultIsolated());
console.log(ReservedValue());
console.log(PrimitiveZero());
console.log(CompositeArgument());
console.log(CompositeSecondArgument());
console.log(CompositeField());
console.log(PositionalComposite());
console.log(OmittedComposite());
console.log(NotEqual());
console.log(ExplicitVarCopyResult());
console.log(ParallelAssignment());
console.log(GroupedResult());
console.log(EmptyEqual());
`)
	writeProgramFile(t, filepath.Join(workingDirectory, "package.json"), "{\"type\":\"module\"}\n")
	targetPaths = append(targetPaths, runnerPath)
	compileStructTypeScript(t, workingDirectory, targetPaths)
	targetModulePath := filepath.Join(
		workingDirectory,
		"out",
		filepath.FromSlash(strings.TrimPrefix(module, "./")),
	)
	runtimeSource, err := os.ReadFile(targetModulePath)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(runtimeSource), "$goType") {
		t.Fatal("erased nominal brand was emitted into JavaScript")
	}
	targetOutput := runProgram(
		t,
		workingDirectory,
		"node",
		filepath.Join(workingDirectory, "out", "runner.js"),
	)
	goOutput := executeStructGo(t, workingDirectory)
	if targetOutput != goOutput {
		t.Fatalf("TypeScript output = %q, Go output = %q", targetOutput, goOutput)
	}
}

func TestNamedStructValuesAreNominalUnderStrictTypeScript(t *testing.T) {
	emission := compileStructFixture(t)
	workingDirectory := t.TempDir()
	targetPaths, module := materializeStructProgram(t, workingDirectory, emission)
	writeProgramFile(t, filepath.Join(workingDirectory, "package.json"), "{\"type\":\"module\"}\n")
	invalidPath := filepath.Join(workingDirectory, "nominal-mutation.ts")
	writeProgramFile(t, invalidPath, `import { Box$zero, Mirror } from "`+module+`";

const invalid: Mirror = Box$zero();
console.log(invalid);
`)
	targetPaths = append(targetPaths, invalidPath)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	arguments := []string{
		"--target", "es2022",
		"--module", "nodenext",
		"--moduleResolution", "nodenext",
		"--strict",
		"--noEmit",
	}
	arguments = append(arguments, targetPaths...)
	err := tsgo.Compile(
		ctx,
		repositoryRoot(),
		workingDirectory,
		arguments,
	)
	var compilerError *tsgo.CompilerError
	if !errors.As(err, &compilerError) ||
		!strings.Contains(compilerError.Diagnostics, "TS2322") ||
		!strings.Contains(compilerError.Diagnostics, "$goType") {
		t.Fatalf("compiler error = %#v, want nominal private-brand rejection", err)
	}
}

func compileStructFixture(t *testing.T) emit.ProgramEmission {
	t.Helper()
	program, err := load.Load(context.Background(), load.Request{
		Directory: structValuesDirectory(),
		Pattern:   ".",
	})
	if err != nil {
		t.Fatal(err)
	}
	roots, err := emit.ExportedAPIRoots(program.Roots()[0])
	if err != nil {
		t.Fatal(err)
	}
	emission, err := emit.Compile(program, roots)
	if err != nil {
		t.Fatal(err)
	}
	return emission
}

func materializeStructProgram(
	t *testing.T,
	workingDirectory string,
	emission emit.ProgramEmission,
) ([]string, string) {
	t.Helper()
	client, err := tsgo.StartClient(repositoryRoot(), workingDirectory)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := client.Close(); err != nil {
			t.Errorf("close TS-Go client: %v", err)
		}
	})
	var targetPaths []string
	var module string
	expected, err := os.ReadFile(
		filepath.Join(structValuesDirectory(), "expected.ts"),
	)
	if err != nil {
		t.Fatal(err)
	}
	for _, file := range emission.Files() {
		printed, err := client.PrintNode(file.SourceFile(), tsgo.PrintOptions{})
		if err != nil {
			t.Fatal(err)
		}
		targetPath := filepath.Join(workingDirectory, filepath.FromSlash(file.OutputPath()))
		writeProgramFile(t, targetPath, printed)
		targetPaths = append(targetPaths, targetPath)
		if file.Kind() == emit.TargetFileSource {
			if module != "" {
				t.Fatal("struct fixture emitted multiple source modules")
			}
			if printed != string(expected) {
				t.Fatalf("printed struct TypeScript:\n%s\nwant:\n%s", printed, expected)
			}
			module = "./" + strings.TrimSuffix(file.OutputPath(), ".ts") + ".js"
		}
	}
	if module == "" {
		t.Fatal("struct fixture emitted no source module")
	}
	return targetPaths, module
}

func structTargetSource(
	t *testing.T,
	emission emit.ProgramEmission,
) tsgo.SourceFile {
	t.Helper()
	var result tsgo.SourceFile
	for _, file := range emission.Files() {
		if file.Kind() != emit.TargetFileSource {
			continue
		}
		if result != nil {
			t.Fatal("struct fixture emitted multiple source modules")
		}
		result = file.SourceFile()
	}
	if result == nil {
		t.Fatal("struct fixture emitted no source module")
	}
	return result
}

func compileStructTypeScript(
	t *testing.T,
	workingDirectory string,
	targetPaths []string,
) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	arguments := []string{
		"--target", "es2022",
		"--module", "nodenext",
		"--moduleResolution", "nodenext",
		"--strict",
		"--outDir", filepath.Join(workingDirectory, "out"),
	}
	arguments = append(arguments, targetPaths...)
	if err := tsgo.Compile(
		ctx,
		repositoryRoot(),
		workingDirectory,
		arguments,
	); err != nil {
		t.Fatal(err)
	}
}

func executeStructGo(t *testing.T, workingDirectory string) string {
	t.Helper()
	modulePath, err := filepath.Abs(structValuesDirectory())
	if err != nil {
		t.Fatal(err)
	}
	runnerDirectory := filepath.Join(workingDirectory, "go-runner")
	writeProgramFile(t, filepath.Join(runnerDirectory, "go.mod"), fmt.Sprintf(`module example.com/runner

go 1.26.4

require example.com/structvalues v0.0.0

replace example.com/structvalues => %s
`, filepath.ToSlash(modulePath)))
	writeProgramFile(t, filepath.Join(runnerDirectory, "main.go"), `package main

import (
	"fmt"

	values "example.com/structvalues"
)

func main() {
	fmt.Println(values.ZeroIsFresh())
	fmt.Println(values.CopyResult())
	fmt.Println(values.AssignResult())
	fmt.Println(values.ParameterResult())
	fmt.Println(values.EqualSameResult())
	fmt.Println(values.EqualDifferentResult())
	fmt.Println(values.MethodResult())
	fmt.Println(values.MultipleResultIsolated())
	fmt.Println(values.ReservedValue())
	fmt.Println(values.PrimitiveZero())
	fmt.Println(values.CompositeArgument())
	fmt.Println(values.CompositeSecondArgument())
	fmt.Println(values.CompositeField())
	fmt.Println(values.PositionalComposite())
	fmt.Println(values.OmittedComposite())
	fmt.Println(values.NotEqual())
	fmt.Println(values.ExplicitVarCopyResult())
	fmt.Println(values.ParallelAssignment())
	fmt.Println(values.GroupedResult())
	fmt.Println(values.EmptyEqual())
}
`)
	return runProgram(
		t,
		runnerDirectory,
		filepath.Join(runtime.GOROOT(), "bin", "go"),
		"run",
		".",
	)
}

func structValuesDirectory() string {
	return filepath.Join(repositoryRoot(), "testdata", "projects", "struct-values")
}

func runProgram(
	t *testing.T,
	directory string,
	name string,
	arguments ...string,
) string {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
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

func writeProgramFile(t *testing.T, path string, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func repositoryRoot() string {
	return filepath.Join("..", "..", "..")
}
