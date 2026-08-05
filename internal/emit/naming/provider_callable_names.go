package naming

import (
	environmentcontract "github.com/tsoniclang/gotots/internal/contracts/environment"
	"github.com/tsoniclang/gotots/internal/contracts/gostdlib"
	gostdlibsource "github.com/tsoniclang/gotots/internal/contracts/gostdlib/sourcecontract"
	"github.com/tsoniclang/gotots/internal/emit/api"
	"go/types"
	"path"
	"slices"
	"strconv"
	"strings"
)

func (r *Registry) indexProviderImportNames(modules []string) error {
	selected := slices.Clone(modules)
	slices.Sort(selected)
	seenModules := make(map[string]struct{}, len(selected))
	usedNames := make(map[string]struct{}, len(selected))
	for _, module := range selected {
		if module == "" {
			return &api.NameError{Reason: "provider module identity is empty"}
		}
		if _, duplicate := seenModules[module]; duplicate {
			return &api.NameError{
				Name:   module,
				Reason: "provider module identity is duplicated",
			}
		}
		seenModules[module] = struct{}{}
		base, err := providerModuleImportBase(module)
		if err != nil {
			return err
		}
		name := base
		for suffix := uint64(1); ; suffix++ {
			if _, duplicate := usedNames[name]; !duplicate {
				break
			}
			name = base + "__provider_" + strconv.FormatUint(suffix, 10)
		}
		usedNames[name] = struct{}{}
		r.providerImportNameByModule[module] = name
	}
	return nil
}

func providerModuleImportBase(module string) (string, error) {
	base := path.Base(module)
	if !strings.HasSuffix(base, ".js") {
		return "", &api.NameError{
			Name:   module,
			Reason: "provider module does not have an ESM JavaScript suffix",
		}
	}
	base = strings.TrimSuffix(base, ".js")
	base = strings.ReplaceAll(base, "-", "_")
	base = portableIdentifier(base)
	if base == "" {
		return "", &api.NameError{
			Name:   module,
			Reason: "provider module has no import-name stem",
		}
	}
	return base, nil
}

func (n *File) RecoveryCallable(
	owner *types.Func,
) (api.RecoveryCallableReference, bool, error) {
	if owner == nil {
		return api.RecoveryCallableReference{}, false, &api.NameError{
			Reason: "recovery-callable owner is nil",
		}
	}
	owner = owner.Origin()
	contract, providerOwned, err := n.providerFacetOwner(owner)
	if err != nil || !providerOwned {
		return api.RecoveryCallableReference{}, false, err
	}
	selected, ok := n.owner.registry.provider.Facet(
		contract.Identity(),
		gostdlib.FacetRecoveryCallable,
		gostdlib.FacetCapabilityRecovery,
	)
	if !ok {
		return api.RecoveryCallableReference{}, false, nil
	}
	if !selected.Effect().Valid() {
		return api.RecoveryCallableReference{}, true, &api.NameError{
			Name:   contract.Identity(),
			Reason: "provider recovery-callable effect is invalid",
		}
	}
	reference, selectedOwner, err := n.providerFacetReference(
		owner,
		gostdlib.FacetRecoveryCallable,
		gostdlib.FacetCapabilityRecovery,
		api.ImportPhaseValue,
	)
	if err != nil {
		return api.RecoveryCallableReference{}, true, err
	}
	if !selectedOwner {
		return api.RecoveryCallableReference{}, true, &api.NameError{
			Name:   contract.Identity(),
			Reason: "provider recovery facet lost its certified owner",
		}
	}
	result, err := api.NewRecoveryCallableReference(
		reference,
		selected.Effect().MaySuspend(),
	)
	return result, true, err
}

func providerNamedStructCapability(
	operation api.NamedStructOperation,
) (gostdlib.FacetCapability, error) {
	switch operation {
	case api.NamedStructOperationZero:
		return gostdlib.FacetCapabilityZero, nil
	case api.NamedStructOperationCopy:
		return gostdlib.FacetCapabilityCopy, nil
	case api.NamedStructOperationEqual:
		return gostdlib.FacetCapabilityEqual, nil
	case api.NamedStructOperationHash:
		return gostdlib.FacetCapabilityHash, nil
	case api.NamedStructOperationConvert:
		return gostdlib.FacetCapabilityConvert, nil
	case api.NamedStructOperationStorage:
		return gostdlib.FacetCapabilityStorage, nil
	case api.NamedStructOperationAssign:
		return gostdlib.FacetCapabilityAssign, nil
	default:
		return gostdlib.FacetCapabilityInvalid, &api.NameError{
			Reason: "named-struct operation has no provider capability",
		}
	}
}

