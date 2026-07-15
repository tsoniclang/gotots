package policy

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// The committed specification states the current contract timelessly. It
// never narrates history ("previously", "changed from") and never
// describes the design in terms of what it is not ("legacy", "dual
// path"): constraints are expressed positively, as in "exactly one
// implementation path". These phrases are therefore forbidden in every
// document under docs/spec.
var forbiddenSpecPhrases = []string{
	"previously",
	"previous ",
	"supersede",
	"legacy",
	"dual path",
	"dual-path",
	"compatibility path",
	"changed from",
	"no longer",
	"used to be",
	"migrated",
	"renamed",
	"old implementation",
	"new implementation",
}

// SpecLanguageProblem is one forbidden phrase found in a spec document.
type SpecLanguageProblem struct {
	Path   string
	Line   int
	Phrase string
}

func (p SpecLanguageProblem) String() string {
	return fmt.Sprintf("GOTOTS_SPEC_HISTORY_LANGUAGE:\n%s:%d uses forbidden phrase %q.\nThe specification states the current contract timelessly.",
		p.Path, p.Line, p.Phrase)
}

// CheckSpecLanguage scans every Markdown document under root/docs/spec for
// forbidden history/contrast language.
func CheckSpecLanguage(root string) ([]SpecLanguageProblem, error) {
	specDir := filepath.Join(root, "docs", "spec")
	entries, err := os.ReadDir(specDir)
	if err != nil {
		return nil, err
	}
	var problems []SpecLanguageProblem
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".md") {
			continue
		}
		data, err := os.ReadFile(filepath.Join(specDir, entry.Name()))
		if err != nil {
			return nil, err
		}
		for lineNumber, line := range strings.Split(string(data), "\n") {
			lower := strings.ToLower(line)
			for _, phrase := range forbiddenSpecPhrases {
				if strings.Contains(lower, phrase) {
					problems = append(problems, SpecLanguageProblem{
						Path:   filepath.ToSlash(filepath.Join("docs", "spec", entry.Name())),
						Line:   lineNumber + 1,
						Phrase: phrase,
					})
				}
			}
		}
	}
	sort.Slice(problems, func(i, j int) bool {
		if problems[i].Path != problems[j].Path {
			return problems[i].Path < problems[j].Path
		}
		return problems[i].Line < problems[j].Line
	})
	return problems, nil
}
