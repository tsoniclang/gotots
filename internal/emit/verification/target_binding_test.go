package emit_test

import (
	"context"
	"encoding/binary"
	"fmt"
	"go/ast"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/tsoniclang/gotots/internal/emit"
	"github.com/tsoniclang/gotots/internal/load"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func cooperativeFunctionName(function string) string {
	start := strings.Index(function, "function ")
	if start < 0 {
		return ""
	}
	start += len("function ")
	end := strings.IndexByte(function[start:], '<')
	if end < 0 {
		end = strings.IndexByte(function[start:], '(')
	}
	if end < 0 {
		return ""
	}
	return function[start : start+end]
}

func packageAssemblyExports(
	files []emit.TargetFile,
	packageName string,
	name string,
) bool {
	for _, file := range files {
		if file.Kind() != emit.TargetFilePackageAssembly ||
			file.PackageName() != packageName {
			continue
		}
		for _, statement := range file.SourceFile().Statements() {
			declaration, ok := statement.(tsgo.ExportDeclaration)
			if !ok {
				continue
			}
			exports, ok := declaration.ExportClause().(tsgo.NamedExports)
			if !ok {
				continue
			}
			for _, specifier := range exports.Elements() {
				identifier, ok := specifier.Name().(tsgo.Identifier)
				if ok && identifier.Text() == name {
					return true
				}
			}
		}
	}
	return false
}

func TestTargetBindingsDeferConstructedConstantsAndProtectIntrinsics(
	t *testing.T,
) {
	directory, err := filepath.Abs(targetBindingDirectory())
	if err != nil {
		t.Fatal(err)
	}
	program, err := load.Load(context.Background(), load.Request{
		Directory: directory,
		Pattern:   ".",
	})
	if err != nil {
		t.Fatal(err)
	}
	roots, err := emit.ExportedAPIRoots(program.Roots()[0])
	if err != nil {
		t.Fatal(err)
	}
	options := emit.DefaultOptions()
	options.IntegerRepresentation = emit.IntegerRepresentationBigInt
	emission, err := emit.CompileWithOptions(program, roots, options)
	if err != nil {
		t.Fatal(err)
	}
	assertTargetBindingAST(t, emission)

	workingDirectory := t.TempDir()
	artifacts := materializeArtifacts(t, emission, workingDirectory)
	if !strings.Contains(artifacts.printed, "globalThis.Number(") {
		t.Fatalf(
			"target intrinsic is not globally anchored:\n%s",
			artifacts.printed,
		)
	}
	if strings.Contains(artifacts.printed, "function globalThis(") {
		t.Fatalf(
			"Go binding captured the target global anchor:\n%s",
			artifacts.printed,
		)
	}
	runner := filepath.Join(workingDirectory, "runner.ts")
	writeProgramFile(t, runner, `import "./program.js";
import { Audit } from "`+artifacts.sourceModule+`";

console.log(String(Audit(5n)));
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
	goRunner := filepath.Join(workingDirectory, "go-runner")
	writeProgramFile(t, filepath.Join(goRunner, "go.mod"), fmt.Sprintf(
		`module example.com/runner

go 1.26.4

require example.com/targetbinding v0.0.0

replace example.com/targetbinding => %s
`,
		filepath.ToSlash(directory),
	))
	writeProgramFile(t, filepath.Join(goRunner, "main.go"), `package main

import (
	"fmt"

	values "example.com/targetbinding"
)

func main() {
	fmt.Println(values.Audit(5))
}
`)
	goOutput := runProgram(
		t,
		goRunner,
		filepath.Join(runtime.GOROOT(), "bin", "go"),
		"run",
		".",
	)
	if targetOutput != goOutput {
		t.Fatalf(
			"target-binding differential differs\nTypeScript: %q\nGo: %q",
			targetOutput,
			goOutput,
		)
	}
}

func assertTargetBindingAST(t *testing.T, emission emit.ProgramEmission) {
	t.Helper()
	for _, file := range emission.Files() {
		if file.Kind() != emit.TargetFileSource {
			continue
		}
		classIndex := -1
		constantIndex := -1
		for index, statement := range file.SourceFile().Statements() {
			switch statement := statement.(type) {
			case tsgo.ClassDeclaration:
				if statement.Name().Text() == "Number" {
					classIndex = index
				}
			case tsgo.FunctionDeclaration:
				if statement.Name().Text() == "Before$constant" {
					constantIndex = index
					body := statement.Body().(tsgo.Block).Statements()
					if len(body) != 1 {
						t.Fatalf("constant thunk body statements = %d, want one", len(body))
					}
					returned := body[0].(tsgo.ReturnStatement).Expression()
					constructed, ok := returned.(tsgo.NewExpression)
					if !ok || constructed.Expression().(tsgo.Identifier).Text() != "Number" {
						t.Fatalf("constant thunk returns %T, want new Number", returned)
					}
				}
			}
		}
		if classIndex < 0 || constantIndex < 0 {
			t.Fatalf(
				"target declarations absent: class=%d constant=%d",
				classIndex,
				constantIndex,
			)
		}
		return
	}
	t.Fatal("target-binding fixture emitted no source file")
}

func targetBindingDirectory() string {
	return filepath.Join(
		repositoryRoot(),
		"testdata",
		"constructs",
		"naming",
		"target-binding",
	)
}

func waveFourEncodedNodes(t *testing.T, encoded []byte) int {
	t.Helper()
	const (
		headerSize       = 44
		nodesOffsetField = 40
		nodeWidth        = 28
	)
	if len(encoded) < headerSize {
		t.Fatalf("encoded target is %d bytes, want protocol header", len(encoded))
	}
	nodesOffset := int(binary.LittleEndian.Uint32(
		encoded[nodesOffsetField:headerSize],
	))
	if nodesOffset < headerSize ||
		nodesOffset > len(encoded) ||
		(len(encoded)-nodesOffset)%nodeWidth != 0 {
		t.Fatalf("encoded target has invalid node offset %d", nodesOffset)
	}
	return (len(encoded) - nodesOffset) / nodeWidth
}

func assertWaveFourLinearDoubling(
	t *testing.T,
	name string,
	values []int,
) {
	t.Helper()
	first := values[1] - values[0]
	second := values[2] - values[1]
	if first <= 0 || second*10 < first*17 || second*10 > first*23 {
		t.Fatalf(
			"%s = %v; doubling deltas %d/%d are not linear",
			name,
			values,
			first,
			second,
		)
	}
}

func waveFourFunction(
	t *testing.T,
	sourcePackage *load.Package,
	name string,
) *ast.FuncDecl {
	t.Helper()
	for _, file := range sourcePackage.Files() {
		for _, declaration := range file.Syntax().Decls {
			function, ok := declaration.(*ast.FuncDecl)
			if ok && function.Name.Name == name {
				return function
			}
		}
	}
	t.Fatalf("Go function %s is absent", name)
	return nil
}

func waveFourTargetFunction(
	t *testing.T,
	emission emit.ProgramEmission,
	name string,
) tsgo.FunctionDeclaration {
	t.Helper()
	for _, file := range emission.Files() {
		if file.Kind() != emit.TargetFileSource {
			continue
		}
		for _, statement := range file.SourceFile().Statements() {
			function, ok := statement.(tsgo.FunctionDeclaration)
			if ok && function.Name().Text() == name {
				return function
			}
		}
	}
	t.Fatalf("target function %s is absent", name)
	return nil
}

func TestNonGenericAliasToGenericInstantiationCompilesInPointerField(
	t *testing.T,
) {
	project := t.TempDir()
	writeProgramFile(
		t,
		filepath.Join(project, "go.mod"),
		"module example.com/genericalias\n\ngo 1.26.4\n",
	)
	writeProgramFile(
		t,
		filepath.Join(project, "source.go"),
		`package genericalias

type Box[T any] struct {
	Value T
}

type IntBox = Box[int]

type Holder struct {
	Box *IntBox
}

func EmptyHolder() Holder {
	return Holder{}
}
`,
	)
	program, err := load.Load(context.Background(), load.Request{
		Directory: project,
		Pattern:   ".",
	})
	if err != nil {
		t.Fatal(err)
	}
	root, err := emit.NewRoot(
		program.Roots()[0].Types().Scope().Lookup("EmptyHolder"),
	)
	if err != nil {
		t.Fatal(err)
	}
	emission, err := emit.Compile(program, []emit.Root{root})
	if err != nil {
		t.Fatal(err)
	}
	workingDirectory := t.TempDir()
	artifacts := materializeArtifacts(t, emission, workingDirectory)
	if !strings.Contains(
		artifacts.printed,
		"public Box: Pointer<Box<int>> | undefined",
	) {
		t.Fatalf(
			"alias to generic instantiation was not canonicalized:\n%s",
			artifacts.printed,
		)
	}
	for _, forbidden := range []string{
		"Box<int64>",
		"GoPointer",
		"runtime/pointer",
		"Box$Storage<int>",
	} {
		if strings.Contains(artifacts.printed, forbidden) {
			t.Fatalf(
				"direct alias pointer contains %q:\n%s",
				forbidden,
				artifacts.printed,
			)
		}
	}
	waveThreeTypecheck(t, workingDirectory, artifacts.paths)
}
