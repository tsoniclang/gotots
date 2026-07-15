package policy

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestCanonicalSpecificationPresent proves that the canonical committed
// specification exists and still carries its machine-checkable normative
// markers. This gate prevents the specification from being silently
// deleted, renamed, or hollowed out; the prose itself is reviewed by
// humans.
func TestCanonicalSpecificationPresent(t *testing.T) {
	root := repositoryRoot(t)

	requirements := map[string][]string{
		"docs/spec/mission-and-scope.md": {
			// Mission and closed-world/detection framing.
			"Mechanically translate the complete declared TS-Go source corpus",
			"closed-world completeness over the current TS-Go corpus",
			// Fail-closed detection of future idioms.
			"GOTOTS_UNSUPPORTED_STATEMENT",
			"Block generation on every unrecognized construct",
			// Exactly four dispositions.
			"exactly one disposition",
			"automatically translated;",
			"declared manual-body implementation;",
			"external declaration/stub obligation;",
			"explicitly excluded by the canonical scope.",
			// Genericity and library boundary.
			"never determine language semantics",
			"implements no library",
			// Representation selection is single-model, evidence-based.
			"static selection from one semantic model",
		},
		"docs/spec/file-size-and-decomposition.md": {
			"600 physical lines",
			"GOTOTS_FILE_TOO_LARGE",
			"There is no allowlist",
		},
		"docs/spec/translation-and-output.md": {
			"Generated compiler core.",
			"strict ESM with explicit `.js` relative imports",
			"Worked Translation Shapes",
			"sourceSelectedDeclaration",
			"Two clean generations from identical inputs",
		},
		"docs/spec/performance-and-representation.md": {
			"at least three warmup runs and seven measured runs",
			"Default Regression Gates",
			"TSTS-Specific Hot Paths",
			"Native-Target Shape",
		},
		"docs/spec/testing-and-acceptance.md": {
			"Gate Ordering",
			"No-Extension TS-Go Differential",
			"Every in-scope TS-Go `_test.go`",
			"Compiler Corpus",
			"Intermediate census, declaration, vertical-slice, or focused-test milestones",
		},
		"README.md": {
			"docs/spec/",
			"translation-and-output.md",
			"testing-and-acceptance.md",
		},
		"CONTRIBUTING.md": {
			"mission-and-scope.md",
			"exactly one disposition",
			"performance-and-representation.md",
		},
	}

	for relative, markers := range requirements {
		data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(relative)))
		if err != nil {
			t.Errorf("canonical specification file missing: %s (%v)", relative, err)
			continue
		}
		content := string(data)
		for _, marker := range markers {
			if !strings.Contains(content, marker) {
				t.Errorf("%s no longer contains required normative marker %q", relative, marker)
			}
		}
	}
}
