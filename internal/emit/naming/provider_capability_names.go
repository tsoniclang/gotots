package naming

import (
	"github.com/tsoniclang/gotots/internal/contracts/gostdlib"
	gostdlibsource "github.com/tsoniclang/gotots/internal/contracts/gostdlib/sourcecontract"
	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/emit/type/typeidentity"
	"github.com/tsoniclang/gotots/internal/output"
	"go/types"
)

func (n *File) providerStatefulOperation(
	owner *types.TypeName,
	capability gostdlib.FacetCapability,
	phase api.ImportPhase,
	facet api.ArtifactFacet,
) (api.NameReference, bool, error) {
	candidates, profiled, err := n.ProviderStatefulProfileCandidates(owner)
	if err != nil || !profiled {
		return api.NameReference{}, profiled, err
	}
	for _, candidate := range candidates {
		if !candidate.Profile().SupportsOperation(capability) {
			return api.NameReference{}, true, &api.NameError{
				Name: owner.Name(),
				Reason: "provider stateful profile omits named-struct operation " +
					string(capability),
			}
		}
	}
	reference, selected, err := n.providerStatefulRepresentation(
		owner,
		phase,
		facet,
	)
	if err != nil || !selected {
		return api.NameReference{}, true, err
	}
	return reference, true, nil
}

func (n *File) providerStatefulRepresentation(
	owner *types.TypeName,
	phase api.ImportPhase,
	_ api.ArtifactFacet,
) (api.NameReference, bool, error) {
	if owner == nil || owner.IsAlias() {
		return api.NameReference{}, false, nil
	}
	named, ok := types.Unalias(owner.Type()).(*types.Named)
	if !ok || named.Obj() == nil {
		return api.NameReference{}, false, nil
	}
	_, profiled, err := n.ProviderStatefulProfileCandidates(owner)
	if err != nil || !profiled {
		return api.NameReference{}, profiled, err
	}
	artifactKey, err := typeidentity.BuildKey(
		named,
		n.generatedNamedObjectIdentity,
	)
	if err != nil {
		return api.NameReference{}, true, err
	}
	name, err := n.semanticGeneratedTypeName("$goProviderState$", named)
	if err != nil {
		return api.NameReference{}, true, err
	}
	binding, err := n.owner.registry.internProviderStatefulRepresentation(
		artifactKey,
		named,
		name,
	)
	if err != nil {
		return api.NameReference{}, true, err
	}
	requirement, err := api.NewProviderStatefulRepresentationRequest(binding.owner)
	if err != nil {
		return api.NameReference{}, true, err
	}
	facet := api.ArtifactFacetInstanceTypeSurface
	if phase == api.ImportPhaseValue {
		facet = api.ArtifactFacetValueSurface
	}
	reference, err := n.generatedReference(
		binding.owner,
		binding.name,
		requirement,
		facet,
		phase,
	)
	return reference, true, err
}

func (r *Registry) internProviderStatefulRepresentation(
	artifactKey string,
	sourceType *types.Named,
	name string,
) (providerStatefulRepresentationBinding, error) {
	if r == nil || sourceType == nil || sourceType.Obj() == nil ||
		artifactKey == "" || name == "" {
		return providerStatefulRepresentationBinding{}, &api.NameError{
			Reason: "provider stateful-representation canonicalization input is invalid",
		}
	}
	if _, interfaceType := sourceType.Underlying().(*types.Interface); interfaceType {
		return providerStatefulRepresentationBinding{}, &api.NameError{
			Name:   sourceType.Obj().Name(),
			Reason: "provider stateful-representation source is an interface",
		}
	}
	if existing, found := r.providerStatefulRepresentations[artifactKey]; found {
		existingType, valid := existing.owner.ProviderStatefulRepresentationType()
		if !valid || !types.Identical(existingType, sourceType) {
			return providerStatefulRepresentationBinding{}, &api.NameError{
				Name:   existing.name,
				Reason: "provider stateful-representation key joined non-identical Go types",
			}
		}
		return existing, nil
	}
	if err := reserveGeneratedName(
		r.providerStatefulRepresentationNames,
		name,
		artifactKey,
		"provider stateful-representation",
	); err != nil {
		return providerStatefulRepresentationBinding{}, err
	}
	owner, err := api.NewCompilationGeneratedArtifact(
		api.GeneratedArtifactProviderStatefulRepresentation,
		sourceType,
		artifactKey,
		name,
		output.ProviderStatefulRepresentationSupportPath,
	)
	if err != nil {
		return providerStatefulRepresentationBinding{}, err
	}
	binding := providerStatefulRepresentationBinding{owner: owner, name: name}
	r.providerStatefulRepresentations[artifactKey] = binding
	return binding, nil
}

