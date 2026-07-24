package semantic

import (
	"runtime"
	"testing"

	"github.com/tsoniclang/gotots/internal/identity"
)

func TestDeclarationTargetsAreOwnedOnceByCanonicalTypes(
	t *testing.T,
) {
	fixture := semanticFixture(t)
	integer := mustSemanticType(t, TypeSpec{
		Kind: TypeBasic, Basic: BasicInt,
	})
	text := mustSemanticType(t, TypeSpec{
		Kind: TypeBasic, Basic: BasicString,
	})
	structure := mustSemanticType(t, TypeSpec{
		Kind: TypeStruct,
		Fields: []TypeField{{
			Name: "Value", Type: integer.ID(), Ordinal: 0,
		}},
	})
	errorSignature := mustSemanticType(t, TypeSpec{
		Kind: TypeSignature,
		Signature: Signature{
			Results: []identity.SemanticTypeID{text.ID()},
		},
	})
	iface := mustSemanticType(t, TypeSpec{
		Kind: TypeInterface,
		Methods: []TypeMethod{{
			Name: "Error", Signature: errorSignature.ID(),
			Ordinal: 0,
		}},
		TypeSet: TypeSetUniverse,
	})
	thingDeclaration := mustPackageDeclarationID(
		t, fixture.pkg, identity.SemanticObjectType, "Thing",
	)
	faultDeclaration := mustPackageDeclarationID(
		t, fixture.pkg, identity.SemanticObjectType, "Fault",
	)
	aliasDeclaration := mustPackageDeclarationID(
		t, fixture.pkg, identity.SemanticObjectAlias, "Alias",
	)
	thing := mustSemanticType(t, TypeSpec{
		Kind:        TypeNamed,
		Declaration: thingDeclaration,
		Underlying:  structure.ID(),
	})
	fault := mustSemanticType(t, TypeSpec{
		Kind:        TypeNamed,
		Declaration: faultDeclaration,
		Underlying:  iface.ID(),
	})
	alias := mustSemanticType(t, TypeSpec{
		Kind:        TypeAlias,
		Declaration: aliasDeclaration,
		Target:      thing.ID(),
	})
	declarations := []Declaration{
		mustSemanticDeclaration(
			t, fixture, thingDeclaration,
			identity.SemanticObjectType, "Thing", thing.ID(),
		),
		mustSemanticDeclaration(
			t, fixture, faultDeclaration,
			identity.SemanticObjectType, "Fault", fault.ID(),
		),
		mustSemanticDeclaration(
			t, fixture, aliasDeclaration,
			identity.SemanticObjectAlias, "Alias", alias.ID(),
		),
	}
	types := []Type{
		integer, text, structure, errorSignature,
		iface, thing, fault, alias,
	}
	witnesses := make([]TypeWitness, 0, len(types))
	for _, record := range types {
		witnesses = append(witnesses, mustTypeWitness(
			t, fixture.pkg, record.ID(), fixture.authority,
		))
	}
	pkg, err := NewPackage(PackageInput{
		ID: fixture.pkg, Provenance: ProvenanceWorkspaceModule,
		Declarations:  declarations,
		Types:         types,
		TypeWitnesses: witnesses,
	})
	if err != nil {
		t.Fatal(err)
	}
	thingField := mustMemberDeclarationID(
		t, thing.ID(), identity.SemanticObjectField, "Value", 0,
	)
	structField := mustMemberDeclarationID(
		t, structure.ID(), identity.SemanticObjectField, "Value", 0,
	)
	faultMethod := mustMemberDeclarationID(
		t, fault.ID(), identity.SemanticObjectMethod, "Error", 0,
	)
	interfaceMethod := mustMemberDeclarationID(
		t, iface.ID(), identity.SemanticObjectMethod, "Error", 0,
	)
	for _, id := range []identity.SemanticDeclarationID{
		thingField, structField, faultMethod, interfaceMethod,
	} {
		target, present := pkg.ResolveDeclarationTarget(id)
		if !present || target.ID() != id ||
			target.OwnerType() != id.OwnerType() {
			t.Fatalf("member target %s resolved as %+v", id, target)
		}
		if _, standalone := target.Standalone(); standalone {
			t.Fatalf("member target %s became standalone", id)
		}
		if _, present := pkg.Declaration(id); present {
			t.Fatalf(
				"member target %s has a standalone record", id,
			)
		}
	}
	var resolved DeclarationTarget
	var present bool
	allocations := testing.AllocsPerRun(1_000, func() {
		resolved, present = pkg.ResolveDeclarationTarget(thingField)
	})
	runtime.KeepAlive(resolved)
	runtime.KeepAlive(present)
	if allocations != 0 {
		t.Fatalf(
			"member target resolution allocated %.2f times",
			allocations,
		)
	}
	aliasField := mustMemberDeclarationID(
		t, alias.ID(), identity.SemanticObjectField, "Value", 0,
	)
	wrongField := mustMemberDeclarationID(
		t, thing.ID(), identity.SemanticObjectField, "Other", 0,
	)
	for _, id := range []identity.SemanticDeclarationID{
		aliasField, wrongField,
	} {
		if _, present := pkg.ResolveDeclarationTarget(id); present {
			t.Fatalf("invalid member target %s resolved", id)
		}
	}
	census, err := pkg.MemberTargetCensus()
	if err != nil {
		t.Fatal(err)
	}
	if census.Count() != 4 || len(census.Digest()) != 64 {
		t.Fatalf(
			"member-target census count=%d digest=%q",
			census.Count(), census.Digest(),
		)
	}
	if allocations := testing.AllocsPerRun(1_000, func() {
		_, _ = pkg.MemberTargetCensus()
	}); allocations != 0 {
		t.Fatalf(
			"sealed member-target census access allocates %.2f times",
			allocations,
		)
	}
	if pkg.DeclarationCount() != len(declarations) {
		t.Fatalf(
			"standalone declarations=%d, want %d",
			pkg.DeclarationCount(), len(declarations),
		)
	}
}

