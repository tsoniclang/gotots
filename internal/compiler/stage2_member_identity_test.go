package compiler

import (
	"testing"

	"github.com/tsoniclang/gotots/internal/identity"
	"github.com/tsoniclang/gotots/internal/language/catalog"
	"github.com/tsoniclang/gotots/internal/language/semantic"
	"github.com/tsoniclang/gotots/internal/scope/contract"
	"github.com/tsoniclang/gotots/internal/source"
)

const stage2MemberIdentityFixture = `package members

import "example.com/members/remote"

type A struct {
	X int
}

type B A

type Alias = A

type Embedded struct {
	A
}

type Box[T any] struct {
	Value T
}

func (box Box[T]) Load() T {
	return box.Value
}

func Read(
	a A,
	b B,
	alias Alias,
	embedded Embedded,
	ints Box[int],
	strings Box[string],
) int {
	_ = strings.Value
	_ = strings.Load()
	return a.X + b.X + alias.X + embedded.X +
		ints.Value + ints.Load()
}

func Anonymous(
	left struct{ X int },
	right struct{ X int },
) int {
	return left.X + right.X
}

type Reader struct{}

func (Reader) Read() int { return 1 }

func CrossPackage(
	value struct{ X int },
	reader interface{ Read() int },
) int {
	return remote.ReadStruct(value) + remote.ReadInterface(reader)
}

func Message(err error) string {
	return err.Error()
}
`

func TestStage2MemberIdentityUsesSemanticOwnerAndGenericOrigin(
	t *testing.T,
) {
	directory := t.TempDir()
	writeCompilerFile(
		t,
		directory,
		"go.mod",
		"module example.com/members\n\ngo 1.26.0\n",
	)
	writeCompilerFile(
		t, directory, "members.go", stage2MemberIdentityFixture,
	)
	writeCompilerFile(
		t,
		directory,
		"remote/remote.go",
		`package remote

func ReadStruct(value struct{ X int }) int { return value.X }

func ReadInterface(value interface{ Read() int }) int {
	return value.Read()
}
`,
	)
	inspection, err := inspectConstructsForTest(t, source.Request{
		Dir: directory, Patterns: []string{"./..."},
		ProviderContract: contract.DefaultID,
	})
	if err != nil {
		t.Fatal(err)
	}
	pkg := semanticPackageByImportPath(
		t, inspection.Semantic(), "example.com/members",
	)
	selected := map[string]map[identity.SemanticDeclarationID]bool{}
	for _, operation := range semanticOperations(pkg) {
		selection := operation.Spec().Selection
		if selection.IsZero() {
			continue
		}
		object := selection.Object()
		if selected[object.Name()] == nil {
			selected[object.Name()] =
				map[identity.SemanticDeclarationID]bool{}
		}
		selected[object.Name()][object] = true
	}
	if len(selected["X"]) != 3 {
		t.Fatalf(
			"defined, aliased, promoted, and structural owners produced X identities %v",
			selected["X"],
		)
	}
	if len(selected["Value"]) != 1 {
		t.Fatalf(
			"Box[int] and Box[string] produced Value identities %v",
			selected["Value"],
		)
	}
	if len(selected["Load"]) != 1 {
		t.Fatalf(
			"Box[int] and Box[string] produced Load identities %v",
			selected["Load"],
		)
	}
	if len(selected["Error"]) != 1 {
		t.Fatalf(
			"predeclared error method produced identities %v",
			selected["Error"],
		)
	}
	for name, identities := range selected {
		for declaration := range identities {
			if declaration.Form() !=
				identity.SemanticDeclarationMember ||
				declaration.OwnerType().IsZero() {
				t.Errorf(
					"selected %s declaration %s has owner=%s",
					name,
					declaration,
					declaration.OwnerType(),
				)
				continue
			}
			if _, present := pkg.Declaration(declaration); present {
				t.Errorf(
					"selected %s member %s has a standalone declaration",
					name, declaration,
				)
			}
			target, present := pkg.ResolveDeclarationTarget(declaration)
			if !present ||
				target.ID() != declaration ||
				target.OwnerType() != declaration.OwnerType() {
				t.Errorf(
					"selected %s member %s did not resolve: %+v",
					name, declaration, target,
				)
			}
		}
	}
	if census, err := pkg.MemberTargetCensus(); err != nil {
		t.Fatal(err)
	} else if census.Count() == 0 || len(census.Digest()) != 64 {
		t.Fatalf("member-target census = %+v", census)
	}
	if pkg.UnsupportedCount() != 0 {
		t.Fatalf(
			"member identity fixture has %d unsupported records",
			pkg.UnsupportedCount(),
		)
	}
	requireCrossPackageStructuralMembers(
		t,
		pkg,
		semanticPackageByImportPath(
			t,
			inspection.Semantic(),
			"example.com/members/remote",
		),
	)
}

