package float_test

import (
	"bytes"
	"context"
	"go/ast"
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

func repositoryRoot() string {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		panic("resolve float-operators repository root")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", "..", "..", "..", ".."))
}

func operatorsDirectory() string {
	return filepath.Join(repositoryRoot(), "testdata", "constructs", "expression", "float-operators")
}

// TestFloatOperatorsExecuteDifferentially proves float64 arithmetic, ordering,
// equality, and unary negation carry IEEE-754 semantics to TypeScript exactly:
// division by zero yields ±Infinity/NaN (never a panic), NaN compares false
// under every ordering and unequal to itself, and signed zeros compare equal.
func TestFloatOperatorsExecuteDifferentially(t *testing.T) {
	emission := compileOperators(t, loadOperators(t))

	workingDirectory := t.TempDir()
	client, err := tsgo.StartClient(repositoryRoot(), workingDirectory)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = client.Close() })

	var targetPaths []string
	sourceModule := ""
	for _, file := range emission.Files() {
		text, err := client.PrintNode(file.SourceFile(), tsgo.PrintOptions{})
		if err != nil {
			t.Fatal(err)
		}
		targetPath := filepath.Join(workingDirectory, filepath.FromSlash(file.OutputPath()))
		writeFile(t, targetPath, text)
		targetPaths = append(targetPaths, targetPath)
		if file.Kind() == emit.TargetFileSource && filepath.Base(file.OutputPath()) == "source.ts" {
			sourceModule = "./" + strings.TrimSuffix(file.OutputPath(), ".ts") + ".js"
		}
	}

	goOutput := runOperatorsGo(t, workingDirectory)
	tsOutput := runOperatorsTS(t, workingDirectory, targetPaths, sourceModule)
	if tsOutput != goOutput {
		t.Fatalf("TypeScript output = %q, Go output = %q", tsOutput, goOutput)
	}
}

