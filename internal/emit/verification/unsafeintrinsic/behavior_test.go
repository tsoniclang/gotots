package unsafeintrinsic_test

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

func TestUnsafeStringIntrinsicPrintsTypechecksAndMatchesGo(t *testing.T) {
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
			project := writeUnsafeStringProject(t)
			program, err := load.Load(context.Background(), load.Request{
				Directory: project,
				Pattern:   ".",
			})
			if err != nil {
				t.Fatal(err)
			}
			scope := program.Roots()[0].Types().Scope()
			roots := make([]emit.Root, 0, 1)
			for _, name := range []string{"BuildString"} {
				root, rootErr := emit.NewRoot(scope.Lookup(name))
				if rootErr != nil {
					t.Fatal(rootErr)
				}
				roots = append(roots, root)
			}
			emission, err := emit.CompileWithOptions(program, roots, testCase.options)
			if err != nil {
				t.Fatal(err)
			}
			workingDirectory := t.TempDir()
			artifacts := materializeArtifacts(t, emission, workingDirectory)
			assertUnsafeStringRuntimeShape(t, artifacts.printed)
			runner := filepath.Join(workingDirectory, "runner.ts")
			writeProgramFile(t, runner, `import { RuntimeSlice } from "./runtime/slice.js";
import { BuildString } from "`+artifacts.sourceModule+`";

function bytes(value: string): string {
    const result: string[] = [];
    for (let index = 0; index < value.length; index++) {
        result.push(value.charCodeAt(index).toString(16).padStart(2, "0"));
    }
    return result.join("");
}

console.log(bytes(BuildString(RuntimeSlice.literal([255, 65]))));
`)
			writeProgramFile(
				t,
				filepath.Join(workingDirectory, "package.json"),
				"{\"type\":\"module\"}\n",
			)
			waveThreeTypecheck(t, workingDirectory, append(artifacts.paths, runner))
			targetOutput := runProgram(
				t,
				workingDirectory,
				"node",
				filepath.Join(workingDirectory, "out", "runner.js"),
			)
			goOutput := executeUnsafeStringGo(t, project, workingDirectory)
			if targetOutput != goOutput {
				t.Fatalf("unsafe.String output differs\nTypeScript:\n%s\nGo:\n%s", targetOutput, goOutput)
			}
		})
	}
}

func writeUnsafeStringProject(t *testing.T) string {
	t.Helper()
	project := t.TempDir()
	writeProgramFile(
		t,
		filepath.Join(project, "go.mod"),
		"module example.com/unsafestringruntime\n\ngo 1.26.4\n",
	)
	writeProgramFile(t, filepath.Join(project, "source.go"), `package unsafestringruntime

import "unsafe"

func BuildString(bytes []byte) string {
	return unsafe.String(&bytes[0], len(bytes))
}

`)
	return project
}

func assertUnsafeStringRuntimeShape(t *testing.T, printed string) {
	t.Helper()
	for _, required := range []string{
		"goUnsafeString<uint8>(bytes, ",
		"bytes.length",
		"export function goUnsafeString<",
		"globalThis.Number",
	} {
		if !strings.Contains(printed, required) {
			t.Fatalf("unsafe.String runtime lacks %q:\n%s", required, printed)
		}
	}
	for _, forbidden := range []string{
		"GoPointer",
		"GoUnsafePointer",
		"goUnsafeSlice",
		"goUnsafeStringData",
		"goUnsafeSliceData",
	} {
		if strings.Contains(printed, forbidden) {
			t.Fatalf("unsafe.String runtime retained %q:\n%s", forbidden, printed)
		}
	}
}

func executeUnsafeStringGo(t *testing.T, project string, workingDirectory string) string {
	t.Helper()
	runnerDirectory := filepath.Join(workingDirectory, "go-runner-unsafe-string")
	writeProgramFile(t, filepath.Join(runnerDirectory, "go.mod"), fmt.Sprintf(
		`module example.com/runner

go 1.26.4

require example.com/unsafestringruntime v0.0.0

replace example.com/unsafestringruntime => %s
`,
		filepath.ToSlash(project),
	))
	writeProgramFile(t, filepath.Join(runnerDirectory, "main.go"), `package main

import (
	"fmt"

	values "example.com/unsafestringruntime"
)

func main() {
	fmt.Printf("%x\n", values.BuildString([]byte{0xff, 65}))
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
