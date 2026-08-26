package naming

import (
	"go/types"
	"slices"
	"sort"

	gostdlibsource "github.com/tsoniclang/gotots/internal/contracts/gostdlib/sourcecontract"
	"github.com/tsoniclang/gotots/internal/emit/api"
)

const (
	interfaceDemandTransition uint8 = iota + 1
	interfaceDemandAdapter
	interfaceDemandProviderBridge
	interfaceDemandReflectionAdapter
)

func (r *Registry) invalidateInterfaceDemandRequests() {
	clear(r.interfaceDemandRequests)
}

func (r *Registry) cachedInterfaceDemandRequests(
	key interfaceDemandRequestKey,
) ([]api.RootRequest, bool) {
	requests, ok := r.interfaceDemandRequests[key]
	return slices.Clone(requests), ok
}

func (r *Registry) cacheInterfaceDemandRequests(
	key interfaceDemandRequestKey,
	requests []api.RootRequest,
) []api.RootRequest {
	compact := api.CombineRequests(requests)
	r.interfaceDemandRequests[key] = compact
	return slices.Clone(compact)
}

func (r *Registry) internInterfaceContract(
	selection interfaceContractSelection,
) (interfaceContractSelection, error) {
	if r == nil || !selection.valid() {
		return interfaceContractSelection{}, &api.NameError{
			Reason: "interface contract demand is invalid",
		}
	}
	bySurface := r.interfaceContracts[selection.contractKey]
	if bySurface == nil {
		bySurface = make(map[string]interfaceContractSelection)
		r.interfaceContracts[selection.contractKey] = bySurface
	}
	for _, existing := range bySurface {
		if !types.Identical(existing.contract, selection.contract) {
			return interfaceContractSelection{}, &api.NameError{
				Reason: "interface contract key joined non-identical Go types",
			}
		}
	}
	if existing, ok := bySurface[selection.surfaceKey]; ok {
		if !sameInterfaceContractSelection(existing, selection) {
			return interfaceContractSelection{}, &api.NameError{
				Reason: "interface contract surface key joined non-identical Go types",
			}
		}
		return existing, nil
	}
	bySurface[selection.surfaceKey] = selection
	return selection, nil
}

func (r *Registry) recordInterfaceContractDemand(
	source interfaceContractSelection,
	target interfaceContractSelection,
) ([]api.RootRequest, error) {
	if r == nil || !source.valid() || !target.valid() {
		return nil, &api.NameError{
			Reason: "interface transition demand is invalid",
		}
	}
	var err error
	source, err = r.internInterfaceContract(source)
	if err != nil {
		return nil, err
	}
	target, err = r.internInterfaceContract(target)
	if err != nil {
		return nil, err
	}
	targets := r.interfaceContractDemands[source.contractKey]
	if targets == nil {
		targets = make(map[string]interfaceContractDemand)
		r.interfaceContractDemands[source.contractKey] = targets
	}
	targetDemandKey := target.demandKey()
	cacheKey := interfaceDemandRequestKey{
		kind:      interfaceDemandTransition,
		sourceKey: source.contractKey,
		targetKey: targetDemandKey,
	}
	if existing, ok := targets[targetDemandKey]; ok {
		if !types.Identical(existing.source, source.contract) ||
			!sameInterfaceContractSelection(existing.target, target) {
			return nil, &api.NameError{
				Reason: "interface transition key joined non-identical Go types",
			}
		}
		if requests, cached := r.cachedInterfaceDemandRequests(cacheKey); cached {
			return requests, nil
		}
	} else {
		targets[targetDemandKey] = interfaceContractDemand{
			source: source.contract,
			target: target,
		}
		r.invalidateInterfaceDemandRequests()
	}
	reached := r.interfaceAdaptersByContract[source.contractKey]
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
		if !types.Implements(sourceType, target.contract) {
			continue
		}
		selected, err := r.interfaceAdapterContractRequests(
			binding,
			&target,
		)
		if err != nil {
			return nil, err
		}
		requests = append(requests, selected...)
	}
	exposed := r.reflectionAdaptersByContract[source.contractKey]
	exposedKeys := make([]string, 0, len(exposed))
	for adapterKey := range exposed {
		exposedKeys = append(exposedKeys, adapterKey)
	}
	sort.Strings(exposedKeys)
	for _, adapterKey := range exposedKeys {
		if _, ordinary := reached[adapterKey]; ordinary {
			continue
		}
		binding, ok := r.interfaceAdapters[adapterKey]
		if !ok || binding.owner == nil {
			return nil, &api.NameError{
				Reason: "reflection interface exposure has no adapter owner",
			}
		}
		sourceType, ok := binding.owner.InterfaceAdapterType()
		if !ok {
			return nil, &api.NameError{
				Reason: "reflection interface exposure has no concrete source type",
			}
		}
		if !types.Implements(sourceType, target.contract) {
			continue
		}
		selected, err := r.interfaceAdapterContractRequests(
			binding,
			&target,
		)
		if err != nil {
			return nil, err
		}
		requests = append(requests, selected...)
	}
	bridges := r.providerInterfaceBridgesByContract[source.contractKey]
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
			source.contractKey,
			source.contract,
		)
		if err != nil {
			return nil, err
		}
		requests = append(requests, selected...)
	}
	return r.cacheInterfaceDemandRequests(cacheKey, requests), nil
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
	cacheKey := interfaceDemandRequestKey{
		kind:       interfaceDemandProviderBridge,
		sourceKey:  directKey,
		adapterKey: binding.key,
	}
	if requests, cached := r.cachedInterfaceDemandRequests(cacheKey); cached {
		return requests, nil
	}
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
		selection, err := r.internInterfaceContract(interfaceContractSelection{
			sourceType:  next.contract,
			contract:    next.contract,
			contractKey: next.key,
			surfaceKey:  next.key,
		})
		if err != nil {
			return nil, err
		}
		next.contract = selection.contract
		visited[next.key] = struct{}{}
		reached := r.providerInterfaceBridgesByContract[next.key]
		if reached == nil {
			reached = make(map[string]struct{})
			r.providerInterfaceBridgesByContract[next.key] = reached
		}
		if _, exists := reached[binding.key]; !exists {
			reached[binding.key] = struct{}{}
			r.invalidateInterfaceDemandRequests()
		}
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
			target := demand.target.contract
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
	return r.cacheInterfaceDemandRequests(cacheKey, requests), nil
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
			if _, exists := r.providerInterfaceCapabilityDemands[capability.demandKey]; !exists {
				r.providerInterfaceCapabilityDemands[capability.demandKey] = capability
				r.invalidateInterfaceDemandRequests()
			}
			selected = append(selected, capability)
		}
	}
	return selected, nil
}

