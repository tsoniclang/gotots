package semantic

import (
	"strings"
	"testing"

	"github.com/tsoniclang/gotots/internal/identity"
)

func TestRecursiveMethodDescriptorRejectsReceiverBackEdge(
	t *testing.T,
) {
	fixture := semanticFixture(t)
	declarationID := mustPackageDeclarationID(
		t,
		fixture.pkg,
		identity.SemanticObjectType,
		"Recursive",
	)
	namedID, err := NominalTypeID(
		TypeNamed, declarationID, nil,
	)
	if err != nil {
		t.Fatal(err)
	}
	validSignature := mustSemanticType(t, TypeSpec{
		Kind: TypeSignature,
		Signature: Signature{
			Results: []identity.SemanticTypeID{namedID},
		},
	})
	validInterface := mustSemanticType(t, TypeSpec{
		Kind: TypeInterface,
		Methods: []TypeMethod{{
			Name:      "Next",
			Signature: validSignature.ID(),
			Ordinal:   0,
		}},
		TypeSet: TypeSetUniverse,
	})
	validNamed := mustSemanticType(t, TypeSpec{
		Kind:        TypeNamed,
		Declaration: declarationID,
		Underlying:  validInterface.ID(),
	})
	if validNamed.ID() != namedID {
		t.Fatalf(
			"recursive nominal identity=%s, want %s",
			validNamed.ID(), namedID,
		)
	}
	declaration := mustSemanticDeclaration(
		t,
		fixture,
		declarationID,
		identity.SemanticObjectType,
		"Recursive",
		namedID,
	)
	if _, err := NewPackage(recursiveTypePackageInput(
		t,
		fixture,
		declaration,
		validSignature,
		validInterface,
		validNamed,
	)); err != nil {
		t.Fatalf("receiver-free recursive interface was rejected: %v", err)
	}

	mutatedSignature := mustSemanticType(t, TypeSpec{
		Kind: TypeSignature,
		Signature: Signature{
			Receiver: namedID,
			Results:  []identity.SemanticTypeID{namedID},
		},
	})
	mutatedInterface := mustSemanticType(t, TypeSpec{
		Kind: TypeInterface,
		Methods: []TypeMethod{{
			Name:      "Next",
			Signature: mutatedSignature.ID(),
			Ordinal:   0,
		}},
		TypeSet: TypeSetUniverse,
	})
	mutatedNamed := mustSemanticType(t, TypeSpec{
		Kind:        TypeNamed,
		Declaration: declarationID,
		Underlying:  mutatedInterface.ID(),
	})
	_, err = NewPackage(recursiveTypePackageInput(
		t,
		fixture,
		declaration,
		mutatedSignature,
		mutatedInterface,
		mutatedNamed,
	))
	if err == nil ||
		!strings.Contains(
			err.Error(),
			"method descriptor signature retains receiver",
		) {
		t.Fatalf(
			"receiver back-edge admission error=%v",
			err,
		)
	}
}

func recursiveTypePackageInput(
	t *testing.T,
	fixture semanticFixtureState,
	declaration Declaration,
	signature Type,
	iface Type,
	named Type,
) PackageInput {
	types := []Type{signature, iface, named}
	witnesses := make([]TypeWitness, 0, len(types))
	for _, record := range types {
		witnesses = append(
			witnesses,
			mustTypeWitness(
				t,
				fixture.pkg,
				record.ID(),
				fixture.authority,
			),
		)
	}
	return PackageInput{
		ID:            fixture.pkg,
		Provenance:    ProvenanceWorkspaceModule,
		Declarations:  []Declaration{declaration},
		Types:         types,
		TypeWitnesses: witnesses,
	}
}
