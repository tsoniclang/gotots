package compiler

import (
	"testing"

	"github.com/tsoniclang/gotots/internal/identity"
	"github.com/tsoniclang/gotots/internal/scope/contract"
	"github.com/tsoniclang/gotots/internal/source"
)

const stage2MemberIdentityFixture = `package members

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
	inspection, err := inspectConstructsForTest(t, source.Request{
		Dir: directory, Patterns: []string{"."},
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
}