func TestTypeMethodsRequireCanonicalLookupOrder(t *testing.T) {
	signature := mustSemanticType(t, TypeSpec{
		Kind: TypeSignature,
	})
	if _, err := NewType(TypeSpec{
		Kind: TypeInterface,
		Methods: []TypeMethod{
			{Name: "Zed", Signature: signature.ID(), Ordinal: 0},
			{Name: "Alpha", Signature: signature.ID(), Ordinal: 1},
		},
		TypeSet: TypeSetUniverse,
	}); err == nil {
		t.Fatal("out-of-order methods were accepted")
	}
}

func mustSemanticType(t *testing.T, spec TypeSpec) Type {
	t.Helper()
	record, err := NewType(spec)
	if err != nil {
		t.Fatal(err)
	}
	return record
}

func mustPackageDeclarationID(
	t *testing.T,
	pkg identity.PackageID,
	class identity.SemanticObjectClass,
	name string,
) identity.SemanticDeclarationID {
	t.Helper()
	id, err := identity.NewPackageDeclarationID(pkg, class, name)
	if err != nil {
		t.Fatal(err)
	}
	return id
}

func mustMemberDeclarationID(
	t *testing.T,
	owner identity.SemanticTypeID,
	class identity.SemanticObjectClass,
	name string,
	ordinal int,
) identity.SemanticDeclarationID {
	t.Helper()
	id, err := identity.NewMemberDeclarationID(
		owner, identity.PackageID{}, class, name, ordinal,
	)
	if err != nil {
		t.Fatal(err)
	}
	return id
}

func mustSemanticDeclaration(
	t *testing.T,
	fixture semanticFixtureState,
	id identity.SemanticDeclarationID,
	class identity.SemanticObjectClass,
	name string,
	typeID identity.SemanticTypeID,
) Declaration {
	t.Helper()
	record, err := NewDeclaration(
		id, fixture.pkg, class, name, typeID, true,
		Constant{}, fixture.authority,
	)
	if err != nil {
		t.Fatal(err)
	}
	return record
}
