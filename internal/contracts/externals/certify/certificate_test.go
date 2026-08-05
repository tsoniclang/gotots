package certify

import (
	"path/filepath"
	"runtime"
	"testing"

	environmentcontract "github.com/tsoniclang/gotots/internal/contracts/environment"
	"github.com/tsoniclang/gotots/internal/contracts/externals"
)

func TestCheckedExternalProviderCertificateIsReproducible(t *testing.T) {
	certificate, err := Verify(testConfig(t))
	if err != nil {
		t.Fatal(err)
	}
	if !certificate.Valid() || certificate.ManifestDigest() == "" ||
		certificate.StandardLibraryDigest() == "" ||
		certificate.IntegerRepresentation() != "number" ||
		certificate.ConcurrencySemantics() != "cooperative" {
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
	profile, err := environmentcontract.NewBuildProfileForToolchain(
		runtime.Version(),
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
		BuildProfile:                profile,
		Backend:                     "node",
		IntegerRepresentation:       "number",
		ConcurrencySemantics:        "cooperative",
	}
}
