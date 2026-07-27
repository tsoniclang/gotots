package constant_test

import (
	"context"
	"fmt"
	"go/ast"
	"go/token"
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
)

// numberRoots are exercised under the number profile; hugeRoot needs the bigint
// profile because its value exceeds the safe-integer range.
var numberRoots = []string{
	"Enum", "Inherited", "MultipleTargets", "Argument", "Assignment",
	"Case", "Conversion", "Arithmetic", "Float32Expression",
	"Float64Expression", "Defaulted", "Untyped", "Typed", "RuneValue", "Local",
}

const hugeRoot = "HugeAsUint"

func repositoryRoot() string {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		panic("resolve constant repository root")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", "..", ".."))
}

func constantFamilyDirectory() string {
	return filepath.Join(repositoryRoot(), "testdata", "constructs", "value", "constant-family")
}

func loadConstantFamily(t *testing.T) *load.Package {
	t.Helper()
	loaded, err := load.One(context.Background(), load.Request{
		Directory: constantFamilyDirectory(),
		Pattern:   ".",
	})
	if err != nil {
		t.Fatal(err)
	}
	return loaded
}

func compileConstantFamily(
	t *testing.T,
	loaded *load.Package,
	options emit.Options,
	names ...string,
) emit.ProgramEmission {
	t.Helper()
	roots := make([]emit.Root, 0, len(names))
	for _, name := range names {
		object := loaded.Types().Scope().Lookup(name)
		if object == nil {
			t.Fatalf("constant root %q is absent", name)
		}
		root, err := emit.NewRoot(object)
		if err != nil {
			t.Fatal(err)
		}
		roots = append(roots, root)
	}
	emission, err := emit.CompileWithOptions(loaded.Program(), roots, options)
	if err != nil {
		t.Fatal(err)
	}
	return emission
}

func printConstantFamily(t *testing.T, emission emit.ProgramEmission) string {
	t.Helper()
	client, err := tsgo.StartClient(repositoryRoot(), t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := client.Close(); err != nil {
			t.Errorf("close TS-Go client: %v", err)
		}
	})
	var result strings.Builder
	for _, file := range emission.Files() {
		printed, err := client.PrintNode(file.SourceFile(), tsgo.PrintOptions{})
		if err != nil {
			t.Fatal(err)
		}
		result.WriteString(printed)
	}
	return result.String()
}

func constantFamilySourceFile(t *testing.T, emission emit.ProgramEmission) tsgo.SourceFile {
	t.Helper()
	for _, file := range emission.Files() {
		if file.Kind() == emit.TargetFileSource && file.PackageName() == "constantfamily" {
			return file.SourceFile()
		}
	}
	t.Fatal("constant family source artifact is absent")
	return nil
}

func assertNoForbiddenConstructs(t *testing.T, printed string) {
	t.Helper()
	for _, forbidden := range []string{" as ", "any", "unknown", ".call(", ".apply(", ".bind(", "iota"} {
		if strings.Contains(printed, forbidden) {
			t.Fatalf("constant artifact contains %q:\n%s", forbidden, printed)
		}
	}
}

func packageConstSpec(t *testing.T, loaded *load.Package, name string) *ast.ValueSpec {
	t.Helper()
	for _, file := range loaded.Files() {
		for _, decl := range file.Syntax().Decls {
			gen, ok := decl.(*ast.GenDecl)
			if !ok || gen.Tok != token.CONST {
				continue
			}
			for _, spec := range gen.Specs {
				value := spec.(*ast.ValueSpec)
				for _, id := range value.Names {
					if id.Name == name {
						return value
					}
				}
			}
		}
	}
	t.Fatalf("package constant %q not found", name)
	return nil
}

func executeConstantFamilyGo(t *testing.T, workingDirectory string, includeHuge bool) string {
	t.Helper()
	modulePath, err := filepath.Abs(constantFamilyDirectory())
	if err != nil {
		t.Fatal(err)
	}
	runnerDirectory := filepath.Join(workingDirectory, "go-runner")
	if err := os.MkdirAll(runnerDirectory, 0o755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(runnerDirectory, "go.mod"), fmt.Sprintf(`module example.com/runner

go 1.26.4

require example.com/constantfamily v0.0.0

replace example.com/constantfamily => %s
`, filepath.ToSlash(modulePath)))
	huge := ""
	if includeHuge {
		huge = "\tfmt.Println(values.HugeAsUint())\n"
	}
	writeFile(t, filepath.Join(runnerDirectory, "main.go"), `package main

import (
	"fmt"
	"strconv"

	values "example.com/constantfamily"
)

func main() {
	fmt.Println(values.Enum())
	fmt.Println(values.Inherited())
	fmt.Println(values.MultipleTargets())
	fmt.Println(values.Argument())
	fmt.Println(values.Assignment())
	fmt.Println(values.Case(100))
	fmt.Println(values.Case(5))
	fmt.Println(values.Conversion())
	fmt.Println(values.Arithmetic())
	fmt.Println(strconv.FormatFloat(float64(values.Float32Expression()), 'g', -1, 64))
	fmt.Println(values.Float64Expression())
	fmt.Println(values.Defaulted())
	fmt.Println(values.Untyped())
	fmt.Println(values.Typed())
	fmt.Println(values.RuneValue())
	fmt.Println(values.Local())
`+huge+`}
`)
	return run(t, runnerDirectory, "go", "run", ".")
}

