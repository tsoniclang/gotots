package naming

import (
	"encoding/base64"
	"encoding/hex"
	"go/types"
	"sort"

	environmentcontract "github.com/tsoniclang/gotots/internal/contracts/environment"
	"github.com/tsoniclang/gotots/internal/contracts/gostdlib"
	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/emit/type/typeidentity"
	"github.com/tsoniclang/gotots/internal/output"
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

const reflectionMethodIdentityHexLength = 20

// ReflectionMethodIdentity returns the compact collision-checked encoding of
// one canonical interface-method identity. The same token binding owner used
// by interface contracts reserves the truncated identity before it is
// encoded, so two distinct methods cannot silently share an identity.
func (n *File) ReflectionMethodIdentity(
	method *types.Func,
) (string, error) {
	runtime, _ := runtimeInterfaceMethodToken(method)
	binding, err := n.interfaceMethodTokenBinding(method, runtime)
	if err != nil {
		return "", err
	}
	key := binding.owner.ArtifactKey()
	if len(key) < reflectionMethodIdentityHexLength {
		return "", &api.NameError{
			Name:   method.Name(),
			Reason: "interface method identity is too short",
		}
	}
	decoded, err := hex.DecodeString(key[:reflectionMethodIdentityHexLength])
	if err != nil {
		return "", &api.NameError{
			Name:   method.Name(),
			Reason: "interface method identity is not hexadecimal",
		}
	}
	return base64.RawURLEncoding.EncodeToString(decoded), nil
}

func (n *File) ReflectionType(
	sourceType types.Type,
	reflectionType *types.TypeName,
) (api.NameReference, error) {
	if sourceType == nil || reflectionType == nil {
		return api.NameReference{}, &api.NameError{
			Reason: "reflection-type identity is invalid",
		}
	}
	artifactKey, err := typeidentity.BuildKey(
		sourceType,
		n.generatedNamedObjectIdentity,
	)
	if err != nil {
		return api.NameReference{}, err
	}
	name, err := n.semanticGeneratedTypeName("$goReflectType$", sourceType)
	if err != nil {
		return api.NameReference{}, err
	}
	binding, err := n.owner.registry.internReflectionType(
		artifactKey,
		sourceType,
		reflectionType,
		name,
	)
	if err != nil {
		return api.NameReference{}, err
	}
	request, err := api.NewReflectionTypeRequest(binding.owner)
	if err != nil {
		return api.NameReference{}, err
	}
	return n.generatedValueReference(
		binding.owner,
		binding.name,
		request,
		api.ArtifactFacetValueSurface,
	)
}

func (n *File) ReflectionOperations(
	reflectionType *types.TypeName,
) (api.NameReference, error) {
	reference, providerOwned, err := n.providerFacetReference(
		reflectionType,
		gostdlib.FacetReflectionTypeOperations,
		gostdlib.FacetCapabilityMetadata,
		api.ImportPhaseValue,
	)
	if err != nil {
		return api.NameReference{}, err
	}
	if !providerOwned {
		return api.NameReference{}, &api.NameError{
			Name:   reflectionType.Name(),
			Reason: "reflection type has no certified metadata operations",
		}
	}
	return reference, nil
}

func (n *File) ReflectionDescriptorType(
	reflectionType *types.TypeName,
) (api.NameReference, error) {
	reference, providerOwned, err := n.providerFacetResultReference(
		reflectionType,
		gostdlib.FacetReflectionTypeOperations,
		gostdlib.FacetCapabilityMetadata,
		api.ImportPhaseType,
	)
	if err != nil {
		return api.NameReference{}, err
	}
	if !providerOwned {
		return api.NameReference{}, &api.NameError{
			Name:   reflectionType.Name(),
			Reason: "reflection type has no certified descriptor result type",
		}
	}
	return reference, nil
}

func (n *File) ReflectionTypeOf(
	argumentType types.Type,
	reflectionType *types.TypeName,
) (api.NameReference, error) {
	if argumentType == nil || reflectionType == nil || reflectionType.IsAlias() {
		return api.NameReference{}, &api.NameError{
			Reason: "reflection TypeOf contract is invalid",
		}
	}
	registry := n.owner.registry
	operations, err := n.ReflectionOperations(reflectionType)
	if err != nil {
		return api.NameReference{}, err
	}
	staticType, err := n.ReflectionType(argumentType, reflectionType)
	if err != nil {
		return api.NameReference{}, err
	}
	readiness := staticType.Requests()
	if _, isInterface := types.Unalias(argumentType).Underlying().(*types.Interface); isInterface {
		contract, contractErr := n.canonicalInterfaceContract(argumentType)
		if contractErr != nil {
			return api.NameReference{}, contractErr
		}
		dynamicReadiness, demandErr := registry.recordInterfaceReflectionDemand(
			contract.contractKey,
			contract.contract,
			reflectionType,
		)
		if demandErr != nil {
			return api.NameReference{}, demandErr
		}
		readiness = api.CombineRequests(readiness, dynamicReadiness)
	}
	modulePath, err := output.ModuleSpecifier(
		n.targetPath,
		output.ReflectionTypeSupportPath,
	)
	if err != nil {
		return api.NameReference{}, err
	}
	initialize, err := api.NewSideEffectImportRequest(n.factory, modulePath)
	if err != nil {
		return api.NameReference{}, err
	}
	requests := api.CombineRequests(
		operations.Requests(),
		readiness,
		[]api.RootRequest{initialize},
	)
	return operations.WithRequests(api.CombineRequests(requests)...)
}

// ReflectionValueOf demands the canonical descriptor plus the generated
// value-operation facet for one reflected operand type. Interface operands
// subscribe the canonical contract so every concrete adapter reaching the
// contract, before or after this observation, demands its value facet.
func (n *File) ReflectionValueOf(
	argumentType types.Type,
	reflectionType *types.TypeName,
) (api.NameReference, error) {
	if argumentType == nil || reflectionType == nil || reflectionType.IsAlias() {
		return api.NameReference{}, &api.NameError{
			Reason: "reflection ValueOf contract is invalid",
		}
	}
	registry := n.owner.registry
	operations, err := n.ReflectionOperations(reflectionType)
	if err != nil {
		return api.NameReference{}, err
	}
	requests := operations.Requests()
	if _, isInterface := types.Unalias(argumentType).Underlying().(*types.Interface); isInterface {
		contract, contractErr := n.canonicalInterfaceContract(argumentType)
		if contractErr != nil {
			return api.NameReference{}, contractErr
		}
		dynamicReadiness, demandErr := registry.recordReflectionValueContract(
			contract,
			reflectionType,
		)
		if demandErr != nil {
			return api.NameReference{}, demandErr
		}
		requests = api.CombineRequests(requests, dynamicReadiness)
	} else {
		artifactKey, keyErr := typeidentity.BuildKey(
			argumentType,
			n.generatedNamedObjectIdentity,
		)
		if keyErr != nil {
			return api.NameReference{}, keyErr
		}
		if _, exists := registry.reflectionValueDemands[artifactKey]; !exists {
			registry.reflectionValueDemands[artifactKey] = struct{}{}
			registry.invalidateInterfaceDemandRequests()
		}
		staticType, typeErr := n.ReflectionType(argumentType, reflectionType)
		if typeErr != nil {
			return api.NameReference{}, typeErr
		}
		facet, facetErr := registry.reflectionValueOperationsRequest(artifactKey)
		if facetErr != nil {
			return api.NameReference{}, facetErr
		}
		requests = api.CombineRequests(
			requests,
			staticType.Requests(),
			[]api.RootRequest{facet},
		)
	}
	modulePath, err := output.ModuleSpecifier(
		n.targetPath,
		output.ReflectionTypeSupportPath,
	)
	if err != nil {
		return api.NameReference{}, err
	}
	initialize, err := api.NewSideEffectImportRequest(n.factory, modulePath)
	if err != nil {
		return api.NameReference{}, err
	}
	requests = api.CombineRequests(requests, []api.RootRequest{initialize})
	return operations.WithRequests(requests...)
}

// ReflectionValueOperationsDemanded reports whether the value-operation
// facet was demanded for one canonical reflection artifact.
func (n *File) ReflectionValueOperationsDemanded(artifactKey string) bool {
	_, demanded := n.owner.registry.reflectionValueDemands[artifactKey]
	return demanded
}

// ReflectionValueType returns the canonical descriptor reference for one
// type while joining its value-operation facet demand, closing the value
// metadata over navigable child types (fields and pointees). The distinct
// value-operation requirement requeues a descriptor that was already
// constructed before this demand arrived.
func (n *File) ReflectionValueType(
	sourceType types.Type,
	reflectionType *types.TypeName,
) (api.NameReference, error) {
	var dynamicReadiness []api.RootRequest
	if _, isInterface := types.Unalias(sourceType).Underlying().(*types.Interface); isInterface {
		contract, contractErr := n.canonicalInterfaceContract(sourceType)
		if contractErr != nil {
			return api.NameReference{}, contractErr
		}
		var demandErr error
		dynamicReadiness, demandErr =
			n.owner.registry.recordReflectionValueContract(
				contract,
				reflectionType,
			)
		if demandErr != nil {
			return api.NameReference{}, demandErr
		}
	}
	artifactKey, err := typeidentity.BuildKey(
		sourceType,
		n.generatedNamedObjectIdentity,
	)
	if err != nil {
		return api.NameReference{}, err
	}
	if _, exists := n.owner.registry.reflectionValueDemands[artifactKey]; !exists {
		n.owner.registry.reflectionValueDemands[artifactKey] = struct{}{}
		n.owner.registry.invalidateInterfaceDemandRequests()
	}
	reference, err := n.ReflectionType(sourceType, reflectionType)
	if err != nil {
		return api.NameReference{}, err
	}
	facet, err := n.owner.registry.reflectionValueOperationsRequest(
		artifactKey,
	)
	if err != nil {
		return api.NameReference{}, err
	}
	return reference.WithRequests(api.CombineRequests(
		reference.Requests(),
		dynamicReadiness,
		[]api.RootRequest{facet},
	)...)
}

// ProviderOwnedDeclaration reports whether one declaration's truth is a
// certified provider facet, which excludes its class internals from the
// generated location model.
func (n *File) ProviderOwnedDeclaration(
	object types.Object,
) (bool, error) {
	_, owned, err := n.providerFacetOwner(object)
	return owned, err
}

// reflectionValueOperationsRequest builds the value-operation facet
// requirement of one interned canonical descriptor.
func (r *Registry) reflectionValueOperationsRequest(
	artifactKey string,
) (api.RootRequest, error) {
	binding, ok := r.reflectionTypes[artifactKey]
	if !ok || binding.owner == nil {
		return api.RootRequest{}, &api.NameError{
			Reason: "reflection value demand has no interned descriptor",
		}
	}
	return api.NewReflectionValueOperationsRequest(binding.owner)
}

func (n *File) ReflectionInterfaceAdapter(sourceType types.Type) (api.NameReference, error) {
	reference, err := n.InterfaceAdapter(sourceType, nil)
	if err != nil {
		return api.NameReference{}, err
	}
	artifactKey, err := typeidentity.BuildKey(sourceType, n.generatedNamedObjectIdentity)
	if err != nil {
		return api.NameReference{}, err
	}
	binding, ok := n.owner.registry.interfaceAdapters[artifactKey]
	if !ok || binding.owner == nil {
		return api.NameReference{}, &api.NameError{
			Name:   types.TypeString(sourceType, nil),
			Reason: "reflection interface adapter was not canonicalized",
		}
	}
	empty, err := n.canonicalInterfaceContract(types.Universe.Lookup("any").Type())
	if err != nil {
		return api.NameReference{}, err
	}
	if err := n.owner.registry.recordReflectionInterfaceAdapter(empty, binding); err != nil {
		return api.NameReference{}, err
	}
	return reference, nil
}

func (registry *Registry) observeReflectionRawPointerUse(object types.Object) error {
	method, callable := object.(*types.Func)
	if !callable || method.Pkg() == nil || method.Type().(*types.Signature).Recv() == nil {
		return nil
	}
	contract, err := environmentcontract.Describe(method.Origin())
	if err != nil {
		return err
	}
	if contract.Identity() == "reflect|kind=4|receiver=reflect.Value|name=UnsafePointer" {
		registry.reflectionRawPointerSelected = true
	}
	return nil
}

func (names *File) ReflectionRawPointerDemanded() bool {
	return names.owner.registry.reflectionRawPointerSelected
}

func (registry *Registry) FlushReflectionRawPointerDemands() ([]api.RootRequest, error) {
	if !registry.reflectionRawPointerSelected {
		return nil, nil
	}
	if registry.reflectionRawPointerDelivered == nil {
		registry.reflectionRawPointerDelivered = make(map[string]struct{})
	}
	keys := make([]string, 0, len(registry.reflectionValueDemands))
	for key := range registry.reflectionValueDemands {
		if _, delivered := registry.reflectionRawPointerDelivered[key]; !delivered {
			keys = append(keys, key)
		}
	}
	sort.Strings(keys)
	var requests []api.RootRequest
	for _, key := range keys {
		binding, exists := registry.reflectionTypes[key]
		if !exists || binding.owner == nil {
			return nil, &api.NameError{Reason: "reflection raw-pointer demand has no descriptor"}
		}
		source, _, valid := binding.owner.ReflectionType()
		if !valid {
			return nil, &api.NameError{Reason: "reflection raw-pointer descriptor has no source type"}
		}
		if _, pointer := source.Underlying().(*types.Pointer); pointer {
			request, err := api.NewReflectionRawPointerRequest(binding.owner)
			if err != nil {
				return nil, err
			}
			requests = append(requests, request)
		}
		registry.reflectionRawPointerDelivered[key] = struct{}{}
	}
	return requests, nil
}
