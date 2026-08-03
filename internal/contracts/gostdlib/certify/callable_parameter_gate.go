package certify

import (
	"fmt"
	"go/types"
	"slices"
	"sort"
	"strings"

	"github.com/tsoniclang/gotots/internal/contracts/gostdlib"
	gostdlibsource "github.com/tsoniclang/gotots/internal/contracts/gostdlib/sourcecontract"
)

func verifyCallableParameterBindings(
	source goSurface,
	modules []gostdlib.ModuleDocument,
) error {
	var mismatches []string
	for _, module := range modules {
		for _, binding := range module.Bindings {
			if binding.Kind != gostdlib.BindingFunction {
				continue
			}
			evidence, ok := source.objects[binding.Identity]
			if !ok {
				mismatches = append(
					mismatches,
					binding.Identity+": selected Go callable is absent",
				)
				continue
			}
			signature, ok := evidence.object.Type().(*types.Signature)
			if !ok {
				mismatches = append(
					mismatches,
					binding.Identity+": selected Go signature is absent",
				)
				continue
			}
			mismatches = append(
				mismatches,
				callableParameterBindingMismatches(
					binding.Identity,
					signature,
					binding.CallableParameters,
				)...,
			)
		}
	}
	if len(mismatches) == 0 {
		return nil
	}
	sort.Strings(mismatches)
	return certifyError(
		"verify callable parameter bindings",
		"provider surface",
		strings.Join(mismatches, "; "),
	)
}

func callableParameterBindingMismatches(
	identity string,
	signature *types.Signature,
	actual []gostdlib.ProviderCallableParameterDocument,
) []string {
	if signature == nil || signature.Params() == nil {
		if len(actual) == 0 {
			return nil
		}
		return []string{identity + ": callable evidence has no source parameters"}
	}
	var expected []int
	var mismatches []string
	for index := range signature.Params().Len() {
		callable, direct := gostdlibsource.DirectCallableParameterSignature(
			signature.Params().At(index).Type(),
		)
		if direct {
			expected = append(expected, index)
			if callableContainsNestedCallable(callable) {
				mismatches = append(mismatches, fmt.Sprintf(
					"%s parameter %d: nested callable transport is unsupported",
					identity,
					index,
				))
			}
		}
	}
	actualIndexes := make([]int, len(actual))
	for index, selected := range actual {
		actualIndexes[index] = selected.Parameter
	}
	if !slices.Equal(expected, actualIndexes) {
		mismatches = append(mismatches, fmt.Sprintf(
			"%s: direct callable parameters %v do not exact-join provider evidence %v",
			identity,
			expected,
			actualIndexes,
		))
	}
	for _, selected := range actual {
		if sourceCallableTypeParameterCount(signature) == 0 &&
			selected.Effect != gostdlib.EffectSynchronous &&
			selected.Effect != gostdlib.EffectAwaitable {
			mismatches = append(mismatches, fmt.Sprintf(
				"%s parameter %d: ordinary provider effect is %s, want sync or awaitable",
				identity,
				selected.Parameter,
				selected.Effect,
			))
		}
	}
	return mismatches
}

func callableContainsNestedCallable(signature *types.Signature) bool {
	if signature == nil {
		return false
	}
	visited := make(map[types.Type]struct{})
	for _, tuple := range []*types.Tuple{signature.Params(), signature.Results()} {
		if tuple == nil {
			continue
		}
		for index := range tuple.Len() {
			if typeContainsCallableValue(tuple.At(index).Type(), visited) {
				return true
			}
		}
	}
	return false
}

func typeContainsCallableValue(
	source types.Type,
	visited map[types.Type]struct{},
) bool {
	if source == nil {
		return false
	}
	source = types.Unalias(source)
	if _, seen := visited[source]; seen {
		return false
	}
	visited[source] = struct{}{}
	switch selected := source.(type) {
	case *types.Signature:
		return true
	case *types.Named:
		if _, callable := selected.Underlying().(*types.Signature); callable {
			return true
		}
		if _, contract := selected.Underlying().(*types.Interface); contract {
			return false
		}
		return typeContainsCallableValue(selected.Underlying(), visited)
	case *types.Struct:
		for index := range selected.NumFields() {
			if typeContainsCallableValue(selected.Field(index).Type(), visited) {
				return true
			}
		}
	case *types.Pointer:
		return typeContainsCallableValue(selected.Elem(), visited)
	case *types.Slice:
		return typeContainsCallableValue(selected.Elem(), visited)
	case *types.Array:
		return typeContainsCallableValue(selected.Elem(), visited)
	case *types.Map:
		return typeContainsCallableValue(selected.Key(), visited) ||
			typeContainsCallableValue(selected.Elem(), visited)
	case *types.Chan:
		return typeContainsCallableValue(selected.Elem(), visited)
	case *types.Tuple:
		for index := range selected.Len() {
			if typeContainsCallableValue(selected.At(index).Type(), visited) {
				return true
			}
		}
	}
	return false
}

func verifyCallableParameterProfileCoverage(
	source goSurface,
	modules []gostdlib.ModuleDocument,
	facetModules []gostdlib.FacetModuleDocument,
) error {
	expected := make(map[string]int)
	for _, module := range modules {
		for _, binding := range module.Bindings {
			if binding.Kind != gostdlib.BindingFunction ||
				len(binding.CallableParameters) == 0 {
				continue
			}
			evidence, ok := source.objects[binding.Identity]
			if !ok {
				continue
			}
			signature, ok := evidence.object.Type().(*types.Signature)
			if !ok || sourceCallableTypeParameterCount(signature) != 0 {
				continue
			}
			var cooperative []gostdlib.ProviderCallableParameterDocument
			for _, selected := range binding.CallableParameters {
				if selected.Effect == gostdlib.EffectSynchronous {
					cooperative = append(
						cooperative,
						gostdlib.ProviderCallableParameterDocument{
							Parameter: selected.Parameter,
							Effect:    gostdlib.EffectAwaitable,
						},
					)
				}
			}
			if len(cooperative) != 0 {
				expected[callableParameterProfileObligation(
					binding.Identity,
					cooperative,
				)]++
			}
		}
	}
	actual := make(map[string]int)
	for _, module := range facetModules {
		for _, profile := range module.CallableProfiles {
			if len(profile.CallableParameters) == 0 {
				continue
			}
			actual[callableParameterProfileObligation(
				profile.SourceIdentity,
				profile.CallableParameters,
			)]++
		}
	}
	var differences []string
	for key, count := range expected {
		if actual[key] != count {
			differences = append(differences, fmt.Sprintf(
				"%s expected %d profile(s), certified %d",
				key,
				count,
				actual[key],
			))
		}
	}
	for key, count := range actual {
		if expected[key] != count {
			differences = append(differences, fmt.Sprintf(
				"%s certified %d profile(s), expected %d",
				key,
				count,
				expected[key],
			))
		}
	}
	if len(differences) == 0 {
		return nil
	}
	sort.Strings(differences)
	return certifyError(
		"verify callable parameter profile coverage",
		"provider surface",
		strings.Join(differences, "; "),
	)
}

func callableParameterProfileObligation(
	identity string,
	parameters []gostdlib.ProviderCallableParameterDocument,
) string {
	var result strings.Builder
	result.WriteString(identity)
	for _, selected := range parameters {
		fmt.Fprintf(
			&result,
			"|parameter=%d|effect=%s",
			selected.Parameter,
			selected.Effect,
		)
	}
	return result.String()
}