func (n *File) providerFacetOwner(
	object types.Object,
) (environmentcontract.ObjectContract, bool, error) {
	if object == nil || n.owner == nil || n.owner.registry == nil {
		return environmentcontract.ObjectContract{}, false, &api.NameError{
			Reason: "provider facet owner is invalid",
		}
	}
	if object.Pkg() != nil && object.Parent() != nil &&
		object.Parent() != object.Pkg().Scope() {
		return environmentcontract.ObjectContract{}, false, nil
	}
	binding, ok := n.owner.registry.byObject[object]
	if !ok {
		return environmentcontract.ObjectContract{}, false, &api.NameError{
			Name:   object.Name(),
			Reason: "provider facet owner has no target binding",
		}
	}
	if binding.kind != targetBindingProvider &&
		binding.kind != targetBindingMissingProvider {
		return environmentcontract.ObjectContract{}, false, nil
	}
	provider := n.owner.registry.provider
	if provider == nil || !provider.Valid() {
		return environmentcontract.ObjectContract{}, true, &api.NameError{
			Name:   object.Name(),
			Reason: "standard-library provider certificate is invalid",
		}
	}
	contract, err := environmentcontract.Describe(object)
	return contract, true, err
}

func (n *File) allocateProviderImportName(preferred string) string {
	candidate := preferred
	for suffix := uint64(0); n.sourceNameExists(candidate) ||
		n.hasImportName(candidate); suffix++ {
		candidate = preferred + "__from_gostdlib"
		if suffix != 0 {
			candidate += "_" + strconv.FormatUint(suffix, 10)
		}
	}
	n.importNames[candidate] = struct{}{}
	return candidate
}

func (r *Registry) ProviderGenericKernel(
	owner *types.Func,
) (gostdlib.Facet, bool, error) {
	if r == nil || owner == nil || owner.Origin() != owner {
		return gostdlib.Facet{}, false, &api.NameError{
			Reason: "provider generic-kernel owner is invalid",
		}
	}
	binding, ok := r.byObject[owner]
	if !ok || binding.kind != targetBindingProvider {
		return gostdlib.Facet{}, false, nil
	}
	if r.provider == nil || !r.provider.Valid() {
		return gostdlib.Facet{}, false, &api.NameError{
			Name:   owner.Name(),
			Reason: "standard-library provider certificate is invalid",
		}
	}
	contract, err := environmentcontract.Describe(owner)
	if err != nil {
		return gostdlib.Facet{}, false, err
	}
	selected, ok := r.provider.Facet(
		contract.Identity(),
		gostdlib.FacetGenericCallableKernel,
		gostdlib.FacetCapabilityKernel,
	)
	if !ok {
		return gostdlib.Facet{}, false, nil
	}
	capabilities := selected.Capabilities()
	if selected.SourceIdentity() != contract.Identity() ||
		selected.Kind() != gostdlib.FacetGenericCallableKernel ||
		len(capabilities) != 1 ||
		capabilities[0] != gostdlib.FacetCapabilityKernel ||
		selected.ModuleSpecifier() == "" || selected.Export() == "" ||
		len(selected.GenericTypeArguments()) == 0 {
		return gostdlib.Facet{}, false, &api.NameError{
			Name:   contract.Identity(),
			Reason: "provider generic-kernel certificate is inconsistent",
		}
	}
	return selected, true, nil
}

func (r *Registry) ProviderGenericTypeArguments(
	owner *types.Func,
) ([]api.GenericTypeArgumentProjection, bool, error) {
	if r == nil || owner == nil {
		return nil, false, &api.NameError{
			Reason: "provider generic type-argument owner is invalid",
		}
	}
	binding, ok := r.byObject[owner.Origin()]
	if !ok || binding.kind != targetBindingProvider {
		return nil, false, nil
	}
	configured := binding.providerGenericTypeArguments
	kernel, kernelOwned, err := r.ProviderGenericKernel(owner.Origin())
	if err != nil {
		return nil, true, err
	}
	if kernelOwned {
		configured = kernel.GenericTypeArguments()
	}
	if len(configured) == 0 {
		return nil, true, &api.NameError{
			Name:   owner.Name(),
			Reason: "provider generic type-argument projection is absent",
		}
	}
	result := make(
		[]api.GenericTypeArgumentProjection,
		0,
		len(configured),
	)
	for _, argument := range configured {
		facet, ok := providerGenericTypeArgumentFacet(argument.Facet)
		if !ok {
			return nil, true, &api.NameError{
				Name:   owner.Name(),
				Reason: "provider generic type-argument facet is invalid",
			}
		}
		projection, projectionErr := api.NewGenericTypeArgumentProjection(
			argument.TypeParameter,
			facet,
		)
		if projectionErr != nil {
			return nil, true, projectionErr
		}
		result = append(result, projection)
	}
	return result, true, nil
}