func (r *Registry) interfaceAdapterContractRequests(
	binding interfaceAdapterBinding,
	direct *interfaceContractSelection,
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
	directKey := ""
	if direct != nil {
		directKey = direct.demandKey()
	}
	cacheKey := interfaceDemandRequestKey{
		kind:       interfaceDemandAdapter,
		targetKey:  directKey,
		adapterKey: binding.key,
	}
	if requests, cached := r.cachedInterfaceDemandRequests(cacheKey); cached {
		return requests, nil
	}
	var pending []interfaceContractSelection
	if direct == nil {
		if _, reflected := r.reflectionValueDemands[binding.key]; !reflected {
			return nil, nil
		}
		keys := make([]string, 0, len(r.reflectionValueContracts))
		for key := range r.reflectionValueContracts {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		for _, key := range keys {
			selection := r.reflectionValueContracts[key]
			if types.Implements(sourceType, selection.contract) {
				pending = append(pending, selection)
			}
		}
	} else {
		if !direct.valid() || !types.Implements(sourceType, direct.contract) {
			return nil, &api.NameError{
				Reason: "interface adapter does not implement its direct target contract",
			}
		}
		selected, err := r.internInterfaceContract(*direct)
		if err != nil {
			return nil, err
		}
		pending = append(pending, selected)
	}
	selected := make(map[string]interfaceContractSelection)
	visited := make(map[string]struct{})
	for len(pending) != 0 {
		next := pending[0]
		pending = pending[1:]
		nextKey := next.demandKey()
		if _, duplicate := visited[nextKey]; duplicate {
			continue
		}
		visited[nextKey] = struct{}{}
		reached := r.interfaceAdaptersByContract[next.contractKey]
		if !types.Implements(sourceType, next.contract) {
			return nil, &api.NameError{
				Reason: "interface adapter reached a contract it does not implement",
			}
		}
		if reached == nil {
			reached = make(map[string]struct{})
			r.interfaceAdaptersByContract[next.contractKey] = reached
		}
		if _, exists := reached[binding.key]; !exists {
			reached[binding.key] = struct{}{}
			r.invalidateInterfaceDemandRequests()
		}
		selected[nextKey] = next

		targets := r.interfaceContractDemands[next.contractKey]
		targetKeys := make([]string, 0, len(targets))
		for targetKey := range targets {
			targetKeys = append(targetKeys, targetKey)
		}
		sort.Strings(targetKeys)
		for _, targetKey := range targetKeys {
			target := targets[targetKey].target
			if types.Implements(sourceType, target.contract) {
				pending = append(pending, target)
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
			selected[key].sourceType,
			selected[key].contract,
			key,
		)
		if err != nil {
			return nil, err
		}
		requests = append(requests, request)
		if demand, ok := r.interfaceReflectionDemands[selected[key].contractKey]; ok {
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
	return r.cacheInterfaceDemandRequests(cacheKey, requests), nil
}
