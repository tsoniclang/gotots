package naming

import (
	"go/types"
	"strconv"

	"github.com/tsoniclang/gotots/internal/contracts/gostdlib"
	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/emit/type/typeidentity"
)

type providerImport struct {
	local string
}

func (n *File) providerImport(
	module string,
	phase api.ImportPhase,
) (string, api.RootRequest, error) {
	if module == "" {
		return "", api.RootRequest{}, &api.NameError{
			Reason: "provider import identity is empty",
		}
	}
	selected := n.providerImports[module]
	if selected.local == "" {
		preferred := n.owner.registry.providerImportNameByModule[module]
		if preferred == "" {
			return "", api.RootRequest{}, &api.NameError{
				Name:   module,
				Reason: "provider module has no preallocated import name",
			}
		}
		selected = providerImport{
			local: n.allocateProviderImportName(preferred),
		}
		n.providerImports[module] = selected
	}
	request, err := api.NewNamespaceImportRequest(
		n.factory,
		phase,
		module,
		selected.local,
	)
	if err != nil {
		return "", api.RootRequest{}, err
	}
	return selected.local, request, nil
}

func (n *File) ExternalProviderFunction(
	module string,
	export string,
) (api.NameReference, error) {
	if export == "" {
		return api.NameReference{}, &api.NameError{
			Reason: "external provider export is empty",
		}
	}
	qualifier, request, err := n.providerImport(
		module,
		api.ImportPhaseValue,
	)
	if err != nil {
		return api.NameReference{}, err
	}
	return api.NewQualifiedNameReference(qualifier, export, request)
}

func (n *File) providerFacetReference(
	object types.Object,
	kind gostdlib.FacetKind,
	capability gostdlib.FacetCapability,
	phase api.ImportPhase,
) (api.NameReference, bool, error) {
	return n.providerFacetTargetReference(
		object,
		kind,
		capability,
		phase,
		false,
	)
}

func (n *File) providerFacetStorageReference(
	object types.Object,
	kind gostdlib.FacetKind,
	capability gostdlib.FacetCapability,
	phase api.ImportPhase,
) (api.NameReference, bool, error) {
	return n.providerFacetTargetReference(
		object,
		kind,
		capability,
		phase,
		true,
	)
}

func (n *File) providerFacetTargetReference(
	object types.Object,
	kind gostdlib.FacetKind,
	capability gostdlib.FacetCapability,
	phase api.ImportPhase,
	storage bool,
) (api.NameReference, bool, error) {
	contract, providerOwned, err := n.providerFacetOwner(object)
	if err != nil || !providerOwned {
		return api.NameReference{}, providerOwned, err
	}
	selected, ok := n.owner.registry.provider.Facet(
		contract.Identity(),
		kind,
		capability,
	)
	if !ok {
		return api.NameReference{}, true, &api.NameError{
			Name: contract.Identity(),
			Reason: "selected standard-library declaration has no certified provider facet for capability " +
				strconv.Quote(string(capability)),
		}
	}
	if selected.SourceIdentity() != contract.Identity() ||
		selected.Kind() != kind {
		return api.NameReference{}, true, &api.NameError{
			Name:   contract.Identity(),
			Reason: "provider facet does not match its selected source owner",
		}
	}
	export := selected.Export()
	if storage {
		export = selected.StorageExport()
		if export == "" {
			return api.NameReference{}, true, &api.NameError{
				Name:   contract.Identity(),
				Reason: "provider facet has no certified storage target",
			}
		}
	}
	qualifier, request, err := n.providerImport(
		selected.ModuleSpecifier(),
		phase,
	)
	if err != nil {
		return api.NameReference{}, true, err
	}
	if n.require != nil {
		if err := n.require(object); err != nil {
			return api.NameReference{}, true, err
		}
	}
	reference, err := api.NewProviderQualifiedNameReference(
		qualifier,
		export,
		request,
	)
	return reference, true, err
}

