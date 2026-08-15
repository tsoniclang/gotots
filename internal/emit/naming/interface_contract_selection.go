package naming

import (
	"go/types"
	"sort"

	"github.com/tsoniclang/gotots/internal/emit/api"
)

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

func (r *Registry) interfaceContractSelections(
	contractKey string,
) []interfaceContractSelection {
	bySurface := r.interfaceContracts[contractKey]
	keys := make([]string, 0, len(bySurface))
	for key := range bySurface {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	selected := make([]interfaceContractSelection, 0, len(keys))
	for _, key := range keys {
		selected = append(selected, bySurface[key])
	}
	return selected
}
