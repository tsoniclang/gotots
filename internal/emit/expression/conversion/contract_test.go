package conversion_test

import (
	"context"
	"errors"
	"go/ast"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tsoniclang/gotots/internal/emit"
	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/load"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func TestConversionASTUsesDirectAndBoundaryShapes(t *testing.T) {
	number := compileConversions(t, emit.DefaultOptions())
	bigintOptions := emit.Options{
		IntegerRepresentation: emit.IntegerRepresentationBigInt,
		EvaluationOrder:       emit.EvaluationOrderDirect,
	}
	bigint := compileConversions(t, bigintOptions)

	if _, ok := returnExpression(t, number, "Widen").(tsgo.Identifier); !ok {
		t.Fatalf(
			"lossless number widening = %T, want direct identifier",
			returnExpression(t, number, "Widen"),
		)
	}
	if _, ok := returnExpression(t, bigint, "Widen").(tsgo.CallExpression); !ok {
		t.Fatalf(
			"number-to-BigInt widening = %T, want static conversion call",
			returnExpression(t, bigint, "Widen"),
		)
	}
	if _, ok := returnExpression(
		t,
		number,
		"NarrowSigned",
	).(tsgo.BinaryExpression); !ok {
		t.Fatalf(
			"number narrowing = %T, want width-normalizing binary AST",
			returnExpression(t, number, "NarrowSigned"),
		)
	}
	assertBigIntToNumberNarrowingOrder(
		t,
		returnExpression(t, bigint, "NarrowSigned"),
	)
	assertOneConversionRuntimeDefinition(t, number)
	assertOneConversionRuntimeDefinition(t, bigint)

	workingDirectory := t.TempDir()
	_, _, numberText := printConversions(t, workingDirectory, number)
	for _, required := range []string{
		"BigInt.asUintN(64",
		"goNumberToBigInt",
		"goFloat32",
		"__gotots_conversion_",
		"GoComplex128.make",
		"GoComplex64.make",
	} {
		if !strings.Contains(numberText, required) {
			t.Fatalf("number conversion artifact lacks %q:\n%s", required, numberText)
		}
	}
	if strings.Count(numberText, "function goNumberToBigInt") != 1 {
		t.Fatalf("number-to-BigInt definition count is not one:\n%s", numberText)
	}
}

func assertBigIntToNumberNarrowingOrder(
	t *testing.T,
	expression tsgo.Expression,
) {
	t.Helper()
	outer, ok := expression.(tsgo.CallExpression)
	if !ok || len(outer.Arguments()) != 1 {
		t.Fatalf("BigInt narrowing = %T, want one-argument Number call", expression)
	}
	number, ok := outer.Expression().(tsgo.PropertyAccessExpression)
	if !ok {
		t.Fatal("BigInt narrowing Number callee is not a property access")
	}
	numberReceiver, receiverOK := number.Expression().(tsgo.Identifier)
	numberName, nameOK := number.Name().(tsgo.Identifier)
	if !receiverOK || !nameOK ||
		numberReceiver.Text() != api.TargetGlobalAnchorName ||
		numberName.Text() != api.TargetIntrinsicNumber.String() {
		t.Fatal("BigInt narrowing does not cross carriers through globalThis.Number")
	}
	normalize, ok := outer.Arguments()[0].(tsgo.CallExpression)
	if !ok || len(normalize.Arguments()) != 2 {
		t.Fatal("BigInt narrowing crosses carriers before exact width normalization")
	}
	member, ok := normalize.Expression().(tsgo.PropertyAccessExpression)
	if !ok {
		t.Fatal("BigInt width normalizer is not a property access")
	}
	memberReceiver, receiverOK := member.Expression().(tsgo.Identifier)
	memberName, nameOK := member.Name().(tsgo.Identifier)
	if !receiverOK || !nameOK ||
		memberReceiver.Text() != "BigInt" ||
		memberName.Text() != "asIntN" {
		t.Fatal("BigInt narrowing does not normalize exact low bits before Number conversion")
	}
}

