package stringvalue_test

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

func TestStringFamilyCreatesExactTargetTrees(t *testing.T) {
	emission := compileStringFixture(t)
	source := targetFile(t, emission, emit.TargetFileSource)

	assertByteLiteral(
		t,
		returnExpression(t, source, "UTF8"),
		[]byte{0xc3, 0xa9},
	)
	assertByteLiteral(
		t,
		returnExpression(t, source, "RawUTF8"),
		[]byte{0xc3, 0xa9},
	)
	assertByteLiteral(
		t,
		returnExpression(t, source, "InvalidBytes"),
		[]byte{0xff, 0x00, 0x41},
	)
	assertRuntimeCall(t, returnExpression(t, source, "ByteAt"), "goStringIndex", 2)
	assertRuntimeCall(t, returnExpression(t, source, "Window"), "goStringSlice", 3)
	assertRuntimeCall(t, returnExpression(t, source, "Prefix"), "goStringSlice", 3)
	assertRuntimeCall(t, returnExpression(t, source, "Suffix"), "goStringSlice", 2)

	length, ok := returnExpression(t, source, "Length").(tsgo.PropertyAccessExpression)
	lengthName, nameOK := length.Name().(tsgo.Identifier)
	if !ok || !nameOK || lengthName.Text() != "length" {
		t.Fatalf("Length return = %T, want direct .length", returnExpression(t, source, "Length"))
	}

	runtimeFile := targetFile(t, emission, emit.TargetFileSupport, "runtime/string.ts")
	assertBoundsFunction(t, targetFunction(t, runtimeFile, "goStringIndex"))
	assertBoundsFunction(t, targetFunction(t, runtimeFile, "goStringSlice"))
}

func TestStringShapeChecksRejectRequiredMutations(t *testing.T) {
	factory := tsgo.NewFactory()
	if byteLiteralMatches(factory.StringLiteral("é", tsgo.TokenFlagsNone), []byte{0xc3, 0xa9}) {
		t.Fatal("code-point literal mutation passed byte-preserving literal check")
	}
	nativeIndex := factory.ElementAccessExpression(
		factory.Identifier("value"),
		nil,
		factory.Identifier("index"),
		tsgo.NodeFlagsNone,
	)
	if runtimeCallMatches(nativeIndex, "goStringIndex", 2) {
		t.Fatal("UTF-16 element-access mutation passed runtime-index check")
	}
	unchecked := factory.FunctionDeclaration(
		[]tsgo.ModifierLike{factory.ExportKeyword()},
		nil,
		factory.Identifier("goStringIndex"),
		nil,
		nil,
		factory.KeywordTypeNode(tsgo.KeywordTypeSyntaxKindNumberKeyword),
		factory.Block(
			[]tsgo.Statement{
				factory.ReturnStatement(factory.NumericLiteral("0", tsgo.TokenFlagsNone)),
			},
			true,
		),
	)
	if hasPanicBoundsCheck(unchecked) {
		t.Fatal("missing-bounds-check mutation passed runtime bounds check")
	}
}

