package certify

import (
	"fmt"
	"path/filepath"

	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

const providerSupportPath = "src/internal/facets/provider-support.ts"

type providerSupportMarkers struct {
	guard        tsgo.ProjectExport
	contract     tsgo.ProjectExport
	fromProvider tsgo.ProjectExport
}

func loadProviderSupportMarkers(
	config resolvedConfig,
	project *tsgo.ProjectInspection,
) (providerSupportMarkers, error) {
	exports, err := project.Exports(filepath.Join(
		config.providerRoot,
		filepath.FromSlash(providerSupportPath),
	))
	if err != nil {
		return providerSupportMarkers{}, err
	}
	byName := make(map[string]tsgo.ProjectExport, len(exports))
	for _, selected := range exports {
		byName[selected.Name()] = selected
	}
	if len(exports) != 3 {
		return providerSupportMarkers{}, certifyError(
			"inspect provider support",
			providerSupportPath,
			fmt.Sprintf("marker export count is %d, want 3", len(exports)),
		)
	}
	result := providerSupportMarkers{
		guard:        byName["InterfaceGuard"],
		contract:     byName["InterfaceContract"],
		fromProvider: byName["FromProviderBridge"],
	}
	if result.guard.Name() == "" || result.contract.Name() == "" ||
		result.fromProvider.Name() == "" {
		return providerSupportMarkers{}, certifyError(
			"inspect provider support",
			providerSupportPath,
			"marker export set is not exact",
		)
	}
	return result, nil
}

func verifyProviderSupportParameters(
	project *tsgo.ProjectInspection,
	target tsgo.ProjectExport,
	baseParameters int,
	guardCount int,
	contractCount int,
	fromProviderCount int,
	markers providerSupportMarkers,
) error {
	type expectedParameter struct {
		kind   string
		marker tsgo.ProjectExport
	}
	expected := make([]expectedParameter, 0,
		guardCount+contractCount+fromProviderCount)
	for range guardCount {
		expected = append(expected, expectedParameter{"guard", markers.guard})
	}
	for range contractCount {
		expected = append(expected, expectedParameter{"contract", markers.contract})
	}
	for range fromProviderCount {
		expected = append(expected, expectedParameter{"from-provider", markers.fromProvider})
	}
	actualCount, err := project.CallableParameterCount(target)
	if err != nil {
		return err
	}
	wantCount := baseParameters + len(expected)
	if actualCount != wantCount {
		return certifyError(
			"verify provider support parameters",
			target.Name(),
			fmt.Sprintf("target has %d parameters, contract requires %d", actualCount, wantCount),
		)
	}
	for offset, selected := range expected {
		parameter := baseParameters + offset
		identity, err := project.CallableParameterTypeIdentity(target, parameter)
		if err != nil {
			return certifyError(
				"verify provider support parameters",
				target.Name(),
				fmt.Sprintf("inspect parameter %d: %v", parameter, err),
			)
		}
		if !identity.Matches(selected.marker) {
			return certifyError(
				"verify provider support parameters",
				target.Name(),
				fmt.Sprintf("parameter %d is not the %s support contract", parameter, selected.kind),
			)
		}
	}
	return nil
}
