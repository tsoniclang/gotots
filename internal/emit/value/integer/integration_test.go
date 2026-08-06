package integer_test

import (
	"context"
	"errors"
	"fmt"
	"go/ast"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/tsoniclang/gotots/internal/emit"
	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/load"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func TestIntegerNumberProfilePrintsTypechecksAndExecutesDifferentially(t *testing.T) {
	loaded := loadIntegerFamily(t)
	names := append(slices.Clone(integerCarrierRoots),
		"NumberBits8",
		"NumberBits16",
		"NumberBits32",
		"NumberShifts",
		"NumberUnary",
		"NumberUnaryUint",
		"NumberWideBits",
		"NumberWideShifts",
		"NumberWideUnary",
		"NumberUnsignedShift",
		"NumberVariableShift",
		"NumberVariableSignedShift",
		"NumberVariableUnsignedShift",
		"DefinedShift",
		"WidenSigned",
		"WidenUnsigned",
	)
	emission := compileIntegerFamily(t, loaded, emit.DefaultOptions(), names...)
	assertIntegerAliases(t, emission, false)
	printed := printIntegerFamily(t, emission)
	assertDirectIntegerArtifact(t, printed, false)

	workingDirectory := t.TempDir()
	goOutput := executeIntegerFamilyGo(t, workingDirectory, false)
	targetOutput := executeIntegerFamilyTS(t, emission, workingDirectory, false)
	if targetOutput != goOutput {
		t.Fatalf("number TypeScript output = %q, Go output = %q", targetOutput, goOutput)
	}
}

func TestIntegerBigIntProfilePrintsTypechecksAndExecutesDifferentially(t *testing.T) {
	loaded := loadIntegerFamily(t)
	names := append(slices.Clone(integerCarrierRoots),
		"BigShifts",
		"BigSigned",
		"BigUnary",
		"BigUnsigned",
		"BigVariableShift",
		"BigVariableSignedShift",
		"BigVariableUnsignedShift",
		"DefinedShift",
		"WidenSigned",
		"WidenUnsigned",
	)
	options := emit.DefaultOptions()
	options.IntegerRepresentation = emit.IntegerRepresentationBigInt
	emission := compileIntegerFamily(t, loaded, options, names...)
	assertIntegerAliases(t, emission, true)
	assertBigIntDivisionUsesRuntime(t, emission)
	printed := printIntegerFamily(t, emission)
	assertDirectIntegerArtifact(t, printed, true)

	workingDirectory := t.TempDir()
	goOutput := executeIntegerFamilyGo(t, workingDirectory, true)
	targetOutput := executeIntegerFamilyTS(t, emission, workingDirectory, true)
	if targetOutput != goOutput {
		t.Fatalf("BigInt TypeScript output = %q, Go output = %q", targetOutput, goOutput)
	}
}

func TestIntegerNumberProfileEmitsWideConstantsWithoutWrapping(t *testing.T) {
	loaded, err := load.One(context.Background(), load.Request{
		Directory: integerBoundaryDirectory(),
		Pattern:   ".",
	})
	if err != nil {
		t.Fatal(err)
	}
	emission := compileIntegerFamily(
		t,
		loaded,
		emit.DefaultOptions(),
		"WideNumber",
		"WideConversion",
	)
	printed := printIntegerFamily(t, emission)
	if strings.Count(printed, "9007199254740992") != 2 {
		t.Fatalf("wide number constants were not emitted directly:\n%s", printed)
	}
	for _, forbidden := range []string{
		"9007199254740992n",
		"BigInt",
		"Math.imul",
	} {
		if strings.Contains(printed, forbidden) {
			t.Fatalf("wide number constants contain %q:\n%s", forbidden, printed)
		}
	}
	workingDirectory := t.TempDir()
	artifacts := materializeIntegerFamily(t, emission, workingDirectory)
	runnerPath := filepath.Join(workingDirectory, "runner.ts")
	writeFile(t, runnerPath, `import * as values from "`+
		artifacts.module(t, "source.ts")+`";

console.log(values.WideNumber(), values.WideConversion());
`)
	output := executeMaterializedTypeScript(
		t,
		workingDirectory,
		artifacts,
		runnerPath,
	)
	if output != "9007199254740992 9007199254740992\n" {
		t.Fatalf("wide number output = %q", output)
	}
}

