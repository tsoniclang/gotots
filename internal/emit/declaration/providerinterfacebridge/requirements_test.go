package providerinterfacebridge

import (
	"go/importer"
	"go/token"
	"go/types"
	"os"
	"path/filepath"
	"testing"

	"github.com/tsoniclang/gotots/internal/contracts/gostdlib"
	"github.com/tsoniclang/gotots/internal/emit/api"
)

func TestRequirementsExactJoinAndOrder(t *testing.T) {
	artifact, first, second := requirementFixture(t)
	definition, err := api.NewProviderInterfaceBridgeRequirement(artifact)
	if err != nil {
		t.Fatal(err)
	}
	secondDemand, err := api.NewProviderInterfaceCapabilityRequirement(
		artifact,
		second,
		"second",
	)
	if err != nil {
		t.Fatal(err)
	}
	firstDemand, err := api.NewProviderInterfaceCapabilityRequirement(
		artifact,
		first,
		"first",
	)
	if err != nil {
		t.Fatal(err)
	}
	contracts, err := Requirements(
		artifact,
		[]api.DeclarationRequirement{
			secondDemand,
			definition,
			firstDemand,
			firstDemand,
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	if len(contracts) != 2 ||
		contracts[0].Key != "first" ||
		contracts[0].Contract != first ||
		contracts[1].Key != "second" ||
		contracts[1].Contract != second {
		t.Fatalf("capability contracts = %#v", contracts)
	}
}

func TestRequirementsRejectsMissingDefinitionAndDivergentDuplicate(t *testing.T) {
	artifact, first, second := requirementFixture(t)
	firstDemand, err := api.NewProviderInterfaceCapabilityRequirement(
		artifact,
		first,
		"same",
	)
	if err != nil {
		t.Fatal(err)
	}
	secondDemand, err := api.NewProviderInterfaceCapabilityRequirement(
		artifact,
		second,
		"same",
	)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Requirements(
		artifact,
		[]api.DeclarationRequirement{firstDemand},
	); err == nil {
		t.Fatal("missing definition requirement was accepted")
	}
	definition, err := api.NewProviderInterfaceBridgeRequirement(artifact)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Requirements(
		artifact,
		[]api.DeclarationRequirement{
			definition,
			firstDemand,
			secondDemand,
		},
	); err == nil {
		t.Fatal("non-identical contracts sharing one key were accepted")
	}
}

func TestProfileRequirementsExactJoinNamedCapabilityTargets(t *testing.T) {
	artifact, target := profileRequirementFixture(t)
	definition, err := api.NewProviderInterfaceBridgeRequirement(artifact)
	if err != nil {
		t.Fatal(err)
	}
	demand, err := api.NewProviderProfileInterfaceCapabilityRequirement(
		artifact,
		target,
	)
	if err != nil {
		t.Fatal(err)
	}
	capabilities, contracts, err := ProfileRequirements(
		artifact,
		[]api.DeclarationRequirement{demand, definition, demand},
	)
	if err != nil {
		t.Fatal(err)
	}
	if len(capabilities) != 0 || len(contracts) != 1 || contracts[0].Target != target {
		t.Fatalf("profile capability contracts = %#v", contracts)
	}
	if _, err := Requirements(
		artifact,
		[]api.DeclarationRequirement{definition},
	); err == nil {
		t.Fatal("profile artifact was admitted by ordinary capability requirements")
	}
}

func requirementFixture(
	t *testing.T,
) (*api.GeneratedArtifact, *types.Interface, *types.Interface) {
	t.Helper()
	sourcePackage := types.NewPackage("example.com/provider", "provider")
	interfaceWith := func(name string) *types.Interface {
		method := types.NewFunc(
			token.NoPos,
			sourcePackage,
			name,
			types.NewSignatureType(nil, nil, nil, nil, nil, false),
		)
		return types.NewInterfaceType([]*types.Func{method}, nil).Complete()
	}
	base := interfaceWith("Base")
	typeName := types.NewTypeName(
		token.NoPos,
		sourcePackage,
		"Provider",
		nil,
	)
	named := types.NewNamed(typeName, base, nil)
	artifact, err := api.NewCompilationGeneratedArtifact(
		api.GeneratedArtifactProviderInterfaceBridge,
		named,
		"provider-key",
		"ProviderBridge",
		"support/provider-bridge.ts",
	)
	if err != nil {
		t.Fatal(err)
	}
	return artifact, interfaceWith("First"), interfaceWith("Second")
}

func profileRequirementFixture(
	t *testing.T,
) (*api.GeneratedArtifact, *api.GeneratedArtifact) {
	t.Helper()
	payload, err := os.ReadFile(filepath.Join(
		"..", "..", "..", "..",
		"gostdlib", "contract", "manifest.json",
	))
	if err != nil {
		t.Fatal(err)
	}
	manifest, err := gostdlib.Parse(payload)
	if err != nil {
		t.Fatal(err)
	}
	profiles := manifest.ProviderCallableProfiles(
		"io/fs|kind=4|receiver=|name=ReadFile",
	)
	if len(profiles) != 1 {
		t.Fatalf("ReadFile profiles = %#v", profiles)
	}
	goPackage, err := importer.Default().Import("io/fs")
	if err != nil {
		t.Fatal(err)
	}
	baseName, _ := goPackage.Scope().Lookup("FS").(*types.TypeName)
	targetName, _ := goPackage.Scope().Lookup("ReadFileFS").(*types.TypeName)
	base, _ := types.Unalias(baseName.Type()).(*types.Named)
	target, _ := types.Unalias(targetName.Type()).(*types.Named)
	if base == nil || target == nil {
		t.Fatal("io/fs named interfaces are absent")
	}
	artifact, err := api.NewCompilationProviderProfileBridgeArtifact(
		base,
		profiles[0].Interfaces(),
		"profile-key",
		"ProfileBridge",
		"support/profile-bridge.ts",
	)
	if err != nil {
		t.Fatal(err)
	}
	targetArtifact, err := api.NewCompilationProviderProfileBridgeArtifact(
		target,
		profiles[0].Interfaces(),
		"target-profile-key",
		"TargetProfileBridge",
		"support/target-profile-bridge.ts",
	)
	if err != nil {
		t.Fatal(err)
	}
	return artifact, targetArtifact
}
