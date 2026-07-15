// Publication containment fixture: output paths overlapping the pinned
// source tree are refused. Broader publication safety contracts live in
// publication_test.go.
package census

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestCensusRejectsOutputInsideSource(t *testing.T) {
	dir, revision := writeFixtureRepo(t, basicFixtureFiles())
	prof := writeFixtureConfig(t, revision, basicFixtureProfile())

	result, err := Run(prof, dir, "fixture")
	if err != nil {
		t.Fatalf("census: %v", err)
	}
	err = WriteReports(result, filepath.Join(dir, "reports"))
	if err == nil || !strings.Contains(err.Error(), "inside the pinned source tree") {
		t.Fatalf("expected containment refusal, got %v", err)
	}
}
