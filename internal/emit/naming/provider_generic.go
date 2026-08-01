package naming

import (
	"go/types"

	"github.com/tsoniclang/gotots/internal/contracts/gostdlib"
	"github.com/tsoniclang/gotots/internal/emit/api"
)

func (r *Registry) ProviderGenericTypeArguments(
	owner *types.Func,
) ([]api.GenericTypeArgumentProjection, bool, error) {
	if r == nil || owner == nil {
		return nil, false, &api.NameError{
			Reason: "provider generic type-argument owner is invalid",
		}
	}
	binding, ok := r.byObject[owner.Origin()]
	if !ok || binding.kind != targetBindingProvider {
		return nil, false, nil
	}
	if len(binding.providerGenericTypeArguments) == 0 {
		return nil, true, &api.NameError{
			Name:   owner.Name(),
			Reason: "provider generic type-argument projection is absent",
		}
	}
	result := make(
		[]api.GenericTypeArgumentProjection,
		0,
		len(binding.providerGenericTypeArguments),
	)
	for _, configured := range binding.providerGenericTypeArguments {
		facet, ok := providerGenericTypeArgumentFacet(configured.Facet)
		if !ok {
			return nil, true, &api.NameError{
				Name:   owner.Name(),
				Reason: "provider generic type-argument facet is invalid",
			}
		}
		projection, projectionErr := api.NewGenericTypeArgumentProjection(
			configured.TypeParameter,
			facet,
		)
		if projectionErr != nil {
			return nil, true, projectionErr
		}
		result = append(result, projection)
	}
	return result, true, nil
}

func (n *File) ProviderGenericTypeArguments(
	owner *types.Func,
) ([]api.GenericTypeArgumentProjection, bool, error) {
	if n == nil || n.owner == nil || n.owner.registry == nil {
		return nil, false, &api.NameError{
			Reason: "provider generic type-argument registry is invalid",
		}
	}
	return n.owner.registry.ProviderGenericTypeArguments(owner)
}

func providerGenericTypeArgumentFacet(
	facet gostdlib.GenericTypeArgumentFacet,
) (api.GenericTypeArgumentFacet, bool) {
	switch facet {
	case gostdlib.GenericTypeArgumentLogical:
		return api.GenericTypeArgumentLogical, true
	case gostdlib.GenericTypeArgumentStorage:
		return api.GenericTypeArgumentStorage, true
	case gostdlib.GenericTypeArgumentContainerStorage:
		return api.GenericTypeArgumentContainerStorage, true
	case gostdlib.GenericTypeArgumentPointer:
		return api.GenericTypeArgumentPointer, true
	default:
		return api.GenericTypeArgumentInvalid, false
	}
}

func (r *Registry) ProviderGenericOperations(
	owner *types.Func,
) ([]gostdlib.GenericOperationDocument, bool, error) {
	if r == nil || owner == nil {
		return nil, false, &api.NameError{
			Reason: "provider generic-operation owner is invalid",
		}
	}
	binding, ok := r.byObject[owner.Origin()]
	if !ok || binding.kind == targetBindingEnvironment ||
		binding.kind == targetBindingSource || binding.kind == targetBindingLocal {
		return nil, false, nil
	}
	if binding.kind == targetBindingMissingProvider {
		return nil, true, nil
	}
	if binding.kind != targetBindingProvider {
		return nil, false, &api.NameError{
			Name:   owner.Name(),
			Reason: "provider generic-operation binding is invalid",
		}
	}
	operations, err := gostdlib.CanonicalGenericOperations(
		binding.providerGenericOperations,
	)
	if err != nil {
		return nil, true, &api.NameError{
			Name:   owner.Name(),
			Reason: "provider generic-operation contract is invalid",
		}
	}
	return operations, true, nil
}
