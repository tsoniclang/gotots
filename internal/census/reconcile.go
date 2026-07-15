package census

import (
	"fmt"
	"sort"
	"strings"

	"github.com/tsoniclang/gotots/internal/inventory"
	"github.com/tsoniclang/gotots/internal/profile"
)

// reconcileFileSets proves exact selected file-set equivalence between the
// pass-1 inventory and the pass-2 typed load for every owned and test-only
// package: production files, in-package tests, and black-box tests each
// match exactly.
func reconcileFileSets(inv *inventory.Inventory, goFiles, testFiles, xtestFiles map[string][]string) error {
	var problems []string
	compare := func(pkg, kind string, want, got []string) {
		sort.Strings(got)
		if len(want) != len(got) {
			problems = append(problems, fmt.Sprintf("%s: %s: inventory has %d files, typed pass analyzed %d", pkg, kind, len(want), len(got)))
			return
		}
		for i := range want {
			if want[i] != got[i] {
				problems = append(problems, fmt.Sprintf("%s: %s: inventory %s vs typed pass %s", pkg, kind, want[i], got[i]))
				return
			}
		}
	}
	for _, pkg := range inv.Module {
		class := profile.PackageClass(pkg.Class)
		if class != profile.ClassOwned && class != profile.ClassTestOnly {
			continue
		}
		if len(pkg.CgoFiles) > 0 {
			problems = append(problems, fmt.Sprintf("%s: cgo files selected but cgo analysis is unsupported", pkg.ImportPath))
		}
		compare(pkg.ImportPath, "go files", pkg.GoFiles, goFiles[pkg.ImportPath])
		compare(pkg.ImportPath, "in-package test files", pkg.TestGoFiles, testFiles[pkg.ImportPath])
		compare(pkg.ImportPath, "black-box test files", pkg.XTestGoFiles, xtestFiles[pkg.ImportPath])
	}
	if len(problems) > 0 {
		sort.Strings(problems)
		return fmt.Errorf("pass-1/pass-2 file reconciliation fails closed on %d mismatches:\n%s",
			len(problems), strings.Join(problems, "\n"))
	}
	return nil
}
