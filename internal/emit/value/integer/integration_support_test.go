package integer_test

import (
	"context"
	"go/ast"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/tsoniclang/gotots/internal/emit"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
	runtimefixture "github.com/tsoniclang/gotots/internal/testfixture/gototsruntime"
)

type materializedProgram struct {
	targetPaths []string
	modules     map[string]string
}

func integerOptions(representation emit.IntegerRepresentation) emit.Options {
	return emit.Options{
		IntegerRepresentation: representation,
		EvaluationOrder:       emit.EvaluationOrderDirect,
	}
}

func (p materializedProgram) module(t *testing.T, base string) string {
	t.Helper()
	module := p.modules[base]
	if module == "" {
		t.Fatalf("emitted source module %q is absent", base)
	}
	return module
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
	if err := runtimefixture.InstallResolution(workingDirectory, outputDirectory); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := tsgo.Compile(
		ctx,
		repositoryRoot(),
		workingDirectory,
		arguments,
	); err != nil {
		t.Fatal(err)
	}
	return run(
		t,
		workingDirectory,
		"node",
		filepath.Join(outputDirectory, "runner.js"),
	)
}

func targetFunction(
	t *testing.T,
	file tsgo.SourceFile,
	name string,
) tsgo.FunctionDeclaration {
	t.Helper()
	for _, statement := range file.Statements() {
		function, ok := statement.(tsgo.FunctionDeclaration)
		if ok && function.Name().Text() == name {
			return function
		}
	}
	t.Fatalf("target function %s not found", name)
	return nil
}

func sourceFunction(
	t *testing.T,
	file *ast.File,
	name string,
) *ast.FuncDecl {
	t.Helper()
	for _, declaration := range file.Decls {
		function, ok := declaration.(*ast.FuncDecl)
		if ok && function.Name.Name == name {
			return function
		}
	}
	t.Fatalf("source function %s not found", name)
	return nil
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

func repositoryRoot() string {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		panic("resolve integer repository root")
	}
	return filepath.Clean(
		filepath.Join(filepath.Dir(file), "..", "..", "..", ".."),
	)
}

func integerBoundaryDirectory() string {
	return filepath.Join(repositoryRoot(), "testdata", "constructs", "value", "integer-boundaries")
}

func assertIntegerAliases(
	t *testing.T,
	emission emit.ProgramEmission,
	exact bool,
) {
	t.Helper()
	want := []string{
		"int8", "int16", "int32", "int64",
		"uint8", "uint16", "uint32", "uint64",
		"int", "uint", "uintptr",
	}
	var got []string
	shared := map[string]string{
		"int8":   "int8",
		"int16":  "int16",
		"int32":  "int32",
		"uint8":  "uint8",
		"uint16": "uint16",
		"uint32": "uint32",
	}
	if exact {
		shared["int64"] = "int64"
		shared["uint64"] = "uint64"
		if strconv.IntSize == 64 {
			shared["int"] = "int64"
			shared["uint"] = "uint64"
			shared["uintptr"] = "uint64"
		} else {
			shared["int"] = "int32"
			shared["uint"] = "uint32"
			shared["uintptr"] = "uint32"
		}
	}
	for _, file := range emission.Files() {
		if file.Kind() != emit.TargetFileSupport {
			continue
		}
		for _, statement := range file.SourceFile().Statements() {
			alias, ok := statement.(tsgo.TypeAliasDeclaration)
			if !ok || alias.Name().Text() == "bool" {
				continue
			}
			if sharedName, selected := shared[alias.Name().Text()]; selected {
				reference, ok := alias.Type().(tsgo.TypeReferenceNode)
				if !ok {
					t.Fatalf("%s carrier = %T, want target-neutral type reference", alias.Name().Text(), alias.Type())
				}
				identifier, ok := reference.TypeName().(tsgo.Identifier)
				if !ok {
					t.Fatalf("%s carrier name = %T, want identifier $go$core$%s", alias.Name().Text(), reference.TypeName(), sharedName)
				}
				if identifier.Text() != "$go$core$"+sharedName {
					t.Fatalf("%s carrier = %q, want $go$core$%s", alias.Name().Text(), identifier.Text(), sharedName)
				}
				got = append(got, alias.Name().Text())
				continue
			}
			carrier := tsgo.SyntaxKindNumberKeyword
			wideNative := strconv.IntSize == 64 &&
				(alias.Name().Text() == "int" ||
					alias.Name().Text() == "uint" ||
					alias.Name().Text() == "uintptr")
			if exact && (alias.Name().Text() == "int64" ||
				alias.Name().Text() == "uint64" || wideNative) {
				carrier = tsgo.SyntaxKindBigIntKeyword
			}
			if alias.Type().Kind() != carrier {
				t.Fatalf("%s carrier = %d, want %d", alias.Name().Text(), alias.Type().Kind(), carrier)
			}
			got = append(got, alias.Name().Text())
		}
	}
	slices.Sort(want)
	slices.Sort(got)
	if !slices.Equal(got, want) {
		t.Fatalf("integer aliases = %v, want %v", got, want)
	}
}
