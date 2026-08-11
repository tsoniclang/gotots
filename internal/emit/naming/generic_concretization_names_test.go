package naming

import (
	"go/token"
	"go/types"
	"strings"
	"testing"

	"github.com/tsoniclang/gotots/internal/emit/api"
)

func TestGenericConcretizationInterningUsesSemanticIdentity(t *testing.T) {
	constraint := types.NewInterfaceType(nil, nil).Complete()
	typeParameter := types.NewTypeParam(
		types.NewTypeName(token.NoPos, nil, "T", nil),
		constraint,
	)
	parameter := types.NewVar(token.NoPos, nil, "value", typeParameter)
	signature := types.NewSignatureType(
		nil,
		nil,
		[]*types.TypeParam{typeParameter},
		types.NewTuple(parameter),
		types.NewTuple(types.NewVar(token.NoPos, nil, "", typeParameter)),
		false,
	)
	owner := types.NewFunc(token.NoPos, types.NewPackage("example.test/p", "p"), "Identity", signature)
	key := strings.Repeat("a", 64)

	first := newGenericConcretizationForTest(
		t, owner, types.Typ[types.Int], key,
		api.GenericConcretizationEffectCanonical,
	)
	equivalent := newGenericConcretizationForTest(
		t, owner, types.Typ[types.Int], key,
		api.GenericConcretizationEffectCanonical,
	)
	conflicting := newGenericConcretizationForTest(
		t, owner, types.Typ[types.String], key,
		api.GenericConcretizationEffectCanonical,
	)
	synchronous := newGenericConcretizationForTest(
		t, owner, types.Typ[types.Int], key,
		api.GenericConcretizationEffectSynchronous,
	)
	if first == equivalent {
		t.Fatal("test requires independently allocated concretizations")
	}

	registry := NewRegistry()
	canonical, err := registry.internGenericConcretization(first)
	if err != nil {
		t.Fatal(err)
	}
	rejoined, err := registry.internGenericConcretization(equivalent)
	if err != nil {
		t.Fatal(err)
	}
	if rejoined.owner != canonical.owner {
		t.Fatal("equivalent concretization created a sibling canonical artifact")
	}
	if _, err := registry.internGenericConcretization(conflicting); err == nil {
		t.Fatal("same-key concretizations with different arguments were joined")
	}
	if _, err := registry.internGenericConcretization(synchronous); err == nil {
		t.Fatal("same-key concretizations with different effects were joined")
	}
}

func newGenericConcretizationForTest(
	t *testing.T,
	owner *types.Func,
	argument types.Type,
	key string,
	effect api.GenericConcretizationEffect,
) *api.GenericConcretization {
	t.Helper()
	instantiated, err := types.Instantiate(
		nil,
		owner.Type(),
		[]types.Type{argument},
		true,
	)
	if err != nil {
		t.Fatal(err)
	}
	concretization, err := api.NewGenericConcretization(
		owner,
		[]types.Type{argument},
		instantiated.(*types.Signature),
		effect,
		key,
		"$concrete_"+key[:20],
		api.GeneratedArtifactPlacementCompilation,
		api.ArtifactOwner{},
		nil,
	)
	if err != nil {
		t.Fatal(err)
	}
	return concretization
}
