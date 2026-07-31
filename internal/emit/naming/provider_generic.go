package naming

import (
	"go/types"

	"github.com/tsoniclang/gotots/internal/contracts/gostdlib"
	"github.com/tsoniclang/gotots/internal/emit/api"
)

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
