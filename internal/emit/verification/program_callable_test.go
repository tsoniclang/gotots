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
	runtimefixture "github.com/tsoniclang/gotots/internal/testfixture/gototsruntime"
	corefixture "github.com/tsoniclang/gotots/internal/testfixture/tsoniccore"
)

func TestDemandCompilerAcceptsPackageChannelStorage(t *testing.T) {
	projectDirectory := filepath.Join(
		repositoryRoot(),
		"testdata",
		"constructs",
		"declaration",
		"variable",
		"channel",
	)
	loaded, err := load.One(context.Background(), load.Request{
		Directory: projectDirectory,
		Pattern:   ".",
	})
	if err != nil {
		t.Fatal(err)
	}

	if _, err = emit.CompileFile(loaded, loaded.Files()[0].Syntax()); err != nil {
		t.Fatal(err)
	}
}

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
	if len(files) != 10 {
		t.Fatalf(
			"emitted files = %d, want source, package, state, program, and support modules",
			len(files),
		)
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
		var expectedPath string
		switch file.Kind() {
		case emit.TargetFileSource:
			expectedPath = filepath.Join(
				demandProgramDirectory(),
				file.PackageName(),
				"expected.ts",
			)
		case emit.TargetFileSupport:
			if file.OutputPath() == "runtime/source-fact.ts" {
				expectedPath = filepath.Join(
					demandProgramDirectory(),
					"expected-source-fact.ts",
				)
			} else {
				expectedPath = filepath.Join(repositoryRoot(), "testdata", "support", "scalars-int32.ts")
			}
		case emit.TargetFilePackageState,
			emit.TargetFilePackageAssembly,
			emit.TargetFileProgramInitialization:
			expectedPath = ""
		default:
			t.Fatalf("unexpected target file %s kind %d", file.OutputPath(), file.Kind())
		}
		if expectedPath != "" {
			expected, err := os.ReadFile(expectedPath)
			if err != nil {
				t.Fatal(err)
			}
			if printed != string(expected) {
				t.Fatalf(
					"%s TypeScript:\n%s\nwant:\n%s",
					file.PackageName(),
					printed,
					expected,
				)
			}
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
		if file.Kind() != emit.TargetFileSource {
			continue
		}
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
	if len(files) != 6 {
		t.Fatalf("explicit target files = %v", files)
	}
	var functions []string
	for _, file := range files {
		if file.Kind() != emit.TargetFileSource ||
			file.PackageName() != "mathx" {
			continue
		}
		for _, statement := range file.SourceFile().Statements() {
			if function, ok := statement.(tsgo.FunctionDeclaration); ok {
				functions = append(functions, function.Name().Text())
			}
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
	if err := corefixture.InstallResolutionOnly(workingDirectory); err != nil {
		t.Fatal(err)
	}
	writeProgramFile(t, filepath.Join(workingDirectory, "package.json"), "{\"type\":\"module\"}\n")
	var apiFile emit.TargetFile
	for _, file := range files {
		if file.Kind() == emit.TargetFilePackageAssembly &&
			file.PackageName() == "api" {
			apiFile = file
			break
		}
	}
	if apiFile.SourceFile() == nil {
		t.Fatal("emitted api file is absent")
	}
	runnerPath := filepath.Join(workingDirectory, "runner.ts")
	module := "./" + strings.TrimSuffix(apiFile.OutputPath(), ".ts") + ".js"
	writeProgramFile(t, runnerPath, `import "./program.js";
import { Run } from "`+module+`";

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
	if err := runtimefixture.InstallResolution(workingDirectory, outputDirectory); err != nil {
		t.Fatal(err)
	}
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
	return filepath.Join("..", "..", "..")
}

func TestGenericReceiverMethodWithoutRecoverDefersThroughOrdinaryEntry(t *testing.T) {
	for _, testCase := range []struct {
		name    string
		options emit.Options
	}{
		{name: "number", options: emit.DefaultOptions()},
		{
			name: "bigint",
			options: emit.Options{
				IntegerRepresentation: emit.IntegerRepresentationBigInt,
				EvaluationOrder:       emit.EvaluationOrderPreserveGo,
			},
		},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			program, err := load.Load(context.Background(), load.Request{
				Directory: genericReceiverDeferDirectory(),
				Pattern:   ".",
			})
			if err != nil {
				t.Fatal(err)
			}
			root, err := emit.NewRoot(
				program.Roots()[0].Types().Scope().Lookup(
					"Audit",
				),
			)
			if err != nil {
				t.Fatal(err)
			}
			emission, err := emit.CompileWithOptions(
				program,
				[]emit.Root{root},
				testCase.options,
			)
			if err != nil {
				t.Fatal(err)
			}
			workingDirectory := t.TempDir()
			artifacts := materializeArtifacts(t, emission, workingDirectory)
			for _, required := range []string{
				"export class Box<T>",
				"static $go$private$",
				"export function Box$store$int32",
				"$kernel<int32>($argument0, ($argument0: int32): int32 =>",
				"__gotots_deferred_0",
				"$go$recovery",
			} {
				if !strings.Contains(artifacts.printed, required) {
					t.Fatalf(
						"generic receiver defer lacks %q:\n%s",
						required,
						artifacts.printed,
					)
				}
			}
			for _, forbidden := range []string{
				"__gotots_defers_",
				"export function Box_store(",
				"$kernel$deferred",
				"$deferred($go$recovery",
				"GoDeferredRegistry",
				".bind(",
				".call(",
				".apply(",
			} {
				if strings.Contains(artifacts.printed, forbidden) {
					t.Fatalf(
						"generic receiver defer contains %q:\n%s",
						forbidden,
						artifacts.printed,
					)
				}
			}
			waveThreeTypecheck(t, workingDirectory, artifacts.paths)
		})
	}
}

func executeGenericReceiverDeferGo(
	t *testing.T,
	workingDirectory string,
) string {
	t.Helper()
	modulePath, err := filepath.Abs(genericReceiverDeferDirectory())
	if err != nil {
		t.Fatal(err)
	}
	runnerDirectory := filepath.Join(
		workingDirectory,
		"go-runner-generic-receiver-defer",
	)
	writeProgramFile(
		t,
		filepath.Join(runnerDirectory, "go.mod"),
		fmt.Sprintf(
			`module example.com/runner

go 1.26.4

require example.com/genericreceiverdefer v0.0.0

replace example.com/genericreceiverdefer => %s
`,
			filepath.ToSlash(modulePath),
		),
	)
	writeProgramFile(
		t,
		filepath.Join(runnerDirectory, "main.go"),
		`package main

import (
	"fmt"

	values "example.com/genericreceiverdefer"
)

func main() {
	fmt.Println(values.Audit())
}
`,
	)
	return runProgram(
		t,
		runnerDirectory,
		filepath.Join(runtime.GOROOT(), "bin", "go"),
		"run",
		".",
	)
}

func genericReceiverDeferDirectory() string {
	return filepath.Join(
		repositoryRoot(),
		"testdata",
		"constructs",
		"control",
		"generic-receiver-defer",
	)
}
