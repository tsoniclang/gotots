package maprepresentation_test

import (
	"context"
	"fmt"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/tsoniclang/gotots/internal/load"
)

func TestMapValuesCrossPackageTypecheckAndExecuteDifferentially(t *testing.T) {
	program, err := load.Load(context.Background(), load.Request{
		Directory: mapCrossProjectDirectory(),
		Pattern:   "./api",
	})
	if err != nil {
		t.Fatal(err)
	}
	loaded := program.Roots()[0]
	workingDirectory := t.TempDir()
	artifacts := materialize(t, compileExported(t, loaded), workingDirectory)
	if runtimeFiles := countPaths(artifacts.paths, "runtime/map.ts"); runtimeFiles != 1 {
		t.Fatalf("cross-package runtime map files = %d, want one", runtimeFiles)
	}

	modulePath, err := filepath.Abs(mapCrossProjectDirectory())
	if err != nil {
		t.Fatal(err)
	}
	goRunner := filepath.Join(workingDirectory, "cross-go")
	writeFile(t, filepath.Join(goRunner, "go.mod"), fmt.Sprintf(`module example.com/runner

go 1.26.4

require example.com/mapcross v0.0.0

replace example.com/mapcross => %s
`, filepath.ToSlash(modulePath)))
	writeFile(t, filepath.Join(goRunner, "main.go"), `package main
import (
    "fmt"
    "example.com/mapcross/api"
)
func main() { fmt.Println(api.Run(20)) }
`)
	goOutput := run(t, goRunner, filepath.Join(runtime.GOROOT(), "bin", "go"), "run", ".")

	runnerPath := filepath.Join(workingDirectory, "cross-runner.ts")
	writeFile(t, runnerPath, `import { Run } from "`+artifacts.module(t, "api.ts")+`";
console.log(...Run(20));
`)
	writeFile(t, filepath.Join(workingDirectory, "package.json"), "{\"type\":\"module\"}\n")
	strictTypecheckWithRunner(t, artifacts, workingDirectory, runnerPath)
	targetOutput := run(
		t,
		workingDirectory,
		"node",
		filepath.Join(workingDirectory, "out", "cross-runner.js"),
	)
	if targetOutput != goOutput {
		t.Fatalf("cross-package TypeScript/Go = %q/%q", targetOutput, goOutput)
	}
}

func mapCrossProjectDirectory() string {
	return filepath.Join(
		repositoryRoot(),
		"testdata",
		"constructs",
		"value",
		"map",
		"cross-package",
	)
}

func countPaths(paths []string, suffix string) int {
	count := 0
	for _, path := range paths {
		if strings.HasSuffix(filepath.ToSlash(path), suffix) {
			count++
		}
	}
	return count
}