func TestFloatOperatorsHaveExactTargetShape(t *testing.T) {
	emission := compileOperators(t, loadOperators(t))
	source := operatorsSourceFile(t, emission)

	arithmetic := map[string]tsgo.SyntaxKind{
		"Add":   tsgo.SyntaxKindPlusToken,
		"Sub":   tsgo.SyntaxKindMinusToken,
		"Mul":   tsgo.SyntaxKindAsteriskToken,
		"Div":   tsgo.SyntaxKindSlashToken,
		"Add32": tsgo.SyntaxKindPlusToken,
		"Sub32": tsgo.SyntaxKindMinusToken,
		"Mul32": tsgo.SyntaxKindAsteriskToken,
		"Div32": tsgo.SyntaxKindSlashToken,
	}
	for name, operator := range arithmetic {
		t.Run(name, func(t *testing.T) {
			value := functionReturn(t, source, name)
			if strings.HasSuffix(name, "32") {
				call, ok := value.(tsgo.CallExpression)
				if !ok {
					t.Fatalf("%s returns %T, want goFloat32 call", name, value)
				}
				callee, ok := call.Expression().(tsgo.Identifier)
				if !ok || callee.Text() != "goFloat32" {
					t.Fatalf("%s calls %T, want goFloat32", name, call.Expression())
				}
				if len(call.Arguments()) != 1 {
					t.Fatalf("%s goFloat32 arguments = %d, want 1", name, len(call.Arguments()))
				}
				value = call.Arguments()[0]
			}
			binary, ok := value.(tsgo.BinaryExpression)
			if !ok || binary.OperatorToken().Kind() != operator {
				t.Fatalf(
					"%s value = %T operator %v, want direct operator %v",
					name,
					value,
					binaryOperatorKind(value),
					operator,
				)
			}
		})
	}

	comparisons := map[string]tsgo.SyntaxKind{
		"Less":           tsgo.SyntaxKindLessThanToken,
		"LessEqual":      tsgo.SyntaxKindLessThanEqualsToken,
		"Greater":        tsgo.SyntaxKindGreaterThanToken,
		"GreaterEqual":   tsgo.SyntaxKindGreaterThanEqualsToken,
		"Equal":          tsgo.SyntaxKindEqualsEqualsEqualsToken,
		"NotEqual":       tsgo.SyntaxKindExclamationEqualsEqualsToken,
		"Less32":         tsgo.SyntaxKindLessThanToken,
		"LessEqual32":    tsgo.SyntaxKindLessThanEqualsToken,
		"Greater32":      tsgo.SyntaxKindGreaterThanToken,
		"GreaterEqual32": tsgo.SyntaxKindGreaterThanEqualsToken,
		"Equal32":        tsgo.SyntaxKindEqualsEqualsEqualsToken,
		"NotEqual32":     tsgo.SyntaxKindExclamationEqualsEqualsToken,
	}
	for name, operator := range comparisons {
		t.Run(name, func(t *testing.T) {
			value := functionReturn(t, source, name)
			binary, ok := value.(tsgo.BinaryExpression)
			if !ok || binary.OperatorToken().Kind() != operator {
				t.Fatalf(
					"%s value = %T operator %v, want direct operator %v",
					name,
					value,
					binaryOperatorKind(value),
					operator,
				)
			}
		})
	}

	if rounds := countFloat32Rounds(functionReturn(t, source, "Nested32Case")); rounds != 2 {
		t.Fatalf("Nested32Case rounding calls = %d, want one per arithmetic boundary (2)", rounds)
	}
	for _, name := range []string{"Negate", "Negate32"} {
		value := functionReturn(t, source, name)
		unary, ok := value.(tsgo.PrefixUnaryExpression)
		if !ok ||
			unary.Operator() != tsgo.PrefixUnaryExpressionOperatorKindMinusToken {
			t.Fatalf("%s returns %T, want direct unary minus", name, value)
		}
	}
	for _, name := range []string{"Identity", "Identity32"} {
		value := functionReturn(t, source, name)
		if _, ok := value.(tsgo.Identifier); !ok {
			t.Fatalf("%s returns %T, want its operand directly", name, value)
		}
	}
	roundDefinitions := 0
	for _, file := range emission.Files() {
		for _, statement := range file.SourceFile().Statements() {
			function, ok := statement.(tsgo.FunctionDeclaration)
			if ok && function.Name().Text() == "goFloat32" {
				roundDefinitions++
			}
		}
	}
	if roundDefinitions != 1 {
		t.Fatalf("goFloat32 definitions = %d, want exactly 1", roundDefinitions)
	}
}

func TestRemovingFloat32RoundingChangesGoBehavior(t *testing.T) {
	emission := compileOperators(t, loadOperators(t))
	workingDirectory := t.TempDir()
	client, err := tsgo.StartClient(repositoryRoot(), workingDirectory)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = client.Close() })

	var targetPaths []string
	sourceModule := ""
	mutated := false
	for _, file := range emission.Files() {
		source := file.SourceFile()
		if file.Kind() == emit.TargetFileSource &&
			file.PackageName() == "floatops" {
			source = removeFloat32Round(t, source, "Add32")
			mutated = true
		}
		text, err := client.PrintNode(source, tsgo.PrintOptions{})
		if err != nil {
			t.Fatal(err)
		}
		targetPath := filepath.Join(
			workingDirectory,
			filepath.FromSlash(file.OutputPath()),
		)
		writeFile(t, targetPath, text)
		targetPaths = append(targetPaths, targetPath)
		if file.Kind() == emit.TargetFileSource &&
			filepath.Base(file.OutputPath()) == "source.ts" {
			sourceModule = "./" +
				strings.TrimSuffix(file.OutputPath(), ".ts") +
				".js"
		}
	}
	if !mutated {
		t.Fatal("float32 production artifact was not mutated")
	}
	goOutput := runOperatorsGo(t, workingDirectory)
	targetOutput := runOperatorsTS(
		t,
		workingDirectory,
		targetPaths,
		sourceModule,
	)
	if targetOutput == goOutput {
		t.Fatal("removing the Add32 rounding boundary preserved Go behavior")
	}
}

