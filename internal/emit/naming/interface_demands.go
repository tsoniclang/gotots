package naming

import (
	"go/types"
	"sort"

	"github.com/tsoniclang/gotots/internal/emit/api"
)

func (r *Registry) internInterfaceContract(
	key string,
	contract *types.Interface,
) (*types.Interface, error) {
	if r == nil ||
		key == "" ||
		contract == nil ||
		!contract.Complete().IsMethodSet() {
		return nil, &api.NameError{
			Reason: "interface contract demand is invalid",
		}
	}
	if existing := r.interfaceContracts[key]; existing != nil {
		if !types.Identical(existing, contract) {
			return nil, &api.NameError{
				Reason: "interface contract key joined non-identical Go types",
			}
		}
		return existing, nil
	}
	r.interfaceContracts[key] = contract
	return contract, nil
}

func (r *Registry) recordInterfaceContractDemand(
	sourceKey string,
	source *types.Interface,
	targetKey string,
	target *types.Interface,
) ([]api.RootRequest, error) {
	if r == nil ||
		sourceKey == "" ||
		source == nil ||
		targetKey == "" ||
		target == nil {
		return nil, &api.NameError{
			Reason: "interface transition demand is invalid",
		}
	}
	targets := r.interfaceContractDemands[sourceKey]
	if targets == nil {
		targets = make(map[string]interfaceContractDemand)
		r.interfaceContractDemands[sourceKey] = targets
	}
	if existing, ok := targets[targetKey]; ok {
		if !types.Identical(existing.source, source) ||
			!types.Identical(existing.target, target) {
			return nil, &api.NameError{
				Reason: "interface transition key joined non-identical Go types",
			}
		}
	} else {
		targets[targetKey] = interfaceContractDemand{
			source: source,
			target: target,
		}
	}
	reached := r.interfaceAdaptersByContract[sourceKey]
	adapterKeys := make([]string, 0, len(reached))
	for adapterKey := range reached {
		adapterKeys = append(adapterKeys, adapterKey)
	}
	sort.Strings(adapterKeys)
	var requests []api.RootRequest
	for _, adapterKey := range adapterKeys {
		binding, ok := r.interfaceAdapters[adapterKey]
		if !ok || binding.owner == nil {
			return nil, &api.NameError{
				Reason: "interface contract reachability has no adapter owner",
			}
		}
		sourceType, ok := binding.owner.InterfaceAdapterType()
		if !ok {
			return nil, &api.NameError{
				Reason: "interface contract reachability has no concrete source type",
			}
		}
		if !types.Implements(sourceType, target) {
			continue
		}
		selected, err := r.interfaceAdapterContractRequests(
			binding,
			targetKey,
			target,
		)
		if err != nil {
			return nil, err
		}
		requests = append(requests, selected...)
	}
	return requests, nil
}

func (r *Registry) recordInterfaceReflectionDemand(
	sourceKey string,
	source *types.Interface,
	reflectionType *types.TypeName,
) ([]api.RootRequest, error) {
	if r == nil || sourceKey == "" || source == nil ||
		reflectionType == nil || reflectionType.IsAlias() {
		return nil, &api.NameError{
			Reason: "interface reflection demand is invalid",
		}
	}
	if existing, ok := r.interfaceReflectionDemands[sourceKey]; ok {
		if !types.Identical(existing.source, source) ||
			existing.reflectionType != reflectionType {
			return nil, &api.NameError{
				Reason: "interface reflection key joined non-identical contracts",
			}
		}
	} else {
		r.interfaceReflectionDemands[sourceKey] = interfaceReflectionDemand{
			source:         source,
			reflectionType: reflectionType,
		}
	}
	reached := r.interfaceAdaptersByContract[sourceKey]
	adapterKeys := make([]string, 0, len(reached))
	for adapterKey := range reached {
		adapterKeys = append(adapterKeys, adapterKey)
	}
	sort.Strings(adapterKeys)
	var requests []api.RootRequest
	for _, adapterKey := range adapterKeys {
		binding, ok := r.interfaceAdapters[adapterKey]
		if !ok {
			return nil, &api.NameError{
				Reason: "interface reflection reachability has no adapter owner",
			}
		}
		selected, err := r.interfaceAdapterReflectionRequest(
			binding,
			reflectionType,
		)
		if err != nil {
			return nil, err
		}
		requests = append(requests, selected)
	}
	return requests, nil
}

