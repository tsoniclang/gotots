package semantic

import (
	"strings"
	"testing"

	"github.com/tsoniclang/gotots/internal/identity"
	"github.com/tsoniclang/gotots/internal/language/catalog"
)

func TestNominalTypesPreserveAliasAndDefinitionIdentity(t *testing.T) {
	fixture := semanticFixture(t)
	basic, err := NewType(TypeSpec{
		Kind:  TypeBasic,
		Basic: BasicInt,
	})
	if err != nil {
		t.Fatal(err)
	}
	definedDeclaration, err := identity.NewPackageDeclarationID(
		fixture.pkg, identity.SemanticObjectType, "Count",
	)
	if err != nil {
		t.Fatal(err)
	}
	aliasDeclaration, err := identity.NewPackageDeclarationID(
		fixture.pkg, identity.SemanticObjectAlias, "CountAlias",
	)
	if err != nil {
		t.Fatal(err)
	}
	defined, err := NewType(TypeSpec{
		Kind:        TypeNamed,
		Declaration: definedDeclaration,
		Underlying:  basic.ID(),
	})
	if err != nil {
		t.Fatal(err)
	}
	alias, err := NewType(TypeSpec{
		Kind:        TypeAlias,
		Declaration: aliasDeclaration,
		Target:      defined.ID(),
	})
	if err != nil {
		t.Fatal(err)
	}
	if basic.ID() == defined.ID() ||
		defined.ID() == alias.ID() ||
		basic.ID() == alias.ID() {
		t.Fatalf(
			"type identities collapsed: basic=%s defined=%s alias=%s",
			basic.ID(), defined.ID(), alias.ID(),
		)
	}
}

