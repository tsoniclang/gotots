package policy

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestSpecificationLanguageIsTimeless is the repository gate: no document
// under docs/spec may narrate history or describe the design by contrast
// with abandoned alternatives.
func TestSpecificationLanguageIsTimeless(t *testing.T) {
	problems, err := CheckSpecLanguage(repositoryRoot(t))
	if err != nil {
		t.Fatal(err)
	}
	for _, problem := range problems {
		t.Error(problem.String())
	}
}

// TestCheckSpecLanguageRejectsHistoryPhrases proves the gate itself
// detects a violation.
func TestCheckSpecLanguageRejectsHistoryPhrases(t *testing.T) {
	root := t.TempDir()
	specDir := filepath.Join(root, "docs", "spec")
	if err := os.MkdirAll(specDir, 0o755); err != nil {
		t.Fatal(err)
	}
	violating := "# Doc\n\nThis was previously a dual path.\n\nClean line.\n"
	if err := os.WriteFile(filepath.Join(specDir, "bad.md"), []byte(violating), 0o644); err != nil {
		t.Fatal(err)
	}
	clean := "# Doc\n\nExactly one implementation path.\n"
	if err := os.WriteFile(filepath.Join(specDir, "good.md"), []byte(clean), 0o644); err != nil {
		t.Fatal(err)
	}

	problems, err := CheckSpecLanguage(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(problems) != 2 { // "previously" and "dual path" on the same line
		t.Fatalf("expected 2 problems, got %v", problems)
	}
	for _, problem := range problems {
		if problem.Path != "docs/spec/bad.md" || problem.Line != 3 {
			t.Errorf("wrong location: %+v", problem)
		}
		if !strings.Contains(problem.String(), "GOTOTS_SPEC_HISTORY_LANGUAGE") {
			t.Errorf("diagnostic format wrong: %s", problem.String())
		}
	}
}
