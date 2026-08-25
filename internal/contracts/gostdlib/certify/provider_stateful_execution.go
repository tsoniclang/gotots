package certify

import (
	"slices"

	"github.com/tsoniclang/gotots/internal/contracts/gostdlib"
)

func verifyStatefulExecutionProfilePairs(
	modules []gostdlib.FacetModuleDocument,
) error {
	profiles := make(map[string][]gostdlib.ProviderStatefulProfileDocument)
	for _, module := range modules {
		for _, profile := range module.StatefulProfiles {
			profiles[profile.SourceIdentity] = append(
				profiles[profile.SourceIdentity],
				profile,
			)
		}
	}
	for identity, selected := range profiles {
		for _, profile := range selected {
			if !statefulProfileMaySuspend(profile) {
				continue
			}
			matched := false
			for _, candidate := range selected {
				if statefulProfileMaySuspend(candidate) ||
					!sameStatefulExecutionShape(profile, candidate) {
					continue
				}
				matched = true
				break
			}
			if !matched {
				return certifyError(
					"verify provider stateful execution profiles",
					identity,
					"suspending profile has no exact synchronous sibling",
				)
			}
		}
	}
	return nil
}

func statefulProfileMaySuspend(
	profile gostdlib.ProviderStatefulProfileDocument,
) bool {
	for _, selected := range profile.Interfaces {
		for _, method := range selected.ProviderInterface.Methods {
			if method.Effect.MaySuspend() {
				return true
			}
		}
	}
	for _, method := range profile.Methods {
		if method.Effect.MaySuspend() {
			return true
		}
	}
	return false
}

func sameStatefulExecutionShape(
	left gostdlib.ProviderStatefulProfileDocument,
	right gostdlib.ProviderStatefulProfileDocument,
) bool {
	if left.SourceIdentity != right.SourceIdentity ||
		!slices.Equal(left.TypeArguments, right.TypeArguments) ||
		!slices.Equal(left.Operations, right.Operations) ||
		len(left.Interfaces) != len(right.Interfaces) ||
		len(left.Fields) != len(right.Fields) ||
		len(left.Methods) != len(right.Methods) {
		return false
	}
	for index := range left.Interfaces {
		leftInterface := left.Interfaces[index]
		rightInterface := right.Interfaces[index]
		if leftInterface.SourceIdentity != rightInterface.SourceIdentity ||
			len(leftInterface.ProviderInterface.Methods) !=
				len(rightInterface.ProviderInterface.Methods) {
			return false
		}
		for methodIndex := range leftInterface.ProviderInterface.Methods {
			leftMethod := leftInterface.ProviderInterface.Methods[methodIndex]
			rightMethod := rightInterface.ProviderInterface.Methods[methodIndex]
			if leftMethod.SourceIdentity != rightMethod.SourceIdentity ||
				leftMethod.Kind != rightMethod.Kind ||
				leftMethod.SourceSignature != rightMethod.SourceSignature ||
				leftMethod.ContractSignature != rightMethod.ContractSignature {
				return false
			}
		}
	}
	for index := range left.Fields {
		leftField := left.Fields[index]
		rightField := right.Fields[index]
		if leftField.Member != rightField.Member ||
			leftField.Ordinal != rightField.Ordinal ||
			leftField.Embedded != rightField.Embedded ||
			leftField.SourceSignature != rightField.SourceSignature {
			return false
		}
	}
	for index := range left.Methods {
		leftMethod := left.Methods[index]
		rightMethod := right.Methods[index]
		if leftMethod.SourceIdentity != rightMethod.SourceIdentity ||
			leftMethod.Member != rightMethod.Member ||
			leftMethod.SourceSignature != rightMethod.SourceSignature {
			return false
		}
	}
	return true
}