func TestSemanticPackageIsImmutableAndResolutionConserved(
	t *testing.T,
) {
	fixture := semanticFixture(t)
	basic, err := NewType(TypeSpec{
		Kind:  TypeBasic,
		Basic: BasicInt,
	})
	if err != nil {
		t.Fatal(err)
	}
	declarationID, err := identity.NewPackageDeclarationID(
		fixture.pkg, identity.SemanticObjectFunction, "F",
	)
	if err != nil {
		t.Fatal(err)
	}
	signature, err := NewType(TypeSpec{
		Kind: TypeSignature,
		Signature: Signature{
			Results: []identity.SemanticTypeID{basic.ID()},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	declaration, err := NewDeclaration(
		declarationID,
		fixture.pkg,
		identity.SemanticObjectFunction,
		"F",
		signature.ID(),
		true,
		Constant{},
		fixture.authority,
	)
	if err != nil {
		t.Fatal(err)
	}
	definition, err := NewDefinitionSemantics(
		DefinitionSemanticsSpec{
			Definition: fixture.definition,
			Package:    fixture.pkg,
			Form:       DefinitionFormCallable,
			Authority:  fixture.authority,
			Name:       "F",
			Declarations: []identity.SemanticDeclarationID{
				declarationID,
			},
			Signature: signature.ID(),
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	operationID, err := identity.NewOperationID(
		fixture.definition, fixture.body,
	)
	if err != nil {
		t.Fatal(err)
	}
	operation, err := NewOperation(OperationSpec{
		ID:         operationID,
		Kind:       OperationLiteral,
		Syntax:     catalog.KindBasicLit,
		Variant:    catalog.VariantNone,
		Role:       catalog.RoleReturnValue,
		Token:      catalog.TokenINT,
		Mode:       ValueModeValue,
		Arity:      ResultArityOne,
		Place:      PlaceNone,
		ResultType: basic.ID(),
		Object:     NoObjectReference(),
	})
	if err != nil {
		t.Fatal(err)
	}
	resolution, err := NewOccurrenceResolution(ResolutionSpec{
		Occurrence: fixture.body,
		Owner:      fixture.definition,
		Syntax:     catalog.KindBasicLit,
		Role:       catalog.RoleReturnValue,
		Variant:    catalog.VariantNone,
		Domain:     catalog.ResolutionDomainExecutable,
		Kind:       ResolutionOperation,
		Operation:  operationID,
	})
	if err != nil {
		t.Fatal(err)
	}
	pkg, err := NewPackage(PackageInput{
		ID:           fixture.pkg,
		Provenance:   ProvenanceWorkspaceModule,
		Definitions:  []DefinitionSemantics{definition},
		Resolutions:  []OccurrenceResolution{resolution},
		Declarations: []Declaration{declaration},
		Types:        []Type{basic, signature},
		TypeWitnesses: []TypeWitness{
			mustTypeWitness(t, fixture.pkg, basic.ID(), fixture.authority),
			mustTypeWitness(t, fixture.pkg, signature.ID(), fixture.authority),
		},
		Operations: []Operation{operation},
	})
	if err != nil {
		t.Fatal(err)
	}
	model, err := NewModel([]Package{pkg})
	if err != nil {
		t.Fatal(err)
	}
	var visited []Package
	if err := model.VisitPackages(func(pkg Package) error {
		visited = append(visited, pkg)
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	visited[0] = Package{}
	var revisited Package
	if err := model.VisitPackages(func(pkg Package) error {
		revisited = pkg
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if revisited.ID() != fixture.pkg {
		t.Fatal("model exposed mutable package storage")
	}
	resolutions := pkg.Resolutions()
	resolutions[0] = OccurrenceResolution{}
	if pkg.Resolutions()[0].Occurrence() != fixture.body {
		t.Fatal("package exposed mutable resolution storage")
	}
}

func mustTypeWitness(
	t *testing.T,
	pkg identity.PackageID,
	typeID identity.SemanticTypeID,
	authority Authority,
) TypeWitness {
	t.Helper()
	record, err := NewTypeWitness(pkg, typeID, authority)
	if err != nil {
		t.Fatal(err)
	}
	return record
}

func TestResolutionRejectsMismatchedOperationOwner(t *testing.T) {
	fixture := semanticFixture(t)
	otherSpan, _ := identity.NewSpanID(
		fixture.file, 100, 120,
	)
	otherRoot, _ := identity.NewOccurrenceID(
		otherSpan, uint16(catalog.KindFuncDecl),
	)
	otherDefinition, _ := identity.NewSourceDefinitionID(
		otherRoot, identity.DefinitionFuncDecl,
	)
	operation, _ := identity.NewOperationID(
		otherDefinition, fixture.body,
	)
	if _, err := NewOccurrenceResolution(ResolutionSpec{
		Occurrence: fixture.body,
		Owner:      fixture.definition,
		Syntax:     catalog.KindBasicLit,
		Role:       catalog.RoleReturnValue,
		Variant:    catalog.VariantNone,
		Domain:     catalog.ResolutionDomainExecutable,
		Kind:       ResolutionOperation,
		Operation:  operation,
	}); err == nil {
		t.Fatal("resolution accepted an operation from another definition")
	}
}

type semanticFixtureState struct {
	pkg        identity.PackageID
	file       identity.FileID
	root       identity.OccurrenceID
	body       identity.OccurrenceID
	definition identity.DefinitionID
	authority  Authority
}

func semanticFixture(t *testing.T) semanticFixtureState {
	t.Helper()
	module, err := identity.NewModuleID("example.com/semantic", "")
	if err != nil {
		t.Fatal(err)
	}
	owner, err := identity.NewModuleOwner(module)
	if err != nil {
		t.Fatal(err)
	}
	pkg, err := identity.NewPackageID(
		owner, "example.com/semantic/model",
	)
	if err != nil {
		t.Fatal(err)
	}
	file, err := identity.NewFileID(owner, "model/model.go")
	if err != nil {
		t.Fatal(err)
	}
	rootSpan, _ := identity.NewSpanID(file, 0, 90)
	root, _ := identity.NewOccurrenceID(
		rootSpan, uint16(catalog.KindFuncDecl),
	)
	bodySpan, _ := identity.NewSpanID(file, 50, 51)
	body, _ := identity.NewOccurrenceID(
		bodySpan, uint16(catalog.KindBasicLit),
	)
	definition, _ := identity.NewSourceDefinitionID(
		root, identity.DefinitionFuncDecl,
	)
	digest := strings.Repeat("ab", 32)
	authority, err := NewCheckerAuthority(
		digest, digest, digest, digest, digest,
	)
	if err != nil {
		t.Fatal(err)
	}
	return semanticFixtureState{
		pkg: pkg, file: file, root: root, body: body,
		definition: definition, authority: authority,
	}
}
