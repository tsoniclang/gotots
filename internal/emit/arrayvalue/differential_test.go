package arrayvalue_test

import (
	"errors"
	"fmt"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/tsoniclang/gotots/internal/emit"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func TestArrayZeroCopyEqualityLiteralAccessAndBoundsMatchGo(t *testing.T) {
	emission := compileArrayFixture(t)
	directory := t.TempDir()
	target := materializeArrayProgram(t, directory, emission)
	sourceArtifact := target.printed[sourceOutputPath(target)]
	if !strings.Contains(
		sourceArtifact,
		"GoArray as GoArray__from_gotots_runtime",
	) || !strings.Contains(sourceArtifact, "function GoArray") {
		t.Fatalf("RuntimeArray import collision artifact:\n%s", sourceArtifact)
	}
	for path, artifact := range target.printed {
		for _, forbidden := range []string{
			": any",
			": unknown",
			".call(",
			".apply(",
			".bind(",
			"import(",
		} {
			if strings.Contains(artifact, forbidden) {
				t.Fatalf("%s contains forbidden %q:\n%s", path, forbidden, artifact)
			}
		}
		if path != "runtime/array.ts" &&
			(strings.Contains(artifact, "int32[]") ||
				strings.Contains(artifact, "int64[]") ||
				strings.Contains(artifact, "bool[]")) {
			t.Fatalf("%s uses a native array as a Go array type:\n%s", path, artifact)
		}
	}
	runner := filepath.Join(directory, "runner.ts")
	writeFile(t, runner, `import "`+target.programInit+`";
import { GoPanic } from "./runtime/panic.js";
import {
    ArgumentAndResultCopy,
    BoolValues,
    CopyIsValue,
    EqualValues,
    IndexStore,
    InferredLength,
    KeyedEvaluationOrder,
    LengthAndCapacity,
    LiteralValues,
    NotEqualValues,
    PackageValuesAreIsolated,
    PackageIndexStore,
    ReadEvaluationOrder,
    RuntimeNameCollision,
    StructFieldCopyAndEquality,
    StoreEvaluationOrder,
    ZeroIsFresh,
    ZeroLength,
} from "`+target.sourceModule+`";

console.log(ZeroIsFresh());
console.log(CopyIsValue());
console.log(EqualValues());
console.log(NotEqualValues());
console.log(LiteralValues());
console.log(BoolValues());
console.log(IndexStore(1, 9));
console.log(InferredLength());
console.log(KeyedEvaluationOrder());
console.log(LengthAndCapacity());
console.log(ArgumentAndResultCopy());
console.log(ZeroLength());
console.log(PackageValuesAreIsolated());
console.log(PackageIndexStore());
console.log(ReadEvaluationOrder());
console.log(RuntimeNameCollision());
console.log(StructFieldCopyAndEquality());
console.log(StoreEvaluationOrder());
try {
    IndexStore(3, 9);
    console.log("bounds-missing");
} catch (error) {
    console.log(error instanceof GoPanic ? "bounds" : "wrong-error");
}
try {
    IndexStore(-1, 9);
    console.log("bounds-missing");
} catch (error) {
    console.log(error instanceof GoPanic ? "bounds" : "wrong-error");
}
`)
	writeFile(t, filepath.Join(directory, "package.json"), "{\"type\":\"module\"}\n")
	target.paths = append(target.paths, runner)
	if err := compileTypeScript(t, directory, target.paths); err != nil {
		t.Fatal(err)
	}
	typeScriptOutput := run(
		t,
		directory,
		"node",
		filepath.Join(directory, "out", "runner.js"),
	)
	goOutput := executeGoArrayRunner(t, directory)
	if typeScriptOutput != goOutput {
		t.Fatalf(
			"TypeScript output = %q, Go output = %q",
			typeScriptOutput,
			goOutput,
		)
	}
}

func TestArrayLengthRemainsPartOfTheStrictTargetType(t *testing.T) {
	emission := compileArrayFixture(t)
	directory := t.TempDir()
	target := materializeArrayProgram(t, directory, emission)
	mutation := filepath.Join(directory, "erased-length-mutation.ts")
	writeFile(t, mutation, `import { AcceptTwo, Three } from "`+target.sourceModule+`";

console.log(AcceptTwo(Three()));
`)
	writeFile(t, filepath.Join(directory, "package.json"), "{\"type\":\"module\"}\n")
	target.paths = append(target.paths, mutation)
	err := compileTypeScript(t, directory, target.paths)
	var compilerError *tsgo.CompilerError
	if !errors.As(err, &compilerError) ||
		!strings.Contains(compilerError.Diagnostics, "TS2345") ||
		!strings.Contains(compilerError.Diagnostics, "3") ||
		!strings.Contains(compilerError.Diagnostics, "2") {
		t.Fatalf("strict compiler error = %#v, want length-distinct rejection", err)
	}
}

