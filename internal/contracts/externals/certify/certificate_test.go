package certify

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	environmentcontract "github.com/tsoniclang/gotots/internal/contracts/environment"
	"github.com/tsoniclang/gotots/internal/contracts/externals"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
	"github.com/tsoniclang/gotots/internal/toolchain"
)

func TestCheckedExternalProviderCertificateIsReproducible(t *testing.T) {
	certificate, err := Verify(testConfig(t))
	if err != nil {
		t.Fatal(err)
	}
	if !certificate.Valid() || certificate.ManifestDigest() == "" ||
		certificate.StandardLibraryDigest() == "" ||
		certificate.ProviderIntegerRepresentation() != "bigint" {
		t.Fatal("external provider certificate is incomplete")
	}
	bindings := certificate.Bindings()
	if len(bindings) != 9 {
		t.Fatalf("binding count = %d, want 9", len(bindings))
	}
	var modules int
	var sources int
	for _, binding := range bindings {
		switch binding.TargetKind() {
		case externals.TargetModule:
			modules++
		case externals.TargetSource:
			sources++
		default:
			t.Fatalf("invalid target kind %q", binding.TargetKind())
		}
	}
	if modules != 3 || sources != 6 {
		t.Fatalf("target counts = module %d, source %d", modules, sources)
	}
}

func TestStandardLibraryRuntimeContractOwnsProviderProfile(t *testing.T) {
	config := testConfig(t)
	payload, err := os.ReadFile(config.StandardLibraryRuntimePath)
	if err != nil {
		t.Fatal(err)
	}
	mutatedPath := filepath.Join(t.TempDir(), "runtime.json")
	if err := os.WriteFile(mutatedPath, append(payload, '\n'), 0o644); err != nil {
		t.Fatal(err)
	}
	_, _, err = readStandardLibraryContract(
		config.StandardLibraryManifestPath,
		mutatedPath,
	)
	if err == nil || !strings.Contains(
		err.Error(),
		"content digest does not match the standard-library manifest",
	) {
		t.Fatalf("runtime-contract mutation error = %v", err)
	}
}

func testConfig(t *testing.T) Config {
	t.Helper()
	_, sourceFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("resolve test source")
	}
	repository := filepath.Clean(filepath.Join(
		filepath.Dir(sourceFile),
		"..", "..", "..", "..",
	))
	selectedGo, err := toolchain.ResolveGo(
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
	profile, err := environmentcontract.NewBuildProfileForToolchain(
		selectedGo.Version(),
		"linux",
		"amd64",
		false,
		[]string{"noasm"},
	)
	if err != nil {
		t.Fatal(err)
	}
	return Config{
		RepositoryRoot:              repository,
		ProviderRoot:                filepath.Join(repository, "externals"),
		ManifestPath:                filepath.Join(repository, "externals", "contract", "manifest.json"),
		BindingMapPath:              filepath.Join(repository, "externals", "contract", "bindings.json"),
		TSConfigPath:                filepath.Join(repository, "externals", "tsconfig.json"),
		StandardLibraryManifestPath: filepath.Join(repository, "gostdlib", "contract", "manifest.json"),
		StandardLibraryRuntimePath:  filepath.Join(repository, "gostdlib", "contract", "runtime.json"),
		BuildProfile:                profile,
		Backend:                     "node",
		GoTool:                      selectedGo,
		TSGoTool:                    selectedTSGo,
	}
}
