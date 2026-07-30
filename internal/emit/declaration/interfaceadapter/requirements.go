package interfaceadapter

import (
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
)

func Contracts(
	artifact *api.GeneratedArtifact,
	requirements []api.DeclarationRequirement,
) ([]*types.Interface, error) {
	if artifact == nil ||
		artifact.Kind() != api.GeneratedArtifactInterfaceAdapter {
		return nil, &api.GeneratedArtifactShapeError{
			Reason: "interface-adapter requirement owner is invalid",
		}
	}
	baseline := 0
	contracts := make([]*types.Interface, 0, len(requirements))
	keys := make(map[string]*types.Interface)
	for _, requirement := range requirements {
		selected, ok := requirement.InterfaceAdapter()
		if !ok || selected != artifact {
			return nil, &api.GeneratedArtifactShapeError{
				Artifact: artifact.TargetName(),
				Reason:   "interface adapter received a foreign requirement",
			}
		}
		_, contract, key, demanded :=
			requirement.InterfaceAdapterContract()
		if !demanded {
			baseline++
			continue
		}
		if existing := keys[key]; existing != nil {
			if !types.Identical(existing, contract) {
				return nil, &api.GeneratedArtifactShapeError{
					Artifact: artifact.TargetName(),
					Reason:   "interface adapter joined non-identical contract demands",
				}
			}
			continue
		}
		keys[key] = contract
		contracts = append(contracts, contract)
	}
	if baseline != 1 {
		return nil, &api.GeneratedArtifactShapeError{
			Artifact: artifact.TargetName(),
			Reason:   "interface adapter requires one definition request",
		}
	}
	return contracts, nil
}