func (r *Registry) indexProviderInterfaceCapabilities() error {
	if r == nil || r.provider == nil || !r.provider.Valid() {
		return &api.NameError{
			Reason: "provider-interface capability certificate is invalid",
		}
	}
	baseIdentities := make([]string, 0, len(r.providerObjectByIdentity)+1)
	baseIdentities = append(baseIdentities, gostdlib.LanguageErrorInterfaceIdentity)
	for identity, object := range r.providerObjectByIdentity {
		if identity != gostdlib.LanguageErrorInterfaceIdentity {
			if _, ok := object.(*types.TypeName); ok {
				baseIdentities = append(baseIdentities, identity)
			}
		}
	}
	for _, baseIdentity := range baseIdentities {
		capabilities := r.provider.ProviderInterfaceCapabilities(baseIdentity)
		if len(capabilities) == 0 {
			continue
		}
		base, err := r.providerInterfaceCapabilityBase(baseIdentity)
		if err != nil {
			return err
		}
		baseInterface := base.Underlying().(*types.Interface).Complete()
		baseKey, err := typeidentity.BuildKey(
			baseInterface,
			typeidentity.NamedObjectKey,
		)
		if err != nil {
			return err
		}
		binding, ok := r.provider.ProviderInterface(baseIdentity)
		if !ok || binding.SourceIdentity() != baseIdentity {
			return &api.NameError{
				Name:   baseIdentity,
				Reason: "provider-interface capability base certificate is absent",
			}
		}
		selected := make(
			map[string]providerInterfaceCapabilityBinding,
			len(capabilities),
		)
		for _, capability := range capabilities {
			if capability.Usage() !=
				gostdlib.ProviderInterfaceCapabilityUsageGeneratedBridge {
				continue
			}
			if !capability.Valid() ||
				capability.BaseSourceIdentity() != baseIdentity {
				return &api.NameError{
					Name:   baseIdentity,
					Reason: "provider-interface capability base exact join failed",
				}
			}
			if err := validateProviderCapabilityBase(capability, base); err != nil {
				return err
			}
			capabilityKey := capability.TargetSourceIdentity() + "\x00" +
				capability.TargetExport() + "\x00" + capability.ViewExport()
			if _, duplicate := selected[capabilityKey]; duplicate {
				return &api.NameError{
					Name:   capability.TargetExport(),
					Reason: "provider-interface capability target is duplicated",
				}
			}
			selected[capabilityKey] = providerInterfaceCapabilityBinding{
				key:         capabilityKey,
				certificate: capability,
				base:        base,
			}
		}
		if len(selected) != 0 {
			r.providerInterfaceCapabilities[baseKey] = selected
		}
	}
	return nil
}

func validateProviderCapabilityBase(
	capability gostdlib.ProviderInterfaceCapability,
	base *types.Named,
) error {
	certificate := capability.BaseInterface()
	provider := certificate.ProviderInterface()
	contract, ok := base.Underlying().(*types.Interface)
	if !ok || provider.Mode() != gostdlib.ProviderInterfaceModeBridge {
		return &api.NameError{
			Name:   capability.BaseExport(),
			Reason: "provider-interface capability base is not a bridge contract",
		}
	}
	contract = contract.Complete()
	methods, matched, err := gostdlibsource.SelectProviderInterfaceMethods(
		certificate,
		contract,
	)
	if err != nil {
		return err
	}
	if !matched || len(methods) != len(provider.Methods()) {
		return &api.NameError{
			Name:   capability.BaseExport(),
			Reason: "provider-interface capability base method contract drifted",
		}
	}
	for _, method := range methods {
		if method.Certificate.Kind() != gostdlib.ProviderInterfaceMethodCallable {
			return &api.NameError{
				Name:   capability.BaseExport(),
				Reason: "provider-interface capability base method exact join failed",
			}
		}
	}
	return nil
}

