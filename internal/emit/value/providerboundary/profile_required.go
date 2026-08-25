package providerboundary

import "github.com/tsoniclang/gotots/internal/contracts/gostdlib"

func requiredProfileParameters(
	profile gostdlib.ProviderCallableProfile,
	roots []map[string]struct{},
) ([]int, error) {
	result := requiredProfileInterfaceRoots(profile, roots)
	for _, selected := range profile.Interfaces() {
		protocol, ok := selected.Protocol()
		if !ok {
			continue
		}
		parameters, err := gostdlib.ProviderProtocolCallableParameters(protocol)
		if err != nil {
			return nil, err
		}
		result = mergeIndexes(result, parameters)
	}
	return result, nil
}

func requiredProfileResults(
	profile gostdlib.ProviderCallableProfile,
	roots []map[string]struct{},
) []int {
	return requiredProfileInterfaceRoots(profile, roots)
}

func requiredProfileInterfaceRoots(
	profile gostdlib.ProviderCallableProfile,
	roots []map[string]struct{},
) []int {
	identities := make(map[string]struct{}, len(profile.Interfaces()))
	for _, selected := range profile.Interfaces() {
		identities[selected.SourceIdentity()] = struct{}{}
	}
	return rootsContainingIdentities(roots, identities)
}

func identityOccursInRoots(
	identity string,
	rootSets ...[]map[string]struct{},
) bool {
	for _, roots := range rootSets {
		for _, root := range roots {
			if _, ok := root[identity]; ok {
				return true
			}
		}
	}
	return false
}
