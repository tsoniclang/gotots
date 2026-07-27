package maprepresentation_test

import (
	"fmt"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/tsoniclang/gotots/internal/emit"
)

func TestMapNegativeRuntimeHintMatchesGoUnderIntegerProfiles(t *testing.T) {
	workingDirectory := t.TempDir()
	goOutput := negativeHintGoOutput(t, workingDirectory)
	if goOutput != "1\n" {
		t.Fatalf("Go runtime-negative map hint output = %q, want evaluated once without panic", goOutput)
	}
	for _, profile := range []struct {
		name           string
		representation emit.IntegerRepresentation
	}{
		{"number", emit.IntegerRepresentationNumber},
		{"bigint", emit.IntegerRepresentationBigInt},
	} {
		t.Run(profile.name, func(t *testing.T) {
			loaded := loadMapValuesProject(t)
			roots, err := emit.ExportedAPIRoots(loaded)
			if err != nil {
				t.Fatal(err)
			}
			options := emit.DefaultOptions()
			options.IntegerRepresentation = profile.representation
			emission, err := emit.CompileWithOptions(
				loaded.Program(),
				roots,
				options,
			)
			if err != nil {
				t.Fatal(err)
			}
			directory := t.TempDir()
			artifacts := materialize(t, emission, directory)
			runner := filepath.Join(directory, "negative-hint-runner.ts")
			writeFile(t, runner, `import { SizeEvaluated } from "`+
				artifacts.module(t, "source.ts")+`";
console.log(String(SizeEvaluated()));
`)
			writeFile(t, filepath.Join(directory, "package.json"), "{\"type\":\"module\"}\n")
			strictTypecheckWithRunner(t, artifacts, directory, runner)
			targetOutput := run(
				t,
				directory,
				"node",
				filepath.Join(directory, "out", "negative-hint-runner.js"),
			)
			if targetOutput != goOutput {
				t.Fatalf(
					"runtime-negative map hint TypeScript/Go = %q/%q",
					targetOutput,
					goOutput,
				)
			}
		})
	}
}

func negativeHintGoOutput(t *testing.T, workingDirectory string) string {
	t.Helper()
	modulePath, err := filepath.Abs(mapValuesProjectDirectory())
	if err != nil {
		t.Fatal(err)
	}
	runner := filepath.Join(workingDirectory, "negative-hint-go")
	writeFile(t, filepath.Join(runner, "go.mod"), fmt.Sprintf(`module example.com/runner

go 1.26.4

require example.com/mapvalues v0.0.0

replace example.com/mapvalues => %s
`, filepath.ToSlash(modulePath)))
	writeFile(t, filepath.Join(runner, "main.go"), `package main

import (
	"fmt"

	mapvalues "example.com/mapvalues"
)

func main() {
	fmt.Println(mapvalues.SizeEvaluated())
}
`)
	return run(
		t,
		runner,
		filepath.Join(runtime.GOROOT(), "bin", "go"),
		"run",
		".",
	)
}
