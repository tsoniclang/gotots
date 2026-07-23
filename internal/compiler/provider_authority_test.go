package compiler

import (
	"testing"

	"github.com/tsoniclang/gotots/internal/identity"
	"github.com/tsoniclang/gotots/internal/language/structure"
)

func assertBoundedProviderAuthorities(
	t *testing.T,
	inspection *Inspection,
	manifest structure.ProviderManifestStats,
) {
	t.Helper()
	projections := inspection.Structure().AuthorityProjections()
	certifiedFiles := 0
	for _, projection := range projections {
		files := projection.CertifiedFiles()
		certifiedFiles += len(files)
		if len(files) == 0 {
			continue
		}
		original := files[0]
		files[0] = identity.FileID{}
		for _, current := range inspection.Structure().
			AuthorityProjections() {
			if current.ID() == projection.ID() &&
				current.CertifiedFiles()[0] != original {
				t.Fatal(
					"structural authority projection exposed mutable file census",
				)
			}
		}
	}
	if len(projections) != inspection.Structure().PackageCount() ||
		certifiedFiles != manifest.Files {
		t.Fatalf(
			"structural authority projections=%d/%d certifiedFiles=%d/%d",
			len(projections),
			inspection.Structure().PackageCount(),
			certifiedFiles,
			manifest.Files,
		)
	}
}
