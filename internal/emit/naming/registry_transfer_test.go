package naming

import (
	"testing"

	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/output"
)

func TestRegistryTransferDropsObservationsAndClaimsOnce(t *testing.T) {
	registry := NewRegistry()
	registry.interfaceContracts["retained-contract"] =
		map[string]interfaceContractSelection{}
	registry.interfaceAdapterNames[generatedArtifactNameScope(
		"retained-artifact",
		generatedArtifactPlacement{
			kind: api.GeneratedArtifactPlacementCompilation,
		},
		output.InterfaceAdapterSupportPath,
	)] = "adapter"
	registry.providerInterfaceCapabilityDemands["capability"] =
		providerInterfaceCapabilityBinding{}
	registry.providerInterfaceBridgesByContract["provider"] =
		map[string]struct{}{"bridge": {}}
	registry.interfaceAdaptersByContract["interface"] =
		map[string]struct{}{"adapter": {}}
	registry.reflectionInterfaceExposures["reflection-interface"] =
		reflectionInterfaceExposure{
			adapters: map[string]struct{}{"adapter": {}},
		}
	registry.reflectionInterfaceContracts["adapter"] =
		map[string]struct{}{"contract": {}}
	registry.reflectionInterfaceDirty = true
	registry.interfaceContractDemands["source"] =
		map[string]interfaceContractDemand{"target": {}}
	registry.interfaceReflectionDemands["reflection"] =
		interfaceReflectionDemand{}
	registry.reflectionValueDemands["value"] = struct{}{}
	registry.reflectionValueContracts["contract"] = interfaceContractSelection{}
	registry.interfaceDemandRequests[interfaceDemandRequestKey{
		kind:      interfaceDemandTransition,
		sourceKey: "source",
		targetKey: "target",
	}] = nil

	transferred, err := registry.TransferCanonicalIdentity()
	if err != nil {
		t.Fatal(err)
	}
	if transferred != registry {
		t.Fatal("registry transfer changed canonical allocation identity")
	}
	if len(registry.interfaceContracts) != 1 ||
		len(registry.interfaceAdapterNames) != 1 {
		t.Fatal("registry transfer discarded canonical identity facts")
	}
	if len(registry.providerInterfaceCapabilityDemands) != 0 ||
		len(registry.providerInterfaceBridgesByContract) != 0 ||
		len(registry.interfaceAdaptersByContract) != 0 ||
		len(registry.reflectionInterfaceExposures) != 0 ||
		len(registry.reflectionInterfaceContracts) != 0 ||
		registry.reflectionInterfaceDirty ||
		len(registry.interfaceContractDemands) != 0 ||
		len(registry.interfaceReflectionDemands) != 0 ||
		len(registry.reflectionValueDemands) != 0 ||
		len(registry.reflectionValueContracts) != 0 ||
		len(registry.interfaceDemandRequests) != 0 {
		t.Fatal("registry transfer retained first-session observations")
	}
	if err := registry.ClaimFinalSession(); err != nil {
		t.Fatal(err)
	}
	if err := registry.ClaimFinalSession(); err == nil {
		t.Fatal("registry admitted a second final-session owner")
	}
	if _, err := registry.TransferCanonicalIdentity(); err == nil {
		t.Fatal("registry admitted a repeated ownership transfer")
	}
}