func (n *File) ProviderGenericTypeArguments(
	owner *types.Func,
) ([]api.GenericTypeArgumentProjection, bool, error) {
	if n == nil || n.owner == nil || n.owner.registry == nil {
		return nil, false, &api.NameError{
			Reason: "provider generic type-argument registry is invalid",
		}
	}
	return n.owner.registry.ProviderGenericTypeArguments(owner)
}

func providerGenericTypeArgumentFacet(
	facet gostdlib.GenericTypeArgumentFacet,
) (api.GenericTypeArgumentFacet, bool) {
	switch facet {
	case gostdlib.GenericTypeArgumentLogical:
		return api.GenericTypeArgumentLogical, true
	case gostdlib.GenericTypeArgumentStorage:
		return api.GenericTypeArgumentStorage, true
	case gostdlib.GenericTypeArgumentContainerStorage:
		return api.GenericTypeArgumentContainerStorage, true
	case gostdlib.GenericTypeArgumentPointer:
		return api.GenericTypeArgumentPointer, true
	default:
		return api.GenericTypeArgumentInvalid, false
	}
}

func (r *Registry) ProviderGenericOperations(
	owner *types.Func,
) ([]gostdlib.GenericOperationDocument, bool, error) {
	if r == nil || owner == nil {
		return nil, false, &api.NameError{
			Reason: "provider generic-operation owner is invalid",
		}
	}
	binding, ok := r.byObject[owner.Origin()]
	if !ok || binding.kind == targetBindingEnvironment ||
		binding.kind == targetBindingSource || binding.kind == targetBindingLocal {
		return nil, false, nil
	}
	if binding.kind == targetBindingMissingProvider {
		return nil, true, nil
	}
	if binding.kind != targetBindingProvider {
		return nil, false, &api.NameError{
			Name:   owner.Name(),
			Reason: "provider generic-operation binding is invalid",
		}
	}
	operations, err := gostdlib.CanonicalGenericOperations(
		binding.providerGenericOperations,
	)
	if err != nil {
		return nil, true, &api.NameError{
			Name:   owner.Name(),
			Reason: "provider generic-operation contract is invalid",
		}
	}
	return operations, true, nil
}

func (n *File) providerCallableProfileValues(
	profile gostdlib.ProviderCallableProfile,
) ([]types.Object, error) {
	canonical := profile.CanonicalValues()
	values := make([]types.Object, 0, len(canonical))
	for _, selected := range canonical {
		identity := selected.SourceIdentity
		object := n.owner.registry.providerObjectByIdentity[identity]
		variable, ok := object.(*types.Var)
		if !ok || variable.IsField() || variable.Pkg() == nil ||
			variable.Parent() != variable.Pkg().Scope() {
			return nil, &api.NameError{
				Name:   identity,
				Reason: "provider callable-profile canonical value has no exact source variable",
			}
		}
		values = append(values, variable)
	}
	return values, nil
}

func (n *File) ProviderCallableProfileCandidates(
	owner *types.Func,
) ([]api.ProviderCallableProfileCandidate, bool, error) {
	if owner == nil {
		return nil, false, &api.NameError{
			Reason: "provider callable-profile owner is nil",
		}
	}
	owner = owner.Origin()
	contract, providerOwned, err := n.providerFacetOwner(owner)
	if err != nil || !providerOwned {
		return nil, providerOwned, err
	}
	profiles := n.owner.registry.provider.ProviderCallableProfiles(
		contract.Identity(),
	)
	result := make([]api.ProviderCallableProfileCandidate, 0, len(profiles))
	for _, profile := range profiles {
		if !profile.Valid() || profile.SourceIdentity() != contract.Identity() {
			return nil, true, &api.NameError{
				Name:   contract.Identity(),
				Reason: "provider callable-profile candidate is inconsistent",
			}
		}
		capabilityViews, capabilityErr :=
			n.providerCallableProfileCapabilityTypes(profile)
		if capabilityErr != nil {
			return nil, true, capabilityErr
		}
		guards, _, guardErr := n.providerCallableProfileGuards(
			owner,
			profile,
			false,
		)
		if guardErr != nil {
			return nil, true, guardErr
		}
		candidate, candidateErr := api.NewProviderCallableProfileCandidate(
			profile,
			capabilityViews,
			guards,
		)
		if candidateErr != nil {
			return nil, true, candidateErr
		}
		result = append(result, candidate)
	}
	return result, true, nil
}

