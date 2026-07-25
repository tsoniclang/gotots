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
		typeID, pkg, SemanticObjectField, "value", 2,
	)
	if err != nil {
		t.Fatal(err)
	}
	if member.OwnerType() != typeID ||
		member.MemberPackage() != pkg ||
		member.Ordinal() != 2 {
		t.Fatalf("unexpected member identity %s", member)
	}
}

func TestSemanticBindingAndOperationIdentityUseCanonicalAnchors(
	t *testing.T,
) {
	module, _ := NewModuleID("example.com/model", "")
	owner, _ := NewModuleOwner(module)
	pkg, _ := NewPackageID(owner, "example.com/model/pkg")
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
	implicitDefinition, _ := NewImplicitDefinitionID(
		pkg, ImplicitDefinitionPackageInit,
	)
	implicit, err := NewImplicitOperationID(
		implicitDefinition,
		ImplicitDefinitionPackageInit,
		0,
	)
	if err != nil {
		t.Fatal(err)
	}
	if implicit.Source() ||
		implicit.ImplicitOp() != ImplicitDefinitionPackageInit ||
		implicit.Ordinal() != 0 {
		t.Fatalf("unexpected implicit operation identity %s", implicit)
	}
}

func TestSemanticIdentityParsersRoundTripCanonicalForms(
	t *testing.T,
) {
	module, _ := NewModuleID("example.com/model", "v1.2.3")
	owner, _ := NewModuleOwner(module)
	pkg, _ := NewPackageID(owner, "example.com/model/pkg")
	file, _ := NewFileID(owner, "pkg/model.go")
	scopeSpan, _ := NewSpanID(file, 10, 100)
	scope, _ := NewOccurrenceID(scopeSpan, 47)
	nameSpan, _ := NewSpanID(file, 20, 25)
	name, _ := NewOccurrenceID(nameSpan, 2)
	definition, _ := NewSourceDefinitionID(
		scope, DefinitionFuncDecl,
	)
	typeID, _ := NewSemanticTypeID(strings.Repeat("ab", 32))
	packageDeclaration, _ := NewPackageDeclarationID(
		pkg, SemanticObjectFunction, "Build",
	)
	exportedMember, _ := NewMemberDeclarationID(
		typeID, PackageID{}, SemanticObjectMethod, "Run", 0,
	)
	privateMember, _ := NewMemberDeclarationID(
		typeID, pkg, SemanticObjectField, "value", 2,
	)
	predeclared, _ := NewPredeclaredDeclarationID(
		1, SemanticObjectType,
	)
	local, _ := NewOccurrenceDeclarationID(
		scope, name, SemanticObjectType, "Local", 0,
	)
	binding, _ := NewSemanticBindingID(
		scope, name, SemanticBindingParameter, 1,
	)
	unnamed, _ := NewSemanticBindingID(
		scope, OccurrenceID{}, SemanticBindingResult, 0,
	)
	operation, _ := NewOperationID(definition, name)
	implicitDefinition, _ := NewImplicitDefinitionID(
		pkg, ImplicitDefinitionPackageInit,
	)
	implicit, _ := NewImplicitOperationID(
		implicitDefinition,
		ImplicitDefinitionPackageInit,
		3,
	)
	unsupported, _ := NewUnsupportedID(definition, name)

	roundTrips := []struct {
		value string
		parse func(string) (string, error)
	}{
		{typeID.String(), func(value string) (string, error) {
			id, err := ParseSemanticTypeID(value)
			return id.String(), err
		}},
	}
	for _, declaration := range []SemanticDeclarationID{
		packageDeclaration,
		exportedMember,
		privateMember,
		predeclared,
		local,
	} {
		roundTrips = append(roundTrips, struct {
			value string
			parse func(string) (string, error)
		}{declaration.String(), func(value string) (string, error) {
			id, err := ParseSemanticDeclarationID(value)
			return id.String(), err
		}})
	}
	for _, candidate := range []SemanticBindingID{binding, unnamed} {
		roundTrips = append(roundTrips, struct {
			value string
			parse func(string) (string, error)
		}{candidate.String(), func(value string) (string, error) {
			id, err := ParseSemanticBindingID(value)
			return id.String(), err
		}})
	}
	for _, candidate := range []OperationID{operation, implicit} {
		roundTrips = append(roundTrips, struct {
			value string
			parse func(string) (string, error)
		}{candidate.String(), func(value string) (string, error) {
			id, err := ParseOperationID(value)
			return id.String(), err
		}})
	}
	roundTrips = append(roundTrips, struct {
		value string
		parse func(string) (string, error)
	}{unsupported.String(), func(value string) (string, error) {
		id, err := ParseUnsupportedID(value)
		return id.String(), err
	}})

	for _, test := range roundTrips {
		got, err := test.parse(test.value)
		if err != nil {
			t.Fatalf("parse %s: %v", test.value, err)
		}
		if got != test.value {
			t.Fatalf("round trip %s => %s", test.value, got)
		}
	}
	if _, err := ParseSemanticDeclarationID(
		strings.Replace(
			packageDeclaration.String(),
			"/function/",
			"/unknown/",
			1,
		),
	); err == nil {
		t.Fatal("noncanonical declaration was accepted")
	}
}
