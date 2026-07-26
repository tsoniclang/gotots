package emit_test

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/tsoniclang/gotots/internal/emit"
	"github.com/tsoniclang/gotots/internal/load"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func TestDemandProgramPrintsTypechecksAndExecutesReachableDefinitions(t *testing.T) {
	program := loadDemandProgram(t)
	roots, err := emit.ExportedAPIRoots(program.Roots()[0])
	if err != nil {
		t.Fatal(err)
	}
	if len(roots) != 2 ||
		roots[0].Object().Name() != "Compute" ||
		roots[1].Object().Name() != "Run" {
		t.Fatalf("roots = %v, want exported Compute and Run", roots)
	}
	emission, err := emit.Compile(program, roots)
	if err != nil {
		t.Fatal(err)
	}
	files := emission.Files()
	if len(files) != 4 {
		t.Fatalf("emitted files = %d, want three source modules plus scalar support", len(files))
	}

	workingDirectory := t.TempDir()
	client, err := tsgo.StartClient(repositoryRoot(), workingDirectory)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := client.Close(); err != nil {
			t.Errorf("close TS-Go client: %v", err)
		}
	})
	targetPaths := make([]string, 0, len(files))
	for _, file := range files {
		printed, err := client.PrintNode(file.SourceFile(), tsgo.PrintOptions{})
		if err != nil {
			t.Fatal(err)
		}
		expectedPath := filepath.Join(demandProgramDirectory(), file.PackageName(), "expected.ts")
		if file.Kind() == emit.TargetFileSupport {
			expectedPath = filepath.Join(repositoryRoot(), "testdata", "support", "scalars-int32.ts")
		}
		expected, err := os.ReadFile(expectedPath)
		if err != nil {
			t.Fatal(err)
		}
		if printed != string(expected) {
			t.Fatalf("%s TypeScript:\n%s\nwant:\n%s", file.PackageName(), printed, expected)
		}
		targetPath := filepath.Join(workingDirectory, filepath.FromSlash(file.OutputPath()))
		writeProgramFile(t, targetPath, printed)
		targetPaths = append(targetPaths, targetPath)
	}

	goOutput := executeDemandGo(t, workingDirectory)
	targetOutput := executeDemandTypeScript(t, workingDirectory, targetPaths, files)
	if targetOutput != goOutput {
		t.Fatalf("TypeScript output = %q, Go output = %q", targetOutput, goOutput)
	}
}

func TestDemandProgramPrunesUnreachableDefinitionsAndReservesCycles(t *testing.T) {
	program := loadDemandProgram(t)
	roots, err := emit.ExportedAPIRoots(program.Roots()[0])
	if err != nil {
		t.Fatal(err)
	}
	emission, err := emit.Compile(program, roots)
	if err != nil {
		t.Fatal(err)
	}
	var declarations []string
	for _, file := range emission.Files() {
		for _, statement := range file.SourceFile().Statements() {
			switch statement := statement.(type) {
			case tsgo.FunctionDeclaration:
				declarations = append(declarations, statement.Name().Text())
			case tsgo.VariableStatement:
				for _, declaration := range statement.DeclarationList().Declarations() {
					declarations = append(
						declarations,
						declaration.Name().(tsgo.Identifier).Text(),
					)
				}
			}
		}
	}
	actual := strings.Join(declarations, ",")
	if actual != "Compute,Run,Offset,Even,Odd,Compute" {
		t.Fatalf("emitted declarations = %s", actual)
	}
	for _, forbidden := range []string{
		"unusedAPI",
		"UnusedService",
		"UnusedMath",
		"unusedUntyped",
	} {
		if strings.Contains(actual, forbidden) {
			t.Fatalf("unreachable declaration %s was emitted", forbidden)
		}
	}
}

func TestDemandProgramIsDeterministicAcrossRootOrder(t *testing.T) {
	program := loadDemandProgram(t)
	roots, err := emit.ExportedAPIRoots(program.Roots()[0])
	if err != nil {
		t.Fatal(err)
	}
	first, err := emit.Compile(program, roots)
	if err != nil {
		t.Fatal(err)
	}
	slices.Reverse(roots)
	second, err := emit.Compile(program, roots)
	if err != nil {
		t.Fatal(err)
	}
	firstFiles := first.Files()
	secondFiles := second.Files()
	if len(firstFiles) != len(secondFiles) {
		t.Fatalf("file counts differ: %d and %d", len(firstFiles), len(secondFiles))
	}
	for index := range firstFiles {
		if firstFiles[index].OutputPath() != secondFiles[index].OutputPath() {
			t.Fatalf("path %d differs", index)
		}
		firstBytes, err := tsgo.EncodeSourceFile(firstFiles[index].SourceFile())
		if err != nil {
			t.Fatal(err)
		}
		secondBytes, err := tsgo.EncodeSourceFile(secondFiles[index].SourceFile())
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(firstBytes, secondBytes) {
			t.Fatalf("encoded target %s depends on root order", firstFiles[index].OutputPath())
		}
	}
}