func (r *Registry) providerInterfaceCapabilityBase(
	identity string,
) (*types.Named, error) {
	object := r.providerObjectByIdentity[identity]
	if identity == gostdlib.LanguageErrorInterfaceIdentity {
		object = types.Universe.Lookup("error")
	}
	typeName, ok := object.(*types.TypeName)
	if !ok || typeName.IsAlias() {
		return nil, &api.NameError{
			Name:   identity,
			Reason: "provider-interface capability base has no exact Go type",
		}
	}
	named, ok := types.Unalias(typeName.Type()).(*types.Named)
	if !ok {
		return nil, &api.NameError{
			Name:   identity,
			Reason: "provider-interface capability base is not named",
		}
	}
	if contract, ok := named.Underlying().(*types.Interface); !ok ||
		!contract.Complete().IsMethodSet() {
		return nil, &api.NameError{
			Name:   identity,
			Reason: "provider-interface capability base is not an interface",
		}
	}
	return named, nil
}

func (r *Registry) providerInterfaceCapability(
	source *types.Named,
	target *types.Interface,
	demandKey string,
) (providerInterfaceCapabilityBinding, bool, error) {
	if r == nil || source == nil || target == nil || demandKey == "" {
		return providerInterfaceCapabilityBinding{}, false, &api.NameError{
			Reason: "provider-interface capability query is invalid",
		}
	}
	if _, ok := source.Underlying().(*types.Interface); !ok {
		return providerInterfaceCapabilityBinding{}, false, nil
	}
	targetKey, err := typeidentity.BuildKey(
		target.Complete(),
		typeidentity.NamedObjectKey,
	)
	if err != nil {
		return providerInterfaceCapabilityBinding{}, false, err
	}
	selected, ok := r.providerInterfaceCapabilityDemands[demandKey]
	if !ok {
		return providerInterfaceCapabilityBinding{}, false, nil
	}
	if !types.Identical(selected.base, source) ||
		!types.Identical(selected.target, target) ||
		selected.targetKey != targetKey || selected.demandKey != demandKey {
		return providerInterfaceCapabilityBinding{}, false, &api.NameError{
			Name:   selected.certificate.TargetExport(),
			Reason: "provider-interface capability key joined non-identical Go types",
		}
	}
	return selected, true, nil
}

func canonicalProviderInterfaceContractKey(
	contract *types.Interface,
) (string, error) {
	return typeidentity.BuildKey(
		contract.Complete(),
		typeidentity.NamedObjectKey,
	)
}

func (n *File) ProviderInterfaceCapability(
	sourceType types.Type,
	targetType types.Type,
	demandKey string,
) (api.ProviderInterfaceCapabilityReference, bool, error) {
	source, ok := types.Unalias(sourceType).(*types.Named)
	if !ok {
		return api.ProviderInterfaceCapabilityReference{}, false, nil
	}
	target, ok := types.Unalias(targetType).Underlying().(*types.Interface)
	if !ok {
		return api.ProviderInterfaceCapabilityReference{}, false, nil
	}
	selected, found, err := n.owner.registry.providerInterfaceCapability(
		source,
		target.Complete(),
		demandKey,
	)
	if err != nil || !found {
		return api.ProviderInterfaceCapabilityReference{}, found, err
	}
	qualifier, request, err := n.providerImport(
		selected.certificate.ModuleSpecifier(),
		api.ImportPhaseValue,
	)
	if err != nil {
		return api.ProviderInterfaceCapabilityReference{}, true, err
	}
	view, err := api.NewQualifiedNameReference(
		qualifier,
		selected.certificate.ViewExport(),
		request,
	)
	if err != nil {
		return api.ProviderInterfaceCapabilityReference{}, true, err
	}
	baseReference, err := api.NewQualifiedNameReference(
		qualifier,
		selected.certificate.BaseExport(),
		request,
	)
	if err != nil {
		return api.ProviderInterfaceCapabilityReference{}, true, err
	}
	targetReference, err := api.NewQualifiedNameReference(
		qualifier,
		selected.certificate.TargetExport(),
		request,
	)
	if err != nil {
		return api.ProviderInterfaceCapabilityReference{}, true, err
	}
	reference, err := api.NewProviderInterfaceCapabilityReference(
		baseReference,
		view,
		targetReference,
		selected.certificate,
	)
	return reference, true, err
}
