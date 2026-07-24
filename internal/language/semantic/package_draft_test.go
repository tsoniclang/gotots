package semantic

import (
	"reflect"
	"testing"

	"github.com/tsoniclang/gotots/internal/identity"
)

func TestPackageDraftRetainsNoPublicRecordPool(t *testing.T) {
	publicRecords := map[reflect.Type]bool{
		reflect.TypeFor[DefinitionSemantics]():  true,
		reflect.TypeFor[OccurrenceResolution](): true,
		reflect.TypeFor[Declaration]():          true,
		reflect.TypeFor[Binding]():              true,
		reflect.TypeFor[Type]():                 true,
		reflect.TypeFor[TypeWitness]():          true,
		reflect.TypeFor[Operation]():            true,
		reflect.TypeFor[Unsupported]():          true,
	}
	if violations := draftPublicRecordPools(
		reflect.TypeFor[PackageDraft](),
		publicRecords,
	); len(violations) != 0 {
		t.Fatalf(
			"semantic package draft retains public record pools: %v",
			violations,
		)
	}
	type invalidDraft struct {
		Input PackageInput
		Types []Type
	}
	if violations := draftPublicRecordPools(
		reflect.TypeFor[invalidDraft](),
		publicRecords,
	); len(violations) != 2 {
		t.Fatalf(
			"draft-storage negative control found %d violations, want 2",
			len(violations),
		)
	}
}

func draftPublicRecordPools(
	draft reflect.Type,
	publicRecords map[reflect.Type]bool,
) []string {
	var violations []string
	packageInput := reflect.TypeFor[PackageInput]()
	semanticPackage := draft.PkgPath()
	seen := map[reflect.Type]bool{}
	var inspect func(reflect.Type, string)
	inspect = func(typ reflect.Type, path string) {
		for typ.Kind() == reflect.Pointer {
			typ = typ.Elem()
		}
		if typ == packageInput {
			violations = append(violations, path)
			return
		}
		if publicRecords[typ] {
			violations = append(violations, path)
			return
		}
		switch typ.Kind() {
		case reflect.Array, reflect.Slice, reflect.Map:
			inspect(typ.Elem(), path)
		case reflect.Struct:
			if typ.PkgPath() != semanticPackage || seen[typ] {
				return
			}
			seen[typ] = true
			for index := 0; index < typ.NumField(); index++ {
				field := typ.Field(index)
				child := field.Name
				if path != "" {
					child = path + "." + child
				}
				inspect(field.Type, child)
			}
		}
	}
	inspect(draft, "")
	return violations
}

func TestArtifactDraftCompactsEveryRetainedRecordClass(t *testing.T) {
	fixture := semanticFixture(t)
	basic, err := NewType(TypeSpec{
		Kind:  TypeBasic,
		Basic: BasicInt,
	})
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
	declarationID, err := identity.NewPackageDeclarationID(
		fixture.pkg, identity.SemanticObjectFunction, "F",
	)
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
	basicWitness := mustTypeWitness(
		t, fixture.pkg, basic.ID(), fixture.authority,
	)
	signatureWitness := mustTypeWitness(
		t, fixture.pkg, signature.ID(), fixture.authority,
	)
	draft, err := NewPackageDraft(
		fixture.pkg,
		ProvenanceWorkspaceModule,
		PackageCapacity{
			Definitions:  1,
			Declarations: 1,
			Types:        2,
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	if err := draft.AddDefinition(definition); err != nil {
		t.Fatal(err)
	}
	if err := draft.AddDeclaration(declaration); err != nil {
		t.Fatal(err)
	}
	if err := draft.AddType(basic, basicWitness); err != nil {
		t.Fatal(err)
	}
	if err := draft.AddType(signature, signatureWitness); err != nil {
		t.Fatal(err)
	}
	if len(draft.normalized.definitions.records) != 1 ||
		len(draft.normalized.declarations.records) != 1 ||
		len(draft.normalized.types.records) != 2 ||
		len(draft.normalized.witnesses.records) != 2 {
		t.Fatal("draft did not transfer records into normalized storage")
	}
	pkg, err := draft.sealArtifact()
	if err != nil {
		t.Fatal(err)
	}
	if pkg.DefinitionCount() != 1 ||
		pkg.DeclarationCount() != 1 ||
		pkg.TypeCount() != 2 ||
		len(pkg.definitions.records) != 1 ||
		len(pkg.declarations.records) != 1 ||
		len(pkg.types.records) != 2 ||
		len(pkg.witnesses.records) != 2 ||
		len(draft.normalized.definitions.records) != 0 {
		t.Fatal("artifact sealing did not transfer compact normalized storage")
	}
	if err := draft.AddType(basic, basicWitness); err == nil {
		t.Fatal("sealed artifact draft accepted another record")
	}
}

func TestPackageDraftTransfersOperationRelationsWithoutPublicClone(
	t *testing.T,
) {
	fixture := semanticFixture(t)
	definition, err := identity.NewImplicitDefinitionID(
		fixture.pkg,
		identity.ImplicitDefinitionPackageInit,
	)
	if err != nil {
		t.Fatal(err)
	}
	operationID, err := identity.NewImplicitOperationID(
		definition,
		identity.ImplicitDefinitionPackageInit,
		0,
	)
	if err != nil {
		t.Fatal(err)
	}
	operands := []identity.OccurrenceID{fixture.body}
	definitions := []identity.DefinitionID{fixture.definition}
	draft, err := NewPackageDraft(
		fixture.pkg,
		ProvenanceWorkspaceModule,
		PackageCapacity{Operations: 1},
	)
	if err != nil {
		t.Fatal(err)
	}
	if err := draft.AddOperation(OperationSpec{
		ID:          operationID,
		Kind:        OperationPackageInitialization,
		Mode:        ValueModeNone,
		Arity:       ResultArityZero,
		Place:       PlaceNone,
		Object:      NoObjectReference(),
		Operands:    operands,
		Definitions: definitions,
	}); err != nil {
		t.Fatal(err)
	}
	operands[0] = identity.OccurrenceID{}
	definitions[0] = identity.DefinitionID{}

	identities := newPackageIdentityProjection(
		draft.normalized.identities.projectionTable(),
	)
	if len(draft.normalized.operations.operands) != 1 ||
		identities.occurrence(
			draft.normalized.operations.operands[0],
		) != fixture.body ||
		len(draft.normalized.operations.definitions) != 1 ||
		identities.definition(
			draft.normalized.operations.definitions[0],
		) != fixture.definition {
		t.Fatal(
			"operation draft retained or lost its caller-owned relation slices",
		)
	}
}