func TestArrayFamilyStrictTypechecksUnderTheBigIntProfile(t *testing.T) {
	emission := compileArrayFixtureWithOptions(t, emit.Options{
		IntegerRepresentation: emit.IntegerRepresentationBigInt,
		EvaluationOrder:       emit.EvaluationOrderDirect,
	})
	directory := t.TempDir()
	target := materializeArrayProgram(t, directory, emission)
	writeFile(t, filepath.Join(directory, "package.json"), "{\"type\":\"module\"}\n")
	if err := compileTypeScript(t, directory, target.paths); err != nil {
		t.Fatal(err)
	}
	runtime := target.printed["runtime/array.ts"]
	if !strings.Contains(runtime, "number | bigint") ||
		!strings.Contains(runtime, "Number(index)") {
		t.Fatalf("BigInt-compatible runtime array artifact:\n%s", runtime)
	}
	assertBigIntConstantReturn(t, emission, "LengthAndCapacity", "10n")
	assertBigIntConstantReturn(t, emission, "InferredLength", "8n")
}

func sourceOutputPath(target materializedProgram) string {
	for path := range target.printed {
		if "./"+strings.TrimSuffix(path, ".ts")+".js" == target.sourceModule {
			return path
		}
	}
	return ""
}

func assertBigIntConstantReturn(
	t *testing.T,
	emission emit.ProgramEmission,
	name string,
	want string,
) {
	t.Helper()
	matches := 0
	for _, file := range emission.Files() {
		if file.Kind() != emit.TargetFileSource {
			continue
		}
		for _, statement := range file.SourceFile().Statements() {
			function, ok := statement.(tsgo.FunctionDeclaration)
			if !ok || function.Name().Text() != name {
				continue
			}
			matches++
			body, ok := function.Body().(tsgo.Block)
			if !ok {
				t.Fatalf("%s body = %T, want tsgo.Block", name, function.Body())
			}
			var expression tsgo.Expression
			for _, bodyStatement := range body.Statements() {
				returned, ok := bodyStatement.(tsgo.ReturnStatement)
				if ok {
					expression = returned.Expression()
				}
			}
			literal, ok := expression.(tsgo.BigIntLiteral)
			if !ok || literal.Text() != want {
				t.Fatalf(
					"%s return = %T/%v, want bigint literal %s",
					name,
					expression,
					expression,
					want,
				)
			}
		}
	}
	if matches != 1 {
		t.Fatalf("%s declarations = %d, want one", name, matches)
	}
}

func executeGoArrayRunner(t *testing.T, directory string) string {
	t.Helper()
	modulePath, err := filepath.Abs(arrayValuesDirectory())
	if err != nil {
		t.Fatal(err)
	}
	runnerDirectory := filepath.Join(directory, "go-runner")
	writeFile(t, filepath.Join(runnerDirectory, "go.mod"), fmt.Sprintf(`module example.com/runner

go 1.26.4

require example.com/arrayvalues v0.0.0

replace example.com/arrayvalues => %s
`, filepath.ToSlash(modulePath)))
	writeFile(t, filepath.Join(runnerDirectory, "main.go"), `package main

import (
	"fmt"

	values "example.com/arrayvalues"
)

func bounds(index int) {
	defer func() {
		if recover() != nil {
			fmt.Println("bounds")
		}
	}()
	values.IndexStore(index, 9)
	fmt.Println("bounds-missing")
}

func main() {
	fmt.Println(values.ZeroIsFresh())
	fmt.Println(values.CopyIsValue())
	fmt.Println(values.EqualValues())
	fmt.Println(values.NotEqualValues())
	fmt.Println(values.LiteralValues())
	fmt.Println(values.BoolValues())
	fmt.Println(values.IndexStore(1, 9))
	fmt.Println(values.InferredLength())
	fmt.Println(values.KeyedEvaluationOrder())
	fmt.Println(values.LengthAndCapacity())
	fmt.Println(values.ArgumentAndResultCopy())
	fmt.Println(values.ZeroLength())
	fmt.Println(values.PackageValuesAreIsolated())
	fmt.Println(values.PackageIndexStore())
	fmt.Println(values.ReadEvaluationOrder())
	fmt.Println(values.RuntimeNameCollision())
	fmt.Println(values.StructFieldCopyAndEquality())
	fmt.Println(values.StoreEvaluationOrder())
	bounds(3)
	bounds(-1)
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
