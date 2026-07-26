package emit_test

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/tsoniclang/gotots/internal/emit"
	"github.com/tsoniclang/gotots/internal/load"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func TestDemandProgramSupportsExplicitPackageVariableRoot(t *testing.T) {
	program := loadDemandProgram(t)
	mathPackage := program.PackageByPath("example.com/demand/mathx")
	root, err := emit.NewRoot(
		mathPackage.Types().Scope().Lookup("unsupportedValue"),
	)
	if err != nil {
		t.Fatal(err)
	}
	emission, err := emit.Compile(program, []emit.Root{root})
	if err != nil {
		t.Fatal(err)
	}
	var fields []string
	for _, file := range emission.Files() {
		if file.Kind() != emit.TargetFilePackageState {
			continue
		}
		for _, statement := range file.SourceFile().Statements() {
			class, ok := statement.(tsgo.ClassDeclaration)
			if !ok {
				continue
			}
			for _, member := range class.Members() {
				field, ok := member.(tsgo.PropertyDeclaration)
				if ok {
					fields = append(fields, field.Name().(tsgo.Identifier).Text())
				}
			}
		}
	}
	if strings.Join(fields, ",") != "unsupportedValue" {
		t.Fatalf("package-state fields = %v, want unsupportedValue", fields)
	}
}

func TestOrdinaryMultiPackageProgramsUseOneDemandEmissionPath(t *testing.T) {
	for _, project := range []struct {
		name       string
		modulePath string
		support    string
	}{
		{
			name:       "demand-results",
			modulePath: "example.com/results",
			support:    "scalars-bool-int32.ts",
		},
		{
			name:       "demand-control",
			modulePath: "example.com/control",
			support:    "scalars-int32.ts",
		},
	} {
		t.Run(project.name, func(t *testing.T) {
			projectDirectory := filepath.Join(
				repositoryRoot(),
				"testdata",
				"projects",
				project.name,
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
			files := emission.Files()
			if len(files) != 6 {
				t.Fatalf(
					"emitted files = %d, want source, assembly, program, and scalar modules",
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
						projectDirectory,
						file.PackageName(),
						"expected.ts",
					)
				case emit.TargetFileSupport:
					expectedPath = filepath.Join(
						repositoryRoot(),
						"testdata",
						"support",
						project.support,
					)
				case emit.TargetFilePackageAssembly,
					emit.TargetFileProgramInitialization:
					expectedPath = ""
				default:
					t.Fatalf(
						"unexpected target file %s kind %d",
						file.OutputPath(),
						file.Kind(),
					)
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
				targetPath := filepath.Join(
					workingDirectory,
					filepath.FromSlash(file.OutputPath()),
				)
				writeProgramFile(t, targetPath, printed)
				targetPaths = append(targetPaths, targetPath)
			}
			targetOutput := executeDemandTypeScript(
				t,
				workingDirectory,
				targetPaths,
				files,
			)
			goOutput := executeMultiPackageGo(
				t,
				workingDirectory,
				projectDirectory,
				project.modulePath,
			)
			if targetOutput != goOutput {
				t.Fatalf("TypeScript output = %q, Go output = %q", targetOutput, goOutput)
			}
		})
	}
}

func TestPackageStateBigIntProfileExecutesInitialization(t *testing.T) {
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
	emission, err := emit.CompileWithOptions(
		program,
		roots,
		emit.Options{
			IntegerRepresentation: emit.IntegerRepresentationBigInt,
			EvaluationOrder:       emit.EvaluationOrderDirect,
		},
	)
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
	var assemblyPath string
	var targetPaths []string
	for _, file := range emission.Files() {
		printed, err := client.PrintNode(file.SourceFile(), tsgo.PrintOptions{})
		if err != nil {
			t.Fatal(err)
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
	goOutput := executePackageStateGo(t, projectDirectory, workingDirectory)
	targetOutput := executePackageStateTypeScript(
		t,
		workingDirectory,
		targetPaths,
		assemblyPath,
		true,
	)
	if targetOutput != goOutput {
		t.Fatalf("BigInt TypeScript output = %q, Go output = %q", targetOutput, goOutput)
	}
}

func executeMultiPackageGo(
	t *testing.T,
	workingDirectory string,
	projectDirectory string,
	modulePath string,
) string {
	t.Helper()
	absoluteProject, err := filepath.Abs(projectDirectory)
	if err != nil {
		t.Fatal(err)
	}
	runnerDirectory := filepath.Join(workingDirectory, "go-runner")
	if err := os.MkdirAll(runnerDirectory, 0o755); err != nil {
		t.Fatal(err)
	}
	writeProgramFile(t, filepath.Join(runnerDirectory, "go.mod"), fmt.Sprintf(`module example.com/runner

go 1.26.4

require %s v0.0.0

replace %s => %s
`, modulePath, modulePath, filepath.ToSlash(absoluteProject)))
	writeProgramFile(t, filepath.Join(runnerDirectory, "main.go"), `package main

import (
	"fmt"

	"`+modulePath+`/api"
)

func main() {
	fmt.Println(api.Run(0))
	fmt.Println(api.Run(1))
	fmt.Println(api.Run(4))
}
`)
	return strings.TrimSpace(runProgram(
		t,
		runnerDirectory,
		filepath.Join(runtime.GOROOT(), "bin", "go"),
		"run",
		".",
	)) + "\n"
}
