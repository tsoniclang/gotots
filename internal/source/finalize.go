package source

import (
	"sort"

	"github.com/tsoniclang/gotots/internal/identity"
)

// RetentionProjection is the opaque, constructor-validated attestation the
// analyze traversal hands to finalization. It carries only canonical
// membership — which units the traversal retained as full-semantic body
// regions — never kinds, edges, roles, tokens, or reference topology. Source
// consumes it as a lifecycle seal and conservation check, not as a second
// analyzer.
type RetentionProjection struct {
	fullUnits    map[identity.SourceUnitID]bool
	fullImplicit map[identity.ImplicitUnitID]bool
}

// NewRetentionProjection validates the traversal's retained-unit attestation.
func NewRetentionProjection(fullUnits []identity.SourceUnitID, fullImplicit []identity.ImplicitUnitID) (RetentionProjection, error) {
	p := RetentionProjection{
		fullUnits:    make(map[identity.SourceUnitID]bool, len(fullUnits)),
		fullImplicit: make(map[identity.ImplicitUnitID]bool, len(fullImplicit)),
	}
	for _, id := range fullUnits {
		if id.IsZero() {
			return RetentionProjection{}, &LoadError{Reason: "retention projection names a zero unit"}
		}
		p.fullUnits[id] = true
	}
	for _, id := range fullImplicit {
		if id.IsZero() {
			return RetentionProjection{}, &LoadError{Reason: "retention projection names a zero implicit unit"}
		}
		p.fullImplicit[id] = true
	}
	return p, nil
}

// Finalize consumes the transient universe, the scope phase's per-unit
// evidence-depth selection, and the analyze traversal's retention projection,
// and produces the finalized Workspace: identity, provenance, acquisition,
// versions, and the per-unit depth partition. It severs the transient
// syntax/checker lifetime — the finalized artifact exposes no raw AST.
//
// The projection is the seal: every full-semantic unit in the selection must
// be attested by the traversal and vice versa, so the depth partition and the
// retained body regions are conserved exactly.
func Finalize(u *Universe, depths map[identity.SourceUnitID]EvidenceDepth, implicitDepths map[identity.ImplicitUnitID]EvidenceDepth, projection RetentionProjection) (*Workspace, error) {
	failScope := func(reason string) (*Workspace, error) {
		return nil, &LoadError{Dir: u.request.Dir, Reason: "scope selection invalid: " + reason}
	}
	remaining := make(map[identity.SourceUnitID]bool, len(depths))
	for id, depth := range depths {
		if !depth.Valid() {
			return failScope(id.String() + " has invalid depth")
		}
		remaining[id] = true
		// Conservation: the traversal's full attestation must equal the
		// selection's full set.
		if (depth == DepthFullSemantic) != projection.fullUnits[id] {
			return failScope("retention projection disagrees with selection on unit " + id.String())
		}
	}
	for id := range projection.fullUnits {
		if !remaining[id] {
			return failScope("retention projection names unselected unit " + id.String())
		}
	}
	remainingImplicit := make(map[identity.ImplicitUnitID]bool, len(implicitDepths))
	for id, depth := range implicitDepths {
		if !depth.Valid() {
			return failScope(id.String() + " has invalid depth")
		}
		remainingImplicit[id] = true
	}
	ws := &Workspace{fset: u.fset, toolchain: u.toolchain}
	for _, loaded := range u.packages {
		record := finalizePackage(loaded, depths)
		for _, implicit := range loaded.implicitUnits {
			depth, selected := implicitDepths[implicit]
			if !selected {
				return failScope("selection missing implicit unit " + implicit.String())
			}
			delete(remainingImplicit, implicit)
			record.implicitUnits = append(record.implicitUnits, ImplicitUnit{id: implicit, depth: depth})
		}
		for _, file := range record.files {
			for i := range file.units {
				depth, selected := depths[file.units[i].id]
				if !selected {
					return nil, &LoadError{Dir: u.request.Dir, Reason: "scope selection missing unit " + file.units[i].id.String()}
				}
				delete(remaining, file.units[i].id)
				file.units[i].depth = depth
			}
		}
		if err := ws.admit(record); err != nil {
			return nil, err
		}
	}
	for id := range remaining {
		return failScope("selection names unknown unit " + id.String())
	}
	for id := range remainingImplicit {
		return failScope("selection names unknown implicit unit " + id.String())
	}
	sort.Slice(ws.packages, func(i, j int) bool { return ws.packages[i].id.String() < ws.packages[j].id.String() })
	sort.Slice(ws.roots, func(i, j int) bool { return ws.roots[i].id.String() < ws.roots[j].id.String() })
	return ws, nil
}

// finalizePackage builds one finalized package record: identity, provenance,
// acquisition, versions, unit boundaries, and cgo mappings/synthetics. No raw
// AST or body-indexed syntax is retained; the shared declaration type graph is
// carried for later phases.
func finalizePackage(loaded *LoadedPackage, depths map[identity.SourceUnitID]EvidenceDepth) *Package {
	out := &Package{
		id: loaded.id, provenance: loaded.provenance, acquisition: loaded.acquisition,
		disposition: loaded.disposition, moduleGoVersion: loaded.moduleGoVersion,
		requestedRoot: loaded.requestedRoot,
		imports:       append([]string(nil), loaded.imports...),
		otherFiles:    append([]string(nil), loaded.otherFiles...),
		embedFiles:    append([]string(nil), loaded.embedFiles...),
		embedPatterns: append([]string(nil), loaded.embedPatterns...),
		types:         loaded.types,
		mappings:      append([]CheckedUnitMapping(nil), loaded.mappings...),
		synthetics:    append([]SyntheticUnit(nil), loaded.synthetics...),
	}
	for _, loadedFile := range loaded.files {
		file := &File{
			path: loadedFile.path, id: loadedFile.id,
			effectiveVersion: loadedFile.effectiveVersion,
			overlaid:         loadedFile.overlaid, cgoOriginal: loadedFile.cgoOriginal,
			byteDigest: loadedFile.byteDigest,
			units:      append([]SourceUnit(nil), loadedFile.units...),
		}
		out.files = append(out.files, file)
	}
	sort.Slice(out.files, func(i, j int) bool { return out.files[i].id.Rel() < out.files[j].id.Rel() })
	return out
}
