package naming

import (
	"go/importer"
	"go/types"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/tsoniclang/gotots/internal/contracts/gostdlib"
)

func TestProviderProfileBridgeIdentityIgnoresUnrelatedCallableInterfaces(
	t *testing.T,
) {
	payload, err := os.ReadFile(filepath.Join(
		"..", "..", "..", "gostdlib", "contract", "manifest.json",
	))
	if err != nil {
		t.Fatal(err)
	}
	manifest, err := gostdlib.Parse(payload)
	if err != nil {
		t.Fatal(err)
	}
	callable := manifest.ProviderCallableProfiles(
		"compress/gzip|kind=4|receiver=|name=NewReader",
	)
	stateful := manifest.ProviderStatefulProfiles(
		"compress/gzip|kind=2|receiver=|name=Reader",
	)
	if len(callable) != 1 || len(stateful) != 1 {
		t.Fatalf("gzip profiles = callable %d, stateful %d", len(callable), len(stateful))
	}
	errorName, ok := types.Universe.Lookup("error").(*types.TypeName)
	if !ok {
		t.Fatal("predeclared error type is absent")
	}
	errorType, ok := types.Unalias(errorName.Type()).(*types.Named)
	if !ok {
		t.Fatal("predeclared error type is not named")
	}

	registry := NewRegistry()
	first, err := registry.internProviderProfileInterfaceBridge(
		"go:universe|error",
		errorType,
		"$goProviderProfileBridge$Named_error",
		callable[0].Interfaces(),
	)
	if err != nil {
		t.Fatal(err)
	}
	second, err := registry.internProviderProfileInterfaceBridge(
		"go:universe|error",
		errorType,
		"$goProviderProfileBridge$Named_error",
		stateful[0].Interfaces(),
	)
	if err != nil {
		t.Fatal(err)
	}
	if first.owner != second.owner || first.name != second.name || first.key != second.key {
		t.Fatalf(
			"same error ABI created distinct bridges: %q/%q and %q/%q",
			first.key,
			first.name,
			second.key,
			second.name,
		)
	}
}

func TestProviderProfileBridgeIdentityIncludesReachableInterfaceABI(
	t *testing.T,
) {
	payload, err := os.ReadFile(filepath.Join(
		"..", "..", "..", "gostdlib", "contract", "manifest.json",
	))
	if err != nil {
		t.Fatal(err)
	}
	manifest, err := gostdlib.Parse(payload)
	if err != nil {
		t.Fatal(err)
	}
	profiles := manifest.ProviderCallableProfiles(
		"compress/gzip|kind=4|receiver=|name=NewReader",
	)
	if len(profiles) != 1 {
		t.Fatalf("gzip NewReader profiles = %d", len(profiles))
	}
	ioPackage, err := importer.Default().Import("io")
	if err != nil {
		t.Fatal(err)
	}
	readerName, ok := ioPackage.Scope().Lookup("Reader").(*types.TypeName)
	if !ok {
		t.Fatal("io.Reader type is absent")
	}
	reader, ok := types.Unalias(readerName.Type()).(*types.Named)
	if !ok {
		t.Fatal("io.Reader type is not named")
	}
	closure, err := providerProfileBridgeClosure(
		reader,
		profiles[0].Interfaces(),
	)
	if err != nil {
		t.Fatal(err)
	}
	identities := make([]string, 0, len(closure))
	for _, selected := range closure {
		identities = append(identities, selected.SourceIdentity())
	}
	want := []string{
		"go:universe|error",
		"io|kind=2|receiver=|name=Reader",
	}
	if !slices.Equal(identities, want) {
		t.Fatalf("io.Reader profile closure = %q, want %q", identities, want)
	}
}