func (n *File) providerCallableProfileCapabilityTypes(
	profile gostdlib.ProviderCallableProfile,
) ([]api.ProviderCallableProfileCapability, error) {
	views := profile.CapabilityViews()
	result := make([]api.ProviderCallableProfileCapability, 0, len(views))
	for _, view := range views {
		baseType, err := n.providerProfileInterfaceType(view.BaseSourceIdentity)
		if err != nil {
			return nil, err
		}
		targetType, err := n.providerProfileInterfaceType(view.TargetSourceIdentity)
		if err != nil {
			return nil, err
		}
		base, baseOK := types.Unalias(baseType).(*types.Named)
		target, targetOK := types.Unalias(targetType).(*types.Named)
		if !baseOK || !targetOK {
			return nil, &api.NameError{
				Name:   view.TargetSourceIdentity,
				Reason: "provider callable-profile capability is not named",
			}
		}
		capability, err := api.NewProviderCallableProfileCapability(base, target)
		if err != nil {
			return nil, err
		}
		result = append(result, capability)
	}
	return result, nil
}

func (n *File) ProviderCallableParameters(
	owner *types.Func,
) ([]gostdlib.ProviderCallableParameterDocument, bool, error) {
	if owner == nil {
		return nil, false, &api.NameError{
			Reason: "provider callable owner is nil",
		}
	}
	owner = owner.Origin()
	contract, providerOwned, err := n.providerFacetOwner(owner)
	if err != nil || !providerOwned {
		return nil, providerOwned, err
	}
	binding, ok := n.owner.registry.provider.Binding(contract.Identity())
	if !ok || binding.Kind() != gostdlib.BindingFunction {
		return nil, true, &api.NameError{
			Name:   contract.Identity(),
			Reason: "provider callable binding evidence is absent",
		}
	}
	return binding.CallableParameters(), true, nil
}

func (n *File) providerCallableProfileGuards(
	owner *types.Func,
	profile gostdlib.ProviderCallableProfile,
	demand bool,
) ([]types.Type, []api.RootRequest, error) {
	if owner == nil {
		return nil, nil, &api.NameError{
			Reason: "provider callable-profile owner is nil",
		}
	}
	signature, ok := owner.Origin().Type().(*types.Signature)
	if !ok {
		return nil, nil, &api.NameError{
			Name:   owner.Name(),
			Reason: "provider callable-profile owner has no signature",
		}
	}
	guardIdentities := profile.GuardInterfaces()
	guards := make([]types.Type, 0, len(guardIdentities))
	var requests []api.RootRequest
	for _, identity := range guardIdentities {
		certificate, found := profile.Interface(identity)
		if !found {
			return nil, nil, &api.NameError{
				Name:   identity,
				Reason: "provider callable-profile guard certificate is absent",
			}
		}
		protocol, synthetic := certificate.Protocol()
		if synthetic {
			selected, err := gostdlibsource.ResolveProviderProtocolInterface(
				protocol,
				signature,
			)
			if err != nil {
				return nil, nil, err
			}
			valueParameter, ok := certificate.ProtocolValueParameter()
			if !ok || valueParameter < 0 ||
				valueParameter >= signature.Params().Len() {
				return nil, nil, &api.NameError{
					Name:   identity,
					Reason: "provider protocol value parameter is invalid",
				}
			}
			if demand {
				selectedRequests, err := n.InterfaceContractDemand(
					signature.Params().At(valueParameter).Type(),
					selected,
				)
				if err != nil {
					return nil, nil, err
				}
				requests = append(requests, selectedRequests...)
			}
			guards = append(guards, selected)
			continue
		}
		selected, err := n.providerProfileInterfaceType(identity)
		if err != nil {
			return nil, nil, err
		}
		guards = append(guards, selected)
	}
	return guards, api.CombineRequests(requests), nil
}
