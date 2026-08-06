package certify

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func TestProviderSupportParameterOrderIsCertified(t *testing.T) {
	repository, err := filepath.Abs("../../../..")
	if err != nil {
		t.Fatal(err)
	}
	provider := filepath.Join(repository, "gostdlib")
	client, err := tsgo.StartClient(repository, provider)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := client.Close(); err != nil {
			t.Errorf("close TS-Go client: %v", err)
		}
	})
	project, err := client.OpenProject(filepath.Join(provider, "tsconfig.json"))
	if err != nil {
		t.Fatal(err)
	}
	markers, err := loadProviderSupportMarkers(
		resolvedConfig{providerRoot: provider},
		project,
	)
	if err != nil {
		t.Fatal(err)
	}
	exports, err := project.Exports(filepath.Join(
		provider,
		"src/internal/facets/provider-io-fs.ts",
	))
	if err != nil {
		t.Fatal(err)
	}
	var target tsgo.ProjectExport
	for _, selected := range exports {
		if selected.Name() == "IoFsWalkDirCanonical" {
			target = selected
			break
		}
	}
	if target.Name() == "" {
		t.Fatal("IoFsWalkDirCanonical is absent")
	}
	if err := verifyProviderSupportParameters(
		project,
		target,
		5,
		3,
		0,
		1,
		1,
		markers,
	); err != nil {
		t.Fatal(err)
	}
	mutated := markers
	mutated.contract, mutated.fromProvider =
		mutated.fromProvider, mutated.contract
	err = verifyProviderSupportParameters(
		project,
		target,
		5,
		3,
		0,
		1,
		1,
		mutated,
	)
	if err == nil || !strings.Contains(err.Error(), "parameter 8") {
		t.Fatalf("contract/bridge ordering mutation error = %v", err)
	}
}