func TestStringFamilyPrintsTypechecksAndExecutesDifferentially(t *testing.T) {
	emission := compileStringFixture(t)
	workingDirectory := t.TempDir()
	targetPaths, module, printed := materializeStringProgram(t, workingDirectory, emission)
	for _, forbidden := range []string{
		"any",
		"unknown",
		".charAt(",
		".charCodeAt(index)",
		"value[index]",
	} {
		if strings.Contains(printed, forbidden) {
			t.Fatalf("printed string artifacts contain %q:\n%s", forbidden, printed)
		}
	}

	runnerPath := filepath.Join(workingDirectory, "runner.ts")
	writeFile(t, runnerPath, `import {
    ASCII,
    Assign,
    ByteAt,
    Concat,
    ConstantByteAt,
    Constants,
    DefinedBounds,
    DefinedWindow,
    Equal,
    Greater,
    GreaterEqual,
    IndexCall,
    InvalidBytes,
    Length,
    Less,
    LessEqual,
    NotEqual,
    PackageAssign,
    PackageZero,
    Path,
    Prefix,
    RawUTF8,
    Suffix,
    SuffixCall,
    UTF8,
    Window,
    Zero,
} from "`+module+`";
import "./program.js";

function bytes(value: string): string {
    return Array.from(value, value => value.charCodeAt(0).toString(16).padStart(2, "0")).join("");
}

function panics(operation: () => void): boolean {
    try {
        operation();
        return false;
    } catch {
        return true;
    }
}

console.log(bytes(ASCII()));
console.log(bytes(UTF8()));
console.log(bytes(RawUTF8()));
console.log(bytes(InvalidBytes()));
console.log(bytes(Constants()));
console.log(bytes(Zero()));
console.log(bytes(Assign("\u00ff\u0000A")));
console.log(bytes(PackageZero()));
console.log(bytes(PackageAssign("\u00fe")));
console.log(bytes(Concat("\u00c3", "\u00a9")));
console.log(Equal("a", "a"));
console.log(NotEqual("a", "b"));
console.log(Less("a", "b"));
console.log(LessEqual("a", "a"));
console.log(Greater("b", "a"));
console.log(GreaterEqual("b", "b"));
console.log(Less("\u0080", "\u00ff"));
console.log(Less("\u00c3\u00a9", "\u00ff"));
console.log(Length(UTF8()).toString());
console.log(ByteAt(UTF8(), 1).toString());
console.log(ConstantByteAt(10).toString());
console.log(bytes(Window(UTF8(), 0, 1)));
console.log(bytes(Prefix(UTF8(), 1)));
console.log(bytes(Suffix(UTF8(), 1)));
const [suffixCallValue, suffixCallCount] = SuffixCall(1);
console.log(bytes(suffixCallValue));
console.log(suffixCallCount.toString());
const [indexCallValue, indexCallCount] = IndexCall(1);
console.log(indexCallValue.toString());
console.log(indexCallCount.toString());
const [definedByte, definedWindow] = DefinedBounds();
console.log(definedByte.toString());
console.log(bytes(definedWindow));
console.log(bytes(DefinedWindow(new Path("abcd"), 1, 3).$value));
console.log(panics(() => ByteAt("a", -1)));
console.log(panics(() => ByteAt("a", 1)));
console.log(panics(() => ByteAt("a", 9007199254740992)));
console.log(panics(() => Window("a", -1, 0)));
console.log(panics(() => Window("a", 1, 0)));
console.log(panics(() => Window("a", 0, 2)));
console.log(panics(() => Suffix("a", -1)));
console.log(panics(() => Prefix("a", 2)));
`)
	writeFile(t, filepath.Join(workingDirectory, "package.json"), "{\"type\":\"module\"}\n")
	targetPaths = append(targetPaths, runnerPath)
	compileTypeScript(t, workingDirectory, targetPaths)
	targetOutput := run(
		t,
		workingDirectory,
		"node",
		filepath.Join(workingDirectory, "out", "runner.js"),
	)
	goOutput := executeGo(t, workingDirectory)
	if targetOutput != goOutput {
		t.Fatalf("TypeScript output = %q, Go output = %q", targetOutput, goOutput)
	}
}

func TestStringFamilyStrictTypechecksWithBigIntIndices(t *testing.T) {
	emission := compileStringFixtureWithOptions(t, emit.Options{
		IntegerRepresentation: emit.IntegerRepresentationBigInt,
		EvaluationOrder:       emit.EvaluationOrderDirect,
	})
	workingDirectory := t.TempDir()
	targetPaths, _, printed := materializeStringProgram(t, workingDirectory, emission)
	writeFile(t, filepath.Join(workingDirectory, "package.json"), "{\"type\":\"module\"}\n")
	compileTypeScript(t, workingDirectory, targetPaths)
	if !strings.Contains(printed, "BigInt(value.length)") ||
		!strings.Contains(printed, "goStringSlice(value, 0n, high)") {
		t.Fatalf("BigInt string artifact lacks exact index adaptation:\n%s", printed)
	}
}

func compileStringFixture(t *testing.T) emit.ProgramEmission {
	t.Helper()
	return compileStringFixtureWithOptions(t, emit.Options{
		IntegerRepresentation: emit.IntegerRepresentationNumber,
		EvaluationOrder:       emit.EvaluationOrderDirect,
	})
}

func compileStringFixtureWithOptions(
	t *testing.T,
	options emit.Options,
) emit.ProgramEmission {
	t.Helper()
	program, err := load.Load(context.Background(), load.Request{
		Directory: stringFixtureDirectory(),
		Pattern:   ".",
	})
	if err != nil {
		t.Fatal(err)
	}
	roots, err := emit.ExportedAPIRoots(program.Roots()[0])
	if err != nil {
		t.Fatal(err)
	}
	emission, err := emit.CompileWithOptions(program, roots, options)
	if err != nil {
		t.Fatal(err)
	}
	return emission
}

