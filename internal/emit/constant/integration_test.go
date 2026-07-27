package constant_test

import (
	"context"
	"errors"
	"go/ast"
	"go/token"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/tsoniclang/gotots/internal/emit"
	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/load"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

// TestConstantFamilyNumberProfileExecutesDifferentially proves iota, inherited
// specs, rune constants, and untyped constants projected at their contextual
// type across multiple targets, argument, assignment, case, and conversion
// contexts, plus typed constant bindings, emit exact TypeScript that
// strict-typechecks and executes identically to Go.
func TestConstantFamilyNumberProfileExecutesDifferentially(t *testing.T) {
	loaded := loadConstantFamily(t)
	emission := compileConstantFamily(t, loaded, emit.DefaultOptions(), numberRoots...)

	printed := printConstantFamily(t, emission)
	assertNoForbiddenConstructs(t, printed)

	workingDirectory := t.TempDir()
	goOutput := executeConstantFamilyGo(t, workingDirectory, false)
	targetOutput := executeConstantFamilyTS(t, emission, workingDirectory, false)
	if targetOutput != goOutput {
		t.Fatalf("TypeScript output = %q, Go output = %q", targetOutput, goOutput)
	}
}

// TestConstantFamilyBigIntProfileExecutesDifferentially additionally covers the
// large untyped constant (1<<63) projected at uint64 — impossible as its default
// int type — proving contextual typing under the bigint profile.
func TestConstantFamilyBigIntProfileExecutesDifferentially(t *testing.T) {
	loaded := loadConstantFamily(t)
	options := emit.DefaultOptions()
	options.IntegerRepresentation = emit.IntegerRepresentationBigInt
	emission := compileConstantFamily(t, loaded, options, append(slices.Clone(numberRoots), hugeRoot)...)

	printed := printConstantFamily(t, emission)
	assertNoForbiddenConstructs(t, printed)
	if !strings.Contains(printed, "9223372036854775808n") {
		t.Fatalf("bigint artifact lacks the projected large constant:\n%s", printed)
	}

	workingDirectory := t.TempDir()
	goOutput := executeConstantFamilyGo(t, workingDirectory, true)
	targetOutput := executeConstantFamilyTS(t, emission, workingDirectory, true)
	if targetOutput != goOutput {
		t.Fatalf("bigint TypeScript output = %q, Go output = %q", targetOutput, goOutput)
	}
}

// TestConstantFamilyShape pins the exact TS-Go AST shape: an untyped constant
// has no declaration and is inlined as a literal at each use, while a typed
// constant is a single exported binding referenced by identity.
func TestConstantFamilyShape(t *testing.T) {
	loaded := loadConstantFamily(t)
	emission := compileConstantFamily(t, loaded, emit.DefaultOptions(), numberRoots...)
	source := constantFamilySourceFile(t, emission)

	// Exactly the typed constants are declared; no untyped constant is.
	var declared []string
	for _, statement := range source.Statements() {
		variable, ok := statement.(tsgo.VariableStatement)
		if !ok {
			continue
		}
		name := variable.DeclarationList().Declarations()[0].Name().(tsgo.Identifier).Text()
		declared = append(declared, name)
	}
	slices.Sort(declared)
	if !slices.Equal(declared, []string{"TypedEnabled", "TypedWidth"}) {
		t.Fatalf("declared constants = %v, want only the typed constants", declared)
	}

	// An untyped constant use is an inline literal, not a reference: Conversion
	// returns the projected literal 100.
	conversion := targetFunction(t, source, "Conversion")
	returned := conversion.Body().(tsgo.Block).Statements()[0].(tsgo.ReturnStatement).Expression()
	literal, ok := returned.(tsgo.NumericLiteral)
	if !ok || literal.Text() != "100" {
		t.Fatalf("Conversion returns %T, want inline NumericLiteral 100", returned)
	}

	// A typed constant use is a reference: Typed returns [TypedWidth, ...].
	typed := targetFunction(t, source, "Typed")
	elements := typed.Body().(tsgo.Block).Statements()[0].(tsgo.ReturnStatement).
		Expression().(tsgo.ArrayLiteralExpression).Elements()
	reference, ok := elements[0].(tsgo.Identifier)
	if !ok || reference.Text() != "TypedWidth" {
		t.Fatalf("Typed element 0 = %T, want TypedWidth reference", elements[0])
	}
}

// TestUntypedConstantValueOwnerRejectsUnrepresentable proves the projection
// reaches the constant-value owner, not merely a type boundary: the large
// untyped constant projected at uint64 (a supported type) fails under the number
// profile at the value owner because its magnitude is not representable there.
func TestUntypedConstantValueOwnerRejectsUnrepresentable(t *testing.T) {
	loaded := loadConstantFamily(t)
	object := loaded.Types().Scope().Lookup(hugeRoot)
	root, err := emit.NewRoot(object)
	if err != nil {
		t.Fatal(err)
	}
	_, err = emit.Compile(loaded.Program(), []emit.Root{root})
	var unsupported *api.UnsupportedError
	if !errors.As(err, &unsupported) ||
		unsupported.Category != api.CategoryExpression ||
		unsupported.Role != api.RoleReturnResult ||
		unsupported.Construct != "*ast.Ident" {
		t.Fatalf("error = %#v, want return-result expression UnsupportedError at the value owner", err)
	}
}

// TestConstantValueSpellingIsIgnored proves the owner emits the checker's
// canonical value, not the source spelling: rewriting a constant's value
// expression to a DIFFERENT spelling of the SAME value (100 -> 0x64) leaves the
// emitted artifact byte-identical.
func TestConstantValueSpellingIsIgnored(t *testing.T) {
	baseline := printConstantFamily(t,
		compileConstantFamily(t, loadConstantFamily(t), emit.DefaultOptions(), "MultipleTargets"))

	loaded := loadConstantFamily(t)
	spec := packageConstSpec(t, loaded, "Scale")
	spec.Values[0] = &ast.BasicLit{
		ValuePos: spec.Values[0].Pos(),
		Kind:     token.INT,
		Value:    "0x64",
	}
	mutated := printConstantFamily(t,
		compileConstantFamily(t, loaded, emit.DefaultOptions(), "MultipleTargets"))
	if baseline != mutated {
		t.Fatalf("artifact changed after an alternate-spelling mutation (same checker value):\n%s\n---\n%s", baseline, mutated)
	}
}

// TestConstantBindingMutationFailsAtItsGate proves the owner relies on checker
// evidence: removing a typed constant's Def fails at a typed gate rather than
// emitting a guessed declaration.
func TestConstantBindingMutationFailsAtItsGate(t *testing.T) {
	loaded := loadConstantFamily(t)
	name := packageConstSpec(t, loaded, "TypedWidth").Names[0]
	delete(loaded.TypesInfo().Defs, name)

	object := loaded.Types().Scope().Lookup("Typed")
	root, err := emit.NewRoot(object)
	if err != nil {
		t.Fatal(err)
	}
	_, err = emit.Compile(loaded.Program(), []emit.Root{root})
	if err == nil {
		t.Fatal("mutation was accepted")
	}
	var unsupported *api.UnsupportedError
	var invariant *api.InvariantError
	if !errors.As(err, &unsupported) && !errors.As(err, &invariant) {
		t.Fatalf("mutation error = %#v, want a typed emit error", err)
	}
}

// TestConstantTypeNeighborsFailAtTypeOwner proves unsupported constant TYPES
// (complex, and float until its value family lands) remain a typed boundary, so
// the handler did not widen silently.
func TestConstantTypeNeighborsFailAtTypeOwner(t *testing.T) {
	dir := filepath.Join(repositoryRoot(), "testdata", "constructs", "value", "constant-boundaries")
	loaded, err := load.One(context.Background(), load.Request{Directory: dir, Pattern: "."})
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"Float", "Complex"} {
		t.Run(name, func(t *testing.T) {
			object := loaded.Types().Scope().Lookup(name)
			root, err := emit.NewRoot(object)
			if err != nil {
				t.Fatal(err)
			}
			_, err = emit.Compile(loaded.Program(), []emit.Root{root})
			var unsupported *api.UnsupportedError
			if !errors.As(err, &unsupported) || unsupported.Category != api.CategoryType {
				t.Fatalf("%s error = %#v, want type UnsupportedError", name, err)
			}
		})
	}
}

func targetFunction(t *testing.T, file tsgo.SourceFile, name string) tsgo.FunctionDeclaration {
	t.Helper()
	for _, statement := range file.Statements() {
		function, ok := statement.(tsgo.FunctionDeclaration)
		if ok && function.Name().Text() == name {
			return function
		}
	}
	t.Fatalf("target function %s not found", name)
	return nil
}
