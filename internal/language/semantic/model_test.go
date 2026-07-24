package semantic

import (
	"fmt"
	"reflect"
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
	model := checkerModelForTest(t, pkg)
	projections := model.AuthorityProjections()
	if len(projections) != 1 ||
		projections[0].ID() != fixture.pkg ||
		!projections[0].HasLocalAuthority() ||
		projections[0].HasCertifiedAuthority() ||
		len(projections[0].ExpectedDefinitions()) != 1 {
		t.Fatalf("semantic authority projections = %+v", projections)
	}
	definitions := projections[0].ExpectedDefinitions()
	definitions[0] = identity.DefinitionID{}
	if model.AuthorityProjections()[0].
		ExpectedDefinitions()[0] != fixture.definition {
		t.Fatal("semantic authority projection exposed mutable census")
	}
	stats := model.ProjectionStats()
	if stats.Packages != 1 ||
		stats.LocalPackages != 1 ||
		stats.CertifiedPackages != 0 ||
		stats.MixedPackages != 0 {
		t.Fatalf("semantic projection stats = %+v", stats)
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
	resolution, present := pkg.Resolution(fixture.body)
	if !present || resolution.Occurrence() != fixture.body {
		t.Fatal("package canonical resolution lookup failed")
	}
}

func TestPackageReadSurfaceExposesNoRecordSlices(t *testing.T) {
	assertNoSemanticRecordSlices(t, reflect.TypeFor[Package]())
	if err := semanticRecordSliceMethod(
		reflect.TypeFor[packageSliceSurfaceControl](),
	); err == nil {
		t.Fatal("semantic record-slice surface control was not detected")
	}
}

type packageSliceSurfaceControl struct{}

func (packageSliceSurfaceControl) Records() []OccurrenceResolution {
	return nil
}

func assertNoSemanticRecordSlices(t *testing.T, typ reflect.Type) {
	t.Helper()
	if err := semanticRecordSliceMethod(typ); err != nil {
		t.Fatal(err)
	}
}

func semanticRecordSliceMethod(typ reflect.Type) error {
	for index := range typ.NumMethod() {
		method := typ.Method(index)
		for output := 0; output < method.Type.NumOut(); output++ {
			result := method.Type.Out(output)
			if result.Kind() == reflect.Slice &&
				result.Elem().PkgPath() ==
					reflect.TypeFor[Package]().PkgPath() {
				return fmt.Errorf(
					"%s exposes semantic record slice %s",
					method.Name, result,
				)
			}
		}
	}
	return nil
}

func checkerModelForTest(t *testing.T, packages ...Package) *Model {
	t.Helper()
	writer, err := NewCheckerStoreWriter()
	if err != nil {
		t.Fatal(err)
	}
	var projections []PackageProjectionInput
	for _, pkg := range packages {
		if err := writer.Append(pkg); err != nil {
			writer.Abort()
			t.Fatal(err)
		}
		input := PackageProjectionInput{
			ID: pkg.ID(), Provenance: pkg.Provenance(), Local: true,
		}
		if err := pkg.VisitDefinitions(func(
			definition DefinitionSemantics,
		) error {
			input.ExpectedDefinitions = append(
				input.ExpectedDefinitions, definition.Definition(),
			)
			return nil
		}); err != nil {
			t.Fatal(err)
		}
		if err := pkg.VisitDeclarations(func(
			declaration Declaration,
		) error {
			input.LocalDeclarations = append(
				input.LocalDeclarations, declaration.ID(),
			)
			return nil
		}); err != nil {
			t.Fatal(err)
		}
		projections = append(projections, input)
	}
	store, _, err := writer.Seal()
	if err != nil {
		t.Fatal(err)
	}
	model, err := NewProjectedModel(projections, store, nil)
	if err != nil {
		_ = store.Close()
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := model.Close(); err != nil {
			t.Errorf("close semantic test model: %v", err)
		}
	})
	return model
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

func TestTransientTypePoolClosesAndArtifactAdmissionRejectsExtras(
	t *testing.T,
) {
	fixture := semanticFixture(t)
	integer, err := NewType(TypeSpec{
		Kind: TypeBasic, Basic: BasicInt,
	})
	if err != nil {
		t.Fatal(err)
	}
	extra, err := NewType(TypeSpec{
		Kind: TypeBasic, Basic: BasicString,
	})
	if err != nil {
		t.Fatal(err)
	}
	declarationID, err := identity.NewPackageDeclarationID(
		fixture.pkg, identity.SemanticObjectVariable, "Value",
	)
	if err != nil {
		t.Fatal(err)
	}
	declaration, err := NewDeclaration(
		declarationID,
		fixture.pkg,
		identity.SemanticObjectVariable,
		"Value",
		integer.ID(),
		true,
		Constant{},
		fixture.authority,
	)
	if err != nil {
		t.Fatal(err)
	}
	input := PackageInput{
		ID:           fixture.pkg,
		Provenance:   ProvenanceWorkspaceModule,
		Declarations: []Declaration{declaration},
		Types:        []Type{integer, extra},
		TypeWitnesses: []TypeWitness{
			mustTypeWitness(
				t, fixture.pkg, integer.ID(), fixture.authority,
			),
			mustTypeWitness(
				t, fixture.pkg, extra.ID(), fixture.authority,
			),
		},
	}
	if _, err := NewPackage(input); err == nil {
		t.Fatal("artifact admission accepted an unreferenced type")
	}
	closed, err := FinalizePackageTypePool(input)
	if err != nil {
		t.Fatal(err)
	}
	if len(closed.Types) != 1 ||
		closed.Types[0].ID() != integer.ID() ||
		len(closed.TypeWitnesses) != 1 ||
		closed.TypeWitnesses[0].Type() != integer.ID() {
		t.Fatalf(
			"type closure types=%v witnesses=%v",
			closed.Types, closed.TypeWitnesses,
		)
	}
	if _, err := NewPackage(closed); err != nil {
		t.Fatal(err)
	}
}

func TestMemberTargetRetainsItsCanonicalOwnerType(t *testing.T) {
	fixture := semanticFixture(t)
	integer, err := NewType(TypeSpec{
		Kind: TypeBasic, Basic: BasicInt,
	})
	if err != nil {
		t.Fatal(err)
	}
	structure, err := NewType(TypeSpec{
		Kind: TypeStruct,
		Fields: []TypeField{{
			Name: "Value", Type: integer.ID(), Ordinal: 0,
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	declarationID, err := identity.NewMemberDeclarationID(
		structure.ID(),
		identity.PackageID{},
		identity.SemanticObjectField,
		"Value",
		0,
	)
	if err != nil {
		t.Fatal(err)
	}
	structural, err := NewStructuralEvidence(
		StructuralCompileTimeExpression,
		declarationID,
		identity.SemanticTypeID{},
	)
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
		Kind:       ResolutionStructuralOnly,
		Structural: structural,
	})
	if err != nil {
		t.Fatal(err)
	}
	signature, err := NewType(TypeSpec{Kind: TypeSignature})
	if err != nil {
		t.Fatal(err)
	}
	definition, err := NewDefinitionSemantics(
		DefinitionSemanticsSpec{
			Definition: fixture.definition,
			Package:    fixture.pkg,
			Form:       DefinitionFormCallable,
			Authority:  fixture.authority,
			Name:       "_",
			Signature:  signature.ID(),
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	input := PackageInput{
		ID:          fixture.pkg,
		Provenance:  ProvenanceWorkspaceModule,
		Definitions: []DefinitionSemantics{definition},
		Resolutions: []OccurrenceResolution{resolution},
		Types:       []Type{integer, structure, signature},
		TypeWitnesses: []TypeWitness{
			mustTypeWitness(
				t, fixture.pkg, integer.ID(), fixture.authority,
			),
			mustTypeWitness(
				t, fixture.pkg, structure.ID(), fixture.authority,
			),
			mustTypeWitness(
				t, fixture.pkg, signature.ID(), fixture.authority,
			),
		},
	}
	closed, err := FinalizePackageTypePool(input)
	if err != nil {
		t.Fatal(err)
	}
	if len(closed.Types) != 3 {
		t.Fatalf(
			"member type closure retained %d types, want 3",
			len(closed.Types),
		)
	}
	pkg, err := NewPackage(closed)
	if err != nil {
		t.Fatal(err)
	}
	if pkg.DeclarationCount() != 0 {
		t.Fatalf(
			"member target produced %d standalone declarations",
			pkg.DeclarationCount(),
		)
	}
	target, present := pkg.ResolveDeclarationTarget(declarationID)
	field, fieldTarget := target.Field()
	if !present ||
		target.OwnerType() != structure.ID() ||
		!fieldTarget ||
		field.Name != "Value" ||
		field.Type != integer.ID() {
		t.Fatalf("resolved member target = %+v", target)
	}
	if _, err := NewDeclaration(
		declarationID,
		fixture.pkg,
		identity.SemanticObjectField,
		"Value",
		integer.ID(),
		true,
		Constant{},
		fixture.authority,
	); err == nil {
		t.Fatal("standalone member declaration was accepted")
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
