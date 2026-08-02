package api_test

import (
	"go/token"
	"go/types"
	"testing"

	. "github.com/tsoniclang/gotots/internal/emit/api"
)

func TestTypeRepresentationFacetIdentityAndRequestOwnership(t *testing.T) {
	if TypeRepresentationStorage != 1 ||
		TypeRepresentationContainerStorage != 2 ||
		TypeRepresentationPointer != 3 {
		t.Fatal("type-representation facet IDs changed")
	}
	owner := representedStructType("Record")
	for _, facet := range []TypeRepresentationFacet{
		TypeRepresentationStorage,
		TypeRepresentationContainerStorage,
		TypeRepresentationPointer,
	} {
		request, err := NewTypeRepresentationRequest(owner, facet)
		if err != nil {
			t.Fatal(err)
		}
		requirement, ok := request.DeclarationRequirement()
		selected, generated, selectedFacet, represented :=
			requirement.TypeRepresentation()
		if !ok || !represented || selected != owner || generated != nil ||
			selectedFacet != facet ||
			requirement.Kind() != DeclarationRequirementTypeRepresentation {
			t.Fatalf("type-representation request = %#v", request)
		}
	}
}

func TestTypeRepresentationRejectsAliasesAndInterfaces(t *testing.T) {
	sourcePackage := types.NewPackage("example.com/rejected", "rejected")
	alias := types.NewAlias(
		types.NewTypeName(token.NoPos, sourcePackage, "Alias", nil),
		types.Typ[types.Int32],
	)
	if SupportsTypeRepresentation(alias.Obj()) {
		t.Fatal("alias acquired a materialized type representation")
	}
	interfaceObject := types.NewTypeName(
		token.NoPos,
		sourcePackage,
		"Contract",
		nil,
	)
	types.NewNamed(
		interfaceObject,
		types.NewInterfaceType(nil, nil).Complete(),
		nil,
	)
	if SupportsTypeRepresentation(interfaceObject) {
		t.Fatal("interface acquired a concrete type representation")
	}
	if _, err := NewTypeRepresentationRequest(
		interfaceObject,
		TypeRepresentationPointer,
	); err == nil {
		t.Fatal("unsupported owner produced a type-representation request")
	}
}

func representedStructType(name string) *types.TypeName {
	sourcePackage := types.NewPackage("example.com/"+name, name)
	owner := types.NewTypeName(token.NoPos, sourcePackage, name, nil)
	types.NewNamed(owner, types.NewStruct(nil, nil), nil)
	return owner
}
