package identity

import (
	"strings"
	"testing"
)

func TestSemanticIdentitiesAreConstructorValidated(t *testing.T) {
	digest := strings.Repeat("ab", 32)
	typeID, err := NewSemanticTypeID(digest)
	if err != nil {
		t.Fatal(err)
	}
	if typeID.Digest() != digest ||
		typeID.String() != "semantic-type/sha256:"+digest {
		t.Fatalf("unexpected type identity %s", typeID)
	}
	if _, err := NewSemanticTypeID(strings.ToUpper(digest)); err == nil {
		t.Fatal("uppercase semantic type digest was accepted")
	}

	module, err := NewModuleID("example.com/model", "v1.2.3")
	if err != nil {
		t.Fatal(err)
	}
	owner, err := NewModuleOwner(module)
	if err != nil {
		t.Fatal(err)
	}
	pkg, err := NewPackageID(owner, "example.com/model/data")
	if err != nil {
		t.Fatal(err)
	}
	declaration, err := NewPackageDeclarationID(
		pkg, SemanticObjectFunction, "Build",
	)
	if err != nil {
		t.Fatal(err)
	}
	if declaration.Package() != pkg ||
		declaration.Class() != SemanticObjectFunction ||
		declaration.Name() != "Build" {
		t.Fatalf("unexpected declaration identity %s", declaration)
	}
	if _, err := NewPackageDeclarationID(
		pkg, SemanticObjectMethod, "Build",
	); err == nil {
		t.Fatal("method was accepted as a package declaration")
	}
	member, err := NewMemberDeclarationID(
		typeID, SemanticObjectField, "Value", 2,
	)
	if err != nil {
		t.Fatal(err)
	}
	if member.OwnerType() != typeID || member.Ordinal() != 2 {
		t.Fatalf("unexpected member identity %s", member)
	}
}

func TestSemanticBindingAndOperationIdentityUseCanonicalAnchors(
	t *testing.T,
) {
	module, _ := NewModuleID("example.com/model", "")
	owner, _ := NewModuleOwner(module)
	file, _ := NewFileID(owner, "pkg/model.go")
	ownerSpan, _ := NewSpanID(file, 10, 90)
	ownerOccurrence, _ := NewOccurrenceID(ownerSpan, 47)
	bindingSpan, _ := NewSpanID(file, 30, 35)
	bindingOccurrence, _ := NewOccurrenceID(bindingSpan, 2)
	binding, err := NewSemanticBindingID(
		ownerOccurrence,
		bindingOccurrence,
		SemanticBindingParameter,
		1,
	)
	if err != nil {
		t.Fatal(err)
	}
	if binding.Owner() != ownerOccurrence ||
		binding.Declaration() != bindingOccurrence ||
		binding.Role() != SemanticBindingParameter ||
		binding.Ordinal() != 1 {
		t.Fatalf("unexpected binding identity %s", binding)
	}
	definition, _ := NewSourceDefinitionID(
		ownerOccurrence, DefinitionFuncDecl,
	)
	operation, err := NewOperationID(
		definition, bindingOccurrence,
	)
	if err != nil {
		t.Fatal(err)
	}
	if operation.Definition() != definition ||
		operation.Occurrence() != bindingOccurrence {
		t.Fatalf("unexpected operation identity %s", operation)
	}
}
