package naming

import (
	"go/importer"
	"go/types"
	"os"
	"path/filepath"
	"slices"
	"strings"
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
	if len(callable) != 2 || len(stateful) != 2 {
		t.Fatalf("gzip profiles = callable %d, stateful %d", len(callable), len(stateful))
	}
	var callableCanonical gostdlib.ProviderCallableProfile
	for _, profile := range callable {
		if profile.Export() == "GzipNewReaderCanonical" {
			callableCanonical = profile
		}
	}
	var statefulCanonical gostdlib.ProviderStatefulProfile
	for _, profile := range stateful {
		if profile.Export() == "CanonicalGzipReader" {
			statefulCanonical = profile
		}
	}
	if !callableCanonical.Valid() || !statefulCanonical.Valid() {
		t.Fatal("canonical gzip profile pair is absent")
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
		callableCanonical.Interfaces(),
	)
	if err != nil {
		t.Fatal(err)
	}
	second, err := registry.internProviderProfileInterfaceBridge(
		"go:universe|error",
		errorType,
		"$goProviderProfileBridge$Named_error",
		statefulCanonical.Interfaces(),
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
	if len(profiles) != 2 {
		t.Fatalf("gzip NewReader profiles = %d", len(profiles))
	}
	var canonical gostdlib.ProviderCallableProfile
	for _, profile := range profiles {
		if profile.Export() == "GzipNewReaderCanonical" {
			canonical = profile
		}
	}
	if !canonical.Valid() {
		t.Fatal("canonical gzip NewReader profile is absent")
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
		canonical.Interfaces(),
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

func TestProviderProfileBridgeIdentityIgnoresEquivalentProviderReexports(
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
	contextProfiles := manifest.ProviderCallableProfiles(
		"context|kind=4|receiver=|name=WithCancel",
	)
	signalProfiles := manifest.ProviderCallableProfiles(
		"os/signal|kind=4|receiver=|name=NotifyContext",
	)
	if len(contextProfiles) != 1 || len(signalProfiles) != 1 {
		t.Fatalf(
			"profiles = context %d, os/signal %d",
			len(contextProfiles),
			len(signalProfiles),
		)
	}
	contextPackage, err := importer.Default().Import("context")
	if err != nil {
		t.Fatal(err)
	}
	contextName, ok := contextPackage.Scope().Lookup("Context").(*types.TypeName)
	if !ok {
		t.Fatal("context.Context type is absent")
	}
	contextType, ok := types.Unalias(contextName.Type()).(*types.Named)
	if !ok {
		t.Fatal("context.Context type is not named")
	}

	registry := NewRegistry()
	first, err := registry.internProviderProfileInterfaceBridge(
		"std::context|Context",
		contextType,
		"$goProviderProfileBridge$Named_context_Context",
		contextProfiles[0].Interfaces(),
	)
	if err != nil {
		t.Fatal(err)
	}
	second, err := registry.internProviderProfileInterfaceBridge(
		"std::context|Context",
		contextType,
		"$goProviderProfileBridge$Named_context_Context",
		signalProfiles[0].Interfaces(),
	)
	if err != nil {
		t.Fatal(err)
	}
	if first.owner != second.owner || first.name != second.name || first.key != second.key {
		t.Fatalf(
			"equivalent context reexports created distinct bridges: %q/%q and %q/%q",
			first.key,
			first.name,
			second.key,
			second.name,
		)
	}
	const want = "$goProviderProfileBridge$Named_context_Context$Using$" +
		"context_Context$Awaitable$And$Error$Awaitable"
	if first.name != want {
		t.Fatalf("context bridge name = %q, want %q", first.name, want)
	}
}

func TestProviderProfileBridgeIdentityDistinguishesMethodEffects(t *testing.T) {
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
	readerProfiles := manifest.ProviderCallableProfiles(
		"bufio|kind=4|receiver=|name=NewReader",
	)
	synchronousProfiles := manifest.ProviderCallableProfiles(
		"errors|kind=4|receiver=|name=Unwrap",
	)
	if len(readerProfiles) != 2 || len(synchronousProfiles) != 2 {
		t.Fatalf(
			"profiles = reader %d, errors %d",
			len(readerProfiles),
			len(synchronousProfiles),
		)
	}
	var awaitableProfile gostdlib.ProviderCallableProfile
	for _, profile := range readerProfiles {
		selected, ok := profile.Interface(gostdlib.LanguageErrorInterfaceIdentity)
		if !ok {
			continue
		}
		methods := selected.ProviderInterface().Methods()
		if len(methods) == 1 && methods[0].Effect() == gostdlib.EffectAwaitable {
			awaitableProfile = profile
			break
		}
	}
	if !awaitableProfile.Valid() {
		t.Fatal("awaitable reader profile is absent")
	}
	var synchronous gostdlib.ProviderCallableProfile
	for _, profile := range synchronousProfiles {
		selected, ok := profile.Interface(gostdlib.LanguageErrorInterfaceIdentity)
		if !ok {
			continue
		}
		methods := selected.ProviderInterface().Methods()
		if len(methods) == 1 && methods[0].Effect() == gostdlib.EffectSynchronous {
			synchronous = profile
			break
		}
	}
	if !synchronous.Valid() {
		t.Fatal("synchronous error profile is absent")
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
	awaitable, err := registry.internProviderProfileInterfaceBridge(
		"go:universe|error",
		errorType,
		"$goProviderProfileBridge$Named_error",
		awaitableProfile.Interfaces(),
	)
	if err != nil {
		t.Fatal(err)
	}
	direct, err := registry.internProviderProfileInterfaceBridge(
		"go:universe|error",
		errorType,
		"$goProviderProfileBridge$Named_error",
		synchronous.Interfaces(),
	)
	if err != nil {
		t.Fatal(err)
	}
	if awaitable.owner == direct.owner || awaitable.name == direct.name ||
		awaitable.key == direct.key {
		t.Fatalf(
			"different error effects shared a bridge: %q/%q and %q/%q",
			awaitable.key,
			awaitable.name,
			direct.key,
			direct.name,
		)
	}
	if !strings.HasSuffix(awaitable.name, "$Error$Awaitable") ||
		!strings.HasSuffix(direct.name, "$Error$Direct") {
		t.Fatalf(
			"effect names are not readable: awaitable=%q direct=%q",
			awaitable.name,
			direct.name,
		)
	}
}

func TestProviderProfileBridgeNameIsBoundedBySemanticShape(t *testing.T) {
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
		"io/fs|kind=4|receiver=|name=ReadDir",
	)
	if len(profiles) != 1 {
		t.Fatalf("io/fs.ReadDir profiles = %d", len(profiles))
	}
	fileSystemPackage, err := importer.Default().Import("io/fs")
	if err != nil {
		t.Fatal(err)
	}
	readDirName, ok := fileSystemPackage.Scope().Lookup("ReadDirFS").(*types.TypeName)
	if !ok {
		t.Fatal("io/fs.ReadDirFS type is absent")
	}
	readDirType, ok := types.Unalias(readDirName.Type()).(*types.Named)
	if !ok {
		t.Fatal("io/fs.ReadDirFS type is not named")
	}

	registry := NewRegistry()
	binding, err := registry.internProviderProfileInterfaceBridge(
		"std::io/fs|ReadDirFS",
		readDirType,
		"$goProviderProfileBridge$Named_io_u2f_fs_ReadDirFS",
		profiles[0].Interfaces(),
	)
	if err != nil {
		t.Fatal(err)
	}
	if len(binding.name) > 320 || strings.Contains(binding.name, "$Method$") ||
		strings.Contains(binding.name, "_u7c_kind") {
		t.Fatalf("provider bridge name is not bounded and readable: %q", binding.name)
	}
}
