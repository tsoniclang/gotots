package naming

import (
	"go/types"
	"strconv"

	environmentcontract "github.com/tsoniclang/gotots/internal/contracts/environment"
	"github.com/tsoniclang/gotots/internal/contracts/gostdlib"
	"github.com/tsoniclang/gotots/internal/emit/api"
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
	reference, err := api.NewQualifiedNameReference(
		qualifier,
		export,
		request,
	)
	return reference, true, err
}

func (n *File) providerGenericCallableProfileReference(
	owner *types.Func,
	profileKey string,
) (api.NameReference, gostdlib.EffectKind, bool, error) {
	contract, providerOwned, err := n.providerFacetOwner(owner)
	if err != nil || !providerOwned {
		return api.NameReference{}, gostdlib.EffectInvalid, providerOwned, err
	}
	selected, ok := n.owner.registry.provider.GenericCallableFacet(
		contract.Identity(),
		profileKey,
	)
	if !ok {
		return api.NameReference{}, gostdlib.EffectInvalid, true, &api.NameError{
			Name: contract.Identity(),
			Reason: "selected generic standard-library callable has no certified provider profile " +
				strconv.Quote(profileKey),
		}
	}
	if selected.SourceIdentity() != contract.Identity() ||
		selected.Kind() != gostdlib.FacetGenericCallableProfile ||
		selected.ProfileKey() != profileKey ||
		!selected.Effect().Valid() {
		return api.NameReference{}, gostdlib.EffectInvalid, true, &api.NameError{
			Name:   contract.Identity(),
			Reason: "provider generic-callable facet does not match its selected source owner",
		}
	}
	qualifier, request, err := n.providerImport(
		selected.ModuleSpecifier(),
		api.ImportPhaseValue,
	)
	if err != nil {
		return api.NameReference{}, gostdlib.EffectInvalid, true, err
	}
	if n.require != nil {
		if err := n.require(owner); err != nil {
			return api.NameReference{}, gostdlib.EffectInvalid, true, err
		}
	}
	reference, err := api.NewQualifiedNameReference(
		qualifier,
		selected.Export(),
		request,
	)
	return reference, selected.Effect(), true, err
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
	reference, providerOwned, err := n.providerFacetReference(
		owner,
		gostdlib.FacetRecoveryCallable,
		gostdlib.FacetCapabilityRecovery,
		api.ImportPhaseValue,
	)
	if err != nil || !providerOwned {
		return api.RecoveryCallableReference{}, providerOwned, err
	}
	contract, _, err := n.providerFacetOwner(owner)
	if err != nil {
		return api.RecoveryCallableReference{}, true, err
	}
	selected, ok := n.owner.registry.provider.Facet(
		contract.Identity(),
		gostdlib.FacetRecoveryCallable,
		gostdlib.FacetCapabilityRecovery,
	)
	if !ok || !selected.Effect().Valid() {
		return api.RecoveryCallableReference{}, true, &api.NameError{
			Name:   contract.Identity(),
			Reason: "provider recovery-callable effect is invalid",
		}
	}
	result, err := api.NewRecoveryCallableReference(
		reference,
		selected.Effect() == gostdlib.EffectAsynchronous,
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