func executeConstantFamilyTS(
	t *testing.T,
	emission emit.ProgramEmission,
	workingDirectory string,
	bigint bool,
) string {
	t.Helper()
	artifacts := materializeConstantFamily(t, emission, workingDirectory)
	// Under the bigint profile every integer carrier is bigint, so Case's int32
	// argument is passed as a bigint literal; the large constant root is also
	// only exercised there.
	suffix := ""
	huge := ""
	if bigint {
		suffix = "n"
		huge = "console.log(String(values.HugeAsUint()));\n"
	}
	runner := `import * as values from "` + artifacts.module(t, "source.ts") + `";

const row = (value: readonly unknown[]): string =>
	value.map((entry) => String(entry)).join(" ");

console.log(row(values.Enum()));
console.log(row(values.Inherited()));
console.log(row(values.MultipleTargets()));
console.log(String(values.Argument()));
console.log(String(values.Assignment()));
console.log(String(values.Case(100` + suffix + `)));
console.log(String(values.Case(5` + suffix + `)));
console.log(String(values.Conversion()));
console.log(String(values.Arithmetic()));
console.log(String(values.Float32Expression()));
console.log(String(values.Float64Expression()));
console.log(row(values.Defaulted()));
console.log(row(values.Untyped()));
console.log(row(values.Typed()));
console.log(String(values.RuneValue()));
console.log(row(values.Local()));
` + huge
	runnerPath := filepath.Join(workingDirectory, "runner.ts")
	writeFile(t, runnerPath, runner)
	return executeMaterializedTypeScript(t, workingDirectory, artifacts, runnerPath)
}

type materializedProgram struct {
	targetPaths []string
	modules     map[string]string
}

func (p materializedProgram) module(t *testing.T, base string) string {
	t.Helper()
	module := p.modules[base]
	if module == "" {
		t.Fatalf("emitted source module %q is absent", base)
	}
	return module
}

func materializeConstantFamily(
	t *testing.T,
	emission emit.ProgramEmission,
	workingDirectory string,
) materializedProgram {
	t.Helper()
	client, err := tsgo.StartClient(repositoryRoot(), workingDirectory)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := client.Close(); err != nil {
			t.Errorf("close TS-Go client: %v", err)
		}
	})
	result := materializedProgram{modules: make(map[string]string)}
	for _, file := range emission.Files() {
		printed, err := client.PrintNode(file.SourceFile(), tsgo.PrintOptions{})
		if err != nil {
			t.Fatal(err)
		}
		targetPath := filepath.Join(workingDirectory, filepath.FromSlash(file.OutputPath()))
		writeFile(t, targetPath, printed)
		result.targetPaths = append(result.targetPaths, targetPath)
		if file.Kind() == emit.TargetFileSource {
			result.modules[filepath.Base(file.OutputPath())] = "./" +
				strings.TrimSuffix(file.OutputPath(), ".ts") + ".js"
		}
	}
	return result
}

func executeMaterializedTypeScript(
	t *testing.T,
	workingDirectory string,
	artifacts materializedProgram,
	runnerPath string,
) string {
	t.Helper()
	writeFile(t, filepath.Join(workingDirectory, "package.json"), "{\"type\":\"module\"}\n")
	outputDirectory := filepath.Join(workingDirectory, "out")
	arguments := []string{
		"--target", "es2022",
		"--module", "nodenext",
		"--moduleResolution", "nodenext",
		"--strict",
		"--outDir", outputDirectory,
	}
	arguments = append(arguments, artifacts.targetPaths...)
	arguments = append(arguments, runnerPath)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := tsgo.Compile(ctx, repositoryRoot(), workingDirectory, arguments); err != nil {
		t.Fatal(err)
	}
	return run(t, workingDirectory, "node", filepath.Join(outputDirectory, "runner.js"))
}

func run(t *testing.T, directory, name string, arguments ...string) string {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	command := exec.CommandContext(ctx, name, arguments...)
	command.Dir = directory
	command.Env = append(os.Environ(), "GOMEMLIMIT=1GiB")
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