func TestFloatLiteralSpellingDoesNotOwnValue(t *testing.T) {
	baseline := compileOperators(t, loadOperators(t))
	mutatedInput := loadOperators(t)
	mutated := false
	for _, file := range mutatedInput.Files() {
		ast.Inspect(file.Syntax(), func(node ast.Node) bool {
			literal, ok := node.(*ast.BasicLit)
			if ok && literal.Value == "3.4e38" {
				literal.Value = "1"
				mutated = true
				return false
			}
			return true
		})
	}
	if !mutated {
		t.Fatal("float literal mutation target was absent")
	}
	changedSyntax := compileOperators(t, mutatedInput)
	assertProgramASTEqual(t, baseline, changedSyntax)
}

func loadOperators(t *testing.T) *load.Package {
	t.Helper()
	loaded, err := load.One(
		context.Background(),
		load.Request{Directory: operatorsDirectory(), Pattern: "."},
	)
	if err != nil {
		t.Fatal(err)
	}
	return loaded
}

func compileOperators(t *testing.T, loaded *load.Package) emit.ProgramEmission {
	t.Helper()
	roots, err := emit.ExportedAPIRoots(loaded)
	if err != nil {
		t.Fatal(err)
	}
	emission, err := emit.Compile(loaded.Program(), roots)
	if err != nil {
		t.Fatalf("float-operators compile failed: %v", err)
	}
	return emission
}

func operatorsSourceFile(
	t *testing.T,
	emission emit.ProgramEmission,
) tsgo.SourceFile {
	t.Helper()
	for _, file := range emission.Files() {
		if file.Kind() == emit.TargetFileSource &&
			file.PackageName() == "floatops" {
			return file.SourceFile()
		}
	}
	t.Fatal("floatops source artifact is absent")
	return nil
}

func functionReturn(
	t *testing.T,
	source tsgo.SourceFile,
	name string,
) tsgo.Expression {
	t.Helper()
	for _, statement := range source.Statements() {
		function, ok := statement.(tsgo.FunctionDeclaration)
		if !ok || function.Name().Text() != name {
			continue
		}
		for _, bodyStatement := range function.Body().(tsgo.Block).Statements() {
			if returned, ok := bodyStatement.(tsgo.ReturnStatement); ok {
				return returned.Expression()
			}
		}
		t.Fatalf("%s has no target return statement", name)
	}
	t.Fatalf("target function %s is absent", name)
	return nil
}

func binaryOperatorKind(value tsgo.Expression) tsgo.SyntaxKind {
	binary, ok := value.(tsgo.BinaryExpression)
	if !ok {
		return tsgo.SyntaxKindUnknown
	}
	return binary.OperatorToken().Kind()
}

func countFloat32Rounds(value tsgo.Expression) int {
	switch value := value.(type) {
	case tsgo.CallExpression:
		count := 0
		if callee, ok := value.Expression().(tsgo.Identifier); ok &&
			callee.Text() == "goFloat32" {
			count = 1
		}
		for _, argument := range value.Arguments() {
			count += countFloat32Rounds(argument)
		}
		return count
	case tsgo.BinaryExpression:
		return countFloat32Rounds(value.Left()) +
			countFloat32Rounds(value.Right())
	case tsgo.ParenthesizedExpression:
		return countFloat32Rounds(value.Expression())
	case tsgo.PrefixUnaryExpression:
		return countFloat32Rounds(value.Operand())
	default:
		return 0
	}
}

