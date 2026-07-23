package source

import "sort"

// Finalize validates immutable acquisition evidence and severs the one
// transient syntax/checker graph. Structural, selection, executable, and
// semantic artifacts are separate outputs and are never copied into source.
func Finalize(universe *Universe) (*Workspace, error) {
	return finalize(universe, true)
}

// FinalizeResolved seals a metadata-only provider-audit universe. Individual
// package hydration forks must already have been discarded; this route never
// fabricates a semantic-hydration state on the base universe.
func FinalizeResolved(universe *Universe) (*Workspace, error) {
	return finalize(universe, false)
}

func finalize(
	universe *Universe,
	requireHydration bool,
) (*Workspace, error) {
	if universe == nil {
		return nil, &LoadError{
			Reason: "source finalization requires a universe",
		}
	}
	if universe.finalized {
		return nil, &LoadError{
			Reason: "source universe is already finalized",
		}
	}
	if requireHydration && !universe.hydrated {
		return nil, &LoadError{
			Reason: "source finalization requires semantic hydration",
		}
	}
	if !requireHydration && universe.hydrated {
		return nil, &LoadError{
			Reason: "resolved-only finalization received hydrated evidence",
		}
	}
	workspace := &Workspace{
		toolchain:             universe.toolchain,
		resolutionFingerprint: universe.ResolutionFingerprint(),
	}
	for _, loaded := range universe.packages {
		record, err := finalizePackage(loaded)
		if err != nil {
			return nil, err
		}
		if err := workspace.admit(record); err != nil {
			return nil, err
		}
	}
	sort.Slice(workspace.packages, func(i, j int) bool {
		return workspace.packages[i].id.String() < workspace.packages[j].id.String()
	})
	sort.Slice(workspace.roots, func(i, j int) bool {
		return workspace.roots[i].id.String() < workspace.roots[j].id.String()
	})
	if universe.hydrated {
		severTransientGraph(universe)
	} else {
		universe.finalized = true
	}
	return workspace, nil
}

func severTransientGraph(universe *Universe) {
	clearTransientEvidence(universe)
	universe.finalized = true
}

func clearTransientEvidence(universe *Universe) {
	for _, loaded := range universe.packages {
		if !universe.hydrationOwners[loaded.id] {
			continue
		}
		loaded.types = nil
		loaded.typesInfo = nil
		loaded.checkedDecls = nil
		for _, file := range loaded.files {
			file.fset = nil
			file.syntax = nil
			file.checkerFile = nil
			file.physicalFset = nil
			file.physicalSyntax = nil
			file.selectedBytes = nil
		}
	}
	universe.fset = nil
	universe.hydrationOwners = nil
	universe.hydrated = false
}

func finalizePackage(loaded *LoadedPackage) (*Package, error) {
	if loaded.types != nil && loaded.types.Path() != loaded.id.ImportPath() {
		return nil, &LoadError{Reason: "type graph path disagrees with package identity"}
	}
	record := &Package{
		id: loaded.id, provenance: loaded.provenance,
		acquisition: loaded.acquisition, disposition: loaded.disposition,
		moduleGoVersion: loaded.moduleGoVersion, requestedRoot: loaded.requestedRoot,
		imports:        append([]string(nil), loaded.imports...),
		embedPatterns:  append([]string(nil), loaded.embedPatterns...),
		hasCheckedView: loaded.hasCheckedView,
	}
	for _, input := range loaded.inputs {
		record.inputs = append(record.inputs, Input{
			id: input.id, kind: input.kind, byteDigest: input.byteDigest,
			overlaid: input.overlaid,
		})
	}
	for _, loadedFile := range loaded.files {
		record.files = append(record.files, &File{
			id:               loadedFile.id,
			effectiveVersion: loadedFile.effectiveVersion,
			overlaid:         loadedFile.overlaid, cgoOriginal: loadedFile.cgoOriginal,
			byteDigest: loadedFile.byteDigest,
		})
	}
	sort.Slice(record.files, func(i, j int) bool {
		return record.files[i].id.Rel() < record.files[j].id.Rel()
	})
	sort.Slice(record.inputs, func(i, j int) bool {
		return record.inputs[i].id.String() <
			record.inputs[j].id.String()
	})
	return record, nil
}