func (n *File) ProviderCallableProfile(
	owner *types.Func,
	profileKey string,
) (api.ProviderCallableProfileReference, bool, error) {
	if owner == nil || profileKey == "" {
		return api.ProviderCallableProfileReference{}, false, &api.NameError{
			Reason: "provider callable-profile identity is invalid",
		}
	}
	owner = owner.Origin()
	contract, providerOwned, err := n.providerFacetOwner(owner)
	if err != nil || !providerOwned {
		return api.ProviderCallableProfileReference{}, providerOwned, err
	}
	selected, ok := n.owner.registry.provider.ProviderCallableProfile(
		contract.Identity(),
		profileKey,
	)
	if !ok {
		return api.ProviderCallableProfileReference{}, true, &api.NameError{
			Name: contract.Identity(),
			Reason: "selected standard-library callable has no certified boundary profile " +
				strconv.Quote(profileKey),
		}
	}
	if !selected.Valid() || selected.SourceIdentity() != contract.Identity() ||
		selected.ProfileKey() != profileKey {
		return api.ProviderCallableProfileReference{}, true, &api.NameError{
			Name:   contract.Identity(),
			Reason: "provider callable-profile certificate is inconsistent",
		}
	}
	qualifier, request, err := n.providerImport(
		selected.ModuleSpecifier(),
		api.ImportPhaseValue,
	)
	if err != nil {
		return api.ProviderCallableProfileReference{}, true, err
	}
	if n.require != nil {
		if err := n.require(owner); err != nil {
			return api.ProviderCallableProfileReference{}, true, err
		}
	}
	reference, err := api.NewQualifiedNameReference(
		qualifier,
		selected.Export(),
		request,
	)
	if err != nil {
		return api.ProviderCallableProfileReference{}, true, err
	}
	guards, guardRequests, err := n.providerCallableProfileGuards(
		owner,
		selected,
		true,
	)
	if err != nil {
		return api.ProviderCallableProfileReference{}, true, err
	}
	capabilityViews, err := n.providerCallableProfileCapabilityViews(selected)
	if err != nil {
		return api.ProviderCallableProfileReference{}, true, err
	}
	reference, err = reference.WithRequests(
		api.CombineRequests(reference.Requests(), guardRequests)...,
	)
	if err != nil {
		return api.ProviderCallableProfileReference{}, true, err
	}
	contracts, err := n.providerProfileInterfaceTypes(
		selected.ContractInterfaces(),
	)
	if err != nil {
		return api.ProviderCallableProfileReference{}, true, err
	}
	fromProvider, err := n.providerCallableProfileBridges(selected)
	if err != nil {
		return api.ProviderCallableProfileReference{}, true, err
	}
	values, err := n.providerCallableProfileValues(selected)
	if err != nil {
		return api.ProviderCallableProfileReference{}, true, err
	}
	typeArguments, err := n.providerProfileInterfaceTypes(
		selected.CanonicalTypeArguments(),
	)
	if err != nil {
		return api.ProviderCallableProfileReference{}, true, err
	}
	result, err := api.NewProviderCallableProfileReference(
		reference,
		selected,
		capabilityViews,
		guards,
		contracts,
		fromProvider,
		values,
		typeArguments,
	)
	return result, true, err
}

func (n *File) providerCallableProfileCapabilityViews(
	profile gostdlib.ProviderCallableProfile,
) ([]api.NameReference, error) {
	capabilities, err := n.providerCallableProfileCapabilityTypes(profile)
	if err != nil {
		return nil, err
	}
	result := make([]api.NameReference, 0, len(capabilities))
	for _, capability := range capabilities {
		adapterDemands, err := n.InterfaceContractDemand(
			capability.Base(),
			capability.Target(),
		)
		if err != nil {
			return nil, err
		}
		baseKey, err := typeidentity.BuildKey(
			capability.Base(),
			n.generatedNamedObjectIdentity,
		)
		if err != nil {
			return nil, err
		}
		binding, err := n.owner.registry.internProviderProfileInterfaceBridge(
			baseKey,
			capability.Base(),
			profile.Interfaces(),
		)
		if err != nil {
			return nil, err
		}
		targetKey, err := typeidentity.BuildKey(
			capability.Target(),
			n.generatedNamedObjectIdentity,
		)
		if err != nil {
			return nil, err
		}
		targetBinding, err := n.owner.registry.internProviderProfileInterfaceBridge(
			targetKey,
			capability.Target(),
			profile.Interfaces(),
		)
		if err != nil {
			return nil, err
		}
		definition, err := api.NewProviderInterfaceBridgeRequest(binding.owner)
		if err != nil {
			return nil, err
		}
		demand, err := api.NewProviderProfileInterfaceCapabilityRequest(
			binding.owner,
			targetBinding.owner,
		)
		if err != nil {
			return nil, err
		}
		name, err := api.ProviderProfileCapabilityName(
			binding.name,
			targetBinding.key,
		)
		if err != nil {
			return nil, err
		}
		reference, err := n.generatedValueReference(
			binding.owner,
			name,
			definition,
			api.ArtifactFacetExportSurface,
		)
		if err != nil {
			return nil, err
		}
		reference, err = reference.WithRequests(
			api.CombineRequests(
				reference.Requests(),
				adapterDemands,
				[]api.RootRequest{demand},
			)...,
		)
		if err != nil {
			return nil, err
		}
		result = append(result, reference)
	}
	return result, nil
}

func (n *File) providerCallableProfileBridges(
	profile gostdlib.ProviderCallableProfile,
) ([]api.NameReference, error) {
	identities := profile.FromProviderInterfaces()
	result := make([]api.NameReference, 0, len(identities))
	for _, identity := range identities {
		source, err := n.providerProfileInterfaceType(identity)
		if err != nil {
			return nil, err
		}
		bridge, provider, err := n.ProviderInterfaceBridge(source)
		if err != nil {
			return nil, err
		}
		if !provider {
			return nil, &api.NameError{
				Name:   identity,
				Reason: "from-provider interface has no certified bridge",
			}
		}
		result = append(result, bridge)
	}
	return result, nil
}

