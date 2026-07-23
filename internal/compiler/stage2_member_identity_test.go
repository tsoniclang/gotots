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

type Box[T any] struct {
	Value T
}

func (box Box[T]) Load() T {
	return box.Value
}

func Read(
	a A,
	b B,
	ints Box[int],
	strings Box[string],
) int {
	_ = strings.Value
	_ = strings.Load()
	return a.X + b.X + ints.Value + ints.Load()
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
	inspection, err := InspectConstructs(source.Request{
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
	for _, operation := range pkg.Operations() {
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
	if len(selected["X"]) != 2 {
		t.Fatalf(
			"defined owners A and B produced X identities %v",
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
	declarationCounts := map[identity.SemanticDeclarationID]int{}
	for _, declaration := range pkg.Declarations() {
		declarationCounts[declaration.ID()]++
	}
	for name, identities := range selected {
		for declaration := range identities {
			if declaration.Form() !=
				identity.SemanticDeclarationMember ||
				declaration.OwnerType().IsZero() ||
				declarationCounts[declaration] != 1 {
				t.Errorf(
					"selected %s declaration %s has owner=%s records=%d",
					name,
					declaration,
					declaration.OwnerType(),
					declarationCounts[declaration],
				)
			}
		}
	}
	if len(pkg.Unsupported()) != 0 {
		t.Fatalf(
			"member identity fixture has %d unsupported records",
			len(pkg.Unsupported()),
		)
	}
}
