package policy

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"
	"unicode"
)

type decisionRegistry struct {
	SchemaVersion int                     `json:"schemaVersion"`
	Decisions     []decisionRegistryEntry `json:"decisions"`
}

type decisionRegistryEntry struct {
	ID                  string   `json:"id"`
	Path                string   `json:"path"`
	Status              string   `json:"status"`
	Owner               string   `json:"owner"`
	AffectedSpecClauses []string `json:"affectedSpecClauses"`
	SchemaOrABIVersions []string `json:"schemaOrAbiVersions"`
	// ImplementationStatus distinguishes the revision that accepted a
	// decision from the revision that implements it. An accepted but
	// unimplemented decision is "planned": it carries the resolvable
	// DecisionRevision its evidence binds to and NOTHING claiming an
	// implementation — no implementation revision, no proof artifacts.
	// An "implemented" decision carries the resolvable implementing
	// revision and its proof artifacts.
	ImplementationStatus   string   `json:"implementationStatus"`
	DecisionRevision       string   `json:"decisionRevision,omitempty"`
	ImplementationRevision string   `json:"implementationRevision,omitempty"`
	ProofArtifacts         []string `json:"proofArtifacts,omitempty"`
	ReopeningConditions    []string `json:"reopeningConditions"`
}

func revisionResolves(t *testing.T, root, decisionID, label, revision string) {
	t.Helper()
	if !regexp.MustCompile(`^[0-9a-f]{7,40}$`).MatchString(revision) {
		t.Errorf("%s has invalid %s revision %q", decisionID, label, revision)
		return
	}
	command := exec.Command("git", "cat-file", "-e", revision+"^{commit}")
	command.Dir = root
	if output, err := command.CombinedOutput(); err != nil {
		t.Errorf("%s %s revision %s does not resolve: %v (%s)",
			decisionID, label, revision, err, strings.TrimSpace(string(output)))
	}
}

