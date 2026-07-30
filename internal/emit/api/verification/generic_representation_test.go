package api_test

import (
	"go/token"
	"go/types"
	"testing"

	. "github.com/tsoniclang/gotots/internal/emit/api"
)

func TestGenericRepresentationProfileIsExactOrderedAndIdentityKeyed(
	t *testing.T,
) {
	owner, first, second := genericRepresentationOwner("Pair")
	requirements := make([]DeclarationRequirement, 0, 3)
	for _, selection := range []struct {
		parameter *types.TypeParam
		facet     GenericRepresentationFacet
	}{
		{parameter: second, facet: GenericRepresentationPointer},
		{parameter: first, facet: GenericRepresentationPointer},
		{parameter: first, facet: GenericRepresentationStorage},
	} {
		requirement, err := NewGenericRepresentationRequirement(
			owner,
			selection.parameter,
			selection.facet,
		)
		if err != nil {
			t.Fatal(err)
		}
		requirements = append(requirements, requirement)
	}
	profile, err := SelectGenericRepresentationProfile(owner, requirements)
	if err != nil {
		t.Fatal(err)
	}
	if !profile.Valid() ||
		len(profile.Parameters()) != 2 ||
		!profile.Requires(first, GenericRepresentationStorage) ||
		!profile.Requires(first, GenericRepresentationPointer) ||
		profile.Requires(second, GenericRepresentationStorage) ||
		!profile.Requires(second, GenericRepresentationPointer) {
		t.Fatalf("generic representation profile = %#v", profile)
	}
	ordered := profile.OrderedFacets()
	if len(ordered) != 3 ||
		ordered[0].Parameter() != first ||
		ordered[0].Facet() != GenericRepresentationStorage ||
		ordered[1].Parameter() != first ||
		ordered[1].Facet() != GenericRepresentationPointer ||
		ordered[2].Parameter() != second ||
		ordered[2].Facet() != GenericRepresentationPointer {
		t.Fatalf("ordered generic representation facets = %#v", ordered)
	}

	foreignOwner, foreign, _ := genericRepresentationOwner("Foreign")
	if _, err := NewGenericRepresentationRequirement(
		owner,
		foreign,
		GenericRepresentationPointer,
	); err == nil {
		t.Fatal("same-spelling foreign type parameter acquired a facet")
	}
	if _, err := SelectGenericRepresentationProfile(
		foreignOwner,
		requirements,
	); err == nil {
		t.Fatal("foreign owner accepted another declaration's facet set")
	}
}

func TestGenericRepresentationRequestCarriesExactFacetIdentity(t *testing.T) {
	owner, parameter, _ := genericRepresentationOwner("Box")
	request, err := NewGenericRepresentationRequest(
		owner,
		parameter,
		GenericRepresentationPointer,
	)
	if err != nil {
		t.Fatal(err)
	}
	requirement, ok := request.DeclarationRequirement()
	selectedOwner, selectedParameter, facet, selected :=
		requirement.GenericRepresentation()
	if !ok ||
		!selected ||
		selectedOwner != owner ||
		selectedParameter != parameter ||
		facet != GenericRepresentationPointer ||
		requirement.Kind() != DeclarationRequirementGenericRepresentation {
		t.Fatalf("generic representation request = %#v", request)
	}
}

func TestPointerRepresentationSelectionRequiresCanonicalDefinitionAndDemand(
	t *testing.T,
) {
	sourcePackage := types.NewPackage(
		"example.com/pointercontract",
		"pointercontract",
	)
	object := types.NewTypeName(1, sourcePackage, "Box", nil)
	named := types.NewNamed(
		object,
		types.NewStruct(
			[]*types.Var{
				types.NewField(
					2,
					sourcePackage,
					"Value",
					types.Typ[types.Int32],
					false,
				),
			},
			nil,
		),
		nil,
	)
	artifact, err := NewContractGeneratedArtifact(
		GeneratedArtifactPointerRepresentation,
		types.NewPointer(named),
		"pointer-contract",
		"$pointer",
	)
	if err != nil {
		t.Fatal(err)
	}
	definition, err := NewPointerRepresentationRequirement(artifact, false)
	if err != nil {
		t.Fatal(err)
	}
	demand, err := NewPointerRepresentationRequirement(artifact, true)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := SelectPointerRepresentation(artifact, nil); err == nil {
		t.Fatal("missing pointer definition was accepted")
	}
	if _, err := SelectPointerRepresentation(
		artifact,
		[]DeclarationRequirement{definition, definition},
	); err == nil {
		t.Fatal("duplicate pointer definition was accepted")
	}
	selected, err := SelectPointerRepresentation(
		artifact,
		[]DeclarationRequirement{definition, demand},
	)
	if err != nil {
		t.Fatal(err)
	}
	if selected != PointerRepresentationCarrierCanonical {
		t.Fatalf("demanded pointer representation = %v", selected)
	}
}

func genericRepresentationOwner(
	name string,
) (*types.Func, *types.TypeParam, *types.TypeParam) {
	constraint := types.NewInterfaceType(nil, nil).Complete()
	first := types.NewTypeParam(
		types.NewTypeName(token.NoPos, nil, "T", nil),
		constraint,
	)
	second := types.NewTypeParam(
		types.NewTypeName(token.NoPos, nil, "U", nil),
		constraint,
	)
	owner := types.NewFunc(
		token.Pos(1),
		types.NewPackage("example.com/"+name, name),
		name,
		types.NewSignatureType(
			nil,
			nil,
			[]*types.TypeParam{first, second},
			types.NewTuple(),
			types.NewTuple(),
			false,
		),
	)
	return owner, first, second
}
