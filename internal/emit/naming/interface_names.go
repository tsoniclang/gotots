package naming

import (
	"go/types"

	"github.com/tsoniclang/gotots/internal/contracts/gostdlib"
	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/emit/type/typeidentity"
)

func (s interfaceContractSelection) valid() bool {
	if s.sourceType == nil || s.contract == nil ||
		s.contractKey == "" || s.surfaceKey == "" ||
		!s.contract.Complete().IsMethodSet() {
		return false
	}
	selected, ok := types.Unalias(s.sourceType).Underlying().(*types.Interface)
	return ok && selected.Complete().IsMethodSet() &&
		types.Identical(selected.Complete(), s.contract)
}

func (s interfaceContractSelection) demandKey() string {
	if !s.valid() {
		return ""
	}
	return s.contractKey + "\x00" + s.surfaceKey
}

func sameInterfaceContractSelection(
	left interfaceContractSelection,
	right interfaceContractSelection,
) bool {
	return left.contractKey == right.contractKey &&
		left.surfaceKey == right.surfaceKey &&
		types.Identical(left.sourceType, right.sourceType) &&
		types.Identical(left.contract, right.contract)
}

func (n *File) InterfaceAdapter(
	sourceType types.Type,
	targetType types.Type,
) (api.NameReference, error) {
	if !interfaceAdapterSource(sourceType) {
		return api.NameReference{}, &api.NameError{
			Name:   types.TypeString(sourceType, nil),
			Reason: "interface-adapter source type is invalid",
		}
	}
	artifactKey, err := typeidentity.BuildKey(
		sourceType,
		n.generatedNamedObjectIdentity,
	)
	if err != nil {
		return api.NameReference{}, err
	}
	name, err := n.semanticGeneratedTypeName(
		"$goInterfaceAdapter$",
		sourceType,
	)
	if err != nil {
		return api.NameReference{}, err
	}
	reflectionName, err := n.semanticGeneratedTypeName(
		"$goReflectType$",
		sourceType,
	)
	if err != nil {
		return api.NameReference{}, err
	}
	placement, err := n.generatedArtifactPlacement(sourceType)
	if err != nil {
		return api.NameReference{}, err
	}
	binding, err := n.owner.registry.internInterfaceAdapter(
		artifactKey,
		sourceType,
		name,
		reflectionName,
		placement,
	)
	if err != nil {
		return api.NameReference{}, err
	}
	requirement, err := api.NewInterfaceAdapterRequest(binding.owner)
	if err != nil {
		return api.NameReference{}, err
	}
	reference, err := n.generatedValueReference(
		binding.owner,
		binding.name,
		requirement,
		api.ArtifactFacetConstructorSurface,
	)
	if err != nil {
		return api.NameReference{}, err
	}
	var target *interfaceContractSelection
	if targetType != nil {
		selected, contractErr := n.canonicalInterfaceContract(targetType)
		err = contractErr
		if err != nil {
			return api.NameReference{}, err
		}
		target = &selected
	}
	demands, err := n.owner.registry.interfaceAdapterContractRequests(
		binding,
		target,
	)
	if err != nil {
		return api.NameReference{}, err
	}
	return reference.WithRequests(
		api.CombineRequests(reference.Requests(), demands)...,
	)
}

func (n *File) InterfaceContractDemand(
	sourceType types.Type,
	targetType types.Type,
) ([]api.RootRequest, error) {
	source, err := n.canonicalInterfaceContract(sourceType)
	if err != nil {
		return nil, err
	}
	target, err := n.canonicalInterfaceContract(targetType)
	if err != nil {
		return nil, err
	}
	return n.owner.registry.recordInterfaceContractDemand(
		source,
		target,
	)
}

