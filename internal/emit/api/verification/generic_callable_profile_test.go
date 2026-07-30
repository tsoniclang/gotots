package api_test

import (
	"go/ast"
	"go/token"
	"go/types"
	"testing"

	. "github.com/tsoniclang/gotots/internal/emit/api"
)

type fixedCooperativeResolver struct {
	selected map[*GeneratedArtifact]bool
}

func (r fixedCooperativeResolver) ObserveCooperativeCallable(
	_ ArtifactOwner,
	facet CallableFacet,
) (CooperativeCallableObservation, error) {
	artifact, ok := facet.ABI()
	return NewCooperativeCallableObservation(
		ok && r.selected[artifact],
	)
}

func TestGenericCallableProfileCanonicalizesExactABIOverrides(
	t *testing.T,
) {
	owner := genericProfileOwner()
	first := genericProfileABI(t, "first")
	second := genericProfileABI(t, "second")
	firstFalse, err := NewGenericCallableABISelection(first, false)
	if err != nil {
		t.Fatal(err)
	}
	firstTrue, err := NewGenericCallableABISelection(first, true)
	if err != nil {
		t.Fatal(err)
	}
	secondTrue, err := NewGenericCallableABISelection(second, true)
	if err != nil {
		t.Fatal(err)
	}
	selection, err := NewGenericCallableProfileSelection(
		[]GenericCallableABISelection{
			secondTrue,
			firstFalse,
			firstTrue,
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	reversed, err := NewGenericCallableProfileSelection(
		[]GenericCallableABISelection{
			firstTrue,
			secondTrue,
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	if !selection.Valid() ||
		!selection.Cooperative() ||
		selection.Key() != reversed.Key() ||
		len(selection.ABIs()) != 2 {
		t.Fatalf("canonical profile = %#v", selection)
	}
	if cooperative, ok := selection.ABI(first); !ok || !cooperative {
		t.Fatal("duplicate ABI selections did not merge to cooperative")
	}
	profile, err := NewGenericCallableProfile(
		owner,
		selection,
		"$cooperative_exact",
	)
	if err != nil {
		t.Fatal(err)
	}
	facet, err := NewGenericCallableProfileFacet(profile)
	if err != nil {
		t.Fatal(err)
	}
	request, err := NewGenericCallableProfileRequest(profile)
	if err != nil {
		t.Fatal(err)
	}
	requirement, ok := request.DeclarationRequirement()
	selectedProfile, selected := requirement.GenericCallableProfile()
	if !ok ||
		!selected ||
		selectedProfile != profile ||
		facet.Owner() != MustSourceArtifactOwner(owner) {
		t.Fatalf("profile request = %#v", request)
	}
}

func TestGenericCallableProfileOverridesOnlySelectedABI(
	t *testing.T,
) {
	owner := genericProfileOwner()
	selectedABI := genericProfileABI(t, "selected")
	unrelatedABI := genericProfileABI(t, "unrelated")
	override, err := NewGenericCallableABISelection(selectedABI, true)
	if err != nil {
		t.Fatal(err)
	}
	selection, err := NewGenericCallableProfileSelection(
		[]GenericCallableABISelection{override},
	)
	if err != nil {
		t.Fatal(err)
	}
	profile, err := NewGenericCallableProfile(
		owner,
		selection,
		"$cooperative_selected",
	)
	if err != nil {
		t.Fatal(err)
	}
	context := (Context{}).
		WithArtifactOwner(MustSourceArtifactOwner(owner)).
		WithCooperativeCallableResolver(fixedCooperativeResolver{
			selected: map[*GeneratedArtifact]bool{
				unrelatedABI: false,
			},
		}).
		WithGenericCallableProfile(profile)
	selectedReference, err := NewCallableABIReference(selectedABI)
	if err != nil {
		t.Fatal(err)
	}
	selectedFacet, err := context.CallableABIFacet(selectedReference)
	if err != nil {
		t.Fatal(err)
	}
	if selectedFacet.Owner() != MustSourceArtifactOwner(owner) {
		t.Fatal("selected ABI did not acquire generic-profile ownership")
	}
	unrelatedReference, err := NewCallableABIReference(unrelatedABI)
	if err != nil {
		t.Fatal(err)
	}
	unrelatedFacet, err := context.CallableABIFacet(unrelatedReference)
	if err != nil {
		t.Fatal(err)
	}
	if unrelatedFacet.Owner() !=
		MustGeneratedArtifactOwner(unrelatedABI) {
		t.Fatal("unrelated ABI leaked into generic-profile ownership")
	}
	scopedReference, err := NewSourceCallableABIReference(
		owner,
		unrelatedABI,
	)
	if err != nil {
		t.Fatal(err)
	}
	scopedFacet, err := context.CallableABIFacet(scopedReference)
	if err != nil {
		t.Fatal(err)
	}
	if scopedFacet.Owner() != MustSourceArtifactOwner(owner) {
		t.Fatal("declaration-scoped ABI lost generic-profile ownership")
	}
	for artifact, want := range map[*GeneratedArtifact]bool{
		selectedABI:  true,
		unrelatedABI: false,
	} {
		facet, err := NewCallableABIFacet(artifact)
		if err != nil {
			t.Fatal(err)
		}
		observation, err := context.ObserveCooperativeCallable(facet)
		if err != nil {
			t.Fatal(err)
		}
		if observation.Cooperative() != want {
			t.Fatalf(
				"ABI %s cooperative = %t, want %t",
				artifact.ArtifactKey(),
				observation.Cooperative(),
				want,
			)
		}
	}
}

func TestFunctionLiteralFacetIdentityIncludesGenericProfile(
	t *testing.T,
) {
	owner := genericProfileOwner()
	literal := &ast.FuncLit{
		Type: &ast.FuncType{},
		Body: &ast.BlockStmt{},
	}
	profiles := make([]*GenericCallableProfile, 2)
	for index, key := range []string{"first", "second"} {
		abi := genericProfileABI(t, key)
		override, err := NewGenericCallableABISelection(abi, true)
		if err != nil {
			t.Fatal(err)
		}
		selection, err := NewGenericCallableProfileSelection(
			[]GenericCallableABISelection{override},
		)
		if err != nil {
			t.Fatal(err)
		}
		profiles[index], err = NewGenericCallableProfile(
			owner,
			selection,
			"$cooperative_"+key,
		)
		if err != nil {
			t.Fatal(err)
		}
	}
	base := Context{}.WithArtifactOwner(MustSourceArtifactOwner(owner))
	first, err := base.WithGenericCallableProfile(
		profiles[0],
	).FunctionLiteralCallableFacet(literal)
	if err != nil {
		t.Fatal(err)
	}
	second, err := base.WithGenericCallableProfile(
		profiles[1],
	).FunctionLiteralCallableFacet(literal)
	if err != nil {
		t.Fatal(err)
	}
	if first == second {
		t.Fatal("distinct generic profiles collapsed one lexical callable facet")
	}
	if selected, ok := first.FunctionLiteralProfile(); !ok ||
		selected != profiles[0] {
		t.Fatalf("first literal profile = %#v, %t", selected, ok)
	}
}

func genericProfileOwner() *types.Func {
	constraint := types.NewInterfaceType(nil, nil).Complete()
	parameter := types.NewTypeParam(
		types.NewTypeName(token.NoPos, nil, "T", nil),
		constraint,
	)
	return types.NewFunc(
		token.Pos(1),
		types.NewPackage("example.com/profile", "profile"),
		"Apply",
		types.NewSignatureType(
			nil,
			nil,
			[]*types.TypeParam{parameter},
			types.NewTuple(
				types.NewVar(token.NoPos, nil, "value", parameter),
			),
			types.NewTuple(),
			false,
		),
	)
}

func genericProfileABI(
	t *testing.T,
	key string,
) *GeneratedArtifact {
	t.Helper()
	signature := types.NewSignatureType(
		nil,
		nil,
		nil,
		types.NewTuple(),
		types.NewTuple(
			types.NewVar(token.NoPos, nil, "", types.Typ[types.Bool]),
		),
		false,
	)
	artifact, err := NewContractGeneratedArtifact(
		GeneratedArtifactCallableABI,
		signature,
		key,
		"$goCallable_"+key,
	)
	if err != nil {
		t.Fatal(err)
	}
	return artifact
}
