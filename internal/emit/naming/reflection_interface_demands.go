package naming

import (
	"go/types"
	"sort"

	"github.com/tsoniclang/gotots/internal/emit/api"
)

func (r *Registry) recordReflectionInterfaceAdapter(
	source interfaceContractSelection,
	binding interfaceAdapterBinding,
) ([]api.RootRequest, error) {
	if r == nil || !source.valid() || binding.owner == nil || binding.key == "" {
		return nil, &api.NameError{
			Reason: "reflection interface exposure is invalid",
		}
	}
	canonical, ok := r.interfaceAdapters[binding.key]
	if !ok || canonical.owner != binding.owner || canonical.name != binding.name ||
		canonical.reflectionName != binding.reflectionName {
		return nil, &api.NameError{
			Reason: "reflection interface exposure has no canonical adapter",
		}
	}
	sourceType, ok := binding.owner.InterfaceAdapterType()
	if !ok || !types.Implements(sourceType, source.contract) {
		return nil, &api.NameError{
			Reason: "reflection interface exposure does not implement its source contract",
		}
	}
	var err error
	source, err = r.internInterfaceContract(source)
	if err != nil {
		return nil, err
	}
	cacheKey := interfaceDemandRequestKey{
		kind:       interfaceDemandReflectionAdapter,
		sourceKey:  source.contractKey,
		adapterKey: binding.key,
	}
	exposed := r.reflectionAdaptersByContract[source.contractKey]
	if exposed == nil {
		exposed = make(map[string]struct{})
		r.reflectionAdaptersByContract[source.contractKey] = exposed
	}
	if _, exists := exposed[binding.key]; exists {
		if requests, cached := r.cachedInterfaceDemandRequests(cacheKey); cached {
			return requests, nil
		}
	} else {
		exposed[binding.key] = struct{}{}
		r.invalidateInterfaceDemandRequests()
	}

	targets := r.interfaceContractDemands[source.contractKey]
	targetKeys := make([]string, 0, len(targets))
	for targetKey := range targets {
		targetKeys = append(targetKeys, targetKey)
	}
	sort.Strings(targetKeys)
	var requests []api.RootRequest
	for _, targetKey := range targetKeys {
		target := targets[targetKey].target
		if !types.Implements(sourceType, target.contract) {
			continue
		}
		selected, requestErr := r.interfaceAdapterContractRequests(
			binding,
			&target,
		)
		if requestErr != nil {
			return nil, requestErr
		}
		requests = append(requests, selected...)
	}
	return r.cacheInterfaceDemandRequests(cacheKey, requests), nil
}
