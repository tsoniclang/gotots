package naming

import (
	"go/types"

	environmentcontract "github.com/tsoniclang/gotots/internal/contracts/environment"
	"github.com/tsoniclang/gotots/internal/contracts/gostdlib"
	"github.com/tsoniclang/gotots/internal/emit/api"
)

func (r *Registry) ProviderGenericKernel(
	owner *types.Func,
) (gostdlib.Facet, bool, error) {
	if r == nil || owner == nil || owner.Origin() != owner {
		return gostdlib.Facet{}, false, &api.NameError{
			Reason: "provider generic-kernel owner is invalid",
		}
	}
	binding, ok := r.byObject[owner]
	if !ok || binding.kind != targetBindingProvider {
		return gostdlib.Facet{}, false, nil
	}
	if r.provider == nil || !r.provider.Valid() {
		return gostdlib.Facet{}, false, &api.NameError{
			Name:   owner.Name(),
			Reason: "standard-library provider certificate is invalid",
		}
	}
	contract, err := environmentcontract.Describe(owner)
	if err != nil {
		return gostdlib.Facet{}, false, err
	}
	selected, ok := r.provider.Facet(
		contract.Identity(),
		gostdlib.FacetGenericCallableKernel,
		gostdlib.FacetCapabilityKernel,
	)
	if !ok {
		return gostdlib.Facet{}, false, nil
	}
	capabilities := selected.Capabilities()
	if selected.SourceIdentity() != contract.Identity() ||
		selected.Kind() != gostdlib.FacetGenericCallableKernel ||
		len(capabilities) != 1 ||
		capabilities[0] != gostdlib.FacetCapabilityKernel ||
		selected.ModuleSpecifier() == "" || selected.Export() == "" ||
		len(selected.GenericTypeArguments()) == 0 {
		return gostdlib.Facet{}, false, &api.NameError{
			Name:   contract.Identity(),
			Reason: "provider generic-kernel certificate is inconsistent",
		}
	}
	return selected, true, nil
}

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
	configured := binding.providerGenericTypeArguments
	kernel, kernelOwned, err := r.ProviderGenericKernel(owner.Origin())
	if err != nil {
		return nil, true, err
	}
	if kernelOwned {
		configured = kernel.GenericTypeArguments()
	}
	if len(configured) == 0 {
		return nil, true, &api.NameError{
			Name:   owner.Name(),
			Reason: "provider generic type-argument projection is absent",
		}
	}
	result := make(
		[]api.GenericTypeArgumentProjection,
		0,
		len(configured),
	)
	for _, argument := range configured {
		facet, ok := providerGenericTypeArgumentFacet(argument.Facet)
		if !ok {
			return nil, true, &api.NameError{
				Name:   owner.Name(),
				Reason: "provider generic type-argument facet is invalid",
			}
		}
		projection, projectionErr := api.NewGenericTypeArgumentProjection(
			argument.TypeParameter,
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