func TestUint32OperationsCarryRequiredTypedASTNormalization(t *testing.T) {
	loaded := loadIntegerFamily(t)
	emission := compileIntegerFamily(
		t,
		loaded,
		emit.DefaultOptions(),
		"NumberBits32",
		"NumberUnsignedShift",
		"NumberUnaryUint",
	)
	source := integerFamilySourceFile(t, emission)
	for _, name := range []string{"NumberBits32", "NumberUnsignedShift"} {
		function := targetFunction(t, source, name)
		returnStatement := function.Body().(tsgo.Block).Statements()[0].(tsgo.ReturnStatement)
		values := returnStatement.Expression().(tsgo.ArrayLiteralExpression).Elements()
		for index, value := range values {
			binary, ok := value.(tsgo.BinaryExpression)
			if !ok ||
				binary.OperatorToken().Kind() !=
					tsgo.SyntaxKindGreaterThanGreaterThanGreaterThanToken {
				t.Fatalf("%s result %d = %T/%d, want unsigned normalization", name, index, value, value.Kind())
			}
		}
	}
	unary := targetFunction(t, source, "NumberUnaryUint")
	result := unary.Body().(tsgo.Block).Statements()[0].(tsgo.ReturnStatement).Expression().(tsgo.BinaryExpression)
	if result.OperatorToken().Kind() != tsgo.SyntaxKindGreaterThanGreaterThanGreaterThanToken {
		t.Fatalf("uint32 complement operator = %d, want unsigned normalization", result.OperatorToken().Kind())
	}
}

func TestIntegerConstantSyntaxMutationFailsAtLiteralOwner(t *testing.T) {
	loaded := loadIntegerFamily(t)
	function := loaded.Files()[0].Syntax().Decls[0].(*ast.FuncDecl)
	result := function.Body.List[2].(*ast.IfStmt).
		Body.List[0].(*ast.ReturnStmt).Results[0].(*ast.BasicLit)
	result.Kind = 0
	object := loaded.TypesInfo().Defs[function.Name]
	root, err := emit.NewRoot(object)
	if err != nil {
		t.Fatal(err)
	}
	_, err = emit.Compile(loaded.Program(), []emit.Root{root})
	var unsupported *api.UnsupportedError
	if !errors.As(err, &unsupported) ||
		unsupported.Role != api.RoleReturnResult ||
		unsupported.Construct != "*ast.BasicLit" {
		t.Fatalf("mutation error = %#v, want return literal UnsupportedError", err)
	}
}

func TestIntegerSemanticEvidenceMutationsFailAtExpressionOwners(t *testing.T) {
	t.Run("conversion result", func(t *testing.T) {
		loaded := loadIntegerFamily(t)
		function := sourceFunction(t, loaded.Files()[0].Syntax(), "WidenSigned")
		call := function.Body.List[0].(*ast.ReturnStmt).Results[0].(*ast.CallExpr)
		delete(loaded.TypesInfo().Types, call)
		assertIntegerExpressionMutationFails(t, loaded, function, "*ast.CallExpr")
	})
	t.Run("shift count evidence", func(t *testing.T) {
		loaded := loadIntegerFamily(t)
		function := sourceFunction(t, loaded.Files()[0].Syntax(), "NumberShifts")
		shift := function.Body.List[0].(*ast.ReturnStmt).Results[0].(*ast.BinaryExpr)
		delete(loaded.TypesInfo().Types, shift.Y)
		assertIntegerExpressionMutationFails(t, loaded, function, "*ast.BinaryExpr")
	})
}

func assertIntegerExpressionMutationFails(
	t *testing.T,
	loaded *load.Package,
	function *ast.FuncDecl,
	construct string,
) {
	t.Helper()
	root, err := emit.NewRoot(loaded.TypesInfo().Defs[function.Name])
	if err != nil {
		t.Fatal(err)
	}
	_, err = emit.Compile(loaded.Program(), []emit.Root{root})
	var unsupported *api.UnsupportedError
	if !errors.As(err, &unsupported) ||
		unsupported.Role != api.RoleReturnResult ||
		unsupported.Category != api.CategoryExpression ||
		unsupported.Construct != construct {
		t.Fatalf("mutation error = %#v, want return %s UnsupportedError", err, construct)
	}
}

