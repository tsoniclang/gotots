package emit_test

import (
	"context"
	"errors"
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

func TestDemandProgramFailsWithTypedObligationForUnsupportedRoot(t *testing.T) {
	program := loadDemandProgram(t)
	mathPackage := program.PackageByPath("example.com/demand/mathx")
	root, err := emit.NewRoot(
		mathPackage.Types().Scope().Lookup("unsupportedValue"),
	)
	if err != nil {
		t.Fatal(err)
	}
	_, err = emit.Compile(program, []emit.Root{root})
	var scheduleError *emit.ScheduleError
	if !errors.As(err, &scheduleError) ||
		scheduleError.Object != "unsupportedValue" ||
		scheduleError.Reason != "object has no supported source declaration" {
		t.Fatalf("error = %#v, want exact unsupported-root obligation", err)
	}
}

func TestOrdinaryMultiPackageProgramsUseOneDemandEmissionPath(t *testing.T) {
	for _, project := range []struct {
		name       string
		modulePath string
	}{
		{name: "demand-results", modulePath: "example.com/results"},
		{name: "demand-control", modulePath: "example.com/control"},
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
			if len(files) != 2 {
				t.Fatalf("emitted files = %d, want api plus one dependency", len(files))
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
				expectedPath := filepath.Join(
					projectDirectory,
					file.PackageName(),
					"expected.ts",
				)
				expected, err := os.ReadFile(expectedPath)
				if err != nil {
					t.Fatal(err)
				}
				if printed != string(expected) {
					t.Fatalf("%s TypeScript:\n%s\nwant:\n%s", file.PackageName(), printed, expected)
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
