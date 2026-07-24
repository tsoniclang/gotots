package semantic

import (
	"os"
	"testing"

	"github.com/tsoniclang/gotots/internal/identity"
)

func TestCheckerStorePreservesAuthorityAndOwnsLifecycle(t *testing.T) {
	pkg := checkerStorePackage(t)
	writer, err := NewCheckerStoreWriter()
	if err != nil {
		t.Fatal(err)
	}
	if err := writer.Append(pkg); err != nil {
		writer.Abort()
		t.Fatal(err)
	}
	store, metrics, err := writer.Seal()
	if err != nil {
		t.Fatal(err)
	}
	path := store.path
	if metrics.Packages() != 1 ||
		metrics.Definitions() != 1 ||
		path == "" {
		t.Fatalf(
			"checker store metrics/path = %+v / %q",
			metrics, path,
		)
	}
	if err := store.VisitPackage(
		pkg.ID(),
		func(decoded Package) error {
			definition, present := decoded.Definition(
				firstDefinitionID(t, decoded),
			)
			if decoded.DefinitionCount() != 1 ||
				!present ||
				definition.Authority().Kind() != AuthorityChecker {
				t.Fatalf(
					"decoded checker authority = %+v",
					definition,
				)
			}
			return nil
		},
	); err != nil {
		t.Fatal(err)
	}
	stats := store.ReadStats()
	if stats.ShardLoads != 1 ||
		stats.MaxPackagesResident != 1 ||
		stats.Metrics().Packages() != 1 {
		t.Fatalf("checker store reads = %+v", stats)
	}
	if err := store.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("closed checker store remains at %q: %v", path, err)
	}
	if err := store.VisitPackage(
		pkg.ID(), func(Package) error { return nil },
	); err == nil {
		t.Fatal("closed checker store accepted a package visit")
	}
}

func TestProjectedModelOwnsCheckerStoreLifecycle(t *testing.T) {
	pkg := checkerStorePackage(t)
	writer, err := NewCheckerStoreWriter()
	if err != nil {
		t.Fatal(err)
	}
	if err := writer.Append(pkg); err != nil {
		writer.Abort()
		t.Fatal(err)
	}
	store, _, err := writer.Seal()
	if err != nil {
		t.Fatal(err)
	}
	path := store.path
	definitionID := firstDefinitionID(t, pkg)
	declarationID := firstDeclarationID(t, pkg)
	model, err := NewProjectedModel(
		[]PackageProjectionInput{{
			ID: pkg.ID(), Provenance: pkg.Provenance(), Local: true,
			ExpectedDefinitions: []identity.DefinitionID{
				definitionID,
			},
			LocalDeclarations: []identity.SemanticDeclarationID{
				declarationID,
			},
		}},
		store,
		nil,
	)
	if err != nil {
		_ = store.Close()
		t.Fatal(err)
	}
	if err := model.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("closed model retained checker store %q: %v", path, err)
	}
	if err := model.VisitPackage(
		pkg.ID(), func(Package) error { return nil },
	); err == nil {
		t.Fatal("closed model accepted a package projection")
	}
}

func firstDefinitionID(
	t *testing.T,
	pkg Package,
) identity.DefinitionID {
	t.Helper()
	var out identity.DefinitionID
	if err := pkg.VisitDefinitions(func(
		record DefinitionSemantics,
	) error {
		out = record.Definition()
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	return out
}

func firstDeclarationID(
	t *testing.T,
	pkg Package,
) identity.SemanticDeclarationID {
	t.Helper()
	var out identity.SemanticDeclarationID
	if err := pkg.VisitDeclarations(func(record Declaration) error {
		out = record.ID()
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	return out
}

func checkerStorePackage(t *testing.T) Package {
	t.Helper()
	fixture := semanticFixture(t)
	result, err := NewType(TypeSpec{
		Kind: TypeBasic, Basic: BasicInt,
	})
	if err != nil {
		t.Fatal(err)
	}
	signature, err := NewType(TypeSpec{
		Kind: TypeSignature,
		Signature: Signature{
			Results: []identity.SemanticTypeID{result.ID()},
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
	pkg, err := NewPackage(PackageInput{
		ID:           fixture.pkg,
		Provenance:   ProvenanceWorkspaceModule,
		Definitions:  []DefinitionSemantics{definition},
		Declarations: []Declaration{declaration},
		Types:        []Type{result, signature},
		TypeWitnesses: []TypeWitness{
			mustTypeWitness(
				t, fixture.pkg, result.ID(), fixture.authority,
			),
			mustTypeWitness(
				t, fixture.pkg, signature.ID(), fixture.authority,
			),
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	return pkg
}