func (n *File) providerProfileInterfaceTypes(
	identities []string,
) ([]types.Type, error) {
	result := make([]types.Type, 0, len(identities))
	for _, identity := range identities {
		selected, err := n.providerProfileInterfaceType(identity)
		if err != nil {
			return nil, err
		}
		result = append(result, selected)
	}
	return result, nil
}

func (n *File) providerProfileInterfaceType(identity string) (types.Type, error) {
	object := n.owner.registry.providerObjectByIdentity[identity]
	if identity == gostdlib.LanguageErrorInterfaceIdentity {
		object = types.Universe.Lookup("error")
	}
	typeName, ok := object.(*types.TypeName)
	if !ok || typeName.IsAlias() {
		return nil, &api.NameError{
			Name:   identity,
			Reason: "provider profile interface has no exact source type",
		}
	}
	if _, ok := types.Unalias(typeName.Type()).Underlying().(*types.Interface); !ok {
		return nil, &api.NameError{
			Name:   identity,
			Reason: "provider profile source type is not an interface",
		}
	}
	return typeName.Type(), nil
}

func (n *File) ProviderStatefulProfileCandidates(
	owner *types.TypeName,
) ([]api.ProviderStatefulProfileCandidate, bool, error) {
	if owner == nil || owner.IsAlias() {
		return nil, false, &api.NameError{
			Reason: "provider stateful-profile owner is invalid",
		}
	}
	contract, providerOwned, err := n.providerFacetOwner(owner)
	if err != nil || !providerOwned {
		return nil, false, err
	}
	profiles := n.owner.registry.provider.ProviderStatefulProfiles(
		contract.Identity(),
	)
	if len(profiles) == 0 {
		return nil, false, nil
	}
	result := make([]api.ProviderStatefulProfileCandidate, 0, len(profiles))
	for _, profile := range profiles {
		if !profile.Valid() || profile.SourceIdentity() != contract.Identity() {
			return nil, true, &api.NameError{
				Name:   contract.Identity(),
				Reason: "provider stateful-profile candidate is inconsistent",
			}
		}
		typeArguments, interfaceErr := n.providerProfileInterfaceTypes(
			profile.TypeArguments(),
		)
		if interfaceErr != nil {
			return nil, true, interfaceErr
		}
		candidate, candidateErr := api.NewProviderStatefulProfileCandidate(
			profile,
			typeArguments,
		)
		if candidateErr != nil {
			return nil, true, candidateErr
		}
		result = append(result, candidate)
	}
	return result, true, nil
}

func (n *File) ProviderStatefulProfileTarget(
	owner *types.TypeName,
	profileKey string,
	phase api.ImportPhase,
) (api.NameReference, error) {
	if owner == nil || owner.IsAlias() ||
		(phase != api.ImportPhaseType && phase != api.ImportPhaseValue) {
		return api.NameReference{}, &api.NameError{
			Reason: "provider stateful-profile target is invalid",
		}
	}
	contract, providerOwned, err := n.providerFacetOwner(owner)
	if err != nil {
		return api.NameReference{}, err
	}
	if !providerOwned {
		return api.NameReference{}, &api.NameError{
			Name:   owner.Name(),
			Reason: "stateful-profile target is not provider-owned",
		}
	}
	module := ""
	export := ""
	if profileKey == "" {
		binding := n.owner.registry.byObject[owner]
		if binding.kind != targetBindingProvider ||
			binding.providerMember != "" ||
			binding.providerModule == "" ||
			binding.providerExport == "" {
			return api.NameReference{}, &api.NameError{
				Name:   contract.Identity(),
				Reason: "ordinary provider representation target is invalid",
			}
		}
		module = binding.providerModule
		export = binding.providerExport
	} else {
		profile, ok := n.owner.registry.provider.ProviderStatefulProfile(
			contract.Identity(),
			profileKey,
		)
		if !ok || !profile.Valid() ||
			profile.SourceIdentity() != contract.Identity() ||
			profile.ProfileKey() != profileKey {
			return api.NameReference{}, &api.NameError{
				Name:   contract.Identity(),
				Reason: "selected provider stateful profile is absent",
			}
		}
		module = profile.ModuleSpecifier()
		export = profile.Export()
	}
	if n.require != nil {
		if err := n.require(owner); err != nil {
			return api.NameReference{}, err
		}
	}
	qualifier, request, err := n.providerImport(module, phase)
	if err != nil {
		return api.NameReference{}, err
	}
	return api.NewProviderQualifiedNameReference(
		qualifier,
		export,
		request,
	)
}
