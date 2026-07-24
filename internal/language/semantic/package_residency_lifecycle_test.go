package semantic

import (
	"errors"
	"path/filepath"
	"testing"
)

func TestCheckerModelReleasesPackageAfterErrorAndPanic(t *testing.T) {
	pkg := semanticWirePackage(t)
	model := checkerModelForTest(t, pkg)
	sentinel := errors.New("stop projection")

	for range 3 {
		err := model.VisitPackage(pkg.ID(), func(Package) error {
			assertCheckerModelResident(t, model, 1)
			return sentinel
		})
		if !errors.Is(err, sentinel) {
			t.Fatalf("projection error = %v, want %v", err, sentinel)
		}
		assertCheckerModelResident(t, model, 0)
	}

	panicked := false
	func() {
		defer func() {
			if recover() != nil {
				panicked = true
			}
		}()
		_ = model.VisitPackage(pkg.ID(), func(Package) error {
			assertCheckerModelResident(t, model, 1)
			panic("stop projection")
		})
	}()
	if !panicked {
		t.Fatal("projection callback panic was not propagated")
	}
	assertCheckerModelResident(t, model, 0)

	modelStats := model.ProjectionReadStats()
	checkerStats := model.CheckerReadStats()
	if modelStats.PackageLoads != 4 ||
		modelStats.MaxPackagesResident != 1 ||
		checkerStats.ShardLoads != 4 ||
		checkerStats.MaxPackagesResident != 1 {
		t.Fatalf(
			"projection lifecycle stats = model %+v, checker %+v",
			modelStats,
			checkerStats,
		)
	}
}

func TestProviderArtifactReleasesPackageAfterErrorAndPanic(t *testing.T) {
	pkg := semanticWirePackage(t)
	fixture := semanticFixture(t)
	path := filepath.Join(t.TempDir(), "semantic-provider.bin")
	writer, err := NewProviderArtifactWriter(
		ProviderArtifactContext{
			ToolchainDigest:     fixture.authority.ToolchainDigest(),
			ConfigurationDigest: fixture.authority.Configuration(),
			ContractID:          "portable@v1",
			ContractFingerprint: fixture.authority.PackageInput(),
		},
		path,
	)
	if err != nil {
		t.Fatal(err)
	}
	if err := writer.Append(pkg); err != nil {
		writer.Abort()
		t.Fatal(err)
	}
	result, err := writer.Finish(fixture.authority.StructureDigest())
	if err != nil {
		writer.Abort()
		t.Fatal(err)
	}
	artifact, err := DecodeProviderArtifact(path, result.Digest)
	if err != nil {
		t.Fatal(err)
	}
	sentinel := errors.New("stop provider projection")

	for range 3 {
		err := artifact.VisitPackage(pkg.ID(), func(Package) error {
			assertProviderResident(t, artifact, 1)
			return sentinel
		})
		if !errors.Is(err, sentinel) {
			t.Fatalf("provider error = %v, want %v", err, sentinel)
		}
		assertProviderResident(t, artifact, 0)
	}

	panicked := false
	func() {
		defer func() {
			if recover() != nil {
				panicked = true
			}
		}()
		_ = artifact.VisitPackage(pkg.ID(), func(Package) error {
			assertProviderResident(t, artifact, 1)
			panic("stop provider projection")
		})
	}()
	if !panicked {
		t.Fatal("provider callback panic was not propagated")
	}
	assertProviderResident(t, artifact, 0)

	stats := artifact.ReadStats()
	if stats.ShardLoads != 4 ||
		stats.MaxProviderPackagesResident != 1 {
		t.Fatalf("provider lifecycle stats = %+v", stats)
	}
}

func assertCheckerModelResident(
	t *testing.T,
	model *Model,
	want int,
) {
	t.Helper()
	model.mu.Lock()
	modelResident := model.resident
	model.mu.Unlock()
	model.checker.mu.Lock()
	checkerResident := model.checker.resident
	model.checker.mu.Unlock()
	if modelResident != want || checkerResident != want {
		t.Fatalf(
			"resident packages = model %d, checker %d, want %d",
			modelResident,
			checkerResident,
			want,
		)
	}
}

func assertProviderResident(
	t *testing.T,
	artifact *ProviderArtifact,
	want int,
) {
	t.Helper()
	artifact.mu.Lock()
	resident := artifact.resident
	artifact.mu.Unlock()
	if resident != want {
		t.Fatalf(
			"resident provider packages = %d, want %d",
			resident,
			want,
		)
	}
}