func (n *File) InterfaceDynamicType(
	sourceType types.Type,
) (api.NameReference, error) {
	if !interfaceAdapterSource(sourceType) {
		return api.NameReference{}, &api.NameError{
			Name:   types.TypeString(sourceType, nil),
			Reason: "interface-dynamic-type source type is invalid",
		}
	}
	artifactKey, err := typeidentity.BuildKey(
		sourceType,
		n.generatedNamedObjectIdentity,
	)
	if err != nil {
		return api.NameReference{}, err
	}
	name, err := n.semanticGeneratedTypeName("$goDynamicType$", sourceType)
	if err != nil {
		return api.NameReference{}, err
	}
	placement, err := n.generatedArtifactPlacement(sourceType)
	if err != nil {
		return api.NameReference{}, err
	}
	binding, err := n.owner.registry.internInterfaceDynamicTypeToken(
		artifactKey,
		sourceType,
		name,
		placement,
	)
	if err != nil {
		return api.NameReference{}, err
	}
	requirement, err := api.NewInterfaceDynamicTypeTokenRequest(binding.owner)
	if err != nil {
		return api.NameReference{}, err
	}
	reference, err := n.generatedValueReference(
		binding.owner,
		binding.name,
		requirement,
		api.ArtifactFacetValueSurface,
	)
	return reference, err
}

func (n *File) ProviderInterfaceBridge(
	sourceType types.Type,
) (api.NameReference, bool, error) {
	named, ok := types.Unalias(sourceType).(*types.Named)
	if !ok || named.Obj() == nil {
		return api.NameReference{}, false, nil
	}
	contract, ok := named.Underlying().(*types.Interface)
	if !ok || !contract.Complete().IsMethodSet() {
		return api.NameReference{}, false, nil
	}
	providerInterface, providerOwned, err := n.owner.registry.ProviderInterface(
		named.Origin().Obj(),
	)
	if err != nil || !providerOwned {
		if err != nil {
			return api.NameReference{}, providerOwned, err
		}
		return api.NameReference{}, false, nil
	}
	if providerInterface.Mode() ==
		gostdlib.ProviderInterfaceModeSealedNative {
		return api.NameReference{}, false, nil
	}
	artifactKey, err := typeidentity.BuildKey(
		named,
		n.generatedNamedObjectIdentity,
	)
	if err != nil {
		return api.NameReference{}, true, err
	}
	name, err := n.semanticGeneratedTypeName(
		"$goProviderInterfaceBridge$",
		named,
	)
	if err != nil {
		return api.NameReference{}, true, err
	}
	binding, err := n.owner.registry.internProviderInterfaceBridge(
		artifactKey,
		named,
		name,
	)
	if err != nil {
		return api.NameReference{}, true, err
	}
	requirement, err := api.NewProviderInterfaceBridgeRequest(binding.owner)
	if err != nil {
		return api.NameReference{}, true, err
	}
	reference, err := n.generatedValueReference(
		binding.owner,
		binding.name,
		requirement,
		api.ArtifactFacetConstructorSurface,
	)
	if err != nil {
		return api.NameReference{}, true, err
	}
	base, err := n.canonicalInterfaceContract(named)
	if err != nil {
		return api.NameReference{}, true, err
	}
	demands, err := n.owner.registry.providerInterfaceBridgeContractRequests(
		binding,
		base.contractKey,
		base.contract,
	)
	if err != nil {
		return api.NameReference{}, true, err
	}
	reference, err = reference.WithRequests(
		api.CombineRequests(reference.Requests(), demands)...,
	)
	return reference, true, err
}

