package certify

import (
	"path/filepath"
	"slices"
	"strings"
	"testing"

	environmentcontract "github.com/tsoniclang/gotots/internal/contracts/environment"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
	gotool "github.com/tsoniclang/gotots/internal/toolchain"
)

func TestCertificationSeparatesNativeToolingFromSelectedSourceProfile(
	t *testing.T,
) {
	repository := filepath.Join("..", "..", "..", "..")
	selectedGo, selectedTSGo := resolveTestTools(t, repository)
	profile, err := environmentcontract.NewBuildProfileForToolchain(
		selectedGo.Version(),
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
		goTool:         selectedGo,
		tsgoTool:       selectedTSGo,
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
		selectedGo.Environment(profile),
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
	projectRoot := filepath.Join("..", "..", "..", "..")
	selectedGo, selectedTSGo := resolveTestTools(t, projectRoot)
	_, err := resolveConfig(Config{
		RepositoryRoot:      repository,
		ProviderRoot:        repository,
		ManifestPath:        filepath.Join(repository, "manifest.json"),
		ModuleMapPath:       filepath.Join(repository, "modules.json"),
		FacetMapPath:        filepath.Join(repository, "facets.json"),
		RuntimeContractPath: filepath.Join(repository, "runtime.json"),
		TSConfigPath:        filepath.Join(repository, "tsconfig.json"),
		ScratchDirectory:    filepath.Join(repository, "scratch"),
		GoTool:              selectedGo,
		TSGoTool:            selectedTSGo,
		Backend:             "node",
		MinimumGoVersion:    selectedGo.Version(),
		MaximumGoVersion:    selectedGo.Version(),
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

func resolveTestTools(t *testing.T, repository string) (gotool.Go, tsgo.Tool) {
	t.Helper()
	repository, err := filepath.Abs(repository)
	if err != nil {
		t.Fatal(err)
	}
	selectedGo, err := gotool.ResolveGo(
		"",
		filepath.Join(repository, ".temp", "cache", "toolchain-tests"),
	)
	if err != nil {
		t.Fatal(err)
	}
	selectedTSGo, err := tsgo.ResolveTool(selectedGo, repository, "")
	if err != nil {
		t.Fatal(err)
	}
	return selectedGo, selectedTSGo
}
