package certify

import (
	"go/token"
	"go/types"
	"slices"
	"strings"
	"testing"

	"github.com/tsoniclang/gotots/internal/contracts/gostdlib"
)

func TestGenericKernelValueArityJoinsCapabilitiesAndGoParameters(t *testing.T) {
	parameter := types.NewTypeParam(
		types.NewTypeName(token.NoPos, nil, "Element", nil),
		types.Universe.Lookup("any").Type(),
	)
	signature := types.NewSignatureType(
		nil,
		nil,
		[]*types.TypeParam{parameter},
		types.NewTuple(
			types.NewVar(token.NoPos, nil, "left", parameter),
			types.NewVar(token.NoPos, nil, "right", parameter),
		),
		types.NewTuple(types.NewVar(token.NoPos, nil, "result", parameter)),
		false,
	)
	projection := []gostdlib.GenericTypeArgumentDocument{{
		TypeParameter: 0,
		Facet:         gostdlib.GenericTypeArgumentLogical,
	}}
	operations := []gostdlib.GenericOperationDocument{{
		Kind: gostdlib.GenericOperationEqual,
	}}
	if err := verifyGenericKernelShape(
		"example.com/source|Equal",
		signature,
		projection,
		operations,
		1,
		3,
	); err != nil {
		t.Fatal(err)
	}
	err := verifyGenericKernelShape(
		"example.com/source|Equal",
		signature,
		projection,
		operations,
		1,
		4,
	)
	if err == nil || !strings.Contains(
		err.Error(),
		"kernel has 4 value parameters, capability and source contract requires 3",
	) {
		t.Fatalf("hidden kernel-parameter mutation error = %v", err)
	}
}

func TestGenericKernelCallableContractJoinsPublicBinding(t *testing.T) {
	identity := "slices|kind=4|receiver=|name=EqualFunc"
	parameters := []gostdlib.ProviderCallableParameterDocument{{
		Parameter: 2,
		Effect:    gostdlib.EffectAwaitable,
	}}
	binding := gostdlib.BindingDocument{
		Kind:               gostdlib.BindingFunction,
		Effect:             gostdlib.EffectAsynchronous,
		CallableParameters: parameters,
	}
	if err := verifyGenericKernelCallableContract(
		identity,
		binding,
		gostdlib.EffectAsynchronous,
		parameters,
	); err != nil {
		t.Fatal(err)
	}
	if err := verifyGenericKernelCallableContract(
		identity,
		binding,
		gostdlib.EffectSynchronous,
		parameters,
	); err == nil || !strings.Contains(err.Error(), "kernel effect") {
		t.Fatalf("effect mutation error = %v", err)
	}
	mutated := slices.Clone(parameters)
	mutated[0].Effect = gostdlib.EffectSynchronous
	if err := verifyGenericKernelCallableContract(
		identity,
		binding,
		gostdlib.EffectAsynchronous,
		mutated,
	); err == nil || !strings.Contains(err.Error(), "callable parameters") {
		t.Fatalf("callback mutation error = %v", err)
	}
}

func TestSynchronousGenericKernelCallableContractNarrowsExactly(t *testing.T) {
	identity := "slices|kind=4|receiver=|name=SortFunc"
	binding := gostdlib.BindingDocument{
		Kind:   gostdlib.BindingFunction,
		Effect: gostdlib.EffectAsynchronous,
		CallableParameters: []gostdlib.ProviderCallableParameterDocument{{
			Parameter: 1,
			Effect:    gostdlib.EffectAwaitable,
		}},
	}
	parameters := []gostdlib.ProviderCallableParameterDocument{{
		Parameter: 1,
		Effect:    gostdlib.EffectSynchronous,
	}}
	if err := verifySynchronousGenericKernelCallableContract(
		identity,
		binding,
		gostdlib.EffectSynchronous,
		parameters,
	); err != nil {
		t.Fatal(err)
	}
	wrongIndex := slices.Clone(parameters)
	wrongIndex[0].Parameter = 0
	if err := verifySynchronousGenericKernelCallableContract(
		identity,
		binding,
		gostdlib.EffectSynchronous,
		wrongIndex,
	); err == nil || !strings.Contains(err.Error(), "exactly narrow") {
		t.Fatalf("callback-index mutation error = %v", err)
	}
	if err := verifySynchronousGenericKernelCallableContract(
		identity,
		binding,
		gostdlib.EffectAsynchronous,
		parameters,
	); err == nil || !strings.Contains(err.Error(), "public effect") {
		t.Fatalf("outer-effect mutation error = %v", err)
	}
}

func TestSynchronousGenericKernelPairJoinsCanonicalProjection(t *testing.T) {
	identity := "slices|kind=4|receiver=|name=SortFunc"
	canonical := synchronousPairFacet(
		identity,
		gostdlib.FacetCapabilityKernel,
		gostdlib.EffectAsynchronous,
		gostdlib.EffectAwaitable,
	)
	narrowed := synchronousPairFacet(
		identity,
		gostdlib.FacetCapabilitySynchronousKernel,
		gostdlib.EffectSynchronous,
		gostdlib.EffectSynchronous,
	)
	modules := []gostdlib.FacetModuleDocument{{
		Facets: []gostdlib.FacetDocument{canonical, narrowed},
	}}
	if err := verifySynchronousGenericKernelPairs(modules); err != nil {
		t.Fatal(err)
	}

	withoutCanonical := []gostdlib.FacetModuleDocument{{
		Facets: []gostdlib.FacetDocument{narrowed},
	}}
	if err := verifySynchronousGenericKernelPairs(withoutCanonical); err == nil ||
		!strings.Contains(err.Error(), "canonical kernel is absent") {
		t.Fatalf("missing canonical mutation error = %v", err)
	}

	wrongProjection := narrowed
	wrongProjection.GenericTypeArguments = []gostdlib.GenericTypeArgumentDocument{{
		TypeParameter: 1,
		Facet:         gostdlib.GenericTypeArgumentLogical,
	}}
	modules[0].Facets[1] = wrongProjection
	if err := verifySynchronousGenericKernelPairs(modules); err == nil ||
		!strings.Contains(err.Error(), "projections differ") {
		t.Fatalf("projection mutation error = %v", err)
	}

	wrongIndex := narrowed
	wrongIndex.CallableParameters = []gostdlib.ProviderCallableParameterDocument{{
		Parameter: 0,
		Effect:    gostdlib.EffectSynchronous,
	}}
	modules[0].Facets[1] = wrongIndex
	if err := verifySynchronousGenericKernelPairs(modules); err == nil ||
		!strings.Contains(err.Error(), "callback indexes differ") {
		t.Fatalf("callback-index mutation error = %v", err)
	}
}

func synchronousPairFacet(
	identity string,
	capability gostdlib.FacetCapability,
	effect gostdlib.EffectKind,
	callbackEffect gostdlib.EffectKind,
) gostdlib.FacetDocument {
	return gostdlib.FacetDocument{
		Kind:           gostdlib.FacetGenericCallableKernel,
		SourceIdentity: identity,
		Capabilities:   []gostdlib.FacetCapability{capability},
		Effect:         effect,
		GenericTypeArguments: []gostdlib.GenericTypeArgumentDocument{{
			TypeParameter: 0,
			Facet:         gostdlib.GenericTypeArgumentLogical,
		}},
		CallableParameters: []gostdlib.ProviderCallableParameterDocument{{
			Parameter: 1,
			Effect:    callbackEffect,
		}},
	}
}
