package environmentobligation_test

import (
	"os"
	"path/filepath"
	"testing"

	environmentcontract "github.com/tsoniclang/gotots/internal/contracts/environment"
	"github.com/tsoniclang/gotots/internal/contracts/gostdlib/certify"
)

func writeProgramFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func linkedProviderCertificate(t *testing.T) *certify.Certificate {
	t.Helper()
	repository := filepath.Join("..", "..", "..", "..")
	certificate, err := certify.Verify(certify.Config{
		RepositoryRoot: repository,
		ProviderRoot:   filepath.Join(repository, "gostdlib"),
		ManifestPath: filepath.Join(
			repository, "gostdlib", "contract", "manifest.json",
		),
		ModuleMapPath: filepath.Join(
			repository, "gostdlib", "contract", "modules.json",
		),
		FacetMapPath: filepath.Join(
			repository, "gostdlib", "contract", "facets.json",
		),
		RuntimeContractPath: filepath.Join(
			repository, "gostdlib", "contract", "runtime.json",
		),
		TSConfigPath:     filepath.Join(repository, "gostdlib", "tsconfig.json"),
		ScratchDirectory: t.TempDir(),
		GoBinary:         "go",
		BuildProfile:     linkedProviderBuildProfile(t),
		Backend:          "node",
		MinimumGoVersion: "go1.26.4",
		MaximumGoVersion: "go1.26.4",
	})
	if err != nil {
		t.Fatal(err)
	}
	return certificate
}

func linkedProviderBuildProfile(t *testing.T) environmentcontract.BuildProfile {
	t.Helper()
	profile, err := environmentcontract.NewBuildProfile(
		"linux",
		"amd64",
		false,
		[]string{"noasm"},
	)
	if err != nil {
		t.Fatal(err)
	}
	return profile
}
