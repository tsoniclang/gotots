package implementation

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

func TestCertificationSourcesAreSealedSortedAndImmutable(t *testing.T) {
	root := t.TempDir()
	first := filepath.Join(root, "a.d.ts")
	second := filepath.Join(root, "b.d.ts")
	writeCertificationFixture(t, first, "declare const first: string;\n")
	writeCertificationFixture(t, second, "declare const second: number;\n")

	sources, err := LoadCertificationSources([]string{second, first})
	if err != nil {
		t.Fatal(err)
	}
	if got := []string{sources[0].SourcePath(), sources[1].SourcePath()}; !slices.Equal(got, []string{first, second}) {
		t.Fatalf("source order = %v", got)
	}
	mutated := slices.Clone(sources)
	mutated[0] = CertificationSource{}
	if !sources[0].Valid() {
		t.Fatal("caller mutation changed sealed source evidence")
	}
	if _, err := VerifyCertificationSource(sources[0]); err != nil {
		t.Fatal(err)
	}
}

func TestCertificationSourceDriftAndDuplicatesFailClosed(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "contract.d.ts")
	writeCertificationFixture(t, path, "declare const selected: string;\n")
	sources, err := LoadCertificationSources([]string{path})
	if err != nil {
		t.Fatal(err)
	}
	writeCertificationFixture(t, path, "declare const changed: string;\n")
	if _, err := VerifyCertificationSource(sources[0]); err == nil ||
		!strings.Contains(err.Error(), "digest changed") {
		t.Fatalf("drift error = %v", err)
	}
	if _, err := LoadCertificationSources([]string{path, path}); err == nil ||
		!strings.Contains(err.Error(), "duplicated") {
		t.Fatalf("selection duplicate error = %v", err)
	}
	if _, err := MergeCertificationSources(sources, sources); err == nil ||
		!strings.Contains(err.Error(), "duplicated") {
		t.Fatalf("merge duplicate error = %v", err)
	}
}

func TestCertificationSourceRejectsNonDeclarationPath(t *testing.T) {
	path := filepath.Join(t.TempDir(), "runtime.ts")
	writeCertificationFixture(t, path, "export const runtime = 1;\n")
	if _, err := LoadCertificationSources([]string{path}); err == nil {
		t.Fatal("non-declaration source was admitted")
	}
}

func writeCertificationFixture(t *testing.T, path string, source string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(source), 0o600); err != nil {
		t.Fatal(err)
	}
}