func TestConversionOperandAndResultEvidenceMutationsFailClosed(t *testing.T) {
	t.Run("callee type evidence", func(t *testing.T) {
		loaded, function, call := loadConversionCall(t, "NarrowSigned")
		delete(loaded.TypesInfo().Types, call.Fun)
		assertConversionCallUnsupported(t, loaded, function)
	})
	t.Run("result evidence", func(t *testing.T) {
		loaded, function, call := loadConversionCall(t, "NarrowSigned")
		delete(loaded.TypesInfo().Types, call)
		assertConversionCallUnsupported(t, loaded, function)
	})
	t.Run("operand evidence", func(t *testing.T) {
		loaded, function, call := loadConversionCall(t, "NarrowSigned")
		delete(loaded.TypesInfo().Types, call.Args[0])
		assertConversionCallUnsupported(t, loaded, function)
	})
}

func TestConversionTargetSpellingDoesNotSelectSemantics(t *testing.T) {
	loaded, function, call := loadConversionCall(t, "NarrowSigned")
	before := compileSingleConversion(t, loaded, function.Name.Name)
	target, ok := call.Fun.(*ast.Ident)
	if !ok {
		t.Fatalf("conversion target = %T, want identifier", call.Fun)
	}
	target.Name = "spelling_must_not_be_read"
	after := compileSingleConversion(t, loaded, function.Name.Name)
	if before != after {
		t.Fatalf(
			"target spelling changed checker-owned output\nbefore:\n%s\nafter:\n%s",
			before,
			after,
		)
	}
}

func TestRawUnsafePointerConversionsFailAtTheTypedBoundary(t *testing.T) {
	for _, testCase := range []struct {
		name     string
		source   string
		category api.Category
	}{
		{
			name: "typed pointer round trip",
			source: `package boundary

import "unsafe"

func Convert(value *int32) *int32 {
	return (*int32)(unsafe.Pointer(value))
}
`,
			category: api.CategoryExpression,
		},
		{
			name: "pointer to integer",
			source: `package boundary

import "unsafe"

func Convert(value *int32) uintptr {
	return uintptr(unsafe.Pointer(value))
}
`,
			category: api.CategoryExpression,
		},
		{
			name: "raw pointer signature",
			source: `package boundary

import "unsafe"

func Convert(value uintptr) unsafe.Pointer {
	return unsafe.Pointer(value)
}
`,
			category: api.CategoryType,
		},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			directory := t.TempDir()
			if err := os.WriteFile(
				filepath.Join(directory, "go.mod"),
				[]byte("module example.com/boundary\n\ngo 1.26.4\n"),
				0o600,
			); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(
				filepath.Join(directory, "source.go"),
				[]byte(testCase.source),
				0o600,
			); err != nil {
				t.Fatal(err)
			}
			loaded, err := load.One(context.Background(), load.Request{
				Directory: directory,
				Pattern:   ".",
			})
			if err != nil {
				t.Fatal(err)
			}
			root, err := emit.NewRoot(loaded.Types().Scope().Lookup("Convert"))
			if err != nil {
				t.Fatal(err)
			}
			_, err = emit.Compile(loaded.Program(), []emit.Root{root})
			var unsupported *api.UnsupportedError
			if !errors.As(err, &unsupported) ||
				unsupported.Category != testCase.category {
				t.Fatalf("raw-pointer boundary error = %#v", err)
			}
		})
	}
}

