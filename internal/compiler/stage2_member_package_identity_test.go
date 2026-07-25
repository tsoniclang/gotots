package compiler

import (
	"strings"
	"testing"

	"github.com/tsoniclang/gotots/internal/identity"
	"github.com/tsoniclang/gotots/internal/language/semantic"
)

func TestUnexportedMembersRetainDeclaringPackageIdentity(
	t *testing.T,
) {
	ownerType, err := identity.NewSemanticTypeID(
		strings.Repeat("a", 64),
	)
	if err != nil {
		t.Fatal(err)
	}
	signature, err := identity.NewSemanticTypeID(
		strings.Repeat("b", 64),
	)
	if err != nil {
		t.Fatal(err)
	}
	left := mutationPackageID(t, "example.com/member-left")
	right := mutationPackageID(t, "example.com/member-right")
	leftMember, err := identity.NewMemberDeclarationID(
		ownerType,
		left,
		identity.SemanticObjectMethod,
		"hidden",
		0,
	)
	if err != nil {
		t.Fatal(err)
	}
	rightMember, err := identity.NewMemberDeclarationID(
		ownerType,
		right,
		identity.SemanticObjectMethod,
		"hidden",
		0,
	)
	if err != nil {
		t.Fatal(err)
	}
	if leftMember == rightMember {
		t.Fatalf(
			"same-spelled unexported members collapsed: %s",
			leftMember,
		)
	}
	interfaceType, err := semantic.NewType(semantic.TypeSpec{
		Kind:    semantic.TypeInterface,
		TypeSet: semantic.TypeSetUniverse,
		Methods: []semantic.TypeMethod{
			{
				Name: "hidden", Package: left,
				Signature: signature, Ordinal: 0,
			},
			{
				Name: "hidden", Package: right,
				Signature: signature, Ordinal: 1,
			},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	mutated := interfaceType.Spec()
	mutated.Methods[0].Package = identity.PackageID{}
	if _, err := semantic.NewType(mutated); err == nil ||
		!strings.Contains(err.Error(), "method 0 is not canonical") {
		t.Fatalf(
			"dropped declaring package mutation error = %v",
			err,
		)
	}
}

func mutationPackageID(
	t *testing.T,
	path string,
) identity.PackageID {
	t.Helper()
	module, err := identity.NewModuleID(path, "")
	if err != nil {
		t.Fatal(err)
	}
	owner, err := identity.NewModuleOwner(module)
	if err != nil {
		t.Fatal(err)
	}
	pkg, err := identity.NewPackageID(owner, path)
	if err != nil {
		t.Fatal(err)
	}
	return pkg
}
