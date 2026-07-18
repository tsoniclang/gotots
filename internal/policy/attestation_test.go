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

// TestAttestationCoversCurrentProduct: the newest attestation must
// cover the current product revision — every commit AFTER the attested
// revision must be evidence-only (touch nothing outside attestations/),
// so a later source commit without a fresh report fails.
func TestAttestationCoversCurrentProduct(t *testing.T) {
	if os.Getenv("GOTOTS_ATTESTING") != "" {
		// The gate run that PRODUCES the next attestation is the one
		// context that cannot yet require it.
		t.Skip("evidence-production run")
	}
	root := repositoryRoot(t)
	dir := filepath.Join(root, "attestations")
	entries, err := os.ReadDir(dir)
	if os.IsNotExist(err) {
		return
	}
	if err != nil {
		t.Fatal(err)
	}
	newest := ""
	for _, entry := range entries {
		rest, ok := strings.CutPrefix(entry.Name(), "gate-report-")
		if !ok {
			continue
		}
		revision := strings.TrimSuffix(rest, ".json")
		if len(revision) != 40 {
			continue
		}
		// newest by commit order: the one whose revision is a descendant
		// of every other attested revision.
		if newest == "" {
			newest = revision
			continue
		}
		check := exec.Command("git", "merge-base", "--is-ancestor", newest, revision)
		check.Dir = root
		if check.Run() == nil {
			newest = revision
		}
	}
	if newest == "" {
		return
	}
	listing, err := gitOutput(root, "log", "--format=%H", newest+"..HEAD")
	if err != nil {
		t.Fatal(err)
	}
	for _, commit := range strings.Split(strings.TrimSpace(listing), "\n") {
		if commit == "" {
			continue
		}
		files, err := gitOutput(root, "diff-tree", "--no-commit-id", "--name-only", "-r", commit)
		if err != nil {
			t.Fatal(err)
		}
		for _, changed := range strings.Split(strings.TrimSpace(files), "\n") {
			if changed != "" && !strings.HasPrefix(changed, "attestations/") {
				t.Errorf("commit %s changes product source %s after the newest attestation (%s); re-run the gate and retain a fresh report", commit[:12], changed, newest[:12])
			}
		}
	}
}

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
		// The attestation's own commit must be an evidence-only commit
		// whose PARENT is exactly the attested revision, and no product
		// source may change in it — the machine-enforced protocol.
		commitOut, err := gitOutput(root, "log", "--format=%H %P", "-1", "--", "attestations/"+name)
		if err != nil {
			t.Fatalf("attestations/%s: %v", name, err)
		}
		fields := strings.Fields(commitOut)
		if len(fields) >= 2 {
			attestCommit, parent := fields[0], fields[1]
			if parent != revision {
				t.Errorf("attestations/%s: evidence commit %s has parent %s, not the attested revision", name, attestCommit[:12], parent[:12])
			}
			filesOut, err := gitOutput(root, "diff-tree", "--no-commit-id", "--name-only", "-r", attestCommit)
			if err != nil {
				t.Fatal(err)
			}
			for _, changed := range strings.Split(strings.TrimSpace(filesOut), "\n") {
				if changed != "" && !strings.HasPrefix(changed, "attestations/") {
					t.Errorf("attestations/%s: evidence commit changes product source %s", name, changed)
				}
			}
		}
		// A retained attestation records the TRUE gate state — a failing
		// or blocked run is an honest record — but must never CLAIM
		// success it did not achieve.
		if report.Passed && (report.Failed > 0 || report.Blocked > 0) {
			t.Errorf("attestations/%s: passed must be false while stages fail or block", name)
		}
	}
}

// gitOutput runs one git query in the repository.
func gitOutput(root string, args ...string) (string, error) {
	command := exec.Command("git", args...)
	command.Dir = root
	out, err := command.Output()
	return strings.TrimSpace(string(out)), err
}