func removeFloat32Round(
	t *testing.T,
	source tsgo.SourceFile,
	name string,
) tsgo.SourceFile {
	t.Helper()
	factory := tsgo.NewFactory()
	statements := source.Statements()
	mutated := false
	for index, statement := range statements {
		function, ok := statement.(tsgo.FunctionDeclaration)
		if !ok || function.Name().Text() != name {
			continue
		}
		body := function.Body().(tsgo.Block)
		bodyStatements := body.Statements()
		for bodyIndex, bodyStatement := range bodyStatements {
			returned, ok := bodyStatement.(tsgo.ReturnStatement)
			if !ok {
				continue
			}
			round, ok := returned.Expression().(tsgo.CallExpression)
			if !ok || len(round.Arguments()) != 1 {
				t.Fatalf("%s does not return one rounding call", name)
			}
			bodyStatements[bodyIndex] = factory.ReturnStatement(
				round.Arguments()[0],
			)
			mutated = true
		}
		statements[index] = factory.FunctionDeclaration(
			function.Modifiers(),
			function.AsteriskToken(),
			function.Name(),
			function.TypeParameters(),
			function.Parameters(),
			function.Type(),
			factory.Block(bodyStatements, body.MultiLine()),
		)
	}
	if !mutated {
		t.Fatalf("target function %s was not mutated", name)
	}
	return factory.SourceFile(
		statements,
		source.EndOfFileToken(),
		source.SourceData(),
	)
}

func assertProgramASTEqual(
	t *testing.T,
	left emit.ProgramEmission,
	right emit.ProgramEmission,
) {
	t.Helper()
	leftFiles := left.Files()
	rightFiles := right.Files()
	if len(leftFiles) != len(rightFiles) {
		t.Fatalf("file counts differ: %d and %d", len(leftFiles), len(rightFiles))
	}
	for index := range leftFiles {
		if leftFiles[index].OutputPath() != rightFiles[index].OutputPath() {
			t.Fatalf(
				"file %d paths differ: %s and %s",
				index,
				leftFiles[index].OutputPath(),
				rightFiles[index].OutputPath(),
			)
		}
		leftBytes, err := tsgo.EncodeSourceFile(leftFiles[index].SourceFile())
		if err != nil {
			t.Fatal(err)
		}
		rightBytes, err := tsgo.EncodeSourceFile(rightFiles[index].SourceFile())
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(leftBytes, rightBytes) {
			t.Fatalf(
				"target AST for %s changed when only stale source spelling changed",
				leftFiles[index].OutputPath(),
			)
		}
	}
}

func runOperatorsGo(t *testing.T, workingDirectory string) string {
	t.Helper()
	modulePath, err := filepath.Abs(operatorsDirectory())
	if err != nil {
		t.Fatal(err)
	}
	runnerDirectory := filepath.Join(workingDirectory, "go-runner")
	writeFile(t, filepath.Join(runnerDirectory, "go.mod"), `module example.com/runner

go 1.26.4

require example.com/floatops v0.0.0

replace example.com/floatops => `+filepath.ToSlash(modulePath)+`
`)
	writeFile(t, filepath.Join(runnerDirectory, "main.go"), `package main

import (
	"fmt"
	"math"
	"strconv"

	values "example.com/floatops"
)

func js(f float64) string {
	switch {
	case math.IsInf(f, 1):
		return "Infinity"
	case math.IsInf(f, -1):
		return "-Infinity"
	case math.IsNaN(f):
		return "NaN"
	case f == 0 && math.Signbit(f):
		return "-0"
	default:
		return strconv.FormatFloat(f, 'g', -1, 64)
	}
}

func main() {
	nan := values.Div(0, 0)
	fmt.Println(js(values.Add(1.5, 2.5)))
	fmt.Println(js(values.Sub(5, 3)))
	fmt.Println(js(values.Mul(2.5, 4)))
	fmt.Println(js(values.Div(7, 2)))
	fmt.Println(js(values.Div(1, 0)))
	fmt.Println(js(values.Div(-1, 0)))
	fmt.Println(js(nan))
	fmt.Println(js(values.Negate(3.5)))
	fmt.Println(js(values.Identity(-2.25)))
	fmt.Println(js(values.ConstantNeg()))
	fmt.Println(values.Less(1, 2), values.Less(nan, 1))
	fmt.Println(values.LessEqual(2, 2), values.GreaterEqual(2, 2))
	fmt.Println(values.Greater(3, 2), values.Greater(nan, nan))
	fmt.Println(values.Equal(0, math.Copysign(0, -1)), values.Equal(nan, nan))
	fmt.Println(values.NotEqual(nan, nan), values.NotEqual(1, 1))
	fmt.Println(js(float64(values.Add32Case())))
	fmt.Println(js(float64(values.Sub32Case())))
	fmt.Println(js(float64(values.Mul32Case())))
	fmt.Println(js(float64(values.Div32Case())))
	fmt.Println(js(float64(values.Negate32Case())))
	fmt.Println(js(float64(values.Identity32Case())))
	fmt.Println(js(float64(values.Nested32Case())))
	fmt.Println(js(float64(values.Overflow32Case())))
	fmt.Println(js(float64(values.Underflow32Case())))
	fmt.Println(js(float64(values.Subnormal32Case())))
	fmt.Println(js(float64(values.NegativeZero32Case())))
	fmt.Println(js(float64(values.Infinity32Case())))
	fmt.Println(js(float64(values.NaN32Case())))
	fmt.Println(values.Comparisons32Case())
	fmt.Println(values.ComparisonEdges32Case())
}
`)
	return runCommand(t, runnerDirectory, filepath.Join(runtime.GOROOT(), "bin", "go"), "run", ".")
}

