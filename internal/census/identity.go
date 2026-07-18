package census

import (
	"fmt"
	"sort"
	"strings"
)

// validateIdentities proves that every purported record identity is unique.
// A collision means the identity scheme is wrong and the census cannot be
// trusted as a completeness boundary.
func validateIdentities(report *Report) error {
	var duplicates []string
	check := func(kind string, keys []string) {
		seen := map[string]bool{}
		for _, key := range keys {
			if seen[key] {
				duplicates = append(duplicates, kind+": "+key)
			}
			seen[key] = true
		}
	}

	declarationIDs := make([]string, len(report.Declarations))
	for i, d := range report.Declarations {
		declarationIDs[i] = d.ID
	}
	check("declaration", declarationIDs)

	filePaths := make([]string, len(report.Files))
	for i, f := range report.Files {
		filePaths[i] = f.Path
	}
	check("file", filePaths)

	directiveKeys := make([]string, len(report.Directives))
	for i, d := range report.Directives {
		directiveKeys[i] = fmt.Sprintf("%s:%d:%d %s", d.File, d.Line, d.Col, d.Directive)
	}
	check("directive", directiveKeys)

	rareKeys := make([]string, len(report.RareConstructs))
	for i, r := range report.RareConstructs {
		rareKeys[i] = fmt.Sprintf("%s:%d:%d %s", r.File, r.Line, r.Col, r.Construct)
	}
	check("rare-construct", rareKeys)

	if len(duplicates) > 0 {
		sort.Strings(duplicates)
		return fmt.Errorf("identity validation fails closed on %d duplicates:\n%s",
			len(duplicates), strings.Join(duplicates, "\n"))
	}
	return nil
}
