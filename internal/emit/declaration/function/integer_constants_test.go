package function_test

import (
	"context"
	"errors"
	"fmt"
	"go/ast"
	"go/token"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/tsoniclang/gotots/internal/emit"
	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/load"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func TestSafeIntegerConstantPrintsTypechecksAndExecutesDifferentially(t *testing.T) {
	loaded := loadIntegerConstantsProject(t)
	emission := compileIntegerRoot(t, loaded, 0)
	workingDirectory := t.TempDir()
	client, err := tsgo.StartClient(repositoryRoot(), workingDirectory)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := client.Close(); err != nil {
			t.Errorf("close TS-Go client: %v", err)
		}
	})

	var sourceModule string
	var targetPaths []string
	for _, file := range emission.Files() {
		printed, err := client.PrintNode(file.SourceFile(), tsgo.PrintOptions{})
		if err != nil {
			t.Fatal(err)
		}
		var expectedPath string
		switch file.Kind() {
		case emit.TargetFileSource:
			expectedPath = filepath.Join(
				integerConstantsProjectDirectory(),
				"expected.ts",
			)
		case emit.TargetFilePackageAssembly:
			sourceModule = "./" +
				strings.TrimSuffix(file.OutputPath(), ".ts") +
				".js"
		case emit.TargetFileSupport:
			expectedPath = filepath.Join(
				repositoryRoot(),
				"testdata",
				"support",
				"scalars-int64.ts",
			)
		case emit.TargetFileProgramInitialization:
		default:
			t.Fatalf(
				"unexpected integer-constant target %s kind %d",
				file.OutputPath(),
				file.Kind(),
			)
		}
		if expectedPath != "" {
			expected, err := os.ReadFile(expectedPath)
			if err != nil {
				t.Fatal(err)
			}
			if printed != string(expected) {
				t.Fatalf("%s:\n%s\nwant:\n%s", file.OutputPath(), printed, expected)
			}
		}
		targetPath := filepath.Join(workingDirectory, filepath.FromSlash(file.OutputPath()))
		writeFile(t, targetPath, printed)
		targetPaths = append(targetPaths, targetPath)
	}
	if sourceModule == "" {
		t.Fatal("safe integer source module is absent")
	}

	runnerPath := filepath.Join(workingDirectory, "runner.ts")
	writeFile(t, runnerPath, `import "./program.js";
import { Small } from "`+sourceModule+`";

console.log(Small());
`)
	writeFile(t, filepath.Join(workingDirectory, "package.json"), "{\"type\":\"module\"}\n")
	outputDirectory := filepath.Join(workingDirectory, "out")
	arguments := []string{
		"--target", "es2022",
		"--module", "nodenext",
		"--moduleResolution", "nodenext",
		"--strict",
		"--outDir", outputDirectory,
	}
	arguments = append(arguments, targetPaths...)
	arguments = append(arguments, runnerPath)
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
	targetOutput := run(
		t,
		workingDirectory,
		"node",
		filepath.Join(outputDirectory, "runner.js"),
	)
	if goOutput := executeSafeIntegerGo(t, workingDirectory); targetOutput != goOutput {
		t.Fatalf("TypeScript output = %q, Go output = %q", targetOutput, goOutput)
	}
}

