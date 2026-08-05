package providerinterfacebridge

import (
	"go/types"
	"sort"

	"github.com/tsoniclang/gotots/internal/emit/api"
)

type CapabilityContract struct {
	Contract *types.Interface
	Key      string
}

type ProfileCapabilityContract struct {
	Target *api.GeneratedArtifact
}

func Requirements(
	artifact *api.GeneratedArtifact,
	requirements []api.DeclarationRequirement,
) ([]CapabilityContract, error) {
	if artifact == nil ||
		artifact.Kind() != api.GeneratedArtifactProviderInterfaceBridge {
		return nil, shapeError("", "provider-interface bridge requirement owner is invalid")
	}
	if _, _, profiled := artifact.ProviderProfileInterfaceBridge(); profiled {
		return nil, shapeError(
			artifact.TargetName(),
			"provider-profile bridge used ordinary capability requirements",
		)
	}
	baseline := 0
	selected := make(map[string]*types.Interface)
	for _, requirement := range requirements {
		if definition, ok := requirement.ProviderInterfaceBridge(); ok {
			if definition != artifact {
				return nil, shapeError(
					artifact.TargetName(),
					"provider-interface bridge received a foreign definition requirement",
				)
			}
			baseline++
			continue
		}
		owner, contract, key, ok :=
			requirement.ProviderInterfaceCapability()
		if !ok || owner != artifact {
			return nil, shapeError(
				artifact.TargetName(),
				"provider-interface bridge received a foreign capability requirement",
			)
		}
		if existing := selected[key]; existing != nil {
			if !types.Identical(existing, contract) {
				return nil, shapeError(
					artifact.TargetName(),
					"provider-interface bridge joined non-identical capability contracts",
				)
			}
			continue
		}
		selected[key] = contract
	}
	if baseline != 1 {
		return nil, shapeError(
			artifact.TargetName(),
			"provider-interface bridge requires one definition request",
		)
	}
	keys := make([]string, 0, len(selected))
	for key := range selected {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	contracts := make([]CapabilityContract, 0, len(keys))
	for _, key := range keys {
		contracts = append(contracts, CapabilityContract{
			Contract: selected[key],
			Key:      key,
		})
	}
	return contracts, nil
}

func ProfileRequirements(
	artifact *api.GeneratedArtifact,
	requirements []api.DeclarationRequirement,
) ([]CapabilityContract, []ProfileCapabilityContract, error) {
	if artifact == nil ||
		artifact.Kind() != api.GeneratedArtifactProviderInterfaceBridge {
		return nil, nil, shapeError("", "provider-profile bridge requirement owner is invalid")
	}
	if _, profile, profiled := artifact.ProviderProfileInterfaceBridge(); !profiled || len(profile) == 0 {
		return nil, nil, shapeError(
			artifact.TargetName(),
			"provider-profile bridge contract is absent",
		)
	}
	baseline := 0
	selected := make(map[string]*types.Interface)
	profileSelected := make(map[string]*api.GeneratedArtifact)
	for _, requirement := range requirements {
		if definition, ok := requirement.ProviderInterfaceBridge(); ok {
			if definition != artifact {
				return nil, nil, shapeError(
					artifact.TargetName(),
					"provider-profile bridge received a foreign definition requirement",
				)
			}
			baseline++
			continue
		}
		if owner, contract, key, ok :=
			requirement.ProviderInterfaceCapability(); ok {
			if owner != artifact {
				return nil, nil, shapeError(
					artifact.TargetName(),
					"provider-profile bridge received a foreign capability requirement",
				)
			}
			if existing := selected[key]; existing != nil {
				if !types.Identical(existing, contract) {
					return nil, nil, shapeError(
						artifact.TargetName(),
						"provider-profile bridge joined non-identical capability contracts",
					)
				}
				continue
			}
			selected[key] = contract
			continue
		}
		owner, target, ok :=
			requirement.ProviderProfileInterfaceCapability()
		if !ok || owner != artifact {
			return nil, nil, shapeError(
				artifact.TargetName(),
				"provider-profile bridge received a foreign capability requirement",
			)
		}
		key := target.ArtifactKey()
		if existing := profileSelected[key]; existing != nil {
			if existing != target {
				return nil, nil, shapeError(
					artifact.TargetName(),
					"provider-profile bridge joined non-identical capability targets",
				)
			}
			continue
		}
		profileSelected[key] = target
	}
	if baseline != 1 {
		return nil, nil, shapeError(
			artifact.TargetName(),
			"provider-profile bridge requires one definition request",
		)
	}
	keys := make([]string, 0, len(selected))
	for key := range selected {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	contracts := make([]CapabilityContract, 0, len(keys))
	for _, key := range keys {
		contracts = append(contracts, CapabilityContract{
			Contract: selected[key],
			Key:      key,
		})
	}
	profileKeys := make([]string, 0, len(profileSelected))
	for key := range profileSelected {
		profileKeys = append(profileKeys, key)
	}
	sort.Strings(profileKeys)
	profileContracts := make(
		[]ProfileCapabilityContract,
		0,
		len(profileKeys),
	)
	for _, key := range profileKeys {
		profileContracts = append(
			profileContracts,
			ProfileCapabilityContract{Target: profileSelected[key]},
		)
	}
	return contracts, profileContracts, nil
}