func (n *File) ProviderProfileInterfaceBridge(
	sourceType types.Type,
	profile []gostdlib.ProviderCallableProfileInterface,
) (api.ProviderProfileBridgeReference, bool, error) {
	named, ok := types.Unalias(sourceType).(*types.Named)
	if !ok || named.Obj() == nil || len(profile) == 0 {
		return api.ProviderProfileBridgeReference{}, false, nil
	}
	contract, ok := named.Underlying().(*types.Interface)
	if !ok || !contract.Complete().IsMethodSet() {
		return api.ProviderProfileBridgeReference{}, false, nil
	}
	sourceKey, err := typeidentity.BuildKey(
		named,
		n.generatedNamedObjectIdentity,
	)
	if err != nil {
		return api.ProviderProfileBridgeReference{}, true, err
	}
	name, err := n.semanticGeneratedTypeName(
		"$goProviderProfileBridge$",
		named,
	)
	if err != nil {
		return api.ProviderProfileBridgeReference{}, true, err
	}
	binding, err := n.owner.registry.internProviderProfileInterfaceBridge(
		sourceKey,
		named,
		name,
		profile,
	)
	if err != nil {
		return api.ProviderProfileBridgeReference{}, true, err
	}
	requirement, err := api.NewProviderInterfaceBridgeRequest(binding.owner)
	if err != nil {
		return api.ProviderProfileBridgeReference{}, true, err
	}
	bridge, err := n.generatedReference(
		binding.owner,
		binding.name,
		requirement,
		api.ArtifactFacetConstructorSurface,
		api.ImportPhaseValue,
	)
	if err != nil {
		return api.ProviderProfileBridgeReference{}, true, err
	}
	contractReference, err := n.generatedReference(
		binding.owner,
		binding.name+api.ProviderProfileContractSuffix,
		requirement,
		api.ArtifactFacetInstanceTypeSurface,
		api.ImportPhaseType,
	)
	if err != nil {
		return api.ProviderProfileBridgeReference{}, true, err
	}
	_, selectedProfile, profiled := binding.owner.ProviderProfileInterfaceBridge()
	if !profiled || len(selectedProfile) == 0 {
		return api.ProviderProfileBridgeReference{}, true, &api.NameError{
			Name:   binding.name,
			Reason: "provider-profile bridge artifact contract is absent",
		}
	}
	base, err := n.canonicalInterfaceContract(named)
	if err != nil {
		return api.ProviderProfileBridgeReference{}, true, err
	}
	demands, err := n.owner.registry.providerInterfaceBridgeContractRequests(
		binding,
		base.contractKey,
		base.contract,
	)
	if err != nil {
		return api.ProviderProfileBridgeReference{}, true, err
	}
	bridge, err = bridge.WithRequests(
		api.CombineRequests(bridge.Requests(), demands)...,
	)
	if err != nil {
		return api.ProviderProfileBridgeReference{}, true, err
	}
	reference, err := api.NewProviderProfileBridgeReference(
		bridge,
		contractReference,
		selectedProfile,
	)
	return reference, true, err
}

func (n *File) ProviderInterface(
	sourceType types.Type,
) (gostdlib.ProviderInterface, bool, error) {
	named, ok := types.Unalias(sourceType).(*types.Named)
	if !ok || named.Obj() == nil {
		return gostdlib.ProviderInterface{}, false, nil
	}
	return n.owner.registry.ProviderInterface(named.Origin().Obj())
}

func (n *File) InterfaceContract(
	sourceType types.Type,
) (api.InterfaceContractReference, error) {
	if typeName, interfaceType, ok := namedInterface(sourceType); ok {
		named := types.Unalias(sourceType).(*types.Named)
		if predeclaredError(typeName) ||
			named.TypeArgs().Len() != 0 ||
			n.providerInterfaceContract(typeName) {
			return n.generatedInterfaceContract(interfaceType)
		}
		return n.namedInterfaceContract(typeName)
	}
	interfaceType, ok := anonymousInterface(sourceType)
	if !ok {
		return api.InterfaceContractReference{}, &api.NameError{
			Reason: "interface-contract source type is invalid",
		}
	}
	return n.generatedInterfaceContract(interfaceType)
}

func (n *File) generatedInterfaceContract(
	interfaceType *types.Interface,
) (api.InterfaceContractReference, error) {
	artifactKey, err := typeidentity.BuildKey(
		interfaceType,
		n.generatedNamedObjectIdentity,
	)
	if err != nil {
		return api.InterfaceContractReference{}, err
	}
	name, err := n.semanticGeneratedTypeName(
		"$goInterface$",
		interfaceType,
	)
	if err != nil {
		return api.InterfaceContractReference{}, err
	}
	placement, err := n.generatedArtifactPlacement(interfaceType)
	if err != nil {
		return api.InterfaceContractReference{}, err
	}
	binding, err := n.owner.registry.internAnonymousInterface(
		artifactKey,
		interfaceType,
		name,
		placement,
	)
	if err != nil {
		return api.InterfaceContractReference{}, err
	}
	requirement, err := api.NewAnonymousInterfaceRequest(binding.owner)
	if err != nil {
		return api.InterfaceContractReference{}, err
	}
	requests := []api.RootRequest{requirement}
	if binding.owner.Placement() ==
		api.GeneratedArtifactPlacementCompilation {
		if n.artifactOwner.Valid() &&
			n.artifactOwner != api.MustGeneratedArtifactOwner(binding.owner) {
			for _, facet := range []api.ArtifactFacet{
				api.ArtifactFacetInstanceTypeSurface,
				api.ArtifactFacetValueSurface,
			} {
				dependency, dependencyErr :=
					api.NewGeneratedArtifactDependencyRequest(
						binding.owner,
						facet,
					)
				if dependencyErr != nil {
					return api.InterfaceContractReference{}, dependencyErr
				}
				requests = append(requests, dependency)
			}
		}
		if binding.owner.OutputPath() != n.targetPath {
			imports, importErr := n.interfaceContractImports(
				binding.owner,
				binding.name,
			)
			if importErr != nil {
				return api.InterfaceContractReference{}, importErr
			}
			requests = append(requests, imports.requests...)
			return api.NewInterfaceContractReference(
				imports.typeName,
				imports.contractName,
				imports.guardName,
				requests...,
			)
		}
	}
	return api.NewInterfaceContractReference(
		binding.name,
		interfaceContractName(binding.name),
		interfaceGuardName(binding.name),
		requests...,
	)
}

