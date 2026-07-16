// The committed attestation protocol, machine-enforced: every retained
// gate report is an immutable revision-named artifact
// (attestations/gate-report-<revision>.json) whose filename revision
// equals its own implementationRevision, which must be an ancestor of
// HEAD reachable on the current branch. A mutable "latest" name is not
// evidence.
package policy

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestAttestationProtocol(t *testing.T) {
	root := repositoryRoot(t)
	dir := filepath.Join(root, "attestations")
	entries, err := os.ReadDir(dir)
	if os.IsNotExist(err) {
		return // no attestations retained yet
	}
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		name := entry.Name()
		if !strings.HasSuffix(name, ".json") {
			continue
		}
		rest, ok := strings.CutPrefix(name, "gate-report-")
		if !ok {
			t.Errorf("attestations/%s: not a revision-named gate report (gate-report-<revision>.json)", name)
			continue
		}
		revision := strings.TrimSuffix(rest, ".json")
		if len(revision) != 40 {
			t.Errorf("attestations/%s: filename revision must be the full 40-hex commit", name)
			continue
		}
		data, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			t.Fatal(err)
		}
		var report struct {
			Passed bool `json:"passed"`
			Inputs struct {
				ImplementationRevision string `json:"implementationRevision"`
			} `json:"inputs"`
			Blocked int `json:"blocked"`
			Failed  int `json:"failed"`
		}
		if err := json.Unmarshal(data, &report); err != nil {
			t.Errorf("attestations/%s: %v", name, err)
			continue
		}
		if report.Inputs.ImplementationRevision != revision {
			t.Errorf("attestations/%s: implementationRevision %s disagrees with the filename",
				name, report.Inputs.ImplementationRevision)
		}
		// The attested revision must be an ancestor of HEAD: a report
		// cannot attest a revision outside this branch's history.
		check := exec.Command("git", "merge-base", "--is-ancestor", revision, "HEAD")
		check.Dir = root
		if err := check.Run(); err != nil {
			t.Errorf("attestations/%s: revision %s is not an ancestor of HEAD", name, revision)
		}
		if report.Failed > 0 {
			t.Errorf("attestations/%s: a failed gate run must not be retained as an attestation", name)
		}
		if report.Blocked > 0 && report.Passed {
			t.Errorf("attestations/%s: passed must be false while stages are blocked", name)
		}
	}
}
