package semantic

import (
	"testing"

	"github.com/tsoniclang/gotots/internal/identity"
	"github.com/tsoniclang/gotots/internal/language/catalog"
)

func semanticWirePackage(t *testing.T) Package {
	t.Helper()
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
		fixture.pkg,
		identity.SemanticObjectFunction,
		"F",
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
	operationID, err := identity.NewOperationID(
		fixture.definition,
		fixture.body,
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
			mustTypeWitness(
				t,
				fixture.pkg,
				basic.ID(),
				fixture.authority,
			),
			mustTypeWitness(
				t,
				fixture.pkg,
				signature.ID(),
				fixture.authority,
			),
		},
		Operations: []Operation{operation},
	})
	if err != nil {
		t.Fatal(err)
	}
	return pkg
}

func semanticWireManifestEntry(
	t *testing.T,
	pkg Package,
	shardBytes int64,
) packageShardManifest {
	t.Helper()
	definitions := make([]string, 0, pkg.DefinitionCount())
	if err := pkg.VisitDefinitions(
		func(record DefinitionSemantics) error {
			definitions = append(
				definitions,
				record.Definition().String(),
			)
			return nil
		},
	); err != nil {
		t.Fatal(err)
	}
	declarations := make([]string, 0, pkg.DeclarationCount())
	if err := pkg.VisitDeclarations(
		func(record Declaration) error {
			declarations = append(
				declarations,
				record.ID().String(),
			)
			return nil
		},
	); err != nil {
		t.Fatal(err)
	}
	memberTargets, err := pkg.MemberTargetCensus()
	if err != nil {
		t.Fatal(err)
	}
	return packageShardManifest{
		Package:            pkg.ID().String(),
		Provenance:         uint8(pkg.Provenance()),
		Definitions:        definitions,
		Declarations:       declarations,
		DefinitionCount:    pkg.DefinitionCount(),
		ResolutionCount:    pkg.ResolutionCount(),
		DeclarationCount:   pkg.DeclarationCount(),
		MemberTargetCount:  memberTargets.Count(),
		MemberTargetDigest: memberTargets.Digest(),
		BindingCount:       pkg.BindingCount(),
		TypeCount:          pkg.TypeCount(),
		OperationCount:     pkg.OperationCount(),
		UnsupportedCount:   pkg.UnsupportedCount(),
		ShardBytes:         shardBytes,
	}
}