func TestDemandProgramRetainsExplicitReferencedFunctionTarget(t *testing.T) {
	program := loadDemandProgram(t)
	mathPackage := program.PackageByPath("example.com/demand/mathx")
	root, err := emit.NewRoot(
		mathPackage.Types().Scope().Lookup("UnusedMath"),
	)
	if err != nil {
		t.Fatal(err)
	}
	emission, err := emit.Compile(program, []emit.Root{root})
	if err != nil {
		t.Fatal(err)
	}
	files := emission.Files()
	if len(files) != 2 ||
		files[0].PackageName() != "mathx" ||
		files[1].Kind() != emit.TargetFileSupport {
		t.Fatalf("explicit target files = %v", files)
	}
	var functions []string
	for _, statement := range files[0].SourceFile().Statements() {
		if function, ok := statement.(tsgo.FunctionDeclaration); ok {
			functions = append(functions, function.Name().Text())
		}
	}
	if strings.Join(functions, ",") != "UnusedMath" {
		t.Fatalf("explicit target declarations = %v", functions)
	}
}

func loadDemandProgram(t *testing.T) *load.Program {
	t.Helper()
	program, err := load.Load(context.Background(), load.Request{
		Directory: demandProgramDirectory(),
		Pattern:   "./api",
	})
	if err != nil {
		t.Fatal(err)
	}
	return program
}

func executeDemandGo(t *testing.T, workingDirectory string) string {
	t.Helper()
	modulePath, err := filepath.Abs(demandProgramDirectory())
	if err != nil {
		t.Fatal(err)
	}
	runnerDirectory := filepath.Join(workingDirectory, "go-runner")
	if err := os.MkdirAll(runnerDirectory, 0o755); err != nil {
		t.Fatal(err)
	}
	writeProgramFile(t, filepath.Join(runnerDirectory, "go.mod"), fmt.Sprintf(`module example.com/runner

go 1.26.4

require example.com/demand v0.0.0

replace example.com/demand => %s
`, filepath.ToSlash(modulePath)))
	writeProgramFile(t, filepath.Join(runnerDirectory, "main.go"), `package main

import (
	"fmt"

	"example.com/demand/api"
)

func main() {
	fmt.Println(api.Run(0))
	fmt.Println(api.Run(1))
	fmt.Println(api.Run(4))
}
`)
	return runProgram(t, runnerDirectory, filepath.Join(runtime.GOROOT(), "bin", "go"), "run", ".")
}

func executeDemandTypeScript(
	t *testing.T,
	workingDirectory string,
	targetPaths []string,
	files []emit.TargetFile,
) string {
	t.Helper()
	writeProgramFile(t, filepath.Join(workingDirectory, "package.json"), "{\"type\":\"module\"}\n")
	var apiFile emit.TargetFile
	for _, file := range files {
		if file.Kind() == emit.TargetFileSource && file.PackageName() == "api" {
			apiFile = file
			break
		}
	}
	if apiFile.SourceFile() == nil {
		t.Fatal("emitted api file is absent")
	}
	runnerPath := filepath.Join(workingDirectory, "runner.ts")
	module := "./" + strings.TrimSuffix(apiFile.OutputPath(), ".ts") + ".js"
	writeProgramFile(t, runnerPath, `import { Run } from "`+module+`";

console.log(Run(0));
console.log(Run(1));
	console.log(Run(4));
`)
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
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	if err := tsgo.Compile(
		ctx,
		repositoryRoot(),
		workingDirectory,
		arguments,
	); err != nil {
		t.Fatal(err)
	}
	return runProgram(t, workingDirectory, "node", filepath.Join(outputDirectory, "runner.js"))
}

func runProgram(t *testing.T, directory, name string, arguments ...string) string {
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

func writeProgramFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func demandProgramDirectory() string {
	return filepath.Join(repositoryRoot(), "testdata", "projects", "demand-program")
}

func repositoryRoot() string {
	return filepath.Join("..", "..")
}
