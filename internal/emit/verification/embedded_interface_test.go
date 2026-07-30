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
)

func TestEmbeddedInterfacePromotionUsesInterfaceDispatch(t *testing.T) {
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
				Directory: embeddedInterfaceDirectory(),
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
			adapter := embeddedInterfaceAdapter(t, artifacts.paths)
			for _, required := range []string{
				"$fromStorage(GoPointer.dereference<",
				").Reader;",
				"goInterfaceNonNil",
				".Read($go$recovery)",
			} {
				if !strings.Contains(adapter, required) {
					t.Fatalf(
						"embedded-interface adapter lacks %q:\n%s",
						required,
						adapter,
					)
				}
			}
			if strings.Contains(adapter, "Reader_Read") {
				t.Fatalf(
					"embedded-interface adapter fabricates a static interface method:\n%s",
					adapter,
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
			goOutput := executeEmbeddedInterfaceGo(t, workingDirectory)
			if targetOutput != goOutput {
				t.Fatalf(
					"embedded-interface output differs\nTypeScript:\n%s\nGo:\n%s",
					targetOutput,
					goOutput,
				)
			}
			if artifacts.bytes > 48_000 || artifacts.largest > 24_000 {
				t.Fatalf(
					"embedded-interface artifacts exceed bounds: total=%d largest=%d",
					artifacts.bytes,
					artifacts.largest,
				)
			}
		})
	}
}

func embeddedInterfaceAdapter(t *testing.T, paths []string) string {
	t.Helper()
	for _, path := range paths {
		if !strings.Contains(
			filepath.ToSlash(path),
			"/support/interfaces/adapters/",
		) {
			continue
		}
		content, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		target := string(content)
		if strings.Contains(target, "Holder__from_embeddedinterface") {
			return target
		}
	}
	t.Fatal("embedded-interface Holder adapter is absent")
	return ""
}

func executeEmbeddedInterfaceGo(
	t *testing.T,
	workingDirectory string,
) string {
	t.Helper()
	modulePath, err := filepath.Abs(embeddedInterfaceDirectory())
	if err != nil {
		t.Fatal(err)
	}
	runnerDirectory := filepath.Join(
		workingDirectory,
		"go-runner-embedded-interface",
	)
	writeProgramFile(
		t,
		filepath.Join(runnerDirectory, "go.mod"),
		fmt.Sprintf(
			`module example.com/runner

go 1.26.4

require example.com/embeddedinterface v0.0.0

replace example.com/embeddedinterface => %s
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

	values "example.com/embeddedinterface"
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

func embeddedInterfaceDirectory() string {
	return filepath.Join(
		repositoryRoot(),
		"testdata",
		"constructs",
		"interface",
		"embedded-interface",
	)
}
