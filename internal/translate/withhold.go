// Withholding: an unimplemented unit withholds its package's runnable
// module, and dependents withhold transitively — honest unimplemented,
// never partial output.
package translate

import (
	"fmt"
	"path"
	"sort"
	"strings"

	"golang.org/x/tools/go/packages"
)

// withholdDependents extends withholding over the unit dependency
// graph: a runnable module cannot import a withheld one, so every
// dependent package is withheld too (its analysis records remain).
// finalizeEvidenceStages is the single, final evidence pass, run after
// EVERY generated file and proof exists (core, ABI, external stubs).
// The invariant is bidirectional: moduleRetained ⇔ the package is
// retained ∧ the generated file exists ∧ the generated symbol appears in
// it. Invalid evidence is a returned defect — never normalized away.
// No-output declarations carry their explicit disposition and are never
// retained bodies.
func finalizeEvidenceStages(out *Generated) error {
	var defects []string
	for i := range out.Proofs {
		proof := &out.Proofs[i]
		if proof.GeneratedFile == "" {
			// A declaration whose exact lowering emits nothing: explicit
			// no-output disposition, never a retained body.
			proof.ModuleRetained = false
			proof.NoOutput = true
			continue
		}
		if _, withheld := out.Withheld[proof.Package]; withheld {
			// The package's runnable module is withheld: the proof keeps
			// its analysis identity, the reference is cleared because the
			// file genuinely does not exist in the bundle, and the stage
			// records the blockage.
			proof.ModuleRetained = false
			proof.GeneratedFile = ""
			proof.GeneratedSymbol = ""
			continue
		}
		content, exists := out.Files[proof.GeneratedFile]
		if !exists {
			defects = append(defects, proof.ID+": references absent generated file "+proof.GeneratedFile)
			continue
		}
		if proof.GeneratedSymbol != "" && !symbolPresent(content, proof.GeneratedSymbol) {
			defects = append(defects, proof.ID+": generated symbol "+proof.GeneratedSymbol+" absent from "+proof.GeneratedFile)
			continue
		}
		proof.ModuleRetained = true
	}
	if len(defects) > 0 {
		sort.Strings(defects)
		if len(defects) > 10 {
			defects = append(defects[:10], "...")
		}
		return fmt.Errorf("GOTOTS_OUTPUT_EVIDENCE_INVALID: %s", strings.Join(defects, "; "))
	}
	return nil
}

// symbolPresent reports whether the generated module declares the
// symbol (function, const, class, or type declaration spelling).
func symbolPresent(content, symbol string) bool {
	for _, form := range []string{
		"function " + symbol + "(",
		"function " + symbol + "<",
		"const " + symbol + ":",
		"const " + symbol + " =",
		"class " + symbol + " ",
		"class " + symbol + "<",
		"type " + symbol + " ",
		"let " + symbol + ":",
	} {
		if strings.Contains(content, form) {
			return true
		}
	}
	return false
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
