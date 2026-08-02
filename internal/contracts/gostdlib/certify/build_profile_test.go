package certify

import (
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"testing"

	environmentcontract "github.com/tsoniclang/gotots/internal/contracts/environment"
)

func TestCertificationSeparatesNativeToolingFromSelectedSourceProfile(
	t *testing.T,
) {
	profile, err := environmentcontract.NewBuildProfile(
		"js",
		"wasm",
		false,
		[]string{"noasm"},
	)
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("GOOS", "plan9")
	t.Setenv("GOARCH", "386")
	t.Setenv("CGO_ENABLED", "1")
	t.Setenv("GOFLAGS", "-tags=ambientmustnotselect")

	selected, err := inspectToolchain(resolvedConfig{
		repositoryRoot: t.TempDir(),
		goBinary:       filepath.Join(runtime.GOROOT(), "bin", "go"),
		buildProfile:   profile,
	})
	if err != nil {
		t.Fatal(err)
	}
	if selected.profile.GOOS() != "js" ||
		selected.profile.GOARCH() != "wasm" ||
		!slices.Equal(selected.profile.Tags(), []string{"noasm"}) {
		t.Fatalf("selected profile = %#v", selected.profile)
	}
	wantKey, err := environmentcontract.ToolchainKey(profile)
	if err != nil {
		t.Fatal(err)
	}
	if selected.key != wantKey {
		t.Fatalf("selected key = %q, want %q", selected.key, wantKey)
	}
	environment := environmentByName(
		exactGoEnvironment(selected, filepath.Dir(selected.root)),
	)
	for name, want := range map[string]string{
		"GOOS":        "js",
		"GOARCH":      "wasm",
		"CGO_ENABLED": "0",
		"GOFLAGS":     "",
		"GOTOOLCHAIN": "local",
	} {
		if environment[name] != want {
			t.Fatalf("source environment %s = %q, want %q", name, environment[name], want)
		}
	}
	if got := selected.profile.BuildFlags(); !slices.Equal(got, []string{"-tags=noasm"}) {
		t.Fatalf("selected build flags = %v", got)
	}
}

func TestCertificationRejectsAbsentBuildProfile(t *testing.T) {
	repository := t.TempDir()
	_, err := resolveConfig(Config{
		RepositoryRoot:      repository,
		ProviderRoot:        repository,
		ManifestPath:        filepath.Join(repository, "manifest.json"),
		ModuleMapPath:       filepath.Join(repository, "modules.json"),
		FacetMapPath:        filepath.Join(repository, "facets.json"),
		RuntimeContractPath: filepath.Join(repository, "runtime.json"),
		TSConfigPath:        filepath.Join(repository, "tsconfig.json"),
		ScratchDirectory:    filepath.Join(repository, "scratch"),
		GoBinary:            filepath.Join(runtime.GOROOT(), "bin", "go"),
		Backend:             "node",
		MinimumGoVersion:    runtime.Version(),
		MaximumGoVersion:    runtime.Version(),
	})
	if err == nil || !strings.Contains(err.Error(), "build profile") {
		t.Fatalf("absent build profile error = %v", err)
	}
}

func environmentByName(values []string) map[string]string {
	result := make(map[string]string, len(values))
	for _, value := range values {
		name, selected, found := strings.Cut(value, "=")
		if found {
			result[name] = selected
		}
	}
	return result
}
