package api_test

import (
	"errors"
	"go/importer"
	"go/types"
	"os"
	"path/filepath"
	"testing"

	"github.com/tsoniclang/gotots/internal/contracts/gostdlib"
	. "github.com/tsoniclang/gotots/internal/emit/api"
)

func TestProviderProfileCapabilityRequirementCarriesNamedTarget(t *testing.T) {
	manifestPath := filepath.Join(
		"..", "..", "..", "..",
		"gostdlib", "contract", "manifest.json",
	)
	payload, err := os.ReadFile(manifestPath)
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
	if len(profiles) != 1 || len(profiles[0].CapabilityViews()) != 1 {
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
	artifact, err := NewCompilationProviderProfileBridgeArtifact(
		base,
		profiles[0].Interfaces(),
		"profile-bridge-key",
		"ProfileBridge",
		"support/profile-bridge.ts",
	)
	if err != nil {
		t.Fatal(err)
	}
	targetArtifact, err := NewCompilationProviderProfileBridgeArtifact(
		target,
		profiles[0].Interfaces(),
		"target-profile-bridge-key",
		"TargetProfileBridge",
		"support/target-profile-bridge.ts",
	)
	if err != nil {
		t.Fatal(err)
	}
	requirement, err := NewProviderProfileInterfaceCapabilityRequirement(
		artifact,
		targetArtifact,
	)
	if err != nil {
		t.Fatal(err)
	}
	selectedArtifact, selectedTarget, ok :=
		requirement.ProviderProfileInterfaceCapability()
	if !ok || selectedArtifact != artifact || selectedTarget != targetArtifact ||
		requirement.Kind() !=
			DeclarationRequirementProviderProfileInterfaceCapability {
		t.Fatalf(
			"profile capability = %p, %p, %t, kind %d",
			selectedArtifact,
			selectedTarget,
			ok,
			requirement.Kind(),
		)
	}
}

func TestProviderProfileCapabilityRequirementRejectsUnprofiledAndUnrelatedTargets(
	t *testing.T,
) {
	manifestPath := filepath.Join(
		"..", "..", "..", "..",
		"gostdlib", "contract", "manifest.json",
	)
	payload, err := os.ReadFile(manifestPath)
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
		t.Fatalf("ReadFile profiles = %d", len(profiles))
	}
	goPackage, err := importer.Default().Import("io/fs")
	if err != nil {
		t.Fatal(err)
	}
	baseName, _ := goPackage.Scope().Lookup("FS").(*types.TypeName)
	targetName, _ := goPackage.Scope().Lookup("ReadFileFS").(*types.TypeName)
	unrelatedName, _ := goPackage.Scope().Lookup("File").(*types.TypeName)
	base, _ := types.Unalias(baseName.Type()).(*types.Named)
	target, _ := types.Unalias(targetName.Type()).(*types.Named)
	unrelated, _ := types.Unalias(unrelatedName.Type()).(*types.Named)
	if base == nil || target == nil || unrelated == nil {
		t.Fatal("io/fs capability types are absent")
	}
	artifact, err := NewCompilationProviderProfileBridgeArtifact(
		base,
		profiles[0].Interfaces(),
		"profile-bridge-key",
		"ProfileBridge",
		"support/profile-bridge.ts",
	)
	if err != nil {
		t.Fatal(err)
	}
	unprofiled, err := NewCompilationGeneratedArtifact(
		GeneratedArtifactProviderInterfaceBridge,
		target,
		"unprofiled-target-key",
		"UnprofiledTarget",
		"support/unprofiled-target.ts",
	)
	if err != nil {
		t.Fatal(err)
	}
	unrelatedArtifact, err := NewCompilationProviderProfileBridgeArtifact(
		unrelated,
		profiles[0].Interfaces(),
		"unrelated-target-key",
		"UnrelatedTarget",
		"support/unrelated-target.ts",
	)
	if err != nil {
		t.Fatal(err)
	}
	for _, testCase := range []struct {
		name   string
		target *GeneratedArtifact
	}{
		{name: "unprofiled target", target: unprofiled},
		{name: "unrelated target", target: unrelatedArtifact},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			_, err := NewProviderProfileInterfaceCapabilityRequirement(
				artifact,
				testCase.target,
			)
			var requestError *RootRequestError
			if !errors.As(err, &requestError) {
				t.Fatalf("error = %#v, want RootRequestError", err)
			}
		})
	}
}