func (r *Registry) interfaceAdapterContractRequests(
	binding interfaceAdapterBinding,
	directKey string,
	direct *types.Interface,
) ([]api.RootRequest, error) {
	if r == nil || binding.owner == nil || binding.key == "" {
		return nil, &api.NameError{
			Reason: "interface adapter demand owner is invalid",
		}
	}
	sourceType, ok := binding.owner.InterfaceAdapterType()
	if !ok {
		return nil, &api.NameError{
			Reason: "interface adapter demand has no concrete source type",
		}
	}
	if direct == nil {
		if directKey != "" {
			return nil, &api.NameError{
				Reason: "interface adapter has a contract key without a contract",
			}
		}
		return nil, nil
	}
	if directKey == "" || !types.Implements(sourceType, direct) {
		return nil, &api.NameError{
			Reason: "interface adapter does not implement its direct target contract",
		}
	}
	type pendingContract struct {
		key      string
		contract *types.Interface
	}
	pending := []pendingContract{{
		key:      directKey,
		contract: direct,
	}}
	selected := make(map[string]*types.Interface)
	visited := make(map[string]struct{})
	for len(pending) != 0 {
		next := pending[0]
		pending = pending[1:]
		if _, duplicate := visited[next.key]; duplicate {
			continue
		}
		visited[next.key] = struct{}{}
		reached := r.interfaceAdaptersByContract[next.key]
		if !types.Implements(sourceType, next.contract) {
			return nil, &api.NameError{
				Reason: "interface adapter reached a contract it does not implement",
			}
		}
		if reached == nil {
			reached = make(map[string]struct{})
			r.interfaceAdaptersByContract[next.key] = reached
		}
		reached[binding.key] = struct{}{}
		selected[next.key] = next.contract

		targets := r.interfaceContractDemands[next.key]
		targetKeys := make([]string, 0, len(targets))
		for targetKey := range targets {
			targetKeys = append(targetKeys, targetKey)
		}
		sort.Strings(targetKeys)
		for _, targetKey := range targetKeys {
			target := targets[targetKey].target
			if types.Implements(sourceType, target) {
				pending = append(pending, pendingContract{
					key:      targetKey,
					contract: target,
				})
			}
		}
	}
	keys := make([]string, 0, len(selected))
	for key := range selected {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	requests := make([]api.RootRequest, 0, len(keys))
	for _, key := range keys {
		request, err := api.NewInterfaceAdapterContractRequest(
			binding.owner,
			selected[key],
			key,
		)
		if err != nil {
			return nil, err
		}
		requests = append(requests, request)
		if demand, ok := r.interfaceReflectionDemands[key]; ok {
			reflection, reflectionErr := r.interfaceAdapterReflectionRequest(
				binding,
				demand.reflectionType,
			)
			if reflectionErr != nil {
				return nil, reflectionErr
			}
			requests = append(requests, reflection)
		}
	}
	return requests, nil
}

func (r *Registry) interfaceAdapterReflectionRequest(
	binding interfaceAdapterBinding,
	reflectionType *types.TypeName,
) (api.RootRequest, error) {
	if binding.owner == nil || binding.key == "" || reflectionType == nil {
		return api.RootRequest{}, &api.NameError{
			Reason: "interface adapter reflection owner is invalid",
		}
	}
	sourceType, ok := binding.owner.InterfaceAdapterType()
	if !ok {
		return api.RootRequest{}, &api.NameError{
			Reason: "interface adapter reflection owner has no source type",
		}
	}
	reflection, err := r.internReflectionType(
		binding.key,
		sourceType,
		reflectionType,
	)
	if err != nil {
		return api.RootRequest{}, err
	}
	return api.NewReflectionTypeRequest(reflection.owner)
}
