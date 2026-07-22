package analyze

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestResealedTamperNeverGainsAuthority proves sealing is producer-owned in
// effect: an artifact whose content was tampered and then CORRECTLY RESEALED
// passes the self-seal check but is rejected by the external certified-digest
// binding — the only authority ordinary consumption honors.
func TestResealedTamperNeverGainsAuthority(t *testing.T) {
	artifact := &AuditArtifact{
		SchemaVersion: AuditArtifactVersion,
		Meta:          AuditMeta{ToolchainVersion: "go1.26", GOOS: "linux", GOARCH: "amd64"},
		Files: []AuditFile{{
			File: "std::errors/errors.go", Digest: "aa11", Occurrences: 3,
			Units: []ManifestUnit{
				{Unit: "std::errors/errors.go#10-20/func-body", Kind: 1, Start: 10, End: 20, Hash: "bb22"},
				{Unit: "std::errors/errors.go#12-18/funclit-body", Kind: 2, Start: 12, End: 18, Hash: "cc33"},
			},
		}},
		Occurrences: 3,
	}
	artifact.seal()
	certified := artifact.ArtifactDigest
	write := func(a *AuditArtifact) string {
		path := filepath.Join(t.TempDir(), "artifact.json")
		if err := WriteAuditArtifact(a, path); err != nil {
			t.Fatal(err)
		}
		return path
	}
	if _, err := DecodeAuditArtifactBound(write(artifact), certified); err != nil {
		t.Fatalf("untampered bound decode failed: %v", err)
	}
	// Reseal an interior-unit omission: the self-seal is VALID again.
	omitted := *artifact
	omitted.Files = []AuditFile{artifact.Files[0]}
	omitted.Files[0].Units = artifact.Files[0].Units[:1]
	omitted.seal()
	if omitted.ArtifactDigest != omitted.canonicalDigest() {
		t.Fatal("reseal did not produce a valid self-seal")
	}
	if _, err := DecodeAuditArtifact(write(&omitted)); err != nil {
		t.Fatalf("resealed artifact must pass the standalone seal check (that is the threat): %v", err)
	}
	if _, err := DecodeAuditArtifactBound(write(&omitted), certified); err == nil ||
		!strings.Contains(err.Error(), "certified digest") {
		t.Errorf("resealed interior omission gained authority: %v", err)
	}
	// Reseal a CDependent flip: same class.
	flipped := *artifact
	flipped.Files = []AuditFile{artifact.Files[0]}
	flipped.Files[0].Units = append([]ManifestUnit(nil), artifact.Files[0].Units...)
	flipped.Files[0].Units[1].CDependent = true
	flipped.seal()
	if _, err := DecodeAuditArtifactBound(write(&flipped), certified); err == nil ||
		!strings.Contains(err.Error(), "certified digest") {
		t.Errorf("resealed CDependent flip gained authority: %v", err)
	}
	// A missing certified digest is a typed failure, never optional trust.
	if _, err := DecodeAuditArtifactBound(write(artifact), ""); err == nil {
		t.Error("bound decode accepted an empty certified digest")
	}
	_ = os.Remove
}
