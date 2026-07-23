package semantic

import "testing"

func TestPackageDraftTransfersOwnershipOnce(t *testing.T) {
	fixture := semanticFixture(t)
	draft, err := NewPackageDraft(
		fixture.pkg,
		ProvenanceWorkspaceModule,
		PackageCapacity{},
	)
	if err != nil {
		t.Fatal(err)
	}
	pkg, err := draft.SealProducer()
	if err != nil {
		t.Fatal(err)
	}
	if pkg.ID() != fixture.pkg ||
		pkg.Provenance() != ProvenanceWorkspaceModule {
		t.Fatalf(
			"sealed package identity=%s provenance=%s",
			pkg.ID(), pkg.Provenance(),
		)
	}
	if _, err := draft.SealProducer(); err == nil {
		t.Fatal("sealed package draft accepted a second seal")
	}
	if err := draft.AddDefinition(DefinitionSemantics{}); err == nil {
		t.Fatal("sealed package draft accepted another record")
	}
}
