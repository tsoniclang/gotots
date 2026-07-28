package constant_test

import (
	"errors"
	"go/ast"
	"go/token"
	"slices"
	"strings"
	"testing"

	"github.com/tsoniclang/gotots/internal/emit"
	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

// TestConstantFamilyNumberProfileExecutesDifferentially proves iota, inherited
// specs, rune constants, and untyped constants projected at their contextual
// type across defaulting, multiple targets, argument, assignment, case, and
// conversion contexts, plus typed constant bindings, emit exact TypeScript that
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

// TestConstantFamilyShape pins the exact TS-Go AST shape of the projection
// design: an untyped constant has no base binding but one exported projection
// per demanded representation, and every use is a constant-size reference to a
// projection — never an inline literal — while a typed constant is a single
// exported binding referenced by identity.
func TestConstantFamilyShape(t *testing.T) {
	loaded := loadConstantFamily(t)
	emission := compileConstantFamily(t, loaded, emit.DefaultOptions(), numberRoots...)
	source := constantFamilySourceFile(t, emission)

	declared := declaredTopLevelBindings(source)

	// No untyped constant is declared by its base name; its representations are.
	for _, base := range []string{"Scale", "Text", "Flag", "Alpha", "Letter"} {
		if slices.Contains(declared, base) {
			t.Fatalf("untyped constant %q must not be declared by its base name", base)
		}
	}
	// Scale is used at four distinct representations; each is one projection.
	for _, projection := range []string{"Scale$int8", "Scale$int32", "Scale$int64", "Scale$uint16"} {
		if !slices.Contains(declared, projection) {
			t.Fatalf("projection %q is absent from %v", projection, declared)
		}
	}
	for _, projection := range []string{
		"Scale$int",
		"Fraction$float64",
		"Letter$int32",
		"Text$string",
		"Flag$bool",
	} {
		if !slices.Contains(declared, projection) {
			t.Fatalf("defaulted projection %q is absent from %v", projection, declared)
		}
	}
	// The typed constants keep their single base bindings.
	for _, typed := range []string{"TypedWidth", "TypedEnabled"} {
		if !slices.Contains(declared, typed) {
			t.Fatalf("typed constant %q must keep its base binding", typed)
		}
	}

	// A bare untyped constant use is a constant-size reference to its
	// projection, never an inline literal — one reference per representation.
	multiple := targetFunction(t, source, "MultipleTargets")
	multipleElements := multiple.Body().(tsgo.Block).Statements()[0].(tsgo.ReturnStatement).
		Expression().(tsgo.ArrayLiteralExpression).Elements()
	for index, want := range []string{"Scale$int8", "Scale$int32", "Scale$int64", "Scale$uint16"} {
		reference, ok := multipleElements[index].(tsgo.Identifier)
		if !ok || reference.Text() != want {
			t.Fatalf("MultipleTargets element %d = %T, want %s reference", index, multipleElements[index], want)
		}
	}

	// A constant conversion expression folds to a value and materializes it, the
	// same as any constant expression — only a bare named-constant reference is
	// projected: Conversion returns int64(Scale) folded to 100.
	conversion := targetFunction(t, source, "Conversion")
	returned := conversion.Body().(tsgo.Block).Statements()[0].(tsgo.ReturnStatement).Expression()
	literal, ok := returned.(tsgo.NumericLiteral)
	if !ok || literal.Text() != "100" {
		t.Fatalf("Conversion returns %T (%v), want folded literal 100", returned, returned)
	}

	// A typed constant use is a reference to its base binding.
	typed := targetFunction(t, source, "Typed")
	elements := typed.Body().(tsgo.Block).Statements()[0].(tsgo.ReturnStatement).
		Expression().(tsgo.ArrayLiteralExpression).Elements()
	typedReference, ok := elements[0].(tsgo.Identifier)
	if !ok || typedReference.Text() != "TypedWidth" {
		t.Fatalf("Typed element 0 = %T, want TypedWidth reference", elements[0])
	}
}

// TestLocalUntypedConstantProjectsAtItsDeclaration proves a function-local
// untyped constant is not inlined at its uses but materialized once, at its
// demanded representation, at its original lexical declaration and referenced
// constant-size.
func TestLocalUntypedConstantProjectsAtItsDeclaration(t *testing.T) {
	loaded := loadConstantFamily(t)
	emission := compileConstantFamily(t, loaded, emit.DefaultOptions(), numberRoots...)
	source := constantFamilySourceFile(t, emission)

	local := targetFunction(t, source, "Local")
	statements := local.Body().(tsgo.Block).Statements()

	// The source declarations at the start of this fixture materialize the two
	// local projections before the return.
	var projectionNames []string
	for _, statement := range statements {
		variable, ok := statement.(tsgo.VariableStatement)
		if !ok {
			continue
		}
		for _, modifier := range variable.Modifiers() {
			if modifier.Kind() == tsgo.SyntaxKindExportKeyword {
				t.Fatalf("local projection %v must not be exported", statement)
			}
		}
		name := variable.DeclarationList().Declarations()[0].Name().(tsgo.Identifier).Text()
		projectionNames = append(projectionNames, name)
	}
	slices.Sort(projectionNames)
	if !slices.Equal(projectionNames, []string{"high$int", "low$int"}) {
		t.Fatalf(
			"Local declarations materialize %v, want [high$int low$int]",
			projectionNames,
		)
	}

	// The return references the projections, never inline literals.
	returned := statements[len(statements)-1].(tsgo.ReturnStatement).
		Expression().(tsgo.ArrayLiteralExpression).Elements()
	for index, want := range []string{"low$int", "high$int"} {
		reference, ok := returned[index].(tsgo.Identifier)
		if !ok || reference.Text() != want {
			t.Fatalf("Local return element %d = %T, want %s reference", index, returned[index], want)
		}
	}
}

func declaredTopLevelBindings(source tsgo.SourceFile) []string {
	var declared []string
	for _, statement := range source.Statements() {
		variable, ok := statement.(tsgo.VariableStatement)
		if !ok {
			continue
		}
		for _, declaration := range variable.DeclarationList().Declarations() {
			declared = append(declared, declaration.Name().(tsgo.Identifier).Text())
		}
	}
	return declared
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
		unsupported.Role != api.RolePackageConstantValue ||
		unsupported.Construct != "*ast.Ident" {
		t.Fatalf("error = %#v, want package-constant-value expression UnsupportedError at the projection's value owner", err)
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

// TestConstantValueSyntaxCannotOverrideCheckerEvidence poisons the source AST
// after type checking while retaining the original checker graph. This is an
// internal mutation, not a second valid source program: a spelling-based value
// path would emit 101, while the authoritative *types.Const must still emit
// 100 and therefore leave the target artifact unchanged.
func TestConstantValueSyntaxCannotOverrideCheckerEvidence(t *testing.T) {
	baseline := printConstantFamily(t,
		compileConstantFamily(t, loadConstantFamily(t), emit.DefaultOptions(), "MultipleTargets"))

	loaded := loadConstantFamily(t)
	spec := packageConstSpec(t, loaded, "Scale")
	spec.Values[0] = &ast.BasicLit{
		ValuePos: spec.Values[0].Pos(),
		Kind:     token.INT,
		Value:    "101",
	}
	mutated := printConstantFamily(t,
		compileConstantFamily(t, loaded, emit.DefaultOptions(), "MultipleTargets"))
	if baseline != mutated {
		t.Fatalf("stale source syntax overrode checker-owned constant value:\n%s\n---\n%s", baseline, mutated)
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
