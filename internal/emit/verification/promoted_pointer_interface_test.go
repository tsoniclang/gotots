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

func TestPromotedPointerInterfaceAdapterPreservesReceiverAddress(
	t *testing.T,
) {
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
				Directory: promotedPointerInterfaceDirectory(),
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
			if !strings.Contains(
				artifacts.printed,
				"$goInterfaceAdapter_",
			) || !strings.Contains(
				artifacts.printed,
				".field<",
			) {
				t.Fatalf(
					"promoted pointer adapter lacks typed field-address projection:\n%s",
					artifacts.printed,
				)
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
			goOutput := executePromotedPointerGo(t, workingDirectory)
			if targetOutput != goOutput {
				t.Fatalf(
					"promoted pointer output differs\nTypeScript:\n%s\nGo:\n%s",
					targetOutput,
					goOutput,
				)
			}
			if artifacts.bytes > 48_000 || artifacts.largest > 24_000 {
				t.Fatalf(
					"promoted pointer artifacts exceed bounds: total=%d largest=%d",
					artifacts.bytes,
					artifacts.largest,
				)
			}
		})
	}
}

func executePromotedPointerGo(
	t *testing.T,
	workingDirectory string,
) string {
	t.Helper()
	modulePath, err := filepath.Abs(promotedPointerInterfaceDirectory())
	if err != nil {
		t.Fatal(err)
	}
	runnerDirectory := filepath.Join(
		workingDirectory,
		"go-runner-promoted-pointer",
	)
	writeProgramFile(
		t,
		filepath.Join(runnerDirectory, "go.mod"),
		fmt.Sprintf(
			`module example.com/runner

go 1.26.4

require example.com/promotedpointer v0.0.0

replace example.com/promotedpointer => %s
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

	values "example.com/promotedpointer"
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

func promotedPointerInterfaceDirectory() string {
	return filepath.Join(
		repositoryRoot(),
		"testdata",
		"constructs",
		"interface",
		"promoted-pointer",
	)
}
