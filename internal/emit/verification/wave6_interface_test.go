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

func TestWaveSixInterfacesCompileThroughThePublicPipeline(t *testing.T) {
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
				Directory: waveSixInterfaceDirectory(),
				Pattern:   ".",
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
				testCase.options,
			)
			if err != nil {
				t.Fatal(err)
			}
			workingDirectory := t.TempDir()
			artifacts := materializeArtifacts(
				t,
				emission,
				workingDirectory,
			)
			if artifacts.bytes > 100_000 || artifacts.largest > 40_000 {
				t.Fatalf(
					"Wave 6 artifact bounds exceeded: total=%d largest=%d",
					artifacts.bytes,
					artifacts.largest,
				)
			}
			assertWaveSixShape(t, artifacts.printed)
			runner := filepath.Join(workingDirectory, "runner.ts")
			writeProgramFile(t, runner, `import "./program.js";
import {
    Audit,
    FailedAssertion,
    UncomparableEquality,
    UnhashableMapKey,
} from "`+artifacts.sourceModule+`";

const output: string[] = [];
const values = Audit();
for (let index = 0; index < values.length; index++) {
    output.push(String(values.get(index)));
}
for (const action of [
    FailedAssertion,
    UncomparableEquality,
    UnhashableMapKey,
]) {
    try {
        action();
        output.push("no-panic");
    } catch {
        output.push("panic");
    }
}
console.log(output.join(" "));
`)
			writeProgramFile(
				t,
				filepath.Join(workingDirectory, "package.json"),
				"{\"type\":\"module\"}\n",
			)
			paths := append(artifacts.paths, runner)
			waveThreeTypecheck(t, workingDirectory, paths)
			targetOutput := runProgram(
				t,
				workingDirectory,
				"node",
				filepath.Join(workingDirectory, "out", "runner.js"),
			)
			goOutput := executeWaveSixGo(t, workingDirectory)
			if targetOutput != goOutput {
				t.Fatalf(
					"Wave 6 output differs\nTypeScript:\n%s\nGo:\n%s",
					targetOutput,
					goOutput,
				)
			}
			t.Logf(
				"Wave 6 artifacts total=%d largest=%d",
				artifacts.bytes,
				artifacts.largest,
			)
			for index, artifact := range artifacts.sizes {
				if index == 20 {
					break
				}
				t.Logf(
					"Wave 6 artifact rank=%d path=%s bytes=%d",
					index+1,
					artifact.path,
					artifact.bytes,
				)
			}
		})
	}
}

func assertWaveSixShape(t *testing.T, printed string) {
	t.Helper()
	for _, required := range []string{
		"abstract class GoInterfaceValue",
		"readonly $go$type",
		"readonly $go$methods",
		"$go$implements(",
		"$go$equal(",
		"$go$hash(",
		"readonly $go$value",
		"Object.freeze(",
		"switch (true)",
		".$is(",
		"$go$type === $goDynamicType_",
	} {
		if !strings.Contains(printed, required) {
			t.Fatalf("Wave 6 artifacts lack %q:\n%s", required, printed)
		}
	}
	if strings.Contains(printed, "instanceof $goInterfaceAdapter_") {
		t.Fatalf(
			"Wave 6 artifacts use constructor identity for Go dynamic types:\n%s",
			printed,
		)
	}
}

func executeWaveSixGo(t *testing.T, workingDirectory string) string {
	t.Helper()
	modulePath, err := filepath.Abs(waveSixInterfaceDirectory())
	if err != nil {
		t.Fatal(err)
	}
	runnerDirectory := filepath.Join(workingDirectory, "go-runner-wave6")
	writeProgramFile(t, filepath.Join(runnerDirectory, "go.mod"), fmt.Sprintf(
		`module example.com/runner

go 1.26.4

require example.com/wave6interfaces v0.0.0

replace example.com/wave6interfaces => %s
`,
		filepath.ToSlash(modulePath),
	))
	writeProgramFile(t, filepath.Join(runnerDirectory, "main.go"), `package main

import (
	"fmt"

	values "example.com/wave6interfaces"
)

func panics(action func()) (result bool) {
	defer func() {
		result = recover() != nil
	}()
	action()
	return false
}

func main() {
	for index, value := range values.Audit() {
		if index != 0 {
			fmt.Print(" ")
		}
		fmt.Print(value)
	}
	for _, action := range []func(){
		values.FailedAssertion,
		values.UncomparableEquality,
		values.UnhashableMapKey,
	} {
		if panics(action) {
			fmt.Print(" panic")
		} else {
			fmt.Print(" no-panic")
		}
	}
	fmt.Println()
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

func waveSixInterfaceDirectory() string {
	return filepath.Join(
		repositoryRoot(),
		"testdata",
		"constructs",
		"interface",
		"wave6",
	)
}
