package ordered_test

import (
	"context"
	"errors"
	"go/ast"
	"go/types"
	"strings"
	"testing"

	"github.com/tsoniclang/gotots/internal/emit"
	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/load"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func TestOrderedBuiltinASTAndDemandDefinitionsAreExact(t *testing.T) {
	number := compileOrdered(t, emit.DefaultOptions())
	bigint := compileOrdered(t, emit.Options{
		IntegerRepresentation: emit.IntegerRepresentationBigInt,
		EvaluationOrder:       emit.EvaluationOrderPreserveGo,
	})
	numberMax := returnExpression(t, number, "MaxInt32")
	numberCall, ok := numberMax.(tsgo.CallExpression)
	if !ok {
		t.Fatalf("number max = %T, want call", numberMax)
	}
	numberMember := numberCall.Expression().(tsgo.PropertyAccessExpression)
	if numberMember.Expression().(tsgo.Identifier).Text() != "Math" ||
		numberMember.Name().(tsgo.Identifier).Text() != "max" {
		t.Fatal("number max does not call Math.max")
	}
	if _, ok := returnExpression(
		t,
		bigint,
		"MaxInt32",
	).(tsgo.CallExpression); !ok {
		t.Fatal("BigInt max is not a typed pair fold")
	}
	if _, ok := returnExpression(t, bigint, "One").(tsgo.Identifier); !ok {
		t.Fatalf(
			"one-argument max = %T, want direct identifier",
			returnExpression(t, bigint, "One"),
		)
	}
	if _, ok := returnExpression(
		t,
		number,
		"ConstantInteger",
	).(tsgo.NumericLiteral); !ok {
		t.Fatal("constant max was not checker-folded")
	}
	if _, ok := returnExpression(
		t,
		number,
		"ConstantString",
	).(tsgo.StringLiteral); !ok {
		t.Fatal("constant string max was not checker-folded")
	}
	assertRuntimeDefinitionCount(t, number, "runtime/string.ts", 2)
	assertNoRuntimeFile(t, number, "runtime/integer.ts")
	assertRuntimeDefinitionCount(t, bigint, "runtime/string.ts", 2)
	assertRuntimeDefinitionCount(t, bigint, "runtime/integer.ts", 2)

	workingDirectory := t.TempDir()
	_, _, printed := printOrdered(t, workingDirectory, bigint)
	if len(printed) > 8_000 {
		t.Fatalf("ordered artifact = %d bytes, want <= 8000", len(printed))
	}
	for _, required := range []string{
		"goIntegerMax",
		"goIntegerMin",
		"goStringMax",
		"goStringMin",
		"Math.max",
		"Math.min",
	} {
		if !strings.Contains(printed, required) {
			t.Fatalf("ordered artifact lacks %q:\n%s", required, printed)
		}
	}
}

func TestOrderedBuiltinIdentityNotSpellingSelectsOperation(t *testing.T) {
	loaded, function, call := loadOrderedCall(t, "MaxInt32")
	before := compileSingleOrdered(t, loaded, function.Name.Name)
	identifier := call.Fun.(*ast.Ident)
	identifier.Name = "spelling_must_not_be_read"
	after := compileSingleOrdered(t, loaded, function.Name.Name)
	if before != after {
		t.Fatalf(
			"builtin spelling changed output\nbefore:\n%s\nafter:\n%s",
			before,
			after,
		)
	}

	loaded, function, call = loadOrderedCall(t, "MaxInt32")
	identifier = call.Fun.(*ast.Ident)
	minimum, ok := types.Universe.Lookup("min").(*types.Builtin)
	if !ok {
		t.Fatal("universe min is not a builtin")
	}
	loaded.TypesInfo().Uses[identifier] = minimum
	mutated := compileSingleOrdered(t, loaded, function.Name.Name)
	if !strings.Contains(mutated, "Math.min") ||
		strings.Contains(mutated, "Math.max") {
		t.Fatalf("builtin identity mutation did not select min:\n%s", mutated)
	}
}

func TestOrderedBuiltinEvidenceAndSyntaxMutationsFailClosed(t *testing.T) {
	t.Run("missing builtin identity", func(t *testing.T) {
		loaded, function, call := loadOrderedCall(t, "MaxInt32")
		delete(loaded.TypesInfo().Uses, call.Fun.(*ast.Ident))
		assertOrderedCallUnsupported(t, loaded, function)
	})
	t.Run("missing result facts", func(t *testing.T) {
		loaded, function, call := loadOrderedCall(t, "MaxInt32")
		delete(loaded.TypesInfo().Types, call)
		assertOrderedCallUnsupported(t, loaded, function)
	})
	t.Run("ellipsis", func(t *testing.T) {
		loaded, function, call := loadOrderedCall(t, "MaxInt32")
		call.Ellipsis = call.Lparen
		assertOrderedCallUnsupported(t, loaded, function)
	})
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
			for _, statement := range function.Body().(tsgo.Block).Statements() {
				if result, ok := statement.(tsgo.ReturnStatement); ok {
					return result.Expression()
				}
			}
		}
	}
	t.Fatalf("target function %s has no return", name)
	return nil
}

func assertRuntimeDefinitionCount(
	t *testing.T,
	emission emit.ProgramEmission,
	path string,
	want int,
) {
	t.Helper()
	for _, file := range emission.Files() {
		if file.OutputPath() == path {
			if got := len(file.SourceFile().Statements()); got != want {
				t.Fatalf("%s definitions = %d, want %d", path, got, want)
			}
			return
		}
	}
	t.Fatalf("runtime file %s is absent", path)
}

func assertNoRuntimeFile(
	t *testing.T,
	emission emit.ProgramEmission,
	path string,
) {
	t.Helper()
	for _, file := range emission.Files() {
		if file.OutputPath() == path {
			t.Fatalf("unrequested runtime file %s was emitted", path)
		}
	}
}

func loadOrderedCall(
	t *testing.T,
	name string,
) (*load.Package, *ast.FuncDecl, *ast.CallExpr) {
	t.Helper()
	loaded, err := load.One(context.Background(), load.Request{
		Directory: orderedFixtureDirectory(),
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
		return loaded, function, result.Results[0].(*ast.CallExpr)
	}
	t.Fatalf("source function %s is absent", name)
	return nil, nil, nil
}

func assertOrderedCallUnsupported(
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

func compileSingleOrdered(
	t *testing.T,
	loaded *load.Package,
	name string,
) string {
	t.Helper()
	root, err := emit.NewRoot(loaded.Types().Scope().Lookup(name))
	if err != nil {
		t.Fatal(err)
	}
	emission, err := emit.Compile(loaded.Program(), []emit.Root{root})
	if err != nil {
		t.Fatal(err)
	}
	_, _, printed := printOrdered(t, t.TempDir(), emission)
	return printed
}
