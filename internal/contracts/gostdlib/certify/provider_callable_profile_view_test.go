package certify

import (
	"go/importer"
	"go/types"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tsoniclang/gotots/internal/contracts/gostdlib"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func TestProviderCapabilityViewExactJoinsGoAndTypeScriptContracts(t *testing.T) {
	const (
		baseIdentity   = "io/fs|kind=2|receiver=|name=FS"
		targetIdentity = "io/fs|kind=2|receiver=|name=ReadFileFS"
	)
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
	exports, err := project.Exports(filepath.Join(
		provider,
		"src/internal/facets/provider-io-fs.ts",
	))
	if err != nil {
		t.Fatal(err)
	}
	targets := make(map[string]tsgo.ProjectExport, len(exports))
	for _, selected := range exports {
		targets[selected.Name()] = selected
	}
	profileTarget := targets["IoFsReadFileCanonical"]
	if profileTarget.Name() == "" {
		t.Fatal("IoFsReadFileCanonical is absent")
	}
	goPackage, err := importer.Default().Import("io/fs")
	if err != nil {
		t.Fatal(err)
	}
	baseObject, _ := goPackage.Scope().Lookup("FS").(*types.TypeName)
	targetObject, _ := goPackage.Scope().Lookup("ReadFileFS").(*types.TypeName)
	if baseObject == nil || targetObject == nil {
		t.Fatal("io/fs interface evidence is absent")
	}
	source := goSurface{objects: map[string]goObject{
		baseIdentity:   {object: baseObject},
		targetIdentity: {object: targetObject},
	}}
	interfaces := []gostdlib.ProviderCallableProfileInterfaceDocument{
		{SourceIdentity: baseIdentity, Export: "CanonicalFS"},
		{SourceIdentity: targetIdentity, Export: "CanonicalReadFileFS"},
	}
	views := []gostdlib.ProviderCallableProfileCapabilityViewDocument{{
		BaseSourceIdentity:   baseIdentity,
		TargetSourceIdentity: targetIdentity,
		TargetParameter:      "asReadFileFS",
	}}
	if err := verifyProviderCapabilityViews(
		project,
		profileTarget,
		3,
		views,
		interfaces,
		targets,
		source,
	); err != nil {
		t.Fatal(err)
	}
	views[0].BaseSourceIdentity, views[0].TargetSourceIdentity =
		views[0].TargetSourceIdentity, views[0].BaseSourceIdentity
	err = verifyProviderCapabilityViews(
		project,
		profileTarget,
		3,
		views,
		interfaces,
		targets,
		source,
	)
	if err == nil || !strings.Contains(err.Error(), "capability-view") {
		t.Fatalf("capability direction mutation error = %v", err)
	}
}
