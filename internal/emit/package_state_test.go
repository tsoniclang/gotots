package emit_test

import (
	"bytes"
	"context"
	"fmt"
	"go/types"
	"os"
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

func TestPackageStatePrintsTypechecksAndExecutesCheckerInitializationOrder(
	t *testing.T,
) {
	projectDirectory := filepath.Join(
		repositoryRoot(),
		"testdata",
		"projects",
		"package-state",
	)
	program, err := load.Load(context.Background(), load.Request{
		Directory: projectDirectory,
		Pattern:   "./api",
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

	var apiAssembly emit.TargetFile
	var targetPaths []string
	files := emission.Files()
	if len(files) != 8 {
		t.Fatalf("target files = %d, want eight exact package-state artifacts", len(files))
	}
	for _, file := range files {
		printed, err := client.PrintNode(file.SourceFile(), tsgo.PrintOptions{})
		if err != nil {
			t.Fatal(err)
		}
		expectedPath := packageStateExpectedPath(
			t,
			projectDirectory,
			file,
		)
		expected, err := os.ReadFile(expectedPath)
		if err != nil {
			t.Fatal(err)
		}
		if printed != string(expected) {
			t.Fatalf(
				"%s TypeScript:\n%s\nwant:\n%s",
				file.OutputPath(),
				printed,
				expected,
			)
		}
		if strings.HasSuffix(file.OutputPath(), "/api/package.ts") {
			apiAssembly = file
		}
		targetPath := filepath.Join(
			workingDirectory,
			filepath.FromSlash(file.OutputPath()),
		)
		writeProgramFile(t, targetPath, printed)
		targetPaths = append(targetPaths, targetPath)
	}
	if apiAssembly.SourceFile() == nil {
		t.Fatal("api package assembly is absent")
	}

	goOutput := executePackageStateGo(t, projectDirectory, workingDirectory)
	targetOutput := executePackageStateTypeScript(
		t,
		workingDirectory,
		targetPaths,
		apiAssembly.OutputPath(),
		false,
	)
	if targetOutput != goOutput {
		t.Fatalf("TypeScript output = %q, Go output = %q", targetOutput, goOutput)
	}
}

func packageStateExpectedPath(
	t *testing.T,
	projectDirectory string,
	file emit.TargetFile,
) string {
	t.Helper()
	switch file.Kind() {
	case emit.TargetFileSource:
		return filepath.Join(
			projectDirectory,
			file.PackageName(),
			"expected-source.ts",
		)
	case emit.TargetFilePackageState:
		return filepath.Join(
			projectDirectory,
			file.PackageName(),
			"expected-state.ts",
		)
	case emit.TargetFilePackageAssembly:
		return filepath.Join(
			projectDirectory,
			file.PackageName(),
			"expected-package.ts",
		)
	case emit.TargetFileProgramInitialization:
		return filepath.Join(projectDirectory, "expected-program.ts")
	case emit.TargetFileSupport:
		return filepath.Join(
			repositoryRoot(),
			"testdata",
			"support",
			"scalars-int32.ts",
		)
	default:
		t.Fatalf("unexpected target file kind %d", file.Kind())
		return ""
	}
}

func executePackageStateGo(
	t *testing.T,
	projectDirectory string,
	workingDirectory string,
) string {
	t.Helper()
	absoluteProject, err := filepath.Abs(projectDirectory)
	if err != nil {
		t.Fatal(err)
	}
	runnerDirectory := filepath.Join(workingDirectory, "go-package-state")
	writeProgramFile(t, filepath.Join(runnerDirectory, "go.mod"), fmt.Sprintf(`module example.com/package-state-runner

go 1.26.4

require example.com/package-state v0.0.0

replace example.com/package-state => %s
`, filepath.ToSlash(absoluteProject)))
	writeProgramFile(t, filepath.Join(runnerDirectory, "main.go"), `package main

import (
	"fmt"

	"example.com/package-state/api"
)

func main() {
	fmt.Println(api.Run())
	fmt.Println(api.Run())
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

func executePackageStateTypeScript(
	t *testing.T,
	workingDirectory string,
	targetPaths []string,
	assemblyPath string,
	stringify bool,
) string {
	t.Helper()
	writeProgramFile(
		t,
		filepath.Join(workingDirectory, "package.json"),
		"{\"type\":\"module\"}\n",
	)
	runnerPath := filepath.Join(workingDirectory, "runner.ts")
	modulePath := "./" + strings.TrimSuffix(assemblyPath, ".ts") + ".js"
	runCall := "Run()"
	if stringify {
		runCall = "Run().toString()"
	}
	writeProgramFile(t, runnerPath, `import "./program.js";
import { Run } from "`+modulePath+`";

console.log(`+runCall+`);
console.log(`+runCall+`);
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
	return runProgram(
		t,
		workingDirectory,
		"node",
		filepath.Join(outputDirectory, "runner.js"),
	)
}

func TestPackageInitializationHandlesMultipleResultsInitFunctionsAndBlankImports(
	t *testing.T,
) {
	projectDirectory := filepath.Join(
		repositoryRoot(),
		"testdata",
		"projects",
		"package-initialization",
	)
	program, err := load.Load(context.Background(), load.Request{
		Directory: projectDirectory,
		Pattern:   "./api",
	})
	if err != nil {
		t.Fatal(err)
	}
	assertOrdinaryInitMethodIsSchedulable(t, program)
	roots, err := emit.ExportedAPIRoots(program.Roots()[0])
	if err != nil {
		t.Fatal(err)
	}
	emission, err := emit.Compile(program, roots)
	if err != nil {
		t.Fatal(err)
	}
	reversedRoots := slices.Clone(roots)
	slices.Reverse(reversedRoots)
	reversed, err := emit.Compile(program, reversedRoots)
	if err != nil {
		t.Fatal(err)
	}
	assertSamePackageInitializationAST(t, emission, reversed)

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
	var assemblyPath string
	var targetPaths []string
	files := emission.Files()
	if len(files) != 12 {
		t.Fatalf(
			"target files = %d, want twelve exact initialization artifacts",
			len(files),
		)
	}
	for _, file := range files {
		printed, err := client.PrintNode(file.SourceFile(), tsgo.PrintOptions{})
		if err != nil {
			t.Fatal(err)
		}
		expectedPath := packageInitializationExpectedPath(
			t,
			projectDirectory,
			file,
		)
		expected, err := os.ReadFile(expectedPath)
		if err != nil {
			t.Fatal(err)
		}
		if printed != string(expected) {
			t.Fatalf(
				"%s TypeScript:\n%s\nwant:\n%s",
				file.OutputPath(),
				printed,
				expected,
			)
		}
		targetPath := filepath.Join(
			workingDirectory,
			filepath.FromSlash(file.OutputPath()),
		)
		writeProgramFile(t, targetPath, printed)
		targetPaths = append(targetPaths, targetPath)
		if file.Kind() == emit.TargetFilePackageAssembly &&
			file.PackageName() == "api" {
			assemblyPath = file.OutputPath()
		}
	}
	if assemblyPath == "" {
		t.Fatal("api package assembly is absent")
	}
	goOutput := executePackageInitializationGo(
		t,
		projectDirectory,
		workingDirectory,
	)
	targetOutput := executePackageInitializationTypeScript(
		t,
		workingDirectory,
		targetPaths,
		assemblyPath,
	)
	if targetOutput != goOutput || goOutput != "69134\n" {
		t.Fatalf(
			"TypeScript/Go output = %q/%q, want 69134",
			targetOutput,
			goOutput,
		)
	}
}

func assertOrdinaryInitMethodIsSchedulable(
	t *testing.T,
	program *load.Program,
) {
	t.Helper()
	sink := program.PackageByPath("example.com/package-initialization/sink")
	typeName, ok := sink.Types().Scope().Lookup("marker").(*types.TypeName)
	if !ok {
		t.Fatal("marker type is absent")
	}
	named, ok := types.Unalias(typeName.Type()).(*types.Named)
	if !ok || named.NumMethods() != 1 || named.Method(0).Name() != "init" {
		t.Fatal("ordinary init method is absent")
	}
	root, err := emit.NewRoot(named.Method(0))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := emit.Compile(program, []emit.Root{root}); err != nil {
		t.Fatalf("ordinary receiver method named init was misclassified: %v", err)
	}
}

func assertSamePackageInitializationAST(
	t *testing.T,
	first emit.ProgramEmission,
	second emit.ProgramEmission,
) {
	t.Helper()
	firstFiles := first.Files()
	secondFiles := second.Files()
	if len(firstFiles) != len(secondFiles) {
		t.Fatalf(
			"root-order target counts = %d and %d",
			len(firstFiles),
			len(secondFiles),
		)
	}
	for index := range firstFiles {
		if firstFiles[index].OutputPath() != secondFiles[index].OutputPath() {
			t.Fatalf("root-order path %d differs", index)
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
			t.Fatalf(
				"package initialization AST %s depends on root order",
				firstFiles[index].OutputPath(),
			)
		}
	}
}

func packageInitializationExpectedPath(
	t *testing.T,
	projectDirectory string,
	file emit.TargetFile,
) string {
	t.Helper()
	switch file.Kind() {
	case emit.TargetFileSource:
		name := "expected-source.ts"
		if file.PackageName() == "sideeffect" {
			base := filepath.Base(file.OutputPath())
			name = "expected-" + base
		}
		return filepath.Join(projectDirectory, file.PackageName(), name)
	case emit.TargetFilePackageState:
		return filepath.Join(
			projectDirectory,
			file.PackageName(),
			"expected-state.ts",
		)
	case emit.TargetFilePackageAssembly:
		return filepath.Join(
			projectDirectory,
			file.PackageName(),
			"expected-package.ts",
		)
	case emit.TargetFileProgramInitialization:
		return filepath.Join(projectDirectory, "expected-program.ts")
	case emit.TargetFileSupport:
		return filepath.Join(
			repositoryRoot(),
			"testdata",
			"support",
			"scalars-int32.ts",
		)
	default:
		t.Fatalf("unexpected target file kind %d", file.Kind())
		return ""
	}
}

func executePackageInitializationGo(
	t *testing.T,
	projectDirectory string,
	workingDirectory string,
) string {
	t.Helper()
	absoluteProject, err := filepath.Abs(projectDirectory)
	if err != nil {
		t.Fatal(err)
	}
	runnerDirectory := filepath.Join(workingDirectory, "go-package-initialization")
	writeProgramFile(t, filepath.Join(runnerDirectory, "go.mod"), fmt.Sprintf(`module example.com/package-initialization-runner

go 1.26.4

require example.com/package-initialization v0.0.0

replace example.com/package-initialization => %s
`, filepath.ToSlash(absoluteProject)))
	writeProgramFile(t, filepath.Join(runnerDirectory, "main.go"), `package main

import (
	"fmt"

	"example.com/package-initialization/api"
)

func main() {
	fmt.Println(api.Run())
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

func executePackageInitializationTypeScript(
	t *testing.T,
	workingDirectory string,
	targetPaths []string,
	assemblyPath string,
) string {
	t.Helper()
	writeProgramFile(
		t,
		filepath.Join(workingDirectory, "package.json"),
		"{\"type\":\"module\"}\n",
	)
	runnerPath := filepath.Join(workingDirectory, "runner.ts")
	modulePath := "./" + strings.TrimSuffix(assemblyPath, ".ts") + ".js"
	writeProgramFile(t, runnerPath, `import "./program.js";
import { Run } from "`+modulePath+`";

console.log(Run());
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
	return runProgram(
		t,
		workingDirectory,
		"node",
		filepath.Join(outputDirectory, "runner.js"),
	)
}
