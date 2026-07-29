package emit_test

import (
	"context"
	"fmt"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"testing"

	"github.com/tsoniclang/gotots/internal/emit"
	"github.com/tsoniclang/gotots/internal/load"
)

func TestWaveSevenGenericFoundationCompilesThroughPublicPipeline(t *testing.T) {
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
				Directory: waveSevenGenericDirectory(),
				Pattern:   ".",
			})
			if err != nil {
				t.Fatal(err)
			}
			root, err := emit.NewRoot(
				program.Roots()[0].Types().Scope().Lookup("AuditFunctions"),
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
			artifacts := materializeArtifacts(
				t,
				emission,
				workingDirectory,
			)
			assertWaveSevenGenericFoundationShape(t, artifacts.printed)
			runner := filepath.Join(workingDirectory, "runner.ts")
			writeProgramFile(t, runner, `import "./program.js";
import { AuditFunctions } from "`+artifacts.sourceModule+`";

const values = AuditFunctions();
const output: string[] = [];
for (let index = 0; index < values.length; index++) {
    output.push(String(values.get(index)));
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
			goOutput := executeWaveSevenGenericGo(t, workingDirectory)
			if targetOutput != goOutput {
				t.Fatalf(
					"Wave 7 generic output differs\nTypeScript:\n%s\nGo:\n%s",
					targetOutput,
					goOutput,
				)
			}
		})
	}
}

func assertWaveSevenGenericFoundationShape(t *testing.T, printed string) {
	t.Helper()
	for _, required := range []string{
		"export function Identity<T>",
		"export function Add<T>",
		"export function Zero<T>",
		"export function Equal<T>",
		"export function Twice<T>",
		"$goCapability_",
	} {
		if !strings.Contains(printed, required) {
			t.Fatalf("Wave 7 generic artifacts lack %q:\n%s", required, printed)
		}
	}
	for _, forbidden := range []string{
		"class GoValueOps",
		"interface GoValueOps",
		"Record<string",
		"switch (typeof",
	} {
		if strings.Contains(printed, forbidden) {
			t.Fatalf(
				"Wave 7 generic artifacts contain %q:\n%s",
				forbidden,
				printed,
			)
		}
	}
	if strings.Count(printed, "export function Add<T>") != 1 {
		t.Fatalf("generic Add body was duplicated:\n%s", printed)
	}
	twice := targetGenericFunctionText(t, printed, "Twice")
	copyNames := regexp.MustCompile(`\$go\$copy_[0-9a-f]+`).
		FindAllString(twice, -1)
	if len(copyNames) != 3 ||
		copyNames[0] != copyNames[1] ||
		copyNames[1] != copyNames[2] {
		t.Fatalf(
			"generic forwarding did not exact-join repeated copy capability: %v\n%s",
			copyNames,
			twice,
		)
	}
}

func targetGenericFunctionText(
	t *testing.T,
	printed string,
	name string,
) string {
	t.Helper()
	startMarker := "export function " + name + "<"
	start := strings.Index(printed, startMarker)
	if start < 0 {
		t.Fatalf("Wave 7 artifacts lack generic function %s", name)
	}
	remaining := printed[start+len(startMarker):]
	end := strings.Index(remaining, "\nexport function ")
	if end < 0 {
		return printed[start:]
	}
	return printed[start : start+len(startMarker)+end]
}

func executeWaveSevenGenericGo(t *testing.T, workingDirectory string) string {
	t.Helper()
	modulePath, err := filepath.Abs(waveSevenGenericDirectory())
	if err != nil {
		t.Fatal(err)
	}
	runnerDirectory := filepath.Join(workingDirectory, "go-runner-wave7")
	writeProgramFile(t, filepath.Join(runnerDirectory, "go.mod"), fmt.Sprintf(
		`module example.com/runner

go 1.26.4

require example.com/wave7generics v0.0.0

replace example.com/wave7generics => %s
`,
		filepath.ToSlash(modulePath),
	))
	writeProgramFile(t, filepath.Join(runnerDirectory, "main.go"), `package main

import (
	"fmt"

	values "example.com/wave7generics"
)

func main() {
	for index, value := range values.AuditFunctions() {
		if index != 0 {
			fmt.Print(" ")
		}
		fmt.Print(value)
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

func waveSevenGenericDirectory() string {
	return filepath.Join(
		repositoryRoot(),
		"testdata",
		"constructs",
		"generic",
		"wave7",
	)
}
