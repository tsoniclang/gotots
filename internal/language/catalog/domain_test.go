package catalog

import (
	"go/constant"
	"go/importer"
	"go/token"
	"go/types"
	"sort"
	"testing"
)

func TestTokenCatalogExactJoinsToolchain(t *testing.T) {
	pkg, err := importer.Default().Import("go/token")
	if err != nil {
		t.Fatal(err)
	}
	var constantNames []string
	for _, name := range pkg.Scope().Names() {
		object, ok := pkg.Scope().Lookup(name).(*types.Const)
		if !ok || object.Type().String() != "go/token.Token" {
			continue
		}
		constantNames = append(constantNames, name)
	}
	sort.Strings(constantNames)

	seen := map[TokenKind]string{}
	for _, name := range constantNames {
		object := pkg.Scope().Lookup(name).(*types.Const)
		value, exact := constant.Int64Val(object.Val())
		if !exact {
			t.Fatalf("toolchain token %s has non-integer value %s", name, object.Val())
		}
		toolchain := token.Token(value)
		member := TokenBySpelling(toolchain.String())
		if !member.Valid() {
			t.Fatalf(
				"toolchain token %s=%d/%q has no catalog member",
				name, toolchain, toolchain,
			)
		}
		if prior, duplicate := seen[member]; duplicate {
			t.Fatalf(
				"toolchain tokens %s and %s map to %s",
				prior, name, member,
			)
		}
		seen[member] = name
		if member.ConstName() == "" ||
			member.ConstName() != name ||
			member.Spelling() != toolchain.String() {
			t.Errorf(
				"token %s has descriptor name=%q spelling=%q, toolchain has name=%q spelling=%q",
				member, member.ConstName(), member.Spelling(), name, toolchain,
			)
		}
		expectedClass := TokenClassSpecial
		switch {
		case toolchain.IsLiteral():
			expectedClass = TokenClassLiteral
		case toolchain.IsOperator():
			expectedClass = TokenClassOperator
		case toolchain.IsKeyword():
			expectedClass = TokenClassKeyword
		}
		if member.Class() != expectedClass {
			t.Errorf(
				"token %s class=%s, toolchain class=%s",
				member, member.Class(), expectedClass,
			)
		}
	}
	if len(seen) != len(AllTokens()) {
		t.Fatalf(
			"toolchain exports %d token constants, catalog has %d",
			len(seen), len(AllTokens()),
		)
	}
	for _, member := range AllTokens() {
		if _, present := seen[member]; !present {
			t.Errorf("catalog token %s has no toolchain token", member)
		}
	}
}

func TestPredeclaredCatalogExactJoinsUniverse(t *testing.T) {
	catalogByName := map[string]PredeclaredKind{}
	for _, member := range AllPredeclared() {
		if member.Name() == "" || !member.Class().Valid() {
			t.Fatalf("predeclared member %d has an incomplete descriptor", member)
		}
		if prior, duplicate := catalogByName[member.Name()]; duplicate {
			t.Fatalf(
				"predeclared members %s and %s share %q",
				prior, member, member.Name(),
			)
		}
		catalogByName[member.Name()] = member
	}
	names := types.Universe.Names()
	if len(names) != len(catalogByName) {
		t.Fatalf(
			"toolchain universe has %d names, catalog has %d",
			len(names), len(catalogByName),
		)
	}
	for _, name := range names {
		object := types.Universe.Lookup(name)
		member, present := catalogByName[name]
		if !present {
			t.Errorf("toolchain predeclared %q is absent from the catalog", name)
			continue
		}
		var expected PredeclaredClass
		switch object.(type) {
		case *types.TypeName:
			expected = PredeclaredClassType
		case *types.Const:
			expected = PredeclaredClassConstant
		case *types.Nil:
			expected = PredeclaredClassNil
		case *types.Builtin:
			expected = PredeclaredClassFunction
		default:
			t.Fatalf(
				"toolchain predeclared %q has unclassified object %T",
				name, object,
			)
		}
		if member.Class() != expected {
			t.Errorf(
				"predeclared %q class=%s, toolchain class=%s",
				name, member.Class(), expected,
			)
		}
	}
}

func TestClosedSemanticCatalogsAreTotal(t *testing.T) {
	directiveNames := map[string]DirectiveKind{}
	for _, member := range AllDirectives() {
		if member.Name() == "" || !member.Disposition().Valid() {
			t.Errorf("directive %d has an incomplete descriptor", member)
		}
		if prior, duplicate := directiveNames[member.Name()]; duplicate {
			t.Errorf(
				"directives %s and %s share %q",
				prior, member, member.Name(),
			)
		}
		directiveNames[member.Name()] = member
		if member >= DirectiveGoBuild &&
			GoDirectiveByName(member.Name()) != member {
			t.Errorf("go directive %s does not round-trip", member)
		}
	}
	if GoDirectiveByName("not-a-toolchain-directive").Valid() {
		t.Error("unknown go directive was admitted")
	}

	variantNames := map[string]Variant{}
	for _, member := range AllVariants() {
		name := member.String()
		if name == "" {
			t.Errorf("variant %d is unnamed", member)
		}
		if prior, duplicate := variantNames[name]; duplicate {
			t.Errorf("variants %s and %s share %q", prior, member, name)
		}
		variantNames[name] = member
	}
	for kind := range variantBearing {
		if !kind.Valid() || kind.Disposition() != DispositionActive {
			t.Errorf("variant-bearing kind %s is not active", kind)
		}
	}

	implicitNames := map[string]ImplicitOp{}
	for _, operation := range AllImplicitOps() {
		if operation.Name() == "" ||
			!operation.Owner().Valid() ||
			operation.Evidence() == "" {
			t.Errorf("implicit operation %d has an incomplete contract", operation)
		}
		if prior, duplicate := implicitNames[operation.Name()]; duplicate {
			t.Errorf(
				"implicit operations %s and %s share %q",
				prior, operation, operation.Name(),
			)
		}
		implicitNames[operation.Name()] = operation
	}
}
