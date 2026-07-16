// Withholding: an unimplemented unit withholds its package's runnable
// module, and dependents withhold transitively — honest unimplemented,
// never partial output.
package translate

import (
	"path"
	"sort"

	"golang.org/x/tools/go/packages"
)

// withholdDependents extends withholding over the unit dependency
// graph: a runnable module cannot import a withheld one, so every
// dependent package is withheld too (its analysis records remain).
// reconcileProofRetention finalizes each proof's evidence stage after
// withholding: a proof in an emitted package is module-retained; a proof
// in a withheld package keeps its analysis identity but must not
// reference the absent generated file.
func reconcileProofRetention(out *Generated) {
	for i := range out.Proofs {
		proof := &out.Proofs[i]
		if _, withheld := out.Withheld[proof.Package]; withheld {
			proof.ModuleRetained = false
			proof.GeneratedFile = ""
			proof.GeneratedSymbol = ""
			continue
		}
		if _, exists := out.Files[proof.GeneratedFile]; proof.GeneratedFile != "" && !exists {
			// A generated-file reference must resolve; failing closed here
			// is a generator defect, surfaced as a stripped reference the
			// gate rejects rather than a phantom.
			proof.ModuleRetained = false
			proof.GeneratedFile = ""
			proof.GeneratedSymbol = ""
			continue
		}
		proof.ModuleRetained = true
	}
}

func withholdDependents(out *Generated, sorted []*packages.Package) {
	imports := map[string][]string{}
	for _, p := range sorted {
		// The emitted module's real import edges — symbol references
		// (including interface-dispatch branch targets) and init edges —
		// are the withholding dependency graph. A package with no emitted
		// module falls back to its source imports.
		if edges, ok := out.ModuleImports[p.PkgPath]; ok {
			imports[p.PkgPath] = append([]string(nil), edges...)
		} else {
			for importPath := range p.Imports {
				imports[p.PkgPath] = append(imports[p.PkgPath], importPath)
			}
		}
		sort.Strings(imports[p.PkgPath])
	}
	for changed := true; changed; {
		changed = false
		for _, p := range sorted {
			if _, withheld := out.Withheld[p.PkgPath]; withheld {
				continue
			}
			for _, importPath := range imports[p.PkgPath] {
				if _, withheld := out.Withheld[importPath]; withheld {
					out.Withheld[p.PkgPath] = "depends on withheld package " + importPath
					corePath := path.Join("core", p.PkgPath, "package.ts")
					delete(out.Files, corePath)
					delete(out.Ownership, corePath)
					changed = true
					break
				}
			}
		}
	}
}