func TestBigIntProfileTypechecksAndExecutesBeyondSafeNumberRange(t *testing.T) {
	loaded := loadIntegerConstantsProject(t)
	roots, err := emit.ExportedAPIRoots(loaded)
	if err != nil {
		t.Fatal(err)
	}
	emission, err := emit.CompileWithOptions(
		loaded.Program(),
		roots,
		emit.Options{
			IntegerRepresentation: emit.IntegerRepresentationBigInt,
			EvaluationOrder:       emit.EvaluationOrderDirect,
		},
	)
	if err != nil {
		t.Fatal(err)
	}

	workingDirectory := t.TempDir()
	client, err := tsgo.StartClient(repositoryRoot(), workingDirectory)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := client.Close(); err != nil {
			t.Errorf("close TS-Go client: %v", err)
		}
	})
	var sourceModule string
	var targetPaths []string
	for _, file := range emission.Files() {
		printed, err := client.PrintNode(file.SourceFile(), tsgo.PrintOptions{})
		if err != nil {
			t.Fatal(err)
		}
		if file.Kind() == emit.TargetFileSupport &&
			!strings.Contains(printed, "export type int64 = bigint;") {
			t.Fatalf("BigInt support artifact:\n%s", printed)
		}
		if file.Kind() == emit.TargetFileSource {
			sourceModule = "./" + strings.TrimSuffix(file.OutputPath(), ".ts") + ".js"
		}
		targetPath := filepath.Join(
			workingDirectory,
			filepath.FromSlash(file.OutputPath()),
		)
		writeFile(t, targetPath, printed)
		targetPaths = append(targetPaths, targetPath)
	}
	if sourceModule == "" {
		t.Fatal("BigInt source module is absent")
	}

	runnerPath := filepath.Join(workingDirectory, "runner.ts")
	writeFile(t, runnerPath, `import {
    BeyondSafe,
    Maximum,
    Minimum,
    WideAdd,
    WideCompound,
    WideIncrement,
    WideLess,
    WideSwitch,
} from "`+sourceModule+`";

const render = (value: bigint): string => value.toString();
console.log(render(BeyondSafe()));
console.log(render(Maximum()));
console.log(render(Minimum()));
console.log(render(WideAdd(9007199254740990n, 3n)));
console.log(WideLess(9007199254740993n, 9007199254740994n));
console.log(render(WideSwitch(0n)));
console.log(render(WideSwitch(7n)));
console.log(render(WideCompound(9007199254740993n, 4n)));
console.log(render(WideIncrement(9007199254740993n)));
`)
	writeFile(t, filepath.Join(workingDirectory, "package.json"), "{\"type\":\"module\"}\n")
	outputDirectory := filepath.Join(workingDirectory, "out")
	arguments := []string{
		"--target", "es2022",
		"--module", "nodenext",
		"--moduleResolution", "nodenext",
		"--strict",
		"--outDir", outputDirectory,
	}
	arguments = append(arguments, targetPaths...)
	arguments = append(arguments, runnerPath)
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
	targetOutput := run(
		t,
		workingDirectory,
		"node",
		filepath.Join(outputDirectory, "runner.js"),
	)
	goOutput := executeWideIntegerGo(t, workingDirectory)
	if targetOutput != goOutput {
		t.Fatalf("TypeScript output = %q, Go output = %q", targetOutput, goOutput)
	}
}

func TestWideIntegerConstantsFailAtTheirExpressionOwner(t *testing.T) {
	for _, declarationIndex := range []int{1, 2, 3} {
		loaded := loadIntegerConstantsProject(t)
		_, err := compileIntegerRootError(loaded, declarationIndex)
		var unsupported *api.UnsupportedError
		if !errors.As(err, &unsupported) ||
			unsupported.Category != api.CategoryExpression {
			t.Fatalf(
				"declaration %d error = %#v, want expression UnsupportedError",
				declarationIndex,
				err,
			)
		}
	}
}

func TestWideIntegerOperationsUseTheSelectedCarrier(t *testing.T) {
	for declarationIndex := 4; declarationIndex <= 9; declarationIndex++ {
		loaded := loadIntegerConstantsProject(t)
		if _, err := compileIntegerRootError(loaded, declarationIndex); err != nil {
			t.Fatalf("declaration %d: %v", declarationIndex, err)
		}
	}
}

func TestIntegerConstantUsesGoValueNotLiteralSpelling(t *testing.T) {
	loaded := loadIntegerConstantsProject(t)
	small := loaded.Files()[0].Syntax().Decls[0].(*ast.FuncDecl)
	literal := small.Body.List[0].(*ast.ReturnStmt).Results[0].(*ast.BasicLit)
	literal.Value = "43"

	emission := compileIntegerRoot(t, loaded, 0)
	target := integerSourceFile(t, emission)
	function := target.Statements()[1].(tsgo.FunctionDeclaration)
	result := function.Body().(tsgo.Block).Statements()[0].(tsgo.ReturnStatement)
	if text := result.Expression().(tsgo.NumericLiteral).Text(); text != "42" {
		t.Fatalf("semantic literal = %q, want 42", text)
	}
}

func TestIntegerConstantRejectsNonIntegerSyntaxMutation(t *testing.T) {
	loaded := loadIntegerConstantsProject(t)
	small := loaded.Files()[0].Syntax().Decls[0].(*ast.FuncDecl)
	literal := small.Body.List[0].(*ast.ReturnStmt).Results[0].(*ast.BasicLit)
	literal.Kind = token.FLOAT

	_, err := compileIntegerRootError(loaded, 0)
	var unsupported *api.UnsupportedError
	if !errors.As(err, &unsupported) ||
		unsupported.Category != api.CategoryExpression ||
		unsupported.Construct != "*ast.BasicLit" ||
		unsupported.Role != api.RoleReturnResult {
		t.Fatalf("unsupported error = %#v", unsupported)
	}
}

