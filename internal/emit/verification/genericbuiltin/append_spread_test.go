package genericbuiltin_test

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/tsoniclang/gotots/internal/emit"
	"github.com/tsoniclang/gotots/internal/load"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
	runtimefixture "github.com/tsoniclang/gotots/internal/testfixture/gototsruntime"
)

func TestGenericAppendSpreadTypeFamiliesExecuteDifferentially(t *testing.T) {
	program, err := load.One(context.Background(), load.Request{
		Directory: fixtureDirectory(),
		Pattern:   ".",
	})
	if err != nil {
		t.Fatal(err)
	}
	var roots []emit.Root
	for _, name := range []string{"BytesResult", "StringResult"} {
		root, rootErr := emit.NewRoot(program.Types().Scope().Lookup(name))
		if rootErr != nil {
			t.Fatal(rootErr)
		}
		roots = append(roots, root)
	}
	emission, err := emit.Compile(program.Program(), roots)
	if err != nil {
		t.Fatal(err)
	}

	workingDirectory := t.TempDir()
	paths, sourceModule, printed := materialize(
		t,
		emission,
		workingDirectory,
	)
	for _, required := range []string{
		"append_spread",
		"appendBytes$kernel",
		"appendBytes$SliceOf_byte",
		"appendBytes$string",
	} {
		if !strings.Contains(printed, required) {
			t.Fatalf("generic append-spread artifact lacks %q:\n%s", required, printed)
		}
	}
	for _, forbidden := range []string{
		"export function appendBytes<",
		" as any",
		" as unknown",
		"typeof ",
		"instanceof ",
	} {
		if strings.Contains(printed, forbidden) {
			t.Fatalf("generic append-spread artifact contains %q:\n%s", forbidden, printed)
		}
	}

	runner := filepath.Join(workingDirectory, "runner.ts")
	writeFile(t, runner, fmt.Sprintf(`import {
  BytesResult,
  StringResult,
} from %q;

console.log(BytesResult());
console.log(StringResult());
`, sourceModule))
	writeFile(
		t,
		filepath.Join(workingDirectory, "package.json"),
		"{\"type\":\"module\"}\n",
	)
	typecheck(t, workingDirectory, append(paths, runner))
	targetOutput := run(
		t,
		workingDirectory,
		"node",
		filepath.Join(workingDirectory, "out", "runner.js"),
	)
	goOutput := runGo(t, workingDirectory)
	if targetOutput != goOutput {
		t.Fatalf(
			"generic append-spread differs\nTypeScript:\n%s\nGo:\n%s",
			targetOutput,
			goOutput,
		)
	}
}

func materialize(
	t *testing.T,
	emission emit.ProgramEmission,
	workingDirectory string,
) ([]string, string, string) {
	t.Helper()
	client, err := tsgo.StartClient(repositoryRoot(), workingDirectory)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = client.Close() })
	var paths []string
	var sourceModule string
	var printed strings.Builder
	for _, file := range emission.Files() {
		source, printErr := client.PrintNode(
			file.SourceFile(),
			tsgo.PrintOptions{},
		)
		if printErr != nil {
			t.Fatal(printErr)
		}
		printed.WriteString(source)
		path := filepath.Join(
			workingDirectory,
			filepath.FromSlash(file.OutputPath()),
		)
		writeFile(t, path, source)
		paths = append(paths, path)
		if file.Kind() == emit.TargetFileSource &&
			file.PackageName() == "genericbuiltins" {
			sourceModule = "./" +
				strings.TrimSuffix(file.OutputPath(), ".ts") + ".js"
		}
	}
	if sourceModule == "" {
		t.Fatal("generic-builtins source module is absent")
	}
	return paths, sourceModule, printed.String()
}

func typecheck(t *testing.T, directory string, paths []string) {
	t.Helper()
	arguments := []string{
		"--target", "es2022",
		"--module", "nodenext",
		"--moduleResolution", "nodenext",
		"--strict",
		"--outDir", filepath.Join(directory, "out"),
	}
	arguments = append(arguments, paths...)
	if err := runtimefixture.InstallResolution(directory, filepath.Join(directory, "out")); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := tsgo.Compile(
		ctx,
		repositoryRoot(),
		directory,
		arguments,
	); err != nil {
		t.Fatal(err)
	}
}

func runGo(t *testing.T, workingDirectory string) string {
	t.Helper()
	runnerDirectory := filepath.Join(workingDirectory, "go-runner")
	writeFile(t, filepath.Join(runnerDirectory, "go.mod"), fmt.Sprintf(`module example.com/runner

go 1.26.4

require example.com/genericbuiltins v0.0.0

replace example.com/genericbuiltins => %s
`, filepath.ToSlash(fixtureDirectory())))
	writeFile(t, filepath.Join(runnerDirectory, "main.go"), `package main

import (
	"fmt"

	values "example.com/genericbuiltins"
)

func main() {
	fmt.Println(values.BytesResult())
	fmt.Println(values.StringResult())
}
`)
	return run(t, runnerDirectory, filepath.Join(runtime.GOROOT(), "bin", "go"), "run", ".")
}

func run(t *testing.T, directory, name string, arguments ...string) string {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	command := exec.CommandContext(ctx, name, arguments...)
	command.Dir = directory
	command.Env = append(os.Environ(), "GOMEMLIMIT=1GiB", "GOMAXPROCS=2")
	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("%s %s: %v\n%s", name, strings.Join(arguments, " "), err, output)
	}
	return string(output)
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func fixtureDirectory() string {
	return filepath.Join(
		repositoryRoot(),
		"testdata",
		"constructs",
		"expression",
		"generic-builtins",
	)
}

func repositoryRoot() string {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		panic("resolve generic-builtins repository root")
	}
	return filepath.Clean(
		filepath.Join(filepath.Dir(file), "..", "..", "..", ".."),
	)
}