func (n *File) InterfaceType(
	sourceType types.Type,
) (api.NameReference, error) {
	if typeName, interfaceType, ok := namedInterface(sourceType); ok {
		if predeclaredError(typeName) {
			return n.generatedInterfaceType(interfaceType)
		}
		if n.providerInterfaceContract(typeName) {
			providerInterface, _, err :=
				n.owner.registry.ProviderInterface(typeName)
			if err != nil {
				return api.NameReference{}, err
			}
			if providerInterface.Mode() ==
				gostdlib.ProviderInterfaceModeSealedNative {
				return n.TypeReference(typeName)
			}
			return n.generatedInterfaceType(interfaceType)
		}
		return n.TypeReference(typeName)
	}
	interfaceType, ok := anonymousInterface(sourceType)
	if !ok {
		return api.NameReference{}, &api.NameError{
			Name: types.TypeString(
				sourceType,
				func(sourcePackage *types.Package) string {
					return sourcePackage.Path()
				},
			),
			Reason: "interface type is invalid",
		}
	}
	return n.generatedInterfaceType(interfaceType)
}

func (n *File) generatedInterfaceType(
	interfaceType *types.Interface,
) (api.NameReference, error) {
	artifactKey, err := typeidentity.BuildKey(
		interfaceType,
		n.generatedNamedObjectIdentity,
	)
	if err != nil {
		return api.NameReference{}, err
	}
	name, err := n.semanticGeneratedTypeName(
		"$goInterface$",
		interfaceType,
	)
	if err != nil {
		return api.NameReference{}, err
	}
	placement, err := n.generatedArtifactPlacement(interfaceType)
	if err != nil {
		return api.NameReference{}, err
	}
	binding, err := n.owner.registry.internAnonymousInterface(
		artifactKey,
		interfaceType,
		name,
		placement,
	)
	if err != nil {
		return api.NameReference{}, err
	}
	requirement, err := api.NewAnonymousInterfaceRequest(binding.owner)
	if err != nil {
		return api.NameReference{}, err
	}
	return n.generatedReference(
		binding.owner,
		binding.name,
		requirement,
		api.ArtifactFacetInstanceTypeSurface,
		api.ImportPhaseType,
	)
}

func (n *File) providerInterfaceContract(typeName *types.TypeName) bool {
	if n == nil || n.owner == nil || n.owner.registry == nil || typeName == nil {
		return false
	}
	binding, ok := n.owner.registry.byObject[typeName]
	return ok && (binding.kind == targetBindingProvider ||
		binding.kind == targetBindingMissingProvider)
}

func (n *File) namedInterfaceContract(
	typeName *types.TypeName,
) (api.InterfaceContractReference, error) {
	typeReference, err := n.TypeReference(typeName)
	if err != nil {
		return api.InterfaceContractReference{}, err
	}
	contract, err := n.derivedSourceReference(
		typeName,
		api.InterfaceContractSuffix,
		api.ArtifactFacetValueSurface,
	)
	if err != nil {
		return api.InterfaceContractReference{}, err
	}
	guard, err := n.derivedSourceReference(
		typeName,
		api.InterfaceGuardSuffix,
		api.ArtifactFacetValueSurface,
	)
	if err != nil {
		return api.InterfaceContractReference{}, err
	}
	return api.NewInterfaceContractReference(
		typeReference.Name(),
		contract.Name(),
		guard.Name(),
		api.CombineRequests(
			typeReference.Requests(),
			contract.Requests(),
			guard.Requests(),
		)...,
	)
}