func compileIntegerRoot(
	t *testing.T,
	loaded *load.Package,
	declarationIndex int,
) emit.ProgramEmission {
	t.Helper()
	emission, err := compileIntegerRootError(loaded, declarationIndex)
	if err != nil {
		t.Fatal(err)
	}
	return emission
}

func compileIntegerRootError(
	loaded *load.Package,
	declarationIndex int,
) (emit.ProgramEmission, error) {
	declaration := loaded.Files()[0].Syntax().Decls[declarationIndex].(*ast.FuncDecl)
	root, err := emit.NewRoot(loaded.TypesInfo().Defs[declaration.Name])
	if err != nil {
		return emit.ProgramEmission{}, err
	}
	return emit.Compile(loaded.Program(), []emit.Root{root})
}

func integerSourceFile(t *testing.T, emission emit.ProgramEmission) tsgo.SourceFile {
	t.Helper()
	for _, file := range emission.Files() {
		if file.Kind() == emit.TargetFileSource {
			return file.SourceFile()
		}
	}
	t.Fatal("integer source file is absent")
	return nil
}

func loadIntegerConstantsProject(t *testing.T) *load.Package {
	t.Helper()
	loaded, err := load.One(context.Background(), load.Request{
		Directory: integerConstantsProjectDirectory(),
		Pattern:   ".",
	})
	if err != nil {
		t.Fatal(err)
	}
	return loaded
}

func printTargetFile(
	t *testing.T,
	targetFile tsgo.SourceFile,
	workingDirectory string,
) string {
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
	printed, err := client.PrintNode(targetFile, tsgo.PrintOptions{})
	if err != nil {
		t.Fatal(err)
	}
	return printed
}

func executeSafeIntegerGo(t *testing.T, workingDirectory string) string {
	t.Helper()
	modulePath, err := filepath.Abs(integerConstantsProjectDirectory())
	if err != nil {
		t.Fatal(err)
	}
	runnerDirectory := filepath.Join(workingDirectory, "go-runner")
	writeFile(t, filepath.Join(runnerDirectory, "go.mod"), fmt.Sprintf(`module example.com/runner

go 1.26.4

require example.com/integerconstants v0.0.0

replace example.com/integerconstants => %s
`, filepath.ToSlash(modulePath)))
	writeFile(t, filepath.Join(runnerDirectory, "main.go"), `package main

import (
	"fmt"

	constants "example.com/integerconstants"
)

func main() {
	fmt.Println(constants.Small())
}
`)
	return run(t, runnerDirectory, filepath.Join(runtime.GOROOT(), "bin", "go"), "run", ".")
}

func executeWideIntegerGo(t *testing.T, workingDirectory string) string {
	t.Helper()
	modulePath, err := filepath.Abs(integerConstantsProjectDirectory())
	if err != nil {
		t.Fatal(err)
	}
	runnerDirectory := filepath.Join(workingDirectory, "go-wide-runner")
	writeFile(t, filepath.Join(runnerDirectory, "go.mod"), fmt.Sprintf(`module example.com/runner

go 1.26.4

require example.com/integerconstants v0.0.0

replace example.com/integerconstants => %s
`, filepath.ToSlash(modulePath)))
	writeFile(t, filepath.Join(runnerDirectory, "main.go"), `package main

import (
	"fmt"

	constants "example.com/integerconstants"
)

func main() {
	fmt.Println(constants.BeyondSafe())
	fmt.Println(constants.Maximum())
	fmt.Println(constants.Minimum())
	fmt.Println(constants.WideAdd(9007199254740990, 3))
	fmt.Println(constants.WideLess(9007199254740993, 9007199254740994))
	fmt.Println(constants.WideSwitch(0))
	fmt.Println(constants.WideSwitch(7))
	fmt.Println(constants.WideCompound(9007199254740993, 4))
	fmt.Println(constants.WideIncrement(9007199254740993))
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

func integerConstantsProjectDirectory() string {
	return filepath.Join(repositoryRoot(), "testdata", "projects", "integer-constants")
}
