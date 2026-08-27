package providerboundary

import (
	"go/types"
	"slices"

	"github.com/tsoniclang/gotots/internal/contracts/gostdlib"
	gostdlibsource "github.com/tsoniclang/gotots/internal/contracts/gostdlib/sourcecontract"
	"github.com/tsoniclang/gotots/internal/emit/api"
)

func profileRootsAccept(
	certified []int,
	selected []int,
	required bool,
) bool {
	if required {
		return slices.Equal(certified, selected)
	}
	for _, root := range selected {
		if _, found := slices.BinarySearch(certified, root); !found {
			return false
		}
	}
	return true
}

func identitySetContains(
	certified map[string]struct{},
	selected map[string]struct{},
) bool {
	for identity := range selected {
		if _, found := certified[identity]; !found {
			return false
		}
	}
	return true
}

func profileCallableParameterRoots(signature *types.Signature) []int {
	if signature == nil || signature.Params() == nil {
		return nil
	}
	var result []int
	for index := range signature.Params().Len() {
		if _, callable := gostdlibsource.DirectCallableParameterSignature(
			signature.Params().At(index).Type(),
		); callable {
			result = append(result, index)
		}
	}
	return result
}

func profileCallableParameterBoundary(
	context api.Context,
	signature *types.Signature,
	evidence []gostdlib.ProviderCallableParameterDocument,
) ([]int, []int, error) {
	roots := profileCallableParameterRoots(signature)
	if len(roots) != len(evidence) {
		return nil, nil, boundaryInvariant(
			context,
			"provider callable-parameter evidence does not exact-join the source signature",
		)
	}
	expected := gostdlib.EffectSynchronous
	var mismatched []int
	for index, root := range roots {
		selected := evidence[index]
		if selected.Parameter != root || !selected.Effect.Valid() {
			return nil, nil, boundaryInvariant(
				context,
				"provider callable-parameter evidence diverged from the source signature",
			)
		}
		if !providerCallableEffectAccepts(selected.Effect, expected) {
			mismatched = append(mismatched, root)
		}
	}
	return roots, mismatched, nil
}

func profileCallablesAccept(
	context api.Context,
	signature *types.Signature,
	canonicalParameters []int,
	callables []gostdlib.ProviderCallableParameterDocument,
) bool {
	var roots []int
	for _, parameter := range canonicalParameters {
		if _, callable := gostdlibsource.DirectCallableParameterSignature(
			signature.Params().At(parameter).Type(),
		); callable {
			roots = append(roots, parameter)
		}
	}
	if len(roots) != len(callables) {
		return false
	}
	expected := gostdlib.EffectSynchronous
	for index, root := range roots {
		if callables[index].Parameter != root ||
			!providerCallableEffectAccepts(callables[index].Effect, expected) {
			return false
		}
	}
	return true
}

func profileKeyCallables(
	source []gostdlib.ProviderCallableParameterDocument,
) []gostdlib.ProviderCallableProfileKeyCallable {
	result := make(
		[]gostdlib.ProviderCallableProfileKeyCallable,
		len(source),
	)
	for index, selected := range source {
		result[index] = gostdlib.ProviderCallableProfileKeyCallable{
			Parameter: selected.Parameter,
			Effect:    selected.Effect,
		}
	}
	return result
}

func providerCallableEffectAccepts(
	provider gostdlib.EffectKind,
	generated gostdlib.EffectKind,
) bool {
	return provider == generated
}

func rootsContainingIdentities(
	roots []map[string]struct{},
	identities map[string]struct{},
) []int {
	result := make([]int, 0, len(roots))
	for index, root := range roots {
		for identity := range identities {
			if _, found := root[identity]; found {
				result = append(result, index)
				break
			}
		}
	}
	return result
}

func cloneIdentitySet(source map[string]struct{}) map[string]struct{} {
	result := make(map[string]struct{}, len(source))
	for identity := range source {
		result[identity] = struct{}{}
	}
	return result
}