func requireCrossPackageStructuralMembers(
	t *testing.T,
	local semantic.Package,
	remote semantic.Package,
) {
	t.Helper()
	localStruct := aggregateTypeWithMember(
		t, local, semantic.TypeStruct, "X",
	)
	remoteStruct := aggregateTypeWithMember(
		t, remote, semantic.TypeStruct, "X",
	)
	localInterface := aggregateTypeWithMember(
		t, local, semantic.TypeInterface, "Read",
	)
	remoteInterface := aggregateTypeWithMember(
		t, remote, semantic.TypeInterface, "Read",
	)
	if localStruct.ID() != remoteStruct.ID() ||
		localInterface.ID() != remoteInterface.ID() {
		t.Fatalf(
			"cross-package structural identities differ: struct=%s/%s interface=%s/%s",
			localStruct.ID(),
			remoteStruct.ID(),
			localInterface.ID(),
			remoteInterface.ID(),
		)
	}
	assertStructuralMemberOccurrences(
		t, local, localStruct.ID(), "X", 3,
	)
	assertStructuralMemberOccurrences(
		t, remote, remoteStruct.ID(), "X", 1,
	)
	assertStructuralMemberOccurrences(
		t, local, localInterface.ID(), "Read", 1,
	)
	assertStructuralMemberOccurrences(
		t, remote, remoteInterface.ID(), "Read", 1,
	)
	assertMemberTarget(
		t,
		local,
		localStruct.ID(),
		identity.SemanticObjectField,
		"X",
	)
	assertMemberTarget(
		t,
		remote,
		remoteStruct.ID(),
		identity.SemanticObjectField,
		"X",
	)
	assertMemberTarget(
		t,
		local,
		localInterface.ID(),
		identity.SemanticObjectMethod,
		"Read",
	)
	assertMemberTarget(
		t,
		remote,
		remoteInterface.ID(),
		identity.SemanticObjectMethod,
		"Read",
	)
	for _, pkg := range []semantic.Package{local, remote} {
		for _, declaration := range semanticDeclarations(pkg) {
			if declaration.ID().Form() ==
				identity.SemanticDeclarationMember {
				t.Fatalf(
					"package %s serialized member declaration %s",
					pkg.ID(), declaration.ID(),
				)
			}
		}
	}
	mutated := localStruct.Spec()
	mutated.Fields[0].Package = local.ID()
	if _, err := semantic.NewType(mutated); err == nil {
		t.Fatal(
			"exported anonymous field accepted package-specific identity",
		)
	}
	mutated = localInterface.Spec()
	mutated.Methods[0].Package = local.ID()
	if _, err := semantic.NewType(mutated); err == nil {
		t.Fatal(
			"exported anonymous method accepted package-specific identity",
		)
	}
}

func aggregateTypeWithMember(
	t *testing.T,
	pkg semantic.Package,
	kind semantic.TypeKind,
	name string,
) semantic.Type {
	t.Helper()
	for _, record := range semanticTypes(pkg) {
		if record.Kind() != kind {
			continue
		}
		spec := record.Spec()
		switch kind {
		case semantic.TypeStruct:
			if len(spec.Fields) == 1 &&
				spec.Fields[0].Name == name {
				return record
			}
		case semantic.TypeInterface:
			if len(spec.Methods) == 1 &&
				spec.Methods[0].Name == name {
				return record
			}
		}
	}
	t.Fatalf(
		"package %s has no %s type with member %s",
		pkg.ID(), kind, name,
	)
	return semantic.Type{}
}

func assertStructuralMemberOccurrences(
	t *testing.T,
	pkg semantic.Package,
	owner identity.SemanticTypeID,
	name string,
	want int,
) {
	t.Helper()
	var count int
	for _, resolution := range semanticResolutions(pkg) {
		if resolution.Kind() != semantic.ResolutionDeclaration ||
			resolution.Role() != catalog.RoleDeclarationName {
			continue
		}
		declaration := resolution.Declaration()
		if declaration.Form() ==
			identity.SemanticDeclarationMember &&
			declaration.OwnerType() == owner &&
			declaration.Name() == name {
			count++
		}
	}
	if count != want {
		t.Fatalf(
			"package %s member %s/%s occurrences=%d, want %d",
			pkg.ID(), owner, name, count, want,
		)
	}
}

func assertMemberTarget(
	t *testing.T,
	pkg semantic.Package,
	owner identity.SemanticTypeID,
	class identity.SemanticObjectClass,
	name string,
) {
	t.Helper()
	id, err := identity.NewMemberDeclarationID(
		owner, identity.PackageID{}, class, name, 0,
	)
	if err != nil {
		t.Fatal(err)
	}
	target, present := pkg.ResolveDeclarationTarget(id)
	if !present ||
		target.ID() != id ||
		target.OwnerType() != owner {
		t.Fatalf(
			"package %s member target %s=%+v",
			pkg.ID(), id, target,
		)
	}
}
