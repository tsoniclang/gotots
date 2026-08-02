package naming

import (
	"go/types"
	"strconv"

	environmentcontract "github.com/tsoniclang/gotots/internal/contracts/environment"
	"github.com/tsoniclang/gotots/internal/contracts/gostdlib"
	"github.com/tsoniclang/gotots/internal/emit/api"
)

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
