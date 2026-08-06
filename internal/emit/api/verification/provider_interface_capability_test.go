package api_test

import (
	"errors"
	"go/token"
	"go/types"
	"testing"

	. "github.com/tsoniclang/gotots/internal/emit/api"
)

func TestProviderInterfaceCapabilityRequirementCarriesExactContract(t *testing.T) {
	artifact, contract := providerBridgeArtifact(t)
	requirement, err := NewProviderInterfaceCapabilityRequirement(
		artifact,
		contract,
		"target-contract",
	)
	if err != nil {
		t.Fatal(err)
	}
	selected, gotContract, key, ok :=
		requirement.ProviderInterfaceCapability()
	if !ok || selected != artifact || gotContract != contract ||
		key != "target-contract" {
		t.Fatalf(
			"provider capability = %p, %p, %q, %t",
			selected,
			gotContract,
			key,
			ok,
		)
	}
	if requirement.Kind() !=
		DeclarationRequirementProviderInterfaceCapability {
		t.Fatalf("requirement kind = %d", requirement.Kind())
	}
}

func TestProviderInterfaceCapabilityRequirementRejectsInvalidContracts(t *testing.T) {
	artifact, contract := providerBridgeArtifact(t)
	for _, testCase := range []struct {
		name     string
		artifact *GeneratedArtifact
		contract *types.Interface
		key      string
	}{
		{name: "nil artifact", contract: contract, key: "target"},
		{name: "nil contract", artifact: artifact, key: "target"},
		{name: "empty key", artifact: artifact, contract: contract},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			_, err := NewProviderInterfaceCapabilityRequirement(
				testCase.artifact,
				testCase.contract,
				testCase.key,
			)
			var requestError *RootRequestError
			if !errors.As(err, &requestError) {
				t.Fatalf("error = %#v, want RootRequestError", err)
			}
		})
	}
}

func providerBridgeArtifact(
	t *testing.T,
) (*GeneratedArtifact, *types.Interface) {
	t.Helper()
	sourcePackage := types.NewPackage("example.com/provider", "provider")
	baseMethod := types.NewFunc(
		token.NoPos,
		sourcePackage,
		"Base",
		types.NewSignatureType(nil, nil, nil, nil, nil, false),
	)
	base := types.NewInterfaceType([]*types.Func{baseMethod}, nil).Complete()
	typeName := types.NewTypeName(
		token.NoPos,
		sourcePackage,
		"Provider",
		nil,
	)
	named := types.NewNamed(typeName, base, nil)
	artifact, err := NewCompilationGeneratedArtifact(
		GeneratedArtifactProviderInterfaceBridge,
		named,
		"provider-key",
		"ProviderBridge",
		"support/provider-bridge.ts",
	)
	if err != nil {
		t.Fatal(err)
	}
	targetMethod := types.NewFunc(
		token.NoPos,
		sourcePackage,
		"Target",
		types.NewSignatureType(nil, nil, nil, nil, nil, false),
	)
	target := types.NewInterfaceType([]*types.Func{targetMethod}, nil).Complete()
	return artifact, target
}
