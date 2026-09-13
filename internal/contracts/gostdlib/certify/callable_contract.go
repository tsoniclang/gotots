package certify

import (
	"fmt"
	"github.com/tsoniclang/gotots/internal/contracts/gostdlib"
	gostdlibsource "github.com/tsoniclang/gotots/internal/contracts/gostdlib/sourcecontract"
	"github.com/tsoniclang/gotots/internal/contracts/tsoniccore"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
	"go/types"
	"path/filepath"
	"slices"
	"sort"
	"strings"
)

const (
	callableEffectMarkerPath = "src/internal/certify/callable-effect.ts"
	callableEffectMarkerName = "AsyncEffectMarker"
)

func loadCallableEffectMarker(
	config resolvedConfig,
	project *tsgo.ProjectInspection,
) (tsgo.ProjectExport, error) {
	exports, err := project.Exports(filepath.Join(
		config.providerRoot,
		filepath.FromSlash(callableEffectMarkerPath),
	))
	if err != nil {
		return tsgo.ProjectExport{}, err
	}
	if len(exports) != 1 || exports[0].Name() != callableEffectMarkerName {
		return tsgo.ProjectExport{}, certifyError(
			"inspect callable effect",
			callableEffectMarkerPath,
			"marker export is not exact",
		)
	}
	return exports[0], nil
}

func exportCallableEffect(
	project *tsgo.ProjectInspection,
	target tsgo.ProjectExport,
	marker tsgo.ProjectExport,
) (gostdlib.EffectKind, error) {
	selected, err := project.CallableEffect(target, marker)
	if err != nil {
		return gostdlib.EffectInvalid, err
	}
	return providerEffect(selected)
}

func memberCallableEffect(
	project *tsgo.ProjectInspection,
	target tsgo.ProjectMember,
	marker tsgo.ProjectExport,
) (gostdlib.EffectKind, error) {
	selected, err := project.CallableEffect(target, marker)
	if err != nil {
		return gostdlib.EffectInvalid, err
	}
	return providerEffect(selected)
}

func parameterCallableEffect(
	project *tsgo.ProjectInspection,
	target tsgo.ProjectExport,
	parameter int,
	marker tsgo.ProjectExport,
) (gostdlib.EffectKind, error) {
	selected, err := project.CallableParameterEffect(target, parameter, marker)
	if err != nil {
		return gostdlib.EffectInvalid, err
	}
	return providerEffect(selected)
}

func providerEffect(source tsgo.CallableEffect) (gostdlib.EffectKind, error) {
	switch source {
	case tsgo.CallableEffectSynchronous:
		return gostdlib.EffectSynchronous, nil
	case tsgo.CallableEffectAsynchronous:
		return gostdlib.EffectInvalid, certifyError(
			"inspect callable effect",
			"target",
			"Promise-only callable is not valid in the fixed synchronous execution contract",
		)
	case tsgo.CallableEffectAwaitable:
		return gostdlib.EffectInvalid, certifyError(
			"inspect callable effect",
			"target",
			"T | Promise<T> callable is not valid in the fixed synchronous execution contract",
		)
	default:
		return gostdlib.EffectInvalid, certifyError(
			"inspect callable effect",
			"target",
			"effect is invalid",
		)
	}
}

type callableParameterEffect func(int) (gostdlib.EffectKind, error)

func exportCallableParameters(
	evidence goObject,
	target tsgo.ProjectExport,
	project *tsgo.ProjectInspection,
	effectMarker tsgo.ProjectExport,
) ([]gostdlib.ProviderCallableParameterDocument, error) {
	return callableParameterDocuments(
		evidence,
		gostdlib.AccessExport,
		func(parameter int) (gostdlib.EffectKind, error) {
			return parameterCallableEffect(
				project,
				target,
				parameter,
				effectMarker,
			)
		},
	)
}

func genericKernelCallableParameters(
	evidence goObject,
	target tsgo.ProjectExport,
	capabilityParameters int,
	project *tsgo.ProjectInspection,
	effectMarker tsgo.ProjectExport,
) ([]gostdlib.ProviderCallableParameterDocument, error) {
	return callableParameterDocuments(
		evidence,
		gostdlib.AccessExport,
		func(parameter int) (gostdlib.EffectKind, error) {
			return parameterCallableEffect(
				project,
				target,
				capabilityParameters+parameter,
				effectMarker,
			)
		},
	)
}

func memberCallableParameters(
	evidence goObject,
	target tsgo.ProjectMember,
	access gostdlib.AccessKind,
	project *tsgo.ProjectInspection,
	effectMarker tsgo.ProjectExport,
) ([]gostdlib.ProviderCallableParameterDocument, error) {
	return callableParameterDocuments(
		evidence,
		access,
		func(parameter int) (gostdlib.EffectKind, error) {
			selected, err := project.CallableParameterEffect(
				target,
				parameter,
				effectMarker,
			)
			if err != nil {
				return gostdlib.EffectInvalid, err
			}
			return providerEffect(selected)
		},
	)
}