func targetFile(
	t *testing.T,
	emission emit.ProgramEmission,
	kind emit.TargetFileKind,
	outputPath ...string,
) tsgo.SourceFile {
	t.Helper()
	var result tsgo.SourceFile
	for _, file := range emission.Files() {
		if file.Kind() != kind ||
			(len(outputPath) != 0 && file.OutputPath() != outputPath[0]) {
			continue
		}
		if result != nil {
			t.Fatalf("multiple target files match kind %d path %v", kind, outputPath)
		}
		result = file.SourceFile()
	}
	if result == nil {
		t.Fatalf("target file kind %d path %v is absent", kind, outputPath)
	}
	return result
}

func targetFunction(
	t *testing.T,
	source tsgo.SourceFile,
	name string,
) tsgo.FunctionDeclaration {
	t.Helper()
	for _, statement := range source.Statements() {
		function, ok := statement.(tsgo.FunctionDeclaration)
		if ok && function.Name().Text() == name {
			return function
		}
	}
	t.Fatalf("target function %q is absent", name)
	return nil
}

func returnExpression(
	t *testing.T,
	source tsgo.SourceFile,
	name string,
) tsgo.Expression {
	t.Helper()
	function := targetFunction(t, source, name)
	body := function.Body().(tsgo.Block)
	for _, statement := range body.Statements() {
		if target, ok := statement.(tsgo.ReturnStatement); ok {
			return target.Expression()
		}
	}
	t.Fatalf("target function %q has no return", name)
	return nil
}

func assertByteLiteral(t *testing.T, expression tsgo.Expression, expected []byte) {
	t.Helper()
	if !byteLiteralMatches(expression, expected) {
		t.Fatalf("string literal = %T, want code units %x", expression, expected)
	}
}

func byteLiteralMatches(expression tsgo.Expression, expected []byte) bool {
	literal, ok := expression.(tsgo.StringLiteral)
	if !ok {
		return false
	}
	codeUnits := []rune(literal.Text())
	if len(codeUnits) != len(expected) {
		return false
	}
	for index, value := range expected {
		if codeUnits[index] != rune(value) {
			return false
		}
	}
	return true
}

func assertRuntimeCall(
	t *testing.T,
	expression tsgo.Expression,
	name string,
	arguments int,
) {
	t.Helper()
	if !runtimeCallMatches(expression, name, arguments) {
		t.Fatalf("expression = %T, want %s call with %d arguments", expression, name, arguments)
	}
}

func runtimeCallMatches(expression tsgo.Expression, name string, arguments int) bool {
	call, ok := expression.(tsgo.CallExpression)
	if !ok || len(call.Arguments()) != arguments {
		return false
	}
	callee, ok := call.Expression().(tsgo.Identifier)
	return ok && callee.Text() == name
}

func assertBoundsFunction(t *testing.T, function tsgo.FunctionDeclaration) {
	t.Helper()
	if !hasPanicBoundsCheck(function) {
		t.Fatalf(
			"runtime function %s lacks a shared-panic bounds check",
			function.Name().Text(),
		)
	}
}

func hasPanicBoundsCheck(function tsgo.FunctionDeclaration) bool {
	body, ok := function.Body().(tsgo.Block)
	if !ok {
		return false
	}
	for _, statement := range body.Statements() {
		check, ok := statement.(tsgo.IfStatement)
		if !ok {
			continue
		}
		target, ok := check.ThenStatement().(tsgo.Block)
		if !ok {
			continue
		}
		for _, nested := range target.Statements() {
			expression, ok := nested.(tsgo.ExpressionStatement)
			if !ok {
				continue
			}
			call, ok := expression.Expression().(tsgo.CallExpression)
			if !ok {
				continue
			}
			member, ok := call.Expression().(tsgo.PropertyAccessExpression)
			if ok &&
				member.Name().(tsgo.Identifier).Text() == "raiseRuntime" {
				return true
			}
		}
	}
	return false
}

func materializeStringProgram(
	t *testing.T,
	workingDirectory string,
	emission emit.ProgramEmission,
) ([]string, string, string) {
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
	var targetPaths []string
	var module string
	var printed strings.Builder
	for _, file := range emission.Files() {
		source, err := client.PrintNode(file.SourceFile(), tsgo.PrintOptions{})
		if err != nil {
			t.Fatal(err)
		}
		targetPath := filepath.Join(workingDirectory, filepath.FromSlash(file.OutputPath()))
		writeFile(t, targetPath, source)
		targetPaths = append(targetPaths, targetPath)
		printed.WriteString(source)
		if file.Kind() == emit.TargetFileSource {
			module = "./" + strings.TrimSuffix(file.OutputPath(), ".ts") + ".js"
		}
	}
	if module == "" {
		t.Fatal("string fixture emitted no source module")
	}
	return targetPaths, module, printed.String()
}

