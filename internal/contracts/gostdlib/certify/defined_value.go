package certify

import (
	"go/types"

	"github.com/tsoniclang/gotots/internal/contracts/gostdlib"
)

func applyDefinedValueRepresentations(
	source goSurface,
	modules []gostdlib.ModuleDocument,
	facets []facetSeed,
	identities map[string]struct{},
) ([]gostdlib.ModuleDocument, error) {
	operations := make(map[string]struct{})
	for _, facet := range facets {
		if facet.Kind != gostdlib.FacetDefinedValueOperations {
			continue
		}
		if _, duplicate := operations[facet.SourceIdentity]; duplicate {
			return nil, certifyError(
				"bind defined value",
				facet.SourceIdentity,
				"operation representation is duplicated",
			)
		}
		operations[facet.SourceIdentity] = struct{}{}
	}
	claimed := make(map[string]struct{}, len(identities)+len(operations))
	result := append([]gostdlib.ModuleDocument(nil), modules...)
	for moduleIndex := range result {
		result[moduleIndex].Bindings = append(
			[]gostdlib.BindingDocument(nil),
			result[moduleIndex].Bindings...,
		)
		for bindingIndex := range result[moduleIndex].Bindings {
			binding := &result[moduleIndex].Bindings[bindingIndex]
			if binding.Kind != gostdlib.BindingType ||
				binding.Access != gostdlib.AccessExport {
				continue
			}
			evidence, ok := source.objects[binding.Identity]
			if !ok {
				return nil, certifyError(
					"bind defined value",
					binding.Identity,
					"selected-Go declaration is absent",
				)
			}
			typeName, ok := evidence.object.(*types.TypeName)
			if !ok || typeName.IsAlias() || !definedValueType(typeName.Type()) {
				continue
			}
			_, identity := identities[binding.Identity]
			_, operation := operations[binding.Identity]
			if identity == operation {
				return nil, certifyError(
					"bind defined value",
					binding.Identity,
					"representation owner is missing or duplicated",
				)
			}
			if identity {
				if _, callable := typeName.Type().Underlying().(*types.Signature); !callable {
					return nil, certifyError(
						"bind defined value",
						binding.Identity,
						"identity representation is not callable",
					)
				}
				binding.DefinedValue = gostdlib.DefinedValueRepresentationIdentity
			} else {
				binding.DefinedValue = gostdlib.DefinedValueRepresentationOperations
			}
			claimed[binding.Identity] = struct{}{}
		}
	}
	for identity := range identities {
		if _, ok := claimed[identity]; !ok {
			return nil, certifyError(
				"bind defined value",
				identity,
				"identity representation has no selected type",
			)
		}
	}
	for identity := range operations {
		if _, ok := claimed[identity]; !ok {
			return nil, certifyError(
				"bind defined value",
				identity,
				"operation representation has no selected type",
			)
		}
	}
	return result, nil
}

func definedValueType(source types.Type) bool {
	named, ok := types.Unalias(source).(*types.Named)
	if !ok || named.Obj() == nil {
		return false
	}
	switch underlying := named.Underlying().(type) {
	case *types.Basic, *types.Array, *types.Slice, *types.Pointer,
		*types.Map, *types.Chan:
		return true
	case *types.Signature:
		return underlying.Recv() == nil
	default:
		return false
	}
}
