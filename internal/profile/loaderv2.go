// Derived schema-2 source-universe accessors: closure seeds and the
// total-coverage check. Classification itself always routes through
// Classify.
package profile

import (
	"fmt"
	"sort"
)

// rootsWithDisposition lists the selector roots of every rule with the
// given disposition, sorted (closure seeds and load patterns — NOT a
// classification shortcut; classification always goes through Classify).
// loadPatternsWithDisposition preserves each selector's KIND in the
// go/packages load pattern: an exact selector loads exactly its
// package (./root), a subtree selector loads the subtree (./root/...).
// An exact selector must never silently widen to a subtree.
func (u *SourceUniverse) loadPatternsWithDisposition(d PackageDisposition) []string {
	var out []string
	for _, rule := range u.PackageRules {
		if rule.Disposition != d {
			continue
		}
		for _, s := range rule.Selectors {
			if s.Kind == "exact" {
				out = append(out, "./"+s.Package)
			} else {
				out = append(out, "./"+s.Root+"/...")
			}
		}
	}
	sort.Strings(out)
	return out
}

// SelectedLoadPatterns seeds the typed source closure.
func (u *SourceUniverse) SelectedLoadPatterns() []string {
	return u.loadPatternsWithDisposition(DispositionSelected)
}

// TestOnlyLoadPatterns seeds the test-scope load patterns.
func (u *SourceUniverse) TestOnlyLoadPatterns() []string {
	return u.loadPatternsWithDisposition(DispositionTestOnly)
}

// CheckCoverage classifies every given in-checkout package and returns
// one error listing every package without exactly one winning rule —
// the contract has no implicit disposition.
func (u *SourceUniverse) CheckCoverage(packages []string) error {
	var problems []string
	for _, pkg := range packages {
		if _, err := u.Classify(pkg); err != nil {
			problems = append(problems, err.Error())
		}
	}
	if len(problems) > 0 {
		sort.Strings(problems)
		return fmt.Errorf("source-universe coverage defects (%d):\n%s", len(problems), joinLines(problems))
	}
	return nil
}

func joinLines(lines []string) string {
	out := ""
	for i, line := range lines {
		if i > 0 {
			out += "\n"
		}
		out += line
	}
	return out
}
