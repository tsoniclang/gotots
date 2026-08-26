package naming

import (
	"github.com/tsoniclang/gotots/internal/emit/api"
)

func (r *Registry) recordReflectionInterfaceAdapter(
	binding interfaceAdapterBinding,
) ([]api.RootRequest, error) {
	if r == nil || binding.owner == nil || binding.key == "" {
		return nil, &api.NameError{
			Reason: "reflection interface method-set demand is invalid",
		}
	}
	canonical, ok := r.interfaceAdapters[binding.key]
	if !ok || canonical.owner != binding.owner || canonical.name != binding.name ||
		canonical.reflectionName != binding.reflectionName {
		return nil, &api.NameError{
			Reason: "reflection interface method-set demand has no canonical adapter",
		}
	}
	request, err := api.NewInterfaceAdapterCompleteMethodSetRequest(binding.owner)
	if err != nil {
		return nil, err
	}
	return []api.RootRequest{request}, nil
}
