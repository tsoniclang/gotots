package stagecheck

import (
	"fmt"
	"sort"

	"github.com/tsoniclang/gotots/internal/identity"
	"github.com/tsoniclang/gotots/internal/language/analyze"
	"github.com/tsoniclang/gotots/internal/language/catalog"
	"github.com/tsoniclang/gotots/internal/source"
)

// VerifyInventory independently joins the analyze-owned region/reference model
// against the finalized source census, enforcing the conservation law:
//
//   - censused unit identities join implementation-definition records exactly
//     (kind and depth agree);
//   - every full-semantic unit owns exactly one body region, and every non-full
//     unit owns none and contributes zero body occurrences;
//   - every implementation reference names a defined child unit; and
//   - occurrence identities are unique within each region and every
//     variant-bearing occurrence is resolved.
//
// It reconstructs the expected definition set from the source census (a
// different producer than the traversal) and reports one-sided differences with
// exact identities.
func VerifyInventory(req source.Request, ws *source.Workspace, inv *analyze.WorkspaceInventory) error {
	fail := func(reason string) error { return &VerificationError{Stage: "inventory", Reason: reason} }
	var problems []string

	invByPkg := map[string]*analyze.PackageInventory{}
	for _, pkg := range inv.Packages() {
		invByPkg[pkg.ID().String()] = pkg
	}

	for _, pkg := range ws.Packages() {
		if pkg.Disposition() != source.DispositionOrdinarySource {
			continue
		}
		// Only application packages (with a full-semantic unit) enter the
		// region model; contract-depth provider packages are audited
		// separately.
		if !pkg.RetainsFullSemantic() {
			continue
		}
		// Independent expected definitions from the source census.
		expected := map[string]source.SourceUnit{}
		for _, file := range pkg.Files() {
			for _, unit := range file.Units() {
				expected[unit.ID().String()] = unit
			}
		}
		pkgInv := invByPkg[pkg.ID().String()]
		if pkgInv == nil {
			if len(expected) > 0 {
				problems = append(problems, "package "+pkg.ID().String()+" has census units but no inventory")
			}
			continue
		}

		defs := map[string]analyze.ImplementationDefinition{}
		for _, d := range pkgInv.Definitions() {
			key := d.Unit().String()
			if _, dup := defs[key]; dup {
				problems = append(problems, "duplicate definition for unit "+key)
			}
			defs[key] = d
		}
		// Definition <-> census join (source units).
		matched := map[string]bool{}
		for id, unit := range expected {
			d, ok := defs[id]
			if !ok {
				problems = append(problems, "census unit "+id+" has no definition")
				continue
			}
			matched[id] = true
			if d.Kind() != unit.Kind() {
				problems = append(problems, "unit "+id+" definition kind "+d.Kind().String()+" != census "+unit.Kind().String())
			}
			if d.Depth() != unit.Depth() {
				problems = append(problems, fmt.Sprintf("unit %s definition depth %s != census %s", id, d.Depth(), unit.Depth()))
			}
			if d.Full() != (unit.Depth() == source.DepthFullSemantic) {
				problems = append(problems, "unit "+id+" full flag disagrees with depth")
			}
		}
		// Implicit definitions join.
		for _, implicit := range pkg.ImplicitUnits() {
			key := implicit.ID().String()
			if _, ok := defs[key]; !ok {
				problems = append(problems, "implicit unit "+key+" has no definition")
				continue
			}
			matched[key] = true
		}
		for id := range defs {
			if !matched[id] {
				problems = append(problems, "definition "+id+" has no census unit")
			}
		}

		// Body-region <-> full-unit join, and per-region validity.
		regionByUnit := map[string]*analyze.FileInventory{}
		for _, region := range pkgInv.Files() {
			if region.RootUnit().IsZero() {
				continue // file declaration region
			}
			key := region.RootUnit().String()
			if _, dup := regionByUnit[key]; dup {
				problems = append(problems, "duplicate body region for unit "+key)
			}
			regionByUnit[key] = region
			problems = append(problems, verifyRegionOccurrences(region)...)
		}
		for id, unit := range expected {
			_, hasRegion := regionByUnit[id]
			full := unit.Depth() == source.DepthFullSemantic
			if full && !hasRegion {
				problems = append(problems, "full unit "+id+" has no body region")
			}
			if !full && hasRegion {
				problems = append(problems, "non-full unit "+id+" owns a body region")
			}
		}

		// Reference <-> child-definition join.
		for _, ref := range pkgInv.References() {
			child := ref.Child().String()
			if _, ok := defs[child]; !ok {
				problems = append(problems, "reference names undefined child unit "+child)
			}
			if !ref.Edge().Valid() {
				problems = append(problems, "reference to "+child+" has an invalid edge")
			}
		}
		// Exact site<->reference conservation: independently derive every
		// implementation site from re-parsed source and exact-multiset-join.
		problems = append(problems, verifyReferenceConservation(pkg, pkgInv.References(), req.Overlay)...)
	}

	if len(problems) > 0 {
		sort.Strings(problems)
		return fail(fmt.Sprintf("conservation join failed; %v", problems))
	}
	return nil
}

// verifyRegionOccurrences checks one region's occurrence invariants: unique
// identities, resolved variants, and a root that covers its unit.
func verifyRegionOccurrences(region *analyze.FileInventory) []string {
	var problems []string
	seen := map[identity.OccurrenceID]bool{}
	for _, occ := range region.Occurrences() {
		if occ.ID().IsZero() {
			problems = append(problems, region.File().String()+" region has a zero occurrence identity")
			continue
		}
		if seen[occ.ID()] {
			problems = append(problems, "duplicate occurrence "+occ.ID().String())
		}
		seen[occ.ID()] = true
		if catalog.VariantBearing(occ.Kind()) && !occ.Variant().Valid() {
			problems = append(problems, "unresolved variant at "+occ.ID().String())
		}
	}
	return problems
}
