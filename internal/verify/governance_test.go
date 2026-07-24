package verify

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type requiredGovernanceText struct {
	path  string
	parts []string
}

func governanceDefects(
	files map[string][]byte,
	requirements []requiredGovernanceText,
) []string {
	var defects []string
	if !bytes.Equal(files["AGENTS.md"], files["CLAUDE.md"]) {
		defects = append(defects, "AGENTS.md and CLAUDE.md differ")
	}
	for _, requirement := range requirements {
		text, ok := files[requirement.path]
		if !ok {
			defects = append(
				defects,
				fmt.Sprintf("%s is absent", requirement.path),
			)
			continue
		}
		normalized := strings.Join(strings.Fields(string(text)), " ")
		for _, part := range requirement.parts {
			if !strings.Contains(normalized, part) {
				defects = append(
					defects,
					fmt.Sprintf(
						"%s lacks required authority %q",
						requirement.path,
						part,
					),
				)
			}
		}
	}
	return defects
}

func TestGovernanceAndStepVerificationAreAuthoritative(t *testing.T) {
	requirements := []requiredGovernanceText{
		{
			path: "AGENTS.md",
			parts: []string{
				"Every dependency-ordered implementation step has its own comprehensive closure gate.",
				"Repeated semantic references use one typed package-local identity dictionary.",
			},
		},
		{
			path: "docs/spec/architecture.md",
			parts: []string{
				"The sole immutable in-memory representation of that logical package is normalized.",
			},
		},
		{
			path: "docs/spec/delivery.md",
			parts: []string{
				"Every numbered dependency step closes independently before work begins on its consumer.",
			},
		},
		{
			path: "docs/spec/verification.md",
			parts: []string{
				"## Per-Step Comprehensive Verification",
			},
		},
	}
	root := repoRoot(t)
	files := make(map[string][]byte, len(requirements)+1)
	for _, path := range []string{
		"AGENTS.md",
		"CLAUDE.md",
		"docs/spec/architecture.md",
		"docs/spec/delivery.md",
		"docs/spec/verification.md",
	} {
		content, err := os.ReadFile(filepath.Join(root, path))
		if err != nil {
			t.Fatalf("read %s: %v", path, err)
		}
		files[path] = content
	}
	if defects := governanceDefects(files, requirements); len(defects) != 0 {
		t.Fatalf("governance defects:\n%s", strings.Join(defects, "\n"))
	}

	mutated := make(map[string][]byte, len(files))
	for path, content := range files {
		mutated[path] = append([]byte(nil), content...)
	}
	mutated["docs/spec/verification.md"] = bytes.Replace(
		mutated["docs/spec/verification.md"],
		[]byte("## Per-Step Comprehensive Verification"),
		[]byte("## Deferred Verification"),
		1,
	)
	if defects := governanceDefects(
		mutated,
		requirements,
	); len(defects) == 0 {
		t.Fatal("governance gate accepted removal of per-step verification")
	}
}
