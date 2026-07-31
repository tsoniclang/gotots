package naming

import (
	"go/types"
	"slices"

	"github.com/tsoniclang/gotots/internal/contracts/gostdlib"
	"github.com/tsoniclang/gotots/internal/emit/api"
)

func (r *Registry) ProviderGenericTypeArguments(
	owner *types.Func,
) ([]int, bool, error) {
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
	return slices.Clone(binding.providerGenericTypeArguments), true, nil
}

func (n *File) ProviderGenericTypeArguments(
	owner *types.Func,
) ([]int, bool, error) {
	if n == nil || n.owner == nil || n.owner.registry == nil {
		return nil, false, &api.NameError{
			Reason: "provider generic type-argument registry is invalid",
		}
	}
	return n.owner.registry.ProviderGenericTypeArguments(owner)
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
