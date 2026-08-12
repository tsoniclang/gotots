package structvalue_test

import (
	"context"
	"errors"
	"fmt"
	"os"
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

func TestNamedStructSyntheticBindingsAndReceiverCopyHaveExactShape(
	t *testing.T,
) {
	source := structTargetSource(t, compileLexicalReceiverFixture(t))
	shift := targetMethod(t, targetClass(t, source, "Mode"), "Shift")
	statements := shift.Body().(tsgo.Block).Statements()
	copyStatement, ok := statements[0].(tsgo.VariableStatement)
	if !ok {
		t.Fatalf("receiver copy statement = %T, want variable statement", statements[0])
	}
	copyDeclaration := copyStatement.DeclarationList().Declarations()[0]
	copyType, ok := copyDeclaration.Type().(tsgo.TypeReferenceNode)
	if !ok ||
		copyType.TypeName().(tsgo.Identifier).Text() != "Mode" ||
		targetName(copyDeclaration.Name()) != "mode" ||
		copyDeclaration.Initializer().Kind() != tsgo.SyntaxKindThisKeyword {
		t.Fatalf(
			"receiver copy = name %q, type %T, initializer %T",
			targetName(copyDeclaration.Name()),
			copyDeclaration.Type(),
			copyDeclaration.Initializer(),
		)
	}

	for className, expectedFields := range map[string][]string{
		"fileRange":    {"label", "fileRange"},
		"derivedRange": {"label", "derivedRange"},
	} {
		class := targetClass(t, source, className)
		assertStaticOperationSequence(t, source, className, nil)
		constructor := classConstructor(t, class)
		if len(constructor.Parameters()) != 1 {
			t.Fatalf("%s constructor parameters = %d, want one named object", className, len(constructor.Parameters()))
		}
		input, ok := constructor.Parameters()[0].Type().(tsgo.TypeLiteralNode)
		if !ok || fmt.Sprint(typeLiteralMemberNames(input)) != fmt.Sprint(expectedFields) {
			t.Fatalf("%s constructor fields = %v, want %v", className, typeLiteralMemberNames(input), expectedFields)
		}
	}
}

func TestLexicalReceiverFixturePrintsTypechecksAndMatchesGo(t *testing.T) {
	emission := compileLexicalReceiverFixture(t)
	workingDirectory := t.TempDir()
	targetPaths, module := materializeStructProgramWithGolden(
		t,
		workingDirectory,
		emission,
		false,
	)
	runner := filepath.Join(workingDirectory, "runner.ts")
	writeProgramFile(t, runner, `import { Audit } from "`+module+`";

console.log(Audit());
`)
	writeProgramFile(
		t,
		filepath.Join(workingDirectory, "package.json"),
		"{\"type\":\"module\"}\n",
	)
	compileStructTypeScript(t, workingDirectory, append(targetPaths, runner))
	targetOutput := runProgram(
		t,
		workingDirectory,
		"node",
		filepath.Join(workingDirectory, "out", "runner.js"),
	)
	goOutput := executeLexicalReceiverGo(t, workingDirectory)
	if targetOutput != goOutput {
		t.Fatalf("lexical fixture TypeScript/Go output = %q/%q", targetOutput, goOutput)
	}
}

func TestLexicalReceiverMutationsFailStrictTypeScript(t *testing.T) {
	emission := compileLexicalReceiverFixture(t)
	t.Run("receiver-copy-type", func(t *testing.T) {
		workingDirectory, targetPaths, sourcePath := materializeLexicalMutation(
			t,
			emission,
		)
		mutateTargetSource(
			t,
			sourcePath,
			"let mode: Mode = this;",
			"let mode = this;",
		)
		assertStrictDiagnostic(t, workingDirectory, targetPaths, "TS2322")
	})
	for _, testCase := range []struct {
		className string
		useBefore string
		useAfter  string
	}{
		{
			className: "fileRange",
			useBefore: "public fileRange: Mode;",
			useAfter:  "public fileRange: fileRange;",
		},
		{
			className: "derivedRange",
			useBefore: "public derivedRange: Mode;",
			useAfter:  "public derivedRange: derivedRange;",
		},
	} {
		t.Run(testCase.className+"-named-field-binding", func(t *testing.T) {
			workingDirectory, targetPaths, sourcePath :=
				materializeLexicalMutation(t, emission)
			mutateTargetSource(
				t,
				sourcePath,
				testCase.useBefore,
				testCase.useAfter,
			)
			assertStrictDiagnostic(t, workingDirectory, targetPaths, "TS2739")
		})
	}
}

func compileLexicalReceiverFixture(t *testing.T) emit.ProgramEmission {
	t.Helper()
	program, err := load.Load(context.Background(), load.Request{
		Directory: lexicalReceiverDirectory(),
		Pattern:   ".",
	})
	if err != nil {
		t.Fatal(err)
	}
	roots, err := emit.ExportedAPIRoots(program.Roots()[0])
	if err != nil {
		t.Fatal(err)
	}
	emission, err := emit.Compile(program, roots)
	if err != nil {
		t.Fatal(err)
	}
	return emission
}

func materializeLexicalMutation(
	t *testing.T,
	emission emit.ProgramEmission,
) (string, []string, string) {
	t.Helper()
	workingDirectory := t.TempDir()
	targetPaths, module := materializeStructProgramWithGolden(
		t,
		workingDirectory,
		emission,
		false,
	)
	writeProgramFile(
		t,
		filepath.Join(workingDirectory, "package.json"),
		"{\"type\":\"module\"}\n",
	)
	if err := strictTypecheck(workingDirectory, targetPaths); err != nil {
		t.Fatalf("unmutated lexical fixture failed strict typechecking: %v", err)
	}
	sourcePath := filepath.Join(
		workingDirectory,
		filepath.FromSlash(
			strings.TrimSuffix(strings.TrimPrefix(module, "./"), ".js")+".ts",
		),
	)
	return workingDirectory, targetPaths, sourcePath
}

func mutateTargetSource(t *testing.T, path, before, after string) {
	t.Helper()
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	mutated := strings.Replace(string(content), before, after, 1)
	if mutated == string(content) {
		t.Fatalf("mutation target %q is absent", before)
	}
	writeProgramFile(t, path, mutated)
}

func assertStrictDiagnostic(
	t *testing.T,
	workingDirectory string,
	targetPaths []string,
	code string,
) {
	t.Helper()
	err := strictTypecheck(workingDirectory, targetPaths)
	var compilerError *tsgo.CompilerError
	if !errors.As(err, &compilerError) ||
		!strings.Contains(compilerError.Diagnostics, code) {
		t.Fatalf("strict mutation error = %#v, want %s", err, code)
	}
}

func strictTypecheck(
	workingDirectory string,
	targetPaths []string,
) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	arguments := []string{
		"--target", "es2022",
		"--module", "nodenext",
		"--moduleResolution", "nodenext",
		"--strict",
		"--noEmit",
	}
	arguments = append(arguments, targetPaths...)
	if err := runtimefixture.InstallResolution(workingDirectory, filepath.Join(workingDirectory, "out")); err != nil {
		return err
	}
	return tsgo.Compile(ctx, repositoryRoot(), workingDirectory, arguments)
}

func executeLexicalReceiverGo(t *testing.T, workingDirectory string) string {
	t.Helper()
	modulePath, err := filepath.Abs(lexicalReceiverDirectory())
	if err != nil {
		t.Fatal(err)
	}
	runnerDirectory := filepath.Join(workingDirectory, "go-runner")
	writeProgramFile(t, filepath.Join(runnerDirectory, "go.mod"), fmt.Sprintf(
		`module example.com/runner

go 1.26.4

require example.com/lexicalreceiver v0.0.0

replace example.com/lexicalreceiver => %s
`,
		filepath.ToSlash(modulePath),
	))
	writeProgramFile(t, filepath.Join(runnerDirectory, "main.go"), `package main

import (
	"fmt"

	values "example.com/lexicalreceiver"
)

func main() {
	fmt.Println(values.Audit())
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

func lexicalReceiverDirectory() string {
	return filepath.Join(
		repositoryRoot(),
		"testdata",
		"constructs",
		"value",
		"lexical-receiver",
	)
}