func TestDecisionRegistryIsCompleteAndResolvable(t *testing.T) {
	root := repositoryRoot(t)
	registryPath := filepath.Join(root, "docs", "decisions", "registry.json")
	data, err := os.ReadFile(registryPath)
	if err != nil {
		t.Fatal(err)
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	var registry decisionRegistry
	if err := decoder.Decode(&registry); err != nil {
		t.Fatalf("parse decision registry: %v", err)
	}
	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		t.Fatal("decision registry has trailing JSON")
	}
	if registry.SchemaVersion != 1 {
		t.Fatalf("decision registry schemaVersion = %d; want 1", registry.SchemaVersion)
	}
	if len(registry.Decisions) == 0 {
		t.Fatal("decision registry has no decisions")
	}

	seen := map[string]bool{}
	for index, decision := range registry.Decisions {
		wantID := fmt.Sprintf("ADR-%04d", index+1)
		if decision.ID != wantID {
			t.Errorf("decision %d ID = %q; want %q", index, decision.ID, wantID)
		}
		if seen[decision.ID] {
			t.Errorf("duplicate decision ID %s", decision.ID)
		}
		seen[decision.ID] = true
		if decision.Status != "accepted" {
			t.Errorf("%s status = %q; only accepted decisions may be registered", decision.ID, decision.Status)
		}
		if decision.Owner == "" || len(decision.AffectedSpecClauses) == 0 ||
			len(decision.SchemaOrABIVersions) == 0 || len(decision.ReopeningConditions) == 0 {
			t.Errorf("%s has an incomplete registry row", decision.ID)
		}
		var revisionMarker string
		switch decision.ImplementationStatus {
		case "implemented":
			revisionResolves(t, root, decision.ID, "implementation", decision.ImplementationRevision)
			if len(decision.ProofArtifacts) == 0 {
				t.Errorf("%s is implemented but carries no proof artifacts", decision.ID)
			}
			revisionMarker = "Implementation revision: `" + decision.ImplementationRevision + "`"
		case "planned":
			revisionResolves(t, root, decision.ID, "decision", decision.DecisionRevision)
			if decision.ImplementationRevision != "" {
				t.Errorf("%s is planned but claims implementation revision %q", decision.ID, decision.ImplementationRevision)
			}
			if len(decision.ProofArtifacts) != 0 {
				t.Errorf("%s is planned but claims proof artifacts %v", decision.ID, decision.ProofArtifacts)
			}
			revisionMarker = "Decision revision: `" + decision.DecisionRevision + "`"
		default:
			t.Errorf("%s implementationStatus = %q; want planned or implemented", decision.ID, decision.ImplementationStatus)
		}

		adrData, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(decision.Path)))
		if err != nil {
			t.Errorf("%s ADR path: %v", decision.ID, err)
			continue
		}
		adr := string(adrData)
		numericID := strings.TrimPrefix(decision.ID, "ADR-")
		markers := []string{
			"# ADR " + numericID + ":",
			"Status: " + decision.Status,
			"Owner: " + decision.Owner,
			"Registry entry: `docs/decisions/registry.json#" + decision.ID + "`",
		}
		if revisionMarker != "" {
			markers = append(markers, revisionMarker)
		}
		if decision.ImplementationStatus == "planned" {
			markers = append(markers, "Implementation status: planned")
			if strings.Contains(adr, "Implementation revision:") {
				t.Errorf("%s is planned but its ADR claims an implementation revision", decision.ID)
			}
		}
		for _, marker := range markers {
			if !strings.Contains(adr, marker) {
				t.Errorf("%s ADR is missing %q", decision.ID, marker)
			}
		}
		for _, heading := range []string{
			"## Context",
			"## Alternatives",
			"## Decision",
			"## Effects",
			"## Migration",
			"## Evidence",
			"## Reconsider when",
		} {
			if !strings.Contains(adr, heading) {
				t.Errorf("%s ADR is missing required section %q", decision.ID, heading)
			}
		}
		for _, clause := range decision.AffectedSpecClauses {
			checkMarkdownClause(t, root, decision.ID, clause)
		}
		// Proof artifacts must exist AT the implementing revision, not
		// merely in the current working tree.
		for _, artifact := range decision.ProofArtifacts {
			command := exec.Command("git", "cat-file", "-e", decision.ImplementationRevision+":"+artifact)
			command.Dir = root
			if output, err := command.CombinedOutput(); err != nil {
				t.Errorf("%s proof artifact %s absent at revision %s: %v (%s)",
					decision.ID, artifact, decision.ImplementationRevision, err, strings.TrimSpace(string(output)))
			}
		}
	}

	entries, err := filepath.Glob(filepath.Join(root, "docs", "decisions", "[0-9][0-9][0-9][0-9]-*.md"))
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != len(registry.Decisions) {
		t.Fatalf("registry has %d decisions but docs/decisions has %d ADR files", len(registry.Decisions), len(entries))
	}
}

func checkMarkdownClause(t *testing.T, root, decisionID, clause string) {
	t.Helper()
	parts := strings.Split(clause, "#")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		t.Errorf("%s has malformed affected clause %q", decisionID, clause)
		return
	}
	data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(parts[0])))
	if err != nil {
		t.Errorf("%s affected clause %s: %v", decisionID, clause, err)
		return
	}
	var anchors []string
	for _, line := range strings.Split(string(data), "\n") {
		if strings.HasPrefix(line, "#") {
			anchors = append(anchors, markdownAnchor(strings.TrimSpace(strings.TrimLeft(line, "#"))))
		}
	}
	sort.Strings(anchors)
	index := sort.SearchStrings(anchors, parts[1])
	if index == len(anchors) || anchors[index] != parts[1] {
		t.Errorf("%s affected clause %s has no matching heading", decisionID, clause)
	}
}

func markdownAnchor(heading string) string {
	var builder strings.Builder
	lastDash := false
	for _, r := range strings.ToLower(heading) {
		switch {
		case unicode.IsLetter(r) || unicode.IsDigit(r):
			builder.WriteRune(r)
			lastDash = false
		case unicode.IsSpace(r) || r == '-':
			if builder.Len() > 0 && !lastDash {
				builder.WriteByte('-')
				lastDash = true
			}
		}
	}
	return strings.TrimSuffix(builder.String(), "-")
}
