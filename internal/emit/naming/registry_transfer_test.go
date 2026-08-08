package naming

import "testing"

func TestRegistryTransferDropsObservationsAndClaimsOnce(t *testing.T) {
	registry := NewRegistry()
	registry.interfaceContracts["retained-contract"] = nil
	registry.interfaceAdapterNames["retained-artifact"] = "adapter"
	registry.providerInterfaceCapabilityDemands["capability"] =
		providerInterfaceCapabilityBinding{}
	registry.providerInterfaceBridgesByContract["provider"] =
		map[string]struct{}{"bridge": {}}
	registry.interfaceAdaptersByContract["interface"] =
		map[string]struct{}{"adapter": {}}
	registry.interfaceContractDemands["source"] =
		map[string]interfaceContractDemand{"target": {}}
	registry.interfaceReflectionDemands["reflection"] =
		interfaceReflectionDemand{}
	registry.reflectionValueDemands["value"] = struct{}{}
	registry.reflectionValueContracts["contract"] = struct{}{}

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
		len(registry.interfaceContractDemands) != 0 ||
		len(registry.interfaceReflectionDemands) != 0 ||
		len(registry.reflectionValueDemands) != 0 ||
		len(registry.reflectionValueContracts) != 0 {
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
