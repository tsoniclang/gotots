package source

import (
	"go/ast"
	"go/token"
	"go/types"
	"sort"

	"github.com/tsoniclang/gotots/internal/identity"
)

// FileEvidence is the sealed structural evidence state of one finalized file.
// The variant IS the state: there is no severed flag and no nil-plus-flag
// dual representation.
type FileEvidence interface{ fileEvidence() }

// FullSyntax retains the complete checked file tree: every unit in the file
// is full-semantic.
type FullSyntax struct{ Syntax *ast.File }

func (FullSyntax) fileEvidence() {}

// ContractOnly retains no syntax: every unit in the file is
// declaration-contract, external-boundary, or intrinsic; unit boundaries and
// the shared type graph carry the evidence.
type ContractOnly struct{}

func (ContractOnly) fileEvidence() {}

// MixedUnits retains exactly the full-semantic units' declaration subtrees
// (go/ast has no parent pointers, so a retained declaration cannot reach the
// enclosing file or sibling declarations).
type MixedUnits struct{ Retained []RetainedUnit }

func (MixedUnits) fileEvidence() {}

// RetainedUnit is one full-semantic unit's retained declaration subtree.
type RetainedUnit struct {
	Unit identity.SourceUnitID
	Decl ast.Node
}

// Finalize consumes the transient universe plus the scope phase's immutable
// per-unit evidence-depth selection and produces the finalized Workspace:
// structural per-file evidence, per-unit filtered type information, and
// depth-aware validated admission. The selection must be total over the
// census — every censused unit selected exactly once, no extras.
func Finalize(u *Universe, depths map[identity.SourceUnitID]EvidenceDepth) (*Workspace, error) {
	failScope := func(reason string) (*Workspace, error) {
		return nil, &LoadError{Dir: u.request.Dir, Reason: "scope selection invalid: " + reason}
	}
	remaining := make(map[identity.SourceUnitID]bool, len(depths))
	for id, depth := range depths {
		if !depth.Valid() {
			return failScope(id.String() + " has invalid depth")
		}
		remaining[id] = true
	}
	ws := &Workspace{fset: u.fset, toolchain: u.toolchain}
	for _, loaded := range u.packages {
		record, err := finalizePackage(u, loaded, depths, remaining)
		if err != nil {
			return nil, err
		}
		if err := ws.admit(record); err != nil {
			return nil, err
		}
	}
	for id := range remaining {
		return failScope("selection names unknown unit " + id.String())
	}
	sort.Slice(ws.packages, func(i, j int) bool { return ws.packages[i].id.String() < ws.packages[j].id.String() })
	sort.Slice(ws.roots, func(i, j int) bool { return ws.roots[i].id.String() < ws.roots[j].id.String() })
	return ws, nil
}

// finalizePackage builds one finalized package record with structural
// evidence per depth.
func finalizePackage(u *Universe, loaded *LoadedPackage, depths map[identity.SourceUnitID]EvidenceDepth, remaining map[identity.SourceUnitID]bool) (*Package, error) {
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
	var fullSpans []Span
	allFull, anyFull := true, false
	for _, loadedFile := range loaded.files {
		file := &File{
			path: loadedFile.path, id: loadedFile.id, fset: loadedFile.fset,
			effectiveVersion: loadedFile.effectiveVersion,
			overlaid:         loadedFile.overlaid, cgoOriginal: loadedFile.cgoOriginal,
			byteDigest: loadedFile.byteDigest,
		}
		fileAllFull, fileAnyFull := true, false
		for _, unit := range loadedFile.units {
			depth, selected := depths[unit.id]
			if !selected {
				return nil, &LoadError{Dir: u.request.Dir, Reason: "scope selection missing unit " + unit.id.String()}
			}
			delete(remaining, unit.id)
			unit.depth = depth
			file.units = append(file.units, unit)
			if depth == DepthFullSemantic {
				fileAnyFull, anyFull = true, true
				fullSpans = append(fullSpans, unit.span)
			} else {
				fileAllFull, allFull = false, false
			}
		}
		if len(loadedFile.units) == 0 {
			// Declaration-only files (types, constants) carry no executable
			// units; their evidence class follows the package's other files
			// and the shared type graph. They are full-syntax retained only
			// when the file itself is checked and the package retains
			// full-semantic evidence.
			fileAllFull = loadedFile.syntax != nil
		}
		switch {
		case loadedFile.syntax != nil && fileAllFull:
			file.evidence = FullSyntax{Syntax: loadedFile.syntax}
			if len(loadedFile.units) == 0 && loaded.disposition == DispositionOrdinarySource {
				// keep declaration-only files with their package's retention
				// class: contract packages retain no syntax
				if !packageRetainsFull(loaded, depths) {
					file.evidence = ContractOnly{}
				}
			}
		case fileAnyFull:
			retained, err := retainUnits(u, loaded, loadedFile, depths)
			if err != nil {
				return nil, err
			}
			file.evidence = MixedUnits{Retained: retained}
		default:
			file.evidence = ContractOnly{}
		}
		out.files = append(out.files, file)
	}
	sort.Slice(out.files, func(i, j int) bool { return out.files[i].id.Rel() < out.files[j].id.Rel() })
	// Type information retention follows the depth partition structurally:
	// uniform-full packages keep the checker's Info (it indexes only
	// full-semantic bodies); mixed packages keep a filtered per-unit view;
	// packages with no full-semantic unit keep none. The shared
	// types.Package declaration graph is always retained.
	switch {
	case anyFull && allFull && loaded.typesInfo != nil:
		out.typesInfo = loaded.typesInfo
	case anyFull && loaded.typesInfo != nil:
		out.typesInfo = filterInfo(loaded.typesInfo, u.fset, fullSpans)
	}
	return out, nil
}