func runOperatorsTS(t *testing.T, workingDirectory string, targetPaths []string, sourceModule string) string {
	t.Helper()
	runner := `import * as values from "` + sourceModule + `";
const nan = values.Div(0, 0);
console.log(String(values.Add(1.5, 2.5)));
console.log(String(values.Sub(5, 3)));
console.log(String(values.Mul(2.5, 4)));
console.log(String(values.Div(7, 2)));
console.log(String(values.Div(1, 0)));
console.log(String(values.Div(-1, 0)));
console.log(String(nan));
console.log(String(values.Negate(3.5)));
console.log(String(values.Identity(-2.25)));
console.log(String(values.ConstantNeg()));
console.log(values.Less(1, 2), values.Less(nan, 1));
console.log(values.LessEqual(2, 2), values.GreaterEqual(2, 2));
console.log(values.Greater(3, 2), values.Greater(nan, nan));
console.log(values.Equal(0, -0), values.Equal(nan, nan));
console.log(values.NotEqual(nan, nan), values.NotEqual(1, 1));
const js = (value: number): string =>
    Object.is(value, -0) ? "-0" : String(value);
console.log(js(values.Add32Case()));
console.log(js(values.Sub32Case()));
console.log(js(values.Mul32Case()));
console.log(js(values.Div32Case()));
console.log(js(values.Negate32Case()));
console.log(js(values.Identity32Case()));
console.log(js(values.Nested32Case()));
console.log(js(values.Overflow32Case()));
console.log(js(values.Underflow32Case()));
console.log(js(values.Subnormal32Case()));
console.log(js(values.NegativeZero32Case()));
console.log(js(values.Infinity32Case()));
console.log(js(values.NaN32Case()));
console.log(...values.Comparisons32Case());
console.log(...values.ComparisonEdges32Case());
`
	runnerPath := filepath.Join(workingDirectory, "runner.ts")
	writeFile(t, runnerPath, runner)
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
	if err := runtimefixture.InstallResolution(workingDirectory, outputDirectory); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := tsgo.Compile(ctx, repositoryRoot(), workingDirectory, arguments); err != nil {
		t.Fatalf("float-operators program failed strict typecheck: %v", err)
	}
	return runCommand(t, workingDirectory, "node", filepath.Join(outputDirectory, "runner.js"))
}

func runCommand(t *testing.T, directory, name string, arguments ...string) string {
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