func callableParameterDocuments(
	evidence goObject,
	access gostdlib.AccessKind,
	effectAt callableParameterEffect,
) ([]gostdlib.ProviderCallableParameterDocument, error) {
	signature, ok := evidence.object.Type().(*types.Signature)
	if !ok {
		return nil, certifyError(
			"inspect callable parameters",
			evidence.contract.Identity(),
			"selected Go callable signature is absent",
		)
	}
	offset := 0
	if access == gostdlib.AccessStaticMethod {
		offset = 1
	}
	var result []gostdlib.ProviderCallableParameterDocument
	for parameter := range signature.Params().Len() {
		if _, callable := gostdlibsource.DirectCallableParameterSignature(
			signature.Params().At(parameter).Type(),
		); !callable {
			continue
		}
		effect, err := effectAt(parameter + offset)
		if err != nil {
			return nil, err
		}
		result = append(result, gostdlib.ProviderCallableParameterDocument{
			Parameter: parameter,
			Effect:    effect,
		})
	}
	return result, nil
}

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
			selected.Effect != gostdlib.EffectSynchronous {
			mismatches = append(mismatches, fmt.Sprintf(
				"%s parameter %d: provider effect is %s, want sync",
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

const coreResolutionPath = "node_modules/@tsonic/core/types.d.ts"

func verifyExportSourceRawPointers(config resolvedConfig, project *tsgo.ProjectInspection, evidence goObject, target tsgo.ProjectExport) error {
	signature, ok := evidence.object.Type().(*types.Signature)
	if !ok {
		return certifyError("verify source raw pointers", evidence.contract.Identity(), "source signature is absent")
	}
	return verifySourceRawPointers(config, project, evidence.contract.Identity(), signature, gostdlib.AccessExport,
		func(index int) (tsgo.ProjectTypeIdentity, error) {
			return project.CallableParameterTypeIdentity(target, index)
		},
		func(index int) (tsgo.ProjectTypeIdentity, error) {
			return project.CallableResultTypeIdentity(target, index, signature.Results().Len())
		})
}

func verifyMethodSourceRawPointers(config resolvedConfig, project *tsgo.ProjectInspection, evidence goObject, target tsgo.ProjectMember, access gostdlib.AccessKind) error {
	signature, ok := evidence.object.Type().(*types.Signature)
	if !ok {
		return certifyError("verify source raw pointers", evidence.contract.Identity(), "source signature is absent")
	}
	return verifySourceRawPointers(config, project, evidence.contract.Identity(), signature, access,
		func(index int) (tsgo.ProjectTypeIdentity, error) {
			return project.CallableParameterTypeIdentity(target, index)
		},
		func(index int) (tsgo.ProjectTypeIdentity, error) {
			return project.CallableResultTypeIdentity(target, index, signature.Results().Len())
		})
}

func verifySourceRawPointers(
	config resolvedConfig, project *tsgo.ProjectInspection, identity string, signature *types.Signature, access gostdlib.AccessKind,
	parameterIdentity func(int) (tsgo.ProjectTypeIdentity, error), resultIdentity func(int) (tsgo.ProjectTypeIdentity, error),
) error {
	offset := 0
	if access == gostdlib.AccessStaticMethod {
		offset = 1
	}
	var canonical tsgo.ProjectExport
	verify := func(kind string, index int, read func(int) (tsgo.ProjectTypeIdentity, error)) error {
		if canonical.Name() == "" {
			declaration, err := tsoniccore.Resolve(tsoniccore.SymbolRawPointer)
			if err != nil {
				return err
			}
			exports, err := project.Exports(filepath.Join(config.providerRoot, coreResolutionPath))
			if err != nil {
				return err
			}
			for _, exported := range exports {
				if exported.Name() == declaration.Export() {
					canonical = exported
				}
			}
			if canonical.Name() == "" {
				return certifyError("verify source raw pointers", identity, "canonical raw-pointer import is absent")
			}
		}
		selected, err := read(index)
		if err != nil {
			return certifyError("verify source raw pointers", identity, fmt.Sprintf("%s %d: %v", kind, index, err))
		}
		if !selected.Matches(canonical) {
			return certifyError("verify source raw pointers", identity, fmt.Sprintf("%s %d is not the canonical RawPointer declaration", kind, index))
		}
		return nil
	}
	for index := range signature.Params().Len() {
		if basic, ok := signature.Params().At(index).Type().Underlying().(*types.Basic); ok && basic.Kind() == types.UnsafePointer {
			if err := verify("parameter", index+offset, parameterIdentity); err != nil {
				return err
			}
		}
	}
	for index := range signature.Results().Len() {
		if basic, ok := signature.Results().At(index).Type().Underlying().(*types.Basic); ok && basic.Kind() == types.UnsafePointer {
			if err := verify("result", index, resultIdentity); err != nil {
				return err
			}
		}
	}
	return nil
}
