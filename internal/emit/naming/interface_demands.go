package naming

import (
	"go/types"
	"sort"

	gostdlibsource "github.com/tsoniclang/gotots/internal/contracts/gostdlib/sourcecontract"
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
	bridges := r.providerInterfaceBridgesByContract[sourceKey]
	bridgeKeys := make([]string, 0, len(bridges))
	for bridgeKey := range bridges {
		bridgeKeys = append(bridgeKeys, bridgeKey)
	}
	sort.Strings(bridgeKeys)
	for _, bridgeKey := range bridgeKeys {
		binding, ok := r.providerInterfaceBridges[bridgeKey]
		if !ok || binding.owner == nil {
			return nil, &api.NameError{
				Reason: "interface contract reachability has no provider bridge owner",
			}
		}
		selected, err := r.providerInterfaceBridgeContractRequests(
			binding,
			sourceKey,
			source,
		)
		if err != nil {
			return nil, err
		}
		requests = append(requests, selected...)
	}
	return requests, nil
}

func (r *Registry) providerInterfaceBridgeContractRequests(
	binding providerInterfaceBridgeBinding,
	directKey string,
	direct *types.Interface,
) ([]api.RootRequest, error) {
	if r == nil || binding.owner == nil || binding.key == "" ||
		directKey == "" || direct == nil {
		return nil, &api.NameError{
			Reason: "provider-interface bridge demand owner is invalid",
		}
	}
	base, ok := binding.owner.ProviderInterfaceBridgeType()
	if !ok {
		return nil, &api.NameError{
			Reason: "provider-interface bridge demand has no source type",
		}
	}
	baseContract, ok := base.Underlying().(*types.Interface)
	if !ok {
		return nil, &api.NameError{
			Reason: "provider-interface bridge demand source is not an interface",
		}
	}
	baseContract = baseContract.Complete()
	type pendingContract struct {
		key      string
		contract *types.Interface
	}
	pending := []pendingContract{{key: directKey, contract: direct}}
	selected := make(map[string]*types.Interface)
	visited := make(map[string]struct{})
	for len(pending) != 0 {
		next := pending[0]
		pending = pending[1:]
		if _, duplicate := visited[next.key]; duplicate {
			continue
		}
		contract, err := r.internInterfaceContract(next.key, next.contract)
		if err != nil {
			return nil, err
		}
		next.contract = contract
		visited[next.key] = struct{}{}
		reached := r.providerInterfaceBridgesByContract[next.key]
		if reached == nil {
			reached = make(map[string]struct{})
			r.providerInterfaceBridgesByContract[next.key] = reached
		}
		reached[binding.key] = struct{}{}
		targets := r.interfaceContractDemands[next.key]
		targetKeys := make([]string, 0, len(targets))
		for targetKey := range targets {
			targetKeys = append(targetKeys, targetKey)
		}
		sort.Strings(targetKeys)
		for _, targetKey := range targetKeys {
			demand := targets[targetKey]
			if !types.Identical(demand.source, next.contract) {
				return nil, &api.NameError{
					Reason: "provider-interface demand source identity drifted",
				}
			}
			target := demand.target
			if types.Implements(baseContract, target) {
				pending = append(pending, pendingContract{
					key:      targetKey,
					contract: target,
				})
				continue
			}
			capabilities, err := r.providerInterfaceBridgeCapabilities(
				base,
				target,
			)
			if err != nil {
				return nil, err
			}
			if len(capabilities) == 0 {
				continue
			}
			pending = append(pending, pendingContract{
				key:      targetKey,
				contract: target,
			})
			for _, capability := range capabilities {
				if existing := selected[capability.demandKey]; existing != nil &&
					!types.Identical(existing, capability.target) {
					return nil, &api.NameError{
						Reason: "provider-interface capability key joined non-identical contracts",
					}
				}
				selected[capability.demandKey] = capability.target
				pending = append(pending, pendingContract{
					key:      capability.targetKey,
					contract: capability.target,
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
		request, err := api.NewProviderInterfaceCapabilityRequest(
			binding.owner,
			selected[key],
			key,
		)
		if err != nil {
			return nil, err
		}
		requests = append(requests, request)
	}
	return requests, nil
}

func (r *Registry) providerInterfaceBridgeCapabilities(
	base *types.Named,
	target *types.Interface,
) ([]providerInterfaceCapabilityBinding, error) {
	baseContract, ok := base.Underlying().(*types.Interface)
	if !ok {
		return nil, &api.NameError{
			Reason: "provider-interface capability base is invalid",
		}
	}
	baseKey, err := canonicalProviderInterfaceContractKey(baseContract)
	if err != nil {
		return nil, err
	}
	byTarget := r.providerInterfaceCapabilities[baseKey]
	keys := make([]string, 0, len(byTarget))
	for key := range byTarget {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	selected := make([]providerInterfaceCapabilityBinding, 0, len(keys))
	targetKey, err := canonicalProviderInterfaceContractKey(target)
	if err != nil {
		return nil, err
	}
	for _, key := range keys {
		capability := byTarget[key]
		if !types.Identical(capability.base, base) {
			return nil, &api.NameError{
				Reason: "provider-interface capability base identity drifted",
			}
		}
		_, matches, matchErr := gostdlibsource.SelectProviderInterfaceMethods(
			capability.certificate.TargetInterface(),
			target,
		)
		if matchErr != nil {
			return nil, matchErr
		}
		if matches {
			capability.target = target
			capability.targetKey = targetKey
			capability.demandKey = capability.key + "\x00" + targetKey
			if existing, ok := r.providerInterfaceCapabilityDemands[capability.demandKey]; ok && (!types.Identical(existing.base, capability.base) ||
				!types.Identical(existing.target, capability.target)) {
				return nil, &api.NameError{
					Reason: "provider-interface capability demand identity drifted",
				}
			}
			r.providerInterfaceCapabilityDemands[capability.demandKey] = capability
			selected = append(selected, capability)
		}
	}
	return selected, nil
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
		requests = append(requests, selected...)
	}
	return requests, nil
}

// recordReflectionValueContract joins one reflected interface slot to every
// reflected concrete adapter that implements it. The join is replayed over
// existing adapters here and over later adapters in
// interfaceAdapterContractRequests, making discovery order irrelevant.
func (r *Registry) recordReflectionValueContract(
	sourceKey string,
	source *types.Interface,
	reflectionType *types.TypeName,
) ([]api.RootRequest, error) {
	contract, err := r.internInterfaceContract(sourceKey, source)
	if err != nil {
		return nil, err
	}
	r.reflectionValueContracts[sourceKey] = struct{}{}
	requests, err := r.recordInterfaceReflectionDemand(
		sourceKey,
		contract,
		reflectionType,
	)
	if err != nil {
		return nil, err
	}
	adapterKeys := make([]string, 0, len(r.interfaceAdapters))
	for key := range r.interfaceAdapters {
		adapterKeys = append(adapterKeys, key)
	}
	sort.Strings(adapterKeys)
	for _, key := range adapterKeys {
		if _, reflected := r.reflectionValueDemands[key]; !reflected {
			continue
		}
		binding := r.interfaceAdapters[key]
		sourceType, ok := binding.owner.InterfaceAdapterType()
		if !ok || !types.Implements(sourceType, contract) {
			continue
		}
		selected, selectedErr := r.interfaceAdapterContractRequests(
			binding,
			sourceKey,
			contract,
		)
		if selectedErr != nil {
			return nil, selectedErr
		}
		requests = append(requests, selected...)
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
	type pendingContract struct {
		key      string
		contract *types.Interface
	}
	var pending []pendingContract
	if direct == nil {
		if directKey != "" {
			return nil, &api.NameError{
				Reason: "interface adapter has a contract key without a contract",
			}
		}
		if _, reflected := r.reflectionValueDemands[binding.key]; !reflected {
			return nil, nil
		}
		keys := make([]string, 0, len(r.reflectionValueContracts))
		for key := range r.reflectionValueContracts {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		for _, key := range keys {
			contract := r.interfaceContracts[key]
			if contract != nil && types.Implements(sourceType, contract) {
				pending = append(pending, pendingContract{
					key:      key,
					contract: contract,
				})
			}
		}
	} else {
		if directKey == "" || !types.Implements(sourceType, direct) {
			return nil, &api.NameError{
				Reason: "interface adapter does not implement its direct target contract",
			}
		}
		pending = append(pending, pendingContract{
			key:      directKey,
			contract: direct,
		})
	}
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
			requests = append(requests, reflection...)
		}
	}
	return requests, nil
}

func (r *Registry) interfaceAdapterReflectionRequest(
	binding interfaceAdapterBinding,
	reflectionType *types.TypeName,
) ([]api.RootRequest, error) {
	if binding.owner == nil || binding.key == "" || reflectionType == nil {
		return nil, &api.NameError{
			Reason: "interface adapter reflection owner is invalid",
		}
	}
	sourceType, ok := binding.owner.InterfaceAdapterType()
	if !ok {
		return nil, &api.NameError{
			Reason: "interface adapter reflection owner has no source type",
		}
	}
	reflection, err := r.internReflectionType(
		binding.key,
		sourceType,
		reflectionType,
	)
	if err != nil {
		return nil, err
	}
	descriptor, err := api.NewReflectionTypeRequest(reflection.owner)
	if err != nil {
		return nil, err
	}
	requests := []api.RootRequest{descriptor}
	if r.contractDemandsValueOperations(binding) {
		r.reflectionValueDemands[binding.key] = struct{}{}
		facet, facetErr := r.reflectionValueOperationsRequest(binding.key)
		if facetErr != nil {
			return nil, facetErr
		}
		requests = append(requests, facet)
	}
	return requests, nil
}

// contractDemandsValueOperations reports whether any observed reflection
// contract reached by one adapter demands generated value operations.
func (r *Registry) contractDemandsValueOperations(
	binding interfaceAdapterBinding,
) bool {
	for contractKey := range r.reflectionValueContracts {
		if reached, ok := r.interfaceAdaptersByContract[contractKey]; ok {
			if _, member := reached[binding.key]; member {
				return true
			}
		}
	}
	return false
}
