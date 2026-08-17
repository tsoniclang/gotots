package naming

import (
	"go/types"
	"sort"

	"github.com/tsoniclang/gotots/internal/emit/api"
)

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
		r.invalidateInterfaceDemandRequests()
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
	source interfaceContractSelection,
	reflectionType *types.TypeName,
) ([]api.RootRequest, error) {
	source, err := r.internInterfaceContract(source)
	if err != nil {
		return nil, err
	}
	valueKey := source.demandKey()
	if existing, ok := r.reflectionValueContracts[valueKey]; ok {
		if !sameInterfaceContractSelection(existing, source) {
			return nil, &api.NameError{
				Reason: "reflection value key joined non-identical contracts",
			}
		}
	} else {
		r.reflectionValueContracts[valueKey] = source
		r.invalidateInterfaceDemandRequests()
	}
	requests, err := r.recordInterfaceReflectionDemand(
		source.contractKey,
		source.contract,
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
		if !ok || !types.Implements(sourceType, source.contract) {
			continue
		}
		selected, selectedErr := r.interfaceAdapterContractRequests(
			binding,
			&source,
		)
		if selectedErr != nil {
			return nil, selectedErr
		}
		requests = append(requests, selected...)
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
		binding.reflectionName,
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
		if _, exists := r.reflectionValueDemands[binding.key]; !exists {
			r.reflectionValueDemands[binding.key] = struct{}{}
			r.invalidateInterfaceDemandRequests()
		}
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
	for _, contract := range r.reflectionValueContracts {
		if reached, ok := r.interfaceAdaptersByContract[contract.contractKey]; ok {
			if _, member := reached[binding.key]; member {
				return true
			}
		}
	}
	return false
}
