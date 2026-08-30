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
	corefixture "github.com/tsoniclang/gotots/internal/testfixture/tsoniccore"
)

func TestNamedStructSyntheticBindingsAndNativeReceiverHaveExactShape(
	t *testing.T,
) {
	source := structTargetSource(t, compileLexicalReceiverFixture(t))
	shift := targetFunction(t, source, "Mode_Shift")
	parameters := shift.Parameters()
	if len(parameters) != 2 {
		t.Fatalf("native receiver parameters = %d, want 2", len(parameters))
	}
	receiverType, ok := parameters[0].Type().(tsgo.TypeReferenceNode)
	if !ok ||
		receiverType.TypeName().(tsgo.Identifier).Text() != "Mode" ||
		targetName(parameters[0].Name()) != "mode" {
		t.Fatalf(
			"native receiver = name %q, type %T",
			targetName(parameters[0].Name()),
			parameters[0].Type(),
		)
	}
	for _, statement := range shift.Body().(tsgo.Block).Statements() {
		if declaration, isVariable := statement.(tsgo.VariableStatement); isVariable &&
			len(declaration.DeclarationList().Declarations()) != 0 &&
			targetName(declaration.DeclarationList().Declarations()[0].Name()) == "mode" {
			t.Fatal("native numeric receiver acquired a redundant copy binding")
		}
	}

	for className, expectedFields := range map[string][]string{
		"fileRange":    {"label", "fileRange"},
		"derivedRange": {"label", "derivedRange"},
	} {
		class := targetClass(t, source, className)
		assertStaticOperationSequence(t, source, className, nil)
		constructor := classConstructor(t, class)
		if len(constructor.Parameters()) != len(expectedFields) {
			t.Fatalf("%s constructor parameters = %d, want %d", className, len(constructor.Parameters()), len(expectedFields))
		}
		actualFields := make([]string, 0, len(constructor.Parameters()))
		for _, parameter := range constructor.Parameters() {
			actualFields = append(actualFields, targetName(parameter.Name()))
		}
		if fmt.Sprint(actualFields) != fmt.Sprint(expectedFields) {
			t.Fatalf("%s constructor fields = %v, want %v", className, actualFields, expectedFields)
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
	t.Run("receiver-parameter-type", func(t *testing.T) {
		workingDirectory, targetPaths, sourcePath := materializeLexicalMutation(
			t,
			emission,
		)
		mutateTargetSource(
			t,
			sourcePath,
			"Mode_Shift(mode: Mode,",
			"Mode_Shift(mode: string,",
		)
		assertStrictDiagnostic(t, workingDirectory, targetPaths, "TS2322")
	})
	for _, testCase := range []struct {
		className string
		useBefore string
		useAfter  string
		code      string
	}{
		{
			className: "fileRange",
			useBefore: "public fileRange: Mode",
			useAfter:  "public fileRange: fileRange",
			code:      "TS2345",
		},
		{
			className: "derivedRange",
			useBefore: "public derivedRange: Mode",
			useAfter:  "public derivedRange: derivedRange",
			code:      "TS2345",
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
			assertStrictDiagnostic(t, workingDirectory, targetPaths, testCase.code)
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
	if err := corefixture.InstallResolutionOnly(workingDirectory); err != nil {
		return err
	}
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
