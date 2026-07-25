package semantic

import (
	"runtime"
	"testing"

	"github.com/tsoniclang/gotots/internal/identity"
)

func TestPackageMembershipUsesCanonicalStoredIdentities(t *testing.T) {
	pkg := semanticWirePackage(t)
	var definition identity.DefinitionID
	var occurrence identity.OccurrenceID
	var declaration identity.SemanticDeclarationID
	var typeID identity.SemanticTypeID
	var operation identity.OperationID
	if err := pkg.VisitDefinitions(func(record DefinitionSemantics) error {
		definition = record.Definition()
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if err := pkg.VisitResolutions(func(
		record OccurrenceResolution,
	) error {
		occurrence = record.Occurrence()
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if err := pkg.VisitDeclarations(func(record Declaration) error {
		declaration = record.ID()
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if err := pkg.VisitTypes(func(record Type) error {
		typeID = record.ID()
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if err := pkg.VisitOperations(func(record Operation) error {
		operation = record.ID()
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if !pkg.HasDefinition(definition) ||
		!pkg.HasResolution(occurrence) ||
		!pkg.HasDeclaration(declaration) ||
		!pkg.HasType(typeID) ||
		!pkg.HasOperation(operation) {
		t.Fatal("stored canonical identity was not found")
	}
	if pkg.HasDefinition(identity.DefinitionID{}) ||
		pkg.HasResolution(identity.OccurrenceID{}) ||
		pkg.HasDeclaration(identity.SemanticDeclarationID{}) ||
		pkg.HasBinding(identity.SemanticBindingID{}) ||
		pkg.HasType(identity.SemanticTypeID{}) ||
		pkg.HasOperation(identity.OperationID{}) ||
		pkg.HasUnsupported(identity.UnsupportedID{}) {
		t.Fatal("zero identity was admitted as package membership")
	}
}

func TestPackageMembershipDoesNotProjectRecords(t *testing.T) {
	pkg := semanticWirePackage(t)
	var typeID identity.SemanticTypeID
	var operation identity.OperationID
	if err := pkg.VisitTypes(func(record Type) error {
		typeID = record.ID()
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if err := pkg.VisitOperations(func(record Operation) error {
		operation = record.ID()
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	var present bool
	allocations := testing.AllocsPerRun(1_000, func() {
		present = pkg.HasType(typeID) && pkg.HasOperation(operation)
	})
	runtime.KeepAlive(present)
	if allocations != 0 {
		t.Fatalf(
			"canonical package membership allocated %.2f times",
			allocations,
		)
	}
}
