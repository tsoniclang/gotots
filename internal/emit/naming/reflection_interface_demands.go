package naming

import (
	"go/types"
	"sort"

	"github.com/tsoniclang/gotots/internal/emit/api"
)

func (r *Registry) recordReflectionInterfaceAdapter(
	source interfaceContractSelection,
	binding interfaceAdapterBinding,
) error {
	if r == nil || !source.valid() || binding.owner == nil || binding.key == "" {
		return &api.NameError{
			Reason: "reflection interface exposure is invalid",
		}
	}
	canonical, ok := r.interfaceAdapters[binding.key]
	if !ok || canonical.owner != binding.owner || canonical.name != binding.name ||
		canonical.reflectionName != binding.reflectionName {
		return &api.NameError{
			Reason: "reflection interface exposure has no canonical adapter",
		}
	}
	sourceType, ok := binding.owner.InterfaceAdapterType()
	if !ok || !types.Implements(sourceType, source.contract) {
		return &api.NameError{
			Reason: "reflection interface exposure does not implement its source contract",
		}
	}
	selected, err := r.internInterfaceContract(source)
	if err != nil {
		return err
	}
	key := selected.demandKey()
	exposure := r.reflectionInterfaceExposures[key]
	if exposure.adapters == nil {
		exposure = reflectionInterfaceExposure{
			source:   selected,
			adapters: make(map[string]struct{}),
		}
	} else if !sameInterfaceContractSelection(exposure.source, selected) {
		return &api.NameError{
			Reason: "reflection interface exposure key joined non-identical contracts",
		}
	}
	if _, exists := exposure.adapters[binding.key]; exists {
		return nil
	}
	exposure.adapters[binding.key] = struct{}{}
	r.reflectionInterfaceExposures[key] = exposure
	r.reflectionInterfaceDirty = true
	return nil
}

func (r *Registry) FlushReflectionInterfaceDemands() (
	[]api.RootRequest,
	error,
) {
	if r == nil {
		return nil, &api.NameError{
			Reason: "reflection interface demand owner is nil",
		}
	}
	if !r.reflectionInterfaceDirty {
		return nil, nil
	}
	r.reflectionInterfaceDirty = false
	exposureKeys := make([]string, 0, len(r.reflectionInterfaceExposures))
	for key := range r.reflectionInterfaceExposures {
		exposureKeys = append(exposureKeys, key)
	}
	sort.Strings(exposureKeys)
	var requests []api.RootRequest
	for _, exposureKey := range exposureKeys {
		exposure := r.reflectionInterfaceExposures[exposureKey]
		if !exposure.source.valid() || len(exposure.adapters) == 0 {
			return nil, &api.NameError{
				Reason: "reflection interface exposure state is invalid",
			}
		}
		adapterKeys := make([]string, 0, len(exposure.adapters))
		for adapterKey := range exposure.adapters {
			adapterKeys = append(adapterKeys, adapterKey)
		}
		sort.Strings(adapterKeys)
		for _, adapterKey := range adapterKeys {
			binding, ok := r.interfaceAdapters[adapterKey]
			if !ok || binding.owner == nil {
				return nil, &api.NameError{
					Reason: "reflection interface exposure has no adapter owner",
				}
			}
			reached, err := r.reflectionInterfaceReachableContracts(
				binding,
				exposure.source,
			)
			if err != nil {
				return nil, err
			}
			settled := r.reflectionInterfaceContracts[adapterKey]
			if settled == nil {
				settled = make(map[string]struct{})
				r.reflectionInterfaceContracts[adapterKey] = settled
			}
			keys := make([]string, 0, len(reached))
			for key := range reached {
				keys = append(keys, key)
			}
			sort.Strings(keys)
			for _, key := range keys {
				if _, exists := settled[key]; exists {
					continue
				}
				target := reached[key]
				request, err := api.NewInterfaceAdapterContractRequest(
					binding.owner,
					target.sourceType,
					target.contract,
					key,
				)
				if err != nil {
					return nil, err
				}
				settled[key] = struct{}{}
				requests = append(requests, request)
			}
		}
	}
	return api.CombineRequests(requests), nil
}

func (r *Registry) reflectionInterfaceReachableContracts(
	binding interfaceAdapterBinding,
	source interfaceContractSelection,
) (map[string]interfaceContractSelection, error) {
	sourceType, ok := binding.owner.InterfaceAdapterType()
	if !ok {
		return nil, &api.NameError{
			Reason: "reflection interface exposure has no concrete source type",
		}
	}
	pending := []interfaceContractSelection{source}
	visited := make(map[string]struct{})
	selected := make(map[string]interfaceContractSelection)
	for len(pending) != 0 {
		next := pending[0]
		pending = pending[1:]
		nextKey := next.demandKey()
		if _, duplicate := visited[nextKey]; duplicate {
			continue
		}
		visited[nextKey] = struct{}{}
		targets := r.interfaceContractDemands[next.contractKey]
		targetKeys := make([]string, 0, len(targets))
		for targetKey := range targets {
			targetKeys = append(targetKeys, targetKey)
		}
		sort.Strings(targetKeys)
		for _, targetKey := range targetKeys {
			demand := targets[targetKey]
			if !types.Identical(demand.source, next.contract) {
				return nil, &api.NameError{
					Reason: "reflection interface demand source identity drifted",
				}
			}
			target := demand.target
			if !types.Implements(sourceType, target.contract) {
				continue
			}
			pending = append(pending, target)
			if target.contract.NumMethods() != 0 {
				selected[target.demandKey()] = target
			}
		}
	}
	return selected, nil
}