// packageRetainsFull reports whether any unit of the package is full-semantic.
func packageRetainsFull(loaded *LoadedPackage, depths map[identity.SourceUnitID]EvidenceDepth) bool {
	for _, file := range loaded.files {
		for _, unit := range file.units {
			if depths[unit.id] == DepthFullSemantic {
				return true
			}
		}
	}
	return false
}

// retainUnits collects the full-semantic units' declaration subtrees of one
// mixed file. For cgo originals the retained subtree comes from the checked
// view through the origin mapping.
func retainUnits(u *Universe, loaded *LoadedPackage, file *LoadedFile, depths map[identity.SourceUnitID]EvidenceDepth) ([]RetainedUnit, error) {
	var out []RetainedUnit
	for _, unit := range file.units {
		if depths[unit.id] != DepthFullSemantic {
			continue
		}
		var decl ast.Node
		if file.syntax != nil {
			decl = unitNodeAt(file.syntax, file.fset, unit)
		} else {
			for _, checked := range loaded.checkedDecls {
				if checked.origin == unit.id {
					decl = checked.node
					break
				}
			}
		}
		if decl == nil {
			return nil, &LoadError{Dir: u.request.Dir, Reason: "no retainable declaration for full-semantic unit " + unit.id.String()}
		}
		out = append(out, RetainedUnit{Unit: unit.id, Decl: decl})
	}
	return out, nil
}

// unitNodeAt finds the tight retainable node of one unit: the FuncDecl whose
// body is the unit, or the exact ValueSpec — never a wider GenDecl that would
// retain sibling non-full specs.
func unitNodeAt(file *ast.File, fset *token.FileSet, unit SourceUnit) ast.Node {
	span := unit.span
	within := func(n ast.Node) bool {
		start := fset.PositionFor(n.Pos(), false).Offset
		end := fset.PositionFor(n.End(), false).Offset
		return start <= span.Start.Offset && span.End.Offset <= end
	}
	for _, decl := range file.Decls {
		if !within(decl) {
			continue
		}
		if generic, ok := decl.(*ast.GenDecl); ok {
			for _, spec := range generic.Specs {
				specStart := fset.PositionFor(spec.Pos(), false).Offset
				specEnd := fset.PositionFor(spec.End(), false).Offset
				if specStart == span.Start.Offset && specEnd == span.End.Offset {
					return spec
				}
			}
			continue
		}
		return decl
	}
	return nil
}

// filterInfo copies exactly the type-information entries whose positions lie
// within full-semantic unit spans; nothing else survives, so no retained
// Info key can reference a non-full body.
func filterInfo(info *types.Info, fset *token.FileSet, spans []Span) *types.Info {
	within := func(pos token.Pos) bool {
		offset := fset.PositionFor(pos, false).Offset
		for _, span := range spans {
			if span.Start.Offset <= offset && offset < span.End.Offset {
				return true
			}
		}
		return false
	}
	out := &types.Info{
		Types: map[ast.Expr]types.TypeAndValue{}, Defs: map[*ast.Ident]types.Object{},
		Uses: map[*ast.Ident]types.Object{}, Selections: map[*ast.SelectorExpr]*types.Selection{},
		Implicits: map[ast.Node]types.Object{}, Instances: map[*ast.Ident]types.Instance{},
		FileVersions: map[*ast.File]string{},
	}
	for key, value := range info.Types {
		if within(key.Pos()) {
			out.Types[key] = value
		}
	}
	for key, value := range info.Defs {
		if within(key.Pos()) {
			out.Defs[key] = value
		}
	}
	for key, value := range info.Uses {
		if within(key.Pos()) {
			out.Uses[key] = value
		}
	}
	for key, value := range info.Selections {
		if within(key.Pos()) {
			out.Selections[key] = value
		}
	}
	for key, value := range info.Implicits {
		if within(key.Pos()) {
			out.Implicits[key] = value
		}
	}
	for key, value := range info.Instances {
		if within(key.Pos()) {
			out.Instances[key] = value
		}
	}
	return out
}