func compileIntegerFamily(
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
			t.Fatalf("integer root %q is absent", name)
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
	for _, file := range emission.Files() {
		if file.Kind() != emit.TargetFileSupport {
			continue
		}
		for _, statement := range file.SourceFile().Statements() {
			alias, ok := statement.(tsgo.TypeAliasDeclaration)
			if !ok || alias.Name().Text() == "bool" {
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

func assertDirectIntegerArtifact(t *testing.T, printed string, bigint bool) {
	t.Helper()
	for _, forbidden := range []string{
		"Math.imul",
		" as ",
		"any",
		"unknown",
		".call(",
		".apply(",
		".bind(",
	} {
		if strings.Contains(printed, forbidden) {
			t.Fatalf("integer artifact contains %q:\n%s", forbidden, printed)
		}
	}
	if bigint && (!strings.Contains(printed, "1n") ||
		!strings.Contains(printed, "export type int32 = number;") ||
		!strings.Contains(printed, "export type int64 = bigint;") ||
		strconv.IntSize == 64 &&
			!strings.Contains(printed, "export type int = bigint;")) {
		t.Fatalf("exact-width artifact lacks its number/BigInt carrier split:\n%s", printed)
	}
	if !bigint && regexp.MustCompile(`[0-9]n(?:\W|$)`).MatchString(printed) {
		t.Fatalf("number artifact contains BigInt syntax:\n%s", printed)
	}
}

func printIntegerFamily(t *testing.T, emission emit.ProgramEmission) string {
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

func integerFamilySourceFile(
	t *testing.T,
	emission emit.ProgramEmission,
) tsgo.SourceFile {
	t.Helper()
	for _, file := range emission.Files() {
		if file.Kind() == emit.TargetFileSource &&
			file.PackageName() == "integerfamily" {
			return file.SourceFile()
		}
	}
	t.Fatal("integer family source artifact is absent")
	return nil
}

func executeIntegerFamilyTS(
	t *testing.T,
	emission emit.ProgramEmission,
	workingDirectory string,
	bigint bool,
) string {
	t.Helper()
	artifacts := materializeIntegerFamily(t, emission, workingDirectory)
	suffix := ""
	if bigint {
		suffix = "n"
	}
	runner := `import * as values from "` + artifacts.module(t, "source.ts") + `";

const show = (value: number | bigint): string => value.toString();
const row = (value: readonly (number | bigint)[]): string => value.map(show).join(" ");
const panics = (operation: () => void): boolean => {
    try {
        operation();
        return false;
    } catch {
        return true;
    }
};
`
	if bigint {
		runner += `
console.log(row(values.BigSigned(9007199254740993n, 7n)));
console.log(row(values.BigUnsigned(18446744073709551600n, 15n)));
console.log(row(values.BigShifts(-9007199254740993n)));
console.log(row(values.BigVariableShift(-9n, 3)));
console.log(row(values.BigVariableShift(-9n, 64)));
console.log(String(panics(() => { values.BigVariableSignedShift(1n, -1); })));
console.log(row(values.BigVariableUnsignedShift(15n, 80)));
console.log(row(values.DefinedShift(-9n, new values.DefinedShiftCount(3))));
console.log(row(values.BigUnary(9007199254740993n)));
console.log(show(values.WidenSigned(-8)));
console.log(show(values.WidenUnsigned(4000000000)));
console.log(values.CompareSigned(-8, 4).join(" "));
console.log(values.CompareUnsigned(8n, 4n).join(" "));
`
	} else {
		runner += `
console.log(row(values.NumberBits8(-7, 3)));
console.log(row(values.NumberBits16(60000, 3855)));
console.log(row(values.NumberBits32(4042322160, 252645135)));
console.log(row(values.NumberShifts(-128)));
console.log(row(values.NumberUnsignedShift(4042322160)));
console.log(row(values.NumberVariableShift(-9, 3)));
console.log(row(values.NumberVariableShift(-9, 32)));
console.log(String(panics(() => { values.NumberVariableSignedShift(1, -1); })));
console.log(row(values.NumberVariableUnsignedShift(15, 40)));
console.log(row(values.DefinedShift(-9, new values.DefinedShiftCount(3))));
console.log(row(values.NumberUnary(-123456)));
console.log(show(values.NumberUnaryUint(4042322160)));
console.log(row(values.NumberWideBits(123456, 255)));
console.log(row(values.NumberWideShifts(-123456)));
console.log(show(values.NumberWideUnary(123456)));
console.log(show(values.WidenSigned(-8)));
console.log(show(values.WidenUnsigned(4000000000)));
console.log(values.CompareSigned(-8, 4).join(" "));
console.log(values.CompareUnsigned(8, 4).join(" "));
`
	}
	runner += `
console.log(show(values.ConstantConversion()));
console.log(show(values.UnsignedComplement8(240)));
console.log(String(values.UntypedBooleanNot(3, 4)));
console.log(show(values.Int8(-5, 3)));
console.log(show(values.Uint64(40` + suffix + `, 2` + suffix + `)));
`
	runnerPath := filepath.Join(workingDirectory, "runner.ts")
	writeFile(t, runnerPath, runner)
	return executeMaterializedTypeScript(t, workingDirectory, artifacts, runnerPath)
}

func materializeIntegerFamily(
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

func executeIntegerFamilyGo(t *testing.T, workingDirectory string, bigint bool) string {
	t.Helper()
	modulePath, err := filepath.Abs(integerFamilyDirectory())
	if err != nil {
		t.Fatal(err)
	}
	runnerDirectory := filepath.Join(workingDirectory, "go-runner")
	if err := os.MkdirAll(runnerDirectory, 0o755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(runnerDirectory, "go.mod"), fmt.Sprintf(`module example.com/runner

go 1.26.4

require example.com/integerfamily v0.0.0

replace example.com/integerfamily => %s
`, filepath.ToSlash(modulePath)))
	body := `
	fmt.Println(values.NumberBits8(-7, 3))
	fmt.Println(values.NumberBits16(60000, 3855))
	fmt.Println(values.NumberBits32(4042322160, 252645135))
	fmt.Println(values.NumberShifts(-128))
	fmt.Println(values.NumberUnsignedShift(4042322160))
	fmt.Println(values.NumberVariableShift(-9, 3))
	fmt.Println(values.NumberVariableShift(-9, 32))
	fmt.Println(panics(func() { values.NumberVariableSignedShift(1, -1) }))
	fmt.Println(values.NumberVariableUnsignedShift(15, 40))
	fmt.Println(values.DefinedShift(-9, 3))
	fmt.Println(values.NumberUnary(-123456))
	fmt.Println(values.NumberUnaryUint(4042322160))
	fmt.Println(values.NumberWideBits(123456, 255))
	fmt.Println(values.NumberWideShifts(-123456))
	fmt.Println(values.NumberWideUnary(123456))
	fmt.Println(values.WidenSigned(-8))
	fmt.Println(values.WidenUnsigned(4000000000))
	fmt.Println(values.CompareSigned(-8, 4))
	fmt.Println(values.CompareUnsigned(8, 4))
`
	if bigint {
		body = `
	fmt.Println(values.BigSigned(9007199254740993, 7))
	fmt.Println(values.BigUnsigned(18446744073709551600, 15))
	fmt.Println(values.BigShifts(-9007199254740993))
	fmt.Println(values.BigVariableShift(-9, 3))
	fmt.Println(values.BigVariableShift(-9, 64))
	fmt.Println(panics(func() { values.BigVariableSignedShift(1, -1) }))
	fmt.Println(values.BigVariableUnsignedShift(15, 80))
	fmt.Println(values.DefinedShift(-9, 3))
	fmt.Println(values.BigUnary(9007199254740993))
	fmt.Println(values.WidenSigned(-8))
	fmt.Println(values.WidenUnsigned(4000000000))
	fmt.Println(values.CompareSigned(-8, 4))
	fmt.Println(values.CompareUnsigned(8, 4))
`
	}
	writeFile(t, filepath.Join(runnerDirectory, "main.go"), `package main

import (
	"fmt"
	values "example.com/integerfamily"
)

func main() {
`+body+`
	fmt.Println(values.ConstantConversion())
	fmt.Println(values.UnsignedComplement8(240))
	fmt.Println(values.UntypedBooleanNot(3, 4))
	fmt.Println(values.Int8(-5, 3))
	fmt.Println(values.Uint64(40, 2))
}

func panics(operation func()) (result bool) {
	defer func() {
		result = recover() != nil
	}()
	operation()
	return false
}
`)
	return run(t, runnerDirectory, filepath.Join(runtime.GOROOT(), "bin", "go"), "run", ".")
}

func loadIntegerFamily(t *testing.T) *load.Package {
	t.Helper()
	loaded, err := load.One(context.Background(), load.Request{
		Directory: integerFamilyDirectory(),
		Pattern:   ".",
	})
	if err != nil {
		t.Fatal(err)
	}
	return loaded
}

func integerFamilyDirectory() string {
	return filepath.Join(repositoryRoot(), "testdata", "constructs", "value", "integer-family")
}
