package function_test

import (
	"context"
	"fmt"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/tsoniclang/gotots/internal/load"
)

func TestDefinedCallableCrossPackageExecutesDifferentially(t *testing.T) {
	program, err := load.Load(context.Background(), load.Request{
		Directory: definedCallableCrossPackageDirectory(),
		Pattern:   "./consumer",
	})
	if err != nil {
		t.Fatal(err)
	}
	workingDirectory := t.TempDir()
	artifacts := materializeExportedProgram(
		t,
		program.Roots()[0],
		workingDirectory,
	)
	printed := readMaterializedProgram(t, artifacts)
	for _, required := range []string{
		"new Transform(",
		"Transform | undefined",
		".$value(",
	} {
		if !strings.Contains(printed, required) {
			t.Fatalf("cross-package callable artifact lacks %q:\n%s", required, printed)
		}
	}
	runnerPath := filepath.Join(workingDirectory, "runner.ts")
	writeFile(t, runnerPath, `import {
    FromRaw,
    IsNil,
    Use,
} from "`+artifacts.module(t, "consumer.ts")+`";

console.log(String(Use(10)));
console.log(String(FromRaw(20)));
console.log(String(IsNil(undefined)));
`)
	targetOutput := executeMaterializedTypeScript(
		t,
		workingDirectory,
		artifacts,
		runnerPath,
	)
	goOutput := executeDefinedCallableCrossPackageGo(t, workingDirectory)
	if targetOutput != goOutput {
		t.Fatalf(
			"cross-package TypeScript output differs from Go\nTypeScript:\n%s\nGo:\n%s",
			targetOutput,
			goOutput,
		)
	}
}

func executeDefinedCallableCrossPackageGo(
	t *testing.T,
	workingDirectory string,
) string {
	t.Helper()
	modulePath, err := filepath.Abs(definedCallableCrossPackageDirectory())
	if err != nil {
		t.Fatal(err)
	}
	runnerDirectory := filepath.Join(workingDirectory, "go-cross-runner")
	writeFile(t, filepath.Join(runnerDirectory, "go.mod"), fmt.Sprintf(`module example.com/runner

go 1.26.4

require example.com/definedcallable v0.0.0

replace example.com/definedcallable => %s
`, filepath.ToSlash(modulePath)))
	writeFile(t, filepath.Join(runnerDirectory, "main.go"), `package main

import (
	"fmt"

	"example.com/definedcallable/consumer"
)

func main() {
	fmt.Println(consumer.Use(10))
	fmt.Println(consumer.FromRaw(20))
	fmt.Println(consumer.IsNil(nil))
}
`)
	return run(
		t,
		runnerDirectory,
		filepath.Join(runtime.GOROOT(), "bin", "go"),
		"run",
		".",
	)
}

func definedCallableCrossPackageDirectory() string {
	return filepath.Join(
		repositoryRoot(),
		"testdata",
		"constructs",
		"expression",
		"function-value",
		"defined-cross-package",
	)
}
