package emit_test

import (
	"context"
	"fmt"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/tsoniclang/gotots/internal/emit"
	"github.com/tsoniclang/gotots/internal/load"
)

func TestGenericInterfaceValueKeepsTypeArgumentsAndNil(t *testing.T) {
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
				Directory: genericInterfaceValueDirectory(),
				Pattern:   ".",
			})
			if err != nil {
				t.Fatal(err)
			}
			root, err := emit.NewRoot(
				program.Roots()[0].Types().Scope().Lookup("Audit"),
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
				"Value<int32> | undefined",
				"goInterfaceNonNil<Value<int32>>",
				".Get()",
				"$goCapability_",
				").Clone()",
			} {
				if !strings.Contains(artifacts.printed, required) {
					t.Fatalf(
						"generic-interface artifact lacks %q:\n%s",
						required,
						artifacts.printed,
					)
				}
			}
			for _, forbidden := range []string{".bind(", ".call(", ".apply("} {
				if strings.Contains(artifacts.printed, forbidden) {
					t.Fatalf(
						"generic-interface artifact contains %q:\n%s",
						forbidden,
						artifacts.printed,
					)
				}
			}
			runner := filepath.Join(workingDirectory, "runner.ts")
			writeProgramFile(t, runner, `import "./program.js";
import { Audit } from "`+artifacts.sourceModule+`";

console.log(String(Audit()));
`)
			writeProgramFile(
				t,
				filepath.Join(workingDirectory, "package.json"),
				"{\"type\":\"module\"}\n",
			)
			waveThreeTypecheck(
				t,
				workingDirectory,
				append(artifacts.paths, runner),
			)
			targetOutput := runProgram(
				t,
				workingDirectory,
				"node",
				filepath.Join(workingDirectory, "out", "runner.js"),
			)
			goOutput := executeGenericInterfaceValueGo(t, workingDirectory)
			if targetOutput != goOutput {
				t.Fatalf(
					"generic-interface output differs\nTypeScript:\n%s\nGo:\n%s",
					targetOutput,
					goOutput,
				)
			}
		})
	}
}

func executeGenericInterfaceValueGo(
	t *testing.T,
	workingDirectory string,
) string {
	t.Helper()
	modulePath, err := filepath.Abs(genericInterfaceValueDirectory())
	if err != nil {
		t.Fatal(err)
	}
	runnerDirectory := filepath.Join(
		workingDirectory,
		"go-runner-generic-interface",
	)
	writeProgramFile(
		t,
		filepath.Join(runnerDirectory, "go.mod"),
		fmt.Sprintf(
			`module example.com/runner

go 1.26.4

require example.com/genericinterface v0.0.0

replace example.com/genericinterface => %s
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

	values "example.com/genericinterface"
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

func genericInterfaceValueDirectory() string {
	return filepath.Join(
		repositoryRoot(),
		"testdata",
		"constructs",
		"interface",
		"generic-value",
	)
}
