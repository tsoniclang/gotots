package interfaceadapter

import (
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
)

type Contract struct {
	sourceType types.Type
	methodSet  *types.Interface
}

func (c Contract) valid() bool {
	if c.sourceType == nil || c.methodSet == nil ||
		!c.methodSet.Complete().IsMethodSet() {
		return false
	}
	selected, ok := types.Unalias(c.sourceType).Underlying().(*types.Interface)
	return ok && selected.Complete().IsMethodSet() &&
		types.Identical(selected.Complete(), c.methodSet)
}

func Contracts(
	artifact *api.GeneratedArtifact,
	requirements []api.DeclarationRequirement,
) ([]Contract, bool, error) {
	if artifact == nil ||
		artifact.Kind() != api.GeneratedArtifactInterfaceAdapter {
		return nil, false, &api.GeneratedArtifactShapeError{
			Reason: "interface-adapter requirement owner is invalid",
		}
	}
	baseline := 0
	completeMethodSet := 0
	contracts := make([]Contract, 0, len(requirements))
	keys := make(map[string]Contract)
	for _, requirement := range requirements {
		selectedArtifact, ok := requirement.InterfaceAdapter()
		if !ok || selectedArtifact != artifact {
			return nil, false, &api.GeneratedArtifactShapeError{
				Artifact: artifact.TargetName(),
				Reason:   "interface adapter received a foreign requirement",
			}
		}
		if completeArtifact, complete :=
			requirement.InterfaceAdapterCompleteMethodSet(); complete {
			if completeArtifact != artifact {
				return nil, false, &api.GeneratedArtifactShapeError{
					Artifact: artifact.TargetName(),
					Reason:   "interface adapter received a foreign complete method set",
				}
			}
			completeMethodSet++
			continue
		}
		_, sourceType, contract, key, demanded :=
			requirement.InterfaceAdapterContract()
		if !demanded {
			baseline++
			continue
		}
		selectedContract := Contract{sourceType: sourceType, methodSet: contract}
		if !selectedContract.valid() {
			return nil, false, &api.GeneratedArtifactShapeError{
				Artifact: artifact.TargetName(),
				Reason:   "interface adapter contract surface is invalid",
			}
		}
		if existing, ok := keys[key]; ok {
			if !types.Identical(existing.sourceType, selectedContract.sourceType) ||
				!types.Identical(existing.methodSet, selectedContract.methodSet) {
				return nil, false, &api.GeneratedArtifactShapeError{
					Artifact: artifact.TargetName(),
					Reason:   "interface adapter joined non-identical contract demands",
				}
			}
			continue
		}
		keys[key] = selectedContract
		contracts = append(contracts, selectedContract)
	}
	if baseline != 1 {
		return nil, false, &api.GeneratedArtifactShapeError{
			Artifact: artifact.TargetName(),
			Reason:   "interface adapter requires one definition request",
		}
	}
	if completeMethodSet > 1 {
		return nil, false, &api.GeneratedArtifactShapeError{
			Artifact: artifact.TargetName(),
			Reason:   "interface adapter has multiple complete method-set demands",
		}
	}
	return contracts, completeMethodSet == 1, nil
}