func compileTypeScript(t *testing.T, directory string, targetPaths []string) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	arguments := []string{
		"--target", "es2022",
		"--module", "nodenext",
		"--moduleResolution", "nodenext",
		"--strict",
		"--outDir", filepath.Join(directory, "out"),
	}
	arguments = append(arguments, targetPaths...)
	if err := runtimefixture.InstallResolution(directory, filepath.Join(directory, "out")); err != nil {
		t.Fatal(err)
	}
	if err := tsgo.Compile(ctx, repositoryRoot(), directory, arguments); err != nil {
		t.Fatal(err)
	}
}

func executeGo(t *testing.T, workingDirectory string) string {
	t.Helper()
	fixture, err := filepath.Abs(stringFixtureDirectory())
	if err != nil {
		t.Fatal(err)
	}
	runner := filepath.Join(workingDirectory, "go-runner")
	writeFile(t, filepath.Join(runner, "go.mod"), fmt.Sprintf(`module example.com/runner

go 1.26.4

require example.com/stringvalues v0.0.0

replace example.com/stringvalues => %s
`, filepath.ToSlash(fixture)))
	writeFile(t, filepath.Join(runner, "main.go"), `package main

import (
	"fmt"

	values "example.com/stringvalues"
)

func panics(operation func()) (result bool) {
	defer func() {
		result = recover() != nil
	}()
	operation()
	return false
}

func main() {
	fmt.Printf("%x\n", values.ASCII())
	fmt.Printf("%x\n", values.UTF8())
	fmt.Printf("%x\n", values.RawUTF8())
	fmt.Printf("%x\n", values.InvalidBytes())
	fmt.Printf("%x\n", values.Constants())
	fmt.Printf("%x\n", values.Zero())
	fmt.Printf("%x\n", values.Assign("\xff\x00A"))
	fmt.Printf("%x\n", values.PackageZero())
	fmt.Printf("%x\n", values.PackageAssign("\xfe"))
	fmt.Printf("%x\n", values.Concat("\xc3", "\xa9"))
	fmt.Println(values.Equal("a", "a"))
	fmt.Println(values.NotEqual("a", "b"))
	fmt.Println(values.Less("a", "b"))
	fmt.Println(values.LessEqual("a", "a"))
	fmt.Println(values.Greater("b", "a"))
	fmt.Println(values.GreaterEqual("b", "b"))
	fmt.Println(values.Less("\x80", "\xff"))
	fmt.Println(values.Less("é", "\xff"))
	fmt.Println(values.Length(values.UTF8()))
	fmt.Println(values.ByteAt(values.UTF8(), 1))
	fmt.Println(values.ConstantByteAt(10))
	fmt.Printf("%x\n", values.Window(values.UTF8(), 0, 1))
	fmt.Printf("%x\n", values.Prefix(values.UTF8(), 1))
	fmt.Printf("%x\n", values.Suffix(values.UTF8(), 1))
	suffixCallValue, suffixCallCount := values.SuffixCall(1)
	fmt.Printf("%x\n", suffixCallValue)
	fmt.Println(suffixCallCount)
	indexCallValue, indexCallCount := values.IndexCall(1)
	fmt.Println(indexCallValue)
	fmt.Println(indexCallCount)
	definedByte, definedWindow := values.DefinedBounds()
	fmt.Println(definedByte)
	fmt.Printf("%x\n", definedWindow)
	fmt.Printf("%x\n", values.DefinedWindow("abcd", 1, 3))
	fmt.Println(panics(func() { values.ByteAt("a", -1) }))
	fmt.Println(panics(func() { values.ByteAt("a", 1) }))
	fmt.Println(panics(func() { values.ByteAt("a", 9007199254740992) }))
	fmt.Println(panics(func() { values.Window("a", -1, 0) }))
	fmt.Println(panics(func() { values.Window("a", 1, 0) }))
	fmt.Println(panics(func() { values.Window("a", 0, 2) }))
	fmt.Println(panics(func() { values.Suffix("a", -1) }))
	fmt.Println(panics(func() { values.Prefix("a", 2) }))
}
`)
	return run(t, runner, filepath.Join(runtime.GOROOT(), "bin", "go"), "run", ".")
}

func run(t *testing.T, directory string, name string, arguments ...string) string {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
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

func writeFile(t *testing.T, path string, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func stringFixtureDirectory() string {
	return filepath.Join(repositoryRoot(), "testdata", "constructs", "value", "string")
}

func repositoryRoot() string {
	return filepath.Join("..", "..", "..")
}