func TestNonFiniteFloatToIntegerUsesSelectedNonPanickingResult(t *testing.T) {
	for _, testCase := range []struct {
		name    string
		options emit.Options
	}{
		{"number", emit.DefaultOptions()},
		{
			"bigint",
			emit.Options{
				IntegerRepresentation: emit.IntegerRepresentationBigInt,
				EvaluationOrder:       emit.EvaluationOrderDirect,
			},
		},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			emission := compileConversions(t, testCase.options)
			workingDirectory := t.TempDir()
			paths, sourceModule, _ := printConversions(
				t,
				workingDirectory,
				emission,
			)
			output := executeConversionTypeScript(
				t,
				workingDirectory,
				paths,
				`import * as values from "`+sourceModule+`";
console.log(String(values.FloatToInt8(Infinity)));
console.log(String(values.FloatToUint32(NaN)));
console.log(String(values.FloatToInt64(-Infinity)));
console.log(String(values.FloatToUint64(NaN)));
`,
			)
			if output != "0\n0\n0\n0\n" {
				t.Fatalf("non-finite conversion output = %q", output)
			}
		})
	}
}

func returnExpression(
	t *testing.T,
	emission emit.ProgramEmission,
	name string,
) tsgo.Expression {
	t.Helper()
	for _, file := range emission.Files() {
		if file.Kind() != emit.TargetFileSource {
			continue
		}
		for _, statement := range file.SourceFile().Statements() {
			function, ok := statement.(tsgo.FunctionDeclaration)
			if !ok || function.Name().Text() != name {
				continue
			}
			body := function.Body().(tsgo.Block)
			for _, statement := range body.Statements() {
				if result, ok := statement.(tsgo.ReturnStatement); ok {
					return result.Expression()
				}
			}
		}
	}
	t.Fatalf("target function %s has no return", name)
	return nil
}

func assertOneConversionRuntimeDefinition(
	t *testing.T,
	emission emit.ProgramEmission,
) {
	t.Helper()
	for _, file := range emission.Files() {
		if file.OutputPath() == "runtime/conversion.ts" {
			if len(file.SourceFile().Statements()) != 1 {
				t.Fatalf(
					"conversion runtime definitions = %d, want 1",
					len(file.SourceFile().Statements()),
				)
			}
			return
		}
	}
	t.Fatal("conversion runtime artifact is absent")
}

func loadConversionCall(
	t *testing.T,
	name string,
) (*load.Package, *ast.FuncDecl, *ast.CallExpr) {
	t.Helper()
	loaded, err := load.One(context.Background(), load.Request{
		Directory: conversionFixtureDirectory(),
		Pattern:   ".",
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, declaration := range loaded.Files()[0].Syntax().Decls {
		function, ok := declaration.(*ast.FuncDecl)
		if !ok || function.Name.Name != name {
			continue
		}
		result := function.Body.List[0].(*ast.ReturnStmt)
		call := result.Results[0].(*ast.CallExpr)
		return loaded, function, call
	}
	t.Fatalf("source function %s is absent", name)
	return nil, nil, nil
}

func assertConversionCallUnsupported(
	t *testing.T,
	loaded *load.Package,
	function *ast.FuncDecl,
) {
	t.Helper()
	root, err := emit.NewRoot(loaded.TypesInfo().Defs[function.Name])
	if err != nil {
		t.Fatal(err)
	}
	_, err = emit.Compile(loaded.Program(), []emit.Root{root})
	var unsupported *api.UnsupportedError
	if !errors.As(err, &unsupported) ||
		unsupported.Construct != "*ast.CallExpr" ||
		unsupported.Category != api.CategoryExpression {
		t.Fatalf("mutation error = %#v, want call-expression unsupported", err)
	}
}

func compileSingleConversion(
	t *testing.T,
	loaded *load.Package,
	name string,
) string {
	t.Helper()
	object := loaded.Types().Scope().Lookup(name)
	root, err := emit.NewRoot(object)
	if err != nil {
		t.Fatal(err)
	}
	emission, err := emit.Compile(loaded.Program(), []emit.Root{root})
	if err != nil {
		t.Fatal(err)
	}
	workingDirectory := t.TempDir()
	_, _, printed := printConversions(t, workingDirectory, emission)
	return printed
}
