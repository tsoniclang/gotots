// Withholding: an unimplemented unit withholds its package's runnable
// module, and dependents withhold transitively — honest unimplemented,
// never partial output.
package translate

import (
	"fmt"
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
			// Only a proof that DECLARED itself no-output at creation may
			// lack a generated file; an undeclared empty reference is an
			// emitter defect, never silently normalized into a valid
			// disposition.
			if !proof.NoOutput {
				defects = append(defects, proof.ID+": no generated file and no declared no-output disposition")
				continue
			}
			proof.ModuleRetained = false
			continue
		}
		if proof.NoOutput {
			defects = append(defects, proof.ID+": declared no-output yet references generated file "+proof.GeneratedFile)
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

// symbolPresent verifies a generated declaration STRUCTURALLY within
// the deterministic generated grammar: a declaration always begins a
// line with the export keyword and one declaration form — a comment or
// string mention can never start a line that way in generated output.
// The acceptance gate additionally joins proofs to the typed-AST export
// list (the authoritative check).
func symbolPresent(content, symbol string) bool {
	for _, line := range strings.Split(content, "\n") {
		rest, ok := strings.CutPrefix(line, "export ")
		if !ok {
			continue
		}
		for _, keyword := range []string{"function ", "const ", "class ", "type ", "let "} {
			body, has := strings.CutPrefix(rest, keyword)
			if !has {
				continue
			}
			if name, matched := strings.CutPrefix(body, symbol); matched {
				if name == "" {
					return true
				}
				switch name[0] {
				case '(', '<', ' ', ':', '=', ';':
					return true
				}
			}
		}
	}
	return false
}

// growWithheldByImports withholds, by ONE level, every retained package
// whose freshly emitted co-generated imports reach an already-withheld
// package, and reports whether anything changed. It does NOT propagate
// transitively in a single call: the caller re-emits between calls so a
// package's droppable (union-member) references to a newly-withheld
// package are filtered out before the next level is computed — only
// UNAVOIDABLE dependencies cascade.
func growWithheldByImports(out *Generated, sorted []*packages.Package) bool {
	changed := false
	for _, p := range sorted {
		if _, withheld := out.Withheld[p.PkgPath]; withheld {
			continue
		}
		edges, emitted := out.ModuleImports[p.PkgPath]
		if !emitted {
			// A package that emitted no runnable module (all declarations
			// compile-time / type-only) records no import edges; its
			// dependency graph is its SOURCE imports, so it is still
			// withheld when it depends on a withheld package (and its
			// type-only proofs are cleared by finalization).
			for importPath := range p.Imports {
				edges = append(edges, importPath)
			}
		}
		sorted := append([]string(nil), edges...)
		sort.Strings(sorted)
		for _, importPath := range sorted {
			if _, withheld := out.Withheld[importPath]; withheld {
				// Publication withholding only: the materialized file is
				// RETAINED for analysis (a withheld package is still
				// independently typechecked and structurally verified). Only
				// a NotMaterialized package emits no file at all.
				out.Withheld[p.PkgPath] = "depends on withheld package " + importPath
				changed = true
				break
			}
		}
	}
	return changed
}

// growNotMaterializedByImports extends materialization blocking over the
// unit's SOURCE import graph: a package that imports a package which cannot
// produce analyzable TypeScript cannot produce it either — the import would
// dangle. Computed over source imports (not emitted edges) because
// declaration blockers are known before emission, so the emitters can skip
// non-materializable packages on their first (only) pass.
func growNotMaterializedByImports(out *Generated, sorted []*packages.Package) bool {
	changed := false
	for _, p := range sorted {
		if _, blocked := out.NotMaterialized[p.PkgPath]; blocked {
			continue
		}
		imports := make([]string, 0, len(p.Imports))
		for importPath := range p.Imports {
			imports = append(imports, importPath)
		}
		sort.Strings(imports)
		for _, importPath := range imports {
			if _, blocked := out.NotMaterialized[importPath]; blocked {
				reason := "depends on non-materialized package " + importPath
				out.NotMaterialized[p.PkgPath] = reason
				out.Withheld[p.PkgPath] = reason
				changed = true
				break
			}
		}
	}
	return changed
}

// assertRetainedImportsResolve fails closed if any retained core module
// imports a package that ended up withheld: the emitted output must be a
// forward-dependency-closed set, so a dangling import is a closure
// defect, never tolerable output.
func assertRetainedImportsResolve(out *Generated) error {
	for pkgPath, edges := range out.ModuleImports {
		if _, withheld := out.Withheld[pkgPath]; withheld {
			continue // the module was withheld; its file is gone
		}
		for _, importPath := range edges {
			if reason, withheld := out.Withheld[importPath]; withheld {
				return fmt.Errorf("retained package %s imports withheld package %s (%s): forward closure incomplete",
					pkgPath, importPath, reason)
			}
		}
	}
	return nil
}
